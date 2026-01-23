-- name: GetPlayerSettings :one
SELECT * FROM player_settings WHERE player_id = ?;

-- name: UpsertPlayerSettings :exec
INSERT INTO player_settings (player_id, key_bindings, mouse_sensitivity, ui_scale, color_blind_mode, subtitles_enabled)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(player_id) DO UPDATE SET
    key_bindings = excluded.key_bindings,
    mouse_sensitivity = excluded.mouse_sensitivity,
    ui_scale = excluded.ui_scale,
    color_blind_mode = excluded.color_blind_mode,
    subtitles_enabled = excluded.subtitles_enabled,
    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now');