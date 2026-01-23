-- name: GetWeeklyLeaderboard :many
SELECT
  p.player_id,
  p.username,
  CAST(SUM(pms.score) AS INTEGER) AS total_score,
  COUNT(pms.match_id) AS matches_played,
  AVG(pms.zombies_killed) AS avg_kills_per_match,
  AVG(pms.waves_survived) AS avg_waves_survived,
  CAST(ROW_NUMBER() OVER (ORDER BY SUM(pms.score) DESC) AS INTEGER) AS ranking
FROM player_match_stats pms
JOIN matches m ON pms.match_id = m.match_id
JOIN players p ON pms.player_id = p.player_id
WHERE date(m.start_time) >= date('now', '-7 days')
GROUP BY pms.player_id
ORDER BY total_score DESC