-- +goose Up
CREATE TABLE server_favorites (
    player_id INTEGER NOT NULL,
    server_id INTEGER NOT NULL,
    added_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    note TEXT,
    PRIMARY KEY (player_id, server_id),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
    FOREIGN KEY (server_id) REFERENCES servers (server_id) ON DELETE CASCADE
);

CREATE INDEX idx_server_favorites_server_id ON server_favorites (server_id);

-- +goose Down
DROP INDEX idx_server_favorites_server_id;
DROP TABLE server_favorites;