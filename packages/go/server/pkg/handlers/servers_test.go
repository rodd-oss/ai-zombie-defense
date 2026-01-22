package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ai-zombie-defense/server/pkg/server"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
)

func createTestServerWithAllRoutes(t *testing.T, db *sql.DB) *fiber.App {
	logger := zaptest.NewLogger(t)
	cfg := getTestConfig()
	srv := server.New(cfg, logger, db)
	return srv.App()
}

func TestUpdateHeartbeat(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createTestServerWithAllRoutes(t, db)

	// First, register a server to get auth token
	registerReq := map[string]interface{}{
		"ip_address":   "127.0.0.1",
		"port":         27015,
		"name":         "Test Server",
		"map_rotation": "map1,map2",
		"max_players":  12,
		"region":       "us-east",
		"version":      "1.0.0",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/servers/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make register request: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201 for server registration, got %d", resp.StatusCode)
	}
	var registerResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		t.Fatalf("Failed to decode registration response: %v", err)
	}
	serverIDFloat, ok := registerResp["server_id"].(float64) // JSON numbers are float64
	if !ok {
		t.Fatal("Missing server_id in response")
	}
	serverID := int64(serverIDFloat)
	authToken, ok := registerResp["auth_token"].(string)
	if !ok || authToken == "" {
		t.Fatal("Missing auth_token in response")
	}

	// Now send heartbeat
	heartbeatReq := map[string]interface{}{
		"current_players": 5,
		"map":             "map1",
	}
	body, _ = json.Marshal(heartbeatReq)
	req = httptest.NewRequest(http.MethodPut, "/servers/"+strconv.FormatInt(serverID, 10)+"/heartbeat", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Server-Token", authToken)
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make heartbeat request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for heartbeat, got %d", resp.StatusCode)
	}
	var heartbeatResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&heartbeatResp); err != nil {
		t.Fatalf("Failed to decode heartbeat response: %v", err)
	}
	if status, ok := heartbeatResp["status"].(string); !ok || status != "ok" {
		t.Errorf("Expected status 'ok', got %v", heartbeatResp)
	}

	// Verify server heartbeat updated in database
	var currentPlayers int64
	var lastHeartbeat, mapRotation *string
	err = db.QueryRow(`SELECT current_players, last_heartbeat, map_rotation FROM servers WHERE server_id = ?`, serverID).Scan(&currentPlayers, &lastHeartbeat, &mapRotation)
	if err != nil {
		t.Fatalf("Failed to query server: %v", err)
	}
	if currentPlayers != 5 {
		t.Errorf("Expected current_players = 5, got %d", currentPlayers)
	}
	if lastHeartbeat == nil || *lastHeartbeat == "" {
		t.Error("last_heartbeat not updated")
	}
	if mapRotation == nil || *mapRotation != "map1" {
		t.Errorf("Expected map_rotation = 'map1', got %v", mapRotation)
	}
}
