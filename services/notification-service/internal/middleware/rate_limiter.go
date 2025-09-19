package middleware

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
	"reciprocal-clubs-backend/pkg/shared/logging"
)

// RateLimiter provides rate limiting functionality
type RateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	logger   logging.Logger
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(rps float64, burst int, logger logging.Logger) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    burst,
		logger:   logger,
	}
}

// GetLimiter returns a rate limiter for the given key
func (rl *RateLimiter) GetLimiter(key string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[key]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[key] = limiter
	}

	return limiter
}

// RateLimitMiddleware creates rate limiting middleware
func (rl *RateLimiter) RateLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get client IP
			ip := getClientIP(r)
			key := fmt.Sprintf("ip:%s", ip)

			limiter := rl.GetLimiter(key)

			if !limiter.Allow() {
				rl.logger.Warn("Rate limit exceeded", map[string]interface{}{
					"ip":     ip,
					"path":   r.URL.Path,
					"method": r.Method,
				})

				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UserRateLimitMiddleware creates user-specific rate limiting middleware
func (rl *RateLimiter) UserRateLimitMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context or headers
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				// Fall back to IP-based limiting
				userID = getClientIP(r)
			}

			key := fmt.Sprintf("user:%s", userID)
			limiter := rl.GetLimiter(key)

			if !limiter.Allow() {
				rl.logger.Warn("User rate limit exceeded", map[string]interface{}{
					"user_id": userID,
					"path":    r.URL.Path,
					"method":  r.Method,
				})

				w.Header().Set("X-RateLimit-Limit", strconv.Itoa(rl.burst))
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("Retry-After", "60")
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// CleanupExpiredLimiters removes inactive limiters to prevent memory leaks
func (rl *RateLimiter) CleanupExpiredLimiters() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			rl.mu.Lock()
			// Remove limiters that haven't been used recently
			// This is a simple cleanup - in production you might want more sophisticated logic
			for key, limiter := range rl.limiters {
				// If the limiter has full burst capacity, it hasn't been used recently
				if limiter.Tokens() == float64(rl.burst) {
					delete(rl.limiters, key)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// getClientIP extracts the client IP from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		if idx := len(xff); idx > 0 {
			if ip := net.ParseIP(xff); ip != nil {
				return ip.String()
			}
		}
	}

	// Check X-Real-IP header
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		if ip := net.ParseIP(xri); ip != nil {
			return ip.String()
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return ip
}

// NotificationRateLimiter provides notification-specific rate limiting
type NotificationRateLimiter struct {
	limiters map[string]*rate.Limiter
	mu       sync.RWMutex
	config   *NotificationRateLimitConfig
	logger   logging.Logger
}

type NotificationRateLimitConfig struct {
	EmailRPS    float64
	EmailBurst  int
	SMSRPS      float64
	SMSBurst    int
	PushRPS     float64
	PushBurst   int
	WebhookRPS  float64
	WebhookBurst int
}

// NewNotificationRateLimiter creates a notification-specific rate limiter
func NewNotificationRateLimiter(config *NotificationRateLimitConfig, logger logging.Logger) *NotificationRateLimiter {
	return &NotificationRateLimiter{
		limiters: make(map[string]*rate.Limiter),
		config:   config,
		logger:   logger,
	}
}

// CheckNotificationLimit checks if a notification can be sent
func (nrl *NotificationRateLimiter) CheckNotificationLimit(clubID string, notificationType string) bool {
	key := fmt.Sprintf("%s:%s", clubID, notificationType)

	nrl.mu.Lock()
	limiter, exists := nrl.limiters[key]
	if !exists {
		var rps float64
		var burst int

		switch notificationType {
		case "email":
			rps = nrl.config.EmailRPS
			burst = nrl.config.EmailBurst
		case "sms":
			rps = nrl.config.SMSRPS
			burst = nrl.config.SMSBurst
		case "push":
			rps = nrl.config.PushRPS
			burst = nrl.config.PushBurst
		case "webhook":
			rps = nrl.config.WebhookRPS
			burst = nrl.config.WebhookBurst
		default:
			rps = 10 // Default rate
			burst = 20
		}

		limiter = rate.NewLimiter(rate.Limit(rps), burst)
		nrl.limiters[key] = limiter
	}
	nrl.mu.Unlock()

	allowed := limiter.Allow()
	if !allowed {
		nrl.logger.Warn("Notification rate limit exceeded", map[string]interface{}{
			"club_id": clubID,
			"type":    notificationType,
		})
	}

	return allowed
}

// GetDefaultNotificationRateLimitConfig returns default configuration
func GetDefaultNotificationRateLimitConfig() *NotificationRateLimitConfig {
	return &NotificationRateLimitConfig{
		EmailRPS:     10,   // 10 emails per second
		EmailBurst:   50,   // Burst of 50 emails
		SMSRPS:       5,    // 5 SMS per second
		SMSBurst:     20,   // Burst of 20 SMS
		PushRPS:      50,   // 50 push notifications per second
		PushBurst:    200,  // Burst of 200 push notifications
		WebhookRPS:   20,   // 20 webhooks per second
		WebhookBurst: 100,  // Burst of 100 webhooks
	}
}