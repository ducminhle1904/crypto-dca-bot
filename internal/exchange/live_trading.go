package exchange

import (
	"context"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// LiveTradingExchange defines the interface for live trading operations
// This abstracts exchange-specific implementations for the live bot
type LiveTradingExchange interface {
	// Exchange identification
	GetName() string
	IsDemo() bool
	GetEnvironment() string

	// Market data operations
	GetLatestPrice(ctx context.Context, symbol string) (float64, error)
	GetKlines(ctx context.Context, params KlineParams) ([]types.OHLCV, error)

	// Account management
	GetTradableBalance(ctx context.Context, accountType AccountType, asset string) (float64, error)
	GetPositions(ctx context.Context, category, symbol string) ([]Position, error)

	// Trading operations
	PlaceMarketOrder(ctx context.Context, params OrderParams) (*Order, error)
	GetOrderStatus(ctx context.Context, orderID string) (*OrderStatus, error)

	// Exchange constraints and limits
	GetTradingConstraints(ctx context.Context, category, symbol string) (*TradingConstraints, error)

	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
}

// KlineParams represents parameters for kline/candlestick data requests
type KlineParams struct {
	Category string        `json:"category"` // spot, linear, inverse
	Symbol   string        `json:"symbol"`
	Interval KlineInterval `json:"interval"`
	Limit    int           `json:"limit"`
	StartTime *time.Time    `json:"start_time,omitempty"`
	EndTime   *time.Time    `json:"end_time,omitempty"`
}

// KlineInterval represents different time intervals for kline data
type KlineInterval string

const (
	Interval1m   KlineInterval = "1"
	Interval3m   KlineInterval = "3"
	Interval5m   KlineInterval = "5"
	Interval15m  KlineInterval = "15"
	Interval30m  KlineInterval = "30"
	Interval1h   KlineInterval = "60"
	Interval4h   KlineInterval = "240"
	Interval1d   KlineInterval = "D"
)

// AccountType represents different account types across exchanges
type AccountType string

const (
	AccountTypeUnified AccountType = "UNIFIED"
	AccountTypeSpot    AccountType = "SPOT"
	AccountTypeContract AccountType = "CONTRACT"
)

// OrderParams represents parameters for placing orders
type OrderParams struct {
	Category  string    `json:"category"`  // spot, linear, inverse
	Symbol    string    `json:"symbol"`
	Side      OrderSide `json:"side"`
	Quantity  string    `json:"quantity"`
	OrderType OrderType `json:"order_type"`
	Price     string    `json:"price,omitempty"` // For limit orders
}

// OrderSide represents buy or sell side (string-based for API compatibility)
type OrderSide string

const (
	OrderSideBuy  OrderSide = "Buy"
	OrderSideSell OrderSide = "Sell"
)

// OrderType represents different order types
type OrderType string

const (
	OrderTypeMarket OrderType = "Market"
	OrderTypeLimit  OrderType = "Limit"
)

// Order represents order information returned by exchanges
type Order struct {
	OrderID       string    `json:"order_id"`
	Symbol        string    `json:"symbol"`
	Side          OrderSide `json:"side"`
	OrderType     OrderType `json:"order_type"`
	Quantity      string    `json:"quantity"`
	Price         string    `json:"price"`
	CumExecQty    string    `json:"cum_exec_qty"`    // Cumulative executed quantity
	CumExecValue  string    `json:"cum_exec_value"`  // Cumulative executed value
	AvgPrice      string    `json:"avg_price"`       // Average execution price
	OrderStatus   string    `json:"order_status"`
	CreatedTime   time.Time `json:"created_time"`
	UpdatedTime   time.Time `json:"updated_time"`
}

// OrderStatus represents the status of an order
type OrderStatus struct {
	OrderID     string    `json:"order_id"`
	Status      string    `json:"status"`
	ExecutedQty string    `json:"executed_qty"`
	Price       string    `json:"price"`
	UpdatedTime time.Time `json:"updated_time"`
}

// Position represents a trading position
type Position struct {
	Symbol          string `json:"symbol"`
	Side            string `json:"side"`             // Buy, Sell, None
	Size            string `json:"size"`             // Position size
	PositionValue   string `json:"position_value"`   // Notional value
	AvgPrice        string `json:"avg_price"`        // Average entry price
	MarkPrice       string `json:"mark_price"`       // Current mark price
	UnrealisedPnl   string `json:"unrealised_pnl"`   // Unrealized P&L
	Leverage        string `json:"leverage"`         // Position leverage
	PositionIM      string `json:"position_im"`      // Initial Margin
	PositionMM      string `json:"position_mm"`      // Maintenance Margin
	CreatedTime     time.Time `json:"created_time"`
	UpdatedTime     time.Time `json:"updated_time"`
}

// TradingConstraints represents exchange-specific trading limits and constraints
type TradingConstraints struct {
	Symbol           string  `json:"symbol"`
	MinOrderQty      float64 `json:"min_order_qty"`      // Minimum order quantity
	MaxOrderQty      float64 `json:"max_order_qty"`      // Maximum order quantity
	QtyStep          float64 `json:"qty_step"`           // Quantity step size
	MinOrderValue    float64 `json:"min_order_value"`    // Minimum notional value
	MaxOrderValue    float64 `json:"max_order_value"`    // Maximum notional value
	MinPriceStep     float64 `json:"min_price_step"`     // Minimum price increment
	MaxLeverage      float64 `json:"max_leverage"`       // Maximum leverage allowed
	MarginCurrency   string  `json:"margin_currency"`    // Currency used for margin
}

// ExchangeError represents standardized errors from exchanges
type ExchangeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
	IsRetryable bool `json:"is_retryable"`
}

func (e *ExchangeError) Error() string {
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// Common error types
var (
	ErrInsufficientBalance = &ExchangeError{
		Code:    "INSUFFICIENT_BALANCE",
		Message: "Insufficient balance for trade",
		IsRetryable: false,
	}
	
	ErrInvalidSymbol = &ExchangeError{
		Code:    "INVALID_SYMBOL",
		Message: "Invalid trading symbol",
		IsRetryable: false,
	}
	
	ErrOrderSizeTooSmall = &ExchangeError{
		Code:    "ORDER_SIZE_TOO_SMALL",
		Message: "Order size below minimum requirements",
		IsRetryable: false,
	}
	
	ErrRateLimitExceeded = &ExchangeError{
		Code:    "RATE_LIMIT_EXCEEDED",
		Message: "API rate limit exceeded",
		IsRetryable: true,
	}
	
	ErrConnectionFailed = &ExchangeError{
		Code:    "CONNECTION_FAILED",
		Message: "Failed to connect to exchange",
		IsRetryable: true,
	}
	
	ErrAuthenticationFailed = &ExchangeError{
		Code:    "AUTHENTICATION_FAILED",
		Message: "API authentication failed",
		IsRetryable: false,
	}
)
