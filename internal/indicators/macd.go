package indicators

import (
	"errors"
	"math"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

// MACD represents the MACD technical indicator
type MACD struct {
	fastPeriod    int
	slowPeriod    int
	signalPeriod  int
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
	}
}

// Calculate calculates the MACD value
func (m *MACD) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < m.slowPeriod {
		return 0, errors.New("insufficient data for MACD calculation")
	}

	// Calculate fast EMA
	fastEMA, err := m.calculateEMA(data, m.fastPeriod)
	if err != nil {
		return 0, err
	}

	// Calculate slow EMA
	slowEMA, err := m.calculateEMA(data, m.slowPeriod)
	if err != nil {
		return 0, err
	}

	// Calculate MACD line
	macdLine := fastEMA - slowEMA
	m.lastMACD = macdLine

	// Calculate signal line (EMA of MACD)
	if !m.initialized {
		m.lastSignal = macdLine
		m.initialized = true
	} else {
		alpha := 2.0 / float64(m.signalPeriod+1)
		m.lastSignal = (macdLine * alpha) + (m.lastSignal * (1 - alpha))
	}

	// Calculate histogram
	m.lastHistogram = macdLine - m.lastSignal

	return macdLine, nil
}

// calculateEMA calculates the Exponential Moving Average
func (m *MACD) calculateEMA(data []types.OHLCV, period int) (float64, error) {
	if len(data) < period {
		return 0, errors.New("insufficient data for EMA calculation")
	}

	// Use SMA for the first period values
	sma := 0.0
	for i := len(data) - period; i < len(data); i++ {
		sma += data[i].Close
	}
	sma /= float64(period)

	// Calculate EMA
	alpha := 2.0 / float64(period+1)
	ema := sma

	// Apply EMA formula to recent data
	for i := len(data) - period; i < len(data); i++ {
		ema = (data[i].Close * alpha) + (ema * (1 - alpha))
	}

	return ema, nil
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
	return "MACD"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (m *MACD) GetRequiredPeriods() int {
	return m.slowPeriod + m.signalPeriod
}
