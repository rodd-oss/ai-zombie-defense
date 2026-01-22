package logging

import (
	"os"
	"testing"

	"go.uber.org/zap"
)

func TestNewLogger(t *testing.T) {
	// Clear environment variables for test
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("LOG_ENCODING")

	// Test default logger (production, info level)
	logger, err := NewLogger()
	if err != nil {
		t.Fatalf("Failed to create default logger: %v", err)
	}
	defer logger.Sync()
	if logger == nil {
		t.Fatal("Logger is nil")
	}

	// Test with console encoding
	t.Setenv("LOG_ENCODING", "console")
	consoleLogger, err := NewLogger()
	if err != nil {
		t.Fatalf("Failed to create console logger: %v", err)
	}
	defer consoleLogger.Sync()

	// Test different log levels
	testLevels := []string{"debug", "info", "warn", "error"}
	for _, level := range testLevels {
		t.Run("level_"+level, func(t *testing.T) {
			t.Setenv("LOG_LEVEL", level)
			logger, err := NewLogger()
			if err != nil {
				t.Errorf("Failed to create logger with level %s: %v", level, err)
				return
			}
			defer logger.Sync()
			// Verify logger works by logging a test message
			logger.Debug("test debug message", zap.String("test", "debug"))
			logger.Info("test info message", zap.String("test", "info"))
			logger.Warn("test warn message", zap.String("test", "warn"))
			logger.Error("test error message", zap.String("test", "error"))
		})
	}

	// Test invalid log level falls back to info
	t.Setenv("LOG_LEVEL", "invalid")
	logger, err = NewLogger()
	if err != nil {
		t.Fatalf("Failed to create logger with invalid level: %v", err)
	}
	defer logger.Sync()
	// Should still be able to log
	logger.Info("test after invalid level")
}

func TestMustNewLogger(t *testing.T) {
	// Should not panic with valid environment
	os.Unsetenv("LOG_LEVEL")
	logger := MustNewLogger()
	if logger == nil {
		t.Fatal("MustNewLogger returned nil")
	}
	defer logger.Sync()

	// Test that it actually logs
	logger.Info("test message from MustNewLogger")
}
