-- loot_table_entries table
CREATE TABLE loot_table_entries (
    loot_entry_id INTEGER PRIMARY KEY AUTOINCREMENT,
    loot_table_id INTEGER NOT NULL,
    cosmetic_id INTEGER NOT NULL,
    weight INTEGER NOT NULL,
    min_quantity INTEGER NOT NULL DEFAULT 1,
    max_quantity INTEGER NOT NULL DEFAULT 1,
    FOREIGN KEY (loot_table_id) REFERENCES loot_tables (loot_table_id) ON DELETE CASCADE,
    FOREIGN KEY (cosmetic_id) REFERENCES cosmetic_items (cosmetic_id) ON DELETE CASCADE
);

CREATE INDEX idx_loot_table_entries_loot_table_id ON loot_table_entries (loot_table_id);
CREATE INDEX idx_loot_table_entries_cosmetic_id ON loot_table_entries (cosmetic_id);