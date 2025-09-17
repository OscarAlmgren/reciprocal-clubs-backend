# Service Port Allocation

## Application Services

| Service | HTTP Port | gRPC Port | Description |
|---------|-----------|-----------|-------------|
| API Gateway | 8080 | 9080 | Main API gateway |
| Auth Service | 8081 | 9081 | Authentication service |
| Member Service | 8082 | 9082 | Member management |
| Reciprocal Service | 8083 | 9083 | Reciprocal club features |
| Blockchain Service | 8084 | 9084 | Blockchain integration |
| Notification Service | 8085 | 9085 | Notifications |
| Analytics Service | 8086 | 9086 | Analytics and reporting |
| Governance Service | 8087 | 9087 | Governance features |

## Infrastructure Services

| Service | Port(s) | Description |
|---------|---------|-------------|
| PostgreSQL | 5432 | Database |
| Redis | 6379 | Cache and sessions |
| NATS | 4222, 6222, 8222 | Messaging (client, cluster, monitor) |
| **Hanko Public** | **8000** | **Authentication API** |
| **Hanko Admin** | **8001** | **Admin API** |

## Hyperledger Fabric

| Service | Port | Description |
|---------|------|-------------|
| Fabric CA | 7054 | Certificate Authority |
| Fabric Orderer | 7050 | Orderer service |
| Fabric Peer | 7051, 7052 | Peer service (peer, chaincode) |

## Monitoring & Development

| Service | Port | Description |
|---------|------|-------------|
| Prometheus | 9090 | Metrics collection |
| Grafana | 3000 | Dashboards |
| Jaeger | 16686, 14268 | Tracing |
| MailHog | 8025, 1025 | Email testing (web, SMTP) |

## Network Configuration

### Docker Compose
- Network: `reciprocal-clubs`
- Internal DNS resolution by service name

### Kubernetes
- Namespace: `reciprocal-clubs`
- Service discovery via Kubernetes DNS

### Podman
- Network: `reciprocal-clubs`
- Container name resolution

## External Access

### Development (Local)
- All services accessible via localhost with mapped ports
- Hanko: http://localhost:8000 (public), http://localhost:8001 (admin)

### Kubernetes Ingress
- Auth API: `auth.reciprocal-clubs.local`
- Auth Admin: `auth-admin.reciprocal-clubs.local`

### Load Balancer Considerations
- Hanko requires sticky sessions for authentication flows
- Auth service should be behind the same load balancer as Hanko