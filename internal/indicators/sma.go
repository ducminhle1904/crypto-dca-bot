package indicators

type MovingAverage struct {
	period int
	prices []float64
}

// NewSMA creates a new Simple Moving Average instance with the given period
func NewSMA(period int) *MovingAverage {
	return &MovingAverage{
		period: period,
		prices: make([]float64, 0),
	}
}

// Calculate computes the SMA over the provided price slice
func (ma *MovingAverage) Calculate(prices []float64) float64 {
	if len(prices) < ma.period {
		return 0
	}

	recent := prices[len(prices)-ma.period:]
	sum := 0.0
	for _, price := range recent {
		sum += price
	}

	return sum / float64(ma.period)
}

// GetTrendStrength returns a multiplier representing trend strength
func (ma *MovingAverage) GetTrendStrength(sma50, sma200, currentPrice float64) float64 {
	if sma200 == 0 {
		return 1.0
	}

	// If price is above both SMAs and SMA50 > SMA200 → strong uptrend
	if currentPrice > sma50 && sma50 > sma200 {
		return 1.0 + ((sma50 - sma200) / sma200) // Increase position size
	}

	// If price is below both SMAs → reduce activity
	if currentPrice < sma50 && currentPrice < sma200 {
		return 0.5
	}

	return 1.0 // Neutral zone
}
