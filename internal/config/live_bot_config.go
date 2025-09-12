package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
)

// LiveBotConfig represents the complete configuration for the live trading bot
type LiveBotConfig struct {
	// Trading strategy configuration
	Strategy StrategyConfig `json:"strategy"`
	
	// Exchange configuration
	Exchange exchange.ExchangeConfig `json:"exchange"`
	
	// Risk management configuration
	Risk RiskConfig `json:"risk"`
	
	// Notification configuration (optional)
	Notifications *NotificationConfig `json:"notifications,omitempty"`
}

// StrategyConfig holds trading strategy configuration
type StrategyConfig struct {
	// Core DCA parameters
	Symbol                   string  `json:"symbol"`                     // Trading symbol (e.g., BTCUSDT)
	Category                 string  `json:"category"`                   // Trading category (spot, linear, inverse)
	BaseAmount               float64 `json:"base_amount"`               // Base DCA amount in USD
	MaxMultiplier            float64 `json:"max_multiplier"`            // Maximum multiplier for DCA
	PriceThreshold           float64 `json:"price_threshold"`           // Price drop threshold to trigger DCA
	PriceThresholdMultiplier float64 `json:"price_threshold_multiplier"` // Multiplier for progressive DCA spacing (e.g., 1.1x per level)
	
	// Market data settings
	Interval   string `json:"interval"`    // Trading interval (5m, 15m, 1h, etc.)
	WindowSize int    `json:"window_size"` // Data window size for indicators
	
	// Take profit settings
	TPPercent float64 `json:"tp_percent"` // Base take profit percentage for multi-level system
	Cycle     bool    `json:"cycle"`      // Whether to cycle after take profit
	AutoTPOrders bool  `json:"auto_tp_orders"` // Whether to place TP orders automatically after buys
	UseTPLevels  bool  `json:"use_tp_levels"`  // Use 5-level TP system (always true by default)
	TPLevels     int   `json:"tp_levels"`      // Number of TP levels (default 5)
	TPQuantity   float64 `json:"tp_quantity"`  // Quantity per TP level (default 0.20 = 20%)
	
	// Order management settings
	CancelOrphanedOrders bool `json:"cancel_orphaned_orders"` // Cancel existing orders on startup (default false)
	
	// Technical indicators
	Indicators []string `json:"indicators"` // List of indicators to use
	
	// Technical indicator parameters
	RSI IndicatorRSIConfig `json:"rsi"`
	MACD IndicatorMACDConfig `json:"macd"`  
	BollingerBands IndicatorBBConfig `json:"bollinger_bands"`
	EMA IndicatorEMAConfig `json:"ema"`
	
	// Additional technical indicators
	HullMA      IndicatorHullMAConfig      `json:"hull_ma"`
	MFI         IndicatorMFIConfig         `json:"mfi"`
	Keltner     IndicatorKeltnerConfig     `json:"keltner_channels"`
	WaveTrend   IndicatorWaveTrendConfig   `json:"wavetrend"`
	
	// Additional trend indicators
	SuperTrend  IndicatorSuperTrendConfig  `json:"supertrend"`
	
	// Volume indicators
	OBV         IndicatorOBVConfig         `json:"obv"`
}


// IndicatorRSIConfig holds RSI indicator configuration
type IndicatorRSIConfig struct {
	Period     int     `json:"period"`      // RSI calculation period
	Oversold   float64 `json:"oversold"`    // Oversold threshold
	Overbought float64 `json:"overbought"`  // Overbought threshold
}

// IndicatorMACDConfig holds MACD indicator configuration
type IndicatorMACDConfig struct {
	FastPeriod   int `json:"fast_period"`   // Fast EMA period
	SlowPeriod   int `json:"slow_period"`   // Slow EMA period
	SignalPeriod int `json:"signal_period"` // Signal line period
}

// IndicatorBBConfig holds Bollinger Bands configuration
type IndicatorBBConfig struct {
	Period int     `json:"period"`  // BB calculation period
	StdDev float64 `json:"std_dev"` // Standard deviation multiplier
}

// IndicatorEMAConfig holds EMA configuration
type IndicatorEMAConfig struct {
	Period int `json:"period"` // EMA calculation period
}

// IndicatorHullMAConfig holds Hull Moving Average configuration
type IndicatorHullMAConfig struct {
	Period int `json:"period"` // Hull MA calculation period
}

// IndicatorMFIConfig holds Money Flow Index configuration
type IndicatorMFIConfig struct {
	Period     int     `json:"period"`      // MFI calculation period
	Oversold   float64 `json:"oversold"`    // Oversold threshold
	Overbought float64 `json:"overbought"`  // Overbought threshold
}

// IndicatorKeltnerConfig holds Keltner Channels configuration
type IndicatorKeltnerConfig struct {
	Period     int     `json:"period"`     // Keltner period
	Multiplier float64 `json:"multiplier"` // ATR multiplier
}

// IndicatorWaveTrendConfig holds WaveTrend configuration
type IndicatorWaveTrendConfig struct {
	N1         int     `json:"n1"`          // First EMA length for channel calculation
	N2         int     `json:"n2"`          // Second EMA length for average calculation
	Oversold   float64 `json:"oversold"`    // Oversold threshold
	Overbought float64 `json:"overbought"`  // Overbought threshold
}

// IndicatorSuperTrendConfig holds SuperTrend configuration
type IndicatorSuperTrendConfig struct {
	Period     int     `json:"period"`     // ATR period for SuperTrend calculation
	Multiplier float64 `json:"multiplier"` // ATR multiplier for band calculation
}

// IndicatorOBVConfig holds OBV (On-Balance Volume) configuration
type IndicatorOBVConfig struct {
	TrendThreshold float64 `json:"trend_threshold"` // Threshold for trend change detection (default 0.01 = 1%)
}

// RiskConfig holds risk management configuration
type RiskConfig struct {
	InitialBalance float64 `json:"initial_balance"` // Initial balance for tracking
	Commission     float64 `json:"commission"`      // Commission rate (0.001 = 0.1%)
}

// NotificationConfig holds notification settings
type NotificationConfig struct {
	Enabled       bool   `json:"enabled"`
	TelegramToken string `json:"telegram_token,omitempty"`
	TelegramChat  string `json:"telegram_chat,omitempty"`
}

// LoadLiveBotConfig loads configuration from file
func LoadLiveBotConfig(configFile string) (*LiveBotConfig, error) {
	// If config file doesn't contain path separators, look in configs/ directory
	if !strings.ContainsAny(configFile, "/\\") {
		configFile = filepath.Join("configs", configFile)
	}

	// Add .json extension if not present
	if !strings.HasSuffix(configFile, ".json") {
		configFile += ".json"
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var config LiveBotConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults and validation
	if err := config.setDefaults(); err != nil {
		return nil, fmt.Errorf("failed to set config defaults: %w", err)
	}

	if err := config.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for missing configuration
func (c *LiveBotConfig) setDefaults() error {
	// Strategy defaults
	if c.Strategy.WindowSize == 0 {
		c.Strategy.WindowSize = 100
	}
	if c.Strategy.MaxMultiplier == 0 {
		c.Strategy.MaxMultiplier = 5.0
	}
	if c.Strategy.PriceThreshold == 0 {
		c.Strategy.PriceThreshold = 0.05 // 5% drop
	}
	if c.Strategy.TPPercent == 0 {
		c.Strategy.TPPercent = 0.02 // 2% profit (aligned with backtest default)
	}
	
	// Multi-level TP defaults (always enabled)
	c.Strategy.UseTPLevels = true
	c.Strategy.AutoTPOrders = true
	if c.Strategy.TPLevels == 0 {
		c.Strategy.TPLevels = 5 // 5 levels by default
	}
	if c.Strategy.TPQuantity == 0 {
		c.Strategy.TPQuantity = 0.20 // 20% per level (1.0 / 5 levels)
	}
	if c.Strategy.Interval == "" {
		c.Strategy.Interval = "5m"
	}
	if c.Strategy.Category == "" {
		// Default category based on exchange and symbol
		c.Strategy.Category = determineDefaultCategory(c.Exchange.Name, c.Strategy.Symbol)
	}



	// RSI defaults
	if c.Strategy.RSI.Period == 0 {
		c.Strategy.RSI.Period = 14
	}
	if c.Strategy.RSI.Oversold == 0 {
		c.Strategy.RSI.Oversold = 30
	}
	if c.Strategy.RSI.Overbought == 0 {
		c.Strategy.RSI.Overbought = 70
	}

	// MACD defaults
	if c.Strategy.MACD.FastPeriod == 0 {
		c.Strategy.MACD.FastPeriod = 12
	}
	if c.Strategy.MACD.SlowPeriod == 0 {
		c.Strategy.MACD.SlowPeriod = 26
	}
	if c.Strategy.MACD.SignalPeriod == 0 {
		c.Strategy.MACD.SignalPeriod = 9
	}

	// Bollinger Bands defaults
	if c.Strategy.BollingerBands.Period == 0 {
		c.Strategy.BollingerBands.Period = 20
	}
	if c.Strategy.BollingerBands.StdDev == 0 {
		c.Strategy.BollingerBands.StdDev = 2.0
	}

	// EMA defaults
	if c.Strategy.EMA.Period == 0 {
		c.Strategy.EMA.Period = 21
	}

	// Additional indicator defaults
	// Hull MA defaults
	if c.Strategy.HullMA.Period == 0 {
		c.Strategy.HullMA.Period = 20
	}

	// MFI defaults
	if c.Strategy.MFI.Period == 0 {
		c.Strategy.MFI.Period = 14
	}
	if c.Strategy.MFI.Oversold == 0 {
		c.Strategy.MFI.Oversold = 20
	}
	if c.Strategy.MFI.Overbought == 0 {
		c.Strategy.MFI.Overbought = 80
	}

	// Keltner defaults
	if c.Strategy.Keltner.Period == 0 {
		c.Strategy.Keltner.Period = 20
	}
	if c.Strategy.Keltner.Multiplier == 0 {
		c.Strategy.Keltner.Multiplier = 2.0
	}

	// WaveTrend defaults
	if c.Strategy.WaveTrend.N1 == 0 {
		c.Strategy.WaveTrend.N1 = 10
	}
	if c.Strategy.WaveTrend.N2 == 0 {
		c.Strategy.WaveTrend.N2 = 21
	}
	if c.Strategy.WaveTrend.Oversold == 0 {
		c.Strategy.WaveTrend.Oversold = -60
	}
	if c.Strategy.WaveTrend.Overbought == 0 {
		c.Strategy.WaveTrend.Overbought = 60
	}

	// SuperTrend defaults
	if c.Strategy.SuperTrend.Period == 0 {
		c.Strategy.SuperTrend.Period = 14
	}
	if c.Strategy.SuperTrend.Multiplier == 0 {
		c.Strategy.SuperTrend.Multiplier = 2.5
	}

	// OBV defaults
	if c.Strategy.OBV.TrendThreshold == 0 {
		c.Strategy.OBV.TrendThreshold = 0.01 // 1% threshold
	}

	// Risk defaults
	if c.Risk.InitialBalance == 0 {
		c.Risk.InitialBalance = 1000.0
	}
	if c.Risk.Commission == 0 {
		c.Risk.Commission = 0.001 // 0.1%
	}

	// Exchange defaults (if not specified)
	if c.Exchange.Name == "" {
		c.Exchange.Name = "bybit" // Default to Bybit
	}

	return nil
}

// validate validates the configuration
func (c *LiveBotConfig) validate() error {
	// Validate strategy config
	if c.Strategy.Symbol == "" {
		return fmt.Errorf("trading symbol is required")
	}
	if c.Strategy.BaseAmount <= 0 {
		return fmt.Errorf("base amount must be greater than 0")
	}
	if c.Strategy.MaxMultiplier < 1.0 {
		return fmt.Errorf("max multiplier must be at least 1.0")
	}
	if c.Strategy.PriceThreshold <= 0 || c.Strategy.PriceThreshold > 1.0 {
		return fmt.Errorf("price threshold must be between 0 and 1.0")
	}



	// Validate risk config
	if c.Risk.InitialBalance <= 0 {
		return fmt.Errorf("initial balance must be greater than 0")
	}

	// Validate exchange config using factory
	factory := exchange.NewExchangeFactory()
	if err := factory.ValidateConfig(c.Exchange); err != nil {
		return fmt.Errorf("exchange config validation failed: %w", err)
	}

	return nil
}

// determineDefaultCategory determines the default trading category based on exchange and symbol
func determineDefaultCategory(exchangeName, symbol string) string {
	switch strings.ToLower(exchangeName) {
	case "bybit":
		// For Bybit, prefer linear futures for crypto pairs
		if strings.Contains(symbol, "USDT") || strings.Contains(symbol, "USD") {
			return "linear"
		}
		return "spot"
	case "binance":
		// For Binance, default to spot trading
		return "spot"
	default:
		return "spot" // Safe default
	}
}

