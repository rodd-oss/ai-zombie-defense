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

func TestAccountHandlers_GetSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	// Test GET settings when no settings exist (should return defaults)
	req := httptest.NewRequest(http.MethodGet, "/account/settings", nil)
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
	// Check defaults
	if result["color_blind_mode"] != float64(0) {
		t.Errorf("Expected color_blind_mode 0, got %v", result["color_blind_mode"])
	}
	if result["subtitles_enabled"] != float64(0) {
		t.Errorf("Expected subtitles_enabled 0, got %v", result["subtitles_enabled"])
	}
	// nullable fields should be null or missing
	if val, ok := result["key_bindings"]; ok && val != nil {
		t.Errorf("Expected key_bindings null or missing, got %v", val)
	}
	if val, ok := result["mouse_sensitivity"]; ok && val != nil {
		t.Errorf("Expected mouse_sensitivity null or missing, got %v", val)
	}
	if val, ok := result["ui_scale"]; ok && val != nil {
		t.Errorf("Expected ui_scale null or missing, got %v", val)
	}
	if _, ok := result["created_at"]; !ok {
		t.Error("Response missing created_at")
	}
	if _, ok := result["updated_at"]; !ok {
		t.Error("Response missing updated_at")
	}
}

func TestAccountHandlers_UpdateSettings(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

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
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if msg, ok := result["message"]; !ok || msg != "settings updated successfully" {
		t.Errorf("Expected success message, got %v", result)
	}

	// Verify changes in database
	var keyBindings sql.NullString
	var mouseSensitivity sql.NullFloat64
	var uiScale sql.NullFloat64
	var colorBlindMode, subtitlesEnabled int64
	err = db.QueryRow(`SELECT key_bindings, mouse_sensitivity, ui_scale, color_blind_mode, subtitles_enabled FROM player_settings WHERE player_id = ?`, playerID).Scan(&keyBindings, &mouseSensitivity, &uiScale, &colorBlindMode, &subtitlesEnabled)
	if err != nil {
		t.Fatalf("Failed to query updated settings: %v", err)
	}
	if !keyBindings.Valid || keyBindings.String != "WASD" {
		t.Errorf("Expected key_bindings WASD, got %v", keyBindings)
	}
	if !mouseSensitivity.Valid || mouseSensitivity.Float64 != 1.5 {
		t.Errorf("Expected mouse_sensitivity 1.5, got %v", mouseSensitivity)
	}
	if !uiScale.Valid || uiScale.Float64 != 1.0 {
		t.Errorf("Expected ui_scale 1.0, got %v", uiScale)
	}
	if colorBlindMode != 1 {
		t.Errorf("Expected color_blind_mode 1, got %d", colorBlindMode)
	}
	if subtitlesEnabled != 1 {
		t.Errorf("Expected subtitles_enabled 1, got %d", subtitlesEnabled)
	}

	// Test partial update with null values
	updateBody2 := map[string]interface{}{
		"key_bindings":      nil,
		"mouse_sensitivity": nil,
		"ui_scale":          nil,
		"color_blind_mode":  0,
		"subtitles_enabled": 0,
	}
	body, _ = json.Marshal(updateBody2)
	req = httptest.NewRequest(http.MethodPut, "/account/settings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify null values
	err = db.QueryRow(`SELECT key_bindings, mouse_sensitivity, ui_scale, color_blind_mode, subtitles_enabled FROM player_settings WHERE player_id = ?`, playerID).Scan(&keyBindings, &mouseSensitivity, &uiScale, &colorBlindMode, &subtitlesEnabled)
	if err != nil {
		t.Fatalf("Failed to query updated settings: %v", err)
	}
	if keyBindings.Valid {
		t.Errorf("Expected key_bindings null, got %v", keyBindings)
	}
	if mouseSensitivity.Valid {
		t.Errorf("Expected mouse_sensitivity null, got %v", mouseSensitivity)
	}
	if uiScale.Valid {
		t.Errorf("Expected ui_scale null, got %v", uiScale)
	}
	if colorBlindMode != 0 {
		t.Errorf("Expected color_blind_mode 0, got %d", colorBlindMode)
	}
	if subtitlesEnabled != 0 {
		t.Errorf("Expected subtitles_enabled 0, got %d", subtitlesEnabled)
	}

	// Test missing authorization header
	req = httptest.NewRequest(http.MethodPut, "/account/settings", bytes.NewReader(body))
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing token, got %d", resp.StatusCode)
	}

	// Test invalid token
	req = httptest.NewRequest(http.MethodPut, "/account/settings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer invalid")
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for invalid token, got %d", resp.StatusCode)
	}

	// Test invalid request body
	req = httptest.NewRequest(http.MethodPut, "/account/settings", bytes.NewReader([]byte("invalid json")))
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

func TestAccountHandlers_GetProgression(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	// Test both /account/progression and /progression routes
	paths := []string{"/account/progression", "/progression"}
	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
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
			if result["level"] != float64(1) {
				t.Errorf("Expected level 1, got %v", result["level"])
			}
			if result["experience"] != float64(0) {
				t.Errorf("Expected experience 0, got %v", result["experience"])
			}
			if result["prestige_level"] != float64(0) {
				t.Errorf("Expected prestige_level 0, got %v", result["prestige_level"])
			}
			if result["data_currency"] != float64(0) {
				t.Errorf("Expected data_currency 0, got %v", result["data_currency"])
			}
			if _, ok := result["updated_at"]; !ok {
				t.Error("Response missing updated_at")
			}
		})
	}

	// Test without authorization header
	t.Run("missing token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/account/progression", nil)
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for missing token, got %d", resp.StatusCode)
		}
	})

	// Test with invalid token
	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/account/progression", nil)
		req.Header.Set("Authorization", "Bearer invalid")
		resp, err := app.Test(req, -1)
		if err != nil {
			t.Fatalf("Failed to make request: %v", err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Errorf("Expected status 401 for invalid token, got %d", resp.StatusCode)
		}
	})
}
