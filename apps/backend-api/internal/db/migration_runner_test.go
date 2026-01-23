package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestRunMigrations(t *testing.T) {
	// Find the actual migrations directory relative to module root
	migrationsSrc, err := findMigrationsDir()
	if err != nil {
		t.Skipf("Could not find migration files: %v", err)
	}

	// Create temporary directory for migration files
	tmpDir := t.TempDir()
	migrationsDst := filepath.Join(tmpDir, "migrations")
	if err := copyMigrationFiles(migrationsSrc, migrationsDst); err != nil {
		t.Skipf("Could not copy migration files: %v", err)
	}

	// Use in-memory SQLite database
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Run migrations
	if err := RunMigrationsWithDir(db, migrationsDst); err != nil {
		t.Fatalf("RunMigrations failed: %v", err)
	}

	// Verify that tables were created
	tables := []string{
		"players",
		"sessions",
		"player_progression",
		"player_settings",
		"cosmetic_items",
		"player_cosmetics",
		"loadouts",
		"loadout_cosmetics",
		"servers",
		"matches",
		"player_match_stats",
		"leaderboard_entries",
		"friends",
		"server_favorites",
		"loot_tables",
		"loot_table_entries",
		"currency_transactions",
		"join_tokens",
	}

	for _, table := range tables {
		var count int
		query := `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`
		err := db.QueryRow(query, table).Scan(&count)
		if err != nil {
			t.Errorf("Failed to query for table %s: %v", table, err)
			continue
		}
		if count != 1 {
			t.Errorf("Table %s was not created", table)
		}
	}

	// Rollback the latest migration
	if err := RollbackWithDir(db, migrationsDst); err != nil {
		t.Fatalf("Rollback failed: %v", err)
	}
}

func findMigrationsDir() (string, error) {
	// Try from current working directory (module root)
	dir := "migrations"
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	// Try relative to test file location
	dir = filepath.Join("..", "..", "migrations")
	if _, err := os.Stat(dir); err == nil {
		return dir, nil
	}
	return "", os.ErrNotExist
}

func copyMigrationFiles(srcDir, dstDir string) error {
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		srcPath := filepath.Join(srcDir, entry.Name())
		dstPath := filepath.Join(dstDir, entry.Name())
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dstPath, data, 0644); err != nil {
			return err
		}
	}
	return nil
}
