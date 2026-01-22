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

func createTestServerRow(t *testing.T, db *sql.DB) int64 {
	result, err := db.Exec(`INSERT INTO servers (ip_address, port, name, max_players) VALUES (?, ?, ?, ?)`,
		"127.0.0.1", 7777, "Test Server", 10)
	if err != nil {
		t.Fatalf("Failed to insert server: %v", err)
	}
	serverID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get server ID: %v", err)
	}
	return serverID
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

func TestAccountHandlers_GetCosmeticCatalog(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Insert a cosmetic item
	_, err := db.Exec(`INSERT INTO cosmetic_items (name, description, slot, category, rarity, unlock_level, data_cost, is_prestige_only) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"Test Skin", "A test cosmetic", "character_skin", "skins", "common", 1, 100, 0)
	if err != nil {
		t.Fatalf("Failed to insert cosmetic item: %v", err)
	}

	// Create a player and token
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	// Test successful catalog retrieval
	req := httptest.NewRequest(http.MethodGet, "/cosmetics/catalog", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 cosmetic item, got %d", len(items))
	}
	// Validate fields
	item := items[0]
	if item["name"] != "Test Skin" {
		t.Errorf("Expected name Test Skin, got %v", item["name"])
	}
	if item["rarity"] != "common" {
		t.Errorf("Expected rarity common, got %v", item["rarity"])
	}
	// Test without authorization header
	req = httptest.NewRequest(http.MethodGet, "/cosmetics/catalog", nil)
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing token, got %d", resp.StatusCode)
	}
}

func TestAccountHandlers_GetPlayerCosmetics(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Insert a cosmetic item
	res, err := db.Exec(`INSERT INTO cosmetic_items (name, description, slot, category, rarity, unlock_level, data_cost, is_prestige_only) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"Test Skin", "A test cosmetic", "character_skin", "skins", "common", 1, 100, 0)
	if err != nil {
		t.Fatalf("Failed to insert cosmetic item: %v", err)
	}
	cosmeticID, _ := res.LastInsertId()

	// Create a player and token
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	// Grant cosmetic to player
	_, err = db.Exec(`INSERT INTO player_cosmetics (player_id, cosmetic_id, unlocked_via) VALUES (?, ?, ?)`,
		playerID, cosmeticID, "purchase")
	if err != nil {
		t.Fatalf("Failed to grant cosmetic: %v", err)
	}

	// Test successful owned cosmetics retrieval
	req := httptest.NewRequest(http.MethodGet, "/cosmetics/owned", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	var items []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(items) != 1 {
		t.Errorf("Expected 1 owned cosmetic, got %d", len(items))
	}
	item := items[0]
	if item["name"] != "Test Skin" {
		t.Errorf("Expected name Test Skin, got %v", item["name"])
	}
	if item["unlocked_via"] != "purchase" {
		t.Errorf("Expected unlocked_via purchase, got %v", item["unlocked_via"])
	}
	// Test without authorization header
	req = httptest.NewRequest(http.MethodGet, "/cosmetics/owned", nil)
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for missing token, got %d", resp.StatusCode)
	}
}
func TestAccountHandlers_EquipCosmetic(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	// Create a cosmetic item
	_, err := db.Exec("INSERT INTO cosmetic_items (name, description, slot, rarity, unlock_level, data_cost, is_prestige_only) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"Test Skin", "A test cosmetic", "character_skin", "common", 1, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create cosmetic item: %v", err)
	}
	var cosmeticID int64
	err = db.QueryRow("SELECT last_insert_rowid()").Scan(&cosmeticID)
	if err != nil {
		t.Fatalf("Failed to get cosmetic ID: %v", err)
	}

	// Grant cosmetic to player
	_, err = db.Exec("INSERT INTO player_cosmetics (player_id, cosmetic_id, unlocked_via) VALUES (?, ?, ?)",
		playerID, cosmeticID, "purchase")
	if err != nil {
		t.Fatalf("Failed to grant cosmetic: %v", err)
	}

	// Equip the cosmetic
	reqBody := map[string]interface{}{"cosmetic_id": cosmeticID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/cosmetics/equip", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	var respBody map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&respBody); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if respBody["message"] != "cosmetic equipped successfully" {
		t.Errorf("Unexpected message: %v", respBody["message"])
	}

	// Verify cosmetic is equipped in loadout
	var slot string
	err = db.QueryRow("SELECT slot FROM loadout_cosmetics WHERE loadout_id = (SELECT loadout_id FROM loadouts WHERE player_id = ?) AND cosmetic_id = ?", playerID, cosmeticID).Scan(&slot)
	if err != nil {
		t.Fatalf("Failed to verify equip: %v", err)
	}
	if slot != "character_skin" {
		t.Errorf("Expected slot character_skin, got %s", slot)
	}
}

func TestAccountHandlers_EquipCosmetic_NotFound(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	reqBody := map[string]interface{}{"cosmetic_id": 999}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/cosmetics/equip", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.StatusCode)
	}
}

func TestAccountHandlers_EquipCosmetic_NotOwned(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)

	// Create a cosmetic item but do not grant
	_, err := db.Exec("INSERT INTO cosmetic_items (name, description, slot, rarity, unlock_level, data_cost, is_prestige_only) VALUES (?, ?, ?, ?, ?, ?, ?)",
		"Test Skin", "A test cosmetic", "character_skin", "common", 1, 0, 0)
	if err != nil {
		t.Fatalf("Failed to create cosmetic item: %v", err)
	}
	var cosmeticID int64
	err = db.QueryRow("SELECT last_insert_rowid()").Scan(&cosmeticID)
	if err != nil {
		t.Fatalf("Failed to get cosmetic ID: %v", err)
	}

	reqBody := map[string]interface{}{"cosmetic_id": cosmeticID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPut, "/cosmetics/equip", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("Expected status 403, got %d", resp.StatusCode)
	}
}

func TestAccountHandlers_StoreMatch(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player and server
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)
	serverID := createTestServerRow(t, db)

	// Create another player for stats
	player2ID := createTestPlayer(t, db, "player2", "player2@example.com", "password")

	// Prepare match request
	reqBody := map[string]interface{}{
		"server_id":            serverID,
		"map_name":             "Test Map",
		"game_mode":            "survival",
		"start_time":           "2026-01-22T15:30:00Z",
		"end_time":             "2026-01-22T16:00:00Z",
		"outcome":              "completed",
		"waves_survived":       5,
		"total_zombies_killed": 100,
		"total_players":        2,
		"player_stats": []map[string]interface{}{
			{
				"player_id":           playerID,
				"waves_survived":      5,
				"zombies_killed":      50,
				"deaths":              2,
				"scrap_earned":        1000,
				"data_earned":         50,
				"damage_dealt":        5000,
				"damage_taken":        1000,
				"buildings_built":     3,
				"buildings_destroyed": 1,
				"healing_given":       200,
				"revives":             2,
				"score":               2500,
			},
			{
				"player_id":           player2ID,
				"waves_survived":      5,
				"zombies_killed":      50,
				"deaths":              3,
				"scrap_earned":        800,
				"data_earned":         40,
				"damage_dealt":        4500,
				"damage_taken":        1200,
				"buildings_built":     2,
				"buildings_destroyed": 0,
				"healing_given":       100,
				"revives":             1,
				"score":               2300,
			},
		},
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/matches", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Verify match was created
	var matchID, totalPlayers int64
	var mapName, gameMode, outcome string
	err = db.QueryRow("SELECT match_id, map_name, game_mode, outcome, total_players FROM matches WHERE server_id = ?", serverID).Scan(&matchID, &mapName, &gameMode, &outcome, &totalPlayers)
	if err != nil {
		t.Fatalf("Failed to query match: %v", err)
	}
	if mapName != "Test Map" {
		t.Errorf("Expected map_name 'Test Map', got %s", mapName)
	}
	if gameMode != "survival" {
		t.Errorf("Expected game_mode 'survival', got %s", gameMode)
	}
	if outcome != "completed" {
		t.Errorf("Expected outcome 'completed', got %s", outcome)
	}
	if totalPlayers != 2 {
		t.Errorf("Expected total_players 2, got %d", totalPlayers)
	}

	// Verify player stats were created
	var count int64
	err = db.QueryRow("SELECT COUNT(*) FROM player_match_stats WHERE match_id = ?", matchID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count player stats: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 player stats rows, got %d", count)
	}
}

func TestAccountHandlers_GetMatchHistory(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player and server
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := createTestAccessToken(t, db, playerID)
	serverID := createTestServerRow(t, db)

	// Create a match with stats (simulate via direct DB insertion for simplicity)
	res, err := db.Exec(`INSERT INTO matches (server_id, map_name, game_mode, start_time, end_time, outcome, waves_survived, total_zombies_killed, total_players) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		serverID, "Test Map", "survival", "2026-01-22T15:30:00Z", "2026-01-22T16:00:00Z", "completed", 5, 100, 2)
	if err != nil {
		t.Fatalf("Failed to insert match: %v", err)
	}
	matchID, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get match ID: %v", err)
	}
	// Insert player match stats
	_, err = db.Exec(`INSERT INTO player_match_stats (player_id, match_id, waves_survived, zombies_killed, deaths, scrap_earned, data_earned, damage_dealt, damage_taken, buildings_built, buildings_destroyed, healing_given, revives, score) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		playerID, matchID, 5, 50, 2, 1000, 50, 5000, 1000, 3, 1, 200, 2, 2500)
	if err != nil {
		t.Fatalf("Failed to insert player match stats: %v", err)
	}

	// Request match history
	req := httptest.NewRequest(http.MethodGet, "/matches/history?limit=5", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var matches []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&matches); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(matches) != 1 {
		t.Errorf("Expected 1 match in history, got %d", len(matches))
	}
	match := matches[0]
	if match["map_name"] != "Test Map" {
		t.Errorf("Expected map_name 'Test Map', got %v", match["map_name"])
	}
	if match["player_zombies_killed"] != float64(50) {
		t.Errorf("Expected player_zombies_killed 50, got %v", match["player_zombies_killed"])
	}
}
