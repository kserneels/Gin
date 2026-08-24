package main

import (
	"database/sql"
	"time"
)

type Recipe struct {
	ID        int64
	Name      string
	Gin       string
	Tonic     string
	Garnish   string
	Ratio     string
	Glassware string
	Location  string
	Rating    int
	Notes     string
	CreatedAt time.Time
}

// Stars renders the rating as filled/empty star characters for templates.
func (r Recipe) Stars() string {
	stars := ""
	for i := 1; i <= 5; i++ {
		if i <= r.Rating {
			stars += "★"
		} else {
			stars += "☆"
		}
	}
	return stars
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(query string) ([]Recipe, error) {
	sqlq := `SELECT id, name, gin, tonic, garnish, ratio, glassware, location, rating, notes, created_at
	          FROM recipes`
	args := []any{}
	if query != "" {
		sqlq += ` WHERE name LIKE ? OR gin LIKE ? OR tonic LIKE ? OR garnish LIKE ? OR location LIKE ?`
		like := "%" + query + "%"
		args = append(args, like, like, like, like, like)
	}
	sqlq += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(sqlq, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recipes []Recipe
	for rows.Next() {
		var r Recipe
		if err := rows.Scan(&r.ID, &r.Name, &r.Gin, &r.Tonic, &r.Garnish, &r.Ratio, &r.Glassware, &r.Location, &r.Rating, &r.Notes, &r.CreatedAt); err != nil {
			return nil, err
		}
		recipes = append(recipes, r)
	}
	return recipes, rows.Err()
}

func (s *Store) Get(id int64) (Recipe, error) {
	var r Recipe
	row := s.db.QueryRow(`SELECT id, name, gin, tonic, garnish, ratio, glassware, location, rating, notes, created_at
	                       FROM recipes WHERE id = ?`, id)
	err := row.Scan(&r.ID, &r.Name, &r.Gin, &r.Tonic, &r.Garnish, &r.Ratio, &r.Glassware, &r.Location, &r.Rating, &r.Notes, &r.CreatedAt)
	return r, err
}

func (s *Store) Create(r Recipe) (int64, error) {
	res, err := s.db.Exec(`INSERT INTO recipes (name, gin, tonic, garnish, ratio, glassware, location, rating, notes)
	                        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.Name, r.Gin, r.Tonic, r.Garnish, r.Ratio, r.Glassware, r.Location, r.Rating, r.Notes)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (s *Store) Update(r Recipe) error {
	_, err := s.db.Exec(`UPDATE recipes SET name=?, gin=?, tonic=?, garnish=?, ratio=?, glassware=?, location=?, rating=?, notes=?
	                      WHERE id=?`,
		r.Name, r.Gin, r.Tonic, r.Garnish, r.Ratio, r.Glassware, r.Location, r.Rating, r.Notes, r.ID)
	return err
}

func (s *Store) Delete(id int64) error {
	_, err := s.db.Exec(`DELETE FROM recipes WHERE id = ?`, id)
	return err
}
