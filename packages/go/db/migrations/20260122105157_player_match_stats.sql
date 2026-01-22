-- player_match_stats table
CREATE TABLE player_match_stats (
    player_id INTEGER NOT NULL,
    match_id INTEGER NOT NULL,
    waves_survived INTEGER NOT NULL DEFAULT 0,
    zombies_killed INTEGER NOT NULL DEFAULT 0,
    deaths INTEGER NOT NULL DEFAULT 0,
    scrap_earned INTEGER NOT NULL DEFAULT 0,
    data_earned INTEGER NOT NULL DEFAULT 0,
    damage_dealt INTEGER NOT NULL DEFAULT 0,
    damage_taken INTEGER NOT NULL DEFAULT 0,
    buildings_built INTEGER NOT NULL DEFAULT 0,
    buildings_destroyed INTEGER NOT NULL DEFAULT 0,
    healing_given INTEGER NOT NULL DEFAULT 0,
    revives INTEGER NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (player_id, match_id),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
    FOREIGN KEY (match_id) REFERENCES matches (match_id) ON DELETE CASCADE
);

CREATE INDEX idx_player_match_stats_match_id ON player_match_stats (match_id);