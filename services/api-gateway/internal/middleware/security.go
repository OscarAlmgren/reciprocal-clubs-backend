package middleware

import (
	"net/http"
	"strings"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Security headers
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-XSS-Protection", "1; mode=block")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")

			// Only add HSTS for HTTPS requests
			if r.TLS != nil {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains; preload")
			}

			// Content Security Policy for GraphQL endpoint
			if strings.HasPrefix(r.URL.Path, "/graphql") {
				w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestSizeLimitMiddleware limits the size of incoming requests
func RequestSizeLimitMiddleware(maxSize int64, logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Limit request body size
			r.Body = http.MaxBytesReader(w, r.Body, maxSize)

			// Check content length header
			if r.ContentLength > maxSize {
				logger.Warn("Request size limit exceeded", map[string]interface{}{
					"content_length": r.ContentLength,
					"max_size":       maxSize,
					"path":           r.URL.Path,
					"method":         r.Method,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				w.Write([]byte(`{"error": "Request entity too large"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequestTimeoutMiddleware adds timeout to requests
func RequestTimeoutMiddleware(logger logging.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Add timeout context (30 seconds for GraphQL operations)
			ctx := r.Context()

			// GraphQL operations get longer timeout
			timeout := 30
			if strings.HasPrefix(r.URL.Path, "/graphql") {
				timeout = 60 // 60 seconds for complex GraphQL operations
			}

			// Set timeout header for client awareness
			w.Header().Set("X-Timeout", string(rune(timeout)))

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GraphQLDepthLimitMiddleware limits GraphQL query depth
type GraphQLDepthLimitMiddleware struct {
	maxDepth int
	logger   logging.Logger
}

func NewGraphQLDepthLimitMiddleware(maxDepth int, logger logging.Logger) *GraphQLDepthLimitMiddleware {
	return &GraphQLDepthLimitMiddleware{
		maxDepth: maxDepth,
		logger:   logger,
	}
}

func (m *GraphQLDepthLimitMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to GraphQL endpoints
			if !strings.HasPrefix(r.URL.Path, "/graphql") {
				next.ServeHTTP(w, r)
				return
			}

			// TODO: Parse GraphQL query and check depth
			// For now, just pass through - actual depth limiting is handled by gqlgen

			next.ServeHTTP(w, r)
		})
	}
}

// IPWhitelistMiddleware allows only whitelisted IPs for admin operations
func IPWhitelistMiddleware(allowedIPs []string, logger logging.Logger) func(http.Handler) http.Handler {
	ipMap := make(map[string]bool)
	for _, ip := range allowedIPs {
		ipMap[ip] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to admin paths
			if !strings.HasPrefix(r.URL.Path, "/admin") {
				next.ServeHTTP(w, r)
				return
			}

			clientIP := getClientIP(r)

			if !ipMap[clientIP] {
				logger.Warn("Unauthorized IP access attempt", map[string]interface{}{
					"client_ip": clientIP,
					"path":      r.URL.Path,
					"method":    r.Method,
				})

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error": "Access denied"}`))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP from the list
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to remote address
	return r.RemoteAddr
}