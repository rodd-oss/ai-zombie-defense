-- +goose Up
ALTER TABLE players ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE players DROP COLUMN is_admin;
