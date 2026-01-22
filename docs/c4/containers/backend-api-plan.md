# Backend API Development Plan

## Overview

This document outlines the comprehensive development plan for the AI Zombie Defense Backend API based on the documented architecture, user stories, and system requirements.

## Current State Analysis

- **Documentation Complete**: Comprehensive PRD, 9 user stories, sequence diagrams, C4 architecture diagrams, and database ERD available
- **No Existing Backend Code**: Only placeholder files in packages/go directory
- **Technology Stack Defined**: Go 1.24.1 with SQLite and sqlc (per PRD technical constraints)

## Technology Stack

### Core Technologies

- **Language**: Go 1.24.1
- **Database**: SQLite with WAL mode enabled for better concurrency
- **Query Generation**: sqlc for type-safe SQL queries from SQL schemas
- **HTTP Framework**: Fiber for routing and middleware
- **Migrations**: Goose for SQLite migration management
- **Authentication**: JWT tokens with bcrypt password hashing

### Development Tools

- **Testing**: Go testing package with testify for assertions
- **Validation**: go-playground/validator for input validation
- **Configuration**: Viper for environment variables
- **Logging**: Zap for structured logging
- **API Documentation**: Swagger/OpenAPI 3.0 with go-swagger

### Infrastructure

- **Containerization**: Docker for consistent deployment
- **Orchestration**: Docker Compose for local development
- **Monitoring**: Prometheus metrics and structured logging

## Architecture Decisions

### Monolithic Backend

Single Go binary with internal packages for each service:

- Simpler deployment and operation
- Reduced network overhead between services
- Shared database connection pool
- Easier debugging and monitoring

### HTTP Framework Choice

- **Fiber**: Chosen over Gin for its performance characteristics, lower memory footprint, and Express-like API that is familiar to developers coming from Node.js backgrounds.
- **Key Features**: Built-in middleware support, route grouping, request context, and JSON handling that align with our API gateway requirements.

### Service Decomposition (Go Packages)

```txt
packages/go/
├── internal/           # Private application code
│   ├── database/      # DB connection, migrations
│   ├── models/        # sqlc-generated types
│   └── middleware/    # Auth, logging, validation
├── pkg/               # Public packages (services)
│   ├── auth/          # Authentication service
│   ├── account/       # Account management
│   ├── progression/   # Progression & cosmetics
│   ├── serverregistry/# Server registration
│   ├── matchhistory/  # Match results storage
│   ├── loot/          # Loot drop system
│   ├── leaderboard/   # Ranking calculations
│   └── social/        # Friends & favorites
├── cmd/backend-api/   # Main application entry
└── sql/               # SQL migrations & queries
```

### Database Design

Based on the ERD diagram (`database-erd.puml`):

- **Authentication & Accounts**: players, sessions, player_settings
- **Progression & Economy**: player_progression, cosmetic_items, player_cosmetics
- **Gameplay & Match Data**: servers, matches, player_match_stats
- **Social Features**: friends, server_favorites
- **Loot System**: loot_tables, loot_table_entries

## Development Phases

### Phase 1: Foundation & Database (Weeks 1-2)

**Priority**: Database schema first (user selection)

**Tasks:**

1. Convert ERD to SQL migration files
2. Set up sqlc configuration (sqlc.yaml) and generate Go models
3. Implement goose migration runner
4. Create database connection with connection pooling
5. Set up structured logging and configuration management
6. Create basic Fiber server with health check endpoint

**Deliverables:**

- SQL migration files for all tables
- sqlc-generated Go models for type-safe queries
- Migration system with version control
- Basic HTTP server with health endpoint

### Phase 2: Authentication & Accounts (Weeks 3-4)

**Tasks:**

1. Implement user registration with password hashing (bcrypt)
2. Create JWT token generation and validation
3. Build session management with token refresh
4. Implement account profile management
5. Add input validation and rate limiting middleware
6. Create player settings storage and retrieval

**Deliverables:**

- Complete authentication service (register, login, refresh, logout)
- JWT middleware for protected endpoints
- Account management API
- Player settings API

### Phase 3: Progression System (Weeks 5-6)

**Tasks:**

1. Implement level and XP progression system
2. Create prestige system for cosmetic rewards
3. Build currency management (Data currency)
4. Develop cosmetic catalog and ownership system
5. Create loadout management for equipped cosmetics
6. Implement match history storage and retrieval

**Deliverables:**

- Progression service with level/XP calculations
- Cosmetic system with catalog and ownership
- Match history service for player statistics
- Currency transaction validation

### Phase 4: Server Infrastructure (Weeks 7-8)

**Tasks:**

1. Build server registration for dedicated servers
2. Implement heartbeat system for server health monitoring
3. Create server browser API with filtering capabilities
4. Develop join token system for secure server access
5. Build server favorites management
6. Implement server region and version tracking

**Deliverables:**

- Server registry service with real-time status
- Server browser API for game client
- Join token generation and validation
- Server favorites system

### Phase 5: Economy & Rewards (Weeks 9-10)

**Tasks:**

1. Implement loot table configuration system
2. Build weighted random drop system for cosmetics
3. Create reward calculation based on match performance
4. Develop Data currency purchase validation
5. Implement special enemy loot drop integration
6. Create admin endpoints for loot table management

**Deliverables:**

- Loot service with weighted drop calculations
- Reward system for match completion
- Currency purchase validation
- Admin tools for loot table configuration

### Phase 6: Social & Leaderboards (Weeks 11-12)

**Tasks:**

1. Implement friend system with request/accept workflow
2. Create blocking and privacy features
3. Build leaderboard calculation service
4. Implement ranking algorithms for different periods
5. Create API gateway with route aggregation
6. Add rate limiting and CORS handling
7. Implement comprehensive error handling

**Deliverables:**

- Social service with friend management
- Leaderboard service with daily/weekly/all-time rankings
- Complete API gateway with middleware
- Production-ready error handling and logging

## API Design

### Authentication Endpoints

```txt
POST   /auth/register     - Create new account
POST   /auth/login        - Authenticate and receive JWT
POST   /auth/refresh      - Refresh expired tokens
POST   /auth/logout       - Invalidate session
```

### Account & Progression Endpoints

```txt
GET    /account/profile   - Get player profile
PUT    /account/profile   - Update profile info
GET    /progression       - Get levels, XP, currency balances
GET    /cosmetics         - Get owned cosmetics
PUT    /cosmetics/equip   - Equip cosmetic to loadout
POST   /cosmetics/purchase - Buy cosmetic with Data currency
```

### Server Registry Endpoints

```txt
GET    /servers           - List active servers (for server browser)
POST   /servers/register  - Dedicated server registration
PUT    /servers/:id/heartbeat - Update server status
POST   /servers/:id/join  - Generate join token for player
```

### Match History Endpoints

```txt
POST   /matches           - Submit match results
GET    /matches/history   - Get player match history
GET    /matches/:id       - Get detailed match data
```

### Loot System Endpoints

```txt
GET    /loot/tables       - Get active loot tables (admin)
POST   /loot/drop         - Request loot drop for player
GET    /loot/catalog      - Get cosmetic catalog for store
```

### Leaderboard Endpoints

```txt
GET    /leaderboards/daily    - Daily rankings
GET    /leaderboards/weekly   - Weekly rankings
GET    /leaderboards/alltime  - All-time rankings
```

### Social Endpoints

```txt
GET    /friends           - Get friend list
POST   /friends/request   - Send friend request
PUT    /friends/:id       - Accept/decline/block friend
GET    /favorites         - Get favorited servers
POST   /favorites         - Add server to favorites
```

### Request/Response Format

All endpoints use JSON with consistent error format:

```json
{
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "details": {}
  }
}
```

Success responses include `data` field with resource-specific payload.

## Integration Points

### Game Client Integration

- **Authentication**: JWT token storage and automatic refresh
- **Server Browser**: Poll `/servers` endpoint for real-time server list
- **Progression Sync**: Fetch player data on main menu load
- **Cosmetic Store**: Display catalog from `/loot/catalog` endpoint
- **Friend System**: Real-time updates (future WebSocket implementation)

### Dedicated Server Integration

- **Server Registration**: POST `/servers/register` on startup with server metadata
- **Heartbeat Updates**: PUT `/servers/:id/heartbeat` every 30 seconds
- **Match Submission**: POST `/matches` with player stats on match completion
- **Loot Requests**: POST `/loot/drop` for special enemy kills
- **Join Validation**: Verify join tokens from connecting players

### Data Flow Examples

1. **Player Joins Server**:
   - Client requests join token from backend
   - Backend validates player and server
   - Client connects to server with token
   - Server validates token with backend

2. **Match Completion**:
   - Server submits match results to backend
   - Backend updates player progression and currency
   - Backend checks for achievement unlocks
   - Backend notifies client of new cosmetics

3. **Cosmetic Purchase**:
   - Client requests cosmetic purchase
   - Backend validates Data currency balance
   - Backend deducts currency and unlocks cosmetic
   - Backend updates player loadout if equipped

## Testing Strategy

### Unit Testing

- **Service Layer**: Mock database dependencies
- **Business Logic**: Currency calculations, loot drop probabilities
- **Validation**: Input validation, authentication checks
- **Target**: 80%+ unit test coverage for business logic

### Integration Testing

- **API Endpoints**: Test full request/response cycle
- **Database Operations**: Real SQLite in-memory database
- **Service Interactions**: Auth → Progression → Database chain
- **Target**: 100% coverage for authentication and security-critical code

### End-to-End Testing

- **Complete Flows**: Registration → Login → Match → Rewards
- **Load Testing**: Simulate multiple servers and players
- **Concurrency Testing**: Race conditions in progression updates
- **Performance Testing**: Leaderboard calculation performance

### Test Automation

- **CI/CD Pipeline**: Automated tests on pull requests
- **Database Fixtures**: Consistent test data setup
- **Mock Servers**: Simulate dedicated server behavior
- **Load Testing**: Locust or k6 for API performance testing

## Deployment Considerations

### Development Environment

- **Local Setup**: Docker Compose with SQLite volume
- **Hot Reload**: Air for automatic server restart
- **Database Viewer**: SQLite browser for data inspection
- **API Documentation**: Swagger UI available at `/swagger`

### Staging Environment

- **Isolated Database**: Separate SQLite file from production
- **Mock Game Clients**: Scripts to simulate player activity
- **Load Testing**: Regular performance testing
- **Monitoring**: Basic metrics and logging

### Production Deployment

- **Containerization**: Docker image with multi-stage build
- **Database Persistence**: Persistent volume for SQLite file
- **Backup Strategy**: Regular SQLite backups to cloud storage
- **Monitoring**: Prometheus metrics, structured JSON logs
- **Scaling**: Horizontal scaling with load balancer (future need)

### Security Measures

- **HTTPS Enforcement**: All API traffic over TLS
- **Rate Limiting**: Per-IP and per-user request limits
- **Input Validation**: All user inputs sanitized
- **SQL Injection Prevention**: sqlc generated queries
- **Secrets Management**: Environment variables for JWT secrets
- **CORS Configuration**: Restricted to game client domains

### Performance Considerations

- **Database Indexes**: Optimize queries based on ERD relationships
- **Connection Pooling**: Configure SQLite connection limits
- **Caching Strategy**: Redis for leaderboards (future implementation)
- **Query Optimization**: Monitor slow queries with logging
- **WAL Mode**: Enable Write-Ahead Logging for better concurrency

## Success Metrics

### Functional Requirements

- Support all 9 documented user stories
- Handle 32-player dedicated servers with real-time status
- Process match results within 1 second of submission
- Provide sub-second response times for progression queries

### Performance Requirements

- 99% uptime target with automated health checks
- Support thousands of player accounts
- Handle hundreds of concurrent dedicated servers
- Process leaderboard calculations within 5 seconds

### Quality Requirements

- Comprehensive API documentation with examples
- Detailed error messages for troubleshooting
- Consistent response formats across all endpoints
- Backward compatibility for API changes

## Next Steps

### Immediate Actions

1. Create SQL migration files from ERD diagram
2. Set up sqlc configuration and generate Go models
3. Implement goose migration runner
4. Create basic Fiber server structure
5. Begin Phase 1 implementation

### Dependencies

- **Game Client**: Mock client for integration testing
- **Dedicated Server**: Reference implementation for protocol validation
- **Documentation**: Keep API documentation updated with implementation

### Risks & Mitigations

- **SQLite Scaling**: Monitor performance; consider PostgreSQL if needed
- **Concurrency Issues**: Implement proper locking for progression updates
- **Security Vulnerabilities**: Regular security reviews and dependency updates

---

_This plan is based on analysis of documentation in `/Users/milanrodd/Projects/ai-zombie-defense/docs/` including PRD, user stories, sequence diagrams, C4 architecture diagrams, and database ERD._

_Last Updated: January 22, 2026_
