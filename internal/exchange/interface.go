package exchange

import (
	"context"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

type Exchange interface {
	GetName() string
	Connect(ctx context.Context) error
	Disconnect() error

	// Market data
	GetTicker(symbol string) (*types.Ticker, error)
	GetKlines(symbol string, interval string, limit int) ([]types.OHLCV, error)

	// Trading
	PlaceMarketOrder(symbol string, side OrderSide, quantity float64) (*types.Order, error)
	GetBalance(asset string) (*types.Balance, error)
}

type OrderSide int

const (
	OrderBuy OrderSide = iota
	OrderSell
)
