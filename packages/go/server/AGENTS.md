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
- Dependencies: zap for logging, viper for configuration
- Workspace includes both `ai-zombie-defense/db` and `ai-zombie-defense/server`
- Use `go.work` to develop both modules together
- Always run `go mod tidy` after adding new dependencies

## Testing

- Tests use `t.Setenv` to set environment variables per test
- Clear environment variables before each test to avoid pollution
- Test both default values and environment overrides
- Use `-race` flag when running tests to detect data races

## HTTP Server with Fiber

- Use Fiber v2 for HTTP server; import `fiberLogger` to avoid naming conflict with zap logger
- Default middleware: `logger.New()` and `recover.New()`
- Health endpoint: `GET /health` returns `{"status":"ok"}`
- Server configuration uses `SERVER_HOST` and `SERVER_PORT` environment variables (default: `0.0.0.0:8080`)
- Shutdown requires context; call `ShutdownWithContext(ctx)` with timeout
- Create server instance via `server.New(cfg, logger)`
- Start server with `srv.Start()`; graceful shutdown with `srv.Shutdown(ctx)`
- Test servers using random free ports via `net.Listen` and `zaptest.Logger`