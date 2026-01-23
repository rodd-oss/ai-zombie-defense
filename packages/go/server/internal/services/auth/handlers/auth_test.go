package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-zombie-defense/server/internal/services/auth"
	"ai-zombie-defense/server/internal/services/auth/handlers"
	"ai-zombie-defense/server/internal/testutils"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func createTestServer(t *testing.T, db *sql.DB) *fiber.App {
	logger := zaptest.NewLogger(t)
	cfg := testutils.GetTestConfig()
	authService := auth.NewAuthService(cfg, logger, db)
	authHandlers := handlers.NewAuthHandlers(authService, cfg, logger)
	app := fiber.New()
	authGroup := app.Group("/auth")
	authGroup.Post("/login", authHandlers.Login)
	authGroup.Post("/register", authHandlers.Register)
	authGroup.Post("/refresh", authHandlers.Refresh)
	authGroup.Post("/logout", authHandlers.Logout)
	return app
}

func TestAuthHandlers_Refresh(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createTestServer(t, db)

	// Create a player and session
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	refreshToken := testutils.CreateTestSession(t, db, playerID)

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
}

func TestAuthHandlers_Logout(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createTestServer(t, db)

	// Create a player and session
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	refreshToken := testutils.CreateTestSession(t, db, playerID)

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
}

func TestAuthHandlers_RegisterAndLogin(t *testing.T) {
	db := testutils.SetupTestDB(t)
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
