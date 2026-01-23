package auth

import (
	"ai-zombie-defense/db"
	"context"
	"errors"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrPlayerBanned        = errors.New("player is banned")
	ErrDuplicateUsername   = errors.New("username already exists")
	ErrDuplicateEmail      = errors.New("email already exists")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrSessionNotFound     = errors.New("session not found")
)

type Service interface {
	Authenticate(ctx context.Context, usernameOrEmail, password string) (*db.Player, error)
	RegisterPlayer(ctx context.Context, username, email, password string) (*db.Player, error)
	GenerateAccessToken(playerID int64) (string, error)
	CreateSession(ctx context.Context, playerID int64, ipAddress, userAgent string) (string, error)
	RefreshSession(ctx context.Context, oldToken, ipAddress, userAgent string) (int64, string, error)
	DeleteSession(ctx context.Context, token string) error
	ValidateToken(tokenString string) (*jwt.RegisteredClaims, error)
}
