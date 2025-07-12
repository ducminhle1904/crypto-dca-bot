package backtest

import (
	"math"
)

// CalculateSharpeRatio calculates the Sharpe ratio for the backtest results
func (b *BacktestResults) CalculateSharpeRatio() float64 {
	if len(b.Trades) == 0 {
		return 0
	}

	// Calculate returns for each trade
	var returns []float64
	for _, trade := range b.Trades {
		if trade.ExitPrice > 0 && trade.EntryPrice > 0 {
			ret := (trade.ExitPrice - trade.EntryPrice) / trade.EntryPrice
			returns = append(returns, ret)
		}
	}

	if len(returns) == 0 {
		return 0
	}

	// Calculate average return
	avgReturn := 0.0
	for _, r := range returns {
		avgReturn += r
	}
	avgReturn /= float64(len(returns))

	// Calculate standard deviation
	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-avgReturn, 2)
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0
	}

	// Sharpe ratio = (average return - risk free rate) / standard deviation
	// Assuming risk-free rate of 0 for simplicity
	return avgReturn / stdDev
}

// CalculateProfitFactor calculates the profit factor
func (b *BacktestResults) CalculateProfitFactor() float64 {
	if len(b.Trades) == 0 {
		return 0
	}

	totalProfit := 0.0
	totalLoss := 0.0

	for _, trade := range b.Trades {
		pnl := trade.PnL
		if pnl > 0 {
			totalProfit += pnl
		} else {
			totalLoss += math.Abs(pnl)
		}
	}

	if totalLoss == 0 {
		return 0
	}

	return totalProfit / totalLoss
}

// CalculateWinRate calculates the win rate percentage
func (b *BacktestResults) CalculateWinRate() float64 {
	if len(b.Trades) == 0 {
		return 0
	}

	wins := 0
	for _, trade := range b.Trades {
		if trade.PnL > 0 {
			wins++
		}
	}

	return float64(wins) / float64(len(b.Trades)) * 100
}

// UpdateMetrics updates all calculated metrics
func (b *BacktestResults) UpdateMetrics() {
	b.SharpeRatio = b.CalculateSharpeRatio()
	b.ProfitFactor = b.CalculateProfitFactor()
	b.WinningTrades = int(b.CalculateWinRate() * float64(b.TotalTrades) / 100)
	b.LosingTrades = b.TotalTrades - b.WinningTrades
}
