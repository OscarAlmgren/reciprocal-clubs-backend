# Member Service

The Member Service manages club memberships, member profiles, and membership-related operations within the Reciprocal Clubs platform.

## Overview

The Member Service provides comprehensive member management functionality including:

- **Member CRUD Operations**: Create, read, update, and delete member records
- **Profile Management**: Detailed member profiles with personal information and preferences
- **Membership Types**: Support for Regular, VIP, Corporate, Student, and Senior memberships
- **Status Management**: Active, suspended, expired, and pending member statuses
- **Access Validation**: Verify member access permissions for club facilities
- **Analytics**: Member statistics and reporting capabilities

## Architecture

The service follows a clean architecture pattern with the following layers:

```
├── cmd/                    # Application entry point
├── internal/
│   ├── models/            # Domain models and entities
│   ├── repository/        # Data access layer
│   ├── service/           # Business logic layer
│   ├── handlers/          # API handlers (HTTP & gRPC)
│   └── monitoring/        # Metrics and observability
└── proto/                 # Protocol buffer definitions
```

## Features

### Member Management
- Create new members with complete profiles
- Update member information and preferences
- Suspend/reactivate member accounts
- Delete member records (soft delete)
- Generate unique member numbers

### Profile Management
- Personal information (name, date of birth, phone)
- Address management
- Emergency contact information
- Communication preferences
- Notification settings

### Membership Types
- **Regular**: Standard membership with basic access
- **VIP**: Premium membership with enhanced privileges
- **Corporate**: Business membership for organizations
- **Student**: Discounted membership for students
- **Senior**: Special rates for senior citizens

### Member Status
- **Active**: Full access to club facilities
- **Suspended**: Temporarily restricted access
- **Expired**: Membership has expired
- **Pending**: Awaiting approval or payment

### Access Control
- Validate member access permissions
- Check membership status and validity
- Enforce club-specific access rules
- Real-time access verification

## API Endpoints

### HTTP REST API

#### Member Operations
```
POST   /api/v1/members                           # Create member
GET    /api/v1/members/{id}                      # Get member by ID
PUT    /api/v1/members/{id}                      # Update member profile
DELETE /api/v1/members/{id}                      # Delete member
POST   /api/v1/members/{id}/suspend              # Suspend member
POST   /api/v1/members/{id}/reactivate           # Reactivate member
```

#### Member Lookup
```
GET    /api/v1/members/by-user/{userId}          # Get by user ID
GET    /api/v1/members/by-number/{memberNumber}  # Get by member number
GET    /api/v1/clubs/{clubId}/members             # Get club members
```

#### Member Validation
```
GET    /api/v1/members/{id}/validate-access      # Validate access
GET    /api/v1/members/{id}/status               # Check status
```

#### Analytics
```
GET    /api/v1/clubs/{clubId}/analytics/members  # Member analytics
```

### gRPC API

The service provides a comprehensive gRPC API for high-performance inter-service communication:

```protobuf
service MemberService {
  rpc CreateMember(CreateMemberRequest) returns (CreateMemberResponse);
  rpc GetMember(GetMemberRequest) returns (GetMemberResponse);
  rpc GetMemberByUserID(GetMemberByUserIDRequest) returns (GetMemberResponse);
  rpc UpdateMemberProfile(UpdateMemberProfileRequest) returns (UpdateMemberProfileResponse);
  rpc SuspendMember(SuspendMemberRequest) returns (SuspendMemberResponse);
  rpc ValidateMemberAccess(ValidateMemberAccessRequest) returns (ValidateMemberAccessResponse);
  // ... additional methods
}
```

## Data Models

### Member
```go
type Member struct {
    ID                 uint           `json:"id"`
    ClubID             uint           `json:"club_id"`
    UserID             uint           `json:"user_id"`
    MemberNumber       string         `json:"member_number"`
    MembershipType     MembershipType `json:"membership_type"`
    Status             MemberStatus   `json:"status"`
    BlockchainIdentity string         `json:"blockchain_identity"`
    Profile            *MemberProfile `json:"profile"`
    JoinedAt           time.Time      `json:"joined_at"`
}
```

### Member Profile
```go
type MemberProfile struct {
    ID               uint                 `json:"id"`
    FirstName        string               `json:"first_name"`
    LastName         string               `json:"last_name"`
    DateOfBirth      *time.Time           `json:"date_of_birth"`
    PhoneNumber      string               `json:"phone_number"`
    Address          *Address             `json:"address"`
    EmergencyContact *EmergencyContact    `json:"emergency_contact"`
    Preferences      *MemberPreferences   `json:"preferences"`
}
```

## Database Schema

The service uses PostgreSQL with the following main tables:

- `members` - Core member information
- `member_profiles` - Detailed member profiles
- `addresses` - Member addresses
- `emergency_contacts` - Emergency contact information
- `member_preferences` - Communication preferences

## Configuration

### Environment Variables

```bash
# Service configuration
MEMBER_SERVICE_SERVICE_PORT=8082
MEMBER_SERVICE_SERVICE_GRPC_PORT=9092
MEMBER_SERVICE_SERVICE_ENVIRONMENT=development

# Database configuration
MEMBER_SERVICE_DATABASE_HOST=localhost
MEMBER_SERVICE_DATABASE_PORT=5432
MEMBER_SERVICE_DATABASE_NAME=member_service
MEMBER_SERVICE_DATABASE_USER=postgres
MEMBER_SERVICE_DATABASE_PASSWORD=postgres

# NATS configuration
MEMBER_SERVICE_NATS_URL=nats://localhost:4222

# Redis configuration (optional)
MEMBER_SERVICE_REDIS_HOST=localhost
MEMBER_SERVICE_REDIS_PORT=6379

# Monitoring configuration
MEMBER_SERVICE_MONITORING_ENABLED=true
MEMBER_SERVICE_MONITORING_PORT=2112
```

## Metrics and Monitoring

The service provides comprehensive metrics via Prometheus:

### HTTP Metrics
- `member_service_http_requests_total` - Total HTTP requests
- `member_service_http_request_duration_seconds` - HTTP request duration
- `member_service_http_errors_total` - HTTP errors

### gRPC Metrics
- `member_service_grpc_requests_total` - Total gRPC requests
- `member_service_grpc_request_duration_seconds` - gRPC request duration
- `member_service_grpc_errors_total` - gRPC errors

### Business Metrics
- `member_service_members_created_total` - Members created
- `member_service_members_updated_total` - Members updated
- `member_service_members_suspended_total` - Members suspended
- `member_service_active_members_total` - Active members count
- `member_service_members_by_status` - Members by status
- `member_service_members_by_type` - Members by type

### Database Metrics
- `member_service_database_queries_total` - Database queries
- `member_service_database_query_duration_seconds` - Query duration
- `member_service_database_errors_total` - Database errors

## Health Checks

The service provides multiple health check endpoints:

- `GET /health` - Overall service health
- `GET /ready` - Readiness probe for Kubernetes
- `gRPC HealthCheck` - gRPC health check service

## Events and Messaging

The service publishes events to NATS for integration with other services:

- `member.created` - New member created
- `member.updated` - Member profile updated
- `member.suspended` - Member suspended
- `member.reactivated` - Member reactivated
- `member.deleted` - Member deleted

## Testing

### Unit Tests
```bash
cd services/member-service
go test ./internal/...
```

### Integration Tests
```bash
cd services/member-service
go test -tags=integration ./...
```

### Load Testing
```bash
# Example using hey
hey -n 1000 -c 10 http://localhost:8082/api/v1/members/1
```

## Development

### Local Development
```bash
# Start dependencies (PostgreSQL, NATS)
docker-compose up -d postgres nats

# Run the service
cd services/member-service
go run cmd/main.go
```

### Building
```bash
# Build binary
cd services/member-service
go build -o member-service cmd/main.go

# Build Docker image
docker build -t reciprocal-member-service:latest .
```

### Code Generation
```bash
# Generate gRPC code from proto files
cd services/member-service
protoc --go_out=. --go-grpc_out=. proto/member.proto
```

## Integration

### API Gateway Integration
The Member Service integrates with the API Gateway through:
- GraphQL resolvers for `createMember`, `updateMember` mutations
- Member queries and profile management
- Access validation for protected operations

### Other Service Dependencies
- **Auth Service**: User authentication and authorization
- **Notification Service**: Member communication
- **Analytics Service**: Member statistics and reporting
- **Blockchain Service**: Member identity on blockchain

## Security

### Authentication
- JWT token validation for HTTP endpoints
- gRPC metadata authentication for service-to-service calls

### Authorization
- Role-based access control
- Club-specific permissions
- Member data privacy protection

### Data Protection
- Sensitive data encryption at rest
- Secure communication with TLS
- PII data handling compliance

## Deployment

### Docker
```bash
docker run -p 8082:8082 -p 9092:9092 \
  -e MEMBER_SERVICE_DATABASE_HOST=postgres \
  -e MEMBER_SERVICE_NATS_URL=nats://nats:4222 \
  reciprocal-member-service:latest
```

### Kubernetes
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: member-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: member-service
  template:
    spec:
      containers:
      - name: member-service
        image: reciprocal-member-service:latest
        ports:
        - containerPort: 8082
        - containerPort: 9092
        env:
        - name: MEMBER_SERVICE_DATABASE_HOST
          value: "postgres"
```

## Performance

### Benchmarks
- HTTP API: ~1000 requests/second
- gRPC API: ~2000 requests/second
- Database operations: <10ms average
- Memory usage: ~50MB typical

### Optimization
- Database connection pooling
- gRPC connection reuse
- Efficient query patterns
- Caching for read-heavy operations

## Troubleshooting

### Common Issues

1. **Database Connection Errors**
   - Check PostgreSQL connectivity
   - Verify credentials and permissions
   - Ensure database exists

2. **NATS Connection Issues**
   - Verify NATS server is running
   - Check network connectivity
   - Validate NATS URL configuration

3. **High Memory Usage**
   - Monitor database connection pools
   - Check for goroutine leaks
   - Review caching strategies

### Logging
The service uses structured logging with configurable levels:
- `ERROR` - Service errors and failures
- `WARN` - Performance and security warnings
- `INFO` - Operational events
- `DEBUG` - Detailed debugging information

## License

This service is part of the Reciprocal Clubs platform. See the main repository LICENSE file for details.