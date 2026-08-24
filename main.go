package main

import (
	"log"
	"net/http"
	"os"
)

func main() {
	addr := envOr("ADDR", ":8080")
	dbPath := envOr("DB_PATH", "ginlog.db")

	db, err := openDB(dbPath)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	srv := &Server{store: NewStore(db)}

	log.Printf("GinLog listening on %s (db: %s)", addr, dbPath)
	if err := http.ListenAndServe(addr, srv.routes()); err != nil {
		log.Fatal(err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
