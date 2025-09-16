# Reciprocal Clubs Backend

A comprehensive microservices backend system for a private blockchain-based reciprocal club membership platform using Hyperledger Fabric.

## Architecture Overview

This system consists of 8 microservices designed for scalability, maintainability, and multi-tenant support:

### Services

1. **API Gateway** (`/services/api-gateway`) - GraphQL gateway with authentication, rate limiting, and service orchestration
2. **Auth Service** (`/services/auth-service`) - Multi-tenant authentication, RBAC, and JWT management
3. **Member Service** (`/services/member-service`) - Member CRUD operations, profiles, and blockchain identity integration
4. **Reciprocal Service** (`/services/reciprocal-service`) - Reciprocal agreements, visit verification, and cross-club communication
5. **Blockchain Service** (`/services/blockchain-service`) - Hyperledger Fabric integration, chaincode interaction, and event processing
6. **Notification Service** (`/services/notification-service`) - Multi-channel notifications (email/SMS/push) with templates
7. **Analytics Service** (`/services/analytics-service`) - Usage analytics, metrics collection, and privacy-preserving aggregation
8. **Governance Service** (`/services/governance-service`) - Network governance, voting mechanisms, and policy management

### Shared Libraries (`/pkg/shared`)

- **config** - Environment configuration management with Viper
- **database** - PostgreSQL connection pooling and GORM integration
- **messaging** - NATS message bus with retry logic and event processing
- **logging** - Structured logging with zerolog and correlation IDs
- **monitoring** - Prometheus metrics and health checks
- **errors** - Structured error handling with custom error types
- **auth** - JWT authentication and multi-tenant authorization
- **utils** - Common utilities for validation, crypto, and data manipulation

## Technology Stack

- **Language**: Go 1.21+
- **GraphQL**: gqlgen for schema-first development
- **Database**: PostgreSQL with GORM ORM
- **Message Queue**: NATS for event-driven architecture
- **Blockchain**: Hyperledger Fabric SDK (MockService for development/testing)
- **Authentication**: JWT with multi-tenant support
- **API**: RESTful APIs + gRPC for inter-service communication
- **Monitoring**: Prometheus metrics with structured logging
- **Configuration**: Viper for environment-based configuration
- **Containerization**: Docker with Podman support

## Project Structure

```
reciprocal-clubs-backend/
├── services/                    # Microservices
│   ├── api-gateway/            # GraphQL gateway
│   ├── auth-service/           # Authentication service
│   ├── member-service/         # Member management
│   ├── reciprocal-service/     # Reciprocal agreements
│   ├── blockchain-service/     # Blockchain integration
│   ├── notification-service/   # Notifications
│   ├── analytics-service/      # Analytics and reporting
│   └── governance-service/     # Network governance
├── pkg/shared/                 # Shared libraries
│   ├── config/                 # Configuration management
│   ├── database/              # Database utilities
│   ├── messaging/             # Message bus client
│   ├── logging/               # Logging utilities
│   ├── monitoring/            # Metrics and health checks
│   ├── errors/                # Error handling
│   ├── auth/                  # Authentication utilities
│   └── utils/                 # Common utilities
├── deployments/k8s/           # Kubernetes manifests
│   ├── infrastructure/        # Infrastructure components
│   ├── services/             # Service deployments
│   └── config/               # Configuration maps
├── scripts/                   # Helper scripts
├── tests/                     # Test suites
│   ├── unit/                 # Unit tests
│   ├── integration/          # Integration tests
│   └── e2e/                  # End-to-end tests
├── go.mod                    # Go module definition
└── README.md                 # This file
```

## Features

### Multi-Tenant Architecture
- Club-based data partitioning
- Tenant-aware authentication and authorization
- Isolated data access per club

### Event-Driven Communication
- NATS message bus for service communication
- Event sourcing for blockchain transactions
- Async processing with retry logic

### Comprehensive Monitoring
- Prometheus metrics for all services
- Health checks and readiness probes
- Structured logging with correlation IDs
- Business metrics tracking

### Security
- JWT-based authentication with refresh tokens
- Role-based access control (RBAC)
- Multi-tenant authorization
- Input validation and sanitization

### Blockchain Integration
- Hyperledger Fabric SDK integration
- Mock services for development/testing
- Event processing and transaction tracking
- Multi-channel support

## Quick Start

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 13+
- NATS Server
- Redis (optional, for caching)
- Podman or Docker

### Development Setup

1. **Clone the repository:**
   ```bash
   git clone <repository-url>
   cd reciprocal-clubs-backend
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   go mod verify
   ```

3. **Set up infrastructure:**
   ```bash
   # Using provided setup script
   ./scripts/setup-dev.sh
   ```

4. **Run database migrations:**
   ```bash
   # Each service handles its own migrations
   go run services/auth-service/cmd/main.go migrate
   go run services/member-service/cmd/main.go migrate
   # ... etc for each service
   ```

5. **Start services:**
   ```bash
   # Start all services (requires infrastructure running)
   ./scripts/run-all.sh
   
   # Or start individual services
   go run services/api-gateway/cmd/main.go
   go run services/auth-service/cmd/main.go
   # ... etc
   ```

### Using Podman

The project supports Podman as the preferred container engine:

```bash
# Build all service images
./scripts/build-podman.sh

# Run infrastructure with Podman Compose
podman-compose -f deployments/podman-compose.yml up -d

# Deploy services
./scripts/deploy-podman.sh
```

## Configuration

Services are configured via environment variables and YAML files. Key configuration areas:

- **Database**: Connection strings, pool settings
- **NATS**: Cluster configuration, retry settings  
- **Authentication**: JWT secrets, token expiration
- **Monitoring**: Metrics endpoints, health check intervals
- **Logging**: Log levels, output formats

See individual service README files for specific configuration options.

## API Documentation

### GraphQL API (API Gateway)
- **Endpoint**: `http://localhost:8080/graphql`
- **Playground**: `http://localhost:8080/playground`

### REST APIs
Each service exposes REST endpoints:
- Auth Service: `http://localhost:8081`
- Member Service: `http://localhost:8082`
- Reciprocal Service: `http://localhost:8083`
- Blockchain Service: `http://localhost:8084`
- Notification Service: `http://localhost:8085`
- Analytics Service: `http://localhost:8086`
- Governance Service: `http://localhost:8087`

### Monitoring
- **Metrics**: `http://localhost:2112/metrics` (each service)
- **Health**: `http://localhost:8080/health`
- **Ready**: `http://localhost:8080/ready`

## Testing

### Unit Tests
```bash
# Run all unit tests
./scripts/run-tests.sh unit

# Run tests for specific service
go test ./services/auth-service/...
```

### Integration Tests
```bash
# Run integration tests (requires test infrastructure)
./scripts/run-tests.sh integration
```

### End-to-End Tests
```bash
# Run e2e tests (requires full system running)
./scripts/run-tests.sh e2e
```

## Deployment

### Local Development
```bash
./scripts/setup-dev.sh
./scripts/run-all.sh
```

### Kubernetes
```bash
# Deploy infrastructure
kubectl apply -f deployments/k8s/infrastructure/

# Deploy services
kubectl apply -f deployments/k8s/services/
```

### Production Considerations

- Use proper secrets management (Kubernetes secrets, HashiCorp Vault)
- Configure persistent volumes for PostgreSQL
- Set up ingress controllers and SSL certificates
- Configure resource limits and auto-scaling
- Implement backup and disaster recovery procedures

## Monitoring and Observability

### Metrics
- HTTP request duration and count
- gRPC call metrics
- Database connection pooling
- Message bus throughput
- Business event tracking
- Service health status

### Logging
- Structured JSON logs
- Correlation ID tracking
- Request/response logging
- Error tracking and alerting

### Health Checks
- Liveness probes for container orchestration
- Readiness probes for traffic management
- Dependency health monitoring

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines

- Follow Go best practices and idioms
- Write comprehensive tests (aim for >80% coverage)
- Use structured logging with appropriate levels
- Implement proper error handling
- Document public APIs and complex business logic
- Follow the established project structure

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For questions, issues, or contributions, please:

1. Check the [Issues](../../issues) page
2. Review service-specific README files
3. Contact the development team

## Roadmap

- [ ] Complete all 8 microservices implementation
- [ ] Comprehensive test suite with >80% coverage
- [ ] Production-ready Kubernetes deployments
- [ ] Integration with real Hyperledger Fabric network
- [ ] Advanced analytics and reporting features
- [ ] Mobile app integration APIs
- [ ] Multi-region deployment support
- [ ] Advanced security features (OAuth2, SAML)