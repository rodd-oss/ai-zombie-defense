package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestFavoriteHandlers_AddFavorite(t *testing.T) {
	db := setupTestDB(t)
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password123")
	serverID := createTestServerRow(t, db)
	app := createFullTestServer(t, db)
	token := createTestAccessToken(t, db, playerID)

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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	if fav, ok := favorites[0]["server_id"].(float64); !ok || int64(fav) != serverID {
		t.Errorf("Favorite server ID mismatch, got %v", favorites[0]["server_id"])
	}
}

func TestFavoriteHandlers_AddFavoriteDuplicate(t *testing.T) {
	db := setupTestDB(t)
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password123")
	serverID := createTestServerRow(t, db)
	app := createFullTestServer(t, db)
	token := createTestAccessToken(t, db, playerID)

	// Add favorite first time
	reqBody := map[string]interface{}{"server_id": serverID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/favorites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Try to add duplicate
	req = httptest.NewRequest(http.MethodPost, "/favorites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate, got %d", resp.StatusCode)
	}
}

func TestFavoriteHandlers_RemoveFavorite(t *testing.T) {
	db := setupTestDB(t)
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password123")
	serverID := createTestServerRow(t, db)
	app := createFullTestServer(t, db)
	token := createTestAccessToken(t, db, playerID)

	// Add favorite
	reqBody := map[string]interface{}{"server_id": serverID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/favorites", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	resp.Body.Close()

	// Remove favorite
	req = httptest.NewRequest(http.MethodDelete, "/favorites/"+strconv.FormatInt(serverID, 10), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify list is empty
	req = httptest.NewRequest(http.MethodGet, "/favorites", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	var favorites []interface{}
	json.NewDecoder(resp.Body).Decode(&favorites)
	if len(favorites) != 0 {
		t.Errorf("Expected 0 favorites after removal, got %d", len(favorites))
	}
}

func TestFavoriteHandlers_RemoveFavoriteNotFound(t *testing.T) {
	db := setupTestDB(t)
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password123")
	app := createFullTestServer(t, db)
	token := createTestAccessToken(t, db, playerID)

	// Try to remove non-existent favorite (server exists?)
	serverID := int64(999)
	req := httptest.NewRequest(http.MethodDelete, "/favorites/"+strconv.FormatInt(serverID, 10), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	// Should succeed (idempotent) or return 404? We'll accept 200 or 404.
	// Currently our service just deletes; if row doesn't exist, it's still okay.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		t.Errorf("Expected status 200 or 404, got %d", resp.StatusCode)
	}
}

func TestFavoriteHandlers_ListFavoritesEmpty(t *testing.T) {
	db := setupTestDB(t)
	playerID := createTestPlayer(t, db, "testuser", "test@example.com", "password123")
	app := createFullTestServer(t, db)
	token := createTestAccessToken(t, db, playerID)

	req := httptest.NewRequest(http.MethodGet, "/favorites", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	var favorites []interface{}
	json.NewDecoder(resp.Body).Decode(&favorites)
	if len(favorites) != 0 {
		t.Errorf("Expected empty list, got %d", len(favorites))
	}
}
