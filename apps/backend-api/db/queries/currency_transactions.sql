-- name: CreateCurrencyTransaction :exec
INSERT INTO currency_transactions (player_id, amount, balance_after, transaction_type, reference_id)
VALUES (?, ?, ?, ?, ?);

-- name: GetCurrencyTransactionsByPlayer :many
SELECT * FROM currency_transactions WHERE player_id = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: GetCurrencyTransactionsByPlayerAndType :many
SELECT * FROM currency_transactions WHERE player_id = ? AND transaction_type = ? ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: CountCurrencyTransactionsByPlayer :one
SELECT COUNT(*) FROM currency_transactions WHERE player_id = ?;