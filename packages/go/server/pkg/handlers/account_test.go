package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/server"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func createTestAccessToken(t *testing.T, db *sql.DB, playerID int64) string {
	logger := zaptest.NewLogger(t)
	cfg := getTestConfig()
	service := auth.NewService(cfg, logger, db)
	token, err := service.GenerateAccessToken(playerID)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}
	return token
}

func createFullTestServer(t *testing.T, db *sql.DB) *fiber.App {
	logger := zaptest.NewLogger(t)
	cfg := getTestConfig()
	srv := server.New(cfg, logger, db)
	return srv.App()
}

func TestAccountHandlers_GetProfile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	// Test successful profile retrieval
	req := httptest.NewRequest(http.MethodGet, "/account/profile", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
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
	if result["player_id"] != float64(playerID) {
		t.Errorf("Expected player_id %d, got %v", playerID, result["player_id"])
	}
	if result["username"] != "testuser" {
		t.Errorf("Expected username testuser, got %v", result["username"])
	}
	if result["email"] != "test@example.com" {
		t.Errorf("Expected email test@example.com, got %v", result["email"])
	}
	if _, ok := result["created_at"]; !ok {
		t.Error("Response missing created_at")
	}
	// last_login_at may be null, that's fine
	if _, ok := result["is_banned"]; !ok {
		t.Error("Response missing is_banned")
	}

	// Test without authorization header
	req = httptest.NewRequest(http.MethodGet, "/account/profile", nil)
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing token, got %d", resp.StatusCode)
	}

	// Test with invalid token
	req = httptest.NewRequest(http.MethodGet, "/account/profile", nil)
	req.Header.Set("Authorization", "Bearer invalid")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid token, got %d", resp.StatusCode)
	}
}

func TestAccountHandlers_UpdateProfile(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	// Test successful profile update
	updateBody := map[string]string{
		"username": "newusername",
		"email":    "newemail@example.com",
	}
	body, _ := json.Marshal(updateBody)
	req := httptest.NewRequest(http.MethodPut, "/account/profile", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
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
	if msg, ok := result["message"]; !ok || msg != "profile updated successfully" {
		t.Errorf("Expected success message, got %v", result)
	}

	// Verify changes in database
	var username, email string
	err = db.QueryRow(`SELECT username, email FROM players WHERE player_id = ?`, playerID).Scan(&username, &email)
	if err != nil {
		t.Fatalf("Failed to query updated player: %v", err)
	}
	if username != "newusername" {
		t.Errorf("Expected username newusername, got %s", username)
	}
	if email != "newemail@example.com" {
		t.Errorf("Expected email newemail@example.com, got %s", email)
	}

	// Test duplicate username (create another player)
	otherPlayerID := createTestPlayer(t, db, "otheruser", "other@example.com", "password")
	otherAccessToken := createTestAccessToken(t, db, otherPlayerID)
	// Try to update other player's username to already taken "newusername"
	dupBody := map[string]string{
		"username": "newusername", // already taken by first player
		"email":    "other@example.com",
	}
	body, _ = json.Marshal(dupBody)
	req = httptest.NewRequest(http.MethodPut, "/account/profile", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+otherAccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate username, got %d", resp.StatusCode)
	}
	var errResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&errResult); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResult["error"] != "username already exists" {
		t.Errorf("Expected error 'username already exists', got %v", errResult["error"])
	}

	// Test duplicate email
	dupBody = map[string]string{
		"username": "otheruser2",
		"email":    "newemail@example.com", // already taken
	}
	body, _ = json.Marshal(dupBody)
	req = httptest.NewRequest(http.MethodPut, "/account/profile", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+otherAccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate email, got %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResult); err != nil {
		t.Fatalf("Failed to decode error response: %v", err)
	}
	if errResult["error"] != "email already exists" {
		t.Errorf("Expected error 'email already exists', got %v", errResult["error"])
	}

	// Test missing required fields
	missingBody := map[string]string{
		"username": "",
		"email":    "valid@example.com",
	}
	body, _ = json.Marshal(missingBody)
	req = httptest.NewRequest(http.MethodPut, "/account/profile", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing username, got %d", resp.StatusCode)
	}

	// Test invalid request body
	req = httptest.NewRequest(http.MethodPut, "/account/profile", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", resp.StatusCode)
	}
}
