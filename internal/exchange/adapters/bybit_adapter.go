package adapters

import (
	"context"
	"log"

	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/bybit"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// BybitAdapter implements the LiveTradingExchange interface for Bybit
type BybitAdapter struct {
	client    *bybit.Client
	config    *exchange.BybitConfig
	connected bool
}

// NewBybitAdapter creates a new Bybit adapter instance
func NewBybitAdapter(config *exchange.BybitConfig) (*BybitAdapter, error) {
	if config == nil {
		return nil, &exchange.ExchangeError{
			Code:    "MISSING_CONFIG",
			Message: "Bybit configuration is required",
			IsRetryable: false,
		}
	}

	// Create Bybit client config
	bybitConfig := bybit.Config{
		APIKey:    config.APIKey,
		APISecret: config.APISecret,
		Testnet:   config.Testnet,
		Demo:      config.Demo,
	}

	// Create Bybit client
	client := bybit.NewClient(bybitConfig)

	return &BybitAdapter{
		client:    client,
		config:    config,
		connected: false,
	}, nil
}

// GetName returns the exchange name
func (b *BybitAdapter) GetName() string {
	return "Bybit"
}

// IsDemo returns whether the adapter is in demo mode
func (b *BybitAdapter) IsDemo() bool {
	return b.client.IsDemo()
}

// GetEnvironment returns the current environment string
func (b *BybitAdapter) GetEnvironment() string {
	return b.client.GetEnvironment()
}

// Connect establishes connection to the exchange
func (b *BybitAdapter) Connect(ctx context.Context) error {
	// For Bybit, we can test connection by getting server time or instrument info
	// This is a basic connectivity test
	_, err := b.GetTradingConstraints(ctx, "linear", "BTCUSDT")
	if err != nil {
		return &exchange.ExchangeError{
			Code:    "CONNECTION_FAILED",
			Message: "Failed to connect to Bybit",
			Details: err.Error(),
			IsRetryable: true,
		}
	}
	
	b.connected = true
	return nil
}

// Disconnect closes connection to the exchange
func (b *BybitAdapter) Disconnect() error {
	b.connected = false
	return nil
}

// IsConnected returns whether the adapter is connected
func (b *BybitAdapter) IsConnected() bool {
	return b.connected
}

// GetLatestPrice retrieves the latest price for a symbol
func (b *BybitAdapter) GetLatestPrice(ctx context.Context, symbol string) (float64, error) {
	// We need to determine the category from the symbol
	// For now, assume linear futures for crypto pairs
	category := "linear"
	if isSpotSymbol(symbol) {
		category = "spot"
	}
	
	price, err := b.client.GetLatestPrice(ctx, category, symbol)
	if err != nil {
		return 0, b.convertError(err)
	}
	
	return price, nil
}

// GetKlines retrieves kline/candlestick data
func (b *BybitAdapter) GetKlines(ctx context.Context, params exchange.KlineParams) ([]types.OHLCV, error) {
	// Convert interval format from "5m" to "5", "1h" to "60", etc.
	bybitInterval := convertIntervalToBybit(params.Interval)
	
	// Convert our generic params to Bybit-specific params
	bybitParams := bybit.KlineParams{
		Category: params.Category,
		Symbol:   params.Symbol,
		Interval: bybitInterval,
		Limit:    params.Limit,
	}
	
	klines, err := b.client.GetKlines(ctx, bybitParams)
	if err != nil {
		return nil, b.convertError(err)
	}

	// Convert Bybit klines to our standard format
	result := make([]types.OHLCV, len(klines))
	for i, kline := range klines {
		result[i] = types.OHLCV{
			Timestamp: kline.StartTime,
			Open:      kline.OpenPrice,
			High:      kline.HighPrice,
			Low:       kline.LowPrice,
			Close:     kline.ClosePrice,
			Volume:    kline.Volume,
		}
	}

	return result, nil
}

// GetTradableBalance retrieves tradable balance for an asset
func (b *BybitAdapter) GetTradableBalance(ctx context.Context, accountType exchange.AccountType, asset string) (float64, error) {
	// Convert our generic account type to Bybit-specific type
	bybitAccountType := convertAccountType(accountType)
	
	balance, err := b.client.GetTradableBalance(ctx, bybitAccountType, asset)
	if err != nil {
		return 0, b.convertError(err)
	}
	
	return balance, nil
}

// GetPositions retrieves current positions
func (b *BybitAdapter) GetPositions(ctx context.Context, category, symbol string) ([]exchange.Position, error) {
	positions, err := b.client.GetPositions(ctx, category, symbol)
	if err != nil {
		return nil, b.convertError(err)
	}

	// Convert Bybit positions to our standard format
	result := make([]exchange.Position, len(positions))
	for i, pos := range positions {
		result[i] = exchange.Position{
			Symbol:          pos.Symbol,
			Side:            pos.Side,
			Size:            pos.Size,
			PositionValue:   pos.PositionValue,
			AvgPrice:        pos.AvgPrice,
			MarkPrice:       pos.MarkPrice,
			UnrealisedPnl:   pos.UnrealisedPnl,
			Leverage:        "1", // Default leverage - Bybit doesn't expose this in PositionInfo
			CreatedTime:     pos.CreatedTime,
			UpdatedTime:     pos.UpdatedTime,
		}
	}

	return result, nil
}

// PlaceMarketOrder places a market order
func (b *BybitAdapter) PlaceMarketOrder(ctx context.Context, params exchange.OrderParams) (*exchange.Order, error) {
	// Convert our generic params to Bybit-specific params
	bybitSide := convertOrderSide(params.Side)
	
	order, err := b.client.PlaceMarketOrder(ctx, params.Category, params.Symbol, bybitSide, params.Quantity)
	if err != nil {
		return nil, b.convertError(err)
	}

	// Convert Bybit order to our standard format
	result := &exchange.Order{
		OrderID:       order.OrderID,
		Symbol:        order.Symbol,
		Side:          params.Side,
		OrderType:     exchange.OrderTypeMarket,
		Quantity:      order.CumExecQty,
		Price:         order.AvgPrice,
		CumExecQty:    order.CumExecQty,
		CumExecValue:  order.CumExecValue,
		AvgPrice:      order.AvgPrice,
		OrderStatus:   string(order.OrderStatus), // Convert OrderStatus to string
		CreatedTime:   order.CreatedTime,
		UpdatedTime:   order.UpdatedTime,
	}

	return result, nil
}

// GetOrderStatus retrieves the status of an order
func (b *BybitAdapter) GetOrderStatus(ctx context.Context, orderID string) (*exchange.OrderStatus, error) {
	// This would require implementing GetOrder in the Bybit client
	// For now, return a not implemented error
	return nil, &exchange.ExchangeError{
		Code:    "NOT_IMPLEMENTED",
		Message: "GetOrderStatus not implemented in Bybit client yet",
		IsRetryable: false,
	}
}

// GetTradingConstraints retrieves trading constraints for a symbol
func (b *BybitAdapter) GetTradingConstraints(ctx context.Context, category, symbol string) (*exchange.TradingConstraints, error) {
	minQty, maxQty, qtyStep, err := b.client.GetInstrumentManager().GetQuantityConstraints(ctx, category, symbol)
	if err != nil {
		return nil, b.convertError(err)
	}

	// Get current price to calculate min notional value
	currentPrice, err := b.GetLatestPrice(ctx, symbol)
	if err != nil {
		// If we can't get price, set min notional to 0
		currentPrice = 0
	}

	minNotional := minQty * currentPrice

	return &exchange.TradingConstraints{
		Symbol:           symbol,
		MinOrderQty:      minQty,
		MaxOrderQty:      maxQty,
		QtyStep:          qtyStep,
		MinOrderValue:    minNotional,
		MaxOrderValue:    0, // Bybit doesn't provide this directly
		MinPriceStep:     0, // Would need to get from instrument info
		MaxLeverage:      100, // Bybit typically supports up to 100x
		MarginCurrency:   "USDT", // Default for most pairs
	}, nil
}

// Helper functions

// convertAccountType converts our generic account type to Bybit-specific type
func convertAccountType(accountType exchange.AccountType) bybit.AccountType {
	switch accountType {
	case exchange.AccountTypeUnified:
		return bybit.AccountTypeUnified
	case exchange.AccountTypeSpot:
		return bybit.AccountTypeUnified // Bybit uses unified for spot
	case exchange.AccountTypeContract:
		return bybit.AccountTypeUnified // Bybit uses unified for contracts
	default:
		return bybit.AccountTypeUnified
	}
}

// convertOrderSide converts our generic order side to Bybit-specific side
func convertOrderSide(side exchange.OrderSide) bybit.OrderSide {
	switch side {
	case exchange.OrderSideBuy:
		return bybit.OrderSideBuy
	case exchange.OrderSideSell:
		return bybit.OrderSideSell
	default:
		return bybit.OrderSideBuy
	}
}

// convertIntervalToBybit converts generic interval format to Bybit's expected format
func convertIntervalToBybit(interval exchange.KlineInterval) bybit.KlineInterval {
	switch interval {
	case exchange.Interval1m:   // "1"
		return bybit.Interval1m   // "1"
	case exchange.Interval3m:   // "3"
		return bybit.Interval3m   // "3"  
	case exchange.Interval5m:   // "5"
		return bybit.Interval5m   // "5"
	case exchange.Interval15m:  // "15"
		return bybit.Interval15m  // "15"
	case exchange.Interval30m:  // "30"
		return bybit.Interval30m  // "30"
	case exchange.Interval1h:   // "60"
		return bybit.Interval1h   // "60"
	case exchange.Interval4h:   // "240"
		return bybit.Interval4h   // "240"
	case exchange.Interval1d:   // "D"
		return bybit.Interval1d   // "D"
	default:
		// Try to handle string formats like "5m", "1h" etc.
		switch string(interval) {
		case "1m":
			return bybit.Interval1m
		case "3m":
			return bybit.Interval3m
		case "5m":
			return bybit.Interval5m
		case "15m":
			return bybit.Interval15m
		case "30m":
			return bybit.Interval30m
		case "1h":
			return bybit.Interval1h
		case "4h":
			return bybit.Interval4h
		case "1d":
			return bybit.Interval1d
		default:
			log.Printf("⚠️ Unknown interval format '%s', defaulting to 5m", string(interval))
			return bybit.Interval5m // Default fallback
		}
	}
}

// isSpotSymbol determines if a symbol is for spot trading
// This is a simple heuristic - in practice you might want more sophisticated logic
func isSpotSymbol(symbol string) bool {
	// For now, assume all symbols are derivatives unless explicitly spot
	// This could be enhanced by checking against known spot symbols
	return false
}

// convertError converts Bybit-specific errors to our standard error format
func (b *BybitAdapter) convertError(err error) error {
	if err == nil {
		return nil
	}

	// Check if it's already our error type
	if exchangeErr, ok := err.(*exchange.ExchangeError); ok {
		return exchangeErr
	}

	// Check for specific Bybit error types
	if bybit.IsAuthenticationError(err) {
		return &exchange.ExchangeError{
			Code:    "AUTHENTICATION_FAILED",
			Message: "Bybit API authentication failed",
			Details: err.Error(),
			IsRetryable: false,
		}
	}

	// Check for rate limiting (this would need to be implemented in bybit package)
	// For now, do a simple string check
	errStr := err.Error()
	if contains(errStr, "rate limit") || contains(errStr, "too many requests") {
		return &exchange.ExchangeError{
			Code:    "RATE_LIMIT_EXCEEDED",
			Message: "Bybit API rate limit exceeded",
			Details: err.Error(),
			IsRetryable: true,
		}
	}

	if contains(errStr, "insufficient") {
		return &exchange.ExchangeError{
			Code:    "INSUFFICIENT_BALANCE",
			Message: "Insufficient balance for trade",
			Details: err.Error(),
			IsRetryable: false,
		}
	}

	// Default to generic error
	return &exchange.ExchangeError{
		Code:    "UNKNOWN_ERROR",
		Message: "Unknown error from Bybit",
		Details: err.Error(),
		IsRetryable: false,
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(str, substr string) bool {
	return len(str) >= len(substr) && 
		   (str == substr || 
		    (len(str) > len(substr) && 
		     (str[:len(substr)] == substr || 
		      str[len(str)-len(substr):] == substr ||
		      findSubstring(str, substr))))
}

// findSubstring performs a simple substring search
func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
