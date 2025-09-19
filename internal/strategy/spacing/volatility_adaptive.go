package spacing

import (
	"fmt"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/base"
)

// VolatilityAdaptiveSpacing implements DCA spacing based on market volatility (ATR)
// The concept: Higher volatility = wider spacing, Lower volatility = tighter spacing
// This prevents overtrading in choppy markets and captures more opportunities in trending markets
type VolatilityAdaptiveSpacing struct {
	baseThreshold         float64 // Base threshold percentage (e.g., 0.01 = 1%)
	volatilitySensitivity float64 // How much volatility affects spacing (e.g., 2.0)
	atrPeriod            int     // ATR calculation period (e.g., 14)
	maxThreshold         float64 // Maximum allowed threshold (e.g., 0.05 = 5%)
	minThreshold         float64 // Minimum allowed threshold (e.g., 0.003 = 0.3%)
	levelMultiplier      float64 // Progressive multiplier per DCA level (e.g., 1.1)

	// ATR calculator for volatility measurement
	atrCalculator *base.ATR
}

// NewVolatilityAdaptiveSpacing creates a new volatility-adaptive spacing strategy
func NewVolatilityAdaptiveSpacing(params map[string]interface{}) (*VolatilityAdaptiveSpacing, error) {
	strategy := &VolatilityAdaptiveSpacing{
		// Default parameters - optimized for crypto markets
		baseThreshold:         0.01,  // 1% base threshold
		volatilitySensitivity: 2.0,   // 2x volatility sensitivity
		atrPeriod:            14,     // 14-period ATR
		maxThreshold:         0.05,   // 5% maximum threshold
		minThreshold:         0.003,  // 0.3% minimum threshold
		levelMultiplier:      1.1,    // 1.1x progressive multiplier
	}

	// Override with provided parameters
	if val, ok := params["base_threshold"].(float64); ok {
		strategy.baseThreshold = val
	}
	if val, ok := params["volatility_sensitivity"].(float64); ok {
		strategy.volatilitySensitivity = val
	}
	if val, ok := params["atr_period"].(int); ok {
		strategy.atrPeriod = val
	}
	if val, ok := params["max_threshold"].(float64); ok {
		strategy.maxThreshold = val
	}
	if val, ok := params["min_threshold"].(float64); ok {
		strategy.minThreshold = val
	}
	if val, ok := params["level_multiplier"].(float64); ok {
		strategy.levelMultiplier = val
	}

	// Create ATR calculator
	strategy.atrCalculator = base.NewATR(strategy.atrPeriod)

	return strategy, nil
}

// CalculateThreshold calculates the volatility-adaptive threshold for the next DCA entry
func (s *VolatilityAdaptiveSpacing) CalculateThreshold(level int, context *MarketContext) float64 {
	// Use ATR provided in context (calculated by strategy's ATR calculator)
	atr := context.ATR
	
	// If no ATR available, use base threshold with level multiplier
	if atr <= 0 || context.CurrentPrice <= 0 {
		return s.applyLevelMultiplier(s.baseThreshold, level)
	}

	// Calculate normalized volatility (ATR as percentage of price)
	normalizedVolatility := atr / context.CurrentPrice

	// Calculate volatility-adjusted base threshold
	// Formula: baseThreshold * (0.5 + normalizedVolatility * sensitivity)
	// This means:
	// - Low volatility (ATR=0.5% of price) → threshold = base * (0.5 + 0.005 * 2) = base * 0.51
	// - High volatility (ATR=3% of price) → threshold = base * (0.5 + 0.03 * 2) = base * 1.1
	volatilityMultiplier := 0.5 + normalizedVolatility*s.volatilitySensitivity
	adaptiveBaseThreshold := s.baseThreshold * volatilityMultiplier

	// Apply level-based progressive multiplier with growth cap
	finalThreshold := s.applyLevelMultiplierCapped(adaptiveBaseThreshold, level)

	// Enforce min/max bounds with reasonable maximum for volatile markets
	reasonableMax := math.Min(s.maxThreshold, 0.06) // Never exceed 6% for adaptive strategy
	finalThreshold = math.Max(finalThreshold, s.minThreshold)
	finalThreshold = math.Min(finalThreshold, reasonableMax)

	return finalThreshold
}

// applyLevelMultiplier applies the progressive multiplier based on DCA level (legacy)
func (s *VolatilityAdaptiveSpacing) applyLevelMultiplier(baseThreshold float64, level int) float64 {
	if s.levelMultiplier <= 1.0 || level == 0 {
		return baseThreshold
	}

	// Apply exponential multiplier: base * multiplier^level
	return baseThreshold * math.Pow(s.levelMultiplier, float64(level))
}

// applyLevelMultiplierCapped applies progressive multiplier with growth cap to prevent unreachable thresholds
func (s *VolatilityAdaptiveSpacing) applyLevelMultiplierCapped(baseThreshold float64, level int) float64 {
	if s.levelMultiplier <= 1.0 || level == 0 {
		return baseThreshold
	}

	// Use exponential growth for first few levels, then linear to prevent extreme thresholds
	if level <= 3 {
		// Exponential for levels 1-3: good initial spacing
		return baseThreshold * math.Pow(s.levelMultiplier, float64(level))
	} else {
		// Linear growth for levels 4+: prevents unreachable thresholds
		baseForLevel3 := baseThreshold * math.Pow(s.levelMultiplier, 3.0)
		additionalLevels := float64(level - 3)
		linearIncrement := baseForLevel3 * 0.15 // 15% increase per level after level 3
		return baseForLevel3 + (additionalLevels * linearIncrement)
	}
}

// GetName returns the strategy name
func (s *VolatilityAdaptiveSpacing) GetName() string {
	return "Volatility-Adaptive (ATR)"
}

// GetParameters returns the current strategy parameters
func (s *VolatilityAdaptiveSpacing) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"base_threshold":         s.baseThreshold,
		"volatility_sensitivity": s.volatilitySensitivity,
		"atr_period":            s.atrPeriod,
		"max_threshold":         s.maxThreshold,
		"min_threshold":         s.minThreshold,
		"level_multiplier":      s.levelMultiplier,
	}
}

// ValidateConfig validates the strategy configuration
func (s *VolatilityAdaptiveSpacing) ValidateConfig() error {
	if s.baseThreshold <= 0 || s.baseThreshold >= 1.0 {
		return fmt.Errorf("base_threshold must be between 0 and 1.0, got: %.6f", s.baseThreshold)
	}

	if s.volatilitySensitivity < 0 || s.volatilitySensitivity > 10 {
		return fmt.Errorf("volatility_sensitivity must be between 0 and 10, got: %.2f", s.volatilitySensitivity)
	}

	if s.atrPeriod < 1 || s.atrPeriod > 100 {
		return fmt.Errorf("atr_period must be between 1 and 100, got: %d", s.atrPeriod)
	}

	if s.maxThreshold <= s.minThreshold {
		return fmt.Errorf("max_threshold (%.6f) must be greater than min_threshold (%.6f)", 
			s.maxThreshold, s.minThreshold)
	}

	if s.minThreshold <= 0 || s.maxThreshold >= 1.0 {
		return fmt.Errorf("thresholds must be between 0 and 1.0, got min: %.6f, max: %.6f", 
			s.minThreshold, s.maxThreshold)
	}

	if s.levelMultiplier < 1.0 || s.levelMultiplier > 2.0 {
		return fmt.Errorf("level_multiplier must be between 1.0 and 2.0, got: %.3f", s.levelMultiplier)
	}

	return nil
}

// Reset resets the strategy state (called at cycle completion)
func (s *VolatilityAdaptiveSpacing) Reset() {
	// Reset ATR calculator state to start fresh for next cycle
	if s.atrCalculator != nil {
		s.atrCalculator.ResetState()
	}
}

// GetThresholdBreakdown returns detailed threshold calculation for debugging/analysis
func (s *VolatilityAdaptiveSpacing) GetThresholdBreakdown(level int, context *MarketContext) map[string]interface{} {
	atr := context.ATR
	if atr <= 0 && len(context.RecentCandles) >= s.atrPeriod {
		if atrValue, err := s.atrCalculator.Calculate(context.RecentCandles); err == nil {
			atr = atrValue
		}
	}

	normalizedVolatility := 0.0
	if context.CurrentPrice > 0 && atr > 0 {
		normalizedVolatility = atr / context.CurrentPrice
	}

	volatilityMultiplier := 0.5 + normalizedVolatility*s.volatilitySensitivity
	adaptiveBaseThreshold := s.baseThreshold * volatilityMultiplier
	levelAdjustedThreshold := s.applyLevelMultiplier(adaptiveBaseThreshold, level)
	
	finalThreshold := levelAdjustedThreshold
	finalThreshold = math.Max(finalThreshold, s.minThreshold)
	finalThreshold = math.Min(finalThreshold, s.maxThreshold)

	return map[string]interface{}{
		"atr":                      atr,
		"normalized_volatility":    normalizedVolatility * 100, // As percentage
		"volatility_multiplier":    volatilityMultiplier,
		"adaptive_base_threshold":  adaptiveBaseThreshold * 100, // As percentage
		"level_adjusted_threshold": levelAdjustedThreshold * 100, // As percentage
		"final_threshold":          finalThreshold * 100,        // As percentage
		"level":                    level,
		"capped_at_min":           finalThreshold == s.minThreshold,
		"capped_at_max":           finalThreshold == s.maxThreshold,
	}
}
