package indicators

import (
	"errors"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DonchianChannels represents the Donchian Channels technical indicator
// Donchian Channels identify breakouts by tracking highest highs and lowest lows
// Used for regime detection to identify trend breakout points
type DonchianChannels struct {
	period      int
	
	// Circular buffers for efficient calculation
	highValues  []float64 // Buffer for highest highs
	lowValues   []float64 // Buffer for lowest lows
	writeIndex  int       // Current write position
	count       int       // Number of values in buffer
	initialized bool
	
	// Cached results
	lastUpper   float64   // Highest high over period
	lastLower   float64   // Lowest low over period
	lastMiddle  float64   // Middle line (upper + lower) / 2
}

// NewDonchianChannels creates a new Donchian Channels indicator
func NewDonchianChannels(period int) *DonchianChannels {
	return &DonchianChannels{
		period:     period,
		highValues: make([]float64, period),
		lowValues:  make([]float64, period),
		writeIndex: 0,
		count:      0,
		initialized: false,
	}
}

// Calculate calculates the Donchian Channel values
func (dc *DonchianChannels) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < dc.period {
		return 0, errors.New("insufficient data for Donchian Channels calculation")
	}

	if !dc.initialized {
		return dc.initialCalculation(data)
	}

	return dc.incrementalCalculation(data[len(data)-1])
}

// initialCalculation performs the initial Donchian Channels calculation
func (dc *DonchianChannels) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < dc.period {
		return 0, errors.New("insufficient data for Donchian Channels initialization")
	}

	// Initialize circular buffers with recent data
	startIdx := len(data) - dc.period
	for i := 0; i < dc.period; i++ {
		candle := data[startIdx+i]
		dc.highValues[i] = candle.High
		dc.lowValues[i] = candle.Low
	}
	
	dc.count = dc.period
	dc.writeIndex = 0
	dc.initialized = true
	
	// Calculate initial upper and lower bands
	dc.calculateChannels()
	
	// Return middle line as the primary value
	return dc.lastMiddle, nil
}

// incrementalCalculation updates Donchian Channels with new price data
func (dc *DonchianChannels) incrementalCalculation(newCandle types.OHLCV) (float64, error) {
	// Add new values to circular buffer
	dc.highValues[dc.writeIndex] = newCandle.High
	dc.lowValues[dc.writeIndex] = newCandle.Low
	
	// Move to next position
	dc.writeIndex = (dc.writeIndex + 1) % dc.period
	
	// Update count if not at full capacity
	if dc.count < dc.period {
		dc.count++
	}
	
	// Recalculate channels
	dc.calculateChannels()
	
	return dc.lastMiddle, nil
}

// calculateChannels finds the highest high and lowest low in the buffer
func (dc *DonchianChannels) calculateChannels() {
	if dc.count == 0 {
		return
	}
	
	// Find highest high and lowest low
	dc.lastUpper = dc.highValues[0]
	dc.lastLower = dc.lowValues[0]
	
	for i := 1; i < dc.count; i++ {
		if dc.highValues[i] > dc.lastUpper {
			dc.lastUpper = dc.highValues[i]
		}
		if dc.lowValues[i] < dc.lastLower {
			dc.lastLower = dc.lowValues[i]
		}
	}
	
	// Calculate middle line
	dc.lastMiddle = (dc.lastUpper + dc.lastLower) / 2.0
}

// ShouldBuy determines if price is breaking above upper Donchian Channel
func (dc *DonchianChannels) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	if len(data) == 0 {
		return false, errors.New("no data provided")
	}
	
	currentPrice := data[len(data)-1].Close
	
	// Buy signal when price breaks above upper channel
	// This indicates a potential bullish breakout
	return currentPrice > dc.lastUpper, nil
}

// ShouldSell determines if price is breaking below lower Donchian Channel
func (dc *DonchianChannels) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	if len(data) == 0 {
		return false, errors.New("no data provided")
	}
	
	currentPrice := data[len(data)-1].Close
	
	// Sell signal when price breaks below lower channel
	// This indicates a potential bearish breakout
	return currentPrice < dc.lastLower, nil
}

// GetSignalStrength returns the strength of the breakout signal
func (dc *DonchianChannels) GetSignalStrength() float64 {
	// Calculate channel width as percentage of middle price
	if dc.lastMiddle == 0 {
		return 0.0
	}
	
	channelWidth := (dc.lastUpper - dc.lastLower) / dc.lastMiddle
	
	// Wider channels indicate higher volatility and stronger potential breakouts
	// Normalize to 0-1 scale (assuming 10% channel width is maximum strength)
	strength := channelWidth / 0.10
	if strength > 1.0 {
		strength = 1.0
	}
	
	return strength
}

// GetName returns the indicator name
func (dc *DonchianChannels) GetName() string {
	return "Donchian Channels"
}

// GetRequiredPeriods returns minimum periods needed for calculation
func (dc *DonchianChannels) GetRequiredPeriods() int {
	return dc.period
}

// ResetState resets internal state for new data periods
func (dc *DonchianChannels) ResetState() {
	dc.writeIndex = 0
	dc.count = 0
	dc.initialized = false
	dc.lastUpper = 0
	dc.lastLower = 0
	dc.lastMiddle = 0
	
	// Clear buffers
	for i := range dc.highValues {
		dc.highValues[i] = 0
	}
	for i := range dc.lowValues {
		dc.lowValues[i] = 0
	}
}

// GetChannelValues returns upper, middle, and lower channel values
func (dc *DonchianChannels) GetChannelValues() (upper, middle, lower float64) {
	return dc.lastUpper, dc.lastMiddle, dc.lastLower
}

// GetChannelWidth returns the width of the channel as percentage of middle price
func (dc *DonchianChannels) GetChannelWidth() float64 {
	if dc.lastMiddle == 0 {
		return 0.0
	}
	return (dc.lastUpper - dc.lastLower) / dc.lastMiddle
}

// IsBreakoutAbove checks if current price is breaking above upper channel
func (dc *DonchianChannels) IsBreakoutAbove(currentPrice float64) bool {
	return currentPrice > dc.lastUpper
}

// IsBreakoutBelow checks if current price is breaking below lower channel  
func (dc *DonchianChannels) IsBreakoutBelow(currentPrice float64) bool {
	return currentPrice < dc.lastLower
}

// IsWithinChannel checks if current price is within the channel
func (dc *DonchianChannels) IsWithinChannel(currentPrice float64) bool {
	return currentPrice >= dc.lastLower && currentPrice <= dc.lastUpper
}

// GetBreakoutStrength returns the strength of a breakout (0-1 scale)
func (dc *DonchianChannels) GetBreakoutStrength(currentPrice float64) float64 {
	if dc.lastMiddle == 0 {
		return 0.0
	}
	
	var breakoutDistance float64
	
	if currentPrice > dc.lastUpper {
		// Breakout above
		breakoutDistance = currentPrice - dc.lastUpper
	} else if currentPrice < dc.lastLower {
		// Breakout below
		breakoutDistance = dc.lastLower - currentPrice
	} else {
		// No breakout
		return 0.0
	}
	
	// Normalize breakout distance by channel width
	channelWidth := dc.lastUpper - dc.lastLower
	if channelWidth == 0 {
		return 0.0
	}
	
	strength := breakoutDistance / channelWidth
	
	// Cap at 1.0 (100% of channel width is considered maximum strength)
	if strength > 1.0 {
		strength = 1.0
	}
	
	return strength
}

// GetTrendDirection returns trend direction based on price position relative to channels
// Returns: 1 = bullish (above upper), -1 = bearish (below lower), 0 = neutral (within)
func (dc *DonchianChannels) GetTrendDirection(currentPrice float64) int {
	if currentPrice > dc.lastUpper {
		return 1  // Bullish breakout
	} else if currentPrice < dc.lastLower {
		return -1 // Bearish breakout  
	} else {
		return 0  // Neutral (within channel)
	}
}
