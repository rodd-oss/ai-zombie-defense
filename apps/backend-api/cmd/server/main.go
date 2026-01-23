package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"ai-zombie-defense/backend-api/internal/api/gateway"
	"ai-zombie-defense/backend-api/internal/db"
	"ai-zombie-defense/backend-api/pkg/config"
	"ai-zombie-defense/backend-api/pkg/logging"
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

	// Handle subcommands
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "migrate":
			handleMigrate(cfg, logger)
			return
		case "help":
			printUsage()
			return
		}
	}

	// Initialize database
	dbConn, err := db.OpenDB(cfg.Database.Path)
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

func handleMigrate(cfg *config.Config, logger *zap.Logger) {
	if len(os.Args) < 3 {
		fmt.Println("Usage: server migrate [up|down|status]")
		os.Exit(1)
	}

	dbConn, err := db.OpenDB(cfg.Database.Path)
	if err != nil {
		logger.Fatal("Failed to open database", zap.Error(err))
	}
	defer dbConn.Close()

	var errMig error
	command := os.Args[2]
	switch command {
	case "up":
		errMig = db.RunMigrations(dbConn)
	case "down":
		errMig = db.Rollback(dbConn)
	case "status":
		errMig = db.Status(dbConn)
	default:
		fmt.Printf("Unknown migration command: %s\n", command)
		os.Exit(1)
	}

	if errMig != nil {
		logger.Fatal("Migration failed", zap.Error(errMig))
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  server              - Start the API server")
	fmt.Println("  server migrate up   - Run pending migrations")
	fmt.Println("  server migrate down - Rollback the last migration")
	fmt.Println("  server migrate status - Show migration status")
	fmt.Println("  server help         - Show this help message")
}
