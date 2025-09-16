package spacing

import (
	"fmt"
	"strings"
)

// CreateSpacingStrategy creates a DCA spacing strategy based on configuration
func CreateSpacingStrategy(config SpacingConfig) (DCASpacingStrategy, error) {
	strategyName := strings.ToLower(strings.TrimSpace(config.Strategy))
	
	switch strategyName {
	case "volatility_adaptive", "atr":
		return NewVolatilityAdaptiveSpacing(config.Parameters)
	
	case "fixed", "fixed_progressive", "":
		return NewFixedProgressiveSpacing(config.Parameters)
	
	default:
		return nil, fmt.Errorf("unknown spacing strategy: %s (supported: volatility_adaptive, fixed)", config.Strategy)
	}
}

// GetAvailableStrategies returns a list of available spacing strategies
func GetAvailableStrategies() []string {
	return []string{
		"fixed",              // Fixed progressive spacing (default)
		"volatility_adaptive", // ATR-based adaptive spacing
	}
}

// GetStrategyDescription returns a description of the specified strategy
func GetStrategyDescription(strategyName string) string {
	switch strings.ToLower(strings.TrimSpace(strategyName)) {
	case "fixed":
		return "Fixed progressive spacing - consistent threshold progression per DCA level"
	case "volatility_adaptive", "atr":
		return "ATR-based volatility-adaptive spacing - wider spacing in volatile markets, tighter in stable markets"
	default:
		return "Unknown strategy"
	}
}

// GetDefaultParameters returns default parameters for a strategy
func GetDefaultParameters(strategyName string) map[string]interface{} {
	switch strings.ToLower(strings.TrimSpace(strategyName)) {
	case "volatility_adaptive", "atr":
		return map[string]interface{}{
			"base_threshold":         0.01,  // 1%
			"volatility_sensitivity": 2.0,   // 2x sensitivity
			"atr_period":            14,     // 14-period ATR
			"max_threshold":         0.05,   // 5% max
			"min_threshold":         0.003,  // 0.3% min
			"level_multiplier":      1.1,    // 1.1x per level
		}
		case "fixed":
		return map[string]interface{}{
			"base_threshold":       0.01, // 1%
			"threshold_multiplier": 1.15, // 1.15x per level
		}
	default:
		return map[string]interface{}{}
	}
}
