# PRD: Flatten Backend API Database Structure

## Introduction
The current database structure in `@apps/backend-api/db/` is overly nested, creating a "package-within-a-package" feel that complicates imports and violates the desired flat structure of the `backend-api` application. This PRD outlines the steps to consolidate the database logic into the main application structure, moving generated code and database utilities into `internal/db`.

## Goals
- Simplify the project structure by removing the redundant `db/` sub-directory.
- Consolidate all database implementation details (generated code, migrations, types) into `internal/db`.
- Enforce encapsulation by making database implementation details internal to the `backend-api` package.
- Integrate the migration runner into the main server binary as a subcommand.
- Maintain `sqlc` functionality with updated paths.

## User Stories

### US-001: Relocate Database Assets
**Description:** As a developer, I want all database-related files (schema, queries, migrations) to be located in standard project directories so that the project structure is predictable.

**Acceptance Criteria:**
- [ ] `db/migrations/` moved to `migrations/` at the root of `backend-api`.
- [ ] `db/queries/` and `db/schema/` moved to `internal/db/sql/`.
- [ ] `db/types/` moved to `internal/db/types/`.
- [ ] `sqlc.yaml` moved to the root of `backend-api` and updated with new paths.
- [ ] All `sqlc` generated code is output to `internal/db/generated/`.

### US-002: Consolidate Database Utilities
**Description:** As a developer, I want the database connection and migration runner utilities to be part of the internal database package.

**Acceptance Criteria:**
- [ ] `db/pkg/database/` logic moved to `internal/db/connection.go`.
- [ ] `db/pkg/migrations/` logic moved to `internal/db/migration_runner.go`.
- [ ] The `db.go` facade (currently at `db/db.go`) is removed in favor of direct internal imports or a simplified internal package structure.

### US-003: Integrate Migration Subcommand
**Description:** As an operator, I want to run database migrations using the main server binary so that I don't have to manage multiple binaries.

**Acceptance Criteria:**
- [ ] `cmd/server/main.go` supports a `migrate` subcommand or flag (e.g., `./server migrate up`).
- [ ] The migration logic correctly locates the `migrations/` directory relative to the binary or via configuration.
- [ ] The existing `db/cmd/migrate/` is removed.

### US-004: Update Service Dependencies
**Description:** As a developer, I want all services to use the new `internal/db` structure so that the application remains functional.

**Acceptance Criteria:**
- [ ] All imports of `ai-zombie-defense/backend-api/db` updated to `ai-zombie-defense/backend-api/internal/db`.
- [ ] All services (account, progression, social) verified to compile and pass tests.

## Functional Requirements
- FR-1: Move `db/migrations/` -> `migrations/`
- FR-2: Move `db/queries/` -> `internal/db/sql/queries/`
- FR-3: Move `db/schema/` -> `internal/db/sql/schema/`
- FR-4: Move `db/types/` -> `internal/db/types/`
- FR-5: Update `sqlc.yaml` to point to new locations.
- FR-6: Implement migration command handling in `cmd/server/main.go`.
- FR-7: Update all internal service references and types.
- FR-8: Delete the `db/` directory after successful migration and verification.

## Non-Goals
- Changing the database schema or migration logic.
- Adding new database features.
- Publicly exposing `internal/db` outside of the `backend-api` module.

## Technical Considerations
- `sqlc` configuration needs to be carefully updated to ensure type overrides still work with the new `internal/db/types` package.
- The `go.mod` file for `backend-api` remains at the root and does not need major changes other than internal reference updates.
- Migration directory discovery needs to be robust (relative to binary or via env var).

## Success Metrics
- Reduction in project depth (fewer nested directories).
- Successful execution of `bun task check` and `bun task test`.
- Single binary (`server`) capable of running both API and migrations.

## Open Questions
- None at this time based on user feedback.
