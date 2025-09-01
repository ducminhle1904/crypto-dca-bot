package indicators

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

type TechnicalIndicator interface {
	Calculate(data []types.OHLCV) (float64, error)
	ShouldBuy(current float64, data []types.OHLCV) (bool, error)
	ShouldSell(current float64, data []types.OHLCV) (bool, error)
	GetSignalStrength() float64
	GetName() string
	GetRequiredPeriods() int
	// ResetState resets internal state for new data periods (walk-forward validation)
	ResetState()
}

type Signal struct {
	Type      SignalType
	Strength  float64
	Price     float64
	Timestamp time.Time
	Source    string
}

type SignalType int

const (
	SignalBuy SignalType = iota
	SignalSell
	SignalHold
)
