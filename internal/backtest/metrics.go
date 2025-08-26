package backtest

import (
	"math"
)

// CalculateSharpeRatio calculates the Sharpe ratio for the backtest results
func (b *BacktestResults) CalculateSharpeRatio() float64 {
	// Prefer cycle partial exits if present (Option A)
	// Build per-exit returns as pnl / (avgEntry * qty)
	var returns []float64
	hasPartials := false
	for _, c := range b.Cycles {
		if len(c.PartialExits) > 0 {
			hasPartials = true
			for _, pe := range c.PartialExits {
				denom := c.AvgEntry * pe.Quantity
				if denom > 0 {
					returns = append(returns, pe.PnL/denom)
				}
			}
		}
	}
	if !hasPartials {
		if len(b.Trades) == 0 {
			return 0
		}
		for _, trade := range b.Trades {
			if trade.ExitPrice > 0 && trade.EntryPrice > 0 {
				ret := (trade.ExitPrice - trade.EntryPrice) / trade.EntryPrice
				returns = append(returns, ret)
			}
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

	if stdDev == 0 || stdDev < 1e-10 {
		return 0
	}

	// Sharpe ratio = (average return - risk free rate) / standard deviation
	// Assuming risk-free rate of 0 for simplicity
	return avgReturn / stdDev
}

// CalculateProfitFactor calculates the profit factor
func (b *BacktestResults) CalculateProfitFactor() float64 {
    // Prefer cycle partial exits if present (Option A)
    totalProfit := 0.0
    totalLoss := 0.0
    counted := false
    for _, c := range b.Cycles {
        if len(c.PartialExits) == 0 {
            continue
        }
        counted = true
        for _, pe := range c.PartialExits {
            if pe.PnL > 0 {
                totalProfit += pe.PnL
            } else {
                totalLoss += math.Abs(pe.PnL)
            }
        }
    }
    if !counted {
        if len(b.Trades) == 0 {
            return 0
        }
        for _, trade := range b.Trades {
            pnl := trade.PnL
            if pnl > 0 {
                totalProfit += pnl
            } else {
                totalLoss += math.Abs(pnl)
            }
        }
    }

    if totalLoss == 0 {
        if totalProfit > 0 {
            return math.Inf(1)
        }
        return 0
    }

    return totalProfit / totalLoss
}

// CalculateWinRate calculates the win rate percentage
func (b *BacktestResults) CalculateWinRate() float64 {
	// Prefer cycle partial exits if present (Option A)
	wins := 0
	total := 0
	counted := false
	for _, c := range b.Cycles {
		if len(c.PartialExits) == 0 {
			continue
		}
		counted = true
		for _, pe := range c.PartialExits {
			if pe.PnL > 0 {
				wins++
			}
			total++
		}
	}
	if !counted {
		if len(b.Trades) == 0 {
			return 0
		}
		for _, trade := range b.Trades {
			if trade.PnL > 0 {
				wins++
			}
		}
		total = len(b.Trades)
	}
	if total == 0 {
		return 0
	}
	return float64(wins) / float64(total) * 100
}

// UpdateMetrics updates all calculated metrics
func (b *BacktestResults) UpdateMetrics() {
	b.SharpeRatio = b.CalculateSharpeRatio()
	b.ProfitFactor = b.CalculateProfitFactor()

	// If we have partial exits, base counts on them; otherwise on trades
	partialCount := 0
	wins := 0
	for _, c := range b.Cycles {
		for _, pe := range c.PartialExits {
			partialCount++
			if pe.PnL > 0 {
				wins++
			}
		}
	}
	if partialCount > 0 {
		b.TotalTrades = partialCount
		b.WinningTrades = wins
		b.LosingTrades = partialCount - wins
	} else {
		b.TotalTrades = len(b.Trades)
		wins = 0
		for _, trade := range b.Trades {
			if trade.PnL > 0 {
				wins++
			}
		}
		b.WinningTrades = wins
		b.LosingTrades = len(b.Trades) - wins
	}
}
