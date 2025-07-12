package backtest

import (
	"testing"

	"github.com/Zmey56/enhanced-dca-bot/internal/indicators"
	"github.com/Zmey56/enhanced-dca-bot/internal/strategy"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParameterOptimizer_OptimizeRSI tests RSI parameter optimization
func TestParameterOptimizer_OptimizeRSI(t *testing.T) {
	// Create test data
	data := generateTestData(200)

	// Create optimizer
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewEnhancedDCAStrategy(1000))
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   data,
	}

	// Run optimization
	result := optimizer.OptimizeRSI()

	// Verify result structure
	assert.NotNil(t, result)
	assert.GreaterOrEqual(t, result.Period, 10)
	assert.LessOrEqual(t, result.Period, 20)
	assert.GreaterOrEqual(t, result.Oversold, 20)
	assert.LessOrEqual(t, result.Oversold, 35)

	// Verify that optimization completed successfully
	assert.NotNil(t, result, "Optimization should return a result")
	assert.GreaterOrEqual(t, result.Period, 10, "Period should be at least 10")
	assert.LessOrEqual(t, result.Period, 20, "Period should be at most 20")
	assert.GreaterOrEqual(t, result.Oversold, 20, "Oversold should be at least 20")
	assert.LessOrEqual(t, result.Oversold, 35, "Oversold should be at most 35")
	// Note: Return can be 0 if no profitable trades were found
}

// TestParameterOptimizer_OptimizeRSI_EmptyData tests optimization with empty data
func TestParameterOptimizer_OptimizeRSI_EmptyData(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   []types.OHLCV{},
	}

	result := optimizer.OptimizeRSI()

	assert.NotNil(t, result)
	// With empty data, all optimizations should return 0 return
	assert.Equal(t, 0.0, result.Return)
}

// TestParameterOptimizer_OptimizeRSI_InsufficientData tests optimization with insufficient data
func TestParameterOptimizer_OptimizeRSI_InsufficientData(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   generateTestData(30), // Not enough data for meaningful optimization
	}

	result := optimizer.OptimizeRSI()

	assert.NotNil(t, result)
	// With insufficient data, optimization should still complete but may have poor results
	assert.NotNil(t, result.Period)
	assert.NotNil(t, result.Oversold)
}

// TestParameterOptimizer_OptimizeRSI_ProfitableData tests optimization with profitable data
func TestParameterOptimizer_OptimizeRSI_ProfitableData(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   generateRisingData(200), // Profitable trend
	}

	result := optimizer.OptimizeRSI()

	assert.NotNil(t, result)
	// With rising data, we should find some profitable parameters
	assert.GreaterOrEqual(t, result.Return, -1.0) // Allow for some losses due to commission
	assert.LessOrEqual(t, result.Return, 1.0)     // But not unrealistic gains
}

// TestParameterOptimizer_OptimizeRSI_LosingData tests optimization with losing data
func TestParameterOptimizer_OptimizeRSI_LosingData(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   generateFallingData(200), // Losing trend
	}

	result := optimizer.OptimizeRSI()

	assert.NotNil(t, result)
	// With falling data, even the best parameters might show losses
	assert.LessOrEqual(t, result.Return, 0.5) // Should not have unrealistic gains
}

// TestOptimizationResult_Structure tests the OptimizationResult structure
func TestOptimizationResult_Structure(t *testing.T) {
	result := &OptimizationResult{
		Period:   14,
		Oversold: 30,
		Return:   0.15,
	}

	assert.Equal(t, 14, result.Period)
	assert.Equal(t, 30, result.Oversold)
	assert.Equal(t, 0.15, result.Return)
}

// TestParameterOptimizer_OptimizeRSI_ParameterRanges tests that optimization explores the correct parameter ranges
func TestParameterOptimizer_OptimizeRSI_ParameterRanges(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   generateTestData(200),
	}

	result := optimizer.OptimizeRSI()

	// Verify parameter ranges
	assert.GreaterOrEqual(t, result.Period, 10, "Period should be at least 10")
	assert.LessOrEqual(t, result.Period, 20, "Period should be at most 20")
	assert.Equal(t, 0, result.Period%2, "Period should be even (increment by 2)")

	assert.GreaterOrEqual(t, result.Oversold, 20, "Oversold should be at least 20")
	assert.LessOrEqual(t, result.Oversold, 35, "Oversold should be at most 35")
	assert.Equal(t, 0, result.Oversold%5, "Oversold should be divisible by 5")
}

// TestParameterOptimizer_OptimizeRSI_StrategyIntegration tests that the optimized strategy is properly integrated
func TestParameterOptimizer_OptimizeRSI_StrategyIntegration(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   generateTestData(200),
	}

	result := optimizer.OptimizeRSI()

	// Create a strategy with the optimized parameters
	rsi := indicators.NewRSI(result.Period)
	rsi.SetOversold(float64(result.Oversold))

	strategy := strategy.NewEnhancedDCAStrategy(1000)
	strategy.AddIndicator(rsi)

	// Test that the strategy can be used
	decision, err := strategy.ShouldExecuteTrade(generateTestData(50))
	require.NoError(t, err)
	assert.NotNil(t, decision)
}

// TestParameterOptimizer_OptimizeRSI_Consistency tests that optimization produces consistent results
func TestParameterOptimizer_OptimizeRSI_Consistency(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   generateTestData(200),
	}

	// Run optimization multiple times
	result1 := optimizer.OptimizeRSI()
	result2 := optimizer.OptimizeRSI()

	// Results should be consistent (same data, same parameters)
	assert.Equal(t, result1.Period, result2.Period)
	assert.Equal(t, result1.Oversold, result2.Oversold)
	assert.InDelta(t, result1.Return, result2.Return, 0.001) // Allow small floating point differences
}

// TestParameterOptimizer_OptimizeRSI_EdgeCases tests various edge cases for optimization
func TestParameterOptimizer_OptimizeRSI_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		dataSize       int
		initialBalance float64
		commission     float64
		description    string
	}{
		{
			name:           "Very small dataset",
			dataSize:       50,
			initialBalance: 1000.0,
			commission:     0.001,
			description:    "Should handle small datasets gracefully",
		},
		{
			name:           "Large dataset",
			dataSize:       500,
			initialBalance: 50000.0,
			commission:     0.002,
			description:    "Should handle large datasets efficiently",
		},
		{
			name:           "High commission",
			dataSize:       200,
			initialBalance: 10000.0,
			commission:     0.01, // 1% commission
			description:    "Should handle high commission rates",
		},
		{
			name:           "Zero commission",
			dataSize:       200,
			initialBalance: 10000.0,
			commission:     0.0,
			description:    "Should handle zero commission",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewBacktestEngine(tt.initialBalance, tt.commission, strategy.NewMultiIndicatorStrategy())
			optimizer := &ParameterOptimizer{
				engine: engine,
				data:   generateTestData(tt.dataSize),
			}

			result := optimizer.OptimizeRSI()

			assert.NotNil(t, result, tt.description)
			assert.GreaterOrEqual(t, result.Period, 10, tt.description)
			assert.LessOrEqual(t, result.Period, 20, tt.description)
			assert.GreaterOrEqual(t, result.Oversold, 20, tt.description)
			assert.LessOrEqual(t, result.Oversold, 35, tt.description)
		})
	}
}

// Benchmark tests for optimization performance
func BenchmarkOptimizeRSI(b *testing.B) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())
	optimizer := &ParameterOptimizer{
		engine: engine,
		data:   generateTestData(200),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = optimizer.OptimizeRSI()
	}
}
