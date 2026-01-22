-- +goose Up
CREATE TABLE join_tokens (
    join_token_id INTEGER PRIMARY KEY AUTOINCREMENT,
    token TEXT NOT NULL UNIQUE,
    player_id INTEGER NOT NULL,
    server_id INTEGER NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    used_at TEXT,
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
    FOREIGN KEY (server_id) REFERENCES servers (server_id) ON DELETE CASCADE
);

CREATE INDEX idx_join_tokens_token ON join_tokens(token);
CREATE INDEX idx_join_tokens_player_id ON join_tokens(player_id);
CREATE INDEX idx_join_tokens_server_id ON join_tokens(server_id);
CREATE INDEX idx_join_tokens_expires_at ON join_tokens(expires_at);

-- +goose Down
DROP INDEX idx_join_tokens_expires_at;
DROP INDEX idx_join_tokens_server_id;
DROP INDEX idx_join_tokens_player_id;
DROP INDEX idx_join_tokens_token;
DROP TABLE join_tokens;