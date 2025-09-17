package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/bot"
	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
)

// BotInstance represents a running bot instance
type BotInstance struct {
	ID         string
	Config     *config.LiveBotConfig
	Bot        *bot.LiveBot
	Status     BotStatus
	StartTime  time.Time
	LastError  error
	ErrorCount int
	mu         sync.RWMutex
}

// BotStatus represents the current status of a bot
type BotStatus string

const (
	StatusStarting BotStatus = "starting"
	StatusRunning  BotStatus = "running"
	StatusStopped  BotStatus = "stopped"
	StatusError    BotStatus = "error"
	StatusShutdown BotStatus = "shutdown"
)

// BotManager manages multiple bot instances
type BotManager struct {
	portfolioConfig *PortfolioConfig
	instances       map[string]*BotInstance
	ctx             context.Context
	cancel          context.CancelFunc
	wg              sync.WaitGroup
	mu              sync.RWMutex
	shutdownChan    chan struct{}
	errorChan       chan BotError
}

// BotError represents an error from a bot instance
type BotError struct {
	BotID string
	Error error
	Time  time.Time
}

// NewBotManager creates a new bot manager
func NewBotManager(portfolioConfig *PortfolioConfig) *BotManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &BotManager{
		portfolioConfig: portfolioConfig,
		instances:       make(map[string]*BotInstance),
		ctx:             ctx,
		cancel:          cancel,
		shutdownChan:    make(chan struct{}),
		errorChan:       make(chan BotError, 100), // Buffered channel for errors
	}
}

// StartAllBots starts all enabled bots in the portfolio
func (bm *BotManager) StartAllBots() error {
	enabledBots := bm.portfolioConfig.GetEnabledBots()
	
	fmt.Printf("🚀 Starting portfolio with %d bots...\n", len(enabledBots))
	
	// Start error handler
	go bm.handleErrors()
	
	for _, botConfig := range enabledBots {
		if err := bm.StartBot(botConfig); err != nil {
			// Continue starting other bots even if one fails
			fmt.Printf("❌ Failed to start bot %s: %v\n", botConfig.BotID, err)
			bm.reportError(botConfig.BotID, err)
		}
	}
	
	// Wait a moment for all bots to initialize
	time.Sleep(2 * time.Second)
	
	// Check if any bots are running
	runningCount := bm.GetRunningBotCount()
	if runningCount == 0 {
		return fmt.Errorf("no bots started successfully")
	}
	
	fmt.Printf("✅ Portfolio started successfully with %d/%d bots running\n", 
		runningCount, len(enabledBots))
	
	return nil
}

// StartBot starts a single bot instance
func (bm *BotManager) StartBot(botConfig BotConfig) error {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	// Check if bot is already running
	if instance, exists := bm.instances[botConfig.BotID]; exists {
		if instance.GetStatus() == StatusRunning {
			return fmt.Errorf("bot %s is already running", botConfig.BotID)
		}
	}
	
	// Load bot configuration
	liveBotConfig, err := config.LoadLiveBotConfig(botConfig.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config for bot %s: %w", botConfig.BotID, err)
	}
	
	// Ensure API credentials are properly loaded from environment
	if err := ensureAPICredentials(liveBotConfig); err != nil {
		return fmt.Errorf("failed to ensure API credentials for bot %s: %w", botConfig.BotID, err)
	}
	
	// Create bot instance
	liveBot, err := bot.NewLiveBot(liveBotConfig)
	if err != nil {
		return fmt.Errorf("failed to create bot %s: %w", botConfig.BotID, err)
	}
	
	// Create instance tracker
	instance := &BotInstance{
		ID:        botConfig.BotID,
		Config:    liveBotConfig,
		Bot:       liveBot,
		Status:    StatusStarting,
		StartTime: time.Now(),
	}
	
	bm.instances[botConfig.BotID] = instance
	
	// Start bot in goroutine
	bm.wg.Add(1)
	go bm.runBot(instance)
	
	fmt.Printf("🤖 Bot %s started\n", botConfig.BotID)
	return nil
}

// runBot runs a bot instance in a goroutine
func (bm *BotManager) runBot(instance *BotInstance) {
	defer bm.wg.Done()
	
	// Set status to running
	instance.SetStatus(StatusRunning)
	
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("bot %s panicked: %v", instance.ID, r)
			bm.reportError(instance.ID, err)
			instance.SetStatus(StatusError)
			instance.SetError(err)
		}
	}()
	
	// Start the bot
	if err := instance.Bot.Start(); err != nil {
		bm.reportError(instance.ID, err)
		instance.SetStatus(StatusError)
		instance.SetError(err)
		return
	}
	
	// Wait for shutdown signal or context cancellation
	select {
	case <-bm.ctx.Done():
		fmt.Printf("🛑 Bot %s received shutdown signal\n", instance.ID)
	case <-bm.shutdownChan:
		fmt.Printf("🛑 Bot %s shutting down\n", instance.ID)
	}
	
	// Graceful shutdown
	instance.Bot.Stop()
	instance.SetStatus(StatusShutdown)
	fmt.Printf("✅ Bot %s stopped gracefully\n", instance.ID)
}

// StopAllBots stops all running bots gracefully
func (bm *BotManager) StopAllBots() {
	fmt.Printf("🛑 Stopping all bots...\n")
	
	// Signal all bots to stop
	close(bm.shutdownChan)
	bm.cancel()
	
	// Wait for all bots to stop with timeout
	done := make(chan struct{})
	go func() {
		bm.wg.Wait()
		close(done)
	}()
	
	select {
	case <-done:
		fmt.Printf("✅ All bots stopped gracefully\n")
	case <-time.After(30 * time.Second):
		fmt.Printf("⚠️ Timeout waiting for bots to stop\n")
	}
}

// StopBot stops a specific bot
func (bm *BotManager) StopBot(botID string) error {
	bm.mu.RLock()
	instance, exists := bm.instances[botID]
	bm.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("bot %s not found", botID)
	}
	
	if instance.GetStatus() != StatusRunning {
		return fmt.Errorf("bot %s is not running", botID)
	}
	
	// Stop the bot
	instance.Bot.Stop()
	instance.SetStatus(StatusStopped)
	
	fmt.Printf("✅ Bot %s stopped\n", botID)
	return nil
}

// GetBotStatus returns the status of a specific bot
func (bm *BotManager) GetBotStatus(botID string) (BotStatus, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	instance, exists := bm.instances[botID]
	if !exists {
		return "", fmt.Errorf("bot %s not found", botID)
	}
	
	return instance.GetStatus(), nil
}

// GetAllBotStatuses returns the status of all bots
func (bm *BotManager) GetAllBotStatuses() map[string]BotStatus {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	statuses := make(map[string]BotStatus)
	for id, instance := range bm.instances {
		statuses[id] = instance.GetStatus()
	}
	
	return statuses
}

// GetRunningBotCount returns the number of currently running bots
func (bm *BotManager) GetRunningBotCount() int {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	count := 0
	for _, instance := range bm.instances {
		if instance.GetStatus() == StatusRunning {
			count++
		}
	}
	
	return count
}

// GetBotInstance returns a bot instance by ID
func (bm *BotManager) GetBotInstance(botID string) (*BotInstance, error) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	instance, exists := bm.instances[botID]
	if !exists {
		return nil, fmt.Errorf("bot %s not found", botID)
	}
	
	return instance, nil
}

// reportError reports an error from a bot
func (bm *BotManager) reportError(botID string, err error) {
	select {
	case bm.errorChan <- BotError{
		BotID: botID,
		Error: err,
		Time:  time.Now(),
	}:
	default:
		// Channel is full, log directly
		fmt.Printf("❌ Bot %s error (channel full): %v\n", botID, err)
	}
}

// handleErrors handles errors from bots
func (bm *BotManager) handleErrors() {
	for {
		select {
		case botError := <-bm.errorChan:
			fmt.Printf("❌ Bot %s error at %s: %v\n", 
				botError.BotID, 
				botError.Time.Format("15:04:05"), 
				botError.Error)
			
			// Update bot instance error info
			if instance, err := bm.GetBotInstance(botError.BotID); err == nil {
				instance.SetError(botError.Error)
				instance.IncrementErrorCount()
			}
			
		case <-bm.ctx.Done():
			return
		}
	}
}

// BotInstance methods

// GetStatus safely gets the bot status
func (bi *BotInstance) GetStatus() BotStatus {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	return bi.Status
}

// SetStatus safely sets the bot status
func (bi *BotInstance) SetStatus(status BotStatus) {
	bi.mu.Lock()
	defer bi.mu.Unlock()
	bi.Status = status
}

// SetError safely sets the last error
func (bi *BotInstance) SetError(err error) {
	bi.mu.Lock()
	defer bi.mu.Unlock()
	bi.LastError = err
}

// GetError safely gets the last error
func (bi *BotInstance) GetError() error {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	return bi.LastError
}

// IncrementErrorCount safely increments the error count
func (bi *BotInstance) IncrementErrorCount() {
	bi.mu.Lock()
	defer bi.mu.Unlock()
	bi.ErrorCount++
}

// GetErrorCount safely gets the error count
func (bi *BotInstance) GetErrorCount() int {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	return bi.ErrorCount
}

// GetUptime returns how long the bot has been running
func (bi *BotInstance) GetUptime() time.Duration {
	bi.mu.RLock()
	defer bi.mu.RUnlock()
	return time.Since(bi.StartTime)
}
