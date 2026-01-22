package handlers

import (
	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/middleware"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type LootHandlers struct {
	service *auth.Service
	logger  *zap.Logger
}

func NewLootHandlers(service *auth.Service, logger *zap.Logger) *LootHandlers {
	return &LootHandlers{
		service: service,
		logger:  logger,
	}
}

type CosmeticDropResponse struct {
	CosmeticID     int64   `json:"cosmetic_id"`
	Name           string  `json:"name"`
	Description    *string `json:"description,omitempty"`
	Slot           string  `json:"slot"`
	Category       *string `json:"category,omitempty"`
	Rarity         string  `json:"rarity"`
	UnlockLevel    int64   `json:"unlock_level"`
	DataCost       int64   `json:"data_cost"`
	IsPrestigeOnly bool    `json:"is_prestige_only"`
	CreatedAt      string  `json:"created_at"`
}

// GenerateLootDrop handles POST /loot/drop
func (h *LootHandlers) GenerateLootDrop(c *fiber.Ctx) error {
	ctx := c.Context()
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("failed to get player ID from context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	cosmetic, err := h.service.GenerateLootDrop(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to generate loot drop", zap.Error(err))
		// Determine appropriate status code
		if err.Error() == "no active loot tables" ||
			err.Error() == "no drop from any loot table" ||
			err.Error() == "loot table has no entries" ||
			err.Error() == "total weight must be positive" ||
			err.Error() == "cosmetic not found" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to generate loot drop",
		})
	}

	// Convert to response
	isPrestigeOnly := cosmetic.IsPrestigeOnly == 1
	response := CosmeticDropResponse{
		CosmeticID:     cosmetic.CosmeticID,
		Name:           cosmetic.Name,
		Description:    cosmetic.Description,
		Slot:           cosmetic.Slot,
		Category:       cosmetic.Category,
		Rarity:         cosmetic.Rarity,
		UnlockLevel:    cosmetic.UnlockLevel,
		DataCost:       cosmetic.DataCost,
		IsPrestigeOnly: isPrestigeOnly,
		CreatedAt:      cosmetic.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}

	return c.JSON(response)
}
