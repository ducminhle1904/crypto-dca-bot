package config

// Package config provides configuration management for crypto trading bots

// Config represents a generic configuration interface that all bot types must implement
type Config interface {
	// GetSymbol returns the trading symbol
	GetSymbol() string
	
	// GetInitialBalance returns the initial balance for backtesting
	GetInitialBalance() float64
	
	// GetCommission returns the commission rate
	GetCommission() float64
	
	// GetWindowSize returns the data window size for analysis
	GetWindowSize() int
	
	// GetMinOrderQty returns the minimum order quantity
	GetMinOrderQty() float64
	
	// GetInterval returns the trading interval
	GetInterval() string
	
	// GetDataFile returns the data file path
	GetDataFile() string
	
	// SetDataFile sets the data file path
	SetDataFile(string)
	
	// GetIndicators returns the list of indicators to use
	GetIndicators() []string
	
	// SetIndicators sets the list of indicators
	SetIndicators([]string)
	
	// Validate validates the configuration
	Validate() error
}

// ConfigManager handles loading, validation, and conversion of configurations
type ConfigManager interface {
	// LoadConfig loads configuration from file and command line parameters
	LoadConfig(configFile, dataFile, symbol string, balance, commission float64,
		windowSize int, params map[string]interface{}) (Config, error)
	
	// ValidateConfig validates a configuration
	ValidateConfig(cfg Config) error
	
	// ConvertToNested converts a flat config to nested format for output
	ConvertToNested(cfg Config) (NestedConfig, error)
	
	// SaveConfig saves configuration to file
	SaveConfig(cfg Config, path string) error
}

// Validator interface for configuration validation
type Validator interface {
	Validate(cfg Config) error
}

// Common configuration constants
const (
	// Default parameter values
	DefaultInitialBalance = 500.0
	DefaultCommission     = 0.0005 // 0.1%
	DefaultWindowSize     = 100
	DefaultMinOrderQty    = 0.01   // Default minimum order quantity (typical for BTCUSDT)
	
	// Data validation constants
	MinDataPoints         = 100    // Minimum data points required for backtest
	MaxCommission         = 1.0    // Maximum commission (100%)
	MinMultiplier         = 1.0    // Minimum multiplier value
	MaxThreshold          = 1.0    // Maximum threshold (100%)
	
	// Display and formatting constants
	ReportLineLength      = 50
	
	// File and directory constants
	DefaultDataRoot       = "data"
	DefaultExchange       = "bybit"          // Default exchange for data
	ResultsDir           = "results"
	BestConfigFile       = "best.json"
	TradesFile           = "trades.xlsx"
)
