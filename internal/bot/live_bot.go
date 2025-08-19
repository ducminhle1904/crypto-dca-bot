package bot

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/adapters"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// LiveBot represents the live trading bot with exchange interface support
type LiveBot struct {
	config   *config.LiveBotConfig
	exchange exchange.LiveTradingExchange
	strategy *strategy.EnhancedDCAStrategy
	
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



	bot := &LiveBot{
		config:   config,
		exchange: exchangeInstance,
		symbol:   symbol,
		interval: interval,
		category: category,
		balance:  config.Risk.InitialBalance,
		stopChan: make(chan struct{}),
	}

	// Initialize strategy
	if err := bot.initializeStrategy(); err != nil {
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
		log.Printf("⚠️ Could not sync account balance: %v", err)
		log.Printf("💡 Using config balance: $%.2f", bot.balance)
		log.Printf("🔧 Check your API key permissions")
	}

	// Check for existing position and sync bot state
	if err := bot.syncExistingPosition(); err != nil {
		log.Printf("⚠️ Could not sync existing position: %v", err)
	}

	// Print startup information
	bot.printStartupInfo()
	bot.printBotConfiguration()

	// Start the main trading loop
	go bot.tradingLoop()

	return nil
}

// Stop gracefully stops the bot
func (bot *LiveBot) Stop() {
	if bot.running {
		bot.running = false
		
		// Close any open positions before stopping
		if err := bot.closeOpenPositions(); err != nil {
			log.Printf("⚠️ Error closing positions during shutdown: %v", err)
		}
		
		// Disconnect from exchange
		if err := bot.exchange.Disconnect(); err != nil {
			log.Printf("⚠️ Error disconnecting from exchange: %v", err)
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
		switch strings.ToLower(indName) {
		case "rsi":
			rsi := indicators.NewRSI(bot.config.Strategy.RSI.Period)
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
				
				fmt.Printf("🔄 Synced with existing position:\n")
				fmt.Printf("   Position Value: $%.2f\n", positionValue)
				fmt.Printf("   Entry Price: $%.2f\n", avgPrice)
				fmt.Printf("   Position Size: %s %s\n", pos.Size, pos.Symbol)
				fmt.Printf("   Unrealized P&L: $%s\n", pos.UnrealisedPnl)
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
	
	// Wait for next candle close
	waitDuration := bot.getTimeUntilNextCandle()
	fmt.Printf("⏰ Waiting %.0f seconds for next %s candle close...\n", waitDuration.Seconds(), bot.interval)
	time.Sleep(waitDuration)
	
	// Run initial check
	fmt.Printf("🕐 Candle closed - running initial check\n")
	bot.checkAndTrade()

	// Create ticker for regular checks
	ticker := time.NewTicker(intervalDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fmt.Printf("🕐 Candle closed - checking market\n")
			bot.checkAndTrade()
		case <-bot.stopChan:
			return
		}
	}
}

// checkAndTrade performs market analysis and executes trades
func (bot *LiveBot) checkAndTrade() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Error in trading loop: %v", r)
		}
	}()

	ctx := context.Background()

	// Refresh account balance
	if err := bot.syncAccountBalance(); err != nil {
		log.Printf("⚠️ Could not refresh balance: %v", err)
	}

	// Sync position data
	if err := bot.syncPositionData(); err != nil {
		log.Printf("⚠️ Could not sync position data: %v", err)
	}

	// Get current market price
	currentPrice, err := bot.exchange.GetLatestPrice(ctx, bot.symbol)
	if err != nil {
		log.Printf("❌ Failed to get current price: %v", err)
		return
	}

	// Get recent klines for analysis
	klines, err := bot.getRecentKlines()
	if err != nil {
		log.Printf("❌ Failed to get recent klines: %v", err)
		return
	}

	// Use a more flexible minimum data requirement
	// Need at least 2x the longest indicator period for basic reliability
	minRequiredDataPoints := 70 // Minimum required for multi-indicator strategy
	if bot.config.Strategy.WindowSize < minRequiredDataPoints {
		minRequiredDataPoints = bot.config.Strategy.WindowSize
	}
	
	if len(klines) < minRequiredDataPoints {
		log.Printf("⚠️ Not enough data points (%d < %d minimum required)", len(klines), minRequiredDataPoints)
		return
	}
	
	// If we have less than the configured window size but more than minimum, proceed with warning
	if len(klines) < bot.config.Strategy.WindowSize {
		log.Printf("⚠️ Using %d data points (less than configured %d, but sufficient for analysis)", len(klines), bot.config.Strategy.WindowSize)
	}

	// Analyze market conditions
	action := bot.analyzeMarket(klines, currentPrice)

	// Log current status
	bot.logStatus(currentPrice, action)

	// Execute trading action
	if action != "HOLD" {
		if bot.exchange.IsDemo() {
			fmt.Printf("🧪 DEMO MODE: Executing %s at $%.2f (paper trading)\n", action, currentPrice)
		} else {
			fmt.Printf("💰 LIVE MODE: Executing %s at $%.2f (real money)\n", action, currentPrice)
		}
		bot.executeTrade(action, currentPrice)
	}
}

// getRecentKlines retrieves recent market data
func (bot *LiveBot) getRecentKlines() ([]types.OHLCV, error) {
	ctx := context.Background()
	
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
		return nil, err
	}

	// If we got less data than expected, try with a smaller request
	if len(klines) < 70 && requestLimit > 100 {
		log.Printf("⚠️ Got only %d klines, retrying with smaller request", len(klines))
		params.Limit = 100
		klines, err = bot.exchange.GetKlines(ctx, params)
		if err != nil {
			return nil, err
		}
	}

	// Return all available data (we'll handle the minimum check in checkAndTrade)
	return klines, nil
}

// analyzeMarket performs technical analysis to determine trading action
func (bot *LiveBot) analyzeMarket(klines []types.OHLCV, currentPrice float64) string {
	// Use strategy to analyze market conditions
	decision, err := bot.strategy.ShouldExecuteTrade(klines)
	if err != nil {
		log.Printf("❌ Strategy error: %v", err)
		return "HOLD"
	}
	
	if decision.Action == strategy.ActionBuy {
		return "BUY"
	}

	// Check for take profit
	if bot.currentPosition > 0 && bot.averagePrice > 0 {
		profitPercent := (currentPrice - bot.averagePrice) / bot.averagePrice
		if profitPercent >= bot.config.Strategy.TPPercent {
			return "SELL"
		}
	}

	return "HOLD"
}

// executeTrade executes the trading action
func (bot *LiveBot) executeTrade(action string, price float64) {
	switch action {
	case "BUY":
		bot.executeBuy(price)
	case "SELL":
		bot.executeSell(price)
	}
}

// executeBuy executes a buy order using exchange interface
func (bot *LiveBot) executeBuy(price float64) {
	ctx := context.Background()

	// Calculate DCA amount based on level
	amount := bot.config.Strategy.BaseAmount
	if bot.dcaLevel > 0 {
		multiplier := 1.0 + float64(bot.dcaLevel)*0.5 // Increase by 50% each level
		if multiplier > bot.config.Strategy.MaxMultiplier {
			multiplier = bot.config.Strategy.MaxMultiplier
		}
		amount *= multiplier
	}

	// Get trading constraints
	constraints, err := bot.exchange.GetTradingConstraints(ctx, bot.category, bot.symbol)
	if err != nil {
		log.Printf("⚠️ Could not get trading constraints: %v", err)
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
			fmt.Printf("📏 Quantity adjusted to step size: %.6f %s\n", quantity, bot.symbol)
		}
	}

	// Check margin requirements (assuming 10x leverage)
	marginRequired := amount / 10
	if marginRequired > bot.balance {
		log.Printf("⚠️ Insufficient margin: $%.2f < $%.2f", bot.balance, marginRequired)
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
		log.Printf("❌ Failed to place buy order: %v", err)
		return
	}

	// Update bot state based on order execution
	bot.updateStateAfterBuy(order, marginRequired, amount)
}

// executeSell executes a sell order to close position
func (bot *LiveBot) executeSell(price float64) {
	if bot.currentPosition <= 0 {
		return
	}

	ctx := context.Background()
	
	// Calculate quantity to close entire position
	assetQuantity := bot.currentPosition / price
	
	orderParams := exchange.OrderParams{
		Category:  bot.category,
		Symbol:    bot.symbol,
		Side:      exchange.OrderSideSell,
		Quantity:  fmt.Sprintf("%.6f", assetQuantity),
		OrderType: exchange.OrderTypeMarket,
	}

	order, err := bot.exchange.PlaceMarketOrder(ctx, orderParams)
	if err != nil {
		log.Printf("❌ Failed to place sell order: %v", err)
		return
	}

	// Update bot state after sale
	bot.updateStateAfterSell(order, price)
}

// Helper functions continue in next part...
func (bot *LiveBot) updateStateAfterBuy(order *exchange.Order, marginUsed, notionalValue float64) {
	// Update position based on order execution
	executedValue := parseFloat(order.CumExecValue)
	if executedValue == 0 {
		executedValue = notionalValue // Fallback
	}
	
	avgPrice := parseFloat(order.AvgPrice)
	if avgPrice == 0 {
		avgPrice = notionalValue / parseFloat(order.CumExecQty) // Calculate from executed data
	}

	// Update bot state
	if bot.currentPosition == 0 {
		bot.averagePrice = avgPrice
		bot.currentPosition = executedValue
	} else {
		// Calculate weighted average
		totalNotional := bot.currentPosition + executedValue
		weightedPrice := (bot.currentPosition*bot.averagePrice + executedValue*avgPrice) / totalNotional
		bot.currentPosition = totalNotional
		bot.averagePrice = weightedPrice
	}

	bot.totalInvested += executedValue
	bot.balance -= marginUsed
	bot.dcaLevel++

	fmt.Printf("✅ BUY ORDER EXECUTED\n")
	fmt.Printf("   Order ID: %s\n", order.OrderID)
	fmt.Printf("   Quantity: %s %s\n", order.CumExecQty, bot.symbol)
	fmt.Printf("   Price: $%.2f\n", avgPrice)
	fmt.Printf("   Notional: $%.2f\n", executedValue)
	fmt.Printf("   Total Position: $%.2f\n", bot.currentPosition)
	fmt.Printf("   DCA Level: %d\n", bot.dcaLevel)
}

func (bot *LiveBot) updateStateAfterSell(order *exchange.Order, price float64) {
	// Calculate P&L
	priceRatio := price / bot.averagePrice
	saleValue := bot.totalInvested * priceRatio
	profit := saleValue - bot.totalInvested
	profitPercent := (price - bot.averagePrice) / bot.averagePrice * 100

	fmt.Printf("✅ SELL ORDER EXECUTED\n")
	fmt.Printf("   Order ID: %s\n", order.OrderID)
	fmt.Printf("   Quantity: %s %s\n", order.CumExecQty, bot.symbol)
	fmt.Printf("   Price: $%.2f\n", price)
	fmt.Printf("   Profit: $%.2f (%.2f%%)\n", profit, profitPercent)

	// Reset position state
	marginReleased := bot.totalInvested / 10 // Original margin used
	bot.balance += marginReleased + profit   // Margin + P&L
	bot.currentPosition = 0
	bot.averagePrice = 0
	bot.totalInvested = 0
	bot.dcaLevel = 0
	
	// Refresh balance to sync with exchange
	if err := bot.syncAccountBalance(); err != nil {
		log.Printf("⚠️ Could not refresh balance after sale: %v", err)
	}
	
	// Notify strategy of cycle completion if configured
	if bot.config.Strategy.Cycle {
		bot.strategy.OnCycleComplete()
	}
}

// Continue with remaining helper functions in next message...
