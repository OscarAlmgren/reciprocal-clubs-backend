# Docker Setup Documentation

This document provides comprehensive documentation for the Docker setup of the Reciprocal Clubs Backend project.

## Overview

All application services now have properly configured Dockerfiles with consistent build patterns, and the project includes comprehensive build and test utilities.

## Services with Dockerfiles

The following services have multi-stage Dockerfiles that build from scratch:

1. **api-gateway** - GraphQL API Gateway service
2. **auth-service** - Authentication service
3. **member-service** - Member management service
4. **reciprocal-service** - Core reciprocal club functionality
5. **analytics-service** - Analytics and reporting service
6. **blockchain-service** - Hyperledger Fabric blockchain service
7. **governance-service** - Governance and voting service
8. **notification-service** - Notification management service

## Docker Architecture

### Multi-stage Build Pattern

All Dockerfiles follow a consistent multi-stage build pattern:

```dockerfile
# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates build-base

# Set working directory
WORKDIR /build

# Create non-root user
RUN adduser -D -g '' appuser

# Copy entire project (enables access to shared packages)
COPY . .

# Build from project root using Go workspace
WORKDIR /build

# Build the service binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s -extldflags \"-static\"' \
    -a -installsuffix cgo \
    -o [service-name] \
    ./services/[service-name]/cmd/main.go

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create working directory and non-root user
WORKDIR /root/
RUN adduser -D -g '' appuser

# Copy binary and set permissions
COPY --from=builder /build/[service-name] .
RUN chown appuser:appuser [service-name]

# Use non-root user
USER appuser

# Expose ports and set entrypoint
EXPOSE [ports]
ENTRYPOINT [\"./[service-name]\"]
```

### Key Features

- **Security**: All containers run as non-root users
- **Optimization**: Multi-stage builds minimize final image size
- **Consistency**: All services use the same Go version (1.23) and build patterns
- **Workspace Support**: Leverages Go workspace for shared package resolution
- **Static Linking**: Binaries are statically linked for better portability

## Go Workspace Configuration

The project uses a Go workspace (`go.work`) to manage multiple modules:

```go
go 1.23.0

toolchain go1.23.4

use (
    ./pkg/shared/auth
    ./pkg/shared/config
    ./pkg/shared/database
    ./pkg/shared/errors
    ./pkg/shared/logging
    ./pkg/shared/messaging
    ./pkg/shared/monitoring
    ./pkg/shared/utils
    ./services/analytics-service
    ./services/api-gateway
    ./services/auth-service
    ./services/blockchain-service
    ./services/governance-service
    ./services/member-service
    ./services/notification-service
    ./services/reciprocal-service
)
```

## Shared Packages

All shared packages in `pkg/shared/` have their own `go.mod` files:

- `pkg/shared/auth` - JWT authentication and authorization
- `pkg/shared/config` - Configuration management
- `pkg/shared/database` - Database utilities and connections
- `pkg/shared/errors` - Structured error handling
- `pkg/shared/logging` - Structured logging utilities
- `pkg/shared/messaging` - NATS message bus integration
- `pkg/shared/monitoring` - Prometheus metrics
- `pkg/shared/utils` - Common utility functions

## Build Scripts

### Module Management

- `scripts/tidy-modules.sh` - Tidies all Go modules (shared packages and services)

### Docker Build Testing

- `scripts/test-docker-builds.sh` - Tests all Docker builds with either Docker or Podman

Usage:
```bash
# Test with Podman (default)
./scripts/test-docker-builds.sh

# Test with Docker
DOCKER_CMD=docker ./scripts/test-docker-builds.sh
```

## Makefile Targets

The project includes comprehensive Makefile targets for building and testing:

### Individual Service Builds

#### Docker
```bash
make docker-build-api-gateway
make docker-build-auth-service
make docker-build-member-service
# ... etc for all services
```

#### Podman
```bash
make podman-build-api-gateway
make podman-build-auth-service
make podman-build-member-service
# ... etc for all services
```

### Bulk Operations

```bash
# Build all services
make docker-build-all
make podman-build-all

# Test all builds
make test-docker-builds-script
make test-podman-builds-script

# Tidy all modules
make tidy-modules
```

### Legacy Build Targets

The following targets are maintained for backward compatibility:

```bash
make build-images-podman       # Build all images with Podman
make test-build-images-podman  # Test all builds with Podman
make test-build-images         # Test all builds with Docker
```

## Container Registries

Images are tagged consistently:

- Local development: `reciprocal-[service-name]:[tag]`
- Testing: `reciprocal-[service-name]:test`

## Docker Ignore

The project includes a comprehensive `.dockerignore` file to optimize build contexts:

```dockerignore
# Build artifacts
bin/
target/
*.out

# Development files
.git/
.gitignore
.vscode/
.idea/
*.md

# Dependencies
node_modules/
vendor/

# Logs and temporary files
*.log
.DS_Store
*.tmp

# Test files
coverage.out
test-results/

# Documentation (except essential)
docs/
examples/
scripts/
!scripts/setup.sh

# Environment files
.env
.env.local
.env.development
.env.production

# Docker files (except Dockerfiles)
docker-compose*.yml
```

## Build Verification

All services have been verified to build successfully with the current setup. The API Gateway service has been thoroughly tested and builds without errors.

## Troubleshooting

### Common Issues

1. **Go Version Mismatch**: Ensure all Dockerfiles use `golang:1.23-alpine`
2. **Module Resolution**: The Go workspace resolves shared package dependencies
3. **Build Context**: All builds run from the project root to access shared packages

### Build Failures

If a service fails to build:

1. Check the service's `go.mod` for correct module path
2. Ensure shared package dependencies are properly referenced
3. Run `make tidy-modules` to refresh module dependencies
4. Check for syntax errors in the service code

## Security Considerations

- All containers run as non-root users
- Static binary compilation reduces attack surface
- Minimal base images (Alpine) reduce vulnerabilities
- No sensitive information in Dockerfiles or build context

## Performance Optimization

- Multi-stage builds minimize image size
- Docker layer caching optimizes rebuild times
- Go workspace reduces duplicate dependency downloads
- Static linking improves startup time

## Next Steps

1. Set up CI/CD pipeline integration
2. Configure container registry pushing
3. Add health check endpoints
4. Implement container orchestration (Kubernetes/Docker Compose)
5. Add monitoring and logging aggregation
6. Set up automated security scanning

## Support

For issues with the Docker setup:

1. Check this documentation
2. Run diagnostic scripts in `scripts/`
3. Check service-specific logs
4. Verify Go workspace configuration
5. Ensure all dependencies are properly resolved

---

**Last Updated**: December 2024
**Go Version**: 1.23
**Docker Pattern**: Multi-stage builds
**Status**: ✅ All services building successfully