package loot_test

import (
	"context"
	"database/sql"
	"testing"

	"ai-zombie-defense/server/internal/services/loot"
	"ai-zombie-defense/server/pkg/config"

	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}
	createTableSQL := `CREATE TABLE players (
    player_id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    last_login_at TEXT,
    is_banned INTEGER NOT NULL DEFAULT 0,
    banned_reason TEXT,
    banned_until TEXT,
    is_admin INTEGER NOT NULL DEFAULT 0
);`
	if _, err := db.Exec(createTableSQL); err != nil {
		t.Fatalf("Failed to create players table: %v", err)
	}
	createCosmeticItemsSQL := `CREATE TABLE cosmetic_items (
		cosmetic_id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		slot TEXT NOT NULL,
		category TEXT,
		rarity TEXT NOT NULL,
		unlock_level INTEGER NOT NULL DEFAULT 1,
		data_cost INTEGER NOT NULL DEFAULT 0,
		is_prestige_only INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);`
	if _, err := db.Exec(createCosmeticItemsSQL); err != nil {
		t.Fatalf("Failed to create cosmetic_items table: %v", err)
	}
	createLootTablesSQL := `CREATE TABLE loot_tables (
		loot_table_id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		drop_chance REAL NOT NULL,
		is_active INTEGER NOT NULL DEFAULT 1,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);`
	if _, err := db.Exec(createLootTablesSQL); err != nil {
		t.Fatalf("Failed to create loot_tables table: %v", err)
	}
	createLootTableEntriesSQL := `CREATE TABLE loot_table_entries (
		loot_entry_id INTEGER PRIMARY KEY AUTOINCREMENT,
		loot_table_id INTEGER NOT NULL,
		cosmetic_id INTEGER NOT NULL,
		weight INTEGER NOT NULL,
		min_quantity INTEGER NOT NULL DEFAULT 1,
		max_quantity INTEGER NOT NULL DEFAULT 1,
		FOREIGN KEY (loot_table_id) REFERENCES loot_tables (loot_table_id) ON DELETE CASCADE,
		FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(createLootTableEntriesSQL); err != nil {
		t.Fatalf("Failed to create loot_table_entries table: %v", err)
	}
	createPlayerCosmeticsSQL := `CREATE TABLE player_cosmetics (
		player_id INTEGER NOT NULL,
		cosmetic_id INTEGER NOT NULL,
		unlocked_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		unlocked_via TEXT NOT NULL,
		PRIMARY KEY (player_id, cosmetic_id),
		FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
		FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(createPlayerCosmeticsSQL); err != nil {
		t.Fatalf("Failed to create player_cosmetics table: %v", err)
	}
	return db
}

func TestLootService_GenerateLootDrop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	cfg := config.Config{}
	service := loot.NewLootService(cfg, logger, dbConn)

	ctx := context.Background()

	// Insert a player
	_, err := dbConn.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testplayer", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = dbConn.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testplayer").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Insert a cosmetic item
	_, err = dbConn.Exec(`INSERT INTO cosmetic_items (name, slot, rarity, unlock_level, data_cost) VALUES (?, ?, ?, ?, ?)`,
		"Test Skin", "character_skin", "rare", 1, 0)
	if err != nil {
		t.Fatalf("Failed to insert cosmetic item: %v", err)
	}
	var cosmeticID int64
	err = dbConn.QueryRow(`SELECT cosmetic_id FROM cosmetic_items WHERE name = ?`, "Test Skin").Scan(&cosmeticID)
	if err != nil {
		t.Fatalf("Failed to get cosmetic ID: %v", err)
	}

	// Insert loot table with 100% drop chance
	_, err = dbConn.Exec(`INSERT INTO loot_tables (name, drop_chance, is_active) VALUES (?, ?, ?)`,
		"Test Loot Table", 1.0, 1)
	if err != nil {
		t.Fatalf("Failed to insert loot table: %v", err)
	}
	var lootTableID int64
	err = dbConn.QueryRow(`SELECT loot_table_id FROM loot_tables WHERE name = ?`, "Test Loot Table").Scan(&lootTableID)
	if err != nil {
		t.Fatalf("Failed to get loot table ID: %v", err)
	}

	// Insert loot table entry
	_, err = dbConn.Exec(`INSERT INTO loot_table_entries (loot_table_id, cosmetic_id, weight, min_quantity, max_quantity) VALUES (?, ?, ?, ?, ?)`,
		lootTableID, cosmeticID, 100, 1, 1)
	if err != nil {
		t.Fatalf("Failed to insert loot table entry: %v", err)
	}

	// Generate loot drop
	cosmetic, err := service.GenerateLootDrop(ctx, playerID)
	if err != nil {
		t.Fatalf("GenerateLootDrop failed: %v", err)
	}

	if cosmetic.CosmeticID != cosmeticID {
		t.Errorf("Expected cosmetic ID %d, got %d", cosmeticID, cosmetic.CosmeticID)
	}
}
