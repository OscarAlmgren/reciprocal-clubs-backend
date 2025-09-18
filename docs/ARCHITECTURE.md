# Reciprocal Clubs Backend - Architecture Documentation

## Executive Summary

The Reciprocal Clubs Backend is a comprehensive microservices-based system built for managing private blockchain-based reciprocal club memberships. The system leverages Hyperledger Fabric for immutable transaction records and provides a multi-tenant architecture supporting multiple clubs with cross-club reciprocal agreements.

## System Overview

### Architecture Paradigm
- **Microservices Architecture**: Loosely coupled services with domain-specific responsibilities
- **Event-Driven Communication**: NATS message bus for asynchronous inter-service communication
- **Multi-Tenant Design**: Club-based data partitioning with tenant-aware authorization
- **Blockchain Integration**: Hyperledger Fabric for immutable audit trails and cross-club transactions
- **API-First Design**: GraphQL gateway with RESTful service endpoints

### Core Principles
1. **Domain Isolation**: Each service owns its data and business logic
2. **Fail-Fast**: Comprehensive input validation and error handling
3. **Observability**: Structured logging, metrics, and distributed tracing
4. **Security by Design**: Multi-layer security with JWT, RBAC, and tenant isolation
5. **Scalability**: Horizontal scaling with stateless services and connection pooling

## Service Architecture

### Service Catalog

| Service | Status | Purpose | Technology Stack |
|---------|---------|---------|------------------|
| **API Gateway** | 🟡 Partial | GraphQL/REST entry point, authentication | Go, gqlgen, GraphQL |
| **Auth Service** | 🟡 Partial | Multi-tenant authentication, RBAC | Go, JWT, bcrypt |
| **Member Service** | 🔴 Planned | Member CRUD, profiles, lifecycle | Go, gRPC, PostgreSQL |
| **Reciprocal Service** | 🟢 Complete | Cross-club agreements, visit verification | Go, gRPC, Blockchain |
| **Blockchain Service** | 🟢 Complete | Hyperledger Fabric integration | Go, Fabric SDK |
| **Notification Service** | 🟢 Complete | Multi-channel notifications | Go, Templates, SMTP/SMS |
| **Analytics Service** | 🟢 Complete | Usage analytics, reporting | Go, Time-series DB |
| **Governance Service** | 🟢 Complete | Network governance, voting | Go, Smart Contracts |

**Legend**: 🟢 Complete, 🟡 Partial, 🔴 Planned

**Current Implementation Status**: 6 out of 8 services fully implemented, 2 services partially implemented. All containerization and deployment configurations complete.

### Service Interaction Model

```
┌─────────────────────────────────────────────────────────────────────┐
│                            Client Layer                            │
├─────────────────────────────────────────────────────────────────────┤
│  Web App        Mobile App       Admin Portal      Third-party APIs  │
└─────────────────┬───────────────────────────────────────────────────┘
                  │ HTTP/GraphQL/WebSocket
┌─────────────────▼───────────────────────────────────────────────────┐
│                        API Gateway Layer                           │
├─────────────────────────────────────────────────────────────────────┤
│ • GraphQL Schema Stitching    • Authentication Middleware           │
│ • Rate Limiting              • Request/Response Transformation       │
│ • Load Balancing             • CORS & Security Headers              │
└─────────────────┬───────────────────────────────────────────────────┘
                  │ gRPC/HTTP
┌─────────────────▼───────────────────────────────────────────────────┐
│                      Business Services Layer                       │
├─────────────────────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
│ │    Auth     │ │   Member    │ │ Reciprocal  │ │ Blockchain  │    │
│ │   Service   │ │   Service   │ │   Service   │ │   Service   │    │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
│                                                                     │
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
│ │Notification │ │ Analytics   │ │ Governance  │ │    (Future  │    │
│ │   Service   │ │   Service   │ │   Service   │ │   Services) │    │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
└─────────────────┬───────────────────────────────────────────────────┘
                  │ NATS Event Bus
┌─────────────────▼───────────────────────────────────────────────────┐
│                      Infrastructure Layer                          │
├─────────────────────────────────────────────────────────────────────┤
│ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐    │
│ │ PostgreSQL  │ │    NATS     │ │ Hyperledger │ │  Prometheus │    │
│ │  Databases  │ │ Msg. Bus    │ │   Fabric    │ │ & Grafana   │    │
│ └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Architecture

### Database Design Strategy

**Multi-Tenant Data Partitioning**:
- Each club maintains isolated data partitions
- Tenant ID (club_id) in all domain entities
- Row-level security policies for data isolation
- Shared reference data (countries, currencies, etc.)

**Data Consistency Model**:
- **Strong Consistency**: Within service boundaries via ACID transactions
- **Eventual Consistency**: Cross-service via event sourcing
- **Immutable Audit Trail**: Blockchain integration for critical operations

### Service Data Ownership

| Service | Database | Owned Entities | Relationships |
|---------|----------|----------------|---------------|
| Auth | `auth_db` | Users, Roles, Sessions, Tenants | → Member (auth_user_id) |
| Member | `member_db` | Members, Profiles, Addresses, Clubs | ← Auth, → Reciprocal |
| Reciprocal | `reciprocal_db` | Agreements, Visits, Verifications | ← Member, → Blockchain |
| Blockchain | `blockchain_db` | Transactions, Blocks, Events | ← All Services |
| Notification | `notification_db` | Templates, Queues, Delivery Status | ← All Services |
| Analytics | `analytics_db` | Metrics, Reports, Aggregations | ← All Services |
| Governance | `governance_db` | Proposals, Votes, Policies | ← All Services |

### Database Schema Highlights

**Member Service Schema**:
```sql
-- Core entities with multi-tenant support
CREATE TABLE clubs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    status club_status_enum DEFAULT 'active',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE members (
    id SERIAL PRIMARY KEY,
    club_id INTEGER REFERENCES clubs(id),
    auth_user_id VARCHAR(255) UNIQUE NOT NULL,
    member_number VARCHAR(50),
    status member_status_enum DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE profiles (
    id SERIAL PRIMARY KEY,
    member_id INTEGER REFERENCES members(id),
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    email VARCHAR(255),
    phone VARCHAR(20),
    date_of_birth DATE,
    updated_at TIMESTAMP DEFAULT NOW()
);
```

## Communication Architecture

### Synchronous Communication (gRPC)
- **API Gateway → Services**: Request/response for real-time operations
- **Service-to-Service**: Direct calls for immediate data needs
- **Authentication**: JWT validation across all service calls
- **Load Balancing**: Service discovery and client-side load balancing

### Asynchronous Communication (NATS)
- **Event Broadcasting**: Domain events published to interested services
- **Saga Orchestration**: Multi-service transaction coordination
- **Notification Delivery**: Decoupled notification processing
- **Analytics Data Pipeline**: Event streaming for metrics collection

### Event Schema Design

**Domain Events Structure**:
```go
type DomainEvent struct {
    ID           string                 `json:"id"`
    Type         string                 `json:"type"`
    Source       string                 `json:"source"`
    TenantID     string                 `json:"tenant_id"`
    Timestamp    time.Time             `json:"timestamp"`
    Version      string                 `json:"version"`
    Data         map[string]interface{} `json:"data"`
    Metadata     EventMetadata         `json:"metadata"`
}

type EventMetadata struct {
    CorrelationID string `json:"correlation_id"`
    CausationID   string `json:"causation_id"`
    UserID        string `json:"user_id,omitempty"`
    IPAddress     string `json:"ip_address,omitempty"`
    UserAgent     string `json:"user_agent,omitempty"`
}
```

**Key Event Types**:
- `member.created`, `member.updated`, `member.status_changed`
- `club.created`, `club.settings_updated`
- `reciprocal.agreement_created`, `reciprocal.visit_recorded`
- `blockchain.transaction_committed`
- `auth.user_logged_in`, `auth.permission_granted`

## Security Architecture

### Authentication & Authorization Flow

```
┌──────────────┐    1. Login Request    ┌─────────────┐
│   Client     │ ──────────────────────► │API Gateway  │
└──────────────┘                        └─────────────┘
                                                │
                                                │ 2. Forward Auth
                                                ▼
                                        ┌─────────────┐
                                        │Auth Service │
                                        └─────────────┘
                                                │
                                                │ 3. Validate Credentials
                                                ▼
                                        ┌─────────────┐
                                        │ Database    │
                                        └─────────────┘
                                                │
                ┌──────────────┐                │ 4. Generate JWT
                │   Client     │ ◄──────────────┘
                └──────────────┘
                        │
                        │ 5. Subsequent Requests (JWT)
                        ▼
                ┌─────────────┐    6. JWT Validation    ┌─────────────┐
                │API Gateway  │ ──────────────────────► │   Service   │
                └─────────────┘                        └─────────────┘
```

### Security Layers

1. **Network Security**:
   - TLS 1.3 for all external communications
   - Internal service mesh with mTLS
   - Network segmentation and firewall rules

2. **Authentication**:
   - JWT tokens with RS256 signing
   - Refresh token rotation
   - Multi-factor authentication support
   - Session management with Redis

3. **Authorization**:
   - Role-Based Access Control (RBAC)
   - Tenant-aware permissions
   - Resource-level access control
   - Dynamic permission evaluation

4. **Data Protection**:
   - Encryption at rest (PostgreSQL TDE)
   - Field-level encryption for PII
   - Data anonymization for analytics
   - GDPR compliance features

## Deployment Architecture

### Container Strategy
- **Base Images**: Distroless images for security
- **Multi-stage Builds**: Optimized image sizes
- **Health Checks**: Comprehensive liveness/readiness probes
- **Resource Limits**: CPU/memory constraints per service

### Kubernetes Deployment Model

```yaml
# Example service deployment structure
apiVersion: apps/v1
kind: Deployment
metadata:
  name: member-service
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxUnavailable: 1
      maxSurge: 1
  template:
    spec:
      containers:
      - name: member-service
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
```

### Infrastructure Components

**Core Infrastructure**:
- **Kubernetes Cluster**: Container orchestration
- **PostgreSQL Cluster**: Primary data storage with read replicas
- **NATS Cluster**: Message bus with clustering
- **Redis Cluster**: Session storage and caching
- **Hyperledger Fabric**: Blockchain network

**Monitoring & Observability**:
- **Prometheus**: Metrics collection
- **Grafana**: Metrics visualization
- **Jaeger**: Distributed tracing
- **ELK Stack**: Log aggregation and analysis

## Performance & Scalability

### Performance Characteristics

| Component | Latency Target | Throughput Target | Scalability Model |
|-----------|---------------|------------------|------------------|
| API Gateway | < 100ms | 10k req/s | Horizontal pods |
| Auth Service | < 50ms | 5k req/s | Horizontal pods |
| Member Service | < 200ms | 2k req/s | Horizontal pods |
| Database | < 10ms | 50k ops/s | Read replicas + sharding |
| Message Bus | < 5ms | 100k msg/s | Clustering |

### Scaling Strategies

**Horizontal Scaling**:
- Kubernetes Horizontal Pod Autoscaler (HPA)
- Custom metrics-based scaling
- Database read replicas
- Message bus clustering

**Caching Strategy**:
- Redis for session and frequently accessed data
- Application-level caching with TTL
- Database query result caching
- CDN for static assets

**Database Optimization**:
- Connection pooling with pgBouncer
- Index optimization for query patterns
- Partitioning for large tables
- Async replication for read replicas

## Quality Assurance

### Testing Strategy

**Test Pyramid Implementation**:
```
     ▲
    /E2E\     <- End-to-End (Real workflows)
   /─────\    <- Integration (Service boundaries)
  /───────\   <- Unit Tests (Business logic)
 /_________\
```

**Coverage Targets**:
- Unit Tests: > 80% code coverage
- Integration Tests: All service boundaries
- E2E Tests: Critical user journeys
- Performance Tests: Load and stress testing

### Code Quality

**Static Analysis**:
- `golangci-lint` for Go code analysis
- `gosec` for security vulnerability scanning
- `goimports` for import organization
- Custom linting rules for domain consistency

**Code Review Process**:
- Mandatory PR reviews
- Automated CI/CD checks
- Architecture decision documentation
- Security review for critical changes

## Technology Stack Deep Dive

### Core Technologies

**Backend Language**: Go 1.25+
- **Rationale**: Performance, concurrency, strong typing
- **Key Libraries**: gRPC, GORM, testify, zerolog

**Database**: PostgreSQL 15+
- **Rationale**: ACID compliance, JSON support, performance
- **Features**: Row-level security, full-text search, extensions

**Message Bus**: NATS
- **Rationale**: Lightweight, high performance, clustering
- **Features**: JetStream, subject hierarchies, authentication

**Blockchain**: Hyperledger Fabric
- **Rationale**: Private networks, chaincode, enterprise features
- **Integration**: Go SDK, event listening, transaction submission

### Development Tools

**Code Generation**:
- `gqlgen` for GraphQL schema-first development
- `protoc` for Protocol Buffer generation
- `mockery` for test mock generation

**Database Migration**:
- Custom migration system with rollback support
- Schema versioning and validation
- Data seeding for development/testing

## Future Architecture Considerations

### Planned Enhancements

1. **Multi-Region Deployment**:
   - Database replication across regions
   - Event sourcing with regional failover
   - CDN integration for global performance

2. **Advanced Security**:
   - OAuth2/OIDC integration
   - SAML for enterprise SSO
   - Zero-trust network model

3. **Performance Optimization**:
   - GraphQL query optimization
   - Database sharding strategies
   - Edge computing for mobile apps

4. **Operational Excellence**:
   - Chaos engineering practices
   - Automated incident response
   - Advanced monitoring and alerting

### Technology Evolution

**Emerging Technologies**:
- **WebAssembly**: For plugin architecture
- **Kubernetes Operators**: For automated operations
- **Service Mesh**: For advanced traffic management
- **Event Streaming**: Apache Kafka for high-volume events

This architecture provides a solid foundation for a scalable, secure, and maintainable reciprocal clubs management system while maintaining flexibility for future enhancements and growth.
