package indicators

import (
	"errors"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// OBV represents the On-Balance Volume technical indicator
// OBV is a technical trading momentum indicator that uses volume flow
// to predict changes in asset price
type OBV struct {
	lastValue       float64
	lastClose       float64
	initialized     bool
	trendThreshold  float64 // Threshold for determining trend changes
}

// NewOBV creates a new OBV indicator
func NewOBV() *OBV {
	return &OBV{
		trendThreshold: 0.01, // 1% threshold for trend change signals
	}
}

// NewOBVWithThreshold creates a new OBV indicator with custom trend threshold
func NewOBVWithThreshold(threshold float64) *OBV {
	return &OBV{
		trendThreshold: threshold,
	}
}

// Calculate calculates the OBV value
// Formula:
//   - If Close[i] > Close[i-1], OBV[i] = OBV[i-1] + Volume[i]
//   - If Close[i] = Close[i-1], OBV[i] = OBV[i-1]
//   - If Close[i] < Close[i-1], OBV[i] = OBV[i-1] - Volume[i]
func (o *OBV) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < 2 {
		return 0, errors.New("insufficient data points for OBV calculation")
	}

	if !o.initialized {
		return o.initialCalculation(data)
	}

	return o.incrementalCalculation(data)
}

// initialCalculation calculates OBV from scratch using all available data
func (o *OBV) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < 2 {
		return 0, errors.New("not enough data for initial OBV calculation")
	}

	// Start with 0 OBV and process each price change
	o.lastValue = 0
	o.lastClose = data[0].Close

	// Process each candle starting from the second one
	for i := 1; i < len(data); i++ {
		current := data[i]
		
		if current.Close > o.lastClose {
			// Price increased: add volume
			o.lastValue += current.Volume
		} else if current.Close < o.lastClose {
			// Price decreased: subtract volume
			o.lastValue -= current.Volume
		}
		// If price unchanged, OBV remains the same
		
		o.lastClose = current.Close
	}

	o.initialized = true
	return o.lastValue, nil
}

// incrementalCalculation updates OBV with the latest data point
func (o *OBV) incrementalCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < 2 {
		return o.lastValue, nil
	}

	// Get the latest two candles for comparison
	latest := data[len(data)-1]
	previous := data[len(data)-2]
	
	// Only update if we have a new candle (different from what we processed last)
	if latest.Close != o.lastClose || previous.Close != o.lastClose {
		if latest.Close > previous.Close {
			// Price increased: add volume
			o.lastValue += latest.Volume
		} else if latest.Close < previous.Close {
			// Price decreased: subtract volume
			o.lastValue -= latest.Volume
		}
		// If price unchanged, OBV remains the same
		
		o.lastClose = latest.Close
	}

	return o.lastValue, nil
}

// ShouldBuy determines if we should buy based on OBV
// Buy signal: OBV is trending upward (positive momentum)
func (o *OBV) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	if len(data) < 10 {
		return false, errors.New("insufficient data for OBV trend analysis")
	}

	currentOBV, err := o.Calculate(data)
	if err != nil {
		return false, err
	}

	// Calculate OBV trend by comparing with previous periods
	// Look back 5-10 periods to determine trend
	lookback := min(10, len(data)/2)
	if lookback < 5 {
		lookback = 5
	}

	// Get OBV value from lookback periods ago
	pastData := data[:len(data)-lookback+1]
	pastOBV := 0.0
	
	// Calculate past OBV value
	if len(pastData) >= 2 {
		tempOBV := NewOBV()
		pastOBV, _ = tempOBV.Calculate(pastData)
	}

	// Buy if OBV is trending upward significantly
	if pastOBV != 0 {
		trend := (currentOBV - pastOBV) / abs(pastOBV)
		return trend > o.trendThreshold, nil
	}

	return false, nil
}

// ShouldSell determines if we should sell based on OBV
// Sell signal: OBV is trending downward (negative momentum)
func (o *OBV) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	if len(data) < 10 {
		return false, errors.New("insufficient data for OBV trend analysis")
	}

	currentOBV, err := o.Calculate(data)
	if err != nil {
		return false, err
	}

	// Calculate OBV trend by comparing with previous periods
	lookback := min(10, len(data)/2)
	if lookback < 5 {
		lookback = 5
	}

	// Get OBV value from lookback periods ago
	pastData := data[:len(data)-lookback+1]
	pastOBV := 0.0
	
	// Calculate past OBV value
	if len(pastData) >= 2 {
		tempOBV := NewOBV()
		pastOBV, _ = tempOBV.Calculate(pastData)
	}

	// Sell if OBV is trending downward significantly
	if pastOBV != 0 {
		trend := (currentOBV - pastOBV) / abs(pastOBV)
		return trend < -o.trendThreshold, nil
	}

	return false, nil
}

// GetSignalStrength returns the signal strength based on OBV trend
func (o *OBV) GetSignalStrength() float64 {
	// Signal strength is moderate as OBV is a momentum indicator
	// that works best in combination with other indicators
	return 0.5
}

// GetName returns the indicator name
func (o *OBV) GetName() string {
	return "OBV"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (o *OBV) GetRequiredPeriods() int {
	return 10 // Need at least 10 periods for trend analysis
}

// GetLastValue returns the last calculated OBV value
func (o *OBV) GetLastValue() float64 {
	return o.lastValue
}

// SetTrendThreshold sets the threshold for trend change detection
func (o *OBV) SetTrendThreshold(threshold float64) {
	o.trendThreshold = threshold
}

// ResetState resets the OBV internal state for new data periods
func (o *OBV) ResetState() {
	o.lastValue = 0.0
	o.lastClose = 0.0
	o.initialized = false
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
