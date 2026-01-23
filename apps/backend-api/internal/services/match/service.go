package match

import (
	"ai-zombie-defense/backend-api/internal/db"
	"context"
	"errors"
)

var (
	ErrMatchNotFound = errors.New("match not found")
)

type Service interface {
	StoreMatchWithStats(ctx context.Context, serverID int64, matchParams *db.CreateMatchParams, playerStats []*db.CreatePlayerMatchStatsParams) error
	GetPlayerMatchHistory(ctx context.Context, playerID int64, limit int32) ([]*db.GetPlayerMatchHistoryRow, error)
}
