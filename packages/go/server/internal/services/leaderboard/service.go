package leaderboard

import (
	"ai-zombie-defense/db"
	"context"
)

type Service interface {
	GetDailyLeaderboard(ctx context.Context) ([]*db.GetDailyLeaderboardRow, error)
	GetWeeklyLeaderboard(ctx context.Context) ([]*db.GetWeeklyLeaderboardRow, error)
	GetAllTimeLeaderboard(ctx context.Context) ([]*db.GetAllTimeLeaderboardRow, error)
}
