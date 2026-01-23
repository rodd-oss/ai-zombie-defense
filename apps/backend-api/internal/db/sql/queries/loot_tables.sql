-- name: CreateLootTable :one
INSERT INTO loot_tables (name, description, drop_chance, is_active)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: GetLootTable :one
SELECT * FROM loot_tables
WHERE loot_table_id = ?;

-- name: ListLootTables :many
SELECT * FROM loot_tables
ORDER BY loot_table_id;

-- name: ListActiveLootTables :many
SELECT * FROM loot_tables
WHERE is_active = 1
ORDER BY loot_table_id;

-- name: UpdateLootTable :exec
UPDATE loot_tables
SET name = ?, description = ?, drop_chance = ?, is_active = ?
WHERE loot_table_id = ?;

-- name: DeleteLootTable :exec
DELETE FROM loot_tables
WHERE loot_table_id = ?;