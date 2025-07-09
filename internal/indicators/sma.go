package indicators

import (
	"errors"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

// SMA represents the Simple Moving Average technical indicator
type SMA struct {
	period    int
	lastValue float64
}

// NewSMA creates a new SMA indicator
func NewSMA(period int) *SMA {
	return &SMA{
		period: period,
	}
}

// Calculate calculates the SMA value
func (s *SMA) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < s.period {
		return 0, errors.New("insufficient data for SMA calculation")
	}

	sum := 0.0
	for i := len(data) - s.period; i < len(data); i++ {
		sum += data[i].Close
	}

	s.lastValue = sum / float64(s.period)
	return s.lastValue, nil
}

// ShouldBuy determines if we should buy based on SMA
func (s *SMA) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	sma, err := s.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy when current price is above SMA (uptrend)
	return current > sma, nil
}

// ShouldSell determines if we should sell based on SMA
func (s *SMA) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	sma, err := s.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell when current price is below SMA (downtrend)
	return current < sma, nil
}

// GetSignalStrength returns the signal strength based on distance from SMA
func (s *SMA) GetSignalStrength() float64 {
	// This is a simplified calculation
	// In practice, you might want to normalize based on historical volatility
	return 0.3 // Default moderate strength for SMA
}

// GetName returns the indicator name
func (s *SMA) GetName() string {
	return "SMA"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (s *SMA) GetRequiredPeriods() int {
	return s.period
}
