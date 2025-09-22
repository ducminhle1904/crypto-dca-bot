package safety

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// String returns the string representation of the circuit breaker state
func (s CircuitBreakerState) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds configuration for a circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold uint32        // Number of failures before opening
	SuccessThreshold uint32        // Number of successes to close from half-open
	Timeout          time.Duration // Time to wait before trying again
	MaxFailures      uint32        // Maximum failures in time window
	ResetTimeout     time.Duration // Time window for failure counting
}

// CircuitBreaker implements the circuit breaker pattern for preventing cascading failures
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitBreakerState
	failures     uint32
	successes    uint32
	lastFailure  time.Time
	nextAttempt  time.Time
	mutex        sync.RWMutex
	name         string
	onStateChange func(from, to CircuitBreakerState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	// Set defaults if not provided
	if config.FailureThreshold == 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold == 0 {
		config.SuccessThreshold = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	if config.MaxFailures == 0 {
		config.MaxFailures = 10
	}
	if config.ResetTimeout == 0 {
		config.ResetTimeout = 5 * time.Minute
	}

	return &CircuitBreaker{
		config: config,
		state:  StateClosed,
		name:   name,
	}
}

// SetStateChangeCallback sets a callback to be called when the state changes
func (cb *CircuitBreaker) SetStateChangeCallback(callback func(from, to CircuitBreakerState)) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	cb.onStateChange = callback
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	if !cb.canExecute() {
		return fmt.Errorf("circuit breaker %s is open", cb.name)
	}

	err := fn()
	
	if err != nil {
		cb.recordFailure()
		return err
	}
	
	cb.recordSuccess()
	return nil
}

// canExecute determines if the circuit breaker allows execution
func (cb *CircuitBreaker) canExecute() bool {
	cb.mutex.RLock()
	state := cb.state
	nextAttempt := cb.nextAttempt
	cb.mutex.RUnlock()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Now().After(nextAttempt) {
			cb.toHalfOpen()
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return false
	}
}

// recordSuccess records a successful execution
func (cb *CircuitBreaker) recordSuccess() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures = 0 // Reset failure count on success

	switch cb.state {
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.config.SuccessThreshold {
			cb.toClosed()
		}
	case StateClosed:
		// Already closed, nothing to do
	case StateOpen:
		// Should not happen, but handle gracefully
		cb.toClosed()
	}
}

// recordFailure records a failed execution
func (cb *CircuitBreaker) recordFailure() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	cb.failures++
	cb.lastFailure = time.Now()

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.toOpen()
		}
	case StateHalfOpen:
		cb.toOpen()
	case StateOpen:
		// Already open, just update the next attempt time
		cb.nextAttempt = time.Now().Add(cb.config.Timeout)
	}

	// Check for maximum failures in time window
	if cb.failures >= cb.config.MaxFailures {
		cb.toOpen()
		// Extend timeout for excessive failures
		cb.nextAttempt = time.Now().Add(cb.config.Timeout * 2)
	}
}

// toClosed transitions to closed state
func (cb *CircuitBreaker) toClosed() {
	cb.changeState(StateClosed)
	cb.failures = 0
	cb.successes = 0
}

// toOpen transitions to open state
func (cb *CircuitBreaker) toOpen() {
	cb.changeState(StateOpen)
	cb.nextAttempt = time.Now().Add(cb.config.Timeout)
	cb.successes = 0
}

// toHalfOpen transitions to half-open state
func (cb *CircuitBreaker) toHalfOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.changeState(StateHalfOpen)
	cb.successes = 0
}

// changeState changes the circuit breaker state and calls the callback
func (cb *CircuitBreaker) changeState(newState CircuitBreakerState) {
	oldState := cb.state
	cb.state = newState
	
	if cb.onStateChange != nil && oldState != newState {
		// Call callback without holding the mutex to avoid deadlock
		go cb.onStateChange(oldState, newState)
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns statistics about the circuit breaker
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return CircuitBreakerStats{
		Name:         cb.name,
		State:        cb.state,
		Failures:     cb.failures,
		Successes:    cb.successes,
		LastFailure:  cb.lastFailure,
		NextAttempt:  cb.nextAttempt,
	}
}

// CircuitBreakerStats holds statistics about a circuit breaker
type CircuitBreakerStats struct {
	Name         string
	State        CircuitBreakerState
	Failures     uint32
	Successes    uint32
	LastFailure  time.Time
	NextAttempt  time.Time
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.toClosed()
}

// ForceOpen forces the circuit breaker to open state
func (cb *CircuitBreaker) ForceOpen() {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()
	
	cb.toOpen()
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetOrCreate gets an existing circuit breaker or creates a new one
func (cbm *CircuitBreakerManager) GetOrCreate(name string, config CircuitBreakerConfig) *CircuitBreaker {
	cbm.mutex.RLock()
	if cb, exists := cbm.breakers[name]; exists {
		cbm.mutex.RUnlock()
		return cb
	}
	cbm.mutex.RUnlock()

	cbm.mutex.Lock()
	defer cbm.mutex.Unlock()
	
	// Double-check after acquiring write lock
	if cb, exists := cbm.breakers[name]; exists {
		return cb
	}
	
	cb := NewCircuitBreaker(name, config)
	cbm.breakers[name] = cb
	return cb
}

// Get gets an existing circuit breaker
func (cbm *CircuitBreakerManager) Get(name string) (*CircuitBreaker, bool) {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	cb, exists := cbm.breakers[name]
	return cb, exists
}

// GetAll returns all circuit breakers
func (cbm *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	result := make(map[string]*CircuitBreaker)
	for name, cb := range cbm.breakers {
		result[name] = cb
	}
	return result
}

// GetStats returns statistics for all circuit breakers
func (cbm *CircuitBreakerManager) GetStats() []CircuitBreakerStats {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	stats := make([]CircuitBreakerStats, 0, len(cbm.breakers))
	for _, cb := range cbm.breakers {
		stats = append(stats, cb.GetStats())
	}
	return stats
}

// Reset resets all circuit breakers
func (cbm *CircuitBreakerManager) Reset() {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	for _, cb := range cbm.breakers {
		cb.Reset()
	}
}

// HasOpenCircuits returns true if any circuit breakers are open
func (cbm *CircuitBreakerManager) HasOpenCircuits() bool {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	for _, cb := range cbm.breakers {
		if cb.GetState() == StateOpen {
			return true
		}
	}
	return false
}

// GetOpenCircuits returns a list of open circuit breaker names
func (cbm *CircuitBreakerManager) GetOpenCircuits() []string {
	cbm.mutex.RLock()
	defer cbm.mutex.RUnlock()
	
	var openCircuits []string
	for name, cb := range cbm.breakers {
		if cb.GetState() == StateOpen {
			openCircuits = append(openCircuits, name)
		}
	}
	return openCircuits
}
