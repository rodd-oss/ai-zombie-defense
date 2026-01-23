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

func TestFriendHandlers_SendFriendRequest(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	player1ID := testutils.CreateTestPlayer(t, db, "player1", "player1@example.com", "password123")
	player2ID := testutils.CreateTestPlayer(t, db, "player2", "player2@example.com", "password123")
	app := createFullTestServer(t, db)
	token1 := testutils.CreateTestAccessToken(t, db, player1ID)

	// Send friend request from player1 to player2
	reqBody := map[string]interface{}{
		"friend_id": player2ID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/friends/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
}

func TestFriendHandlers_AcceptFriendRequest(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	player1ID := testutils.CreateTestPlayer(t, db, "player1", "player1@example.com", "password123")
	player2ID := testutils.CreateTestPlayer(t, db, "player2", "player2@example.com", "password123")
	app := createFullTestServer(t, db)
	token1 := testutils.CreateTestAccessToken(t, db, player1ID)
	token2 := testutils.CreateTestAccessToken(t, db, player2ID)

	// Send friend request from player1 to player2
	reqBody := map[string]interface{}{
		"friend_id": player2ID,
	}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/friends/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to send friend request: expected 201 got %d", resp.StatusCode)
	}

	// Accept request as player2
	acceptBody := map[string]interface{}{
		"action": "accept",
	}
	bodyAccept, _ := json.Marshal(acceptBody)
	req = httptest.NewRequest(http.MethodPut, "/friends/"+strconv.FormatInt(player1ID, 10), bytes.NewReader(bodyAccept))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token2)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for accept, got %d", resp.StatusCode)
	}

	// Verify they are friends via GET /friends (player2 perspective)
	req = httptest.NewRequest(http.MethodGet, "/friends", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for list friends, got %d", resp.StatusCode)
	}
}
