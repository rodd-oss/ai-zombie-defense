package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"ai-zombie-defense/backend-api/db/pkg/migrations"
	_ "modernc.org/sqlite"
)

func main() {
	up := flag.Bool("up", false, "Run migrations up")
	down := flag.Bool("down", false, "Rollback the latest migration")
	status := flag.Bool("status", false, "Show migration status")
	dbPath := flag.String("db", "./data.db", "Path to SQLite database file")
	migrationsDir := flag.String("migrations", "./migrations", "Path to migrations directory")
	flag.Parse()

	if !(*up || *down || *status) {
		flag.Usage()
		os.Exit(1)
	}

	// Verify migrations directory exists
	if _, err := os.Stat(*migrationsDir); os.IsNotExist(err) {
		log.Fatalf("Migrations directory does not exist: %s", *migrationsDir)
	}

	// Open database connection
	db, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		log.Printf("Warning: failed to enable WAL mode: %v", err)
	}

	switch {
	case *up:
		fmt.Println("Running migrations...")
		if err := migrations.RunMigrationsWithDir(db, *migrationsDir); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")
	case *down:
		fmt.Println("Rolling back latest migration...")
		if err := migrations.RollbackWithDir(db, *migrationsDir); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		fmt.Println("Rollback completed successfully")
	case *status:
		fmt.Println("Migration status:")
		if err := migrations.StatusWithDir(db, *migrationsDir); err != nil {
			log.Fatalf("Status check failed: %v", err)
		}
	}
}
