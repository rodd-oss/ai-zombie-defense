package middleware

import (
	"errors"
	"strconv"

	"ai-zombie-defense/backend-api/internal/services/server"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

const (
	// ServerIDKey is the key used to store server ID in Fiber's locals.
	ServerIDKey = "server_id"
)

var (
	// ErrMissingServerToken indicates the X-Server-Token header is missing.
	ErrMissingServerToken = errors.New("missing server token")
	// ErrInvalidServerToken indicates the token is invalid or server not found.
	ErrInvalidServerToken = errors.New("invalid server token")
	// ErrServerMismatch indicates the token does not match the requested server ID.
	ErrServerMismatch = errors.New("server token does not match server ID")
)

// ServerAuthMiddleware creates a middleware that validates server authentication token.
func ServerAuthMiddleware(serverService server.Service, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract server ID from path parameter
		serverIDStr := c.Params("id")
		if serverIDStr == "" {
			logger.Debug("missing server ID in path")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "server ID is required",
			})
		}

		serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
		if err != nil {
			logger.Debug("invalid server ID format", zap.String("server_id", serverIDStr), zap.Error(err))
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "invalid server ID format",
			})
		}

		// Extract token from X-Server-Token header
		token := c.Get("X-Server-Token")
		if token == "" {
			logger.Debug("missing X-Server-Token header")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": ErrMissingServerToken.Error(),
			})
		}

		// Look up server by auth token
		server, err := serverService.GetServerByAuthToken(c.Context(), token)
		if err != nil {
			logger.Debug("server lookup failed", zap.Error(err))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": ErrInvalidServerToken.Error(),
			})
		}

		// Verify server ID matches
		if server.ServerID != serverID {
			logger.Debug("server ID mismatch",
				zap.Int64("token_server_id", server.ServerID),
				zap.Int64("path_server_id", serverID))
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": ErrServerMismatch.Error(),
			})
		}

		// Store server ID in locals for downstream handlers
		c.Locals(ServerIDKey, serverID)

		logger.Debug("server authentication successful", zap.Int64("server_id", serverID))
		return c.Next()
	}
}

// GetServerID retrieves server ID from Fiber's locals.
func GetServerID(c *fiber.Ctx) (int64, bool) {
	serverID, ok := c.Locals(ServerIDKey).(int64)
	return serverID, ok
}
