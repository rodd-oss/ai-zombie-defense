package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ai-zombie-defense/server/internal/api/gateway"
	"ai-zombie-defense/server/internal/testutils"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func createFullTestServer(t *testing.T, db *sql.DB) *fiber.App {
	logger := zaptest.NewLogger(t)
	cfg := testutils.GetTestConfig()
	gw := gateway.NewAPIGateway(cfg, logger, db)
	return gw.Router()
}

func TestAccountHandlers_GetProfile(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)

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
}

func TestAccountHandlers_UpdateProfile(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)

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

	// Verify changes in database
	var username, email string
	err = db.QueryRow(`SELECT username, email FROM players WHERE player_id = ?`, playerID).Scan(&username, &email)
	if err != nil {
		t.Fatalf("Failed to query updated player: %v", err)
	}
	if username != "newusername" {
		t.Errorf("Expected username newusername, got %s", username)
	}
}

func TestAccountHandlers_GetSettings(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)

	// Test GET settings
	req := httptest.NewRequest(http.MethodGet, "/account/settings", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestAccountHandlers_UpdateSettings(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)

	// Test successful settings update
	updateBody := map[string]interface{}{
		"key_bindings":      "WASD",
		"mouse_sensitivity": 1.5,
		"ui_scale":          1.0,
		"color_blind_mode":  1,
		"subtitles_enabled": 1,
	}
	body, _ := json.Marshal(updateBody)
	req := httptest.NewRequest(http.MethodPut, "/account/settings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
