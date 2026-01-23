-- name: CreateJoinToken :one
INSERT INTO join_tokens (
    token,
    player_id,
    server_id,
    expires_at
) VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetJoinToken :one
SELECT * FROM join_tokens WHERE token = ?;

-- name: GetValidJoinToken :one
SELECT * FROM join_tokens
WHERE token = ?
  AND expires_at > strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
  AND used_at IS NULL;

-- name: MarkTokenUsed :exec
UPDATE join_tokens
SET used_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE token = ?;

-- name: DeleteExpiredTokens :exec
DELETE FROM join_tokens
WHERE expires_at <= strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
   OR used_at IS NOT NULL;