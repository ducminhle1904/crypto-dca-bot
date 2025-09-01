package backtest

import (
	"math"
	"time"
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
            // Skip open entry trades in multi-TP mode: no exit time AND part of a cycle AND zero PnL
            // This preserves test trades and other completed trades that may not have exit times set
            if trade.ExitTime.IsZero() && trade.Cycle > 0 && trade.PnL == 0 {
                continue
            }
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
			// Skip open entry trades in multi-TP mode: no exit time AND part of a cycle AND zero PnL
			// This preserves test trades and other completed trades that may not have exit times set
			if trade.ExitTime.IsZero() && trade.Cycle > 0 && trade.PnL == 0 {
				continue
			}
			if trade.PnL > 0 {
				wins++
			}
			total++
		}
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

	// Calculate enhanced metrics from equity curve
	b.calculateEnhancedMetrics()

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
		// Count only completed trades (those with exit times and non-zero PnL calculations)
		completedTrades := 0
		wins = 0
		for _, trade := range b.Trades {
			// Skip open entry trades in multi-TP mode: no exit time AND part of a cycle AND zero PnL
			// This preserves test trades and other completed trades that may not have exit times set
			if trade.ExitTime.IsZero() && trade.Cycle > 0 && trade.PnL == 0 {
				continue
			}
			completedTrades++
			if trade.PnL > 0 {
				wins++
			}
		}
		b.TotalTrades = completedTrades
		b.WinningTrades = wins
		b.LosingTrades = completedTrades - wins
	}
}

// calculateEnhancedMetrics computes time-aware and advanced metrics from equity curve
func (b *BacktestResults) calculateEnhancedMetrics() {
	if len(b.EquityCurve) == 0 {
		return
	}

	// Calculate annualized metrics
	b.calculateAnnualizedMetrics()
	
	// Calculate Sortino ratio (downside deviation)
	b.SortinoRatio = b.calculateSortinoRatio()
	
	// Calculate Calmar ratio (return / max drawdown)
	b.CalmarRatio = b.calculateCalmarRatio()
	
	// Calculate exposure metrics
	b.calculateExposureMetrics()
	
	// Calculate turnover
	b.TotalTurnover = b.calculateTurnover()
}

// calculateAnnualizedMetrics computes annualized return and Sharpe ratio
func (b *BacktestResults) calculateAnnualizedMetrics() {
	if len(b.EquityCurve) < 2 {
		return
	}

	first := b.EquityCurve[0]
	last := b.EquityCurve[len(b.EquityCurve)-1]
	
	// Calculate time period in years
	duration := last.Timestamp.Sub(first.Timestamp)
	years := duration.Hours() / (24 * 365.25)
	
	if years <= 0 {
		return
	}

	// Annualized return: (ending / beginning)^(1/years) - 1
	if first.Equity > 0 {
		b.AnnualizedReturn = math.Pow(last.Equity/first.Equity, 1.0/years) - 1.0
	}

	// Annualized Sharpe: daily Sharpe * sqrt(252) for daily, adjust for actual frequency
	// Estimate frequency from equity curve
	if len(b.EquityCurve) > 1 {
		avgInterval := duration / time.Duration(len(b.EquityCurve)-1)
		periodsPerYear := time.Duration(24*365.25) * time.Hour / avgInterval
		b.AnnualizedSharpe = b.SharpeRatio * math.Sqrt(float64(periodsPerYear))
	}
}

// calculateSortinoRatio computes Sortino ratio (return / downside deviation)
func (b *BacktestResults) calculateSortinoRatio() float64 {
	if len(b.EquityCurve) < 2 {
		return 0
	}

	returns := make([]float64, 0, len(b.EquityCurve)-1)
	for i := 1; i < len(b.EquityCurve); i++ {
		if b.EquityCurve[i-1].Equity > 0 {
			ret := (b.EquityCurve[i].Equity - b.EquityCurve[i-1].Equity) / b.EquityCurve[i-1].Equity
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

	// Calculate downside deviation (only negative returns)
	downsideVariance := 0.0
	downsideCount := 0
	for _, r := range returns {
		if r < 0 {
			downsideVariance += r * r
			downsideCount++
		}
	}

	if downsideCount == 0 || downsideVariance == 0 {
		return math.Inf(1) // All positive returns
	}

	downsideStdDev := math.Sqrt(downsideVariance / float64(downsideCount))
	return avgReturn / downsideStdDev
}

// calculateCalmarRatio computes Calmar ratio (annualized return / max drawdown)
func (b *BacktestResults) calculateCalmarRatio() float64 {
	if b.MaxDrawdown == 0 {
		return math.Inf(1)
	}
	return b.AnnualizedReturn / b.MaxDrawdown
}

// calculateExposureMetrics computes max and average exposure
func (b *BacktestResults) calculateExposureMetrics() {
	if len(b.EquityCurve) == 0 {
		return
	}

	maxExp := 0.0
	totalExp := 0.0
	
	for _, point := range b.EquityCurve {
		if point.Exposure > maxExp {
			maxExp = point.Exposure
		}
		totalExp += point.Exposure
	}

	b.MaxExposure = maxExp
	b.AvgExposure = totalExp / float64(len(b.EquityCurve))
}

// calculateTurnover computes total turnover as sum of trade volumes / average equity
func (b *BacktestResults) calculateTurnover() float64 {
	if len(b.EquityCurve) == 0 {
		return 0
	}

	totalVolume := 0.0
	for _, trade := range b.Trades {
		// Count both entry and exit as turnover
		entryVolume := trade.EntryPrice * trade.Quantity
		if trade.ExitPrice > 0 {
			exitVolume := trade.ExitPrice * trade.Quantity
			totalVolume += entryVolume + exitVolume
		} else {
			totalVolume += entryVolume
		}
	}

	// Calculate average equity
	totalEquity := 0.0
	for _, point := range b.EquityCurve {
		totalEquity += point.Equity
	}
	avgEquity := totalEquity / float64(len(b.EquityCurve))

	if avgEquity == 0 {
		return 0
	}

	return totalVolume / avgEquity
}
