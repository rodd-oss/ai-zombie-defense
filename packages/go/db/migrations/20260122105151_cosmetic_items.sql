-- +goose Up
CREATE TABLE cosmetic_items (
    cosmetic_id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    description TEXT,
    slot TEXT NOT NULL CHECK (slot IN ('character_skin', 'weapon_skin', 'emote', 'taunt', 'badge', 'title', 'particle_effect', 'other')),
    category TEXT,
    rarity TEXT NOT NULL CHECK (rarity IN ('common', 'uncommon', 'rare', 'epic', 'legendary')),
    unlock_level INTEGER NOT NULL DEFAULT 1,
    data_cost INTEGER NOT NULL DEFAULT 0,
    is_prestige_only INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- +goose Down
DROP TABLE cosmetic_items;