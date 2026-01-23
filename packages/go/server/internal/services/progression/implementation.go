package progression

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/db/types"
	"ai-zombie-defense/server/pkg/config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type progressionService struct {
	config  config.Config
	logger  *zap.Logger
	dbConn  db.DBTX
	queries *db.Queries
}

func NewProgressionService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) Service {
	return &progressionService{
		config:  cfg,
		logger:  logger,
		dbConn:  dbConn,
		queries: db.New(),
	}
}

func (s *progressionService) GetPlayerProgression(ctx context.Context, playerID int64) (*db.PlayerProgression, error) {
	progression, err := s.queries.GetPlayerProgression(ctx, s.dbConn, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create default progression row
			err = s.queries.CreatePlayerProgression(ctx, s.dbConn, playerID)
			if err != nil {
				s.logger.Warn("Failed to create player progression row",
					zap.Int64("player_id", playerID),
					zap.Error(err))
			}
			return &db.PlayerProgression{
				PlayerID:           playerID,
				Level:              1,
				Experience:         0,
				PrestigeLevel:      0,
				DataCurrency:       0,
				TotalMatchesPlayed: 0,
				TotalWavesSurvived: 0,
				TotalKills:         0,
				TotalDeaths:        0,
				TotalScrapEarned:   0,
				TotalDataEarned:    0,
				UpdatedAt:          types.Timestamp{},
			}, nil
		}
		return nil, fmt.Errorf("failed to get player progression: %w", err)
	}
	return progression, nil
}

func (s *progressionService) calculateLevelFromXP(xp int64) int64 {
	if xp <= 0 {
		return 1
	}
	base := int64(s.config.Progression.BaseXPPerLevel)
	if base <= 0 {
		base = 1000
	}
	level := xp/base + 1
	if level < 1 {
		return 1
	}
	return level
}

func (s *progressionService) AddExperience(ctx context.Context, playerID int64, xpGain int64) error {
	if xpGain <= 0 {
		return nil
	}
	progression, err := s.GetPlayerProgression(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player progression: %w", err)
	}
	oldLevel := progression.Level

	err = s.queries.IncrementExperience(ctx, s.dbConn, &db.IncrementExperienceParams{
		Experience: xpGain,
		PlayerID:   playerID,
	})
	if err != nil {
		return fmt.Errorf("failed to increment experience: %w", err)
	}

	newXP := progression.Experience + xpGain
	newLevel := s.calculateLevelFromXP(newXP)
	if newLevel > oldLevel {
		err = s.queries.UpdateLevel(ctx, s.dbConn, &db.UpdateLevelParams{
			Level:    newLevel,
			PlayerID: playerID,
		})
		if err != nil {
			s.logger.Warn("Failed to update level after XP gain",
				zap.Int64("player_id", playerID),
				zap.Int64("new_level", newLevel),
				zap.Error(err))
		}
	}
	return nil
}

func (s *progressionService) addExperienceWithTx(ctx context.Context, dbTx db.DBTX, playerID int64, xpGain int64) error {
	if xpGain <= 0 {
		return nil
	}
	progression, err := s.queries.GetPlayerProgression(ctx, dbTx, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := s.queries.CreatePlayerProgression(ctx, dbTx, playerID); err != nil {
				return fmt.Errorf("failed to create player progression: %w", err)
			}
			progression = &db.PlayerProgression{
				PlayerID:           playerID,
				Level:              1,
				Experience:         0,
				PrestigeLevel:      0,
				DataCurrency:       0,
				TotalMatchesPlayed: 0,
				TotalWavesSurvived: 0,
				TotalKills:         0,
				TotalDeaths:        0,
				TotalScrapEarned:   0,
				TotalDataEarned:    0,
				UpdatedAt:          types.Timestamp{},
			}
		} else {
			return fmt.Errorf("failed to get player progression: %w", err)
		}
	}
	oldLevel := progression.Level
	err = s.queries.IncrementExperience(ctx, dbTx, &db.IncrementExperienceParams{
		Experience: xpGain,
		PlayerID:   playerID,
	})
	if err != nil {
		return fmt.Errorf("failed to increment experience: %w", err)
	}
	newXP := progression.Experience + xpGain
	newLevel := s.calculateLevelFromXP(newXP)
	if newLevel > oldLevel {
		err = s.queries.UpdateLevel(ctx, dbTx, &db.UpdateLevelParams{
			Level:    newLevel,
			PlayerID: playerID,
		})
		if err != nil {
			s.logger.Warn("Failed to update level after XP gain",
				zap.Int64("player_id", playerID),
				zap.Int64("new_level", newLevel),
				zap.Error(err))
		}
	}
	return nil
}

func (s *progressionService) AddMatchRewards(ctx context.Context, playerID int64, kills, deaths, wavesSurvived, scrapEarned, dataEarned int64) error {
	return s.addMatchRewardsWithTx(ctx, s.dbConn, playerID, kills, deaths, wavesSurvived, scrapEarned, dataEarned)
}

func (s *progressionService) addMatchRewardsWithTx(ctx context.Context, dbTx db.DBTX, playerID int64, kills, deaths, wavesSurvived, scrapEarned, dataEarned int64) error {
	if kills < 0 || deaths < 0 || wavesSurvived < 0 || scrapEarned < 0 || dataEarned < 0 {
		return fmt.Errorf("match stats cannot be negative")
	}
	baseXP := int64(100)
	xpPerKill := int64(10)
	xpPerWave := int64(50)
	xpPerScrap := int64(1)

	totalXP := baseXP + (kills * xpPerKill) + (wavesSurvived * xpPerWave) + (scrapEarned * xpPerScrap)

	err := s.queries.IncrementMatchStats(ctx, dbTx, &db.IncrementMatchStatsParams{
		TotalMatchesPlayed: 1,
		TotalWavesSurvived: wavesSurvived,
		TotalKills:         kills,
		TotalDeaths:        deaths,
		TotalScrapEarned:   scrapEarned,
		TotalDataEarned:    dataEarned,
		PlayerID:           playerID,
	})
	if err != nil {
		return fmt.Errorf("failed to increment match stats: %w", err)
	}
	err = s.addExperienceWithTx(ctx, dbTx, playerID, totalXP)
	if err != nil {
		return fmt.Errorf("failed to add experience: %w", err)
	}
	if dataEarned > 0 {
		err = s.queries.AddDataCurrency(ctx, dbTx, &db.AddDataCurrencyParams{
			DataCurrency: dataEarned,
			PlayerID:     playerID,
		})
		if err != nil {
			s.logger.Warn("Failed to add data currency",
				zap.Int64("player_id", playerID),
				zap.Int64("data_earned", dataEarned),
				zap.Error(err))
		}
	}
	return nil
}

func (s *progressionService) AddDataCurrency(ctx context.Context, playerID int64, amount int64, transactionType string, referenceID *int64) error {
	if amount == 0 {
		return nil
	}
	var dbTx db.DBTX
	var tx *sql.Tx
	var err error

	if db, ok := s.dbConn.(*sql.DB); ok {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		dbTx = tx
	} else {
		dbTx = s.dbConn
	}

	balance, err := s.queries.GetDataCurrency(ctx, dbTx, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := s.queries.CreatePlayerProgression(ctx, dbTx, playerID); err != nil {
				return fmt.Errorf("failed to create player progression: %w", err)
			}
			balance = 0
		} else {
			return fmt.Errorf("failed to get data currency: %w", err)
		}
	}
	newBalance := balance + amount
	if err := s.queries.SetDataCurrency(ctx, dbTx, &db.SetDataCurrencyParams{
		DataCurrency: newBalance,
		PlayerID:     playerID,
	}); err != nil {
		return fmt.Errorf("failed to set data currency: %w", err)
	}
	if err := s.queries.CreateCurrencyTransaction(ctx, dbTx, &db.CreateCurrencyTransactionParams{
		PlayerID:        playerID,
		Amount:          amount,
		BalanceAfter:    newBalance,
		TransactionType: transactionType,
		ReferenceID:     referenceID,
	}); err != nil {
		return fmt.Errorf("failed to create currency transaction: %w", err)
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}
	return nil
}

func (s *progressionService) PrestigePlayer(ctx context.Context, playerID int64) error {
	var dbTx db.DBTX
	var tx *sql.Tx
	var err error

	if db, ok := s.dbConn.(*sql.DB); ok {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		dbTx = tx
	} else {
		dbTx = s.dbConn
	}

	err = s.queries.PrestigePlayer(ctx, dbTx, playerID)
	if err != nil {
		return fmt.Errorf("failed to prestige player: %w", err)
	}

	progression, err := s.queries.GetPlayerProgression(ctx, dbTx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player progression: %w", err)
	}

	cosmetics, err := s.queries.GetPrestigeCosmetics(ctx, dbTx, &db.GetPrestigeCosmeticsParams{
		PlayerID:    playerID,
		UnlockLevel: progression.PrestigeLevel,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get prestige cosmetics: %w", err)
	}

	for _, cosmetic := range cosmetics {
		err = s.queries.GrantCosmeticToPlayer(ctx, dbTx, &db.GrantCosmeticToPlayerParams{
			PlayerID:    playerID,
			CosmeticID:  cosmetic.CosmeticID,
			UnlockedVia: "prestige",
		})
		if err != nil {
			s.logger.Warn("Failed to grant cosmetic to player",
				zap.Int64("player_id", playerID),
				zap.Int64("cosmetic_id", cosmetic.CosmeticID),
				zap.Error(err))
		}
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}
	return nil
}

func (s *progressionService) GetCosmeticCatalog(ctx context.Context) ([]*db.CosmeticItem, error) {
	return s.queries.GetCosmeticCatalog(ctx, s.dbConn)
}

func (s *progressionService) GetPlayerCosmetics(ctx context.Context, playerID int64) ([]*db.GetPlayerCosmeticsRow, error) {
	return s.queries.GetPlayerCosmetics(ctx, s.dbConn, playerID)
}

func (s *progressionService) EquipCosmetic(ctx context.Context, playerID int64, cosmeticID int64) error {
	cosmetic, err := s.queries.GetCosmeticItem(ctx, s.dbConn, cosmeticID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCosmeticNotFound
		}
		return fmt.Errorf("failed to get cosmetic item: %w", err)
	}

	_, err = s.queries.GetPlayerCosmetic(ctx, s.dbConn, &db.GetPlayerCosmeticParams{
		PlayerID:   playerID,
		CosmeticID: cosmeticID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCosmeticNotOwned
		}
		return fmt.Errorf("failed to check cosmetic ownership: %w", err)
	}

	loadout, err := s.queries.GetActiveLoadout(ctx, s.dbConn, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			params := &db.CreateLoadoutParams{
				PlayerID: playerID,
				Name:     "Default",
				IsActive: 1,
			}
			err = s.queries.CreateLoadout(ctx, s.dbConn, params)
			if err != nil {
				return fmt.Errorf("failed to create default loadout: %w", err)
			}
			loadout, err = s.queries.GetActiveLoadout(ctx, s.dbConn, playerID)
			if err != nil {
				return fmt.Errorf("failed to retrieve created loadout: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get active loadout: %w", err)
		}
	}

	var dbTx db.DBTX
	var tx *sql.Tx
	if db, ok := s.dbConn.(*sql.DB); ok {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		dbTx = tx
	} else {
		dbTx = s.dbConn
	}

	err = s.queries.DeleteLoadoutCosmeticBySlot(ctx, dbTx, &db.DeleteLoadoutCosmeticBySlotParams{
		LoadoutID: loadout.LoadoutID,
		Slot:      cosmetic.Slot,
	})
	if err != nil {
		return fmt.Errorf("failed to clear slot: %w", err)
	}

	err = s.queries.InsertLoadoutCosmetic(ctx, dbTx, &db.InsertLoadoutCosmeticParams{
		LoadoutID:  loadout.LoadoutID,
		CosmeticID: cosmeticID,
		Slot:       cosmetic.Slot,
	})
	if err != nil {
		return fmt.Errorf("failed to equip cosmetic: %w", err)
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}
	return nil
}

func (s *progressionService) PurchaseCosmetic(ctx context.Context, playerID int64, cosmeticID int64) error {
	cosmetic, err := s.queries.GetCosmeticItem(ctx, s.dbConn, cosmeticID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCosmeticNotFound
		}
		return fmt.Errorf("failed to get cosmetic item: %w", err)
	}

	_, err = s.queries.GetPlayerCosmetic(ctx, s.dbConn, &db.GetPlayerCosmeticParams{
		PlayerID:   playerID,
		CosmeticID: cosmeticID,
	})
	if err == nil {
		return ErrCosmeticAlreadyOwned
	}

	balance, err := s.queries.GetDataCurrency(ctx, s.dbConn, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := s.queries.CreatePlayerProgression(ctx, s.dbConn, playerID); err != nil {
				return fmt.Errorf("failed to create player progression: %w", err)
			}
			balance = 0
		} else {
			return fmt.Errorf("failed to get data currency: %w", err)
		}
	}

	if balance < cosmetic.DataCost {
		return ErrInsufficientCurrency
	}

	var dbTx db.DBTX
	var tx *sql.Tx
	if db, ok := s.dbConn.(*sql.DB); ok {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		dbTx = tx
	} else {
		dbTx = s.dbConn
	}

	newBalance := balance - cosmetic.DataCost
	if err := s.queries.SetDataCurrency(ctx, dbTx, &db.SetDataCurrencyParams{
		DataCurrency: newBalance,
		PlayerID:     playerID,
	}); err != nil {
		return fmt.Errorf("failed to set data currency: %w", err)
	}

	if err := s.queries.CreateCurrencyTransaction(ctx, dbTx, &db.CreateCurrencyTransactionParams{
		PlayerID:        playerID,
		Amount:          -cosmetic.DataCost,
		BalanceAfter:    newBalance,
		TransactionType: "purchase",
		ReferenceID:     &cosmeticID,
	}); err != nil {
		return fmt.Errorf("failed to create currency transaction: %w", err)
	}

	if err := s.queries.GrantCosmeticToPlayer(ctx, dbTx, &db.GrantCosmeticToPlayerParams{
		PlayerID:    playerID,
		CosmeticID:  cosmeticID,
		UnlockedVia: "purchase",
	}); err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrCosmeticAlreadyOwned
		}
		return fmt.Errorf("failed to grant cosmetic: %w", err)
	}

	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}
	return nil
}
