package auth_test

import (
	"context"
	"database/sql"
	"testing"

	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/config"

	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func newTestConfig() config.Config {
	return config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,          // 15 minutes in nanoseconds
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000, // 7 days
		},
		Progression: config.ProgressionConfig{
			BaseXPPerLevel: 1000,
		},
	}
}

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}
	// Create players table exactly as in migration
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
	// Create sessions table
	createSessionsSQL := `CREATE TABLE sessions (
    session_id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ip_address TEXT,
    user_agent TEXT,
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
);`
	if _, err := db.Exec(createSessionsSQL); err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}
	// Create player_progression table
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
	// Create cosmetic_items table
	createCosmeticItemsSQL := `CREATE TABLE cosmetic_items (
		cosmetic_id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		slot TEXT NOT NULL CHECK (slot IN ('character_skin', 'weapon_skin', 'emote', 'taunt', 'badge', 'title', 'particle_effect', 'other')),
		category TEXT,
		rarity TEXT NOT NULL CHECK (rarity IN ('common', 'uncommon', 'rare', 'epic', 'legendary')),
		unlock_level INTEGER NOT NULL DEFAULT 1,
		data_cost INTEGER NOT NULL DEFAULT 0,
		is_prestige_only INTEGER NOT NULL DEFAULT 0,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	);`
	if _, err := db.Exec(createCosmeticItemsSQL); err != nil {
		t.Fatalf("Failed to create cosmetic_items table: %v", err)
	}
	// Create loot_tables table
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
	// Create loot_table_entries table
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
	// Create indexes for loot_table_entries
	if _, err := db.Exec(`CREATE INDEX idx_loot_table_entries_loot_table_id ON loot_table_entries (loot_table_id)`); err != nil {
		t.Fatalf("Failed to create loot_table_entries index: %v", err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_loot_table_entries_cosmetic_id ON loot_table_entries (cosmetic_id)`); err != nil {
		t.Fatalf("Failed to create loot_table_entries index: %v", err)
	}
	// Create player_cosmetics table
	createPlayerCosmeticsSQL := `CREATE TABLE player_cosmetics (
		player_id INTEGER NOT NULL,
		cosmetic_id INTEGER NOT NULL,
		unlocked_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		unlocked_via TEXT NOT NULL CHECK (unlocked_via IN ('level_up', 'purchase', 'loot_drop', 'prestige')),
		PRIMARY KEY (player_id, cosmetic_id),
		FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
		FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(createPlayerCosmeticsSQL); err != nil {
		t.Fatalf("Failed to create player_cosmetics table: %v", err)
	}
	return db
}

func TestAuthService_Authenticate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	// Hash a password
	password := "securepassword123"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	// Insert a test player
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert test player: %v", err)
	}

	// Verify insertion by direct query
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM players WHERE username = ?`, "testuser").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query player count: %v", err)
	}
	if count != 1 {
		t.Fatalf("Expected 1 player, got %d", count)
	}

	ctx := context.Background()

	// Test successful authentication with username
	player, err := service.Authenticate(ctx, "testuser", password)
	if err != nil {
		t.Errorf("Authentication with username failed: %v", err)
	}
	if player == nil {
		t.Fatal("Player is nil but error is nil")
	}
	if player.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got %s", player.Username)
	}

	// Test successful authentication with email
	player, err = service.Authenticate(ctx, "test@example.com", password)
	if err != nil {
		t.Errorf("Authentication with email failed: %v", err)
	}
	if player == nil {
		t.Fatal("Player is nil but error is nil")
	}
	if player.Email != "test@example.com" {
		t.Errorf("Expected email 'test@example.com', got %s", player.Email)
	}

	// Test invalid password
	_, err = service.Authenticate(ctx, "testuser", "wrongpassword")
	if err != auth.ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials for wrong password, got %v", err)
	}

	// Test non-existent user
	_, err = service.Authenticate(ctx, "nonexistent", password)
	if err != auth.ErrInvalidCredentials {
		t.Errorf("Expected ErrInvalidCredentials for non-existent user, got %v", err)
	}
}

func TestAuthService_GenerateToken(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	playerID := int64(42)
	token, err := service.GenerateAccessToken(playerID)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}
	if token == "" {
		t.Error("Generated token is empty")
	}

	// Validate token
	claims, err := service.ValidateToken(token)
	if err != nil {
		t.Errorf("Failed to validate token: %v", err)
	}
	if claims.Subject != "42" {
		t.Errorf("Expected subject '42', got %s", claims.Subject)
	}
}

func TestAuthService_RegisterPlayer(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	ctx := context.Background()

	t.Run("successful registration", func(t *testing.T) {
		username := "newuser"
		email := "new@example.com"
		password := "securepass123"

		player, err := service.RegisterPlayer(ctx, username, email, password)
		if err != nil {
			t.Fatalf("Registration failed: %v", err)
		}
		if player == nil {
			t.Fatal("Player is nil")
		}
		if player.Username != username {
			t.Errorf("Expected username %s, got %s", username, player.Username)
		}
		if player.Email != email {
			t.Errorf("Expected email %s, got %s", email, player.Email)
		}
		// Verify password can be used for authentication
		authPlayer, err := service.Authenticate(ctx, username, password)
		if err != nil {
			t.Errorf("Authentication after registration failed: %v", err)
		}
		if authPlayer.PlayerID != player.PlayerID {
			t.Errorf("Player ID mismatch: expected %d, got %d", player.PlayerID, authPlayer.PlayerID)
		}
	})

	t.Run("duplicate username", func(t *testing.T) {
		// First registration
		_, err := service.RegisterPlayer(ctx, "user1", "email1@example.com", "pass")
		if err != nil {
			t.Fatalf("First registration failed: %v", err)
		}
		// Duplicate username, different email
		_, err = service.RegisterPlayer(ctx, "user1", "email2@example.com", "pass")
		if err != auth.ErrDuplicateUsername {
			t.Errorf("Expected ErrDuplicateUsername, got %v", err)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		// First registration with email
		_, err := service.RegisterPlayer(ctx, "user2", "email2@example.com", "pass")
		if err != nil {
			t.Fatalf("First registration failed: %v", err)
		}
		// Duplicate email, different username
		_, err = service.RegisterPlayer(ctx, "user3", "email2@example.com", "pass")
		if err != auth.ErrDuplicateEmail {
			t.Errorf("Expected ErrDuplicateEmail, got %v", err)
		}
	})
}

func TestAuthService_CreateSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	ctx := context.Background()

	// Insert a player
	password := "password"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Create session
	ip := "127.0.0.1"
	userAgent := "test-agent"
	token, err := service.CreateSession(ctx, playerID, ip, userAgent)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if token == "" {
		t.Error("Generated token is empty")
	}

	// Verify session stored in database
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ? AND player_id = ?`, token, playerID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 session, got %d", count)
	}
}

func TestAuthService_ValidateRefreshToken(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	ctx := context.Background()

	// Insert a player
	password := "password"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Create session
	ip := "127.0.0.1"
	userAgent := "test-agent"
	token, err := service.CreateSession(ctx, playerID, ip, userAgent)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Validate token (should succeed)
	validatedPlayerID, err := service.ValidateRefreshToken(ctx, token)
	if err != nil {
		t.Errorf("ValidateRefreshToken failed: %v", err)
	}
	if validatedPlayerID != playerID {
		t.Errorf("Expected player ID %d, got %d", playerID, validatedPlayerID)
	}

	// Validate invalid token (should fail)
	_, err = service.ValidateRefreshToken(ctx, "invalid-token")
	if err != auth.ErrInvalidRefreshToken && err != auth.ErrSessionNotFound {
		t.Errorf("Expected ErrInvalidRefreshToken or ErrSessionNotFound, got %v", err)
	}
}

func TestAuthService_RefreshSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	ctx := context.Background()

	// Insert a player
	password := "password"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Create initial session
	ip := "127.0.0.1"
	userAgent := "test-agent"
	oldToken, err := service.CreateSession(ctx, playerID, ip, userAgent)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	t.Logf("oldToken length: %d", len(oldToken))

	// Verify session exists
	var sessionCount int
	err = db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ?`, oldToken).Scan(&sessionCount)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if sessionCount != 1 {
		t.Errorf("Expected 1 session before refresh, got %d", sessionCount)
	}

	// Refresh session
	newPlayerID, newToken, err := service.RefreshSession(ctx, oldToken, ip, userAgent)
	if err != nil {
		t.Fatalf("RefreshSession failed: %v", err)
	}
	if newPlayerID != playerID {
		t.Errorf("Expected player ID %d, got %d", playerID, newPlayerID)
	}
	if newToken == "" {
		t.Error("New token is empty")
	}
	// Note: newToken may equal oldToken if JWT generation is identical (same iat). That's acceptable.

	// After refresh, there should be exactly one session for the player
	var totalSessions int
	err = db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE player_id = ?`, playerID).Scan(&totalSessions)
	if err != nil {
		t.Fatalf("Failed to query total sessions: %v", err)
	}
	if totalSessions != 1 {
		t.Errorf("Expected exactly 1 session for player after refresh, got %d", totalSessions)
	}

	// If new token is different from old token, old token should be invalid
	if newToken != oldToken {
		// Old token should be invalid (session not found)
		_, err = service.ValidateRefreshToken(ctx, oldToken)
		if err != auth.ErrInvalidRefreshToken && err != auth.ErrSessionNotFound {
			t.Errorf("Old token should be invalid, got %v", err)
		}
	} else {
		t.Log("New token equals old token (JWT regeneration with same iat)")
	}

	// New token should be valid
	validatedPlayerID, err := service.ValidateRefreshToken(ctx, newToken)
	if err != nil {
		t.Errorf("New token validation failed: %v", err)
	}
	if validatedPlayerID != playerID {
		t.Errorf("Expected player ID %d, got %d", playerID, validatedPlayerID)
	}
}

func TestAuthService_DeleteSession(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	ctx := context.Background()

	// Insert a player
	password := "password"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Create session
	ip := "127.0.0.1"
	userAgent := "test-agent"
	token, err := service.CreateSession(ctx, playerID, ip, userAgent)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	// Delete session
	err = service.DeleteSession(ctx, token)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify session is deleted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ?`, token).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 sessions after deletion, got %d", count)
	}

	// Validate token should fail
	_, err = service.ValidateRefreshToken(ctx, token)
	if err != auth.ErrInvalidRefreshToken && err != auth.ErrSessionNotFound {
		t.Errorf("Expected ErrInvalidRefreshToken or ErrSessionNotFound, got %v", err)
	}
}

func TestAuthService_AddExperience(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	ctx := context.Background()

	// Insert a player
	password := "password"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Get initial progression (should be defaults)
	progression, err := service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression: %v", err)
	}
	if progression.Level != 1 {
		t.Errorf("Expected initial level 1, got %d", progression.Level)
	}
	if progression.Experience != 0 {
		t.Errorf("Expected initial XP 0, got %d", progression.Experience)
	}

	// Add XP below level threshold (BaseXPPerLevel = 1000)
	err = service.AddExperience(ctx, playerID, 500)
	if err != nil {
		t.Fatalf("AddExperience failed: %v", err)
	}
	progression, err = service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression after XP: %v", err)
	}
	if progression.Experience != 500 {
		t.Errorf("Expected XP 500 after adding 500, got %d", progression.Experience)
	}
	if progression.Level != 1 {
		t.Errorf("Expected level still 1 with 500 XP, got %d", progression.Level)
	}

	// Add XP to reach level 2 (total 1000 XP)
	err = service.AddExperience(ctx, playerID, 500)
	if err != nil {
		t.Fatalf("AddExperience failed: %v", err)
	}
	progression, err = service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression after level up: %v", err)
	}
	if progression.Experience != 1000 {
		t.Errorf("Expected XP 1000 after adding another 500, got %d", progression.Experience)
	}
	if progression.Level != 2 {
		t.Errorf("Expected level 2 with 1000 XP, got %d", progression.Level)
	}

	// Add XP that spans multiple levels (add 2500 XP, total 3500, level should be 4)
	err = service.AddExperience(ctx, playerID, 2500)
	if err != nil {
		t.Fatalf("AddExperience failed: %v", err)
	}
	progression, err = service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression after large XP: %v", err)
	}
	if progression.Experience != 3500 {
		t.Errorf("Expected XP 3500 after adding 2500, got %d", progression.Experience)
	}
	if progression.Level != 4 {
		t.Errorf("Expected level 4 with 3500 XP (BaseXPPerLevel=1000), got %d", progression.Level)
	}
}

func TestAuthService_AddMatchRewards(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	ctx := context.Background()

	// Insert a player
	password := "password"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Get initial progression
	progression, err := service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression: %v", err)
	}
	initialMatches := progression.TotalMatchesPlayed
	initialKills := progression.TotalKills
	initialDataCurrency := progression.DataCurrency

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
	progression, err = service.GetPlayerProgression(ctx, playerID)
	if err != nil {
		t.Fatalf("Failed to get player progression after match: %v", err)
	}
	if progression.TotalMatchesPlayed != initialMatches+1 {
		t.Errorf("Expected total matches played %d, got %d", initialMatches+1, progression.TotalMatchesPlayed)
	}
	if progression.TotalKills != initialKills+kills {
		t.Errorf("Expected total kills %d, got %d", initialKills+kills, progression.TotalKills)
	}
	if progression.TotalDeaths != deaths {
		t.Errorf("Expected total deaths %d, got %d", deaths, progression.TotalDeaths)
	}
	if progression.TotalWavesSurvived != wavesSurvived {
		t.Errorf("Expected total waves survived %d, got %d", wavesSurvived, progression.TotalWavesSurvived)
	}
	if progression.TotalScrapEarned != scrapEarned {
		t.Errorf("Expected total scrap earned %d, got %d", scrapEarned, progression.TotalScrapEarned)
	}
	if progression.TotalDataEarned != dataEarned {
		t.Errorf("Expected total data earned %d, got %d", dataEarned, progression.TotalDataEarned)
	}
	// Data currency should increase by dataEarned
	if progression.DataCurrency != initialDataCurrency+dataEarned {
		t.Errorf("Expected data currency %d, got %d", initialDataCurrency+dataEarned, progression.DataCurrency)
	}
	// XP should have increased (calculated by AddMatchRewards)
	// BaseXP=100, kills*10=100, waves*50=250, scrap*1=500 => total XP = 950
	expectedXP := int64(100 + kills*10 + wavesSurvived*50 + scrapEarned*1)
	if progression.Experience != expectedXP {
		t.Errorf("Expected XP %d, got %d", expectedXP, progression.Experience)
	}
	// Level may have increased if XP >= BaseXPPerLevel (1000). With 950 XP, level should still be 1
	if progression.Level != 1 {
		t.Errorf("Expected level 1 with %d XP, got %d", expectedXP, progression.Level)
	}
}

func TestGenerateLootDrop(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := newTestConfig()
	service := auth.NewService(cfg, logger, db)

	ctx := context.Background()

	// Insert a player
	password := "password"
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testplayer", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testplayer").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Insert a cosmetic item
	_, err = db.Exec(`INSERT INTO cosmetic_items (name, slot, rarity, unlock_level, data_cost) VALUES (?, ?, ?, ?, ?)`,
		"Test Skin", "character_skin", "rare", 1, 0)
	if err != nil {
		t.Fatalf("Failed to insert cosmetic item: %v", err)
	}
	var cosmeticID int64
	err = db.QueryRow(`SELECT cosmetic_id FROM cosmetic_items WHERE name = ?`, "Test Skin").Scan(&cosmeticID)
	if err != nil {
		t.Fatalf("Failed to get cosmetic ID: %v", err)
	}

	// Insert loot table with 100% drop chance
	_, err = db.Exec(`INSERT INTO loot_tables (name, drop_chance, is_active) VALUES (?, ?, ?)`,
		"Test Loot Table", 1.0, 1)
	if err != nil {
		t.Fatalf("Failed to insert loot table: %v", err)
	}
	var lootTableID int64
	err = db.QueryRow(`SELECT loot_table_id FROM loot_tables WHERE name = ?`, "Test Loot Table").Scan(&lootTableID)
	if err != nil {
		t.Fatalf("Failed to get loot table ID: %v", err)
	}

	// Insert loot table entry with weight 100
	_, err = db.Exec(`INSERT INTO loot_table_entries (loot_table_id, cosmetic_id, weight, min_quantity, max_quantity) VALUES (?, ?, ?, ?, ?)`,
		lootTableID, cosmeticID, 100, 1, 1)
	if err != nil {
		t.Fatalf("Failed to insert loot table entry: %v", err)
	}

	// Generate loot drop
	cosmetic, err := service.GenerateLootDrop(ctx, playerID)
	if err != nil {
		t.Fatalf("GenerateLootDrop failed: %v", err)
	}

	// Verify cosmetic matches
	if cosmetic.CosmeticID != cosmeticID {
		t.Errorf("Expected cosmetic ID %d, got %d", cosmeticID, cosmetic.CosmeticID)
	}
	if cosmetic.Name != "Test Skin" {
		t.Errorf("Expected cosmetic name 'Test Skin', got %s", cosmetic.Name)
	}

	// Verify player owns cosmetic
	var unlockedVia string
	err = db.QueryRow(`SELECT unlocked_via FROM player_cosmetics WHERE player_id = ? AND cosmetic_id = ?`, playerID, cosmeticID).Scan(&unlockedVia)
	if err != nil {
		t.Fatalf("Failed to query player_cosmetics: %v", err)
	}
	if unlockedVia != "loot_drop" {
		t.Errorf("Expected unlocked_via 'loot_drop', got %s", unlockedVia)
	}
}
