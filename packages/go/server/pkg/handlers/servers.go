package handlers

import (
	"ai-zombie-defense/server/pkg/auth"
	"ai-zombie-defense/server/pkg/middleware"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type ServerHandlers struct {
	service *auth.Service
	logger  *zap.Logger
}

func NewServerHandlers(service *auth.Service, logger *zap.Logger) *ServerHandlers {
	return &ServerHandlers{
		service: service,
		logger:  logger,
	}
}

type RegisterServerRequest struct {
	IPAddress   string  `json:"ip_address"`
	Port        int64   `json:"port"`
	Name        string  `json:"name"`
	MapRotation *string `json:"map_rotation,omitempty"`
	MaxPlayers  int64   `json:"max_players"`
	Region      *string `json:"region,omitempty"`
	Version     *string `json:"version,omitempty"`
}

type RegisterServerResponse struct {
	ServerID    int64   `json:"server_id"`
	AuthToken   string  `json:"auth_token"`
	IPAddress   string  `json:"ip_address"`
	Port        int64   `json:"port"`
	Name        string  `json:"name"`
	MapRotation *string `json:"map_rotation,omitempty"`
	MaxPlayers  int64   `json:"max_players"`
	Region      *string `json:"region,omitempty"`
	Version     *string `json:"version,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// RegisterServer handles POST /servers/register
func (h *ServerHandlers) RegisterServer(c *fiber.Ctx) error {
	var req RegisterServerRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate required fields
	if req.IPAddress == "" || req.Port <= 0 || req.Name == "" || req.MaxPlayers <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Missing or invalid required fields (ip_address, port, name, max_players)",
		})
	}

	// Register server via auth service
	server, authToken, err := h.service.RegisterServer(c.Context(), req.IPAddress, req.Port, req.Name, req.MapRotation, req.MaxPlayers, req.Region, req.Version)
	if err != nil {
		h.logger.Error("Failed to register server", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to register server",
		})
	}

	// Build response
	createdAt := server.CreatedAt.Time.Format("2006-01-02T15:04:05Z")
	resp := RegisterServerResponse{
		ServerID:    server.ServerID,
		AuthToken:   authToken,
		IPAddress:   server.IpAddress,
		Port:        server.Port,
		Name:        server.Name,
		MapRotation: server.MapRotation,
		MaxPlayers:  server.MaxPlayers,
		Region:      server.Region,
		Version:     server.Version,
		CreatedAt:   createdAt,
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// UpdateHeartbeatRequest defines the request body for updating server heartbeat.
type UpdateHeartbeatRequest struct {
	CurrentPlayers int64   `json:"current_players"`
	Map            *string `json:"map,omitempty"`
}

// UpdateHeartbeat handles PUT /servers/:id/heartbeat
func (h *ServerHandlers) UpdateHeartbeat(c *fiber.Ctx) error {
	serverID, ok := middleware.GetServerID(c)
	if !ok {
		h.logger.Error("server ID not found in context")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "internal server error",
		})
	}

	var req UpdateHeartbeatRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	// Validate current players not negative
	if req.CurrentPlayers < 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "current_players cannot be negative",
		})
	}

	err := h.service.UpdateServerHeartbeat(c.Context(), serverID, req.CurrentPlayers, req.Map)
	if err != nil {
		h.logger.Error("Failed to update server heartbeat", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update heartbeat",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": "ok",
	})
}

// ListServers handles GET /servers
func (h *ServerHandlers) ListServers(c *fiber.Ctx) error {
	// Parse query parameters
	region := c.Query("region")
	mapRotation := c.Query("map")
	version := c.Query("version")
	minPlayersStr := c.Query("min_players")
	maxPlayersStr := c.Query("max_players")

	// Convert strings to pointers
	var regionPtr, mapPtr, versionPtr *string
	if region != "" {
		regionPtr = &region
	}
	if mapRotation != "" {
		mapPtr = &mapRotation
	}
	if version != "" {
		versionPtr = &version
	}

	var minPlayersPtr, maxPlayersPtr *int64
	if minPlayersStr != "" {
		val, err := strconv.ParseInt(minPlayersStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid min_players parameter",
			})
		}
		minPlayersPtr = &val
	}
	if maxPlayersStr != "" {
		val, err := strconv.ParseInt(maxPlayersStr, 10, 64)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid max_players parameter",
			})
		}
		maxPlayersPtr = &val
	}

	servers, err := h.service.ListActiveServers(c.Context(), regionPtr, mapPtr, versionPtr, minPlayersPtr, maxPlayersPtr)
	if err != nil {
		h.logger.Error("Failed to list servers", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve servers",
		})
	}

	return c.Status(fiber.StatusOK).JSON(servers)
}
