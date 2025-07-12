package backtest

import (
	"testing"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/internal/strategy"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
	"github.com/stretchr/testify/assert"
)

// TestNewBacktestEngine tests the creation of a new backtest engine
func TestNewBacktestEngine(t *testing.T) {
	initialBalance := 10000.0
	commission := 0.001
	strat := strategy.NewMultiIndicatorStrategy()

	engine := NewBacktestEngine(initialBalance, commission, strat)

	assert.NotNil(t, engine)
	assert.Equal(t, initialBalance, engine.initialBalance)
	assert.Equal(t, commission, engine.commission)
	assert.Equal(t, strat, engine.strategy)
	assert.NotNil(t, engine.results)
	assert.Equal(t, initialBalance, engine.results.StartBalance)
	assert.Empty(t, engine.results.Trades)
}

// TestBacktestEngine_Run_EmptyData tests running backtest with empty data
func TestBacktestEngine_Run_EmptyData(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())

	results := engine.Run([]types.OHLCV{}, 50)

	assert.NotNil(t, results)
	assert.Equal(t, 10000.0, results.StartBalance)
	assert.Equal(t, 10000.0, results.EndBalance)
	assert.Equal(t, 0.0, results.TotalReturn)
	assert.Equal(t, 0, results.TotalTrades)
}

// TestBacktestEngine_Run_InsufficientData tests running backtest with insufficient data
func TestBacktestEngine_Run_InsufficientData(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, strategy.NewMultiIndicatorStrategy())

	data := generateTestData(30) // Less than window size
	results := engine.Run(data, 50)

	assert.NotNil(t, results)
	assert.Equal(t, 10000.0, results.StartBalance)
	assert.Equal(t, 10000.0, results.EndBalance)
	assert.Equal(t, 0.0, results.TotalReturn)
	assert.Equal(t, 0, results.TotalTrades)
}

// TestBacktestEngine_Run_ProfitableTrades tests running backtest with profitable trades
func TestBacktestEngine_Run_ProfitableTrades(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, createMockStrategy(strategy.ActionBuy, 100.0))

	data := generateRisingData(100)
	results := engine.Run(data, 20)

	assert.NotNil(t, results)
	assert.Greater(t, results.TotalTrades, 0)
	assert.GreaterOrEqual(t, results.EndBalance, results.StartBalance)
	assert.GreaterOrEqual(t, results.TotalReturn, 0.0)
}

// TestBacktestEngine_Run_LosingTrades tests running backtest with losing trades
func TestBacktestEngine_Run_LosingTrades(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, createMockStrategy(strategy.ActionBuy, 100.0))

	data := generateFallingData(100)
	results := engine.Run(data, 20)

	assert.NotNil(t, results)
	assert.Greater(t, results.TotalTrades, 0)
	assert.LessOrEqual(t, results.EndBalance, results.StartBalance)
	assert.LessOrEqual(t, results.TotalReturn, 0.0)
}

// TestBacktestEngine_Run_CommissionImpact tests the impact of commission on results
func TestBacktestEngine_Run_CommissionImpact(t *testing.T) {
	// Test with no commission
	engineNoCommission := NewBacktestEngine(10000.0, 0.0, createMockStrategy(strategy.ActionBuy, 100.0))
	data := generateRisingData(100)
	resultsNoCommission := engineNoCommission.Run(data, 20)

	// Test with commission
	engineWithCommission := NewBacktestEngine(10000.0, 0.001, createMockStrategy(strategy.ActionBuy, 100.0))
	resultsWithCommission := engineWithCommission.Run(data, 20)

	// Results with commission should be lower due to trading costs
	assert.LessOrEqual(t, resultsWithCommission.EndBalance, resultsNoCommission.EndBalance)
}

// TestBacktestEngine_Run_DrawdownCalculation tests drawdown calculation
func TestBacktestEngine_Run_DrawdownCalculation(t *testing.T) {
	engine := NewBacktestEngine(10000.0, 0.001, createMockStrategy(strategy.ActionBuy, 100.0))

	// Create data with a peak followed by a decline
	data := generateVolatileData(100)
	results := engine.Run(data, 20)

	assert.NotNil(t, results)
	assert.GreaterOrEqual(t, results.MaxDrawdown, 0.0)
	assert.LessOrEqual(t, results.MaxDrawdown, 1.0) // Drawdown should be between 0% and 100%
}

// TestBacktestResults_PrintSummary tests the summary printing functionality
func TestBacktestResults_PrintSummary(t *testing.T) {
	results := &BacktestResults{
		StartBalance: 10000.0,
		EndBalance:   11000.0,
		TotalReturn:  0.10,
		MaxDrawdown:  0.05,
		TotalTrades:  10,
		ProfitFactor: 1.5,
	}

	// This test mainly ensures the function doesn't panic
	assert.NotPanics(t, func() {
		results.PrintSummary()
	})
}

// TestTrade_Structure tests the Trade structure
func TestTrade_Structure(t *testing.T) {
	now := time.Now()
	trade := Trade{
		EntryTime:  now,
		ExitTime:   now.Add(time.Hour),
		EntryPrice: 100.0,
		ExitPrice:  110.0,
		Quantity:   1.0,
		PnL:        10.0,
		Commission: 0.1,
	}

	assert.Equal(t, now, trade.EntryTime)
	assert.Equal(t, now.Add(time.Hour), trade.ExitTime)
	assert.Equal(t, 100.0, trade.EntryPrice)
	assert.Equal(t, 110.0, trade.ExitPrice)
	assert.Equal(t, 1.0, trade.Quantity)
	assert.Equal(t, 10.0, trade.PnL)
	assert.Equal(t, 0.1, trade.Commission)
}

// TestBacktestEngine_Run_EdgeCases tests various edge cases
func TestBacktestEngine_Run_EdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		initialBalance float64
		commission     float64
		dataSize       int
		windowSize     int
		expectedTrades int
	}{
		{
			name:           "Zero initial balance",
			initialBalance: 0.0,
			commission:     0.001,
			dataSize:       100,
			windowSize:     20,
			expectedTrades: 0,
		},
		{
			name:           "High commission",
			initialBalance: 10000.0,
			commission:     0.1, // 10% commission
			dataSize:       100,
			windowSize:     20,
			expectedTrades: 80, // Mock strategy will trade despite high costs
		},
		{
			name:           "Large window size",
			initialBalance: 10000.0,
			commission:     0.001,
			dataSize:       100,
			windowSize:     80,
			expectedTrades: 20, // Some trades will still occur with mock strategy
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := NewBacktestEngine(tt.initialBalance, tt.commission, createMockStrategy(strategy.ActionBuy, 100.0))
			data := generateTestData(tt.dataSize)
			results := engine.Run(data, tt.windowSize)

			assert.Equal(t, tt.expectedTrades, results.TotalTrades)
		})
	}
}

// Helper functions

// createMockStrategy creates a mock strategy that always returns the specified action
func createMockStrategy(action strategy.TradeAction, amount float64) strategy.Strategy {
	return &mockStrategy{
		action: action,
		amount: amount,
	}
}

// mockStrategy is a simple mock implementation for testing
type mockStrategy struct {
	action strategy.TradeAction
	amount float64
}

func (m *mockStrategy) ShouldExecuteTrade(data []types.OHLCV) (*strategy.TradeDecision, error) {
	return &strategy.TradeDecision{
		Action:     m.action,
		Amount:     m.amount,
		Confidence: 0.8,
		Strength:   0.7,
		Reason:     "Mock strategy decision",
	}, nil
}

func (m *mockStrategy) GetName() string {
	return "Mock Strategy"
}

// generateTestData creates test data with random price movements
func generateTestData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		// Add some randomness to price movements
		change := (float64(i%3) - 1) * 2.0 // -2, 0, or 2
		price := basePrice + change

		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 1.0,
			Low:       price - 1.0,
			Close:     price,
			Volume:    1000.0,
		}
		basePrice = price
	}

	return data
}

// generateRisingData creates data with a rising trend
func generateRisingData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		price := basePrice + float64(i)*0.5
		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 1.0,
			Low:       price - 1.0,
			Close:     price,
			Volume:    1000.0,
		}
	}

	return data
}

// generateFallingData creates data with a falling trend
func generateFallingData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		price := basePrice - float64(i)*0.5
		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 1.0,
			Low:       price - 1.0,
			Close:     price,
			Volume:    1000.0,
		}
	}

	return data
}

// generateVolatileData creates data with high volatility
func generateVolatileData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		// Large price swings
		change := (float64(i%2)*2 - 1) * 10.0 // -10 or +10
		price := basePrice + change

		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 5.0,
			Low:       price - 5.0,
			Close:     price,
			Volume:    1000.0,
		}
	}

	return data
}
