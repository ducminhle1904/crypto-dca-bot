package bot

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"sync"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/adapters"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// TPOrderInfo holds information about a take profit order
type TPOrderInfo struct {
	Level       int     `json:"level"`        // TP level (1-5)
	Percent     float64 `json:"percent"`      // TP percentage (e.g., 0.004 for 0.4%)
	Quantity    string  `json:"quantity"`     // Order quantity
	Price       string  `json:"price"`        // TP price
	OrderID     string  `json:"order_id"`     // Exchange order ID
	Filled      bool    `json:"filled"`       // Whether this level was filled
	FilledQty   string  `json:"filled_qty"`   // Actual filled quantity
}

// LiveBot represents the live trading bot with exchange interface support
type LiveBot struct {
	config   *config.LiveBotConfig
	exchange exchange.LiveTradingExchange
	strategy *strategy.EnhancedDCAStrategy
	logger   *logger.Logger
	
	// Trading parameters extracted from config
	symbol   string
	interval string
	category string
	
	// Bot control
	running  bool
	stopChan chan struct{}
	
	// Trading state - exchange agnostic
	currentPosition float64
	averagePrice    float64
	totalInvested   float64
	balance         float64
	dcaLevel        int
	
	// TP order management - multi-level support
	activeTPOrders map[string]*TPOrderInfo // orderID -> TP order info mapping
	filledTPOrders map[string]*TPOrderInfo // orderID -> filled TP order info mapping
	tpOrderMutex   sync.RWMutex            // Protect TP order map access
	
	// Position synchronization
	positionMutex  sync.RWMutex      // Protect position data access
	
	// Debugging counters
	holdLogCounter int               // Counter for HOLD decision logging
}

// NewLiveBot creates a new live trading bot instance
func NewLiveBot(config *config.LiveBotConfig) (*LiveBot, error) {
	if config == nil {
		return nil, fmt.Errorf("bot configuration is required")
	}

	// Create exchange instance using factory
	factory := adapters.NewFactory()
	exchangeInstance, err := factory.CreateExchange(config.Exchange)
	if err != nil {
		return nil, fmt.Errorf("failed to create exchange: %w", err)
	}

	// Extract trading parameters
	symbol := config.Strategy.Symbol
	interval := config.Strategy.Interval
	category := config.Strategy.Category



	// Initialize file logger
	fileLogger, err := logger.NewLogger(symbol, interval)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	bot := &LiveBot{
		config:   config,
		exchange: exchangeInstance,
		strategy: nil, // Will be initialized below
		logger:   fileLogger,
		symbol:   symbol,
		interval: interval,
		category: category,
		balance:  config.Risk.InitialBalance,
		stopChan: make(chan struct{}),
		activeTPOrders: make(map[string]*TPOrderInfo),
		filledTPOrders: make(map[string]*TPOrderInfo),
		tpOrderMutex: sync.RWMutex{},
	}

	// Initialize strategy
	if err := bot.initializeStrategy(); err != nil {
		fileLogger.Close()
		return nil, fmt.Errorf("failed to initialize strategy: %w", err)
	}

	return bot, nil
}

// Start initializes and starts the bot
func (bot *LiveBot) Start() error {
	bot.running = true

	// Connect to exchange
	ctx := context.Background()
	if err := bot.exchange.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to exchange: %w", err)
	}

	// Sync with real account balance if possible
	if err := bot.syncAccountBalance(); err != nil {
		bot.logger.LogWarning("Could not sync account balance", "Using config balance: $%.2f. Check API key permissions", bot.balance)
		// Still show critical info on console
		fmt.Printf("⚠️ Using config balance: $%.2f (see log for details)\n", bot.balance)
	}

	// Check for existing position and sync bot state
	if err := bot.syncExistingPosition(); err != nil {
		bot.logger.LogWarning("Could not sync existing position", "%v", err)
	}
	
	// Sync existing orders on startup
	if err := bot.syncExistingOrders(); err != nil {
		bot.logger.LogWarning("Could not sync existing orders", "%v", err)
	}

	// Print startup information
	bot.printStartupInfo()
	bot.printBotConfiguration()

	// Show log file location
	fmt.Printf("📝 Trading logs: %s\n", bot.logger.GetLogPath())
	fmt.Printf("🔄 Bot is running... (trading activity logged to file)\n\n")

	// Start the main trading loop
	go bot.tradingLoop()

	return nil
}

// Stop gracefully stops the bot
func (bot *LiveBot) Stop() {
	if !bot.running {
		return // Already stopped
	}
	
	bot.running = false
	fmt.Printf("🛑 Stopping bot...\n")
	
	// Signal trading loop to stop FIRST - this prevents new trading operations
	// Use defer-recover to prevent panic if channel is already closed
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Channel was already closed, which is fine
			}
		}()
		close(bot.stopChan)
	}()
	
	// Give trading loop a moment to exit gracefully
	fmt.Printf("⏱️ Waiting for trading loop to stop...\n")
	time.Sleep(2 * time.Second)
	
	// Use timeout for cleanup operations to prevent hanging
	cleanupTimeout := 15 * time.Second
	cleanupDone := make(chan struct{})
	
	go func() {
		defer close(cleanupDone)
		
		// Cancel all active TP orders before closing positions
		fmt.Printf("🧹 Cleaning up TP orders...\n")
		if err := bot.cancelAllTPOrders(); err != nil {
			fmt.Printf("⚠️ Error canceling TP orders: %v\n", err)
			if bot.logger != nil {
				bot.logger.Error("Error canceling TP orders during shutdown: %v", err)
			}
		}
		
		// Close any open positions before stopping
		fmt.Printf("🔄 Closing open positions...\n")
		if err := bot.closeOpenPositions(); err != nil {
			fmt.Printf("⚠️ Error closing positions: %v\n", err)
			if bot.logger != nil {
				bot.logger.Error("Error closing positions during shutdown: %v", err)
			}
		}
		
		// Disconnect from exchange
		fmt.Printf("🔌 Disconnecting from exchange...\n")
		if err := bot.exchange.Disconnect(); err != nil {
			fmt.Printf("⚠️ Error disconnecting: %v\n", err)
			if bot.logger != nil {
				bot.logger.Error("Error disconnecting from exchange: %v", err)
			}
		}
		
		// Close logger
		if bot.logger != nil {
			bot.logger.Close()
		}
	}()
	
	// Wait for cleanup or timeout
	select {
	case <-cleanupDone:
		fmt.Printf("✅ Cleanup completed successfully\n")
	case <-time.After(cleanupTimeout):
		fmt.Printf("⚠️ Cleanup timed out after %v - forcing exit\n", cleanupTimeout)
	}
}

// initializeStrategy sets up the trading strategy with indicators
func (bot *LiveBot) initializeStrategy() error {
	// Create strategy
	bot.strategy = strategy.NewEnhancedDCAStrategy(bot.config.Strategy.BaseAmount)
	bot.strategy.SetPriceThreshold(bot.config.Strategy.PriceThreshold)

	// Add indicators based on configuration
	for _, indName := range bot.config.Strategy.Indicators {
		if bot.config.Strategy.UseAdvancedCombo {
			// Advanced combo indicators
			switch strings.ToLower(indName) {
			case "hull_ma", "hullma":
				hullMA := indicators.NewHullMA(bot.config.Strategy.HullMA.Period)
				bot.strategy.AddIndicator(hullMA)
			case "mfi":
				mfi := indicators.NewMFIWithPeriod(bot.config.Strategy.MFI.Period)
				mfi.SetOversold(bot.config.Strategy.MFI.Oversold)
				mfi.SetOverbought(bot.config.Strategy.MFI.Overbought)
				bot.strategy.AddIndicator(mfi)
			case "keltner_channels", "kc":
				keltner := indicators.NewKeltnerChannelsCustom(
					bot.config.Strategy.Keltner.Period,
					bot.config.Strategy.Keltner.Multiplier,
				)
				bot.strategy.AddIndicator(keltner)
			case "wavetrend", "wt":
				wavetrend := indicators.NewWaveTrendCustom(
					bot.config.Strategy.WaveTrend.N1,
					bot.config.Strategy.WaveTrend.N2,
				)
				wavetrend.SetOverbought(bot.config.Strategy.WaveTrend.Overbought)
				wavetrend.SetOversold(bot.config.Strategy.WaveTrend.Oversold)
				bot.strategy.AddIndicator(wavetrend)
			}
		} else {
			// Classic combo indicators
			switch strings.ToLower(indName) {
			case "rsi":
				rsi := indicators.NewRSI(bot.config.Strategy.RSI.Period)
				rsi.SetOversold(bot.config.Strategy.RSI.Oversold)
				rsi.SetOverbought(bot.config.Strategy.RSI.Overbought)
				bot.strategy.AddIndicator(rsi)
			case "macd":
				macd := indicators.NewMACD(
					bot.config.Strategy.MACD.FastPeriod,
					bot.config.Strategy.MACD.SlowPeriod,
					bot.config.Strategy.MACD.SignalPeriod,
				)
				bot.strategy.AddIndicator(macd)
			case "bb", "bollinger":
				bb := indicators.NewBollingerBands(
					bot.config.Strategy.BollingerBands.Period,
					bot.config.Strategy.BollingerBands.StdDev,
				)
				bot.strategy.AddIndicator(bb)
			case "ema":
				ema := indicators.NewEMA(bot.config.Strategy.EMA.Period)
				bot.strategy.AddIndicator(ema)
			case "sma":
				sma := indicators.NewSMA(bot.config.Strategy.EMA.Period)
				bot.strategy.AddIndicator(sma)
			}
		}
	}
	
	return nil
}

// syncAccountBalance syncs bot balance with real exchange balance
func (bot *LiveBot) syncAccountBalance() error {
	ctx := context.Background()
	
	// Determine account type and currency based on exchange and category
	accountType := exchange.AccountTypeUnified
	baseCurrency := "USDT"
	
	switch bot.category {
	case "spot":
		accountType = exchange.AccountTypeUnified
		baseCurrency = "USDT"
	case "linear":
		accountType = exchange.AccountTypeUnified
		baseCurrency = "USDT"
	case "inverse":
		accountType = exchange.AccountTypeUnified
		baseCurrency = "USDT" // For simplicity, still check USDT
	}
	
	// Get real balance from exchange
	realBalance, err := bot.exchange.GetTradableBalance(ctx, accountType, baseCurrency)
	if err != nil {
		return fmt.Errorf("failed to fetch account balance: %w", err)
	}
	
	// Update bot balance
	bot.balance = realBalance
	return nil
}

// syncExistingPosition syncs bot state with any existing position
func (bot *LiveBot) syncExistingPosition() error {
	ctx := context.Background()
	
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get existing positions: %w", err)
	}
	
	// Look for our symbol position with enhanced identification
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			positionValue, err := parseFloat(pos.PositionValue)
			if err != nil {
				bot.logger.LogWarning("Position Sync", "Failed to parse position value '%s': %v", pos.PositionValue, err)
				continue
			}
			avgPrice, err := parseFloat(pos.AvgPrice)
			if err != nil {
				bot.logger.LogWarning("Position Sync", "Failed to parse average price '%s': %v", pos.AvgPrice, err)
				continue
			}
			
			if positionValue > 0 && avgPrice > 0 {
				// Protect state modifications with mutex
				bot.positionMutex.Lock()
				bot.currentPosition = positionValue
				bot.averagePrice = avgPrice
				bot.totalInvested = positionValue
				// Estimate DCA level based on position size (more accurate than assuming 1)
				if bot.config.Strategy.BaseAmount > 0 {
					bot.dcaLevel = max(1, int(positionValue/bot.config.Strategy.BaseAmount))
				} else {
					bot.dcaLevel = 1 // Fallback to level 1
				}
				bot.positionMutex.Unlock()
				
				bot.logger.LogPositionSync(positionValue, avgPrice, pos.Size, pos.UnrealisedPnl)
				// Show brief console message
				fmt.Printf("🔄 Existing position synced (see log for details)\n")
				return nil
			}
		}
	}
	
	// No existing position found - protect state reset with mutex
	bot.positionMutex.Lock()
	bot.currentPosition = 0
	bot.averagePrice = 0
	bot.totalInvested = 0
	bot.dcaLevel = 0
	bot.positionMutex.Unlock()
	fmt.Printf("✅ No existing position found - starting fresh\n")
	
	return nil
}

// tradingLoop runs the main trading logic
func (bot *LiveBot) tradingLoop() {
	// Calculate interval duration
	intervalDuration := bot.getIntervalDuration()
	bot.logger.Info("Trading interval: %s (%v)", bot.interval, intervalDuration)
	
	// Wait for next candle close - but make it interruptible
	waitDuration := bot.getTimeUntilNextCandle()
	bot.logger.Info("Waiting %.0f seconds for next %s candle close", waitDuration.Seconds(), bot.interval)
	
	// Use interruptible wait instead of blocking sleep
	waitTimer := time.NewTimer(waitDuration)
	defer waitTimer.Stop()
	
	select {
	case <-waitTimer.C:
		// Timer expired - continue to initial check
		bot.logger.Info("First candle closed - running initial check")
		bot.checkAndTrade()
	case <-bot.stopChan:
		bot.logger.Info("Stop signal received during initial wait - ending trading loop")
		return
	}

	// Create ticker for regular checks
	ticker := time.NewTicker(intervalDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check for stop signal before processing
			if bot.shouldStop() {
				bot.logger.Info("Stop signal detected - ending trading loop")
				return
			}
			bot.checkAndTrade()
		case <-bot.stopChan:
			bot.logger.Info("Stop signal received - ending trading loop")
			return
		}
	}
}

// shouldStop checks if the bot should stop (non-blocking)
func (bot *LiveBot) shouldStop() bool {
	return !bot.running
}

// checkAndTrade performs market analysis and executes trades
func (bot *LiveBot) checkAndTrade() {
	defer func() {
		if r := recover(); r != nil {
			bot.logger.Error("Error in trading loop: %v", r)
		}
	}()

	// Check for stop signal before starting
	if bot.shouldStop() {
		return
	}

	// Use context with timeout to prevent hanging on API calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Refresh account balance
	if err := bot.syncAccountBalance(); err != nil {
		bot.logger.LogWarning("Could not refresh balance", "%v", err)
		// Continue despite balance sync failure
	}

	// Check for stop signal after balance sync
	if bot.shouldStop() {
		return
	}

	// Sync position data
	if err := bot.syncPositionData(); err != nil {
		bot.logger.LogWarning("Could not sync position data", "%v", err)
		// Continue despite position sync failure
	}

	// Check for stop signal after position sync
	if bot.shouldStop() {
		return
	}

	// Get current market price
	currentPrice, err := bot.exchange.GetLatestPrice(ctx, bot.symbol)
	if err != nil {
		bot.logger.Error("Failed to get current price: %v", err)
		return
	}

	// Get recent klines for analysis
	klines, err := bot.getRecentKlines()
	if err != nil {
		bot.logger.Error("Failed to get recent klines: %v", err)
		return
	}

	// Use a more flexible minimum data requirement
	// Need at least 2x the longest indicator period for basic reliability
	minRequiredDataPoints := 70 // Minimum required for multi-indicator strategy
	if bot.config.Strategy.WindowSize < minRequiredDataPoints {
		minRequiredDataPoints = bot.config.Strategy.WindowSize
	}
	
	if len(klines) < minRequiredDataPoints {
		bot.logger.LogWarning("Insufficient data", "Not enough data points (%d < %d minimum required)", len(klines), minRequiredDataPoints)
		return
	}
	
	// If we have less than the configured window size but more than minimum, proceed with warning
	if len(klines) < bot.config.Strategy.WindowSize {
		bot.logger.LogWarning("Limited data", "Using %d data points (less than configured %d, but sufficient for analysis)", len(klines), bot.config.Strategy.WindowSize)
	}

	// Analyze market conditions
	decision, action := bot.analyzeMarket(klines, currentPrice)

	// Check for filled TP orders and gather TP information
	filledOrders := bot.detectFilledTPOrders()
	filledTPSummary := bot.getFilledTPSummary()
	
	// Count active TP orders
	bot.tpOrderMutex.RLock()
	activeTPCount := len(bot.activeTPOrders)
	bot.tpOrderMutex.RUnlock()
	
	// Notify about newly filled orders
	if len(filledOrders) > 0 {
		bot.logger.Info("🎯 TP Orders FILLED: %s", strings.Join(filledOrders, ", "))
		fmt.Printf("🎯 TP Orders Filled: %s\n", strings.Join(filledOrders, ", "))
	}
	
	// Log market status to file with TP information
	exchangePnL := bot.getExchangePnL()
	bot.logger.LogMarketStatus(currentPrice, action, bot.balance, bot.currentPosition, bot.averagePrice, bot.dcaLevel, exchangePnL, filledTPSummary, activeTPCount)

	// Execute trading action
	if action != "HOLD" {
		if bot.exchange.IsDemo() {
			bot.logger.Trade("🧪 DEMO MODE: Executing %s at $%.2f (paper trading)", action, currentPrice)
		} else {
			bot.logger.Trade("💰 LIVE MODE: Executing %s at $%.2f (real money)", action, currentPrice)
		}
		bot.executeTrade(decision, action, currentPrice)
	}
}

// getRecentKlines retrieves recent market data with timeout protection
func (bot *LiveBot) getRecentKlines() ([]types.OHLCV, error) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Start with a reasonable request size, but be prepared to accept less
	requestLimit := bot.config.Strategy.WindowSize + 50
	if requestLimit > 200 {
		requestLimit = 200 // Limit max request to avoid exchange limits
	}
	
	params := exchange.KlineParams{
		Category: bot.category,
		Symbol:   bot.symbol,
		Interval: exchange.KlineInterval(bot.interval),
		Limit:    requestLimit,
	}
	
	klines, err := bot.exchange.GetKlines(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	// If we got less data than expected, try with a smaller request
	if len(klines) < 70 && requestLimit > 100 {
		bot.logger.LogWarning("Limited klines", "Got only %d klines, retrying with smaller request", len(klines))
		params.Limit = 100
		klines, err = bot.exchange.GetKlines(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to get klines (retry): %w", err)
		}
	}

	// Return all available data (we'll handle the minimum check in checkAndTrade)
	return klines, nil
}

// getExchangePnL retrieves unrealized PnL from exchange
func (bot *LiveBot) getExchangePnL() string {
	if bot.currentPosition <= 0 {
		return ""
	}
	
	ctx := context.Background()
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return ""
	}
	
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			return pos.UnrealisedPnl
		}
	}
	
	return ""
}

// analyzeMarket performs technical analysis to determine trading action
func (bot *LiveBot) analyzeMarket(klines []types.OHLCV, currentPrice float64) (*strategy.TradeDecision, string) {
	// Use strategy to analyze market conditions
	decision, err := bot.strategy.ShouldExecuteTrade(klines)
	if err != nil {
		bot.logger.Error("Strategy error: %v", err)
		return nil, "HOLD"
	}
	
	// Log strategy decision reasoning for better debugging
	if decision.Action == strategy.ActionBuy {
		bot.logger.Info("🎯 BUY Signal: %s (Confidence: %.1f%%, Strength: %.1f%%)", 
			decision.Reason, decision.Confidence*100, decision.Strength*100)
		// Reset hold log counter so next HOLD decision is logged
		bot.holdLogCounter = 0
		return decision, "BUY"
	} else {
		// Log HOLD reasoning every 10th check to avoid spam, but always log immediately after cycle completion
		if bot.dcaLevel == 0 || bot.holdLogCounter%10 == 0 {
			bot.logger.Info("⏸️ HOLD Decision: %s", decision.Reason)
		}
		bot.holdLogCounter++
		return decision, "HOLD"
	}

	// Note: Take profit is now handled by exchange limit orders placed after each buy
	// This eliminates the need to wait for candle closes and provides immediate TP protection
	// The exchange will automatically execute TP orders when price reaches target levels
}

// executeTrade executes the trading action
func (bot *LiveBot) executeTrade(decision *strategy.TradeDecision, action string, price float64) {
	switch action {
	case "BUY":
		bot.executeBuy(decision, price)
	case "SELL":
		bot.executeSell(price)
	}
}

// executeBuy executes a buy order using exchange interface
func (bot *LiveBot) executeBuy(decision *strategy.TradeDecision, price float64) {
	ctx := context.Background()

	// Use strategy's calculated amount (confidence-based)
	amount := decision.Amount
	
	// Apply DCA level multiplier on top of strategy's calculation
	if bot.dcaLevel > 0 {
		multiplier := 1.0 + float64(bot.dcaLevel)*0.5 // Increase by 50% each level
		if multiplier > bot.config.Strategy.MaxMultiplier {
			multiplier = bot.config.Strategy.MaxMultiplier
		}
		amount *= multiplier
	}
	
	// Log advanced sizing info
	bot.logger.Info("🧠 Advanced Position Sizing: Confidence: %.1f%%, Strength: %.1f%%, Strategy: $%.2f, DCA Level: %d, Final: $%.2f", 
		decision.Confidence*100, decision.Strength*100, decision.Amount, bot.dcaLevel, amount)

	// Get trading constraints
	constraints, err := bot.exchange.GetTradingConstraints(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("Trading constraints", "Could not get trading constraints: %v", err)
		return
	}

	// Calculate quantity and apply constraints with zero-value check
	if price <= 0 {
		bot.logger.LogWarning("Order constraints", "Invalid price: %.4f", price)
		return
	}
	quantity := amount / price

	// Apply minimum quantity constraint using floor to avoid overshooting
	if constraints.QtyStep > 0 {
		multiplier := math.Floor(quantity / constraints.QtyStep)
		if multiplier < 1 {
			multiplier = 1
		}
		adjustedQuantity := multiplier * constraints.QtyStep
		
		if adjustedQuantity != quantity {
			quantity = adjustedQuantity
			amount = quantity * price
			bot.logger.Info("📏 Quantity adjusted to step size: %.6f %s (floored to avoid overshoot)", quantity, bot.symbol)
		}
	}

	// Check minimum order constraints
	orderValue := quantity * price
	if quantity < constraints.MinOrderQty {
		bot.logger.LogWarning("Order constraints", "Quantity %.6f < min %.6f %s", quantity, constraints.MinOrderQty, bot.symbol)
		return
	}
	if orderValue < constraints.MinOrderValue {
		bot.logger.LogWarning("Order constraints", "Order value $%.2f < min $%.2f", orderValue, constraints.MinOrderValue)
		return
	}

	// Check available balance for margin (don't assume 10x leverage)
	if amount > bot.balance {
		bot.logger.LogWarning("Insufficient balance", "Balance: $%.2f < Required: $%.2f", bot.balance, amount)
		return
	}

	// Place order with timeout and retry
	orderParams := exchange.OrderParams{
		Category:  bot.category,
		Symbol:    bot.symbol,
		Side:      exchange.OrderSideBuy,
		Quantity:  fmt.Sprintf("%.6f", quantity),
		OrderType: exchange.OrderTypeMarket,
	}

	order, err := bot.placeOrderWithRetry(orderParams, true) // true for market order
	if err != nil {
		bot.logger.Error("Failed to place buy order after retries: %v", err)
		return
	}

	// Sync with exchange data first to get actual executed values
	bot.logger.Info("🔄 Syncing position data after order execution...")
	bot.syncAfterTrade(order, "BUY")
	bot.logger.Info("✅ Sync complete - Position: %.6f, AvgPrice: %.4f", bot.currentPosition, bot.averagePrice)

	// Place multi-level take profit orders using actual executed values (if enabled)
	if bot.config.Strategy.AutoTPOrders {
		// Use position data from sync instead of order response (more reliable)
		avgPrice := bot.averagePrice // Updated by syncAfterTrade
		
		if bot.currentPosition > 0 && avgPrice > 0 {
			// Get current position size from exchange for accurate TP sizing
			ctx := context.Background()
			positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
			if err != nil {
				bot.logger.LogWarning("Multi-Level TP Setup", "Could not get position for TP orders: %v", err)
			} else {
				// Find our position and use its size
				for _, pos := range positions {
					if pos.Symbol == bot.symbol && pos.Side == "Buy" {
						bot.logger.Info("🎯 Setting up TP orders - Position Size: %s, Avg Price: $%.4f", pos.Size, avgPrice)
						
						// Place TP orders with proper error handling (no panic recovery needed)
						if err := bot.placeMultiLevelTPOrders(pos.Size, avgPrice); err != nil {
							bot.logger.LogWarning("Multi-Level TP Setup", "Could not place TP orders: %v", err)
						}
						break
					}
				}
			}
		} else {
			bot.logger.LogWarning("Multi-Level TP Setup", "No position found after trade execution (position: %.6f, avgPrice: %.4f)", bot.currentPosition, avgPrice)
		}
	}
}

// placeMultiLevelTPOrders places multiple take profit limit orders after a buy
func (bot *LiveBot) placeMultiLevelTPOrders(totalQuantity string, avgEntryPrice float64) error {
	// Use timeout for placing multiple orders (reduced with rate limiting delays)
	// 5 levels * (10s max per order + 0.5s delay) = ~55s + buffer = 90s
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	
	// First, cancel any existing TP orders to prevent duplicates
	bot.tpOrderMutex.Lock()
	existingOrders := make(map[string]*TPOrderInfo)
	for orderID, tpInfo := range bot.activeTPOrders {
		existingOrders[orderID] = &TPOrderInfo{
			Level:   tpInfo.Level,
			OrderID: orderID,
		}
	}
	// Clear active orders map
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	bot.tpOrderMutex.Unlock()
	
	// Cancel existing orders without holding mutex
	for orderID, tpInfo := range existingOrders {
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("TP Cleanup", "Failed to cancel existing TP Level %d order %s: %v", tpInfo.Level, orderID, err)
		}
	}
	
	// Add a channel to handle timeout gracefully
	done := make(chan error, 1)
	
	go func() {
		done <- bot.placeMultiLevelTPOrdersInternal(ctx, totalQuantity, avgEntryPrice)
	}()
	
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		bot.logger.LogWarning("TP Placement Timeout", "TP order placement timed out after 90 seconds")
		return fmt.Errorf("TP order placement timed out")
	}
}

// placeMultiLevelTPOrdersInternal is the internal implementation
func (bot *LiveBot) placeMultiLevelTPOrdersInternal(ctx context.Context, totalQuantity string, avgEntryPrice float64) error {
	startTime := time.Now()
	
	
	// Parse total quantity
	totalQty, err := parseFloat(totalQuantity)
	if err != nil {
		return fmt.Errorf("invalid quantity: %w", err)
	}
	
	// Get trading constraints for proper rounding
	constraints, err := bot.exchange.GetTradingConstraints(ctx, bot.category, bot.symbol)
	
	if err != nil {
		bot.logger.LogWarning("TP Constraints", "Could not get trading constraints: %v", err)
		// Continue with default constraints
		constraints = &exchange.TradingConstraints{QtyStep: 0.001, MinPriceStep: 0.0001}
		bot.logger.Info("🐛 DEBUG: Using default constraints")
	} else {
		bot.logger.Info("🐛 DEBUG: Constraints - MinQty: %.6f, QtyStep: %.6f, MinValue: %.2f, PriceStep: %.4f", 
			constraints.MinOrderQty, constraints.QtyStep, constraints.MinOrderValue, constraints.MinPriceStep)
	}
	
	// Pre-calculate quantities for all TP levels to ensure proper distribution
	levelQuantities := bot.calculateTPLevelQuantities(totalQty, constraints)
	
	bot.logger.Info("🎯 Placing %d-level TP orders from avg entry $%.4f: %.6f %s total", 
		bot.config.Strategy.TPLevels, avgEntryPrice, totalQty, bot.symbol)
	
	successCount := 0
	skippedLevels := 0
	
	// Place TP orders for each level using pre-calculated quantities
	for level := 1; level <= bot.config.Strategy.TPLevels; level++ {
		// Check for context cancellation and timeout safety
		select {
		case <-ctx.Done():
			bot.logger.LogWarning("TP Placement", "Context cancelled during TP level %d placement, stopping", level)
			// Return partial success if we placed some orders
			if successCount > 0 {
				bot.logger.Info("✅ Partial TP success: %d/%d levels placed before timeout", successCount, bot.config.Strategy.TPLevels)
			}
			return fmt.Errorf("context cancelled during TP placement at level %d", level)
		default:
		}
		
		// Additional safety check: if we're close to timeout, stop placing more orders  
		deadline, hasDeadline := ctx.Deadline()
		if hasDeadline && time.Until(deadline) < 20*time.Second {
			bot.logger.LogWarning("TP Placement", "Stopping at level %d - insufficient time remaining (<%ds)", level, 20)
			// Return partial success if we've placed some orders
			if successCount > 0 {
				bot.logger.Info("✅ Partial TP success due to timeout: %d/%d levels placed", successCount, bot.config.Strategy.TPLevels)
			}
			break
		}
		
		// Get pre-calculated quantity for this level with bounds checking
		if level < 1 || level > len(levelQuantities) {
			bot.logger.LogWarning("TP Level %d", "Invalid level index: %d (valid range: 1-%d)", level, level, len(levelQuantities))
			skippedLevels++
			continue
		}
		levelQty := levelQuantities[level-1] // Array is 0-indexed
		if levelQty <= 0 {
			skippedLevels++
			continue
		}
		
		// Calculate TP price for this level based on actual average entry
		// Add zero-value check to prevent division by zero
		if bot.config.Strategy.TPLevels <= 0 {
			bot.logger.LogWarning("TP Level %d", "Invalid TP levels configuration: %d", level, bot.config.Strategy.TPLevels)
			skippedLevels++
			continue
		}
		levelPercent := bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
		tpPrice := avgEntryPrice * (1 + levelPercent)
		
		// Round price to exchange tick size
		if constraints.MinPriceStep > 0 {
			tpPrice = math.Round(tpPrice/constraints.MinPriceStep) * constraints.MinPriceStep
		}
		
		// Validate order constraints (should already be satisfied by pre-calculation)
		orderValue := levelQty * tpPrice
		if levelQty < constraints.MinOrderQty || orderValue < constraints.MinOrderValue {
			bot.logger.LogWarning("TP Level %d", "Constraint violation: qty=%.6f, value=$%.2f, skipping", level, levelQty, orderValue)
			skippedLevels++
			continue
		}
		
		// Format values with proper precision
		formattedQty := fmt.Sprintf("%.6f", levelQty)
		formattedPrice := fmt.Sprintf("%.4f", tpPrice)
		
		// Place TP limit order with timeout
		orderParams := exchange.OrderParams{
			Category:  bot.category,
			Symbol:    bot.symbol,
			Side:      exchange.OrderSideSell,
			Quantity:  formattedQty,
			OrderType: exchange.OrderTypeLimit,
			Price:     formattedPrice,
		}
		
		// Add debugging and timing for individual order placement
		orderStartTime := time.Now()
		timeRemaining := "unknown"
		if hasDeadline {
			timeRemaining = fmt.Sprintf("%.0fs", time.Until(deadline).Seconds())
		}
		
		bot.logger.Info("📤 Placing TP Level %d: %s %s at $%s (%.2f%%) - Time remaining: %s", 
			level, formattedQty, bot.symbol, formattedPrice, levelPercent*100, timeRemaining)
		
		tpOrder, err := bot.placeOrderWithRetry(orderParams, false) // false for limit order
		orderDuration := time.Since(orderStartTime)
		
		if err != nil {
			bot.logger.LogWarning("TP Level %d", "Failed to place TP order after %v: %v", level, orderDuration, err)
			skippedLevels++
			continue
		}
		
		bot.logger.Info("✅ TP Level %d placed successfully - Order ID: %s", level, tpOrder.OrderID)
		
		// Track the TP order
		bot.tpOrderMutex.Lock()
		bot.activeTPOrders[tpOrder.OrderID] = &TPOrderInfo{
			Level:     level,
			Percent:   levelPercent,
			Quantity:  formattedQty,
			Price:     formattedPrice,
			OrderID:   tpOrder.OrderID,
			Filled:    false,
			FilledQty: "0",
		}
		bot.tpOrderMutex.Unlock() // Unlock immediately, not with defer
		bot.logger.Info("✅ TP Level %d placed: %s %s at $%s (%.2f%%) - Order ID: %s", 
			level, formattedQty, bot.symbol, formattedPrice, levelPercent*100, tpOrder.OrderID)
		
		// Track successful placement
		successCount++
		
		// Add rate limiting delay between orders (except for the last one)
		if level < bot.config.Strategy.TPLevels {
			delayMs := 500 // 500ms delay between orders to avoid rate limiting
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}
	
	if skippedLevels > 0 {
		bot.logger.LogWarning("TP Placement", "%d levels skipped due to constraints", skippedLevels)
	}
	
	// Add comprehensive completion logging
	totalDuration := time.Since(startTime)
	
	// Log summary to console
	if successCount > 0 {
		// Calculate total allocated quantity from pre-calculated levels
		var allocatedQty float64
		for _, qty := range levelQuantities {
			allocatedQty += qty
		}
		fmt.Printf("🎯 Multi-Level TP: %d/%d orders placed successfully (%.6f %s allocated, %.1f%%) in %v\n", 
			successCount, bot.config.Strategy.TPLevels, allocatedQty, bot.symbol, (allocatedQty/totalQty)*100, totalDuration)
		
		if skippedLevels > 0 {
			fmt.Printf("⚠️  %d TP levels skipped due to constraints\n", skippedLevels)
		}
		
		return nil // Don't fail if we placed some orders

	} else {
		return fmt.Errorf("failed to place any TP orders")
	}
}

// updateMultiLevelTPOrders updates existing multi-level TP orders when average price changes
func (bot *LiveBot) updateMultiLevelTPOrders(newAveragePrice float64) error {
	// First, collect order information and clear map while holding mutex
	bot.tpOrderMutex.Lock()
	memoryOrderCount := len(bot.activeTPOrders)
	
	if memoryOrderCount == 0 {
		bot.tpOrderMutex.Unlock()
		bot.logger.Info("✅ No TP orders in memory to update")
		return nil // No TP orders to update
	}
	
	ctx := context.Background()
	
	// First, sync with exchange to get current TP orders (some may have been filled)
	ordersToCancel := make(map[string]*TPOrderInfo)
	for orderID, tpInfo := range bot.activeTPOrders {
		ordersToCancel[orderID] = &TPOrderInfo{
			Level:    tpInfo.Level,
			OrderID:  orderID,
			Quantity: tpInfo.Quantity,
		}
	}
	
	// Clear the map properly while still holding mutex
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	bot.tpOrderMutex.Unlock()
	
	// Now perform sync and cancellation operations without holding mutex
	if err := bot.syncTPOrdersFromExchange(ctx); err != nil {
		bot.logger.LogWarning("TP Sync", "Failed to sync TP orders from exchange: %v", err)
		// Continue with collected order data as fallback
	} else {
	}
	
	if len(ordersToCancel) == 0 {
		bot.logger.Info("✅ No active TP orders found to cancel after exchange sync")
		return nil
	}
	
	bot.logger.Info("🔄 Updating TP orders: New avg $%.4f (%d orders to cancel)", newAveragePrice, len(ordersToCancel))
	
	// Cancel orders without holding mutex (to avoid blocking other operations)
	cancelledCount := 0
	failedCancellations := 0
	for orderID, tpInfo := range ordersToCancel {
		
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("TP Order Update", "Failed to cancel TP Level %d order %s: %v", tpInfo.Level, orderID, err)
			failedCancellations++
			continue
		}
		
		bot.logger.Info("❌ Cancelled TP Level %d order: %s (qty: %s)", tpInfo.Level, orderID, tpInfo.Quantity)
		cancelledCount++
	}
	
	if failedCancellations > 0 {
		bot.logger.LogWarning("TP Update", "%d TP orders failed to cancel, continuing with new orders", failedCancellations)
	}
	
	// Get current total position size after any TP fills
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get position for TP update: %w", err)
	}
	
	var totalPositionSize string
	var exchangeAvgPrice string
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			totalPositionSize = pos.Size
			exchangeAvgPrice = pos.AvgPrice
			break
		}
	}
	
	if totalPositionSize == "" || totalPositionSize == "0" {
		bot.logger.LogWarning("TP Update", "No position found for TP update - possibly all TP orders executed")
		return nil
	}
	
	exchangeAvgPriceFloat, err := parseFloat(exchangeAvgPrice)
	if err != nil {
		bot.logger.LogWarning("TP Update", "Failed to parse exchange avg price '%s': %v", exchangeAvgPrice, err)
		exchangeAvgPriceFloat = newAveragePrice // Use bot's price as fallback
	}
	
	// Use exchange average price if there's a significant difference (safety check)
	priceDiff := math.Abs(newAveragePrice - exchangeAvgPriceFloat)
	if priceDiff > 0.001 {
		bot.logger.LogWarning("TP Update", "Price mismatch: bot $%.4f vs exchange $%.4f (diff: $%.4f)", 
			newAveragePrice, exchangeAvgPriceFloat, priceDiff)
		newAveragePrice = exchangeAvgPriceFloat
	}
	
	// Place new multi-level TP orders based on new average price
	if err := bot.placeMultiLevelTPOrders(totalPositionSize, newAveragePrice); err != nil {
		bot.logger.LogWarning("TP Update", "Failed to place updated multi-level TP orders: %v", err)
		return fmt.Errorf("failed to place updated multi-level TP orders: %w", err)
	}
	
	
	// Log to console for visibility
	fmt.Printf("🔄 Multi-Level TP Updated: Cancelled %d old orders → %d new TP levels placed at avg $%.4f\n", 
		cancelledCount, bot.config.Strategy.TPLevels, newAveragePrice)
	
	return nil
}

// executeSell executes a sell order to close position
func (bot *LiveBot) executeSell(price float64) {
	if bot.currentPosition <= 0 {
		return
	}

	ctx := context.Background()
	
	// Get current position from exchange to ensure accuracy
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("Position check", "Could not get current position for sell: %v", err)
		return
	}
	
	// Find our position
	var positionSize string
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			positionSize = pos.Size
			break
		}
	}
	
	if positionSize == "" {
		bot.logger.LogWarning("Position close", "No position found to close")
		return
	}
	
	orderParams := exchange.OrderParams{
		Category:  bot.category,
		Symbol:    bot.symbol,
		Side:      exchange.OrderSideSell,
		Quantity:  positionSize, // Use exact position size from exchange
		OrderType: exchange.OrderTypeMarket,
	}

	order, err := bot.exchange.PlaceMarketOrder(ctx, orderParams)
	if err != nil {
		bot.logger.Error("Failed to place sell order: %v", err)
		return
	}

	// Sync with exchange data instead of self-calculating P&L
	bot.syncAfterTrade(order, "SELL")
}

// syncAfterTrade syncs bot state with exchange after trade execution
func (bot *LiveBot) syncAfterTrade(order *exchange.Order, tradeType string) {
	// Wait briefly for exchange to settle the trade
	time.Sleep(500 * time.Millisecond)
	
	// Sync balance with exchange
	if err := bot.syncAccountBalance(); err != nil {
		bot.logger.LogWarning("Could not refresh balance after trade", "%v", err)
	}
	
	// Capture DCA level BEFORE position sync to avoid interference
	bot.positionMutex.RLock()
	previousDCALevel := bot.dcaLevel
	bot.positionMutex.RUnlock()
	
	// Sync position data with exchange
	if err := bot.syncPositionData(); err != nil {
		bot.logger.LogWarning("Could not sync position after trade", "%v", err)
		// Don't return here - continue with the rest of the function
	}
	
	// Update DCA level for buy trades
	if tradeType == "BUY" {
		// Protect DCA level operations with mutex
		bot.positionMutex.Lock()
		// Use the captured previousDCALevel (before position sync interference)
		// This ensures proper DCA level progression regardless of sync side effects
		if previousDCALevel == 0 {
			bot.dcaLevel = 1 // First trade: 0 → 1
		} else {
			bot.dcaLevel = previousDCALevel + 1 // Subsequent trades: increment
		}
		
		// Validation: ensure DCA level makes sense relative to position size
		if bot.config.Strategy.BaseAmount > 0 {
			expectedLevel := max(1, int(bot.currentPosition/bot.config.Strategy.BaseAmount))
			if bot.dcaLevel != expectedLevel {
				// For large discrepancies, use position-based calculation
				diff := bot.dcaLevel - expectedLevel
				if diff < 0 {
					diff = -diff // abs
				}
				if diff > 1 {
					bot.dcaLevel = expectedLevel
				}
			}
		}
		
		currentDCALevel := bot.dcaLevel
		currentPosition := bot.currentPosition
		avgPrice := bot.averagePrice
		bot.positionMutex.Unlock()
		
		// Debug log for DCA level tracking
		bot.logger.Info("📊 DCA Level Update: %d → %d (position: $%.2f)", previousDCALevel, currentDCALevel, currentPosition)
		
		// Log trade execution details with reliable data (don't rely on potentially empty order response fields)
		// Use the calculated values and exchange-synced data instead
		var executedQty string
		if avgPrice > 0 {
			executedQty = fmt.Sprintf("%.6f", currentPosition/avgPrice) // Calculate from position
		} else {
			executedQty = "0.000000" // Safety fallback
		}
		executedPrice := fmt.Sprintf("%.4f", avgPrice)
		executedValue := fmt.Sprintf("%.2f", currentPosition)
		
		bot.logger.LogTradeExecution(tradeType, order.OrderID, executedQty, executedPrice, executedValue, currentDCALevel, currentPosition, avgPrice)
		
		// Update multi-level TP orders if this is not the first buy (DCA level > 1)
		// Note: We do this in syncAfterTrade to ensure it happens after position sync
		if currentDCALevel > 1 && bot.config.Strategy.AutoTPOrders {
			bot.logger.Info("🔄 DCA Level %d detected - updating TP orders for new average price $%.4f", currentDCALevel, avgPrice)
			fmt.Printf("🔄 DCA Trade Detected: Updating TP orders for level %d with new avg price $%.4f\n", currentDCALevel, avgPrice)
			
			// Update TP orders with proper error handling (no panic recovery needed)
			if err := bot.updateMultiLevelTPOrders(avgPrice); err != nil {
				bot.logger.LogWarning("TP Update", "Failed to update TP orders after DCA: %v", err)
				fmt.Printf("⚠️ TP update failed but bot continues normally\n")
				// Continue execution - don't let TP update failure stop the bot
			} else {
				fmt.Printf("✅ TP orders updated successfully for DCA level %d\n", currentDCALevel)
			}
		} else if currentDCALevel == 1 && bot.config.Strategy.AutoTPOrders {
			bot.logger.Info("🐛 DEBUG: First trade (level %d) - initial TP orders should be placed by executeBuy", currentDCALevel)
		} else if !bot.config.Strategy.AutoTPOrders {
			bot.logger.Info("🐛 DEBUG: AutoTPOrders disabled - skipping TP order management")
		}
		
	} else if tradeType == "SELL" {
		// Create dedicated context for cleanup operations
		cleanupCtx := context.Background()
		
		// For sell trades, calculate realized P&L from position data before sync
		if currentPrice, err := bot.exchange.GetLatestPrice(cleanupCtx, bot.symbol); err == nil {
			// Get average price safely with mutex protection
			bot.positionMutex.RLock()
			avgPrice := bot.averagePrice
			bot.positionMutex.RUnlock()
			
			// Add zero-value checks to prevent division by zero
			if avgPrice > 0 && currentPrice > 0 {
				profitPercent := (currentPrice - avgPrice) / avgPrice * 100
				
				// Log cycle completion
				bot.logger.LogCycleCompletion(currentPrice, avgPrice, profitPercent)
			}
		}
		
		// Clean up multi-level TP orders since position is closed
		// First, collect order information while holding mutex
		bot.tpOrderMutex.Lock()
		ordersToCancel := make(map[string]*TPOrderInfo)
		for orderID, tpInfo := range bot.activeTPOrders {
			ordersToCancel[orderID] = &TPOrderInfo{
				Level:   tpInfo.Level,
				OrderID: orderID,
			}
		}
		// Clear map while still holding mutex
		for k := range bot.activeTPOrders {
			delete(bot.activeTPOrders, k)
		}
		bot.tpOrderMutex.Unlock()
		
		// Now cancel orders without holding mutex (to avoid blocking other operations)
		for orderID, tpInfo := range ordersToCancel {
			if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
				bot.logger.LogWarning("TP Cleanup", "Failed to cancel TP Level %d order %s: %v", tpInfo.Level, orderID, err)
			} else {
				bot.logger.Info("🧹 Cancelled TP Level %d order %s (position closed)", tpInfo.Level, orderID)
			}
		}
		
		// Log trade execution details
		bot.logger.LogTradeExecution(tradeType, order.OrderID, order.CumExecQty, order.AvgPrice, order.CumExecValue, 0, 0, 0)
		
		// Reset internal counters after sell with mutex protection
		bot.positionMutex.Lock()
		bot.dcaLevel = 0
		bot.positionMutex.Unlock()
		
		// Notify strategy of cycle completion if configured
		if bot.config.Strategy.Cycle {
			bot.strategy.OnCycleComplete()
			// Reset hold log counter to ensure first HOLD decision after cycle completion is logged
			bot.holdLogCounter = 0
		}
	}
}

// syncExistingOrders syncs existing orders from exchange on startup
func (bot *LiveBot) syncExistingOrders() error {
	ctx := context.Background()
	
	// Get open orders for this symbol
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get existing orders: %w", err)
	}
	
	if len(orders) == 0 {
		fmt.Printf("✅ No existing orders found\n")
		return nil
	}
	
	// Check if we should cancel orphaned orders or sync them
	if bot.config.Strategy.CancelOrphanedOrders {
		return bot.cancelOrphanedOrders(orders)
	}
	
	// Sync existing orders instead of canceling
	tpOrderCount := 0
	otherOrderCount := 0
	
	bot.tpOrderMutex.Lock()
	defer bot.tpOrderMutex.Unlock()
	
	for _, order := range orders {
		// Check if this looks like a TP order (sell side, limit order)
		if order.Side == "Sell" && order.OrderType == "Limit" {
			tpOrderCount++
			
			// Try to reconstruct TP order info
			// Note: We can't perfectly reconstruct level/percent without more context
			// but we can track the order for cancellation purposes
			bot.activeTPOrders[order.OrderID] = &TPOrderInfo{
				Level:     0, // Unknown level
				Percent:   0, // Unknown percent
				Quantity:  order.Quantity,
				Price:     order.Price,
				OrderID:   order.OrderID,
			}
			
		} else {
			otherOrderCount++
		}
	}
	
	if tpOrderCount > 0 {
		fmt.Printf("🎯 Synced %d existing TP orders\n", tpOrderCount)
	}
	if otherOrderCount > 0 {
		fmt.Printf("📋 Found %d other orders (not TP orders)\n", otherOrderCount)
	}
	
	return nil
}

// calculateTPLevelQuantities pre-calculates quantities for all TP levels to ensure proper distribution
func (bot *LiveBot) calculateTPLevelQuantities(totalQty float64, constraints *exchange.TradingConstraints) []float64 {
	numLevels := bot.config.Strategy.TPLevels
	// Add safety check for invalid configuration
	if numLevels <= 0 {
		bot.logger.LogWarning("TP Calculation", "Invalid TP levels configuration: %d", numLevels)
		return []float64{} // Return empty slice
	}
	quantities := make([]float64, numLevels)
	
	// Calculate base quantity per level
	baseQtyPerLevel := totalQty * bot.config.Strategy.TPQuantity
	
	// Apply MinOrderQty constraint to base quantity
	if constraints.MinOrderQty > 0 && baseQtyPerLevel < constraints.MinOrderQty {
		baseQtyPerLevel = constraints.MinOrderQty
	}
	
	// Apply QtyStep constraint to base quantity
	if constraints.QtyStep > 0 {
		multiplier := math.Floor(baseQtyPerLevel / constraints.QtyStep)
		if multiplier < 1 {
			multiplier = 1
		}
		baseQtyPerLevel = multiplier * constraints.QtyStep
	}
	
	// Calculate total quantity needed for all levels with base quantity
	totalNeeded := baseQtyPerLevel * float64(numLevels)
	
	if totalNeeded <= totalQty {
		// Simple case: all levels get the same base quantity
		for i := 0; i < numLevels; i++ {
			quantities[i] = baseQtyPerLevel
		}
		
		// Distribute any remaining quantity to the last level
		remaining := totalQty - totalNeeded
		if remaining > 0 {
			quantities[numLevels-1] += remaining
		}
	} else {
		// Complex case: need to distribute available quantity across levels
		// Start with minimum quantities and distribute the rest
		remaining := totalQty
		
		// First pass: give each level the minimum required quantity
		minQty := math.Max(constraints.MinOrderQty, constraints.QtyStep)
		if minQty == 0 {
			minQty = 1 // Default minimum
		}
		
		for i := 0; i < numLevels && remaining >= minQty; i++ {
			quantities[i] = minQty
			remaining -= minQty
		}
		
		// Second pass: distribute remaining quantity proportionally
		if remaining > 0 {
			// Find how many levels got the minimum quantity
			activeLevels := 0
			for i := 0; i < numLevels; i++ {
				if quantities[i] > 0 {
					activeLevels++
				}
			}
			
			if activeLevels > 0 {
				extraPerLevel := remaining / float64(activeLevels)
				
				// Apply QtyStep constraint to extra quantity
				if constraints.QtyStep > 0 {
					multiplier := math.Floor(extraPerLevel / constraints.QtyStep)
					extraPerLevel = multiplier * constraints.QtyStep
				}
				
				// Distribute extra quantity
				for i := 0; i < numLevels && extraPerLevel > 0; i++ {
					if quantities[i] > 0 {
						quantities[i] += extraPerLevel
						remaining -= extraPerLevel
					}
				}
				
				// Add any final remaining to the last level
				if remaining > 0 && quantities[numLevels-1] > 0 {
					quantities[numLevels-1] += remaining
				}
			}
		}
	}
	
	// Only log if there's something notable about the distribution
	if totalNeeded > totalQty {
		bot.logger.LogWarning("TP Calculation", "Insufficient quantity: need %.6f, have %.6f", totalNeeded, totalQty)
	}
	
	return quantities
}

// syncTPOrdersFromExchange syncs internal TP order tracking with exchange state
func (bot *LiveBot) syncTPOrdersFromExchange(ctx context.Context) error {
	// Get current open orders from exchange
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get open orders: %w", err)
	}
	
	// Get current average price for TP order validation
	positions, posErr := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	var currentAvgPrice float64 = 0
	if posErr == nil {
		for _, pos := range positions {
			if pos.Symbol == bot.symbol && pos.Side == "Buy" {
				if avgPrice, err := parseFloat(pos.AvgPrice); err == nil {
					currentAvgPrice = avgPrice
				}
				break
			}
		}
	}
	
	// Create a map of current TP orders on exchange with proper identification
	exchangeTPOrders := make(map[string]*exchange.Order)
	for _, order := range orders {
		if bot.isTPOrder(order, currentAvgPrice) {
			exchangeTPOrders[order.OrderID] = order
		}
	}
	
	// Remove filled/cancelled orders from our tracking
	removedCount := 0
	for orderID := range bot.activeTPOrders {
		if _, exists := exchangeTPOrders[orderID]; !exists {
			// Order no longer exists on exchange (filled or cancelled)
			delete(bot.activeTPOrders, orderID)
			removedCount++
		}
	}
	
	// Add any TP orders we weren't tracking (shouldn't happen, but safety check)
	addedCount := 0
	for orderID, order := range exchangeTPOrders {
		if _, exists := bot.activeTPOrders[orderID]; !exists {
			// Found TP order we weren't tracking
			bot.activeTPOrders[orderID] = &TPOrderInfo{
				Level:     0, // Unknown level
				Percent:   0, // Unknown percent
				Quantity:  order.Quantity,
				Price:     order.Price,
				OrderID:   orderID,
			}
			addedCount++
		}
	}
	
	if removedCount > 0 || addedCount > 0 {
		bot.logger.Info("🔄 TP Sync: %d filled orders removed, %d untracked added", removedCount, addedCount)
	}
	
	return nil
}

// cancelOrphanedOrders cancels all existing orders on startup
func (bot *LiveBot) cancelOrphanedOrders(orders []*exchange.Order) error {
	bot.logger.Info("🧹 Canceling %d orphaned orders on startup...", len(orders))
	fmt.Printf("🧹 Canceling %d orphaned orders...\n", len(orders))
	
	cancelledCount := 0
	failedCount := 0
	
	for _, order := range orders {
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, order.OrderID); err != nil {
			bot.logger.LogWarning("Startup Cleanup", "Failed to cancel order %s (%s %s): %v", 
				order.OrderID, order.Side, order.OrderType, err)
			failedCount++
			continue
		}
		
		bot.logger.Info("❌ Cancelled orphaned order: %s (%s %s %s)", 
			order.OrderID, order.Side, order.OrderType, order.Quantity)
		cancelledCount++
	}
	
	if failedCount > 0 {
		bot.logger.LogWarning("Startup Cleanup", "Cancelled %d orders, %d failed", cancelledCount, failedCount)
		fmt.Printf("⚠️ Cancelled %d orders, %d failed to cancel\n", cancelledCount, failedCount)
	} else {
		fmt.Printf("✅ Cancelled %d orphaned orders successfully\n", cancelledCount)
	}
	
	return nil
}

// cancelAllTPOrders cancels all active TP orders during shutdown
func (bot *LiveBot) cancelAllTPOrders() error {
	// First, sync with exchange to get REAL state (not just memory)
	ctx := context.Background()
	fmt.Printf("🔍 Checking exchange for active TP orders...\n")
	
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("Shutdown Cleanup", "Failed to get open orders from exchange: %v", err)
		fmt.Printf("⚠️ Failed to fetch exchange orders, using memory fallback\n")
		// Fall back to memory-based cancellation
		return bot.cancelTPOrdersFromMemory()
	}
	
	// Find TP orders on exchange with proper identification
	exchangeTPOrders := make(map[string]*exchange.Order)
	
	// Get current position to validate TP order prices
	positions, posErr := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	var currentAvgPrice float64 = 0
	if posErr == nil {
		for _, pos := range positions {
			if pos.Symbol == bot.symbol && pos.Side == "Buy" {
				if avgPrice, err := parseFloat(pos.AvgPrice); err == nil {
					currentAvgPrice = avgPrice
				}
				break
			}
		}
	}
	
	for _, order := range orders {
		if bot.isTPOrder(order, currentAvgPrice) {
			exchangeTPOrders[order.OrderID] = order
		}
	}
	
	orderCount := len(exchangeTPOrders)
	
	if orderCount == 0 {
		bot.logger.Info("✅ No active TP orders found on exchange")
		fmt.Printf("✅ No active TP orders found on exchange\n")
		return nil
	}
	
	bot.logger.Info("🧹 Canceling %d active TP orders from exchange...", orderCount)
	fmt.Printf("🧹 Found %d TP orders on exchange to cancel...\n", orderCount)
	
	// Cancel orders directly from exchange data (no need for memory collection)
	// Clear our internal tracking since we're cancelling everything
	bot.tpOrderMutex.Lock()
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	bot.tpOrderMutex.Unlock()
	
	// Now cancel orders based on exchange data
	cancelledCount := 0
	failedCount := 0
	
	// Create level counter for better logging (since we don't know actual levels from exchange)
	level := 1
	for orderID, order := range exchangeTPOrders {
		fmt.Printf("🔄 Cancelling TP order %d/%d...\n", level, orderCount)
		
		// Use existing method with built-in timeout and retry logic
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("Shutdown Cleanup", "Failed to cancel TP order %s: %v", orderID, err)
			fmt.Printf("❌ Failed to cancel TP order %d: %v\n", level, err)
			failedCount++
			level++
			continue
		}
		
		bot.logger.Info("❌ Cancelled TP order: %s (qty: %s, price: %s)", orderID, order.Quantity, order.Price)
		fmt.Printf("✅ Cancelled TP order %d successfully\n", level)
		cancelledCount++
		level++
	}
	
	if failedCount > 0 {
		bot.logger.LogWarning("Shutdown Cleanup", "Cancelled %d TP orders, %d failed", cancelledCount, failedCount)
		fmt.Printf("⚠️ Cancelled %d TP orders, %d failed to cancel\n", cancelledCount, failedCount)
	} else {
		fmt.Printf("✅ Cancelled %d TP orders successfully\n", cancelledCount)
	}
	
	return nil
}

// cancelTPOrdersFromMemory is a fallback when exchange fetch fails
func (bot *LiveBot) cancelTPOrdersFromMemory() error {
	// First, collect order information while holding mutex
	bot.tpOrderMutex.Lock()
	orderCount := len(bot.activeTPOrders)
	
	if orderCount == 0 {
		bot.tpOrderMutex.Unlock()
		bot.logger.Info("✅ No active TP orders in memory to cancel")
		fmt.Printf("✅ No active TP orders found in memory\n")
		return nil
	}
	
	bot.logger.Info("🧹 Canceling %d TP orders from memory...", orderCount)
	fmt.Printf("🧹 Found %d TP orders in memory to cancel...\n", orderCount)
	
	// Collect order information to cancel
	ordersToCancel := make(map[string]*TPOrderInfo)
	for orderID, tpInfo := range bot.activeTPOrders {
		ordersToCancel[orderID] = &TPOrderInfo{
			Level:    tpInfo.Level,
			OrderID:  orderID,
			Quantity: tpInfo.Quantity,
		}
	}
	
	// Clear the map while still holding mutex
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	bot.tpOrderMutex.Unlock()
	
	// Now cancel orders without holding mutex
	cancelledCount := 0
	failedCount := 0
	
	for orderID, tpInfo := range ordersToCancel {
		fmt.Printf("🔄 Cancelling TP Level %d order...\n", tpInfo.Level)
		
		// Use existing method with built-in timeout and retry logic
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("Shutdown Cleanup", "Failed to cancel TP Level %d order %s: %v", tpInfo.Level, orderID, err)
			fmt.Printf("❌ Failed to cancel TP Level %d: %v\n", tpInfo.Level, err)
			failedCount++
			continue
		}
		
		bot.logger.Info("❌ Cancelled TP Level %d order: %s (qty: %s)", tpInfo.Level, orderID, tpInfo.Quantity)
		fmt.Printf("✅ Cancelled TP Level %d successfully\n", tpInfo.Level)
		cancelledCount++
	}
	
	if failedCount > 0 {
		bot.logger.LogWarning("Shutdown Cleanup", "Cancelled %d TP orders, %d failed", cancelledCount, failedCount)
		fmt.Printf("⚠️ Cancelled %d TP orders, %d failed to cancel\n", cancelledCount, failedCount)
	} else {
		fmt.Printf("✅ Cancelled %d TP orders successfully\n", cancelledCount)
	}
	
	return nil
}

// isTPOrder determines if an order is a take profit order placed by this bot
func (bot *LiveBot) isTPOrder(order *exchange.Order, currentAvgPrice float64) bool {
	// Must be a sell limit order
	if order.Side != "Sell" || order.OrderType != "Limit" {
		return false
	}
	
	// Must be for our symbol (should already be filtered, but safety check)
	if order.Symbol != bot.symbol {
		return false
	}
	
	// Parse order price
	orderPrice, err := parseFloat(order.Price)
	if err != nil {
		bot.logger.LogWarning("TP Identification", "Failed to parse order price '%s': %v", order.Price, err)
		return false
	}
	
	// If we don't have current avg price, we can't validate - be conservative
	if currentAvgPrice <= 0 {
		bot.logger.Info("🐛 DEBUG: No avg price available, checking if order is in tracked TP orders")
		// Check if this order is in our internal tracking
		bot.tpOrderMutex.RLock()
		_, isTracked := bot.activeTPOrders[order.OrderID]
		bot.tpOrderMutex.RUnlock()
		return isTracked
	}
	
	// TP orders should be ABOVE average price (for profit)
	if orderPrice <= currentAvgPrice {
		bot.logger.Info("🐛 DEBUG: Order price $%.4f <= avg price $%.4f, not a TP order", orderPrice, currentAvgPrice)
		return false
	}
	
	// TP orders should be within reasonable profit range (0.1% to 15% above avg price)
	profitPercent := (orderPrice - currentAvgPrice) / currentAvgPrice * 100
	minProfitPercent := 0.1  // 0.1% minimum
	maxProfitPercent := 15.0 // 15% maximum (beyond this is likely not a TP order)
	
	if profitPercent < minProfitPercent || profitPercent > maxProfitPercent {
		bot.logger.Info("🐛 DEBUG: Order profit %.2f%% outside TP range (%.1f%%-%.1f%%), not a TP order", 
			profitPercent, minProfitPercent, maxProfitPercent)
		return false
	}
	
	// Check quantity - TP orders typically have smaller quantities than full position
	orderQty, qtyErr := parseFloat(order.Quantity)
	if qtyErr == nil && currentAvgPrice > 0 {
		// Estimate full position size from our current position value
		bot.positionMutex.RLock()
		fullPositionQty := bot.currentPosition / currentAvgPrice
		bot.positionMutex.RUnlock()
		
		if fullPositionQty > 0 {
			qtyPercent := orderQty / fullPositionQty * 100
			// TP orders are typically 10%-50% of position (not 100%+ which would be full exit)
			if qtyPercent > 70 {
				bot.logger.Info("🐛 DEBUG: Order quantity %.1f%% of position suggests full exit, not TP order", qtyPercent)
				return false
			}
		}
	}
	
	return true
}

// detectFilledTPOrders detects which TP orders have been filled since last check
func (bot *LiveBot) detectFilledTPOrders() []string {
	ctx := context.Background()
	
	// Get current open orders
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("TP Fill Detection", "Failed to get open orders: %v", err)
		return nil
	}
	
	// Get current average price for TP validation
	positions, posErr := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	var currentAvgPrice float64 = 0
	if posErr == nil {
		for _, pos := range positions {
			if pos.Symbol == bot.symbol && pos.Side == "Buy" {
				if avgPrice, err := parseFloat(pos.AvgPrice); err == nil {
					currentAvgPrice = avgPrice
				}
				break
			}
		}
	}
	
	// Create map of currently active TP orders on exchange
	activeOnExchange := make(map[string]bool)
	for _, order := range orders {
		if bot.isTPOrder(order, currentAvgPrice) {
			activeOnExchange[order.OrderID] = true
		}
	}
	
	// Find filled TP orders (in our tracking but not on exchange)
	bot.tpOrderMutex.Lock()
	var filledOrderDetails []string
	
	for orderID, tpInfo := range bot.activeTPOrders {
		if !activeOnExchange[orderID] {
			// Order is no longer on exchange - likely filled
			bot.logger.Info("🎯 TP Order FILLED: Level %d at $%s (%.2f%%) - OrderID: %s", 
				tpInfo.Level, tpInfo.Price, tpInfo.Percent*100, orderID)
			
			// Move to filled orders tracking
			tpInfo.Filled = true
			bot.filledTPOrders[orderID] = tpInfo
			
			// Create detailed string for display
			filledDetail := fmt.Sprintf("TP%d@$%s(%.1f%%)", tpInfo.Level, tpInfo.Price, tpInfo.Percent*100)
			filledOrderDetails = append(filledOrderDetails, filledDetail)
			
			// Remove from active orders
			delete(bot.activeTPOrders, orderID)
		}
	}
	
	bot.tpOrderMutex.Unlock()
	
	return filledOrderDetails
}

// getFilledTPSummary returns a summary of filled TP orders for display
func (bot *LiveBot) getFilledTPSummary() string {
	bot.tpOrderMutex.RLock()
	defer bot.tpOrderMutex.RUnlock()
	
	if len(bot.filledTPOrders) == 0 {
		return ""
	}
	
	var filledSummary []string
	for _, tpInfo := range bot.filledTPOrders {
		summary := fmt.Sprintf("TP%d@$%s", tpInfo.Level, tpInfo.Price)
		filledSummary = append(filledSummary, summary)
	}
	
	return strings.Join(filledSummary, ", ")
}

// parseFloat safely parses a string to float64 with comprehensive validation
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	// Trim whitespace that might cause parsing errors
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("string contains only whitespace")
	}
	// Check for common invalid values
	if s == "null" || s == "undefined" || s == "NaN" {
		return 0, fmt.Errorf("invalid numeric value: %s", s)
	}
	return strconv.ParseFloat(s, 64)
}

// placeOrderWithRetry places an order with timeout and retry logic
func (bot *LiveBot) placeOrderWithRetry(params exchange.OrderParams, isMarket bool) (*exchange.Order, error) {
	maxRetries := 3
	baseDelay := 1 * time.Second
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		// Use a shorter timeout per attempt to prevent hanging
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		
		// Create a channel to handle the order placement with timeout
		orderChan := make(chan *exchange.Order, 1)
		errChan := make(chan error, 1)
		
		go func() {
			var order *exchange.Order
			var err error
			
			if isMarket {
				order, err = bot.exchange.PlaceMarketOrder(ctx, params)
			} else {
				order, err = bot.exchange.PlaceLimitOrder(ctx, params)
			}
			
			if err != nil {
				errChan <- err
			} else {
				orderChan <- order
			}
		}()
		
		// Wait for either success, error, or timeout
		select {
		case order := <-orderChan:
			cancel()
			return order, nil
		case err := <-errChan:
			cancel()
			
			// Check if error is retryable
			if exchangeErr, ok := err.(*exchange.ExchangeError); ok && !exchangeErr.IsRetryable {
				return nil, fmt.Errorf("non-retryable error: %w", err)
			}
			
			if attempt < maxRetries {
				delay := time.Duration(attempt) * baseDelay
				bot.logger.LogWarning("Order Retry", "Attempt %d/%d failed: %v. Retrying in %v", attempt, maxRetries, err, delay)
				time.Sleep(delay)
			} else {
				return nil, fmt.Errorf("failed after %d attempts, last error: %w", maxRetries, err)
			}
		case <-ctx.Done():
			cancel()
			if attempt < maxRetries {
				delay := time.Duration(attempt) * baseDelay
				bot.logger.LogWarning("Order Retry", "Attempt %d/%d timed out. Retrying in %v", attempt, maxRetries, delay)
				time.Sleep(delay)
			} else {
				return nil, fmt.Errorf("order placement timed out after %d attempts", maxRetries)
			}
		}
	}
	
	return nil, fmt.Errorf("failed after %d attempts", maxRetries)
}

// cancelOrderWithRetry cancels an order with timeout and retry logic
func (bot *LiveBot) cancelOrderWithRetry(category, symbol, orderID string) error {
	maxRetries := 3
	baseDelay := 500 * time.Millisecond
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := bot.exchange.CancelOrder(ctx, category, symbol, orderID)
		cancel()
		
		if err == nil {
			return nil
		}
		
		// Check if error is retryable
		if exchangeErr, ok := err.(*exchange.ExchangeError); ok && !exchangeErr.IsRetryable {
			return fmt.Errorf("non-retryable error: %w", err)
		}
		
		if attempt < maxRetries {
			delay := time.Duration(attempt) * baseDelay
			bot.logger.LogWarning("Cancel Retry", "Attempt %d/%d failed: %v. Retrying in %v", attempt, maxRetries, err, delay)
			time.Sleep(delay)
		}
	}
	
	return fmt.Errorf("failed to cancel order after %d attempts", maxRetries)
}

// placeFallbackTPOrder places a single TP order for leftover quantity at level 5
func (bot *LiveBot) placeFallbackTPOrder(quantity float64, avgEntryPrice float64, constraints *exchange.TradingConstraints, level int) error {
	// Add zero-value check to prevent division by zero
	if bot.config.Strategy.TPLevels <= 0 {
		return fmt.Errorf("invalid TP levels configuration: %d", bot.config.Strategy.TPLevels)
	}
	
	// Use level 5 TP percentage for fallback order (highest level)
	levelPercent := bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
	tpPrice := avgEntryPrice * (1 + levelPercent)
	
	// Round price to exchange tick size
	if constraints.MinPriceStep > 0 {
		tpPrice = math.Round(tpPrice/constraints.MinPriceStep) * constraints.MinPriceStep
	}
	
	// Apply quantity step constraint using floor
	if constraints.QtyStep > 0 {
		multiplier := math.Floor(quantity / constraints.QtyStep)
		if multiplier < 1 {
			return fmt.Errorf("fallback quantity %.6f < step size %.6f", quantity, constraints.QtyStep)
		}
		quantity = multiplier * constraints.QtyStep
	}
	
	// Check minimum order constraints
	orderValue := quantity * tpPrice
	if quantity < constraints.MinOrderQty {
		return fmt.Errorf("fallback quantity %.6f < min %.6f", quantity, constraints.MinOrderQty)
	}
	if orderValue < constraints.MinOrderValue {
		return fmt.Errorf("fallback order value $%.2f < min $%.2f", orderValue, constraints.MinOrderValue)
	}
	
	// Format values
	formattedQty := fmt.Sprintf("%.6f", quantity)
	formattedPrice := fmt.Sprintf("%.4f", tpPrice)
	
	// Place fallback TP limit order
	orderParams := exchange.OrderParams{
		Category:  bot.category,
		Symbol:    bot.symbol,
		Side:      exchange.OrderSideSell,
		Quantity:  formattedQty,
		OrderType: exchange.OrderTypeLimit,
		Price:     formattedPrice,
	}
	
	tpOrder, err := bot.placeOrderWithRetry(orderParams, false) // false for limit order
	if err != nil {
		return fmt.Errorf("failed to place fallback TP order: %w", err)
	}
	
	// Track the fallback TP order
	bot.tpOrderMutex.Lock()
	defer bot.tpOrderMutex.Unlock()
	bot.activeTPOrders[tpOrder.OrderID] = &TPOrderInfo{
		Level:     level, // Use next available level number
		Percent:   levelPercent,
		Quantity:  formattedQty,
		Price:     formattedPrice,
		OrderID:   tpOrder.OrderID,
		Filled:    false,
		FilledQty: "0",
	}
	
	bot.logger.Info("✅ Fallback TP combined with Level %d: %s %s at $%s (%.2f%%) - Order ID: %s", 
		level, formattedQty, bot.symbol, formattedPrice, levelPercent*100, tpOrder.OrderID)
	
	return nil
}
