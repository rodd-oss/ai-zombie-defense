-- name: GetCosmeticCatalog :many
SELECT * FROM cosmetic_items
ORDER BY cosmetic_id;

-- name: GetPrestigeCosmetics :many
SELECT ci.* FROM cosmetic_items ci
LEFT JOIN player_cosmetics pc ON ci.cosmetic_id = pc.cosmetic_id AND pc.player_id = ?1
WHERE ci.is_prestige_only = 1
    AND ci.unlock_level <= ?2
    AND pc.cosmetic_id IS NULL;

-- name: GrantCosmeticToPlayer :exec
INSERT INTO player_cosmetics (player_id, cosmetic_id, unlocked_via)
VALUES (?, ?, ?);