package match

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/db/types"
	"ai-zombie-defense/server/internal/services/progression"
	"ai-zombie-defense/server/pkg/config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"go.uber.org/zap"
)

type matchService struct {
	config         config.Config
	logger         *zap.Logger
	dbConn         db.DBTX
	queries        *db.Queries
	progressionSvc progression.Service
}

func NewMatchService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX, progressionSvc progression.Service) Service {
	return &matchService{
		config:         cfg,
		logger:         logger,
		dbConn:         dbConn,
		queries:        db.New(),
		progressionSvc: progressionSvc,
	}
}

func (s *matchService) StoreMatchWithStats(ctx context.Context, serverID int64, matchParams *db.CreateMatchParams, playerStats []*db.CreatePlayerMatchStatsParams) error {
	// Ensure matchParams.ServerID matches the provided serverID
	if matchParams.ServerID != serverID {
		return fmt.Errorf("server ID mismatch: expected %d, got %d", serverID, matchParams.ServerID)
	}

	// Start transaction
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
		s.logger.Warn("dbConn is not *sql.DB, proceeding without transaction")
		dbTx = s.dbConn
	}

	// Create match
	match, err := s.queries.CreateMatch(ctx, dbTx, matchParams)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}

	// Insert player stats
	for _, stats := range playerStats {
		// Ensure stats.MatchID matches the created match
		stats.MatchID = match.MatchID
		_, err := s.queries.CreatePlayerMatchStats(ctx, dbTx, stats)
		if err != nil {
			return fmt.Errorf("failed to create player match stats: %w", err)
		}
	}

	// Award rewards based on player performance
	for _, stats := range playerStats {
		err := s.addMatchRewardsWithTx(ctx, dbTx, stats.PlayerID, stats.ZombiesKilled, stats.Deaths, stats.WavesSurvived, stats.ScrapEarned, stats.DataEarned)
		if err != nil {
			return fmt.Errorf("failed to award match rewards: %w", err)
		}
	}

	// Commit transaction if we started one
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	s.logger.Info("Match stored successfully",
		zap.Int64("match_id", match.MatchID),
		zap.Int64("server_id", serverID),
		zap.Int("player_count", len(playerStats)))
	return nil
}

func (s *matchService) addMatchRewardsWithTx(ctx context.Context, dbTx db.DBTX, playerID int64, kills, deaths, wavesSurvived, scrapEarned, dataEarned int64) error {
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

func (s *matchService) addExperienceWithTx(ctx context.Context, dbTx db.DBTX, playerID int64, xpGain int64) error {
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

func (s *matchService) calculateLevelFromXP(xp int64) int64 {
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

func (s *matchService) GetPlayerMatchHistory(ctx context.Context, playerID int64, limit int32) ([]*db.GetPlayerMatchHistoryRow, error) {
	matches, err := s.queries.GetPlayerMatchHistory(ctx, s.dbConn, &db.GetPlayerMatchHistoryParams{
		PlayerID: playerID,
		Limit:    int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get player match history: %w", err)
	}
	return matches, nil
}
