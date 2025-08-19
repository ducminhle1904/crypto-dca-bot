package adapters

import (
	"fmt"
	"strings"

	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
)

// Factory creates exchange instances based on configuration
type Factory struct{}

// NewFactory creates a new exchange factory instance
func NewFactory() *Factory {
	return &Factory{}
}

// CreateExchange creates an exchange instance based on the provided configuration
func (f *Factory) CreateExchange(config exchange.ExchangeConfig) (exchange.LiveTradingExchange, error) {
	exchangeName := strings.ToLower(strings.TrimSpace(config.Name))
	
	switch exchangeName {
	case "bybit":
		return f.createBybitExchange(config.Bybit)
	case "binance":
		return f.createBinanceExchange(config.Binance)
	default:
		return nil, &exchange.ExchangeError{
			Code:    "UNSUPPORTED_EXCHANGE",
			Message: fmt.Sprintf("Exchange '%s' is not supported", config.Name),
			Details: "Supported exchanges: bybit, binance",
			IsRetryable: false,
		}
	}
}

// GetSupportedExchanges returns a list of supported exchange names
func (f *Factory) GetSupportedExchanges() []string {
	return []string{"bybit", "binance"}
}

// ValidateConfig validates the exchange configuration
func (f *Factory) ValidateConfig(config exchange.ExchangeConfig) error {
	if config.Name == "" {
		return &exchange.ExchangeError{
			Code:    "MISSING_EXCHANGE_NAME",
			Message: "Exchange name is required",
			IsRetryable: false,
		}
	}
	
	exchangeName := strings.ToLower(strings.TrimSpace(config.Name))
	
	switch exchangeName {
	case "bybit":
		return f.validateBybitConfig(config.Bybit)
	case "binance":
		return f.validateBinanceConfig(config.Binance)
	default:
		return &exchange.ExchangeError{
			Code:    "UNSUPPORTED_EXCHANGE",
			Message: fmt.Sprintf("Exchange '%s' is not supported", config.Name),
			Details: fmt.Sprintf("Supported exchanges: %v", f.GetSupportedExchanges()),
			IsRetryable: false,
		}
	}
}

// createBybitExchange creates a Bybit exchange instance
func (f *Factory) createBybitExchange(config *exchange.BybitConfig) (exchange.LiveTradingExchange, error) {
	if config == nil {
		return nil, &exchange.ExchangeError{
			Code:    "MISSING_BYBIT_CONFIG",
			Message: "Bybit configuration is required",
			IsRetryable: false,
		}
	}
	
	if err := f.validateBybitConfig(config); err != nil {
		return nil, err
	}
	
	// Create Bybit adapter
	adapter, err := NewBybitAdapter(config)
	if err != nil {
		return nil, &exchange.ExchangeError{
			Code:    "ADAPTER_CREATION_FAILED",
			Message: "Failed to create Bybit adapter",
			Details: err.Error(),
			IsRetryable: false,
		}
	}
	
	return adapter, nil
}

// createBinanceExchange creates a Binance exchange instance
func (f *Factory) createBinanceExchange(config *exchange.BinanceConfig) (exchange.LiveTradingExchange, error) {
	if config == nil {
		return nil, &exchange.ExchangeError{
			Code:    "MISSING_BINANCE_CONFIG",
			Message: "Binance configuration is required",
			IsRetryable: false,
		}
	}
	
	if err := f.validateBinanceConfig(config); err != nil {
		return nil, err
	}
	
	// Create Binance adapter
	adapter, err := NewBinanceAdapter(config)
	if err != nil {
		return nil, &exchange.ExchangeError{
			Code:    "ADAPTER_CREATION_FAILED",
			Message: "Failed to create Binance adapter",
			Details: err.Error(),
			IsRetryable: false,
		}
	}
	
	return adapter, nil
}

// validateBybitConfig validates Bybit-specific configuration
func (f *Factory) validateBybitConfig(config *exchange.BybitConfig) error {
	if config == nil {
		return &exchange.ExchangeError{
			Code:    "MISSING_BYBIT_CONFIG",
			Message: "Bybit configuration is required",
			IsRetryable: false,
		}
	}
	
	if config.APIKey == "" {
		return &exchange.ExchangeError{
			Code:    "MISSING_API_KEY",
			Message: "Bybit API key is required",
			Details: "Set BYBIT_API_KEY environment variable or provide in config",
			IsRetryable: false,
		}
	}
	
	if config.APISecret == "" {
		return &exchange.ExchangeError{
			Code:    "MISSING_API_SECRET",
			Message: "Bybit API secret is required",
			Details: "Set BYBIT_API_SECRET environment variable or provide in config",
			IsRetryable: false,
		}
	}
	
	// Validate environment combination
	if config.Testnet && config.Demo {
		return &exchange.ExchangeError{
			Code:    "INVALID_ENVIRONMENT_CONFIG",
			Message: "Cannot use both testnet and demo mode simultaneously",
			Details: "Choose either testnet OR demo mode, not both",
			IsRetryable: false,
		}
	}
	
	return nil
}

// validateBinanceConfig validates Binance-specific configuration
func (f *Factory) validateBinanceConfig(config *exchange.BinanceConfig) error {
	if config == nil {
		return &exchange.ExchangeError{
			Code:    "MISSING_BINANCE_CONFIG",
			Message: "Binance configuration is required",
			IsRetryable: false,
		}
	}
	
	if config.APIKey == "" {
		return &exchange.ExchangeError{
			Code:    "MISSING_API_KEY",
			Message: "Binance API key is required",
			Details: "Set BINANCE_API_KEY environment variable or provide in config",
			IsRetryable: false,
		}
	}
	
	if config.APISecret == "" {
		return &exchange.ExchangeError{
			Code:    "MISSING_API_SECRET",
			Message: "Binance API secret is required",
			Details: "Set BINANCE_API_SECRET environment variable or provide in config",
			IsRetryable: false,
		}
	}
	
	// Validate environment combination
	if config.Testnet && config.Demo {
		return &exchange.ExchangeError{
			Code:    "INVALID_ENVIRONMENT_CONFIG",
			Message: "Cannot use both testnet and demo mode simultaneously",
			Details: "Choose either testnet OR demo mode, not both",
			IsRetryable: false,
		}
	}
	
	return nil
}

// ExchangeCapabilities represents what features each exchange supports
type ExchangeCapabilities struct {
	SpotTrading     bool `json:"spot_trading"`
	FuturesTrading  bool `json:"futures_trading"`
	OptionsTrading  bool `json:"options_trading"`
	DemoMode        bool `json:"demo_mode"`
	TestnetMode     bool `json:"testnet_mode"`
	Leverage        bool `json:"leverage"`
	MaxLeverage     int  `json:"max_leverage"`
}

// GetExchangeCapabilities returns the capabilities of a specific exchange
func (f *Factory) GetExchangeCapabilities(exchangeName string) (*ExchangeCapabilities, error) {
	switch strings.ToLower(strings.TrimSpace(exchangeName)) {
	case "bybit":
		return &ExchangeCapabilities{
			SpotTrading:     true,
			FuturesTrading:  true,
			OptionsTrading:  false, // Bybit options not implemented yet
			DemoMode:        true,
			TestnetMode:     true,
			Leverage:        true,
			MaxLeverage:     100,
		}, nil
	case "binance":
		return &ExchangeCapabilities{
			SpotTrading:     true,
			FuturesTrading:  true,
			OptionsTrading:  false, // Binance options not implemented yet
			DemoMode:        false, // Binance doesn't have official demo mode
			TestnetMode:     true,
			Leverage:        true,
			MaxLeverage:     125,
		}, nil
	default:
		return nil, &exchange.ExchangeError{
			Code:    "UNSUPPORTED_EXCHANGE",
			Message: fmt.Sprintf("Exchange '%s' is not supported", exchangeName),
			IsRetryable: false,
		}
	}
}
