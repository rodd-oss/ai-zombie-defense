package handlers

import (
	"ai-zombie-defense/server/pkg/auth"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type LeaderboardHandlers struct {
	service *auth.Service
	logger  *zap.Logger
}

func NewLeaderboardHandlers(service *auth.Service, logger *zap.Logger) *LeaderboardHandlers {
	return &LeaderboardHandlers{
		service: service,
		logger:  logger,
	}
}

type LeaderboardEntryResponse struct {
	PlayerID         int64    `json:"player_id"`
	Username         string   `json:"username"`
	TotalScore       int64    `json:"total_score"`
	MatchesPlayed    int64    `json:"matches_played"`
	AvgKillsPerMatch *float64 `json:"avg_kills_per_match,omitempty"`
	AvgWavesSurvived *float64 `json:"avg_waves_survived,omitempty"`
	Ranking          int64    `json:"ranking"`
}

// GetDailyLeaderboard handles GET /leaderboards/daily
func (h *LeaderboardHandlers) GetDailyLeaderboard(c *fiber.Ctx) error {
	entries, err := h.service.GetDailyLeaderboard(c.Context())
	if err != nil {
		h.logger.Error("Failed to get daily leaderboard", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve daily leaderboard",
		})
	}

	response := make([]LeaderboardEntryResponse, 0, len(entries))
	for _, e := range entries {
		response = append(response, LeaderboardEntryResponse{
			PlayerID:         e.PlayerID,
			Username:         e.Username,
			TotalScore:       e.TotalScore,
			MatchesPlayed:    e.MatchesPlayed,
			AvgKillsPerMatch: e.AvgKillsPerMatch,
			AvgWavesSurvived: e.AvgWavesSurvived,
			Ranking:          e.Ranking,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetWeeklyLeaderboard handles GET /leaderboards/weekly
func (h *LeaderboardHandlers) GetWeeklyLeaderboard(c *fiber.Ctx) error {
	entries, err := h.service.GetWeeklyLeaderboard(c.Context())
	if err != nil {
		h.logger.Error("Failed to get weekly leaderboard", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve weekly leaderboard",
		})
	}

	response := make([]LeaderboardEntryResponse, 0, len(entries))
	for _, e := range entries {
		response = append(response, LeaderboardEntryResponse{
			PlayerID:         e.PlayerID,
			Username:         e.Username,
			TotalScore:       e.TotalScore,
			MatchesPlayed:    e.MatchesPlayed,
			AvgKillsPerMatch: e.AvgKillsPerMatch,
			AvgWavesSurvived: e.AvgWavesSurvived,
			Ranking:          e.Ranking,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}

// GetAllTimeLeaderboard handles GET /leaderboards/alltime
func (h *LeaderboardHandlers) GetAllTimeLeaderboard(c *fiber.Ctx) error {
	entries, err := h.service.GetAllTimeLeaderboard(c.Context())
	if err != nil {
		h.logger.Error("Failed to get all-time leaderboard", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve all-time leaderboard",
		})
	}

	response := make([]LeaderboardEntryResponse, 0, len(entries))
	for _, e := range entries {
		response = append(response, LeaderboardEntryResponse{
			PlayerID:         e.PlayerID,
			Username:         e.Username,
			TotalScore:       e.TotalScore,
			MatchesPlayed:    e.MatchesPlayed,
			AvgKillsPerMatch: e.AvgKillsPerMatch,
			AvgWavesSurvived: e.AvgWavesSurvived,
			Ranking:          e.Ranking,
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
