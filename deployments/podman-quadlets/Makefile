# Reciprocal Clubs Platform - Podman Quadlets Makefile

.PHONY: help setup-podman build-images start-all stop-all restart-all status clean logs

# Default target
help:
	@echo "Reciprocal Clubs Platform - Podman Quadlets"
	@echo ""
	@echo "Available targets:"
	@echo "  setup-podman     - Set up Podman quadlets directory and copy files"
	@echo "  build-images     - Build all application Docker images"
	@echo "  start-all        - Start all services"
	@echo "  start-infra      - Start only infrastructure services"
	@echo "  start-fabric     - Start Hyperledger Fabric services"
	@echo "  start-apps       - Start application services"
	@echo "  start-monitoring - Start monitoring services"
	@echo "  stop-all         - Stop all services"
	@echo "  restart-all      - Restart all services"
	@echo "  status           - Show status of all services"
	@echo "  logs SERVICE     - Show logs for a specific service"
	@echo "  clean            - Clean up all containers and volumes"
	@echo "  reload-quadlets  - Reload systemd quadlet configuration"

# Variables
QUADLETS_DIR := ~/.config/containers/systemd
PWD := $(shell pwd)

# Infrastructure services
INFRA_SERVICES := postgres redis nats mailhog

# Hyperledger Fabric services
FABRIC_SERVICES := fabric-ca fabric-orderer fabric-peer

# Application services
APP_SERVICES := api-gateway auth-service member-service reciprocal-service \
                blockchain-service notification-service analytics-service \
                governance-service

# Monitoring services
MONITORING_SERVICES := prometheus grafana jaeger

# All services
ALL_SERVICES := $(INFRA_SERVICES) $(FABRIC_SERVICES) $(APP_SERVICES) $(MONITORING_SERVICES)

setup-podman:
	@echo "Setting up Podman quadlets..."
	@mkdir -p $(QUADLETS_DIR)
	@cp *.container $(QUADLETS_DIR)/ 2>/dev/null || true
	@cp *.network $(QUADLETS_DIR)/ 2>/dev/null || true
	@echo "Quadlet files copied to $(QUADLETS_DIR)"
	@echo "Reloading systemd user daemon..."
	@systemctl --user daemon-reload 2>/dev/null || echo "Note: systemctl not available on macOS"

build-images:
	@echo "Building all application images..."
	@cd ../.. && podman build -t localhost/reciprocal-api-gateway:latest -f services/api-gateway/Dockerfile .
	@cd ../.. && podman build -t localhost/reciprocal-auth-service:latest -f services/auth-service/Dockerfile .
	@cd ../.. && podman build -t localhost/reciprocal-member-service:latest -f services/member-service/Dockerfile .
	@cd ../.. && podman build -t localhost/reciprocal-reciprocal-service:latest -f services/reciprocal-service/Dockerfile .
	@cd ../.. && podman build -t localhost/reciprocal-blockchain-service:latest -f services/blockchain-service/Dockerfile .
	@cd ../.. && podman build -t localhost/reciprocal-notification-service:latest -f services/notification-service/Dockerfile .
	@cd ../.. && podman build -t localhost/reciprocal-analytics-service:latest -f services/analytics-service/Dockerfile .
	@cd ../.. && podman build -t localhost/reciprocal-governance-service:latest -f services/governance-service/Dockerfile .

start-all: setup-podman
	@echo "Starting all services..."
	@echo "Use ./podman-compose.sh up for full deployment"
	@echo "Or ./start-dev.sh start for development"
	@./start-dev.sh start

start-infra:
	@echo "Starting infrastructure services..."
	@echo "Use ./start-dev.sh start instead of make for better reliability"
	@./start-dev.sh start

start-fabric:
	@echo "Starting Hyperledger Fabric services..."
	@for service in $(FABRIC_SERVICES); do \
		echo "Starting $$service..."; \
		podman run -d --name reciprocal-$$service --replace $$(cat $$service.container | grep -E '^(Image|PublishPort|Volume|Environment|Network|Exec|WorkingDir)' | sed 's/^Image=/--image /' | sed 's/^PublishPort=/-p /' | sed 's/^Volume=/-v /' | sed 's/^Environment=/-e /' | sed 's/^Network=/--network /' | sed 's/^Exec=//' | sed 's/^WorkingDir=/-w /' | tr '\n' ' '); \
	done

start-apps: build-images
	@echo "Starting application services..."
	@for service in $(APP_SERVICES); do \
		echo "Starting $$service..."; \
		podman run -d --name reciprocal-$$service --replace $$(cat $$service.container | grep -E '^(Image|PublishPort|Environment|Network)' | sed 's/^Image=/--image /' | sed 's/^PublishPort=/-p /' | sed 's/^Environment=/-e /' | sed 's/^Network=/--network /' | tr '\n' ' '); \
	done

start-monitoring:
	@echo "Starting monitoring services..."
	@for service in $(MONITORING_SERVICES); do \
		echo "Starting $$service..."; \
		podman run -d --name reciprocal-$$service --replace $$(cat $$service.container | grep -E '^(Image|PublishPort|Volume|Environment|Network|Exec)' | sed 's/^Image=/--image /' | sed 's/^PublishPort=/-p /' | sed 's/^Volume=/-v /' | sed 's/^Environment=/-e /' | sed 's/^Network=/--network /' | sed 's/^Exec=//' | tr '\n' ' '); \
	done

stop-all:
	@echo "Stopping all services..."
	@./start-dev.sh stop

restart-all: stop-all start-all

status:
	@./start-dev.sh status

logs:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make logs SERVICE=<service_name>"; \
		echo "Available services: $(ALL_SERVICES)"; \
	else \
		podman logs -f reciprocal-$(SERVICE); \
	fi

clean:
	@echo "Cleaning up containers and volumes..."
	@for service in $(ALL_SERVICES); do \
		podman rm -f reciprocal-$$service 2>/dev/null || true; \
	done
	@podman volume rm -f $$(podman volume ls -q | grep reciprocal) 2>/dev/null || true
	@podman network rm reciprocal-clubs 2>/dev/null || true
	@echo "Cleanup complete"

reload-quadlets:
	@echo "Reloading systemd quadlet configuration..."
	@systemctl --user daemon-reload 2>/dev/null || echo "Note: systemctl not available on macOS"

# Utility targets
network:
	@podman network create reciprocal-clubs 2>/dev/null || echo "Network already exists"

# Health check
health-check:
	@echo "Checking service health..."
	@echo "=========================="
	@for service in postgres redis nats; do \
		echo -n "$$service: "; \
		if podman exec reciprocal-$$service echo "OK" 2>/dev/null; then \
			echo "Healthy"; \
		else \
			echo "Unhealthy"; \
		fi; \
	done

# Individual service controls
define service-target
$(1)-start:
	@echo "Starting $(1)..."
	@podman run -d --name reciprocal-$(1) --replace $$(cat $(1).container | grep -E '^(Image|PublishPort|Volume|Environment|Network|Exec)' | sed 's/^Image=/--image /' | sed 's/^PublishPort=/-p /' | sed 's/^Volume=/-v /' | sed 's/^Environment=/-e /' | sed 's/^Network=/--network /' | sed 's/^Exec=//' | tr '\n' ' ')

$(1)-stop:
	@echo "Stopping $(1)..."
	@podman stop reciprocal-$(1) 2>/dev/null || true

$(1)-logs:
	@podman logs -f reciprocal-$(1)

$(1)-shell:
	@podman exec -it reciprocal-$(1) /bin/sh
endef

# Generate individual service targets
$(foreach service,$(ALL_SERVICES),$(eval $(call service-target,$(service))))