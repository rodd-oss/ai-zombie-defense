-- matches table
CREATE TABLE matches (
    match_id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER NOT NULL,
    map_name TEXT NOT NULL,
    game_mode TEXT NOT NULL,
    start_time TEXT NOT NULL,
    end_time TEXT,
    outcome TEXT NOT NULL CHECK (outcome IN ('completed', 'failed', 'abandoned')),
    waves_survived INTEGER NOT NULL DEFAULT 0,
    total_zombies_killed INTEGER NOT NULL DEFAULT 0,
    total_players INTEGER NOT NULL DEFAULT 0,
    FOREIGN KEY (server_id) REFERENCES servers (server_id) ON DELETE CASCADE
);

CREATE INDEX idx_matches_server_id ON matches (server_id);