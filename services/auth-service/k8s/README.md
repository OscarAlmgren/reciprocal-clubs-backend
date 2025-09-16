# Auth Service Kubernetes Deployment

This directory contains Kubernetes manifests for deploying the Auth Service with all its dependencies.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                        Ingress                              │
│  ┌─────────────────────────┬─────────────────────────────┐  │
│  │   auth.your-domain.com  │   hanko.your-domain.com     │  │
│  └─────────────────────────┴─────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
          │                           │
          ▼                           ▼
┌─────────────────────┐     ┌─────────────────────┐
│   Auth Service      │     │      Hanko          │
│   (3 replicas)      │◄────┤   (2 replicas)      │
└─────────────────────┘     └─────────────────────┘
          │                           │
          ▼                           ▼
┌─────────────────────┐     ┌─────────────────────┐
│   PostgreSQL        │     │   Hanko PostgreSQL  │
│   (1 replica)       │     │   (1 replica)       │
└─────────────────────┘     └─────────────────────┘
          │
          ▼
┌─────────────────────┐
│   NATS              │
│   (3 replicas)      │
└─────────────────────┘
```

## Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl configured
- NGINX Ingress Controller installed
- cert-manager for TLS certificates (optional)
- Prometheus for monitoring (optional)

## Quick Start

1. **Update secrets and configuration:**
   ```bash
   # Edit secrets.yaml with your actual secrets
   vim k8s/secrets.yaml
   
   # Update domain names in ingress.yaml and rbac.yaml
   vim k8s/ingress.yaml
   vim k8s/rbac.yaml
   ```

2. **Deploy everything:**
   ```bash
   kubectl apply -f k8s/
   ```

3. **Check deployment status:**
   ```bash
   kubectl get all -n auth-service
   ```

4. **Access the services:**
   - Auth Service API: `https://auth.your-domain.com`
   - Hanko API: `https://hanko.your-domain.com`
   - Metrics: `http://auth-service.auth-service.svc.cluster.local:9090/metrics`

## Deployment Order

For controlled deployment, apply manifests in this order:

```bash
# 1. Namespace and RBAC
kubectl apply -f namespace.yaml
kubectl apply -f rbac.yaml

# 2. Secrets and ConfigMaps
kubectl apply -f secrets.yaml
kubectl apply -f configmap.yaml

# 3. Storage
kubectl apply -f rbac.yaml  # Contains PVCs

# 4. Dependencies
kubectl apply -f dependencies.yaml

# 5. Services
kubectl apply -f service.yaml

# 6. Main application
kubectl apply -f deployment.yaml

# 7. Ingress and scaling
kubectl apply -f ingress.yaml
```

## Configuration

### Environment-specific Settings

Before deploying, update the following files for your environment:

#### secrets.yaml
- `DATABASE_PASSWORD`: Strong PostgreSQL password
- `AUTH_TOKEN_SECRET_KEY`: 32+ character secret key
- `HANKO_API_KEY`: Your Hanko API key
- `HANKO_PROJECT_ID`: Your Hanko project ID

#### configmap.yaml
Update database hosts, URLs, and other environment-specific settings.

#### rbac.yaml (Hanko config section)
- `relying_party.id`: Your domain
- `allow_origins`: Your frontend URLs
- `cookie.domain`: Your domain

#### ingress.yaml
- `host`: Your actual domain names
- `cors-allow-origin`: Your frontend domain
- `cluster-issuer`: Your cert-manager issuer

### Scaling Configuration

The deployment includes:
- **Auth Service**: 3 replicas with HPA (2-10 replicas)
- **Hanko**: 2 replicas
- **NATS**: 3 replicas for HA
- **PostgreSQL**: 1 replica (consider external managed DB for production)

## Security Features

### Network Policies
- Restricts pod-to-pod communication
- Only allows necessary traffic flows
- Blocks unauthorized access between services

### RBAC
- Minimal permissions for service accounts
- Read-only access to required resources
- Namespace-scoped permissions

### Pod Security
- Non-root containers
- Read-only root filesystem
- Dropped capabilities
- Resource limits

## Monitoring

### Health Checks
- **Liveness probes**: Detect and restart unhealthy pods
- **Readiness probes**: Remove pods from load balancing when not ready
- **Startup probes**: Handle slow-starting applications

### Metrics
- Prometheus metrics on `/metrics`
- Pod annotations for automatic scraping
- Custom business metrics included

### Observability
- Structured JSON logging
- Distributed tracing ready
- Error tracking integration points

## Storage

### Persistent Volumes
- **PostgreSQL**: 10Gi for auth data
- **Hanko PostgreSQL**: 5Gi for auth data
- **NATS**: 2Gi for message persistence

### Backup Strategy
Consider implementing:
- Regular database backups
- Cross-region replication
- Point-in-time recovery

## Production Checklist

### Security
- [ ] Update all default passwords
- [ ] Configure proper TLS certificates
- [ ] Enable network policies
- [ ] Review RBAC permissions
- [ ] Configure image pull policies

### Performance
- [ ] Set appropriate resource requests/limits
- [ ] Configure HPA based on load testing
- [ ] Optimize database connection pooling
- [ ] Enable caching where appropriate

### Reliability
- [ ] Configure pod disruption budgets
- [ ] Set up monitoring and alerting
- [ ] Test disaster recovery procedures
- [ ] Configure backups

### Operations
- [ ] Set up log aggregation
- [ ] Configure metrics collection
- [ ] Document runbooks
- [ ] Set up CI/CD pipelines

## Troubleshooting

### Common Issues

1. **Pods not starting:**
   ```bash
   kubectl describe pod -n auth-service
   kubectl logs -n auth-service -l app.kubernetes.io/name=auth-service
   ```

2. **Database connection issues:**
   ```bash
   kubectl exec -it -n auth-service deployment/postgres -- psql -U postgres
   ```

3. **Ingress not working:**
   ```bash
   kubectl describe ingress -n auth-service
   kubectl logs -n ingress-nginx -l app.kubernetes.io/name=ingress-nginx
   ```

4. **Check service connectivity:**
   ```bash
   kubectl exec -it -n auth-service deployment/auth-service -- wget -qO- http://auth-service-postgres:5432
   ```

### Debugging Commands

```bash
# Check all resources
kubectl get all -n auth-service

# Check events
kubectl get events -n auth-service --sort-by=.metadata.creationTimestamp

# Check logs
kubectl logs -n auth-service deployment/auth-service -f

# Port forward for local testing
kubectl port-forward -n auth-service svc/auth-service 8080:8080

# Check resource usage
kubectl top pods -n auth-service
```

## Cleanup

To remove all resources:

```bash
kubectl delete namespace auth-service
```

This will remove all resources in the namespace including persistent volumes.

## Development vs Production

### Development
- Use smaller resource requests
- Single replica deployments
- Local storage classes
- Simplified networking

### Production
- Multi-replica deployments
- Proper resource limits
- Managed database services
- Load balancers and CDN
- Monitoring and alerting