package loot

import (
	"ai-zombie-defense/backend-api/internal/db"
	"ai-zombie-defense/backend-api/pkg/config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	randmath "math/rand"
	"strings"

	"go.uber.org/zap"
)

type lootService struct {
	config  config.Config
	logger  *zap.Logger
	dbConn  db.DBTX
	queries *db.Queries
}

func NewLootService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) Service {
	return &lootService{
		config:  cfg,
		logger:  logger,
		dbConn:  dbConn,
		queries: db.New(),
	}
}

func (s *lootService) CreateLootTable(ctx context.Context, name string, description *string, dropChance float64, isActive bool) (*db.LootTable, error) {
	isActiveInt := int64(0)
	if isActive {
		isActiveInt = 1
	}
	params := &db.CreateLootTableParams{
		Name:        name,
		Description: description,
		DropChance:  dropChance,
		IsActive:    isActiveInt,
	}
	lootTable, err := s.queries.CreateLootTable(ctx, s.dbConn, params)
	if err != nil {
		return nil, fmt.Errorf("failed to create loot table: %w", err)
	}
	return lootTable, nil
}

func (s *lootService) GetLootTable(ctx context.Context, lootTableID int64) (*db.LootTable, error) {
	lootTable, err := s.queries.GetLootTable(ctx, s.dbConn, lootTableID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLootTableNotFound
		}
		return nil, fmt.Errorf("failed to get loot table: %w", err)
	}
	return lootTable, nil
}

func (s *lootService) ListLootTables(ctx context.Context) ([]*db.LootTable, error) {
	return s.queries.ListLootTables(ctx, s.dbConn)
}

func (s *lootService) ListActiveLootTables(ctx context.Context) ([]*db.LootTable, error) {
	return s.queries.ListActiveLootTables(ctx, s.dbConn)
}

func (s *lootService) UpdateLootTable(ctx context.Context, lootTableID int64, name string, description *string, dropChance float64, isActive bool) error {
	isActiveInt := int64(0)
	if isActive {
		isActiveInt = 1
	}
	params := &db.UpdateLootTableParams{
		Name:        name,
		Description: description,
		DropChance:  dropChance,
		IsActive:    isActiveInt,
		LootTableID: lootTableID,
	}
	err := s.queries.UpdateLootTable(ctx, s.dbConn, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLootTableNotFound
		}
		return fmt.Errorf("failed to update loot table: %w", err)
	}
	return nil
}

func (s *lootService) DeleteLootTable(ctx context.Context, lootTableID int64) error {
	err := s.queries.DeleteLootTable(ctx, s.dbConn, lootTableID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLootTableNotFound
		}
		return fmt.Errorf("failed to delete loot table: %w", err)
	}
	return nil
}

func (s *lootService) CreateLootTableEntry(ctx context.Context, lootTableID int64, cosmeticID int64, weight int64, minQuantity int64, maxQuantity int64) (*db.LootTableEntry, error) {
	params := &db.CreateLootTableEntryParams{
		LootTableID: lootTableID,
		CosmeticID:  cosmeticID,
		Weight:      weight,
		MinQuantity: minQuantity,
		MaxQuantity: maxQuantity,
	}
	return s.queries.CreateLootTableEntry(ctx, s.dbConn, params)
}

func (s *lootService) GetLootTableEntry(ctx context.Context, lootEntryID int64) (*db.LootTableEntry, error) {
	entry, err := s.queries.GetLootTableEntry(ctx, s.dbConn, lootEntryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrLootTableEntryNotFound
		}
		return nil, fmt.Errorf("failed to get loot table entry: %w", err)
	}
	return entry, nil
}

func (s *lootService) GetLootTableEntriesByLootTableID(ctx context.Context, lootTableID int64) ([]*db.LootTableEntry, error) {
	return s.queries.GetLootTableEntriesByLootTableID(ctx, s.dbConn, lootTableID)
}

func (s *lootService) GetLootTableEntriesWithCosmeticDetails(ctx context.Context, lootTableID int64) ([]*db.GetLootTableEntriesWithCosmeticDetailsRow, error) {
	return s.queries.GetLootTableEntriesWithCosmeticDetails(ctx, s.dbConn, lootTableID)
}

func (s *lootService) UpdateLootTableEntry(ctx context.Context, lootEntryID int64, lootTableID int64, cosmeticID int64, weight int64, minQuantity int64, maxQuantity int64) error {
	params := &db.UpdateLootTableEntryParams{
		LootEntryID: lootEntryID,
		LootTableID: lootTableID,
		CosmeticID:  cosmeticID,
		Weight:      weight,
		MinQuantity: minQuantity,
		MaxQuantity: maxQuantity,
	}
	err := s.queries.UpdateLootTableEntry(ctx, s.dbConn, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLootTableEntryNotFound
		}
		return fmt.Errorf("failed to update loot table entry: %w", err)
	}
	return nil
}

func (s *lootService) DeleteLootTableEntry(ctx context.Context, lootEntryID int64) error {
	err := s.queries.DeleteLootTableEntry(ctx, s.dbConn, lootEntryID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrLootTableEntryNotFound
		}
		return fmt.Errorf("failed to delete loot table entry: %w", err)
	}
	return nil
}

func (s *lootService) GenerateLootDrop(ctx context.Context, playerID int64) (*db.CosmeticItem, error) {
	tables, err := s.ListActiveLootTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active loot tables: %w", err)
	}
	if len(tables) == 0 {
		return nil, errors.New("no active loot tables")
	}

	var selectedTable *db.LootTable
	for _, table := range tables {
		roll := randmath.Float64()
		if roll < table.DropChance {
			selectedTable = table
			break
		}
	}
	if selectedTable == nil {
		return nil, errors.New("no drop from any loot table")
	}

	entries, err := s.GetLootTableEntriesByLootTableID(ctx, selectedTable.LootTableID)
	if err != nil {
		return nil, fmt.Errorf("failed to get loot table entries: %w", err)
	}
	if len(entries) == 0 {
		return nil, errors.New("loot table has no entries")
	}

	var totalWeight int64
	for _, entry := range entries {
		totalWeight += entry.Weight
	}
	if totalWeight <= 0 {
		return nil, errors.New("total weight must be positive")
	}

	randomWeight := randmath.Int63n(totalWeight)
	var selectedEntry *db.LootTableEntry
	var cumulativeWeight int64
	for _, entry := range entries {
		cumulativeWeight += entry.Weight
		if randomWeight < cumulativeWeight {
			selectedEntry = entry
			break
		}
	}
	if selectedEntry == nil {
		selectedEntry = entries[len(entries)-1]
	}

	err = s.queries.GrantCosmeticToPlayer(ctx, s.dbConn, &db.GrantCosmeticToPlayerParams{
		PlayerID:    playerID,
		CosmeticID:  selectedEntry.CosmeticID,
		UnlockedVia: "loot_drop",
	})
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			s.logger.Debug("player already owns cosmetic", zap.Int64("player_id", playerID), zap.Int64("cosmetic_id", selectedEntry.CosmeticID))
		} else {
			return nil, fmt.Errorf("failed to grant cosmetic: %w", err)
		}
	}

	cosmetic, err := s.queries.GetCosmeticItem(ctx, s.dbConn, selectedEntry.CosmeticID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("cosmetic not found")
		}
		return nil, fmt.Errorf("failed to get cosmetic item: %w", err)
	}

	return cosmetic, nil
}
