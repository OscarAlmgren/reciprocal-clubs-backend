# Fixed Issues and Usage Guide

## Issues Fixed

### 1. **Network Configuration**
- ✅ Added proper systemd configuration to network quadlet file
- ✅ Added IPv6 disable and Install section

### 2. **Volume Mount Issues**
- ✅ Commented out non-existent volume mounts that would cause failures:
  - Fabric crypto-config directories
  - Prometheus configuration file
  - Grafana configuration directories
  - PostgreSQL init script
- ✅ Added notes explaining what needs to be created first

### 3. **Complex Makefile Parsing**
- ✅ Replaced complex shell parsing in Makefile with simpler script calls
- ✅ Makefile now delegates to shell scripts for reliability

### 4. **Development Workflow**
- ✅ Created `start-dev.sh` script for simplified development workflow
- ✅ Focuses on essential services only (PostgreSQL, Redis, NATS, MailHog)
- ✅ Provides step-by-step service building and starting

## Recommended Usage

### For Development (Recommended)

```bash
cd deployments/podman-quadlets

# Start essential infrastructure
./start-dev.sh start

# Check status
./start-dev.sh status

# Start specific services (requires Dockerfiles)
./start-dev.sh member-service
./start-dev.sh auth-service
./start-dev.sh api-gateway

# Stop everything
./start-dev.sh stop

# Clean up
./start-dev.sh clean
```

### For Full Production-Like Setup

```bash
cd deployments/podman-quadlets

# Full setup (may fail if Dockerfiles don't exist)
./podman-compose.sh up

# Infrastructure only
./podman-compose.sh infra

# Check status
./podman-compose.sh status
```

### Using Make (Simplified)

```bash
# From project root
make podman-up     # Uses start-dev.sh
make podman-status # Shows service status  
make podman-down   # Stops services
```

## What Works Now

✅ **Infrastructure Services**: PostgreSQL, Redis, NATS, MailHog
✅ **Network Creation**: Automatic network setup
✅ **Health Checks**: Wait for services to be ready
✅ **Status Monitoring**: Clear service status display
✅ **Clean Shutdown**: Proper service stopping
✅ **Volume Management**: Persistent data storage

## What Needs Additional Setup

⚠️ **Application Services**: Require Dockerfiles to be built
⚠️ **Fabric Services**: Need crypto-config generation first
⚠️ **Monitoring**: Need configuration files created

## Quick Start

1. **Start infrastructure**:
   ```bash
   ./start-dev.sh start
   ```

2. **Verify services**:
   ```bash
   ./start-dev.sh status
   ```

3. **Access services**:
   - PostgreSQL: `localhost:5432`
   - Redis: `localhost:6379`
   - NATS: `localhost:4222`
   - MailHog: `localhost:8025`

## Troubleshooting

If you encounter issues:

1. **Check Podman machine**:
   ```bash
   podman machine list
   podman machine start
   ```

2. **Check container logs**:
   ```bash
   podman logs reciprocal-postgres
   podman logs reciprocal-redis
   ```

3. **Clean start**:
   ```bash
   ./start-dev.sh clean
   ./start-dev.sh start
   ```

The setup is now more reliable and focused on development needs!