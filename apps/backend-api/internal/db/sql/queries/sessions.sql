-- name: CreateSession :exec
INSERT INTO sessions (player_id, token, expires_at, ip_address, user_agent)
VALUES (?, ?, ?, ?, ?);

-- name: GetSessionByToken :one
SELECT * FROM sessions WHERE token = ?;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at < ?;

-- name: DeleteSessionsByPlayer :exec
DELETE FROM sessions WHERE player_id = ?;