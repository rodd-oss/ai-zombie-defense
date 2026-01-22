package middleware_test

import (
	"database/sql"
	"net/http/httptest"
	"testing"

	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/config"
	"ai-zombie-defense/server/pkg/middleware"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}
	// Create players table exactly as in migration
	createTableSQL := `CREATE TABLE players (
    player_id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    last_login_at TEXT,
    is_banned INTEGER NOT NULL DEFAULT 0,
    banned_reason TEXT,
    banned_until TEXT
);`
	if _, err := db.Exec(createTableSQL); err != nil {
		t.Fatalf("Failed to create players table: %v", err)
	}
	// Create sessions table
	createSessionsSQL := `CREATE TABLE sessions (
    session_id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    token TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    ip_address TEXT,
    user_agent TEXT,
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
);`
	if _, err := db.Exec(createSessionsSQL); err != nil {
		t.Fatalf("Failed to create sessions table: %v", err)
	}
	return db
}

func TestAuthMiddleware(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000, // 15 minutes
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000,
		},
	}
	authService := auth.NewService(cfg, logger, db)

	// Insert a test player
	password := "securepassword123"
	hash, err := authService.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser", "test@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert test player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	// Generate a valid access token
	token, err := authService.GenerateAccessToken(playerID)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Create Fiber app with middleware
	app := fiber.New()
	app.Use(middleware.AuthMiddleware(authService, logger))
	// Test endpoint that returns player ID
	app.Get("/test", func(c *fiber.Ctx) error {
		id, ok := middleware.GetPlayerID(c)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "player ID not found",
			})
		}
		return c.JSON(fiber.Map{"player_id": id})
	})

	t.Run("valid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("Expected status 200, got %d", resp.StatusCode)
		}
		// Could parse JSON response, but we can also trust locals.
		// For simplicity, just check status.
	})

	t.Run("missing authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("malformed authorization header", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Invalid "+token)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("empty token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer ")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})

	t.Run("invalid token", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("Request failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected status 401, got %d", resp.StatusCode)
		}
	})
}

func TestAuthMiddleware_PlayerIDInLocals(t *testing.T) {
	logger := zaptest.NewLogger(t)
	db := setupTestDB(t)
	defer db.Close()

	cfg := config.Config{
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * 60 * 1_000_000_000,
			RefreshExpiration: 7 * 24 * 60 * 60 * 1_000_000_000,
		},
	}
	authService := auth.NewService(cfg, logger, db)

	// Insert a test player
	password := "securepassword123"
	hash, err := authService.HashPassword(password)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}
	_, err = db.Exec(`INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?)`,
		"testuser2", "test2@example.com", hash)
	if err != nil {
		t.Fatalf("Failed to insert test player: %v", err)
	}
	var playerID int64
	err = db.QueryRow(`SELECT player_id FROM players WHERE username = ?`, "testuser2").Scan(&playerID)
	if err != nil {
		t.Fatalf("Failed to get player ID: %v", err)
	}

	token, err := authService.GenerateAccessToken(playerID)
	if err != nil {
		t.Fatalf("Failed to generate access token: %v", err)
	}

	// Create Fiber app with middleware and a handler that checks locals
	app := fiber.New()
	app.Use(middleware.AuthMiddleware(authService, logger))
	app.Get("/test", func(c *fiber.Ctx) error {
		id, ok := middleware.GetPlayerID(c)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "player ID not found",
			})
		}
		claims, ok := middleware.GetClaims(c)
		if !ok {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "claims not found",
			})
		}
		return c.JSON(fiber.Map{
			"player_id": id,
			"subject":   claims.Subject,
		})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
	// Optionally parse JSON and verify player_id matches.
}
