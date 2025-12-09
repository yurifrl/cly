package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb"
)

// Store provides namespace/key storage for persistent app state.
type Store interface {
	List(namespace string) ([]string, error)
	Add(namespace, key string) error
	Remove(namespace, key string) error
	Close() error
}

// DuckDBStore implements Store using DuckDB.
type DuckDBStore struct {
	db *sql.DB
}

// New creates a new DuckDB-backed Store at the given path.
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

	db, err := sql.Open("duckdb", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create schema
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS packages (
			type VARCHAR NOT NULL,
			name VARCHAR NOT NULL,
			installed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (type, name)
		)
	`)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create schema: %w", err)
	}

	return &DuckDBStore{db: db}, nil
}

// List returns all keys in the given namespace.
func (s *DuckDBStore) List(namespace string) ([]string, error) {
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
func (s *DuckDBStore) Add(namespace, key string) error {
	_, err := s.db.Exec(`
		INSERT INTO packages (type, name) VALUES (?, ?)
		ON CONFLICT (type, name) DO NOTHING
	`, namespace, key)
	if err != nil {
		return fmt.Errorf("failed to add package: %w", err)
	}
	return nil
}

// Remove deletes a key from the namespace. Idempotent - removing non-existent key is no-op.
func (s *DuckDBStore) Remove(namespace, key string) error {
	_, err := s.db.Exec("DELETE FROM packages WHERE type = ? AND name = ?", namespace, key)
	if err != nil {
		return fmt.Errorf("failed to remove package: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (s *DuckDBStore) Close() error {
	return s.db.Close()
}
