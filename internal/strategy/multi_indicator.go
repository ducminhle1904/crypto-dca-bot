package strategy

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/bands"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/common"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/oscillators"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// MarketRegime represents the detected market regime
// (trending, sideways, volatile)
type MarketRegime int

const (
	RegimeTrending MarketRegime = iota
	RegimeSideways
	RegimeVolatile
)

// WeightedIndicator links an indicator with weights for each regime
type WeightedIndicator struct {
	Indicator indicators.TechnicalIndicator
	Weight    map[MarketRegime]float64
}

// MultiIndicatorStrategy aggregates signals from multiple indicators
// and makes a trading decision based on weighted consensus.
type MultiIndicatorStrategy struct {
	indicators          []WeightedIndicator
	volatilityThreshold float64
}

// NewMultiIndicatorStrategy creates a new multi-indicator strategy
func NewMultiIndicatorStrategy() *MultiIndicatorStrategy {
	return &MultiIndicatorStrategy{
		indicators: []WeightedIndicator{
			{
				Indicator: oscillators.NewRSI(14),
				Weight: map[MarketRegime]float64{
					RegimeTrending: 0.2,
					RegimeSideways: 0.4,
					RegimeVolatile: 0.1,
				},
			},
			{
				Indicator: common.NewSMA(50),
				Weight: map[MarketRegime]float64{
					RegimeTrending: 0.4,
					RegimeSideways: 0.1,
					RegimeVolatile: 0.2,
				},
			},
			{
				Indicator: bands.NewBollingerBands(20, 2.0),
				Weight: map[MarketRegime]float64{
					RegimeTrending: 0.2,
					RegimeSideways: 0.3,
					RegimeVolatile: 0.4,
				},
			},
			{
				Indicator: oscillators.NewMACD(12, 26, 9),
				Weight: map[MarketRegime]float64{
					RegimeTrending: 0.2,
					RegimeSideways: 0.2,
					RegimeVolatile: 0.3,
				},
			},
		},
		volatilityThreshold: 0.05, // 5% threshold for volatility regime
	}
}

// ShouldExecuteTrade aggregates indicator signals and returns a trade decision
func (m *MultiIndicatorStrategy) ShouldExecuteTrade(data []types.OHLCV) (*TradeDecision, error) {
	if len(data) < 50 {
		return nil, errors.New("insufficient data for multi-indicator analysis")
	}

	regime := m.detectMarketRegime(data)
	currentPrice := data[len(data)-1].Close

	// Determine market regime (no logging)

	buyScore := 0.0
	sellScore := 0.0
	totalWeight := 0.0

	for _, wi := range m.indicators {
		weight := wi.Weight[regime]

		shouldBuy, _ := wi.Indicator.ShouldBuy(currentPrice, data)
		shouldSell, _ := wi.Indicator.ShouldSell(currentPrice, data)
		strength := wi.Indicator.GetSignalStrength()

		if shouldBuy {
			buyScore += weight * strength
		} else if shouldSell {
			sellScore += weight * strength
		}
		totalWeight += weight

	}

	// Normalize scores
	var action TradeAction
	confidence := 0.0
	strength := 0.0
	var reason string

	if totalWeight == 0 {
		action = ActionHold
		reason = "No indicator weights configured for this regime"
	} else {
		buyScore /= totalWeight
		sellScore /= totalWeight

		if buyScore > 0.6 && buyScore > sellScore {
			action = ActionBuy
			confidence = buyScore
			strength = buyScore
			reason = "Buy consensus among indicators"
		} else if sellScore > 0.6 && sellScore > buyScore {
			action = ActionSell
			confidence = sellScore
			strength = sellScore
			reason = "Sell consensus among indicators"
		} else {
			action = ActionHold
			confidence = math.Max(buyScore, sellScore)
			strength = confidence
			reason = "No strong consensus"
		}
	}

	return &TradeDecision{
		Action:     action,
		Amount:     0, // Amount should be set by risk manager
		Confidence: confidence,
		Strength:   strength,
		Reason:     reason,
		Timestamp:  data[len(data)-1].Timestamp,
	}, nil
}

// detectMarketRegime determines the current market regime
func (m *MultiIndicatorStrategy) detectMarketRegime(data []types.OHLCV) MarketRegime {
	if len(data) < 50 {
		return RegimeSideways
	}

	atr := calculateATR(data, 14)
	avgPrice := calculateAvgPrice(data, 20)
	volatility := atr / avgPrice

	sma20 := calculateSMA(data, 20)
	sma50 := calculateSMA(data, 50)

	if volatility > m.volatilityThreshold {
		return RegimeVolatile
	}
	if math.Abs(sma20-sma50)/sma50 > 0.02 {
		return RegimeTrending
	}
	return RegimeSideways
}

// calculateATR computes the Average True Range for volatility estimation
func calculateATR(data []types.OHLCV, period int) float64 {
	if len(data) < period+1 {
		return 0
	}
	atr := 0.0
	for i := len(data) - period; i < len(data); i++ {
		tr := math.Max(
			data[i].High-data[i].Low,
			math.Max(
				math.Abs(data[i].High-data[i-1].Close),
				math.Abs(data[i].Low-data[i-1].Close),
			),
		)
		atr += tr
	}
	return atr / float64(period)
}

// calculateAvgPrice returns the average close price for the given period
func calculateAvgPrice(data []types.OHLCV, period int) float64 {
	if len(data) < period {
		return 0
	}
	sum := 0.0
	for i := len(data) - period; i < len(data); i++ {
		sum += data[i].Close
	}
	return sum / float64(period)
}

// calculateSMA returns the simple moving average for the given period
func calculateSMA(data []types.OHLCV, period int) float64 {
	if len(data) < period {
		return 0
	}
	sum := 0.0
	for i := len(data) - period; i < len(data); i++ {
		sum += data[i].Close
	}
	return sum / float64(period)
}

// GetName returns the strategy name
func (m *MultiIndicatorStrategy) GetName() string {
	return "Multi-Indicator Strategy"
}

// OnCycleComplete is called when a take-profit cycle is completed
// MultiIndicatorStrategy doesn't maintain cycle-specific state, so this is a no-op
func (m *MultiIndicatorStrategy) OnCycleComplete() {
	// No state to reset for multi-indicator strategy
}

// ResetForNewPeriod resets strategy state for walk-forward validation periods
func (m *MultiIndicatorStrategy) ResetForNewPeriod() {
	// Reset all indicators to prevent state contamination between validation folds
	for _, weightedIndicator := range m.indicators {
		weightedIndicator.Indicator.ResetState()
	}
}
