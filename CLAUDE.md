# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

This is a comprehensive microservices backend for a blockchain-based reciprocal club membership platform. The system consists of 8 microservices with multi-tenant architecture, event-driven communication via NATS, and Hyperledger Fabric blockchain integration.

## Common Development Commands

### Building and Running

```bash
# Build all services
make build

# Build specific service
cd services/auth-service && go build -o bin/auth-service ./cmd/main.go

# Run all tests across services
make test

# Run tests for specific service
cd services/auth-service && go test ./...

# Run only unit tests (faster)
make test-unit

# Test compilation without running tests
go test -c ./internal/...

# Format all code
make format

# Run linters
make lint
```

### Container Orchestration

```bash
# Start full system with Podman (preferred)
make podman-up

# Start with Docker Compose
make docker-up

# Check service status
make podman-status

# View logs for specific service
make podman-logs SERVICE=postgres

# Stop services
make podman-down

# Build container images
make build-images-podman
```

### Development Workflow

```bash
# Setup development environment
make setup-dev

# Reset everything and start fresh
make dev-reset

# Quick restart services
make restart

# Run database migrations
make db-migrate

# Tidy all Go modules
make tidy
```

## Architecture Overview

### Microservices Structure
- **8 services**: API Gateway, Auth, Member, Reciprocal, Blockchain, Notification, Analytics, Governance
- **Shared packages**: Located in `pkg/shared/` for common functionality
- **Multi-tenant**: Club-based data partitioning with tenant isolation
- **Event-driven**: NATS message bus for inter-service communication

### Service Structure Pattern
Each service follows a consistent internal structure:
```
services/{service-name}/
├── cmd/main.go              # Entry point
├── internal/
│   ├── handlers/            # HTTP/gRPC handlers
│   ├── service/             # Business logic
│   ├── repository/          # Data access layer
│   ├── models/              # Domain models
│   ├── config/              # Service configuration
│   └── testutil/            # Test utilities
├── Dockerfile
└── go.mod
```

### Shared Libraries
Key shared packages in `pkg/shared/`:
- **config**: Environment configuration with Viper
- **database**: PostgreSQL connections and GORM integration
- **messaging**: NATS client with retry logic
- **logging**: Structured logging with correlation IDs
- **monitoring**: Prometheus metrics and health checks
- **auth**: JWT authentication and multi-tenant authorization
- **errors**: Structured error handling

## Key Architectural Patterns

### Multi-Tenant Data Access
- All database operations must include `club_id` for tenant isolation
- Repository methods typically require `clubID` as first parameter
- Example: `repo.GetUserByID(ctx, clubID, userID)`

### Service Communication
- **Synchronous**: gRPC for request/response patterns
- **Asynchronous**: NATS events for eventual consistency
- **External APIs**: HTTP REST with structured error responses

### Configuration Management
- Services use `config.Load(serviceName)` to load configuration
- Environment variables override YAML defaults
- Shared configuration patterns via `pkg/shared/config`

### Testing Patterns
- **Unit tests**: Mock external dependencies, focus on business logic
- **Integration tests**: Use `testutil.NewTestDB()` for database testing
- **Service tests**: Test full service layer with mocked repositories
- All test files must compile with `go test -c ./internal/...`

## Database Considerations

### Connection Management
- Use `database.NewConnection(&cfg.Database, logger)` for connections
- All services share PostgreSQL but with tenant isolation
- Auto-migration on service startup via `db.Migrate(models...)`

### Repository Pattern
- Repository constructors: `NewAuthRepository(db, logger)`
- Always pass context as first parameter: `repo.Method(ctx, ...)`
- Use tenant-aware queries with `db.WithTenant(clubID)`

## Error Handling Standards

### Structured Errors
- Use `pkg/shared/errors` for consistent error types
- Return structured errors: `errors.NotFound("Resource not found", context)`
- Log errors with correlation IDs from context

### HTTP Error Responses
- 400 for validation errors
- 401 for authentication failures
- 403 for authorization failures
- 404 for not found resources
- 500 for internal server errors

## Recent Changes and Context

### September 19, 2024 - Auth Service Complete Implementation
- **COMPLETED**: Auth Service now fully implemented following the "winning formula" approach
- Added comprehensive production features: 25+ Prometheus metrics, rate limiting, circuit breakers
- Implemented complete HTTP API with 30+ endpoints matching gRPC functionality
- Created comprehensive testing framework with SQLite in-memory, mocks, and integration tests
- Updated documentation to reflect all implemented features and capabilities

### September 17, 2024 - Testing Infrastructure Improvements
- Fixed all service test compilation issues
- Enhanced Hanko client with nil-safe logging
- Standardized test patterns across all services
- Updated to Go 1.25 compatibility

### Key Test Fixes Applied
- Auth service: Fixed Hanko client test endpoints (`/webauthn/authentication/initialize`)
- Repository tests: Updated constructor calls (`NewAuthRepository` vs `NewRepository`)
- Service tests: Simplified mock usage and dependency injection
- Cross-service: Ensured all test files compile with `go test -c`

## Service-Specific Notes

### Auth Service ✅ COMPLETED
- **PRODUCTION READY**: Comprehensive implementation with enterprise-grade features
- **Passkey Authentication**: Full Hanko integration with WebAuthn/FIDO2 support
- **Complete APIs**: 30+ HTTP REST endpoints + 25+ gRPC services for all auth operations
- **Production Middleware**: Rate limiting, circuit breakers, comprehensive instrumentation
- **Advanced Monitoring**: 25+ Prometheus metrics covering auth, security, business, and performance
- **Comprehensive Testing**: SQLite in-memory tests, mocks, integration tests with full coverage
- **Multi-tenant Security**: JWT with club-based isolation and RBAC implementation
- **Audit & Compliance**: Complete audit logging for all authentication and authorization activities
- All Hanko client operations are nil-safe for logger
- Test endpoints match WebAuthn standards
- **Reference Implementation**: Use as template for completing other services

### Member Service ✅ COMPLETED
- **PRODUCTION READY**: Most complete service implementation
- Full CRUD operations with comprehensive testing
- **Reference Implementation**: Good reference for implementing other services

### Blockchain Service ✅ COMPLETED
- **PRODUCTION READY**: Full Hyperledger Fabric integration implemented
- Complete transaction recording and chaincode interaction
- Supports channels, chaincodes, endorsement, and event processing
- No longer uses mock implementation - production-ready

### Reciprocal Service ✅ COMPLETED
- **PRODUCTION READY**: Complete reciprocal agreements and visit management system
- Full agreement lifecycle (pending → approved → active)
- Complete visit workflow (request → confirm → check-in → check-out)
- Comprehensive business logic with proper validation and restrictions
- QR code generation for visit verification
- Complete test coverage (models, service, repository, handlers)
- Production-ready with both HTTP and gRPC interfaces

## Build and Deployment

### Container Images
- All services have Dockerfiles in their root directories
- Build from repository root: `podman build -f services/{service}/Dockerfile .`
- Use multi-stage builds for optimized production images

### Orchestration Options
- **Local Development**: Podman Compose (preferred) or Docker Compose
- **Kubernetes**: Manifests in `deployments/k8s/`
- **Podman Systemd**: Quadlets in `deployments/podman-quadlets/`

## Port Allocation
- API Gateway: 8080 (HTTP), 9080 (gRPC)
- Auth Service: 8081 (HTTP), 9081 (gRPC)
- Member Service: 8082 (HTTP), 9092 (gRPC)
- Reciprocal Service: 8083 (HTTP), 9093 (gRPC)
- Blockchain Service: 8084 (HTTP), 9094 (gRPC)
- Notification Service: 8085 (HTTP), 9095 (gRPC)
- Analytics Service: 8086 (HTTP), 9096 (gRPC)
- Governance Service: 8087 (HTTP), 9097 (gRPC)

## Critical Dependencies

### Infrastructure Services
- PostgreSQL (5432): Primary database
- NATS (4222): Message bus
- Redis (6379): Caching and sessions
- Prometheus (9090): Metrics collection

### External Integrations
- Hanko: Passkey authentication service
- Hyperledger Fabric: Blockchain network
- Email/SMS providers: For notifications

## Service Implementation Status

### ✅ COMPLETED SERVICES (Production Ready)
1. **Auth Service** - Complete authentication, authorization, and user management
2. **Member Service** - Complete member lifecycle management
3. **Blockchain Service** - Full Hyperledger Fabric integration
4. **Reciprocal Service** - Complete reciprocal agreements and visit management

### 🚧 REMAINING SERVICES (Need Implementation)
5. **API Gateway** - Service orchestration and routing
6. **Notification Service** - Email, SMS, push notifications
7. **Analytics Service** - Business intelligence and reporting
8. **Governance Service** - Club governance and voting

## Implementation Pattern - "Winning Formula"

For completing remaining services, follow this proven 6-step approach used successfully for Auth Service:

### Step 1: Analysis & Planning
- Analyze current implementation state
- Create comprehensive completion plan
- Identify gaps and requirements

### Step 2: Protocol Definitions
- Create/enhance Protocol Buffer definitions
- Generate Go code with `protoc`
- Define comprehensive gRPC services

### Step 3: Repository Layer
- Complete all database operations
- Implement tenant-aware queries
- Add comprehensive error handling

### Step 4: Testing Framework
- SQLite in-memory database testing
- Mock external dependencies with `mockery`
- Comprehensive unit and integration tests

### Step 5: Production Features
- Service-specific Prometheus metrics (20+ metrics recommended)
- Rate limiting middleware
- Circuit breakers for external services
- gRPC instrumentation and logging

### Step 6: Complete HTTP API
- 25+ comprehensive REST endpoints
- Full CRUD operations matching gRPC
- Middleware integration (auth, rate limiting, metrics)
- Admin monitoring endpoints

### Step 7: Documentation & Git
- Update README.md with all implemented features
- Commit each step with verbose git history
- Update CLAUDE.md with completion status

## Development Best Practices

### Testing Requirements
- All tests must compile: `go test -c ./internal/...`
- Use SQLite in-memory for fast database tests
- Mock external services with generated mocks
- Achieve high test coverage across all layers

### Production Monitoring
- Implement service-specific metrics (minimum 20+)
- Include business metrics alongside technical metrics
- Monitor rate limits, circuit breakers, and external service health
- Use correlation IDs for request tracing

### Security & Resilience
- Implement rate limiting for all public endpoints
- Use circuit breakers for external service calls
- Log all security events and audit activities
- Ensure multi-tenant data isolation