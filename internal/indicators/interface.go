package indicators

import (
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
	"time"
)

type TechnicalIndicator interface {
	Calculate(data []types.OHLCV) (float64, error)
	ShouldBuy(current float64, data []types.OHLCV) (bool, error)
	ShouldSell(current float64, data []types.OHLCV) (bool, error)
	GetSignalStrength() float64
	GetName() string
	GetRequiredPeriods() int
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
