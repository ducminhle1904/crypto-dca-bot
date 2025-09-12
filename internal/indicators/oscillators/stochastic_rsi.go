package oscillators

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// StochasticRSI represents the Stochastic Relative Strength Index technical indicator
// Stochastic RSI is a momentum indicator that uses RSI values to create an oscillator
// that ranges between 0 and 1 (or 0 and 100). It helps identify overbought and oversold
// conditions more sensitively than regular RSI.
type StochasticRSI struct {
	period      int       // Period for RSI calculation and min/max window
	overbought  float64   // Overbought threshold (typically 0.8 or 80)
	oversold    float64   // Oversold threshold (typically 0.2 or 20)
	lastValue   float64   // Last calculated Stochastic RSI value
	initialized bool      // Whether the indicator has been initialized
	
	// RSI calculator
	rsi *RSI
	
	// Historical RSI values for min/max calculation
	rsiValues   []float64
	dataPoints  int
}

// NewStochasticRSI creates a new Stochastic RSI indicator with default parameters
func NewStochasticRSI() *StochasticRSI {
	return NewStochasticRSIWithPeriod(14)
}

// NewStochasticRSIWithPeriod creates a new Stochastic RSI indicator with custom period
func NewStochasticRSIWithPeriod(period int) *StochasticRSI {
	return &StochasticRSI{
		period:     period,
		overbought: 80.0,
		oversold:   20.0,
		rsi:        NewRSI(period),
		rsiValues:  make([]float64, 0, period),
	}
}

// NewStochasticRSIWithThresholds creates a new Stochastic RSI indicator with custom thresholds
func NewStochasticRSIWithThresholds(period int, overbought, oversold float64) *StochasticRSI {
	return &StochasticRSI{
		period:     period,
		overbought: overbought,
		oversold:   oversold,
		rsi:        NewRSI(period),
		rsiValues:  make([]float64, 0, period),
	}
}

// Calculate calculates the Stochastic RSI value
// Formula: Stochastic RSI = (RSI - Min(RSI, period)) / (Max(RSI, period) - Min(RSI, period)) * 100
func (s *StochasticRSI) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < s.GetRequiredPeriods() {
		return 0, errors.New("insufficient data points for Stochastic RSI calculation")
	}

	// First, calculate the current RSI value
	rsiValue, err := s.rsi.Calculate(data)
	if err != nil {
		return 0, err
	}

	if !s.initialized {
		return s.initialCalculation(data, rsiValue)
	}

	return s.incrementalCalculation(data, rsiValue)
}

// initialCalculation performs the initial calculation of Stochastic RSI
func (s *StochasticRSI) initialCalculation(data []types.OHLCV, currentRSI float64) (float64, error) {
	// Calculate RSI for enough data points to build the initial window
	requiredData := len(data) - s.rsi.GetRequiredPeriods() + 1
	if requiredData < s.period {
		return 0, errors.New("insufficient data for initial Stochastic RSI calculation")
	}

	// Build initial RSI values array
	s.rsiValues = make([]float64, 0, s.period)
	
	// Calculate RSI values for the rolling window
	for i := len(data) - requiredData; i < len(data); i++ {
		subData := data[:i+1]
		if len(subData) >= s.rsi.GetRequiredPeriods() {
			rsiVal, err := s.rsi.Calculate(subData)
			if err == nil {
				s.rsiValues = append(s.rsiValues, rsiVal)
				if len(s.rsiValues) > s.period {
					s.rsiValues = s.rsiValues[1:] // Keep only the last 'period' values
				}
			}
		}
	}

	if len(s.rsiValues) < s.period {
		return 0, errors.New("insufficient RSI values for Stochastic RSI calculation")
	}

	// Calculate Stochastic RSI
	s.lastValue = s.calculateStochasticRSI(s.rsiValues)
	s.initialized = true
	s.dataPoints = len(data)
	
	return s.lastValue, nil
}

// incrementalCalculation performs incremental calculation for efficiency
func (s *StochasticRSI) incrementalCalculation(data []types.OHLCV, currentRSI float64) (float64, error) {
	// Update the RSI values window
	if len(s.rsiValues) >= s.period {
		// Remove the oldest value
		s.rsiValues = s.rsiValues[1:]
	}
	
	// Add the new RSI value
	s.rsiValues = append(s.rsiValues, currentRSI)
	
	if len(s.rsiValues) < s.period {
		return s.lastValue, nil
	}

	// Calculate new Stochastic RSI
	s.lastValue = s.calculateStochasticRSI(s.rsiValues)
	s.dataPoints = len(data)
	
	return s.lastValue, nil
}

// calculateStochasticRSI calculates the Stochastic RSI from RSI values
func (s *StochasticRSI) calculateStochasticRSI(rsiValues []float64) float64 {
	if len(rsiValues) == 0 {
		return 0
	}

	// Find min and max RSI values
	minRSI := rsiValues[0]
	maxRSI := rsiValues[0]
	
	for _, value := range rsiValues {
		if value < minRSI {
			minRSI = value
		}
		if value > maxRSI {
			maxRSI = value
		}
	}

	// Avoid division by zero
	if maxRSI == minRSI {
		return 50.0 // Return neutral value when no variation
	}

	// Calculate Stochastic RSI
	currentRSI := rsiValues[len(rsiValues)-1]
	stochasticRSI := ((currentRSI - minRSI) / (maxRSI - minRSI)) * 100

	return stochasticRSI
}

// ShouldBuy determines if a buy signal should be generated
// Buy signal when Stochastic RSI crosses above oversold level
func (s *StochasticRSI) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	stochRSI, err := s.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy signal when crossing above oversold threshold
	return stochRSI > s.oversold && stochRSI < (s.oversold + 10), nil
}

// ShouldSell determines if a sell signal should be generated
// Sell signal when Stochastic RSI crosses below overbought level
func (s *StochasticRSI) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	stochRSI, err := s.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell signal when crossing below overbought threshold
	return stochRSI < s.overbought && stochRSI > (s.overbought - 10), nil
}

// GetSignalStrength returns the strength of the current signal (0-1)
func (s *StochasticRSI) GetSignalStrength() float64 {
	if s.lastValue <= s.oversold {
		// Stronger buy signal the further below oversold
		return math.Min(1.0, (s.oversold - s.lastValue) / s.oversold)
	}
	if s.lastValue >= s.overbought {
		// Stronger sell signal the further above overbought
		return math.Min(1.0, (s.lastValue - s.overbought) / (100 - s.overbought))
	}
	return 0
}

// GetName returns the name of the indicator
func (s *StochasticRSI) GetName() string {
	return "Stochastic RSI"
}

// GetRequiredPeriods returns the minimum number of data points required
func (s *StochasticRSI) GetRequiredPeriods() int {
	return s.rsi.GetRequiredPeriods() + s.period - 1
}

// SetOversold sets the oversold threshold
func (s *StochasticRSI) SetOversold(threshold float64) {
	s.oversold = threshold
}

// SetOverbought sets the overbought threshold
func (s *StochasticRSI) SetOverbought(threshold float64) {
	s.overbought = threshold
}

// ResetState resets the internal state of the indicator
func (s *StochasticRSI) ResetState() {
	s.lastValue = 0.0
	s.initialized = false
	s.dataPoints = 0
	s.rsiValues = make([]float64, 0, s.period)
	s.rsi.ResetState()
}

// GetLastValue returns the last calculated value
func (s *StochasticRSI) GetLastValue() float64 {
	return s.lastValue
}

// GetPeriod returns the period setting
func (s *StochasticRSI) GetPeriod() int {
	return s.period
}

// GetOverbought returns the overbought threshold
func (s *StochasticRSI) GetOverbought() float64 {
	return s.overbought
}

// GetOversold returns the oversold threshold
func (s *StochasticRSI) GetOversold() float64 {
	return s.oversold
}
