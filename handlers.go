package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"
	"strconv"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var tmpl = template.Must(template.ParseFS(templateFS, "templates/*.html"))

type Server struct {
	store *Store
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /static/", http.FileServerFS(staticFS))

	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /new", s.handleNewForm)
	mux.HandleFunc("POST /recipes", s.handleCreate)
	mux.HandleFunc("GET /recipes/{id}", s.handleShow)
	mux.HandleFunc("GET /recipes/{id}/edit", s.handleEditForm)
	mux.HandleFunc("POST /recipes/{id}", s.handleUpdate)
	mux.HandleFunc("POST /recipes/{id}/delete", s.handleDelete)

	return mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	recipes, err := s.store.List(q)
	if err != nil {
		http.Error(w, "could not load recipes", http.StatusInternalServerError)
		log.Println("list:", err)
		return
	}
	render(w, "index.html", map[string]any{
		"Recipes": recipes,
		"Query":   q,
	})
}

func (s *Server) handleNewForm(w http.ResponseWriter, r *http.Request) {
	render(w, "form.html", map[string]any{
		"Recipe": Recipe{},
		"Action": "/recipes",
		"Title":  "Add a Gin Mix",
	})
}

func (s *Server) handleCreate(w http.ResponseWriter, r *http.Request) {
	rec, err := recipeFromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	id, err := s.store.Create(rec)
	if err != nil {
		http.Error(w, "could not save recipe", http.StatusInternalServerError)
		log.Println("create:", err)
		return
	}
	http.Redirect(w, r, "/recipes/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (s *Server) handleShow(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rec, err := s.store.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "detail.html", map[string]any{"Recipe": rec})
}

func (s *Server) handleEditForm(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rec, err := s.store.Get(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	render(w, "form.html", map[string]any{
		"Recipe": rec,
		"Action": "/recipes/" + strconv.FormatInt(id, 10),
		"Title":  "Edit " + rec.Name,
	})
}

func (s *Server) handleUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	rec, err := recipeFromForm(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rec.ID = id
	if err := s.store.Update(rec); err != nil {
		http.Error(w, "could not update recipe", http.StatusInternalServerError)
		log.Println("update:", err)
		return
	}
	http.Redirect(w, r, "/recipes/"+strconv.FormatInt(id, 10), http.StatusSeeOther)
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	id, err := idParam(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if err := s.store.Delete(id); err != nil {
		http.Error(w, "could not delete recipe", http.StatusInternalServerError)
		log.Println("delete:", err)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func idParam(r *http.Request) (int64, error) {
	return strconv.ParseInt(r.PathValue("id"), 10, 64)
}

func recipeFromForm(r *http.Request) (Recipe, error) {
	if err := r.ParseForm(); err != nil {
		return Recipe{}, err
	}
	name := r.FormValue("name")
	gin := r.FormValue("gin")
	if name == "" || gin == "" {
		return Recipe{}, errBadRequest("name and gin are required")
	}
	rating, _ := strconv.Atoi(r.FormValue("rating"))
	if rating < 0 {
		rating = 0
	}
	if rating > 5 {
		rating = 5
	}
	return Recipe{
		Name:      name,
		Gin:       gin,
		Tonic:     r.FormValue("tonic"),
		Garnish:   r.FormValue("garnish"),
		Ratio:     r.FormValue("ratio"),
		Glassware: r.FormValue("glassware"),
		Location:  r.FormValue("location"),
		Rating:    rating,
		Notes:     r.FormValue("notes"),
	}, nil
}

type errBadRequest string

func (e errBadRequest) Error() string { return string(e) }

func render(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		log.Println("render:", err)
		http.Error(w, "render error", http.StatusInternalServerError)
	}
}
