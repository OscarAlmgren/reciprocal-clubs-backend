package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"reciprocal-clubs-backend/pkg/shared/auth"
	"reciprocal-clubs-backend/pkg/shared/logging"
)

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	GlobalLimit    int           // requests per window
	PerUserLimit   int           // requests per user per window
	PerIPLimit     int           // requests per IP per window
	Window         time.Duration // time window
	BurstLimit     int           // burst allowance
	GraphQLLimit   int           // special limit for GraphQL operations
	HealthLimit    int           // higher limit for health checks
	RedisEnabled   bool          // use Redis for distributed rate limiting
	RedisKeyPrefix string        // prefix for Redis keys
}

// DefaultRateLimitConfig returns a sensible default configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		GlobalLimit:    10000,               // 10k requests per minute globally
		PerUserLimit:   1000,                // 1k requests per user per minute
		PerIPLimit:     100,                 // 100 requests per IP per minute
		Window:         time.Minute,         // 1 minute window
		BurstLimit:     50,                  // allow burst of 50 requests
		GraphQLLimit:   50,                  // 50 GraphQL operations per minute
		HealthLimit:    1000,                // high limit for health checks
		RedisEnabled:   false,               // use in-memory by default
		RedisKeyPrefix: "api_gateway:ratelimit:",
	}
}

// AdvancedRateLimitMiddleware implements sophisticated rate limiting
func AdvancedRateLimitMiddleware(config *RateLimitConfig, logger logging.Logger) func(http.Handler) http.Handler {
	var limiter RateLimiter

	if config.RedisEnabled {
		// TODO: Implement Redis-based rate limiter for production
		logger.Info("Redis rate limiting not implemented, falling back to in-memory", nil)
		limiter = NewMemoryRateLimiter(config, logger)
	} else {
		limiter = NewMemoryRateLimiter(config, logger)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Determine appropriate limit for this request
			limit := config.getLimit(r)

			// Get client identifier
			clientID := getAdvancedClientID(r)

			// Check rate limit
			allowed, remaining, resetTime, err := limiter.Allow(ctx, clientID, limit, config.Window)
			if err != nil {
				logger.Error("Rate limit check failed", map[string]interface{}{
					"error":     err.Error(),
					"client_id": clientID,
					"path":      r.URL.Path,
				})
				// Allow request on error to avoid blocking legitimate traffic
			} else if !allowed {
				// Rate limit exceeded
				logger.Warn("Rate limit exceeded", map[string]interface{}{
					"client_id":  clientID,
					"path":       r.URL.Path,
					"method":     r.Method,
					"limit":      limit,
					"reset_time": resetTime,
				})

				// Set rate limit headers
				w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", time.Until(resetTime).Seconds()))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Rate limit exceeded", "retry_after": "` +
					fmt.Sprintf("%.0f", time.Until(resetTime).Seconds()) + `"}`))
				return
			}

			// Set rate limit headers for successful requests
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", limit))
			w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", resetTime.Unix()))

			next.ServeHTTP(w, r)
		})
	}
}

// getLimit determines the appropriate rate limit for a request
func (c *RateLimitConfig) getLimit(r *http.Request) int {
	path := r.URL.Path

	// Health check endpoints get higher limits
	if strings.HasPrefix(path, "/health") || strings.HasPrefix(path, "/ready") {
		return c.HealthLimit
	}

	// GraphQL endpoints get special limits
	if strings.HasPrefix(path, "/graphql") {
		return c.GraphQLLimit
	}

	// Check if user is authenticated for per-user limits
	if user := auth.GetUserFromContext(r.Context()); user != nil {
		return c.PerUserLimit
	}

	// Default to per-IP limit
	return c.PerIPLimit
}

func getAdvancedClientID(r *http.Request) string {
	// Try to get user ID from context first (more specific)
	if user := auth.GetUserFromContext(r.Context()); user != nil {
		return fmt.Sprintf("user:%d", user.ID)
	}

	// Fall back to IP address with proper extraction
	clientIP := getClientIP(r)
	return fmt.Sprintf("ip:%s", clientIP)
}

// RateLimiter interface for different implementations
type RateLimiter interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, resetTime time.Time, err error)
}

// MemoryRateLimiter implements in-memory rate limiting
type MemoryRateLimiter struct {
	config   *RateLimitConfig
	limiters map[string]*tokenBucket
	logger   logging.Logger
}

func NewMemoryRateLimiter(config *RateLimitConfig, logger logging.Logger) *MemoryRateLimiter {
	return &MemoryRateLimiter{
		config:   config,
		limiters: make(map[string]*tokenBucket),
		logger:   logger,
	}
}

func (m *MemoryRateLimiter) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, int, time.Time, error) {
	bucket := m.getBucket(key, limit, window)
	return bucket.allow()
}

func (m *MemoryRateLimiter) getBucket(key string, limit int, window time.Duration) *tokenBucket {
	// In a real implementation, you'd want proper cleanup of old buckets
	// and thread-safe access. This is a simplified version.

	if bucket, exists := m.limiters[key]; exists {
		return bucket
	}

	bucket := newTokenBucket(limit, window)
	m.limiters[key] = bucket
	return bucket
}

// tokenBucket implements a token bucket algorithm
type tokenBucket struct {
	capacity     int
	tokens       int
	refillRate   time.Duration
	lastRefill   time.Time
	window       time.Duration
}

func newTokenBucket(capacity int, window time.Duration) *tokenBucket {
	return &tokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: window / time.Duration(capacity),
		lastRefill: time.Now(),
		window:     window,
	}
}

func (tb *tokenBucket) allow() (bool, int, time.Time, error) {
	now := time.Now()

	// Calculate tokens to add based on time passed
	timePassed := now.Sub(tb.lastRefill)
	tokensToAdd := int(timePassed / tb.refillRate)

	if tokensToAdd > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+tokensToAdd)
		tb.lastRefill = now
	}

	// Check if we can allow this request
	if tb.tokens > 0 {
		tb.tokens--
		resetTime := now.Add(tb.window)
		return true, tb.tokens, resetTime, nil
	}

	// Calculate when tokens will be available
	resetTime := tb.lastRefill.Add(tb.refillRate)
	return false, 0, resetTime, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GraphQLOperationRateLimitMiddleware provides operation-specific rate limiting for GraphQL
type GraphQLOperationRateLimitMiddleware struct {
	config *RateLimitConfig
	logger logging.Logger
	limits map[string]int // operation name -> limit
}

func NewGraphQLOperationRateLimitMiddleware(config *RateLimitConfig, logger logging.Logger) *GraphQLOperationRateLimitMiddleware {
	// Define operation-specific limits
	limits := map[string]int{
		"login":                   10,  // 10 login attempts per minute
		"register":               5,   // 5 registration attempts per minute
		"createMember":           20,  // 20 member creations per minute
		"recordVisit":            100, // 100 visit records per minute
		"generateAnalyticsReport": 5,   // 5 report generations per minute
	}

	return &GraphQLOperationRateLimitMiddleware{
		config: config,
		logger: logger,
		limits: limits,
	}
}

func (m *GraphQLOperationRateLimitMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only apply to GraphQL operations
			if !strings.HasPrefix(r.URL.Path, "/graphql") {
				next.ServeHTTP(w, r)
				return
			}

			// TODO: Parse GraphQL operation name and apply specific limits
			// For now, just pass through - this would require GraphQL query parsing

			next.ServeHTTP(w, r)
		})
	}
}