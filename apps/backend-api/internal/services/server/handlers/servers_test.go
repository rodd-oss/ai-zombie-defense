package handlers_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"ai-zombie-defense/backend-api/internal/api/gateway"
	"ai-zombie-defense/backend-api/internal/testutils"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func createTestServerWithAllRoutes(t *testing.T, db *sql.DB) *fiber.App {
	logger := zaptest.NewLogger(t)
	cfg := testutils.GetTestConfig()
	gw := gateway.NewAPIGateway(cfg, logger, db)
	return gw.Router()
}

func TestUpdateHeartbeat(t *testing.T) {
	db := testutils.SetupTestDB(t)
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
}

func TestListServers(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createTestServerWithAllRoutes(t, db)

	// Register server
	registerReq := map[string]interface{}{
		"ip_address":   "127.0.0.1",
		"port":         27015,
		"name":         "Test Server",
		"map_rotation": "map1",
		"max_players":  12,
		"region":       "us-east",
		"version":      "1.0.0",
	}
	body, _ := json.Marshal(registerReq)
	req := httptest.NewRequest(http.MethodPost, "/servers/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to register server: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("Expected status 201, got %d", resp.StatusCode)
	}
	var registerResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&registerResp); err != nil {
		t.Fatalf("Failed to decode registration response: %v", err)
	}
	serverID := int64(registerResp["server_id"].(float64))
	authToken := registerResp["auth_token"].(string)

	// Send heartbeat to make server online
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
		t.Fatalf("Failed to send heartbeat: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// GET /servers should return the server
	req = httptest.NewRequest(http.MethodGet, "/servers", nil)
	resp, err = app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to list servers: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
