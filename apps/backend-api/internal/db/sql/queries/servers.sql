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

-- name: ListActiveServers :many
SELECT * FROM servers
WHERE is_online = 1
  AND (region = ?1 OR ?1 IS NULL)
  AND (map_rotation = ?2 OR ?2 IS NULL)
  AND (version = ?3 OR ?3 IS NULL)
  AND (current_players >= ?4 OR ?4 = -1)
  AND (current_players <= ?5 OR ?5 = -1)
ORDER BY server_id;