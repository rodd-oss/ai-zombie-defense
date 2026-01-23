package handlers

import (
	"ai-zombie-defense/backend-api/internal/middleware"
	"ai-zombie-defense/backend-api/internal/services/server"
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type FavoriteHandlers struct {
	service server.Service
	logger  *zap.Logger
}

func NewFavoriteHandlers(service server.Service, logger *zap.Logger) *FavoriteHandlers {
	return &FavoriteHandlers{
		service: service,
		logger:  logger,
	}
}

type AddFavoriteRequest struct {
	ServerID int64   `json:"server_id"`
	Note     *string `json:"note,omitempty"`
}

type FavoriteResponse struct {
	ServerID int64       `json:"server_id"`
	AddedAt  string      `json:"added_at"`
	Note     *string     `json:"note,omitempty"`
	Server   *ServerInfo `json:"server"`
}

type ServerInfo struct {
	ServerID       int64   `json:"server_id"`
	IPAddress      string  `json:"ip_address"`
	Port           int64   `json:"port"`
	Name           string  `json:"name"`
	MapRotation    *string `json:"map_rotation,omitempty"`
	MaxPlayers     int64   `json:"max_players"`
	CurrentPlayers int64   `json:"current_players"`
	IsOnline       bool    `json:"is_online"`
	LastHeartbeat  *string `json:"last_heartbeat,omitempty"`
	Region         *string `json:"region,omitempty"`
	Version        *string `json:"version,omitempty"`
	CreatedAt      string  `json:"created_at"`
}

// AddFavorite handles POST /favorites
func (h *FavoriteHandlers) AddFavorite(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID not found in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req AddFavoriteRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.ServerID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	err := h.service.AddFavorite(c.Context(), playerID, req.ServerID, req.Note)
	if err != nil {
		if errors.Is(err, server.ErrFavoriteAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		h.logger.Error("Failed to add favorite", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to add favorite",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "added",
	})
}

// RemoveFavorite handles DELETE /favorites/:id
func (h *FavoriteHandlers) RemoveFavorite(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID not found in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	serverID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid server ID",
		})
	}

	err = h.service.RemoveFavorite(c.Context(), playerID, int64(serverID))
	if err != nil {
		h.logger.Error("Failed to remove favorite", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to remove favorite",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "removed",
	})
}

// ListFavorites handles GET /favorites
func (h *FavoriteHandlers) ListFavorites(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID not found in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	favorites, err := h.service.ListPlayerFavorites(c.Context(), playerID)
	if err != nil {
		h.logger.Error("Failed to list favorites", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve favorites",
		})
	}

	// Transform to response format
	response := make([]FavoriteResponse, 0, len(favorites))
	for _, fav := range favorites {
		serverInfo := ServerInfo{
			ServerID:       fav.ServerID,
			IPAddress:      fav.IpAddress,
			Port:           fav.Port,
			Name:           fav.Name,
			MapRotation:    fav.MapRotation,
			MaxPlayers:     fav.MaxPlayers,
			CurrentPlayers: fav.CurrentPlayers,
			IsOnline:       fav.IsOnline == 1,
			LastHeartbeat:  fav.LastHeartbeat,
			Region:         fav.Region,
			Version:        fav.Version,
			CreatedAt:      fav.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
		}
		resp := FavoriteResponse{
			ServerID: fav.ServerID,
			AddedAt:  fav.AddedAt.Time.Format("2006-01-02T15:04:05Z"),
			Note:     fav.Note,
			Server:   &serverInfo,
		}
		response = append(response, resp)
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
