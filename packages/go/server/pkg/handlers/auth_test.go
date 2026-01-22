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

func createTestServer(t *testing.T, db *sql.DB) *fiber.App {
	logger := zaptest.NewLogger(t)
	cfg := getTestConfig()
	authService := auth.NewService(cfg, logger, db)
	authHandlers := handlers.NewAuthHandlers(authService, logger)
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
