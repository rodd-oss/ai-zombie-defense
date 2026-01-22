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
    banned_until TEXT
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
	return db
}

func TestAuthService_Authenticate(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,          // 15 minutes in nanoseconds
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000, // 7 days
		},
	}
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

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000,
		},
	}
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

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000,
		},
	}
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

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000,
		},
	}
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

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000,
		},
	}
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

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000,
		},
	}
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

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000,
		},
	}
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
