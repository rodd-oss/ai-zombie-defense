-- name: CreateServer :one
INSERT INTO servers (
    ip_address,
    port,
    auth_token,
    name,
    map_rotation,
    max_players,
    region,
    version
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetServer :one
SELECT * FROM servers WHERE server_id = ?;

-- name: GetServerByAuthToken :one
SELECT * FROM servers WHERE auth_token = ?;

-- name: ListServers :many
SELECT * FROM servers ORDER BY server_id;

-- name: UpdateServerHeartbeat :exec
UPDATE servers
SET last_heartbeat = ?, current_players = ?, is_online = 1, map_rotation = ?
WHERE server_id = ?;

-- name: MarkServerOffline :exec
UPDATE servers
SET is_online = 0
WHERE server_id = ?;

-- name: DeleteServer :exec
DELETE FROM servers WHERE server_id = ?;