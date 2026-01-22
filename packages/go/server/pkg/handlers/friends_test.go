package handlers_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestFriendHandlers_SendFriendRequest(t *testing.T) {
	db := setupTestDB(t)
	player1ID := createTestPlayer(t, db, "player1", "player1@example.com", "password123")
	player2ID := createTestPlayer(t, db, "player2", "player2@example.com", "password123")
	app := createFullTestServer(t, db)
	token1 := createTestAccessToken(t, db, player1ID)

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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}

	// Try duplicate request - should conflict
	req = httptest.NewRequest(http.MethodPost, "/friends/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusConflict {
		t.Errorf("Expected status 409 for duplicate request, got %d", resp.StatusCode)
	}

	// Try self request - should bad request
	selfReq := map[string]interface{}{
		"friend_id": player1ID,
	}
	bodySelf, _ := json.Marshal(selfReq)
	req = httptest.NewRequest(http.MethodPost, "/friends/request", bytes.NewReader(bodySelf))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status 400 for self request, got %d", resp.StatusCode)
	}
}

func TestFriendHandlers_AcceptFriendRequest(t *testing.T) {
	db := setupTestDB(t)
	player1ID := createTestPlayer(t, db, "player1", "player1@example.com", "password123")
	player2ID := createTestPlayer(t, db, "player2", "player2@example.com", "password123")
	app := createFullTestServer(t, db)
	token1 := createTestAccessToken(t, db, player1ID)
	token2 := createTestAccessToken(t, db, player2ID)

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
	defer resp.Body.Close()
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
	defer resp.Body.Close()
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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for list friends, got %d", resp.StatusCode)
	}
	var friends []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&friends); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(friends) != 1 {
		t.Errorf("Expected 1 friend, got %d", len(friends))
	} else {
		if friendID, ok := friends[0]["friend_player_id"].(float64); !ok || int64(friendID) != player1ID {
			t.Errorf("Friend player ID mismatch, got %v", friends[0]["friend_player_id"])
		}
		if status, ok := friends[0]["status"].(string); !ok || status != "accepted" {
			t.Errorf("Friend status mismatch, got %v", status)
		}
	}
}

func TestFriendHandlers_DeclineFriendRequest(t *testing.T) {
	db := setupTestDB(t)
	player1ID := createTestPlayer(t, db, "player1", "player1@example.com", "password123")
	player2ID := createTestPlayer(t, db, "player2", "player2@example.com", "password123")
	app := createFullTestServer(t, db)
	token1 := createTestAccessToken(t, db, player1ID)
	token2 := createTestAccessToken(t, db, player2ID)

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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to send friend request: expected 201 got %d", resp.StatusCode)
	}

	// Decline request as player2
	declineBody := map[string]interface{}{
		"action": "decline",
	}
	bodyDecline, _ := json.Marshal(declineBody)
	req = httptest.NewRequest(http.MethodPut, "/friends/"+strconv.FormatInt(player1ID, 10), bytes.NewReader(bodyDecline))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token2)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for decline, got %d", resp.StatusCode)
	}

	// Verify no friends exist
	req = httptest.NewRequest(http.MethodGet, "/friends", nil)
	req.Header.Set("Authorization", "Bearer "+token2)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for list friends, got %d", resp.StatusCode)
	}
	var friends []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&friends); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(friends) != 0 {
		t.Errorf("Expected 0 friends after decline, got %d", len(friends))
	}
}

func TestFriendHandlers_ListFriends(t *testing.T) {
	db := setupTestDB(t)
	player1ID := createTestPlayer(t, db, "player1", "player1@example.com", "password123")
	player2ID := createTestPlayer(t, db, "player2", "player2@example.com", "password123")
	player3ID := createTestPlayer(t, db, "player3", "player3@example.com", "password123")
	app := createFullTestServer(t, db)
	token1 := createTestAccessToken(t, db, player1ID)
	token2 := createTestAccessToken(t, db, player2ID)

	// Send request player1 -> player2 and accept
	reqBody := map[string]interface{}{"friend_id": player2ID}
	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest(http.MethodPost, "/friends/request", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to send friend request: expected 201 got %d", resp.StatusCode)
	}
	acceptBody := map[string]interface{}{"action": "accept"}
	bodyAccept, _ := json.Marshal(acceptBody)
	req = httptest.NewRequest(http.MethodPut, "/friends/"+strconv.FormatInt(player1ID, 10), bytes.NewReader(bodyAccept))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token2)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Failed to accept friend request: expected 200 got %d", resp.StatusCode)
	}

	// Send request player1 -> player3 (pending) - should not appear in accepted list
	reqBody3 := map[string]interface{}{"friend_id": player3ID}
	body3, _ := json.Marshal(reqBody3)
	req = httptest.NewRequest(http.MethodPost, "/friends/request", bytes.NewReader(body3))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token1)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Failed to send friend request to player3: expected 201 got %d", resp.StatusCode)
	}

	// List friends for player1 - should only see player2 (accepted)
	req = httptest.NewRequest(http.MethodGet, "/friends", nil)
	req.Header.Set("Authorization", "Bearer "+token1)
	resp, err = app.Test(req)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for list friends, got %d", resp.StatusCode)
	}
	var friends []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&friends); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}
	if len(friends) != 1 {
		t.Errorf("Expected 1 friend (accepted), got %d", len(friends))
	} else {
		if friendID, ok := friends[0]["friend_player_id"].(float64); !ok || int64(friendID) != player2ID {
			t.Errorf("Friend player ID mismatch, got %v", friends[0]["friend_player_id"])
		}
	}
}
