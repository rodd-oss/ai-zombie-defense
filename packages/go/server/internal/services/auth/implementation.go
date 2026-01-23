package auth

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/db/types"
	"ai-zombie-defense/server/internal/services/account"
	"ai-zombie-defense/server/pkg/config"
	"context"
	cryptorand "crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	config  config.Config
	logger  *zap.Logger
	dbConn  db.DBTX
	queries *db.Queries
}

func NewAuthService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) Service {
	return &authService{
		config:  cfg,
		logger:  logger,
		dbConn:  dbConn,
		queries: db.New(),
	}
}

func (s *authService) Authenticate(ctx context.Context, usernameOrEmail, password string) (*db.Player, error) {
	var player *db.Player
	var err error

	// Try username first
	player, err = s.queries.GetPlayerByUsername(ctx, s.dbConn, usernameOrEmail)
	if err != nil {
		s.logger.Debug("GetPlayerByUsername failed", zap.String("usernameOrEmail", usernameOrEmail), zap.Error(err))
		// Try email
		player, err = s.queries.GetPlayerByEmail(ctx, s.dbConn, usernameOrEmail)
		if err != nil {
			s.logger.Debug("GetPlayerByEmail failed", zap.String("usernameOrEmail", usernameOrEmail), zap.Error(err))
			return nil, ErrInvalidCredentials
		}
	}

	// Check if player is banned
	if player.IsBanned != 0 {
		return nil, ErrPlayerBanned
	}

	// Verify password
	if !s.verifyPassword(player.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	return player, nil
}

func (s *authService) RegisterPlayer(ctx context.Context, username, email, password string) (*db.Player, error) {
	// Hash password
	hash, err := s.hashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create player
	err = s.queries.CreatePlayer(ctx, s.dbConn, &db.CreatePlayerParams{
		Username:     username,
		Email:        email,
		PasswordHash: hash,
	})
	if err != nil {
		// Check for duplicate username/email
		if s.isDuplicateError(err, "username") {
			return nil, account.ErrDuplicateUsername
		}
		if s.isDuplicateError(err, "email") {
			return nil, account.ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to create player: %w", err)
	}

	// Retrieve created player
	player, err := s.queries.GetPlayerByUsername(ctx, s.dbConn, username)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created player: %w", err)
	}

	// Create player progression row with default values
	err = s.queries.CreatePlayerProgression(ctx, s.dbConn, player.PlayerID)
	if err != nil {
		// Log but continue - progression row may already exist or other issue
		s.logger.Warn("Failed to create player progression row",
			zap.Int64("player_id", player.PlayerID),
			zap.Error(err))
	}

	return player, nil
}

func (s *authService) GenerateAccessToken(playerID int64) (string, error) {
	exp := time.Now().Add(s.config.JWT.AccessExpiration)
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", playerID),
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

func (s *authService) CreateSession(ctx context.Context, playerID int64, ipAddress, userAgent string) (string, error) {
	refreshToken, err := s.generateRefreshToken(playerID)
	if err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}
	s.logger.Debug("CreateSession generating token", zap.String("token", refreshToken), zap.Int64("playerID", playerID))

	expiresAt := time.Now().Add(s.config.JWT.RefreshExpiration)
	params := &db.CreateSessionParams{
		PlayerID:  playerID,
		Token:     refreshToken,
		ExpiresAt: types.Timestamp{Time: expiresAt},
		IpAddress: &ipAddress,
		UserAgent: &userAgent,
	}
	if ipAddress == "" {
		params.IpAddress = nil
	}
	if userAgent == "" {
		params.UserAgent = nil
	}

	err = s.queries.CreateSession(ctx, s.dbConn, params)
	if err != nil {
		s.logger.Error("CreateSession query failed", zap.Error(err))
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	s.logger.Debug("CreateSession inserted", zap.String("token", refreshToken))
	return refreshToken, nil
}

func (s *authService) RefreshSession(ctx context.Context, oldToken, ipAddress, userAgent string) (int64, string, error) {
	playerID, err := s.validateRefreshToken(ctx, oldToken)
	if err != nil {
		return 0, "", err
	}

	// Delete old session
	err = s.DeleteSession(ctx, oldToken)
	if err != nil {
		return 0, "", fmt.Errorf("failed to delete old session: %w", err)
	}

	// Create new session
	newToken, err := s.CreateSession(ctx, playerID, ipAddress, userAgent)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create new session: %w", err)
	}
	return playerID, newToken, nil
}

func (s *authService) DeleteSession(ctx context.Context, token string) error {
	s.logger.Debug("deleting session", zap.String("token", token))
	err := s.queries.DeleteSession(ctx, s.dbConn, token)
	if err != nil {
		s.logger.Error("DeleteSession query failed", zap.Error(err), zap.String("token", token))
	}
	return err
}

func (s *authService) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.config.JWT.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token")
}

// Internal helpers

func (s *authService) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func (s *authService) verifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func (s *authService) isDuplicateError(err error, column string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "UNIQUE constraint failed: players."+column)
}

func (s *authService) generateRefreshToken(playerID int64) (string, error) {
	exp := time.Now().Add(s.config.JWT.RefreshExpiration)
	randBytes := make([]byte, 16)
	if _, err := cryptorand.Read(randBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	jti := hex.EncodeToString(randBytes)
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", playerID),
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ID:        jti,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

func (s *authService) validateRefreshToken(ctx context.Context, token string) (int64, error) {
	claims, err := s.ValidateToken(token)
	if err != nil {
		return 0, ErrInvalidRefreshToken
	}
	playerID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, ErrInvalidRefreshToken
	}

	session, err := s.queries.GetSessionByToken(ctx, s.dbConn, token)
	if err != nil {
		return 0, ErrSessionNotFound
	}
	if session.ExpiresAt.Time.Before(time.Now()) {
		_ = s.queries.DeleteSession(ctx, s.dbConn, token)
		return 0, ErrInvalidRefreshToken
	}
	if session.PlayerID != playerID {
		return 0, ErrInvalidRefreshToken
	}
	return playerID, nil
}

func (s *authService) IsAdmin(ctx context.Context, playerID int64) (bool, error) {
	player, err := s.queries.GetPlayer(ctx, s.dbConn, playerID)
	if err != nil {
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}
	return player.IsAdmin == 1, nil
}
