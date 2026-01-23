package handlers

import (
	"ai-zombie-defense/backend-api/internal/middleware"
	"ai-zombie-defense/backend-api/internal/services/social"
	"errors"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"
)

type FriendHandlers struct {
	service social.Service
	logger  *zap.Logger
}

func NewFriendHandlers(service social.Service, logger *zap.Logger) *FriendHandlers {
	return &FriendHandlers{
		service: service,
		logger:  logger,
	}
}

type SendFriendRequestRequest struct {
	FriendID int64 `json:"friend_id"`
}

type UpdateFriendRequestRequest struct {
	Action string `json:"action"` // "accept" or "decline"
}

type FriendResponse struct {
	FriendPlayerID int64  `json:"friend_player_id"`
	FriendUsername string `json:"friend_username"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

// SendFriendRequest handles POST /friends/request
func (h *FriendHandlers) SendFriendRequest(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID not found in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	var req SendFriendRequestRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	if req.FriendID <= 0 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid friend ID",
		})
	}

	err := h.service.SendFriendRequest(c.Context(), playerID, req.FriendID)
	if err != nil {
		if errors.Is(err, social.ErrCannotFriendSelf) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, social.ErrFriendRequestAlreadyExists) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		h.logger.Error("Failed to send friend request", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to send friend request",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status": "request_sent",
	})
}

// UpdateFriendRequest handles PUT /friends/:id
func (h *FriendHandlers) UpdateFriendRequest(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID not found in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	requesterID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid player ID",
		})
	}

	var req UpdateFriendRequestRequest
	if err := c.BodyParser(&req); err != nil {
		h.logger.Error("Failed to parse request body", zap.Error(err))
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	switch req.Action {
	case "accept":
		err = h.service.AcceptFriendRequest(c.Context(), int64(requesterID), playerID)
	case "decline":
		err = h.service.DeclineFriendRequest(c.Context(), int64(requesterID), playerID)
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid action, must be 'accept' or 'decline'",
		})
	}

	if err != nil {
		if errors.Is(err, social.ErrFriendRequestNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		if errors.Is(err, social.ErrFriendRequestNotPending) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		h.logger.Error("Failed to update friend request", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update friend request",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status": req.Action + "ed",
	})
}

// ListFriends handles GET /friends
func (h *FriendHandlers) ListFriends(c *fiber.Ctx) error {
	playerID, ok := middleware.GetPlayerID(c)
	if !ok {
		h.logger.Error("player ID not found in context")
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}

	friends, err := h.service.ListFriends(c.Context(), playerID)
	if err != nil {
		h.logger.Error("Failed to list friends", zap.Error(err))
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to retrieve friends",
		})
	}

	response := make([]FriendResponse, 0, len(friends))
	for _, f := range friends {
		response = append(response, FriendResponse{
			FriendPlayerID: f.FriendPlayerID,
			FriendUsername: f.FriendUsername,
			Status:         f.Status,
			CreatedAt:      f.CreatedAt.Time.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:      f.UpdatedAt.Time.Format("2006-01-02T15:04:05Z"),
		})
	}

	return c.Status(fiber.StatusOK).JSON(response)
}
