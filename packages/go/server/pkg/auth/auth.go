package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-zombie-defense/db"
	"ai-zombie-defense/db/types"
	"ai-zombie-defense/server/pkg/config"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrPlayerBanned        = errors.New("player is banned")
	ErrDuplicateUsername   = errors.New("username already exists")
	ErrDuplicateEmail      = errors.New("email already exists")
	ErrInvalidRefreshToken = errors.New("invalid refresh token")
	ErrSessionNotFound     = errors.New("session not found")
)

type Service struct {
	config  config.Config
	logger  *zap.Logger
	dbConn  db.DBTX
	queries *db.Queries
}

func NewService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) *Service {
	return &Service{
		config:  cfg,
		logger:  logger,
		dbConn:  dbConn,
		queries: db.New(),
	}
}

// HashPassword creates a bcrypt hash of the password.
func (s *Service) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword compares a password with a bcrypt hash.
func (s *Service) VerifyPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// isDuplicateError checks if the error is a SQLite UNIQUE constraint failure for a specific column.
func (s *Service) isDuplicateError(err error, column string) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// SQLite error format: "UNIQUE constraint failed: players.username"
	// Also check for "constraint failed: UNIQUE constraint failed: players.username (2067)"
	return strings.Contains(errStr, "UNIQUE constraint failed: players."+column)
}

// Authenticate validates username/email and password.
func (s *Service) Authenticate(ctx context.Context, usernameOrEmail, password string) (*db.Player, error) {
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
	if !s.VerifyPassword(player.PasswordHash, password) {
		return nil, ErrInvalidCredentials
	}

	return player, nil
}

// RegisterPlayer creates a new player account.
func (s *Service) RegisterPlayer(ctx context.Context, username, email, password string) (*db.Player, error) {
	// Hash password
	hash, err := s.HashPassword(password)
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
			return nil, ErrDuplicateUsername
		}
		if s.isDuplicateError(err, "email") {
			return nil, ErrDuplicateEmail
		}
		return nil, fmt.Errorf("failed to create player: %w", err)
	}

	// Retrieve created player
	player, err := s.queries.GetPlayerByUsername(ctx, s.dbConn, username)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve created player: %w", err)
	}
	return player, nil
}

// GenerateAccessToken creates a JWT token for a player.
func (s *Service) GenerateAccessToken(playerID int64) (string, error) {
	exp := time.Now().Add(s.config.JWT.AccessExpiration)
	claims := jwt.RegisteredClaims{
		Subject:   fmt.Sprintf("%d", playerID),
		ExpiresAt: jwt.NewNumericDate(exp),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

// GenerateRefreshToken creates a refresh token for a player.
func (s *Service) GenerateRefreshToken(playerID int64) (string, error) {
	exp := time.Now().Add(s.config.JWT.RefreshExpiration)
	// Generate random JWT ID to ensure uniqueness
	randBytes := make([]byte, 16)
	if _, err := rand.Read(randBytes); err != nil {
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

// ValidateToken parses and validates a JWT token.
func (s *Service) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
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

// CreateSession creates a new refresh token session for a player.
func (s *Service) CreateSession(ctx context.Context, playerID int64, ipAddress, userAgent string) (string, error) {
	refreshToken, err := s.GenerateRefreshToken(playerID)
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

// ValidateRefreshToken validates a refresh token and returns the player ID.
func (s *Service) ValidateRefreshToken(ctx context.Context, token string) (int64, error) {
	// Validate JWT
	claims, err := s.ValidateToken(token)
	if err != nil {
		return 0, ErrInvalidRefreshToken
	}
	playerID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil {
		return 0, ErrInvalidRefreshToken
	}

	// Check session exists and not expired
	session, err := s.queries.GetSessionByToken(ctx, s.dbConn, token)
	if err != nil {
		return 0, ErrSessionNotFound
	}
	if session.ExpiresAt.Time.Before(time.Now()) {
		// Delete expired session
		_ = s.queries.DeleteSession(ctx, s.dbConn, token)
		return 0, ErrInvalidRefreshToken
	}
	// Ensure session belongs to the same player
	if session.PlayerID != playerID {
		return 0, ErrInvalidRefreshToken
	}
	return playerID, nil
}

// DeleteSession removes a session by token.
func (s *Service) DeleteSession(ctx context.Context, token string) error {
	s.logger.Debug("deleting session", zap.String("token", token))
	err := s.queries.DeleteSession(ctx, s.dbConn, token)
	if err != nil {
		s.logger.Error("DeleteSession query failed", zap.Error(err), zap.String("token", token))
	}
	return err
}

// RefreshSession validates an existing refresh token and creates a new session.
func (s *Service) RefreshSession(ctx context.Context, oldToken, ipAddress, userAgent string) (playerID int64, newToken string, err error) {
	playerID, err = s.ValidateRefreshToken(ctx, oldToken)
	if err != nil {
		return 0, "", err
	}

	// Delete old session
	err = s.DeleteSession(ctx, oldToken)
	if err != nil {
		return 0, "", fmt.Errorf("failed to delete old session: %w", err)
	}

	// Create new session
	newToken, err = s.CreateSession(ctx, playerID, ipAddress, userAgent)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create new session: %w", err)
	}
	return playerID, newToken, nil
}

// GetPlayer retrieves a player by ID.
func (s *Service) GetPlayer(ctx context.Context, playerID int64) (*db.Player, error) {
	player, err := s.queries.GetPlayer(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}
	return player, nil
}

// UpdatePlayerProfile updates a player's username and email.
func (s *Service) UpdatePlayerProfile(ctx context.Context, playerID int64, username, email string) error {
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

// UpdatePlayerPassword updates a player's password.
func (s *Service) UpdatePlayerPassword(ctx context.Context, playerID int64, newPassword string) error {
	hash, err := s.HashPassword(newPassword)
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

// Config returns the service configuration.
func (s *Service) Config() config.Config {
	return s.config
}
