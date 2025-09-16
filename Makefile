# Reciprocal Clubs Platform - Main Makefile

.PHONY: help docker podman k8s build test clean

# Default target
help:
	@echo "Reciprocal Clubs Platform"
	@echo "========================"
	@echo ""
	@echo "Container Orchestration:"
	@echo "  docker-up       - Start with Docker Compose"
	@echo "  docker-down     - Stop Docker Compose"
	@echo "  docker-logs     - View Docker Compose logs"
	@echo "  docker-clean    - Clean Docker resources"
	@echo ""
	@echo "  podman-up       - Start with Podman"
	@echo "  podman-down     - Stop Podman services" 
	@echo "  podman-status   - Check Podman service status"
	@echo "  podman-clean    - Clean Podman resources"
	@echo ""
	@echo "  k8s-deploy      - Deploy to Kubernetes"
	@echo "  k8s-delete      - Delete from Kubernetes"
	@echo ""
	@echo "Development:"
	@echo "  build           - Build all services"
	@echo "  build-images-podman - Build all Docker images with Podman"
	@echo "  test-build-images - Test build all Docker images"
	@echo "  test-build-images-podman - Test build all images with Podman"
	@echo "  test            - Run all tests"
	@echo "  test-unit       - Run unit tests"
	@echo "  test-integration- Run integration tests"
	@echo "  lint            - Run linters"
	@echo "  format          - Format code"
	@echo ""
	@echo "Database:"
	@echo "  db-migrate      - Run database migrations"
	@echo "  db-seed         - Seed database with test data"
	@echo "  db-reset        - Reset database"
	@echo ""
	@echo "Utilities:"
	@echo "  clean           - Clean all build artifacts"
	@echo "  deps            - Install dependencies"
	@echo "  security-scan   - Run security scans"

# Docker Compose targets
docker-up:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

docker-down:
	@echo "Stopping Docker Compose services..."
	docker-compose down

docker-logs:
	@echo "Viewing Docker Compose logs..."
	docker-compose logs -f

docker-build:
	@echo "Building images with Docker Compose..."
	docker-compose build

docker-clean:
	@echo "Cleaning Docker resources..."
	docker-compose down -v --remove-orphans
	docker system prune -f

# Podman targets
podman-up:
	@echo "Starting services with Podman..."
	@cd deployments/podman-quadlets && ./podman-compose.sh up

podman-down:
	@echo "Stopping Podman services..."
	@cd deployments/podman-quadlets && ./podman-compose.sh down

podman-status:
	@echo "Checking Podman service status..."
	@cd deployments/podman-quadlets && ./podman-compose.sh status

podman-build:
	@echo "Building images with Podman..."
	@cd deployments/podman-quadlets && ./podman-compose.sh build

podman-clean:
	@echo "Cleaning Podman resources..."
	@cd deployments/podman-quadlets && ./podman-compose.sh clean

podman-logs:
	@if [ -z "$(SERVICE)" ]; then \
		echo "Usage: make podman-logs SERVICE=<service_name>"; \
		echo "Available services: postgres, redis, nats, etc."; \
	else \
		cd deployments/podman-quadlets && ./podman-compose.sh logs $(SERVICE); \
	fi

# Kubernetes targets
k8s-deploy:
	@echo "Deploying to Kubernetes..."
	kubectl apply -f deployments/k8s/

k8s-delete:
	@echo "Deleting from Kubernetes..."
	kubectl delete -f deployments/k8s/

k8s-status:
	@echo "Kubernetes status..."
	kubectl get pods,services,deployments

# Build targets
build:
	@echo "Building all services..."
	@for service in api-gateway auth-service member-service reciprocal-service blockchain-service notification-service analytics-service governance-service; do \
		echo "Building $$service..."; \
		cd services/$$service && go build -o bin/$$service ./cmd/main.go && cd ../..; \
	done

build-images:
	@echo "Building Docker images..."
	@$(MAKE) docker-build

# Build all service images with Podman
build-images-podman:
	@echo "Building all service images with Podman..."
	@for service in api-gateway auth-service member-service reciprocal-service blockchain-service notification-service analytics-service governance-service; do \
		echo "Building $$service..."; \
		podman build -t "localhost/reciprocal-$$service:latest" \
			-f "services/$$service/Dockerfile" \
			. || echo "Failed to build $$service"; \
	done

# Test build all images (without pushing)
test-build-images:
	@echo "Testing Docker image builds..."
	@for service in api-gateway auth-service member-service reciprocal-service blockchain-service notification-service analytics-service governance-service; do \
		echo "Testing build for $$service..."; \
		docker build --no-cache -t "test-reciprocal-$$service:latest" \
			-f "services/$$service/Dockerfile" \
			. && echo "✅ $$service build successful" || echo "❌ $$service build failed"; \
	done

# Test build all images with Podman
test-build-images-podman:
	@echo "Testing Podman image builds..."
	@for service in api-gateway auth-service member-service reciprocal-service blockchain-service notification-service analytics-service governance-service; do \
		echo "Testing build for $$service..."; \
		podman build --no-cache -t "test-reciprocal-$$service:latest" \
			-f "services/$$service/Dockerfile" \
			. && echo "✅ $$service build successful" || echo "❌ $$service build failed"; \
	done

# Clean test images
clean-test-images:
	@echo "Cleaning test images..."
	@for service in api-gateway auth-service member-service reciprocal-service blockchain-service notification-service analytics-service governance-service; do \
		docker rmi -f "test-reciprocal-$$service:latest" 2>/dev/null || true; \
	done

clean-test-images-podman:
	@echo "Cleaning test images with Podman..."
	@for service in api-gateway auth-service member-service reciprocal-service blockchain-service notification-service analytics-service governance-service; do \
		podman rmi -f "test-reciprocal-$$service:latest" 2>/dev/null || true; \
	done

# Test targets
test:
	@echo "Running all tests..."
	@$(MAKE) test-unit
	@$(MAKE) test-integration

test-unit:
	@echo "Running unit tests..."
	@for service in services/*/; do \
		echo "Testing $$service"; \
		cd "$$service" && go test -v ./... -short && cd ../..; \
	done

test-integration:
	@echo "Running integration tests..."
	go test -v ./tests/integration/...

test-coverage:
	@echo "Running tests with coverage..."
	@for service in services/*/; do \
		echo "Coverage for $$service"; \
		cd "$$service" && go test -v -coverprofile=coverage.out ./... && cd ../..; \
	done

# Linting and formatting
lint:
	@echo "Running linters..."
	@for service in services/*/; do \
		echo "Linting $$service"; \
		cd "$$service" && golangci-lint run && cd ../..; \
	done

format:
	@echo "Formatting code..."
	@for service in services/*/; do \
		echo "Formatting $$service"; \
		cd "$$service" && go fmt ./... && cd ../..; \
	done

# Database targets
db-migrate:
	@echo "Running database migrations..."
	@cd services/member-service && go run cmd/migrate/main.go

db-seed:
	@echo "Seeding database..."
	@cd services/member-service && go run cmd/seed/main.go

db-reset:
	@echo "Resetting database..."
	@cd services/member-service && go run cmd/reset/main.go

# Development utilities
deps:
	@echo "Installing dependencies..."
	@for service in services/*/; do \
		echo "Installing deps for $$service"; \
		cd "$$service" && go mod download && cd ../..; \
	done

tidy:
	@echo "Tidying Go modules..."
	@for service in services/*/; do \
		echo "Tidying $$service"; \
		cd "$$service" && go mod tidy && cd ../..; \
	done

# Security
security-scan:
	@echo "Running security scans..."
	@for service in services/*/; do \
		echo "Scanning $$service"; \
		cd "$$service" && gosec ./... && cd ../..; \
	done

vulnerability-check:
	@echo "Checking for vulnerabilities..."
	@for service in services/*/; do \
		echo "Checking $$service"; \
		cd "$$service" && govulncheck ./... && cd ../..; \
	done

# Cleanup
clean:
	@echo "Cleaning build artifacts..."
	@find . -name "bin" -type d -exec rm -rf {} + 2>/dev/null || true
	@find . -name "*.out" -type f -delete
	@find . -name ".DS_Store" -type f -delete

clean-all: clean docker-clean podman-clean
	@echo "Deep cleaning everything..."

# Environment setup
setup-dev:
	@echo "Setting up development environment..."
	@$(MAKE) deps
	@$(MAKE) build

# Health checks
health-check-docker:
	@echo "Checking Docker service health..."
	@docker-compose exec postgres pg_isready -U postgres || echo "PostgreSQL not ready"
	@docker-compose exec redis redis-cli ping || echo "Redis not ready"

health-check-podman:
	@echo "Checking Podman service health..."
	@cd deployments/podman-quadlets && ./podman-compose.sh health

# Service-specific targets
postgres-logs:
	@docker-compose logs -f postgres

redis-logs:
	@docker-compose logs -f redis

nats-logs:
	@docker-compose logs -f nats

# Fabric blockchain utilities
fabric-setup:
	@echo "Setting up Hyperledger Fabric network..."
	@cd services/blockchain-service/scripts && ./setup-fabric.sh

fabric-deploy-chaincode:
	@echo "Deploying chaincode..."
	@cd services/blockchain-service/scripts && ./deploy-chaincode.sh

fabric-clean:
	@echo "Cleaning Fabric network..."
	@cd services/blockchain-service/scripts && ./cleanup-fabric.sh

# Monitoring
logs-all:
	@docker-compose logs -f

logs-apps:
	@docker-compose logs -f api-gateway auth-service member-service reciprocal-service blockchain-service notification-service analytics-service governance-service

# Production utilities
backup-db:
	@echo "Backing up database..."
	@docker-compose exec postgres pg_dump -U postgres reciprocal_clubs > backup_$(shell date +%Y%m%d_%H%M%S).sql

restore-db:
	@if [ -z "$(FILE)" ]; then \
		echo "Usage: make restore-db FILE=<backup_file>"; \
	else \
		echo "Restoring database from $(FILE)..."; \
		docker-compose exec -T postgres psql -U postgres reciprocal_clubs < $(FILE); \
	fi

# Documentation
docs:
	@echo "Generating API documentation..."
	@cd services/api-gateway && swag init -g cmd/main.go

docs-serve:
	@echo "Serving documentation..."
	@cd docs && python3 -m http.server 8000

# Version and release
version:
	@echo "Current version: $(shell git describe --tags --always --dirty)"

tag:
	@if [ -z "$(TAG)" ]; then \
		echo "Usage: make tag TAG=v1.0.0"; \
	else \
		git tag -a $(TAG) -m "Release $(TAG)"; \
		git push origin $(TAG); \
	fi

# Development workflow helpers
dev-reset: clean-all podman-clean setup-dev podman-up
	@echo "Development environment reset complete"

dev-restart: podman-down podman-up
	@echo "Services restarted"

# Quick start targets
start: podman-up
stop: podman-down
restart: podman-down podman-up
status: podman-status

# CI/CD helpers
ci-test: deps test lint security-scan

ci-build: deps build build-images

ci-deploy-staging:
	@echo "Deploying to staging..."
	@# Add staging deployment commands

ci-deploy-prod:
	@echo "Deploying to production..."
	@# Add production deployment commands

# Enhanced Docker build targets for individual services
docker-build-api-gateway:
	docker build -f services/api-gateway/Dockerfile -t reciprocal-api-gateway .

docker-build-reciprocal-service:
	docker build -f services/reciprocal-service/Dockerfile -t reciprocal-reciprocal-service .

docker-build-auth-service:
	docker build -f services/auth-service/Dockerfile -t reciprocal-auth-service .

docker-build-member-service:
	docker build -f services/member-service/Dockerfile -t reciprocal-member-service .

docker-build-analytics-service:
	docker build -f services/analytics-service/Dockerfile -t reciprocal-analytics-service .

docker-build-blockchain-service:
	docker build -f services/blockchain-service/Dockerfile -t reciprocal-blockchain-service .

docker-build-governance-service:
	docker build -f services/governance-service/Dockerfile -t reciprocal-governance-service .

docker-build-notification-service:
	docker build -f services/notification-service/Dockerfile -t reciprocal-notification-service .

# Enhanced Podman build targets for individual services
podman-build-api-gateway:
	podman build -f services/api-gateway/Dockerfile -t reciprocal-api-gateway .

podman-build-reciprocal-service:
	podman build -f services/reciprocal-service/Dockerfile -t reciprocal-reciprocal-service .

podman-build-auth-service:
	podman build -f services/auth-service/Dockerfile -t reciprocal-auth-service .

podman-build-member-service:
	podman build -f services/member-service/Dockerfile -t reciprocal-member-service .

podman-build-analytics-service:
	podman build -f services/analytics-service/Dockerfile -t reciprocal-analytics-service .

podman-build-blockchain-service:
	podman build -f services/blockchain-service/Dockerfile -t reciprocal-blockchain-service .

podman-build-governance-service:
	podman build -f services/governance-service/Dockerfile -t reciprocal-governance-service .

podman-build-notification-service:
	podman build -f services/notification-service/Dockerfile -t reciprocal-notification-service .

# Enhanced Docker build targets - build all services
docker-build-all: docker-build-api-gateway docker-build-reciprocal-service docker-build-auth-service docker-build-member-service docker-build-analytics-service docker-build-blockchain-service docker-build-governance-service docker-build-notification-service
	@echo "All services built with Docker"

podman-build-all: podman-build-api-gateway podman-build-reciprocal-service podman-build-auth-service podman-build-member-service podman-build-analytics-service podman-build-blockchain-service podman-build-governance-service podman-build-notification-service
	@echo "All services built with Podman"

# Test Docker builds with comprehensive script
test-docker-builds-script:
	@echo "Testing all Docker builds with script..."
	@DOCKER_CMD=docker ./scripts/test-docker-builds.sh

test-podman-builds-script:
	@echo "Testing all Podman builds with script..."
	@DOCKER_CMD=podman ./scripts/test-docker-builds.sh

# Utility targets
tidy-modules:
	@echo "Tidying all Go modules including shared packages..."
	./scripts/tidy-modules.sh
