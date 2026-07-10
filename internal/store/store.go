package store

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

type Item struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func (s *Store) ListItems() ([]Item, error) {
	rows, err := s.db.Query(`SELECT id, name, created_at FROM items ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var it Item
		if err := rows.Scan(&it.ID, &it.Name, &it.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, rows.Err()
}

func (s *Store) GetItem(id int64) (Item, error) {
	var it Item
	err := s.db.QueryRow(`SELECT id, name, created_at FROM items WHERE id = ?`, id).
		Scan(&it.ID, &it.Name, &it.CreatedAt)
	return it, err
}

func (s *Store) CreateItem(name string) (Item, error) {
	res, err := s.db.Exec(`INSERT INTO items (name) VALUES (?)`, name)
	if err != nil {
		return Item{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Item{}, err
	}
	return s.GetItem(id)
}

func (s *Store) DeleteItem(id int64) error {
	_, err := s.db.Exec(`DELETE FROM items WHERE id = ?`, id)
	return err
}
