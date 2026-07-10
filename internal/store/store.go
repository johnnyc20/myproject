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

type Widget struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Price     int64     `json:"price"`
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
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS widgets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			price INTEGER NOT NULL,
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

func (s *Store) ListWidgets() ([]Widget, error) {
	rows, err := s.db.Query(`SELECT id, name, price, created_at FROM widgets ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	widgets := []Widget{}
	for rows.Next() {
		var w Widget
		if err := rows.Scan(&w.ID, &w.Name, &w.Price, &w.CreatedAt); err != nil {
			return nil, err
		}
		widgets = append(widgets, w)
	}
	return widgets, rows.Err()
}

func (s *Store) GetWidget(id int64) (Widget, error) {
	var w Widget
	err := s.db.QueryRow(`SELECT id, name, price, created_at FROM widgets WHERE id = ?`, id).
		Scan(&w.ID, &w.Name, &w.Price, &w.CreatedAt)
	return w, err
}

func (s *Store) CreateWidget(name string, price int64) (Widget, error) {
	res, err := s.db.Exec(`INSERT INTO widgets (name, price) VALUES (?, ?)`, name, price)
	if err != nil {
		return Widget{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Widget{}, err
	}
	return s.GetWidget(id)
}

func (s *Store) UpdateWidget(id int64, name string, price int64) (Widget, error) {
	res, err := s.db.Exec(`UPDATE widgets SET name = ?, price = ? WHERE id = ?`, name, price, id)
	if err != nil {
		return Widget{}, err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return Widget{}, err
	}
	if rows == 0 {
		return Widget{}, sql.ErrNoRows
	}
	return s.GetWidget(id)
}

func (s *Store) DeleteWidget(id int64) error {
	_, err := s.db.Exec(`DELETE FROM widgets WHERE id = ?`, id)
	return err
}
