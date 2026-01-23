package auth_test

import (
	"context"
	"database/sql"
	"testing"

	"ai-zombie-defense/server/internal/services/account"
	"ai-zombie-defense/server/internal/services/auth"
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

func TestAuthService_Authenticate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	cfg := newTestConfig()
	service := auth.NewAuthService(cfg, logger, dbConn)

	ctx := context.Background()
	username := "testuser"
	email := "test@example.com"
	password := "securepassword123"

	// Register a player first
	_, err := service.RegisterPlayer(ctx, username, email, password)
	if err != nil {
		t.Fatalf("Failed to register player: %v", err)
	}

	// Test successful authentication with username
	player, err := service.Authenticate(ctx, username, password)
	if err != nil {
		t.Errorf("Authentication with username failed: %v", err)
	}
	if player == nil {
		t.Fatal("Player is nil but error is nil")
	}
	if player.Username != username {
		t.Errorf("Expected username %s, got %s", username, player.Username)
	}

	// Test successful authentication with email
	player, err = service.Authenticate(ctx, email, password)
	if err != nil {
		t.Errorf("Authentication with email failed: %v", err)
	}
	if player == nil {
		t.Fatal("Player is nil but error is nil")
	}
	if player.Email != email {
		t.Errorf("Expected email %s, got %s", email, player.Email)
	}

	// Test invalid password
	_, err = service.Authenticate(ctx, username, "wrongpassword")
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
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	cfg := newTestConfig()
	service := auth.NewAuthService(cfg, logger, dbConn)

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
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	cfg := newTestConfig()
	service := auth.NewAuthService(cfg, logger, dbConn)

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
	})

	t.Run("duplicate username", func(t *testing.T) {
		_, _ = service.RegisterPlayer(ctx, "user1", "email1@example.com", "pass")
		_, err := service.RegisterPlayer(ctx, "user1", "email2@example.com", "pass")
		if err != account.ErrDuplicateUsername {
			t.Errorf("Expected account.ErrDuplicateUsername, got %v", err)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		_, _ = service.RegisterPlayer(ctx, "user2", "email2@example.com", "pass")
		_, err := service.RegisterPlayer(ctx, "user3", "email2@example.com", "pass")
		if err != account.ErrDuplicateEmail {
			t.Errorf("Expected account.ErrDuplicateEmail, got %v", err)
		}
	})
}

func TestAuthService_Sessions(t *testing.T) {
	logger := zaptest.NewLogger(t)
	dbConn := setupTestDB(t)
	defer dbConn.Close()

	cfg := newTestConfig()
	service := auth.NewAuthService(cfg, logger, dbConn)

	ctx := context.Background()
	playerID := int64(1)
	// Create a player record for foreign key constraint
	_, err := dbConn.Exec(`INSERT INTO players (player_id, username, email, password_hash) VALUES (?, ?, ?, ?)`,
		playerID, "testuser", "test@example.com", "hash")
	if err != nil {
		t.Fatalf("Failed to insert player: %v", err)
	}

	ip := "127.0.0.1"
	ua := "test-agent"

	// Create session
	token, err := service.CreateSession(ctx, playerID, ip, ua)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if token == "" {
		t.Fatal("Token is empty")
	}

	// Refresh session
	newPlayerID, newToken, err := service.RefreshSession(ctx, token, ip, ua)
	if err != nil {
		t.Fatalf("RefreshSession failed: %v", err)
	}
	if newPlayerID != playerID {
		t.Errorf("Expected player ID %d, got %d", playerID, newPlayerID)
	}
	if newToken == "" {
		t.Fatal("New token is empty")
	}

	// Delete session
	err = service.DeleteSession(ctx, newToken)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	// Verify session deleted
	var count int
	err = dbConn.QueryRow(`SELECT COUNT(*) FROM sessions WHERE token = ?`, newToken).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query session: %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 sessions, got %d", count)
	}
}
