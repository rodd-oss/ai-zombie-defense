-- +goose Up
CREATE TABLE friends (
    player_id INTEGER NOT NULL,
    friend_id INTEGER NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'blocked')),
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    PRIMARY KEY (player_id, friend_id),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE,
    FOREIGN KEY (friend_id) REFERENCES players (player_id) ON DELETE CASCADE
);

CREATE INDEX idx_friends_friend_id ON friends (friend_id);

-- +goose Down
DROP INDEX idx_friends_friend_id;
DROP TABLE friends;