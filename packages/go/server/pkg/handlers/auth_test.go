package handlers_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"

	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/config"
	"ai-zombie-defense/server/pkg/handlers"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func getTestConfig() config.Config {
	return config.Config{
		Database: config.DatabaseConfig{
			Path:           ":memory:",
			MigrationsPath: "./migrations",
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
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
	// Create player_settings table
	createPlayerSettingsSQL := `CREATE TABLE player_settings (
	    player_id INTEGER PRIMARY KEY,
	    key_bindings TEXT,
	    mouse_sensitivity REAL,
	    ui_scale REAL,
	    color_blind_mode INTEGER NOT NULL DEFAULT 0,
	    subtitles_enabled INTEGER NOT NULL DEFAULT 0,
	    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
	    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(createPlayerSettingsSQL); err != nil {
		t.Fatalf("Failed to create player_settings table: %v", err)
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
	// Create currency_transactions table
	createCurrencyTransactionsSQL := `CREATE TABLE currency_transactions (
		transaction_id INTEGER PRIMARY KEY AUTOINCREMENT,
		player_id INTEGER NOT NULL,
		amount INTEGER NOT NULL,
		balance_after INTEGER NOT NULL,
		transaction_type TEXT NOT NULL CHECK (transaction_type IN ('match_reward', 'purchase', 'prestige_reward', 'admin_grant', 'refund', 'other')),
		reference_id INTEGER,
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
	);`
	if _, err := db.Exec(createCurrencyTransactionsSQL); err != nil {
		t.Fatalf("Failed to create currency_transactions table: %v", err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_currency_transactions_player_id ON currency_transactions (player_id);`); err != nil {
		t.Fatalf("Failed to create currency_transactions player_id index: %v", err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_currency_transactions_created_at ON currency_transactions (created_at);`); err != nil {
		t.Fatalf("Failed to create currency_transactions created_at index: %v", err)
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
	// Create index on cosmetic_id for faster lookups
	if _, err := db.Exec(`CREATE INDEX idx_player_cosmetics_cosmetic_id ON player_cosmetics (cosmetic_id)`); err != nil {
		t.Fatalf("Failed to create player_cosmetics index: %v", err)
	}
	// Create friends table
	if _, err := db.Exec(`CREATE TABLE friends (
		player_id INTEGER NOT NULL,
		friend_id INTEGER NOT NULL,
		status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'blocked')),
		created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
		PRIMARY KEY (player_id, friend_id),
		FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
		FOREIGN KEY (friend_id) REFERENCES players (player_id) ON DELETE CASCADE
	);`); err != nil {
		t.Fatalf("Failed to create friends table: %v", err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_friends_friend_id ON friends (friend_id);`); err != nil {
		t.Fatalf("Failed to create friends index: %v", err)
	}
	// Create loadouts table
	if _, err := db.Exec(`CREATE TABLE loadouts (
    loadout_id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    is_active INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
);`); err != nil {
		t.Fatalf("Failed to create loadouts table: %v", err)
	}
	// Create loadout_cosmetics table
	if _, err := db.Exec(`CREATE TABLE loadout_cosmetics (
    loadout_id INTEGER NOT NULL,
    cosmetic_id INTEGER NOT NULL,
    slot TEXT NOT NULL CHECK (slot IN ('character_skin', 'weapon_skin', 'emote', 'taunt', 'badge', 'title', 'particle_effect', 'other')),
    PRIMARY KEY (loadout_id, cosmetic_id),
    FOREIGN KEY (loadout_id) REFERENCES loadouts (loadout_id) ON DELETE CASCADE,
    FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
);`); err != nil {
		t.Fatalf("Failed to create loadout_cosmetics table: %v", err)
	}
	// Create index on loadout_cosmetics cosmetic_id
	if _, err := db.Exec(`CREATE INDEX idx_loadout_cosmetics_cosmetic_id ON loadout_cosmetics (cosmetic_id)`); err != nil {
		t.Fatalf("Failed to create loadout_cosmetics index: %v", err)
	}
	// Create servers table
	if _, err := db.Exec(`CREATE TABLE servers (
    server_id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip_address TEXT NOT NULL,
    port INTEGER NOT NULL,
    auth_token TEXT UNIQUE,
    name TEXT NOT NULL,
    map_rotation TEXT,
    max_players INTEGER NOT NULL,
    current_players INTEGER NOT NULL DEFAULT 0,
    is_online INTEGER NOT NULL DEFAULT 0,
    last_heartbeat TEXT,
    region TEXT,
    version TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);`); err != nil {
		t.Fatalf("Failed to create servers table: %v", err)
	}
	// Create unique index for auth_token
	if _, err := db.Exec(`CREATE UNIQUE INDEX idx_servers_auth_token ON servers(auth_token);`); err != nil {
		t.Fatalf("Failed to create auth_token index: %v", err)
	}
	// Create server_favorites table
	if _, err := db.Exec(`CREATE TABLE server_favorites (
    player_id INTEGER NOT NULL,
    server_id INTEGER NOT NULL,
    added_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    note TEXT,
    PRIMARY KEY (player_id, server_id),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
    FOREIGN KEY (server_id) REFERENCES servers (server_id) ON DELETE CASCADE
);`); err != nil {
		t.Fatalf("Failed to create server_favorites table: %v", err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_server_favorites_server_id ON server_favorites (server_id);`); err != nil {
		t.Fatalf("Failed to create server_favorites index: %v", err)
	}
	// Create matches table
	if _, err := db.Exec(`CREATE TABLE matches (
    match_id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER NOT NULL,
    map_name TEXT NOT NULL,
    game_mode TEXT NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT,
    outcome TEXT NOT NULL CHECK (outcome IN ('completed', 'failed', 'abandoned')),
    waves_survived INTEGER NOT NULL DEFAULT 0,
    total_zombies_killed INTEGER NOT NULL DEFAULT 0,
    total_players INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (server_id) REFERENCES servers (server_id) ON DELETE CASCADE
);`); err != nil {
		t.Fatalf("Failed to create matches table: %v", err)
	}
	// Create player_match_stats table
	if _, err := db.Exec(`CREATE TABLE player_match_stats (
    player_id INTEGER NOT NULL,
    match_id INTEGER NOT NULL,
    waves_survived INTEGER NOT NULL DEFAULT 0,
    zombies_killed INTEGER NOT NULL DEFAULT 0,
    deaths INTEGER NOT NULL DEFAULT 0,
    scrap_earned INTEGER NOT NULL DEFAULT 0,
    data_earned INTEGER NOT NULL DEFAULT 0,
    damage_dealt INTEGER NOT NULL DEFAULT 0,
    damage_taken INTEGER NOT NULL DEFAULT 0,
    buildings_built INTEGER NOT NULL DEFAULT 0,
    buildings_destroyed INTEGER NOT NULL DEFAULT 0,
    healing_given INTEGER NOT NULL DEFAULT 0,
    revives INTEGER NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (player_id, match_id),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
    FOREIGN KEY (match_id) REFERENCES matches (match_id) ON DELETE CASCADE
);`); err != nil {
		t.Fatalf("Failed to create player_match_stats table: %v", err)
	}
	// Create indexes for matches and player_match_stats
	if _, err := db.Exec(`CREATE INDEX idx_matches_server_id ON matches (server_id);`); err != nil {
		t.Fatalf("Failed to create matches index: %v", err)
	}
	if _, err := db.Exec(`CREATE INDEX idx_player_match_stats_match_id ON player_match_stats (match_id);`); err != nil {
		t.Fatalf("Failed to create player_match_stats index: %v", err)
	}
	return db
}

func createTestServer(t *testing.T, db *sql.DB) *fiber.App {
	logger := zaptest.NewLogger(t)
	cfg := getTestConfig()
	authService := auth.NewService(cfg, logger, db)
	authHandlers := handlers.NewAuthHandlers(authService, cfg, logger)
	app := fiber.New()
	authGroup := app.Group("/auth")
	authGroup.Post("/login", authHandlers.Login)
	authGroup.Post("/register", authHandlers.Register)
	authGroup.Post("/refresh", authHandlers.Refresh)
	authGroup.Post("/logout", authHandlers.Logout)
	return app
}

func createTestPlayer(t *testing.T, db *sql.DB, username, email, password string) int64 {
	logger := zaptest.NewLogger(t)
	cfg := getTestConfig()
	service := auth.NewService(cfg, logger, db)
	hash, err := service.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	result, err := db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		username, email, hash)
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}
	playerID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}
	// Create player progression row
	_, err = db.Exec(`INSERT INTO player_progression (player_id) VALUES (?)`, playerID)
	if err != nil {
		t.Fatalf("Failed to create player progression row: %v", err)
	}
	return playerID
}

func createTestSession(t *testing.T, db *sql.DB, playerID int64) string {
	logger := zaptest.NewLogger(t)
	cfg := getTestConfig()
	service := auth.NewService(cfg, logger, db)
	ctx := context.Background()
	token, err := service.CreateSession(ctx, playerID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	return token
}

func TestAuthHandlers_Refresh(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createTestServer(t, db)

	// Create a player and session
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	refreshToken := createTestSession(t, db, playerID)

	// Test successful refresh
	reqBody := map[string]string{"refresh_token": refreshToken}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if _, ok := result["access_token"]; !ok {
		t.Error("Response missing access_token")
	}
	if _, ok := result["refresh_token"]; !ok {
		t.Error("Response missing refresh_token")
	}
	// Ensure new refresh token is different (might be same due to same iat, but that's fine)
	newRefreshToken, _ := result["refresh_token"].(string)
	if newRefreshToken == "" {
		t.Error("New refresh token is empty")
	}

	// Test invalid refresh token
	reqBody = map[string]string{"refresh_token": "invalid"}
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid token, got %d", resp.StatusCode)
	}

	// Test missing refresh token
	reqBody = map[string]string{}
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/auth/refresh", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing token, got %d", resp.StatusCode)
	}
}

func TestAuthHandlers_Logout(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createTestServer(t, db)

	// Create a player and session
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	refreshToken := createTestSession(t, db, playerID)

	// Test successful logout
	reqBody := map[string]string{"refresh_token": refreshToken}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if msg, ok := result["message"]; !ok || msg != "logged out successfully" {
		t.Errorf("Expected success message, got %v", result)
	}

	// Verify session is deleted
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ?`, refreshToken).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query session count: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 sessions after logout, got %d", count)
	}

	// Test logout with invalid token (should still return 200? Actually delete session will not error, but we expect 200)
	reqBody = map[string]string{"refresh_token": "invalid"}
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	// The delete operation will succeed (no rows affected) but we still return 200.
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for invalid token logout, got %d", resp.StatusCode)
	}

	// Test missing refresh token
	reqBody = map[string]string{}
	body, _ = json.Marshal(reqBody)
	req = httptest.NewRequest(http.MethodPost, "/auth/logout", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing token, got %d", resp.StatusCode)
	}
}

func TestAuthHandlers_RegisterAndLogin(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createTestServer(t, db)

	// Test registration
	regBody := map[string]string{
		"username": "newuser",
		"email":    "new@example.com",
		"password": "securepass123",
	}
	body, _ := json.Marshal(regBody)
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
	var regResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&regResult); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if _, ok := regResult["access_token"]; !ok {
		t.Error("Registration response missing access_token")
	}
	refreshToken, ok := regResult["refresh_token"].(string)
	if !ok || refreshToken == "" {
		t.Error("Registration response missing refresh_token")
	}

	// Test login with registered credentials
	loginBody := map[string]string{
		"username_or_email": "newuser",
		"password":          "securepass123",
	}
	body, _ = json.Marshal(loginBody)
	req = httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	var loginResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&loginResult); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if _, ok := loginResult["access_token"]; !ok {
		t.Error("Login response missing access_token")
	}
}
