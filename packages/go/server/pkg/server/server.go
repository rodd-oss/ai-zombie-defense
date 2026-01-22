package server

import (
	"context"
	"fmt"

	"ai-zombie-defense/server/pkg/config"

	"github.com/gofiber/fiber/v2"
	fiberLogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"
)

// Server holds the Fiber application and configuration.
type Server struct {
	app    *fiber.App
	config config.ServerConfig
	logger *zap.Logger
}

// New creates a new Fiber server with default middleware and routes.
func New(cfg config.Config, logger *zap.Logger) *Server {
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
		config: cfg.Server,
		logger: logger,
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
}

// App returns the underlying Fiber application.
func (s *Server) App() *fiber.App {
	return s.app
}

// Start begins listening on the configured host and port.
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	s.logger.Info("Starting server", zap.String("address", addr))
	return s.app.Listen(addr)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}
