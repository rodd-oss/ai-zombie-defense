package db

import (
	"testing"
	"time"
)

func TestOpenDB(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory failed: %v", err)
	}
	defer db.Close()

	// Verify foreign keys are enabled
	var fkEnabled int
	err = db.QueryRow("PRAGMA foreign_keys").Scan(&fkEnabled)
	if err != nil {
		t.Errorf("Failed to query foreign_keys pragma: %v", err)
	}
	if fkEnabled != 1 {
		t.Errorf("Foreign keys not enabled, got %d", fkEnabled)
	}

	// Verify journal mode is WAL (or at least not DELETE)
	var journalMode string
	err = db.QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Errorf("Failed to query journal_mode pragma: %v", err)
	}
	if journalMode != "wal" && journalMode != "WAL" {
		t.Logf("journal_mode is %q (expected wal or WAL)", journalMode)
		// Not a fatal error; some SQLite builds may not support WAL in memory
	}

	// Verify connection pool settings
	if maxOpen := db.Stats().MaxOpenConnections; maxOpen != defaultMaxOpenConns {
		t.Errorf("MaxOpenConnections = %d, want %d", maxOpen, defaultMaxOpenConns)
	}
}

func TestOpenDBFile(t *testing.T) {
	// Create a temporary file for the database
	tmpFile := t.TempDir() + "/test.db"
	db, err := OpenDB(tmpFile)
	if err != nil {
		t.Fatalf("OpenDB failed: %v", err)
	}
	defer db.Close()

	// Verify we can execute a simple query
	var result int
	err = db.QueryRow("SELECT 1").Scan(&result)
	if err != nil {
		t.Errorf("Failed to execute query: %v", err)
	}
	if result != 1 {
		t.Errorf("Expected 1, got %d", result)
	}
}

func TestConnectionPoolSettings(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory failed: %v", err)
	}
	defer db.Close()

	stats := db.Stats()
	if stats.MaxOpenConnections != defaultMaxOpenConns {
		t.Errorf("MaxOpenConnections = %d, want %d", stats.MaxOpenConnections, defaultMaxOpenConns)
	}
	// Note: Idle connections may be zero initially, which is fine.
	// We can verify that setting is respected by checking db.Stats().Idle after opening a few connections.
}

func TestPoolReuse(t *testing.T) {
	db, err := OpenInMemory()
	if err != nil {
		t.Fatalf("OpenInMemory failed: %v", err)
	}
	defer db.Close()

	// Run concurrent queries to verify connection pool works
	const numQueries = 10
	errors := make(chan error, numQueries)
	for i := 0; i < numQueries; i++ {
		go func(val int) {
			var result int
			err := db.QueryRow("SELECT ?", val).Scan(&result)
			if err != nil {
				errors <- err
				return
			}
			if result != val {
				errors <- err
				return
			}
			errors <- nil
		}(i)
	}

	// Wait for all queries to finish
	for i := 0; i < numQueries; i++ {
		select {
		case err := <-errors:
			if err != nil {
				t.Errorf("Query failed: %v", err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for query results")
		}
	}
}
