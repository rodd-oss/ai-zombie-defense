package handlers

import (
	"ai-zombie-defense/db"
	"ai-zombie-defense/db/types"
	"ai-zombie-defense/server/internal/services/account"
	"ai-zombie-defense/server/internal/services/match"
	"ai-zombie-defense/server/internal/services/progression"
	"ai-zombie-defense/server/internal/services/server"
	"ai-zombie-defense/server/pkg/middleware"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type AccountHandlers struct {
	accSvc         account.Service
	progressionSvc progression.Service
	matchSvc       match.Service
	logger         *zap.Logger
}

func NewAccountHandlers(accSvc account.Service, progressionSvc progression.Service, matchSvc match.Service, logger *zap.Logger) *AccountHandlers {
	return &AccountHandlers{
		accSvc:         accSvc,
		progressionSvc: progressionSvc,
		matchSvc:       matchSvc,
		logger:         logger,
	}
}

type ProfileResponse struct {
	PlayerID    int64   `json:"player_id"`
	Username    string  `json:"username"`
	Email       string  `json:"email"`
	CreatedAt   string  `json:"created_at"`
	LastLoginAt *string `json:"last_login_at,omitempty"`
	IsBanned    bool    `json:"is_banned"`
}

type UpdateProfileRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

type SettingsResponse struct {
	PlayerID         int64    `json:"player_id"`
	KeyBindings      *string  `json:"key_bindings,omitempty"`
	MouseSensitivity *float64 `json:"mouse_sensitivity,omitempty"`
	UiScale          *float64 `json:"ui_scale,omitempty"`
	ColorBlindMode   int64    `json:"color_blind_mode"`
	SubtitlesEnabled int64    `json:"subtitles_enabled"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}

type ProgressionResponse struct {
	PlayerID           int64  `json:"player_id"`
	Level              int64  `json:"level"`
	Experience         int64  `json:"experience"`
	PrestigeLevel      int64  `json:"prestige_level"`
	DataCurrency       int64  `json:"data_currency"`
	TotalMatchesPlayed int64  `json:"total_matches_played"`
	TotalWavesSurvived int64  `json:"total_waves_survived"`
	TotalKills         int64  `json:"total_kills"`
	TotalDeaths        int64  `json:"total_deaths"`
	TotalScrapEarned   int64  `json:"total_scrap_earned"`
	TotalDataEarned    int64  `json:"total_data_earned"`
	UpdatedAt          string `json:"updated_at"`
}

type PrestigeResponse struct {
	Message          string  `json:"message"`
	NewPrestigeLevel int64   `json:"new_prestige_level"`
	GrantedCosmetics []int64 `json:"granted_cosmetics"`
}

type UpdateSettingsRequest struct {
	KeyBindings      *string  `json:"key_bindings"`
	MouseSensitivity *float64 `json:"mouse_sensitivity"`
	UiScale          *float64 `json:"ui_scale"`
	ColorBlindMode   int64    `json:"color_blind_mode"`
	SubtitlesEnabled int64    `json:"subtitles_enabled"`
}

type PlayerMatchStatsRequest struct {
	PlayerID           int64 `json:"player_id"`
	WavesSurvived      int64 `json:"waves_survived"`
	ZombiesKilled      int64 `json:"zombies_killed"`
	Deaths             int64 `json:"deaths"`
	ScrapEarned        int64 `json:"scrap_earned"`
	DataEarned         int64 `json:"data_earned"`
	DamageDealt        int64 `json:"damage_dealt"`
	DamageTaken        int64 `json:"damage_taken"`
	BuildingsBuilt     int64 `json:"buildings_built"`
	BuildingsDestroyed int64 `json:"buildings_destroyed"`
	HealingGiven       int64 `json:"healing_given"`
	Revives            int64 `json:"revives"`
	Score              int64 `json:"score"`
}

type StoreMatchRequest struct {
	ServerID           int64                     `json:"server_id"`
	MapName            string                    `json:"map_name"`
	GameMode           string                    `json:"game_mode"`
	StartTime          types.Timestamp           `json:"start_time"`
	EndTime            *types.NullTimestamp      `json:"end_time,omitempty"`
	Outcome            string                    `json:"outcome"`
	WavesSurvived      int64                     `json:"waves_survived"`
	TotalZombiesKilled int64                     `json:"total_zombies_killed"`
	TotalPlayers       int64                     `json:"total_players"`
	PlayerStats        []PlayerMatchStatsRequest `json:"player_stats"`
}

// GetProfile handles GET /account/profile
func (h *AccountHandlers) GetProfile(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	ctx := c.Context()
	player, err := h.accSvc.GetPlayer(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	// Convert timestamps to ISO 8601 strings
	createdAt := player.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
	var lastLoginAt *string
	if player.LastLoginAt.Valid {
		str := player.LastLoginAt.Time.Format("2006-01-02T15:04:05Z")
		lastLoginAt = &str
	}

	resp := ProfileResponse{
		PlayerID:    player.PlayerID,
		Username:    player.Username,
		Email:       player.Email,
		CreatedAt:   createdAt,
		LastLoginAt: lastLoginAt,
		IsBanned:    player.IsBanned != 0,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// UpdateProfile handles PUT /account/profile
func (h *AccountHandlers) UpdateProfile(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var req UpdateProfileRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Username == "" || req.Email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "username and email are required",
		})
	}

	ctx := c.Context()
	err := h.accSvc.UpdatePlayerProfile(ctx, playerID, req.Username, req.Email)
	if err != nil {
		if err == account.ErrDuplicateUsername {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "username already exists",
			})
		}
		if err == account.ErrDuplicateEmail {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "email already exists",
			})
		}
		h.logger.Error("failed to update player profile", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "profile updated successfully",
	})
}

// GetSettings handles GET /account/settings
func (h *AccountHandlers) GetSettings(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	settings, err := h.accSvc.GetPlayerSettings(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player settings", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	// Convert timestamps to ISO 8601 strings
	createdAt := settings.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
	updatedAt := settings.UpdatedAt.Time.Format("2006-01-02T15:04:05Z")
	resp := SettingsResponse{
		PlayerID:         settings.PlayerID,
		KeyBindings:      settings.KeyBindings,
		MouseSensitivity: settings.MouseSensitivity,
		UiScale:          settings.UiScale,
		ColorBlindMode:   settings.ColorBlindMode,
		SubtitlesEnabled: settings.SubtitlesEnabled,
		CreatedAt:        createdAt,
		UpdatedAt:        updatedAt,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// UpdateSettings handles PUT /account/settings
func (h *AccountHandlers) UpdateSettings(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	var req UpdateSettingsRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	params := &db.UpsertPlayerSettingsParams{
		PlayerID:         playerID,
		KeyBindings:      req.KeyBindings,
		MouseSensitivity: req.MouseSensitivity,
		UiScale:          req.UiScale,
		ColorBlindMode:   req.ColorBlindMode,
		SubtitlesEnabled: req.SubtitlesEnabled,
	}
	ctx := c.Context()
	err := h.accSvc.UpsertPlayerSettings(ctx, params)
	if err != nil {
		h.logger.Error("failed to upsert player settings", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "settings updated successfully",
	})
}

// GetProgression handles GET /account/progression
func (h *AccountHandlers) GetProgression(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	progression, err := h.progressionSvc.GetPlayerProgression(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player progression", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	// Convert timestamp to ISO 8601 string
	updatedAt := progression.UpdatedAt.Time.Format("2006-01-02T15:04:05Z")
	resp := ProgressionResponse{
		PlayerID:           progression.PlayerID,
		Level:              progression.Level,
		Experience:         progression.Experience,
		PrestigeLevel:      progression.PrestigeLevel,
		DataCurrency:       progression.DataCurrency,
		TotalMatchesPlayed: progression.TotalMatchesPlayed,
		TotalWavesSurvived: progression.TotalWavesSurvived,
		TotalKills:         progression.TotalKills,
		TotalDeaths:        progression.TotalDeaths,
		TotalScrapEarned:   progression.TotalScrapEarned,
		TotalDataEarned:    progression.TotalDataEarned,
		UpdatedAt:          updatedAt,
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// PrestigePlayer handles POST /progression/prestige
func (h *AccountHandlers) PrestigePlayer(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	err := h.progressionSvc.PrestigePlayer(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to prestige player", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	// Get updated progression to include in response
	progression, err := h.progressionSvc.GetPlayerProgression(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player progression after prestige", zap.Error(err), zap.Int64("player_id", playerID))
		// Still return success because prestige succeeded
		return c.Status(fiber.StatusOK).JSON(PrestigeResponse{
			Message:          "prestige successful",
			NewPrestigeLevel: 0, // unknown
			GrantedCosmetics: []int64{},
		})
	}
	// For simplicity, we don't return granted cosmetics list (could be fetched via separate query)
	return c.Status(fiber.StatusOK).JSON(PrestigeResponse{
		Message:          "prestige successful",
		NewPrestigeLevel: progression.PrestigeLevel,
		GrantedCosmetics: []int64{},
	})
}

// GetCurrencyBalance handles GET /progression/currency
func (h *AccountHandlers) GetCurrencyBalance(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	progression, err := h.progressionSvc.GetPlayerProgression(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player progression", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"data_currency": progression.DataCurrency,
	})
}

// GetCosmeticCatalog handles GET /cosmetics/catalog
func (h *AccountHandlers) GetCosmeticCatalog(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	items, err := h.progressionSvc.GetCosmeticCatalog(ctx)
	if err != nil {
		h.logger.Error("failed to get cosmetic catalog", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(items)
}

// GetPlayerCosmetics handles GET /cosmetics/owned
func (h *AccountHandlers) GetPlayerCosmetics(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	ctx := c.Context()
	items, err := h.progressionSvc.GetPlayerCosmetics(ctx, playerID)
	if err != nil {
		h.logger.Error("failed to get player cosmetics", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(items)
}

// EquipCosmetic handles PUT /cosmetics/equip
func (h *AccountHandlers) EquipCosmetic(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}
	var req struct {
		CosmeticID int64 `json:"cosmetic_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.CosmeticID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cosmetic_id must be positive",
		})
	}
	ctx := c.Context()
	err := h.progressionSvc.EquipCosmetic(ctx, playerID, req.CosmeticID)
	if err != nil {
		if err == progression.ErrCosmeticNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "cosmetic not found",
			})
		}
		if err == progression.ErrCosmeticNotOwned {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "cosmetic not owned",
			})
		}
		if err == progression.ErrLoadoutNotFound {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "loadout not found",
			})
		}
		h.logger.Error("failed to equip cosmetic", zap.Error(err), zap.Int64("player_id", playerID), zap.Int64("cosmetic_id", req.CosmeticID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "cosmetic equipped successfully",
	})
}

// PurchaseCosmetic handles POST /cosmetics/purchase
func (h *AccountHandlers) PurchaseCosmetic(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var req struct {
		CosmeticID int64 `json:"cosmetic_id"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if req.CosmeticID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "cosmetic_id must be positive",
		})
	}

	ctx := c.Context()
	err := h.progressionSvc.PurchaseCosmetic(ctx, playerID, req.CosmeticID)
	if err != nil {
		if err == progression.ErrCosmeticNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "cosmetic not found",
			})
		}
		if err == progression.ErrInsufficientCurrency {
			return c.Status(fiber.StatusPaymentRequired).JSON(fiber.Map{
				"error": "insufficient data currency",
			})
		}
		if err == progression.ErrCosmeticAlreadyOwned {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": "cosmetic already owned",
			})
		}
		h.logger.Error("failed to purchase cosmetic", zap.Error(err), zap.Int64("player_id", playerID), zap.Int64("cosmetic_id", req.CosmeticID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "cosmetic purchased successfully",
	})
}

// StoreMatch handles POST /matches
func (h *AccountHandlers) StoreMatch(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	var req StoreMatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	// Validate required fields
	if req.ServerID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "server_id must be positive",
		})
	}
	if req.MapName == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "map_name is required",
		})
	}
	if req.GameMode == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "game_mode is required",
		})
	}
	if req.Outcome == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "outcome is required",
		})
	}
	if req.TotalPlayers <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "total_players must be positive",
		})
	}
	if len(req.PlayerStats) == 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "player_stats cannot be empty",
		})
	}

	// Build match params
	matchParams := &db.CreateMatchParams{
		ServerID:           req.ServerID,
		MapName:            req.MapName,
		GameMode:           req.GameMode,
		StartTime:          req.StartTime,
		EndTime:            types.NullTimestamp{},
		Outcome:            req.Outcome,
		WavesSurvived:      req.WavesSurvived,
		TotalZombiesKilled: req.TotalZombiesKilled,
		TotalPlayers:       req.TotalPlayers,
	}
	if req.EndTime != nil {
		matchParams.EndTime = *req.EndTime
	}

	// Convert player stats
	playerStats := make([]*db.CreatePlayerMatchStatsParams, 0, len(req.PlayerStats))
	for _, ps := range req.PlayerStats {
		if ps.PlayerID <= 0 {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "player_stats.player_id must be positive",
			})
		}
		playerStats = append(playerStats, &db.CreatePlayerMatchStatsParams{
			PlayerID:           ps.PlayerID,
			MatchID:            0, // will be set by service
			WavesSurvived:      ps.WavesSurvived,
			ZombiesKilled:      ps.ZombiesKilled,
			Deaths:             ps.Deaths,
			ScrapEarned:        ps.ScrapEarned,
			DataEarned:         ps.DataEarned,
			DamageDealt:        ps.DamageDealt,
			DamageTaken:        ps.DamageTaken,
			BuildingsBuilt:     ps.BuildingsBuilt,
			BuildingsDestroyed: ps.BuildingsDestroyed,
			HealingGiven:       ps.HealingGiven,
			Revives:            ps.Revives,
			Score:              ps.Score,
		})
	}

	ctx := c.Context()
	err := h.matchSvc.StoreMatchWithStats(ctx, req.ServerID, matchParams, playerStats)
	if err != nil {
		if err == server.ErrServerNotFound {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "server not found",
			})
		}
		h.logger.Error("failed to store match", zap.Error(err), zap.Int64("player_id", playerID), zap.Int64("server_id", req.ServerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "match stored successfully",
	})
}

// GetMatchHistory handles GET /matches/history
func (h *AccountHandlers) GetMatchHistory(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID missing from context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "unauthorized",
		})
	}

	// Parse limit query parameter (default 10, max 100)
	limit := c.QueryInt("limit", 10)
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	ctx := c.Context()
	matches, err := h.matchSvc.GetPlayerMatchHistory(ctx, playerID, int32(limit))
	if err != nil {
		h.logger.Error("failed to get match history", zap.Error(err), zap.Int64("player_id", playerID))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(matches)
}
