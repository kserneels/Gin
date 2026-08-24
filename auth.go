package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	sessionCookieName = "ginlog_session"
	sessionTTL        = 30 * 24 * time.Hour
)

type Auth struct {
	db           *sql.DB
	username     string
	passwordHash []byte
	cookieSecure bool

	mu       sync.Mutex
	attempts map[string]*attemptState
}

type attemptState struct {
	count        int
	last         time.Time
	blockedUntil time.Time
}

func NewAuth(db *sql.DB, username, passwordHash string, cookieSecure bool) *Auth {
	return &Auth{
		db:           db,
		username:     username,
		passwordHash: []byte(passwordHash),
		cookieSecure: cookieSecure,
		attempts:     map[string]*attemptState{},
	}
}

func (a *Auth) enabled() bool {
	return a.username != ""
}

func (a *Auth) checkCredentials(username, password string) bool {
	if username != a.username {
		return false
	}
	return bcrypt.CompareHashAndPassword(a.passwordHash, []byte(password)) == nil
}

// tooManyAttempts and recordFailure implement a simple per-key (usually
// remote address) exponential backoff to slow down password guessing.
func (a *Auth) tooManyAttempts(key string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, ok := a.attempts[key]
	if !ok {
		return false
	}
	return time.Now().Before(st.blockedUntil)
}

func (a *Auth) recordFailure(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	st, ok := a.attempts[key]
	if !ok || time.Since(st.last) > 15*time.Minute {
		st = &attemptState{}
		a.attempts[key] = st
	}
	st.count++
	st.last = time.Now()
	if st.count >= 3 {
		backoff := time.Duration(st.count-2) * 5 * time.Second
		if backoff > 5*time.Minute {
			backoff = 5 * time.Minute
		}
		st.blockedUntil = time.Now().Add(backoff)
	}
}

func (a *Auth) recordSuccess(key string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.attempts, key)
}

func (a *Auth) createSession() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	_, err := a.db.Exec(`INSERT INTO sessions (token, expires_at) VALUES (?, ?)`, token, time.Now().Add(sessionTTL))
	return token, err
}

func (a *Auth) validSession(token string) bool {
	if token == "" {
		return false
	}
	var expiresAt time.Time
	err := a.db.QueryRow(`SELECT expires_at FROM sessions WHERE token = ?`, token).Scan(&expiresAt)
	if err != nil {
		return false
	}
	return time.Now().Before(expiresAt)
}

func (a *Auth) deleteSession(token string) {
	a.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
}

func (a *Auth) cleanupExpired() {
	a.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, time.Now())
}

func (a *Auth) setSessionCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})
}

func (a *Auth) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

// requireAuth wraps a handler so it only runs for a logged-in session,
// redirecting to /login otherwise. When auth is not configured, it's a
// no-op passthrough.
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.auth.enabled() {
			next(w, r)
			return
		}
		c, err := r.Cookie(sessionCookieName)
		if err != nil || !s.auth.validSession(c.Value) {
			http.Redirect(w, r, "/login?next="+url.QueryEscape(r.URL.Path), http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

func (s *Server) handleLoginForm(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.URL.Query().Get("next"))
	if s.auth.enabled() {
		if c, err := r.Cookie(sessionCookieName); err == nil && s.auth.validSession(c.Value) {
			http.Redirect(w, r, next, http.StatusSeeOther)
			return
		}
	}
	render(w, "login.html", map[string]any{"Next": next, "Error": ""})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	next := safeNext(r.FormValue("next"))
	key := r.RemoteAddr

	if s.auth.tooManyAttempts(key) {
		render(w, "login.html", map[string]any{"Next": next, "Error": "Too many attempts. Try again in a bit."})
		return
	}

	if !s.auth.checkCredentials(r.FormValue("username"), r.FormValue("password")) {
		s.auth.recordFailure(key)
		render(w, "login.html", map[string]any{"Next": next, "Error": "Invalid username or password."})
		return
	}
	s.auth.recordSuccess(key)
	s.auth.cleanupExpired()

	token, err := s.auth.createSession()
	if err != nil {
		http.Error(w, "could not create session", http.StatusInternalServerError)
		return
	}
	s.auth.setSessionCookie(w, token)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookieName); err == nil {
		s.auth.deleteSession(c.Value)
	}
	s.auth.clearSessionCookie(w)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// safeNext keeps the post-login redirect target confined to this site.
func safeNext(next string) string {
	if next == "" || !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") {
		return "/"
	}
	return next
}
