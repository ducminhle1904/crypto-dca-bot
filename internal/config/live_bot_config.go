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
	Symbol         string  `json:"symbol"`          // Trading symbol (e.g., BTCUSDT)
	BaseAmount     float64 `json:"base_amount"`     // Base DCA amount in USD
	MaxMultiplier  float64 `json:"max_multiplier"`  // Maximum multiplier for DCA
	PriceThreshold float64 `json:"price_threshold"` // Price drop threshold to trigger DCA
	
	// Market data settings
	Interval   string `json:"interval"`    // Trading interval (5m, 15m, 1h, etc.)
	WindowSize int    `json:"window_size"` // Data window size for indicators
	
	// Take profit settings
	TPPercent float64 `json:"tp_percent"` // Take profit percentage
	Cycle     bool    `json:"cycle"`      // Whether to cycle after take profit
	
	// Technical indicators
	Indicators []string `json:"indicators"` // List of indicators to use
	
	// Indicator parameters
	RSI IndicatorRSIConfig `json:"rsi"`
	MACD IndicatorMACDConfig `json:"macd"`  
	BollingerBands IndicatorBBConfig `json:"bollinger_bands"`
	EMA IndicatorEMAConfig `json:"ema"`
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
		c.Strategy.TPPercent = 0.10 // 10% profit
	}
	if c.Strategy.Interval == "" {
		c.Strategy.Interval = "5m"
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

// MigrateFromLegacyConfig converts old config format to new format
func MigrateFromLegacyConfig(legacyConfigFile string, exchangeName string) (*LiveBotConfig, error) {
	// Legacy config structure (matching current main.go BotConfig)
	type LegacyBotConfig struct {
		DataFile        string    `json:"data_file"`
		Symbol          string    `json:"symbol"`
		InitialBalance  float64   `json:"initial_balance"`
		Commission      float64   `json:"commission"`
		WindowSize      int       `json:"window_size"`
		BaseAmount      float64   `json:"base_amount"`
		MaxMultiplier   float64   `json:"max_multiplier"`
		PriceThreshold  float64   `json:"price_threshold"`
		RSIPeriod       int       `json:"rsi_period"`
		RSIOversold     float64   `json:"rsi_oversold"`
		RSIOverbought   float64   `json:"rsi_overbought"`
		MACDFast        int       `json:"macd_fast"`
		MACDSlow        int       `json:"macd_slow"`
		MACDSignal      int       `json:"macd_signal"`
		BBPeriod        int       `json:"bb_period"`
		BBStdDev        float64   `json:"bb_std_dev"`
		EMAPeriod       int       `json:"ema_period"`
		Indicators      []string  `json:"indicators"`
		TPPercent       float64   `json:"tp_percent"`
		Cycle           bool      `json:"cycle"`
	}

	// Load legacy config
	data, err := os.ReadFile(legacyConfigFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read legacy config: %w", err)
	}

	var legacy LegacyBotConfig
	if err := json.Unmarshal(data, &legacy); err != nil {
		return nil, fmt.Errorf("failed to parse legacy config: %w", err)
	}

	// Convert to new format
	newConfig := &LiveBotConfig{
		Strategy: StrategyConfig{
			Symbol:         legacy.Symbol,
			BaseAmount:     legacy.BaseAmount,
			MaxMultiplier:  legacy.MaxMultiplier,
			PriceThreshold: legacy.PriceThreshold,
			Interval:       "5m", // Default, will be extracted from filename later
			WindowSize:     legacy.WindowSize,
			TPPercent:      legacy.TPPercent,
			Cycle:          legacy.Cycle,
			Indicators:     legacy.Indicators,
			RSI: IndicatorRSIConfig{
				Period:     legacy.RSIPeriod,
				Oversold:   legacy.RSIOversold,
				Overbought: legacy.RSIOverbought,
			},
			MACD: IndicatorMACDConfig{
				FastPeriod:   legacy.MACDFast,
				SlowPeriod:   legacy.MACDSlow,
				SignalPeriod: legacy.MACDSignal,
			},
			BollingerBands: IndicatorBBConfig{
				Period: legacy.BBPeriod,
				StdDev: legacy.BBStdDev,
			},
			EMA: IndicatorEMAConfig{
				Period: legacy.EMAPeriod,
			},
		},
		Risk: RiskConfig{
			InitialBalance: legacy.InitialBalance,
			Commission:     legacy.Commission,
		},
		Exchange: exchange.ExchangeConfig{
			Name: exchangeName,
		},
	}

	// Set exchange-specific config from environment
	switch strings.ToLower(exchangeName) {
	case "bybit":
		newConfig.Exchange.Bybit = &exchange.BybitConfig{
			APIKey:    os.Getenv("BYBIT_API_KEY"),
			APISecret: os.Getenv("BYBIT_API_SECRET"),
			Demo:      true, // Default to demo mode for safety
		}
	case "binance":
		newConfig.Exchange.Binance = &exchange.BinanceConfig{
			APIKey:    os.Getenv("BINANCE_API_KEY"),
			APISecret: os.Getenv("BINANCE_API_SECRET"),
			Testnet:   true, // Default to testnet for safety
		}
	}

	return newConfig, nil
}
