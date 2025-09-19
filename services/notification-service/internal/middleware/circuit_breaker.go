package middleware

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"reciprocal-clubs-backend/pkg/shared/logging"
)

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	// StateClosed allows all requests through
	StateClosed CircuitState = iota
	// StateOpen blocks all requests
	StateOpen
	// StateHalfOpen allows limited requests to test if service recovered
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name            string
	maxRequests     uint32
	interval        time.Duration
	timeout         time.Duration
	readyToTrip     func(counts Counts) bool
	onStateChange   func(name string, from CircuitState, to CircuitState)

	mutex       sync.Mutex
	state       CircuitState
	generation  uint64
	counts      Counts
	expiry      time.Time
	logger      logging.Logger
}

// Counts holds the numbers of requests and their successes/failures
type Counts struct {
	Requests             uint32
	TotalSuccesses       uint32
	TotalFailures        uint32
	ConsecutiveSuccesses uint32
	ConsecutiveFailures  uint32
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	Name         string
	MaxRequests  uint32
	Interval     time.Duration
	Timeout      time.Duration
	ReadyToTrip  func(counts Counts) bool
	OnStateChange func(name string, from CircuitState, to CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig, logger logging.Logger) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:        config.Name,
		maxRequests: config.MaxRequests,
		interval:    config.Interval,
		timeout:     config.Timeout,
		logger:      logger,
	}

	if config.ReadyToTrip == nil {
		cb.readyToTrip = defaultReadyToTrip
	} else {
		cb.readyToTrip = config.ReadyToTrip
	}

	if config.OnStateChange != nil {
		cb.onStateChange = config.OnStateChange
	}

	cb.toNewGeneration(time.Now())
	return cb
}

// defaultReadyToTrip is the default implementation of ReadyToTrip
func defaultReadyToTrip(counts Counts) bool {
	return counts.ConsecutiveFailures > 5
}

// Execute runs the given request if the circuit breaker allows it
func (cb *CircuitBreaker) Execute(req func() (interface{}, error)) (interface{}, error) {
	generation, err := cb.beforeRequest()
	if err != nil {
		return nil, err
	}

	defer func() {
		e := recover()
		if e != nil {
			cb.afterRequest(generation, false)
			panic(e)
		}
	}()

	result, err := req()
	cb.afterRequest(generation, err == nil)
	return result, err
}

// Call is a shorthand for Execute
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(ctx context.Context) error) error {
	_, err := cb.Execute(func() (interface{}, error) {
		return nil, fn(ctx)
	})
	return err
}

// beforeRequest is called before a request
func (cb *CircuitBreaker) beforeRequest() (uint64, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)

	if state == StateOpen {
		cb.logger.Debug("Circuit breaker is open, rejecting request", map[string]interface{}{
			"name": cb.name,
		})
		return generation, errors.New("circuit breaker is open")
	} else if state == StateHalfOpen && cb.counts.Requests >= cb.maxRequests {
		cb.logger.Debug("Circuit breaker is half-open with max requests, rejecting request", map[string]interface{}{
			"name":     cb.name,
			"requests": cb.counts.Requests,
		})
		return generation, errors.New("circuit breaker is half-open with too many requests")
	}

	cb.counts.onRequest()
	return generation, nil
}

// afterRequest is called after a request
func (cb *CircuitBreaker) afterRequest(before uint64, success bool) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, generation := cb.currentState(now)
	if generation != before {
		return
	}

	if success {
		cb.onSuccess(state, now)
	} else {
		cb.onFailure(state, now)
	}
}

// onSuccess handles successful requests
func (cb *CircuitBreaker) onSuccess(state CircuitState, now time.Time) {
	cb.counts.onSuccess()

	if state == StateHalfOpen && cb.counts.ConsecutiveSuccesses >= cb.maxRequests {
		cb.setState(StateClosed, now)
	}
}

// onFailure handles failed requests
func (cb *CircuitBreaker) onFailure(state CircuitState, now time.Time) {
	cb.counts.onFailure()

	if cb.readyToTrip(cb.counts) {
		cb.setState(StateOpen, now)
	}
}

// currentState returns the current state of the circuit breaker
func (cb *CircuitBreaker) currentState(now time.Time) (CircuitState, uint64) {
	switch cb.state {
	case StateClosed:
		if !cb.expiry.IsZero() && cb.expiry.Before(now) {
			cb.toNewGeneration(now)
		}
	case StateOpen:
		if cb.expiry.Before(now) {
			cb.setState(StateHalfOpen, now)
		}
	}
	return cb.state, cb.generation
}

// setState changes the state of the circuit breaker
func (cb *CircuitBreaker) setState(state CircuitState, now time.Time) {
	if cb.state == state {
		return
	}

	prev := cb.state
	cb.state = state

	cb.toNewGeneration(now)

	if cb.onStateChange != nil {
		cb.onStateChange(cb.name, prev, state)
	}

	cb.logger.Info("Circuit breaker state changed", map[string]interface{}{
		"name":      cb.name,
		"from":      prev.String(),
		"to":        state.String(),
		"counts":    fmt.Sprintf("%+v", cb.counts),
	})
}

// toNewGeneration starts a new generation
func (cb *CircuitBreaker) toNewGeneration(now time.Time) {
	cb.generation++
	cb.counts.clear()

	var zero time.Time
	switch cb.state {
	case StateClosed:
		if cb.interval == 0 {
			cb.expiry = zero
		} else {
			cb.expiry = now.Add(cb.interval)
		}
	case StateOpen:
		cb.expiry = now.Add(cb.timeout)
	default: // StateHalfOpen
		cb.expiry = zero
	}
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() CircuitState {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	state, _ := cb.currentState(now)
	return state
}

// Counts returns a copy of the current counts
func (cb *CircuitBreaker) Counts() Counts {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	return cb.counts
}

// Methods for Counts
func (c *Counts) onRequest() {
	c.Requests++
}

func (c *Counts) onSuccess() {
	c.TotalSuccesses++
	c.ConsecutiveSuccesses++
	c.ConsecutiveFailures = 0
}

func (c *Counts) onFailure() {
	c.TotalFailures++
	c.ConsecutiveFailures++
	c.ConsecutiveSuccesses = 0
}

func (c *Counts) clear() {
	c.Requests = 0
	c.TotalSuccesses = 0
	c.TotalFailures = 0
	c.ConsecutiveSuccesses = 0
	c.ConsecutiveFailures = 0
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
	logger   logging.Logger
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(logger logging.Logger) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
		logger:   logger,
	}
}

// GetBreaker returns a circuit breaker for the given name
func (cbm *CircuitBreakerManager) GetBreaker(name string) *CircuitBreaker {
	cbm.mutex.RLock()
	breaker, exists := cbm.breakers[name]
	cbm.mutex.RUnlock()

	if exists {
		return breaker
	}

	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	// Double check after acquiring write lock
	if breaker, exists := cbm.breakers[name]; exists {
		return breaker
	}

	// Create default circuit breaker
	config := CircuitBreakerConfig{
		Name:        name,
		MaxRequests: 5,
		Interval:    time.Minute,
		Timeout:     30 * time.Second,
		OnStateChange: func(name string, from CircuitState, to CircuitState) {
			cbm.logger.Info("Circuit breaker state changed", map[string]interface{}{
				"name": name,
				"from": from.String(),
				"to":   to.String(),
			})
		},
	}

	breaker = NewCircuitBreaker(config, cbm.logger)
	cbm.breakers[name] = breaker

	return breaker
}

// GetBreakerStatus returns the status of all circuit breakers
func (cbm *CircuitBreakerManager) GetBreakerStatus() map[string]interface{} {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()

	status := make(map[string]interface{})
	for name, breaker := range cbm.breakers {
		status[name] = map[string]interface{}{
			"state":  breaker.State().String(),
			"counts": breaker.Counts(),
		}
	}

	return status
}

// ResetBreaker resets a specific circuit breaker
func (cbm *CircuitBreakerManager) ResetBreaker(name string) error {
	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()

	breaker, exists := cbm.breakers[name]
	if !exists {
		return fmt.Errorf("circuit breaker %s not found", name)
	}

	breaker.mutex.Lock()
	breaker.setState(StateClosed, time.Now())
	breaker.mutex.Unlock()

	cbm.logger.Info("Circuit breaker reset", map[string]interface{}{
		"name": name,
	})

	return nil
}