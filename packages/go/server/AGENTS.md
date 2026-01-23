# Server Module Guidelines

## Logging Configuration

- Use `pkg/logging.NewLogger()` to create a configured zap.Logger
- Environment variables:
  - `LOG_LEVEL`: debug, info, warn, error, dpanic, panic, fatal (default: info)
  - `LOG_ENCODING`: "json" (production) or "console" (development with colors)
- The logger automatically uses ISO8601 timestamps in JSON mode
- For development, set `LOG_ENCODING=console` for human-readable colored output
- Always call `defer logger.Sync()` in main, but note that Sync may fail on stdout

## Configuration Management

- Use `pkg/config.LoadConfig()` to load configuration from environment variables
- Required environment variable: `JWT_SECRET` (no default)
- Database, server, and JWT settings have sensible defaults
- Environment variable naming: uppercase with underscores (e.g., `DB_PATH`, `SERVER_PORT`)
- Duration values use Go's time.ParseDuration format (e.g., "5m", "1h", "7d")
- Viper automatically binds environment variables; no config files needed

## Module Structure

- The server module is a separate Go module (`ai-zombie-defense/server`)
- Transitioning to a **Modular Monolith** structure (see C4 docs)
- Service interfaces and logic are being moved to `internal/services/<module>/`
- Handlers should depend on service interfaces
- Dependencies: zap for logging, viper for configuration
- Workspace includes both `ai-zombie-defense/db` and `ai-zombie-defense/server`
- Use `go.work` to develop both modules together
- Always run `go mod tidy` after adding new dependencies

## Testing

- Tests use `t.Setenv` to set environment variables per test
- Clear environment variables before each test to avoid pollution
- Test both default values and environment overrides
- Use `-race` flag when running tests to detect data races
- Service tests should use in-memory SQLite and setup required tables manually in `setupTestDB`

## HTTP Server with Fiber

- Use Fiber v2 for HTTP server; import `fiberLogger` to avoid naming conflict with zap logger
- Default middleware: `logger.New()` and `recover.New()`
- Health endpoint: `GET /health` returns `{"status":"ok"}`
- Server configuration uses `SERVER_HOST` and `SERVER_PORT` environment variables (default: `0.0.0.0:8080`)
- Shutdown requires context; call `ShutdownWithContext(ctx)` with timeout
- Create server instance via `server.New(cfg, logger)`
- Start server with `srv.Start()`; graceful shutdown with `srv.Shutdown(ctx)`
- Test servers using random free ports via `net.Listen` and `zaptest.Logger`

## Global Middleware

- CORS middleware is enabled by default with configurable origins via `CORS_ALLOW_ORIGINS` environment variable (default: "*")
- Rate limiting middleware is enabled with configurable max requests and duration via `RATE_LIMIT_MAX` (default: 10) and `RATE_LIMIT_DURATION` (default: 1m)
- Error handler returns consistent JSON error responses with status codes
- 404 handler returns JSON `{"error": "route not found"}`
- Middleware order: CORS → Logger → Recovery → Rate Limiter

## Authentication

- Use `internal/services/auth.Service` for authentication logic (legacy: `pkg/auth.Service` delegates to it)
- JWT tokens use HS256 signing with configurable expiration
- Access tokens are short-lived (default 15 minutes)
- Refresh tokens are long-lived (default 7 days) and stored in `sessions` table
- Include a random JWT ID (jti) claim in refresh tokens to ensure uniqueness
- Password hashing uses bcrypt with default cost
- Handle duplicate token errors gracefully (retry generation if collision occurs)
- Handle duplicate username/email constraints by checking SQLite error strings; return user-friendly conflict errors
- Validate refresh tokens against both JWT signature and session store
- Refresh endpoint rotates tokens (deletes old session, creates new one)
- Logout endpoint deletes the session by token

## Account Service

- Use `internal/services/account.Service` for player profile and settings logic (legacy: `pkg/auth.Service` delegates to it)
- `GetPlayer` retrieves basic player information
- `UpdatePlayerProfile` handles username and email changes; returns `ErrDuplicateUsername` or `ErrDuplicateEmail` on conflict
- `UpdatePlayerPassword` handles secure password updates via bcrypt
- `GetPlayerSettings` returns player-specific settings (mouse sensitivity, keybindings, etc.) or defaults if none exist
- `UpsertPlayerSettings` creates or updates settings in a single operation

## Progression Service

- Use `internal/services/progression.Service` for XP, prestige, and currency logic
- `AddExperience` handles level-ups automatically based on `BaseXPPerLevel` config
- `AddMatchRewards` calculates and awards XP/Data based on match performance (kills, waves, etc.)
- `PrestigePlayer` resets level/XP and grants exclusive cosmetics
- `PurchaseCosmetic` handles currency deduction and ownership granting in a transaction

## Loot Service

- Use `internal/services/loot.Service` for loot table management and drop generation
- `GenerateLootDrop` selects a random active loot table and entry based on weights
- Loot tables and entries should be managed via administrative endpoints (coming soon)

## Middleware

- JWT middleware is available in `pkg/middleware.AuthMiddleware`
- Use `middleware.AuthMiddleware(authService, logger)` to protect routes
- Extracts player ID from token subject claim and stores in `c.Locals("player_id")`
- Helper functions `middleware.GetPlayerID(c)` and `middleware.GetClaims(c)` retrieve data
- Returns 401 for missing/invalid tokens with JSON error response
- Always use Bearer token format: `Authorization: Bearer <token>`

## Server Authentication Middleware

- Server authentication uses `X-Server-Token` header and path parameter validation
- Middleware: `pkg/middleware.ServerAuthMiddleware(authService, logger)`
- Extracts server ID from path param `:id`, validates token via `authService.GetServerByAuthToken`
- Stores server ID in `c.Locals("server_id")`; retrieve with `middleware.GetServerID(c)`
- Returns 401 for missing/invalid tokens, 403 for server ID mismatch
- Used for heartbeat endpoint and future server-authenticated endpoints

## Adding New Endpoints
- Pattern for adding new endpoints:
  1. Add SQL queries in `db/queries/` (`.sql` files)
  2. Run `sqlc generate` in `packages/go/db/` to update Go models
  3. Add service methods in `internal/services/<module>/`
  4. Add handlers in `pkg/handlers/`
  5. Register routes in `pkg/server/server.go` with appropriate middleware
  6. Write unit tests for services and integration tests for handlers
  7. Ensure test database includes required tables (update `setupTestDB`)
  8. Run `go mod tidy` in affected modules
  9. Run `bun task check` to verify linting and type checks
  10. Run `go test ./...` to ensure all tests pass

