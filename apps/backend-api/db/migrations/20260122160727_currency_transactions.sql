-- +goose Up
CREATE TABLE currency_transactions (
    transaction_id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    amount INTEGER NOT NULL,
    balance_after INTEGER NOT NULL,
    transaction_type TEXT NOT NULL CHECK (transaction_type IN ('match_reward', 'purchase', 'prestige_reward', 'admin_grant', 'refund', 'other')),
    reference_id INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (player_id) REFERENCES players (player_id) ON DELETE CASCADE
);

CREATE INDEX idx_currency_transactions_player_id ON currency_transactions (player_id);
CREATE INDEX idx_currency_transactions_created_at ON currency_transactions (created_at);

-- +goose Down
DROP INDEX idx_currency_transactions_created_at;
DROP INDEX idx_currency_transactions_player_id;
DROP TABLE currency_transactions;