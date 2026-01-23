package server

import (
	"ai-zombie-defense/db"
	"context"
	"errors"
	"time"
)

var (
	ErrServerNotFound        = errors.New("server not found")
	ErrJoinTokenInvalid      = errors.New("join token invalid")
	ErrJoinTokenExpired      = errors.New("join token expired")
	ErrJoinTokenAlreadyUsed  = errors.New("join token already used")
	ErrFavoriteAlreadyExists = errors.New("server already favorited")
	ErrFavoriteNotFound      = errors.New("favorite not found")
)

type Service interface {
	RegisterServer(ctx context.Context, ipAddress string, port int64, name string, mapRotation *string, maxPlayers int64, region *string, version *string) (*db.Server, string, error)
	GetServerByAuthToken(ctx context.Context, authToken string) (*db.Server, error)
	UpdateServerHeartbeat(ctx context.Context, serverID int64, currentPlayers int64, mapRotation *string) error
	ListActiveServers(ctx context.Context, region, mapRotation, version *string, minPlayers, maxPlayers *int64) ([]*db.Server, error)
	GenerateJoinToken(ctx context.Context, playerID int64, serverID int64, expiresIn time.Duration) (string, error)
	ValidateJoinToken(ctx context.Context, token string) (playerID int64, serverID int64, err error)
	MarkTokenUsed(ctx context.Context, token string) error
	AddFavorite(ctx context.Context, playerID int64, serverID int64, note *string) error
	RemoveFavorite(ctx context.Context, playerID int64, serverID int64) error
	ListPlayerFavorites(ctx context.Context, playerID int64) ([]*db.ListPlayerFavoritesRow, error)
}
