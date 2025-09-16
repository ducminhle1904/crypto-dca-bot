package spacing

import (
	"testing"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

func TestVolatilityAdaptiveSpacing_CalculateThreshold(t *testing.T) {
	// Create strategy with test parameters
	params := map[string]interface{}{
		"base_threshold":         0.01,  // 1%
		"volatility_sensitivity": 2.0,   // 2x sensitivity
		"atr_period":            3,      // Short period for testing
		"max_threshold":         0.05,   // 5% max
		"min_threshold":         0.003,  // 0.3% min
		"level_multiplier":      1.1,    // 1.1x per level
	}

	strategy, err := NewVolatilityAdaptiveSpacing(params)
	if err != nil {
		t.Fatalf("Failed to create strategy: %v", err)
	}

	// Create test market context with sample data
	testCandles := []types.OHLCV{
		{High: 100.0, Low: 95.0, Close: 98.0, Timestamp: time.Now()},
		{High: 102.0, Low: 97.0, Close: 99.0, Timestamp: time.Now()},
		{High: 101.0, Low: 96.0, Close: 97.0, Timestamp: time.Now()},
	}

	context := &MarketContext{
		CurrentPrice:   97.0,
		LastEntryPrice: 100.0,
		ATR:           0.0, // Will be calculated from candles
		CurrentCandle:  testCandles[2],
		RecentCandles:  testCandles,
		Timestamp:     time.Now(),
	}

	tests := []struct {
		name     string
		level    int
		expected struct {
			min float64 // Minimum expected threshold
			max float64 // Maximum expected threshold
		}
	}{
		{
			name:  "Level 0 - Base threshold",
			level: 0,
			expected: struct {
				min float64
				max float64
			}{min: 0.003, max: 0.02}, // Should be between min threshold and reasonable volatility adjustment
		},
		{
			name:  "Level 1 - Progressive multiplier",
			level: 1,
			expected: struct {
				min float64
				max float64
			}{min: 0.003, max: 0.025}, // Should be higher than level 0
		},
		{
			name:  "Level 3 - Higher level",
			level: 3,
			expected: struct {
				min float64
				max float64
			}{min: 0.003, max: 0.05}, // Should approach max threshold
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			threshold := strategy.CalculateThreshold(tt.level, context)

			if threshold < tt.expected.min {
				t.Errorf("Threshold %.6f is below minimum expected %.6f", threshold, tt.expected.min)
			}
			if threshold > tt.expected.max {
				t.Errorf("Threshold %.6f is above maximum expected %.6f", threshold, tt.expected.max)
			}

			t.Logf("Level %d: Threshold = %.4f%% (%.6f)", tt.level, threshold*100, threshold)
		})
	}
}

func TestVolatilityAdaptiveSpacing_ValidateConfig(t *testing.T) {
	validParams := map[string]interface{}{
		"base_threshold":         0.01,
		"volatility_sensitivity": 2.0,
		"atr_period":            14,
		"max_threshold":         0.05,
		"min_threshold":         0.003,
		"level_multiplier":      1.1,
	}

	strategy, err := NewVolatilityAdaptiveSpacing(validParams)
	if err != nil {
		t.Fatalf("Failed to create strategy with valid params: %v", err)
	}

	if err := strategy.ValidateConfig(); err != nil {
		t.Errorf("Valid configuration should not produce error: %v", err)
	}

	// Test invalid configurations
	invalidTests := []struct {
		name   string
		params map[string]interface{}
	}{
		{
			name: "Invalid base_threshold - too high",
			params: map[string]interface{}{
				"base_threshold": 1.5, // > 1.0
			},
		},
		{
			name: "Invalid max < min threshold",
			params: map[string]interface{}{
				"max_threshold": 0.001,
				"min_threshold": 0.002,
			},
		},
		{
			name: "Invalid level_multiplier - too high",
			params: map[string]interface{}{
				"level_multiplier": 3.0, // > 2.0
			},
		},
	}

	for _, tt := range invalidTests {
		t.Run(tt.name, func(t *testing.T) {
			// Start with valid params and override specific ones
			testParams := make(map[string]interface{})
			for k, v := range validParams {
				testParams[k] = v
			}
			for k, v := range tt.params {
				testParams[k] = v
			}

			testStrategy, err := NewVolatilityAdaptiveSpacing(testParams)
			if err != nil {
				t.Logf("Strategy creation failed as expected: %v", err)
				return
			}

			if err := testStrategy.ValidateConfig(); err == nil {
				t.Errorf("Expected validation error for %s, but got none", tt.name)
			} else {
				t.Logf("Validation failed as expected: %v", err)
			}
		})
	}
}

func TestVolatilityAdaptiveSpacing_GetName(t *testing.T) {
	params := map[string]interface{}{}
	strategy, err := NewVolatilityAdaptiveSpacing(params)
	if err != nil {
		t.Fatalf("Failed to create strategy: %v", err)
	}

	name := strategy.GetName()
	expected := "Volatility-Adaptive (ATR)"
	
	if name != expected {
		t.Errorf("Expected name '%s', got '%s'", expected, name)
	}
}

func TestVolatilityAdaptiveSpacing_Parameters(t *testing.T) {
	params := map[string]interface{}{
		"base_threshold":         0.015,
		"volatility_sensitivity": 1.5,
		"atr_period":            21,
		"max_threshold":         0.06,
		"min_threshold":         0.004,
		"level_multiplier":      1.2,
	}

	strategy, err := NewVolatilityAdaptiveSpacing(params)
	if err != nil {
		t.Fatalf("Failed to create strategy: %v", err)
	}

	retrievedParams := strategy.GetParameters()

	// Check that all parameters are correctly set and retrieved
	expectedChecks := map[string]float64{
		"base_threshold":         0.015,
		"volatility_sensitivity": 1.5,
		"max_threshold":         0.06,
		"min_threshold":         0.004,
		"level_multiplier":      1.2,
	}

	for key, expectedValue := range expectedChecks {
		if value, ok := retrievedParams[key].(float64); ok {
			if value != expectedValue {
				t.Errorf("Parameter %s: expected %.3f, got %.3f", key, expectedValue, value)
			}
		} else {
			t.Errorf("Parameter %s not found or wrong type", key)
		}
	}

	// Check integer parameter
	if period, ok := retrievedParams["atr_period"].(int); ok {
		if period != 21 {
			t.Errorf("ATR period: expected 21, got %d", period)
		}
	} else {
		t.Errorf("ATR period not found or wrong type")
	}
}

func TestVolatilityAdaptiveSpacing_DCALevelProgression(t *testing.T) {
	// Test that DCA level progression works correctly
	params := map[string]interface{}{
		"base_threshold":         0.01,  // 1%
		"volatility_sensitivity": 2.0,   // 2x sensitivity
		"atr_period":            3,      // Short period for testing
		"max_threshold":         0.05,   // 5% max
		"min_threshold":         0.003,  // 0.3% min
		"level_multiplier":      1.1,    // 1.1x per level
	}

	strategy, err := NewVolatilityAdaptiveSpacing(params)
	if err != nil {
		t.Fatalf("Failed to create strategy: %v", err)
	}

	// Create test market context with consistent data
	testCandles := []types.OHLCV{
		{High: 100.0, Low: 95.0, Close: 98.0, Timestamp: time.Now()},
		{High: 102.0, Low: 97.0, Close: 99.0, Timestamp: time.Now()},
		{High: 101.0, Low: 96.0, Close: 97.0, Timestamp: time.Now()},
	}

	context := &MarketContext{
		CurrentPrice:   97.0,
		LastEntryPrice: 100.0,
		ATR:           2.0, // Fixed ATR for consistent testing
		CurrentCandle:  testCandles[2],
		RecentCandles: testCandles,
		Timestamp:     time.Now(),
	}

	// Test progressive threshold increase with DCA levels
	thresholds := make([]float64, 5)
	for level := 0; level < 5; level++ {
		thresholds[level] = strategy.CalculateThreshold(level, context)
	}

	// Verify that thresholds increase with DCA level (progressive spacing)
	for i := 1; i < len(thresholds); i++ {
		if thresholds[i] <= thresholds[i-1] {
			t.Errorf("Threshold should increase with DCA level: Level %d (%.6f) should be > Level %d (%.6f)", 
				i, thresholds[i], i-1, thresholds[i-1])
		}
	}

	// Verify that all thresholds are within bounds
	for level, threshold := range thresholds {
		if threshold < 0.003 || threshold > 0.05 {
			t.Errorf("Level %d threshold %.6f is outside bounds [0.003, 0.05]", level, threshold)
		}
	}

	// Log the progression for verification
	t.Logf("DCA Level Progression:")
	for level, threshold := range thresholds {
		t.Logf("  Level %d: %.4f%% (%.6f)", level, threshold*100, threshold)
	}
}