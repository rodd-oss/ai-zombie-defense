-- name: CreateLootTableEntry :one
INSERT INTO loot_table_entries (loot_table_id, cosmetic_id, weight, min_quantity, max_quantity)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: GetLootTableEntry :one
SELECT * FROM loot_table_entries
WHERE loot_entry_id = ?;

-- name: GetLootTableEntriesByLootTableID :many
SELECT * FROM loot_table_entries
WHERE loot_table_id = ?
ORDER BY loot_entry_id;

-- name: GetLootTableEntriesWithCosmeticDetails :many
SELECT lte.*, ci.name AS cosmetic_name, ci.rarity AS cosmetic_rarity, ci.slot AS cosmetic_slot
FROM loot_table_entries lte
JOIN cosmetic_items ci ON lte.cosmetic_id = ci.cosmetic_id
WHERE lte.loot_table_id = ?
ORDER BY lte.loot_entry_id;

-- name: UpdateLootTableEntry :exec
UPDATE loot_table_entries
SET loot_table_id = ?, cosmetic_id = ?, weight = ?, min_quantity = ?, max_quantity = ?
WHERE loot_entry_id = ?;

-- name: DeleteLootTableEntry :exec
DELETE FROM loot_table_entries
WHERE loot_entry_id = ?;