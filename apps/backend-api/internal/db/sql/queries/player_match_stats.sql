-- name: CreatePlayerMatchStats :one
INSERT INTO player_match_stats (
    player_id,
    match_id,
    waves_survived,
    zombies_killed,
    deaths,
    scrap_earned,
    data_earned,
    damage_dealt,
    damage_taken,
    buildings_built,
    buildings_destroyed,
    healing_given,
    revives,
    score
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetPlayerMatchStats :one
SELECT * FROM player_match_stats 
WHERE player_id = ? AND match_id = ?;

-- name: GetMatchPlayerStats :many
SELECT * FROM player_match_stats 
WHERE match_id = ?;