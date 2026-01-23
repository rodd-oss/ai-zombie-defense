package middleware

import (
	"errors"
	"fmt"
	"strings"

	"ai-zombie-defense/server/internal/services/auth"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const (
	// PlayerIDKey is the key used to store player ID in Fiber's locals.
	PlayerIDKey = "player_id"
	// ClaimsKey is the key used to store JWT claims in Fiber's locals.
	ClaimsKey = "claims"
)

var (
	// ErrMissingToken indicates the Authorization header is missing or malformed.
	ErrMissingToken = errors.New("missing or malformed authorization header")
	// ErrInvalidToken indicates the token is invalid or expired.
	ErrInvalidToken = errors.New("invalid or expired token")
)

// AuthMiddleware creates a middleware that validates JWT tokens.
func AuthMiddleware(authService auth.Service, logger *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// Extract token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			logger.Debug("missing Authorization header")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": ErrMissingToken.Error(),
			})
		}

		// Check Bearer prefix
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			logger.Debug("malformed Authorization header", zap.String("header", authHeader))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": ErrMissingToken.Error(),
			})
		}

		tokenString := parts[1]
		if tokenString == "" {
			logger.Debug("empty token")
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": ErrMissingToken.Error(),
			})
		}

		// Validate token
		claims, err := authService.ValidateToken(tokenString)
		if err != nil {
			logger.Debug("token validation failed", zap.Error(err))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": ErrInvalidToken.Error(),
			})
		}

		// Extract player ID from subject claim
		playerID, err := parsePlayerID(claims.Subject)
		if err != nil {
			logger.Debug("invalid player ID in token", zap.String("subject", claims.Subject), zap.Error(err))
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": ErrInvalidToken.Error(),
			})
		}

		// Store player ID and claims in locals for downstream handlers
		c.Locals(PlayerIDKey, playerID)
		c.Locals(ClaimsKey, claims)

		logger.Debug("token validated", zap.Int64("player_id", playerID))
		return c.Next()
	}
}

// parsePlayerID converts a string subject to int64 player ID.
func parsePlayerID(subject string) (int64, error) {
	if subject == "" {
		return 0, errors.New("empty subject")
	}
	var playerID int64
	_, err := fmt.Sscanf(subject, "%d", &playerID)
	if err != nil {
		return 0, err
	}
	return playerID, nil
}

// GetPlayerID retrieves player ID from Fiber's locals.
func GetPlayerID(c *fiber.Ctx) (int64, bool) {
	playerID, ok := c.Locals(PlayerIDKey).(int64)
	return playerID, ok
}

// GetClaims retrieves JWT claims from Fiber's locals.
func GetClaims(c *fiber.Ctx) (*jwt.RegisteredClaims, bool) {
	claims, ok := c.Locals(ClaimsKey).(*jwt.RegisteredClaims)
	return claims, ok
}
