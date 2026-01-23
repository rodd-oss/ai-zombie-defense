package auth

import (
	"context"
	cryptorand "crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"ai-zombie-defense/db"
	"ai-zombie-defense/server/internal/services/account"
	"ai-zombie-defense/server/internal/services/auth"
	"ai-zombie-defense/server/internal/services/leaderboard"
	"ai-zombie-defense/server/internal/services/loot"
	"ai-zombie-defense/server/internal/services/match"
	"ai-zombie-defense/server/internal/services/progression"
	"ai-zombie-defense/server/internal/services/server"
	"ai-zombie-defense/server/internal/services/social"
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
	ErrServerNotFound             = server.ErrServerNotFound
	ErrJoinTokenInvalid           = server.ErrJoinTokenInvalid
	ErrJoinTokenExpired           = server.ErrJoinTokenExpired
	ErrJoinTokenAlreadyUsed       = server.ErrJoinTokenAlreadyUsed
	ErrFavoriteAlreadyExists      = server.ErrFavoriteAlreadyExists
	ErrFavoriteNotFound           = server.ErrFavoriteNotFound
	ErrLootTableNotFound          = loot.ErrLootTableNotFound
	ErrLootTableEntryNotFound     = loot.ErrLootTableEntryNotFound
	ErrInsufficientCurrency       = progression.ErrInsufficientCurrency
	ErrCosmeticAlreadyOwned       = progression.ErrCosmeticAlreadyOwned
	ErrFriendRequestAlreadyExists = social.ErrFriendRequestAlreadyExists
	ErrFriendRequestNotFound      = social.ErrFriendRequestNotFound
	ErrFriendRequestNotPending    = social.ErrFriendRequestNotPending
	ErrCannotFriendSelf           = social.ErrCannotFriendSelf
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
	matchSvc       match.Service
	serverSvc      server.Service
	socialSvc      social.Service
	leaderboardSvc leaderboard.Service
}

func NewService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) *Service {
	progressionSvc := progression.NewProgressionService(cfg, logger, dbConn)
	return &Service{
		config:         cfg,
		logger:         logger,
		dbConn:         dbConn,
		queries:        db.New(),
		authSvc:        auth.NewAuthService(cfg, logger, dbConn),
		accSvc:         account.NewAccountService(cfg, logger, dbConn),
		progressionSvc: progressionSvc,
		lootSvc:        loot.NewLootService(cfg, logger, dbConn),
		matchSvc:       match.NewMatchService(cfg, logger, dbConn, progressionSvc),
		serverSvc:      server.NewServerService(cfg, logger, dbConn),
		socialSvc:      social.NewSocialService(cfg, logger, dbConn),
		leaderboardSvc: leaderboard.NewLeaderboardService(cfg, logger, dbConn),
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

// AddDataCurrency adds data currency to a player and logs a transaction.
func (s *Service) AddDataCurrency(ctx context.Context, playerID int64, amount int64, transactionType string, referenceID *int64) error {
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
func (s *Service) StoreMatchWithStats(ctx context.Context, serverID int64, matchParams *db.CreateMatchParams, playerStats []*db.CreatePlayerMatchStatsParams) error {
	return s.matchSvc.StoreMatchWithStats(ctx, serverID, matchParams, playerStats)
}

// GetPlayerMatchHistory retrieves a player's recent matches with their personal statistics.
func (s *Service) GetPlayerMatchHistory(ctx context.Context, playerID int64, limit int32) ([]*db.GetPlayerMatchHistoryRow, error) {
	return s.matchSvc.GetPlayerMatchHistory(ctx, playerID, limit)
}

// RegisterServer registers a new dedicated server and returns authentication token.
func (s *Service) RegisterServer(ctx context.Context, ipAddress string, port int64, name string, mapRotation *string, maxPlayers int64, region *string, version *string) (*db.Server, string, error) {
	return s.serverSvc.RegisterServer(ctx, ipAddress, port, name, mapRotation, maxPlayers, region, version)
}

// GetServerByAuthToken retrieves a server by its authentication token.
func (s *Service) GetServerByAuthToken(ctx context.Context, authToken string) (*db.Server, error) {
	return s.serverSvc.GetServerByAuthToken(ctx, authToken)
}

// UpdateServerHeartbeat updates the last heartbeat timestamp, current player count, and map rotation for a server.
func (s *Service) UpdateServerHeartbeat(ctx context.Context, serverID int64, currentPlayers int64, mapRotation *string) error {
	return s.serverSvc.UpdateServerHeartbeat(ctx, serverID, currentPlayers, mapRotation)
}

// ListActiveServers returns a filtered list of active servers.
func (s *Service) ListActiveServers(ctx context.Context, region, mapRotation, version *string, minPlayers, maxPlayers *int64) ([]*db.Server, error) {
	return s.serverSvc.ListActiveServers(ctx, region, mapRotation, version, minPlayers, maxPlayers)
}

// GenerateJoinToken creates a new join token for a player to join a specific server.
// The token expires after the specified duration.
func (s *Service) GenerateJoinToken(ctx context.Context, playerID int64, serverID int64, expiresIn time.Duration) (string, error) {
	return s.serverSvc.GenerateJoinToken(ctx, playerID, serverID, expiresIn)
}

// ValidateJoinToken validates a join token and returns the associated player and server IDs.
func (s *Service) ValidateJoinToken(ctx context.Context, token string) (playerID int64, serverID int64, err error) {
	return s.serverSvc.ValidateJoinToken(ctx, token)
}

// MarkTokenUsed marks a join token as used (consumed).
func (s *Service) MarkTokenUsed(ctx context.Context, token string) error {
	return s.serverSvc.MarkTokenUsed(ctx, token)
}

// AddFavorite adds a server to the player's favorites.
func (s *Service) AddFavorite(ctx context.Context, playerID int64, serverID int64, note *string) error {
	return s.serverSvc.AddFavorite(ctx, playerID, serverID, note)
}

// RemoveFavorite removes a server from the player's favorites.
func (s *Service) RemoveFavorite(ctx context.Context, playerID int64, serverID int64) error {
	return s.serverSvc.RemoveFavorite(ctx, playerID, serverID)
}

// GetFavorite retrieves a favorite entry.
func (s *Service) GetFavorite(ctx context.Context, playerID int64, serverID int64) (*db.ServerFavorite, error) {
	// Re-implementing since it's simple and I didn't add it to interface yet?
	// Actually I should add it to interface if it's needed.
	// Let's check the interface.
	return s.queries.GetFavorite(ctx, s.dbConn, &db.GetFavoriteParams{
		PlayerID: playerID,
		ServerID: serverID,
	})
}

// ListPlayerFavorites returns the player's favorite servers with server details.
func (s *Service) ListPlayerFavorites(ctx context.Context, playerID int64) ([]*db.ListPlayerFavoritesRow, error) {
	return s.serverSvc.ListPlayerFavorites(ctx, playerID)
}

// SendFriendRequest sends a friend request from playerID to friendID.
func (s *Service) SendFriendRequest(ctx context.Context, playerID int64, friendID int64) error {
	return s.socialSvc.SendFriendRequest(ctx, playerID, friendID)
}

// AcceptFriendRequest accepts a pending friend request.
func (s *Service) AcceptFriendRequest(ctx context.Context, requesterPlayerID int64, friendID int64) error {
	return s.socialSvc.AcceptFriendRequest(ctx, requesterPlayerID, friendID)
}

// DeclineFriendRequest declines a pending friend request.
func (s *Service) DeclineFriendRequest(ctx context.Context, requesterPlayerID int64, friendID int64) error {
	return s.socialSvc.DeclineFriendRequest(ctx, requesterPlayerID, friendID)
}

// ListFriends returns the player's accepted friends.
func (s *Service) ListFriends(ctx context.Context, playerID int64) ([]*db.ListFriendsRow, error) {
	return s.socialSvc.ListFriends(ctx, playerID)
}

// ListPendingIncoming returns pending incoming friend requests.
func (s *Service) ListPendingIncoming(ctx context.Context, playerID int64) ([]*db.ListPendingIncomingRow, error) {
	return s.socialSvc.ListPendingIncoming(ctx, playerID)
}

// ListPendingOutgoing returns pending outgoing friend requests.
func (s *Service) ListPendingOutgoing(ctx context.Context, playerID int64) ([]*db.ListPendingOutgoingRow, error) {
	return s.socialSvc.ListPendingOutgoing(ctx, playerID)
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
	return s.leaderboardSvc.GetDailyLeaderboard(ctx)
}

// GetWeeklyLeaderboard returns the weekly leaderboard rankings.
func (s *Service) GetWeeklyLeaderboard(ctx context.Context) ([]*db.GetWeeklyLeaderboardRow, error) {
	return s.leaderboardSvc.GetWeeklyLeaderboard(ctx)
}

// GetAllTimeLeaderboard returns the all-time leaderboard rankings.
func (s *Service) GetAllTimeLeaderboard(ctx context.Context) ([]*db.GetAllTimeLeaderboardRow, error) {
	return s.leaderboardSvc.GetAllTimeLeaderboard(ctx)
}
