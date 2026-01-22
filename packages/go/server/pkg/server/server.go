package server

import (
	"context"
	"fmt"

	"ai-zombie-defense/db"
	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/config"
	"ai-zombie-defense/server/pkg/handlers"
	"ai-zombie-defense/server/pkg/middleware"

	"github.com/gofiber/fiber/v2"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

// Server holds the Fiber application and configuration.
type Server struct {
	app    *fiber.App
	cfg    config.Config
	logger *zap.Logger
	db     db.DBTX
}

// New creates a new Fiber server with default middleware and routes.
func New(cfg config.Config, logger *zap.Logger, db db.DBTX) *Server {
	// Create Fiber app with default settings
	app := fiber.New(fiber.Config{
		AppName: "AI Zombie Defense",
	})

	// Add default middleware
	app.Use(fiberLogger.New()) // Request logging
	app.Use(recover.New())     // Panic recovery

	// Create server instance
	srv := &Server{
		app:    app,
		cfg:    cfg,
		logger: logger,
		db:     db,
	}

	// Register routes
	srv.registerRoutes()

	return srv
}

// registerRoutes sets up all HTTP routes.
func (s *Server) registerRoutes() {
	// Health check endpoint
	s.app.Get("/health", func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status": "ok",
		})
	})

	// Auth routes
	if s.db != nil {
		authService := auth.NewService(s.cfg, s.logger, s.db)
		authHandlers := handlers.NewAuthHandlers(authService, s.logger)

		authGroup := s.app.Group("/auth")
		authGroup.Post("/login", authHandlers.Login)
		authGroup.Post("/register", authHandlers.Register)
		authGroup.Post("/refresh", authHandlers.Refresh)
		authGroup.Post("/logout", authHandlers.Logout)

		// Account routes (protected by JWT middleware)
		accountHandlers := handlers.NewAccountHandlers(authService, s.logger)
		accountGroup := s.app.Group("/account", middleware.AuthMiddleware(authService, s.logger))
		accountGroup.Get("/profile", accountHandlers.GetProfile)
		accountGroup.Put("/profile", accountHandlers.UpdateProfile)
	}
}

// App returns the underlying Fiber application.
func (s *Server) App() *fiber.App {
	return s.app
}

// Start begins listening on the configured host and port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.logger.Info("Starting server", zap.String("address", addr))
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
