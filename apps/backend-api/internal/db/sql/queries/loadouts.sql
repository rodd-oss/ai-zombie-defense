-- name: GetPlayerLoadouts :many
SELECT * FROM loadouts WHERE player_id = ? ORDER BY loadout_id;

-- name: GetActiveLoadout :one
SELECT * FROM loadouts WHERE player_id = ? AND is_active = 1;

-- name: CreateLoadout :exec
INSERT INTO loadouts (player_id, name, is_active) VALUES (?, ?, ?);

-- name: UpdateLoadoutActive :exec
UPDATE loadouts SET is_active = ? WHERE loadout_id = ? AND player_id = ?;

-- name: GetLoadoutCosmetics :many
SELECT lc.*, ci.slot AS cosmetic_slot FROM loadout_cosmetics lc
JOIN cosmetic_items ci ON lc.cosmetic_id = ci.cosmetic_id
WHERE lc.loadout_id = ?;

-- name: GetLoadoutCosmeticBySlot :one
SELECT lc.* FROM loadout_cosmetics lc
WHERE lc.loadout_id = ? AND lc.slot = ?;

-- name: DeleteLoadoutCosmeticBySlot :exec
DELETE FROM loadout_cosmetics WHERE loadout_id = ? AND slot = ?;

-- name: InsertLoadoutCosmetic :exec
INSERT INTO loadout_cosmetics (loadout_id, cosmetic_id, slot) VALUES (?, ?, ?);

-- name: GetCosmeticItem :one
SELECT * FROM cosmetic_items WHERE cosmetic_id = ?;