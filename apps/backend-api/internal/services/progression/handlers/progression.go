package handlers

import (
	"ai-zombie-defense/backend-api/internal/middleware"
	"ai-zombie-defense/backend-api/internal/services/progression"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type ProgressionHandlers struct {
	progressionSvc progression.Service
	logger         *zap.Logger
}

func NewProgressionHandlers(progressionSvc progression.Service, logger *zap.Logger) *ProgressionHandlers {
	return &ProgressionHandlers{
		progressionSvc: progressionSvc,
		logger:         logger,
	}
}

type ProgressionResponse struct {
	PlayerID           int64  `json:"player_id"`
	Level              int64  `json:"level"`
	Experience         int64  `json:"experience"`
	PrestigeLevel      int64  `json:"prestige_level"`
	DataCurrency       int64  `json:"data_currency"`
	TotalMatchesPlayed int64  `json:"total_matches_played"`
	TotalWavesSurvived int64  `json:"total_waves_survived"`
	TotalKills         int64  `json:"total_kills"`
	TotalDeaths        int64  `json:"total_deaths"`
	TotalScrapEarned   int64  `json:"total_scrap_earned"`
	TotalDataEarned    int64  `json:"total_data_earned"`
	UpdatedAt          string `json:"updated_at"`
}

type PrestigeResponse struct {
	Message          string  `json:"message"`
	NewPrestigeLevel int64   `json:"new_prestige_level"`
	GrantedCosmetics []int64 `json:"granted_cosmetics"`
}

// GetProgression handles GET /account/progression
func (h *ProgressionHandlers) GetProgression(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	progression, err := h.progressionSvc.GetPlayerProgression(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player progression", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	// Convert timestamp to ISO 8601 string
	updatedAt := progression.UpdatedAt.Time.Format("2006-01-02T15:04:05Z")
	resp := ProgressionResponse{
		PlayerID:           progression.PlayerID,
		Level:              progression.Level,
		Experience:         progression.Experience,
		PrestigeLevel:      progression.PrestigeLevel,
		DataCurrency:       progression.DataCurrency,
		TotalMatchesPlayed: progression.TotalMatchesPlayed,
		TotalWavesSurvived: progression.TotalWavesSurvived,
		TotalKills:         progression.TotalKills,
		TotalDeaths:        progression.TotalDeaths,
		TotalScrapEarned:   progression.TotalScrapEarned,
		TotalDataEarned:    progression.TotalDataEarned,
		UpdatedAt:          updatedAt,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// PrestigePlayer handles POST /progression/prestige
func (h *ProgressionHandlers) PrestigePlayer(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	err := h.progressionSvc.PrestigePlayer(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to prestige player", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	// Get updated progression to include in response
	progression, err := h.progressionSvc.GetPlayerProgression(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player progression after prestige", zap.Error(err), zap.Int64("player_id", playerID))
		// Still return success because prestige succeeded
		return c.Status(fiber.StatusOK).JSON(PrestigeResponse{
			Message:          "prestige successful",
			NewPrestigeLevel: 0, // unknown
			GrantedCosmetics: []int64{},
		})
	}
	// For simplicity, we don't return granted cosmetics list (could be fetched via separate query)
	return c.Status(fiber.StatusOK).JSON(PrestigeResponse{
		Message:          "prestige successful",
		NewPrestigeLevel: progression.PrestigeLevel,
		GrantedCosmetics: []int64{},
	})
}

// GetCurrencyBalance handles GET /progression/currency
func (h *ProgressionHandlers) GetCurrencyBalance(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	progression, err := h.progressionSvc.GetPlayerProgression(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player progression", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data_currency": progression.DataCurrency,
	})
}

// GetCosmeticCatalog handles GET /cosmetics/catalog
func (h *ProgressionHandlers) GetCosmeticCatalog(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	items, err := h.progressionSvc.GetCosmeticCatalog(ctx)
	if err != nil {
		h.logger.Error("failed to get cosmetic catalog", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(items)
}

// GetPlayerCosmetics handles GET /cosmetics/owned
func (h *ProgressionHandlers) GetPlayerCosmetics(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	items, err := h.progressionSvc.GetPlayerCosmetics(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player cosmetics", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(items)
}

// EquipCosmetic handles PUT /cosmetics/equip
func (h *ProgressionHandlers) EquipCosmetic(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	var req struct {
		CosmeticID int64 `json:"cosmetic_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.CosmeticID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cosmetic_id must be positive",
		})
	}
	ctx := c.Context()
	err := h.progressionSvc.EquipCosmetic(ctx, playerID, req.CosmeticID)
	if err != nil {
		if err == progression.ErrCosmeticNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "cosmetic not found",
			})
		}
		if err == progression.ErrCosmeticNotOwned {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "cosmetic not owned",
			})
		}
		if err == progression.ErrLoadoutNotFound {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "loadout not found",
			})
		}
		h.logger.Error("failed to equip cosmetic", zap.Error(err), zap.Int64("player_id", playerID), zap.Int64("cosmetic_id", req.CosmeticID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "cosmetic equipped successfully",
	})
}

// PurchaseCosmetic handles POST /cosmetics/purchase
func (h *ProgressionHandlers) PurchaseCosmetic(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var req struct {
		CosmeticID int64 `json:"cosmetic_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.CosmeticID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cosmetic_id must be positive",
		})
	}

	ctx := c.Context()
	err := h.progressionSvc.PurchaseCosmetic(ctx, playerID, req.CosmeticID)
	if err != nil {
		if err == progression.ErrCosmeticNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "cosmetic not found",
			})
		}
		if err == progression.ErrInsufficientCurrency {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error": "insufficient data currency",
			})
		}
		if err == progression.ErrCosmeticAlreadyOwned {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "cosmetic already owned",
			})
		}
		h.logger.Error("failed to purchase cosmetic", zap.Error(err), zap.Int64("player_id", playerID), zap.Int64("cosmetic_id", req.CosmeticID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "cosmetic purchased successfully",
	})
}
