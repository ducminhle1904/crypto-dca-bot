package spacing

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DCASpacingStrategy defines the interface for DCA entry spacing strategies
type DCASpacingStrategy interface {
	// CalculateThreshold calculates the price drop threshold required for the next DCA entry
	// level: current DCA level (0 = first entry, 1+ = subsequent entries)  
	// context: market data and context for calculation
	CalculateThreshold(level int, context *MarketContext) float64

	// GetName returns the human-readable name of the strategy
	GetName() string

	// GetParameters returns the current strategy parameters
	GetParameters() map[string]interface{}

	// ValidateConfig validates the strategy configuration
	ValidateConfig() error

	// Reset resets the strategy state (called at cycle completion)
	Reset()
}

// MarketContext contains market data needed by spacing strategies
type MarketContext struct {
	// Price information
	CurrentPrice   float64   // Current market price
	LastEntryPrice float64   // Price of last DCA entry (0 if no previous entry)
	PriceHistory   []float64 // Recent price history for calculations

	// Technical indicators
	ATR    float64 // Average True Range (volatility measure)
	RSI    float64 // Relative Strength Index
	MACD   float64 // MACD value
	Volume float64 // Current volume

	// OHLCV data for advanced calculations
	CurrentCandle types.OHLCV   // Current price candle
	RecentCandles []types.OHLCV // Recent candles for indicator calculations

	// Timing information
	Timestamp time.Time // Current timestamp
}

// SpacingConfig holds configuration for spacing strategies
type SpacingConfig struct {
	Strategy   string                 `json:"strategy"`   // Strategy name (e.g., "volatility_adaptive")
	Parameters map[string]interface{} `json:"parameters"` // Strategy-specific parameters
}
