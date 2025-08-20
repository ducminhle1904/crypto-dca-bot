package bot

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/adapters"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

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
	category := determineTradingCategory(config.Exchange.Name, symbol)



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
			positionValue := parseFloat(pos.PositionValue)
			avgPrice := parseFloat(pos.AvgPrice)
			
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

	// Check for take profit
	if bot.currentPosition > 0 && bot.averagePrice > 0 {
		profitPercent := (currentPrice - bot.averagePrice) / bot.averagePrice
		if profitPercent >= bot.config.Strategy.TPPercent {
			return decision, "SELL"
		}
	}

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

	// Apply minimum quantity constraint
	if constraints.QtyStep > 0 {
		multiplier := math.Round(quantity / constraints.QtyStep)
		if multiplier < 1 {
			multiplier = 1
		}
		adjustedQuantity := multiplier * constraints.QtyStep
		
		if adjustedQuantity != quantity {
			quantity = adjustedQuantity
			amount = quantity * price
			bot.logger.Info("📏 Quantity adjusted to step size: %.6f %s", quantity, bot.symbol)
		}
	}

	// Check available balance for margin (don't assume 10x leverage)
	if amount > bot.balance {
		bot.logger.LogWarning("Insufficient balance", "Balance: $%.2f < Required: $%.2f", bot.balance, amount)
		return
	}

	// Place order
	orderParams := exchange.OrderParams{
		Category:  bot.category,
		Symbol:    bot.symbol,
		Side:      exchange.OrderSideBuy,
		Quantity:  fmt.Sprintf("%.6f", quantity),
		OrderType: exchange.OrderTypeMarket,
	}

	order, err := bot.exchange.PlaceMarketOrder(ctx, orderParams)
	if err != nil {
		bot.logger.Error("Failed to place buy order: %v", err)
		return
	}

	// Sync with exchange data instead of self-calculating
	bot.syncAfterTrade(order, "BUY")
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
		
	} else if tradeType == "SELL" {
		// For sell trades, calculate realized P&L from position data before sync
		ctx := context.Background()
		if currentPrice, err := bot.exchange.GetLatestPrice(ctx, bot.symbol); err == nil {
			if bot.averagePrice > 0 && currentPrice > 0 {
				profitPercent := (currentPrice - bot.averagePrice) / bot.averagePrice * 100
				
				// Log cycle completion
				bot.logger.LogCycleCompletion(currentPrice, bot.averagePrice, profitPercent)
			}
		}
		
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
