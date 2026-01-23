package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ai-zombie-defense/backend-api/internal/testutils"

	_ "modernc.org/sqlite"
)

func TestFavoriteHandlers_AddFavorite(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password123")
	serverID := testutils.CreateTestServerRow(t, db)
	app := createFullTestServer(t, db)
	token := testutils.CreateTestAccessToken(t, db, playerID)

	// Add favorite
	reqBody := map[string]interface{}{
		"server_id": serverID,
		"note":      "My favorite server",
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/favorites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Verify favorite exists via GET /favorites
	req = httptest.NewRequest(http.MethodGet, "/favorites", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var favorites []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&favorites); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(favorites) != 1 {
		t.Errorf("Expected 1 favorite, got %d", len(favorites))
	}
}

func TestFavoriteHandlers_RemoveFavorite(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password123")
	serverID := testutils.CreateTestServerRow(t, db)
	app := createFullTestServer(t, db)
	token := testutils.CreateTestAccessToken(t, db, playerID)

	// Add favorite
	_, err := db.Exec("INSERT INTO server_favorites (player_id, server_id) VALUES (?, ?)", playerID, serverID)
	if err != nil {
		t.Fatalf("Failed to add favorite: %v", err)
	}

	// Remove favorite
	req := httptest.NewRequest(http.MethodDelete, "/favorites/"+strconv.FormatInt(serverID, 10), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
