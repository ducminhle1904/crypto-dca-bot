package backtest

import (
	"github.com/Zmey56/enhanced-dca-bot/internal/indicators"
	"github.com/Zmey56/enhanced-dca-bot/internal/strategy"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

type ParameterOptimizer struct {
	engine *BacktestEngine
	data   []types.OHLCV
}

func (o *ParameterOptimizer) OptimizeRSI() *OptimizationResult {
	var bestResult *OptimizationResult
	firstRun := true

	// Going through the RSI parameters
	for period := 10; period <= 20; period += 2 {
		for oversold := 20; oversold <= 35; oversold += 5 {
			// Creating strategy with new parameters
			rsi := indicators.NewRSI(period)
			rsi.SetOversold(float64(oversold))

			strategy := strategy.NewEnhancedDCAStrategy(1000)
			strategy.AddIndicator(rsi)

			// Launching the backtest
			o.engine.strategy = strategy
			result := o.engine.Run(o.data, 50)

			// Initialize bestResult on first run or update if better
			if firstRun || result.TotalReturn > bestResult.Return {
				bestResult = &OptimizationResult{
					Period:   period,
					Oversold: oversold,
					Return:   result.TotalReturn,
				}
				firstRun = false
			}
		}
	}

	// Return default result if no optimization was performed
	if bestResult == nil {
		bestResult = &OptimizationResult{
			Period:   14,
			Oversold: 30,
			Return:   0.0,
		}
	}

	return bestResult
}

type OptimizationResult struct {
	Period   int
	Oversold int
	Return   float64
}
