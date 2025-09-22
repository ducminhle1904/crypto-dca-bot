package safety

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Valid   bool
	Message string
	Code    string
}

// Validator provides defensive validation methods
type Validator struct{}

// NewValidator creates a new validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// ValidatePrice validates a price value for trading
func (v *Validator) ValidatePrice(price float64, symbol string) ValidationResult {
	if price <= 0 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("invalid price %.8f for %s: price must be positive", price, symbol),
			Code:    "INVALID_PRICE_NEGATIVE",
		}
	}
	
	if math.IsNaN(price) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("invalid price for %s: price is NaN", symbol),
			Code:    "INVALID_PRICE_NAN",
		}
	}
	
	if math.IsInf(price, 0) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("invalid price for %s: price is infinite", symbol),
			Code:    "INVALID_PRICE_INF",
		}
	}
	
	// Check for reasonable price bounds (prevent obvious data errors)
	if price > 1e10 { // $10 billion per unit seems unreasonable
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("suspicious price %.8f for %s: exceeds reasonable bounds", price, symbol),
			Code:    "PRICE_OUT_OF_BOUNDS",
		}
	}
	
	if price < 1e-8 { // Less than 1 satoshi equivalent
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("suspicious price %.8f for %s: below reasonable bounds", price, symbol),
			Code:    "PRICE_TOO_SMALL",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateQuantity validates a quantity value for trading
func (v *Validator) ValidateQuantity(quantity float64, symbol string) ValidationResult {
	if quantity <= 0 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("invalid quantity %.8f for %s: quantity must be positive", quantity, symbol),
			Code:    "INVALID_QUANTITY_NEGATIVE",
		}
	}
	
	if math.IsNaN(quantity) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("invalid quantity for %s: quantity is NaN", symbol),
			Code:    "INVALID_QUANTITY_NAN",
		}
	}
	
	if math.IsInf(quantity, 0) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("invalid quantity for %s: quantity is infinite", symbol),
			Code:    "INVALID_QUANTITY_INF",
		}
	}
	
	// Check for reasonable quantity bounds
	if quantity > 1e12 { // 1 trillion units seems excessive
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("suspicious quantity %.8f for %s: exceeds reasonable bounds", quantity, symbol),
			Code:    "QUANTITY_OUT_OF_BOUNDS",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateOrderValue validates the total value of an order
func (v *Validator) ValidateOrderValue(price, quantity float64, symbol string) ValidationResult {
	// First validate individual components
	if priceResult := v.ValidatePrice(price, symbol); !priceResult.Valid {
		return priceResult
	}
	
	if quantityResult := v.ValidateQuantity(quantity, symbol); !quantityResult.Valid {
		return quantityResult
	}
	
	orderValue := price * quantity
	
	if math.IsNaN(orderValue) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("invalid order value for %s: calculation resulted in NaN", symbol),
			Code:    "INVALID_ORDER_VALUE_NAN",
		}
	}
	
	if math.IsInf(orderValue, 0) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("invalid order value for %s: calculation resulted in infinity", symbol),
			Code:    "INVALID_ORDER_VALUE_INF",
		}
	}
	
	// Check for reasonable order value bounds
	if orderValue > 1e9 { // $1 billion order seems excessive for DCA bot
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("suspicious order value $%.2f for %s: exceeds reasonable bounds", orderValue, symbol),
			Code:    "ORDER_VALUE_TOO_LARGE",
		}
	}
	
	if orderValue < 0.01 { // Less than 1 cent
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("order value $%.8f for %s: below minimum reasonable value", orderValue, symbol),
			Code:    "ORDER_VALUE_TOO_SMALL",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateSymbol validates a trading symbol format
func (v *Validator) ValidateSymbol(symbol string) ValidationResult {
	if symbol == "" {
		return ValidationResult{
			Valid:   false,
			Message: "symbol cannot be empty",
			Code:    "SYMBOL_EMPTY",
		}
	}
	
	// Basic format validation
	symbol = strings.TrimSpace(symbol)
	if len(symbol) < 3 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("symbol '%s' too short: minimum 3 characters required", symbol),
			Code:    "SYMBOL_TOO_SHORT",
		}
	}
	
	if len(symbol) > 20 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("symbol '%s' too long: maximum 20 characters allowed", symbol),
			Code:    "SYMBOL_TOO_LONG",
		}
	}
	
	// Check for valid characters (alphanumeric only)
	for _, char := range symbol {
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
			return ValidationResult{
				Valid:   false,
				Message: fmt.Sprintf("symbol '%s' contains invalid characters: only alphanumeric allowed", symbol),
				Code:    "SYMBOL_INVALID_CHARS",
			}
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateDCALevel validates a DCA level value
func (v *Validator) ValidateDCALevel(level int) ValidationResult {
	if level < 0 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("DCA level %d cannot be negative", level),
			Code:    "DCA_LEVEL_NEGATIVE",
		}
	}
	
	if level > 100 { // Reasonable upper bound for DCA levels
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("DCA level %d exceeds reasonable maximum (100)", level),
			Code:    "DCA_LEVEL_TOO_HIGH",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateTPPercentage validates a take profit percentage
func (v *Validator) ValidateTPPercentage(percentage float64) ValidationResult {
	if percentage <= 0 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("TP percentage %.4f must be positive", percentage),
			Code:    "TP_PERCENTAGE_NON_POSITIVE",
		}
	}
	
	if math.IsNaN(percentage) {
		return ValidationResult{
			Valid:   false,
			Message: "TP percentage is NaN",
			Code:    "TP_PERCENTAGE_NAN",
		}
	}
	
	if percentage > 1.0 { // 100% profit seems excessive
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("TP percentage %.4f exceeds 100%% - this seems excessive", percentage),
			Code:    "TP_PERCENTAGE_TOO_HIGH",
		}
	}
	
	if percentage < 0.0001 { // 0.01% seems too low
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("TP percentage %.4f is too low - minimum 0.01%% recommended", percentage),
			Code:    "TP_PERCENTAGE_TOO_LOW",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateTimeInterval validates a time interval string
func (v *Validator) ValidateTimeInterval(interval string) ValidationResult {
	if interval == "" {
		return ValidationResult{
			Valid:   false,
			Message: "time interval cannot be empty",
			Code:    "INTERVAL_EMPTY",
		}
	}
	
	// Supported intervals
	validIntervals := map[string]bool{
		"1m": true, "3m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "2h": true, "4h": true, "6h": true, "8h": true, "12h": true,
		"1d": true, "3d": true, "1w": true, "1M": true,
		"1": true, "3": true, "5": true, "15": true, "30": true, "60": true,
		"240": true, "D": true,
	}
	
	if !validIntervals[interval] {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("unsupported time interval '%s'", interval),
			Code:    "INTERVAL_UNSUPPORTED",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateBalance validates an account balance
func (v *Validator) ValidateBalance(balance float64, currency string) ValidationResult {
	if balance < 0 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("balance %.8f %s cannot be negative", balance, currency),
			Code:    "BALANCE_NEGATIVE",
		}
	}
	
	if math.IsNaN(balance) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("balance for %s is NaN", currency),
			Code:    "BALANCE_NAN",
		}
	}
	
	if math.IsInf(balance, 0) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("balance for %s is infinite", currency),
			Code:    "BALANCE_INF",
		}
	}
	
	return ValidationResult{Valid: true}
}

// SafeDivision performs division with zero-check
func (v *Validator) SafeDivision(dividend, divisor float64) (float64, error) {
	if divisor == 0 {
		return 0, fmt.Errorf("division by zero: %.8f / %.8f", dividend, divisor)
	}
	
	if math.IsNaN(dividend) || math.IsNaN(divisor) {
		return 0, fmt.Errorf("division with NaN: %.8f / %.8f", dividend, divisor)
	}
	
	if math.IsInf(dividend, 0) || math.IsInf(divisor, 0) {
		return 0, fmt.Errorf("division with infinity: %.8f / %.8f", dividend, divisor)
	}
	
	result := dividend / divisor
	
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, fmt.Errorf("division resulted in invalid value: %.8f / %.8f = %.8f", 
			dividend, divisor, result)
	}
	
	return result, nil
}

// SafeMultiplication performs multiplication with overflow check
func (v *Validator) SafeMultiplication(a, b float64) (float64, error) {
	if math.IsNaN(a) || math.IsNaN(b) {
		return 0, fmt.Errorf("multiplication with NaN: %.8f * %.8f", a, b)
	}
	
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return 0, fmt.Errorf("multiplication with infinity: %.8f * %.8f", a, b)
	}
	
	result := a * b
	
	if math.IsNaN(result) || math.IsInf(result, 0) {
		return 0, fmt.Errorf("multiplication resulted in invalid value: %.8f * %.8f = %.8f", 
			a, b, result)
	}
	
	return result, nil
}

// ValidateOrderID validates an order ID format
func (v *Validator) ValidateOrderID(orderID string) ValidationResult {
	if orderID == "" {
		return ValidationResult{
			Valid:   false,
			Message: "order ID cannot be empty",
			Code:    "ORDER_ID_EMPTY",
		}
	}
	
	orderID = strings.TrimSpace(orderID)
	if len(orderID) < 5 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("order ID '%s' too short: minimum 5 characters expected", orderID),
			Code:    "ORDER_ID_TOO_SHORT",
		}
	}
	
	if len(orderID) > 100 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("order ID '%s' too long: maximum 100 characters allowed", orderID),
			Code:    "ORDER_ID_TOO_LONG",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateTimestamp validates a timestamp for reasonable bounds
func (v *Validator) ValidateTimestamp(timestamp time.Time, context string) ValidationResult {
	now := time.Now()
	
	// Check if timestamp is too far in the past (more than 1 year)
	if timestamp.Before(now.AddDate(-1, 0, 0)) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("%s timestamp %v is too old (more than 1 year ago)", context, timestamp),
			Code:    "TIMESTAMP_TOO_OLD",
		}
	}
	
	// Check if timestamp is in the future (more than 1 hour ahead)
	if timestamp.After(now.Add(time.Hour)) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("%s timestamp %v is too far in the future", context, timestamp),
			Code:    "TIMESTAMP_FUTURE",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidatePercentageRange validates a percentage is within expected bounds
func (v *Validator) ValidatePercentageRange(percentage float64, min, max float64, context string) ValidationResult {
	if math.IsNaN(percentage) {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("%s percentage is NaN", context),
			Code:    "PERCENTAGE_NAN",
		}
	}
	
	if percentage < min {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("%s percentage %.4f below minimum %.4f", context, percentage, min),
			Code:    "PERCENTAGE_BELOW_MIN",
		}
	}
	
	if percentage > max {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("%s percentage %.4f above maximum %.4f", context, percentage, max),
			Code:    "PERCENTAGE_ABOVE_MAX",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidateStringNotEmpty validates that a string field is not empty
func (v *Validator) ValidateStringNotEmpty(value, fieldName string) ValidationResult {
	if strings.TrimSpace(value) == "" {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("%s cannot be empty", fieldName),
			Code:    "STRING_EMPTY",
		}
	}
	
	return ValidationResult{Valid: true}
}

// ValidatePositiveInteger validates that a value is a positive integer
func (v *Validator) ValidatePositiveInteger(value int, fieldName string) ValidationResult {
	if value <= 0 {
		return ValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("%s must be positive, got %d", fieldName, value),
			Code:    "INTEGER_NOT_POSITIVE",
		}
	}
	
	return ValidationResult{Valid: true}
}
