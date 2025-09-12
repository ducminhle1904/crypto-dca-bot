package bands

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/common"
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
	
	// %B thresholds for enhanced signal logic
	percentBOverbought float64  // %B overbought threshold (default 0.8)
	percentBOversold   float64  // %B oversold threshold (default 0.2)
	
	// Stateful components for efficient calculation
	ema        *common.EMA      // For EMA-based middle band
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
	lastPercentB float64 // Cached %B value
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
	return NewBollingerBandsWithThresholds(period, stdDev, maType, 0.9, 0.1)
}

// NewBollingerBandsWithThresholds creates a new Bollinger Bands indicator with custom %B thresholds
func NewBollingerBandsWithThresholds(period int, stdDev float64, maType MovingAverageType, overbought, oversold float64) *BollingerBands {
	bb := &BollingerBands{
		period:             period,
		stdDev:             stdDev,
		maType:             maType,
		percentBOverbought: overbought,
		percentBOversold:   oversold,
		values:             make([]float64, period), // Pre-allocated circular buffer
		writeIndex:         0,
		count:              0,
	}
	
	// Initialize EMA if using EMA mode
	if maType == MA_EMA {
		bb.ema = common.NewEMA(period)
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
		bb.ema.UpdateSingle(newPrice)
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

	// Calculate and cache %B value for the current middle band price
	bb.calculatePercentB(middleBand)

	return middleBand, nil
}

// calculatePercentB calculates and caches the %B value for given price
func (bb *BollingerBands) calculatePercentB(currentPrice float64) {
	bandWidth := bb.lastUpper - bb.lastLower
	if bandWidth == 0 {
		bb.lastPercentB = 0.5 // Neutral when bands are collapsed
		return
	}
	
	bb.lastPercentB = (currentPrice - bb.lastLower) / bandWidth
}

// GetPercentB returns the %B value for a given price (or last calculated if no price given)
func (bb *BollingerBands) GetPercentB(prices ...float64) float64 {
	if len(prices) > 0 {
		// Calculate %B for specific price
		price := prices[0]
		bandWidth := bb.lastUpper - bb.lastLower
		if bandWidth == 0 {
			return 0.5
		}
		return (price - bb.lastLower) / bandWidth
	}
	
	// Return cached %B value
	return bb.lastPercentB
}

// ShouldBuy determines if we should buy based on Bollinger Bands %B
func (bb *BollingerBands) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	_, err := bb.Calculate(data)
	if err != nil {
		return false, err
	}

	// Calculate %B for current price
	currentPercentB := bb.GetPercentB(current)
	
	// Buy when %B is in oversold territory
	// This indicates price is near or below lower band
	return currentPercentB <= bb.percentBOversold, nil
}

// ShouldSell determines if we should sell based on Bollinger Bands %B
func (bb *BollingerBands) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	_, err := bb.Calculate(data)
	if err != nil {
		return false, err
	}

	// Calculate %B for current price  
	currentPercentB := bb.GetPercentB(current)
	
	// Sell when %B is in overbought territory
	// This indicates price is near or above upper band
	return currentPercentB >= bb.percentBOverbought, nil
}

// GetSignalStrength returns the signal strength based on %B position
func (bb *BollingerBands) GetSignalStrength() float64 {
	percentB := bb.lastPercentB
	
	// Calculate signal strength based on %B position
	if percentB <= bb.percentBOversold {
		// Buy signal strength: stronger the further below oversold threshold
		// Max strength when %B = 0 (at lower band), min strength at oversold threshold
		if bb.percentBOversold == 0 {
			return 1.0 // Avoid division by zero
		}
		strength := (bb.percentBOversold - percentB) / bb.percentBOversold
		return math.Min(1.0, math.Max(0.0, strength))
		
	} else if percentB >= bb.percentBOverbought {
		// Sell signal strength: stronger the further above overbought threshold  
		// Max strength when %B = 1 (at upper band), min strength at overbought threshold
		if bb.percentBOverbought >= 1.0 {
			return 1.0 // Avoid division by zero
		}
		strength := (percentB - bb.percentBOverbought) / (1.0 - bb.percentBOverbought)
		return math.Min(1.0, math.Max(0.0, strength))
		
	} else {
		// No signal when %B is between thresholds
		return 0.0
	}
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

// ResetState resets the Bollinger Bands internal state for new data periods
func (bb *BollingerBands) ResetState() {
	// Reset stateful EMA if using EMA mode
	if bb.ema != nil {
		bb.ema.ResetState()
	}
	
	// Reset circular buffer and statistics
	bb.values = make([]float64, bb.period)
	bb.writeIndex = 0
	bb.sum = 0.0
	bb.sumSquares = 0.0
	bb.count = 0
	bb.initialized = false
	
	// Reset cached results
	bb.lastUpper = 0.0
	bb.lastMiddle = 0.0
	bb.lastLower = 0.0
	bb.lastPercentB = 0.0
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

// SetPercentBOverbought sets the %B overbought threshold
func (bb *BollingerBands) SetPercentBOverbought(threshold float64) {
	bb.percentBOverbought = threshold
}

// SetPercentBOversold sets the %B oversold threshold  
func (bb *BollingerBands) SetPercentBOversold(threshold float64) {
	bb.percentBOversold = threshold
}

// GetPercentBThresholds returns the current %B thresholds
func (bb *BollingerBands) GetPercentBThresholds() (overbought, oversold float64) {
	return bb.percentBOverbought, bb.percentBOversold
}

// GetBandsWithPercentB returns all band values plus current %B
func (bb *BollingerBands) GetBandsWithPercentB() (upper, middle, lower, percentB float64) {
	return bb.lastUpper, bb.lastMiddle, bb.lastLower, bb.lastPercentB
}

// IsOverbought returns true if current %B indicates overbought condition
func (bb *BollingerBands) IsOverbought() bool {
	return bb.lastPercentB >= bb.percentBOverbought
}

// IsOversold returns true if current %B indicates oversold condition  
func (bb *BollingerBands) IsOversold() bool {
	return bb.lastPercentB <= bb.percentBOversold
}

// GetTrendStrength returns trend strength based on %B position
// Values: Strong Downtrend (<0), Downtrend (0-0.2), Neutral (0.2-0.8), Uptrend (0.8-1), Strong Uptrend (>1)
func (bb *BollingerBands) GetTrendStrength() string {
	percentB := bb.lastPercentB
	
	if percentB > 1.0 {
		return "Strong Uptrend"
	} else if percentB >= bb.percentBOverbought {
		return "Uptrend"
	} else if percentB > bb.percentBOversold {
		return "Neutral"
	} else if percentB >= 0.0 {
		return "Downtrend"
	} else {
		return "Strong Downtrend"
	}
}
