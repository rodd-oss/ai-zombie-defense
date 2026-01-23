package loot

import (
	"ai-zombie-defense/db"
	"context"
)

type Service interface {
	CreateLootTable(ctx context.Context, name string, description *string, dropChance float64, isActive bool) (*db.LootTable, error)
	GetLootTable(ctx context.Context, lootTableID int64) (*db.LootTable, error)
	ListLootTables(ctx context.Context) ([]*db.LootTable, error)
	ListActiveLootTables(ctx context.Context) ([]*db.LootTable, error)
	UpdateLootTable(ctx context.Context, lootTableID int64, name string, description *string, dropChance float64, isActive bool) error
	DeleteLootTable(ctx context.Context, lootTableID int64) error
	CreateLootTableEntry(ctx context.Context, lootTableID int64, cosmeticID int64, weight int64, minQuantity int64, maxQuantity int64) (*db.LootTableEntry, error)
	GetLootTableEntry(ctx context.Context, lootEntryID int64) (*db.LootTableEntry, error)
	GetLootTableEntriesByLootTableID(ctx context.Context, lootTableID int64) ([]*db.LootTableEntry, error)
	GetLootTableEntriesWithCosmeticDetails(ctx context.Context, lootTableID int64) ([]*db.GetLootTableEntriesWithCosmeticDetailsRow, error)
	UpdateLootTableEntry(ctx context.Context, lootEntryID int64, lootTableID int64, cosmeticID int64, weight int64, minQuantity int64, maxQuantity int64) error
	DeleteLootTableEntry(ctx context.Context, lootEntryID int64) error
	GenerateLootDrop(ctx context.Context, playerID int64) (*db.CosmeticItem, error)
}
