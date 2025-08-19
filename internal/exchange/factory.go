package exchange

import (
	"fmt"
	"strings"
)

// ExchangeConfig holds configuration for creating exchange instances
type ExchangeConfig struct {
	Name    string         `json:"name"`               // Exchange name (bybit, binance, etc.)
	Bybit   *BybitConfig   `json:"bybit,omitempty"`    // Bybit-specific config
	Binance *BinanceConfig `json:"binance,omitempty"`  // Binance-specific config
}

// BybitConfig holds Bybit-specific configuration
type BybitConfig struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	Testnet   bool   `json:"testnet"`   // Use testnet infrastructure
	Demo      bool   `json:"demo"`      // Use demo trading (paper trading)
}

// BinanceConfig holds Binance-specific configuration  
type BinanceConfig struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	Testnet   bool   `json:"testnet"`   // Use testnet infrastructure
	Demo      bool   `json:"demo"`      // Use demo trading (paper trading)
}

// ExchangeFactory creates exchange instances based on configuration
type ExchangeFactory struct{}

// NewExchangeFactory creates a new exchange factory instance
func NewExchangeFactory() *ExchangeFactory {
	return &ExchangeFactory{}
}

// CreateExchange creates an exchange instance based on the provided configuration
func (f *ExchangeFactory) CreateExchange(config ExchangeConfig) (LiveTradingExchange, error) {
	exchangeName := strings.ToLower(strings.TrimSpace(config.Name))
	
	switch exchangeName {
	case "bybit":
		return f.createBybitExchange(config.Bybit)
	case "binance":
		return f.createBinanceExchange(config.Binance)
	default:
		return nil, &ExchangeError{
			Code:    "UNSUPPORTED_EXCHANGE",
			Message: fmt.Sprintf("Exchange '%s' is not supported", config.Name),
			Details: "Supported exchanges: bybit, binance",
			IsRetryable: false,
		}
	}
}

// GetSupportedExchanges returns a list of supported exchange names
func (f *ExchangeFactory) GetSupportedExchanges() []string {
	return []string{"bybit", "binance"}
}

// ValidateConfig validates the exchange configuration
func (f *ExchangeFactory) ValidateConfig(config ExchangeConfig) error {
	if config.Name == "" {
		return &ExchangeError{
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
		return &ExchangeError{
			Code:    "UNSUPPORTED_EXCHANGE",
			Message: fmt.Sprintf("Exchange '%s' is not supported", config.Name),
			Details: fmt.Sprintf("Supported exchanges: %v", f.GetSupportedExchanges()),
			IsRetryable: false,
		}
	}
}

// createBybitExchange creates a Bybit exchange instance
func (f *ExchangeFactory) createBybitExchange(config *BybitConfig) (LiveTradingExchange, error) {
	if config == nil {
		return nil, &ExchangeError{
			Code:    "MISSING_BYBIT_CONFIG",
			Message: "Bybit configuration is required",
			IsRetryable: false,
		}
	}
	
	if err := f.validateBybitConfig(config); err != nil {
		return nil, err
	}
	
	// For now, return an error indicating we need to use the factory from adapters package
	return nil, &ExchangeError{
		Code:    "USE_ADAPTERS_FACTORY",
		Message: "Use factory from adapters package to avoid circular imports",
		Details: "Import github.com/ducminhle1904/crypto-dca-bot/internal/exchange/adapters and use adapters.NewExchangeFactory()",
		IsRetryable: false,
	}
}

// createBinanceExchange creates a Binance exchange instance
func (f *ExchangeFactory) createBinanceExchange(config *BinanceConfig) (LiveTradingExchange, error) {
	if config == nil {
		return nil, &ExchangeError{
			Code:    "MISSING_BINANCE_CONFIG",
			Message: "Binance configuration is required",
			IsRetryable: false,
		}
	}
	
	if err := f.validateBinanceConfig(config); err != nil {
		return nil, err
	}
	
	// Import will be added when we create the adapter
	// For now, return an error indicating it's not implemented yet
	return nil, &ExchangeError{
		Code:    "NOT_IMPLEMENTED",
		Message: "Binance adapter not implemented yet",
		Details: "The Binance exchange adapter will be implemented in a later step",
		IsRetryable: false,
	}
}

// validateBybitConfig validates Bybit-specific configuration
func (f *ExchangeFactory) validateBybitConfig(config *BybitConfig) error {
	if config == nil {
		return &ExchangeError{
			Code:    "MISSING_BYBIT_CONFIG",
			Message: "Bybit configuration is required",
			IsRetryable: false,
		}
	}
	
	if config.APIKey == "" {
		return &ExchangeError{
			Code:    "MISSING_API_KEY",
			Message: "Bybit API key is required",
			Details: "Set BYBIT_API_KEY environment variable or provide in config",
			IsRetryable: false,
		}
	}
	
	if config.APISecret == "" {
		return &ExchangeError{
			Code:    "MISSING_API_SECRET",
			Message: "Bybit API secret is required",
			Details: "Set BYBIT_API_SECRET environment variable or provide in config",
			IsRetryable: false,
		}
	}
	
	// Validate environment combination
	if config.Testnet && config.Demo {
		return &ExchangeError{
			Code:    "INVALID_ENVIRONMENT_CONFIG",
			Message: "Cannot use both testnet and demo mode simultaneously",
			Details: "Choose either testnet OR demo mode, not both",
			IsRetryable: false,
		}
	}
	
	return nil
}

// validateBinanceConfig validates Binance-specific configuration
func (f *ExchangeFactory) validateBinanceConfig(config *BinanceConfig) error {
	if config == nil {
		return &ExchangeError{
			Code:    "MISSING_BINANCE_CONFIG",
			Message: "Binance configuration is required",
			IsRetryable: false,
		}
	}
	
	if config.APIKey == "" {
		return &ExchangeError{
			Code:    "MISSING_API_KEY",
			Message: "Binance API key is required",
			Details: "Set BINANCE_API_KEY environment variable or provide in config",
			IsRetryable: false,
		}
	}
	
	if config.APISecret == "" {
		return &ExchangeError{
			Code:    "MISSING_API_SECRET",
			Message: "Binance API secret is required",
			Details: "Set BINANCE_API_SECRET environment variable or provide in config",
			IsRetryable: false,
		}
	}
	
	// Validate environment combination
	if config.Testnet && config.Demo {
		return &ExchangeError{
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
func (f *ExchangeFactory) GetExchangeCapabilities(exchangeName string) (*ExchangeCapabilities, error) {
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
		return nil, &ExchangeError{
			Code:    "UNSUPPORTED_EXCHANGE",
			Message: fmt.Sprintf("Exchange '%s' is not supported", exchangeName),
			IsRetryable: false,
		}
	}
}
