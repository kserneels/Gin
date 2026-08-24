package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "hash" {
		runHashCommand()
		return
	}

	addr := envOr("ADDR", ":8080")
	dbPath := envOr("DB_PATH", "ginlog.db")

	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	authUsername := os.Getenv("AUTH_USERNAME")
	authPassword := os.Getenv("AUTH_PASSWORD")
	authPasswordHash := os.Getenv("AUTH_PASSWORD_HASH")
	if authUsername != "" && authPassword == "" && authPasswordHash == "" {
		log.Fatal("AUTH_USERNAME is set but neither AUTH_PASSWORD nor AUTH_PASSWORD_HASH is set")
	}
	cookieSecure := os.Getenv("INSECURE_COOKIE") == ""
	auth := NewAuth(db, authUsername, authPassword, authPasswordHash, cookieSecure)
	if !auth.enabled() {
		log.Println("WARNING: AUTH_USERNAME/AUTH_PASSWORD_HASH not set — running without authentication")
	}

	srv := &Server{store: NewStore(db), auth: auth}

	log.Printf("GinLog listening on %s (db: %s, auth: %v)", addr, dbPath, auth.enabled())
	if err := http.ListenAndServe(addr, srv.routes()); err != nil {
		log.Fatal(err)
	}
}

func runHashCommand() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "usage: ginlog hash <password>")
		os.Exit(1)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(os.Args[2]), bcrypt.DefaultCost)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(string(hash))
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
