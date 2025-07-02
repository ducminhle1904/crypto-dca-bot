package indicators

// BollingerBands represents the Bollinger Bands indicator
type BollingerBands struct {
	period         int
	stdDevMultiple float64
}

// NewBollingerBands creates a new BollingerBands instance with the given period and standard deviation multiplier
func NewBollingerBands(period int, stdDev float64) *BollingerBands {
	return &BollingerBands{
		period:         period,
		stdDevMultiple: stdDev,
	}
}

// Calculate computes the upper, middle, and lower Bollinger Bands, and the BB% (price position within the bands)
func (bb *BollingerBands) Calculate(prices []float64) (upper, middle, lower, bbPercent float64) {
	if len(prices) < bb.period {
		return 0, 0, 0, 0
	}

	recent := prices[len(prices)-bb.period:]
	middle = bb.sma(recent)
	stdDev := bb.standardDeviation(recent, middle)

	upper = middle + (bb.stdDevMultiple * stdDev)
	lower = middle - (bb.stdDevMultiple * stdDev)

	currentPrice := prices[len(prices)-1]
	if upper == lower {
		bbPercent = 50
	} else {
		bbPercent = ((currentPrice - lower) / (upper - lower)) * 100
	}

	return upper, middle, lower, bbPercent
}

// ShouldBuy returns true if the price is near the lower Bollinger Band
func (bb *BollingerBands) ShouldBuy(bbPercent float64) bool {
	return bbPercent < 20 // Price is close to the lower band
}
