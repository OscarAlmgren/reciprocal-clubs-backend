# Podman Quadlet Deployment for Reciprocal Clubs Platform

This directory contains Podman Quadlet configurations to run the Reciprocal Clubs Platform using Podman instead of Docker Compose.

## Prerequisites

### Install Podman on macOS

```bash
# Install Podman using Homebrew
brew install podman

# Initialize and start Podman machine
podman machine init
podman machine start

# Verify installation
podman --version
podman system info
```

## Quick Start

### Option 1: Using the Podman Compose Script (Recommended for macOS)

The `podman-compose.sh` script provides Docker Compose-like functionality:

```bash
# Start all services
./podman-compose.sh up

# Stop all services
./podman-compose.sh down

# Check status
./podman-compose.sh status

# View logs for a specific service
./podman-compose.sh logs postgres

# Clean up everything
./podman-compose.sh clean
```

### Option 2: Using Make Commands

```bash
# Set up and start all services
make start-all

# Or start services in stages
make start-infra     # PostgreSQL, Redis, NATS, MailHog
make start-fabric    # Hyperledger Fabric components
make start-apps      # Application microservices
make start-monitoring # Prometheus, Grafana, Jaeger

# Check status
make status

# Stop all services
make stop-all

# Clean up
make clean
```

### Option 3: Using Individual Podman Commands

```bash
# Create network
podman network create reciprocal-clubs

# Start PostgreSQL
podman run -d \
  --name reciprocal-postgres \
  --network reciprocal-clubs \
  -p 5432:5432 \
  -e POSTGRES_DB=reciprocal_clubs \
  -e POSTGRES_USER=postgres \
  -e POSTGRES_PASSWORD=postgres \
  -v reciprocal-postgres-data:/var/lib/postgresql/data \
  postgres:15-alpine

# Continue with other services...
```

## Available Commands

### Podman Compose Script

| Command | Description |
|---------|-------------|
| `up` | Start all services |
| `down` | Stop all services |
| `build` | Build application images |
| `infra` | Start infrastructure services only |
| `fabric` | Start Hyperledger Fabric services only |
| `apps` | Start application services only |
| `monitoring` | Start monitoring services only |
| `status` | Show status of all services |
| `logs <service>` | Show logs for a specific service |
| `health` | Check health of core services |
| `clean` | Remove all containers and volumes |
| `restart` | Restart all services |

### Make Commands

| Command | Description |
|---------|-------------|
| `make help` | Show available commands |
| `make setup-podman` | Set up Podman quadlets |
| `make build-images` | Build application images |
| `make start-all` | Start all services |
| `make start-infra` | Start infrastructure services |
| `make start-fabric` | Start Fabric services |
| `make start-apps` | Start application services |
| `make start-monitoring` | Start monitoring services |
| `make stop-all` | Stop all services |
| `make status` | Show service status |
| `make clean` | Clean up everything |
| `make logs SERVICE=<name>` | Show logs for specific service |

## Service Architecture

### Infrastructure Services

- **PostgreSQL** (`reciprocal-postgres`) - Main database
- **Redis** (`reciprocal-redis`) - Caching and sessions
- **NATS** (`reciprocal-nats`) - Message queue
- **MailHog** (`reciprocal-mailhog`) - Email testing

### Hyperledger Fabric Services

- **Fabric CA** (`reciprocal-fabric-ca`) - Certificate Authority
- **Fabric Orderer** (`reciprocal-orderer`) - Transaction ordering
- **Fabric Peer** (`reciprocal-peer0-org1`) - Peer node

### Application Services

- **API Gateway** (`reciprocal-api-gateway`) - Main entry point
- **Auth Service** (`reciprocal-auth-service`) - Authentication
- **Member Service** (`reciprocal-member-service`) - Member management
- **Reciprocal Service** (`reciprocal-reciprocal-service`) - Reciprocal agreements
- **Blockchain Service** (`reciprocal-blockchain-service`) - Blockchain integration
- **Notification Service** (`reciprocal-notification-service`) - Notifications
- **Analytics Service** (`reciprocal-analytics-service`) - Analytics
- **Governance Service** (`reciprocal-governance-service`) - Governance

### Monitoring Services

- **Prometheus** (`reciprocal-prometheus`) - Metrics collection
- **Grafana** (`reciprocal-grafana`) - Dashboards
- **Jaeger** (`reciprocal-jaeger`) - Distributed tracing

## Port Mappings

| Service | HTTP Port | gRPC Port | Purpose |
|---------|-----------|-----------|---------|
| API Gateway | 8080 | 9080 | Main API entry |
| Auth Service | 8081 | 9081 | Authentication |
| Member Service | 8082 | 9082 | Member management |
| Reciprocal Service | 8083 | 9083 | Reciprocal agreements |
| Blockchain Service | 8084 | 9084 | Blockchain operations |
| Notification Service | 8085 | 9085 | Notifications |
| Analytics Service | 8086 | 9086 | Analytics |
| Governance Service | 8087 | 9087 | Governance |
| PostgreSQL | 5432 | - | Database |
| Redis | 6379 | - | Cache |
| NATS | 4222, 8222 | 6222 | Message queue |
| MailHog | 8025 | 1025 | Email testing |
| Fabric CA | 7054 | - | Certificate Authority |
| Fabric Orderer | 7050 | - | Transaction ordering |
| Fabric Peer | 7051 | 7052 | Peer node |
| Prometheus | 9090 | - | Metrics |
| Grafana | 3000 | - | Dashboards |
| Jaeger | 16686 | 14268 | Tracing |

## Volumes

Persistent data is stored in Podman volumes:

- `reciprocal-postgres-data` - PostgreSQL data
- `reciprocal-redis-data` - Redis data
- `reciprocal-nats-data` - NATS JetStream data
- `reciprocal-fabric-ca` - Fabric CA data
- `reciprocal-fabric-orderer` - Fabric Orderer data
- `reciprocal-fabric-peer` - Fabric Peer data
- `reciprocal-prometheus-data` - Prometheus data
- `reciprocal-grafana-data` - Grafana dashboards

## Networking

All services run on the `reciprocal-clubs` network, allowing internal communication using container names.

## Environment Variables

Key environment variables can be customized:

```bash
# Database
POSTGRES_DB=reciprocal_clubs
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# JWT Secret
JWT_SECRET=your-secret-key

# Service Endpoints
DATABASE_HOST=reciprocal-postgres
NATS_URL=nats://reciprocal-nats:4222
REDIS_HOST=reciprocal-redis
```

## Building Application Images

Before starting the application services, build the Docker images:

```bash
# Build all images
./podman-compose.sh build

# Or using Make
make build-images
```

The script will build images for:
- `localhost/reciprocal-api-gateway:latest`
- `localhost/reciprocal-auth-service:latest`
- `localhost/reciprocal-member-service:latest`
- `localhost/reciprocal-reciprocal-service:latest`
- `localhost/reciprocal-blockchain-service:latest`
- `localhost/reciprocal-notification-service:latest`
- `localhost/reciprocal-analytics-service:latest`
- `localhost/reciprocal-governance-service:latest`

## Health Checking

Check service health:

```bash
# Using the script
./podman-compose.sh health

# Using Make
make health-check

# Manual check
podman exec reciprocal-postgres pg_isready -U postgres
podman exec reciprocal-redis redis-cli ping
```

## Troubleshooting

### Common Issues

1. **Podman machine not running**:
   ```bash
   podman machine start
   ```

2. **Network issues**:
   ```bash
   podman network rm reciprocal-clubs
   podman network create reciprocal-clubs
   ```

3. **Port conflicts**:
   ```bash
   # Check what's using the port
   lsof -i :5432
   
   # Kill the process or change port mapping
   ```

4. **Volume permission issues**:
   ```bash
   # Reset volumes
   podman volume rm -f reciprocal-postgres-data
   ```

5. **Image build failures**:
   ```bash
   # Clean build cache
   podman system prune -a
   
   # Rebuild specific image
   podman build -t localhost/reciprocal-member-service:latest \
     -f services/member-service/Dockerfile .
   ```

### Viewing Logs

```bash
# All logs for a service
./podman-compose.sh logs postgres

# Follow logs
podman logs -f reciprocal-postgres

# Last 100 lines
podman logs --tail 100 reciprocal-postgres
```

### Resource Usage

```bash
# Check resource usage
podman stats

# Check container info
podman inspect reciprocal-postgres
```

## Differences from Docker Compose

1. **No built-in orchestration**: Services are started individually
2. **No automatic dependency management**: Must start services in correct order
3. **Different networking**: Uses Podman networking instead of Docker networks
4. **Volume handling**: Podman volumes work slightly differently
5. **Systemd integration**: On Linux, can use systemd for service management

## Systemd Integration (Linux only)

On Linux systems, you can use the quadlet files with systemd:

```bash
# Copy quadlet files
mkdir -p ~/.config/containers/systemd
cp *.container *.network ~/.config/containers/systemd/

# Reload systemd
systemctl --user daemon-reload

# Start services
systemctl --user start postgres.service
systemctl --user start redis.service
```

Note: This doesn't work on macOS as it doesn't have systemd.

## Comparison with Docker Compose

| Feature | Docker Compose | Podman Quadlets |
|---------|----------------|-----------------|
| Single command start | ✅ | ✅ (with script) |
| Dependency management | ✅ | ⚠️ Manual |
| Networking | ✅ | ✅ |
| Volume management | ✅ | ✅ |
| Health checks | ✅ | ✅ |
| Scaling | ✅ | ⚠️ Manual |
| systemd integration | ❌ | ✅ (Linux) |
| Rootless by default | ❌ | ✅ |

## Migration from Docker Compose

To migrate from Docker Compose to Podman:

1. **Install Podman** and initialize machine
2. **Build images** using Podman
3. **Use the script** for orchestration
4. **Adjust volume paths** if needed
5. **Update CI/CD** to use Podman commands

The provided script maintains compatibility with Docker Compose workflows while leveraging Podman's security and rootless features.

## Production Considerations

For production deployments:

1. **Use Kubernetes** for better orchestration
2. **Implement proper secrets management**
3. **Set up monitoring and alerting**
4. **Configure backup strategies**
5. **Use systemd** on Linux for service management
6. **Implement rolling updates**
7. **Set resource limits**

## Contributing

When adding new services:

1. Create `.container` quadlet file
2. Update the `podman-compose.sh` script
3. Add to Makefile
4. Update this README
5. Test startup order and dependencies