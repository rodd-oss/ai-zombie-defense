# Database Internal Module Guidelines

## Generated Code
- **Path**: `internal/db/generated/`
- **Package**: `generated`
- **Tool**: Generated using `sqlc generate` from the `apps/backend-api/` root.

## SQL Sources
- **Schema**: `internal/db/sql/schema/schema.sql`
- **Queries**: `internal/db/sql/queries/*.sql`

## Custom Types
- **Path**: `internal/db/types/`
- **Usage**: Used in `sqlc.yaml` as overrides for SQLite TEXT timestamps to provide proper `time.Time` support with JSON marshaling.
- **Import Path**: `ai-zombie-defense/backend-api/internal/db/types`

## Configuration
- `sqlc.yaml` is located in the `apps/backend-api/` root.
- Always run `sqlc generate` after modifying SQL files.
