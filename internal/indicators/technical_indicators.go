package indicators

import "time"

type TechnicalIndicator interface {
	Calculate(prices []float64) float64
	ShouldBuy(currentPrice float64, historicalData []PriceData) bool
	GetSignalStrength() float64
	GetName() string
}

type PriceData struct {
	Price     float64
	Volume    float64
	Timestamp time.Time
}
