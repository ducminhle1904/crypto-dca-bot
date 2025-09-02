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
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
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
	tpOrderMutex   sync.RWMutex            // Protect TP order map access
	
	// Position synchronization
	positionMutex  sync.RWMutex      // Protect position data access
	
	// Regime detection (Phase 1) - Foundation for dual engine system
	regimeDetector    *regime.RegimeDetector     // Market regime detection
	currentRegime     regime.RegimeType          // Current detected regime
	lastRegimeSignal  *regime.RegimeSignal       // Last regime detection result
	regimeHistory     []*regime.RegimeChange     // History of regime changes
	regimeMutex       sync.RWMutex               // Protect regime data access
	
	// Market data buffer for regime detection
	marketDataBuffer  []types.OHLCV              // Historical data for regime analysis
	bufferSize        int                        // Maximum buffer size
	bufferMutex       sync.RWMutex               // Protect buffer access
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
		tpOrderMutex: sync.RWMutex{},
		
		// Initialize regime detection (Phase 1)
		regimeDetector:    regime.NewRegimeDetector(),
		currentRegime:     regime.RegimeUncertain,
		lastRegimeSignal:  nil,
		regimeHistory:     make([]*regime.RegimeChange, 0, 100),
		regimeMutex:       sync.RWMutex{},
		
		// Initialize market data buffer (500 periods = ~40 hours of 5m data)
		marketDataBuffer:  make([]types.OHLCV, 0, 500),
		bufferSize:        500,
		bufferMutex:       sync.RWMutex{},
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
	if bot.running {
		bot.running = false
		
		fmt.Printf("🛑 Stopping bot...\n")
		
		// Cancel all active TP orders before closing positions
		if err := bot.cancelAllTPOrders(); err != nil {
			bot.logger.Error("Error canceling TP orders during shutdown: %v", err)
		}
		
		// Close any open positions before stopping
		if err := bot.closeOpenPositions(); err != nil {
			bot.logger.Error("Error closing positions during shutdown: %v", err)
		}
		
		// Disconnect from exchange
		if err := bot.exchange.Disconnect(); err != nil {
			bot.logger.Error("Error disconnecting from exchange: %v", err)
		}
		
		// Close logger
		if bot.logger != nil {
			bot.logger.Close()
		}
		
		close(bot.stopChan)
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
	
	// Look for our symbol position
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			positionValue, _ := parseFloat(pos.PositionValue)
			avgPrice, _ := parseFloat(pos.AvgPrice)
			
			if positionValue > 0 && avgPrice > 0 {
				bot.currentPosition = positionValue
				bot.averagePrice = avgPrice
				bot.totalInvested = positionValue
				bot.dcaLevel = 1 // Assume at least level 1
				
				bot.logger.LogPositionSync(positionValue, avgPrice, pos.Size, pos.UnrealisedPnl)
				// Show brief console message
				fmt.Printf("🔄 Existing position synced (see log for details)\n")
				return nil
			}
		}
	}
	
	// No existing position found
	bot.currentPosition = 0
	bot.averagePrice = 0
	bot.totalInvested = 0
	bot.dcaLevel = 0
	fmt.Printf("✅ No existing position found - starting fresh\n")
	
	return nil
}

// tradingLoop runs the main trading logic
func (bot *LiveBot) tradingLoop() {
		// Calculate interval duration
	intervalDuration := bot.getIntervalDuration()
	bot.logger.Info("Trading interval: %s (%v)", bot.interval, intervalDuration)
	
	// Wait for next candle close
	waitDuration := bot.getTimeUntilNextCandle()
	bot.logger.Info("Waiting %.0f seconds for next %s candle close", waitDuration.Seconds(), bot.interval)
	time.Sleep(waitDuration)

	// Run initial check
	bot.logger.Info("First candle closed - running initial check")
	bot.checkAndTrade()

	// Create ticker for regular checks
	ticker := time.NewTicker(intervalDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bot.checkAndTrade()
		case <-bot.stopChan:
			bot.logger.Info("Stop signal received - ending trading loop")
			return
		}
	}
}

// checkAndTrade performs market analysis and executes trades
func (bot *LiveBot) checkAndTrade() {
	defer func() {
		if r := recover(); r != nil {
			bot.logger.Error("Error in trading loop: %v", r)
		}
	}()

	ctx := context.Background()

	// Refresh account balance
	if err := bot.syncAccountBalance(); err != nil {
		bot.logger.LogWarning("Could not refresh balance", "%v", err)
		// Continue despite balance sync failure
	}

	// Sync position data
	if err := bot.syncPositionData(); err != nil {
		bot.logger.LogWarning("Could not sync position data", "%v", err)
		// Continue despite position sync failure
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

	// Update market data buffer and detect regime (Phase 1)
	bot.updateMarketDataBuffer(klines)
	bot.detectAndUpdateRegime()

	// Analyze market conditions
	decision, action := bot.analyzeMarket(klines, currentPrice)

	// Log market status to file
	exchangePnL := bot.getExchangePnL()
	bot.logger.LogMarketStatus(currentPrice, action, bot.balance, bot.currentPosition, bot.averagePrice, bot.dcaLevel, exchangePnL)

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
	
	if decision.Action == strategy.ActionBuy {
		return decision, "BUY"
	}

	// Note: Take profit is now handled by exchange limit orders placed after each buy
	// This eliminates the need to wait for candle closes and provides immediate TP protection
	// The exchange will automatically execute TP orders when price reaches target levels

	return decision, "HOLD"
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

	// Calculate quantity and apply constraints
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
						
						// Add recovery mechanism for TP placement
						func() {
							defer func() {
								if r := recover(); r != nil {
									bot.logger.Error("TP placement panic recovered: %v", r)
								}
							}()
							
							if err := bot.placeMultiLevelTPOrders(pos.Size, avgPrice); err != nil {
								bot.logger.LogWarning("Multi-Level TP Setup", "Could not place TP orders: %v", err)
								// Continue execution - don't let TP failure stop the bot
							}
						}()
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
	// Use a longer timeout for placing multiple orders
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	// Add a channel to handle timeout gracefully
	done := make(chan error, 1)
	
	go func() {
		done <- bot.placeMultiLevelTPOrdersInternal(ctx, totalQuantity, avgEntryPrice)
	}()
	
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		bot.logger.LogWarning("TP Placement Timeout", "TP order placement timed out after 60 seconds")
		return fmt.Errorf("TP order placement timed out")
	}
}

// placeMultiLevelTPOrdersInternal is the internal implementation
func (bot *LiveBot) placeMultiLevelTPOrdersInternal(ctx context.Context, totalQuantity string, avgEntryPrice float64) error {
	
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
	}
	
	// Pre-calculate quantities for all TP levels to ensure proper distribution
	levelQuantities := bot.calculateTPLevelQuantities(totalQty, constraints)
	
	bot.logger.Info("🎯 Placing %d-level TP orders from avg entry $%.4f: %.6f %s total", 
		bot.config.Strategy.TPLevels, avgEntryPrice, totalQty, bot.symbol)
	
	successCount := 0
	skippedLevels := 0
	
	// Place TP orders for each level using pre-calculated quantities
	for level := 1; level <= bot.config.Strategy.TPLevels; level++ {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			bot.logger.LogWarning("TP Placement", "Context cancelled during TP level %d placement", level)
			return fmt.Errorf("context cancelled during TP placement")
		default:
		}
		
		// Get pre-calculated quantity for this level
		levelQty := levelQuantities[level-1] // Array is 0-indexed
		if levelQty <= 0 {
			skippedLevels++
			continue
		}
		
		// Calculate TP price for this level based on actual average entry
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
		
		// Add extra timeout protection for TP order placement
		bot.logger.Info("📤 Placing TP Level %d: %s %s at $%s (%.2f%%)", 
			level, formattedQty, bot.symbol, formattedPrice, levelPercent*100)
		
		tpOrder, err := bot.placeOrderWithRetry(orderParams, false) // false for limit order
		if err != nil {
			bot.logger.LogWarning("TP Level %d", "Failed to place TP order: %v", level, err)
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
		bot.tpOrderMutex.Unlock()
		
		bot.logger.Info("✅ TP Level %d placed: %s %s at $%s (%.2f%%) - Order ID: %s", 
			level, formattedQty, bot.symbol, formattedPrice, levelPercent*100, tpOrder.OrderID)
		
		// Track successful placement
		successCount++
	}
	
	if skippedLevels > 0 {
		bot.logger.LogWarning("TP Placement", "%d levels skipped due to constraints", skippedLevels)
	}
	
	// Log summary to console
	if successCount > 0 {
		// Calculate total allocated quantity from pre-calculated levels
		var allocatedQty float64
		for _, qty := range levelQuantities {
			allocatedQty += qty
		}
		fmt.Printf("🎯 Multi-Level TP: %d/%d orders placed successfully (%.6f %s allocated, %.1f%%)\n", 
			successCount, bot.config.Strategy.TPLevels, allocatedQty, bot.symbol, (allocatedQty/totalQty)*100)
		
		if skippedLevels > 0 {
			fmt.Printf("⚠️  %d TP levels skipped due to minStep constraints\n", skippedLevels)
		}

	} else {
		return fmt.Errorf("failed to place any TP orders")
	}
	
	return nil
}

// updateMultiLevelTPOrders updates existing multi-level TP orders when average price changes
func (bot *LiveBot) updateMultiLevelTPOrders(newAveragePrice float64) error {
	bot.tpOrderMutex.Lock()
	defer bot.tpOrderMutex.Unlock()
	
	if len(bot.activeTPOrders) == 0 {
		return nil // No TP orders to update
	}
	
	ctx := context.Background()
	
	// First, sync with exchange to get current TP orders (some may have been filled)
	if err := bot.syncTPOrdersFromExchange(ctx); err != nil {
		bot.logger.LogWarning("TP Sync", "Failed to sync TP orders from exchange: %v", err)
		// Continue with internal state as fallback
	}
	
	// Check again after sync - some orders might have been filled
	if len(bot.activeTPOrders) == 0 {
		bot.logger.Info("✅ No active TP orders found after sync - all may have been filled")
		return nil
	}
	
	bot.logger.Info("🔄 Updating TP orders: New avg $%.4f (%d active orders)", newAveragePrice, len(bot.activeTPOrders))
	
	// Cancel all remaining TP orders with retry logic
	cancelledCount := 0
	failedCancellations := 0
	for orderID, tpInfo := range bot.activeTPOrders {
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
	
	// Clear the map properly
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
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
		bot.logger.LogWarning("TP Update", "No position found for TP update")
		return nil
	}
	
	exchangeAvgPriceFloat, _ := parseFloat(exchangeAvgPrice)
	
	// Use exchange average price if there's a significant difference (safety check)
	if math.Abs(newAveragePrice-exchangeAvgPriceFloat) > 0.001 {
		bot.logger.LogWarning("TP Update", "Price mismatch: bot $%.4f → exchange $%.4f", 
			newAveragePrice, exchangeAvgPriceFloat)
		newAveragePrice = exchangeAvgPriceFloat
	}
	
	// Place new multi-level TP orders based on new average price
	if err := bot.placeMultiLevelTPOrders(totalPositionSize, newAveragePrice); err != nil {
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
	
	// Sync position data with exchange
	if err := bot.syncPositionData(); err != nil {
		bot.logger.LogWarning("Could not sync position after trade", "%v", err)
		// Don't return here - continue with the rest of the function
	}
	
	// Update DCA level for buy trades
	if tradeType == "BUY" {
		bot.dcaLevel++
		
		// Log trade execution details
		bot.logger.LogTradeExecution(tradeType, order.OrderID, order.CumExecQty, order.AvgPrice, order.CumExecValue, bot.dcaLevel, bot.currentPosition, bot.averagePrice)
		
		// Update multi-level TP orders if this is not the first buy (DCA level > 1)
		// Note: We do this in syncAfterTrade to ensure it happens after position sync
		if bot.dcaLevel > 1 && bot.config.Strategy.AutoTPOrders {
			bot.logger.Info("🔄 DCA Level %d detected - updating TP orders for new average price", bot.dcaLevel)
			
			// Add recovery mechanism for TP update
			func() {
				defer func() {
					if r := recover(); r != nil {
						bot.logger.Error("TP update panic recovered: %v", r)
					}
				}()
				
				if err := bot.updateMultiLevelTPOrders(bot.averagePrice); err != nil {
					bot.logger.LogWarning("TP Update", "Failed to update TP orders after DCA: %v", err)
					// Continue execution - don't let TP update failure stop the bot
				}
			}()
		}
		
	} else if tradeType == "SELL" {
		// Create dedicated context for cleanup operations
		cleanupCtx := context.Background()
		
		// For sell trades, calculate realized P&L from position data before sync
		if currentPrice, err := bot.exchange.GetLatestPrice(cleanupCtx, bot.symbol); err == nil {
			if bot.averagePrice > 0 && currentPrice > 0 {
				profitPercent := (currentPrice - bot.averagePrice) / bot.averagePrice * 100
				
				// Log cycle completion
				bot.logger.LogCycleCompletion(currentPrice, bot.averagePrice, profitPercent)
			}
		}
		
		// Clean up multi-level TP orders since position is closed
		bot.tpOrderMutex.Lock()
		for orderID, tpInfo := range bot.activeTPOrders {
			// Cancel any remaining TP orders with retry logic
			if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
				bot.logger.LogWarning("TP Cleanup", "Failed to cancel TP Level %d order %s: %v", tpInfo.Level, orderID, err)
			} else {
				bot.logger.Info("🧹 Cancelled TP Level %d order %s (position closed)", tpInfo.Level, orderID)
			}
		}
		// Clear map properly instead of recreating
		for k := range bot.activeTPOrders {
			delete(bot.activeTPOrders, k)
		}
		bot.tpOrderMutex.Unlock()
		
		// Log trade execution details
		bot.logger.LogTradeExecution(tradeType, order.OrderID, order.CumExecQty, order.AvgPrice, order.CumExecValue, 0, 0, 0)
		
		// Reset internal counters after sell
		bot.dcaLevel = 0
		
		// Notify strategy of cycle completion if configured
		if bot.config.Strategy.Cycle {
			bot.strategy.OnCycleComplete()
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
	
	// Create a map of current TP orders on exchange
	exchangeTPOrders := make(map[string]*exchange.Order)
	for _, order := range orders {
		// Check if this looks like a TP order (sell side, limit order)
		if order.Side == "Sell" && order.OrderType == "Limit" {
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
	bot.tpOrderMutex.Lock()
	defer bot.tpOrderMutex.Unlock()
	
	if len(bot.activeTPOrders) == 0 {
		bot.logger.Info("✅ No active TP orders to cancel")
		return nil
	}
	
	bot.logger.Info("🧹 Canceling %d active TP orders during shutdown...", len(bot.activeTPOrders))
	
	cancelledCount := 0
	failedCount := 0
	
	for orderID, tpInfo := range bot.activeTPOrders {
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("Shutdown Cleanup", "Failed to cancel TP Level %d order %s: %v", tpInfo.Level, orderID, err)
			failedCount++
			continue
		}
		
		bot.logger.Info("❌ Cancelled TP Level %d order: %s (qty: %s)", tpInfo.Level, orderID, tpInfo.Quantity)
		cancelledCount++
	}
	
	// Clear the map
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	
	if failedCount > 0 {
		bot.logger.LogWarning("Shutdown Cleanup", "Cancelled %d TP orders, %d failed", cancelledCount, failedCount)
		fmt.Printf("⚠️ Cancelled %d TP orders, %d failed to cancel\n", cancelledCount, failedCount)
	} else {
		fmt.Printf("✅ Cancelled %d TP orders successfully\n", cancelledCount)
	}
	
	return nil
}

// parseFloat safely parses a string to float64
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
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
	bot.activeTPOrders[tpOrder.OrderID] = &TPOrderInfo{
		Level:     level, // Use next available level number
		Percent:   levelPercent,
		Quantity:  formattedQty,
		Price:     formattedPrice,
		OrderID:   tpOrder.OrderID,
		Filled:    false,
		FilledQty: "0",
	}
	bot.tpOrderMutex.Unlock()
	
	bot.logger.Info("✅ Fallback TP combined with Level %d: %s %s at $%s (%.2f%%) - Order ID: %s", 
		level, formattedQty, bot.symbol, formattedPrice, levelPercent*100, tpOrder.OrderID)
	
	return nil
}

// updateMarketDataBuffer maintains a rolling buffer of market data for regime analysis
func (bot *LiveBot) updateMarketDataBuffer(klines []types.OHLCV) {
	bot.bufferMutex.Lock()
	defer bot.bufferMutex.Unlock()
	
	// Add new candle data (use the most recent candle)
	if len(klines) > 0 {
		latestCandle := klines[len(klines)-1]
		
		// Add to buffer
		bot.marketDataBuffer = append(bot.marketDataBuffer, latestCandle)
		
		// Maintain buffer size
		if len(bot.marketDataBuffer) > bot.bufferSize {
			// Remove oldest entries to maintain buffer size
			excess := len(bot.marketDataBuffer) - bot.bufferSize
			bot.marketDataBuffer = bot.marketDataBuffer[excess:]
		}
	}
}

// detectAndUpdateRegime performs regime detection and updates bot state
func (bot *LiveBot) detectAndUpdateRegime() {
	bot.bufferMutex.RLock()
	dataBuffer := make([]types.OHLCV, len(bot.marketDataBuffer))
	copy(dataBuffer, bot.marketDataBuffer)
	bot.bufferMutex.RUnlock()
	
	// Need sufficient data for regime detection
	minDataForRegime := 250 // Same as our regime analyzer tool
	if len(dataBuffer) < minDataForRegime {
		// Not enough data yet - keep current regime as uncertain
		bot.regimeMutex.Lock()
		if bot.currentRegime == regime.RegimeUncertain {
			bot.logger.Info("Regime Detection: Insufficient data for regime analysis (%d/%d points)", len(dataBuffer), minDataForRegime)
		}
		bot.regimeMutex.Unlock()
		return
	}
	
	// Perform regime detection
	signal, err := bot.regimeDetector.DetectRegime(dataBuffer)
	if err != nil {
		bot.logger.LogWarning("Regime Detection Error", "%v", err)
		return
	}
	
	// Update regime state
	bot.regimeMutex.Lock()
	defer bot.regimeMutex.Unlock()
	
	oldRegime := bot.currentRegime
	bot.currentRegime = signal.Type
	bot.lastRegimeSignal = signal
	
	// Log regime change if occurred
	if oldRegime != signal.Type && oldRegime != regime.RegimeUncertain {
		regimeChange := &regime.RegimeChange{
			Timestamp:    signal.Timestamp,
			OldRegime:    oldRegime,
			NewRegime:    signal.Type,
			Confidence:   signal.Confidence,
			Reason:       fmt.Sprintf("Trend: %.2f, Volatility: %.2f, Noise: %.2f", signal.TrendStrength, signal.Volatility, signal.NoiseLevel),
			TriggerPrice: dataBuffer[len(dataBuffer)-1].Close,
		}
		
		// Add to history
		bot.regimeHistory = append(bot.regimeHistory, regimeChange)
		if len(bot.regimeHistory) > 50 {
			// Keep only last 50 regime changes
			bot.regimeHistory = bot.regimeHistory[1:]
		}
		
		// Log regime change
		bot.logger.Trade("🔄 REGIME CHANGE: %s → %s (Confidence: %.1f%%) | Trend: %.2f, Vol: %.2f, Noise: %.2f",
			oldRegime.String(),
			signal.Type.String(),
			signal.Confidence*100,
			signal.TrendStrength,
			signal.Volatility,
			signal.NoiseLevel,
		)
		
		// Log to console for visibility
		fmt.Printf("🔄 REGIME CHANGE: %s → %s (%.1f%% confidence)\n", 
			oldRegime.String(), signal.Type.String(), signal.Confidence*100)
			
	} else if signal.TransitionFlag {
		// Log transition detection even if regime hasn't officially changed
		bot.logger.Info("Regime Transition: Potential transition detected - confidence: %.1f%%, current: %s", signal.Confidence*100, signal.Type.String())
	}
	
	// Periodic regime status logging (every 20 detections ≈ every ~2 hours for 5m intervals)
	detectionCount := len(bot.regimeHistory) + 1
	if detectionCount%20 == 0 {
		bot.logger.Info("Regime Status: Current: %s | Confidence: %.1f%% | Trend: %.2f | Volatility: %.2f | Noise: %.2f",
			signal.Type.String(),
			signal.Confidence*100,
			signal.TrendStrength,
			signal.Volatility,
			signal.NoiseLevel,
		)
	}
}

// GetCurrentRegime returns the current market regime (thread-safe)
func (bot *LiveBot) GetCurrentRegime() regime.RegimeType {
	bot.regimeMutex.RLock()
	defer bot.regimeMutex.RUnlock()
	return bot.currentRegime
}

// GetLastRegimeSignal returns the most recent regime detection signal (thread-safe)
func (bot *LiveBot) GetLastRegimeSignal() *regime.RegimeSignal {
	bot.regimeMutex.RLock()
	defer bot.regimeMutex.RUnlock()
	if bot.lastRegimeSignal == nil {
		return nil
	}
	// Return a copy to prevent concurrent access issues
	signalCopy := *bot.lastRegimeSignal
	return &signalCopy
}

// GetRegimeHistory returns the recent regime change history (thread-safe)
func (bot *LiveBot) GetRegimeHistory() []*regime.RegimeChange {
	bot.regimeMutex.RLock()
	defer bot.regimeMutex.RUnlock()
	
	// Return a copy to prevent concurrent access issues
	historyCopy := make([]*regime.RegimeChange, len(bot.regimeHistory))
	copy(historyCopy, bot.regimeHistory)
	return historyCopy
}

// GetRegimeAnalytics returns basic analytics about regime detection performance
func (bot *LiveBot) GetRegimeAnalytics() map[string]interface{} {
	bot.regimeMutex.RLock()
	defer bot.regimeMutex.RUnlock()
	
	analytics := map[string]interface{}{
		"current_regime":        bot.currentRegime.String(),
		"total_regime_changes":  len(bot.regimeHistory),
		"buffer_size":          len(bot.marketDataBuffer),
		"detection_ready":      len(bot.marketDataBuffer) >= 250,
	}
	
	if bot.lastRegimeSignal != nil {
		analytics["last_confidence"] = bot.lastRegimeSignal.Confidence
		analytics["last_trend_strength"] = bot.lastRegimeSignal.TrendStrength
		analytics["last_volatility"] = bot.lastRegimeSignal.Volatility
		analytics["last_noise_level"] = bot.lastRegimeSignal.NoiseLevel
		analytics["last_detection"] = bot.lastRegimeSignal.Timestamp.Format("2006-01-02 15:04:05")
	}
	
	return analytics
}
