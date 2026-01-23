-- name: CreateFriendRequest :exec
INSERT INTO friends (player_id, friend_id, status) VALUES (?1, ?2, 'pending');

-- name: AcceptFriendRequest :exec
UPDATE friends 
SET status = 'accepted', updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE player_id = ?1 AND friend_id = ?2 AND status = 'pending';

-- name: DeclineFriendRequest :exec
DELETE FROM friends 
WHERE player_id = ?1 AND friend_id = ?2 AND status = 'pending';

-- name: GetFriendRequest :one
SELECT * FROM friends WHERE player_id = ?1 AND friend_id = ?2;

-- name: ListFriends :many
SELECT 
  CAST(CASE 
    WHEN f.player_id = ?1 THEN f.friend_id 
    ELSE f.player_id 
  END AS INTEGER) AS friend_player_id,
  p.username AS friend_username,
  f.status,
  f.created_at,
  f.updated_at
FROM friends f
JOIN players p ON p.player_id = CAST(CASE 
    WHEN f.player_id = ?1 THEN f.friend_id 
    ELSE f.player_id 
  END AS INTEGER)
WHERE (f.player_id = ?1 OR f.friend_id = ?1) AND f.status = 'accepted';

-- name: ListPendingIncoming :many
SELECT f.player_id AS requester_player_id, p.username AS requester_username, f.created_at
FROM friends f
JOIN players p ON f.player_id = p.player_id
WHERE f.friend_id = ?1 AND f.status = 'pending';

-- name: ListPendingOutgoing :many
SELECT f.friend_id AS target_player_id, p.username AS target_username, f.created_at
FROM friends f
JOIN players p ON f.friend_id = p.player_id
WHERE f.player_id = ?1 AND f.status = 'pending';