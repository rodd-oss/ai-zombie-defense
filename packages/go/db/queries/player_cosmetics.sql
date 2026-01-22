-- name: GetPlayerCosmetics :many
SELECT ci.*, pc.unlocked_at, pc.unlocked_via
FROM cosmetic_items ci
JOIN player_cosmetics pc ON ci.cosmetic_id = pc.cosmetic_id
WHERE pc.player_id = ?
ORDER BY pc.unlocked_at DESC;