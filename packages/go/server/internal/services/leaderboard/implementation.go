package leaderboard

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/server/pkg/config"
	"context"
	"fmt"

	"go.uber.org/zap"
)

type leaderboardService struct {
	config  config.Config
	logger  *zap.Logger
	dbConn  db.DBTX
	queries *db.Queries
}

func NewLeaderboardService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) Service {
	return &leaderboardService{
		config:  cfg,
		logger:  logger,
		dbConn:  dbConn,
		queries: db.New(),
	}
}

func (s *leaderboardService) GetDailyLeaderboard(ctx context.Context) ([]*db.GetDailyLeaderboardRow, error) {
	entries, err := s.queries.GetDailyLeaderboard(ctx, s.dbConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily leaderboard: %w", err)
	}
	return entries, nil
}

func (s *leaderboardService) GetWeeklyLeaderboard(ctx context.Context) ([]*db.GetWeeklyLeaderboardRow, error) {
	entries, err := s.queries.GetWeeklyLeaderboard(ctx, s.dbConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly leaderboard: %w", err)
	}
	return entries, nil
}

func (s *leaderboardService) GetAllTimeLeaderboard(ctx context.Context) ([]*db.GetAllTimeLeaderboardRow, error) {
	entries, err := s.queries.GetAllTimeLeaderboard(ctx, s.dbConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get all-time leaderboard: %w", err)
	}
	return entries, nil
}
