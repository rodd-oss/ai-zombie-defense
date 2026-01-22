package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"
	"time"

	"ai-zombie-defense/server/pkg/config"
	"go.uber.org/zap/zaptest"
)

// getFreePort returns a free TCP port for testing.
func getFreePort(t *testing.T) int {
	t.Helper()
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("Failed to find free port: %v", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port
}

func TestNewServer(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: getFreePort(t),
		},
		Database: config.DatabaseConfig{
			Path:           ":memory:",
			MigrationsPath: "./migrations",
		},
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
	}

	srv := New(cfg, logger, nil)
	if srv == nil {
		t.Fatal("Expected server instance, got nil")
	}
}

func TestHealthEndpoint(t *testing.T) {
	logger := zaptest.NewLogger(t)

	cfg := config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: getFreePort(t),
		},
		Database: config.DatabaseConfig{
			Path:           ":memory:",
			MigrationsPath: "./migrations",
		},
		JWT: config.JWTConfig{
			Secret:            "test-secret",
			AccessExpiration:  15 * time.Minute,
			RefreshExpiration: 7 * 24 * time.Hour,
		},
	}

	srv := New(cfg, logger, nil)

	// Start server in background
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			t.Errorf("Server failed: %v", err)
		}
	}()
	defer srv.Shutdown(context.Background())

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Make request to health endpoint
	resp, err := http.Get(fmt.Sprintf("http://%s:%d/health", cfg.Server.Host, cfg.Server.Port))
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}
}
