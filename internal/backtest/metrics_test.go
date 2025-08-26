package backtest

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestCalculateSharpeRatio_EmptyTrades tests Sharpe ratio calculation with no trades
func TestCalculateSharpeRatio_EmptyTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{},
	}

	sharpeRatio := results.CalculateSharpeRatio()
	assert.Equal(t, 0.0, sharpeRatio)
}

// TestCalculateSharpeRatio_NoExitPrices tests Sharpe ratio with trades that have no exit prices
func TestCalculateSharpeRatio_NoExitPrices(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{EntryPrice: 100.0, ExitPrice: 0.0, PnL: 10.0},
			{EntryPrice: 110.0, ExitPrice: 0.0, PnL: -5.0},
		},
	}

	sharpeRatio := results.CalculateSharpeRatio()
	assert.Equal(t, 0.0, sharpeRatio)
}

// TestCalculateSharpeRatio_ProfitableTrades tests Sharpe ratio with profitable trades
func TestCalculateSharpeRatio_ProfitableTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{EntryPrice: 100.0, ExitPrice: 110.0, PnL: 10.0},
			{EntryPrice: 110.0, ExitPrice: 120.0, PnL: 10.0},
			{EntryPrice: 120.0, ExitPrice: 130.0, PnL: 10.0},
		},
	}

	sharpeRatio := results.CalculateSharpeRatio()
	assert.Greater(t, sharpeRatio, 0.0)
}

// TestCalculateSharpeRatio_LosingTrades tests Sharpe ratio with losing trades
func TestCalculateSharpeRatio_LosingTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{EntryPrice: 100.0, ExitPrice: 90.0, PnL: -10.0},
			{EntryPrice: 90.0, ExitPrice: 80.0, PnL: -10.0},
			{EntryPrice: 80.0, ExitPrice: 70.0, PnL: -10.0},
		},
	}

	sharpeRatio := results.CalculateSharpeRatio()
	assert.Less(t, sharpeRatio, 0.0)
}

// TestCalculateSharpeRatio_MixedTrades tests Sharpe ratio with mixed profitable and losing trades
func TestCalculateSharpeRatio_MixedTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{EntryPrice: 100.0, ExitPrice: 110.0, PnL: 10.0},
			{EntryPrice: 110.0, ExitPrice: 100.0, PnL: -10.0},
			{EntryPrice: 100.0, ExitPrice: 120.0, PnL: 20.0},
		},
	}

	sharpeRatio := results.CalculateSharpeRatio()
	// Should be positive since average return is positive
	assert.Greater(t, sharpeRatio, 0.0)
}

// TestCalculateSharpeRatio_ZeroVolatility tests Sharpe ratio with zero volatility
func TestCalculateSharpeRatio_ZeroVolatility(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{EntryPrice: 100.0, ExitPrice: 110.0, PnL: 10.0},
			{EntryPrice: 200.0, ExitPrice: 220.0, PnL: 20.0},
			{EntryPrice: 300.0, ExitPrice: 330.0, PnL: 30.0},
		},
	}

	sharpeRatio := results.CalculateSharpeRatio()
	// With identical percentage returns (10% each), volatility is zero, so Sharpe ratio should be zero
	assert.Equal(t, 0.0, sharpeRatio) // Zero volatility results in zero Sharpe ratio
}

// TestCalculateProfitFactor_EmptyTrades tests profit factor calculation with no trades
func TestCalculateProfitFactor_EmptyTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{},
	}

	profitFactor := results.CalculateProfitFactor()
	assert.Equal(t, 0.0, profitFactor)
}

// TestCalculateProfitFactor_AllProfitableTrades tests profit factor with all profitable trades
func TestCalculateProfitFactor_AllProfitableTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{PnL: 10.0},
			{PnL: 20.0},
			{PnL: 15.0},
		},
	}

	profitFactor := results.CalculateProfitFactor()
	assert.True(t, math.IsInf(profitFactor, 1)) // No losses with profits -> +Inf
}

// TestCalculateProfitFactor_AllLosingTrades tests profit factor with all losing trades
func TestCalculateProfitFactor_AllLosingTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{PnL: -10.0},
			{PnL: -20.0},
			{PnL: -15.0},
		},
	}

	profitFactor := results.CalculateProfitFactor()
	assert.Equal(t, 0.0, profitFactor) // No profits, so profit factor is 0
}

// TestCalculateProfitFactor_MixedTrades tests profit factor with mixed trades
func TestCalculateProfitFactor_MixedTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{PnL: 20.0},  // Profit
			{PnL: -10.0}, // Loss
			{PnL: 30.0},  // Profit
			{PnL: -5.0},  // Loss
		},
	}

	profitFactor := results.CalculateProfitFactor()
	expected := 50.0 / 15.0 // Total profit / Total loss = 50 / 15 = 3.33
	assert.InDelta(t, expected, profitFactor, 0.01)
}

// TestCalculateWinRate_EmptyTrades tests win rate calculation with no trades
func TestCalculateWinRate_EmptyTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{},
	}

	winRate := results.CalculateWinRate()
	assert.Equal(t, 0.0, winRate)
}

// TestCalculateWinRate_AllWinningTrades tests win rate with all winning trades
func TestCalculateWinRate_AllWinningTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{PnL: 10.0},
			{PnL: 20.0},
			{PnL: 15.0},
		},
	}

	winRate := results.CalculateWinRate()
	assert.Equal(t, 100.0, winRate)
}

// TestCalculateWinRate_AllLosingTrades tests win rate with all losing trades
func TestCalculateWinRate_AllLosingTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{PnL: -10.0},
			{PnL: -20.0},
			{PnL: -15.0},
		},
	}

	winRate := results.CalculateWinRate()
	assert.Equal(t, 0.0, winRate)
}

// TestCalculateWinRate_MixedTrades tests win rate with mixed trades
func TestCalculateWinRate_MixedTrades(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{PnL: 10.0},  // Win
			{PnL: -5.0},  // Loss
			{PnL: 15.0},  // Win
			{PnL: -10.0}, // Loss
			{PnL: 20.0},  // Win
		},
	}

	winRate := results.CalculateWinRate()
	expected := 60.0 // 3 wins out of 5 trades = 60%
	assert.Equal(t, expected, winRate)
}

// TestUpdateMetrics tests the UpdateMetrics function
func TestUpdateMetrics(t *testing.T) {
	results := &BacktestResults{
		Trades: []Trade{
			{EntryPrice: 100.0, ExitPrice: 110.0, PnL: 10.0},  // Win
			{EntryPrice: 110.0, ExitPrice: 100.0, PnL: -10.0}, // Loss
			{EntryPrice: 100.0, ExitPrice: 120.0, PnL: 20.0},  // Win
		},
		TotalTrades: 3,
	}

	results.UpdateMetrics()

	// Check that metrics were calculated
	assert.NotEqual(t, 0.0, results.SharpeRatio)
	assert.NotEqual(t, 0.0, results.ProfitFactor)
	assert.Equal(t, 2, results.WinningTrades) // 2 winning trades (PnL > 0)
	assert.Equal(t, 1, results.LosingTrades)  // 1 losing trade (PnL < 0)
}

// TestUpdateMetrics_EmptyTrades tests UpdateMetrics with no trades
func TestUpdateMetrics_EmptyTrades(t *testing.T) {
	results := &BacktestResults{
		Trades:      []Trade{},
		TotalTrades: 0,
	}

	results.UpdateMetrics()

	assert.Equal(t, 0.0, results.SharpeRatio)
	assert.Equal(t, 0.0, results.ProfitFactor)
	assert.Equal(t, 0, results.WinningTrades)
	assert.Equal(t, 0, results.LosingTrades)
}

// TestMetrics_EdgeCases tests various edge cases for metrics calculations
func TestMetrics_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		trades   []Trade
		expected struct {
			sharpeRatio   float64
			profitFactor  float64
			winRate       float64
			winningTrades int
			losingTrades  int
		}
	}{
		{
			name:   "Single profitable trade",
			trades: []Trade{{EntryPrice: 100.0, ExitPrice: 110.0, PnL: 10.0}},
			expected: struct {
				sharpeRatio   float64
				profitFactor  float64
				winRate       float64
				winningTrades int
				losingTrades  int
			}{
				sharpeRatio:   0.0,       // Zero volatility
				profitFactor:  math.Inf(1), // No losses with profit -> +Inf
				winRate:       100.0,
				winningTrades: 1,
				losingTrades:  0,
			},
		},
		{
			name:   "Single losing trade",
			trades: []Trade{{EntryPrice: 100.0, ExitPrice: 90.0, PnL: -10.0}},
			expected: struct {
				sharpeRatio   float64
				profitFactor  float64
				winRate       float64
				winningTrades int
				losingTrades  int
			}{
				sharpeRatio:   0.0, // Zero volatility
				profitFactor:  0.0, // No profits
				winRate:       0.0,
				winningTrades: 0,
				losingTrades:  1,
			},
		},
		{
			name: "Break-even trades",
			trades: []Trade{
				{EntryPrice: 100.0, ExitPrice: 100.0, PnL: 0.0},
				{EntryPrice: 110.0, ExitPrice: 110.0, PnL: 0.0},
			},
			expected: struct {
				sharpeRatio   float64
				profitFactor  float64
				winRate       float64
				winningTrades int
				losingTrades  int
			}{
				sharpeRatio:   0.0, // Zero volatility
				profitFactor:  0.0, // No profits or losses
				winRate:       0.0, // No winning trades (PnL = 0 is not considered winning)
				winningTrades: 0,
				losingTrades:  2, // PnL = 0 is considered losing in this implementation
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := &BacktestResults{
				Trades:      tt.trades,
				TotalTrades: len(tt.trades),
			}

			results.UpdateMetrics()

			assert.Equal(t, tt.expected.sharpeRatio, results.SharpeRatio)
			assert.Equal(t, tt.expected.profitFactor, results.ProfitFactor)
			assert.Equal(t, tt.expected.winRate, results.CalculateWinRate())
			assert.Equal(t, tt.expected.winningTrades, results.WinningTrades)
			assert.Equal(t, tt.expected.losingTrades, results.LosingTrades)
		})
	}
}

// Benchmark tests for performance
func BenchmarkCalculateSharpeRatio(b *testing.B) {
	results := &BacktestResults{
		Trades: generateBenchmarkTrades(1000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = results.CalculateSharpeRatio()
	}
}

func BenchmarkCalculateProfitFactor(b *testing.B) {
	results := &BacktestResults{
		Trades: generateBenchmarkTrades(1000),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = results.CalculateProfitFactor()
	}
}

func BenchmarkUpdateMetrics(b *testing.B) {
	results := &BacktestResults{
		Trades:      generateBenchmarkTrades(1000),
		TotalTrades: 1000,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		results.UpdateMetrics()
	}
}

// Helper function for benchmark tests
func generateBenchmarkTrades(count int) []Trade {
	trades := make([]Trade, count)
	for i := 0; i < count; i++ {
		entryPrice := 100.0 + float64(i)
		exitPrice := entryPrice + (float64(i%3)-1)*5.0 // Varying PnL
		pnl := exitPrice - entryPrice

		trades[i] = Trade{
			EntryTime:  time.Now().Add(time.Duration(i) * time.Hour),
			ExitTime:   time.Now().Add(time.Duration(i+1) * time.Hour),
			EntryPrice: entryPrice,
			ExitPrice:  exitPrice,
			Quantity:   1.0,
			PnL:        pnl,
			Commission: 0.1,
		}
	}
	return trades
}
