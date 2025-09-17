package portfolio

import (
	"fmt"
	"sync"
	"time"
)

// AllocationManager handles balance allocation and tracking across multiple bots
type AllocationManager struct {
	mu                sync.RWMutex
	totalBalance      float64
	allocations       map[string]*BotAllocation
	allocationHistory []AllocationEvent
	rebalanceConfig   *RebalanceConfig
}

// RebalanceConfig holds rebalancing configuration
type RebalanceConfig struct {
	Strategy           string        `json:"strategy"`            // equal_weight, performance_based, custom
	MinRebalanceAmount float64       `json:"min_rebalance_amount"` // Minimum amount to trigger rebalance
	RebalanceThreshold float64       `json:"rebalance_threshold"`  // % deviation to trigger rebalance
	RebalanceInterval  time.Duration `json:"rebalance_interval"`   // How often to check for rebalancing
	LastRebalance      time.Time     `json:"last_rebalance"`       // Last rebalancing time
}

// AllocationEvent represents a balance allocation or rebalancing event
type AllocationEvent struct {
	Timestamp   time.Time                `json:"timestamp"`
	EventType   string                   `json:"event_type"`   // ALLOCATE, DEALLOCATE, REBALANCE, PROFIT_SHARE
	BotID       string                   `json:"bot_id"`
	Amount      float64                  `json:"amount"`
	Description string                   `json:"description"`
	BeforeState map[string]*BotAllocation `json:"before_state,omitempty"`
	AfterState  map[string]*BotAllocation `json:"after_state,omitempty"`
}

// NewAllocationManager creates a new allocation manager
func NewAllocationManager(totalBalance float64, strategy string) *AllocationManager {
	return &AllocationManager{
		totalBalance:      totalBalance,
		allocations:       make(map[string]*BotAllocation),
		allocationHistory: make([]AllocationEvent, 0),
		rebalanceConfig: &RebalanceConfig{
			Strategy:           strategy,
			MinRebalanceAmount: 10.0, // $10 minimum
			RebalanceThreshold: 0.1,  // 10% deviation
			RebalanceInterval:  1 * time.Hour,
			LastRebalance:      time.Now(),
		},
	}
}

// AllocateToBot allocates balance to a specific bot
func (am *AllocationManager) AllocateToBot(botID string, config BotConfig) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	// Validate configuration parameters
	if config.Leverage <= 0 {
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   fmt.Sprintf("Invalid leverage %.1f: must be greater than 0", config.Leverage),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	if config.Leverage > 125.0 { // Maximum leverage commonly allowed by exchanges
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   fmt.Sprintf("Invalid leverage %.1f: must be less than or equal to 125x", config.Leverage),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	if config.AllocationPercentage <= 0 || config.AllocationPercentage > 1.0 {
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   fmt.Sprintf("Invalid allocation percentage %.1f%%: must be between 0%% and 100%%", config.AllocationPercentage*100),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	if config.Symbol == "" {
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   "Symbol cannot be empty",
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Check if bot already has allocation
	if _, exists := am.allocations[botID]; exists {
		return &PortfolioError{
			Code:      ErrBotAlreadyRegistered,
			Message:   fmt.Sprintf("Bot %s already has allocation", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Calculate allocation amount
	allocationAmount := am.totalBalance * config.AllocationPercentage
	
	// Check if we have enough balance
	totalAllocated := am.getTotalAllocatedBalance()
	if totalAllocated+allocationAmount > am.totalBalance {
		return &PortfolioError{
			Code:      ErrExceedsAllocation,
			Message:   fmt.Sprintf("Allocation $%.2f would exceed total balance $%.2f (already allocated: $%.2f)", 
				allocationAmount, am.totalBalance, totalAllocated),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Create bot allocation
	allocation := &BotAllocation{
		BotID:                botID,
		Symbol:               config.Symbol,
		AllocatedBalance:     allocationAmount,
		UsedBalance:          0.0,
		AvailableBalance:     allocationAmount,
		CurrentPosition:      0.0,
		AveragePrice:         0.0,
		Leverage:             config.Leverage,
		UnrealizedPnL:        0.0,
		RealizedPnL:          0.0,
		LastUpdated:          time.Now(),
		PositionMarginUsed:   0.0,
		AllocationPercentage: config.AllocationPercentage,
	}

	am.allocations[botID] = allocation

	// Record allocation event
	event := AllocationEvent{
		Timestamp:   time.Now(),
		EventType:   "ALLOCATE",
		BotID:       botID,
		Amount:      allocationAmount,
		Description: fmt.Sprintf("Initial allocation: %.1f%% = $%.2f", config.AllocationPercentage*100, allocationAmount),
		AfterState:  am.copyAllocations(),
	}
	am.allocationHistory = append(am.allocationHistory, event)

	return nil
}

// DeallocateFromBot removes allocation from a bot
func (am *AllocationManager) DeallocateFromBot(botID string) (*BotAllocation, error) {
	am.mu.Lock()
	defer am.mu.Unlock()

	allocation, exists := am.allocations[botID]
	if !exists {
		return nil, &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s has no allocation", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Check if bot has open positions
	if allocation.CurrentPosition > 0 {
		return nil, &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   fmt.Sprintf("Cannot deallocate from bot %s with open position: $%.2f", botID, allocation.CurrentPosition),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Create copy for return
	allocationCopy := *allocation
	
	// Remove allocation
	delete(am.allocations, botID)

	// Record deallocation event
	event := AllocationEvent{
		Timestamp:   time.Now(),
		EventType:   "DEALLOCATE",
		BotID:       botID,
		Amount:      allocation.AllocatedBalance,
		Description: fmt.Sprintf("Deallocated $%.2f", allocation.AllocatedBalance),
		AfterState:  am.copyAllocations(),
	}
	am.allocationHistory = append(am.allocationHistory, event)

	return &allocationCopy, nil
}

// UpdateBotPosition updates position information for a bot
func (am *AllocationManager) UpdateBotPosition(botID string, positionValue, avgPrice, leverage float64) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	allocation, exists := am.allocations[botID]
	if !exists {
		return &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s has no allocation", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Calculate margin used for this position
	leverageCalc := NewLeverageCalculator()
	marginUsed := leverageCalc.CalculateRequiredMargin(positionValue, leverage)

	// Check if margin usage is within allocation
	if marginUsed > allocation.AllocatedBalance {
		return &PortfolioError{
			Code:      ErrExceedsAllocation,
			Message:   fmt.Sprintf("Margin required $%.2f exceeds allocated balance $%.2f", 
				marginUsed, allocation.AllocatedBalance),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Update position information
	allocation.CurrentPosition = positionValue
	allocation.AveragePrice = avgPrice
	allocation.Leverage = leverage
	allocation.PositionMarginUsed = marginUsed
	allocation.UsedBalance = marginUsed
	allocation.AvailableBalance = allocation.AllocatedBalance - marginUsed
	allocation.LastUpdated = time.Now()

	return nil
}

// RecordProfit records profit for a bot and handles profit sharing
func (am *AllocationManager) RecordProfit(botID string, profit float64, shareProfit bool) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	allocation, exists := am.allocations[botID]
	if !exists {
		return &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s has no allocation", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Update realized profit
	allocation.RealizedPnL += profit
	allocation.LastUpdated = time.Now()

	// Record profit event
	event := AllocationEvent{
		Timestamp:   time.Now(),
		EventType:   "PROFIT_RECORD",
		BotID:       botID,
		Amount:      profit,
		Description: fmt.Sprintf("Profit recorded: $%.2f (Total: $%.2f)", profit, allocation.RealizedPnL),
	}
	am.allocationHistory = append(am.allocationHistory, event)

	// Handle profit sharing if enabled
	if shareProfit && profit > 0 {
		return am.redistributeProfit(botID, profit)
	}

	return nil
}

// redistributeProfit redistributes profit across all bots based on strategy
func (am *AllocationManager) redistributeProfit(sourceBotID string, profit float64) error {
	switch am.rebalanceConfig.Strategy {
	case "equal_weight":
		return am.redistributeProfitEqually(sourceBotID, profit)
	case "performance_based":
		return am.redistributeProfitByPerformance(sourceBotID, profit)
	default:
		// No redistribution for other strategies
		return nil
	}
}

// redistributeProfitEqually distributes profit equally across all bots
func (am *AllocationManager) redistributeProfitEqually(sourceBotID string, profit float64) error {
	if len(am.allocations) <= 1 {
		return nil // No other bots to share with
	}

	// Calculate equal share for each bot (including source)
	sharePerBot := profit / float64(len(am.allocations))

	// Update total balance to reflect profit
	am.totalBalance += profit
	
	// Distribute to all bots (including source gets its share back)
	for _, allocation := range am.allocations {
		allocation.AllocatedBalance += sharePerBot
		allocation.AvailableBalance += sharePerBot
		allocation.LastUpdated = time.Now()
	}

	// Record redistribution event
	event := AllocationEvent{
		Timestamp:   time.Now(),
		EventType:   "PROFIT_SHARE",
		BotID:       sourceBotID,
		Amount:      profit,
		Description: fmt.Sprintf("Profit $%.2f redistributed equally: $%.2f per bot", profit, sharePerBot),
		AfterState:  am.copyAllocations(),
	}
	am.allocationHistory = append(am.allocationHistory, event)

	return nil
}

// redistributeProfitByPerformance distributes profit based on historical performance
func (am *AllocationManager) redistributeProfitByPerformance(sourceBotID string, profit float64) error {
	// Calculate total realized PnL for all bots
	totalPnL := 0.0
	positivePnLBots := make(map[string]float64)
	
	for botID, allocation := range am.allocations {
		if allocation.RealizedPnL > 0 {
			positivePnLBots[botID] = allocation.RealizedPnL
			totalPnL += allocation.RealizedPnL
		}
	}

	if totalPnL <= 0 || len(positivePnLBots) == 0 {
		// No positive performance to base distribution on, fall back to equal distribution
		return am.redistributeProfitEqually(sourceBotID, profit)
	}

	// Update total balance to reflect profit
	am.totalBalance += profit
	
	// Distribute based on performance ratio
	for botID, allocation := range am.allocations {
		if pnl, exists := positivePnLBots[botID]; exists {
			performanceRatio := pnl / totalPnL
			shareAmount := profit * performanceRatio
			
			allocation.AllocatedBalance += shareAmount
			allocation.AvailableBalance += shareAmount
			allocation.LastUpdated = time.Now()
		}
	}

	// Record redistribution event
	event := AllocationEvent{
		Timestamp:   time.Now(),
		EventType:   "PROFIT_SHARE",
		BotID:       sourceBotID,
		Amount:      profit,
		Description: fmt.Sprintf("Profit $%.2f redistributed by performance", profit),
		AfterState:  am.copyAllocations(),
	}
	am.allocationHistory = append(am.allocationHistory, event)

	return nil
}

// CheckRebalanceNeeded checks if portfolio needs rebalancing
func (am *AllocationManager) CheckRebalanceNeeded() (bool, []string) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	// Check if enough time has passed since last rebalance
	if time.Since(am.rebalanceConfig.LastRebalance) < am.rebalanceConfig.RebalanceInterval {
		return false, nil
	}

	reasons := make([]string, 0)
	needsRebalance := false

	// Check for allocation drift
	for botID, allocation := range am.allocations {
		currentPercent := allocation.AllocatedBalance / am.totalBalance
		targetPercent := allocation.AllocationPercentage
		
		deviation := abs(currentPercent - targetPercent)
		if deviation > am.rebalanceConfig.RebalanceThreshold {
			needsRebalance = true
			reasons = append(reasons, 
				fmt.Sprintf("Bot %s drift: %.1f%% vs target %.1f%% (deviation: %.1f%%)", 
					botID, currentPercent*100, targetPercent*100, deviation*100))
		}
	}

	return needsRebalance, reasons
}

// ExecuteRebalance performs portfolio rebalancing
func (am *AllocationManager) ExecuteRebalance() error {
	am.mu.Lock()
	defer am.mu.Unlock()

	beforeState := am.copyAllocations()
	rebalanceActions := make([]string, 0)

	// Calculate target allocations
	for botID, allocation := range am.allocations {
		targetBalance := am.totalBalance * allocation.AllocationPercentage
		currentBalance := allocation.AllocatedBalance
		
		difference := targetBalance - currentBalance
		
		if abs(difference) > am.rebalanceConfig.MinRebalanceAmount {
			// Adjust allocation
			allocation.AllocatedBalance = targetBalance
			allocation.AvailableBalance = targetBalance - allocation.UsedBalance
			allocation.LastUpdated = time.Now()
			
			action := fmt.Sprintf("Bot %s: $%.2f → $%.2f (%+.2f)", 
				botID, currentBalance, targetBalance, difference)
			rebalanceActions = append(rebalanceActions, action)
		}
	}

	if len(rebalanceActions) == 0 {
		return nil // No rebalancing needed
	}

	// Update last rebalance time
	am.rebalanceConfig.LastRebalance = time.Now()

	// Record rebalance event
	event := AllocationEvent{
		Timestamp:   time.Now(),
		EventType:   "REBALANCE",
		BotID:       "SYSTEM",
		Amount:      0,
		Description: fmt.Sprintf("Portfolio rebalanced: %d adjustments", len(rebalanceActions)),
		BeforeState: beforeState,
		AfterState:  am.copyAllocations(),
	}
	am.allocationHistory = append(am.allocationHistory, event)

	return nil
}

// GetAllocation returns allocation for a specific bot
func (am *AllocationManager) GetAllocation(botID string) (*BotAllocation, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	allocation, exists := am.allocations[botID]
	if !exists {
		return nil, &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s has no allocation", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}

	// Return a copy to prevent external modification
	allocationCopy := *allocation
	return &allocationCopy, nil
}

// GetAllAllocations returns all current allocations
func (am *AllocationManager) GetAllAllocations() map[string]*BotAllocation {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	return am.copyAllocations()
}

// GetPortfolioSummary returns portfolio summary statistics
func (am *AllocationManager) GetPortfolioSummary() *PortfolioSummary {
	am.mu.RLock()
	defer am.mu.RUnlock()

	summary := &PortfolioSummary{
		TotalBalance:      am.totalBalance,
		TotalAllocated:    am.getTotalAllocatedBalance(),
		TotalUsed:         am.getTotalUsedBalance(),
		TotalAvailable:    am.getTotalAvailableBalance(),
		TotalPosition:     am.getTotalPositionValue(),
		TotalRealizedPnL:  am.getTotalRealizedPnL(),
		TotalUnrealizedPnL: am.getTotalUnrealizedPnL(),
		ActiveBots:        len(am.allocations),
		LastRebalance:     am.rebalanceConfig.LastRebalance,
		AllocationHistory: len(am.allocationHistory),
	}

	summary.UtilizationPercent = (summary.TotalUsed / summary.TotalBalance) * 100
	summary.AllocationPercent = (summary.TotalAllocated / summary.TotalBalance) * 100
	summary.TotalPnLPercent = ((summary.TotalRealizedPnL + summary.TotalUnrealizedPnL) / summary.TotalBalance) * 100

	return summary
}

// GetAllocationHistory returns allocation history
func (am *AllocationManager) GetAllocationHistory(limit int) []AllocationEvent {
	am.mu.RLock()
	defer am.mu.RUnlock()

	historyLen := len(am.allocationHistory)
	if limit <= 0 || limit > historyLen {
		limit = historyLen
	}

	// Return most recent events
	startIndex := historyLen - limit
	if startIndex < 0 {
		startIndex = 0
	}

	result := make([]AllocationEvent, limit)
	copy(result, am.allocationHistory[startIndex:])
	return result
}

// PortfolioSummary provides summary statistics for the portfolio
type PortfolioSummary struct {
	TotalBalance        float64   `json:"total_balance"`
	TotalAllocated      float64   `json:"total_allocated"`
	TotalUsed           float64   `json:"total_used"`
	TotalAvailable      float64   `json:"total_available"`
	TotalPosition       float64   `json:"total_position"`
	TotalRealizedPnL    float64   `json:"total_realized_pnl"`
	TotalUnrealizedPnL  float64   `json:"total_unrealized_pnl"`
	UtilizationPercent  float64   `json:"utilization_percent"`
	AllocationPercent   float64   `json:"allocation_percent"`
	TotalPnLPercent     float64   `json:"total_pnl_percent"`
	ActiveBots          int       `json:"active_bots"`
	LastRebalance       time.Time `json:"last_rebalance"`
	AllocationHistory   int       `json:"allocation_history_count"`
}

// Private helper methods

func (am *AllocationManager) getTotalAllocatedBalance() float64 {
	total := 0.0
	for _, allocation := range am.allocations {
		total += allocation.AllocatedBalance
	}
	return total
}

func (am *AllocationManager) getTotalUsedBalance() float64 {
	total := 0.0
	for _, allocation := range am.allocations {
		total += allocation.UsedBalance
	}
	return total
}

func (am *AllocationManager) getTotalAvailableBalance() float64 {
	total := 0.0
	for _, allocation := range am.allocations {
		total += allocation.AvailableBalance
	}
	return total
}

func (am *AllocationManager) getTotalPositionValue() float64 {
	total := 0.0
	for _, allocation := range am.allocations {
		total += allocation.CurrentPosition
	}
	return total
}

func (am *AllocationManager) getTotalRealizedPnL() float64 {
	total := 0.0
	for _, allocation := range am.allocations {
		total += allocation.RealizedPnL
	}
	return total
}

func (am *AllocationManager) getTotalUnrealizedPnL() float64 {
	total := 0.0
	for _, allocation := range am.allocations {
		total += allocation.UnrealizedPnL
	}
	return total
}

func (am *AllocationManager) copyAllocations() map[string]*BotAllocation {
	copy := make(map[string]*BotAllocation)
	for botID, allocation := range am.allocations {
		allocationCopy := *allocation
		copy[botID] = &allocationCopy
	}
	return copy
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
