# Database Module Guidelines

## Migration Management

- **Goose Migration Files**: Must include `-- +goose Up` and `-- +goose Down` annotations
- **Down Migrations**: Must drop indexes before tables (reverse order of creation)
- **SQLite Specifics**:
  - Use `TEXT` datatype for timestamps with ISO 8601 format (`strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`)
  - Foreign key indexes are NOT automatically created; create explicit `CREATE INDEX` statements
  - Enable foreign keys with `PRAGMA foreign_keys = ON` on connection
  - Enable WAL mode for better concurrency: `PRAGMA journal_mode = WAL`
- **Migration Testing**: Use in-memory SQLite database (`:memory:`) for fast migration tests
- **Test Maintenance**: When adding new migrations, update the migration test's tables list and rollback verification

## Running Migrations

- Use the migration CLI: `go run ./cmd/migrate -up -db ./data.db`
- The migration runner automatically enables foreign keys and WAL mode
- Migration directory defaults to `./migrations` relative to working directory
- For rollback: `go run ./cmd/migrate -down -db ./data.db`

## Integration with Application

- Call `migrations.RunMigrations(db)` on application startup
- The `migrations` package is located at `pkg/migrations`
- Ensure database connection is established before running migrations

## Database Connection Pooling

- Use `database.OpenDB(path)` from `pkg/database` for production connections
- Default pool settings: MaxOpenConns=5, MaxIdleConns=2, ConnMaxLifetime=5m, ConnMaxIdleTime=2m
- Foreign keys and WAL mode are automatically enabled
- For testing, use `database.OpenInMemory()` which uses the same settings
- The connection pool is shared across the application via `*sql.DB` instance

## Package Structure

- **Generated Code**: Located in `internal/db/` to keep the root clean.
- **Public API**: The root `db` package exports types and the `Queries` struct via type aliases and wraps `New()`.
- **Custom Types**: Located in `types/` (e.g., `Timestamp`).
- **SQLC Configuration**: `sqlc.yaml` is configured to output to `internal/db`.
