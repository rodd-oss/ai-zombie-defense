package handlers_test

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

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
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// Create two players
	player1ID := createTestPlayer(t, db, "player1", "player1@example.com", "password")
	player2ID := createTestPlayer(t, db, "player2", "player2@example.com", "password")
	// Create a server for matches
	serverID := createTestServerRow(t, db)

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
	defer resp.Body.Close()

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

	// Check ordering: player1 should be rank 1 with higher score
	if entries[0]["username"] != "player1" {
		t.Errorf("Expected rank 1 to be player1, got %v", entries[0]["username"])
	}
	if entries[0]["total_score"] != float64(5000) {
		t.Errorf("Expected rank 1 total_score 5000, got %v", entries[0]["total_score"])
	}
	if entries[1]["username"] != "player2" {
		t.Errorf("Expected rank 2 to be player2, got %v", entries[1]["username"])
	}
}

func TestLeaderboardHandlers_GetWeeklyLeaderboard(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	player1ID := createTestPlayer(t, db, "player1", "player1@example.com", "password")
	player2ID := createTestPlayer(t, db, "player2", "player2@example.com", "password")
	serverID := createTestServerRow(t, db)

	// Create a match within the last 7 days (yesterday)
	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	matchID := createTestMatch(t, db, serverID, yesterday)
	createTestPlayerMatchStats(t, db, player1ID, matchID, 7000, 70, 12)
	createTestPlayerMatchStats(t, db, player2ID, matchID, 4000, 40, 9)

	req := httptest.NewRequest(http.MethodGet, "/leaderboards/weekly", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

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

	if entries[0]["username"] != "player1" {
		t.Errorf("Expected rank 1 to be player1, got %v", entries[0]["username"])
	}
}

func TestLeaderboardHandlers_GetAllTimeLeaderboard(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	player1ID := createTestPlayer(t, db, "player1", "player1@example.com", "password")
	player2ID := createTestPlayer(t, db, "player2", "player2@example.com", "password")
	serverID := createTestServerRow(t, db)

	// Create a match with an old date (still counts for all-time)
	pastDate := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	matchID := createTestMatch(t, db, serverID, pastDate)
	createTestPlayerMatchStats(t, db, player1ID, matchID, 9000, 90, 15)
	createTestPlayerMatchStats(t, db, player2ID, matchID, 6000, 60, 10)

	req := httptest.NewRequest(http.MethodGet, "/leaderboards/alltime", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

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

	if entries[0]["username"] != "player1" {
		t.Errorf("Expected rank 1 to be player1, got %v", entries[0]["username"])
	}
}

// Edge case: no matches in period should return empty array
func TestLeaderboardHandlers_EmptyLeaderboard(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()
	app := createFullTestServer(t, db)

	// No matches created

	req := httptest.NewRequest(http.MethodGet, "/leaderboards/daily", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var entries []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected empty leaderboard, got %d entries", len(entries))
	}
}
