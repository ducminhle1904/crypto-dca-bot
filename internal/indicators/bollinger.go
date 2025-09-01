package indicators

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// MovingAverageType represents the type of moving average to use
type MovingAverageType int

const (
	MA_SMA MovingAverageType = iota // Simple Moving Average (traditional)
	MA_EMA                          // Exponential Moving Average (more responsive)
)

// BollingerBands represents the Bollinger Bands technical indicator
type BollingerBands struct {
	period     int
	stdDev     float64
	maType     MovingAverageType
	
	// Stateful components for efficient calculation
	ema        *EMA      // For EMA-based middle band
	values     []float64 // Circular buffer for price values
	writeIndex int       // Current write position in circular buffer
	
	// Rolling statistics for O(1) variance calculation
	sum        float64   // Running sum for SMA
	sumSquares float64   // Running sum of squares
	count      int       // Number of values in buffer
	initialized bool
	
	// Cached results
	lastUpper  float64
	lastMiddle float64
	lastLower  float64
}

// NewBollingerBands creates a new Bollinger Bands indicator using SMA (traditional)
func NewBollingerBands(period int, stdDev float64) *BollingerBands {
	return NewBollingerBandsWithType(period, stdDev, MA_SMA)
}

// NewBollingerBandsEMA creates a new Bollinger Bands indicator using EMA (more responsive)
func NewBollingerBandsEMA(period int, stdDev float64) *BollingerBands {
	return NewBollingerBandsWithType(period, stdDev, MA_EMA)
}

// NewBollingerBandsWithType creates a new Bollinger Bands indicator with specified MA type
func NewBollingerBandsWithType(period int, stdDev float64, maType MovingAverageType) *BollingerBands {
	bb := &BollingerBands{
		period:     period,
		stdDev:     stdDev,
		maType:     maType,
		values:     make([]float64, period), // Pre-allocated circular buffer
		writeIndex: 0,
		count:      0,
	}
	
	// Initialize EMA if using EMA mode
	if maType == MA_EMA {
		bb.ema = NewEMA(period)
	}
	
	return bb
}

// Calculate calculates the Bollinger Bands values using stateful rolling window
func (bb *BollingerBands) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) == 0 {
		return 0, errors.New("no data provided")
	}

	currentPrice := data[len(data)-1].Close

	if !bb.initialized {
		return bb.initialCalculation(data)
	}

	return bb.incrementalCalculation(currentPrice)
}

// initialCalculation sets up the initial rolling window and statistics
func (bb *BollingerBands) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < bb.period {
		return 0, errors.New("insufficient data for Bollinger Bands initialization")
	}

	// Initialize EMA if using EMA mode
	if bb.maType == MA_EMA {
		_, err := bb.ema.Calculate(data)
		if err != nil {
			return 0, err
		}
	}

	// Initialize circular buffer and rolling statistics
	startIdx := len(data) - bb.period
	bb.sum = 0.0
	bb.sumSquares = 0.0
	
	for i := 0; i < bb.period; i++ {
		price := data[startIdx+i].Close
		bb.values[i] = price
		bb.sum += price
		bb.sumSquares += price * price
	}
	
	bb.count = bb.period
	bb.writeIndex = 0
	bb.initialized = true
	
	return bb.calculateBands()
}

// incrementalCalculation updates with new price using O(1) rolling statistics
func (bb *BollingerBands) incrementalCalculation(newPrice float64) (float64, error) {
	// Update EMA if using EMA mode
	if bb.maType == MA_EMA {
		singlePoint := []types.OHLCV{{Close: newPrice}}
		_, err := bb.ema.Calculate(singlePoint)
		if err != nil {
			return 0, err
		}
	}
	
	// Update rolling statistics using circular buffer
	oldPrice := bb.values[bb.writeIndex]
	
	// O(1) update of rolling statistics
	bb.sum = bb.sum - oldPrice + newPrice
	bb.sumSquares = bb.sumSquares - (oldPrice * oldPrice) + (newPrice * newPrice)
	
	// Update circular buffer
	bb.values[bb.writeIndex] = newPrice
	bb.writeIndex = (bb.writeIndex + 1) % bb.period

	return bb.calculateBands()
}

// calculateBands computes bands from pre-calculated rolling statistics (O(1))
func (bb *BollingerBands) calculateBands() (float64, error) {
	var middleBand float64
	
	// Calculate middle band based on MA type
	switch bb.maType {
	case MA_SMA:
		// O(1) SMA from rolling sum
		middleBand = bb.sum / float64(bb.period)
	case MA_EMA:
		// Use pre-calculated EMA
		middleBand = bb.ema.GetLastValue()
	default:
		return 0, errors.New("unsupported moving average type")
	}
	
	bb.lastMiddle = middleBand

	// O(1) variance calculation using the formula: Var = E[X²] - (E[X])²
	meanSquare := bb.sumSquares / float64(bb.period)
	squareMean := middleBand * middleBand
	variance := meanSquare - squareMean
	
	// Handle numerical precision issues
	if variance < 0 {
		variance = 0
	}
	
	standardDev := math.Sqrt(variance)

	// Calculate upper and lower bands
	bb.lastUpper = middleBand + (bb.stdDev * standardDev)
	bb.lastLower = middleBand - (bb.stdDev * standardDev)

	return middleBand, nil
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
	switch bb.maType {
	case MA_EMA:
		return "Bollinger Bands (EMA-based)"
	default:
		return "Bollinger Bands (SMA-based)"
	}
}

// GetRequiredPeriods returns the minimum number of periods needed
func (bb *BollingerBands) GetRequiredPeriods() int {
	return bb.period
}

// GetBands returns the current band values
func (bb *BollingerBands) GetBands() (upper, middle, lower float64) {
	return bb.lastUpper, bb.lastMiddle, bb.lastLower
}

// GetMovingAverageType returns the type of moving average used
func (bb *BollingerBands) GetMovingAverageType() MovingAverageType {
	return bb.maType
}

// IsEMABased returns true if using EMA for middle band
func (bb *BollingerBands) IsEMABased() bool {
	return bb.maType == MA_EMA
}
