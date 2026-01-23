# PRD: Refactor Go Backend to Unified Backend-API App

## Introduction
The current Go backend implementation is split across `packages/go/server` and `packages/go/db`. This structure is fragmented, as the database logic is tightly coupled with the server, and the server itself functions as a standalone application. This PRD outlines the consolidation of these components into a single `backend-api` application within the `apps/` directory, aligning with the project's C4 architecture.

## Goals
- Consolidate Go server and database logic into a single Go module.
- Relocate the backend to `apps/backend-api`.
- Simplify the Go workspace structure.
- Maintain `packages/go/` for future shared utility packages.

## User Stories

### US-001: Initialize Backend-API Application
**Description:** As a developer, I want a single entry point and module for the backend so that I can manage dependencies and builds more easily.

**Acceptance Criteria:**
- [ ] Directory `apps/backend-api` created.
- [ ] New `go.mod` initialized with module name `ai-zombie-defense/backend-api`.
- [ ] `go.work` updated to include `./apps/backend-api` and remove old paths.

### US-002: Migrate Database Logic
**Description:** As a developer, I want the database schema, migrations, and generated code to live within the backend app since they are tightly coupled.

**Acceptance Criteria:**
- [ ] `packages/go/db` contents moved to `apps/backend-api/db/`.
- [ ] `sqlc.yaml` and migrations relocated and verified.
- [ ] Database internal packages merged into the new module structure.

### US-003: Migrate Server Logic
**Description:** As a developer, I want the server implementation to be the core of the new `backend-api` app.

**Acceptance Criteria:**
- [ ] `packages/go/server` contents moved to `apps/backend-api/`.
- [ ] All internal imports updated to use `ai-zombie-defense/backend-api/...`.
- [ ] Main entry point located at `apps/backend-api/cmd/server/main.go`.

### US-004: Verify Unified Build and Tests
**Description:** As a developer, I want to ensure the unified application builds and passes all existing tests.

**Acceptance Criteria:**
- [ ] `go build ./...` succeeds in `apps/backend-api`.
- [ ] All unit and integration tests pass in the new structure.
- [ ] Linter/Check tasks pass.

## Functional Requirements
- FR-1: Create `apps/backend-api` with a unified `go.mod`.
- FR-2: Move all files from `packages/go/db` to `apps/backend-api/db`.
- FR-3: Move all files from `packages/go/server` to `apps/backend-api`.
- FR-4: Refactor all import paths from `ai-zombie-defense/server` and `ai-zombie-defense/db` to `ai-zombie-defense/backend-api`.
- FR-5: Consolidate `go.sum` and dependencies.
- FR-6: Update `go.work` at the repository root.

## Non-Goals
- No architectural changes to the Go services logic (keeping current Fiber/Service pattern).
- No changes to the SQLite database schema.
- No deletion of the `packages/go/` directory itself (keep it for future shared packages).

## Technical Considerations
- **Module Path:** The new module path will be `ai-zombie-defense/backend-api`.
- **Workspace:** Ensure `go.work` is updated *before* cleaning up old modules to avoid IDE/Tooling errors during migration.
- **SQLC:** The `sqlc.yaml` paths will need adjustment to point to the new relative locations of queries and schema.

## Success Metrics
- Backend application successfully builds from the `apps/backend-api` directory.
- `go.work` contains only the new app path for backend logic.
- Zero regressions in API functionality.

## Open Questions
- Should we move `pkg/config` and `pkg/logging` into `internal/` if they aren't intended for external use? (To be decided during implementation).
