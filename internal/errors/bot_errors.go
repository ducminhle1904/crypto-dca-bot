package errors

import (
	"fmt"
	"strings"
)

// ErrorCategory represents different types of errors that can occur
type ErrorCategory string

const (
	// Critical errors that should stop the bot
	ErrorCategoryFatal        ErrorCategory = "FATAL"
	ErrorCategoryExchange     ErrorCategory = "EXCHANGE"
	ErrorCategoryCredentials  ErrorCategory = "CREDENTIALS"
	ErrorCategoryConfiguration ErrorCategory = "CONFIG"
	
	// Non-critical errors that can be retried or recovered from
	ErrorCategoryNetwork      ErrorCategory = "NETWORK"
	ErrorCategoryTimeout      ErrorCategory = "TIMEOUT"
	ErrorCategoryValidation   ErrorCategory = "VALIDATION"
	ErrorCategoryOrder        ErrorCategory = "ORDER"
	ErrorCategoryPosition     ErrorCategory = "POSITION"
	ErrorCategoryStrategy     ErrorCategory = "STRATEGY"
	
	// Temporary errors
	ErrorCategoryTemporary    ErrorCategory = "TEMPORARY"
	ErrorCategoryRateLimit    ErrorCategory = "RATE_LIMIT"
)

// BotError represents a categorized error with context
type BotError struct {
	Category   ErrorCategory
	Component  string
	Operation  string
	Message    string
	Underlying error
	Context    map[string]interface{}
	Retryable  bool
}

// Error implements the error interface
func (e *BotError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("[%s:%s] %s in %s: %v", e.Category, e.Component, e.Operation, e.Message, e.Underlying)
	}
	return fmt.Sprintf("[%s:%s] %s in %s", e.Category, e.Component, e.Operation, e.Message)
}

// Unwrap returns the underlying error for error unwrapping
func (e *BotError) Unwrap() error {
	return e.Underlying
}

// IsRetryable returns whether this error can be retried
func (e *BotError) IsRetryable() bool {
	return e.Retryable
}

// IsFatal returns whether this error should stop the bot
func (e *BotError) IsFatal() bool {
	return e.Category == ErrorCategoryFatal || 
		   e.Category == ErrorCategoryCredentials ||
		   e.Category == ErrorCategoryConfiguration
}

// NewBotError creates a new categorized bot error
func NewBotError(category ErrorCategory, component, operation, message string) *BotError {
	return &BotError{
		Category:  category,
		Component: component,
		Operation: operation,
		Message:   message,
		Context:   make(map[string]interface{}),
		Retryable: isRetryableCategory(category),
	}
}

// WrapError wraps an existing error with bot error context
func WrapError(err error, category ErrorCategory, component, operation string) *BotError {
	if err == nil {
		return nil
	}
	
	return &BotError{
		Category:   category,
		Component:  component,
		Operation:  operation,
		Message:    "operation failed",
		Underlying: err,
		Context:    make(map[string]interface{}),
		Retryable:  isRetryableCategory(category),
	}
}

// WithContext adds context information to the error
func (e *BotError) WithContext(key string, value interface{}) *BotError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithRetryable sets the retryable flag
func (e *BotError) WithRetryable(retryable bool) *BotError {
	e.Retryable = retryable
	return e
}

// isRetryableCategory determines if an error category is generally retryable
func isRetryableCategory(category ErrorCategory) bool {
	switch category {
	case ErrorCategoryNetwork, ErrorCategoryTimeout, ErrorCategoryTemporary, ErrorCategoryRateLimit:
		return true
	case ErrorCategoryFatal, ErrorCategoryCredentials, ErrorCategoryConfiguration:
		return false
	default:
		return true // Default to retryable for safety
	}
}

// CategorizeError attempts to categorize a generic error
func CategorizeError(err error, component, operation string) *BotError {
	if err == nil {
		return nil
	}
	
	// Check if it's already a BotError
	if botErr, ok := err.(*BotError); ok {
		return botErr
	}
	
	errMsg := strings.ToLower(err.Error())
	
	// Network-related errors
	if strings.Contains(errMsg, "timeout") || strings.Contains(errMsg, "context deadline exceeded") {
		return WrapError(err, ErrorCategoryTimeout, component, operation)
	}
	
	if strings.Contains(errMsg, "connection") || strings.Contains(errMsg, "network") || 
	   strings.Contains(errMsg, "dns") || strings.Contains(errMsg, "dial") {
		return WrapError(err, ErrorCategoryNetwork, component, operation)
	}
	
	// Exchange-related errors
	if strings.Contains(errMsg, "api key") || strings.Contains(errMsg, "api secret") || 
	   strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "unauthorized") {
		return WrapError(err, ErrorCategoryCredentials, component, operation)
	}
	
	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "too many requests") {
		return WrapError(err, ErrorCategoryRateLimit, component, operation)
	}
	
	if strings.Contains(errMsg, "insufficient") || strings.Contains(errMsg, "balance") {
		return WrapError(err, ErrorCategoryOrder, component, operation).WithRetryable(false)
	}
	
	if strings.Contains(errMsg, "invalid") || strings.Contains(errMsg, "constraint") ||
	   strings.Contains(errMsg, "minimum") || strings.Contains(errMsg, "maximum") {
		return WrapError(err, ErrorCategoryValidation, component, operation).WithRetryable(false)
	}
	
	// Default to temporary error for unknown cases
	return WrapError(err, ErrorCategoryTemporary, component, operation)
}

// Common error constructors
func NewNetworkError(component, operation string, err error) *BotError {
	return WrapError(err, ErrorCategoryNetwork, component, operation)
}

func NewTimeoutError(component, operation string, err error) *BotError {
	return WrapError(err, ErrorCategoryTimeout, component, operation)
}

func NewValidationError(component, operation, message string) *BotError {
	return NewBotError(ErrorCategoryValidation, component, operation, message).WithRetryable(false)
}

func NewConfigurationError(component, operation, message string) *BotError {
	return NewBotError(ErrorCategoryConfiguration, component, operation, message).WithRetryable(false)
}

func NewCredentialsError(component, operation, message string) *BotError {
	return NewBotError(ErrorCategoryCredentials, component, operation, message).WithRetryable(false)
}

func NewOrderError(component, operation string, err error) *BotError {
	return WrapError(err, ErrorCategoryOrder, component, operation)
}

func NewPositionError(component, operation string, err error) *BotError {
	return WrapError(err, ErrorCategoryPosition, component, operation)
}

func NewStrategyError(component, operation string, err error) *BotError {
	return WrapError(err, ErrorCategoryStrategy, component, operation)
}

func NewFatalError(component, operation, message string) *BotError {
	return NewBotError(ErrorCategoryFatal, component, operation, message).WithRetryable(false)
}

// Error recovery strategies
type RecoveryAction string

const (
	RecoveryActionRetry     RecoveryAction = "RETRY"
	RecoveryActionSkip      RecoveryAction = "SKIP"
	RecoveryActionStop      RecoveryAction = "STOP"
	RecoveryActionFallback  RecoveryAction = "FALLBACK"
	RecoveryActionWait      RecoveryAction = "WAIT"
)

// GetRecoveryAction suggests a recovery action based on error category
func (e *BotError) GetRecoveryAction() RecoveryAction {
	switch e.Category {
	case ErrorCategoryFatal, ErrorCategoryCredentials, ErrorCategoryConfiguration:
		return RecoveryActionStop
	case ErrorCategoryRateLimit:
		return RecoveryActionWait
	case ErrorCategoryNetwork, ErrorCategoryTimeout, ErrorCategoryTemporary:
		return RecoveryActionRetry
	case ErrorCategoryValidation:
		return RecoveryActionSkip
	case ErrorCategoryOrder, ErrorCategoryPosition:
		if e.Retryable {
			return RecoveryActionRetry
		}
		return RecoveryActionSkip
	default:
		return RecoveryActionRetry
	}
}

// ErrorStats tracks error statistics
type ErrorStats struct {
	TotalErrors     int
	ErrorsByCategory map[ErrorCategory]int
	RecentErrors    []*BotError
	MaxRecentErrors int
}

// NewErrorStats creates a new error statistics tracker
func NewErrorStats(maxRecentErrors int) *ErrorStats {
	return &ErrorStats{
		ErrorsByCategory: make(map[ErrorCategory]int),
		RecentErrors:    make([]*BotError, 0, maxRecentErrors),
		MaxRecentErrors: maxRecentErrors,
	}
}

// RecordError records an error in the statistics
func (es *ErrorStats) RecordError(err *BotError) {
	es.TotalErrors++
	es.ErrorsByCategory[err.Category]++
	
	// Add to recent errors
	es.RecentErrors = append(es.RecentErrors, err)
	
	// Keep only the most recent errors
	if len(es.RecentErrors) > es.MaxRecentErrors {
		es.RecentErrors = es.RecentErrors[1:]
	}
}

// GetErrorRate returns the error rate for a specific category
func (es *ErrorStats) GetErrorRate(category ErrorCategory) float64 {
	if es.TotalErrors == 0 {
		return 0.0
	}
	return float64(es.ErrorsByCategory[category]) / float64(es.TotalErrors)
}

// HasRecentErrors checks if there have been errors in the recent history
func (es *ErrorStats) HasRecentErrors(category ErrorCategory, count int) bool {
	recentCount := 0
	for _, err := range es.RecentErrors {
		if err.Category == category {
			recentCount++
		}
	}
	return recentCount >= count
}
