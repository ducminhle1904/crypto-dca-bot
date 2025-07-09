package risk

import (
	"fmt"
)

// RiskManagerImpl implements risk management logic
type RiskManagerImpl struct {
	maxPositionSize float64
	maxMultiplier   float64
}

// NewRiskManager creates a new risk manager instance
func NewRiskManager(maxMultiplier float64) RiskManager {
	return &RiskManagerImpl{
		maxPositionSize: 3000.0, // Default max position size
		maxMultiplier:   maxMultiplier,
	}
}

// ValidateOrder validates if an order meets risk management criteria
func (rm *RiskManagerImpl) ValidateOrder(order *Order, portfolio *Portfolio) error {
	// Check if we have sufficient balance
	if order.Amount > portfolio.Balance {
		return fmt.Errorf("insufficient balance: required %.2f, available %.2f", order.Amount, portfolio.Balance)
	}

	// Check position size limits
	if order.Amount > rm.maxPositionSize {
		return fmt.Errorf("order amount %.2f exceeds maximum position size %.2f", order.Amount, rm.maxPositionSize)
	}

	// Check if order amount is reasonable relative to balance
	balancePercentage := (order.Amount / portfolio.Balance) * 100
	if balancePercentage > 50 { // Don't use more than 50% of balance in one trade
		return fmt.Errorf("order amount %.2f%% of balance exceeds 50%% limit", balancePercentage)
	}

	return nil
}

// CalculatePositionSize calculates the appropriate position size based on signal strength
func (rm *RiskManagerImpl) CalculatePositionSize(signal Signal, balance float64) float64 {
	// Base position size is 10% of balance
	baseSize := balance * 0.1

	// Adjust based on signal strength
	adjustedSize := baseSize * (1 + signal.Strength)

	// Apply maximum multiplier
	if adjustedSize > baseSize*rm.maxMultiplier {
		adjustedSize = baseSize * rm.maxMultiplier
	}

	// Ensure we don't exceed maximum position size
	if adjustedSize > rm.maxPositionSize {
		adjustedSize = rm.maxPositionSize
	}

	// Ensure we don't exceed available balance
	if adjustedSize > balance {
		adjustedSize = balance
	}

	return adjustedSize
}

// ShouldStopTrading determines if trading should be stopped based on portfolio state
func (rm *RiskManagerImpl) ShouldStopTrading(portfolio *Portfolio) bool {
	// Stop trading if balance is too low
	if portfolio.Balance < 100 {
		return true
	}

	return false
}
