package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"ai-zombie-defense/backend-api/internal/db"
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
	sqlDB, err := sql.Open("sqlite", *dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer sqlDB.Close()

	// Enable foreign keys
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := sqlDB.Exec("PRAGMA journal_mode = WAL"); err != nil {
		log.Printf("Warning: failed to enable WAL mode: %v", err)
	}

	switch {
	case *up:
		fmt.Println("Running migrations...")
		if err := db.RunMigrationsWithDir(sqlDB, *migrationsDir); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		fmt.Println("Migrations completed successfully")
	case *down:
		fmt.Println("Rolling back latest migration...")
		if err := db.RollbackWithDir(sqlDB, *migrationsDir); err != nil {
			log.Fatalf("Rollback failed: %v", err)
		}
		fmt.Println("Rollback completed successfully")
	case *status:
		fmt.Println("Migration status:")
		if err := db.StatusWithDir(sqlDB, *migrationsDir); err != nil {
			log.Fatalf("Status check failed: %v", err)
		}
	}
}
