package handlers

import (
	"ai-zombie-defense/backend-api/internal/services/account"
	"ai-zombie-defense/backend-api/internal/services/auth"
	"ai-zombie-defense/backend-api/pkg/config"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AuthHandlers struct {
	service auth.Service
	config  config.Config
	logger  *zap.Logger
}

func NewAuthHandlers(service auth.Service, cfg config.Config, logger *zap.Logger) *AuthHandlers {
	return &AuthHandlers{
		service: service,
		config:  cfg,
		logger:  logger,
	}
}

type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email"`
	Password        string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	PlayerID     int64     `json:"player_id"`
}

type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	PlayerID     int64     `json:"player_id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
}

// Login handles POST /auth/login
func (h *AuthHandlers) Login(c *fiber.Ctx) error {
	var req LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.UsernameOrEmail == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username_or_email and password are required",
		})
	}

	ctx := c.Context()
	player, err := h.service.Authenticate(ctx, req.UsernameOrEmail, req.Password)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid credentials",
			})
		}
		if err == auth.ErrPlayerBanned {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "player is banned",
			})
		}
		h.logger.Error("authentication failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	accessToken, err := h.service.GenerateAccessToken(player.PlayerID)
	if err != nil {
		h.logger.Error("failed to generate access token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	ip := c.IP()
	userAgent := c.Get("User-Agent")
	refreshToken, err := h.service.CreateSession(ctx, player.PlayerID, ip, userAgent)
	if err != nil {
		h.logger.Error("failed to create session", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	exp := time.Now().Add(h.config.JWT.AccessExpiration)
	resp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    exp,
		PlayerID:     player.PlayerID,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// Register handles POST /auth/register
func (h *AuthHandlers) Register(c *fiber.Ctx) error {
	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Username == "" || req.Email == "" || req.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username, email, and password are required",
		})
	}

	ctx := c.Context()
	player, err := h.service.RegisterPlayer(ctx, req.Username, req.Email, req.Password)
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
		h.logger.Error("registration failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	accessToken, err := h.service.GenerateAccessToken(player.PlayerID)
	if err != nil {
		h.logger.Error("failed to generate access token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	ip := c.IP()
	userAgent := c.Get("User-Agent")
	refreshToken, err := h.service.CreateSession(ctx, player.PlayerID, ip, userAgent)
	if err != nil {
		h.logger.Error("failed to create session", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	exp := time.Now().Add(h.config.JWT.AccessExpiration)
	resp := RegisterResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    exp,
		PlayerID:     player.PlayerID,
		Username:     player.Username,
		Email:        player.Email,
	}
	return c.Status(fiber.StatusCreated).JSON(resp)
}

// Refresh handles POST /auth/refresh
func (h *AuthHandlers) Refresh(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "refresh_token is required",
		})
	}

	ctx := c.Context()
	ip := c.IP()
	userAgent := c.Get("User-Agent")
	playerID, newRefreshToken, err := h.service.RefreshSession(ctx, req.RefreshToken, ip, userAgent)
	if err != nil {
		if err == auth.ErrInvalidRefreshToken || err == auth.ErrSessionNotFound {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid refresh token",
			})
		}
		h.logger.Error("refresh failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	accessToken, err := h.service.GenerateAccessToken(playerID)
	if err != nil {
		h.logger.Error("failed to generate access token", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	exp := time.Now().Add(h.config.JWT.AccessExpiration)
	resp := LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresAt:    exp,
		PlayerID:     playerID,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// Logout handles POST /auth/logout
func (h *AuthHandlers) Logout(c *fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "refresh_token is required",
		})
	}

	ctx := c.Context()
	err := h.service.DeleteSession(ctx, req.RefreshToken)
	if err != nil {
		h.logger.Error("logout failed", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "logged out successfully",
	})
}
