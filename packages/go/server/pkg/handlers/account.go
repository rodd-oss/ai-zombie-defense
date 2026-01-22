package handlers

import (
	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/middleware"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AccountHandlers struct {
	service *auth.Service
	logger  *zap.Logger
}

func NewAccountHandlers(service *auth.Service, logger *zap.Logger) *AccountHandlers {
	return &AccountHandlers{
		service: service,
		logger:  logger,
	}
}

type ProfileResponse struct {
	PlayerID    int64   `json:"player_id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	CreatedAt   string  `json:"created_at"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
	IsBanned    bool    `json:"is_banned"`
}

type UpdateProfileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// GetProfile handles GET /account/profile
func (h *AccountHandlers) GetProfile(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	ctx := c.Context()
	player, err := h.service.GetPlayer(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	// Convert timestamps to ISO 8601 strings
	createdAt := player.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
	var lastLoginAt *string
	if player.LastLoginAt.Valid {
		str := player.LastLoginAt.Time.Format("2006-01-02T15:04:05Z")
		lastLoginAt = &str
	}

	resp := ProfileResponse{
		PlayerID:    player.PlayerID,
		Username:    player.Username,
		Email:       player.Email,
		CreatedAt:   createdAt,
		LastLoginAt: lastLoginAt,
		IsBanned:    player.IsBanned != 0,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// UpdateProfile handles PUT /account/profile
func (h *AccountHandlers) UpdateProfile(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Username == "" || req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username and email are required",
		})
	}

	ctx := c.Context()
	err := h.service.UpdatePlayerProfile(ctx, playerID, req.Username, req.Email)
	if err != nil {
		if err == auth.ErrDuplicateUsername {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "username already exists",
			})
		}
		if err == auth.ErrDuplicateEmail {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "email already exists",
			})
		}
		h.logger.Error("failed to update player profile", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "profile updated successfully",
	})
}
