package strategy

import (
	"time"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

// Strategy defines the interface for trading strategies
type Strategy interface {
	// ShouldExecuteTrade analyzes market data and returns a trading decision
	ShouldExecuteTrade(data []types.OHLCV) (*TradeDecision, error)

	// GetName returns the name of the strategy
	GetName() string
}

// TradeDecision represents a trading decision made by a strategy
type TradeDecision struct {
	Action     TradeAction
	Amount     float64
	Confidence float64
	Strength   float64
	Reason     string
	Timestamp  time.Time
}

// TradeAction represents the type of trading action
type TradeAction int

const (
	ActionHold TradeAction = iota
	ActionBuy
	ActionSell
)

func (ta TradeAction) String() string {
	switch ta {
	case ActionHold:
		return "HOLD"
	case ActionBuy:
		return "BUY"
	case ActionSell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}
