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

func TestAccountHandlers_GetProgression(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)

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
		})
	}
}

func TestAccountHandlers_GetCosmeticCatalog(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Insert a cosmetic item
	_, err := db.Exec(`INSERT INTO cosmetic_items (name, description, slot, category, rarity, unlock_level, data_cost, is_prestige_only) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"Test Skin", "A test cosmetic", "character_skin", "skins", "common", 1, 100, 0)
	if err != nil {
		t.Fatalf("Failed to insert cosmetic item: %v", err)
	}

	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)

	req := httptest.NewRequest(http.MethodGet, "/cosmetics/catalog", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}

func TestAccountHandlers_PurchaseCosmetic(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)

	// Insert a cosmetic item with data_cost
	_, err := db.Exec(`INSERT INTO cosmetic_items (name, description, slot, rarity, unlock_level, data_cost, is_prestige_only) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"Test Skin", "A test cosmetic", "character_skin", "common", 1, 150, 0)
	if err != nil {
		t.Fatalf("Failed to insert cosmetic item: %v", err)
	}
	var cosmeticID int64
	err = db.QueryRow(`SELECT cosmetic_id FROM cosmetic_items WHERE name = ?`, "Test Skin").Scan(&cosmeticID)
	if err != nil {
		t.Fatalf("Failed to get cosmetic ID: %v", err)
	}

	// Give player some data currency (200)
	_, err = db.Exec(`UPDATE player_progression SET data_currency = 200 WHERE player_id = ?`, playerID)
	if err != nil {
		t.Fatalf("Failed to set data currency: %v", err)
	}

	reqBody := map[string]interface{}{"cosmetic_id": cosmeticID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/cosmetics/purchase", bytes.NewReader(body))
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
