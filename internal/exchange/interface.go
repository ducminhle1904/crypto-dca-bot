package exchange

import (
	"context"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

type Exchange interface {
	GetName() string
	Connect(ctx context.Context) error
	Disconnect() error

	// Market data
	GetTicker(symbol string) (*types.Ticker, error)
	GetKlines(symbol string, interval string, limit int) ([]types.OHLCV, error)
	SubscribeToTicker(symbol string, callback func(*types.Ticker)) error

	// Trading
	PlaceMarketOrder(symbol string, side OrderSide, quantity float64) (*types.Order, error)
	GetBalance(asset string) (*types.Balance, error)

	// WebSocket
	StartWebSocket(ctx context.Context) error
	SubscribeToKlines(symbol string, interval string) error
}

type OrderSide int

const (
	OrderBuy OrderSide = iota
	OrderSell
)
