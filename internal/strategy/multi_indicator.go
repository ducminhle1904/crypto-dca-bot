package strategy

import (
	"errors"
	"github.com/Zmey56/enhanced-dca-bot/internal/indicators"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
	"math"
	"time"
)

type MultiIndicatorStrategy struct {
	indicators          []WeightedIndicator
	marketRegime        MarketRegime
	volatilityThreshold float64
}

type WeightedIndicator struct {
	Indicator indicators.TechnicalIndicator
	Weight    map[MarketRegime]float64
	LastValue float64
}

type MarketRegime int

const (
	RegimeTrending MarketRegime = iota
	RegimeSideways
	RegimeVolatile
)

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
		volatilityThreshold: 0.05,
	}
}

func (m *MultiIndicatorStrategy) CalculateSignal(data []types.OHLCV) (*AggregatedSignal, error) {
	if len(data) < 50 {
		return nil, errors.New("insufficient data for multi-indicator analysis")
	}

	// Определяем рыночный режим
	regime := m.detectMarketRegime(data)
	m.marketRegime = regime

	// Собираем взвешенные сигналы
	totalBuyWeight := 0.0
	totalSellWeight := 0.0
	totalWeight := 0.0

	for i := range m.indicators {
		indicator := &m.indicators[i]
		weight := indicator.Weight[regime]

		currentPrice := data[len(data)-1].Close
		shouldBuy, _ := indicator.Indicator.ShouldBuy(currentPrice, data)
		shouldSell, _ := indicator.Indicator.ShouldSell(currentPrice, data)

		if shouldBuy {
			totalBuyWeight += weight * indicator.Indicator.GetSignalStrength()
		} else if shouldSell {
			totalSellWeight += weight * indicator.Indicator.GetSignalStrength()
		}

		totalWeight += weight
	}

	// Нормализуем сигналы
	buyStrength := totalBuyWeight / totalWeight
	sellStrength := totalSellWeight / totalWeight

	return &AggregatedSignal{
		BuyStrength:  buyStrength,
		SellStrength: sellStrength,
		Confidence:   math.Max(buyStrength, sellStrength),
		MarketRegime: regime,
		Timestamp:    data[len(data)-1].Timestamp,
	}, nil
}

func (m *MultiIndicatorStrategy) detectMarketRegime(data []types.OHLCV) MarketRegime {
	if len(data) < 20 {
		return RegimeSideways
	}

	// Вычисляем волатильность через ATR
	atr := m.calculateATR(data, 14)
	avgPrice := m.calculateAvgPrice(data, 20)

	volatility := atr / avgPrice

	// Определяем тренд через наклон SMA
	sma20 := m.calculateSMA(data, 20)
	sma50 := m.calculateSMA(data, 50)

	if volatility > m.volatilityThreshold {
		return RegimeVolatile
	}

	if math.Abs(sma20-sma50)/sma50 > 0.02 {
		return RegimeTrending
	}

	return RegimeSideways
}

type AggregatedSignal struct {
	BuyStrength  float64
	SellStrength float64
	Confidence   float64
	MarketRegime MarketRegime
	Timestamp    time.Time
}
