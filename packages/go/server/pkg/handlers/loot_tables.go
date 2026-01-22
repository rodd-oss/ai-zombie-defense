package handlers

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/server/pkg/auth"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type LootTableHandlers struct {
	service *auth.Service
	logger  *zap.Logger
}

func NewLootTableHandlers(service *auth.Service, logger *zap.Logger) *LootTableHandlers {
	return &LootTableHandlers{
		service: service,
		logger:  logger,
	}
}

// Request/Response types

type CreateLootTableRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	DropChance  float64 `json:"drop_chance"`
	IsActive    bool    `json:"is_active"`
}

type LootTableResponse struct {
	LootTableID int64   `json:"loot_table_id"`
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	DropChance  float64 `json:"drop_chance"`
	IsActive    bool    `json:"is_active"`
	CreatedAt   string  `json:"created_at"`
}

type UpdateLootTableRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	DropChance  float64 `json:"drop_chance"`
	IsActive    bool    `json:"is_active"`
}

type CreateLootTableEntryRequest struct {
	CosmeticID  int64 `json:"cosmetic_id"`
	Weight      int64 `json:"weight"`
	MinQuantity int64 `json:"min_quantity"`
	MaxQuantity int64 `json:"max_quantity"`
}

type LootTableEntryResponse struct {
	LootEntryID int64 `json:"loot_entry_id"`
	LootTableID int64 `json:"loot_table_id"`
	CosmeticID  int64 `json:"cosmetic_id"`
	Weight      int64 `json:"weight"`
	MinQuantity int64 `json:"min_quantity"`
	MaxQuantity int64 `json:"max_quantity"`
}

type UpdateLootTableEntryRequest struct {
	LootTableID int64 `json:"loot_table_id"`
	CosmeticID  int64 `json:"cosmetic_id"`
	Weight      int64 `json:"weight"`
	MinQuantity int64 `json:"min_quantity"`
	MaxQuantity int64 `json:"max_quantity"`
}

// Helper function to convert db.LootTable to LootTableResponse
func lootTableToResponse(lt *db.LootTable) LootTableResponse {
	isActive := lt.IsActive == 1
	return LootTableResponse{
		LootTableID: lt.LootTableID,
		Name:        lt.Name,
		Description: lt.Description,
		DropChance:  lt.DropChance,
		IsActive:    isActive,
		CreatedAt:   lt.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
	}
}

// Helper function to convert db.LootTableEntry to LootTableEntryResponse
func lootTableEntryToResponse(lte *db.LootTableEntry) LootTableEntryResponse {
	return LootTableEntryResponse{
		LootEntryID: lte.LootEntryID,
		LootTableID: lte.LootTableID,
		CosmeticID:  lte.CosmeticID,
		Weight:      lte.Weight,
		MinQuantity: lte.MinQuantity,
		MaxQuantity: lte.MaxQuantity,
	}
}

// ListLootTables handles GET /admin/loot-tables
func (h *LootTableHandlers) ListLootTables(c *fiber.Ctx) error {
	ctx := c.Context()
	tables, err := h.service.ListLootTables(ctx)
	if err != nil {
		h.logger.Error("failed to list loot tables", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve loot tables",
		})
	}
	responses := make([]LootTableResponse, len(tables))
	for i, table := range tables {
		responses[i] = lootTableToResponse(table)
	}
	return c.JSON(fiber.Map{
		"loot_tables": responses,
	})
}

// GetLootTable handles GET /admin/loot-tables/:id
func (h *LootTableHandlers) GetLootTable(c *fiber.Ctx) error {
	ctx := c.Context()
	idStr := c.Params("id")
	lootTableID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid loot table ID",
		})
	}
	table, err := h.service.GetLootTable(ctx, lootTableID)
	if err != nil {
		if errors.Is(err, auth.ErrLootTableNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "loot table not found",
			})
		}
		h.logger.Error("failed to get loot table", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve loot table",
		})
	}
	return c.JSON(lootTableToResponse(table))
}

// CreateLootTable handles POST /admin/loot-tables
func (h *LootTableHandlers) CreateLootTable(c *fiber.Ctx) error {
	ctx := c.Context()
	var req CreateLootTableRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	// Validate drop chance
	if req.DropChance < 0 || req.DropChance > 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "drop_chance must be between 0 and 1",
		})
	}
	table, err := h.service.CreateLootTable(ctx, req.Name, req.Description, req.DropChance, req.IsActive)
	if err != nil {
		h.logger.Error("failed to create loot table", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create loot table",
		})
	}
	return c.Status(fiber.StatusCreated).JSON(lootTableToResponse(table))
}

// UpdateLootTable handles PUT /admin/loot-tables/:id
func (h *LootTableHandlers) UpdateLootTable(c *fiber.Ctx) error {
	ctx := c.Context()
	idStr := c.Params("id")
	lootTableID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid loot table ID",
		})
	}
	var req UpdateLootTableRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.DropChance < 0 || req.DropChance > 1 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "drop_chance must be between 0 and 1",
		})
	}
	err = h.service.UpdateLootTable(ctx, lootTableID, req.Name, req.Description, req.DropChance, req.IsActive)
	if err != nil {
		if errors.Is(err, auth.ErrLootTableNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "loot table not found",
			})
		}
		h.logger.Error("failed to update loot table", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update loot table",
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// DeleteLootTable handles DELETE /admin/loot-tables/:id
func (h *LootTableHandlers) DeleteLootTable(c *fiber.Ctx) error {
	ctx := c.Context()
	idStr := c.Params("id")
	lootTableID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid loot table ID",
		})
	}
	err = h.service.DeleteLootTable(ctx, lootTableID)
	if err != nil {
		if errors.Is(err, auth.ErrLootTableNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "loot table not found",
			})
		}
		h.logger.Error("failed to delete loot table", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete loot table",
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// ListLootTableEntries handles GET /admin/loot-tables/:id/entries
func (h *LootTableHandlers) ListLootTableEntries(c *fiber.Ctx) error {
	ctx := c.Context()
	idStr := c.Params("id")
	lootTableID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid loot table ID",
		})
	}
	entries, err := h.service.GetLootTableEntriesByLootTableID(ctx, lootTableID)
	if err != nil {
		h.logger.Error("failed to list loot table entries", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve loot table entries",
		})
	}
	responses := make([]LootTableEntryResponse, len(entries))
	for i, entry := range entries {
		responses[i] = lootTableEntryToResponse(entry)
	}
	return c.JSON(fiber.Map{
		"entries": responses,
	})
}

// CreateLootTableEntry handles POST /admin/loot-tables/:id/entries
func (h *LootTableHandlers) CreateLootTableEntry(c *fiber.Ctx) error {
	ctx := c.Context()
	idStr := c.Params("id")
	lootTableID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid loot table ID",
		})
	}
	var req CreateLootTableEntryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	// Validate quantity range
	if req.MinQuantity < 1 || req.MaxQuantity < req.MinQuantity {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid quantity range",
		})
	}
	entry, err := h.service.CreateLootTableEntry(ctx, lootTableID, req.CosmeticID, req.Weight, req.MinQuantity, req.MaxQuantity)
	if err != nil {
		h.logger.Error("failed to create loot table entry", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to create loot table entry",
		})
	}
	return c.Status(fiber.StatusCreated).JSON(lootTableEntryToResponse(entry))
}

// GetLootTableEntry handles GET /admin/loot-tables/entries/:entryId
func (h *LootTableHandlers) GetLootTableEntry(c *fiber.Ctx) error {
	ctx := c.Context()
	entryIDStr := c.Params("entryId")
	entryID, err := strconv.ParseInt(entryIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid loot table entry ID",
		})
	}
	entry, err := h.service.GetLootTableEntry(ctx, entryID)
	if err != nil {
		if errors.Is(err, auth.ErrLootTableEntryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "loot table entry not found",
			})
		}
		h.logger.Error("failed to get loot table entry", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to retrieve loot table entry",
		})
	}
	return c.JSON(lootTableEntryToResponse(entry))
}

// UpdateLootTableEntry handles PUT /admin/loot-tables/entries/:entryId
func (h *LootTableHandlers) UpdateLootTableEntry(c *fiber.Ctx) error {
	ctx := c.Context()
	entryIDStr := c.Params("entryId")
	entryID, err := strconv.ParseInt(entryIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid loot table entry ID",
		})
	}
	var req UpdateLootTableEntryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.MinQuantity < 1 || req.MaxQuantity < req.MinQuantity {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid quantity range",
		})
	}
	err = h.service.UpdateLootTableEntry(ctx, entryID, req.LootTableID, req.CosmeticID, req.Weight, req.MinQuantity, req.MaxQuantity)
	if err != nil {
		if errors.Is(err, auth.ErrLootTableEntryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "loot table entry not found",
			})
		}
		h.logger.Error("failed to update loot table entry", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to update loot table entry",
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// DeleteLootTableEntry handles DELETE /admin/loot-tables/entries/:entryId
func (h *LootTableHandlers) DeleteLootTableEntry(c *fiber.Ctx) error {
	ctx := c.Context()
	entryIDStr := c.Params("entryId")
	entryID, err := strconv.ParseInt(entryIDStr, 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid loot table entry ID",
		})
	}
	err = h.service.DeleteLootTableEntry(ctx, entryID)
	if err != nil {
		if errors.Is(err, auth.ErrLootTableEntryNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "loot table entry not found",
			})
		}
		h.logger.Error("failed to delete loot table entry", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to delete loot table entry",
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
