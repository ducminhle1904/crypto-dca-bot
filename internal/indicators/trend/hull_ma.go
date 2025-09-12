package trend

import (
	"errors"
	"fmt"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// WMA represents a Weighted Moving Average
type WMA struct {
	period      int
	values      []float64
	weightSum   float64
	initialized bool
}

// NewWMA creates a new Weighted Moving Average
func NewWMA(period int) *WMA {
	// Validate period
	if period <= 0 {
		period = 1 // Minimum valid period
	}
	
	return &WMA{
		period:    period,
		values:    make([]float64, 0, period),
		weightSum: float64(period * (period + 1) / 2), // Sum of weights: n*(n+1)/2
	}
}

// Calculate calculates the WMA value
func (w *WMA) Calculate(price float64) float64 {
	// Add new value to the rolling window
	w.values = append(w.values, price)
	
	// Keep only the last 'period' values
	if len(w.values) > w.period {
		w.values = w.values[1:]
	}
	
	if len(w.values) < w.period {
		// Not enough data yet, return simple average
		sum := 0.0
		for _, val := range w.values {
			sum += val
		}
		return sum / float64(len(w.values))
	}
	
	// Calculate weighted average
	weightedSum := 0.0
	for i, val := range w.values {
		weight := float64(i + 1) // Weights: 1, 2, 3, ..., n
		weightedSum += val * weight
	}
	
	w.initialized = true
	return weightedSum / w.weightSum
}

// IsReady returns true if WMA has enough data
func (w *WMA) IsReady() bool {
	return len(w.values) >= w.period
}

// ResetState resets the WMA internal state for new data periods
func (w *WMA) ResetState() {
	w.values = make([]float64, 0, w.period)
	w.initialized = false
}

// HullMA represents the Hull Moving Average technical indicator
type HullMA struct {
	period       int
	halfPeriod   int
	sqrtPeriod   int
	
	// Internal WMAs
	wmaHalf      *WMA // WMA(period/2)
	wmaFull      *WMA // WMA(period)
	wmaSqrt      *WMA // WMA(sqrt(period))
	
	// State tracking
	lastValue    float64
	intermediate []float64 // Store intermediate values for final WMA
	initialized  bool
}

// NewHullMA creates a new Hull Moving Average indicator
func NewHullMA(period int) *HullMA {
	// Validate period
	if period <= 0 {
		// Use a sensible default if invalid period is provided
		period = 14
	}
	
	// Use proper rounding like the reference implementation
	halfPeriod := int(math.Round(float64(period) / 2.0))
	if halfPeriod <= 0 {
		halfPeriod = 1
	}
	
	sqrtPeriod := int(math.Round(math.Sqrt(float64(period))))
	if sqrtPeriod <= 0 {
		sqrtPeriod = 1
	}
	
	return &HullMA{
		period:       period,
		halfPeriod:   halfPeriod,
		sqrtPeriod:   sqrtPeriod,
		wmaHalf:      NewWMA(halfPeriod),
		wmaFull:      NewWMA(period),
		wmaSqrt:      NewWMA(sqrtPeriod),
		intermediate: make([]float64, 0),
	}
}

// Calculate calculates the Hull MA value
func (h *HullMA) Calculate(data []types.OHLCV) (float64, error) {
	minRequired := h.GetRequiredPeriods()
	if len(data) < minRequired {
		return 0, errors.New("insufficient data points for Hull MA calculation")
	}

	if !h.initialized {
		return h.initialCalculation(data)
	}

	return h.incrementalCalculation(data)
}

// initialCalculation calculates the initial Hull MA values
func (h *HullMA) initialCalculation(data []types.OHLCV) (float64, error) {
	minRequired := h.GetRequiredPeriods()
	if len(data) < minRequired {
		return 0, errors.New("not enough data for initial Hull MA calculation")
	}

	// Process all available data to build up the WMAs
	for i := 0; i < len(data); i++ {
		price := data[i].Close
		
		// Calculate WMA(period/2) and WMA(period)
		wmaHalfValue := h.wmaHalf.Calculate(price)
		wmaFullValue := h.wmaFull.Calculate(price)
		
		// Calculate intermediate value: 2 * WMA(period/2) - WMA(period)
		if h.wmaHalf.IsReady() && h.wmaFull.IsReady() {
			intermediateValue := 2*wmaHalfValue - wmaFullValue
			h.intermediate = append(h.intermediate, intermediateValue)
			
			// Calculate final Hull MA using WMA of intermediate values
			if len(h.intermediate) >= h.sqrtPeriod {
				// Keep only the required number of intermediate values
				if len(h.intermediate) > h.sqrtPeriod {
					h.intermediate = h.intermediate[len(h.intermediate)-h.sqrtPeriod:]
				}
				
				// Calculate WMA of intermediate values
				h.lastValue = h.wmaSqrt.Calculate(intermediateValue)
			}
		}
	}

	h.initialized = true
	return h.lastValue, nil
}

// incrementalCalculation updates Hull MA with the latest data point
func (h *HullMA) incrementalCalculation(data []types.OHLCV) (float64, error) {
	if len(data) == 0 {
		return h.lastValue, nil
	}

	// Get the latest price
	price := data[len(data)-1].Close

	// Calculate WMA(period/2) and WMA(period)
	wmaHalfValue := h.wmaHalf.Calculate(price)
	wmaFullValue := h.wmaFull.Calculate(price)

	// Calculate intermediate value: 2 * WMA(period/2) - WMA(period)
	if h.wmaHalf.IsReady() && h.wmaFull.IsReady() {
		intermediateValue := 2*wmaHalfValue - wmaFullValue
		
		// Calculate final Hull MA using WMA of intermediate values
		h.lastValue = h.wmaSqrt.Calculate(intermediateValue)
	}

	return h.lastValue, nil
}

// ShouldBuy determines if we should buy based on Hull MA
func (h *HullMA) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	hmaValue, err := h.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy when current price is above Hull MA (uptrend)
	// and Hull MA is trending upward
	return current > hmaValue, nil
}

// ShouldSell determines if we should sell based on Hull MA
func (h *HullMA) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	hmaValue, err := h.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell when current price is below Hull MA (downtrend)
	// and Hull MA is trending downward
	return current < hmaValue, nil
}

// GetSignalStrength returns the signal strength based on distance from Hull MA
func (h *HullMA) GetSignalStrength() float64 {
	// This could be enhanced by calculating the percentage distance from Hull MA
	// and the slope of the Hull MA to determine trend strength
	
	// For now, return a moderate strength
	// In a real implementation, you might calculate:
	// - Distance between price and Hull MA
	// - Recent slope of Hull MA
	// - Volatility adjustments
	return 0.5
}

// GetName returns the indicator name
func (h *HullMA) GetName() string {
	return "Hull MA"
}

// String returns the string representation of the Hull MA (like the reference implementation)
func (h *HullMA) String() string {
	return fmt.Sprintf("HMA(%d)", h.period)
}

// GetRequiredPeriods returns the minimum number of periods needed
// This accounts for the idle period like the reference implementation
func (h *HullMA) GetRequiredPeriods() int {
	// Hull MA needs: period (for WMA2) + sqrtPeriod (for WMA3) for full calculation
	// This matches the IdlePeriod calculation in the reference: wma2.IdlePeriod() + wma3.IdlePeriod()
	return h.period + h.sqrtPeriod
}

// GetIdlePeriod returns the initial period that Hull MA won't yield reliable results
// This matches the reference implementation's IdlePeriod method
func (h *HullMA) GetIdlePeriod() int {
	return h.period + h.sqrtPeriod
}

// GetLastValue returns the last calculated Hull MA value
func (h *HullMA) GetLastValue() float64 {
	return h.lastValue
}

// GetPeriod returns the period used for Hull MA calculation
func (h *HullMA) GetPeriod() int {
	return h.period
}

// GetHalfPeriod returns the half period used in calculation
func (h *HullMA) GetHalfPeriod() int {
	return h.halfPeriod
}

// GetSqrtPeriod returns the square root period used in calculation
func (h *HullMA) GetSqrtPeriod() int {
	return h.sqrtPeriod
}

// IsReady returns true if Hull MA has enough data for calculation
func (h *HullMA) IsReady() bool {
	return h.initialized && h.wmaSqrt.IsReady()
}

// GetTrend returns the trend direction based on Hull MA slope
// Returns: 1 for uptrend, -1 for downtrend, 0 for sideways
func (h *HullMA) GetTrend(data []types.OHLCV) int {
	if len(data) < 2 {
		return 0
	}
	
	// Calculate Hull MA for the last two periods to determine slope
	currentHMA, err := h.Calculate(data)
	if err != nil {
		return 0
	}
	
	// For a more accurate trend, we would need to store previous Hull MA values
	// For now, this is a simplified implementation
	if len(data) >= h.period+1 {
		previousData := data[:len(data)-1]
		previousHMA, err := h.Calculate(previousData)
		if err == nil {
			if currentHMA > previousHMA {
				return 1  // Uptrend
			} else if currentHMA < previousHMA {
				return -1 // Downtrend
			}
		}
	}
	
	return 0 // Sideways or unknown
}

// GetSlope returns the approximate slope of Hull MA
func (h *HullMA) GetSlope(data []types.OHLCV) float64 {
	trend := h.GetTrend(data)
	
	// This is a simplified slope calculation
	// In a real implementation, you would calculate the actual slope
	// using multiple previous Hull MA values
	return float64(trend) * 0.1 // Simplified slope value
}

// ResetState resets the Hull MA internal state for new data periods
func (h *HullMA) ResetState() {
	// Reset WMA components
	h.wmaHalf.ResetState()
	h.wmaFull.ResetState()
	h.wmaSqrt.ResetState()
	
	// Reset state values
	h.lastValue = 0.0
	h.initialized = false
}
