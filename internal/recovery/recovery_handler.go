package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/errors"
)

// RecoveryHandler handles error recovery strategies
type RecoveryHandler struct {
	errorStats    *errors.ErrorStats
	retryConfig   RetryConfig
	logger        Logger
	maxRetries    map[errors.ErrorCategory]int
	backoffConfig BackoffConfig
}

// RetryConfig defines retry behavior for different error categories
type RetryConfig struct {
	MaxRetries map[errors.ErrorCategory]int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
}

// BackoffConfig defines backoff strategies
type BackoffConfig struct {
	Strategy     BackoffStrategy
	Multiplier   float64
	Jitter       bool
	MaxBackoff   time.Duration
}

// BackoffStrategy defines different backoff strategies
type BackoffStrategy string

const (
	BackoffExponential BackoffStrategy = "exponential"
	BackoffLinear      BackoffStrategy = "linear"
	BackoffFixed       BackoffStrategy = "fixed"
)

// Logger interface for recovery handler
type Logger interface {
	Info(format string, args ...interface{})
	LogWarning(component, message string, args ...interface{})
	Error(format string, args ...interface{})
	LogDebugOnly(format string, args ...interface{})
}

// RecoveryResult represents the result of a recovery attempt
type RecoveryResult struct {
	Action    errors.RecoveryAction
	Delay     time.Duration
	ShouldStop bool
	Message   string
}

// NewRecoveryHandler creates a new recovery handler
func NewRecoveryHandler(logger Logger) *RecoveryHandler {
	defaultRetryConfig := RetryConfig{
		MaxRetries: map[errors.ErrorCategory]int{
			errors.ErrorCategoryNetwork:    5,
			errors.ErrorCategoryTimeout:   3,
			errors.ErrorCategoryTemporary: 3,
			errors.ErrorCategoryRateLimit: 10,
			errors.ErrorCategoryOrder:     2,
			errors.ErrorCategoryPosition:  3,
			errors.ErrorCategoryStrategy:  1,
		},
		BaseDelay: 1 * time.Second,
		MaxDelay:  30 * time.Second,
	}

	backoffConfig := BackoffConfig{
		Strategy:   BackoffExponential,
		Multiplier: 1.5,
		Jitter:     true,
		MaxBackoff: 5 * time.Minute,
	}

	return &RecoveryHandler{
		errorStats:    errors.NewErrorStats(50), // Keep last 50 errors
		retryConfig:   defaultRetryConfig,
		logger:        logger,
		backoffConfig: backoffConfig,
	}
}

// HandleError processes an error and returns a recovery strategy
func (rh *RecoveryHandler) HandleError(err error, component, operation string, attempt int) *RecoveryResult {
	// Categorize the error
	botError := errors.CategorizeError(err, component, operation)
	
	// Record the error in statistics
	rh.errorStats.RecordError(botError)
	
	// Log the error with appropriate level
	rh.logError(botError, attempt)
	
	// Determine recovery action
	action := botError.GetRecoveryAction()
	
	// Check if we should stop based on error patterns
	if rh.shouldStop(botError, attempt) {
		return &RecoveryResult{
			Action:     errors.RecoveryActionStop,
			ShouldStop: true,
			Message:    rh.getStopReason(botError, attempt),
		}
	}
	
	// Calculate delay for retry/wait actions
	delay := rh.calculateDelay(botError.Category, attempt)
	
	return &RecoveryResult{
		Action:  action,
		Delay:   delay,
		Message: rh.getRecoveryMessage(action, botError, attempt),
	}
}

// shouldStop determines if the bot should stop based on error patterns
func (rh *RecoveryHandler) shouldStop(botError *errors.BotError, attempt int) bool {
	// Always stop for fatal errors
	if botError.IsFatal() {
		return true
	}
	
	// Stop if we've exceeded retry limits
	maxRetries, exists := rh.retryConfig.MaxRetries[botError.Category]
	if exists && attempt > maxRetries {
		rh.logger.Error("Maximum retries exceeded for %s errors (%d attempts)", botError.Category, attempt)
		return true
	}
	
	// Stop if we have too many recent errors of the same type
	if rh.errorStats.HasRecentErrors(botError.Category, 10) {
		rh.logger.Error("Too many recent %s errors, stopping for safety", botError.Category)
		return true
	}
	
	// Stop for critical error combinations
	if rh.hasCriticalErrorCombination() {
		return true
	}
	
	return false
}

// hasCriticalErrorCombination checks for dangerous error patterns
func (rh *RecoveryHandler) hasCriticalErrorCombination() bool {
	// Check for high credential error rate
	if rh.errorStats.GetErrorRate(errors.ErrorCategoryCredentials) > 0.5 {
		rh.logger.Error("High credential error rate detected, stopping")
		return true
	}
	
	// Check for high order error rate
	if rh.errorStats.GetErrorRate(errors.ErrorCategoryOrder) > 0.8 && rh.errorStats.TotalErrors > 10 {
		rh.logger.Error("High order error rate detected, possible trading issues")
		return true
	}
	
	return false
}

// calculateDelay calculates the delay before retry based on backoff strategy
func (rh *RecoveryHandler) calculateDelay(category errors.ErrorCategory, attempt int) time.Duration {
	baseDelay := rh.retryConfig.BaseDelay
	
	// Rate limiting needs longer delays
	if category == errors.ErrorCategoryRateLimit {
		baseDelay = 30 * time.Second
	}
	
	var delay time.Duration
	
	switch rh.backoffConfig.Strategy {
	case BackoffExponential:
		multiplier := 1.0
		for i := 0; i < attempt; i++ {
			multiplier *= rh.backoffConfig.Multiplier
		}
		delay = time.Duration(float64(baseDelay) * multiplier)
		
	case BackoffLinear:
		delay = baseDelay * time.Duration(attempt+1)
		
	case BackoffFixed:
		delay = baseDelay
		
	default:
		delay = baseDelay
	}
	
	// Apply maximum delay limit
	if delay > rh.retryConfig.MaxDelay {
		delay = rh.retryConfig.MaxDelay
	}
	
	if delay > rh.backoffConfig.MaxBackoff {
		delay = rh.backoffConfig.MaxBackoff
	}
	
	// Add jitter if enabled
	if rh.backoffConfig.Jitter {
		delay = addJitter(delay)
	}
	
	return delay
}

// addJitter adds random jitter to delay to avoid thundering herd
func addJitter(delay time.Duration) time.Duration {
	jitter := time.Duration(float64(delay) * 0.1) // 10% jitter
	return delay + time.Duration(time.Now().UnixNano()%int64(jitter))
}

// logError logs the error with appropriate level and context
func (rh *RecoveryHandler) logError(botError *errors.BotError, attempt int) {
	context := make(map[string]interface{})
	context["category"] = botError.Category
	context["component"] = botError.Component
	context["operation"] = botError.Operation
	context["attempt"] = attempt
	context["retryable"] = botError.Retryable
	
	// Merge with error context
	for k, v := range botError.Context {
		context[k] = v
	}
	
	if botError.IsFatal() {
		rh.logger.Error("FATAL ERROR: %s", botError.Error())
	} else if attempt > 1 {
		rh.logger.LogWarning("Error Recovery", "Attempt %d - %s", attempt, botError.Error())
	} else {
		rh.logger.LogDebugOnly("Error occurred: %s", botError.Error())
	}
}

// getRecoveryMessage returns a human-readable recovery message
func (rh *RecoveryHandler) getRecoveryMessage(action errors.RecoveryAction, botError *errors.BotError, attempt int) string {
	switch action {
	case errors.RecoveryActionRetry:
		return fmt.Sprintf("Retrying %s operation (attempt %d) after %s error", 
			botError.Operation, attempt+1, botError.Category)
	case errors.RecoveryActionWait:
		return fmt.Sprintf("Waiting before retry due to %s", botError.Category)
	case errors.RecoveryActionSkip:
		return fmt.Sprintf("Skipping operation due to non-retryable %s error", botError.Category)
	case errors.RecoveryActionStop:
		return fmt.Sprintf("Stopping bot due to %s error", botError.Category)
	case errors.RecoveryActionFallback:
		return fmt.Sprintf("Using fallback strategy for %s error", botError.Category)
	default:
		return fmt.Sprintf("Unknown recovery action for %s error", botError.Category)
	}
}

// getStopReason returns the reason why the bot should stop
func (rh *RecoveryHandler) getStopReason(botError *errors.BotError, attempt int) string {
	if botError.IsFatal() {
		return fmt.Sprintf("Fatal error in %s: %s", botError.Component, botError.Message)
	}
	
	maxRetries, exists := rh.retryConfig.MaxRetries[botError.Category]
	if exists && attempt > maxRetries {
		return fmt.Sprintf("Maximum retry attempts (%d) exceeded for %s errors", maxRetries, botError.Category)
	}
	
	if rh.errorStats.HasRecentErrors(botError.Category, 10) {
		return fmt.Sprintf("Too many recent %s errors detected", botError.Category)
	}
	
	return "Critical error pattern detected"
}

// ExecuteWithRecovery executes an operation with automatic recovery
func (rh *RecoveryHandler) ExecuteWithRecovery(
	ctx context.Context,
	component, operation string,
	fn func() error,
) error {
	var lastError error
	
	for attempt := 0; attempt < 10; attempt++ { // Hard limit on attempts
		// Check context cancellation
		if err := ctx.Err(); err != nil {
			return err
		}
		
		// Execute the function
		err := fn()
		if err == nil {
			// Success
			if attempt > 0 {
				rh.logger.Info("Operation %s.%s succeeded after %d attempts", component, operation, attempt+1)
			}
			return nil
		}
		
		lastError = err
		
		// Handle the error
		result := rh.HandleError(err, component, operation, attempt)
		
		// Check if we should stop
		if result.ShouldStop {
			rh.logger.Error("Stopping execution: %s", result.Message)
			return lastError
		}
		
		// Handle different recovery actions
		switch result.Action {
		case errors.RecoveryActionSkip:
			rh.logger.LogWarning("Recovery", "Skipping operation: %s", result.Message)
			return lastError
			
		case errors.RecoveryActionRetry, errors.RecoveryActionWait:
			if result.Delay > 0 {
				rh.logger.LogDebugOnly("Waiting %v before retry: %s", result.Delay, result.Message)
				
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(result.Delay):
					// Continue to next attempt
				}
			}
			
		case errors.RecoveryActionFallback:
			rh.logger.Info("Using fallback strategy: %s", result.Message)
			// For now, just continue with retry
			
		default:
			rh.logger.LogWarning("Recovery", "Unknown recovery action: %s", result.Action)
		}
	}
	
	return fmt.Errorf("operation failed after maximum attempts: %w", lastError)
}

// GetErrorStats returns the current error statistics
func (rh *RecoveryHandler) GetErrorStats() *errors.ErrorStats {
	return rh.errorStats
}

// ResetStats resets error statistics
func (rh *RecoveryHandler) ResetStats() {
	rh.errorStats = errors.NewErrorStats(50)
}
