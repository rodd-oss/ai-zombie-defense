package gateway

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/config"
	"ai-zombie-defense/server/pkg/handlers"
	"ai-zombie-defense/server/pkg/middleware"
	"context"
	"fmt"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

// APIGateway handles the central routing and global middleware for the modular monolith.
type APIGateway struct {
	router *fiber.App
	logger *zap.Logger
	cfg    config.Config
	db     db.DBTX
}

// NewAPIGateway creates a new instance of APIGateway with a configured Fiber router.
func NewAPIGateway(cfg config.Config, logger *zap.Logger, db db.DBTX) *APIGateway {
	app := fiber.New(fiber.Config{
		AppName: "AI Zombie Defense API Gateway",
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			if code >= fiber.StatusInternalServerError {
				logger.Error("gateway error", zap.Error(err))
			}
			c.Set(fiber.HeaderContentType, fiber.MIMEApplicationJSON)
			return c.Status(code).JSON(fiber.Map{
				"error": err.Error(),
			})
		},
	})

	gw := &APIGateway{
		router: app,
		logger: logger,
		cfg:    cfg,
		db:     db,
	}

	gw.applyMiddleware()
	gw.setupHealthCheck()

	if db != nil {
		authService := auth.NewService(cfg, logger, db)
		gw.registerRoutes(authService)
	}

	return gw
}

func (g *APIGateway) registerRoutes(authService *auth.Service) {
	// Auth routes
	authHandlers := handlers.NewAuthHandlers(authService, g.cfg, g.logger)
	authGroup := g.MountGroup("/auth")
	authGroup.Post("/login", authHandlers.Login)
	authGroup.Post("/register", authHandlers.Register)
	authGroup.Post("/refresh", authHandlers.Refresh)
	authGroup.Post("/logout", authHandlers.Logout)

	// Protected routes
	authMiddleware := middleware.AuthMiddleware(authService, g.logger)

	// Account routes
	accountHandlers := handlers.NewAccountHandlers(authService, authService, authService, g.logger)
	accountGroup := g.MountGroup("/account", authMiddleware)
	accountGroup.Get("/profile", accountHandlers.GetProfile)
	accountGroup.Put("/profile", accountHandlers.UpdateProfile)
	accountGroup.Get("/settings", accountHandlers.GetSettings)
	accountGroup.Put("/settings", accountHandlers.UpdateSettings)
	accountGroup.Get("/progression", accountHandlers.GetProgression)

	// Progression routes
	progressionGroup := g.MountGroup("/progression", authMiddleware)
	progressionGroup.Get("/", accountHandlers.GetProgression)
	progressionGroup.Get("/currency", accountHandlers.GetCurrencyBalance)
	progressionGroup.Post("/prestige", accountHandlers.PrestigePlayer)

	// Cosmetics routes
	cosmeticsGroup := g.MountGroup("/cosmetics", authMiddleware)
	cosmeticsGroup.Get("/catalog", accountHandlers.GetCosmeticCatalog)
	cosmeticsGroup.Get("/owned", accountHandlers.GetPlayerCosmetics)
	cosmeticsGroup.Put("/equip", accountHandlers.EquipCosmetic)
	cosmeticsGroup.Post("/purchase", accountHandlers.PurchaseCosmetic)

	// Matches routes
	matchesGroup := g.MountGroup("/matches", authMiddleware)
	matchesGroup.Post("/", accountHandlers.StoreMatch)
	matchesGroup.Get("/history", accountHandlers.GetMatchHistory)

	// Server routes
	serverHandlers := handlers.NewServerHandlers(authService, g.logger)
	serversGroup := g.MountGroup("/servers")
	serversGroup.Post("/register", serverHandlers.RegisterServer)
	serversGroup.Get("/", serverHandlers.ListServers)
	serversGroup.Put("/:id/heartbeat", middleware.ServerAuthMiddleware(authService, g.logger), serverHandlers.UpdateHeartbeat)
	serversGroup.Post("/:id/join", authMiddleware, serverHandlers.GenerateJoinToken)
	serversGroup.Post("/join-token/:token/validate", middleware.ServerAuthMiddleware(authService, g.logger), serverHandlers.ValidateJoinToken)

	// Favorites routes
	favoriteHandlers := handlers.NewFavoriteHandlers(authService, g.logger)
	favoritesGroup := g.MountGroup("/favorites", authMiddleware)
	favoritesGroup.Post("/", favoriteHandlers.AddFavorite)
	favoritesGroup.Get("/", favoriteHandlers.ListFavorites)
	favoritesGroup.Delete("/:id", favoriteHandlers.RemoveFavorite)

	// Friends routes
	friendHandlers := handlers.NewFriendHandlers(authService, g.logger)
	friendsGroup := g.MountGroup("/friends", authMiddleware)
	friendsGroup.Post("/request", friendHandlers.SendFriendRequest)
	friendsGroup.Put("/:id", friendHandlers.UpdateFriendRequest)
	friendsGroup.Get("/", friendHandlers.ListFriends)

	// Leaderboard routes
	leaderboardHandlers := handlers.NewLeaderboardHandlers(authService, g.logger)
	leaderboardsGroup := g.MountGroup("/leaderboards")
	leaderboardsGroup.Get("/daily", leaderboardHandlers.GetDailyLeaderboard)
	leaderboardsGroup.Get("/weekly", leaderboardHandlers.GetWeeklyLeaderboard)
	leaderboardsGroup.Get("/alltime", leaderboardHandlers.GetAllTimeLeaderboard)

	// Loot routes
	lootHandlers := handlers.NewLootHandlers(authService, g.logger)
	lootGroup := g.MountGroup("/loot", authMiddleware)
	lootGroup.Post("/drop", lootHandlers.GenerateLootDrop)

	// Admin routes
	lootTableHandlers := handlers.NewLootTableHandlers(authService, g.logger)
	adminGroup := g.MountGroup("/admin", authMiddleware, middleware.AdminMiddleware(authService, g.logger))
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

// applyMiddleware sets up global middleware for the gateway.
func (g *APIGateway) applyMiddleware() {
	g.router.Use(cors.New(cors.Config{
		AllowOrigins: g.cfg.Server.CORSAllowOrigins,
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))
	g.router.Use(fiberLogger.New())
	g.router.Use(recover.New())
	g.router.Use(limiter.New(limiter.Config{
		Max:        g.cfg.Server.RateLimitMax,
		Expiration: g.cfg.Server.RateLimitDuration,
	}))
}

// setupHealthCheck adds a basic health check endpoint to the gateway.
func (g *APIGateway) setupHealthCheck() {
	g.router.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})
}

// MountGroup allows services to mount their own route groups on the gateway.
func (g *APIGateway) MountGroup(prefix string, handlers ...fiber.Handler) fiber.Router {
	return g.router.Group(prefix, handlers...)
}

// Router returns the underlying Fiber app (useful for testing).
func (g *APIGateway) Router() *fiber.App {
	return g.router
}

// Start begins listening on the configured host and port.
func (g *APIGateway) Start() error {
	addr := fmt.Sprintf("%s:%d", g.cfg.Server.Host, g.cfg.Server.Port)
	g.logger.Info("Starting API Gateway", zap.String("address", addr))
	return g.router.Listen(addr)
}

// Shutdown gracefully stops the gateway.
func (g *APIGateway) Shutdown(ctx context.Context) error {
	g.logger.Info("Shutting down API Gateway...")
	return g.router.ShutdownWithContext(ctx)
}
