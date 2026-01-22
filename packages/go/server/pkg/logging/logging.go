package logging

import (
	"os"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// NewLogger creates a configured zap.Logger with JSON output format.
// The log level is determined by the LOG_LEVEL environment variable
// (default: "info"). Valid levels: debug, info, warn, error, dpanic, panic, fatal.
// If LOG_ENCODING is set to "console", uses console encoding for development.
func NewLogger() (*zap.Logger, error) {
	var config zap.Config

	// Determine encoding from environment
	encoding := os.Getenv("LOG_ENCODING")
	if encoding == "console" {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	// Set log level from environment
	level := strings.ToLower(os.Getenv("LOG_LEVEL"))
	if level == "" {
		level = "info"
	}
	var zapLevel zapcore.Level
	if err := zapLevel.Set(level); err != nil {
		zapLevel = zapcore.InfoLevel
	}
	config.Level = zap.NewAtomicLevelAt(zapLevel)

	// Build logger
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

// MustNewLogger creates a logger and panics if initialization fails.
// Useful for application startup where logging is critical.
func MustNewLogger() *zap.Logger {
	logger, err := NewLogger()
	if err != nil {
		panic(err)
	}
	return logger
}
