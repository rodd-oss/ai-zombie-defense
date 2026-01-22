package auth

import (
	"context"
	"crypto/rand"
	"database/sql"
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
	ErrCosmeticNotFound    = errors.New("cosmetic not found")
	ErrCosmeticNotOwned    = errors.New("cosmetic not owned")
	ErrLoadoutNotFound     = errors.New("loadout not found")
	ErrMatchNotFound       = errors.New("match not found")
	ErrServerNotFound      = errors.New("server not found")
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

// GetPlayerSettings retrieves player settings or returns defaults if not found.
func (s *Service) GetPlayerSettings(ctx context.Context, playerID int64) (*db.PlayerSetting, error) {
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

// GetPlayerProgression retrieves player progression or creates a default row if not found.
func (s *Service) GetPlayerProgression(ctx context.Context, playerID int64) (*db.PlayerProgression, error) {
	progression, err := s.queries.GetPlayerProgression(ctx, s.dbConn, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create default progression row
			err = s.queries.CreatePlayerProgression(ctx, s.dbConn, playerID)
			if err != nil {
				// Log but continue with default values
				s.logger.Warn("Failed to create player progression row",
					zap.Int64("player_id", playerID),
					zap.Error(err))
			}
			// Return default progression
			return &db.PlayerProgression{
				PlayerID:           playerID,
				Level:              1,
				Experience:         0,
				PrestigeLevel:      0,
				DataCurrency:       0,
				TotalMatchesPlayed: 0,
				TotalWavesSurvived: 0,
				TotalKills:         0,
				TotalDeaths:        0,
				TotalScrapEarned:   0,
				TotalDataEarned:    0,
				UpdatedAt:          types.Timestamp{},
			}, nil
		}
		return nil, fmt.Errorf("failed to get player progression: %w", err)
	}
	return progression, nil
}

// GetCosmeticCatalog returns all cosmetic items available in the catalog.
func (s *Service) GetCosmeticCatalog(ctx context.Context) ([]*db.CosmeticItem, error) {
	items, err := s.queries.GetCosmeticCatalog(ctx, s.dbConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get cosmetic catalog: %w", err)
	}
	return items, nil
}

// GetPlayerCosmetics returns cosmetic items owned by the player.
func (s *Service) GetPlayerCosmetics(ctx context.Context, playerID int64) ([]*db.GetPlayerCosmeticsRow, error) {
	items, err := s.queries.GetPlayerCosmetics(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get player cosmetics: %w", err)
	}
	return items, nil
}

// UpsertPlayerSettings creates or updates player settings.
func (s *Service) UpsertPlayerSettings(ctx context.Context, params *db.UpsertPlayerSettingsParams) error {
	err := s.queries.UpsertPlayerSettings(ctx, s.dbConn, params)
	if err != nil {
		return fmt.Errorf("failed to upsert player settings: %w", err)
	}
	return nil
}

// Config returns the service configuration.
func (s *Service) Config() config.Config {
	return s.config
}

// calculateLevelFromXP computes the player level based on their XP using linear scaling.
// Level = floor(XP / BaseXPPerLevel) + 1, with minimum level of 1.
func (s *Service) calculateLevelFromXP(xp int64) int64 {
	if xp <= 0 {
		return 1
	}
	base := int64(s.config.Progression.BaseXPPerLevel)
	if base <= 0 {
		base = 1000 // fallback
	}
	level := xp/base + 1
	if level < 1 {
		return 1
	}
	return level
}

// xpForNextLevel returns the XP needed to reach the next level from current XP.
func (s *Service) xpForNextLevel(xp int64) int64 {
	base := int64(s.config.Progression.BaseXPPerLevel)
	if base <= 0 {
		base = 1000
	}
	currentLevel := s.calculateLevelFromXP(xp)
	xpForNextLevel := currentLevel * base
	return xpForNextLevel - xp
}

// AddExperience adds XP to a player's progression and updates level if needed.
func (s *Service) AddExperience(ctx context.Context, playerID int64, xpGain int64) error {
	if xpGain <= 0 {
		return nil
	}
	// Get current progression
	progression, err := s.GetPlayerProgression(ctx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player progression: %w", err)
	}
	oldLevel := progression.Level
	// Add XP
	err = s.queries.IncrementExperience(ctx, s.dbConn, &db.IncrementExperienceParams{
		Experience: xpGain,
		PlayerID:   playerID,
	})
	if err != nil {
		return fmt.Errorf("failed to increment experience: %w", err)
	}
	// Compute new level based on updated XP (need to fetch again or compute)
	newXP := progression.Experience + xpGain
	newLevel := s.calculateLevelFromXP(newXP)
	if newLevel > oldLevel {
		// Level up
		err = s.queries.UpdateLevel(ctx, s.dbConn, &db.UpdateLevelParams{
			Level:    newLevel,
			PlayerID: playerID,
		})
		if err != nil {
			s.logger.Warn("Failed to update level after XP gain",
				zap.Int64("player_id", playerID),
				zap.Int64("new_level", newLevel),
				zap.Error(err))
			// Continue without returning error
		}
	}
	return nil
}

// AddMatchRewards updates player progression with match results and awards XP and data currency.
// This implements the XP calculation from match results as required by US-013.
func (s *Service) AddMatchRewards(ctx context.Context, playerID int64, kills, deaths, wavesSurvived, scrapEarned, dataEarned int64) error {
	if kills < 0 || deaths < 0 || wavesSurvived < 0 || scrapEarned < 0 || dataEarned < 0 {
		return fmt.Errorf("match stats cannot be negative")
	}
	// Calculate XP gain based on match performance
	// Base XP per match completion
	baseXP := int64(100)
	// XP per kill
	xpPerKill := int64(10)
	// XP per wave survived
	xpPerWave := int64(50)
	// XP per scrap (small amount)
	xpPerScrap := int64(1) // 1 XP per 10 scrap? We'll keep simple 1:1 for now

	totalXP := baseXP + (kills * xpPerKill) + (wavesSurvived * xpPerWave) + (scrapEarned * xpPerScrap)

	// Update match statistics
	err := s.queries.IncrementMatchStats(ctx, s.dbConn, &db.IncrementMatchStatsParams{
		TotalMatchesPlayed: 1,
		TotalWavesSurvived: wavesSurvived,
		TotalKills:         kills,
		TotalDeaths:        deaths,
		TotalScrapEarned:   scrapEarned,
		TotalDataEarned:    dataEarned,
		PlayerID:           playerID,
	})
	if err != nil {
		return fmt.Errorf("failed to increment match stats: %w", err)
	}
	// Award XP (which handles level-ups)
	err = s.AddExperience(ctx, playerID, totalXP)
	if err != nil {
		return fmt.Errorf("failed to add experience: %w", err)
	}
	// Award data currency 1:1 with data earned
	if dataEarned > 0 {
		err = s.queries.AddDataCurrency(ctx, s.dbConn, &db.AddDataCurrencyParams{
			DataCurrency: dataEarned,
			PlayerID:     playerID,
		})
		if err != nil {
			s.logger.Warn("Failed to add data currency",
				zap.Int64("player_id", playerID),
				zap.Int64("data_earned", dataEarned),
				zap.Error(err))
			// Continue without returning error
		}
	}
	return nil
}

// AddDataCurrencyWithTransaction adds data currency to a player and logs a transaction.
func (s *Service) AddDataCurrencyWithTransaction(ctx context.Context, playerID int64, amount int64, transactionType string, referenceID *int64) error {
	if amount == 0 {
		return nil
	}
	// Try to get *sql.DB for transaction support
	var dbTx db.DBTX
	var tx *sql.Tx
	var err error

	if db, ok := s.dbConn.(*sql.DB); ok {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		dbTx = tx
	} else {
		// Already a transaction or other DBTX; use directly without transaction
		s.logger.Warn("dbConn is not *sql.DB, proceeding without transaction")
		dbTx = s.dbConn
	}

	// Get current balance
	balance, err := s.queries.GetDataCurrency(ctx, dbTx, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create progression row
			if err := s.queries.CreatePlayerProgression(ctx, dbTx, playerID); err != nil {
				return fmt.Errorf("failed to create player progression: %w", err)
			}
			balance = 0
		} else {
			return fmt.Errorf("failed to get data currency: %w", err)
		}
	}
	newBalance := balance + amount
	// Update balance
	if err := s.queries.SetDataCurrency(ctx, dbTx, &db.SetDataCurrencyParams{
		DataCurrency: newBalance,
		PlayerID:     playerID,
	}); err != nil {
		return fmt.Errorf("failed to set data currency: %w", err)
	}
	// Log transaction
	if err := s.queries.CreateCurrencyTransaction(ctx, dbTx, &db.CreateCurrencyTransactionParams{
		PlayerID:        playerID,
		Amount:          amount,
		BalanceAfter:    newBalance,
		TransactionType: transactionType,
		ReferenceID:     referenceID,
	}); err != nil {
		return fmt.Errorf("failed to create currency transaction: %w", err)
	}
	// Commit transaction if we started one
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}
	s.logger.Debug("Data currency transaction completed",
		zap.Int64("player_id", playerID),
		zap.Int64("amount", amount),
		zap.Int64("new_balance", newBalance),
		zap.String("transaction_type", transactionType))
	return nil
}

// PrestigePlayer resets player level and experience, increments prestige level,
// and grants exclusive cosmetic items based on the new prestige level.
func (s *Service) PrestigePlayer(ctx context.Context, playerID int64) error {
	// Try to get *sql.DB for transaction support
	var dbTx db.DBTX
	var tx *sql.Tx
	var err error

	if db, ok := s.dbConn.(*sql.DB); ok {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		dbTx = tx
	} else {
		// Already a transaction or other DBTX; use directly without transaction
		s.logger.Warn("dbConn is not *sql.DB, proceeding without transaction")
		dbTx = s.dbConn
	}

	// Prestige player (reset level/XP, increment prestige)
	err = s.queries.PrestigePlayer(ctx, dbTx, playerID)
	if err != nil {
		return fmt.Errorf("failed to prestige player: %w", err)
	}

	// Get updated progression to know new prestige level
	progression, err := s.queries.GetPlayerProgression(ctx, dbTx, playerID)
	if err != nil {
		return fmt.Errorf("failed to get player progression: %w", err)
	}

	// Get prestige cosmetics not already owned
	cosmetics, err := s.queries.GetPrestigeCosmetics(ctx, dbTx, &db.GetPrestigeCosmeticsParams{
		PlayerID:    playerID,
		UnlockLevel: progression.PrestigeLevel,
	})
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to get prestige cosmetics: %w", err)
	}

	// Grant each cosmetic
	for _, cosmetic := range cosmetics {
		err = s.queries.GrantCosmeticToPlayer(ctx, dbTx, &db.GrantCosmeticToPlayerParams{
			PlayerID:    playerID,
			CosmeticID:  cosmetic.CosmeticID,
			UnlockedVia: "prestige",
		})
		if err != nil {
			// Log but continue - duplicate cosmetic ownership may occur
			s.logger.Warn("Failed to grant cosmetic to player",
				zap.Int64("player_id", playerID),
				zap.Int64("cosmetic_id", cosmetic.CosmeticID),
				zap.Error(err))
		}
	}

	// Commit transaction if we started one
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	s.logger.Info("Player prestiged successfully",
		zap.Int64("player_id", playerID),
		zap.Int64("new_prestige_level", progression.PrestigeLevel),
		zap.Int("cosmetics_granted", len(cosmetics)))
	return nil
}

// EquipCosmetic equips a cosmetic item to the player's active loadout.
func (s *Service) EquipCosmetic(ctx context.Context, playerID int64, cosmeticID int64) error {
	// Get cosmetic item to determine slot
	cosmetic, err := s.queries.GetCosmeticItem(ctx, s.dbConn, cosmeticID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCosmeticNotFound
		}
		return fmt.Errorf("failed to get cosmetic item: %w", err)
	}

	// Verify player owns the cosmetic
	_, err = s.queries.GetPlayerCosmetic(ctx, s.dbConn, &db.GetPlayerCosmeticParams{
		PlayerID:   playerID,
		CosmeticID: cosmeticID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrCosmeticNotOwned
		}
		return fmt.Errorf("failed to check cosmetic ownership: %w", err)
	}

	// Get or create active loadout
	loadout, err := s.queries.GetActiveLoadout(ctx, s.dbConn, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Create default loadout
			params := &db.CreateLoadoutParams{
				PlayerID: playerID,
				Name:     "Default",
				IsActive: 1,
			}
			err = s.queries.CreateLoadout(ctx, s.dbConn, params)
			if err != nil {
				return fmt.Errorf("failed to create default loadout: %w", err)
			}
			// Fetch newly created loadout (sqlite last_insert_rowid not directly available)
			// We'll retrieve the active loadout again
			loadout, err = s.queries.GetActiveLoadout(ctx, s.dbConn, playerID)
			if err != nil {
				return fmt.Errorf("failed to retrieve created loadout: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get active loadout: %w", err)
		}
	}

	// Use transaction to ensure atomic replace
	var dbTx db.DBTX
	var tx *sql.Tx
	if db, ok := s.dbConn.(*sql.DB); ok {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		dbTx = tx
	} else {
		// Already a transaction or other DBTX; use directly without transaction
		s.logger.Warn("dbConn is not *sql.DB, proceeding without transaction")
		dbTx = s.dbConn
	}

	// Remove any existing cosmetic in the same slot for this loadout
	err = s.queries.DeleteLoadoutCosmeticBySlot(ctx, dbTx, &db.DeleteLoadoutCosmeticBySlotParams{
		LoadoutID: loadout.LoadoutID,
		Slot:      cosmetic.Slot,
	})
	if err != nil {
		return fmt.Errorf("failed to clear slot: %w", err)
	}

	// Insert new mapping
	err = s.queries.InsertLoadoutCosmetic(ctx, dbTx, &db.InsertLoadoutCosmeticParams{
		LoadoutID:  loadout.LoadoutID,
		CosmeticID: cosmeticID,
		Slot:       cosmetic.Slot,
	})
	if err != nil {
		return fmt.Errorf("failed to equip cosmetic: %w", err)
	}

	// Commit transaction if we started one
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	s.logger.Info("Cosmetic equipped successfully",
		zap.Int64("player_id", playerID),
		zap.Int64("cosmetic_id", cosmeticID),
		zap.Int64("loadout_id", loadout.LoadoutID),
		zap.String("slot", cosmetic.Slot))
	return nil
}

// StoreMatchWithStats creates a new match record and associated player statistics.
// It uses a transaction to ensure atomicity.
func (s *Service) StoreMatchWithStats(ctx context.Context, serverID int64, matchParams *db.CreateMatchParams, playerStats []*db.CreatePlayerMatchStatsParams) error {
	// Ensure matchParams.ServerID matches the provided serverID
	if matchParams.ServerID != serverID {
		return fmt.Errorf("server ID mismatch: expected %d, got %d", serverID, matchParams.ServerID)
	}

	// Start transaction
	var dbTx db.DBTX
	var tx *sql.Tx
	var err error
	if db, ok := s.dbConn.(*sql.DB); ok {
		tx, err = db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback()
		dbTx = tx
	} else {
		s.logger.Warn("dbConn is not *sql.DB, proceeding without transaction")
		dbTx = s.dbConn
	}

	// Create match
	match, err := s.queries.CreateMatch(ctx, dbTx, matchParams)
	if err != nil {
		return fmt.Errorf("failed to create match: %w", err)
	}

	// Insert player stats
	for _, stats := range playerStats {
		// Ensure stats.MatchID matches the created match
		stats.MatchID = match.MatchID
		_, err := s.queries.CreatePlayerMatchStats(ctx, dbTx, stats)
		if err != nil {
			return fmt.Errorf("failed to create player match stats: %w", err)
		}
	}

	// Commit transaction if we started one
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	}

	s.logger.Info("Match stored successfully",
		zap.Int64("match_id", match.MatchID),
		zap.Int64("server_id", serverID),
		zap.Int("player_count", len(playerStats)))
	return nil
}

// GetPlayerMatchHistory retrieves a player's recent matches with their personal statistics.
func (s *Service) GetPlayerMatchHistory(ctx context.Context, playerID int64, limit int32) ([]*db.GetPlayerMatchHistoryRow, error) {
	matches, err := s.queries.GetPlayerMatchHistory(ctx, s.dbConn, &db.GetPlayerMatchHistoryParams{
		PlayerID: playerID,
		Limit:    int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get player match history: %w", err)
	}
	return matches, nil
}
