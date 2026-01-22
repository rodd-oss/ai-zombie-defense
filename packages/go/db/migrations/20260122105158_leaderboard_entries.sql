-- leaderboard_entries table
CREATE TABLE leaderboard_entries (
    leaderboard_id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    period TEXT NOT NULL CHECK (period IN ('daily', 'weekly', 'monthly', 'all_time')),
    rank INTEGER NOT NULL,
    score INTEGER NOT NULL,
    matches_played INTEGER NOT NULL DEFAULT 0,
    avg_kills_per_match REAL NOT NULL DEFAULT 0,
    avg_waves_survived REAL NOT NULL DEFAULT 0,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
);

CREATE INDEX idx_leaderboard_entries_player_id ON leaderboard_entries (player_id);
CREATE INDEX idx_leaderboard_entries_period ON leaderboard_entries (period);