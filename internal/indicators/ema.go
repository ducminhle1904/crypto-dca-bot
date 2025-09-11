package indicators

import (
	"errors"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// EMA represents the Exponential Moving Average technical indicator
type EMA struct {
	period      int
	alpha       float64
	lastValue   float64
	initialized bool
}

// NewEMA creates a new EMA indicator
func NewEMA(period int) *EMA {
	return &EMA{
		period: period,
		alpha:  2.0 / float64(period+1), // Standard EMA alpha calculation
	}
}

// Calculate calculates the EMA value
func (e *EMA) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < e.period {
		return 0, errors.New("insufficient data for EMA calculation")
	}

	if !e.initialized {
		return e.initialCalculation(data)
	}

	return e.incrementalCalculation(data)
}

// initialCalculation calculates the first EMA value using SMA as the initial value
func (e *EMA) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < e.period {
		return 0, errors.New("not enough data for initial EMA calculation")
	}

	// Use SMA of the first 'period' values as the initial EMA
	sum := 0.0
	startIdx := len(data) - e.period
	for i := startIdx; i < len(data); i++ {
		sum += data[i].Close
	}

	e.lastValue = sum / float64(e.period)
	e.initialized = true

	return e.lastValue, nil
}

// incrementalCalculation updates EMA with the latest price
func (e *EMA) incrementalCalculation(data []types.OHLCV) (float64, error) {
	if len(data) == 0 {
		return e.lastValue, nil
	}

	// Get the latest close price
	latestPrice := data[len(data)-1].Close

	// Apply EMA formula: EMA = (Close * Alpha) + (Previous EMA * (1 - Alpha))
	e.lastValue = (latestPrice * e.alpha) + (e.lastValue * (1 - e.alpha))

	return e.lastValue, nil
}

// UpdateSingle updates the EMA with a single data point (for incremental indicators)
func (e *EMA) UpdateSingle(value float64) float64 {
	if !e.initialized {
		// Initialize with first value as starting point
		e.lastValue = value
		e.initialized = true
	} else {
		// Apply EMA formula: EMA = (Value * Alpha) + (Previous EMA * (1 - Alpha))
		e.lastValue = (value * e.alpha) + (e.lastValue * (1 - e.alpha))
	}
	
	return e.lastValue
}

// IsInitialized returns whether the EMA has been initialized
func (e *EMA) IsInitialized() bool {
	return e.initialized
}

// ShouldBuy determines if we should buy based on EMA
func (e *EMA) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	ema, err := e.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy when current price is above EMA (uptrend)
	return current > ema, nil
}

// ShouldSell determines if we should sell based on EMA
func (e *EMA) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	ema, err := e.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell when current price is below EMA (downtrend)
	return current < ema, nil
}

// GetSignalStrength returns the signal strength based on distance from EMA
func (e *EMA) GetSignalStrength() float64 {
	// This could be enhanced by calculating the percentage distance from EMA
	// For now, return a moderate strength
	return 0.4
}

// GetName returns the indicator name
func (e *EMA) GetName() string {
	return "EMA"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (e *EMA) GetRequiredPeriods() int {
	return e.period
}

// GetLastValue returns the last calculated EMA value
func (e *EMA) GetLastValue() float64 {
	return e.lastValue
}

// ResetState resets the EMA internal state for new data periods
func (e *EMA) ResetState() {
	e.lastValue = 0.0
	e.initialized = false
}