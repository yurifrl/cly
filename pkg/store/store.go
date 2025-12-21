package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/tursodatabase/go-libsql"
)

// Store provides namespace/key storage for persistent app state.
type Store interface {
	List(namespace string) ([]string, error)
	Add(namespace, key string) error
	Remove(namespace, key string) error
	Close() error
}

// SQLiteStore implements Store using libSQL (Turso).
type SQLiteStore struct {
	db *sql.DB
}

// New creates a new libSQL-backed Store at the given path.
// Creates the directory and database if they don't exist.
func New(dbPath string) (Store, error) {
	// Expand ~ to home directory
	if len(dbPath) > 0 && dbPath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		dbPath = filepath.Join(home, dbPath[1:])
	}

	// Create directory if needed
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	db, err := sql.Open("libsql", "file:"+dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS packages (
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			installed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (type, name)
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

// List returns all keys in the given namespace.
func (s *SQLiteStore) List(namespace string) ([]string, error) {
	rows, err := s.db.Query("SELECT name FROM packages WHERE type = ?", namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		keys = append(keys, name)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return keys, nil
}

// Add inserts a key into the namespace. Idempotent - duplicate adds are no-ops.
func (s *SQLiteStore) Add(namespace, key string) error {
	_, err := s.db.Exec(`
		INSERT OR IGNORE INTO packages (type, name) VALUES (?, ?)
	`, namespace, key)
	if err != nil {
		return fmt.Errorf("failed to add package: %w", err)
	}
	return nil
}

// Remove deletes a key from the namespace. Idempotent - removing non-existent key is no-op.
func (s *SQLiteStore) Remove(namespace, key string) error {
	_, err := s.db.Exec("DELETE FROM packages WHERE type = ? AND name = ?", namespace, key)
	if err != nil {
		return fmt.Errorf("failed to remove package: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
