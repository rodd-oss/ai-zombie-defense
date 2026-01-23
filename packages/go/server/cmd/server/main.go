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
	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/config"
	"ai-zombie-defense/server/pkg/handlers"
	"ai-zombie-defense/server/pkg/logging"
	"ai-zombie-defense/server/pkg/middleware"
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

	// Initialize Mega-Service (facade)
	authService := auth.NewService(*cfg, logger, dbConn)

	// Initialize API Gateway
	gw := gateway.NewAPIGateway(*cfg, logger)

	// Register routes
	registerRoutes(gw, authService, cfg, logger)

	// Start server in background
	go func() {
		addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
		logger.Info("Starting server", zap.String("address", addr))
		if err := gw.Router().Listen(addr); err != nil {
			logger.Fatal("Gateway failed", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := gw.Router().ShutdownWithContext(ctx); err != nil {
		logger.Error("Server shutdown failed", zap.Error(err))
	}

	logger.Info("Server stopped")
}

func registerRoutes(gw *gateway.APIGateway, authService *auth.Service, cfg *config.Config, logger *zap.Logger) {
	// Auth routes
	authHandlers := handlers.NewAuthHandlers(authService, *cfg, logger)
	authGroup := gw.MountGroup("/auth")
	authGroup.Post("/login", authHandlers.Login)
	authGroup.Post("/register", authHandlers.Register)
	authGroup.Post("/refresh", authHandlers.Refresh)
	authGroup.Post("/logout", authHandlers.Logout)

	// Protected routes
	authMiddleware := middleware.AuthMiddleware(authService, logger)

	// Account routes
	accountHandlers := handlers.NewAccountHandlers(authService, authService, authService, logger)
	accountGroup := gw.MountGroup("/account", authMiddleware)
	accountGroup.Get("/profile", accountHandlers.GetProfile)
	accountGroup.Put("/profile", accountHandlers.UpdateProfile)
	accountGroup.Get("/settings", accountHandlers.GetSettings)
	accountGroup.Put("/settings", accountHandlers.UpdateSettings)
	accountGroup.Get("/progression", accountHandlers.GetProgression)

	// Progression routes
	progressionGroup := gw.MountGroup("/progression", authMiddleware)
	progressionGroup.Get("/", accountHandlers.GetProgression)
	progressionGroup.Get("/currency", accountHandlers.GetCurrencyBalance)
	progressionGroup.Post("/prestige", accountHandlers.PrestigePlayer)

	// Cosmetics routes
	cosmeticsGroup := gw.MountGroup("/cosmetics", authMiddleware)
	cosmeticsGroup.Get("/catalog", accountHandlers.GetCosmeticCatalog)
	cosmeticsGroup.Get("/owned", accountHandlers.GetPlayerCosmetics)
	cosmeticsGroup.Put("/equip", accountHandlers.EquipCosmetic)
	cosmeticsGroup.Post("/purchase", accountHandlers.PurchaseCosmetic)

	// Matches routes
	matchesGroup := gw.MountGroup("/matches", authMiddleware)
	matchesGroup.Post("/", accountHandlers.StoreMatch)
	matchesGroup.Get("/history", accountHandlers.GetMatchHistory)

	// Server routes
	serverHandlers := handlers.NewServerHandlers(authService, logger)
	serversGroup := gw.MountGroup("/servers")
	serversGroup.Post("/register", serverHandlers.RegisterServer)
	serversGroup.Get("/", serverHandlers.ListServers)
	serversGroup.Put("/:id/heartbeat", middleware.ServerAuthMiddleware(authService, logger), serverHandlers.UpdateHeartbeat)
	serversGroup.Post("/:id/join", authMiddleware, serverHandlers.GenerateJoinToken)
	serversGroup.Post("/join-token/:token/validate", middleware.ServerAuthMiddleware(authService, logger), serverHandlers.ValidateJoinToken)

	// Favorites routes
	favoriteHandlers := handlers.NewFavoriteHandlers(authService, logger)
	favoritesGroup := gw.MountGroup("/favorites", authMiddleware)
	favoritesGroup.Post("/", favoriteHandlers.AddFavorite)
	favoritesGroup.Get("/", favoriteHandlers.ListFavorites)
	favoritesGroup.Delete("/:id", favoriteHandlers.RemoveFavorite)

	// Friends routes
	friendHandlers := handlers.NewFriendHandlers(authService, logger)
	friendsGroup := gw.MountGroup("/friends", authMiddleware)
	friendsGroup.Post("/request", friendHandlers.SendFriendRequest)
	friendsGroup.Put("/:id", friendHandlers.UpdateFriendRequest)
	friendsGroup.Get("/", friendHandlers.ListFriends)

	// Leaderboard routes
	leaderboardHandlers := handlers.NewLeaderboardHandlers(authService, logger)
	leaderboardsGroup := gw.MountGroup("/leaderboards")
	leaderboardsGroup.Get("/daily", leaderboardHandlers.GetDailyLeaderboard)
	leaderboardsGroup.Get("/weekly", leaderboardHandlers.GetWeeklyLeaderboard)
	leaderboardsGroup.Get("/alltime", leaderboardHandlers.GetAllTimeLeaderboard)

	// Loot routes
	lootHandlers := handlers.NewLootHandlers(authService, logger)
	lootGroup := gw.MountGroup("/loot", authMiddleware)
	lootGroup.Post("/drop", lootHandlers.GenerateLootDrop)

	// Admin routes
	lootTableHandlers := handlers.NewLootTableHandlers(authService, logger)
	adminGroup := gw.MountGroup("/admin", authMiddleware, middleware.AdminMiddleware(authService, logger))
	adminGroup.Get("/loot-tables", lootTableHandlers.ListLootTables)
	adminGroup.Post("/loot-tables", lootTableHandlers.CreateLootTable)
	adminGroup.Get("/loot-tables/:id", lootTableHandlers.GetLootTable)
	adminGroup.Put("/loot-tables/:id", lootTableHandlers.UpdateLootTable)
	adminGroup.Delete("/loot-tables/:id", lootTableHandlers.DeleteLootTable)
	adminGroup.Get("/loot-tables/:id/entries", lootTableHandlers.ListLootTableEntries)
	adminGroup.Post("/loot-tables/:id/entries", lootTableHandlers.CreateLootTableEntry)
	adminGroup.Get("/loot-tables/entries/:entryId", lootTableHandlers.GetLootTableEntry)
	adminGroup.Put("/loot-tables/entries/:entryId", lootTableHandlers.UpdateLootTableEntry)
	adminGroup.Delete("/loot-tables/entries/:entryId", lootTableHandlers.DeleteLootTableEntry)
}
