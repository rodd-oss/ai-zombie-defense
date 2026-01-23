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

func TestAccountHandlers_StoreMatch(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player and server
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)
	serverID := testutils.CreateTestServerRow(t, db)

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
		"total_players":        1,
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
}

func TestAccountHandlers_GetMatchHistory(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create a player and server
	playerID := testutils.CreateTestPlayer(t, db, "testuser", "test@example.com", "password")
	accessToken := testutils.CreateTestAccessToken(t, db, playerID)
	serverID := testutils.CreateTestServerRow(t, db)

	// Create a match with stats
	res, err := db.Exec(`INSERT INTO matches (server_id, map_name, game_mode, start_time, end_time, outcome, waves_survived, total_zombies_killed, total_players) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		serverID, "Test Map", "survival", "2026-01-22T15:30:00Z", "2026-01-22T16:00:00Z", "completed", 5, 100, 1)
	if err != nil {
		t.Fatalf("Failed to insert match: %v", err)
	}
	matchID, _ := res.LastInsertId()
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
}
