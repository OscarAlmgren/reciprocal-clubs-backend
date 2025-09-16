package handlers

import (
	"bufio"
	"compress/gzip"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"reciprocal-clubs-backend/pkg/shared/errors"
	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/pkg/shared/monitoring"

	"golang.org/x/time/rate"
)

// MiddlewareChain represents a chain of middleware
type MiddlewareChain struct {
	middlewares []func(http.Handler) http.Handler
}

// NewMiddlewareChain creates a new middleware chain
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]func(http.Handler) http.Handler, 0),
	}
}

// Use adds a middleware to the chain
func (m *MiddlewareChain) Use(middleware func(http.Handler) http.Handler) *MiddlewareChain {
	m.middlewares = append(m.middlewares, middleware)
	return m
}

// Handler applies all middleware to the given handler
func (m *MiddlewareChain) Handler(handler http.Handler) http.Handler {
	// Apply middleware in reverse order so they execute in the correct order
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		handler = m.middlewares[i](handler)
	}
	return handler
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
				r.Header.Set("X-Request-ID", requestID)
			}

			// Add request ID to context
			ctx := context.WithValue(r.Context(), "request_id", requestID)
			ctx = logging.ContextWithCorrelationID(ctx, requestID)

			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// LoggingMiddleware logs HTTP requests and responses
func LoggingMiddleware(logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create response writer wrapper to capture status code
			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// Log incoming request
			logger.WithContext(r.Context()).Info("HTTP request started", map[string]interface{}{
				"method":     r.Method,
				"path":       r.URL.Path,
				"query":      r.URL.RawQuery,
				"remote_ip":  getClientIP(r),
				"user_agent": r.Header.Get("User-Agent"),
			})

			next.ServeHTTP(wrapper, r)

			duration := time.Since(start)

			// Log response
			fields := map[string]interface{}{
				"method":        r.Method,
				"path":          r.URL.Path,
				"status_code":   wrapper.statusCode,
				"duration_ms":   duration.Milliseconds(),
				"response_size": wrapper.bytesWritten,
			}

			switch {
			case wrapper.statusCode >= 500:
				logger.WithContext(r.Context()).Error("HTTP request completed with server error", fields)
			case wrapper.statusCode >= 400:
				logger.WithContext(r.Context()).Warn("HTTP request completed with client error", fields)
			default:
				logger.WithContext(r.Context()).Info("HTTP request completed", fields)
			}
		})
	}
}

// MetricsMiddleware records metrics for HTTP requests
func MetricsMiddleware(monitor *monitoring.Monitor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			wrapper := &responseWriterWrapper{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			next.ServeHTTP(wrapper, r)

			duration := time.Since(start)

			// Record metrics
			monitor.RecordHTTPRequest(
				r.Method,
				r.URL.Path,
				wrapper.statusCode,
				duration,
			)
		})
	}
}

// CORSMiddleware handles CORS headers and preflight requests
func CORSMiddleware(config CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
			} else if len(config.AllowedOrigins) == 1 && config.AllowedOrigins[0] == "*" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			// Set other CORS headers
			if len(config.AllowedMethods) > 0 {
				w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
			}

			if len(config.AllowedHeaders) > 0 {
				w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
			}

			if len(config.ExposedHeaders) > 0 {
				w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
			}

			if config.AllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}

			if config.MaxAge > 0 {
				w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowedHeaders: []string{"Content-Type", "Authorization", "X-Requested-With"},
		ExposedHeaders: []string{"X-Request-ID"},
		MaxAge:         86400, // 24 hours
	}
}

// RateLimitMiddleware implements rate limiting
func RateLimitMiddleware(logger logging.Logger, config RateLimitConfig) func(http.Handler) http.Handler {
	limiters := make(map[string]*rate.Limiter)
	mu := &sync.RWMutex{}

	// Clean up old limiters
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			mu.Lock()
			// In a real implementation, you'd track last access time and clean up old limiters
			mu.Unlock()
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := getRateLimitKey(r, config.KeyFunc)

			mu.Lock()
			limiter, exists := limiters[key]
			if !exists {
				limiter = rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)
				limiters[key] = limiter
			}
			mu.Unlock()

			if !limiter.Allow() {
				logger.WithContext(r.Context()).Warn("Rate limit exceeded", map[string]interface{}{
					"key":    key,
					"path":   r.URL.Path,
					"method": r.Method,
				})

				handler := NewHTTPHandler(logger)
				err := errors.Unavailable("Rate limit exceeded", map[string]interface{}{
					"retry_after": "60s",
				}, nil)

				w.Header().Set("Retry-After", "60")
				handler.WriteError(w, r, err)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond float64
	Burst             int
	KeyFunc           func(*http.Request) string
}

// DefaultRateLimitConfig returns a default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 100, // 100 requests per second
		Burst:             200, // Allow bursts up to 200 requests
		KeyFunc: func(r *http.Request) string {
			return getClientIP(r)
		},
	}
}

// CompressionMiddleware handles response compression
func CompressionMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Check if client accepts gzip
			if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
				next.ServeHTTP(w, r)
				return
			}

			// Create gzip writer
			gz := gzip.NewWriter(w)
			defer gz.Close()

			// Create gzip response writer wrapper
			gzw := &gzipResponseWriter{
				ResponseWriter: w,
				Writer:         gz,
			}

			// Set content encoding header
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length") // Let gzip handle this

			next.ServeHTTP(gzw, r)
		})
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Content-Security-Policy", "default-src 'self'")

			// HSTS header for HTTPS
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RecoveryMiddleware handles panics and returns proper error responses
func RecoveryMiddleware(logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.WithContext(r.Context()).Error("Panic recovered in HTTP handler", map[string]interface{}{
						"error":  fmt.Sprintf("%v", err),
						"path":   r.URL.Path,
						"method": r.Method,
					})

					handler := NewHTTPHandler(logger)
					appErr := errors.Internal("Internal server error", nil, fmt.Errorf("panic: %v", err))
					handler.WriteError(w, r, appErr)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware adds request timeouts
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.TimeoutHandler(next, timeout, `{"error": {"code": "TIMEOUT", "message": "Request timeout"}}`)
	}
}

// Helper types and functions

// responseWriterWrapper wraps http.ResponseWriter to capture status code and bytes written
type responseWriterWrapper struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (w *responseWriterWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterWrapper) Write(data []byte) (int, error) {
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += n
	return n, err
}

// gzipResponseWriter implements http.ResponseWriter for gzip compression
type gzipResponseWriter struct {
	http.ResponseWriter
	io.Writer
}

func (w *gzipResponseWriter) Write(data []byte) (int, error) {
	return w.Writer.Write(data)
}

func (w *gzipResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("hijack not supported")
}

// Helper functions

// generateRequestID generates a unique request ID
func generateRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("req_%d", time.Now().UnixNano())
	}
	return fmt.Sprintf("req_%x", bytes)
}

// getClientIP extracts client IP from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

// isOriginAllowed checks if origin is in allowed list
func isOriginAllowed(origin string, allowed []string) bool {
	for _, a := range allowed {
		if a == "*" || a == origin {
			return true
		}
	}
	return false
}

// getRateLimitKey gets the key for rate limiting
func getRateLimitKey(r *http.Request, keyFunc func(*http.Request) string) string {
	if keyFunc != nil {
		return keyFunc(r)
	}
	return getClientIP(r)
}

// HealthCheckSkipMiddleware creates middleware that skips other middleware for health check endpoints
func HealthCheckSkipMiddleware(healthPaths []string, next func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	pathMap := make(map[string]bool)
	for _, path := range healthPaths {
		pathMap[path] = true
	}

	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip middleware for health check paths
			if pathMap[r.URL.Path] {
				handler.ServeHTTP(w, r)
				return
			}

			// Apply middleware for other paths
			next(handler).ServeHTTP(w, r)
		})
	}
}

// ConditionalMiddleware applies middleware only if condition is met
func ConditionalMiddleware(condition func(*http.Request) bool, middleware func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if condition(r) {
				middleware(next).ServeHTTP(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
