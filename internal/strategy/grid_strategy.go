package strategy

import (
	"fmt"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/grid"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// GridStrategy implements the Strategy interface using grid trading logic
type GridStrategy struct {
	name       string
	gridEngine *grid.GridEngine
	config     *config.GridConfig
	
	// Strategy state
	initialized bool
	lastPrice   float64
	lastAction  TradeAction
	
	// Performance tracking
	totalSignals    int
	executedSignals int
}

// NewGridStrategy creates a new grid trading strategy
func NewGridStrategy(name string, gridConfig *config.GridConfig) (*GridStrategy, error) {
	// Validate grid configuration
	if err := gridConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid grid configuration: %w", err)
	}
	
	// Create grid engine
	gridEngine, err := grid.NewGridEngine(gridConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create grid engine: %w", err)
	}
	
	strategy := &GridStrategy{
		name:            name,
		gridEngine:      gridEngine,
		config:          gridConfig,
		initialized:     false,
		lastPrice:       0,
		lastAction:      ActionHold,
		totalSignals:    0,
		executedSignals: 0,
	}
	
	return strategy, nil
}

// ShouldExecuteTrade analyzes current market data and returns grid-based trading decisions
func (gs *GridStrategy) ShouldExecuteTrade(data []types.OHLCV) (*TradeDecision, error) {
	if len(data) == 0 {
		return &TradeDecision{
			Action:    ActionHold,
			Amount:    0,
			Reason:    "No market data provided",
			Timestamp: time.Now(),
		}, nil
	}
	
	currentCandle := data[len(data)-1]
	currentPrice := currentCandle.Close
	
	gs.totalSignals++
	
	// Initialize strategy on first call
	if !gs.initialized {
		// Validate that current price is within grid range
		if err := gs.config.ValidateCurrentPrice(currentPrice); err != nil {
			return &TradeDecision{
				Action:    ActionHold,
				Amount:    0,
				Reason:    fmt.Sprintf("Price outside grid range: %v", err),
				Timestamp: currentCandle.Timestamp,
			}, nil
		}
		
		gs.initialized = true
		gs.lastPrice = currentPrice
		
		// No action on first candle - just initialize
		return &TradeDecision{
			Action:     ActionHold,
			Amount:     0,
			Confidence: 1.0,
			Strength:   0.0,
			Reason:     "Grid strategy initialized",
			Timestamp:  currentCandle.Timestamp,
		}, nil
	}
	
	// Store previous active positions count for comparison
	prevActivePositions := len(gs.gridEngine.GetActivePositions())
	
	// Process the current market tick through grid engine
	err := gs.gridEngine.ProcessTick(currentCandle)
	if err != nil {
		return nil, fmt.Errorf("grid engine processing error: %w", err)
	}
	
	// Get updated active positions count
	currentActivePositions := len(gs.gridEngine.GetActivePositions())
	
	// Determine what action the grid engine took
	var action TradeAction
	var amount float64
	var reason string
	var confidence float64
	var strength float64
	
	// Check if new positions were opened
	if currentActivePositions > prevActivePositions {
		// New position(s) opened
		newPositions := currentActivePositions - prevActivePositions
		
		// Determine action type based on trading mode and price movement
		if gs.config.TradingMode == config.TradingModeLong {
			action = ActionBuy
			reason = fmt.Sprintf("Grid buy triggered: %d new long position(s) at $%.2f", newPositions, currentPrice)
		} else if gs.config.TradingMode == config.TradingModeShort {
			action = ActionSell
			reason = fmt.Sprintf("Grid sell triggered: %d new short position(s) at $%.2f", newPositions, currentPrice)
		} else {
			// Both mode - determine based on price movement
			if currentPrice <= gs.lastPrice {
				action = ActionBuy
				reason = fmt.Sprintf("Grid buy triggered: %d new position(s) at $%.2f", newPositions, currentPrice)
			} else {
				action = ActionSell  
				reason = fmt.Sprintf("Grid sell triggered: %d new position(s) at $%.2f", newPositions, currentPrice)
			}
		}
		
		amount = float64(newPositions) * gs.config.PositionSize
		confidence = 1.0 // Grid triggers are deterministic
		strength = 0.8   // High strength for grid signals
		gs.executedSignals++
		
	} else if currentActivePositions < prevActivePositions {
		// Positions were closed (profit targets hit)
		closedPositions := prevActivePositions - currentActivePositions
		
		// This represents profit-taking, which is technically a sell for longs or buy for shorts
		if gs.config.TradingMode == config.TradingModeLong {
			action = ActionSell
			reason = fmt.Sprintf("Grid profit-taking: %d long position(s) closed at $%.2f", closedPositions, currentPrice)
		} else if gs.config.TradingMode == config.TradingModeShort {
			action = ActionBuy
			reason = fmt.Sprintf("Grid profit-taking: %d short position(s) closed at $%.2f", closedPositions, currentPrice)
		} else {
			// Both mode - this is always profit-taking
			action = ActionSell // Represent profit-taking as sell action
			reason = fmt.Sprintf("Grid profit-taking: %d position(s) closed at $%.2f", closedPositions, currentPrice)
		}
		
		amount = float64(closedPositions) * gs.config.PositionSize
		confidence = 1.0 // Profit targets are deterministic
		strength = 1.0   // Maximum strength for profit-taking
		gs.executedSignals++
		
	} else {
		// No change in positions
		action = ActionHold
		amount = 0
		reason = fmt.Sprintf("Grid monitoring: %d active positions at $%.2f", currentActivePositions, currentPrice)
		confidence = 0.0
		strength = 0.0
	}
	
	// Update state
	gs.lastPrice = currentPrice
	gs.lastAction = action
	
	decision := &TradeDecision{
		Action:     action,
		Amount:     amount,
		Confidence: confidence,
		Strength:   strength,
		Reason:     reason,
		Timestamp:  currentCandle.Timestamp,
	}
	
	return decision, nil
}

// GetName returns the strategy name
func (gs *GridStrategy) GetName() string {
	return gs.name
}

// OnCycleComplete is called when a take-profit cycle completes
// For grid strategies, this doesn't require special handling since
// each grid level is independent
func (gs *GridStrategy) OnCycleComplete() {
	// Grid strategies don't have traditional "cycles" like DCA
	// Each grid level operates independently
	// No special reset needed
}

// ResetForNewPeriod resets strategy state for walk-forward validation
func (gs *GridStrategy) ResetForNewPeriod() {
	// Reset strategy state but keep configuration
	gs.initialized = false
	gs.lastPrice = 0
	gs.lastAction = ActionHold
	gs.totalSignals = 0
	gs.executedSignals = 0
	
	// Create a new grid engine to reset all positions
	newEngine, err := grid.NewGridEngine(gs.config)
	if err != nil {
		// This should not happen since config was already validated
		// But handle gracefully
		return
	}
	
	gs.gridEngine = newEngine
}

// GetGridEngine returns the underlying grid engine for advanced operations
func (gs *GridStrategy) GetGridEngine() *grid.GridEngine {
	return gs.gridEngine
}

// GetStatistics returns current strategy statistics
func (gs *GridStrategy) GetStatistics() map[string]interface{} {
	gridStats := gs.gridEngine.GetStatistics()
	
	// Add strategy-specific statistics
	executionRate := 0.0
	if gs.totalSignals > 0 {
		executionRate = float64(gs.executedSignals) / float64(gs.totalSignals)
	}
	
	strategyStats := map[string]interface{}{
		"strategy_name":     gs.name,
		"total_signals":     gs.totalSignals,
		"executed_signals":  gs.executedSignals,
		"execution_rate":    executionRate,
		"last_price":        gs.lastPrice,
		"last_action":       gs.lastAction.String(),
		"initialized":       gs.initialized,
		"trading_mode":      gs.config.TradingMode,
	}
	
	// Merge grid engine statistics
	for k, v := range gridStats {
		strategyStats[k] = v
	}
	
	return strategyStats
}

// GetActivePositions returns current active positions from the grid engine
func (gs *GridStrategy) GetActivePositions() map[int]*grid.GridPosition {
	return gs.gridEngine.GetActivePositions()
}

// GetAllPositions returns all positions (active + closed) from the grid engine for reporting
func (gs *GridStrategy) GetAllPositions() []*grid.GridPosition {
	return gs.gridEngine.GetAllPositions()
}

// GetGridLevels returns all grid levels from the grid engine  
func (gs *GridStrategy) GetGridLevels() []*grid.GridLevel {
	return gs.gridEngine.GetGridLevels()
}

// GetConfiguration returns the grid configuration
func (gs *GridStrategy) GetConfiguration() *config.GridConfig {
	// Return a copy to prevent external modification
	configCopy := *gs.config
	return &configCopy
}

// IsWithinGridRange checks if a price is within the configured grid range
func (gs *GridStrategy) IsWithinGridRange(price float64) bool {
	return gs.config.ValidateCurrentPrice(price) == nil
}
