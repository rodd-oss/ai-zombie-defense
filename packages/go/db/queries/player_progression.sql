-- name: GetPlayerProgression :one
SELECT * FROM player_progression WHERE player_id = ?;

-- name: GetDataCurrency :one
SELECT data_currency FROM player_progression WHERE player_id = ?;

-- name: CreatePlayerProgression :exec
INSERT INTO player_progression (player_id) VALUES (?);

-- name: UpdatePlayerProgression :exec
UPDATE player_progression
SET level = ?,
    experience = ?,
    prestige_level = ?,
    data_currency = ?,
    total_matches_played = ?,
    total_waves_survived = ?,
    total_kills = ?,
    total_deaths = ?,
    total_scrap_earned = ?,
    total_data_earned = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE player_id = ?;

-- name: IncrementExperience :exec
UPDATE player_progression
SET experience = experience + ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE player_id = ?;

-- name: AddDataCurrency :exec
UPDATE player_progression
SET data_currency = data_currency + ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE player_id = ?;

-- name: SetDataCurrency :exec
UPDATE player_progression
SET data_currency = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE player_id = ?;

-- name: UpdateLevel :exec
UPDATE player_progression
SET level = ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE player_id = ?;

-- name: IncrementMatchStats :exec
UPDATE player_progression
SET total_matches_played = total_matches_played + ?,
    total_waves_survived = total_waves_survived + ?,
    total_kills = total_kills + ?,
    total_deaths = total_deaths + ?,
    total_scrap_earned = total_scrap_earned + ?,
    total_data_earned = total_data_earned + ?,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE player_id = ?;

-- name: PrestigePlayer :exec
UPDATE player_progression
SET level = 1,
    experience = 0,
    prestige_level = prestige_level + 1,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE player_id = ?;