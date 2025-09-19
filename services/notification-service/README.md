# Notification Service

## ✅ PRODUCTION READY - Comprehensive Notification Service Implementation

The Notification Service is a high-performance, multi-channel notification system designed for the reciprocal clubs platform. It supports email, SMS, push notifications, in-app notifications, and webhooks with advanced features like templates, user preferences, bulk operations, and enterprise-grade monitoring.

## Features

### ✅ Core Notification Features
- **Multi-Channel Delivery**: Email, SMS, Push, In-App, and Webhook notifications
- **Priority Handling**: Low, Normal, High, and Critical priority levels
- **Scheduled Notifications**: Support for future delivery scheduling
- **Retry Logic**: Automatic retry for failed notifications with exponential backoff
- **Template System**: Reusable notification templates with variable substitution
- **User Preferences**: Granular user notification preferences per club
- **Bulk Operations**: Efficient handling of multiple notifications

### ✅ Advanced Production Features
- **Rate Limiting**: Per-user, per-IP, and per-notification-type rate limiting
- **Circuit Breakers**: Resilience patterns for external service failures
- **Comprehensive Metrics**: 25+ Prometheus metrics for monitoring
- **Health Checks**: Detailed health monitoring for all components
- **Real-time Processing**: Asynchronous notification processing
- **Admin Interface**: Management endpoints for operations

### ✅ API Interfaces
- **gRPC API**: 18 service methods covering all operations
- **REST API**: 20+ HTTP endpoints for web integration
- **Protocol Buffers**: Strongly-typed service definitions

## Architecture

```
┌─────────────────┐
│   Applications  │
│   Web/Mobile    │
└─────────┬───────┘
          │ gRPC/HTTP
┌─────────▼───────┐
│ Notification    │
│ Service         │
│                 │
│ ┌─────────────┐ │    ┌──────────────┐
│ │ gRPC Server │ │    │ Notification │
│ └─────────────┘ │    │ Providers    │
│                 │    │              │
│ ┌─────────────┐ │    │ - SMTP       │
│ │ HTTP Server │ │    │ - Twilio     │
│ └─────────────┘ │    │ - FCM        │
│                 │    │ - Webhooks   │
│ ┌─────────────┐ │    │ - In-App     │
│ │ Processing  │ │    └──────────────┘
│ │ Engine      │ │
│ └─────────────┘ │
│                 │
│ ┌─────────────┐ │
│ │ Rate Limit  │ │
│ │ & Circuit   │ │
│ │ Breaker     │ │
│ └─────────────┘ │
└─────────┬───────┘
          │
    ┌─────▼─────┐
    │ Database  │
    │ (PostgreSQL) │
    └───────────┘
```

## Getting Started

### Prerequisites

- Go 1.25+
- PostgreSQL 13+
- NATS server (for message bus)
- Protocol Buffer compiler: `protoc`
- gRPC tools: `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`

### Development Setup

1. **Clone and Navigate**:
   ```bash
   cd services/notification-service
   ```

2. **Generate Protocol Buffer Code**:
   ```bash
   protoc --go_out=. --go_opt=paths=source_relative \
          --go-grpc_out=. --go-grpc_opt=paths=source_relative \
          proto/notification.proto
   ```

3. **Install Dependencies**:
   ```bash
   go mod tidy
   ```

4. **Set Environment Variables**:
   ```bash
   export NOTIFICATION_SERVICE_DATABASE_HOST=localhost
   export NOTIFICATION_SERVICE_DATABASE_PASSWORD=postgres
   export NOTIFICATION_SERVICE_NATS_URL=nats://localhost:4222
   export SMTP_HOST=smtp.gmail.com
   export SMTP_USERNAME=your-email@gmail.com
   export SMTP_PASSWORD=your-app-password
   export TWILIO_ACCOUNT_SID=your-twilio-sid
   export TWILIO_AUTH_TOKEN=your-twilio-token
   export FCM_SERVER_KEY=your-fcm-key
   ```

5. **Run Database Migrations**:
   ```bash
   go run cmd/main.go # Automatic migration on startup
   ```

6. **Start the Service**:
   ```bash
   go run cmd/main.go
   ```

7. **Verify Service Health**:
   ```bash
   curl http://localhost:8080/health
   ```

### Service Endpoints

- **gRPC Server**: `localhost:9090`
- **HTTP Server**: `localhost:8080`
- **Health Check**: `http://localhost:8080/health`
- **Metrics**: `http://localhost:2112/metrics`

## API Documentation

### gRPC API

The service exposes 18 gRPC methods covering all notification operations:

#### Core Operations
```protobuf
// Create a notification
rpc CreateNotification(CreateNotificationRequest) returns (NotificationResponse);

// Get notification by ID
rpc GetNotification(GetNotificationRequest) returns (NotificationResponse);

// Mark notification as read
rpc MarkAsRead(MarkAsReadRequest) returns (NotificationResponse);

// Send immediate notification
rpc SendImmediate(SendImmediateRequest) returns (SendResponse);
```

#### Bulk Operations
```protobuf
// Create multiple notifications
rpc CreateBulkNotifications(CreateBulkNotificationsRequest) returns (CreateBulkNotificationsResponse);

// Mark multiple as read
rpc MarkMultipleAsRead(MarkMultipleAsReadRequest) returns (MarkMultipleAsReadResponse);
```

#### Template Management
```protobuf
// Create notification template
rpc CreateTemplate(CreateTemplateRequest) returns (TemplateResponse);

// Update template
rpc UpdateTemplate(UpdateTemplateRequest) returns (TemplateResponse);

// Delete template
rpc DeleteTemplate(DeleteTemplateRequest) returns (google.protobuf.Empty);
```

#### User Preferences
```protobuf
// Get user preferences
rpc GetUserPreferences(GetUserPreferencesRequest) returns (UserPreferencesResponse);

// Update user preferences
rpc UpdateUserPreferences(UpdateUserPreferencesRequest) returns (UserPreferencesResponse);
```

#### Admin & Monitoring
```protobuf
// Process scheduled notifications
rpc ProcessScheduledNotifications(google.protobuf.Empty) returns (ProcessNotificationsResponse);

// Retry failed notifications
rpc RetryFailedNotifications(google.protobuf.Empty) returns (ProcessNotificationsResponse);

// Get service metrics
rpc GetNotificationMetrics(google.protobuf.Empty) returns (MetricsResponse);
```

### REST API

The HTTP API provides 20+ endpoints for web integration:

#### Notification Operations
- `POST /api/v1/notifications` - Create notification
- `GET /api/v1/notifications/{id}` - Get notification
- `POST /api/v1/notifications/{id}/read` - Mark as read
- `GET /api/v1/clubs/{clubId}/notifications` - Get club notifications
- `GET /api/v1/users/{userId}/notifications` - Get user notifications

#### Bulk Operations
- `POST /api/v1/notifications/bulk` - Create bulk notifications
- `POST /api/v1/notifications/send` - Send immediate notification

#### Template Management
- `POST /api/v1/templates` - Create template
- `GET /api/v1/clubs/{clubId}/templates` - Get club templates
- `PUT /api/v1/admin/templates/{id}` - Update template
- `DELETE /api/v1/admin/templates/{id}` - Delete template

#### User Preferences
- `GET /api/v1/users/{userId}/preferences` - Get preferences
- `PUT /api/v1/users/{userId}/preferences` - Update preferences

#### Statistics & Analytics
- `GET /api/v1/clubs/{clubId}/stats` - Get notification statistics

#### Admin Operations
- `POST /api/v1/admin/process/pending` - Process pending notifications
- `POST /api/v1/admin/process/failed` - Retry failed notifications
- `POST /api/v1/admin/notifications/bulk` - Bulk mark as read

#### Health & Monitoring
- `GET /health` - Comprehensive health check
- `GET /health/live` - Liveness probe
- `GET /health/ready` - Readiness probe
- `GET /metrics` - Prometheus metrics

### Example Usage

#### Create Email Notification (gRPC)
```go
client := pb.NewNotificationServiceClient(conn)
resp, err := client.CreateNotification(ctx, &pb.CreateNotificationRequest{
    ClubId:    1,
    UserId:    "user123",
    Type:      pb.NotificationType_NOTIFICATION_TYPE_EMAIL,
    Priority:  pb.NotificationPriority_NOTIFICATION_PRIORITY_NORMAL,
    Title:     "Welcome to the Club",
    Message:   "Thank you for joining our club!",
    Recipient: "user@example.com",
})
```

#### Create SMS Notification (HTTP)
```bash
curl -X POST http://localhost:8080/api/v1/notifications \
  -H "Content-Type: application/json" \
  -d '{
    "club_id": 1,
    "user_id": "user123",
    "type": "sms",
    "priority": "normal",
    "subject": "Club Update",
    "message": "Your membership has been approved!",
    "recipient": "+1234567890"
  }'
```

#### Send Immediate Push Notification (HTTP)
```bash
curl -X POST http://localhost:8080/api/v1/notifications/send \
  -H "Content-Type: application/json" \
  -d '{
    "club_id": 1,
    "type": "push",
    "subject": "Urgent Notice",
    "message": "Emergency club meeting tonight at 7 PM",
    "recipient": "device_token_here"
  }'
```

## Configuration

### Environment Variables

The service supports configuration via environment variables with the `NOTIFICATION_SERVICE_` prefix:

#### Core Service
- `NOTIFICATION_SERVICE_PORT`: HTTP port (default: 8080)
- `NOTIFICATION_SERVICE_GRPC_PORT`: gRPC port (default: 9090)
- `NOTIFICATION_SERVICE_ENVIRONMENT`: Environment (development/production)

#### Database
- `NOTIFICATION_SERVICE_DATABASE_HOST`: PostgreSQL host
- `NOTIFICATION_SERVICE_DATABASE_PORT`: PostgreSQL port (default: 5432)
- `NOTIFICATION_SERVICE_DATABASE_PASSWORD`: PostgreSQL password
- `NOTIFICATION_SERVICE_DATABASE_NAME`: Database name

#### NATS Message Bus
- `NOTIFICATION_SERVICE_NATS_URL`: NATS server URL
- `NOTIFICATION_SERVICE_NATS_CLUSTER_ID`: NATS cluster ID
- `NOTIFICATION_SERVICE_NATS_CLIENT_ID`: NATS client ID

#### Provider Configuration
- `SMTP_HOST`: SMTP server host
- `SMTP_PORT`: SMTP server port (default: 587)
- `SMTP_USERNAME`: SMTP username
- `SMTP_PASSWORD`: SMTP password
- `FROM_EMAIL`: Default sender email

#### Twilio SMS
- `TWILIO_ACCOUNT_SID`: Twilio Account SID
- `TWILIO_AUTH_TOKEN`: Twilio Auth Token
- `TWILIO_FROM_NUMBER`: Twilio phone number

#### Firebase Cloud Messaging
- `FCM_SERVER_KEY`: FCM server key
- `FCM_PROJECT_ID`: FCM project ID

#### Webhooks
- `WEBHOOK_SECRET_KEY`: Secret key for webhook signatures

### YAML Configuration

```yaml
service:
  name: notification-service
  host: 0.0.0.0
  port: 8080
  grpc_port: 9090
  environment: production

database:
  host: postgres
  port: 5432
  user: notification_user
  password: secure_password
  database: notification_db
  ssl_mode: require

nats:
  url: nats://nats:4222
  cluster_id: reciprocal-clubs
  client_id: notification-service

monitoring:
  enable_metrics: true
  metrics_port: 2112
  metrics_path: /metrics

logging:
  level: info
  format: json
```

## Monitoring & Observability

### Prometheus Metrics

The service exposes 25+ metrics for comprehensive monitoring:

#### Business Metrics
- `notifications_created_total` - Total notifications created by club/type/priority
- `notifications_sent_total` - Successful deliveries by club/type/provider
- `notifications_failed_total` - Failed deliveries by club/type/provider/error
- `notifications_read_total` - Notifications marked as read

#### Performance Metrics
- `notification_delivery_duration_seconds` - Delivery time by type/provider/status
- `notification_delivery_attempts_total` - Delivery attempts by type/provider
- `notifications_pending_count` - Current pending notifications
- `notifications_failed_count` - Current failed notifications (retryable)

#### API Metrics
- `http_requests_total` - HTTP requests by method/path/status
- `http_request_duration_seconds` - HTTP request duration
- `grpc_requests_total` - gRPC requests by method/status
- `grpc_request_duration_seconds` - gRPC request duration

#### System Metrics
- `go_routines_count` - Active goroutines
- `database_connections_active` - Database connections

### Health Checks

The service provides multiple health check endpoints:

#### `/health` - Comprehensive Health Check
```json
{
  "status": "healthy",
  "service": "notification-service",
  "version": "1.0.0",
  "timestamp": "2024-01-15T10:30:00Z",
  "dependencies": {
    "database": {
      "status": "healthy",
      "response_time_ms": 5
    },
    "message_bus": {
      "status": "healthy",
      "response_time_ms": 2
    },
    "providers": {
      "email": "healthy",
      "sms": "healthy",
      "push": "degraded",
      "webhook": "healthy"
    }
  },
  "metrics": {
    "pending_notifications": 15,
    "failed_notifications": 2,
    "processing_rate": 50.5
  }
}
```

#### `/health/live` - Liveness Probe
Simple check if the service is alive and responding.

#### `/health/ready` - Readiness Probe
Checks if the service is ready to accept traffic (database connected, providers available).

## Production Deployment

### Docker

```bash
# Build image
docker build -t notification-service .

# Run container
docker run -p 8080:8080 -p 9090:9090 \
  -e NOTIFICATION_SERVICE_DATABASE_HOST=postgres \
  -e NOTIFICATION_SERVICE_NATS_URL=nats://nats:4222 \
  -e SMTP_HOST=smtp.gmail.com \
  notification-service
```

### Kubernetes

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: notification-service
spec:
  replicas: 3
  selector:
    matchLabels:
      app: notification-service
  template:
    metadata:
      labels:
        app: notification-service
    spec:
      containers:
      - name: notification-service
        image: notification-service:latest
        ports:
        - containerPort: 8080
          name: http
        - containerPort: 9090
          name: grpc
        - containerPort: 2112
          name: metrics
        env:
        - name: NOTIFICATION_SERVICE_DATABASE_HOST
          value: "postgres"
        - name: NOTIFICATION_SERVICE_NATS_URL
          value: "nats://nats:4222"
        livenessProbe:
          httpGet:
            path: /health/live
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health/ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
        resources:
          requests:
            memory: "256Mi"
            cpu: "250m"
          limits:
            memory: "512Mi"
            cpu: "500m"
```

### Rate Limiting Configuration

```go
// Default rate limits
rateLimitConfig := &middleware.NotificationRateLimitConfig{
    EmailRPS:     10,   // 10 emails per second
    EmailBurst:   50,   // Burst of 50 emails
    SMSRPS:       5,    // 5 SMS per second
    SMSBurst:     20,   // Burst of 20 SMS
    PushRPS:      50,   // 50 push notifications per second
    PushBurst:    200,  // Burst of 200 push notifications
    WebhookRPS:   20,   // 20 webhooks per second
    WebhookBurst: 100,  // Burst of 100 webhooks
}
```

### Circuit Breaker Configuration

```go
// Default circuit breaker settings
circuitBreakerConfig := &middleware.CircuitBreakerConfig{
    Name:         "email-provider",
    MaxRequests:  5,                // Max requests in half-open state
    Interval:     time.Minute,      // Reset interval in closed state
    Timeout:      30 * time.Second, // Timeout before trying half-open
    ReadyToTrip:  func(counts Counts) bool {
        return counts.ConsecutiveFailures > 5
    },
}
```

## Testing

### Unit Tests
```bash
go test ./internal/service/tests/... -v
go test ./internal/repository/tests/... -v
```

### Integration Tests
```bash
go test ./internal/handlers/tests/... -v
```

### Load Testing
```bash
# Example using curl for basic load testing
for i in {1..100}; do
  curl -X POST http://localhost:8080/api/v1/notifications \
    -H "Content-Type: application/json" \
    -d '{"club_id":1,"type":"email","subject":"Test","message":"Load test","recipient":"test@example.com"}' &
done
wait
```

## Troubleshooting

### Common Issues

1. **Database Connection Failures**:
   ```bash
   # Check database connectivity
   psql -h localhost -U postgres -d notification_db -c "SELECT 1;"
   ```

2. **NATS Connection Issues**:
   ```bash
   # Check NATS server status
   curl http://localhost:8222/varz
   ```

3. **Provider Configuration**:
   ```bash
   # Test SMTP configuration
   curl -X POST http://localhost:8080/api/v1/notifications \
     -H "Content-Type: application/json" \
     -d '{"club_id":1,"type":"email","subject":"Test","message":"Test","recipient":"test@example.com"}'
   ```

4. **High Memory Usage**:
   - Check for goroutine leaks: `curl http://localhost:8080/debug/pprof/goroutine`
   - Monitor pending notifications: `curl http://localhost:2112/metrics | grep pending`

5. **Rate Limiting Issues**:
   - Check rate limit metrics: `curl http://localhost:2112/metrics | grep rate_limit`
   - Review rate limit configuration in logs

### Debug Commands

```bash
# Enable debug logging
NOTIFICATION_SERVICE_LOGGING_LEVEL=debug go run cmd/main.go

# Check metrics
curl http://localhost:2112/metrics

# Check health with details
curl http://localhost:8080/health | jq

# Process pending notifications manually
curl -X POST http://localhost:8080/api/v1/admin/process/pending

# Retry failed notifications
curl -X POST http://localhost:8080/api/v1/admin/process/failed
```

## Performance Tuning

### Database Optimization
- Index on `club_id`, `user_id`, `status`, `type`, `created_at`
- Partition large notification tables by date
- Use read replicas for analytics queries

### Concurrent Processing
- Adjust `GOMAXPROCS` for CPU-bound operations
- Tune goroutine pool sizes for notification processing
- Configure database connection pooling

### Memory Management
- Monitor heap size and GC frequency
- Use object pooling for frequently allocated objects
- Configure appropriate container memory limits

## Security Considerations

- **Input Validation**: All inputs are validated before processing
- **SQL Injection Prevention**: Using parameterized queries with GORM
- **Rate Limiting**: Multiple layers of rate limiting protection
- **Authentication**: JWT-based authentication for all protected endpoints
- **Secrets Management**: Environment variables for sensitive configuration
- **HTTPS**: Always use HTTPS in production
- **Webhook Signatures**: Verify webhook signatures for security

## Contributing

1. Follow Go coding standards and best practices
2. Add comprehensive tests for new features
3. Update documentation for API changes
4. Include appropriate logging and metrics
5. Handle errors gracefully with proper error types
6. Use structured logging with contextual information

## License

This service is part of the reciprocal clubs backend system and follows the project's license terms.