package middleware

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/auth-service/internal/metrics"

	"golang.org/x/time/rate"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

// RateLimitConfig defines rate limiting configuration
type RateLimitConfig struct {
	// Global rate limiting
	GlobalRequestsPerSecond int           `json:\"global_requests_per_second\"`
	GlobalBurstSize         int           `json:\"global_burst_size\"`

	// Per-IP rate limiting
	IPRequestsPerSecond     int           `json:\"ip_requests_per_second\"`
	IPBurstSize             int           `json:\"ip_burst_size\"`
	IPCleanupInterval       time.Duration `json:\"ip_cleanup_interval\"`

	// Per-user rate limiting
	UserRequestsPerSecond   int           `json:\"user_requests_per_second\"`
	UserBurstSize           int           `json:\"user_burst_size\"`
	UserCleanupInterval     time.Duration `json:\"user_cleanup_interval\"`

	// Authentication specific limits
	AuthAttemptsPerMinute   int           `json:\"auth_attempts_per_minute\"`
	AuthBurstSize          int           `json:\"auth_burst_size\"`
	AuthWindowDuration     time.Duration `json:\"auth_window_duration\"`

	// Passkey specific limits
	PasskeyOpsPerMinute     int           `json:\"passkey_ops_per_minute\"`
	PasskeyBurstSize       int           `json:\"passkey_burst_size\"`

	// Registration limits
	RegistrationsPerHour    int           `json:\"registrations_per_hour\"`
	RegistrationBurstSize  int           `json:\"registration_burst_size\"`
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		GlobalRequestsPerSecond: 1000,
		GlobalBurstSize:         100,
		IPRequestsPerSecond:     10,
		IPBurstSize:            20,
		IPCleanupInterval:      5 * time.Minute,
		UserRequestsPerSecond:  100,
		UserBurstSize:         10,
		UserCleanupInterval:   10 * time.Minute,
		AuthAttemptsPerMinute: 5,
		AuthBurstSize:        2,
		AuthWindowDuration:   time.Minute,
		PasskeyOpsPerMinute:  20,
		PasskeyBurstSize:     5,
		RegistrationsPerHour: 10,
		RegistrationBurstSize: 2,
	}
}

// RateLimitMiddleware provides comprehensive rate limiting
type RateLimitMiddleware struct {
	config      *RateLimitConfig
	authMetrics *metrics.AuthMetrics
	logger      logging.Logger

	// Global rate limiter
	globalLimiter *rate.Limiter

	// Per-IP rate limiters
	ipLimiters    map[string]*rate.Limiter
	ipMutex       sync.RWMutex
	ipLastAccess  map[string]time.Time

	// Per-user rate limiters
	userLimiters    map[string]*rate.Limiter
	userMutex       sync.RWMutex
	userLastAccess  map[string]time.Time

	// Authentication specific limiters
	authLimiters   map[string]*rate.Limiter
	authMutex      sync.RWMutex
	authLastAccess map[string]time.Time

	// Passkey specific limiters
	passkeyLimiters   map[string]*rate.Limiter
	passkeyMutex      sync.RWMutex
	passkeyLastAccess map[string]time.Time

	// Registration limiters
	registrationLimiters   map[string]*rate.Limiter
	registrationMutex      sync.RWMutex
	registrationLastAccess map[string]time.Time

	// Cleanup ticker
	cleanupTicker *time.Ticker
	stopCleanup   chan bool
}

// NewRateLimitMiddleware creates a new rate limiting middleware
func NewRateLimitMiddleware(config *RateLimitConfig, authMetrics *metrics.AuthMetrics, logger logging.Logger) *RateLimitMiddleware {
	if config == nil {
		config = DefaultRateLimitConfig()
	}

	rl := &RateLimitMiddleware{
		config:      config,
		authMetrics: authMetrics,
		logger:      logger,

		globalLimiter: rate.NewLimiter(rate.Limit(config.GlobalRequestsPerSecond), config.GlobalBurstSize),

		ipLimiters:    make(map[string]*rate.Limiter),
		ipLastAccess:  make(map[string]time.Time),

		userLimiters:    make(map[string]*rate.Limiter),
		userLastAccess:  make(map[string]time.Time),

		authLimiters:   make(map[string]*rate.Limiter),
		authLastAccess: make(map[string]time.Time),

		passkeyLimiters:   make(map[string]*rate.Limiter),
		passkeyLastAccess: make(map[string]time.Time),

		registrationLimiters:   make(map[string]*rate.Limiter),
		registrationLastAccess: make(map[string]time.Time),

		cleanupTicker: time.NewTicker(time.Minute),
		stopCleanup:   make(chan bool),
	}

	// Start cleanup goroutine
	go rl.cleanupRoutine()

	return rl
}

// UnaryServerInterceptor returns a gRPC unary server interceptor for rate limiting
func (rl *RateLimitMiddleware) UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Check global rate limit
		if !rl.globalLimiter.Allow() {
			rl.authMetrics.RecordSecurityEvent(\"global\", \"rate_limit_exceeded\", \"medium\", \"global\", \"system\")
			rl.logger.Warn(\"Global rate limit exceeded\", map[string]interface{}{
				\"method\": info.FullMethod,
			})
			return nil, status.Error(codes.ResourceExhausted, \"Global rate limit exceeded\")
		}

		// Extract IP address
		clientIP := rl.extractClientIP(ctx)
		if clientIP != \"\" {
			if !rl.checkIPRateLimit(clientIP) {
				rl.authMetrics.RecordSecurityEvent(\"ip\", \"rate_limit_exceeded\", \"high\", clientIP, \"unknown\")
				rl.logger.Warn(\"IP rate limit exceeded\", map[string]interface{}{
					\"method\":    info.FullMethod,
					\"client_ip\": clientIP,
				})
				return nil, status.Error(codes.ResourceExhausted, \"IP rate limit exceeded\")
			}
		}

		// Extract user ID if available
		userID := rl.extractUserID(ctx)
		if userID != \"\" {
			if !rl.checkUserRateLimit(userID) {
				rl.authMetrics.RecordSecurityEvent(\"user\", \"rate_limit_exceeded\", \"medium\", clientIP, userID)
				rl.logger.Warn(\"User rate limit exceeded\", map[string]interface{}{
					\"method\":    info.FullMethod,
					\"user_id\":   userID,
					\"client_ip\": clientIP,
				})
				return nil, status.Error(codes.ResourceExhausted, \"User rate limit exceeded\")
			}
		}

		// Check method-specific rate limits
		if err := rl.checkMethodSpecificLimits(info.FullMethod, ctx, clientIP, userID); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

// checkIPRateLimit checks IP-based rate limiting
func (rl *RateLimitMiddleware) checkIPRateLimit(clientIP string) bool {
	rl.ipMutex.Lock()
	defer rl.ipMutex.Unlock()

	limiter, exists := rl.ipLimiters[clientIP]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rl.config.IPRequestsPerSecond), rl.config.IPBurstSize)
		rl.ipLimiters[clientIP] = limiter
	}

	rl.ipLastAccess[clientIP] = time.Now()
	return limiter.Allow()
}

// checkUserRateLimit checks user-based rate limiting
func (rl *RateLimitMiddleware) checkUserRateLimit(userID string) bool {
	rl.userMutex.Lock()
	defer rl.userMutex.Unlock()

	limiter, exists := rl.userLimiters[userID]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rl.config.UserRequestsPerSecond), rl.config.UserBurstSize)
		rl.userLimiters[userID] = limiter
	}

	rl.userLastAccess[userID] = time.Now()
	return limiter.Allow()
}

// checkMethodSpecificLimits checks method-specific rate limits
func (rl *RateLimitMiddleware) checkMethodSpecificLimits(method string, ctx context.Context, clientIP, userID string) error {
	switch method {
	case \"/auth.AuthService/RegisterUser\":
		return rl.checkRegistrationLimit(clientIP)

	case \"/auth.AuthService/InitiatePasskeyLogin\", \"/auth.AuthService/CompletePasskeyLogin\":
		return rl.checkAuthLimit(clientIP, userID)

	case \"/auth.AuthService/InitiatePasskeyRegistration\", \"/auth.AuthService/CompletePasskeyRegistration\":
		return rl.checkPasskeyLimit(clientIP, userID)
	}

	return nil
}

// checkAuthLimit checks authentication-specific rate limiting
func (rl *RateLimitMiddleware) checkAuthLimit(clientIP, userID string) error {
	key := fmt.Sprintf(\"%s:%s\", clientIP, userID)
	if userID == \"\" {
		key = clientIP
	}

	rl.authMutex.Lock()
	defer rl.authMutex.Unlock()

	limiter, exists := rl.authLimiters[key]
	if !exists {
		// Create a more restrictive limiter for auth operations
		authRate := rate.Every(rl.config.AuthWindowDuration / time.Duration(rl.config.AuthAttemptsPerMinute))
		limiter = rate.NewLimiter(authRate, rl.config.AuthBurstSize)
		rl.authLimiters[key] = limiter
	}

	rl.authLastAccess[key] = time.Now()

	if !limiter.Allow() {
		rl.authMetrics.RecordSecurityEvent(\"auth\", \"rate_limit_exceeded\", \"high\", clientIP, userID)
		rl.logger.Warn(\"Authentication rate limit exceeded\", map[string]interface{}{
			\"client_ip\": clientIP,
			\"user_id\":   userID,
			\"key\":       key,
		})
		return status.Error(codes.ResourceExhausted, \"Authentication rate limit exceeded\")
	}

	return nil
}

// checkPasskeyLimit checks passkey-specific rate limiting
func (rl *RateLimitMiddleware) checkPasskeyLimit(clientIP, userID string) error {
	key := fmt.Sprintf(\"%s:%s\", clientIP, userID)
	if userID == \"\" {
		key = clientIP
	}

	rl.passkeyMutex.Lock()
	defer rl.passkeyMutex.Unlock()

	limiter, exists := rl.passkeyLimiters[key]
	if !exists {
		passkeyRate := rate.Every(time.Minute / time.Duration(rl.config.PasskeyOpsPerMinute))
		limiter = rate.NewLimiter(passkeyRate, rl.config.PasskeyBurstSize)
		rl.passkeyLimiters[key] = limiter
	}

	rl.passkeyLastAccess[key] = time.Now()

	if !limiter.Allow() {
		rl.authMetrics.RecordSecurityEvent(\"passkey\", \"rate_limit_exceeded\", \"medium\", clientIP, userID)
		rl.logger.Warn(\"Passkey operation rate limit exceeded\", map[string]interface{}{
			\"client_ip\": clientIP,
			\"user_id\":   userID,
		})
		return status.Error(codes.ResourceExhausted, \"Passkey operation rate limit exceeded\")
	}

	return nil
}

// checkRegistrationLimit checks registration-specific rate limiting
func (rl *RateLimitMiddleware) checkRegistrationLimit(clientIP string) error {
	rl.registrationMutex.Lock()
	defer rl.registrationMutex.Unlock()

	limiter, exists := rl.registrationLimiters[clientIP]
	if !exists {
		registrationRate := rate.Every(time.Hour / time.Duration(rl.config.RegistrationsPerHour))
		limiter = rate.NewLimiter(registrationRate, rl.config.RegistrationBurstSize)
		rl.registrationLimiters[clientIP] = limiter
	}

	rl.registrationLastAccess[clientIP] = time.Now()

	if !limiter.Allow() {
		rl.authMetrics.RecordSecurityEvent(\"registration\", \"rate_limit_exceeded\", \"high\", clientIP, \"unknown\")
		rl.logger.Warn(\"Registration rate limit exceeded\", map[string]interface{}{
			\"client_ip\": clientIP,
		})
		return status.Error(codes.ResourceExhausted, \"Registration rate limit exceeded\")
	}

	return nil
}

// extractClientIP extracts the client IP address from the gRPC context
func (rl *RateLimitMiddleware) extractClientIP(ctx context.Context) string {
	// Try to get IP from peer info
	if peer, ok := peer.FromContext(ctx); ok {
		return peer.Addr.String()
	}

	// Try to get IP from metadata (for proxy scenarios)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if xForwardedFor := md.Get(\"x-forwarded-for\"); len(xForwardedFor) > 0 {
			return xForwardedFor[0]
		}
		if xRealIP := md.Get(\"x-real-ip\"); len(xRealIP) > 0 {
			return xRealIP[0]
		}
	}

	return \"\"
}

// extractUserID extracts the user ID from the gRPC context
func (rl *RateLimitMiddleware) extractUserID(ctx context.Context) string {
	// Try to get user ID from context values
	if userIDValue := ctx.Value(\"user_id\"); userIDValue != nil {
		if userID, ok := userIDValue.(string); ok {
			return userID
		}
		if userID, ok := userIDValue.(uint); ok {
			return strconv.FormatUint(uint64(userID), 10)
		}
	}

	// Try to get user ID from metadata
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if userIDs := md.Get(\"user-id\"); len(userIDs) > 0 {
			return userIDs[0]
		}
	}

	return \"\"
}

// cleanupRoutine periodically cleans up old rate limiters
func (rl *RateLimitMiddleware) cleanupRoutine() {
	for {
		select {
		case <-rl.cleanupTicker.C:
			rl.cleanupOldLimiters()
		case <-rl.stopCleanup:
			return
		}
	}
}

// cleanupOldLimiters removes old rate limiters to prevent memory leaks
func (rl *RateLimitMiddleware) cleanupOldLimiters() {
	now := time.Now()

	// Cleanup IP limiters
	rl.ipMutex.Lock()
	for ip, lastAccess := range rl.ipLastAccess {
		if now.Sub(lastAccess) > rl.config.IPCleanupInterval {
			delete(rl.ipLimiters, ip)
			delete(rl.ipLastAccess, ip)
		}
	}
	rl.ipMutex.Unlock()

	// Cleanup user limiters
	rl.userMutex.Lock()
	for userID, lastAccess := range rl.userLastAccess {
		if now.Sub(lastAccess) > rl.config.UserCleanupInterval {
			delete(rl.userLimiters, userID)
			delete(rl.userLastAccess, userID)
		}
	}
	rl.userMutex.Unlock()

	// Cleanup auth limiters
	rl.authMutex.Lock()
	for key, lastAccess := range rl.authLastAccess {
		if now.Sub(lastAccess) > rl.config.AuthWindowDuration*2 {
			delete(rl.authLimiters, key)
			delete(rl.authLastAccess, key)
		}
	}
	rl.authMutex.Unlock()

	// Cleanup passkey limiters
	rl.passkeyMutex.Lock()
	for key, lastAccess := range rl.passkeyLastAccess {
		if now.Sub(lastAccess) > time.Hour {
			delete(rl.passkeyLimiters, key)
			delete(rl.passkeyLastAccess, key)
		}
	}
	rl.passkeyMutex.Unlock()

	// Cleanup registration limiters
	rl.registrationMutex.Lock()
	for ip, lastAccess := range rl.registrationLastAccess {
		if now.Sub(lastAccess) > time.Hour*2 {
			delete(rl.registrationLimiters, ip)
			delete(rl.registrationLastAccess, ip)
		}
	}
	rl.registrationMutex.Unlock()

	rl.logger.Debug(\"Rate limiter cleanup completed\", map[string]interface{}{
		\"ip_limiters\":           len(rl.ipLimiters),
		\"user_limiters\":         len(rl.userLimiters),
		\"auth_limiters\":         len(rl.authLimiters),
		\"passkey_limiters\":      len(rl.passkeyLimiters),
		\"registration_limiters\": len(rl.registrationLimiters),
	})
}

// GetStats returns current rate limiting statistics
func (rl *RateLimitMiddleware) GetStats() map[string]interface{} {
	rl.ipMutex.RLock()
	ipCount := len(rl.ipLimiters)
	rl.ipMutex.RUnlock()

	rl.userMutex.RLock()
	userCount := len(rl.userLimiters)
	rl.userMutex.RUnlock()

	rl.authMutex.RLock()
	authCount := len(rl.authLimiters)
	rl.authMutex.RUnlock()

	rl.passkeyMutex.RLock()
	passkeyCount := len(rl.passkeyLimiters)
	rl.passkeyMutex.RUnlock()

	rl.registrationMutex.RLock()
	registrationCount := len(rl.registrationLimiters)
	rl.registrationMutex.RUnlock()

	return map[string]interface{}{
		\"active_ip_limiters\":           ipCount,
		\"active_user_limiters\":         userCount,
		\"active_auth_limiters\":         authCount,
		\"active_passkey_limiters\":      passkeyCount,
		\"active_registration_limiters\": registrationCount,
		\"global_limit_rps\":             rl.config.GlobalRequestsPerSecond,
		\"ip_limit_rps\":                 rl.config.IPRequestsPerSecond,
		\"user_limit_rps\":               rl.config.UserRequestsPerSecond,
	}
}

// Close gracefully shuts down the rate limiting middleware
func (rl *RateLimitMiddleware) Close() {
	rl.cleanupTicker.Stop()
	close(rl.stopCleanup)
	rl.logger.Info(\"Rate limiting middleware shut down\", nil)
}