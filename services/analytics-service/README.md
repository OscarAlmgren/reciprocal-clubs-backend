# Analytics Service

The Analytics Service is a comprehensive microservice for collecting, processing, and analyzing club data within the reciprocal clubs network. It provides real-time metrics, report generation, data export capabilities, and advanced analytics features.

## 🚀 Features

### Core Analytics
- **Event Tracking**: Record and analyze member activities, facility usage, and system events
- **Metrics Collection**: Capture quantitative data with tags and metadata
- **Real-time Processing**: Live metrics and statistics with sub-second updates
- **Batch Operations**: Bulk event recording for high-throughput scenarios

### Reporting & Dashboards
- **Automated Report Generation**: Usage, engagement, performance, and financial reports
- **Custom Dashboards**: Interactive analytics dashboards with configurable panels
- **Scheduled Reports**: Automated report generation and delivery
- **Report Templates**: Pre-built report formats for common analytics needs

### Data Export & Integration
- **Multiple Export Formats**: JSON, CSV, Excel, and PDF export options
- **External Integrations**: Elasticsearch, DataDog, Grafana, BigQuery, and S3
- **Data Streaming**: Real-time event streaming capabilities
- **API Access**: Comprehensive REST and gRPC APIs

### Advanced Analytics
- **Trend Analysis**: Statistical trend detection and forecasting
- **Correlation Analysis**: Cross-metric correlation detection
- **Predictive Analytics**: Machine learning-based predictions
- **Anomaly Detection**: Automated detection of unusual patterns

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP API      │    │    gRPC API     │    │  Event Stream   │
├─────────────────┤    ├─────────────────┤    ├─────────────────┤
│                 │    │                 │    │                 │
│ REST Endpoints  │    │ Proto Buffers   │    │ NATS Streaming  │
│ JSON Responses  │    │ Type Safety     │    │ Real-time Data  │
│                 │    │                 │    │                 │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────┴─────────────┐
                    │     Service Layer        │
                    ├───────────────────────────┤
                    │                           │
                    │ • Business Logic          │
                    │ • Event Processing        │
                    │ • Report Generation       │
                    │ • Analytics Computation   │
                    │                           │
                    └─────────────┬─────────────┘
                                  │
                    ┌─────────────┴─────────────┐
                    │   Repository Layer       │
                    ├───────────────────────────┤
                    │                           │
                    │ • Database Operations     │
                    │ • Data Aggregation        │
                    │ • Query Optimization      │
                    │ • Schema Management       │
                    │                           │
                    └─────────────┬─────────────┘
                                  │
            ┌─────────────────────┼─────────────────────┐
            │                     │                     │
   ┌────────┴────────┐   ┌────────┴────────┐   ┌────────┴────────┐
   │   PostgreSQL    │   │ External APIs   │   │  File Storage   │
   │                 │   │                 │   │                 │
   │ • Events        │   │ • Elasticsearch │   │ • S3 Exports    │
   │ • Metrics       │   │ • DataDog       │   │ • Local Files   │
   │ • Reports       │   │ • Grafana       │   │ • Temp Storage  │
   │ • Dashboards    │   │ • BigQuery      │   │                 │
   │                 │   │                 │   │                 │
   └─────────────────┘   └─────────────────┘   └─────────────────┘
```

## 📚 API Documentation

### HTTP REST API

#### Core Analytics Endpoints

**Get Metrics**
```http
GET /api/v1/analytics/metrics?club_id=club123&time_range=24h
```

**Record Event**
```http
POST /api/v1/analytics/events
Content-Type: application/json

{
  "club_id": "club123",
  "event_type": "member_visit",
  "user_id": "user456",
  "data": {
    "facility": "gym",
    "duration": 60
  },
  "metadata": {
    "device": "mobile"
  }
}
```

**Record Metric**
```http
POST /api/v1/analytics/metrics
Content-Type: application/json

{
  "club_id": "club123",
  "metric_name": "visitor_count",
  "metric_value": 25.0,
  "tags": {
    "location": "entrance",
    "time_period": "peak"
  }
}
```

**Generate Report**
```http
POST /api/v1/analytics/reports/generate
Content-Type: application/json

{
  "club_id": "club123",
  "report_type": "usage"
}
```

#### Dashboard Operations

**List Dashboards**
```http
GET /api/v1/analytics/dashboards?club_id=club123
```

**Create Dashboard**
```http
POST /api/v1/analytics/dashboards
Content-Type: application/json

{
  "club_id": "club123",
  "name": "Member Analytics",
  "description": "Member activity dashboard",
  "panels": {
    "visitor_chart": {
      "type": "line",
      "query": "visitor_count"
    }
  },
  "is_public": false
}
```

#### Data Export

**Export Events**
```http
GET /api/v1/analytics/export/events?club_id=club123&format=csv&time_range=7d
```

**Export Metrics**
```http
GET /api/v1/analytics/export/metrics?club_id=club123&format=json&time_range=30d
```

### gRPC API

The service also provides a comprehensive gRPC API defined in `proto/analytics.proto` with 25+ methods covering:

- Core analytics operations (GetMetrics, RecordEvent, RecordMetric)
- Real-time analytics (GetRealtimeMetrics, StreamEvents)
- Report generation (GenerateReport, GetReportStatus)
- Dashboard management (CreateDashboard, UpdateDashboard)
- Data export (ExportData, SendMetricsToExternal)
- System operations (GetSystemHealth, CleanupOldData)
- Advanced analytics (GetTrendAnalysis, GetAnomalyDetection)

## 🔧 Configuration

### Environment Variables

```bash
# Service Configuration
SERVICE_NAME=analytics-service
SERVICE_PORT=8080
SERVICE_GRPC_PORT=9090
SERVICE_ENVIRONMENT=development

# Database Configuration
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=analytics
DATABASE_PASSWORD=secret
DATABASE_NAME=analytics_db
DATABASE_SSL_MODE=disable

# NATS Configuration
NATS_URL=nats://localhost:4222
NATS_CLUSTER_ID=reciprocal-clubs
NATS_CLIENT_ID=analytics-service

# Redis Configuration (for caching)
REDIS_HOST=localhost
REDIS_PORT=6379

# External Integrations
ELASTICSEARCH_URL=http://localhost:9200
DATADOG_API_KEY=your_api_key
GRAFANA_URL=http://localhost:3000
BIGQUERY_PROJECT_ID=your_project
S3_BUCKET=analytics-exports
```

### Configuration File

Create `config/config.yaml`:

```yaml
service:
  name: analytics-service
  version: 1.0.0
  environment: development
  host: 0.0.0.0
  port: 8080
  grpc_port: 9090
  timeout: 30

database:
  host: localhost
  port: 5432
  user: analytics
  password: secret
  database: analytics_db
  ssl_mode: disable
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 3600

logging:
  level: info
  format: json
  output: stdout
  time_format: "2006-01-02T15:04:05Z07:00"

monitoring:
  enabled: true
  metrics_path: /metrics
  health_path: /health

integrations:
  elasticsearch:
    url: http://localhost:9200
    username: ""
    password: ""
    index: analytics-events

  datadog:
    api_key: your_api_key
    app_key: your_app_key
    site: datadoghq.com
    namespace: analytics

  grafana:
    url: http://localhost:3000
    api_key: your_api_key
    org_id: 1

  bigquery:
    project_id: your_project
    dataset_id: analytics
    credentials_path: /path/to/credentials.json

  s3:
    region: us-west-2
    bucket: analytics-exports
    access_key: your_access_key
    secret_key: your_secret_key
    path_prefix: exports/
```

## 🚀 Getting Started

### Prerequisites

- Go 1.21+
- PostgreSQL 14+
- NATS Server 2.9+
- Redis 6.0+ (optional, for caching)

### Installation

1. **Clone the repository**
```bash
git clone https://github.com/your-org/reciprocal-clubs-backend.git
cd reciprocal-clubs-backend/services/analytics-service
```

2. **Install dependencies**
```bash
make deps
```

3. **Generate protocol buffers**
```bash
make proto
```

4. **Set up the database**
```bash
# Create database
createdb analytics_db

# Run migrations (if available)
make db-migrate
```

5. **Build the service**
```bash
make build
```

6. **Run the service**
```bash
make run
```

### Development Setup

1. **Install development tools**
```bash
make setup
```

2. **Run in development mode with hot reload**
```bash
make dev
```

3. **Run tests**
```bash
make test
```

4. **Run with coverage**
```bash
make coverage
```

### Docker Setup

1. **Build Docker image**
```bash
make docker-build
```

2. **Run with Docker Compose**
```bash
make docker-compose-up
```

## 🧪 Testing

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run benchmark tests
make test-bench

# Generate coverage report
make coverage
```

### Test Structure

```
internal/
├── repository/
│   └── repository_test.go      # Database layer tests
├── service/
│   └── service_test.go         # Business logic tests
└── handlers/
    ├── grpc/
    │   └── handler_test.go     # gRPC API tests
    └── http/
        └── handler_test.go     # HTTP API tests
```

### Test Categories

- **Unit Tests**: Fast, isolated tests for individual components
- **Integration Tests**: Tests with real database and external services
- **Benchmark Tests**: Performance tests for critical paths
- **Contract Tests**: API contract validation tests

## 📊 Monitoring & Observability

### Health Checks

- **Health**: `/health` - Overall service health
- **Readiness**: `/ready` - Ready to accept traffic
- **Liveness**: `/live` - Service is running

### Metrics

The service exposes Prometheus metrics at `/metrics`:

- **Business Metrics**: Events processed, reports generated, dashboards created
- **Performance Metrics**: Request duration, queue size, processing errors
- **System Metrics**: Memory usage, CPU usage, goroutine count
- **Integration Metrics**: External API calls, latency, errors

### Logging

Structured JSON logging with the following fields:
- `timestamp`: ISO 8601 timestamp
- `level`: Log level (debug, info, warn, error, fatal)
- `service`: Service name
- `correlation_id`: Request correlation ID
- `user_id`: User ID (when available)
- `club_id`: Club ID (when available)

### Distributed Tracing

Integration with OpenTelemetry for distributed tracing across the microservices ecosystem.

## 🔐 Security

### Authentication & Authorization

- JWT token validation for API access
- Club-based data isolation
- Role-based access control (RBAC)
- API key authentication for service-to-service communication

### Data Protection

- Encryption at rest for sensitive data
- TLS encryption for all network communication
- Data anonymization for exports
- GDPR compliance features

### Security Best Practices

- Input validation and sanitization
- SQL injection prevention
- Rate limiting and throttling
- Security headers for HTTP responses

## 🚀 Deployment

### Kubernetes

Deploy using the provided Kubernetes manifests:

```bash
kubectl apply -f deployments/k8s/
```

### Docker Compose

For local development:

```bash
docker-compose up analytics-service
```

### Environment-specific Configurations

- **Development**: Local database, debug logging, mock integrations
- **Staging**: Shared database, info logging, test integrations
- **Production**: Clustered database, warn logging, full integrations

## 🔧 Maintenance

### Database Management

- **Migrations**: Use built-in migration system
- **Backups**: Automated daily backups
- **Cleanup**: Configurable data retention policies

### Performance Optimization

- **Indexing**: Optimized database indexes for common queries
- **Caching**: Redis caching for frequently accessed data
- **Connection Pooling**: Efficient database connection management
- **Query Optimization**: Analyzed and optimized slow queries

### Scaling

- **Horizontal Scaling**: Stateless service design supports multiple instances
- **Load Balancing**: Built-in health checks for load balancer integration
- **Resource Limits**: Configured CPU and memory limits
- **Auto-scaling**: Kubernetes HPA configuration available

## 🤝 Contributing

### Development Workflow

1. Create a feature branch
2. Make changes with tests
3. Run linting and tests: `make ci`
4. Submit a pull request

### Code Standards

- Go formatting: `make fmt`
- Linting: `make lint`
- Testing: Maintain >80% test coverage
- Documentation: Update README and API docs

### Git Hooks

Set up pre-commit hooks:

```bash
make setup-hooks
```

## 📈 Roadmap

### Near Term (Next 3 months)
- [ ] Machine learning integration for predictive analytics
- [ ] Real-time alerting system
- [ ] Advanced data visualization components
- [ ] Multi-tenant architecture enhancements

### Medium Term (6 months)
- [ ] Stream processing with Apache Kafka
- [ ] Time-series database integration (InfluxDB)
- [ ] Advanced security features (data encryption)
- [ ] Mobile SDK for direct event tracking

### Long Term (12 months)
- [ ] AI-powered insights and recommendations
- [ ] Custom analytics DSL
- [ ] Enterprise reporting features
- [ ] Data mesh architecture implementation

## 🆘 Troubleshooting

### Common Issues

**Service won't start**
- Check database connectivity
- Verify NATS server is running
- Check environment variables

**High memory usage**
- Review query patterns
- Check for memory leaks in event processing
- Monitor goroutine count

**Slow performance**
- Enable database query logging
- Check database indexes
- Monitor external integration latency

### Support

- **Documentation**: [Internal Wiki](https://wiki.company.com/analytics-service)
- **Issue Tracking**: [GitHub Issues](https://github.com/your-org/reciprocal-clubs-backend/issues)
- **Slack**: #analytics-service channel

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.

---

**Analytics Service** - Part of the Reciprocal Clubs Backend Ecosystem
Built with ❤️ using Go, PostgreSQL, and modern microservices architecture.