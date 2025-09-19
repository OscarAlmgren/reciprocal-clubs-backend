# 🚀 Reciprocal Clubs Backend - Comprehensive Deployment Guide

## 📋 Table of Contents

1. [Overview](#overview)
2. [Prerequisites](#prerequisites)
3. [Quick Start](#quick-start)
4. [Development Deployments](#development-deployments)
5. [Production Deployments](#production-deployments)
6. [Deployment Scenarios](#deployment-scenarios)
7. [Configuration Management](#configuration-management)
8. [Monitoring & Observability](#monitoring--observability)
9. [Security](#security)
10. [Troubleshooting](#troubleshooting)
11. [Maintenance](#maintenance)

## 🎯 Overview

This guide provides comprehensive instructions for deploying the Reciprocal Clubs Backend platform across different environments and orchestration platforms. The platform supports multiple deployment scenarios:

### 🏗️ Deployment Options

| Deployment Method | Environment | Use Case | Complexity |
|------------------|-------------|----------|------------|
| **Docker Compose** | Development/Testing | Local development, CI/CD testing | ⭐ Easy |
| **Docker Swarm** | Production | Single-node or small cluster production | ⭐⭐ Medium |
| **Podman Quadlets** | Development/Production | Systemd-managed containers | ⭐⭐ Medium |
| **Kubernetes** | Production | Enterprise-scale production | ⭐⭐⭐ Advanced |

### 🏛️ Architecture Overview

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Load Balancer │────│   API Gateway    │────│  Auth Service   │
│ (NGINX/Istio)   │    │   (GraphQL)      │    │  (Hanko + MFA)  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
┌───────▼──────┐    ┌───────────▼────┐    ┌─────────────▼───┐
│ Member       │    │ Reciprocal     │    │ Analytics       │
│ Service      │    │ Service        │    │ Service         │
└──────────────┘    └────────────────┘    └─────────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                │
        ┌───────────────────────┼───────────────────────┐
        │                       │                       │
┌───────▼──────┐    ┌───────────▼────┐    ┌─────────────▼───┐
│ Blockchain   │    │ Notification   │    │ Governance      │
│ Service      │    │ Service        │    │ Service         │
└──────────────┘    └────────────────┘    └─────────────────┘
        │                       │                       │
        └───────────────────────┼───────────────────────┘
                                │
    ┌───────────────────────────▼───────────────────────────┐
    │                Infrastructure                         │
    │ PostgreSQL │ Redis │ NATS │ Hyperledger Fabric       │
    └───────────────────────────────────────────────────────┘
```

## 🛠️ Prerequisites

### Base Requirements

| Component | Version | Notes |
|-----------|---------|-------|
| **Docker** | 24.0+ | Required for all deployment methods |
| **Docker Compose** | 2.20+ | For local development |
| **Go** | 1.25+ | For building from source |
| **Git** | 2.30+ | For source code management |

### Platform-Specific Requirements

#### For Kubernetes Deployments
```bash
# Required tools
kubectl >= 1.28
helm >= 3.12
istio >= 1.19 (optional, for service mesh)

# Cluster requirements
- Kubernetes 1.28+
- StorageClass for persistent volumes
- Ingress controller (NGINX recommended)
- cert-manager for TLS certificates
```

#### For Podman Deployments
```bash
# Install Podman (macOS)
brew install podman

# Install Podman (Linux)
sudo apt-get install podman

# Initialize Podman machine (macOS)
podman machine init
podman machine start
```

#### For Production Deployments
```bash
# External dependencies
PostgreSQL 15+ (if not using containerized)
Redis 7+ (if not using containerized)
NATS 2.10+ (if not using containerized)

# SSL certificates
Let's Encrypt or custom CA certificates
DNS management for domain resolution
```

## 🚀 Quick Start

### 1. Clone and Setup

```bash
# Clone the repository
git clone https://github.com/reciprocal-clubs/backend.git
cd reciprocal-clubs-backend

# Copy environment configuration
cp .env.example .env.dev
cp .env.example .env.prod

# Edit configuration files
nano .env.dev    # For development
nano .env.prod   # For production
```

### 2. Choose Your Deployment Method

#### Quick Development Setup (Docker Compose)
```bash
# Start development environment
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose -f docker-compose.dev.yml logs -f

# Access services
echo "API Gateway: http://localhost:8080"
echo "Auth Service: http://localhost:8081"
echo "Grafana: http://localhost:3000 (admin/devpassword)"
echo "MailHog: http://localhost:8025"
```

#### Quick Production Setup (Docker Swarm)
```bash
# Initialize Docker Swarm
docker swarm init

# Create production secrets
echo "production-postgres-password" | docker secret create postgres_password -
echo "production-jwt-secret-key" | docker secret create jwt_secret -

# Deploy production stack
docker stack deploy -c docker-stack.prod.yml reciprocal-clubs

# Check deployment status
docker service ls
```

## 🧪 Development Deployments

### Docker Compose Development

#### Start Development Environment
```bash
# Full stack with all services
docker-compose -f docker-compose.dev.yml up -d

# Start specific services only
docker-compose -f docker-compose.dev.yml up -d postgres redis nats
docker-compose -f docker-compose.dev.yml up -d auth-service member-service

# Build and start (if you've made code changes)
docker-compose -f docker-compose.dev.yml up -d --build
```

#### Development Environment Features
- **Auto-reloading**: Services automatically reload on code changes
- **Debug logging**: All services run in debug mode
- **Development tools**: MailHog for email testing, pgAdmin for database management
- **Relaxed security**: Simplified passwords and configurations
- **Volume mounts**: Code and configuration mounted as volumes

#### Development URLs
```bash
# Core Services
API Gateway:     http://localhost:8080
Auth Service:    http://localhost:8081
Member Service:  http://localhost:8082
Analytics:       http://localhost:8086

# Development Tools
MailHog:         http://localhost:8025
pgAdmin:         http://localhost:5050
Grafana:         http://localhost:3000 (admin/devpassword)
Prometheus:      http://localhost:9090
Jaeger:          http://localhost:16686
```

### Podman Quadlets Development

#### Install and Configure
```bash
# Copy Quadlet files to systemd directory
sudo cp deployments/podman-quadlets/dev/*.container /etc/containers/systemd/
sudo cp deployments/podman-quadlets/dev/*.network /etc/containers/systemd/

# Reload systemd daemon
sudo systemctl daemon-reload

# Start development environment target
sudo systemctl start reciprocal-clubs-dev.target

# Check service status
sudo systemctl status reciprocal-clubs-dev.target
systemctl --user list-dependencies reciprocal-clubs-dev.target
```

#### Managing Podman Services
```bash
# Start/stop individual services
sudo systemctl start auth-service-dev
sudo systemctl stop auth-service-dev

# View logs
journalctl -u auth-service-dev -f

# Enable auto-start
sudo systemctl enable reciprocal-clubs-dev.target
```

### Kubernetes Development

#### Prerequisites
```bash
# Local Kubernetes cluster (choose one)
# Option 1: minikube
minikube start --cpus=4 --memory=8192 --kubernetes-version=v1.28.0

# Option 2: Docker Desktop Kubernetes
# Enable Kubernetes in Docker Desktop settings

# Option 3: kind
kind create cluster --config=deployments/k8s/dev/kind-config.yaml
```

#### Deploy Development Environment
```bash
# Create namespace and basic resources
kubectl apply -f deployments/k8s/dev/namespace.yaml
kubectl apply -f deployments/k8s/dev/configmaps.yaml
kubectl apply -f deployments/k8s/dev/secrets.yaml

# Deploy infrastructure services
kubectl apply -f deployments/k8s/infrastructure/

# Deploy application services
kubectl apply -f deployments/k8s/services/

# Check deployment status
kubectl get pods -n reciprocal-clubs-dev
kubectl get services -n reciprocal-clubs-dev
```

#### Access Development Services
```bash
# Port forward API Gateway
kubectl port-forward -n reciprocal-clubs-dev service/api-gateway-dev-service 8080:8080

# Port forward individual services
kubectl port-forward -n reciprocal-clubs-dev service/auth-service-dev-service 8081:8081
kubectl port-forward -n reciprocal-clubs-dev service/grafana-dev-service 3000:3000

# Get all service URLs
kubectl get ingress -n reciprocal-clubs-dev
```

## 🏭 Production Deployments

### Production Requirements Checklist

#### Infrastructure
- [ ] **Compute**: Minimum 8 CPU cores, 16GB RAM per node
- [ ] **Storage**: SSD storage with IOPS 3000+ for databases
- [ ] **Network**: Dedicated network with low latency
- [ ] **Load Balancer**: External load balancer or cloud LB
- [ ] **DNS**: Configured DNS records for production domains
- [ ] **SSL/TLS**: Valid SSL certificates for all domains

#### Security
- [ ] **Secrets Management**: Vault, AWS Secrets Manager, or similar
- [ ] **Network Policies**: Firewall rules and network segmentation
- [ ] **RBAC**: Role-based access control configured
- [ ] **Image Security**: Container images scanned for vulnerabilities
- [ ] **Backup Strategy**: Automated backups for all persistent data

#### Monitoring
- [ ] **Metrics**: Prometheus with persistent storage
- [ ] **Logging**: Centralized logging (ELK, Grafana Loki)
- [ ] **Alerting**: Alert manager with notification channels
- [ ] **Tracing**: Distributed tracing with Jaeger or Zipkin
- [ ] **Uptime Monitoring**: External uptime monitoring service

### Docker Swarm Production

#### Swarm Cluster Setup
```bash
# Initialize Swarm on manager node
docker swarm init --advertise-addr <MANAGER-IP>

# Join worker nodes (run on each worker)
docker swarm join --token <WORKER-TOKEN> <MANAGER-IP>:2377

# Verify cluster
docker node ls
```

#### Production Secrets Management
```bash
# Create production secrets
echo "$(openssl rand -base64 32)" | docker secret create postgres_password -
echo "$(openssl rand -base64 64)" | docker secret create jwt_secret -
echo "$(openssl rand -base64 32)" | docker secret create redis_password -
echo "your-hanko-api-key" | docker secret create hanko_api_key -
echo "your-smtp-password" | docker secret create smtp_password -

# Create TLS certificates
docker secret create tls_cert /path/to/cert.pem
docker secret create tls_key /path/to/key.pem

# List secrets
docker secret ls
```

#### Deploy Production Stack
```bash
# Deploy the full production stack
docker stack deploy -c docker-stack.prod.yml reciprocal-clubs

# Verify deployment
docker service ls
docker stack ps reciprocal-clubs

# Check service logs
docker service logs reciprocal-clubs_api-gateway
docker service logs reciprocal-clubs_auth-service -f
```

#### Production Stack Management
```bash
# Update a service (rolling update)
docker service update --image reciprocal-clubs/auth-service:v1.2.0 reciprocal-clubs_auth-service

# Scale services
docker service scale reciprocal-clubs_api-gateway=5
docker service scale reciprocal-clubs_auth-service=3

# Remove stack
docker stack rm reciprocal-clubs

# View stack status
watch 'docker stack ps reciprocal-clubs'
```

### Kubernetes Production

#### Production Cluster Setup

##### Cloud Provider Setup (AWS EKS Example)
```bash
# Install eksctl
curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
sudo mv /tmp/eksctl /usr/local/bin

# Create EKS cluster
eksctl create cluster \
  --name reciprocal-clubs-prod \
  --region us-west-2 \
  --nodegroup-name workers \
  --node-type t3.large \
  --nodes 3 \
  --nodes-min 2 \
  --nodes-max 10 \
  --managed

# Configure kubectl
aws eks update-kubeconfig --region us-west-2 --name reciprocal-clubs-prod
```

##### On-Premise Kubernetes Setup
```bash
# Install kubeadm, kubelet, kubectl on all nodes
curl -s https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
echo "deb https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl

# Initialize cluster (master node)
sudo kubeadm init --pod-network-cidr=10.244.0.0/16

# Join worker nodes
sudo kubeadm join <MASTER-IP>:6443 --token <TOKEN> --discovery-token-ca-cert-hash <HASH>

# Install network plugin (Calico)
kubectl apply -f https://docs.projectcalico.org/manifests/calico.yaml
```

#### Install Required Components

##### NGINX Ingress Controller
```bash
# Install NGINX Ingress
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml

# Verify installation
kubectl get pods -n ingress-nginx
kubectl get service -n ingress-nginx
```

##### cert-manager for TLS
```bash
# Install cert-manager
kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.13.1/cert-manager.yaml

# Create Let's Encrypt issuer
kubectl apply -f deployments/k8s/prod/cert-manager-issuer.yaml
```

##### External Secrets Operator
```bash
# Install External Secrets Operator
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets external-secrets/external-secrets -n external-secrets --create-namespace

# Configure secret store (AWS Secrets Manager example)
kubectl apply -f deployments/k8s/prod/external-secrets-store.yaml
```

#### Deploy Production Application

##### 1. Create Namespace and Basic Resources
```bash
# Apply namespace and security policies
kubectl apply -f deployments/k8s/prod/namespace.yaml

# Apply configuration
kubectl apply -f deployments/k8s/prod/configmaps.yaml

# Apply secrets (external secrets will sync automatically)
kubectl apply -f deployments/k8s/prod/secrets.yaml
```

##### 2. Deploy Infrastructure Services
```bash
# Deploy PostgreSQL, Redis, NATS, Hanko
kubectl apply -f deployments/k8s/prod/infrastructure-prod.yaml

# Wait for infrastructure to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n reciprocal-clubs-prod --timeout=300s
kubectl wait --for=condition=ready pod -l app=redis -n reciprocal-clubs-prod --timeout=300s
kubectl wait --for=condition=ready pod -l app=nats -n reciprocal-clubs-prod --timeout=300s
```

##### 3. Deploy Application Services
```bash
# Deploy all microservices
kubectl apply -f deployments/k8s/prod/application-services-prod.yaml

# Wait for services to be ready
kubectl wait --for=condition=ready pod -l app=auth-service -n reciprocal-clubs-prod --timeout=300s
kubectl wait --for=condition=ready pod -l app=api-gateway -n reciprocal-clubs-prod --timeout=300s
```

##### 4. Configure Ingress and TLS
```bash
# Deploy ingress configuration
kubectl apply -f deployments/k8s/prod/ingress.yaml

# Verify TLS certificates
kubectl get certificates -n reciprocal-clubs-prod
kubectl describe certificate api-tls -n reciprocal-clubs-prod
```

##### 5. Optional: Deploy Service Mesh (Istio)
```bash
# Install Istio
istioctl install --set values.defaultRevision=default

# Label namespace for Istio injection
kubectl label namespace reciprocal-clubs-prod istio-injection=enabled

# Apply Istio configuration
kubectl apply -f deployments/k8s/prod/istio-gateway.yaml

# Restart pods to inject sidecar
kubectl rollout restart deployment -n reciprocal-clubs-prod
```

#### Production Verification
```bash
# Check all pods are running
kubectl get pods -n reciprocal-clubs-prod

# Check services
kubectl get services -n reciprocal-clubs-prod

# Check ingress
kubectl get ingress -n reciprocal-clubs-prod

# Test external access
curl https://api.reciprocal-clubs.com/health
curl https://auth.reciprocal-clubs.com/health

# Check resource usage
kubectl top pods -n reciprocal-clubs-prod
kubectl top nodes
```

### Podman Quadlets Production

#### Production Setup
```bash
# Copy production Quadlet files
sudo cp deployments/podman-quadlets/prod/*.container /etc/containers/systemd/
sudo cp deployments/podman-quadlets/prod/*.network /etc/containers/systemd/
sudo cp deployments/podman-quadlets/prod/*.target /etc/containers/systemd/

# Create production secrets
sudo mkdir -p /etc/reciprocal-clubs/secrets
sudo chmod 700 /etc/reciprocal-clubs/secrets

echo "production-postgres-password" | sudo tee /etc/reciprocal-clubs/secrets/postgres_password
echo "production-jwt-secret" | sudo tee /etc/reciprocal-clubs/secrets/jwt_secret
sudo chmod 600 /etc/reciprocal-clubs/secrets/*
```

#### Deploy Production Services
```bash
# Reload systemd daemon
sudo systemctl daemon-reload

# Start production target
sudo systemctl start reciprocal-clubs-prod.target

# Enable auto-start
sudo systemctl enable reciprocal-clubs-prod.target

# Check service status
sudo systemctl status reciprocal-clubs-prod.target
systemctl list-dependencies reciprocal-clubs-prod.target
```

## 🔧 Configuration Management

### Environment Variables

#### Development Environment Variables
```bash
# Core Application Settings
APP_ENV=development
DEBUG=true
LOG_LEVEL=debug

# Database Configuration
DATABASE_HOST=postgres-dev-service
DATABASE_PORT=5432
DATABASE_USERNAME=postgres
DATABASE_PASSWORD=devpassword
DATABASE_SSL_MODE=disable

# Redis Configuration
REDIS_HOST=redis-dev-service
REDIS_PORT=6379
REDIS_PASSWORD=devredispass

# Authentication
JWT_SECRET=development-jwt-secret-key
HANKO_BASE_URL=http://hanko-dev-service:8000
MFA_ISSUER=Reciprocal Clubs Dev

# CORS (Development)
CORS_ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080
```

#### Production Environment Variables
```bash
# Core Application Settings
APP_ENV=production
DEBUG=false
LOG_LEVEL=info

# Database Configuration (with SSL)
DATABASE_HOST=postgres-prod-service
DATABASE_PORT=5432
DATABASE_USERNAME=postgres
DATABASE_PASSWORD_FILE=/run/secrets/postgres_password
DATABASE_SSL_MODE=require
DATABASE_MAX_CONNECTIONS=100
DATABASE_CONNECTION_TIMEOUT=30s

# Redis Configuration (with AUTH)
REDIS_HOST=redis-prod-service
REDIS_PORT=6379
REDIS_PASSWORD_FILE=/run/secrets/redis_password
REDIS_MAX_CONNECTIONS=1000

# Authentication (with secrets)
JWT_SECRET_FILE=/run/secrets/jwt_secret
HANKO_BASE_URL=http://hanko-prod-service:8000
HANKO_API_KEY_FILE=/run/secrets/hanko_api_key
MFA_ISSUER=Reciprocal Clubs

# CORS (Production)
CORS_ALLOWED_ORIGINS=https://app.reciprocal-clubs.com,https://admin.reciprocal-clubs.com
```

### Secrets Management

#### Docker Swarm Secrets
```bash
# Create secrets
echo "secure-postgres-password" | docker secret create postgres_password -
echo "secure-jwt-secret-64-chars" | docker secret create jwt_secret -

# List secrets
docker secret ls

# Inspect secret (metadata only)
docker secret inspect postgres_password
```

#### Kubernetes Secrets
```bash
# Create secrets manually
kubectl create secret generic reciprocal-clubs-prod-secrets \
  --from-literal=postgres-password="secure-postgres-password" \
  --from-literal=jwt-secret="secure-jwt-secret" \
  -n reciprocal-clubs-prod

# Create TLS secret
kubectl create secret tls api-tls \
  --cert=api.crt \
  --key=api.key \
  -n reciprocal-clubs-prod

# Using External Secrets Operator (recommended)
kubectl apply -f deployments/k8s/prod/external-secrets.yaml
```

#### External Secrets Integration
```yaml
# Example: AWS Secrets Manager
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-manager
  namespace: reciprocal-clubs-prod
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-west-2
      auth:
        jwt:
          serviceAccountRef:
            name: external-secrets-sa
---
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: reciprocal-clubs-secrets
  namespace: reciprocal-clubs-prod
spec:
  refreshInterval: 300s
  secretStoreRef:
    name: aws-secrets-manager
    kind: SecretStore
  target:
    name: reciprocal-clubs-prod-secrets
    creationPolicy: Owner
  data:
  - secretKey: postgres-password
    remoteRef:
      key: reciprocal-clubs/prod/postgres
      property: password
  - secretKey: jwt-secret
    remoteRef:
      key: reciprocal-clubs/prod/jwt
      property: secret
```

### ConfigMaps and Configuration Files

#### Application Configuration Structure
```
config/
├── dev/
│   ├── app.yaml
│   ├── database.yaml
│   ├── redis.yaml
│   └── logging.yaml
├── prod/
│   ├── app.yaml
│   ├── database.yaml
│   ├── redis.yaml
│   └── logging.yaml
└── common/
    ├── features.yaml
    └── security.yaml
```

#### Example Production Configuration
```yaml
# config/prod/app.yaml
app:
  environment: production
  debug: false
  log_level: info
  graceful_shutdown_timeout: 30s

server:
  http:
    port: 8080
    timeout:
      read: 30s
      write: 30s
      idle: 120s
  grpc:
    port: 9080
    timeout: 30s

security:
  cors:
    allowed_origins:
      - "https://app.reciprocal-clubs.com"
      - "https://admin.reciprocal-clubs.com"
    allowed_methods: ["GET", "POST", "PUT", "DELETE", "OPTIONS"]
    allowed_headers: ["Content-Type", "Authorization", "X-Request-ID"]
    max_age: 86400

monitoring:
  metrics:
    enabled: true
    path: "/metrics"
    port: 9090
  health:
    enabled: true
    path: "/health"
  jaeger:
    enabled: true
    endpoint: "http://jaeger-collector:14268/api/traces"
```

## 📊 Monitoring & Observability

### Metrics Collection (Prometheus)

#### Prometheus Configuration
```yaml
# monitoring/prometheus.prod.yml
global:
  scrape_interval: 30s
  evaluation_interval: 30s
  external_labels:
    cluster: 'reciprocal-clubs-prod'
    region: 'us-west-2'

rule_files:
  - "/etc/prometheus/rules/*.yml"

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093

scrape_configs:
  # Application services
  - job_name: 'reciprocal-clubs-services'
    kubernetes_sd_configs:
      - role: pod
        namespaces:
          names:
            - reciprocal-clubs-prod
    relabel_configs:
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_scrape]
        action: keep
        regex: true
      - source_labels: [__meta_kubernetes_pod_annotation_prometheus_io_path]
        action: replace
        target_label: __metrics_path__
        regex: (.+)

  # Infrastructure services
  - job_name: 'postgres-exporter'
    static_configs:
      - targets: ['postgres-exporter:9187']

  - job_name: 'redis-exporter'
    static_configs:
      - targets: ['redis-exporter:9121']

  - job_name: 'nats-exporter'
    static_configs:
      - targets: ['nats-exporter:7777']
```

#### Key Metrics to Monitor

##### Application Metrics
```
# Request metrics
http_requests_total{service="api-gateway", method="GET", status="200"}
http_request_duration_seconds{service="auth-service", quantile="0.95"}
grpc_server_handled_total{service="member-service", grpc_code="OK"}

# Business metrics
reciprocal_clubs_total_users{club="club-123"}
reciprocal_clubs_active_agreements{status="active"}
reciprocal_clubs_auth_attempts_total{method="mfa", result="success"}

# Error metrics
reciprocal_clubs_errors_total{service="notification-service", error_type="smtp_failure"}
reciprocal_clubs_database_errors_total{service="analytics-service", operation="query"}
```

##### Infrastructure Metrics
```
# PostgreSQL
postgres_up
postgres_stat_database_tup_inserted{datname="reciprocal_clubs_prod"}
postgres_stat_database_conflicts{datname="reciprocal_clubs_prod"}

# Redis
redis_up
redis_memory_used_bytes
redis_connected_clients

# NATS
nats_core_total_connections
nats_core_mem_bytes
nats_jetstream_streams
```

### Logging (Structured)

#### Log Format Configuration
```yaml
# logging.yaml
logging:
  level: info
  format: json
  timestamp_format: "2006-01-02T15:04:05.000Z07:00"

  fields:
    service: "${SERVICE_NAME}"
    version: "${SERVICE_VERSION}"
    environment: "${APP_ENV}"

  outputs:
    - type: stdout
      level: info
    - type: file
      level: debug
      path: "/app/logs/app.log"
      max_size: 100MB
      max_files: 10
      max_age: 30
```

#### Example Log Entries
```json
{
  "timestamp": "2024-09-19T10:30:45.123Z",
  "level": "info",
  "service": "auth-service",
  "version": "1.0.0",
  "environment": "production",
  "correlation_id": "req_123456789",
  "user_id": "user_987654321",
  "message": "User authenticated successfully",
  "auth_method": "mfa_totp",
  "duration_ms": 45,
  "ip_address": "192.168.1.100"
}

{
  "timestamp": "2024-09-19T10:30:46.789Z",
  "level": "error",
  "service": "notification-service",
  "version": "1.0.0",
  "environment": "production",
  "correlation_id": "req_123456790",
  "message": "Failed to send email notification",
  "error": "SMTP connection timeout",
  "email_template": "welcome_email",
  "recipient": "user@example.com",
  "retry_count": 3
}
```

### Distributed Tracing (Jaeger)

#### Jaeger Configuration
```yaml
# jaeger-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: jaeger-config
  namespace: reciprocal-clubs-prod
data:
  jaeger.yaml: |
    reporting:
      localAgentHostPort: "jaeger-agent:6831"
      logSpans: false
    sampler:
      type: probabilistic
      param: 0.1
    rpc_metrics: true
    headers:
      jaegerDebugHeader: "jaeger-debug-id"
      jaegerBaggageHeader: "jaeger-baggage"
      traceContextHeaderName: "uber-trace-id"
```

#### Tracing Best Practices
```go
// Example: Tracing in Go services
func (s *AuthService) AuthenticateUser(ctx context.Context, req *AuthRequest) (*AuthResponse, error) {
    span, ctx := opentracing.StartSpanFromContext(ctx, "AuthService.AuthenticateUser")
    defer span.Finish()

    span.SetTag("user.email", req.Email)
    span.SetTag("auth.method", req.Method)

    // Add correlation ID
    correlationID := generateCorrelationID()
    span.SetBaggageItem("correlation.id", correlationID)

    // Database operation
    user, err := s.repo.GetUserByEmail(ctx, req.Email)
    if err != nil {
        span.SetTag("error", true)
        span.LogFields(
            log.String("error.kind", "database_error"),
            log.String("error.message", err.Error()),
        )
        return nil, err
    }

    span.SetTag("user.id", user.ID)
    return &AuthResponse{Token: "..."}, nil
}
```

### Alerting Rules

#### Prometheus Alerting Rules
```yaml
# monitoring/alerts/application.yml
groups:
  - name: reciprocal-clubs-application
    rules:
      # High error rate
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) / rate(http_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "High error rate detected"
          description: "Error rate is {{ $value | humanizePercentage }} for {{ $labels.service }}"

      # Database connection issues
      - alert: DatabaseConnectionFailure
        expr: postgres_up == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Database is down"
          description: "PostgreSQL database is not responding"

      # Memory usage
      - alert: HighMemoryUsage
        expr: (container_memory_usage_bytes / container_spec_memory_limit_bytes) > 0.9
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "High memory usage"
          description: "Container {{ $labels.container }} memory usage is {{ $value | humanizePercentage }}"

      # Auth service specific
      - alert: AuthServiceDown
        expr: up{job="auth-service"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Auth service is down"
          description: "Auth service is not responding - authentication will fail"
```

### Dashboard Configuration (Grafana)

#### Application Dashboard
```json
{
  "dashboard": {
    "title": "Reciprocal Clubs - Application Overview",
    "panels": [
      {
        "title": "Request Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(rate(http_requests_total[5m])) by (service)",
            "legendFormat": "{{ service }}"
          }
        ]
      },
      {
        "title": "Error Rate",
        "type": "graph",
        "targets": [
          {
            "expr": "sum(rate(http_requests_total{status=~\"5..\"}[5m])) by (service)",
            "legendFormat": "{{ service }} errors"
          }
        ]
      },
      {
        "title": "Response Time P95",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, sum(rate(http_request_duration_seconds_bucket[5m])) by (service, le))",
            "legendFormat": "{{ service }} p95"
          }
        ]
      }
    ]
  }
}
```

## 🔒 Security

### Container Security

#### Dockerfile Security Best Practices
```dockerfile
# Use specific version tags
FROM golang:1.25-alpine AS builder

# Run as non-root user
RUN addgroup -S appgroup -g 1001 && \
    adduser -S appuser -u 1001 -G appgroup

# Copy files with proper ownership
COPY --from=builder --chown=appuser:appgroup /app/service ./

# Use non-root user
USER appuser

# Read-only root filesystem
# Add in deployment: readOnlyRootFilesystem: true

# Security labels
LABEL security.scanning.enabled="true"
LABEL security.vulnerability.policy="high"
```

#### Kubernetes Security Context
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: auth-service-prod
spec:
  template:
    spec:
      securityContext:
        runAsNonRoot: true
        runAsUser: 1001
        runAsGroup: 1001
        fsGroup: 1001
        seccompProfile:
          type: RuntimeDefault
      containers:
      - name: auth-service
        securityContext:
          allowPrivilegeEscalation: false
          readOnlyRootFilesystem: true
          capabilities:
            drop:
            - ALL
        volumeMounts:
        - name: tmp
          mountPath: /tmp
          readOnly: false
        - name: app-logs
          mountPath: /app/logs
          readOnly: false
      volumes:
      - name: tmp
        emptyDir: {}
      - name: app-logs
        emptyDir: {}
```

### Network Security

#### Kubernetes Network Policies
```yaml
# Restrict ingress to only necessary services
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: auth-service-netpol
  namespace: reciprocal-clubs-prod
spec:
  podSelector:
    matchLabels:
      app: auth-service
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: api-gateway
    ports:
    - protocol: TCP
      port: 8081
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

#### Istio Security Policies
```yaml
# mTLS enforcement
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: reciprocal-clubs-prod
spec:
  mtls:
    mode: STRICT

---
# Authorization policy
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: auth-service-policy
  namespace: reciprocal-clubs-prod
spec:
  selector:
    matchLabels:
      app: auth-service
  rules:
  - from:
    - source:
        principals: ["cluster.local/ns/reciprocal-clubs-prod/sa/api-gateway"]
  - to:
    - operation:
        methods: ["GET", "POST"]
        paths: ["/auth/*", "/health"]
```

### TLS Configuration

#### cert-manager Certificate
```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: api-tls
  namespace: reciprocal-clubs-prod
spec:
  secretName: api-tls-secret
  issuerRef:
    name: letsencrypt-prod
    kind: ClusterIssuer
  dnsNames:
  - api.reciprocal-clubs.com
  - auth.reciprocal-clubs.com
  - ws.reciprocal-clubs.com
```

#### NGINX TLS Configuration
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: nginx-tls-config
data:
  ssl.conf: |
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers 'ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256';
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 10m;
    ssl_stapling on;
    ssl_stapling_verify on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
```

### Secrets Management Best Practices

#### Vault Integration
```yaml
# Vault configuration for external secrets
apiVersion: v1
kind: Secret
metadata:
  name: vault-token
type: Opaque
data:
  token: <base64-encoded-vault-token>

---
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: vault-backend
spec:
  provider:
    vault:
      server: "https://vault.company.com"
      path: "secret"
      version: "v2"
      auth:
        tokenSecretRef:
          name: "vault-token"
          key: "token"
```

#### Secret Rotation
```bash
#!/bin/bash
# Secret rotation script

# Generate new secrets
NEW_POSTGRES_PASSWORD=$(openssl rand -base64 32)
NEW_JWT_SECRET=$(openssl rand -base64 64)

# Update in Vault
vault kv put secret/reciprocal-clubs/prod \
  postgres_password="$NEW_POSTGRES_PASSWORD" \
  jwt_secret="$NEW_JWT_SECRET"

# Trigger secret refresh in Kubernetes
kubectl annotate externalsecret reciprocal-clubs-secrets \
  force-sync=$(date +%s) -n reciprocal-clubs-prod

# Rolling restart to pick up new secrets
kubectl rollout restart deployment auth-service -n reciprocal-clubs-prod
kubectl rollout restart deployment api-gateway -n reciprocal-clubs-prod

echo "Secret rotation completed"
```

## 🛠️ Troubleshooting

### Common Issues and Solutions

#### Container Issues

##### Container Won't Start
```bash
# Check container logs
docker logs <container-name> --tail 50 -f

# Kubernetes pod logs
kubectl logs <pod-name> -n reciprocal-clubs-prod -f

# Debug with shell access
docker run -it --entrypoint=/bin/sh reciprocal-clubs/auth-service:latest
kubectl exec -it <pod-name> -n reciprocal-clubs-prod -- /bin/sh
```

**Common causes:**
- Missing environment variables
- Database connection failure
- Port conflicts
- Resource constraints
- Health check failures

##### Memory Issues
```bash
# Check memory usage
docker stats <container-name>
kubectl top pods -n reciprocal-clubs-prod

# Kubernetes resource limits
kubectl describe pod <pod-name> -n reciprocal-clubs-prod | grep -A 10 "Limits\|Requests"

# Out of Memory errors
kubectl get events -n reciprocal-clubs-prod | grep OOMKilled
```

**Solutions:**
```yaml
# Increase memory limits
resources:
  limits:
    memory: "2Gi"
  requests:
    memory: "1Gi"

# Add memory monitoring
livenessProbe:
  exec:
    command:
    - /bin/sh
    - -c
    - "ps aux | awk 'NR>1{memory+=$6} END {if(memory>1000000) exit 1}'"
```

#### Database Connection Issues

##### PostgreSQL Connection Failures
```bash
# Test database connectivity
docker exec -it postgres-container psql -U postgres -d reciprocal_clubs_prod -c "SELECT 1;"

# Kubernetes database connection test
kubectl run --rm -it --restart=Never postgres-client \
  --image=postgres:15-alpine \
  --env="PGPASSWORD=password" \
  -- psql -h postgres-service -U postgres -d reciprocal_clubs_prod

# Check database logs
kubectl logs postgres-pod -n reciprocal-clubs-prod
```

**Common solutions:**
```yaml
# Increase connection limits in PostgreSQL
data:
  postgresql.conf: |
    max_connections = 200
    shared_buffers = 256MB
    work_mem = 4MB

# Update application connection pool
environment:
  - name: DATABASE_MAX_CONNECTIONS
    value: "50"
  - name: DATABASE_CONNECTION_TIMEOUT
    value: "30s"
```

##### Redis Connection Issues
```bash
# Test Redis connectivity
docker exec -it redis-container redis-cli ping

# Kubernetes Redis test
kubectl run --rm -it --restart=Never redis-client \
  --image=redis:7-alpine \
  -- redis-cli -h redis-service ping

# Check Redis memory usage
kubectl exec redis-pod -n reciprocal-clubs-prod -- redis-cli info memory
```

#### Service Discovery Issues

##### Kubernetes DNS Resolution
```bash
# Test service DNS resolution
kubectl run --rm -it --restart=Never dns-test \
  --image=busybox \
  -- nslookup auth-service.reciprocal-clubs-prod.svc.cluster.local

# Check kube-dns logs
kubectl logs -n kube-system -l k8s-app=kube-dns

# Test connectivity between services
kubectl exec api-gateway-pod -n reciprocal-clubs-prod \
  -- curl http://auth-service:8081/health
```

##### Docker Swarm Service Discovery
```bash
# Check Docker Swarm networks
docker network ls
docker network inspect reciprocal-clubs-overlay

# Test service connectivity
docker exec reciprocal-clubs_api-gateway.1.xxx \
  curl http://auth-service:8081/health

# Check service endpoints
docker service inspect reciprocal-clubs_auth-service
```

#### Ingress and Load Balancer Issues

##### NGINX Ingress Troubleshooting
```bash
# Check ingress controller logs
kubectl logs -n ingress-nginx deployment/ingress-nginx-controller -f

# Check ingress resources
kubectl get ingress -n reciprocal-clubs-prod
kubectl describe ingress api-ingress -n reciprocal-clubs-prod

# Test backend connectivity
kubectl exec -n ingress-nginx deployment/ingress-nginx-controller \
  -- curl -H "Host: api.reciprocal-clubs.com" http://api-gateway-service:8080/health
```

##### TLS Certificate Issues
```bash
# Check certificate status
kubectl get certificates -n reciprocal-clubs-prod
kubectl describe certificate api-tls -n reciprocal-clubs-prod

# Check cert-manager logs
kubectl logs -n cert-manager deployment/cert-manager -f

# Manual certificate verification
openssl s_client -connect api.reciprocal-clubs.com:443 -servername api.reciprocal-clubs.com
```

#### Performance Issues

##### Slow Response Times
```bash
# Check application metrics
curl http://localhost:9090/metrics | grep http_request_duration

# Kubernetes resource utilization
kubectl top pods -n reciprocal-clubs-prod
kubectl top nodes

# Database performance
kubectl exec postgres-pod -n reciprocal-clubs-prod \
  -- psql -U postgres -d reciprocal_clubs_prod \
  -c "SELECT query, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
```

**Performance optimization:**
```yaml
# Increase replicas
spec:
  replicas: 5

# Add resource requests/limits
resources:
  requests:
    cpu: 500m
    memory: 1Gi
  limits:
    cpu: 2000m
    memory: 2Gi

# Configure HPA
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: api-gateway-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: api-gateway
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
```

### Debugging Tools and Commands

#### Kubernetes Debugging
```bash
# Pod debugging
kubectl describe pod <pod-name> -n reciprocal-clubs-prod
kubectl get events -n reciprocal-clubs-prod --sort-by='.lastTimestamp'
kubectl logs <pod-name> -n reciprocal-clubs-prod --previous

# Service debugging
kubectl get endpoints -n reciprocal-clubs-prod
kubectl describe service <service-name> -n reciprocal-clubs-prod

# Network debugging
kubectl run --rm -it --restart=Never network-debug \
  --image=nicolaka/netshoot \
  -- /bin/bash

# Resource debugging
kubectl top pods -n reciprocal-clubs-prod
kubectl describe nodes
```

#### Docker Debugging
```bash
# Container debugging
docker inspect <container-name>
docker exec -it <container-name> /bin/sh
docker logs <container-name> --since=10m

# Network debugging
docker network ls
docker network inspect <network-name>

# Volume debugging
docker volume ls
docker volume inspect <volume-name>
```

#### Application-Level Debugging

##### Enable Debug Logging
```yaml
# Kubernetes ConfigMap update
apiVersion: v1
kind: ConfigMap
metadata:
  name: reciprocal-clubs-prod-config
data:
  LOG_LEVEL: "debug"
  DEBUG: "true"

# Rolling restart to apply config
kubectl rollout restart deployment -n reciprocal-clubs-prod
```

##### Health Check Debugging
```bash
# Manual health check
curl -v http://auth-service:8081/health
curl -v http://auth-service:8081/ready

# Health check with detailed response
kubectl exec auth-service-pod -n reciprocal-clubs-prod \
  -- curl -v http://localhost:8081/health?detailed=true
```

## 🔄 Maintenance

### Regular Maintenance Tasks

#### Daily Tasks
```bash
#!/bin/bash
# daily-maintenance.sh

# Check cluster health
kubectl get nodes
kubectl get pods -A | grep -v Running

# Check storage usage
kubectl get pv
df -h /var/lib/docker

# Review recent logs for errors
kubectl logs -n reciprocal-clubs-prod --selector=app=auth-service --since=24h | grep ERROR

# Check certificate expiry
kubectl get certificates -n reciprocal-clubs-prod -o custom-columns=NAME:.metadata.name,READY:.status.conditions[0].status,SECRET:.spec.secretName

# Backup verification
./scripts/verify-backups.sh
```

#### Weekly Tasks
```bash
#!/bin/bash
# weekly-maintenance.sh

# Update container images
docker system prune -f
kubectl rollout restart deployment -n reciprocal-clubs-prod

# Database maintenance
kubectl exec postgres-pod -n reciprocal-clubs-prod \
  -- psql -U postgres -d reciprocal_clubs_prod -c "VACUUM ANALYZE;"

# Certificate renewal check
certbot renew --dry-run

# Security scan
trivy image reciprocal-clubs/auth-service:latest
```

#### Monthly Tasks
```bash
#!/bin/bash
# monthly-maintenance.sh

# Full system backup
./scripts/full-backup.sh

# Security updates
apt update && apt upgrade -y
docker pull --all-tags reciprocal-clubs/auth-service

# Performance review
kubectl top nodes
kubectl top pods -A

# Resource optimization review
kubectl describe resourcequota -n reciprocal-clubs-prod
```

### Backup and Recovery

#### Database Backup
```bash
#!/bin/bash
# backup-postgres.sh

BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/postgres"
POSTGRES_POD=$(kubectl get pods -n reciprocal-clubs-prod -l app=postgres -o jsonpath='{.items[0].metadata.name}')

# Create backup directory
mkdir -p "$BACKUP_DIR"

# Dump database
kubectl exec "$POSTGRES_POD" -n reciprocal-clubs-prod \
  -- pg_dumpall -U postgres > "$BACKUP_DIR/postgres_backup_$BACKUP_DATE.sql"

# Compress backup
gzip "$BACKUP_DIR/postgres_backup_$BACKUP_DATE.sql"

# Upload to cloud storage (example: AWS S3)
aws s3 cp "$BACKUP_DIR/postgres_backup_$BACKUP_DATE.sql.gz" \
  s3://reciprocal-clubs-backups/postgres/

# Cleanup old backups (keep 30 days)
find "$BACKUP_DIR" -name "postgres_backup_*.sql.gz" -mtime +30 -delete

echo "Database backup completed: postgres_backup_$BACKUP_DATE.sql.gz"
```

#### Application State Backup
```bash
#!/bin/bash
# backup-application-state.sh

BACKUP_DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backup/k8s"

mkdir -p "$BACKUP_DIR"

# Backup Kubernetes resources
kubectl get all -n reciprocal-clubs-prod -o yaml > "$BACKUP_DIR/k8s_resources_$BACKUP_DATE.yaml"
kubectl get configmaps -n reciprocal-clubs-prod -o yaml > "$BACKUP_DIR/configmaps_$BACKUP_DATE.yaml"
kubectl get secrets -n reciprocal-clubs-prod -o yaml > "$BACKUP_DIR/secrets_$BACKUP_DATE.yaml"

# Backup persistent volumes
kubectl get pv -o yaml > "$BACKUP_DIR/persistent_volumes_$BACKUP_DATE.yaml"
kubectl get pvc -n reciprocal-clubs-prod -o yaml > "$BACKUP_DIR/persistent_volume_claims_$BACKUP_DATE.yaml"

# Compress and upload
tar -czf "$BACKUP_DIR/k8s_backup_$BACKUP_DATE.tar.gz" -C "$BACKUP_DIR" \
  k8s_resources_$BACKUP_DATE.yaml \
  configmaps_$BACKUP_DATE.yaml \
  secrets_$BACKUP_DATE.yaml \
  persistent_volumes_$BACKUP_DATE.yaml \
  persistent_volume_claims_$BACKUP_DATE.yaml

aws s3 cp "$BACKUP_DIR/k8s_backup_$BACKUP_DATE.tar.gz" \
  s3://reciprocal-clubs-backups/k8s/

echo "Kubernetes state backup completed: k8s_backup_$BACKUP_DATE.tar.gz"
```

#### Disaster Recovery
```bash
#!/bin/bash
# disaster-recovery.sh

BACKUP_DATE=$1
if [ -z "$BACKUP_DATE" ]; then
  echo "Usage: $0 <backup_date>"
  echo "Example: $0 20241001_120000"
  exit 1
fi

echo "Starting disaster recovery for backup date: $BACKUP_DATE"

# Download backups from cloud storage
aws s3 cp "s3://reciprocal-clubs-backups/postgres/postgres_backup_$BACKUP_DATE.sql.gz" /tmp/
aws s3 cp "s3://reciprocal-clubs-backups/k8s/k8s_backup_$BACKUP_DATE.tar.gz" /tmp/

# Extract backups
gunzip "/tmp/postgres_backup_$BACKUP_DATE.sql.gz"
tar -xzf "/tmp/k8s_backup_$BACKUP_DATE.tar.gz" -C /tmp/

# Restore Kubernetes resources
kubectl apply -f "/tmp/k8s_resources_$BACKUP_DATE.yaml"
kubectl apply -f "/tmp/configmaps_$BACKUP_DATE.yaml"
kubectl apply -f "/tmp/persistent_volumes_$BACKUP_DATE.yaml"
kubectl apply -f "/tmp/persistent_volume_claims_$BACKUP_DATE.yaml"

# Wait for PostgreSQL to be ready
kubectl wait --for=condition=ready pod -l app=postgres -n reciprocal-clubs-prod --timeout=300s

# Restore database
POSTGRES_POD=$(kubectl get pods -n reciprocal-clubs-prod -l app=postgres -o jsonpath='{.items[0].metadata.name}')
kubectl exec -i "$POSTGRES_POD" -n reciprocal-clubs-prod \
  -- psql -U postgres < "/tmp/postgres_backup_$BACKUP_DATE.sql"

# Restart all services
kubectl rollout restart deployment -n reciprocal-clubs-prod

echo "Disaster recovery completed"
```

### Scaling and Capacity Planning

#### Horizontal Pod Autoscaler Configuration
```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: auth-service-hpa
  namespace: reciprocal-clubs-prod
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: auth-service
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
  - type: Pods
    pods:
      metric:
        name: http_requests_per_second
      target:
        type: AverageValue
        averageValue: "100"
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
      - type: Percent
        value: 10
        periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 60
      policies:
      - type: Percent
        value: 50
        periodSeconds: 60
      - type: Pods
        value: 5
        periodSeconds: 60
      selectPolicy: Max
```

#### Vertical Pod Autoscaler
```yaml
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: auth-service-vpa
  namespace: reciprocal-clubs-prod
spec:
  targetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: auth-service
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
    - containerName: auth-service
      minAllowed:
        cpu: 100m
        memory: 128Mi
      maxAllowed:
        cpu: 2
        memory: 4Gi
      controlledResources: ["cpu", "memory"]
```

#### Cluster Autoscaler
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: cluster-autoscaler-status
  namespace: kube-system
data:
  nodes.max: "50"
  nodes.min: "3"
  scale-down-delay-after-add: "10m"
  scale-down-unneeded-time: "10m"
  skip-nodes-with-local-storage: "false"
```

### Update and Upgrade Procedures

#### Rolling Updates
```bash
#!/bin/bash
# rolling-update.sh

SERVICE=$1
NEW_VERSION=$2

if [ -z "$SERVICE" ] || [ -z "$NEW_VERSION" ]; then
  echo "Usage: $0 <service> <version>"
  echo "Example: $0 auth-service v1.2.0"
  exit 1
fi

echo "Starting rolling update for $SERVICE to $NEW_VERSION"

# Update deployment image
kubectl set image deployment/$SERVICE -n reciprocal-clubs-prod \
  $SERVICE=reciprocal-clubs/$SERVICE:$NEW_VERSION

# Wait for rollout to complete
kubectl rollout status deployment/$SERVICE -n reciprocal-clubs-prod --timeout=600s

# Verify health
kubectl get pods -n reciprocal-clubs-prod -l app=$SERVICE
kubectl exec deployment/$SERVICE -n reciprocal-clubs-prod -- curl -f http://localhost:8081/health

echo "Rolling update completed for $SERVICE"
```

#### Blue-Green Deployment
```bash
#!/bin/bash
# blue-green-deployment.sh

SERVICE=$1
NEW_VERSION=$2
CURRENT_COLOR=$(kubectl get deployment $SERVICE-blue -n reciprocal-clubs-prod >/dev/null 2>&1 && echo "blue" || echo "green")
NEW_COLOR=$([ "$CURRENT_COLOR" = "blue" ] && echo "green" || echo "blue")

echo "Current active: $SERVICE-$CURRENT_COLOR"
echo "Deploying to: $SERVICE-$NEW_COLOR"

# Deploy new version to inactive color
kubectl set image deployment/$SERVICE-$NEW_COLOR -n reciprocal-clubs-prod \
  $SERVICE=reciprocal-clubs/$SERVICE:$NEW_VERSION

# Wait for new deployment to be ready
kubectl rollout status deployment/$SERVICE-$NEW_COLOR -n reciprocal-clubs-prod

# Run health checks on new deployment
kubectl wait --for=condition=ready pod -l app=$SERVICE,color=$NEW_COLOR -n reciprocal-clubs-prod

# Switch traffic to new version
kubectl patch service $SERVICE -n reciprocal-clubs-prod \
  -p '{"spec":{"selector":{"color":"'$NEW_COLOR'"}}}'

echo "Traffic switched to $SERVICE-$NEW_COLOR"

# Optional: Scale down old version after verification
sleep 300
kubectl scale deployment $SERVICE-$CURRENT_COLOR -n reciprocal-clubs-prod --replicas=0

echo "Blue-green deployment completed"
```

#### Canary Deployment with Istio
```yaml
apiVersion: argoproj.io/v1alpha1
kind: Rollout
metadata:
  name: auth-service-rollout
  namespace: reciprocal-clubs-prod
spec:
  replicas: 10
  strategy:
    canary:
      canaryService: auth-service-canary
      stableService: auth-service-stable
      trafficRouting:
        istio:
          virtualService:
            name: auth-service-vs
            routes:
            - primary
          destinationRule:
            name: auth-service-dr
            canarySubsetName: canary
            stableSubsetName: stable
      steps:
      - setWeight: 10
      - pause: {duration: 2m}
      - setWeight: 20
      - pause: {duration: 5m}
      - setWeight: 50
      - pause: {duration: 10m}
      - setWeight: 100
  selector:
    matchLabels:
      app: auth-service
  template:
    metadata:
      labels:
        app: auth-service
    spec:
      containers:
      - name: auth-service
        image: reciprocal-clubs/auth-service:latest
```

## 📞 Support and Contact

### Getting Help

#### Documentation
- **Official Documentation**: https://docs.reciprocal-clubs.com
- **API Documentation**: https://api.reciprocal-clubs.com/docs
- **Deployment Guide**: This document

#### Community Support
- **GitHub Issues**: https://github.com/reciprocal-clubs/backend/issues
- **Discord Server**: https://discord.gg/reciprocal-clubs
- **Stack Overflow**: Tag questions with `reciprocal-clubs`

#### Enterprise Support
For production deployments and enterprise support:
- **Email**: support@reciprocal-clubs.com
- **Slack**: #support channel in enterprise workspace
- **Phone**: +1-800-RECIPROCAL (24/7 for critical issues)

### Contributing to Documentation

This deployment guide is maintained as part of the project repository. To contribute improvements:

1. Fork the repository
2. Create a feature branch: `git checkout -b improve-deployment-docs`
3. Make your changes to `DEPLOYMENT_GUIDE.md`
4. Test your changes with a real deployment
5. Submit a pull request with detailed description

### License

This deployment guide and all associated configuration files are licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

---

**Last Updated**: September 19, 2024
**Version**: 2.0
**Compatibility**: Kubernetes 1.28+, Docker 24.0+, Go 1.25+