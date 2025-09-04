package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
)

// ConfigLoader handles loading and validation of grid configurations
type ConfigLoader struct{}

// NewConfigLoader creates a new config loader
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

// LoadGridConfig loads and validates a grid configuration from file
func (cl *ConfigLoader) LoadGridConfig(configFile string) (*config.GridConfig, error) {
	// Check if file exists
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		return nil, &ConfigError{
			Type:    ErrTypeFileNotFound,
			Message: fmt.Sprintf("configuration file not found: %s", configFile),
			Cause:   err,
		}
	}

	// Read file content
	configData, err := os.ReadFile(configFile)
	if err != nil {
		return nil, &ConfigError{
			Type:    ErrTypeFileRead,
			Message: fmt.Sprintf("failed to read config file: %s", configFile),
			Cause:   err,
		}
	}

	// Parse JSON into the expected nested structure
	var configFileData GridConfigFile
	if err := json.Unmarshal(configData, &configFileData); err != nil {
		return nil, &ConfigError{
			Type:    ErrTypeJSONParse,
			Message: fmt.Sprintf("failed to parse config JSON: %s", configFile),
			Cause:   err,
		}
	}

	// Convert to GridConfig
	gridConfig := &config.GridConfig{
		Symbol:                 configFileData.Strategy.Symbol,
		Category:               configFileData.Strategy.Category,
		TradingMode:            configFileData.Strategy.TradingMode,
		Interval:               configFileData.Strategy.Interval,
		LowerBound:             configFileData.Strategy.LowerBound,
		UpperBound:             configFileData.Strategy.UpperBound,
		GridCount:              configFileData.Strategy.GridCount,
		GridSpacing:            configFileData.Strategy.GridSpacingPercent,
		ProfitPercent:          configFileData.Strategy.ProfitPercent,
		PositionSize:           configFileData.Strategy.PositionSize,
		Leverage:               configFileData.Strategy.Leverage,
		InitialBalance:         configFileData.Risk.InitialBalance,
		Commission:             configFileData.Risk.Commission,
		UseExchangeConstraints: configFileData.Strategy.UseExchangeConstraints,
		ExchangeName:           configFileData.Exchange.Name,
		MinOrderQty:            configFileData.Risk.MinOrderQty,
		QtyStep:                configFileData.Risk.QtyStep,
		TickSize:               configFileData.Risk.TickSize,
		MinNotional:            configFileData.Risk.MinNotional,
		MaxLeverage:            configFileData.Risk.MaxLeverage,
	}

	// Validate configuration
	if err := gridConfig.Validate(); err != nil {
		return nil, &ConfigError{
			Type:    ErrTypeValidation,
			Message: "configuration validation failed",
			Cause:   err,
		}
	}

	// Calculate grid levels
	if err := gridConfig.CalculateGridLevels(); err != nil {
		return nil, &ConfigError{
			Type:    ErrTypeValidation,
			Message: "failed to calculate grid levels",
			Cause:   err,
		}
	}

	return gridConfig, nil
}

// ApplyOverrides applies command-line overrides to the configuration
func (cl *ConfigLoader) ApplyOverrides(gridConfig *config.GridConfig, symbolOverride, intervalOverride string) {
	if symbolOverride != "" {
		gridConfig.Symbol = symbolOverride
	}
	if intervalOverride != "" {
		gridConfig.Interval = intervalOverride
	}
}

// ConfigError represents configuration-related errors
type ConfigError struct {
	Type    ConfigErrorType
	Message string
	Cause   error
}

func (e *ConfigError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("config error: %s: %v", e.Message, e.Cause)
	}
	return fmt.Sprintf("config error: %s", e.Message)
}

func (e *ConfigError) Unwrap() error {
	return e.Cause
}

// GridConfigFile represents the JSON structure for grid configuration files
type GridConfigFile struct {
	Strategy struct {
		Symbol                  string  `json:"symbol"`
		Category                string  `json:"category"`
		TradingMode             string  `json:"trading_mode"`
		Interval                string  `json:"interval"`
		LowerBound              float64 `json:"lower_bound"`
		UpperBound              float64 `json:"upper_bound"`
		GridCount               int     `json:"grid_count"`
		GridSpacingPercent      float64 `json:"grid_spacing_percent"`
		ProfitPercent           float64 `json:"profit_percent"`
		PositionSize            float64 `json:"position_size"`
		Leverage                float64 `json:"leverage"`
		UseExchangeConstraints  bool    `json:"use_exchange_constraints"`
	} `json:"strategy"`
	Exchange struct {
		Name string `json:"name"`
	} `json:"exchange"`
	Risk struct {
		InitialBalance float64 `json:"initial_balance"`
		Commission     float64 `json:"commission"`
		MinOrderQty    float64 `json:"min_order_qty"`
		QtyStep        float64 `json:"qty_step"`
		TickSize       float64 `json:"tick_size"`
		MinNotional    float64 `json:"min_notional"`
		MaxLeverage    float64 `json:"max_leverage"`
	} `json:"risk"`
}

// ConfigErrorType represents the type of configuration error
type ConfigErrorType int

const (
	ErrTypeFileNotFound ConfigErrorType = iota
	ErrTypeFileRead
	ErrTypeJSONParse
	ErrTypeValidation
)
