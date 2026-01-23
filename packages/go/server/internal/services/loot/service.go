package loot

import (
	"ai-zombie-defense/db"
	"context"
)

type Service interface {
	CreateLootTable(ctx context.Context, name string, description *string, dropChance float64, isActive bool) (*db.LootTable, error)
	GetLootTable(ctx context.Context, lootTableID int64) (*db.LootTable, error)
	ListLootTables(ctx context.Context) ([]*db.LootTable, error)
	UpdateLootTable(ctx context.Context, lootTableID int64, name string, description *string, dropChance float64, isActive bool) error
	DeleteLootTable(ctx context.Context, lootTableID int64) error
	CreateLootTableEntry(ctx context.Context, lootTableID int64, cosmeticID int64, weight int64, minQuantity int64, maxQuantity int64) (*db.LootTableEntry, error)
	GenerateLootDrop(ctx context.Context, playerID int64) (*db.CosmeticItem, error)
}
