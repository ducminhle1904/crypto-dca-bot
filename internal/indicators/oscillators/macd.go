package oscillators

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/common"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// MACD represents the MACD technical indicator
type MACD struct {
	fastPeriod    int
	slowPeriod    int
	signalPeriod  int
	
	// Use stateful EMA instances for efficiency
	fastEMA       *common.EMA
	slowEMA       *common.EMA
	
	lastMACD      float64
	lastSignal    float64
	lastHistogram float64
	initialized   bool
}

// NewMACD creates a new MACD indicator
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) *MACD {
	return &MACD{
		fastPeriod:   fastPeriod,
		slowPeriod:   slowPeriod,
		signalPeriod: signalPeriod,
		fastEMA:      common.NewEMA(fastPeriod),
		slowEMA:      common.NewEMA(slowPeriod),
	}
}

// Calculate calculates the MACD value efficiently using stateful EMAs
func (m *MACD) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < m.slowPeriod {
		return 0, errors.New("insufficient data for MACD calculation")
	}

	// Calculate fast and slow EMAs using stateful instances
	fastValue, err := m.fastEMA.Calculate(data)
	if err != nil {
		return 0, err
	}

	slowValue, err := m.slowEMA.Calculate(data)
	if err != nil {
		return 0, err
	}

	// Calculate MACD line (fast EMA - slow EMA)
	macdLine := fastValue - slowValue
	m.lastMACD = macdLine

	// Calculate signal line (EMA of MACD values) using direct EMA formula
	if !m.initialized {
		// Initialize signal line with first MACD value
		m.lastSignal = macdLine
		m.initialized = true
	} else {
		// Update signal line using EMA formula: Signal = (MACD * alpha) + (Previous Signal * (1 - alpha))
		alpha := 2.0 / float64(m.signalPeriod+1)
		m.lastSignal = (macdLine * alpha) + (m.lastSignal * (1 - alpha))
	}

	// Calculate histogram (MACD - Signal)
	m.lastHistogram = macdLine - m.lastSignal

	return macdLine, nil
}

// ShouldBuy determines if we should buy based on MACD
func (m *MACD) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	_, err := m.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy when MACD line crosses above signal line (bullish crossover)
	// and histogram is positive
	return m.lastMACD > m.lastSignal && m.lastHistogram > 0, nil
}

// ShouldSell determines if we should sell based on MACD
func (m *MACD) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	_, err := m.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell when MACD line crosses below signal line (bearish crossover)
	// and histogram is negative
	return m.lastMACD < m.lastSignal && m.lastHistogram < 0, nil
}

// GetSignalStrength returns the signal strength based on MACD histogram
func (m *MACD) GetSignalStrength() float64 {
	// Normalize histogram value to a 0-1 range
	// This is a simplified approach - in practice you might want to use
	// historical histogram values to determine the range
	return math.Abs(m.lastHistogram) / 100.0 // Assuming typical histogram range
}

// GetName returns the indicator name
func (m *MACD) GetName() string {
	return "MACD (Optimized)"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (m *MACD) GetRequiredPeriods() int {
	return m.slowPeriod + m.signalPeriod
}

// GetLastValues returns MACD, Signal, and Histogram values for debugging
func (m *MACD) GetLastValues() (macd, signal, histogram float64) {
	return m.lastMACD, m.lastSignal, m.lastHistogram
}

// ResetState resets the MACD internal state for new data periods
func (m *MACD) ResetState() {
	// Reset stateful EMAs
	m.fastEMA.ResetState()
	m.slowEMA.ResetState()
	
	// Reset calculated values
	m.lastMACD = 0.0
	m.lastSignal = 0.0
	m.lastHistogram = 0.0
	m.initialized = false
}
