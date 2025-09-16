# API Gateway Service

The API Gateway is the main entry point for the reciprocal clubs backend system, providing a unified GraphQL interface and REST API for all client applications.

## Features

- **GraphQL API**: Schema-first development with gqlgen
- **REST API**: Traditional REST endpoints for authentication and quick operations
- **Authentication**: JWT-based authentication with multi-tenant support
- **Rate Limiting**: Per-user and per-IP rate limiting
- **Service Discovery**: Dynamic service client management
- **WebSocket Support**: Real-time subscriptions via GraphQL subscriptions
- **CORS Support**: Cross-origin resource sharing for web clients
- **Request Tracing**: Correlation ID tracking across services
- **Metrics & Monitoring**: Comprehensive Prometheus metrics
- **Health Checks**: Liveness and readiness probes

## Architecture

```
┌─────────────────┐
│   Web Client    │
│   Mobile App    │
│   Admin Panel   │
└─────────┬───────┘
          │ HTTP/GraphQL/WS
┌─────────▼───────┐
│  API Gateway    │
│                 │
│ ┌─────────────┐ │
│ │   GraphQL   │ │
│ │   Server    │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ REST Endpoints│ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Middleware  │ │
│ │ - Auth      │ │
│ │ - Rate Limit│ │
│ │ - Metrics   │ │
│ │ - Logging   │ │
│ └─────────────┘ │
└─────────┬───────┘
          │ gRPC
    ┌─────▼─────┐
    │ Services  │
    │ - Auth    │
    │ - Member  │
    │ - Recipr. │
    │ - Blockchain │
    │ - Notify  │
    │ - Analytics │
    │ - Governance │
    └───────────┘
```

## Getting Started

### Prerequisites

- Go 1.21+
- gqlgen CLI: `go install github.com/99designs/gqlgen@latest`
- NATS server running
- Backend services running (for full functionality)

### Development Setup

1. **Generate GraphQL code**:
   ```bash
   cd services/api-gateway
   go run github.com/99designs/gqlgen generate
   ```

2. **Install dependencies**:
   ```bash
   go mod tidy
   ```

3. **Run the service**:
   ```bash
   go run cmd/main.go
   ```

4. **Access the API**:
   - GraphQL endpoint: `http://localhost:8080/graphql`
   - GraphQL playground: `http://localhost:8080/playground`
   - REST API: `http://localhost:8080/api/v1`
   - Health check: `http://localhost:8080/health`
   - Metrics: `http://localhost:2112/metrics`

### Configuration

The API Gateway can be configured via environment variables or YAML files:

```yaml
service:
  name: api-gateway
  host: 0.0.0.0
  port: 8080
  grpc_port: 9090
  environment: development

database:
  host: localhost
  port: 5432
  user: postgres
  password: postgres
  database: reciprocal_clubs

nats:
  url: nats://localhost:4222
  cluster_id: reciprocal-clubs
  client_id: api-gateway

auth:
  jwt_secret: your-secret-key
  jwt_expiration: 3600
  issuer: reciprocal-clubs
  audience: reciprocal-clubs

monitoring:
  enable_metrics: true
  metrics_port: 2112
  metrics_path: /metrics
```

### Environment Variables

Key environment variables (with `API_GATEWAY_` prefix):

- `API_GATEWAY_SERVICE_PORT`: HTTP port (default: 8080)
- `API_GATEWAY_DATABASE_HOST`: PostgreSQL host
- `API_GATEWAY_DATABASE_PASSWORD`: PostgreSQL password
- `API_GATEWAY_NATS_URL`: NATS server URL
- `API_GATEWAY_AUTH_JWT_SECRET`: JWT signing secret
- `API_GATEWAY_LOGGING_LEVEL`: Log level (debug, info, warn, error)

## API Documentation

### GraphQL API

The GraphQL API provides a comprehensive interface for all operations:

**Key Types:**
- `User`: Authentication and user management
- `Member`: Club member information and profiles
- `ReciprocalAgreement`: Inter-club agreements
- `Visit`: Visit tracking and verification
- `Notification`: Multi-channel notifications
- `Transaction`: Blockchain transaction records
- `Analytics`: Usage statistics and reporting
- `Proposal`: Governance proposals and voting

**Sample Queries:**

```graphql
# Get current user
query Me {
  me {
    id
    email
    username
    roles
    club {
      id
      name
    }
  }
}

# List members with pagination
query ListMembers($pagination: PaginationInput) {
  members(pagination: $pagination) {
    nodes {
      id
      memberNumber
      status
      profile {
        firstName
        lastName
      }
    }
    pageInfo {
      total
      hasNextPage
    }
  }
}

# Get visit analytics
query VisitAnalytics {
  analytics {
    visits {
      totalVisits
      monthlyVisits {
        month
        count
      }
      topDestinations {
        club {
          name
        }
        count
      }
    }
  }
}
```

**Sample Mutations:**

```graphql
# Login
mutation Login($input: LoginInput!) {
  login(input: $input) {
    token
    refreshToken
    user {
      id
      email
    }
    expiresAt
  }
}

# Create member
mutation CreateMember($input: CreateMemberInput!) {
  createMember(input: $input) {
    id
    memberNumber
    status
  }
}

# Record visit
mutation RecordVisit($input: RecordVisitInput!) {
  recordVisit(input: $input) {
    id
    checkInTime
    status
  }
}
```

**Sample Subscriptions:**

```graphql
# Real-time notifications
subscription NotificationReceived {
  notificationReceived {
    id
    title
    message
    type
    createdAt
  }
}

# Visit status updates
subscription VisitStatusChanged($clubId: ID) {
  visitStatusChanged(clubId: $clubId) {
    id
    status
    checkInTime
    checkOutTime
  }
}
```

### REST API

The REST API provides traditional HTTP endpoints for common operations:

#### Authentication Endpoints

- `POST /api/v1/auth/login`: User login
- `POST /api/v1/auth/register`: User registration
- `POST /api/v1/auth/refresh`: Token refresh
- `POST /api/v1/auth/logout`: User logout (authenticated)
- `GET /api/v1/auth/me`: Get current user (authenticated)

#### System Endpoints

- `GET /api/v1/status`: Service status (authenticated)
- `GET /health`: Health check
- `GET /ready`: Readiness check

#### Example Requests

**Login:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "password123"
  }'
```

**Get User Info:**
```bash
curl -X GET http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer <jwt_token>"
```

## Architecture Details

### Middleware Stack

1. **CORS**: Cross-origin resource sharing
2. **Request ID**: Unique request tracking
3. **Logging**: Structured request/response logging
4. **Metrics**: Prometheus metrics collection
5. **Rate Limiting**: Request rate limiting per user/IP
6. **Authentication**: JWT token validation (for protected routes)

### Service Clients

The API Gateway maintains gRPC connections to all backend services:

- **Auth Service**: User authentication and authorization
- **Member Service**: Member management operations
- **Reciprocal Service**: Reciprocal agreement handling
- **Blockchain Service**: Blockchain transaction management
- **Notification Service**: Multi-channel notifications
- **Analytics Service**: Usage analytics and reporting
- **Governance Service**: Network governance and voting

### GraphQL Resolvers

Resolvers are organized by domain and delegate to appropriate service clients:

- **Query Resolvers**: Read operations across all services
- **Mutation Resolvers**: Write operations with proper authorization
- **Subscription Resolvers**: Real-time updates via WebSocket
- **Field Resolvers**: Lazy loading of related data

### Error Handling

- **GraphQL Errors**: Structured error responses with error codes
- **HTTP Errors**: Standard HTTP status codes for REST endpoints
- **Logging**: All errors are logged with context
- **User-Friendly Messages**: Client-safe error messages

### Security Features

- **JWT Authentication**: Stateless token-based auth
- **Role-Based Access**: Fine-grained permission checking
- **Multi-Tenant**: Club-based data isolation
- **Rate Limiting**: Protection against abuse
- **Input Validation**: Request payload validation
- **CORS Protection**: Origin-based access control

## Development

### Adding New GraphQL Operations

1. **Update Schema**: Add types/queries/mutations to `schema.graphql`
2. **Regenerate Code**: Run `gqlgen generate`
3. **Implement Resolvers**: Add resolver logic in `graph/schema.resolvers.go`
4. **Add Service Calls**: Integrate with backend service clients
5. **Test**: Add unit and integration tests

### Adding New REST Endpoints

1. **Define Handler**: Add handler function in `server/server.go`
2. **Add Route**: Register route in `setupRESTRoutes()`
3. **Add Middleware**: Apply appropriate middleware
4. **Document**: Update API documentation
5. **Test**: Add endpoint tests

### Middleware Development

1. **Create Middleware**: Add to `middleware/middleware.go`
2. **Apply Middleware**: Register in `setupRoutes()`
3. **Configure**: Add configuration options if needed
4. **Test**: Verify middleware behavior

### Testing

Run tests:
```bash
go test ./...
```

Run with coverage:
```bash
go test -cover ./...
```

Run integration tests:
```bash
go test -tags=integration ./...
```

### Performance

The API Gateway is designed for high performance:

- **Connection Pooling**: Efficient gRPC connection management
- **Caching**: Response caching for expensive operations
- **Async Processing**: Non-blocking operations where possible
- **Metrics**: Performance monitoring and alerting
- **Load Balancing**: Multiple instance support

### Monitoring

Key metrics exposed:

- **HTTP Requests**: Duration, count, status codes
- **GraphQL Operations**: Query complexity, execution time
- **gRPC Calls**: Service call metrics
- **Rate Limiting**: Rejection rates
- **Authentication**: Success/failure rates
- **Service Health**: Backend service availability

### Scaling

Horizontal scaling considerations:

- **Stateless Design**: No local state dependencies
- **Service Discovery**: Dynamic backend service resolution  
- **Load Balancer**: Multiple gateway instances
- **Circuit Breakers**: Fault tolerance for backend failures
- **Caching Layer**: Redis for shared caching

## Deployment

### Docker

```bash
# Build image
docker build -t api-gateway .

# Run container
docker run -p 8080:8080 \
  -e API_GATEWAY_DATABASE_HOST=postgres \
  -e API_GATEWAY_NATS_URL=nats://nats:4222 \
  api-gateway
```

### Kubernetes

See `deployments/k8s/services/api-gateway.yaml` for K8s manifests.

### Health Checks

Configure health checks:

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10

readinessProbe:
  httpGet:
    path: /ready
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 5
```

## Troubleshooting

### Common Issues

1. **Service Connection Failures**:
   - Check backend service availability
   - Verify gRPC port configuration
   - Check network connectivity

2. **Authentication Errors**:
   - Verify JWT secret configuration
   - Check token expiration
   - Validate user permissions

3. **Rate Limiting**:
   - Check rate limit configuration
   - Monitor client request patterns
   - Consider increasing limits for valid users

4. **GraphQL Errors**:
   - Validate schema syntax
   - Check resolver implementations
   - Monitor query complexity

### Debugging

Enable debug logging:
```bash
API_GATEWAY_LOGGING_LEVEL=debug go run cmd/main.go
```

View metrics:
```bash
curl http://localhost:2112/metrics
```

Check health:
```bash
curl http://localhost:8080/health
```

## Contributing

1. Follow Go conventions and best practices
2. Add tests for new functionality
3. Update documentation
4. Use structured logging
5. Include metrics for monitoring
6. Handle errors gracefully

## Related Services

- [Auth Service](../auth-service/README.md)
- [Member Service](../member-service/README.md)  
- [Reciprocal Service](../reciprocal-service/README.md)
- [Blockchain Service](../blockchain-service/README.md)