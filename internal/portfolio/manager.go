package portfolio

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// DefaultPortfolioManager implements the PortfolioManager interface
type DefaultPortfolioManager struct {
	mu               sync.RWMutex
	config           *PortfolioConfig
	state            *PortfolioState
	stateManager     StateManager
	leverageCalc     LeverageCalculator
	riskManager      RiskManager
	bots             map[string]*BotAllocation
	isInitialized    bool
	lastHealthCheck  time.Time
	shutdownChan     chan struct{}
}

// NewPortfolioManager creates a new portfolio manager instance
func NewPortfolioManager(config *PortfolioConfig, stateManager StateManager) PortfolioManager {
	if config == nil {
		config = getDefaultPortfolioConfig()
	}
	
	return &DefaultPortfolioManager{
		config:       config,
		stateManager: stateManager,
		leverageCalc: NewLeverageCalculator(),
		bots:         make(map[string]*BotAllocation),
		shutdownChan: make(chan struct{}),
		state: &PortfolioState{
			TotalBalance:   config.TotalBalance,
			TotalProfit:    0.0,
			LastUpdated:    time.Now(),
			Allocations:    make(map[string]*BotAllocation),
			GlobalSettings: config,
			Version:        "1.0",
		},
	}
}

// Initialize initializes the portfolio manager and loads existing state
func (p *DefaultPortfolioManager) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if p.isInitialized {
		return nil
	}
	
	// Try to load existing state
	if err := p.loadStateInternal(); err != nil {
		// If no existing state found, start with fresh state
		fmt.Printf("⚠️ No existing portfolio state found, starting fresh: %v\n", err)
	}
	
	// Validate configuration
	if err := p.validateConfig(); err != nil {
		return fmt.Errorf("invalid portfolio configuration: %w", err)
	}
	
	// Initialize risk manager if not set
	if p.riskManager == nil {
		p.riskManager = NewBasicRiskManager(p.config)
	}
	
	p.isInitialized = true
	p.lastHealthCheck = time.Now()
	
	fmt.Printf("✅ Portfolio Manager initialized - Total Balance: $%.2f\n", p.state.TotalBalance)
	return nil
}

// RegisterBot registers a new bot with the portfolio manager
func (p *DefaultPortfolioManager) RegisterBot(botID string, config BotConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.isInitialized {
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   "Portfolio manager not initialized",
			Timestamp: time.Now(),
		}
	}
	
	// Check if bot already registered
	if _, exists := p.bots[botID]; exists {
		return &PortfolioError{
			Code:      ErrBotAlreadyRegistered,
			Message:   fmt.Sprintf("Bot %s is already registered", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	// Validate bot configuration
	if err := p.validateBotConfig(config); err != nil {
		return err
	}
	
	// Calculate allocated balance based on percentage
	allocatedBalance := p.state.TotalBalance * config.AllocationPercentage
	
	// Create bot allocation
	allocation := &BotAllocation{
		BotID:                botID,
		Symbol:               config.Symbol,
		AllocatedBalance:     allocatedBalance,
		UsedBalance:          0.0,
		AvailableBalance:     allocatedBalance,
		CurrentPosition:      0.0,
		AveragePrice:         0.0,
		Leverage:             config.Leverage,
		UnrealizedPnL:        0.0,
		RealizedPnL:          0.0,
		LastUpdated:          time.Now(),
		PositionMarginUsed:   0.0,
		AllocationPercentage: config.AllocationPercentage,
	}
	
	// Add to tracking
	p.bots[botID] = allocation
	p.state.Allocations[botID] = allocation
	p.state.LastUpdated = time.Now()
	
	// Save state
	if err := p.saveStateInternal(); err != nil {
		// Rollback registration
		delete(p.bots, botID)
		delete(p.state.Allocations, botID)
		return fmt.Errorf("failed to save state after bot registration: %w", err)
	}
	
	fmt.Printf("✅ Bot registered: %s (%s) - Allocated: $%.2f (%.1f%%)\n", 
		botID, config.Symbol, allocatedBalance, config.AllocationPercentage*100)
	
	return nil
}

// UnregisterBot removes a bot from the portfolio manager
func (p *DefaultPortfolioManager) UnregisterBot(botID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	allocation, exists := p.bots[botID]
	if !exists {
		return &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s is not registered", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	// Check if bot has open positions
	if allocation.CurrentPosition > 0 {
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   fmt.Sprintf("Cannot unregister bot %s with open position: $%.2f", botID, allocation.CurrentPosition),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	// Remove from tracking
	delete(p.bots, botID)
	delete(p.state.Allocations, botID)
	p.state.LastUpdated = time.Now()
	
	// Save state
	if err := p.saveStateInternal(); err != nil {
		return fmt.Errorf("failed to save state after bot unregistration: %w", err)
	}
	
	fmt.Printf("✅ Bot unregistered: %s\n", botID)
	return nil
}

// GetAvailableBalance returns the available balance for a specific bot
func (p *DefaultPortfolioManager) GetAvailableBalance(botID string) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	allocation, exists := p.bots[botID]
	if !exists {
		return 0, &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s is not registered", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	return allocation.AvailableBalance, nil
}

// GetRequiredMargin calculates the margin required for a position
func (p *DefaultPortfolioManager) GetRequiredMargin(botID string, positionValue float64) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	allocation, exists := p.bots[botID]
	if !exists {
		return 0, &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s is not registered", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	return p.leverageCalc.CalculateRequiredMargin(positionValue, allocation.Leverage), nil
}

// UpdateBotBalance updates the balance for a specific bot
func (p *DefaultPortfolioManager) UpdateBotBalance(botID string, newBalance float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	allocation, exists := p.bots[botID]
	if !exists {
		return &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s is not registered", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	// Update balance
	oldBalance := allocation.AvailableBalance
	allocation.AvailableBalance = newBalance
	allocation.LastUpdated = time.Now()
	p.state.LastUpdated = time.Now()
	
	// Log the change
	fmt.Printf("📊 Balance updated for %s: $%.2f → $%.2f\n", botID, oldBalance, newBalance)
	
	return p.saveStateInternal()
}

// UpdatePosition updates position information for a bot
func (p *DefaultPortfolioManager) UpdatePosition(botID string, positionValue, avgPrice, leverage float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	allocation, exists := p.bots[botID]
	if !exists {
		return &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s is not registered", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	// Calculate margin used for this position
	marginUsed := p.leverageCalc.CalculateRequiredMargin(positionValue, leverage)
	
	// Update position information
	allocation.CurrentPosition = positionValue
	allocation.AveragePrice = avgPrice
	allocation.Leverage = leverage
	allocation.PositionMarginUsed = marginUsed
	allocation.UsedBalance = marginUsed
	allocation.AvailableBalance = allocation.AllocatedBalance - marginUsed
	allocation.LastUpdated = time.Now()
	p.state.LastUpdated = time.Now()
	
	fmt.Printf("📊 Position updated for %s: $%.2f @ $%.4f (%.1fx leverage, $%.2f margin)\n", 
		botID, positionValue, avgPrice, leverage, marginUsed)
	
	return p.saveStateInternal()
}

// GetTotalPortfolioValue returns the total value of the portfolio
func (p *DefaultPortfolioManager) GetTotalPortfolioValue() (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	totalValue := p.state.TotalBalance + p.state.TotalProfit
	
	// Add unrealized PnL from all bots
	for _, allocation := range p.bots {
		totalValue += allocation.UnrealizedPnL
	}
	
	return totalValue, nil
}

// GetBotAllocation returns the allocation details for a specific bot
func (p *DefaultPortfolioManager) GetBotAllocation(botID string) (*BotAllocation, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	allocation, exists := p.bots[botID]
	if !exists {
		return nil, &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s is not registered", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	// Return a copy to prevent external modification
	allocationCopy := *allocation
	return &allocationCopy, nil
}

// RecordProfit records profit for a specific bot
func (p *DefaultPortfolioManager) RecordProfit(botID string, profit float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	allocation, exists := p.bots[botID]
	if !exists {
		return &PortfolioError{
			Code:      ErrBotNotRegistered,
			Message:   fmt.Sprintf("Bot %s is not registered", botID),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	// Update realized profit
	allocation.RealizedPnL += profit
	allocation.LastUpdated = time.Now()
	
	// Update portfolio total profit
	p.state.TotalProfit += profit
	p.state.LastUpdated = time.Now()
	
	fmt.Printf("💰 Profit recorded for %s: $%.2f (Total: $%.2f)\n", 
		botID, profit, allocation.RealizedPnL)
	
	return p.saveStateInternal()
}

// GetTotalProfit returns the total profit across all bots
func (p *DefaultPortfolioManager) GetTotalProfit() (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	return p.state.TotalProfit, nil
}

// SaveState saves the current portfolio state
func (p *DefaultPortfolioManager) SaveState() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	return p.saveStateInternal()
}

// LoadState loads the portfolio state from storage
func (p *DefaultPortfolioManager) LoadState() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	return p.loadStateInternal()
}

// GetPortfolioHealth returns current portfolio health metrics
func (p *DefaultPortfolioManager) GetPortfolioHealth() (*PortfolioHealth, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	health := &PortfolioHealth{
		Status:          "healthy",
		LastHealthCheck: time.Now(),
		ActiveBots:      len(p.bots),
		Warnings:        make([]string, 0),
		Errors:          make([]string, 0),
	}
	
	// Calculate total values
	totalValue, _ := p.GetTotalPortfolioValue()
	health.TotalValue = totalValue
	health.TotalPnL = p.state.TotalProfit
	
	if p.state.TotalBalance > 0 {
		health.PnLPercent = (p.state.TotalProfit / p.state.TotalBalance) * 100
	}
	
	// Calculate total exposure
	totalExposure := 0.0
	for _, allocation := range p.bots {
		totalExposure += allocation.CurrentPosition
	}
	health.TotalExposure = totalExposure
	
	if p.state.TotalBalance > 0 {
		health.ExposurePercent = (totalExposure / p.state.TotalBalance) * 100
	}
	
	// Determine health status
	p.assessHealthStatus(health)
	
	return health, nil
}

// IsHealthy returns true if the portfolio is in a healthy state
func (p *DefaultPortfolioManager) IsHealthy() bool {
	health, err := p.GetPortfolioHealth()
	if err != nil {
		return false
	}
	
	return health.Status == "healthy"
}

// Close gracefully shuts down the portfolio manager
func (p *DefaultPortfolioManager) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	if !p.isInitialized {
		return nil
	}
	
	// Save final state
	if err := p.saveStateInternal(); err != nil {
		fmt.Printf("⚠️ Failed to save final state: %v\n", err)
	}
	
	// Signal shutdown
	close(p.shutdownChan)
	
	p.isInitialized = false
	fmt.Printf("✅ Portfolio Manager closed\n")
	
	return nil
}

// Internal helper methods

func (p *DefaultPortfolioManager) saveStateInternal() error {
	if p.stateManager == nil {
		return fmt.Errorf("no state manager configured")
	}
	
	p.state.LastUpdated = time.Now()
	return p.stateManager.Save(p.state)
}

func (p *DefaultPortfolioManager) loadStateInternal() error {
	if p.stateManager == nil {
		return fmt.Errorf("no state manager configured")
	}
	
	state, err := p.stateManager.Load()
	if err != nil {
		return err
	}
	
	p.state = state
	p.bots = state.Allocations
	
	return nil
}

func (p *DefaultPortfolioManager) validateConfig() error {
	if p.config.TotalBalance <= 0 {
		return fmt.Errorf("total balance must be greater than 0")
	}
	
	if p.config.MaxTotalExposure <= 0 || p.config.MaxTotalExposure > 10.0 {
		p.config.MaxTotalExposure = 3.0 // Default to 3x max exposure
	}
	
	return nil
}

func (p *DefaultPortfolioManager) validateBotConfig(config BotConfig) error {
	if config.Symbol == "" {
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   "Bot symbol cannot be empty",
			Timestamp: time.Now(),
		}
	}
	
	if config.AllocationPercentage <= 0 || config.AllocationPercentage > 1.0 {
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   fmt.Sprintf("Invalid allocation percentage: %.2f (must be 0.0 < x <= 1.0)", config.AllocationPercentage),
			Timestamp: time.Now(),
		}
	}
	
	if err := p.leverageCalc.ValidateLeverage(config.Leverage); err != nil {
		return &PortfolioError{
			Code:      ErrInvalidLeverage,
			Message:   err.Error(),
			Timestamp: time.Now(),
		}
	}
	
	return nil
}

func (p *DefaultPortfolioManager) assessHealthStatus(health *PortfolioHealth) {
	// Check exposure levels
	if health.ExposurePercent > p.config.MaxTotalExposure*100 {
		health.Status = "critical"
		health.Errors = append(health.Errors, 
			fmt.Sprintf("Total exposure %.1f%% exceeds limit %.1f%%", 
				health.ExposurePercent, p.config.MaxTotalExposure*100))
	} else if health.ExposurePercent > p.config.MaxTotalExposure*80 {
		health.Status = "warning"
		health.Warnings = append(health.Warnings, 
			fmt.Sprintf("High exposure: %.1f%%", health.ExposurePercent))
	}
	
	// Check drawdown
	if p.config.MaxDrawdownPercent > 0 && health.PnLPercent < -p.config.MaxDrawdownPercent {
		health.Status = "critical"
		health.Errors = append(health.Errors, 
			fmt.Sprintf("Drawdown %.1f%% exceeds limit %.1f%%", 
				-health.PnLPercent, p.config.MaxDrawdownPercent))
	}
	
	// Check individual bot risks
	for botID, allocation := range p.bots {
		if allocation.PositionMarginUsed > allocation.AllocatedBalance*0.9 {
			health.Warnings = append(health.Warnings, 
				fmt.Sprintf("Bot %s using high margin: %.1f%%", 
					botID, (allocation.PositionMarginUsed/allocation.AllocatedBalance)*100))
		}
	}
}

func getDefaultPortfolioConfig() *PortfolioConfig {
	return &PortfolioConfig{
		TotalBalance:         1000.0,
		AllocationStrategy:   "equal_weight",
		SharedStateFile:      "portfolio_state.json",
		MaxTotalExposure:     3.0,  // 3x max exposure
		MaxDrawdownPercent:   25.0, // 25% max drawdown
		RebalanceFrequency:   "1h",
		RiskLimitPerBot:      0.2,  // 20% max risk per bot
		EmergencyStopEnabled: true,
	}
}
