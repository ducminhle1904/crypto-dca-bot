package adapters

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// BinanceAdapter implements the LiveTradingExchange interface for Binance
type BinanceAdapter struct {
	client    *exchange.BinanceExchange
	config    *exchange.BinanceConfig
	connected bool
}

// NewBinanceAdapter creates a new Binance adapter instance
func NewBinanceAdapter(config *exchange.BinanceConfig) (*BinanceAdapter, error) {
	if config == nil {
		return nil, &exchange.ExchangeError{
			Code:    "MISSING_CONFIG",
			Message: "Binance configuration is required",
			IsRetryable: false,
		}
	}

	// Create Binance client using existing implementation
	client := exchange.NewBinanceExchange(config.APIKey, config.APISecret, config.Testnet)

	return &BinanceAdapter{
		client:    client,
		config:    config,
		connected: false,
	}, nil
}

// GetName returns the exchange name
func (b *BinanceAdapter) GetName() string {
	return "Binance"
}

// IsDemo returns whether the adapter is in demo mode
func (b *BinanceAdapter) IsDemo() bool {
	// Binance doesn't have official demo mode, use testnet as demo equivalent
	return b.config.Testnet
}

// GetEnvironment returns the current environment string
func (b *BinanceAdapter) GetEnvironment() string {
	if b.config.Testnet {
		return "testnet"
	}
	return "mainnet"
}

// Connect establishes connection to the exchange
func (b *BinanceAdapter) Connect(ctx context.Context) error {
	err := b.client.Connect(ctx)
	if err != nil {
		return &exchange.ExchangeError{
			Code:    "CONNECTION_FAILED",
			Message: "Failed to connect to Binance",
			Details: err.Error(),
			IsRetryable: true,
		}
	}
	
	b.connected = true
	return nil
}

// Disconnect closes connection to the exchange
func (b *BinanceAdapter) Disconnect() error {
	b.connected = false
	return b.client.Disconnect()
}

// IsConnected returns whether the adapter is connected
func (b *BinanceAdapter) IsConnected() bool {
	return b.connected
}

// GetLatestPrice retrieves the latest price for a symbol
func (b *BinanceAdapter) GetLatestPrice(ctx context.Context, symbol string) (float64, error) {
	ticker, err := b.client.GetTicker(symbol)
	if err != nil {
		return 0, b.convertError(err)
	}
	
	return ticker.Price, nil
}

// GetKlines retrieves kline/candlestick data
func (b *BinanceAdapter) GetKlines(ctx context.Context, params exchange.KlineParams) ([]types.OHLCV, error) {
	// Convert interval to Binance format
	binanceInterval := convertIntervalToBinance(params.Interval)
	
	klines, err := b.client.GetKlines(params.Symbol, binanceInterval, params.Limit)
	if err != nil {
		return nil, b.convertError(err)
	}

	return klines, nil
}

// GetTradableBalance retrieves tradable balance for an asset
func (b *BinanceAdapter) GetTradableBalance(ctx context.Context, accountType exchange.AccountType, asset string) (float64, error) {
	balance, err := b.client.GetBalance(asset)
	if err != nil {
		return 0, b.convertError(err)
	}
	
	return balance.Free, nil
}

// GetPositions retrieves current positions (for futures)
func (b *BinanceAdapter) GetPositions(ctx context.Context, category, symbol string) ([]exchange.Position, error) {
	// For spot trading, Binance doesn't have positions in the same way
	// This would need to be implemented differently for Binance Futures
	// For now, return empty positions for spot trading
	if category == "spot" {
		return []exchange.Position{}, nil
	}
	
	// For futures, this would require Binance Futures API implementation
	return nil, &exchange.ExchangeError{
		Code:    "NOT_IMPLEMENTED",
		Message: "Binance Futures positions not implemented yet",
		Details: "Only spot trading is currently supported for Binance",
		IsRetryable: false,
	}
}

// PlaceMarketOrder places a market order
func (b *BinanceAdapter) PlaceMarketOrder(ctx context.Context, params exchange.OrderParams) (*exchange.Order, error) {
	// Convert our params to Binance format
	side := convertOrderSideToBinance(params.Side)
	quantity, err := strconv.ParseFloat(params.Quantity, 64)
	if err != nil {
		return nil, &exchange.ExchangeError{
			Code:    "INVALID_QUANTITY",
			Message: "Invalid quantity format",
			Details: err.Error(),
			IsRetryable: false,
		}
	}
	
	order, err := b.client.PlaceMarketOrder(params.Symbol, side, quantity)
	if err != nil {
		return nil, b.convertError(err)
	}

	// Convert Binance order to our standard format
	result := &exchange.Order{
		OrderID:       order.ID,
		Symbol:        order.Symbol,
		Side:          params.Side,
		OrderType:     exchange.OrderTypeMarket,
		Quantity:      fmt.Sprintf("%.8f", order.Quantity),
		Price:         fmt.Sprintf("%.8f", order.Price),
		CumExecQty:    fmt.Sprintf("%.8f", order.Quantity), // For market orders, assume fully executed
		CumExecValue:  fmt.Sprintf("%.8f", order.Quantity*order.Price),
		AvgPrice:      fmt.Sprintf("%.8f", order.Price),
		OrderStatus:   order.Status,
		CreatedTime:   order.Timestamp,
		UpdatedTime:   order.Timestamp,
	}

	return result, nil
}

// CancelOrder cancels an existing order
func (b *BinanceAdapter) CancelOrder(ctx context.Context, category, symbol, orderID string) error {
	// Note: This would need to be implemented in the Binance client
	// For now, return a not implemented error
	return &exchange.ExchangeError{
		Code:    "NOT_IMPLEMENTED",
		Message: "CancelOrder not implemented for Binance yet",
		Details: "Order cancellation needs to be added to the Binance client",
		IsRetryable: false,
	}
}

// PlaceLimitOrder places a limit order (used for take profit orders)
func (b *BinanceAdapter) PlaceLimitOrder(ctx context.Context, params exchange.OrderParams) (*exchange.Order, error) {
	// Note: This would need to be implemented in the Binance client
	// For now, return a not implemented error
	return nil, &exchange.ExchangeError{
		Code:    "NOT_IMPLEMENTED",
		Message: "PlaceLimitOrder not implemented for Binance yet",
		Details: "Limit orders need to be added to the Binance client",
		IsRetryable: false,
	}
}

// GetOrderStatus retrieves the status of an order
func (b *BinanceAdapter) GetOrderStatus(ctx context.Context, orderID string) (*exchange.OrderStatus, error) {
	// This would require implementing order status tracking in the Binance client
	return nil, &exchange.ExchangeError{
		Code:    "NOT_IMPLEMENTED",
		Message: "GetOrderStatus not implemented for Binance yet",
		IsRetryable: false,
	}
}

// GetTradingConstraints retrieves trading constraints for a symbol
func (b *BinanceAdapter) GetTradingConstraints(ctx context.Context, category, symbol string) (*exchange.TradingConstraints, error) {
	// This would require getting symbol info from Binance API
	// For now, return reasonable defaults
	return &exchange.TradingConstraints{
		Symbol:           symbol,
		MinOrderQty:      0.00001,    // Typical minimum for most crypto pairs
		MaxOrderQty:      9999999,    // Large number
		QtyStep:          0.00001,    // Typical step size
		MinOrderValue:    10.0,       // $10 minimum notional
		MaxOrderValue:    0,          // No limit
		MinPriceStep:     0.01,       // Typical price step
		MaxLeverage:      1,          // Spot trading, no leverage
		MarginCurrency:   "USDT",     // Default quote currency
	}, nil
}

// Helper functions

// convertIntervalToBinance converts our generic interval to Binance format
func convertIntervalToBinance(interval exchange.KlineInterval) string {
	switch interval {
	case exchange.Interval1m:
		return "1m"
	case exchange.Interval3m:
		return "3m"
	case exchange.Interval5m:
		return "5m"
	case exchange.Interval15m:
		return "15m"
	case exchange.Interval30m:
		return "30m"
	case exchange.Interval1h:
		return "1h"
	case exchange.Interval4h:
		return "4h"
	case exchange.Interval1d:
		return "1d"
	default:
		return "5m" // Default fallback
	}
}

// convertOrderSideToBinance converts our generic order side to Binance format
func convertOrderSideToBinance(side exchange.OrderSide) exchange.OrderSide {
	// For the current Binance implementation, we can use the same enum values
	return side
}

// convertError converts Binance-specific errors to our standard error format
func (b *BinanceAdapter) convertError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's already our error type
	if exchangeErr, ok := err.(*exchange.ExchangeError); ok {
		return exchangeErr
	}

	// Convert common Binance errors
	errStr := err.Error()
	
	if contains(errStr, "Invalid API-key") || contains(errStr, "Signature for this request") {
		return &exchange.ExchangeError{
			Code:    "AUTHENTICATION_FAILED",
			Message: "Binance API authentication failed",
			Details: err.Error(),
			IsRetryable: false,
		}
	}

	if contains(errStr, "rate limit") || contains(errStr, "too many requests") {
		return &exchange.ExchangeError{
			Code:    "RATE_LIMIT_EXCEEDED",
			Message: "Binance API rate limit exceeded",
			Details: err.Error(),
			IsRetryable: true,
		}
	}

	if contains(errStr, "insufficient") || contains(errStr, "balance") {
		return &exchange.ExchangeError{
			Code:    "INSUFFICIENT_BALANCE",
			Message: "Insufficient balance for trade",
			Details: err.Error(),
			IsRetryable: false,
		}
	}

	if contains(errStr, "symbol") && contains(errStr, "not") {
		return &exchange.ExchangeError{
			Code:    "INVALID_SYMBOL",
			Message: "Invalid trading symbol",
			Details: err.Error(),
			IsRetryable: false,
		}
	}

	// Default to generic error
	return &exchange.ExchangeError{
		Code:    "UNKNOWN_ERROR",
		Message: "Unknown error from Binance",
		Details: err.Error(),
		IsRetryable: false,
	}
}
