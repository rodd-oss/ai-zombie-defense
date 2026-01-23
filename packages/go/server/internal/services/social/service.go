package social

import (
	"ai-zombie-defense/db"
	"context"
	"errors"
)

var (
	ErrFriendRequestAlreadyExists = errors.New("friend request already exists")
	ErrFriendRequestNotFound      = errors.New("friend request not found")
	ErrFriendRequestNotPending    = errors.New("friend request not pending")
	ErrCannotFriendSelf           = errors.New("cannot send friend request to yourself")
)

type Service interface {
	SendFriendRequest(ctx context.Context, playerID int64, friendID int64) error
	AcceptFriendRequest(ctx context.Context, requesterPlayerID int64, friendID int64) error
	DeclineFriendRequest(ctx context.Context, requesterPlayerID int64, friendID int64) error
	ListFriends(ctx context.Context, playerID int64) ([]*db.ListFriendsRow, error)
	ListPendingIncoming(ctx context.Context, playerID int64) ([]*db.ListPendingIncomingRow, error)
	ListPendingOutgoing(ctx context.Context, playerID int64) ([]*db.ListPendingOutgoingRow, error)
}
