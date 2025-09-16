# Auth Service

A microservice that provides authentication and authorization functionality for the Reciprocal Clubs platform using Hanko passkey integration.

## Features

- **Passkey Authentication**: WebAuthn/FIDO2 authentication via Hanko integration
- **User Management**: User registration, profile management, and role-based access control
- **Session Management**: Secure session handling with configurable expiration
- **Multi-tenant Support**: Club-based isolation for users and permissions
- **Event-driven Architecture**: NATS integration for publishing authentication events
- **Comprehensive Audit Logging**: Track all authentication and authorization activities
- **Health Monitoring**: Health checks, metrics, and observability

## Architecture

The Auth Service is built with a clean architecture pattern:

```
┌─────────────────────────────────────────────────────────────────┐
│                           Handlers                              │
│  ┌─────────────────┐              ┌─────────────────┐         │
│  │   HTTP REST     │              │      gRPC       │         │
│  └─────────────────┘              └─────────────────┘         │
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                         Service Layer                           │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │  Authentication  │  Authorization  │  User Management     ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────┐
│                        Repository Layer                         │
│  ┌─────────────────────────────────────────────────────────────┐│
│  │     PostgreSQL GORM     │     NATS Events    │   Hanko    ││
│  └─────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────┘
```

## Dependencies

- **PostgreSQL**: Primary database for user data, roles, and sessions
- **Hanko**: External authentication service for passkey management
- **NATS**: Message broker for event-driven architecture
- **Prometheus**: Metrics collection and monitoring

## Getting Started

### Prerequisites

- Go 1.21+
- Docker and Docker Compose
- PostgreSQL 15+
- NATS 2.10+

### Environment Variables

Create a `.env` file or set the following environment variables:

```bash
# Required
AUTH_TOKEN_SECRET_KEY=your-secret-key-at-least-32-characters-long
HANKO_API_KEY=your-hanko-api-key
HANKO_PROJECT_ID=your-hanko-project-id

# Optional (defaults provided)
AUTH_SERVICE_SERVER_ENVIRONMENT=development
AUTH_SERVICE_DATABASE_HOST=localhost
AUTH_SERVICE_DATABASE_PORT=5432
AUTH_SERVICE_DATABASE_USERNAME=postgres
AUTH_SERVICE_DATABASE_PASSWORD=postgres
AUTH_SERVICE_DATABASE_DATABASE=auth_service
AUTH_SERVICE_HANKO_URL=http://localhost:8000
AUTH_SERVICE_NATS_URL=nats://localhost:4222
```

### Running with Docker Compose

The easiest way to run the Auth Service with all its dependencies:

```bash
# Start all services
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f auth-service

# Stop all services
docker-compose down
```

### Running Locally

1. **Start dependencies**:
   ```bash
   # Start only the dependencies
   docker-compose up -d postgres nats hanko hanko-postgres
   ```

2. **Install dependencies**:
   ```bash
   go mod download
   ```

3. **Run the service**:
   ```bash
   go run cmd/main.go
   ```

## API Endpoints

### Health & Monitoring

- `GET /health` - Service health check
- `GET /ready` - Service readiness check
- `GET /metrics` - Prometheus metrics

### Authentication

- `POST /auth/register` - Register new user with passkey
- `POST /auth/login/initiate` - Initiate passkey login
- `POST /auth/login/complete` - Complete passkey login
- `POST /auth/logout` - Logout and invalidate session
- `POST /auth/session/validate` - Validate session token
- `POST /auth/passkey/register/initiate` - Register additional passkey

### User Management

- `GET /users/{clubId}/{userId}` - Get user with roles and permissions

### Webhooks

- `POST /webhook/hanko` - Handle Hanko webhook events

## gRPC Services

The service also exposes gRPC endpoints for inter-service communication:

- `RegisterUser` - User registration
- `InitiatePasskeyLogin` - Start passkey authentication
- `CompletePasskeyLogin` - Complete passkey authentication
- `ValidateSession` - Session validation
- `GetUserWithRoles` - User data with permissions
- `HealthCheck` - Service health

## Configuration

The service supports configuration through:

1. **YAML file**: `config.yaml` in the root directory
2. **Environment variables**: Prefixed with `AUTH_SERVICE_`
3. **Command-line flags**: Via Viper integration

See `config.yaml` for all available configuration options.

## Database Migrations

The service uses GORM AutoMigrate to handle database schema changes. Migrations run automatically on startup when `database.auto_migrate` is enabled.

### Models

- `User` - User accounts with Hanko integration
- `Club` - Multi-tenant organizations
- `Role` - User roles within clubs
- `Permission` - Granular permissions for actions
- `UserRole` - Many-to-many relationship between users and roles
- `RolePermission` - Many-to-many relationship between roles and permissions
- `UserSession` - Active user sessions
- `AuditLog` - Audit trail for all actions

## Event Publishing

The service publishes events to NATS for:

- User registration
- User login/logout
- Role assignments
- Permission changes
- Security events

Events are published to subjects:
- `users.events` - User lifecycle events
- `auth.events` - Authentication events
- `audit.logs` - Audit log events

## Security Features

- **Passkey Authentication**: FIDO2/WebAuthn for passwordless authentication
- **Session Management**: Secure session tokens with configurable expiration
- **Rate Limiting**: Protection against brute force attacks
- **Audit Logging**: Comprehensive logging of all security events
- **RBAC**: Role-based access control with granular permissions
- **Multi-tenant Isolation**: Club-based data segregation

## Monitoring & Observability

### Metrics

The service exposes Prometheus metrics on `/metrics`:

- HTTP request counts and durations
- Database connection pool metrics
- NATS message counts
- Custom business metrics
- Go runtime metrics

### Health Checks

- **Liveness**: `/health` - Service is running
- **Readiness**: `/ready` - Service can handle requests

### Logging

Structured logging with configurable levels and formats:
- JSON format for production
- Human-readable format for development
- Contextual logging with request IDs
- Error tracking and alerting

## Development

### Project Structure

```
auth-service/
├── cmd/
│   └── main.go                 # Application entrypoint
├── internal/
│   ├── config/                 # Configuration management
│   ├── handlers/               # HTTP & gRPC handlers
│   ├── hanko/                  # Hanko client integration
│   ├── models/                 # Database models
│   ├── repository/             # Data access layer
│   └── service/                # Business logic layer
├── scripts/                    # Database scripts
├── config.yaml                 # Configuration file
├── docker-compose.yml          # Docker composition
├── Dockerfile                  # Container definition
└── README.md                   # This file
```

### Testing

```bash
# Run unit tests
go test ./...

# Run with coverage
go test -cover ./...

# Run integration tests (requires dependencies)
go test -tags=integration ./...
```

### Building

```bash
# Build locally
go build -o auth-service cmd/main.go

# Build Docker image
docker build -t auth-service .
```

## Deployment

### Kubernetes

Kubernetes manifests are available in the `k8s/` directory:

```bash
kubectl apply -f k8s/
```

### Production Considerations

1. **Security**:
   - Use proper TLS certificates
   - Configure proper CORS settings
   - Set strong secret keys
   - Enable rate limiting

2. **Scalability**:
   - Use connection pooling
   - Configure proper resource limits
   - Set up horizontal pod autoscaling

3. **Monitoring**:
   - Set up Prometheus monitoring
   - Configure alerting rules
   - Enable distributed tracing

4. **Backup**:
   - Regular database backups
   - Test disaster recovery procedures

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Support

For support, please contact the development team or create an issue in the repository.