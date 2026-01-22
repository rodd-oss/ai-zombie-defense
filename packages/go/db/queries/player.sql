-- name: GetPlayer :one
SELECT * FROM players WHERE player_id = ?;

-- name: ListPlayers :many
SELECT * FROM players ORDER BY username;

-- name: CreatePlayer :exec
INSERT INTO players (username, email, password_hash) VALUES (?, ?, ?);

-- name: UpdatePlayerLastLogin :exec
UPDATE players SET last_login_at = ? WHERE player_id = ?;

-- name: DeletePlayer :exec
DELETE FROM players WHERE player_id = ?;