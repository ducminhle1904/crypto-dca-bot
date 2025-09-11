package indicators

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

const (
	// DefaultSuperTrendPeriod is the default period value for ATR calculation
	DefaultSuperTrendPeriod = 14

	// DefaultSuperTrendMultiplier is the default multiplier value for bands calculation
	DefaultSuperTrendMultiplier = 2.5
)

// SuperTrend represents the SuperTrend technical indicator
// SuperTrend is a trend-following indicator that uses ATR to create dynamic support and resistance levels
type SuperTrend struct {
	period     int     // Period for ATR calculation
	multiplier float64 // Multiplier for ATR to create bands
	
	// Dependencies
	atr *ATR // ATR indicator for volatility measurement
	
	// State variables
	initialized      bool
	upTrend         bool    // Current trend direction
	previousClose   float64 // Previous candle close price
	finalUpperBand  float64 // Final upper band value
	finalLowerBand  float64 // Final lower band value
	superTrendValue float64 // Current SuperTrend value
	
	// Signal state
	lastSignalStrength float64
}

// NewSuperTrend creates a new SuperTrend indicator with default parameters
func NewSuperTrend() *SuperTrend {
	return NewSuperTrendWithParams(DefaultSuperTrendPeriod, DefaultSuperTrendMultiplier)
}

// NewSuperTrendWithParams creates a new SuperTrend indicator with custom parameters
func NewSuperTrendWithParams(period int, multiplier float64) *SuperTrend {
	return &SuperTrend{
		period:     period,
		multiplier: multiplier,
		atr:        NewATR(period),
		upTrend:    false, // Start assuming downtrend
	}
}

// Calculate calculates the SuperTrend value
func (st *SuperTrend) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < st.GetRequiredPeriods() {
		return 0, errors.New("insufficient data points for SuperTrend calculation")
	}

	// Calculate ATR first
	atrValue, err := st.atr.Calculate(data)
	if err != nil {
		return 0, err
	}

	current := data[len(data)-1]
	
	if !st.initialized {
		return st.initialCalculation(current, atrValue)
	}

	return st.incrementalCalculation(current, atrValue)
}

// initialCalculation performs the initial SuperTrend calculation
func (st *SuperTrend) initialCalculation(current types.OHLCV, atrValue float64) (float64, error) {
	// Calculate median price (HL2)
	medianPrice := (current.High + current.Low) / 2.0

	// Calculate basic bands
	// BasicUpperBand = (High + Low) / 2 + Multiplier * ATR
	basicUpperBand := medianPrice + (st.multiplier * atrValue)
	// BasicLowerBand = (High + Low) / 2 - Multiplier * ATR
	basicLowerBand := medianPrice - (st.multiplier * atrValue)

	// For first calculation, final bands equal basic bands
	st.finalUpperBand = basicUpperBand
	st.finalLowerBand = basicLowerBand
	
	// Initial SuperTrend value is lower band (assuming uptrend start)
	st.superTrendValue = st.finalLowerBand
	st.upTrend = true
	
	st.previousClose = current.Close
	st.initialized = true

	return st.superTrendValue, nil
}

// incrementalCalculation updates SuperTrend with new data point
func (st *SuperTrend) incrementalCalculation(current types.OHLCV, atrValue float64) (float64, error) {
	// Calculate median price (HL2)
	medianPrice := (current.High + current.Low) / 2.0

	// Calculate basic bands
	basicUpperBand := medianPrice + (st.multiplier * atrValue)
	basicLowerBand := medianPrice - (st.multiplier * atrValue)

	// Calculate final bands with conditions
	// FinalUpperBand = If (BasicUpperBand < PreviousFinalUpperBand) 
	//                  Or (PreviousClose > PreviousFinalUpperBand)
	//                  Then BasicUpperBand Else PreviousFinalUpperBand
	if (basicUpperBand < st.finalUpperBand) || (st.previousClose > st.finalUpperBand) {
		st.finalUpperBand = basicUpperBand
	}

	// FinalLowerBand = If (BasicLowerBand > PreviousFinalLowerBand)
	//                  Or (PreviousClose < PreviousFinalLowerBand) 
	//                  Then BasicLowerBand Else PreviousFinalLowerBand
	if (basicLowerBand > st.finalLowerBand) || (st.previousClose < st.finalLowerBand) {
		st.finalLowerBand = basicLowerBand
	}

	// Calculate SuperTrend value and determine trend
	// SuperTrend = If upTrend
	//              Then
	//                If (Close <= FinalUpperBand) Then FinalUpperBand Else FinalLowerBand
	//              Else  
	//                If (Close >= FinalLowerBand) Then FinalLowerBand Else FinalUpperBand
	if st.upTrend {
		if current.Close <= st.finalUpperBand {
			st.superTrendValue = st.finalUpperBand
		} else {
			st.superTrendValue = st.finalLowerBand
			st.upTrend = false // Switch to downtrend
		}
	} else {
		if current.Close >= st.finalLowerBand {
			st.superTrendValue = st.finalLowerBand
			st.upTrend = true // Switch to uptrend
		} else {
			st.superTrendValue = st.finalUpperBand
		}
	}

	// Update state
	st.previousClose = current.Close
	
	// Calculate signal strength based on price distance from SuperTrend
	st.calculateSignalStrength(current.Close)

	return st.superTrendValue, nil
}

// calculateSignalStrength calculates the signal strength based on price position relative to SuperTrend
func (st *SuperTrend) calculateSignalStrength(currentPrice float64) {
	if st.superTrendValue == 0 {
		st.lastSignalStrength = 0
		return
	}

	// Calculate percentage distance from SuperTrend line
	distance := math.Abs(currentPrice-st.superTrendValue) / st.superTrendValue
	
	// Normalize to 0-1 range, stronger signals when price is further from SuperTrend
	// Cap at 10% distance for normalization
	maxDistance := 0.10
	if distance > maxDistance {
		distance = maxDistance
	}
	
	st.lastSignalStrength = distance / maxDistance
}

// ShouldBuy determines if we should buy based on SuperTrend
func (st *SuperTrend) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	_, err := st.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy signal: uptrend is active and price is above SuperTrend line
	// Additional confirmation: price should be reasonably close to SuperTrend for entry
	if st.upTrend && current > st.superTrendValue {
		// Ensure price is not too far above SuperTrend (avoid buying at tops)
		distancePercent := (current - st.superTrendValue) / st.superTrendValue
		return distancePercent <= 0.05, nil // Within 5% of SuperTrend line
	}

	return false, nil
}

// ShouldSell determines if we should sell based on SuperTrend
func (st *SuperTrend) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	_, err := st.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell signal: downtrend is active and price is below SuperTrend line
	// Additional confirmation: price should be reasonably close to SuperTrend for exit
	if !st.upTrend && current < st.superTrendValue {
		// Ensure price is not too far below SuperTrend (avoid selling at bottoms)
		distancePercent := (st.superTrendValue - current) / st.superTrendValue
		return distancePercent <= 0.05, nil // Within 5% of SuperTrend line
	}

	return false, nil
}

// GetSignalStrength returns the current signal strength
func (st *SuperTrend) GetSignalStrength() float64 {
	return st.lastSignalStrength
}

// GetName returns the indicator name
func (st *SuperTrend) GetName() string {
	return "SuperTrend"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (st *SuperTrend) GetRequiredPeriods() int {
	return st.atr.GetRequiredPeriods()
}

// ResetState resets the SuperTrend internal state for new data periods
func (st *SuperTrend) ResetState() {
	st.atr.ResetState()
	st.initialized = false
	st.upTrend = false
	st.previousClose = 0.0
	st.finalUpperBand = 0.0
	st.finalLowerBand = 0.0
	st.superTrendValue = 0.0
	st.lastSignalStrength = 0.0
}

// GetSuperTrendValue returns the current SuperTrend value
func (st *SuperTrend) GetSuperTrendValue() float64 {
	return st.superTrendValue
}

// GetBands returns the current upper and lower bands
func (st *SuperTrend) GetBands() (upper, lower float64) {
	return st.finalUpperBand, st.finalLowerBand
}

// IsUpTrend returns true if currently in an uptrend
func (st *SuperTrend) IsUpTrend() bool {
	return st.upTrend
}

// GetTrendDirection returns trend direction as string
func (st *SuperTrend) GetTrendDirection() string {
	if st.upTrend {
		return "UP"
	}
	return "DOWN"
}

// GetPeriod returns the period used for ATR calculation
func (st *SuperTrend) GetPeriod() int {
	return st.period
}

// GetMultiplier returns the multiplier used for band calculation
func (st *SuperTrend) GetMultiplier() float64 {
	return st.multiplier
}
