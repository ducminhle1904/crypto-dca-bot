package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
)

// GridConfig represents the configuration for grid trading strategy
type GridConfig struct {
	// Basic Configuration
	Symbol        string  `json:"symbol"`        // Trading pair symbol (e.g., "BTCUSDT")
	Category      string  `json:"category"`      // "linear" for futures, "spot" for spot trading
	TradingMode   string  `json:"trading_mode"`  // "long", "short", "both"
	
	// Grid Parameters
	LowerBound    float64 `json:"lower_bound"`    // Minimum price for grid operation
	UpperBound    float64 `json:"upper_bound"`    // Maximum price for grid operation
	GridCount     int     `json:"grid_count"`     // Total number of grid levels
	GridSpacing   float64 `json:"grid_spacing_percent"` // Percentage spacing between grids
	
	// Profit Configuration
	ProfitPercent float64 `json:"profit_percent"` // Profit target per grid (e.g., 0.01 = 1%)
	
	// Position Sizing
	PositionSize  float64 `json:"position_size"`  // Fixed amount per grid level in USDT
	Leverage      float64 `json:"leverage"`       // Leverage for futures (1.0 = no leverage)
	
	// Backtest Configuration
	InitialBalance float64 `json:"initial_balance"` // Starting balance in USDT
	Commission     float64 `json:"commission"`      // Trading commission rate
	DataFile       string  `json:"data_file"`       // Path to CSV data file
	Interval       string  `json:"interval"`        // Trading interval (e.g., "5m")
	
	// Exchange Integration
	UseExchangeConstraints bool   `json:"use_exchange_constraints"` // Whether to apply real exchange constraints
	ExchangeName          string `json:"exchange_name,omitempty"`  // Exchange name ("bybit", "binance", etc.)
	
	// Exchange Constraints (populated from exchange)
	MinOrderQty           float64 `json:"min_order_qty,omitempty"`   // Minimum order quantity
	MaxOrderQty           float64 `json:"max_order_qty,omitempty"`   // Maximum order quantity  
	QtyStep               float64 `json:"qty_step,omitempty"`        // Quantity step size
	TickSize              float64 `json:"tick_size,omitempty"`       // Price tick size
	MinNotional           float64 `json:"min_notional,omitempty"`    // Minimum notional value
	MaxLeverage           float64 `json:"max_leverage,omitempty"`    // Maximum leverage allowed
	
	// Calculated Fields (populated during initialization)
	GridLevels    []float64 `json:"-"` // Calculated grid price levels
	MaxPositions  int       `json:"-"` // Maximum concurrent positions based on trading mode
}

// TradingMode constants
const (
	TradingModeLong  = "long"
	TradingModeShort = "short"
	TradingModeBoth  = "both"
)

// Validate performs comprehensive validation of grid configuration
func (gc *GridConfig) Validate() error {
	// Basic field validation
	if gc.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	
	if gc.Category == "" {
		gc.Category = "linear" // Default to futures
	}
	
	// Trading mode validation
	switch gc.TradingMode {
	case TradingModeLong, TradingModeShort, TradingModeBoth:
		// Valid modes
	default:
		return fmt.Errorf("trading_mode must be 'long', 'short', or 'both', got: %s", gc.TradingMode)
	}
	
	// Price range validation
	if gc.LowerBound <= 0 {
		return fmt.Errorf("lower_bound must be positive, got: %f", gc.LowerBound)
	}
	
	if gc.UpperBound <= 0 {
		return fmt.Errorf("upper_bound must be positive, got: %f", gc.UpperBound)
	}
	
	if gc.UpperBound <= gc.LowerBound {
		return fmt.Errorf("upper_bound (%f) must be greater than lower_bound (%f)", gc.UpperBound, gc.LowerBound)
	}
	
	// Grid configuration validation
	if gc.GridCount <= 0 {
		return fmt.Errorf("grid_count must be positive, got: %d", gc.GridCount)
	}
	
	if gc.GridCount > 1000 {
		return fmt.Errorf("grid_count too large (max 1000), got: %d", gc.GridCount)
	}
	
	if gc.GridSpacing <= 0 {
		return fmt.Errorf("grid_spacing_percent must be positive, got: %f", gc.GridSpacing)
	}
	
	// Profit validation
	if gc.ProfitPercent <= 0 {
		return fmt.Errorf("profit_percent must be positive, got: %f", gc.ProfitPercent)
	}
	
	if gc.ProfitPercent > 1.0 {
		return fmt.Errorf("profit_percent seems too high (>100%%), got: %f", gc.ProfitPercent)
	}
	
	// Position sizing validation
	if gc.PositionSize <= 0 {
		return fmt.Errorf("position_size must be positive, got: %f", gc.PositionSize)
	}
	
	if gc.Leverage <= 0 {
		return fmt.Errorf("leverage must be positive, got: %f", gc.Leverage)
	}
	
	if gc.Leverage > 100 {
		return fmt.Errorf("leverage too high (max 100x), got: %f", gc.Leverage)
	}
	
	// Backtest validation
	if gc.InitialBalance <= 0 {
		return fmt.Errorf("initial_balance must be positive, got: %f", gc.InitialBalance)
	}
	
	if gc.Commission < 0 || gc.Commission > 0.01 {
		return fmt.Errorf("commission rate seems invalid (should be 0-1%%), got: %f", gc.Commission)
	}
	
	return nil
}

// CalculateGridLevels computes the actual price levels for the grid
func (gc *GridConfig) CalculateGridLevels() error {
	if err := gc.Validate(); err != nil {
		return fmt.Errorf("config validation failed: %w", err)
	}
	
	// Calculate grid levels using percentage spacing
	gc.GridLevels = make([]float64, 0, gc.GridCount)
	
	// Convert percentage spacing to decimal
	spacingPercent := gc.GridSpacing / 100.0
	
	// Use percentage-based spacing from lower bound
	currentPrice := gc.LowerBound
	
	for i := 0; i < gc.GridCount; i++ {
		if currentPrice > gc.UpperBound {
			break // Don't exceed upper bound
		}
		
		gc.GridLevels = append(gc.GridLevels, currentPrice)
		
		// Calculate next level using percentage spacing
		currentPrice = currentPrice * (1.0 + spacingPercent)
	}
	
	// Ensure we don't exceed the upper bound
	filteredLevels := make([]float64, 0, len(gc.GridLevels))
	for _, level := range gc.GridLevels {
		if level <= gc.UpperBound {
			filteredLevels = append(filteredLevels, level)
		}
	}
	
	gc.GridLevels = filteredLevels
	
	// Calculate max positions based on trading mode
	switch gc.TradingMode {
	case TradingModeLong, TradingModeShort:
		gc.MaxPositions = len(gc.GridLevels)
	case TradingModeBoth:
		gc.MaxPositions = len(gc.GridLevels) * 2 // Long and short for each level
	}
	
	if len(gc.GridLevels) == 0 {
		return fmt.Errorf("no valid grid levels calculated")
	}
	
	return nil
}

// ValidateCurrentPrice checks if the current price is within grid range
func (gc *GridConfig) ValidateCurrentPrice(currentPrice float64) error {
	if currentPrice < gc.LowerBound || currentPrice > gc.UpperBound {
		return fmt.Errorf("current price %f is outside grid range [%f, %f]", 
			currentPrice, gc.LowerBound, gc.UpperBound)
	}
	return nil
}

// CalculateRequiredBalance estimates the total balance needed for the grid
func (gc *GridConfig) CalculateRequiredBalance() float64 {
	if len(gc.GridLevels) == 0 {
		return 0
	}
	
	totalRequired := 0.0
	
	// Calculate required balance based on trading mode
	switch gc.TradingMode {
	case TradingModeLong:
		// For long mode, we buy at each level
		for range gc.GridLevels {
			margin := gc.PositionSize / gc.Leverage
			totalRequired += margin
		}
	case TradingModeShort:
		// For short mode, we sell at each level (margin requirement)
		for range gc.GridLevels {
			margin := gc.PositionSize / gc.Leverage
			totalRequired += margin
		}
	case TradingModeBoth:
		// For both modes, we need margin for both directions
		for range gc.GridLevels {
			margin := gc.PositionSize / gc.Leverage
			totalRequired += margin * 2 // Long and short
		}
	}
	
	// Add buffer for commission and price movements
	buffer := totalRequired * 0.1 // 10% buffer
	return totalRequired + buffer
}

// GetGridInfo returns formatted information about the grid configuration
func (gc *GridConfig) GetGridInfo() string {
	if len(gc.GridLevels) == 0 {
		return "Grid levels not calculated"
	}
	
	requiredBalance := gc.CalculateRequiredBalance()
	
	return fmt.Sprintf(
		"Grid Configuration:\n"+
		"  Symbol: %s (%s)\n"+
		"  Trading Mode: %s\n"+
		"  Price Range: $%.2f - $%.2f\n"+
		"  Grid Levels: %d\n"+
		"  Grid Spacing: %.2f%%\n"+
		"  Profit Target: %.2f%%\n"+
		"  Position Size: $%.2f\n"+
		"  Leverage: %.1fx\n"+
		"  Required Balance: $%.2f\n"+
		"  Available Balance: $%.2f",
		gc.Symbol, gc.Category, gc.TradingMode,
		gc.LowerBound, gc.UpperBound,
		len(gc.GridLevels), gc.GridSpacing, gc.ProfitPercent*100,
		gc.PositionSize, gc.Leverage,
		requiredBalance, gc.InitialBalance)
}

// LoadFromJSON loads grid configuration from JSON file
func LoadGridConfigFromJSON(filename string) (*GridConfig, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	var config GridConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}
	
	// Validate and calculate grid levels
	if err := config.CalculateGridLevels(); err != nil {
		return nil, fmt.Errorf("failed to calculate grid levels: %w", err)
	}
	
	return &config, nil
}

// ToJSON converts grid configuration to JSON string
func (gc *GridConfig) ToJSON() (string, error) {
	data, err := json.MarshalIndent(gc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config to JSON: %w", err)
	}
	return string(data), nil
}

// PopulateExchangeConstraints fetches and populates exchange constraints
func (gc *GridConfig) PopulateExchangeConstraints(ctx context.Context, exchange exchange.Exchange) error {
	if !gc.UseExchangeConstraints {
		return nil // Nothing to do if constraints are disabled
	}
	
	// Get instrument constraints from exchange
	constraints, err := exchange.GetInstrumentConstraints(ctx, gc.Category, gc.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get instrument constraints: %w", err)
	}
	
	// Populate configuration with exchange constraints
	gc.MinOrderQty = constraints.MinOrderQty
	gc.MaxOrderQty = constraints.MaxOrderQty
	gc.QtyStep = constraints.QtyStep
	gc.TickSize = constraints.TickSize
	gc.MinNotional = constraints.MinNotional
	gc.MaxLeverage = constraints.MaxLeverage
	gc.ExchangeName = exchange.GetName()
	
	return nil
}

// CalculateQuantityForGrid calculates the exact quantity needed for a grid position
// considering exchange constraints and minimum order requirements
func (gc *GridConfig) CalculateQuantityForGrid(price float64) float64 {
	// Start with position size in quote currency (USDT)
	notionalValue := gc.PositionSize
	
	// Convert to base currency quantity
	baseQuantity := notionalValue / price
	
	// Apply exchange constraints if enabled
	if gc.UseExchangeConstraints {
		// Apply minimum order quantity constraint
		if baseQuantity < gc.MinOrderQty {
			baseQuantity = gc.MinOrderQty
		}
		
		// Apply quantity step constraint
		if gc.QtyStep > 0 {
			// Round to nearest step
			steps := baseQuantity / gc.QtyStep
			baseQuantity = float64(int64(steps+0.5)) * gc.QtyStep
		}
		
		// Ensure minimum notional value is met
		if gc.MinNotional > 0 {
			minQuantityForNotional := gc.MinNotional / price
			if baseQuantity < minQuantityForNotional {
				baseQuantity = minQuantityForNotional
				
				// Re-apply step constraint after notional adjustment
				if gc.QtyStep > 0 {
					steps := baseQuantity / gc.QtyStep
					baseQuantity = float64(int64(steps+0.5)) * gc.QtyStep
				}
			}
		}
		
		// Apply maximum order quantity constraint (only if set)
		if gc.MaxOrderQty > 0 && baseQuantity > gc.MaxOrderQty {
			baseQuantity = gc.MaxOrderQty
		}
	}
	return baseQuantity
}

// ValidateExchangeConstraints validates that the configuration is compatible with exchange constraints
func (gc *GridConfig) ValidateExchangeConstraints(ctx context.Context, exchange exchange.Exchange) error {
	if !gc.UseExchangeConstraints {
		return nil // Skip validation if constraints are disabled
	}
	
	// Populate constraints first
	if err := gc.PopulateExchangeConstraints(ctx, exchange); err != nil {
		return err
	}
	
	// Validate leverage against exchange limits
	if gc.Leverage > gc.MaxLeverage {
		return fmt.Errorf("leverage %.2f exceeds exchange maximum %.2f for %s", 
			gc.Leverage, gc.MaxLeverage, gc.Symbol)
	}
	
	// Validate that position size can meet minimum requirements
	avgPrice := (gc.LowerBound + gc.UpperBound) / 2
	requiredQuantity := gc.CalculateQuantityForGrid(avgPrice)
	
	if requiredQuantity > gc.MaxOrderQty {
		return fmt.Errorf("calculated quantity %.6f exceeds exchange maximum %.6f for %s",
			requiredQuantity, gc.MaxOrderQty, gc.Symbol)
	}
	
	// Check minimum notional value requirements
	if gc.MinNotional > 0 && gc.PositionSize < gc.MinNotional {
		return fmt.Errorf("position size %.2f is below minimum notional %.2f for %s",
			gc.PositionSize, gc.MinNotional, gc.Symbol)
	}
	
	return nil
}

// GetExchangeInfo returns a formatted string with exchange constraint information
func (gc *GridConfig) GetExchangeInfo() string {
	if !gc.UseExchangeConstraints {
		return "Exchange constraints: DISABLED"
	}
	
	return fmt.Sprintf("Exchange: %s | Min Qty: %.6f | Step: %.6f | Min Notional: %.2f | Max Leverage: %.1fx",
		gc.ExchangeName, gc.MinOrderQty, gc.QtyStep, gc.MinNotional, gc.MaxLeverage)
}
