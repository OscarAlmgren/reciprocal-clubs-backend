#!/bin/bash

# Reciprocal Clubs Platform - Podman Compose Script for macOS
# This script provides Docker Compose-like functionality using Podman

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
NETWORK_NAME="reciprocal-clubs"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Check if Podman is installed
check_podman() {
    if ! command -v podman &> /dev/null; then
        log_error "Podman is not installed. Please install Podman first."
        echo "On macOS: brew install podman"
        exit 1
    fi
    
    # Check if Podman machine is running
    if ! podman system info &> /dev/null; then
        log_warning "Podman machine is not running. Starting Podman machine..."
        podman machine start 2>/dev/null || {
            log_info "Initializing new Podman machine..."
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
        log_success "Network created"
    else
        log_info "Network $NETWORK_NAME already exists"
    fi
}

# Build application images
build_images() {
    log_info "Building application images..."
    
    local services=(
        "api-gateway"
        "auth-service" 
        "member-service"
        "reciprocal-service"
        "blockchain-service"
        "notification-service"
        "analytics-service"
        "governance-service"
    )
    
    for service in "${services[@]}"; do
        log_info "Building $service..."
        podman build -t "localhost/reciprocal-$service:latest" \
            -f "$PROJECT_ROOT/services/$service/Dockerfile" \
            "$PROJECT_ROOT" || {
            log_warning "Failed to build $service (Dockerfile might not exist)"
        }
    done
    
    log_success "Image building completed"
}

# Start infrastructure services
start_infra() {
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
        -v "$PROJECT_ROOT/scripts/init-db.sql:/docker-entrypoint-initdb.d/init-db.sql:ro" \
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
        -p 6222:6222 \
        -p 8222:8222 \
        -v reciprocal-nats-data:/data \
        nats:2.10-alpine \
        --js --sd /data --cluster_name reciprocal-clubs \
        --cluster nats://0.0.0.0:6222 --routes nats://0.0.0.0:6222 \
        --http_port 8222

    # Hanko Migration
    log_info "Starting Hanko Migration..."
    podman run --rm \
        --name reciprocal-hanko-migrate \
        --network "$NETWORK_NAME" \
        -v "$PROJECT_ROOT/config/podman/hanko-config.yaml:/etc/config/config.yaml:ro" \
        teamhanko/hanko:latest \
        migrate up --config /etc/config/config.yaml || {
        log_warning "Hanko migration failed or already completed"
    }

    # Hanko Authentication Service
    log_info "Starting Hanko Authentication Service..."
    podman run -d \
        --name reciprocal-hanko \
        --network "$NETWORK_NAME" \
        --replace \
        -p 8000:8000 \
        -p 8001:8001 \
        -e PASSWORD_ENABLED=true \
        -v "$PROJECT_ROOT/config/podman/hanko-config.yaml:/etc/config/config.yaml:ro" \
        teamhanko/hanko:latest \
        serve --config /etc/config/config.yaml all

    # MailHog
    log_info "Starting MailHog..."
    podman run -d \
        --name reciprocal-mailhog \
        --network "$NETWORK_NAME" \
        --replace \
        -p 8025:8025 \
        -p 1025:1025 \
        mailhog/mailhog
}

# Start Hyperledger Fabric services
start_fabric() {
    log_info "Starting Hyperledger Fabric services..."
    
    # Fabric CA
    log_info "Starting Fabric CA..."
    podman run -d \
        --name reciprocal-fabric-ca \
        --network "$NETWORK_NAME" \
        --replace \
        -p 7054:7054 \
        -e FABRIC_CA_HOME=/etc/hyperledger/fabric-ca-server \
        -e FABRIC_CA_SERVER_CA_NAME=ca-org1 \
        -e FABRIC_CA_SERVER_TLS_ENABLED=false \
        -e FABRIC_CA_SERVER_PORT=7054 \
        -v reciprocal-fabric-ca:/etc/hyperledger/fabric-ca-server \
        -v "$PROJECT_ROOT/fabric/ca:/etc/hyperledger/fabric-ca-server-config:ro" \
        hyperledger/fabric-ca:1.5 \
        sh -c 'fabric-ca-server start -b admin:adminpw -d'
    
    sleep 5
    
    # Fabric Orderer
    log_info "Starting Fabric Orderer..."
    podman run -d \
        --name reciprocal-orderer \
        --network "$NETWORK_NAME" \
        --replace \
        -p 7050:7050 \
        -e FABRIC_LOGGING_SPEC=INFO \
        -e ORDERER_GENERAL_LISTENADDRESS=0.0.0.0 \
        -e ORDERER_GENERAL_LISTENPORT=7050 \
        -e ORDERER_GENERAL_GENESISMETHOD=file \
        -e ORDERER_GENERAL_GENESISFILE=/var/hyperledger/orderer/orderer.genesis.block \
        -e ORDERER_GENERAL_LOCALMSPID=OrdererMSP \
        -e ORDERER_GENERAL_LOCALMSPDIR=/var/hyperledger/orderer/msp \
        -e ORDERER_GENERAL_TLS_ENABLED=false \
        -w /opt/gopath/src/github.com/hyperledger/fabric \
        -v reciprocal-fabric-orderer:/var/hyperledger/production/orderer \
        -v "$PROJECT_ROOT/fabric/config/genesis.block:/var/hyperledger/orderer/orderer.genesis.block:ro" \
        -v "$PROJECT_ROOT/fabric/crypto-config/ordererOrganizations/reciprocal-clubs.com/orderers/orderer.reciprocal-clubs.com/msp:/var/hyperledger/orderer/msp:ro" \
        -v "$PROJECT_ROOT/fabric/crypto-config/ordererOrganizations/reciprocal-clubs.com/orderers/orderer.reciprocal-clubs.com/tls:/var/hyperledger/orderer/tls:ro" \
        hyperledger/fabric-orderer:2.5 orderer || {
        log_warning "Failed to start Fabric Orderer (crypto-config might not exist)"
    }
    
    sleep 5
    
    # Fabric Peer
    log_info "Starting Fabric Peer..."
    podman run -d \
        --name reciprocal-peer0-org1 \
        --network "$NETWORK_NAME" \
        --replace \
        -p 7051:7051 \
        -p 7052:7052 \
        -e CORE_VM_ENDPOINT=unix:///host/var/run/docker.sock \
        -e CORE_VM_DOCKER_HOSTCONFIG_NETWORKMODE="$NETWORK_NAME" \
        -e FABRIC_LOGGING_SPEC=INFO \
        -e CORE_PEER_TLS_ENABLED=false \
        -e CORE_PEER_GOSSIP_USELEADERELECTION=true \
        -e CORE_PEER_GOSSIP_ORGLEADER=false \
        -e CORE_PEER_PROFILE_ENABLED=true \
        -e CORE_PEER_ID=peer0.org1.reciprocal-clubs.com \
        -e CORE_PEER_ADDRESS=peer0.org1.reciprocal-clubs.com:7051 \
        -e CORE_PEER_LISTENADDRESS=0.0.0.0:7051 \
        -e CORE_PEER_CHAINCODEADDRESS=peer0.org1.reciprocal-clubs.com:7052 \
        -e CORE_PEER_CHAINCODELISTENADDRESS=0.0.0.0:7052 \
        -e CORE_PEER_GOSSIP_BOOTSTRAP=peer0.org1.reciprocal-clubs.com:7051 \
        -e CORE_PEER_GOSSIP_EXTERNALENDPOINT=peer0.org1.reciprocal-clubs.com:7051 \
        -e CORE_PEER_LOCALMSPID=Org1MSP \
        -w /opt/gopath/src/github.com/hyperledger/fabric/peer \
        -v /var/run/docker.sock:/host/var/run/docker.sock \
        -v reciprocal-fabric-peer:/var/hyperledger/production \
        -v "$PROJECT_ROOT/fabric/crypto-config/peerOrganizations/org1.reciprocal-clubs.com/peers/peer0.org1.reciprocal-clubs.com/msp:/etc/hyperledger/fabric/msp:ro" \
        -v "$PROJECT_ROOT/fabric/crypto-config/peerOrganizations/org1.reciprocal-clubs.com/peers/peer0.org1.reciprocal-clubs.com/tls:/etc/hyperledger/fabric/tls:ro" \
        hyperledger/fabric-peer:2.5 peer node start || {
        log_warning "Failed to start Fabric Peer (crypto-config might not exist)"
    }
}

# Start application services
start_apps() {
    log_info "Starting application services..."
    
    # API Gateway
    podman run -d \
        --name reciprocal-api-gateway \
        --network "$NETWORK_NAME" \
        --replace \
        -p 8080:8080 \
        -p 9080:9090 \
        -e API_GATEWAY_SERVICE_PORT=8080 \
        -e API_GATEWAY_SERVICE_GRPC_PORT=9090 \
        -e API_GATEWAY_DATABASE_HOST=reciprocal-postgres \
        -e API_GATEWAY_DATABASE_PASSWORD=postgres \
        -e API_GATEWAY_NATS_URL=nats://reciprocal-nats:4222 \
        -e API_GATEWAY_REDIS_HOST=reciprocal-redis \
        -e API_GATEWAY_AUTH_JWT_SECRET=your-secret-key \
        localhost/reciprocal-api-gateway:latest || {
        log_warning "Failed to start API Gateway (image might not exist)"
    }
    
    # Auth Service
    podman run -d \
        --name reciprocal-auth-service \
        --network "$NETWORK_NAME" \
        --replace \
        -p 8081:8081 \
        -p 9081:9091 \
        -e AUTH_SERVICE_SERVICE_PORT=8081 \
        -e AUTH_SERVICE_SERVICE_GRPC_PORT=9091 \
        -e AUTH_SERVICE_DATABASE_HOST=reciprocal-postgres \
        -e AUTH_SERVICE_DATABASE_PASSWORD=postgres \
        -e AUTH_SERVICE_NATS_URL=nats://reciprocal-nats:4222 \
        -e AUTH_SERVICE_REDIS_HOST=reciprocal-redis \
        -e AUTH_SERVICE_AUTH_JWT_SECRET=your-secret-key \
        -e AUTH_SERVICE_HANKO_BASE_URL=http://reciprocal-hanko:8000 \
        -e AUTH_SERVICE_HANKO_API_KEY= \
        localhost/reciprocal-auth-service:latest || {
        log_warning "Failed to start Auth Service (image might not exist)"
    }
    
    # Member Service
    podman run -d \
        --name reciprocal-member-service \
        --network "$NETWORK_NAME" \
        --replace \
        -p 8082:8082 \
        -p 9082:9092 \
        -e MEMBER_SERVICE_SERVICE_PORT=8082 \
        -e MEMBER_SERVICE_SERVICE_GRPC_PORT=9092 \
        -e MEMBER_SERVICE_DATABASE_HOST=reciprocal-postgres \
        -e MEMBER_SERVICE_DATABASE_PASSWORD=postgres \
        -e MEMBER_SERVICE_NATS_URL=nats://reciprocal-nats:4222 \
        -e MEMBER_SERVICE_REDIS_HOST=reciprocal-redis \
        localhost/reciprocal-member-service:latest || {
        log_warning "Failed to start Member Service (image might not exist)"
    }
    
    # Additional services would be added here...
    # (Reciprocal, Blockchain, Notification, Analytics, Governance services)
}

# Start monitoring services
start_monitoring() {
    log_info "Starting monitoring services..."
    
    # Prometheus
    if [ -f "$PROJECT_ROOT/monitoring/prometheus.yml" ]; then
        podman run -d \
            --name reciprocal-prometheus \
            --network "$NETWORK_NAME" \
            --replace \
            -p 9090:9090 \
            -v "$PROJECT_ROOT/monitoring/prometheus.yml:/etc/prometheus/prometheus.yml:ro" \
            -v reciprocal-prometheus-data:/prometheus \
            prom/prometheus:latest \
            --config.file=/etc/prometheus/prometheus.yml \
            --storage.tsdb.path=/prometheus \
            --web.console.libraries=/etc/prometheus/console_libraries \
            --web.console.templates=/etc/prometheus/consoles \
            --storage.tsdb.retention.time=200h \
            --web.enable-lifecycle
    else
        log_warning "Prometheus config not found, skipping..."
    fi
    
    # Grafana
    podman run -d \
        --name reciprocal-grafana \
        --network "$NETWORK_NAME" \
        --replace \
        -p 3000:3000 \
        -e GF_SECURITY_ADMIN_PASSWORD=admin \
        -v reciprocal-grafana-data:/var/lib/grafana \
        grafana/grafana:latest
    
    # Jaeger
    podman run -d \
        --name reciprocal-jaeger \
        --network "$NETWORK_NAME" \
        --replace \
        -p 16686:16686 \
        -p 14268:14268 \
        -e COLLECTOR_OTLP_ENABLED=true \
        jaegertracing/all-in-one:1.50
}

# Stop all services
stop_all() {
    log_info "Stopping all services..."
    
    local containers=(
        "reciprocal-api-gateway"
        "reciprocal-auth-service"
        "reciprocal-member-service"
        "reciprocal-reciprocal-service"
        "reciprocal-blockchain-service"
        "reciprocal-notification-service"
        "reciprocal-analytics-service"
        "reciprocal-governance-service"
        "reciprocal-prometheus"
        "reciprocal-grafana"
        "reciprocal-jaeger"
        "reciprocal-mailhog"
        "reciprocal-peer0-org1"
        "reciprocal-orderer"
        "reciprocal-fabric-ca"
        "reciprocal-hanko"
        "reciprocal-nats"
        "reciprocal-redis"
        "reciprocal-postgres"
    )
    
    for container in "${containers[@]}"; do
        if podman ps -a --filter "name=$container" --format "{{.Names}}" | grep -q "$container"; then
            log_info "Stopping $container..."
            podman stop "$container" 2>/dev/null || true
        fi
    done
    
    log_success "All services stopped"
}

# Remove all containers and volumes
clean() {
    log_info "Cleaning up containers and volumes..."
    
    stop_all
    
    # Remove containers
    local containers=(
        "reciprocal-api-gateway"
        "reciprocal-auth-service"
        "reciprocal-member-service"
        "reciprocal-reciprocal-service"
        "reciprocal-blockchain-service"
        "reciprocal-notification-service"
        "reciprocal-analytics-service"
        "reciprocal-governance-service"
        "reciprocal-prometheus"
        "reciprocal-grafana"
        "reciprocal-jaeger"
        "reciprocal-mailhog"
        "reciprocal-peer0-org1"
        "reciprocal-orderer"
        "reciprocal-fabric-ca"
        "reciprocal-hanko"
        "reciprocal-nats"
        "reciprocal-redis"
        "reciprocal-postgres"
    )
    
    for container in "${containers[@]}"; do
        podman rm -f "$container" 2>/dev/null || true
    done
    
    # Remove volumes
    local volumes=(
        "reciprocal-postgres-data"
        "reciprocal-redis-data"
        "reciprocal-nats-data"
        "reciprocal-fabric-ca"
        "reciprocal-fabric-orderer"
        "reciprocal-fabric-peer"
        "reciprocal-prometheus-data"
        "reciprocal-grafana-data"
    )
    
    for volume in "${volumes[@]}"; do
        podman volume rm -f "$volume" 2>/dev/null || true
    done
    
    # Remove network
    podman network rm -f "$NETWORK_NAME" 2>/dev/null || true
    
    log_success "Cleanup completed"
}

# Show status of all services
status() {
    log_info "Service Status:"
    echo "==============="
    
    local services=(
        "reciprocal-postgres"
        "reciprocal-redis"
        "reciprocal-nats"
        "reciprocal-hanko"
        "reciprocal-mailhog"
        "reciprocal-fabric-ca"
        "reciprocal-orderer"
        "reciprocal-peer0-org1"
        "reciprocal-api-gateway"
        "reciprocal-auth-service"
        "reciprocal-member-service"
        "reciprocal-reciprocal-service"
        "reciprocal-blockchain-service"
        "reciprocal-notification-service"
        "reciprocal-analytics-service"
        "reciprocal-governance-service"
        "reciprocal-prometheus"
        "reciprocal-grafana"
        "reciprocal-jaeger"
    )
    
    for service in "${services[@]}"; do
        if podman ps --filter "name=$service" --format "{{.Names}} {{.Status}}" | grep -q "$service"; then
            status=$(podman ps --filter "name=$service" --format "{{.Status}}")
            printf "%-30s %s\n" "$service:" "$status"
        else
            printf "%-30s %s\n" "$service:" "Not running"
        fi
    done
}

# Show logs for a specific service
logs() {
    local service=$1
    if [ -z "$service" ]; then
        log_error "Please specify a service name"
        echo "Usage: $0 logs <service_name>"
        return 1
    fi
    
    local container_name="reciprocal-$service"
    if podman ps -a --filter "name=$container_name" --format "{{.Names}}" | grep -q "$container_name"; then
        podman logs -f "$container_name"
    else
        log_error "Container $container_name not found"
    fi
}

# Health check
health_check() {
    log_info "Checking service health..."
    echo "=========================="
    
    local services=("postgres" "redis" "nats" "hanko")
    
    for service in "${services[@]}"; do
        local container_name="reciprocal-$service"
        echo -n "$service: "
        
        if podman exec "$container_name" echo "OK" 2>/dev/null >/dev/null; then
            echo -e "${GREEN}Healthy${NC}"
        else
            echo -e "${RED}Unhealthy${NC}"
        fi
    done
}

# Main script logic
case "$1" in
    "up"|"start")
        check_podman
        create_network
        build_images
        start_infra
        sleep 15
        start_fabric
        sleep 10
        start_apps
        start_monitoring
        log_success "All services started!"
        ;;
    "down"|"stop")
        stop_all
        ;;
    "build")
        check_podman
        build_images
        ;;
    "infra")
        check_podman
        create_network
        start_infra
        ;;
    "fabric")
        check_podman
        create_network
        start_fabric
        ;;
    "apps")
        check_podman
        create_network
        build_images
        start_apps
        ;;
    "monitoring")
        check_podman
        create_network
        start_monitoring
        ;;
    "clean")
        clean
        ;;
    "status"|"ps")
        status
        ;;
    "logs")
        logs "$2"
        ;;
    "health")
        health_check
        ;;
    "restart")
        stop_all
        sleep 5
        check_podman
        create_network
        start_infra
        sleep 15
        start_fabric
        sleep 10
        start_apps
        start_monitoring
        ;;
    *)
        echo "Reciprocal Clubs Platform - Podman Compose Script"
        echo ""
        echo "Usage: $0 {up|down|build|infra|fabric|apps|monitoring|clean|status|logs|health|restart}"
        echo ""
        echo "Commands:"
        echo "  up          - Start all services"
        echo "  down        - Stop all services"
        echo "  build       - Build application images"
        echo "  infra       - Start infrastructure services only"
        echo "  fabric      - Start Hyperledger Fabric services only"
        echo "  apps        - Start application services only"
        echo "  monitoring  - Start monitoring services only"
        echo "  clean       - Remove all containers and volumes"
        echo "  status      - Show status of all services"
        echo "  logs <svc>  - Show logs for a specific service"
        echo "  health      - Check health of services"
        echo "  restart     - Restart all services"
        exit 1
        ;;
esac