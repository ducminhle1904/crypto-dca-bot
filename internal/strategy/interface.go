package strategy

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// Strategy defines the interface for trading strategies
type Strategy interface {
	// ShouldExecuteTrade analyzes market data and returns a trading decision
	ShouldExecuteTrade(data []types.OHLCV) (*TradeDecision, error)

	// GetName returns the name of the strategy
	GetName() string
	
	// OnCycleComplete is called when a take-profit cycle is completed
	// This allows strategies to reset state for the next cycle
	OnCycleComplete()
	
	// ResetForNewPeriod resets strategy state for walk-forward validation periods
	// This clears all indicator state to prevent contamination between validation folds
	ResetForNewPeriod()

	// GetDynamicTPPercent calculates dynamic TP percentage based on current market conditions
	// Returns the calculated TP percentage, or 0 if dynamic TP is not enabled
	GetDynamicTPPercent(currentCandle types.OHLCV, data []types.OHLCV) (float64, error)
	
	// IsDynamicTPEnabled returns true if dynamic TP is configured and enabled
	IsDynamicTPEnabled() bool
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
