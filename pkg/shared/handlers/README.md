# Shared Handlers Package

This package provides reusable HTTP and gRPC handlers, middleware, and utilities for the reciprocal clubs backend services.

## Overview

The handlers package includes comprehensive functionality for:

- **HTTP Request Handling**: Standardized response formatting, error handling, and request parsing
- **Authentication**: Complete auth handlers for login, registration, token refresh, and logout
- **Middleware**: Security, logging, metrics, CORS, rate limiting, compression, and recovery
- **gRPC Services**: Interceptors, error handling, and common service utilities
- **Validation**: Input validation helpers for both HTTP and gRPC

## Components

### HTTP Handlers

#### `HTTPHandler` (`http.go`)
- Standardized JSON response formatting
- Error handling and status code conversion
- Query parameter parsing with pagination support
- Request body parsing and validation
- Security headers and CORS support

**Key Methods:**
```go
func (h *HTTPHandler) WriteResponse(w http.ResponseWriter, r *http.Request, statusCode int, data interface{})
func (h *HTTPHandler) WriteError(w http.ResponseWriter, r *http.Request, err error)
func (h *HTTPHandler) ParseJSONBody(r *http.Request, target interface{}) error
func (h *HTTPHandler) ParseQueryParams(r *http.Request) (*QueryParams, error)
```

#### `AuthHandler` (`auth.go`)
Complete authentication handling with JWT support.

**Endpoints:**
- `POST /login` - User authentication
- `POST /register` - User registration  
- `POST /refresh` - Token refresh
- `POST /logout` - User logout
- `GET /me` - Current user info
- `PUT /password` - Password change

**Usage:**
```go
authHandler := handlers.NewAuthHandler(logger, jwtProvider)
http.HandleFunc("/auth/login", authHandler.Login)
http.HandleFunc("/auth/register", authHandler.Register)
```

### Middleware (`middleware.go`)

#### Available Middleware
- **RequestIDMiddleware**: Adds unique request IDs
- **LoggingMiddleware**: Request/response logging
- **MetricsMiddleware**: Prometheus metrics collection
- **CORSMiddleware**: Cross-origin resource sharing
- **RateLimitMiddleware**: Rate limiting with configurable limits
- **CompressionMiddleware**: Gzip response compression
- **SecurityHeadersMiddleware**: Security headers (HSTS, CSP, etc.)
- **RecoveryMiddleware**: Panic recovery
- **TimeoutMiddleware**: Request timeouts

#### Middleware Chain
```go
chain := handlers.NewMiddlewareChain()
chain.Use(handlers.RecoveryMiddleware(logger))
chain.Use(handlers.SecurityHeadersMiddleware())
chain.Use(handlers.CORSMiddleware(corsConfig))
chain.Use(handlers.RateLimitMiddleware(logger, rateLimitConfig))

handler := chain.Handler(yourHandler)
```

### gRPC Handlers (`grpc.go`)

#### `GRPCHandler`
- Unary and streaming interceptors
- Error handling and status code conversion
- Request validation
- Panic recovery
- Metrics collection

**Usage:**
```go
grpcHandler := handlers.NewGRPCHandler(logger, monitor)
unary, stream := grpcHandler.GetDefaultInterceptors()

server := grpc.NewServer(
    grpc.ChainUnaryInterceptor(unary...),
    grpc.ChainStreamInterceptor(stream...),
)
```

#### `BaseService`
Common functionality for gRPC services:
- Request/response logging
- User and club ID extraction from context
- Error handling
- Request validation

```go
service := handlers.NewBaseService(logger, monitor)
userID, err := service.GetUserID(ctx)
if err := service.ValidateRequest(req); err != nil {
    return nil, service.HandleError(err)
}
```

## Response Format

### Standard HTTP Response
```json
{
  "success": true,
  "data": {...},
  "meta": {
    "pagination": {
      "current_page": 1,
      "page_size": 20,
      "total_pages": 5,
      "total_items": 100,
      "has_next": true,
      "has_prev": false
    },
    "duration": "45ms"
  },
  "timestamp": "2023-12-07T10:30:00.000Z"
}
```

### Error Response
```json
{
  "success": false,
  "error": {
    "code": "INVALID_INPUT",
    "message": "Email is required",
    "details": {
      "field": "email"
    }
  },
  "timestamp": "2023-12-07T10:30:00.000Z"
}
```

## Configuration

### CORS Configuration
```go
corsConfig := handlers.CORSConfig{
    AllowedOrigins:   []string{"https://example.com"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
    AllowedHeaders:   []string{"Content-Type", "Authorization"},
    AllowCredentials: true,
    MaxAge:           86400,
}
```

### Rate Limiting Configuration
```go
rateLimitConfig := handlers.RateLimitConfig{
    RequestsPerSecond: 100,
    Burst:             200,
    KeyFunc: func(r *http.Request) string {
        return getClientIP(r)
    },
}
```

## Error Handling

The package uses the shared `errors.AppError` type for consistent error handling across HTTP and gRPC interfaces. Errors are automatically converted to appropriate status codes:

- `ErrNotFound` → HTTP 404 / gRPC NotFound
- `ErrInvalidInput` → HTTP 400 / gRPC InvalidArgument
- `ErrUnauthorized` → HTTP 401 / gRPC Unauthenticated
- `ErrForbidden` → HTTP 403 / gRPC PermissionDenied
- `ErrConflict` → HTTP 409 / gRPC AlreadyExists
- `ErrTimeout` → HTTP 408 / gRPC DeadlineExceeded
- `ErrUnavailable` → HTTP 503 / gRPC Unavailable
- `ErrInternal` → HTTP 500 / gRPC Internal

## Validation

### HTTP Validation
```go
// Validate required fields
if err := handlers.ValidateRequiredField(req.Email, "email"); err != nil {
    return err
}

// Validate ID fields
if err := handlers.ValidateIDField(req.UserID, "user_id"); err != nil {
    return err
}

// Validate enum fields
validStatuses := []string{"active", "inactive", "pending"}
if err := handlers.ValidateEnumField(req.Status, validStatuses, "status"); err != nil {
    return err
}
```

### gRPC Validation
```go
// Validate IDs
if err := handlers.ValidateID(req.GetUserId(), "user_id"); err != nil {
    return nil, status.Error(codes.InvalidArgument, err.Error())
}

// Validate pagination
if err := handlers.ValidatePagination(req.GetPage(), req.GetPageSize()); err != nil {
    return nil, status.Error(codes.InvalidArgument, err.Error())
}
```

## Dependencies

The package relies on several shared components:
- `pkg/shared/auth` - Authentication provider interface
- `pkg/shared/errors` - Application error types
- `pkg/shared/logging` - Structured logging
- `pkg/shared/monitoring` - Metrics collection
- `pkg/shared/utils` - Utility functions

External dependencies:
- `google.golang.org/grpc` - gRPC framework
- `golang.org/x/time/rate` - Rate limiting

## Usage Examples

### Complete HTTP Server Setup
```go
// Initialize dependencies
logger := logging.NewLogger(loggingConfig, "api-service")
monitor := monitoring.NewMonitor(monitoringConfig, logger, "api-service", "v1.0.0")
authProvider := auth.NewJWTProvider(authConfig, logger)

// Create handlers
httpHandler := handlers.NewHTTPHandler(logger)
authHandler := handlers.NewAuthHandler(logger, authProvider)

// Setup middleware chain
chain := handlers.NewHTTPHandlerChain(logger, monitor)
chain.Use(handlers.CORSMiddleware(corsConfig))
chain.Use(handlers.RateLimitMiddleware(logger, rateLimitConfig))

// Setup routes
router := http.NewServeMux()
router.HandleFunc("/auth/login", authHandler.Login)
router.HandleFunc("/auth/register", authHandler.Register)
router.HandleFunc("/health", handlers.HealthCheckHandler())

// Apply middleware and start server
server := &http.Server{
    Handler: chain.Handler(router),
    Addr:    ":8080",
}
```

### Complete gRPC Server Setup
```go
// Initialize dependencies
logger := logging.NewLogger(loggingConfig, "grpc-service")
monitor := monitoring.NewMonitor(monitoringConfig, logger, "grpc-service", "v1.0.0")

// Create gRPC handler and interceptors
grpcHandler := handlers.NewGRPCHandler(logger, monitor)
unaryInterceptors, streamInterceptors := grpcHandler.GetDefaultInterceptors()

// Create server with interceptors
opts := handlers.DefaultServerOptions()
serverOpts := opts.ApplyServerOptions()
serverOpts = append(serverOpts, grpc.ChainUnaryInterceptor(unaryInterceptors...))
serverOpts = append(serverOpts, grpc.ChainStreamInterceptor(streamInterceptors...))

server := grpc.NewServer(serverOpts...)
```

This package provides a solid foundation for building consistent, well-structured HTTP and gRPC services in the reciprocal clubs backend ecosystem.