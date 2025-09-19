# Analytics Service Makefile

.PHONY: help build test test-unit test-integration test-bench coverage clean proto lint fmt vet deps run dev docker

# Default target
help: ## Show this help message
	@echo "Analytics Service - Available Commands:"
	@echo "====================================="
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Build targets
build: ## Build the analytics service binary
	@echo "🔨 Building analytics service..."
	go build -o bin/analytics-service ./cmd

build-docker: ## Build Docker image
	@echo "🐳 Building Docker image..."
	docker build -t analytics-service:latest .

# Test targets
test: ## Run all tests
	@echo "🧪 Running all tests..."
	go run test_runner.go

test-unit: ## Run unit tests only
	@echo "🧪 Running unit tests..."
	go test -v -race -cover ./...

test-integration: ## Run integration tests
	@echo "🧪 Running integration tests..."
	go test -v -tags=integration ./...

test-bench: ## Run benchmark tests
	@echo "📊 Running benchmark tests..."
	go test -bench=. -benchmem ./...

coverage: ## Generate test coverage report
	@echo "📊 Generating coverage report..."
	go test -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-watch: ## Run tests in watch mode (requires entr)
	@echo "👀 Running tests in watch mode..."
	find . -name "*.go" | entr -c make test-unit

# Code quality targets
lint: ## Run linter
	@echo "🔍 Running linter..."
	golangci-lint run

fmt: ## Format code
	@echo "✨ Formatting code..."
	go fmt ./...

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	go vet ./...

# Protocol buffer targets
proto: ## Generate protobuf code
	@echo "🔧 Generating protobuf code..."
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/analytics.proto

proto-install: ## Install protobuf dependencies
	@echo "📦 Installing protobuf dependencies..."
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Dependency management
deps: ## Download and tidy dependencies
	@echo "📦 Managing dependencies..."
	go mod download
	go mod tidy

deps-update: ## Update all dependencies
	@echo "📦 Updating dependencies..."
	go get -u ./...
	go mod tidy

# Development targets
run: ## Run the service locally
	@echo "🚀 Starting analytics service..."
	go run ./cmd

dev: ## Run the service in development mode with hot reload (requires air)
	@echo "🔥 Starting development server..."
	air

# Database targets
db-migrate: ## Run database migrations
	@echo "🗄️  Running database migrations..."
	@echo "Migration support to be implemented"

db-seed: ## Seed database with test data
	@echo "🌱 Seeding database..."
	@echo "Database seeding to be implemented"

# Docker targets
docker-run: ## Run service in Docker
	@echo "🐳 Running service in Docker..."
	docker run -p 8080:8080 -p 9090:9090 analytics-service:latest

docker-compose-up: ## Start all services with docker-compose
	@echo "🐳 Starting services with docker-compose..."
	cd ../../ && docker-compose up analytics-service

docker-compose-down: ## Stop all services
	@echo "🐳 Stopping services..."
	cd ../../ && docker-compose down

# Cleanup targets
clean: ## Clean build artifacts and test files
	@echo "🧹 Cleaning up..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	go clean -testcache
	go clean -modcache

clean-all: clean ## Clean everything including Docker
	@echo "🧹 Deep cleaning..."
	docker system prune -f

# Monitoring and debugging
logs: ## Show service logs (when running in Docker)
	@echo "📋 Showing logs..."
	docker logs -f analytics-service

health: ## Check service health
	@echo "❤️  Checking service health..."
	curl -f http://localhost:8080/health || echo "Service not responding"

# Performance targets
profile: ## Run CPU profiling
	@echo "⚡ Running CPU profiling..."
	go test -cpuprofile=cpu.prof -bench=. ./...

profile-mem: ## Run memory profiling
	@echo "🧠 Running memory profiling..."
	go test -memprofile=mem.prof -bench=. ./...

# Security targets
security-scan: ## Run security scan
	@echo "🔒 Running security scan..."
	gosec ./...

# Documentation targets
docs: ## Generate documentation
	@echo "📚 Generating documentation..."
	godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

# Setup targets
setup: ## Setup development environment
	@echo "🛠️  Setting up development environment..."
	@make deps
	@make proto-install
	@echo "✅ Development environment setup complete"

setup-hooks: ## Setup git hooks
	@echo "🪝 Setting up git hooks..."
	@echo "#!/bin/sh\nmake lint && make test-unit" > .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "✅ Git hooks setup complete"

# CI/CD targets
ci: ## Run CI pipeline locally
	@echo "🤖 Running CI pipeline..."
	@make lint
	@make test
	@make build

# Performance and load testing
load-test: ## Run load tests (requires wrk or similar)
	@echo "⚡ Running load tests..."
	@echo "Load testing setup to be implemented"

# Environment specific targets
dev-env: ## Setup development environment variables
	@echo "🌍 Setting up development environment..."
	cp .env.example .env.local || echo "Create .env.local file manually"

prod-build: ## Build for production
	@echo "🏭 Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/analytics-service ./cmd

# Database backup and restore (for production)
backup: ## Backup database
	@echo "💾 Creating database backup..."
	@echo "Database backup to be implemented"

restore: ## Restore database from backup
	@echo "🔄 Restoring database..."
	@echo "Database restore to be implemented"

# Kubernetes targets
k8s-deploy: ## Deploy to Kubernetes
	@echo "☸️  Deploying to Kubernetes..."
	kubectl apply -f deployments/k8s/

k8s-logs: ## Get Kubernetes logs
	@echo "📋 Getting Kubernetes logs..."
	kubectl logs -l app=analytics-service

# Version management
version: ## Show version information
	@echo "📊 Analytics Service Version Information"
	@echo "======================================="
	@go version
	@echo "Git commit: $$(git rev-parse --short HEAD)"
	@echo "Build date: $$(date)"

# All-in-one targets
fresh: clean deps build test ## Fresh build (clean, deps, build, test)

all: fmt vet lint test build ## Run all checks and build

.DEFAULT_GOAL := help