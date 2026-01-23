package middleware

import (
	"errors"

	"ai-zombie-defense/server/pkg/auth"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

var (
	// ErrNotAdmin indicates the player is not an administrator.
	ErrNotAdmin = errors.New("administrator access required")
)

// AdminMiddleware creates a middleware that requires the player to be an administrator.
// This middleware expects that AuthMiddleware has already run and stored player_id in locals.
func AdminMiddleware(authService *auth.Service, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Retrieve player ID from locals (set by AuthMiddleware)
		playerID, ok := GetPlayerID(c)
		if !ok {
			logger.Debug("missing player ID in admin middleware")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authentication required",
			})
		}

		// Check if player is admin
		isAdmin, err := authService.IsAdmin(c.Context(), playerID)
		if err != nil {
			logger.Error("failed to check admin status", zap.Int64("player_id", playerID), zap.Error(err))
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "internal server error",
			})
		}
		if !isAdmin {
			logger.Debug("player is not admin", zap.Int64("player_id", playerID))
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": ErrNotAdmin.Error(),
			})
		}

		logger.Debug("admin access granted", zap.Int64("player_id", playerID))
		return c.Next()
	}
}
