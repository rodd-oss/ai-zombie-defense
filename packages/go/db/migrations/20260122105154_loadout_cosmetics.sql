-- +goose Up
CREATE TABLE loadout_cosmetics (
    loadout_id INTEGER NOT NULL,
    cosmetic_id INTEGER NOT NULL,
    slot TEXT NOT NULL CHECK (slot IN ('character_skin', 'weapon_skin', 'emote', 'taunt', 'badge', 'title', 'particle_effect', 'banner', 'other')),
    PRIMARY KEY (loadout_id, cosmetic_id),
    FOREIGN KEY (loadout_id) REFERENCES loadouts (loadout_id) ON DELETE CASCADE,
    FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
);

CREATE INDEX idx_loadout_cosmetics_cosmetic_id ON loadout_cosmetics (cosmetic_id);

-- +goose Down
DROP INDEX idx_loadout_cosmetics_cosmetic_id;
DROP TABLE loadout_cosmetics;