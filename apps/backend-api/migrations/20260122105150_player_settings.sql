-- +goose Up
CREATE TABLE player_settings (
    player_id INTEGER PRIMARY KEY,
    key_bindings TEXT,
    mouse_sensitivity REAL,
    ui_scale REAL,
    color_blind_mode INTEGER NOT NULL DEFAULT 0,
    subtitles_enabled INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE player_settings;