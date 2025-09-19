package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
	"reciprocal-clubs-backend/services/auth-service/internal/metrics"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return \"closed\"
	case StateOpen:
		return \"open\"
	case StateHalfOpen:
		return \"half_open\"
	default:
		return \"unknown\"
	}
}

// CircuitBreakerConfig defines circuit breaker configuration
type CircuitBreakerConfig struct {
	Name                   string        `json:\"name\"`
	FailureThreshold       int           `json:\"failure_threshold\"`
	SuccessThreshold       int           `json:\"success_threshold\"`
	Timeout                time.Duration `json:\"timeout\"`
	MaxConcurrentRequests  int           `json:\"max_concurrent_requests\"`
	RequestVolumeThreshold int           `json:\"request_volume_threshold\"`
	SleepWindow            time.Duration `json:\"sleep_window\"`
	ErrorPercentThreshold  int           `json:\"error_percent_threshold\"`
}

// DefaultCircuitBreakerConfigs returns default circuit breaker configurations for different services
func DefaultCircuitBreakerConfigs() map[string]*CircuitBreakerConfig {
	return map[string]*CircuitBreakerConfig{
		\"hanko\": {
			Name:                   \"hanko\",
			FailureThreshold:       5,
			SuccessThreshold:       3,
			Timeout:                30 * time.Second,
			MaxConcurrentRequests:  100,
			RequestVolumeThreshold: 10,
			SleepWindow:            60 * time.Second,
			ErrorPercentThreshold:  50,
		},
		\"database\": {
			Name:                   \"database\",
			FailureThreshold:       3,
			SuccessThreshold:       2,
			Timeout:                10 * time.Second,
			MaxConcurrentRequests:  200,
			RequestVolumeThreshold: 20,
			SleepWindow:            30 * time.Second,
			ErrorPercentThreshold:  30,
		},
		\"messagebus\": {
			Name:                   \"messagebus\",
			FailureThreshold:       10,
			SuccessThreshold:       5,
			Timeout:                5 * time.Second,
			MaxConcurrentRequests:  50,
			RequestVolumeThreshold: 15,
			SleepWindow:            45 * time.Second,
			ErrorPercentThreshold:  60,
		},
	}
}

// CircuitBreakerMetrics holds metrics for a circuit breaker
type CircuitBreakerMetrics struct {
	Requests         int64
	Successes        int64
	Failures         int64
	Timeouts         int64
	CircuitOpens     int64
	CircuitCloses    int64
	FallbackSuccess  int64
	FallbackFailures int64
	LastFailureTime  time.Time
	LastSuccessTime  time.Time
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name                   string
	config                 *CircuitBreakerConfig
	state                  CircuitBreakerState
	failureCount          int
	successCount          int
	requestCount          int64
	concurrentRequests    int64
	lastFailureTime       time.Time
	lastSuccessTime       time.Time
	lastStateChangeTime   time.Time
	metrics               *CircuitBreakerMetrics
	authMetrics           *metrics.AuthMetrics
	logger                logging.Logger
	mu                    sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig, authMetrics *metrics.AuthMetrics, logger logging.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		name:                config.Name,
		config:              config,
		state:               StateClosed,
		lastStateChangeTime: time.Now(),
		metrics:             &CircuitBreakerMetrics{},
		authMetrics:         authMetrics,
		logger:              logger,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, operation func(context.Context) (interface{}, error)) (interface{}, error) {
	// Check if we can execute the request
	if err := cb.allowRequest(); err != nil {
		return nil, err
	}

	// Increment concurrent requests
	cb.incrementConcurrentRequests()
	defer cb.decrementConcurrentRequests()

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, cb.config.Timeout)
	defer cancel()

	// Execute the operation
	resultChan := make(chan struct {
		result interface{}
		err    error
	}, 1)

	go func() {
		result, err := operation(ctx)
		resultChan <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	select {
	case res := <-resultChan:
		// Operation completed
		if res.err != nil {
			cb.recordFailure(res.err)
			return nil, res.err
		}
		cb.recordSuccess()
		return res.result, nil

	case <-ctx.Done():
		// Operation timed out
		cb.recordTimeout()
		return nil, status.Error(codes.DeadlineExceeded, fmt.Sprintf(\"Circuit breaker timeout for %s\", cb.name))
	}
}

// ExecuteWithFallback executes a function with circuit breaker protection and fallback
func (cb *CircuitBreaker) ExecuteWithFallback(ctx context.Context, operation func(context.Context) (interface{}, error), fallback func(context.Context, error) (interface{}, error)) (interface{}, error) {
	result, err := cb.Execute(ctx, operation)
	if err != nil && fallback != nil {
		// Record that we're using fallback
		cb.mu.Lock()
		cb.metrics.FallbackSuccess++
		cb.mu.Unlock()

		cb.logger.Warn(\"Circuit breaker executing fallback\", map[string]interface{}{
			\"circuit_breaker\": cb.name,
			\"error\":           err.Error(),
			\"state\":           cb.state.String(),
		})

		fallbackResult, fallbackErr := fallback(ctx, err)
		if fallbackErr != nil {
			cb.mu.Lock()
			cb.metrics.FallbackFailures++
			cb.mu.Unlock()
		}
		return fallbackResult, fallbackErr
	}
	return result, err
}

// allowRequest checks if a request can be executed based on circuit breaker state
func (cb *CircuitBreaker) allowRequest() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		// Always allow requests when circuit is closed
		return nil

	case StateOpen:
		// Check if sleep window has passed
		if time.Since(cb.lastStateChangeTime) >= cb.config.SleepWindow {
			cb.setState(StateHalfOpen)
			cb.logger.Info(\"Circuit breaker transitioning to half-open\", map[string]interface{}{
				\"circuit_breaker\": cb.name,
				\"sleep_window\":    cb.config.SleepWindow,
			})
			return nil
		}
		return status.Error(codes.Unavailable, fmt.Sprintf(\"Circuit breaker %s is open\", cb.name))

	case StateHalfOpen:
		// Allow limited requests in half-open state
		if cb.concurrentRequests >= int64(cb.config.MaxConcurrentRequests/10) { // Allow 10% of normal traffic
			return status.Error(codes.Unavailable, fmt.Sprintf(\"Circuit breaker %s is half-open with limited capacity\", cb.name))
		}
		return nil

	default:
		return status.Error(codes.Internal, \"Unknown circuit breaker state\")
	}
}

// recordSuccess records a successful operation
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.Requests++
	cb.metrics.Successes++
	cb.successCount++
	cb.failureCount = 0 // Reset failure count on success
	cb.lastSuccessTime = time.Now()

	// Record external service metrics
	cb.authMetrics.RecordHankoOperation(\"success\", \"success\", \"\")

	// Check if we should close the circuit in half-open state
	if cb.state == StateHalfOpen && cb.successCount >= cb.config.SuccessThreshold {
		cb.setState(StateClosed)
		cb.logger.Info(\"Circuit breaker closed after successful operations\", map[string]interface{}{
			\"circuit_breaker\":   cb.name,
			\"success_threshold\": cb.config.SuccessThreshold,
		})
	}
}

// recordFailure records a failed operation
func (cb *CircuitBreaker) recordFailure(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.Requests++
	cb.metrics.Failures++
	cb.failureCount++
	cb.successCount = 0 // Reset success count on failure
	cb.lastFailureTime = time.Now()

	// Record external service metrics
	errorType := cb.categorizeError(err)
	cb.authMetrics.RecordHankoOperation(\"failure\", \"error\", errorType)

	// Check if we should open the circuit
	cb.checkAndUpdateState()
}

// recordTimeout records a timeout
func (cb *CircuitBreaker) recordTimeout() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.metrics.Requests++
	cb.metrics.Timeouts++
	cb.failureCount++
	cb.successCount = 0
	cb.lastFailureTime = time.Now()

	// Record external service metrics
	cb.authMetrics.RecordHankoOperation(\"timeout\", \"error\", \"timeout\")

	// Check if we should open the circuit
	cb.checkAndUpdateState()
}

// checkAndUpdateState checks if the circuit breaker state should change
func (cb *CircuitBreaker) checkAndUpdateState() {
	// Only check state changes if we have enough requests
	if cb.metrics.Requests < int64(cb.config.RequestVolumeThreshold) {
		return
	}

	// Calculate error percentage
	errorRate := float64(cb.metrics.Failures+cb.metrics.Timeouts) / float64(cb.metrics.Requests) * 100

	// Open circuit if failure threshold or error percentage is exceeded
	if (cb.failureCount >= cb.config.FailureThreshold || errorRate >= float64(cb.config.ErrorPercentThreshold)) && cb.state != StateOpen {
		cb.setState(StateOpen)
		cb.logger.Error(\"Circuit breaker opened due to failures\", map[string]interface{}{
			\"circuit_breaker\":       cb.name,
			\"failure_count\":         cb.failureCount,
			\"failure_threshold\":     cb.config.FailureThreshold,
			\"error_rate\":            errorRate,
			\"error_threshold\":       cb.config.ErrorPercentThreshold,
			\"total_requests\":        cb.metrics.Requests,
		})

		// Record security event for circuit opening
		cb.authMetrics.RecordSecurityEvent(
			\"external_service\",
			\"circuit_breaker_opened\",
			\"high\",
			cb.name,
			\"system\",
		)
	}
}

// setState changes the circuit breaker state
func (cb *CircuitBreaker) setState(state CircuitBreakerState) {
	oldState := cb.state
	cb.state = state
	cb.lastStateChangeTime = time.Now()

	if state == StateOpen {
		cb.metrics.CircuitOpens++
	} else if state == StateClosed && oldState == StateHalfOpen {
		cb.metrics.CircuitCloses++
	}

	// Reset counters when state changes
	if state == StateClosed {
		cb.failureCount = 0
		cb.successCount = 0
		cb.requestCount = 0
		cb.metrics = &CircuitBreakerMetrics{
			LastFailureTime: cb.metrics.LastFailureTime,
			LastSuccessTime: cb.metrics.LastSuccessTime,
		}
	}
}

// incrementConcurrentRequests increments the concurrent request counter
func (cb *CircuitBreaker) incrementConcurrentRequests() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.concurrentRequests++
}

// decrementConcurrentRequests decrements the concurrent request counter
func (cb *CircuitBreaker) decrementConcurrentRequests() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.concurrentRequests--
}

// categorizeError categorizes an error for metrics
func (cb *CircuitBreaker) categorizeError(err error) string {
	if err == nil {
		return \"\"
	}

	grpcStatus := status.Code(err)
	switch grpcStatus {
	case codes.DeadlineExceeded:
		return \"timeout\"
	case codes.Unavailable:
		return \"unavailable\"
	case codes.Internal:
		return \"internal\"
	case codes.ResourceExhausted:
		return \"resource_exhausted\"
	case codes.Unauthenticated:
		return \"unauthenticated\"
	case codes.PermissionDenied:
		return \"permission_denied\"
	default:
		return \"unknown\"
	}
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// GetMetrics returns a copy of the current metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return *cb.metrics
}

// GetStats returns detailed statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	successRate := float64(0)
	if cb.metrics.Requests > 0 {
		successRate = float64(cb.metrics.Successes) / float64(cb.metrics.Requests) * 100
	}

	errorRate := float64(0)
	if cb.metrics.Requests > 0 {
		errorRate = float64(cb.metrics.Failures+cb.metrics.Timeouts) / float64(cb.metrics.Requests) * 100
	}

	return map[string]interface{}{
		\"name\":                  cb.name,
		\"state\":                 cb.state.String(),
		\"total_requests\":        cb.metrics.Requests,
		\"successes\":             cb.metrics.Successes,
		\"failures\":              cb.metrics.Failures,
		\"timeouts\":              cb.metrics.Timeouts,
		\"success_rate\":          successRate,
		\"error_rate\":            errorRate,
		\"concurrent_requests\":   cb.concurrentRequests,
		\"failure_count\":         cb.failureCount,
		\"success_count\":         cb.successCount,
		\"circuit_opens\":         cb.metrics.CircuitOpens,
		\"circuit_closes\":        cb.metrics.CircuitCloses,
		\"last_failure_time\":     cb.lastFailureTime,
		\"last_success_time\":     cb.lastSuccessTime,
		\"last_state_change\":     cb.lastStateChangeTime,
		\"fallback_successes\":    cb.metrics.FallbackSuccess,
		\"fallback_failures\":     cb.metrics.FallbackFailures,
		\"config\": map[string]interface{}{
			\"failure_threshold\":       cb.config.FailureThreshold,
			\"success_threshold\":       cb.config.SuccessThreshold,
			\"timeout\":                 cb.config.Timeout,
			\"max_concurrent_requests\": cb.config.MaxConcurrentRequests,
			\"request_volume_threshold\": cb.config.RequestVolumeThreshold,
			\"sleep_window\":            cb.config.SleepWindow,
			\"error_percent_threshold\": cb.config.ErrorPercentThreshold,
		},
	}
}

// Reset resets the circuit breaker to its initial state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(StateClosed)
	cb.failureCount = 0
	cb.successCount = 0
	cb.concurrentRequests = 0
	cb.metrics = &CircuitBreakerMetrics{}

	cb.logger.Info(\"Circuit breaker reset\", map[string]interface{}{
		\"circuit_breaker\": cb.name,
	})
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	circuitBreakers map[string]*CircuitBreaker
	authMetrics     *metrics.AuthMetrics
	logger          logging.Logger
	mu              sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(authMetrics *metrics.AuthMetrics, logger logging.Logger) *CircuitBreakerManager {
	manager := &CircuitBreakerManager{
		circuitBreakers: make(map[string]*CircuitBreaker),
		authMetrics:     authMetrics,
		logger:          logger,
	}

	// Initialize default circuit breakers
	for name, config := range DefaultCircuitBreakerConfigs() {
		manager.AddCircuitBreaker(name, config)
	}

	return manager
}

// AddCircuitBreaker adds a new circuit breaker
func (cbm *CircuitBreakerManager) AddCircuitBreaker(name string, config *CircuitBreakerConfig) {
	cbm.mu.Lock()
	defer cbm.mu.Unlock()

	cbm.circuitBreakers[name] = NewCircuitBreaker(config, cbm.authMetrics, cbm.logger)
	cbm.logger.Info(\"Circuit breaker added\", map[string]interface{}{
		\"name\":   name,
		\"config\": config,
	})
}

// GetCircuitBreaker gets a circuit breaker by name
func (cbm *CircuitBreakerManager) GetCircuitBreaker(name string) *CircuitBreaker {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()
	return cbm.circuitBreakers[name]
}

// GetAllStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetAllStats() map[string]interface{} {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	stats := make(map[string]interface{})
	for name, cb := range cbm.circuitBreakers {
		stats[name] = cb.GetStats()
	}
	return stats
}

// ResetAll resets all circuit breakers
func (cbm *CircuitBreakerManager) ResetAll() {
	cbm.mu.RLock()
	defer cbm.mu.RUnlock()

	for name, cb := range cbm.circuitBreakers {
		cb.Reset()
		cbm.logger.Info(\"Circuit breaker reset\", map[string]interface{}{\"name\": name})
	}
}