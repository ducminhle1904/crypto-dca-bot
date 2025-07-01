package indicators

import (
	"errors"
	"math"
)

// RSI calculates the Relative Strength Index
type RSI struct {
	period int
	gains  []float64
	losses []float64
}

// NewRSI creates a new RSI instance with the given period
func NewRSI(period int) *RSI {
	return &RSI{
		period: period,
		gains:  make([]float64, 0),
		losses: make([]float64, 0),
	}
}

// Calculate computes the RSI value based on the given price slice
func (r *RSI) Calculate(prices []float64) (float64, error) {
	if len(prices) < r.period+1 {
		return 0, errors.New("insufficient data for RSI calculation")
	}

	// Calculate price changes
	changes := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		changes[i-1] = prices[i] - prices[i-1]
	}

	// Separate gains and losses
	gains := make([]float64, len(changes))
	losses := make([]float64, len(changes))
	for i, change := range changes {
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = math.Abs(change)
		}
	}

	// Calculate average gains and losses
	avgGain := r.sma(gains[len(gains)-r.period:])
	avgLoss := r.sma(losses[len(losses)-r.period:])

	if avgLoss == 0 {
		return 100, nil
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi, nil
}

// ShouldBuy returns true if the RSI indicates an oversold condition
func (r *RSI) ShouldBuy(currentRSI float64) bool {
	return currentRSI < 30
}

// sma computes the Simple Moving Average of the given values
func (r *RSI) sma(values []float64) float64 {
	sum := 0.0
	for _, value := range values {
		sum += value
	}
	return sum / float64(len(values))
}
