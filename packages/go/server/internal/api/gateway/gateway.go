package gateway

import (
	"ai-zombie-defense/server/pkg/config"
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
}

// NewAPIGateway creates a new instance of APIGateway with a configured Fiber router.
func NewAPIGateway(cfg config.Config, logger *zap.Logger) *APIGateway {
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
	}

	gw.applyMiddleware()
	gw.setupHealthCheck()

	return gw
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

// Router returns the underlying Fiber app (useful for testing or starting the server).
func (g *APIGateway) Router() *fiber.App {
	return g.router
}
