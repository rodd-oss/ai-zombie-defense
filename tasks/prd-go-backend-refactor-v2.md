# PRD: Go Backend Architectural Refactor (V2)

## Introduction
The current Go backend implementation in `packages/go/server` has significant logic duplication and structural "spaghettiness." Specifically, `pkg/auth/auth.go` acts as a monolithic facade, and there is confusion between the `Server` in `pkg/server` and the `APIGateway` in `internal/api/gateway`. This refactor will align the codebase with the Modular Monolith pattern, move logic to `internal/`, and clean up the `packages/go/db` package.

## Goals
- **Modular Monolith Enforcement:** Move all service implementations, handlers, and middleware from `pkg/` to `internal/`.
- **Unify Entry Point:** Consolidate `pkg/server` and `internal/api/gateway` into a single, clean `APIGateway`.
- **Interface-Driven Design:** Handlers must depend on small, specific service interfaces (Interface Segregation).
- **Localized Error Handling:** Move shared errors from the global facade to their respective service packages.
- **DB Layer Cleanup:** Remove root-level clutter in `packages/go/db` and organize generated code into `internal/db`.

## User Stories

### US-001: Centralize Gateway Routing
**Description:** As a developer, I want a single gateway to manage all routing so that middleware and endpoint registration are consistent.
**Acceptance Criteria:**
- [ ] `pkg/server/server.go` is deleted; logic is merged into `internal/api/gateway/gateway.go`.
- [ ] Gateway supports `MountGroup` for services to register their own routes.
- [ ] `cmd/server/main.go` instantiates only the `APIGateway`.

### US-002: Internalize Business Logic
**Description:** As a developer, I want business logic to be hidden from the public API of the package.
**Acceptance Criteria:**
- [ ] `pkg/handlers/` contents moved to service-specific `internal/services/<name>/` directories.
- [ ] `pkg/middleware/` moved to `internal/middleware/`.
- [ ] `pkg/auth/auth.go` (the facade) is removed. Handlers now call specific services directly via interfaces.
- [ ] `pkg/` should contain only shared utilities or public interfaces if absolutely necessary.

### US-003: Refactor Database Package
**Description:** As a developer, I want the `db` package to be professional and well-organized.
**Acceptance Criteria:**
- [ ] SQLC generated files (`*.sql.go`, `db.go`, `models.go`) moved to `packages/go/db/internal/db/`.
- [ ] `generated_backup/` directory is removed.
- [ ] `pkg/database/` provides a unified connection/pool management utility.
- [ ] `sqlc.yaml` updated to reflect the new structure.

### US-004: Decentralize Errors
**Description:** As a developer, I want errors to be defined near the logic that produces them.
**Acceptance Criteria:**
- [ ] `ErrPlayerBanned`, `ErrInvalidCredentials`, etc., moved from the central facade to `internal/services/auth/`.
- [ ] `ErrDuplicateUsername` moved to `internal/services/account/`.
- [ ] All handlers updated to use the new error locations.

## Functional Requirements
- **FR-1:** Services must communicate via interfaces defined in their respective `service.go` files.
- **FR-2:** The `APIGateway` must handle global middleware: CORS, Recovery, Request Logging, and Rate Limiting.
- **FR-3:** `packages/go/db` must support transaction-safe operations usable across multiple services.

## Non-Goals
- No changes to the database schema or SQL queries (logic-neutral refactor).
- No new features or API endpoints.
- No changes to the Godot client or Godot dedicated server code.

## Technical Considerations
- **SQLC Generation:** Need to run `sqlc generate` and verify output locations.
- **Workspace Dependencies:** Update `go.work` and `go.mod` files to ensure all internal imports are correct.
- **Testing:** Relocate unit and integration tests from `pkg/` to `internal/` alongside the code they test.

## Success Metrics
- `pkg/handlers`, `pkg/middleware`, and `pkg/auth` are empty/deleted.
- `packages/go/db` root directory contains only configuration files and subdirectories.
- `bun task check` (specifically Go linting/build) passes without errors.
- All Go tests pass.
