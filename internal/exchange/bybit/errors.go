package bybit

import (
	"fmt"
	"net/http"
)

// BybitError represents a Bybit API error with additional context
type BybitError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *BybitError) Error() string {
	if e.Details != "" {
		return fmt.Sprintf("Bybit API error %d: %s (%s)", e.Code, e.Message, e.Details)
	}
	return fmt.Sprintf("Bybit API error %d: %s", e.Code, e.Message)
}

// Common Bybit error codes
const (
	ErrCodeInvalidAPIKey       = 10003
	ErrCodeInvalidSignature    = 10004
	ErrCodeInvalidTimestamp    = 10005
	ErrCodeInsufficientBalance = 110007
	ErrCodeOrderNotFound       = 110001
	ErrCodeSymbolNotFound      = 110009
	ErrCodeInvalidOrderType    = 110004
	ErrCodeInvalidQuantity     = 110020
	ErrCodeInvalidPrice        = 110021
	ErrCodeRateLimitExceeded   = 10006
	ErrCodeMarketClosed        = 110043
)

// IsRetryableError determines if an error should be retried
func IsRetryableError(err error) bool {
	if bybitErr, ok := err.(*BybitError); ok {
		switch bybitErr.Code {
		case ErrCodeRateLimitExceeded:
			return true
		case http.StatusInternalServerError:
			return true
		case http.StatusBadGateway:
			return true
		case http.StatusServiceUnavailable:
			return true
		case http.StatusGatewayTimeout:
			return true
		}
	}
	return false
}

// IsAuthenticationError checks if the error is related to authentication
func IsAuthenticationError(err error) bool {
	if bybitErr, ok := err.(*BybitError); ok {
		switch bybitErr.Code {
		case ErrCodeInvalidAPIKey, ErrCodeInvalidSignature, ErrCodeInvalidTimestamp:
			return true
		}
	}
	return false
}

// IsInsufficientBalanceError checks if the error is due to insufficient balance
func IsInsufficientBalanceError(err error) bool {
	if bybitErr, ok := err.(*BybitError); ok {
		return bybitErr.Code == ErrCodeInsufficientBalance
	}
	return false
}

// IsOrderNotFoundError checks if the error is due to order not found
func IsOrderNotFoundError(err error) bool {
	if bybitErr, ok := err.(*BybitError); ok {
		return bybitErr.Code == ErrCodeOrderNotFound
	}
	return false
}

// IsRateLimitError checks if the error is due to rate limiting
func IsRateLimitError(err error) bool {
	if bybitErr, ok := err.(*BybitError); ok {
		return bybitErr.Code == ErrCodeRateLimitExceeded
	}
	return false
}

// NewBybitError creates a new BybitError
func NewBybitError(code int, message string, details ...string) *BybitError {
	err := &BybitError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// WrapAPIError wraps a generic error with additional context
func WrapAPIError(operation string, err error) error {
	if err == nil {
		return nil
	}
	
	if bybitErr, ok := err.(*BybitError); ok {
		bybitErr.Details = fmt.Sprintf("Operation: %s", operation)
		return bybitErr
	}
	
	return fmt.Errorf("%s failed: %w", operation, err)
}

// ParseAPIError extracts error information from the API response
func ParseAPIError(retCode int, retMsg string) error {
	if retCode == 0 {
		return nil
	}
	
	return NewBybitError(retCode, retMsg)
}

// ErrorCodes maps common error codes to human-readable messages
var ErrorCodes = map[int]string{
	ErrCodeInvalidAPIKey:       "Invalid API key",
	ErrCodeInvalidSignature:    "Invalid signature",
	ErrCodeInvalidTimestamp:    "Invalid timestamp",
	ErrCodeInsufficientBalance: "Insufficient balance",
	ErrCodeOrderNotFound:       "Order not found",
	ErrCodeSymbolNotFound:      "Symbol not found",
	ErrCodeInvalidOrderType:    "Invalid order type",
	ErrCodeInvalidQuantity:     "Invalid quantity",
	ErrCodeInvalidPrice:        "Invalid price",
	ErrCodeRateLimitExceeded:   "Rate limit exceeded",
	ErrCodeMarketClosed:        "Market is closed",
}

// GetErrorDescription returns a human-readable description for an error code
func GetErrorDescription(code int) string {
	if desc, exists := ErrorCodes[code]; exists {
		return desc
	}
	return fmt.Sprintf("Unknown error code: %d", code)
}
