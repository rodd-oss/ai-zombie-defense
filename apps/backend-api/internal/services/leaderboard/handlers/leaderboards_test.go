package handlers_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ai-zombie-defense/backend-api/internal/api/gateway"
	"ai-zombie-defense/backend-api/internal/testutils"

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

func createTestMatch(t *testing.T, db *sql.DB, serverID int64, startTime time.Time) int64 {
	formatted := startTime.UTC().Format("2006-01-02T15:04:05Z")
	result, err := db.Exec(`INSERT INTO matches (server_id, map_name, game_mode, start_time, outcome, waves_survived, total_zombies_killed, total_players) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		serverID, "Map1", "survival", formatted, "completed", 10, 100, 4)
	if err != nil {
		t.Fatalf("Failed to insert match: %v", err)
	}
	matchID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get match ID: %v", err)
	}
	return matchID
}

func createTestPlayerMatchStats(t *testing.T, db *sql.DB, playerID, matchID int64, score, kills, waves int) {
	_, err := db.Exec(`INSERT INTO player_match_stats (player_id, match_id, score, zombies_killed, waves_survived) VALUES (?, ?, ?, ?, ?)`,
		playerID, matchID, score, kills, waves)
	if err != nil {
		t.Fatalf("Failed to insert player match stats: %v", err)
	}
}

func TestLeaderboardHandlers_GetDailyLeaderboard(t *testing.T) {
	db := testutils.SetupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create two players
	player1ID := testutils.CreateTestPlayer(t, db, "player1", "player1@example.com", "password")
	player2ID := testutils.CreateTestPlayer(t, db, "player2", "player2@example.com", "password")
	// Create a server for matches
	serverID := testutils.CreateTestServerRow(t, db)

	// Create a match with today's start time
	now := time.Now().UTC()
	matchID := createTestMatch(t, db, serverID, now)
	// Insert player match stats with different scores
	createTestPlayerMatchStats(t, db, player1ID, matchID, 5000, 50, 10)
	createTestPlayerMatchStats(t, db, player2ID, matchID, 3000, 30, 8)

	// Request daily leaderboard (no auth required)
	req := httptest.NewRequest(http.MethodGet, "/leaderboards/daily", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var entries []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(entries))
	}
}
