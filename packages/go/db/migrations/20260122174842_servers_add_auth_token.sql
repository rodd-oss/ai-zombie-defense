-- +goose Up
ALTER TABLE servers ADD COLUMN auth_token TEXT;
CREATE UNIQUE INDEX idx_servers_auth_token ON servers(auth_token);

-- +goose Down
DROP INDEX idx_servers_auth_token;
ALTER TABLE servers DROP COLUMN auth_token;