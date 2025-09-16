#!/bin/bash

# Simplified development startup script for Reciprocal Clubs Platform
# This script starts only the essential services needed for development

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NETWORK_NAME="reciprocal-clubs"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check Podman
check_podman() {
    if ! command -v podman &> /dev/null; then
        log_error "Podman is not installed. Install with: brew install podman"
        exit 1
    fi
    
    if ! podman system info &> /dev/null; then
        log_warning "Starting Podman machine..."
        podman machine start 2>/dev/null || {
            log_info "Initializing Podman machine..."
            podman machine init
            podman machine start
        }
    fi
}

# Create network
create_network() {
    if ! podman network exists "$NETWORK_NAME" 2>/dev/null; then
        log_info "Creating network: $NETWORK_NAME"
        podman network create "$NETWORK_NAME"
    fi
}

# Start essential infrastructure
start_infrastructure() {
    log_info "Starting infrastructure services..."
    
    # PostgreSQL
    log_info "Starting PostgreSQL..."
    podman run -d \
        --name reciprocal-postgres \
        --network "$NETWORK_NAME" \
        --replace \
        -p 5432:5432 \
        -e POSTGRES_DB=reciprocal_clubs \
        -e POSTGRES_USER=postgres \
        -e POSTGRES_PASSWORD=postgres \
        -v reciprocal-postgres-data:/var/lib/postgresql/data \
        postgres:15-alpine
    
    # Redis
    log_info "Starting Redis..."
    podman run -d \
        --name reciprocal-redis \
        --network "$NETWORK_NAME" \
        --replace \
        -p 6379:6379 \
        -v reciprocal-redis-data:/data \
        redis:7-alpine redis-server --appendonly yes
    
    # NATS
    log_info "Starting NATS..."
    podman run -d \
        --name reciprocal-nats \
        --network "$NETWORK_NAME" \
        --replace \
        -p 4222:4222 \
        -p 8222:8222 \
        -v reciprocal-nats-data:/data \
        nats:2.10-alpine \
        --js --sd /data --http_port 8222
    
    # MailHog for email testing
    log_info "Starting MailHog..."
    podman run -d \
        --name reciprocal-mailhog \
        --network "$NETWORK_NAME" \
        --replace \
        -p 8025:8025 \
        -p 1025:1025 \
        mailhog/mailhog
}

# Wait for services to be ready
wait_for_services() {
    log_info "Waiting for services to be ready..."
    
    # Wait for PostgreSQL
    for i in {1..30}; do
        if podman exec reciprocal-postgres pg_isready -U postgres &> /dev/null; then
            log_success "PostgreSQL is ready"
            break
        fi
        if [ $i -eq 30 ]; then
            log_error "PostgreSQL failed to start"
            exit 1
        fi
        sleep 2
    done
    
    # Wait for Redis
    for i in {1..10}; do
        if podman exec reciprocal-redis redis-cli ping | grep -q PONG; then
            log_success "Redis is ready"
            break
        fi
        if [ $i -eq 10 ]; then
            log_error "Redis failed to start"
            exit 1
        fi
        sleep 1
    done
    
    # Wait for NATS
    for i in {1..10}; do
        if curl -s http://localhost:8222/healthz &> /dev/null; then
            log_success "NATS is ready"
            break
        fi
        if [ $i -eq 10 ]; then
            log_warning "NATS health check failed, but continuing..."
            break
        fi
        sleep 1
    done
}

# Build a specific service image
build_service() {
    local service=$1
    log_info "Building $service..."
    
    if [ -f "$PROJECT_ROOT/services/$service/Dockerfile" ]; then
        podman build -t "localhost/reciprocal-$service:latest" \
            -f "$PROJECT_ROOT/services/$service/Dockerfile" \
            "$PROJECT_ROOT" || {
            log_warning "Failed to build $service"
            return 1
        }
    else
        log_warning "Dockerfile not found for $service"
        return 1
    fi
}

# Start a specific service
start_service() {
    local service=$1
    local port=$2
    local grpc_port=$3
    
    log_info "Starting $service..."
    
    # Build the service first
    if ! build_service "$service"; then
        log_warning "Skipping $service due to build failure"
        return 1
    fi
    
    podman run -d \
        --name "reciprocal-$service" \
        --network "$NETWORK_NAME" \
        --replace \
        -p "$port:$port" \
        -p "$grpc_port:$grpc_port" \
        -e "${service^^}_SERVICE_PORT=$port" \
        -e "${service^^}_SERVICE_GRPC_PORT=$grpc_port" \
        -e "${service^^}_DATABASE_HOST=reciprocal-postgres" \
        -e "${service^^}_DATABASE_PASSWORD=postgres" \
        -e "${service^^}_NATS_URL=nats://reciprocal-nats:4222" \
        -e "${service^^}_REDIS_HOST=reciprocal-redis" \
        "localhost/reciprocal-$service:latest" || {
        log_warning "Failed to start $service"
        return 1
    }
}

# Show status
show_status() {
    log_info "Service Status:"
    echo "==============="
    
    local services=(
        "reciprocal-postgres"
        "reciprocal-redis"
        "reciprocal-nats"
        "reciprocal-mailhog"
    )
    
    for service in "${services[@]}"; do
        if podman ps --filter "name=$service" --format "{{.Names}} {{.Status}}" | grep -q "$service"; then
            status=$(podman ps --filter "name=$service" --format "{{.Status}}")
            printf "%-25s %s\n" "$service:" "$status"
        else
            printf "%-25s %s\n" "$service:" "Not running"
        fi
    done
}

# Stop all services
stop_services() {
    log_info "Stopping all services..."
    
    local containers=(
        "reciprocal-postgres"
        "reciprocal-redis"
        "reciprocal-nats"
        "reciprocal-mailhog"
        "reciprocal-member-service"
        "reciprocal-auth-service"
        "reciprocal-api-gateway"
    )
    
    for container in "${containers[@]}"; do
        if podman ps -a --filter "name=$container" --format "{{.Names}}" | grep -q "$container"; then
            log_info "Stopping $container..."
            podman stop "$container" 2>/dev/null || true
        fi
    done
}

# Clean everything
clean_all() {
    log_info "Cleaning up..."
    stop_services
    
    local containers=(
        "reciprocal-postgres"
        "reciprocal-redis"
        "reciprocal-nats"
        "reciprocal-mailhog"
        "reciprocal-member-service"
        "reciprocal-auth-service"
        "reciprocal-api-gateway"
    )
    
    for container in "${containers[@]}"; do
        podman rm -f "$container" 2>/dev/null || true
    done
    
    # Optionally remove volumes (uncomment if needed)
    # podman volume rm -f reciprocal-postgres-data reciprocal-redis-data reciprocal-nats-data 2>/dev/null || true
    
    log_success "Cleanup completed"
}

# Main script
case "$1" in
    "start"|"up")
        check_podman
        create_network
        start_infrastructure
        wait_for_services
        log_success "Infrastructure services started!"
        echo
        echo "Available services:"
        echo "- PostgreSQL: localhost:5432"
        echo "- Redis: localhost:6379"
        echo "- NATS: localhost:4222 (HTTP: localhost:8222)"
        echo "- MailHog: localhost:8025"
        echo
        echo "To start application services:"
        echo "./start-dev.sh member-service"
        echo "./start-dev.sh auth-service"
        ;;
    "stop"|"down")
        stop_services
        ;;
    "clean")
        clean_all
        ;;
    "status")
        show_status
        ;;
    "member-service")
        start_service "member-service" 8082 9082
        ;;
    "auth-service")
        start_service "auth-service" 8081 9081
        ;;
    "api-gateway")
        start_service "api-gateway" 8080 9080
        ;;
    *)
        echo "Reciprocal Clubs Platform - Development Startup Script"
        echo ""
        echo "Usage: $0 {start|stop|clean|status|member-service|auth-service|api-gateway}"
        echo ""
        echo "Commands:"
        echo "  start           - Start infrastructure services"
        echo "  stop            - Stop all services"
        echo "  clean           - Remove all containers"
        echo "  status          - Show service status"
        echo "  member-service  - Start member service"
        echo "  auth-service    - Start auth service"
        echo "  api-gateway     - Start API gateway"
        exit 1
        ;;
esac