package strategy

import (
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
	"math"
)

type AdaptiveRiskManager struct {
	basePositionSize float64
	maxPositionSize  float64
	minPositionSize  float64
	atrPeriod        int
	atrMultiplier    float64
	stopLossATR      float64
}

func NewAdaptiveRiskManager(baseSize float64) *AdaptiveRiskManager {
	return &AdaptiveRiskManager{
		basePositionSize: baseSize,
		maxPositionSize:  baseSize * 3.0,
		minPositionSize:  baseSize * 0.25,
		atrPeriod:        14,
		atrMultiplier:    2.0,
		stopLossATR:      3.0,
	}
}

func (r *AdaptiveRiskManager) CalculatePositionSize(
	data []types.OHLCV,
	signalStrength float64,
) float64 {
	if len(data) < r.atrPeriod {
		return r.basePositionSize
	}

	//Calculating the ATR to determine volatility
	atr := r.calculateATR(data)
	avgPrice := data[len(data)-1].Close
	volatility := atr / avgPrice

	// Base position size
	positionSize := r.basePositionSize

	//Adjusting for signal strength
	positionSize *= (0.5 + signalStrength) //From 50% to 150% of the base size

	//Adjusting for volatility (inverse relationship)
	volatilityAdjustment := 1.0 / (1.0 + volatility*10)
	positionSize *= volatilityAdjustment

	// Applying restrictions
	if positionSize > r.maxPositionSize {
		positionSize = r.maxPositionSize
	}
	if positionSize < r.minPositionSize {
		positionSize = r.minPositionSize
	}

	return positionSize
}

func (r *AdaptiveRiskManager) CalculateStopLoss(
	entryPrice float64,
	data []types.OHLCV,
) float64 {
	atr := r.calculateATR(data)
	return entryPrice - (atr * r.stopLossATR)
}

func (r *AdaptiveRiskManager) calculateATR(data []types.OHLCV) float64 {
	if len(data) < r.atrPeriod+1 {
		return 0
	}

	trueRanges := make([]float64, 0, r.atrPeriod)

	for i := len(data) - r.atrPeriod; i < len(data); i++ {
		current := data[i]
		previous := data[i-1]

		tr1 := current.High - current.Low
		tr2 := math.Abs(current.High - previous.Close)
		tr3 := math.Abs(current.Low - previous.Close)

		trueRange := math.Max(tr1, math.Max(tr2, tr3))
		trueRanges = append(trueRanges, trueRange)
	}

	// Простое скользящее среднее TR
	sum := 0.0
	for _, tr := range trueRanges {
		sum += tr
	}

	return sum / float64(len(trueRanges))
}
