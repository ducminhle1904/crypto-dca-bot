package base

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/common"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// ATR represents the Average True Range technical indicator
// ATR measures market volatility by decomposing the entire range of an asset price for that period
type ATR struct {
	period      int
	ema         *common.EMA // Using EMA for ATR smoothing (Wilder's smoothing)
	lastClose   float64
	initialized bool
}

// NewATR creates a new ATR indicator
func NewATR(period int) *ATR {
	return &ATR{
		period: period,
		ema:    common.NewEMA(period),
	}
}

// Calculate calculates the ATR value
func (a *ATR) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < a.period {
		return 0, errors.New("insufficient data points for ATR calculation")
	}

	if !a.initialized {
		return a.initialCalculation(data)
	}

	return a.incrementalCalculation(data)
}

// initialCalculation calculates the initial ATR values
func (a *ATR) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < a.period {
		return 0, errors.New("not enough data for initial ATR calculation")
	}

	// Process all data to build up the EMA
	for i := 0; i < len(data); i++ {
		candle := data[i]
		
		// Calculate True Range
		var trueRange float64
		if i > 0 {
			trueRange = a.calculateTrueRange(candle, a.lastClose)
		} else {
			trueRange = candle.High - candle.Low // First candle
		}
		
		// Update EMA with true range
		a.ema.UpdateSingle(trueRange)
		
		a.lastClose = candle.Close
	}

	a.initialized = true
	return a.ema.GetLastValue(), nil
}

// incrementalCalculation updates ATR with the latest data point
func (a *ATR) incrementalCalculation(data []types.OHLCV) (float64, error) {
	if len(data) == 0 {
		return a.ema.GetLastValue(), nil
	}

	// Get the latest candle
	latest := data[len(data)-1]
	
	// Calculate True Range
	trueRange := a.calculateTrueRange(latest, a.lastClose)
	
	// Update EMA with true range
	atrValue := a.ema.UpdateSingle(trueRange)

	a.lastClose = latest.Close
	return atrValue, nil
}

// calculateTrueRange calculates the True Range for a given candle
func (a *ATR) calculateTrueRange(current types.OHLCV, prevClose float64) float64 {
	// True Range = max(High-Low, abs(High-PrevClose), abs(Low-PrevClose))
	hl := current.High - current.Low
	hc := math.Abs(current.High - prevClose)
	lc := math.Abs(current.Low - prevClose)
	
	return math.Max(hl, math.Max(hc, lc))
}

// GetName returns the indicator name
func (a *ATR) GetName() string {
	return "ATR"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (a *ATR) GetRequiredPeriods() int {
	return a.period + 1 // Need extra period for True Range calculation
}

// GetIdlePeriod returns the initial period that ATR won't yield reliable results
func (a *ATR) GetIdlePeriod() int {
	return a.period + 1 // Same as required periods for ATR
}

// GetLastValue returns the last calculated ATR value
func (a *ATR) GetLastValue() float64 {
	if a.ema == nil {
		return 0
	}
	return a.ema.GetLastValue()
}

// GetPeriod returns the period used for ATR calculation
func (a *ATR) GetPeriod() int {
	return a.period
}

// ResetState resets the ATR internal state for new data periods
func (a *ATR) ResetState() {
	a.ema.ResetState()
	a.lastClose = 0.0
	a.initialized = false
}