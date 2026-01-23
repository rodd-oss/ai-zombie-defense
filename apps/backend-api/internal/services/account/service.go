package account

import (
	"ai-zombie-defense/backend-api/db"
	"context"
	"errors"
)

var (
	ErrDuplicateUsername = errors.New("username already exists")
	ErrDuplicateEmail    = errors.New("email already exists")
)

type Service interface {
	GetPlayer(ctx context.Context, playerID int64) (*db.Player, error)
	UpdatePlayerProfile(ctx context.Context, playerID int64, username, email string) error
	UpdatePlayerPassword(ctx context.Context, playerID int64, newPassword string) error
	GetPlayerSettings(ctx context.Context, playerID int64) (*db.PlayerSetting, error)
	UpsertPlayerSettings(ctx context.Context, params *db.UpsertPlayerSettingsParams) error
}
