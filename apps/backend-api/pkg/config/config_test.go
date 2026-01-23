package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfigDefaults(t *testing.T) {
	// Clear relevant environment variables
	os.Unsetenv("DB_PATH")
	os.Unsetenv("DB_MAX_OPEN_CONNS")
	os.Unsetenv("DB_MAX_IDLE_CONNS")
	os.Unsetenv("DB_CONN_MAX_LIFETIME")
	os.Unsetenv("DB_CONN_MAX_IDLE_TIME")
	os.Unsetenv("DB_MIGRATIONS_PATH")
	os.Unsetenv("SERVER_HOST")
	os.Unsetenv("SERVER_PORT")
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_ACCESS_EXPIRATION")
	os.Unsetenv("JWT_REFRESH_EXPIRATION")

	// Should fail because JWT_SECRET is required (no default)
	_, err := LoadConfig()
	if err == nil {
		t.Fatal("Expected error due to missing JWT_SECRET")
	}
	if err.Error() != "JWT_SECRET environment variable is required" {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Set JWT_SECRET and try again
	t.Setenv("JWT_SECRET", "test-secret")
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config with JWT_SECRET: %v", err)
	}

	// Check defaults
	if cfg.Database.Path != "./data.db" {
		t.Errorf("Default DB_PATH mismatch: got %s", cfg.Database.Path)
	}
	if cfg.Database.MaxOpenConns != 5 {
		t.Errorf("Default DB_MAX_OPEN_CONNS mismatch: got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 2 {
		t.Errorf("Default DB_MAX_IDLE_CONNS mismatch: got %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("Default DB_CONN_MAX_LIFETIME mismatch: got %v", cfg.Database.ConnMaxLifetime)
	}
	if cfg.Database.ConnMaxIdleTime != 2*time.Minute {
		t.Errorf("Default DB_CONN_MAX_IDLE_TIME mismatch: got %v", cfg.Database.ConnMaxIdleTime)
	}
	if cfg.Database.MigrationsPath != "./migrations" {
		t.Errorf("Default DB_MIGRATIONS_PATH mismatch: got %s", cfg.Database.MigrationsPath)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Default SERVER_HOST mismatch: got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Default SERVER_PORT mismatch: got %d", cfg.Server.Port)
	}
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("JWT_SECRET mismatch: got %s", cfg.JWT.Secret)
	}
	if cfg.JWT.AccessExpiration != 15*time.Minute {
		t.Errorf("Default JWT_ACCESS_EXPIRATION mismatch: got %v", cfg.JWT.AccessExpiration)
	}
	if cfg.JWT.RefreshExpiration != 7*24*time.Hour {
		t.Errorf("Default JWT_REFRESH_EXPIRATION mismatch: got %v", cfg.JWT.RefreshExpiration)
	}
}

func TestLoadConfigEnvironmentOverride(t *testing.T) {
	// Set environment variables
	t.Setenv("JWT_SECRET", "custom-secret")
	t.Setenv("DB_PATH", "/custom/path.db")
	t.Setenv("DB_MAX_OPEN_CONNS", "10")
	t.Setenv("DB_MAX_IDLE_CONNS", "5")
	t.Setenv("DB_CONN_MAX_LIFETIME", "10m")
	t.Setenv("DB_CONN_MAX_IDLE_TIME", "5m")
	t.Setenv("DB_MIGRATIONS_PATH", "/custom/migrations")
	t.Setenv("SERVER_HOST", "127.0.0.1")
	t.Setenv("SERVER_PORT", "3000")
	t.Setenv("JWT_ACCESS_EXPIRATION", "1h")
	t.Setenv("JWT_REFRESH_EXPIRATION", "48h")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check overrides
	if cfg.Database.Path != "/custom/path.db" {
		t.Errorf("DB_PATH override mismatch: got %s", cfg.Database.Path)
	}
	if cfg.Database.MaxOpenConns != 10 {
		t.Errorf("DB_MAX_OPEN_CONNS override mismatch: got %d", cfg.Database.MaxOpenConns)
	}
	if cfg.Database.MaxIdleConns != 5 {
		t.Errorf("DB_MAX_IDLE_CONNS override mismatch: got %d", cfg.Database.MaxIdleConns)
	}
	if cfg.Database.ConnMaxLifetime != 10*time.Minute {
		t.Errorf("DB_CONN_MAX_LIFETIME override mismatch: got %v", cfg.Database.ConnMaxLifetime)
	}
	if cfg.Database.ConnMaxIdleTime != 5*time.Minute {
		t.Errorf("DB_CONN_MAX_IDLE_TIME override mismatch: got %v", cfg.Database.ConnMaxIdleTime)
	}
	if cfg.Database.MigrationsPath != "/custom/migrations" {
		t.Errorf("DB_MIGRATIONS_PATH override mismatch: got %s", cfg.Database.MigrationsPath)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("SERVER_HOST override mismatch: got %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 3000 {
		t.Errorf("SERVER_PORT override mismatch: got %d", cfg.Server.Port)
	}
	if cfg.JWT.Secret != "custom-secret" {
		t.Errorf("JWT_SECRET override mismatch: got %s", cfg.JWT.Secret)
	}
	if cfg.JWT.AccessExpiration != time.Hour {
		t.Errorf("JWT_ACCESS_EXPIRATION override mismatch: got %v", cfg.JWT.AccessExpiration)
	}
	if cfg.JWT.RefreshExpiration != 48*time.Hour {
		t.Errorf("JWT_REFRESH_EXPIRATION override mismatch: got %v", cfg.JWT.RefreshExpiration)
	}
}

func TestLoadConfigInvalidDuration(t *testing.T) {
	// Set invalid duration for JWT_ACCESS_EXPIRATION
	t.Setenv("JWT_SECRET", "secret")
	t.Setenv("JWT_ACCESS_EXPIRATION", "invalid")
	// Viper will treat invalid duration as zero, which is fine for test
	// We just ensure config still loads
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config with invalid duration: %v", err)
	}
	// Expect zero duration
	if cfg.JWT.AccessExpiration != 0 {
		t.Errorf("Expected zero duration for invalid input, got %v", cfg.JWT.AccessExpiration)
	}
}
