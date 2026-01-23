package account

import (
	"ai-zombie-defense/backend-api/db"
	"ai-zombie-defense/backend-api/internal/db/types"
	"ai-zombie-defense/backend-api/pkg/config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type accountService struct {
	config  config.Config
	logger  *zap.Logger
	dbConn  db.DBTX
	queries *db.Queries
}

func NewAccountService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) Service {
	return &accountService{
		config:  cfg,
		logger:  logger,
		dbConn:  dbConn,
		queries: db.New(),
	}
}

func (s *accountService) GetPlayer(ctx context.Context, playerID int64) (*db.Player, error) {
	player, err := s.queries.GetPlayer(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}
	return player, nil
}

func (s *accountService) UpdatePlayerProfile(ctx context.Context, playerID int64, username, email string) error {
	params := &db.UpdatePlayerProfileParams{
		PlayerID: playerID,
		Username: username,
		Email:    email,
	}
	err := s.queries.UpdatePlayerProfile(ctx, s.dbConn, params)
	if err != nil {
		if s.isDuplicateError(err, "username") {
			return ErrDuplicateUsername
		}
		if s.isDuplicateError(err, "email") {
			return ErrDuplicateEmail
		}
		return fmt.Errorf("failed to update player profile: %w", err)
	}
	return nil
}

func (s *accountService) UpdatePlayerPassword(ctx context.Context, playerID int64, newPassword string) error {
	hash, err := s.hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}
	params := &db.UpdatePlayerPasswordParams{
		PlayerID:     playerID,
		PasswordHash: hash,
	}
	err = s.queries.UpdatePlayerPassword(ctx, s.dbConn, params)
	if err != nil {
		return fmt.Errorf("failed to update player password: %w", err)
	}
	return nil
}

func (s *accountService) GetPlayerSettings(ctx context.Context, playerID int64) (*db.PlayerSetting, error) {
	settings, err := s.queries.GetPlayerSettings(ctx, s.dbConn, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return default settings
			return &db.PlayerSetting{
				PlayerID:         playerID,
				KeyBindings:      nil,
				MouseSensitivity: nil,
				UiScale:          nil,
				ColorBlindMode:   0,
				SubtitlesEnabled: 0,
				CreatedAt:        types.Timestamp{},
				UpdatedAt:        types.Timestamp{},
			}, nil
		}
		return nil, fmt.Errorf("failed to get player settings: %w", err)
	}
	return settings, nil
}

func (s *accountService) UpsertPlayerSettings(ctx context.Context, params *db.UpsertPlayerSettingsParams) error {
	err := s.queries.UpsertPlayerSettings(ctx, s.dbConn, params)
	if err != nil {
		return fmt.Errorf("failed to upsert player settings: %w", err)
	}
	return nil
}

// Internal helpers

func (s *accountService) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *accountService) isDuplicateError(err error, column string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "UNIQUE constraint failed: players."+column)
}
