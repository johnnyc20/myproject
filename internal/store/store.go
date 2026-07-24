package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

// ErrInvalidReference is returned when a memory relationship refers to a
// from/to memory id that doesn't exist.
var ErrInvalidReference = errors.New("invalid reference")

// sqliteConstraintForeignKey is SQLITE_CONSTRAINT_FOREIGNKEY (787) — pulled
// in as a plain constant rather than importing modernc.org/sqlite/lib for
// one value.
const sqliteConstraintForeignKey = 787

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

type Note struct {
	ID        int64     `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
}

type Memory struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MemoryRelationship is a directed edge between two memories, making the
// memories table a lightweight knowledge graph (nodes = memories, edges =
// relationships) on top of the existing SQLite store rather than a
// separate graph database.
type MemoryRelationship struct {
	ID           int64     `json:"id"`
	FromMemoryID int64     `json:"from_memory_id"`
	ToMemoryID   int64     `json:"to_memory_id"`
	Type         string    `json:"type"`
	CreatedAt    time.Time `json:"created_at"`
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
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
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			body TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	if err := s.migrateMemories(); err != nil {
		return err
	}
	return s.migrateMemoryRelationships()
}

func (s *Store) migrateMemories() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			description TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(
			name, description, content,
			content='memories', content_rowid='id'
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_ai AFTER INSERT ON memories BEGIN
			INSERT INTO memories_fts(rowid, name, description, content)
			VALUES (new.id, new.name, new.description, new.content);
		END
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_ad AFTER DELETE ON memories BEGIN
			INSERT INTO memories_fts(memories_fts, rowid, name, description, content)
			VALUES ('delete', old.id, old.name, old.description, old.content);
		END
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE TRIGGER IF NOT EXISTS memories_au AFTER UPDATE ON memories BEGIN
			INSERT INTO memories_fts(memories_fts, rowid, name, description, content)
			VALUES ('delete', old.id, old.name, old.description, old.content);
			INSERT INTO memories_fts(rowid, name, description, content)
			VALUES (new.id, new.name, new.description, new.content);
		END
	`)
	return err
}

func (s *Store) migrateMemoryRelationships() error {
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS memory_relationships (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			from_memory_id INTEGER NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
			to_memory_id INTEGER NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
			type TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS memory_relationships_from_idx
			ON memory_relationships(from_memory_id)
	`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
		CREATE INDEX IF NOT EXISTS memory_relationships_to_idx
			ON memory_relationships(to_memory_id)
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

func (s *Store) ListNotes() ([]Note, error) {
	rows, err := s.db.Query(`SELECT id, body, created_at FROM notes ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notes := []Note{}
	for rows.Next() {
		var n Note
		if err := rows.Scan(&n.ID, &n.Body, &n.CreatedAt); err != nil {
			return nil, err
		}
		notes = append(notes, n)
	}
	return notes, rows.Err()
}

func (s *Store) GetNote(id int64) (Note, error) {
	var n Note
	err := s.db.QueryRow(`SELECT id, body, created_at FROM notes WHERE id = ?`, id).
		Scan(&n.ID, &n.Body, &n.CreatedAt)
	return n, err
}

func (s *Store) CreateNote(body string) (Note, error) {
	res, err := s.db.Exec(`INSERT INTO notes (body) VALUES (?)`, body)
	if err != nil {
		return Note{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Note{}, err
	}
	return s.GetNote(id)
}

const memoryColumns = `id, name, type, description, content, created_at, updated_at`

func scanMemory(row interface{ Scan(...any) error }) (Memory, error) {
	var m Memory
	err := row.Scan(&m.ID, &m.Name, &m.Type, &m.Description, &m.Content, &m.CreatedAt, &m.UpdatedAt)
	return m, err
}

func (s *Store) ListMemories(memType string) ([]Memory, error) {
	query := `SELECT ` + memoryColumns + ` FROM memories`
	args := []any{}
	if memType != "" {
		query += ` WHERE type = ?`
		args = append(args, memType)
	}
	query += ` ORDER BY id`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := []Memory{}
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func (s *Store) GetMemory(id int64) (Memory, error) {
	row := s.db.QueryRow(`SELECT `+memoryColumns+` FROM memories WHERE id = ?`, id)
	return scanMemory(row)
}

func (s *Store) CreateMemory(name, memType, description, content string) (Memory, error) {
	res, err := s.db.Exec(
		`INSERT INTO memories (name, type, description, content) VALUES (?, ?, ?, ?)`,
		name, memType, description, content,
	)
	if err != nil {
		return Memory{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Memory{}, err
	}
	return s.GetMemory(id)
}

func (s *Store) DeleteMemory(id int64) error {
	res, err := s.db.Exec(`DELETE FROM memories WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// ftsPhraseQuery wraps user input as a quoted FTS5 phrase so characters like
// "-" aren't interpreted as query syntax (NOT, column filters, etc).
func ftsPhraseQuery(query string) string {
	return `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
}

func (s *Store) SearchMemories(query string) ([]Memory, error) {
	rows, err := s.db.Query(`
		SELECT m.id, m.name, m.type, m.description, m.content, m.created_at, m.updated_at
		FROM memories_fts
		JOIN memories m ON m.id = memories_fts.rowid
		WHERE memories_fts MATCH ?
		ORDER BY rank
	`, ftsPhraseQuery(query))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	memories := []Memory{}
	for rows.Next() {
		m, err := scanMemory(rows)
		if err != nil {
			return nil, err
		}
		memories = append(memories, m)
	}
	return memories, rows.Err()
}

func scanMemoryRelationship(row interface{ Scan(...any) error }) (MemoryRelationship, error) {
	var r MemoryRelationship
	err := row.Scan(&r.ID, &r.FromMemoryID, &r.ToMemoryID, &r.Type, &r.CreatedAt)
	return r, err
}

func (s *Store) CreateMemoryRelationship(fromID, toID int64, relType string) (MemoryRelationship, error) {
	res, err := s.db.Exec(
		`INSERT INTO memory_relationships (from_memory_id, to_memory_id, type) VALUES (?, ?, ?)`,
		fromID, toID, relType,
	)
	if err != nil {
		var sqliteErr *sqlite.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code() == sqliteConstraintForeignKey {
			return MemoryRelationship{}, ErrInvalidReference
		}
		return MemoryRelationship{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return MemoryRelationship{}, err
	}
	row := s.db.QueryRow(
		`SELECT id, from_memory_id, to_memory_id, type, created_at FROM memory_relationships WHERE id = ?`,
		id,
	)
	return scanMemoryRelationship(row)
}

// ListMemoryRelationships returns every edge touching memoryID, in either
// direction, since a knowledge-graph traversal from a node needs both its
// outgoing and incoming edges.
func (s *Store) ListMemoryRelationships(memoryID int64) ([]MemoryRelationship, error) {
	rows, err := s.db.Query(`
		SELECT id, from_memory_id, to_memory_id, type, created_at
		FROM memory_relationships
		WHERE from_memory_id = ? OR to_memory_id = ?
		ORDER BY id
	`, memoryID, memoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	relationships := []MemoryRelationship{}
	for rows.Next() {
		r, err := scanMemoryRelationship(rows)
		if err != nil {
			return nil, err
		}
		relationships = append(relationships, r)
	}
	return relationships, rows.Err()
}

func (s *Store) DeleteMemoryRelationship(id int64) error {
	res, err := s.db.Exec(`DELETE FROM memory_relationships WHERE id = ?`, id)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}
