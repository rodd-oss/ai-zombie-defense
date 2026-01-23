package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-zombie-defense/db/pkg/database"
	"ai-zombie-defense/server/internal/api/gateway"
	"ai-zombie-defense/server/pkg/config"
	"ai-zombie-defense/server/pkg/logging"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logger, err := logging.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize database
	dbConn, err := database.OpenDB(cfg.Database.Path)
	if err != nil {
		logger.Fatal("Failed to open database", zap.Error(err))
	}
	defer dbConn.Close()

	// Initialize API Gateway
	gw := gateway.NewAPIGateway(*cfg, logger, dbConn)

	// Start server in background
	go func() {
		if err := gw.Start(); err != nil {
			logger.Fatal("Gateway failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := gw.Shutdown(ctx); err != nil {
		logger.Error("Gateway shutdown failed", zap.Error(err))
	}

	logger.Info("Server stopped")
}
