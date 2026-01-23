package db

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

// Default configuration values for connection pooling.
const (
	defaultMaxOpenConns    = 5
	defaultMaxIdleConns    = 2
	defaultConnMaxLifetime = 5 * time.Minute
	defaultConnMaxIdleTime = 2 * time.Minute
)

// OpenDB opens a SQLite database at the given path with connection pooling,
// foreign keys enabled, and WAL mode for better concurrency.
func OpenDB(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	db.SetMaxOpenConns(defaultMaxOpenConns)
	db.SetMaxIdleConns(defaultMaxIdleConns)
	db.SetConnMaxLifetime(defaultConnMaxLifetime)
	db.SetConnMaxIdleTime(defaultConnMaxIdleTime)

	// Enable foreign keys (required for referential integrity)
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, err
	}

	// Enable WAL mode for better concurrency and performance
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, err
	}

	// Verify connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

// OpenInMemory opens an in‑memory SQLite database with the same settings
// as OpenDB. Useful for testing.
func OpenInMemory() (*sql.DB, error) {
	return OpenDB(":memory:")
}
