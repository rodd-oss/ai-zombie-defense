-- player_cosmetics table
CREATE TABLE player_cosmetics (
    player_id INTEGER NOT NULL,
    cosmetic_id INTEGER NOT NULL,
    unlocked_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    unlocked_via TEXT NOT NULL CHECK (unlocked_via IN ('level_up', 'purchase', 'loot_drop', 'prestige')),
    PRIMARY KEY (player_id, cosmetic_id),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
    FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
);

CREATE INDEX idx_player_cosmetics_cosmetic_id ON player_cosmetics (cosmetic_id);