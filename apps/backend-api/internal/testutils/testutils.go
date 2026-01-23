package testutils

import (
	"ai-zombie-defense/backend-api/internal/services/auth"
	"ai-zombie-defense/backend-api/pkg/config"
	"context"
	"database/sql"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func GetTestConfig() config.Config {
	return config.Config{
		Database: config.DatabaseConfig{
			Path:           ":memory:",
			MigrationsPath: "./migrations",
		},
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
		Progression: config.ProgressionConfig{
			BaseXPPerLevel: 1000,
		},
	}
}

func SetupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}
	// Create tables (simplified for test utility, in a real app use migrations)
	createTables(t, db)
	return db
}

func createTables(t *testing.T, db *sql.DB) {
	tables := []string{
		`CREATE TABLE players (
            player_id INTEGER PRIMARY KEY AUTOINCREMENT,
            username TEXT NOT NULL UNIQUE,
            email TEXT NOT NULL UNIQUE,
            password_hash TEXT NOT NULL,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            last_login_at TEXT,
            is_banned INTEGER NOT NULL DEFAULT 0,
            banned_reason TEXT,
            banned_until TEXT,
            is_admin INTEGER NOT NULL DEFAULT 0
        );`,
		`CREATE TABLE sessions (
            session_id INTEGER PRIMARY KEY AUTOINCREMENT,
            player_id INTEGER NOT NULL,
            token TEXT NOT NULL UNIQUE,
            expires_at TEXT NOT NULL,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            ip_address TEXT,
            user_agent TEXT,
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE player_settings (
            player_id INTEGER PRIMARY KEY,
            key_bindings TEXT,
            mouse_sensitivity REAL,
            ui_scale REAL,
            color_blind_mode INTEGER NOT NULL DEFAULT 0,
            subtitles_enabled INTEGER NOT NULL DEFAULT 0,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE player_progression (
            player_id INTEGER PRIMARY KEY,
            level INTEGER NOT NULL DEFAULT 1,
            experience INTEGER NOT NULL DEFAULT 0,
            prestige_level INTEGER NOT NULL DEFAULT 0,
            data_currency INTEGER NOT NULL DEFAULT 0,
            total_matches_played INTEGER NOT NULL DEFAULT 0,
            total_waves_survived INTEGER NOT NULL DEFAULT 0,
            total_kills INTEGER NOT NULL DEFAULT 0,
            total_deaths INTEGER NOT NULL DEFAULT 0,
            total_scrap_earned INTEGER NOT NULL DEFAULT 0,
            total_data_earned INTEGER NOT NULL DEFAULT 0,
            updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE currency_transactions (
            transaction_id INTEGER PRIMARY KEY AUTOINCREMENT,
            player_id INTEGER NOT NULL,
            amount INTEGER NOT NULL,
            balance_after INTEGER NOT NULL,
            transaction_type TEXT NOT NULL CHECK (transaction_type IN ('match_reward', 'purchase', 'prestige_reward', 'admin_grant', 'refund', 'other')),
            reference_id INTEGER,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE cosmetic_items (
            cosmetic_id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            description TEXT,
            slot TEXT NOT NULL CHECK (slot IN ('character_skin', 'weapon_skin', 'emote', 'taunt', 'badge', 'title', 'particle_effect', 'other')),
            category TEXT,
            rarity TEXT NOT NULL CHECK (rarity IN ('common', 'uncommon', 'rare', 'epic', 'legendary')),
            unlock_level INTEGER NOT NULL DEFAULT 1,
            data_cost INTEGER NOT NULL DEFAULT 0,
            is_prestige_only INTEGER NOT NULL DEFAULT 0,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
        );`,
		`CREATE TABLE loot_tables (
            loot_table_id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            description TEXT,
            drop_chance REAL NOT NULL,
            is_active INTEGER NOT NULL DEFAULT 1,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
        );`,
		`CREATE TABLE loot_table_entries (
            loot_entry_id INTEGER PRIMARY KEY AUTOINCREMENT,
            loot_table_id INTEGER NOT NULL,
            cosmetic_id INTEGER NOT NULL,
            weight INTEGER NOT NULL,
            min_quantity INTEGER NOT NULL DEFAULT 1,
            max_quantity INTEGER NOT NULL DEFAULT 1,
            FOREIGN KEY (loot_table_id) REFERENCES loot_tables (loot_table_id) ON DELETE CASCADE,
            FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE player_cosmetics (
            player_id INTEGER NOT NULL,
            cosmetic_id INTEGER NOT NULL,
            unlocked_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            unlocked_via TEXT NOT NULL CHECK (unlocked_via IN ('level_up', 'purchase', 'loot_drop', 'prestige')),
            PRIMARY KEY (player_id, cosmetic_id),
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
            FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE friends (
            player_id INTEGER NOT NULL,
            friend_id INTEGER NOT NULL,
            status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'blocked')),
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            PRIMARY KEY (player_id, friend_id),
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
            FOREIGN KEY (friend_id) REFERENCES players (player_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE loadouts (
            loadout_id INTEGER PRIMARY KEY AUTOINCREMENT,
            player_id INTEGER NOT NULL,
            name TEXT NOT NULL,
            is_active INTEGER NOT NULL DEFAULT 0,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE loadout_cosmetics (
            loadout_id INTEGER NOT NULL,
            cosmetic_id INTEGER NOT NULL,
            slot TEXT NOT NULL CHECK (slot IN ('character_skin', 'weapon_skin', 'emote', 'taunt', 'badge', 'title', 'particle_effect', 'other')),
            PRIMARY KEY (loadout_id, cosmetic_id),
            FOREIGN KEY (loadout_id) REFERENCES loadouts (loadout_id) ON DELETE CASCADE,
            FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE servers (
            server_id INTEGER PRIMARY KEY AUTOINCREMENT,
            ip_address TEXT NOT NULL,
            port INTEGER NOT NULL,
            auth_token TEXT UNIQUE,
            name TEXT NOT NULL,
            map_rotation TEXT,
            max_players INTEGER NOT NULL,
            current_players INTEGER NOT NULL DEFAULT 0,
            is_online INTEGER NOT NULL DEFAULT 0,
            last_heartbeat TEXT,
            region TEXT,
            version TEXT,
            created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
        );`,
		`CREATE TABLE server_favorites (
            player_id INTEGER NOT NULL,
            server_id INTEGER NOT NULL,
            added_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
            note TEXT,
            PRIMARY KEY (player_id, server_id),
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
            FOREIGN KEY (server_id) REFERENCES servers (server_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE matches (
            match_id INTEGER PRIMARY KEY AUTOINCREMENT,
            server_id INTEGER NOT NULL,
            map_name TEXT NOT NULL,
            game_mode TEXT NOT NULL,
            start_time TEXT NOT NULL,
            end_time TEXT,
            outcome TEXT NOT NULL CHECK (outcome IN ('completed', 'failed', 'abandoned')),
            waves_survived INTEGER NOT NULL DEFAULT 0,
            total_zombies_killed INTEGER NOT NULL DEFAULT 0,
            total_players INTEGER NOT NULL DEFAULT 0,
            FOREIGN KEY (server_id) REFERENCES servers (server_id) ON DELETE CASCADE
        );`,
		`CREATE TABLE player_match_stats (
            player_id INTEGER NOT NULL,
            match_id INTEGER NOT NULL,
            waves_survived INTEGER NOT NULL DEFAULT 0,
            zombies_killed INTEGER NOT NULL DEFAULT 0,
            deaths INTEGER NOT NULL DEFAULT 0,
            scrap_earned INTEGER NOT NULL DEFAULT 0,
            data_earned INTEGER NOT NULL DEFAULT 0,
            damage_dealt INTEGER NOT NULL DEFAULT 0,
            damage_taken INTEGER NOT NULL DEFAULT 0,
            buildings_built INTEGER NOT NULL DEFAULT 0,
            buildings_destroyed INTEGER NOT NULL DEFAULT 0,
            healing_given INTEGER NOT NULL DEFAULT 0,
            revives INTEGER NOT NULL DEFAULT 0,
            score INTEGER NOT NULL DEFAULT 0,
            PRIMARY KEY (player_id, match_id),
            FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
            FOREIGN KEY (match_id) REFERENCES matches (match_id) ON DELETE CASCADE
        );`,
	}

	for _, sql := range tables {
		if _, err := db.Exec(sql); err != nil {
			t.Fatalf("Failed to execute SQL: %v\nSQL: %s", err, sql)
		}
	}
}

func CreateTestPlayer(t *testing.T, dbConn *sql.DB, username, email, password string) int64 {
	logger := zaptest.NewLogger(t)
	cfg := GetTestConfig()
	service := auth.NewAuthService(cfg, logger, dbConn)

	// Use bcrypt directly for hashing if needed, or use service
	// For simplicity, let's just use the service since we have it
	// Wait, RegisterPlayer also creates progression row etc.
	player, err := service.RegisterPlayer(context.Background(), username, email, password)
	if err != nil {
		t.Fatalf("Failed to register player: %v", err)
	}
	return player.PlayerID
}

func CreateTestAccessToken(t *testing.T, dbConn *sql.DB, playerID int64) string {
	logger := zaptest.NewLogger(t)
	cfg := GetTestConfig()
	service := auth.NewAuthService(cfg, logger, dbConn)
	token, err := service.GenerateAccessToken(playerID)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}
	return token
}

func CreateTestSession(t *testing.T, dbConn *sql.DB, playerID int64) string {
	logger := zaptest.NewLogger(t)
	cfg := GetTestConfig()
	service := auth.NewAuthService(cfg, logger, dbConn)
	ctx := context.Background()
	token, err := service.CreateSession(ctx, playerID, "127.0.0.1", "test-agent")
	if err != nil {
		t.Fatalf("Failed to create session: %v", err)
	}
	return token
}

func CreateTestServerRow(t *testing.T, dbConn *sql.DB) int64 {
	result, err := dbConn.Exec(`INSERT INTO servers (ip_address, port, name, max_players) VALUES (?, ?, ?, ?)`,
		"127.0.0.1", 7777, "Test Server", 10)
	if err != nil {
		t.Fatalf("Failed to insert server: %v", err)
	}
	serverID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get server ID: %v", err)
	}
	return serverID
}
