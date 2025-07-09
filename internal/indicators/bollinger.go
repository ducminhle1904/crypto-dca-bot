package indicators

import (
	"errors"
	"math"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

// BollingerBands represents the Bollinger Bands technical indicator
type BollingerBands struct {
	period     int
	stdDev     float64
	lastUpper  float64
	lastMiddle float64
	lastLower  float64
}

// NewBollingerBands creates a new Bollinger Bands indicator
func NewBollingerBands(period int, stdDev float64) *BollingerBands {
	return &BollingerBands{
		period: period,
		stdDev: stdDev,
	}
}

// Calculate calculates the Bollinger Bands values
func (bb *BollingerBands) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < bb.period {
		return 0, errors.New("insufficient data for Bollinger Bands calculation")
	}

	// Calculate SMA (middle band)
	sma := 0.0
	for i := len(data) - bb.period; i < len(data); i++ {
		sma += data[i].Close
	}
	sma /= float64(bb.period)
	bb.lastMiddle = sma

	// Calculate standard deviation
	variance := 0.0
	for i := len(data) - bb.period; i < len(data); i++ {
		diff := data[i].Close - sma
		variance += diff * diff
	}
	variance /= float64(bb.period)
	stdDev := math.Sqrt(variance)

	// Calculate upper and lower bands
	bb.lastUpper = sma + (bb.stdDev * stdDev)
	bb.lastLower = sma - (bb.stdDev * stdDev)

	return sma, nil
}

// ShouldBuy determines if we should buy based on Bollinger Bands
func (bb *BollingerBands) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	_, err := bb.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy when price is near or below the lower band (oversold condition)
	// and price is starting to move up
	lowerBandThreshold := bb.lastLower * 1.01 // 1% above lower band
	return current <= lowerBandThreshold, nil
}

// ShouldSell determines if we should sell based on Bollinger Bands
func (bb *BollingerBands) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	_, err := bb.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell when price is near or above the upper band (overbought condition)
	// and price is starting to move down
	upperBandThreshold := bb.lastUpper * 0.99 // 1% below upper band
	return current >= upperBandThreshold, nil
}

// GetSignalStrength returns the signal strength based on position within bands
func (bb *BollingerBands) GetSignalStrength() float64 {
	// Calculate how far the current price is from the middle band
	// Normalize to a 0-1 range
	bandWidth := bb.lastUpper - bb.lastLower
	if bandWidth == 0 {
		return 0
	}

	// For buy signals, strength increases as price approaches lower band
	// For sell signals, strength increases as price approaches upper band
	// This is a simplified calculation
	return 0.5 // Default moderate strength
}

// GetName returns the indicator name
func (bb *BollingerBands) GetName() string {
	return "Bollinger Bands"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (bb *BollingerBands) GetRequiredPeriods() int {
	return bb.period
}

// GetBands returns the current band values
func (bb *BollingerBands) GetBands() (upper, middle, lower float64) {
	return bb.lastUpper, bb.lastMiddle, bb.lastLower
}
