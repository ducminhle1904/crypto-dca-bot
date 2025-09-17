package portfolio

import (
	"fmt"
	"math"
)

// DefaultLeverageCalculator implements the LeverageCalculator interface
type DefaultLeverageCalculator struct {
	maxLeverage float64
	minLeverage float64
}

// NewLeverageCalculator creates a new leverage calculator with default limits
func NewLeverageCalculator() LeverageCalculator {
	return &DefaultLeverageCalculator{
		maxLeverage: 125.0, // Maximum leverage (Bybit limit)
		minLeverage: 1.0,   // Minimum leverage (spot trading)
	}
}

// NewLeverageCalculatorWithLimits creates a leverage calculator with custom limits
func NewLeverageCalculatorWithLimits(minLev, maxLev float64) LeverageCalculator {
	return &DefaultLeverageCalculator{
		maxLeverage: maxLev,
		minLeverage: minLev,
	}
}

// CalculateRequiredMargin calculates the margin required for a position with given leverage
// Formula: Required Margin = Position Value / Leverage
//
// Example: $100 position with 10x leverage = $10 margin required
func (c *DefaultLeverageCalculator) CalculateRequiredMargin(positionValue, leverage float64) float64 {
	if leverage <= 0 {
		return positionValue // If no leverage specified, require full amount
	}
	
	// Ensure leverage is within bounds
	if leverage > c.maxLeverage {
		leverage = c.maxLeverage
	}
	if leverage < c.minLeverage {
		leverage = c.minLeverage
	}
	
	return positionValue / leverage
}

// CalculateMaxPositionSize calculates the maximum position size for given available margin and leverage
// Formula: Max Position = Available Margin × Leverage
//
// Example: $50 margin with 10x leverage = $500 max position
func (c *DefaultLeverageCalculator) CalculateMaxPositionSize(availableMargin, leverage float64) float64 {
	if availableMargin <= 0 || leverage <= 0 {
		return 0
	}
	
	// Ensure leverage is within bounds
	if leverage > c.maxLeverage {
		leverage = c.maxLeverage
	}
	if leverage < c.minLeverage {
		leverage = c.minLeverage
	}
	
	return availableMargin * leverage
}

// ValidateLeverage validates if the leverage value is acceptable
func (c *DefaultLeverageCalculator) ValidateLeverage(leverage float64) error {
	if leverage <= 0 {
		return fmt.Errorf("leverage must be greater than 0, got: %.2f", leverage)
	}
	
	if leverage < c.minLeverage {
		return fmt.Errorf("leverage %.2f is below minimum allowed %.2f", leverage, c.minLeverage)
	}
	
	if leverage > c.maxLeverage {
		return fmt.Errorf("leverage %.2f exceeds maximum allowed %.2f", leverage, c.maxLeverage)
	}
	
	return nil
}

// GetEffectiveLeverage calculates the actual leverage based on position value and margin used
// Formula: Effective Leverage = Position Value / Margin Used
//
// This is useful for validation and monitoring actual leverage vs configured leverage
func (c *DefaultLeverageCalculator) GetEffectiveLeverage(positionValue, margin float64) float64 {
	if margin <= 0 {
		return 1.0 // No leverage if no margin used
	}
	
	return positionValue / margin
}

// LeverageHelper provides additional utility functions for leverage calculations
type LeverageHelper struct {
	calculator LeverageCalculator
}

// NewLeverageHelper creates a new leverage helper
func NewLeverageHelper(calc LeverageCalculator) *LeverageHelper {
	if calc == nil {
		calc = NewLeverageCalculator()
	}
	return &LeverageHelper{calculator: calc}
}

// CalculateMarginPercent calculates what percentage of position value is required as margin
// Formula: Margin Percent = 1 / Leverage
//
// Example: 10x leverage = 1/10 = 0.1 = 10% margin required
func (h *LeverageHelper) CalculateMarginPercent(leverage float64) float64 {
	if leverage <= 0 {
		return 1.0 // 100% margin if no leverage
	}
	return 1.0 / leverage
}

// CalculateLeverageFromMargin calculates leverage from margin percentage
// Formula: Leverage = 1 / Margin Percent
//
// Example: 10% margin = 1/0.1 = 10x leverage
func (h *LeverageHelper) CalculateLeverageFromMargin(marginPercent float64) float64 {
	if marginPercent <= 0 || marginPercent > 1.0 {
		return 1.0 // No leverage for invalid margin percentages
	}
	return 1.0 / marginPercent
}

// CalculateLiquidationPrice calculates approximate liquidation price for a leveraged position
// This is a simplified calculation - actual exchanges use more complex formulas
func (h *LeverageHelper) CalculateLiquidationPrice(entryPrice, leverage float64, isLong bool) float64 {
	if leverage <= 1 {
		return 0 // No liquidation for spot positions
	}
	
	// Simplified liquidation calculation
	// For long: liquidation = entry * (1 - 1/leverage * 0.9) 
	// For short: liquidation = entry * (1 + 1/leverage * 0.9)
	// 0.9 factor accounts for fees and safety margin
	
	marginPercent := 1.0 / leverage
	liquidationFactor := marginPercent * 0.9
	
	if isLong {
		return entryPrice * (1.0 - liquidationFactor)
	} else {
		return entryPrice * (1.0 + liquidationFactor)
	}
}

// CalculateMaxSafePositionSize calculates maximum position size with safety margin
// Applies a safety factor to leave some margin buffer
func (h *LeverageHelper) CalculateMaxSafePositionSize(availableMargin, leverage, safetyFactor float64) float64 {
	if safetyFactor <= 0 || safetyFactor > 1.0 {
		safetyFactor = 0.8 // Default 80% utilization
	}
	
	maxSize := h.calculator.CalculateMaxPositionSize(availableMargin, leverage)
	return maxSize * safetyFactor
}

// ValidatePositionSafety performs comprehensive safety checks for a leveraged position
type PositionSafety struct {
	IsValid              bool    `json:"is_valid"`
	RequiredMargin       float64 `json:"required_margin"`
	AvailableMargin      float64 `json:"available_margin"`
	MarginUtilization    float64 `json:"margin_utilization"`    // % of available margin used
	EffectiveLeverage    float64 `json:"effective_leverage"`
	LiquidationPrice     float64 `json:"liquidation_price"`
	DistanceToLiquidation float64 `json:"distance_to_liquidation"` // % price move to liquidation
	RiskLevel           string  `json:"risk_level"`              // LOW, MEDIUM, HIGH, CRITICAL
	Warnings            []string `json:"warnings"`
	Errors              []string `json:"errors"`
}

// ValidatePositionSafety performs comprehensive validation of a leveraged position
func (h *LeverageHelper) ValidatePositionSafety(
	positionValue, entryPrice, leverage, availableMargin float64, isLong bool) *PositionSafety {
	
	safety := &PositionSafety{
		IsValid:   true,
		Warnings:  make([]string, 0),
		Errors:    make([]string, 0),
	}
	
	// Calculate required margin
	safety.RequiredMargin = h.calculator.CalculateRequiredMargin(positionValue, leverage)
	safety.AvailableMargin = availableMargin
	
	// Check if sufficient margin available
	if safety.RequiredMargin > availableMargin {
		safety.IsValid = false
		safety.Errors = append(safety.Errors, 
			fmt.Sprintf("Insufficient margin: need $%.2f, have $%.2f", 
				safety.RequiredMargin, availableMargin))
	}
	
	// Calculate margin utilization
	if availableMargin > 0 {
		safety.MarginUtilization = safety.RequiredMargin / availableMargin
	}
	
	// Calculate effective leverage
	safety.EffectiveLeverage = h.calculator.GetEffectiveLeverage(positionValue, safety.RequiredMargin)
	
	// Calculate liquidation price and distance
	safety.LiquidationPrice = h.CalculateLiquidationPrice(entryPrice, leverage, isLong)
	if safety.LiquidationPrice > 0 && entryPrice > 0 {
		safety.DistanceToLiquidation = math.Abs(safety.LiquidationPrice-entryPrice) / entryPrice
	}
	
	// Determine risk level and add warnings
	safety.RiskLevel = h.determineRiskLevel(safety)
	h.addRiskWarnings(safety)
	
	return safety
}

// determineRiskLevel categorizes the risk level of a position
func (h *LeverageHelper) determineRiskLevel(safety *PositionSafety) string {
	if !safety.IsValid {
		return "CRITICAL"
	}
	
	// High risk indicators
	if safety.MarginUtilization > 0.9 || safety.EffectiveLeverage > 50 || safety.DistanceToLiquidation < 0.1 {
		return "HIGH"
	}
	
	// Medium risk indicators  
	if safety.MarginUtilization > 0.7 || safety.EffectiveLeverage > 20 || safety.DistanceToLiquidation < 0.2 {
		return "MEDIUM"
	}
	
	return "LOW"
}

// addRiskWarnings adds appropriate warnings based on position characteristics
func (h *LeverageHelper) addRiskWarnings(safety *PositionSafety) {
	if safety.MarginUtilization > 0.8 {
		safety.Warnings = append(safety.Warnings, 
			fmt.Sprintf("High margin utilization: %.1f%%", safety.MarginUtilization*100))
	}
	
	if safety.EffectiveLeverage > 25 {
		safety.Warnings = append(safety.Warnings, 
			fmt.Sprintf("High leverage: %.1fx", safety.EffectiveLeverage))
	}
	
	if safety.DistanceToLiquidation < 0.15 {
		safety.Warnings = append(safety.Warnings, 
			fmt.Sprintf("Close to liquidation: %.1f%% price move", safety.DistanceToLiquidation*100))
	}
}
