package handlers

import (
	"ai-zombie-defense/backend-api/internal/db"
	"ai-zombie-defense/backend-api/internal/db/types"
	"ai-zombie-defense/backend-api/internal/middleware"
	"ai-zombie-defense/backend-api/internal/services/match"
	"ai-zombie-defense/backend-api/internal/services/server"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type MatchHandlers struct {
	matchSvc match.Service
	logger   *zap.Logger
}

func NewMatchHandlers(matchSvc match.Service, logger *zap.Logger) *MatchHandlers {
	return &MatchHandlers{
		matchSvc: matchSvc,
		logger:   logger,
	}
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

// StoreMatch handles POST /matches
func (h *MatchHandlers) StoreMatch(c *fiber.Ctx) error {
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
func (h *MatchHandlers) GetMatchHistory(c *fiber.Ctx) error {
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
