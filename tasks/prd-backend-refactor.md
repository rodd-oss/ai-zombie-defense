# PRD: Backend Refactor - Modular Monolith

## Introduction
Refactor the existing Go backend implemented by the Ralph agent to align with the C4 architecture defined in `docs/c4/`. The current implementation is a monolithic structure with tightly coupled handlers and logic. This refactor will transition the codebase to a Modular Monolith, improving maintainability, testability, and architectural alignment while retaining the simplicity of a single deployment unit.

## Goals
- Reorganize the codebase into distinct service modules (Auth, Account, Progression, etc.) as defined in C4 containers.
- Decouple business logic from HTTP transport layers using interface-based dependency injection.
- Implement a central API Gateway for routing, logging, and cross-cutting concerns.
- Ensure 100% feature parity with the existing implementation.
- Maintain a single SQLite database with logical separation of concerns.

## User Stories

### US-001: Implement Service Module Structure
**Description:** As a developer, I want a clear directory structure for individual services so that business logic is isolated.

**Acceptance Criteria:**
- [ ] Create `internal/services/` directory.
- [ ] Implement sub-packages for: `auth`, `account`, `progression`, `match`, `server`, `social`, `leaderboard`, `loot`.
- [ ] Each service package contains its own logic and interface definitions.
- [ ] Typecheck passes.

### US-002: Decouple Business Logic from Handlers
**Description:** As a developer, I want handlers to depend on service interfaces rather than direct implementations or database calls.

**Acceptance Criteria:**
- [ ] Define interfaces for each service (e.g., `AuthService`, `AccountService`).
- [ ] Move logic from `server/pkg/handlers/` to corresponding service implementations.
- [ ] Handlers updated to accept interfaces via constructor injection.
- [ ] Unit tests for services are independent of HTTP context.
- [ ] Typecheck passes.

### US-003: Implement Centralized API Gateway
**Description:** As a developer, I want a unified entry point for all API requests that handles routing and middleware aggregation.

**Acceptance Criteria:**
- [ ] Implement `APIGateway` in `internal/api/gateway`.
- [ ] Gateway aggregates routes from all service handlers.
- [ ] Apply global middleware (CORS, Logging, Auth) at the gateway level.
- [ ] Alignment with `docs/c4/containers/backend-api-components/api-gateway.puml`.
- [ ] Typecheck passes.

### US-004: Dependency Injection and Startup Wiring
**Description:** As a developer, I want a clean main entry point that wires all services and handlers together.

**Acceptance Criteria:**
- [ ] Update `cmd/server/main.go` to initialize the database pool once.
- [ ] Instantiate all services and inject dependencies.
- [ ] Pass service instances to handlers.
- [ ] Initialize and start the API Gateway.
- [ ] Server starts successfully and passes health checks.

### US-005: Verification and Feature Parity
**Description:** As a user, I want the system to behave exactly as before but with a better internal structure.

**Acceptance Criteria:**
- [ ] All existing integration tests in `packages/go/server/pkg/handlers/` pass after refactor.
- [ ] Run `bun task check` and ensure no regressions.
- [ ] Verify that SQL migrations and `sqlc` generated code are correctly utilized in the new structure.

## Functional Requirements
- **FR-1:** Every service must define an interface for its public API.
- **FR-2:** Handlers must not contain any SQL queries; they must delegate to services.
- **FR-3:** Services must use the shared database connection pool but restrict queries to their domain.
- **FR-4:** The API Gateway must handle JWT validation for protected routes before delegating to service handlers.
- **FR-5:** Error handling must be consistent across all modules using a shared error utility or standard Fiber error handling.

## Non-Goals
- Migrating to microservices (out of scope for this phase).
- Splitting the SQLite database into multiple files.
- Introducing a message broker or external event bus.
- Rewriting the database schema or `sqlc` queries (unless necessary for decoupling).

## Technical Considerations
- **Framework:** Continue using Fiber as the web framework.
- **Database:** SQLite with WAL mode, managed by the existing `db` package.
- **Dependency Management:** Use standard Go constructor injection (NewService, NewHandler).
- **Project Structure:** Follow the `internal/` convention to prevent external packages from importing service implementations directly.

## Success Metrics
- Zero functionality loss (100% test pass rate).
- Reduced cognitive complexity in handlers (measured by code size and nesting).
- Clean architectural alignment with C4 diagrams.

## Open Questions
- Should we use a DI container (like Wire or Dig) or stick to manual wiring in `main.go`? (Start with manual wiring).
- How should cross-service dependencies be handled (e.g., Loot service needing Account data)? (Inject the required service interface).
