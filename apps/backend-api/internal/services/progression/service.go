package progression

import (
	"ai-zombie-defense/backend-api/db"
	"context"
	"errors"
)

var (
	ErrCosmeticNotFound     = errors.New("cosmetic not found")
	ErrCosmeticNotOwned     = errors.New("cosmetic not owned")
	ErrLoadoutNotFound      = errors.New("loadout not found")
	ErrInsufficientCurrency = errors.New("insufficient data currency")
	ErrCosmeticAlreadyOwned = errors.New("cosmetic already owned")
)

type Service interface {
	GetPlayerProgression(ctx context.Context, playerID int64) (*db.PlayerProgression, error)
	AddExperience(ctx context.Context, playerID int64, xpGain int64) error
	PrestigePlayer(ctx context.Context, playerID int64) error
	AddDataCurrency(ctx context.Context, playerID int64, amount int64, transactionType string, referenceID *int64) error
	GetCosmeticCatalog(ctx context.Context) ([]*db.CosmeticItem, error)
	GetPlayerCosmetics(ctx context.Context, playerID int64) ([]*db.GetPlayerCosmeticsRow, error)
	EquipCosmetic(ctx context.Context, playerID int64, cosmeticID int64) error
	AddMatchRewards(ctx context.Context, playerID int64, kills, deaths, wavesSurvived, scrapEarned, dataEarned int64) error
	PurchaseCosmetic(ctx context.Context, playerID int64, cosmeticID int64) error
}
