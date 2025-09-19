# Auth Service

🚀 **PRODUCTION READY** - A comprehensive microservice providing enterprise-grade authentication and authorization functionality for the Reciprocal Clubs platform.

## Features

- **🔐 Passkey Authentication**: WebAuthn/FIDO2 passwordless authentication via Hanko integration
- **📱 Multi-Factor Authentication (MFA)**: TOTP, SMS, Email, and backup codes for enhanced security
- **🔑 Password Management**: Secure password reset, strength validation, and breach detection
- **✉️ Email Verification**: Token-based email verification for account security
- **👥 User Management**: Complete user registration, profile management, and role-based access control
- **🎫 Session Management**: Secure JWT session handling with configurable expiration
- **🏢 Multi-tenant Support**: Club-based isolation for users and permissions
- **⚡ Event-driven Architecture**: NATS integration for publishing authentication events
- **📋 Comprehensive Audit Logging**: Track all authentication and authorization activities
- **📊 Health Monitoring**: Advanced health checks, metrics, and observability

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

- Go 1.25+
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
- `POST /auth/passkey/register/complete` - Complete additional passkey registration

### Multi-Factor Authentication (MFA)

- `POST /auth/mfa/setup` - Setup MFA with TOTP authenticator app
- `POST /auth/mfa/verify` - Verify MFA code (TOTP, SMS, Email, or backup)
- `POST /auth/mfa/disable` - Disable MFA for user account
- `GET /auth/mfa/backup-codes` - Generate new backup codes
- `POST /auth/mfa/sms/send` - Send SMS verification code
- `POST /auth/mfa/email/send` - Send email verification code

### Password Management

- `POST /auth/password/reset/request` - Request password reset via email
- `POST /auth/password/reset/confirm` - Confirm password reset with token
- `POST /auth/password/change` - Change password (authenticated users)
- `GET /auth/password/strength` - Check password strength score

### Email Verification

- `POST /auth/email/verify/send` - Send email verification token
- `POST /auth/email/verify/confirm` - Confirm email verification with token

### User Management (Authentication Required)

- `GET /users/{clubId}/{userId}` - Get user details with roles and permissions
- `PUT /users/{clubId}/{userId}` - Update user profile
- `DELETE /users/{clubId}/{userId}` - Delete user account
- `POST /users/{clubId}/{userId}/suspend` - Suspend user account
- `POST /users/{clubId}/{userId}/activate` - Activate suspended user account
- `GET /users/{clubId}/{userId}/permissions` - Get user permissions

### Role Management (Admin Required)

- `POST /roles` - Create new role
- `GET /roles/{clubId}` - List all roles for club
- `GET /roles/{clubId}/{roleId}` - Get specific role details
- `PUT /roles/{clubId}/{roleId}` - Update role details
- `DELETE /roles/{clubId}/{roleId}` - Delete role
- `POST /roles/{clubId}/assign` - Assign role to user
- `DELETE /roles/{clubId}/remove` - Remove role from user

### Permission Management (Admin Required)

- `POST /permissions/check` - Check user permissions for specific action
- `GET /permissions/{clubId}` - List all available permissions for club

### Club Management

- `POST /clubs` - Create new club
- `GET /clubs/{clubId}` - Get club details
- `PUT /clubs/{clubId}` - Update club details
- `POST /clubs/{clubId}/join` - Join club (with invitation code)
- `POST /clubs/{clubId}/leave` - Leave club
- `GET /clubs/{clubId}/members` - List club members

### Audit & Compliance (Admin Required)

- `GET /audit/{clubId}` - Get audit logs for club
- `GET /audit/{clubId}/user/{userId}` - Get audit logs for specific user

### Admin & Monitoring (Admin Required)

- `GET /admin/rate-limits` - Get current rate limit status
- `GET /admin/circuit-breakers` - Get circuit breaker status
- `POST /admin/circuit-breakers/{name}/reset` - Reset circuit breaker

### Webhooks

- `POST /webhook/hanko` - Handle Hanko webhook events

## gRPC Services

The service also exposes comprehensive gRPC endpoints for inter-service communication:

### Authentication Services
- `RegisterUser` - User registration with passkey
- `InitiatePasskeyLogin` - Start passkey authentication flow
- `CompletePasskeyLogin` - Complete passkey authentication
- `InitiatePasskeyRegistration` - Start additional passkey registration
- `CompletePasskeyRegistration` - Complete additional passkey registration
- `ValidateSession` - Session token validation
- `Logout` - User logout and session invalidation

### Multi-Factor Authentication Services
- `SetupMFA` - Setup MFA with TOTP authenticator app
- `VerifyMFA` - Verify MFA codes (TOTP, SMS, Email, backup)
- `DisableMFA` - Disable MFA for user account
- `GenerateBackupCodes` - Generate new MFA backup codes
- `SendSMSCode` - Send SMS verification code
- `SendEmailCode` - Send email verification code

### Password Management Services
- `RequestPasswordReset` - Request password reset via email
- `ConfirmPasswordReset` - Confirm password reset with token
- `ChangePassword` - Change user password (authenticated)
- `CheckPasswordStrength` - Validate password strength

### Email Verification Services
- `SendEmailVerification` - Send email verification token
- `ConfirmEmailVerification` - Confirm email verification

### User Management Services
- `GetUserWithRoles` - Get user data with roles and permissions
- `UpdateUser` - Update user profile information
- `DeleteUser` - Delete user account
- `SuspendUser` - Suspend user account
- `ActivateUser` - Activate suspended user account
- `GetUserPermissions` - Get user's effective permissions

### Role Management Services
- `CreateRole` - Create new role
- `GetRole` - Get role details
- `UpdateRole` - Update role information
- `DeleteRole` - Delete role
- `ListRoles` - List all roles for a club
- `AssignRole` - Assign role to user
- `RemoveRole` - Remove role from user

### Permission Services
- `CheckPermission` - Check if user has specific permission
- `ListPermissions` - List all available permissions

### Club Management Services
- `CreateClub` - Create new club
- `GetClub` - Get club details
- `UpdateClub` - Update club information
- `JoinClub` - Join club with invitation
- `LeaveClub` - Leave club
- `GetClubMembers` - Get club member list

### Audit & Compliance Services
- `GetAuditLogs` - Retrieve audit logs with filtering
- `CreateAuditLog` - Create audit log entry

### System Services
- `HealthCheck` - Service health check
- `GetMetrics` - Service metrics and statistics

## API Examples

### Complete MFA Setup Flow

```bash
# 1. Setup MFA for a user (returns QR code URL and secret)
curl -X POST http://localhost:8080/auth/mfa/setup \
  -H "Authorization: Bearer <session_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "club_id": "club-123",
    "user_id": "user-456"
  }'

# Response:
{
  "secret": "JBSWY3DPEHPK3PXP",
  "qr_code_url": "otpauth://totp/Reciprocal%20Clubs:user@example.com?secret=JBSWY3DPEHPK3PXP&issuer=Reciprocal%20Clubs",
  "backup_codes": ["ABCD-EFGH", "IJKL-MNOP", "QRST-UVWX"]
}

# 2. Verify MFA setup with TOTP code
curl -X POST http://localhost:8080/auth/mfa/verify \
  -H "Authorization: Bearer <session_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "club_id": "club-123",
    "user_id": "user-456",
    "code": "123456",
    "code_type": "totp"
  }'
```

### Password Reset Flow

```bash
# 1. Request password reset
curl -X POST http://localhost:8080/auth/password/reset/request \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "club_id": "club-123"
  }'

# 2. Confirm password reset with token from email
curl -X POST http://localhost:8080/auth/password/reset/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "token": "reset-token-from-email",
    "new_password": "NewSecurePassword123!",
    "club_id": "club-123"
  }'
```

### Email Verification Flow

```bash
# 1. Send email verification
curl -X POST http://localhost:8080/auth/email/verify/send \
  -H "Authorization: Bearer <session_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "club_id": "club-123",
    "user_id": "user-456"
  }'

# 2. Confirm email verification
curl -X POST http://localhost:8080/auth/email/verify/confirm \
  -H "Content-Type: application/json" \
  -d '{
    "token": "verification-token-from-email",
    "club_id": "club-123"
  }'
```

### MFA Verification During Login

```bash
# Login with MFA verification
curl -X POST http://localhost:8080/auth/mfa/verify \
  -H "Content-Type: application/json" \
  -d '{
    "club_id": "club-123",
    "user_id": "user-456",
    "code": "123456",
    "code_type": "totp"
  }'

# Or use backup code
curl -X POST http://localhost:8080/auth/mfa/verify \
  -H "Content-Type: application/json" \
  -d '{
    "club_id": "club-123",
    "user_id": "user-456",
    "code": "ABCD-EFGH",
    "code_type": "backup"
  }'
```

## Configuration

The service supports configuration through:

1. **YAML file**: `config.yaml` in the root directory
2. **Environment variables**: Prefixed with `AUTH_SERVICE_`
3. **Command-line flags**: Via Viper integration

See `config.yaml` for all available configuration options.

### MFA Configuration

Additional environment variables for MFA features:

```bash
# MFA Settings
AUTH_SERVICE_MFA_ISSUER=Reciprocal Clubs
AUTH_SERVICE_MFA_TOTP_SKEW=1
AUTH_SERVICE_MFA_BACKUP_CODES_COUNT=8

# SMS Provider (optional)
AUTH_SERVICE_SMS_PROVIDER=twilio
AUTH_SERVICE_SMS_API_KEY=your-sms-api-key
AUTH_SERVICE_SMS_FROM_NUMBER=+1234567890

# Email Provider (optional)
AUTH_SERVICE_EMAIL_PROVIDER=sendgrid
AUTH_SERVICE_EMAIL_API_KEY=your-email-api-key
AUTH_SERVICE_EMAIL_FROM_ADDRESS=noreply@reciprocalclubs.com
```

## Database Migrations

The service uses GORM AutoMigrate to handle database schema changes. Migrations run automatically on startup when `database.auto_migrate` is enabled.

### Models

- `User` - User accounts with Hanko, MFA, and password reset integration
- `Club` - Multi-tenant organizations
- `Role` - User roles within clubs
- `Permission` - Granular permissions for actions
- `UserRole` - Many-to-many relationship between users and roles
- `RolePermission` - Many-to-many relationship between roles and permissions
- `UserSession` - Active user sessions
- `MFAToken` - MFA verification tokens (SMS, Email)
- `AuditLog` - Comprehensive audit trail for all actions

## Event Publishing

The service publishes events to NATS for:

- User registration and email verification
- User login/logout and session management
- MFA setup, verification, and security events
- Password reset requests and confirmations
- Role assignments and permission changes
- Security events and audit trail

Events are published to subjects:
- `users.events` - User lifecycle events
- `auth.events` - Authentication events
- `audit.logs` - Audit log events

## Security Features

- **🔐 Passkey Authentication**: FIDO2/WebAuthn for passwordless authentication
- **📱 Multi-Factor Authentication**: TOTP, SMS, Email, and backup codes
- **🔑 Advanced Password Security**: bcrypt hashing, strength validation, breach detection
- **✉️ Email Verification**: Secure token-based email confirmation
- **🎫 Session Management**: Secure JWT session tokens with configurable expiration
- **🛡️ Rate Limiting**: Advanced protection against brute force attacks
- **📋 Comprehensive Audit Logging**: Real-time logging of all security events
- **🔒 RBAC**: Role-based access control with granular permissions
- **🏢 Multi-tenant Isolation**: Club-based data segregation and security
- **⚡ Circuit Breakers**: Protection against external service failures
- **🔍 Security Monitoring**: Real-time threat detection and alerting

## Monitoring & Observability

### Metrics

The service exposes comprehensive Prometheus metrics on `/metrics`:

#### Standard Metrics
- HTTP request counts and durations by endpoint and status
- gRPC request counts and durations by method and status
- Database connection pool metrics and query performance
- NATS message counts and processing times
- Go runtime metrics (memory, goroutines, GC)

#### Auth Service Specific Metrics
- **Authentication Metrics**: Login attempts, successes, failures by method and club
- **MFA Metrics**: Setup rates, verification attempts, backup code usage, method preferences
- **Password Security**: Reset requests, strength scores, breach detection hits
- **Email Verification**: Verification rates, bounce rates, delivery success
- **User Management**: Registration counts, active users, suspended users
- **Passkey Operations**: Initiation/completion rates, success rates, duration
- **Security Events**: Failed logins, suspicious activities, rate limit hits, MFA bypass attempts
- **Role & Permission**: Role assignments, permission checks, authorization failures
- **External Service**: Hanko API calls, response times, error rates
- **Session Management**: Active sessions, session duration, logout events
- **Business Metrics**: Club activity, member growth, engagement metrics
- **Performance**: Concurrent requests, request queue depth, circuit breaker status

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
│   │   ├── grpc.go            # gRPC service implementation
│   │   ├── grpc_types.go      # gRPC type conversions
│   │   └── http.go            # HTTP REST API handlers
│   ├── hanko/                  # Hanko client integration
│   ├── mfa/                    # Multi-Factor Authentication service
│   │   ├── mfa.go             # TOTP, SMS, Email, backup codes
│   │   └── mfa_test.go        # Comprehensive MFA tests
│   ├── password/               # Password management service
│   │   ├── password.go        # Hashing, validation, reset tokens
│   │   └── password_test.go   # Password security tests
│   ├── metrics/                # Auth-specific Prometheus metrics
│   │   └── auth_metrics.go    # 25+ auth service metrics
│   ├── middleware/             # Production middleware
│   │   ├── circuitbreaker.go  # Circuit breaker for external services
│   │   ├── instrumentation.go # gRPC instrumentation & logging
│   │   └── ratelimit.go       # Rate limiting protection
│   ├── models/                 # Database models with MFA/password fields
│   ├── repository/             # Data access layer with MFA/email support
│   ├── service/                # Business logic layer with complete auth
│   └── testutil/               # Testing utilities and mocks
├── proto/                      # Protocol Buffer definitions
│   └── auth.proto             # gRPC service definitions
├── scripts/                    # Database and build scripts
├── config.yaml                 # Configuration file
├── docker-compose.yml          # Docker composition
├── Dockerfile                  # Container definition
└── README.md                   # This file
```

### Testing

The service includes comprehensive testing with multiple test types:

```bash
# Run all unit tests
go test ./...

# Run with coverage report
go test -cover ./...

# Run specific package tests
go test ./internal/service/
go test ./internal/repository/
go test ./internal/handlers/

# Run integration tests (requires dependencies)
go test -tags=integration ./...

# Run tests with verbose output
go test -v ./...

# Generate coverage HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

#### Test Coverage

The service includes comprehensive test coverage for:

- **Repository Layer**: Database operations with SQLite in-memory testing
- **Service Layer**: Business logic with mocked dependencies including MFA and password services
- **MFA Layer**: Complete TOTP, SMS, Email, and backup code testing with security validation
- **Password Layer**: Password hashing, strength validation, reset workflow, and breach detection
- **Handler Layer**: HTTP and gRPC endpoints with test servers including new auth features
- **Integration Tests**: End-to-end testing with real dependencies and complete auth flows
- **Security Testing**: Comprehensive testing of all security features and edge cases
- **Mock Testing**: Generated mocks for external services (Hanko, NATS, MFA providers)

#### Test Utilities

- **Mocks**: Auto-generated mocks using `mockery` for external dependencies
- **Test Database**: SQLite in-memory database for fast, isolated testing
- **Test Fixtures**: Realistic test data for clubs, users, roles, and permissions
- **Integration Helpers**: Docker Compose setup for integration testing

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
   - Use proper TLS certificates for HTTPS/gRPC
   - Configure proper CORS settings for frontend integration
   - Set strong secret keys (minimum 32 characters)
   - Enable rate limiting (configured per endpoint)
   - Monitor rate limit violations and security events
   - Regular security audits and vulnerability scanning

2. **Scalability**:
   - Use connection pooling for database connections
   - Configure proper resource limits (CPU, memory)
   - Set up horizontal pod autoscaling based on metrics
   - Implement circuit breakers for external service resilience
   - Monitor concurrent request limits and performance

3. **Monitoring**:
   - Set up Prometheus monitoring with 25+ auth-specific metrics
   - Configure alerting rules for critical issues
   - Enable distributed tracing with correlation IDs
   - Monitor rate limit violations and circuit breaker trips
   - Set up dashboards for business and operational metrics

4. **Backup & Recovery**:
   - Regular database backups with point-in-time recovery
   - Test disaster recovery procedures regularly
   - Monitor backup success and retention policies
   - Document recovery procedures and RTO/RPO requirements

5. **Performance**:
   - Monitor response times and set SLA targets
   - Use connection pooling and proper database indexing
   - Implement caching where appropriate
   - Monitor memory usage and garbage collection metrics

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