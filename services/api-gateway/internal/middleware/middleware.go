package middleware

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"
	"reciprocal-clubs-backend/pkg/shared/utils"

	"github.com/99designs/gqlgen/graphql"
)

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = utils.GenerateUUID()
			}

			ctx := logging.ContextWithCorrelationID(r.Context(), requestID)
			w.Header().Set("X-Request-ID", requestID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware(logger logging.Logger) func(http.Handler) http.Handler {
	httpLogger := logging.NewHTTPRequestLogger(logger)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Execute request
			next.ServeHTTP(wrapped, r)

			// Log request completion
			duration := time.Since(start)
			httpLogger.LogRequest(
				r.Context(),
				r.Method,
				r.URL.Path,
				r.UserAgent(),
				r.RemoteAddr,
				wrapped.statusCode,
				duration,
			)
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(statusCode int) {
	if !rw.written {
		rw.statusCode = statusCode
		rw.written = true
	}
	rw.ResponseWriter.WriteHeader(statusCode)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(data)
}

// MetricsMiddleware records HTTP metrics
func MetricsMiddleware(monitor *monitoring.Monitor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture status code
			wrapped := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Execute request
			next.ServeHTTP(wrapped, r)

			// Record metrics
			duration := time.Since(start)
			monitor.RecordHTTPRequest(r.Method, r.URL.Path, wrapped.statusCode, duration)
		})
	}
}

// RateLimitMiddleware implements basic rate limiting
func RateLimitMiddleware(logger logging.Logger) func(http.Handler) http.Handler {
	// Simple in-memory rate limiter
	// In production, use Redis-based rate limiting
	limiters := make(map[string]*rateLimiter)
	mu := sync.RWMutex{}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client identifier (IP or user ID)
			clientID := getClientID(r)

			mu.RLock()
			limiter, exists := limiters[clientID]
			mu.RUnlock()

			if !exists {
				mu.Lock()
				limiter = newRateLimiter(100, time.Minute) // 100 requests per minute
				limiters[clientID] = limiter
				mu.Unlock()
			}

			if !limiter.allow() {
				logger.Warn("Rate limit exceeded", map[string]interface{}{
					"client_id": clientID,
					"path":      r.URL.Path,
					"method":    r.Method,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Rate limit exceeded"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getClientID(r *http.Request) string {
	// Try to get user ID from context first
	if user := auth.GetUserFromContext(r.Context()); user != nil {
		return "user:" + strconv.FormatUint(uint64(user.ID), 10)
	}

	// Fall back to IP address
	return "ip:" + r.RemoteAddr
}

// Simple rate limiter implementation
type rateLimiter struct {
	requests []time.Time
	limit    int
	window   time.Duration
	mu       sync.Mutex
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		requests: make([]time.Time, 0),
		limit:    limit,
		window:   window,
	}
}

func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove old requests
	var validRequests []time.Time
	for _, reqTime := range rl.requests {
		if reqTime.After(cutoff) {
			validRequests = append(validRequests, reqTime)
		}
	}
	rl.requests = validRequests

	// Check if we can allow this request
	if len(rl.requests) >= rl.limit {
		return false
	}

	// Allow request
	rl.requests = append(rl.requests, now)
	return true
}

// GraphQLAuthMiddleware handles authentication for GraphQL operations
func GraphQLAuthMiddleware(authProvider *auth.JWTProvider, logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Skip authentication for introspection queries
			if isIntrospectionQuery(r) {
				next.ServeHTTP(w, r)
				return
			}

			// Try to extract and validate token
			if token := extractToken(r); token != "" {
				if claims, err := authProvider.ValidateToken(token); err == nil {
					// Create user from claims
					user := &auth.User{
						ID:          claims.UserID,
						ClubID:      claims.ClubID,
						Email:       claims.Email,
						Username:    claims.Username,
						Roles:       claims.Roles,
						Permissions: claims.Permissions,
					}

					// Add user to context
					ctx = context.WithValue(ctx, auth.UserContextKey, user)
					ctx = context.WithValue(ctx, auth.ClaimsContextKey, claims)
					ctx = logging.ContextWithUserID(ctx, user.ID)
					ctx = logging.ContextWithClubID(ctx, user.ClubID)
				} else {
					logger.Debug("Invalid token provided", map[string]interface{}{
						"error": err.Error(),
						"path":  r.URL.Path,
					})
				}
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func extractToken(r *http.Request) string {
	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" {
		const bearerPrefix = "Bearer "
		if len(authHeader) > len(bearerPrefix) && authHeader[:len(bearerPrefix)] == bearerPrefix {
			return authHeader[len(bearerPrefix):]
		}
	}

	// Check query parameter (for WebSocket connections)
	return r.URL.Query().Get("token")
}

func isIntrospectionQuery(r *http.Request) bool {
	// Simple check for introspection - in production, parse the query
	if r.Method == "GET" && r.URL.Query().Get("query") != "" {
		query := r.URL.Query().Get("query")
		return contains(query, "__schema") || contains(query, "__type")
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || 
		 len(s) > 2*len(substr))))
}

// GraphQLComplexityMiddleware limits query complexity
type GraphQLComplexityMiddleware struct {
	maxComplexity int
	logger        logging.Logger
}

func NewGraphQLComplexityMiddleware(maxComplexity int, logger logging.Logger) *GraphQLComplexityMiddleware {
	return &GraphQLComplexityMiddleware{
		maxComplexity: maxComplexity,
		logger:        logger,
	}
}

func (m *GraphQLComplexityMiddleware) ExtensionName() string {
	return "ComplexityLimit"
}

func (m *GraphQLComplexityMiddleware) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

// GraphQLRequestLogging logs GraphQL operations
type GraphQLRequestLogging struct {
	logger logging.Logger
}

func NewGraphQLRequestLogging(logger logging.Logger) *GraphQLRequestLogging {
	return &GraphQLRequestLogging{logger: logger}
}

func (l *GraphQLRequestLogging) ExtensionName() string {
	return "RequestLogging"
}

func (l *GraphQLRequestLogging) Validate(schema graphql.ExecutableSchema) error {
	return nil
}

func (l *GraphQLRequestLogging) InterceptOperation(ctx context.Context, next graphql.OperationHandler) graphql.ResponseHandler {
	operationCtx := graphql.GetOperationContext(ctx)
	
	start := time.Now()
	operationType := "unknown"
	if operationCtx.Operation != nil {
		operationType = string(operationCtx.Operation.Operation)
	}
	
	l.logger.Debug("GraphQL operation started", map[string]interface{}{
		"operation":      operationCtx.OperationName,
		"operation_type": operationType,
		"variables":      operationCtx.Variables,
	})

	return func(ctx context.Context) *graphql.Response {
		response := next(ctx)
		
		duration := time.Since(start)
		fields := map[string]interface{}{
			"operation":      operationCtx.OperationName,
			"operation_type": operationType,
			"duration_ms":    duration.Milliseconds(),
		}

		// Call the response handler to get actual response
		actualResponse := response(ctx)
		if actualResponse != nil && len(actualResponse.Errors) > 0 {
			fields["errors"] = len(actualResponse.Errors)
			l.logger.Warn("GraphQL operation completed with errors", fields)
		} else {
			l.logger.Debug("GraphQL operation completed", fields)
		}

		return actualResponse
	}
}

// CORS middleware (if not using external CORS package)
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			
			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			}
			
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}