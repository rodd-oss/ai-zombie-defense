package social

import (
	"ai-zombie-defense/backend-api/internal/db"
	"ai-zombie-defense/backend-api/pkg/config"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"
)

type socialService struct {
	config  config.Config
	logger  *zap.Logger
	dbConn  db.DBTX
	queries *db.Queries
}

func NewSocialService(cfg config.Config, logger *zap.Logger, dbConn db.DBTX) Service {
	return &socialService{
		config:  cfg,
		logger:  logger,
		dbConn:  dbConn,
		queries: db.New(),
	}
}

func (s *socialService) SendFriendRequest(ctx context.Context, playerID int64, friendID int64) error {
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

func (s *socialService) AcceptFriendRequest(ctx context.Context, requesterPlayerID int64, friendID int64) error {
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

func (s *socialService) DeclineFriendRequest(ctx context.Context, requesterPlayerID int64, friendID int64) error {
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

func (s *socialService) ListFriends(ctx context.Context, playerID int64) ([]*db.ListFriendsRow, error) {
	friends, err := s.queries.ListFriends(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list friends: %w", err)
	}
	return friends, nil
}

func (s *socialService) ListPendingIncoming(ctx context.Context, playerID int64) ([]*db.ListPendingIncomingRow, error) {
	requests, err := s.queries.ListPendingIncoming(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending incoming requests: %w", err)
	}
	return requests, nil
}

func (s *socialService) ListPendingOutgoing(ctx context.Context, playerID int64) ([]*db.ListPendingOutgoingRow, error) {
	requests, err := s.queries.ListPendingOutgoing(ctx, s.dbConn, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending outgoing requests: %w", err)
	}
	return requests, nil
}
