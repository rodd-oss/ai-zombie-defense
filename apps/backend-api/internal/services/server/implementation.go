package server

import (
	"ai-zombie-defense/backend-api/internal/db"
	"ai-zombie-defense/backend-api/internal/db/types"
	"ai-zombie-defense/backend-api/pkg/config"
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
)

type serverService struct {
	config  config.Config
	logger  *zap.Logger
	dbConn  db.DBTX
	queries *db.Queries
}

func NewServerService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) Service {
	return &serverService{
		config:  cfg,
		logger:  logger,
		dbConn:  dbConn,
		queries: db.New(),
	}
}

func (s *serverService) RegisterServer(ctx context.Context, ipAddress string, port int64, name string, mapRotation *string, maxPlayers int64, region *string, version *string) (*db.Server, string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := cryptorand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate random token: %w", err)
	}
	authToken := hex.EncodeToString(tokenBytes)

	params := &db.CreateServerParams{
		IpAddress:   ipAddress,
		Port:        port,
		AuthToken:   &authToken,
		Name:        name,
		MapRotation: mapRotation,
		MaxPlayers:  maxPlayers,
		Region:      region,
		Version:     version,
	}

	server, err := s.queries.CreateServer(ctx, s.dbConn, params)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create server: %w", err)
	}

	return server, authToken, nil
}

func (s *serverService) GetServerByAuthToken(ctx context.Context, authToken string) (*db.Server, error) {
	server, err := s.queries.GetServerByAuthToken(ctx, s.dbConn, &authToken)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrServerNotFound
		}
		return nil, fmt.Errorf("failed to get server by auth token: %w", err)
	}
	return server, nil
}

func (s *serverService) UpdateServerHeartbeat(ctx context.Context, serverID int64, currentPlayers int64, mapRotation *string) error {
	now := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	params := &db.UpdateServerHeartbeatParams{
		LastHeartbeat:  &now,
		CurrentPlayers: currentPlayers,
		MapRotation:    mapRotation,
		ServerID:       serverID,
	}
	err := s.queries.UpdateServerHeartbeat(ctx, s.dbConn, params)
	if err != nil {
		return fmt.Errorf("failed to update server heartbeat: %w", err)
	}
	return nil
}

func (s *serverService) ListActiveServers(ctx context.Context, region, mapRotation, version *string, minPlayers, maxPlayers *int64) ([]*db.Server, error) {
	params := &db.ListActiveServersParams{
		Region:      region,
		MapRotation: mapRotation,
		Version:     version,
	}
	if minPlayers != nil {
		params.CurrentPlayers = *minPlayers
	} else {
		params.CurrentPlayers = -1
	}
	if maxPlayers != nil {
		params.CurrentPlayers_2 = *maxPlayers
	} else {
		params.CurrentPlayers_2 = -1
	}
	servers, err := s.queries.ListActiveServers(ctx, s.dbConn, params)
	if err != nil {
		return nil, fmt.Errorf("failed to list active servers: %w", err)
	}
	return servers, nil
}

func (s *serverService) GenerateJoinToken(ctx context.Context, playerID int64, serverID int64, expiresIn time.Duration) (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := cryptorand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}
	token := hex.EncodeToString(tokenBytes)

	expiresAt := time.Now().UTC().Add(expiresIn)
	params := &db.CreateJoinTokenParams{
		Token:     token,
		PlayerID:  playerID,
		ServerID:  serverID,
		ExpiresAt: types.Timestamp{Time: expiresAt},
	}

	_, err := s.queries.CreateJoinToken(ctx, s.dbConn, params)
	if err != nil {
		return "", fmt.Errorf("failed to create join token: %w", err)
	}

	s.logger.Debug("Join token generated",
		zap.Int64("player_id", playerID),
		zap.Int64("server_id", serverID),
		zap.Time("expires_at", expiresAt))
	return token, nil
}

func (s *serverService) ValidateJoinToken(ctx context.Context, token string) (playerID int64, serverID int64, err error) {
	joinToken, err := s.queries.GetValidJoinToken(ctx, s.dbConn, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			tokenRow, err2 := s.queries.GetJoinToken(ctx, s.dbConn, token)
			if err2 != nil {
				if errors.Is(err2, sql.ErrNoRows) {
					return 0, 0, ErrJoinTokenInvalid
				}
				return 0, 0, fmt.Errorf("failed to get join token: %w", err2)
			}
			expiresAt := tokenRow.ExpiresAt.Time
			if expiresAt.Before(time.Now().UTC()) {
				return 0, 0, ErrJoinTokenExpired
			}
			if tokenRow.UsedAt.Valid {
				return 0, 0, ErrJoinTokenAlreadyUsed
			}
			return 0, 0, ErrJoinTokenInvalid
		}
		return 0, 0, fmt.Errorf("failed to validate join token: %w", err)
	}

	return joinToken.PlayerID, joinToken.ServerID, nil
}

func (s *serverService) MarkTokenUsed(ctx context.Context, token string) error {
	err := s.queries.MarkTokenUsed(ctx, s.dbConn, token)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	s.logger.Debug("Join token marked as used", zap.String("token", token))
	return nil
}

func (s *serverService) AddFavorite(ctx context.Context, playerID int64, serverID int64, note *string) error {
	existing, err := s.queries.GetFavorite(ctx, s.dbConn, &db.GetFavoriteParams{
		PlayerID: playerID,
		ServerID: serverID,
	})
	if err == nil && existing != nil {
		return ErrFavoriteAlreadyExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing favorite: %w", err)
	}
	params := &db.AddFavoriteParams{
		PlayerID: playerID,
		ServerID: serverID,
		Note:     note,
	}
	err = s.queries.AddFavorite(ctx, s.dbConn, params)
	if err != nil {
		return fmt.Errorf("failed to add favorite: %w", err)
	}
	s.logger.Debug("Favorite added", zap.Int64("player_id", playerID), zap.Int64("server_id", serverID))
	return nil
}

func (s *serverService) RemoveFavorite(ctx context.Context, playerID int64, serverID int64) error {
	params := &db.RemoveFavoriteParams{
		PlayerID: playerID,
		ServerID: serverID,
	}
	err := s.queries.RemoveFavorite(ctx, s.dbConn, params)
	if err != nil {
		return fmt.Errorf("failed to remove favorite: %w", err)
	}
	s.logger.Debug("Favorite removed", zap.Int64("player_id", playerID), zap.Int64("server_id", serverID))
	return nil
}

func (s *serverService) ListPlayerFavorites(ctx context.Context, playerID int64) ([]*db.ListPlayerFavoritesRow, error) {
	favorites, err := s.queries.ListPlayerFavorites(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list player favorites: %w", err)
	}
	return favorites, nil
}
