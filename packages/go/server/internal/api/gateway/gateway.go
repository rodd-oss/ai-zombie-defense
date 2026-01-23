package gateway

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/server/internal/middleware"
	"ai-zombie-defense/server/internal/services/account"
	accHandlers "ai-zombie-defense/server/internal/services/account/handlers"
	"ai-zombie-defense/server/internal/services/auth"
	authHandlers "ai-zombie-defense/server/internal/services/auth/handlers"
	"ai-zombie-defense/server/internal/services/leaderboard"
	lbHandlers "ai-zombie-defense/server/internal/services/leaderboard/handlers"
	"ai-zombie-defense/server/internal/services/loot"
	lootHandlers "ai-zombie-defense/server/internal/services/loot/handlers"
	"ai-zombie-defense/server/internal/services/match"
	matchHandlers "ai-zombie-defense/server/internal/services/match/handlers"
	"ai-zombie-defense/server/internal/services/progression"
	progHandlers "ai-zombie-defense/server/internal/services/progression/handlers"
	"ai-zombie-defense/server/internal/services/server"
	srvHandlers "ai-zombie-defense/server/internal/services/server/handlers"
	"ai-zombie-defense/server/internal/services/social"
	socialHandlers "ai-zombie-defense/server/internal/services/social/handlers"
	"ai-zombie-defense/server/pkg/config"
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
		authSvc := auth.NewAuthService(cfg, logger, db)
		accSvc := account.NewAccountService(cfg, logger, db)
		progSvc := progression.NewProgressionService(cfg, logger, db)
		lootSvc := loot.NewLootService(cfg, logger, db)
		matchSvc := match.NewMatchService(cfg, logger, db, progSvc)
		serverSvc := server.NewServerService(cfg, logger, db)
		socialSvc := social.NewSocialService(cfg, logger, db)
		lbSvc := leaderboard.NewLeaderboardService(cfg, logger, db)

		gw.registerRoutes(authSvc, accSvc, progSvc, matchSvc, serverSvc, socialSvc, lbSvc, lootSvc)
	}

	return gw
}

func (g *APIGateway) registerRoutes(
	authSvc auth.Service,
	accSvc account.Service,
	progSvc progression.Service,
	matchSvc match.Service,
	serverSvc server.Service,
	socialSvc social.Service,
	lbSvc leaderboard.Service,
	lootSvc loot.Service,
) {
	// Auth routes
	authH := authHandlers.NewAuthHandlers(authSvc, g.cfg, g.logger)
	authGroup := g.MountGroup("/auth")
	authGroup.Post("/login", authH.Login)
	authGroup.Post("/register", authH.Register)
	authGroup.Post("/refresh", authH.Refresh)
	authGroup.Post("/logout", authH.Logout)

	// Protected routes
	authMiddleware := middleware.AuthMiddleware(authSvc, g.logger)

	// Account routes
	accountH := accHandlers.NewAccountHandlers(accSvc, g.logger)
	accountGroup := g.MountGroup("/account", authMiddleware)
	accountGroup.Get("/profile", accountH.GetProfile)
	accountGroup.Put("/profile", accountH.UpdateProfile)
	accountGroup.Get("/settings", accountH.GetSettings)
	accountGroup.Put("/settings", accountH.UpdateSettings)

	// Progression routes
	progressionH := progHandlers.NewProgressionHandlers(progSvc, g.logger)
	progressionGroup := g.MountGroup("/progression", authMiddleware)
	progressionGroup.Get("/", progressionH.GetProgression)
	progressionGroup.Get("/currency", progressionH.GetCurrencyBalance)
	progressionGroup.Post("/prestige", progressionH.PrestigePlayer)
	// Duplicate route for legacy support if needed, but prd says update gateway routing
	accountGroup.Get("/progression", progressionH.GetProgression)

	// Cosmetics routes
	cosmeticsGroup := g.MountGroup("/cosmetics", authMiddleware)
	cosmeticsGroup.Get("/catalog", progressionH.GetCosmeticCatalog)
	cosmeticsGroup.Get("/owned", progressionH.GetPlayerCosmetics)
	cosmeticsGroup.Put("/equip", progressionH.EquipCosmetic)
	cosmeticsGroup.Post("/purchase", progressionH.PurchaseCosmetic)

	// Matches routes
	matchH := matchHandlers.NewMatchHandlers(matchSvc, g.logger)
	matchesGroup := g.MountGroup("/matches", authMiddleware)
	matchesGroup.Post("/", matchH.StoreMatch)
	matchesGroup.Get("/history", matchH.GetMatchHistory)

	// Server routes
	serverH := srvHandlers.NewServerHandlers(serverSvc, g.logger)
	serversGroup := g.MountGroup("/servers")
	serversGroup.Post("/register", serverH.RegisterServer)
	serversGroup.Get("/", serverH.ListServers)
	serversGroup.Put("/:id/heartbeat", middleware.ServerAuthMiddleware(serverSvc, g.logger), serverH.UpdateHeartbeat)
	serversGroup.Post("/:id/join", authMiddleware, serverH.GenerateJoinToken)
	serversGroup.Post("/join-token/:token/validate", middleware.ServerAuthMiddleware(serverSvc, g.logger), serverH.ValidateJoinToken)

	// Favorites routes
	favoriteH := socialHandlers.NewFavoriteHandlers(serverSvc, g.logger)
	favoritesGroup := g.MountGroup("/favorites", authMiddleware)
	favoritesGroup.Post("/", favoriteH.AddFavorite)
	favoritesGroup.Get("/", favoriteH.ListFavorites)
	favoritesGroup.Delete("/:id", favoriteH.RemoveFavorite)

	// Friends routes
	socialH := socialHandlers.NewFriendHandlers(socialSvc, g.logger)
	friendsGroup := g.MountGroup("/friends", authMiddleware)

	friendsGroup.Post("/request", socialH.SendFriendRequest)
	friendsGroup.Put("/:id", socialH.UpdateFriendRequest)
	friendsGroup.Get("/", socialH.ListFriends)

	// Leaderboard routes
	leaderboardH := lbHandlers.NewLeaderboardHandlers(lbSvc, g.logger)
	leaderboardsGroup := g.MountGroup("/leaderboards")
	leaderboardsGroup.Get("/daily", leaderboardH.GetDailyLeaderboard)
	leaderboardsGroup.Get("/weekly", leaderboardH.GetWeeklyLeaderboard)
	leaderboardsGroup.Get("/alltime", leaderboardH.GetAllTimeLeaderboard)

	// Loot routes
	lootH := lootHandlers.NewLootHandlers(lootSvc, g.logger)
	lootGroup := g.MountGroup("/loot", authMiddleware)
	lootGroup.Post("/drop", lootH.GenerateLootDrop)

	// Admin routes
	lootTableH := lootHandlers.NewLootTableHandlers(lootSvc, g.logger)
	adminGroup := g.MountGroup("/admin", authMiddleware, middleware.AdminMiddleware(authSvc, g.logger))
	adminGroup.Get("/loot-tables", lootTableH.ListLootTables)
	adminGroup.Post("/loot-tables", lootTableH.CreateLootTable)
	adminGroup.Get("/loot-tables/:id", lootTableH.GetLootTable)
	adminGroup.Put("/loot-tables/:id", lootTableH.UpdateLootTable)
	adminGroup.Delete("/loot-tables/:id", lootTableH.DeleteLootTable)
	adminGroup.Get("/loot-tables/:id/entries", lootTableH.ListLootTableEntries)
	adminGroup.Post("/loot-tables/:id/entries", lootTableH.CreateLootTableEntry)
	adminGroup.Get("/loot-tables/entries/:entryId", lootTableH.GetLootTableEntry)
	adminGroup.Put("/loot-tables/entries/:entryId", lootTableH.UpdateLootTableEntry)
	adminGroup.Delete("/loot-tables/entries/:entryId", lootTableH.DeleteLootTableEntry)

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
