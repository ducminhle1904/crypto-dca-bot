package grid

import (
	"fmt"
	"sort"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// GridLevel represents a single level in the grid with its position tracking
type GridLevel struct {
	Level         int            `json:"level"`          // Grid level number (1, 2, 3, ...)
	Price         float64        `json:"price"`          // Price at this grid level
	Direction     string         `json:"direction"`      // "long" or "short"
	IsActive      bool           `json:"is_active"`      // Whether this level has an active position
	Position      *GridPosition  `json:"position"`       // Current position at this level
	ProfitTarget  float64        `json:"profit_target"`  // Target price for profit taking
	TriggerTime   *time.Time     `json:"trigger_time"`   // When this level was last triggered
}

// GridPosition represents an individual position at a grid level
type GridPosition struct {
	// Grid Information
	GridLevel     int        `json:"grid_level"`     // Which grid level this position belongs to
	
	// Entry Information
	EntryTime     time.Time  `json:"entry_time"`     // When position was opened
	EntryPrice    float64    `json:"entry_price"`    // Exact entry price
	Quantity      float64    `json:"quantity"`       // Position size in base currency
	Commission    float64    `json:"commission"`     // Commission paid on entry
	
	// Position Details
	Direction     string     `json:"direction"`      // "long" or "short"
	MarginUsed    float64    `json:"margin_used"`    // Margin allocated for this position
	Notional      float64    `json:"notional"`       // Position notional value (quantity * price)
	
	// P&L Tracking
	UnrealizedPnL float64    `json:"unrealized_pnl"` // Current unrealized P&L
	MarkPrice     float64    `json:"mark_price"`     // Current mark price for P&L calculation
	
	// Exit Information (optional - only set when position is closed)
	ExitTime      *time.Time `json:"exit_time,omitempty"`      // When position was closed
	ExitPrice     *float64   `json:"exit_price,omitempty"`     // Exit price
	ExitCommission *float64  `json:"exit_commission,omitempty"` // Commission paid on exit
	RealizedPnL   *float64   `json:"realized_pnl,omitempty"`   // Final realized P&L
	
	// Status
	Status        string     `json:"status"`         // "open", "closed", "cancelled"
}

// GridEngine manages the grid trading logic and position tracking
type GridEngine struct {
	// Configuration
	config         *config.GridConfig `json:"config"`
	
	// Grid Management
	gridLevels     []*GridLevel       `json:"grid_levels"`      // All grid levels
	levelIndex     map[int]*GridLevel `json:"-"`                // Fast lookup map for levels by level number
	activePositions map[int]*GridPosition `json:"active_positions"` // Active positions by level
	closedPositions []*GridPosition   `json:"closed_positions"` // Historical closed positions for reporting
	
	// Balance and Margin Tracking
	availableBalance float64           `json:"available_balance"` // Available balance for new positions
	totalMarginUsed  float64           `json:"total_margin_used"` // Total margin currently used
	totalUnrealized  float64           `json:"total_unrealized"`  // Total unrealized P&L
	totalRealized    float64           `json:"total_realized"`    // Total realized P&L
	
	// Performance Tracking
	totalTrades      int               `json:"total_trades"`       // Total number of completed trades
	successfulTrades int               `json:"successful_trades"`  // Number of profitable trades
	maxConcurrentPos int               `json:"max_concurrent_pos"` // Maximum concurrent positions held
	
	// Statistics
	startTime        *time.Time        `json:"start_time,omitempty"`   // Backtest start time
	currentTime      *time.Time        `json:"current_time,omitempty"` // Current backtest time
}

// NewGridEngine creates a new grid engine with the given configuration
func NewGridEngine(config *config.GridConfig) (*GridEngine, error) {
	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}
	
	// Calculate grid levels if not already done
	if len(config.GridLevels) == 0 {
		if err := config.CalculateGridLevels(); err != nil {
			return nil, fmt.Errorf("failed to calculate grid levels: %w", err)
		}
	}
	
	engine := &GridEngine{
		config:           config,
		gridLevels:       make([]*GridLevel, 0, len(config.GridLevels)),
		levelIndex:       make(map[int]*GridLevel),
		activePositions:  make(map[int]*GridPosition),
		closedPositions:  make([]*GridPosition, 0), // Historical positions for reporting
		availableBalance: config.InitialBalance,
		totalMarginUsed:  0,
		totalUnrealized:  0,
		totalRealized:    0,
		totalTrades:      0,
		successfulTrades: 0,
		maxConcurrentPos: 0,
	}
	
	// Initialize grid levels
	if err := engine.initializeGridLevels(); err != nil {
		return nil, fmt.Errorf("failed to initialize grid levels: %w", err)
	}
	
	return engine, nil
}

// initializeGridLevels creates GridLevel structures from configuration
func (ge *GridEngine) initializeGridLevels() error {
	for i, price := range ge.config.GridLevels {
		// Calculate profit target for this level
		profitTarget := price * (1.0 + ge.config.ProfitPercent)
		
		// Create grid levels based on trading mode
		switch ge.config.TradingMode {
		case config.TradingModeLong:
			level := &GridLevel{
				Level:        i + 1,
				Price:        price,
				Direction:    "long",
				IsActive:     false,
				Position:     nil,
				ProfitTarget: profitTarget,
			}
			ge.gridLevels = append(ge.gridLevels, level)
			ge.levelIndex[level.Level] = level
			
		case config.TradingModeShort:
			// For short positions, profit target is below entry price
			profitTarget = price * (1.0 - ge.config.ProfitPercent)
			level := &GridLevel{
				Level:        i + 1,
				Price:        price,
				Direction:    "short",
				IsActive:     false,
				Position:     nil,
				ProfitTarget: profitTarget,
			}
			ge.gridLevels = append(ge.gridLevels, level)
			ge.levelIndex[level.Level] = level
			
		case config.TradingModeBoth:
			// For "both" mode, create realistic long/short distribution:
			// - Lower half of price range: long positions (buy when price drops)
			// - Upper half of price range: short positions (sell when price rises)
			midPrice := (ge.config.LowerBound + ge.config.UpperBound) / 2
			
			if price <= midPrice {
				// Lower half: create long positions to buy on dips
				level := &GridLevel{
					Level:        i + 1,
					Price:        price,
					Direction:    "long",
					IsActive:     false,
					Position:     nil,
					ProfitTarget: price * (1.0 + ge.config.ProfitPercent),
				}
				ge.gridLevels = append(ge.gridLevels, level)
				ge.levelIndex[level.Level] = level
			} else {
				// Upper half: create short positions to sell on rallies
				level := &GridLevel{
					Level:        i + 1,
					Price:        price,
					Direction:    "short",
					IsActive:     false,
					Position:     nil,
					ProfitTarget: price * (1.0 - ge.config.ProfitPercent),
				}
				ge.gridLevels = append(ge.gridLevels, level)
				ge.levelIndex[level.Level] = level
			}
			
		default:
			return fmt.Errorf("unsupported trading mode: %s", ge.config.TradingMode)
		}
	}
	
	return nil
}

// ProcessTick processes a new price tick and executes grid logic using OHLCV data
func (ge *GridEngine) ProcessTick(tick types.OHLCV) error {
	currentTime := tick.Timestamp
	
	// Set start time on first tick
	if ge.startTime == nil {
		ge.startTime = &currentTime
	}
	ge.currentTime = &currentTime
	
	// Update unrealized P&L for all open positions using current close price
	ge.updateUnrealizedPnL(tick.Close)
	
	// Check for new position entries using the OHLCV range (intrabar analysis)
	if err := ge.checkGridTriggersOHLCV(tick, currentTime); err != nil {
		return fmt.Errorf("error checking grid triggers: %w", err)
	}
	
	// Check for profit targets using the OHLCV range
	if err := ge.checkProfitTargetsOHLCV(tick, currentTime); err != nil {
		return fmt.Errorf("error checking profit targets: %w", err)
	}
	
	// Update statistics
	ge.updateStatistics()
	
	return nil
}

// checkGridTriggersOHLCV checks if the OHLCV price range has crossed any grid levels
func (ge *GridEngine) checkGridTriggersOHLCV(tick types.OHLCV, currentTime time.Time) error {
	// Collect all triggered levels to process them in the correct order
	var triggeredLevels []*GridLevel
	
	for _, level := range ge.gridLevels {
		// Skip if this level already has an active position
		if level.IsActive {
			continue
		}
		
		// Check if the OHLCV price range crossed this grid level
		if ge.priceRangeCrossedLevel(tick, level.Price) {
			triggeredLevels = append(triggeredLevels, level)
		}
	}
	
	// Process triggered levels in order based on price movement direction
	if len(triggeredLevels) > 0 {
		// Sort levels by price to process them in the order they would be hit
		sortedLevels := ge.sortLevelsByExecutionOrder(triggeredLevels, tick)
		
		// Track available balance for this processing cycle to avoid race conditions
		availableForCycle := ge.availableBalance
		
		// Process each triggered level with running balance check
		for _, level := range sortedLevels {
			// Double-check that level is still inactive
			if level.IsActive {
				continue
			}
			
			// Calculate margin required for this position
			quantity := ge.config.CalculateQuantityForGrid(level.Price)
			positionNotional := quantity * level.Price
			marginRequired := positionNotional / ge.config.Leverage
			
			// Check if we have enough balance remaining in this cycle
			if marginRequired > availableForCycle {
				// Not enough balance for this level, skip remaining levels
				break
			}
			
			// Open position at the grid level price
			if err := ge.openPosition(level, level.Price, currentTime); err != nil {
				// Log the error but continue processing other levels
				continue
			}
			
			// Update available balance for remaining levels in this cycle
			availableForCycle -= marginRequired
		}
	}
	
	return nil
}

// priceRangeCrossedLevel checks if the OHLCV candle price range crossed a specific grid level
func (ge *GridEngine) priceRangeCrossedLevel(tick types.OHLCV, gridPrice float64) bool {
	// The grid level is crossed if it falls within the High-Low range of the candle
	return gridPrice >= tick.Low && gridPrice <= tick.High
}

// sortLevelsByExecutionOrder sorts triggered levels in the order they would be executed
func (ge *GridEngine) sortLevelsByExecutionOrder(levels []*GridLevel, tick types.OHLCV) []*GridLevel {
	// Create a copy to avoid modifying the original slice
	sortedLevels := make([]*GridLevel, len(levels))
	copy(sortedLevels, levels)
	
	// Determine price movement direction
	priceMovedUp := tick.Close > tick.Open
	
	if priceMovedUp {
		// Price moved up: levels would be hit from lowest to highest
		sort.Slice(sortedLevels, func(i, j int) bool {
			return sortedLevels[i].Price < sortedLevels[j].Price
		})
	} else {
		// Price moved down: levels would be hit from highest to lowest
		sort.Slice(sortedLevels, func(i, j int) bool {
			return sortedLevels[i].Price > sortedLevels[j].Price
		})
	}
	
	return sortedLevels
}

// openPosition opens a new position at the specified grid level
func (ge *GridEngine) openPosition(level *GridLevel, currentPrice float64, currentTime time.Time) error {
	// Calculate quantity based on exchange constraints if available
	quantity := ge.config.CalculateQuantityForGrid(currentPrice)
	
	// Recalculate notional based on actual quantity (may be adjusted by exchange constraints)
	positionNotional := quantity * currentPrice
	marginRequired := positionNotional / ge.config.Leverage
	commission := positionNotional * ge.config.Commission
	
	// Check if we have enough balance (only check margin, commission is handled in P&L)
	if marginRequired > ge.availableBalance {
		// Not enough balance - skip this trade
		return nil
	}
	
	// Create new position
	position := &GridPosition{
		GridLevel:     level.Level,          // Track which grid level this position belongs to
		EntryTime:     currentTime,
		EntryPrice:    currentPrice,
		Quantity:      quantity,
		Commission:    commission,
		Direction:     level.Direction,
		MarginUsed:    marginRequired,
		Notional:      positionNotional,
		UnrealizedPnL: 0, // Will be calculated in next tick
		MarkPrice:     currentPrice,
		Status:        "open",
	}
	
	// Update level and engine state
	level.IsActive = true
	level.Position = position
	level.TriggerTime = &currentTime
	ge.activePositions[level.Level] = position
	
	// Update balances (commission is accounted in P&L, not balance)
	ge.availableBalance -= marginRequired
	ge.totalMarginUsed += marginRequired
	
	return nil
}

// checkProfitTargetsOHLCV checks if any open positions have reached their profit targets using OHLCV range
func (ge *GridEngine) checkProfitTargetsOHLCV(tick types.OHLCV, currentTime time.Time) error {
	for levelNum, position := range ge.activePositions {
		if position.Status != "open" {
			continue
		}
		
		// Fast lookup of the corresponding grid level
		targetLevel, exists := ge.levelIndex[levelNum]
		if !exists {
			continue
		}
		
		// Check if profit target was reached within the OHLCV range
		shouldClose := false
		var exitPrice float64
		
		if position.Direction == "long" {
			// For long positions, check if the High reached or exceeded the profit target
			if tick.High >= targetLevel.ProfitTarget {
				shouldClose = true
				// Apply realistic slippage - long positions sell at slightly lower price
				slippageMultiplier := 1.0 - (ge.config.SlippageBps / 10000.0)
				exitPrice = targetLevel.ProfitTarget * slippageMultiplier
				
				// Ensure we don't exit below the tick high (price was available)
				if exitPrice > tick.High {
					exitPrice = tick.High
				}
			}
		} else if position.Direction == "short" {
			// For short positions, check if the Low reached or went below the profit target
			if tick.Low <= targetLevel.ProfitTarget {
				shouldClose = true
				// Apply realistic slippage - short positions cover at slightly higher price
				slippageMultiplier := 1.0 + (ge.config.SlippageBps / 10000.0)
				exitPrice = targetLevel.ProfitTarget * slippageMultiplier
				
				// Ensure we don't exit below the tick low (price was available)
				if exitPrice < tick.Low {
					exitPrice = tick.Low
				}
			}
		}
		
		if shouldClose {
			if err := ge.closePosition(targetLevel, position, exitPrice, currentTime); err != nil {
				return fmt.Errorf("failed to close position at level %d: %w", levelNum, err)
			}
		}
	}
	
	return nil
}

// closePosition closes an open position and realizes P&L
func (ge *GridEngine) closePosition(level *GridLevel, position *GridPosition, currentPrice float64, currentTime time.Time) error {
	// Calculate exit commission
	exitNotional := position.Quantity * currentPrice
	exitCommission := exitNotional * ge.config.Commission
	
	// Calculate realized P&L
	var pnl float64
	if position.Direction == "long" {
		pnl = (currentPrice - position.EntryPrice) * position.Quantity
	} else {
		pnl = (position.EntryPrice - currentPrice) * position.Quantity
	}
	
	// Subtract commissions
	realizedPnL := pnl - position.Commission - exitCommission
	
	
	// Update position with exit information
	position.ExitTime = &currentTime
	position.ExitPrice = &currentPrice
	position.ExitCommission = &exitCommission
	position.RealizedPnL = &realizedPnL
	position.Status = "closed"
	
	// Update level state - clear position reference to maintain sync
	level.IsActive = false
	level.Position = nil
	
	// Store closed position for reporting (before removing from active positions)
	ge.closedPositions = append(ge.closedPositions, position)
	
	// Update engine state - remove from active positions map
	delete(ge.activePositions, level.Level)
	ge.availableBalance += position.MarginUsed + realizedPnL
	ge.totalMarginUsed -= position.MarginUsed
	ge.totalRealized += realizedPnL
	ge.totalTrades++
	
	if realizedPnL > 0 {
		ge.successfulTrades++
	}
	
	return nil
}

// updateUnrealizedPnL updates the unrealized P&L for all open positions
func (ge *GridEngine) updateUnrealizedPnL(currentPrice float64) {
	totalUnrealized := 0.0
	
	for _, position := range ge.activePositions {
		if position.Status != "open" {
			continue
		}
		
		// Calculate unrealized P&L
		var pnl float64
		if position.Direction == "long" {
			pnl = (currentPrice - position.EntryPrice) * position.Quantity
		} else {
			pnl = (position.EntryPrice - currentPrice) * position.Quantity
		}
		
		position.UnrealizedPnL = pnl
		position.MarkPrice = currentPrice
		totalUnrealized += pnl
	}
	
	ge.totalUnrealized = totalUnrealized
}

// updateStatistics updates engine statistics
func (ge *GridEngine) updateStatistics() {
	currentConcurrent := len(ge.activePositions)
	if currentConcurrent > ge.maxConcurrentPos {
		ge.maxConcurrentPos = currentConcurrent
	}
}

// GetCurrentBalance returns the current total balance (available + margin + unrealized P&L)
func (ge *GridEngine) GetCurrentBalance() float64 {
	return ge.availableBalance + ge.totalMarginUsed + ge.totalUnrealized
}

// GetStatistics returns current engine statistics
func (ge *GridEngine) GetStatistics() map[string]interface{} {
	currentBalance := ge.GetCurrentBalance()
	totalReturn := (currentBalance - ge.config.InitialBalance) / ge.config.InitialBalance
	
	winRate := 0.0
	if ge.totalTrades > 0 {
		winRate = float64(ge.successfulTrades) / float64(ge.totalTrades)
	}
	
	return map[string]interface{}{
		"initial_balance":      ge.config.InitialBalance,
		"current_balance":      currentBalance,
		"available_balance":    ge.availableBalance,
		"total_margin_used":    ge.totalMarginUsed,
		"total_unrealized":     ge.totalUnrealized,
		"total_realized":       ge.totalRealized,
		"total_return":         totalReturn,
		"total_trades":         ge.totalTrades,
		"successful_trades":    ge.successfulTrades,
		"win_rate":             winRate,
		"active_positions":     len(ge.activePositions),
		"max_concurrent_pos":   ge.maxConcurrentPos,
		"grid_levels":          len(ge.gridLevels),
		"trading_mode":         ge.config.TradingMode,
	}
}

// GetActivePositions returns a copy of all active positions
func (ge *GridEngine) GetActivePositions() map[int]*GridPosition {
	positions := make(map[int]*GridPosition)
	for k, v := range ge.activePositions {
		// Create a copy to avoid external modifications
		posCopy := *v
		positions[k] = &posCopy
	}
	return positions
}

// GetClosedPositions returns a copy of all closed positions for reporting
func (ge *GridEngine) GetClosedPositions() []*GridPosition {
	closedPositions := make([]*GridPosition, len(ge.closedPositions))
	for i, pos := range ge.closedPositions {
		// Create a copy to prevent external modification
		posCopy := *pos
		closedPositions[i] = &posCopy
	}
	return closedPositions
}

// GetAllPositions returns all positions (active + closed) for comprehensive reporting
func (ge *GridEngine) GetAllPositions() []*GridPosition {
	allPositions := make([]*GridPosition, 0, len(ge.activePositions)+len(ge.closedPositions))
	
	// Add closed positions first
	for _, pos := range ge.closedPositions {
		posCopy := *pos
		allPositions = append(allPositions, &posCopy)
	}
	
	// Add active positions
	for _, pos := range ge.activePositions {
		posCopy := *pos
		allPositions = append(allPositions, &posCopy)
	}
	
	return allPositions
}

// GetGridLevels returns a copy of all grid levels
func (ge *GridEngine) GetGridLevels() []*GridLevel {
	levels := make([]*GridLevel, len(ge.gridLevels))
	for i, level := range ge.gridLevels {
		// Create a copy to avoid external modifications
		levelCopy := *level
		if level.Position != nil {
			posCopy := *level.Position
			levelCopy.Position = &posCopy
		}
		levels[i] = &levelCopy
	}
	return levels
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
