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
	bestResult := &OptimizationResult{}

	// Going through the RSI parameters
	for period := 10; period <= 20; period += 2 {
		for oversold := 20; oversold <= 35; oversold += 5 {
			// Создаем стратегию с новыми параметрами
			rsi := indicators.NewRSI(period)
			rsi.SetOversold(float64(oversold))

			strategy := strategy.NewEnhancedDCAStrategy(1000)
			strategy.AddIndicator(rsi)

			// Launching the backtest
			o.engine.strategy = strategy
			result := o.engine.Run(o.data, 50)

			// Comparing with the best result
			if result.TotalReturn > bestResult.Return {
				bestResult = &OptimizationResult{
					Period:   period,
					Oversold: oversold,
					Return:   result.TotalReturn,
				}
			}
		}
	}

	return bestResult
}

type OptimizationResult struct {
	Period   int
	Oversold int
	Return   float64
}
