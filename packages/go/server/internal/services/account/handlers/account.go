package handlers

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/server/internal/middleware"
	"ai-zombie-defense/server/internal/services/account"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AccountHandlers struct {
	accSvc account.Service
	logger *zap.Logger
}

func NewAccountHandlers(accSvc account.Service, logger *zap.Logger) *AccountHandlers {
	return &AccountHandlers{
		accSvc: accSvc,
		logger: logger,
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

type SettingsResponse struct {
	PlayerID         int64    `json:"player_id"`
	KeyBindings      *string  `json:"key_bindings,omitempty"`
	MouseSensitivity *float64 `json:"mouse_sensitivity,omitempty"`
	UiScale          *float64 `json:"ui_scale,omitempty"`
	ColorBlindMode   int64    `json:"color_blind_mode"`
	SubtitlesEnabled int64    `json:"subtitles_enabled"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type UpdateSettingsRequest struct {
	KeyBindings      *string  `json:"key_bindings"`
	MouseSensitivity *float64 `json:"mouse_sensitivity"`
	UiScale          *float64 `json:"ui_scale"`
	ColorBlindMode   int64    `json:"color_blind_mode"`
	SubtitlesEnabled int64    `json:"subtitles_enabled"`
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
	player, err := h.accSvc.GetPlayer(ctx, playerID)
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
	err := h.accSvc.UpdatePlayerProfile(ctx, playerID, req.Username, req.Email)
	if err != nil {
		if err == account.ErrDuplicateUsername {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "username already exists",
			})
		}
		if err == account.ErrDuplicateEmail {
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

// GetSettings handles GET /account/settings
func (h *AccountHandlers) GetSettings(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	settings, err := h.accSvc.GetPlayerSettings(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player settings", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	// Convert timestamps to ISO 8601 strings
	createdAt := settings.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
	updatedAt := settings.UpdatedAt.Time.Format("2006-01-02T15:04:05Z")
	resp := SettingsResponse{
		PlayerID:         settings.PlayerID,
		KeyBindings:      settings.KeyBindings,
		MouseSensitivity: settings.MouseSensitivity,
		UiScale:          settings.UiScale,
		ColorBlindMode:   settings.ColorBlindMode,
		SubtitlesEnabled: settings.SubtitlesEnabled,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// UpdateSettings handles PUT /account/settings
func (h *AccountHandlers) UpdateSettings(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	var req UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	params := &db.UpsertPlayerSettingsParams{
		PlayerID:         playerID,
		KeyBindings:      req.KeyBindings,
		MouseSensitivity: req.MouseSensitivity,
		UiScale:          req.UiScale,
		ColorBlindMode:   req.ColorBlindMode,
		SubtitlesEnabled: req.SubtitlesEnabled,
	}
	ctx := c.Context()
	err := h.accSvc.UpsertPlayerSettings(ctx, params)
	if err != nil {
		h.logger.Error("failed to upsert player settings", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "settings updated successfully",
	})
}
