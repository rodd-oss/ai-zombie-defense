package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application.
type Config struct {
	Database DatabaseConfig
	Server   ServerConfig
	JWT      JWTConfig
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Path            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	MigrationsPath  string
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string
	Port int
}

// JWTConfig holds JWT token generation and validation settings.
type JWTConfig struct {
	Secret            string
	AccessExpiration  time.Duration
	RefreshExpiration time.Duration
}

// LoadConfig loads configuration from environment variables and defaults.
// Environment variables should be uppercase with underscores, e.g., DB_PATH.
// Uses viper for automatic env binding.
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Set defaults
	setDefaults(v)

	// Bind environment variables
	bindEnv(v)

	// Read environment variables
	v.AutomaticEnv()

	// Validate required settings
	if err := validateRequired(v); err != nil {
		return nil, err
	}

	// Build config struct
	cfg := &Config{
		Database: DatabaseConfig{
			Path:            v.GetString("db_path"),
			MaxOpenConns:    v.GetInt("db_max_open_conns"),
			MaxIdleConns:    v.GetInt("db_max_idle_conns"),
			ConnMaxLifetime: v.GetDuration("db_conn_max_lifetime"),
			ConnMaxIdleTime: v.GetDuration("db_conn_max_idle_time"),
			MigrationsPath:  v.GetString("db_migrations_path"),
		},
		Server: ServerConfig{
			Host: v.GetString("server_host"),
			Port: v.GetInt("server_port"),
		},
		JWT: JWTConfig{
			Secret:            v.GetString("jwt_secret"),
			AccessExpiration:  v.GetDuration("jwt_access_expiration"),
			RefreshExpiration: v.GetDuration("jwt_refresh_expiration"),
		},
	}

	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Database defaults
	v.SetDefault("db_path", "./data.db")
	v.SetDefault("db_max_open_conns", 5)
	v.SetDefault("db_max_idle_conns", 2)
	v.SetDefault("db_conn_max_lifetime", 5*time.Minute)
	v.SetDefault("db_conn_max_idle_time", 2*time.Minute)
	v.SetDefault("db_migrations_path", "./migrations")

	// Server defaults
	v.SetDefault("server_host", "0.0.0.0")
	v.SetDefault("server_port", 8080)

	// JWT defaults
	v.SetDefault("jwt_access_expiration", 15*time.Minute)
	v.SetDefault("jwt_refresh_expiration", 7*24*time.Hour) // 7 days
}

func bindEnv(v *viper.Viper) {
	// Database
	_ = v.BindEnv("db_path", "DB_PATH")
	_ = v.BindEnv("db_max_open_conns", "DB_MAX_OPEN_CONNS")
	_ = v.BindEnv("db_max_idle_conns", "DB_MAX_IDLE_CONNS")
	_ = v.BindEnv("db_conn_max_lifetime", "DB_CONN_MAX_LIFETIME")
	_ = v.BindEnv("db_conn_max_idle_time", "DB_CONN_MAX_IDLE_TIME")
	_ = v.BindEnv("db_migrations_path", "DB_MIGRATIONS_PATH")

	// Server
	_ = v.BindEnv("server_host", "SERVER_HOST")
	_ = v.BindEnv("server_port", "SERVER_PORT")

	// JWT
	_ = v.BindEnv("jwt_secret", "JWT_SECRET")
	_ = v.BindEnv("jwt_access_expiration", "JWT_ACCESS_EXPIRATION")
	_ = v.BindEnv("jwt_refresh_expiration", "JWT_REFRESH_EXPIRATION")
}

func validateRequired(v *viper.Viper) error {
	// JWT secret is required
	if v.GetString("jwt_secret") == "" {
		return fmt.Errorf("JWT_SECRET environment variable is required")
	}
	return nil
}
