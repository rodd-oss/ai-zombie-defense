package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

// RunMigrations runs all pending migrations on the database using migrations directory
// located in "./migrations" relative to the current working directory.
func RunMigrations(db *sql.DB) error {
	migrationDir, err := getMigrationDir()
	if err != nil {
		return fmt.Errorf("failed to get migration directory: %w", err)
	}
	return RunMigrationsWithDir(db, migrationDir)
}

// RunMigrationsWithDir runs all pending migrations from the specified directory.
func RunMigrationsWithDir(db *sql.DB, migrationDir string) error {
	goose.SetBaseFS(nil)
	goose.SetLogger(log.New(os.Stdout, "[migrations] ", log.LstdFlags))

	if err := goose.SetDialect("sqlite"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Up(db, migrationDir); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// Rollback rolls back the latest migration using migrations directory
// located in "./migrations" relative to the current working directory.
func Rollback(db *sql.DB) error {
	migrationDir, err := getMigrationDir()
	if err != nil {
		return fmt.Errorf("failed to get migration directory: %w", err)
	}
	return RollbackWithDir(db, migrationDir)
}

// RollbackWithDir rolls back the latest migration from the specified directory.
func RollbackWithDir(db *sql.DB, migrationDir string) error {
	goose.SetBaseFS(nil)
	goose.SetLogger(log.New(os.Stdout, "[migrations] ", log.LstdFlags))

	if err := goose.SetDialect("sqlite"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	if err := goose.Down(db, migrationDir); err != nil {
		return fmt.Errorf("failed to rollback migration: %w", err)
	}

	return nil
}

// Status prints the migration status using migrations directory
// located in "./migrations" relative to the current working directory.
func Status(db *sql.DB) error {
	migrationDir, err := getMigrationDir()
	if err != nil {
		return fmt.Errorf("failed to get migration directory: %w", err)
	}
	return StatusWithDir(db, migrationDir)
}

// StatusWithDir prints the migration status from the specified directory.
func StatusWithDir(db *sql.DB, migrationDir string) error {
	goose.SetBaseFS(nil)
	goose.SetLogger(log.New(os.Stdout, "[migrations] ", log.LstdFlags))

	if err := goose.SetDialect("sqlite"); err != nil {
		return fmt.Errorf("failed to set dialect: %w", err)
	}

	return goose.Status(db, migrationDir)
}

func getMigrationDir() (string, error) {
	// Try relative to current working directory
	dir := filepath.Join(".", "migrations")
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	// Try relative to the package root (where go.mod is)
	dir = filepath.Join("..", "migrations")
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	// Try absolute path from module root (assuming binary runs from module root)
	dir = "migrations"
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	return "", fmt.Errorf("migration directory not found (looked in ./migrations, ../migrations, migrations)")
}
