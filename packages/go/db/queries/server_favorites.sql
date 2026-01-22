-- name: AddFavorite :exec
INSERT INTO server_favorites (player_id, server_id, note)
VALUES (?, ?, ?);

-- name: RemoveFavorite :exec
DELETE FROM server_favorites
WHERE player_id = ? AND server_id = ?;

-- name: GetFavorite :one
SELECT * FROM server_favorites
WHERE player_id = ? AND server_id = ?;

-- name: ListPlayerFavorites :many
SELECT s.*, sf.added_at, sf.note
FROM servers s
JOIN server_favorites sf ON s.server_id = sf.server_id
WHERE sf.player_id = ?
ORDER BY sf.added_at DESC;