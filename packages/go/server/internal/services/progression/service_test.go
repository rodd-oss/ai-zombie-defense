package progression_test

import (
	"context"
	"database/sql"
	"testing"

	"ai-zombie-defense/server/internal/services/progression"
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
	createProgressionSQL := `CREATE TABLE player_progression (
    player_id INTEGER PRIMARY KEY,
    level INTEGER NOT NULL DEFAULT 1,
    experience INTEGER NOT NULL DEFAULT 0,
    prestige_level INTEGER NOT NULL DEFAULT 0,
    data_currency INTEGER NOT NULL DEFAULT 0,
    total_matches_played INTEGER NOT NULL DEFAULT 0,
    total_waves_survived INTEGER NOT NULL DEFAULT 0,
    total_kills INTEGER NOT NULL DEFAULT 0,
    total_deaths INTEGER NOT NULL DEFAULT 0,
    total_scrap_earned INTEGER NOT NULL DEFAULT 0,
    total_data_earned INTEGER NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
);`
	if _, err := db.Exec(createProgressionSQL); err != nil {
		t.Fatalf("Failed to create player_progression table: %v", err)
	}
	return db
}

func TestProgressionService_AddExperience(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	cfg := config.Config{
		Progression: config.ProgressionConfig{
			BaseXPPerLevel: 1000,
		},
	}
	service := progression.NewProgressionService(cfg, logger, dbConn)

	ctx := context.Background()

	// Insert a player
	_, err := dbConn.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = dbConn.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Get initial progression (should be defaults)
	progressionData, err := service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression: %v", err)
	}
	if progressionData.Level != 1 {
		t.Errorf("Expected initial level 1, got %d", progressionData.Level)
	}

	// Add XP below level threshold
	err = service.AddExperience(ctx, playerID, 500)
	if err != nil {
		t.Fatalf("AddExperience failed: %v", err)
	}
	progressionData, err = service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression after XP: %v", err)
	}
	if progressionData.Experience != 500 {
		t.Errorf("Expected XP 500, got %d", progressionData.Experience)
	}
	if progressionData.Level != 1 {
		t.Errorf("Expected level 1, got %d", progressionData.Level)
	}

	// Add XP to reach level 2
	err = service.AddExperience(ctx, playerID, 500)
	if err != nil {
		t.Fatalf("AddExperience failed: %v", err)
	}
	progressionData, err = service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression after level up: %v", err)
	}
	if progressionData.Level != 2 {
		t.Errorf("Expected level 2, got %d", progressionData.Level)
	}
}

func TestProgressionService_AddMatchRewards(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	cfg := config.Config{
		Progression: config.ProgressionConfig{
			BaseXPPerLevel: 1000,
		},
	}
	service := progression.NewProgressionService(cfg, logger, dbConn)

	ctx := context.Background()

	// Insert a player
	_, err := dbConn.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = dbConn.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Ensure progression row exists
	_, err = service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to ensure progression row: %v", err)
	}

	// Add match rewards
	kills := int64(10)
	deaths := int64(2)
	wavesSurvived := int64(5)
	scrapEarned := int64(500)
	dataEarned := int64(100)
	err = service.AddMatchRewards(ctx, playerID, kills, deaths, wavesSurvived, scrapEarned, dataEarned)
	if err != nil {
		t.Fatalf("AddMatchRewards failed: %v", err)
	}

	// Verify progression after match
	progressionData, err := service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression after match: %v", err)
	}
	if progressionData.TotalMatchesPlayed != 1 {
		t.Errorf("Expected 1 match played, got %d", progressionData.TotalMatchesPlayed)
	}
	if progressionData.DataCurrency != dataEarned {
		t.Errorf("Expected data currency %d, got %d", dataEarned, progressionData.DataCurrency)
	}
}
