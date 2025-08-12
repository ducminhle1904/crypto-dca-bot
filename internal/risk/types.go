package risk

import (
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
)

// Order represents a trading order
type Order struct {
	Symbol string
	Side   exchange.OrderSide
	Amount float64
	Price  float64
}

// Portfolio represents the current portfolio state
type Portfolio struct {
	Balance float64
	Symbol  string
}

// Signal represents a trading signal
type Signal struct {
	Type     string
	Strength float64
	Price    float64
}
