package auth

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"ai-zombie-defense/db"
	"ai-zombie-defense/db/types"
	"ai-zombie-defense/server/internal/services/account"
	"ai-zombie-defense/server/internal/services/auth"
	"ai-zombie-defense/server/internal/services/loot"
	"ai-zombie-defense/server/internal/services/progression"
	"ai-zombie-defense/server/pkg/config"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials         = auth.ErrInvalidCredentials
	ErrPlayerBanned               = auth.ErrPlayerBanned
	ErrDuplicateUsername          = account.ErrDuplicateUsername
	ErrDuplicateEmail             = account.ErrDuplicateEmail
	ErrInvalidRefreshToken        = auth.ErrInvalidRefreshToken
	ErrSessionNotFound            = auth.ErrSessionNotFound
	ErrCosmeticNotFound           = progression.ErrCosmeticNotFound
	ErrCosmeticNotOwned           = progression.ErrCosmeticNotOwned
	ErrLoadoutNotFound            = progression.ErrLoadoutNotFound
	ErrMatchNotFound              = errors.New("match not found")
	ErrServerNotFound             = errors.New("server not found")
	ErrJoinTokenInvalid           = errors.New("join token invalid")
	ErrJoinTokenExpired           = errors.New("join token expired")
	ErrJoinTokenAlreadyUsed       = errors.New("join token already used")
	ErrFavoriteAlreadyExists      = errors.New("server already favorited")
	ErrFavoriteNotFound           = errors.New("favorite not found")
	ErrLootTableNotFound          = loot.ErrLootTableNotFound
	ErrLootTableEntryNotFound     = loot.ErrLootTableEntryNotFound
	ErrInsufficientCurrency       = progression.ErrInsufficientCurrency
	ErrCosmeticAlreadyOwned       = progression.ErrCosmeticAlreadyOwned
	ErrFriendRequestAlreadyExists = errors.New("friend request already exists")
	ErrFriendRequestNotFound      = errors.New("friend request not found")
	ErrFriendRequestNotPending    = errors.New("friend request not pending")
	ErrCannotFriendSelf           = errors.New("cannot send friend request to yourself")
)

type Service struct {
	config         config.Config
	logger         *zap.Logger
	dbConn         db.DBTX
	queries        *db.Queries
	authSvc        auth.Service
	accSvc         account.Service
	progressionSvc progression.Service
	lootSvc        loot.Service
}

func NewService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) *Service {
	return &Service{
		config:         cfg,
		logger:         logger,
		dbConn:         dbConn,
		queries:        db.New(),
		authSvc:        auth.NewAuthService(cfg, logger, dbConn),
		accSvc:         account.NewAccountService(cfg, logger, dbConn),
		progressionSvc: progression.NewProgressionService(cfg, logger, dbConn),
		lootSvc:        loot.NewLootService(cfg, logger, dbConn),
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

// Authenticate validates username/email and password.
func (s *Service) Authenticate(ctx context.Context, usernameOrEmail, password string) (*db.Player, error) {
	return s.authSvc.Authenticate(ctx, usernameOrEmail, password)
}

// RegisterPlayer creates a new player account.
func (s *Service) RegisterPlayer(ctx context.Context, username, email, password string) (*db.Player, error) {
	return s.authSvc.RegisterPlayer(ctx, username, email, password)
}

// GenerateAccessToken creates a JWT token for a player.
func (s *Service) GenerateAccessToken(playerID int64) (string, error) {
	return s.authSvc.GenerateAccessToken(playerID)
}

// GenerateRefreshToken creates a refresh token for a player.
func (s *Service) GenerateRefreshToken(playerID int64) (string, error) {
	// Keep internal for now as it's not in the interface, but used by CreateSession
	exp := time.Now().Add(s.config.JWT.RefreshExpiration)
	// Generate random JWT ID to ensure uniqueness
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

// ValidateToken parses and validates a JWT token.
func (s *Service) ValidateToken(tokenString string) (*jwt.RegisteredClaims, error) {
	return s.authSvc.ValidateToken(tokenString)
}

// CreateSession creates a new refresh token session for a player.
func (s *Service) CreateSession(ctx context.Context, playerID int64, ipAddress, userAgent string) (string, error) {
	return s.authSvc.CreateSession(ctx, playerID, ipAddress, userAgent)
}

// ValidateRefreshToken validates a refresh token and returns the player ID.
func (s *Service) ValidateRefreshToken(ctx context.Context, token string) (int64, error) {
	// Re-implementing here to avoid making it public in internal/services/auth if possible,
	// or just use it if I want to expose it.
	// Actually, the new service has it as a private method.
	// For now, I'll keep the logic here or expose it.
	// Let's check if I can just use it.

	// Since I want to "move logic", I should probably just keep it here for now
	// as it's used by RefreshSession which I will delegate.

	// Wait, if I delegate RefreshSession, I don't need ValidateRefreshToken in pkg/auth unless someone else uses it.
	return s.validateRefreshToken(ctx, token)
}

func (s *Service) validateRefreshToken(ctx context.Context, token string) (int64, error) {
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
	return s.authSvc.DeleteSession(ctx, token)
}

// RefreshSession validates an existing refresh token and creates a new session.
func (s *Service) RefreshSession(ctx context.Context, oldToken, ipAddress, userAgent string) (playerID int64, newToken string, err error) {
	return s.authSvc.RefreshSession(ctx, oldToken, ipAddress, userAgent)
}

// GetPlayer retrieves a player by ID.
func (s *Service) GetPlayer(ctx context.Context, playerID int64) (*db.Player, error) {
	return s.accSvc.GetPlayer(ctx, playerID)
}

// UpdatePlayerProfile updates a player's username and email.
func (s *Service) UpdatePlayerProfile(ctx context.Context, playerID int64, username, email string) error {
	return s.accSvc.UpdatePlayerProfile(ctx, playerID, username, email)
}

// UpdatePlayerPassword updates a player's password.
func (s *Service) UpdatePlayerPassword(ctx context.Context, playerID int64, newPassword string) error {
	return s.accSvc.UpdatePlayerPassword(ctx, playerID, newPassword)
}

// GetPlayerSettings retrieves player settings or returns defaults if not found.
func (s *Service) GetPlayerSettings(ctx context.Context, playerID int64) (*db.PlayerSetting, error) {
	return s.accSvc.GetPlayerSettings(ctx, playerID)
}

// GetPlayerProgression retrieves player progression or creates a default row if not found.
func (s *Service) GetPlayerProgression(ctx context.Context, playerID int64) (*db.PlayerProgression, error) {
	return s.progressionSvc.GetPlayerProgression(ctx, playerID)
}

// GetCosmeticCatalog returns all cosmetic items available in the catalog.
func (s *Service) GetCosmeticCatalog(ctx context.Context) ([]*db.CosmeticItem, error) {
	return s.progressionSvc.GetCosmeticCatalog(ctx)
}

// GetPlayerCosmetics returns cosmetic items owned by the player.
func (s *Service) GetPlayerCosmetics(ctx context.Context, playerID int64) ([]*db.GetPlayerCosmeticsRow, error) {
	return s.progressionSvc.GetPlayerCosmetics(ctx, playerID)
}

// UpsertPlayerSettings creates or updates player settings.
func (s *Service) UpsertPlayerSettings(ctx context.Context, params *db.UpsertPlayerSettingsParams) error {
	return s.accSvc.UpsertPlayerSettings(ctx, params)
}

// Config returns the service configuration.
func (s *Service) Config() config.Config {
	return s.config
}

// AddExperience adds XP to a player's progression and updates level if needed.
func (s *Service) AddExperience(ctx context.Context, playerID int64, xpGain int64) error {
	return s.progressionSvc.AddExperience(ctx, playerID, xpGain)
}

// AddMatchRewards updates player progression with match results and awards XP and data currency.
func (s *Service) AddMatchRewards(ctx context.Context, playerID int64, kills, deaths, wavesSurvived, scrapEarned, dataEarned int64) error {
	return s.progressionSvc.AddMatchRewards(ctx, playerID, kills, deaths, wavesSurvived, scrapEarned, dataEarned)
}

// AddDataCurrencyWithTransaction adds data currency to a player and logs a transaction.
func (s *Service) AddDataCurrencyWithTransaction(ctx context.Context, playerID int64, amount int64, transactionType string, referenceID *int64) error {
	return s.progressionSvc.AddDataCurrency(ctx, playerID, amount, transactionType, referenceID)
}

// PrestigePlayer resets player level and experience, increments prestige level,
// and grants exclusive cosmetic items based on the new prestige level.
func (s *Service) PrestigePlayer(ctx context.Context, playerID int64) error {
	return s.progressionSvc.PrestigePlayer(ctx, playerID)
}

// EquipCosmetic equips a cosmetic item to the player's active loadout.
func (s *Service) EquipCosmetic(ctx context.Context, playerID int64, cosmeticID int64) error {
	return s.progressionSvc.EquipCosmetic(ctx, playerID, cosmeticID)
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

	// Award rewards based on player performance
	for _, stats := range playerStats {
		err := s.addMatchRewardsWithTx(ctx, dbTx, stats.PlayerID, stats.ZombiesKilled, stats.Deaths, stats.WavesSurvived, stats.ScrapEarned, stats.DataEarned)
		if err != nil {
			return fmt.Errorf("failed to award match rewards: %w", err)
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

// addMatchRewardsWithTx is a temporary helper for StoreMatchWithStats until it moves to MatchService.
func (s *Service) addMatchRewardsWithTx(ctx context.Context, dbTx db.DBTX, playerID int64, kills, deaths, wavesSurvived, scrapEarned, dataEarned int64) error {
	if kills < 0 || deaths < 0 || wavesSurvived < 0 || scrapEarned < 0 || dataEarned < 0 {
		return fmt.Errorf("match stats cannot be negative")
	}
	baseXP := int64(100)
	xpPerKill := int64(10)
	xpPerWave := int64(50)
	xpPerScrap := int64(1)

	totalXP := baseXP + (kills * xpPerKill) + (wavesSurvived * xpPerWave) + (scrapEarned * xpPerScrap)

	err := s.queries.IncrementMatchStats(ctx, dbTx, &db.IncrementMatchStatsParams{
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
	err = s.addExperienceWithTx(ctx, dbTx, playerID, totalXP)
	if err != nil {
		return fmt.Errorf("failed to add experience: %w", err)
	}
	if dataEarned > 0 {
		err = s.queries.AddDataCurrency(ctx, dbTx, &db.AddDataCurrencyParams{
			DataCurrency: dataEarned,
			PlayerID:     playerID,
		})
		if err != nil {
			s.logger.Warn("Failed to add data currency",
				zap.Int64("player_id", playerID),
				zap.Int64("data_earned", dataEarned),
				zap.Error(err))
		}
	}
	return nil
}

// addExperienceWithTx is a temporary helper for StoreMatchWithStats.
func (s *Service) addExperienceWithTx(ctx context.Context, dbTx db.DBTX, playerID int64, xpGain int64) error {
	if xpGain <= 0 {
		return nil
	}
	progression, err := s.queries.GetPlayerProgression(ctx, dbTx, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := s.queries.CreatePlayerProgression(ctx, dbTx, playerID); err != nil {
				return fmt.Errorf("failed to create player progression: %w", err)
			}
			progression = &db.PlayerProgression{
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
			}
		} else {
			return fmt.Errorf("failed to get player progression: %w", err)
		}
	}
	oldLevel := progression.Level
	err = s.queries.IncrementExperience(ctx, dbTx, &db.IncrementExperienceParams{
		Experience: xpGain,
		PlayerID:   playerID,
	})
	if err != nil {
		return fmt.Errorf("failed to increment experience: %w", err)
	}
	newXP := progression.Experience + xpGain
	newLevel := s.calculateLevelFromXP(newXP)
	if newLevel > oldLevel {
		err = s.queries.UpdateLevel(ctx, dbTx, &db.UpdateLevelParams{
			Level:    newLevel,
			PlayerID: playerID,
		})
		if err != nil {
			s.logger.Warn("Failed to update level after XP gain",
				zap.Int64("player_id", playerID),
				zap.Int64("new_level", newLevel),
				zap.Error(err))
		}
	}
	return nil
}

// calculateLevelFromXP is a temporary helper for StoreMatchWithStats.
func (s *Service) calculateLevelFromXP(xp int64) int64 {
	if xp <= 0 {
		return 1
	}
	base := int64(s.config.Progression.BaseXPPerLevel)
	if base <= 0 {
		base = 1000
	}
	level := xp/base + 1
	if level < 1 {
		return 1
	}
	return level
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

// RegisterServer registers a new dedicated server and returns authentication token.
func (s *Service) RegisterServer(ctx context.Context, ipAddress string, port int64, name string, mapRotation *string, maxPlayers int64, region *string, version *string) (*db.Server, string, error) {
	// Generate random authentication token (hex)
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

// GetServerByAuthToken retrieves a server by its authentication token.
func (s *Service) GetServerByAuthToken(ctx context.Context, authToken string) (*db.Server, error) {
	server, err := s.queries.GetServerByAuthToken(ctx, s.dbConn, &authToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get server by auth token: %w", err)
	}
	return server, nil
}

// UpdateServerHeartbeat updates the last heartbeat timestamp, current player count, and map rotation for a server.
func (s *Service) UpdateServerHeartbeat(ctx context.Context, serverID int64, currentPlayers int64, mapRotation *string) error {
	// Use current time as heartbeat timestamp
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

// ListActiveServers returns a filtered list of active servers.
func (s *Service) ListActiveServers(ctx context.Context, region, mapRotation, version *string, minPlayers, maxPlayers *int64) ([]*db.Server, error) {
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

// GenerateJoinToken creates a new join token for a player to join a specific server.
// The token expires after the specified duration.
func (s *Service) GenerateJoinToken(ctx context.Context, playerID int64, serverID int64, expiresIn time.Duration) (string, error) {
	// Generate random token (hex)
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

// ValidateJoinToken validates a join token and returns the associated player and server IDs.
// Returns ErrJoinTokenInvalid if token not found, ErrJoinTokenExpired if expired,
// ErrJoinTokenAlreadyUsed if already used.
func (s *Service) ValidateJoinToken(ctx context.Context, token string) (playerID int64, serverID int64, err error) {
	joinToken, err := s.queries.GetValidJoinToken(ctx, s.dbConn, token)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Could be invalid, expired, or already used
			// Try to get token to determine exact reason
			tokenRow, err2 := s.queries.GetJoinToken(ctx, s.dbConn, token)
			if err2 != nil {
				if errors.Is(err2, sql.ErrNoRows) {
					return 0, 0, ErrJoinTokenInvalid
				}
				return 0, 0, fmt.Errorf("failed to get join token: %w", err2)
			}
			// Token exists but not valid
			expiresAt := tokenRow.ExpiresAt.Time
			if expiresAt.Before(time.Now().UTC()) {
				return 0, 0, ErrJoinTokenExpired
			}
			if tokenRow.UsedAt.Valid {
				return 0, 0, ErrJoinTokenAlreadyUsed
			}
			// Should not happen (token not expired, not used, but still invalid?)
			return 0, 0, ErrJoinTokenInvalid
		}
		return 0, 0, fmt.Errorf("failed to validate join token: %w", err)
	}

	// Token is valid
	return joinToken.PlayerID, joinToken.ServerID, nil
}

// MarkTokenUsed marks a join token as used (consumed).
func (s *Service) MarkTokenUsed(ctx context.Context, token string) error {
	err := s.queries.MarkTokenUsed(ctx, s.dbConn, token)
	if err != nil {
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	s.logger.Debug("Join token marked as used", zap.String("token", token))
	return nil
}

// AddFavorite adds a server to the player's favorites.
func (s *Service) AddFavorite(ctx context.Context, playerID int64, serverID int64, note *string) error {
	// Check if favorite already exists
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
	// Insert favorite
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

// RemoveFavorite removes a server from the player's favorites.
func (s *Service) RemoveFavorite(ctx context.Context, playerID int64, serverID int64) error {
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

// GetFavorite retrieves a favorite entry.
func (s *Service) GetFavorite(ctx context.Context, playerID int64, serverID int64) (*db.ServerFavorite, error) {
	params := &db.GetFavoriteParams{
		PlayerID: playerID,
		ServerID: serverID,
	}
	fav, err := s.queries.GetFavorite(ctx, s.dbConn, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrFavoriteNotFound
		}
		return nil, fmt.Errorf("failed to get favorite: %w", err)
	}
	return fav, nil
}

// ListPlayerFavorites returns the player's favorite servers with server details.
func (s *Service) ListPlayerFavorites(ctx context.Context, playerID int64) ([]*db.ListPlayerFavoritesRow, error) {
	favorites, err := s.queries.ListPlayerFavorites(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list player favorites: %w", err)
	}
	return favorites, nil
}

// SendFriendRequest sends a friend request from playerID to friendID.
func (s *Service) SendFriendRequest(ctx context.Context, playerID int64, friendID int64) error {
	if playerID == friendID {
		return ErrCannotFriendSelf
	}
	existing, err := s.queries.GetFriendRequest(ctx, s.dbConn, &db.GetFriendRequestParams{
		PlayerID: playerID,
		FriendID: friendID,
	})
	if err == nil && existing != nil {
		return ErrFriendRequestAlreadyExists
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("failed to check existing friend request: %w", err)
	}
	params := &db.CreateFriendRequestParams{
		PlayerID: playerID,
		FriendID: friendID,
	}
	err = s.queries.CreateFriendRequest(ctx, s.dbConn, params)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed: friends.player_id, friends.friend_id") {
			return ErrFriendRequestAlreadyExists
		}
		return fmt.Errorf("failed to create friend request: %w", err)
	}
	s.logger.Debug("Friend request sent", zap.Int64("player_id", playerID), zap.Int64("friend_id", friendID))
	return nil
}

// AcceptFriendRequest accepts a pending friend request.
func (s *Service) AcceptFriendRequest(ctx context.Context, requesterPlayerID int64, friendID int64) error {
	request, err := s.queries.GetFriendRequest(ctx, s.dbConn, &db.GetFriendRequestParams{
		PlayerID: requesterPlayerID,
		FriendID: friendID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFriendRequestNotFound
		}
		return fmt.Errorf("failed to get friend request: %w", err)
	}
	if request.Status != "pending" {
		return ErrFriendRequestNotPending
	}
	params := &db.AcceptFriendRequestParams{
		PlayerID: requesterPlayerID,
		FriendID: friendID,
	}
	err = s.queries.AcceptFriendRequest(ctx, s.dbConn, params)
	if err != nil {
		return fmt.Errorf("failed to accept friend request: %w", err)
	}
	s.logger.Debug("Friend request accepted", zap.Int64("player_id", requesterPlayerID), zap.Int64("friend_id", friendID))
	return nil
}

// DeclineFriendRequest declines a pending friend request.
func (s *Service) DeclineFriendRequest(ctx context.Context, requesterPlayerID int64, friendID int64) error {
	request, err := s.queries.GetFriendRequest(ctx, s.dbConn, &db.GetFriendRequestParams{
		PlayerID: requesterPlayerID,
		FriendID: friendID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrFriendRequestNotFound
		}
		return fmt.Errorf("failed to get friend request: %w", err)
	}
	if request.Status != "pending" {
		return ErrFriendRequestNotPending
	}
	params := &db.DeclineFriendRequestParams{
		PlayerID: requesterPlayerID,
		FriendID: friendID,
	}
	err = s.queries.DeclineFriendRequest(ctx, s.dbConn, params)
	if err != nil {
		return fmt.Errorf("failed to decline friend request: %w", err)
	}
	s.logger.Debug("Friend request declined", zap.Int64("player_id", requesterPlayerID), zap.Int64("friend_id", friendID))
	return nil
}

// ListFriends returns the player's accepted friends.
func (s *Service) ListFriends(ctx context.Context, playerID int64) ([]*db.ListFriendsRow, error) {
	friends, err := s.queries.ListFriends(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list friends: %w", err)
	}
	return friends, nil
}

// ListPendingIncoming returns pending incoming friend requests.
func (s *Service) ListPendingIncoming(ctx context.Context, playerID int64) ([]*db.ListPendingIncomingRow, error) {
	requests, err := s.queries.ListPendingIncoming(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending incoming requests: %w", err)
	}
	return requests, nil
}

// ListPendingOutgoing returns pending outgoing friend requests.
func (s *Service) ListPendingOutgoing(ctx context.Context, playerID int64) ([]*db.ListPendingOutgoingRow, error) {
	requests, err := s.queries.ListPendingOutgoing(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending outgoing requests: %w", err)
	}
	return requests, nil
}

// CreateLootTable creates a new loot table.
func (s *Service) CreateLootTable(ctx context.Context, name string, description *string, dropChance float64, isActive bool) (*db.LootTable, error) {
	return s.lootSvc.CreateLootTable(ctx, name, description, dropChance, isActive)
}

// GetLootTable retrieves a loot table by ID.
func (s *Service) GetLootTable(ctx context.Context, lootTableID int64) (*db.LootTable, error) {
	return s.lootSvc.GetLootTable(ctx, lootTableID)
}

// ListLootTables returns all loot tables.
func (s *Service) ListLootTables(ctx context.Context) ([]*db.LootTable, error) {
	return s.lootSvc.ListLootTables(ctx)
}

// ListActiveLootTables returns only active loot tables.
func (s *Service) ListActiveLootTables(ctx context.Context) ([]*db.LootTable, error) {
	return s.lootSvc.ListActiveLootTables(ctx)
}

// UpdateLootTable updates an existing loot table.
func (s *Service) UpdateLootTable(ctx context.Context, lootTableID int64, name string, description *string, dropChance float64, isActive bool) error {
	return s.lootSvc.UpdateLootTable(ctx, lootTableID, name, description, dropChance, isActive)
}

// DeleteLootTable deletes a loot table (hard delete).
func (s *Service) DeleteLootTable(ctx context.Context, lootTableID int64) error {
	return s.lootSvc.DeleteLootTable(ctx, lootTableID)
}

// CreateLootTableEntry creates a new entry in a loot table.
func (s *Service) CreateLootTableEntry(ctx context.Context, lootTableID int64, cosmeticID int64, weight int64, minQuantity int64, maxQuantity int64) (*db.LootTableEntry, error) {
	return s.lootSvc.CreateLootTableEntry(ctx, lootTableID, cosmeticID, weight, minQuantity, maxQuantity)
}

// GetLootTableEntry retrieves a loot table entry by ID.
func (s *Service) GetLootTableEntry(ctx context.Context, lootEntryID int64) (*db.LootTableEntry, error) {
	return s.lootSvc.GetLootTableEntry(ctx, lootEntryID)
}

// GetLootTableEntriesByLootTableID returns all entries for a loot table.
func (s *Service) GetLootTableEntriesByLootTableID(ctx context.Context, lootTableID int64) ([]*db.LootTableEntry, error) {
	return s.lootSvc.GetLootTableEntriesByLootTableID(ctx, lootTableID)
}

// GetLootTableEntriesWithCosmeticDetails returns entries with cosmetic details.
func (s *Service) GetLootTableEntriesWithCosmeticDetails(ctx context.Context, lootTableID int64) ([]*db.GetLootTableEntriesWithCosmeticDetailsRow, error) {
	return s.lootSvc.GetLootTableEntriesWithCosmeticDetails(ctx, lootTableID)
}

// UpdateLootTableEntry updates an existing loot table entry.
func (s *Service) UpdateLootTableEntry(ctx context.Context, lootEntryID int64, lootTableID int64, cosmeticID int64, weight int64, minQuantity int64, maxQuantity int64) error {
	return s.lootSvc.UpdateLootTableEntry(ctx, lootEntryID, lootTableID, cosmeticID, weight, minQuantity, maxQuantity)
}

// DeleteLootTableEntry deletes a loot table entry.
func (s *Service) DeleteLootTableEntry(ctx context.Context, lootEntryID int64) error {
	return s.lootSvc.DeleteLootTableEntry(ctx, lootEntryID)
}

// GenerateLootDrop generates a random cosmetic drop for the player based on active loot tables.
func (s *Service) GenerateLootDrop(ctx context.Context, playerID int64) (*db.CosmeticItem, error) {
	return s.lootSvc.GenerateLootDrop(ctx, playerID)
}

// PurchaseCosmetic purchases a cosmetic item with Data currency.
// Deducts data_cost from player's balance, logs a transaction, and grants cosmetic ownership.
func (s *Service) PurchaseCosmetic(ctx context.Context, playerID int64, cosmeticID int64) error {
	return s.progressionSvc.PurchaseCosmetic(ctx, playerID, cosmeticID)
}

// IsAdmin checks if a player is an administrator.
func (s *Service) IsAdmin(ctx context.Context, playerID int64) (bool, error) {
	player, err := s.queries.GetPlayer(ctx, s.dbConn, playerID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil // player not found, not admin
		}
		return false, fmt.Errorf("failed to check admin status: %w", err)
	}
	return player.IsAdmin == 1, nil
}

// GetDailyLeaderboard returns the daily leaderboard rankings.
func (s *Service) GetDailyLeaderboard(ctx context.Context) ([]*db.GetDailyLeaderboardRow, error) {
	entries, err := s.queries.GetDailyLeaderboard(ctx, s.dbConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily leaderboard: %w", err)
	}
	return entries, nil
}

// GetWeeklyLeaderboard returns the weekly leaderboard rankings.
func (s *Service) GetWeeklyLeaderboard(ctx context.Context) ([]*db.GetWeeklyLeaderboardRow, error) {
	entries, err := s.queries.GetWeeklyLeaderboard(ctx, s.dbConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get weekly leaderboard: %w", err)
	}
	return entries, nil
}

// GetAllTimeLeaderboard returns the all-time leaderboard rankings.
func (s *Service) GetAllTimeLeaderboard(ctx context.Context) ([]*db.GetAllTimeLeaderboardRow, error) {
	entries, err := s.queries.GetAllTimeLeaderboard(ctx, s.dbConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get all-time leaderboard: %w", err)
	}
	return entries, nil
}
