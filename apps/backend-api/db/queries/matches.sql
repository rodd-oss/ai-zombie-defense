-- name: CreateMatch :one
INSERT INTO matches (
    server_id,
    map_name,
    game_mode,
    start_time,
    end_time,
    outcome,
    waves_survived,
    total_zombies_killed,
    total_players
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: GetMatch :one
SELECT * FROM matches WHERE match_id = ?;

-- name: GetPlayerMatchHistory :many
SELECT 
    m.*,
    pms.waves_survived as player_waves_survived,
    pms.zombies_killed as player_zombies_killed,
    pms.deaths as player_deaths,
    pms.scrap_earned as player_scrap_earned,
    pms.data_earned as player_data_earned,
    pms.damage_dealt as player_damage_dealt,
    pms.damage_taken as player_damage_taken,
    pms.buildings_built as player_buildings_built,
    pms.buildings_destroyed as player_buildings_destroyed,
    pms.healing_given as player_healing_given,
    pms.revives as player_revives,
    pms.score as player_score
FROM matches m
JOIN player_match_stats pms ON m.match_id = pms.match_id
WHERE pms.player_id = ?
ORDER BY m.start_time DESC
LIMIT ?;

-- name: UpdateMatchOutcome :exec
UPDATE matches
SET outcome = ?, end_time = ?
WHERE match_id = ?;