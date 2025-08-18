package bybit

import (
	"context"
	"math"
	"time"
)

// RetryConfig holds configuration for retry mechanisms
type RetryConfig struct {
	MaxRetries      int           `json:"maxRetries"`
	InitialDelay    time.Duration `json:"initialDelay"`
	MaxDelay        time.Duration `json:"maxDelay"`
	BackoffFactor   float64       `json:"backoffFactor"`
	JitterEnabled   bool          `json:"jitterEnabled"`
	RetryableErrors []int         `json:"retryableErrors"`
}

// DefaultRetryConfig returns a default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      time.Minute,
		BackoffFactor: 2.0,
		JitterEnabled: true,
		RetryableErrors: []int{
			ErrCodeRateLimitExceeded,
			500, // Internal Server Error
			502, // Bad Gateway
			503, // Service Unavailable
			504, // Gateway Timeout
		},
	}
}

// RetryableFunc represents a function that can be retried
type RetryableFunc func() error

// Retry executes a function with retry logic
func (c *Client) Retry(ctx context.Context, fn RetryableFunc) error {
	return c.RetryWithConfig(ctx, fn, DefaultRetryConfig())
}

// RetryWithConfig executes a function with custom retry configuration
func (c *Client) RetryWithConfig(ctx context.Context, fn RetryableFunc, config RetryConfig) error {
	var lastErr error
	
	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		
		// Execute the function
		err := fn()
		if err == nil {
			return nil // Success
		}
		
		lastErr = err
		
		// Check if this is the last attempt
		if attempt == config.MaxRetries {
			break
		}
		
		// Check if the error is retryable
		if !c.isRetryableError(err, config.RetryableErrors) {
			break
		}
		
		// Calculate delay for next attempt
		delay := c.calculateDelay(attempt, config)
		
		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
	
	return WrapAPIError("retry exhausted", lastErr)
}

// isRetryableError checks if an error should be retried based on configuration
func (c *Client) isRetryableError(err error, retryableCodes []int) bool {
	if IsRetryableError(err) {
		return true
	}
	
	if bybitErr, ok := err.(*BybitError); ok {
		for _, code := range retryableCodes {
			if bybitErr.Code == code {
				return true
			}
		}
	}
	
	return false
}

// calculateDelay calculates the delay for a retry attempt with exponential backoff
func (c *Client) calculateDelay(attempt int, config RetryConfig) time.Duration {
	delay := config.InitialDelay
	
	// Apply exponential backoff
	if attempt > 0 {
		delay = time.Duration(float64(config.InitialDelay) * math.Pow(config.BackoffFactor, float64(attempt)))
	}
	
	// Cap at maximum delay
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}
	
	// Add jitter if enabled
	if config.JitterEnabled {
		jitter := time.Duration(float64(delay) * 0.1 * (2*randFloat() - 1))
		delay += jitter
	}
	
	return delay
}

// randFloat returns a random float between 0 and 1
func randFloat() float64 {
	// Simple pseudo-random implementation
	// In production, you might want to use crypto/rand
	return float64(time.Now().UnixNano()%1000) / 1000.0
}

// RetryableAPICall wraps an API call with retry logic
func (c *Client) RetryableAPICall(ctx context.Context, operation string, fn func() (interface{}, error)) (interface{}, error) {
	var result interface{}
	var resultErr error
	
	retryFn := func() error {
		var err error
		result, err = fn()
		resultErr = err
		return err
	}
	
	err := c.Retry(ctx, retryFn)
	if err != nil {
		return nil, WrapAPIError(operation, err)
	}
	
	return result, resultErr
}

// CircuitBreaker represents a simple circuit breaker pattern
type CircuitBreaker struct {
	MaxFailures    int           `json:"maxFailures"`
	ResetTimeout   time.Duration `json:"resetTimeout"`
	failures       int
	lastFailTime   time.Time
	state          CircuitState
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		MaxFailures:  maxFailures,
		ResetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

// Call executes a function through the circuit breaker
func (cb *CircuitBreaker) Call(fn func() error) error {
	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailTime) > cb.ResetTimeout {
			cb.state = CircuitHalfOpen
		} else {
			return NewBybitError(503, "Circuit breaker is open")
		}
	}
	
	err := fn()
	
	if err != nil {
		cb.onFailure()
		return err
	}
	
	cb.onSuccess()
	return nil
}

// onFailure handles failure cases
func (cb *CircuitBreaker) onFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()
	
	if cb.failures >= cb.MaxFailures {
		cb.state = CircuitOpen
	}
}

// onSuccess handles success cases
func (cb *CircuitBreaker) onSuccess() {
	cb.failures = 0
	cb.state = CircuitClosed
}
