package spacing

import (
	"fmt"
	"math"
)

// FixedProgressiveSpacing implements traditional fixed progressive DCA spacing
// The classic approach: base_threshold * threshold_multiplier^level
// Simple, predictable, and reliable spacing progression
type FixedProgressiveSpacing struct {
	baseThreshold       float64 // Base threshold percentage (e.g., 0.01 = 1%)
	thresholdMultiplier float64 // Multiplier per DCA level (e.g., 1.15 = 15% increase per level)
	maxThreshold        float64 // Maximum allowed threshold (safety limit)
	minThreshold        float64 // Minimum allowed threshold (safety limit)
}

// NewFixedProgressiveSpacing creates a new fixed progressive spacing strategy
func NewFixedProgressiveSpacing(params map[string]interface{}) (*FixedProgressiveSpacing, error) {
	strategy := &FixedProgressiveSpacing{
		// Default parameters - matching existing behavior
		baseThreshold:       0.01,  // 1% base threshold
		thresholdMultiplier: 1.15,  // 1.15x multiplier per level
		maxThreshold:        0.20,  // 20% maximum threshold (safety)
		minThreshold:        0.001, // 0.1% minimum threshold (safety)
	}

	// Override with provided parameters
	if val, ok := params["base_threshold"].(float64); ok {
		strategy.baseThreshold = val
	}
	if val, ok := params["threshold_multiplier"].(float64); ok {
		strategy.thresholdMultiplier = val
	}
	if val, ok := params["max_threshold"].(float64); ok {
		strategy.maxThreshold = val
	}
	if val, ok := params["min_threshold"].(float64); ok {
		strategy.minThreshold = val
	}

	return strategy, nil
}

// CalculateThreshold calculates the fixed progressive threshold for the next DCA entry
func (s *FixedProgressiveSpacing) CalculateThreshold(level int, context *MarketContext) float64 {
	// Classic fixed progressive formula: base * multiplier^level
	threshold := s.baseThreshold
	
	if s.thresholdMultiplier > 1.0 && level > 0 {
		threshold = s.baseThreshold * math.Pow(s.thresholdMultiplier, float64(level))
	}

	// Apply safety bounds
	threshold = math.Max(threshold, s.minThreshold)
	threshold = math.Min(threshold, s.maxThreshold)

	return threshold
}

// GetName returns the strategy name
func (s *FixedProgressiveSpacing) GetName() string {
	return "Fixed Progressive"
}

// GetParameters returns the current strategy parameters
func (s *FixedProgressiveSpacing) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"base_threshold":       s.baseThreshold,
		"threshold_multiplier": s.thresholdMultiplier,
		"max_threshold":        s.maxThreshold,
		"min_threshold":        s.minThreshold,
	}
}

// ValidateConfig validates the strategy configuration
func (s *FixedProgressiveSpacing) ValidateConfig() error {
	if s.baseThreshold <= 0 || s.baseThreshold >= 1.0 {
		return fmt.Errorf("base_threshold must be between 0 and 1.0, got: %.6f", s.baseThreshold)
	}

	if s.thresholdMultiplier < 1.0 || s.thresholdMultiplier > 5.0 {
		return fmt.Errorf("threshold_multiplier must be between 1.0 and 5.0, got: %.3f", s.thresholdMultiplier)
	}

	if s.maxThreshold <= s.minThreshold {
		return fmt.Errorf("max_threshold (%.6f) must be greater than min_threshold (%.6f)", 
			s.maxThreshold, s.minThreshold)
	}

	if s.minThreshold <= 0 || s.maxThreshold >= 1.0 {
		return fmt.Errorf("thresholds must be between 0 and 1.0, got min: %.6f, max: %.6f", 
			s.minThreshold, s.maxThreshold)
	}

	return nil
}

// Reset resets the strategy state (called at cycle completion)
func (s *FixedProgressiveSpacing) Reset() {
	// Fixed spacing has no internal state to reset
}

// GetThresholdProgression returns the threshold progression for display/analysis
func (s *FixedProgressiveSpacing) GetThresholdProgression(maxLevels int) []float64 {
	if maxLevels <= 0 {
		maxLevels = 10 // Default to 10 levels for display
	}

	progression := make([]float64, maxLevels)
	for level := 0; level < maxLevels; level++ {
		// Use nil context since fixed spacing doesn't need market data
		progression[level] = s.CalculateThreshold(level, nil)
	}

	return progression
}

// GetProgressionDisplay returns a formatted string showing the threshold progression
func (s *FixedProgressiveSpacing) GetProgressionDisplay(maxLevels int) string {
	progression := s.GetThresholdProgression(maxLevels)
	
	var display string
	for i, threshold := range progression {
		if i > 0 {
			display += " → "
		}
		display += fmt.Sprintf("%.2f%%", threshold*100)
	}
	
	return display
}
