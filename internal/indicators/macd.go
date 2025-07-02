package indicators

type MACD struct {
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
}

// NewMACD creates a new MACD instance with specified fast, slow, and signal periods
func NewMACD(fast, slow, signal int) *MACD {
	return &MACD{
		fastPeriod:   fast,
		slowPeriod:   slow,
		signalPeriod: signal,
	}
}

// Calculate computes the MACD line, signal line, and histogram
func (m *MACD) Calculate(prices []float64) (macdLine, signalLine, histogram float64) {
	if len(prices) < m.slowPeriod {
		return 0, 0, 0
	}

	fastEMA := m.ema(prices, m.fastPeriod)
	slowEMA := m.ema(prices, m.slowPeriod)
	macdLine = fastEMA - slowEMA

	// For simplicity, we use SMA for the signal line
	// In real-world usage, EMA is preferred
	macdHistory := []float64{macdLine} // Normally, you’d keep a history of MACD values
	signalLine = m.sma(macdHistory)
	histogram = macdLine - signalLine

	return macdLine, signalLine, histogram
}

// ShouldBuy returns true on a bullish crossover:
// when the MACD line crosses the signal line from below
func (m *MACD) ShouldBuy(macdLine, signalLine, prevMACD, prevSignal float64) bool {
	return prevMACD <= prevSignal && macdLine > signalLine
}
