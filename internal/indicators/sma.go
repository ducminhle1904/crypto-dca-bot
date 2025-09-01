package indicators

import (
	"errors"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// SMA represents the Simple Moving Average technical indicator
type SMA struct {
	period      int
	lastValue   float64
	values      []float64 // Circular buffer for rolling sum
	writeIndex  int       // Current write position in circular buffer
	sum         float64   // Rolling sum for O(1) calculation
	count       int       // Number of values in buffer
	initialized bool
}

// NewSMA creates a new SMA indicator
func NewSMA(period int) *SMA {
	return &SMA{
		period:     period,
		values:     make([]float64, period), // Pre-allocated circular buffer
		writeIndex: 0,
		count:      0,
	}
}

// Calculate calculates the SMA value using optimized rolling sum
func (s *SMA) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) == 0 {
		return 0, errors.New("no data provided")
	}

	if len(data) < s.period {
		return 0, errors.New("insufficient data for SMA calculation")
	}

	currentPrice := data[len(data)-1].Close

	if !s.initialized {
		return s.initialCalculation(data)
	}

	return s.incrementalCalculation(currentPrice)
}

// initialCalculation sets up the initial rolling sum
func (s *SMA) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < s.period {
		return 0, errors.New("insufficient data for SMA initialization")
	}

	// Initialize circular buffer and rolling sum
	startIdx := len(data) - s.period
	s.sum = 0.0
	
	for i := 0; i < s.period; i++ {
		price := data[startIdx+i].Close
		s.values[i] = price
		s.sum += price
	}
	
	s.count = s.period
	s.writeIndex = 0
	s.initialized = true
	
	s.lastValue = s.sum / float64(s.period)
	return s.lastValue, nil
}

// incrementalCalculation updates with new price using O(1) rolling sum
func (s *SMA) incrementalCalculation(newPrice float64) (float64, error) {
	// Update rolling sum using circular buffer
	oldPrice := s.values[s.writeIndex]
	
	// O(1) update of rolling sum
	s.sum = s.sum - oldPrice + newPrice
	
	// Update circular buffer
	s.values[s.writeIndex] = newPrice
	s.writeIndex = (s.writeIndex + 1) % s.period

	s.lastValue = s.sum / float64(s.period)
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
