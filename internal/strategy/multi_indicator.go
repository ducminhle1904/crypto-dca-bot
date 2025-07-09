package strategy

import (
	"errors"
	"fmt"
	"math"

	"github.com/Zmey56/enhanced-dca-bot/internal/indicators"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
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
				Indicator: indicators.NewRSI(14),
				Weight: map[MarketRegime]float64{
					RegimeTrending: 0.2,
					RegimeSideways: 0.4,
					RegimeVolatile: 0.1,
				},
			},
			{
				Indicator: indicators.NewSMA(50),
				Weight: map[MarketRegime]float64{
					RegimeTrending: 0.4,
					RegimeSideways: 0.1,
					RegimeVolatile: 0.2,
				},
			},
			{
				Indicator: indicators.NewBollingerBands(20, 2.0),
				Weight: map[MarketRegime]float64{
					RegimeTrending: 0.2,
					RegimeSideways: 0.3,
					RegimeVolatile: 0.4,
				},
			},
			{
				Indicator: indicators.NewMACD(12, 26, 9),
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

	// Log market regime
	regimeNames := map[MarketRegime]string{
		RegimeTrending: "TRENDING",
		RegimeSideways: "SIDEWAYS",
		RegimeVolatile: "VOLATILE",
	}
	fmt.Printf("🌍 Market Regime: %s\n", regimeNames[regime])

	buyScore := 0.0
	sellScore := 0.0
	totalWeight := 0.0

	fmt.Println("📊 === Individual Indicator Analysis ===")
	for _, wi := range m.indicators {
		weight := wi.Weight[regime]
		indicatorName := wi.Indicator.GetName()

		shouldBuy, _ := wi.Indicator.ShouldBuy(currentPrice, data)
		shouldSell, _ := wi.Indicator.ShouldSell(currentPrice, data)
		strength := wi.Indicator.GetSignalStrength()

		// Log individual indicator analysis
		signal := "HOLD"
		if shouldBuy {
			signal = "BUY"
			buyScore += weight * strength
		} else if shouldSell {
			signal = "SELL"
			sellScore += weight * strength
		}
		totalWeight += weight

		fmt.Printf("  %s: %s (Strength: %.2f%%, Weight: %.2f)\n",
			indicatorName, signal, strength*100, weight)
	}
	fmt.Println("📋 === End Indicator Analysis ===")

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

		fmt.Printf("📈 Aggregated Scores - Buy: %.2f%%, Sell: %.2f%%\n",
			buyScore*100, sellScore*100)

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
