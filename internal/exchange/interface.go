package exchange

import (
	"context"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// InstrumentConstraints represents trading constraints for a symbol
type InstrumentConstraints struct {
	MinOrderQty  float64 `json:"min_order_qty"`
	MaxOrderQty  float64 `json:"max_order_qty"`
	QtyStep      float64 `json:"qty_step"`
	TickSize     float64 `json:"tick_size"`
	MinPrice     float64 `json:"min_price"`
	MaxPrice     float64 `json:"max_price"`
	MinNotional  float64 `json:"min_notional"`
	MaxLeverage  float64 `json:"max_leverage"`
	MinLeverage  float64 `json:"min_leverage"`
}

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
	
	// Instrument Info (for grid trading)
	GetInstrumentConstraints(ctx context.Context, category, symbol string) (*InstrumentConstraints, error)
	ValidateAndAdjustQuantity(ctx context.Context, category, symbol string, quantity float64) (float64, error)
}