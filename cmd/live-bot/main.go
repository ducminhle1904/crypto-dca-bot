package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/bybit"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/joho/godotenv"
)

// BotConfig represents the configuration loaded from config files
type BotConfig struct {
	DataFile        string    `json:"data_file"`
	Symbol          string    `json:"symbol"`
	InitialBalance  float64   `json:"initial_balance"`
	Commission      float64   `json:"commission"`
	WindowSize      int       `json:"window_size"`
	BaseAmount      float64   `json:"base_amount"`
	MaxMultiplier   float64   `json:"max_multiplier"`
	PriceThreshold  float64   `json:"price_threshold"`
	RSIPeriod       int       `json:"rsi_period"`
	RSIOversold     float64   `json:"rsi_oversold"`
	RSIOverbought   float64   `json:"rsi_overbought"`
	MACDFast        int       `json:"macd_fast"`
	MACDSlow        int       `json:"macd_slow"`
	MACDSignal      int       `json:"macd_signal"`
	BBPeriod        int       `json:"bb_period"`
	BBStdDev        float64   `json:"bb_std_dev"`
	EMAPeriod       int       `json:"ema_period"`
	Indicators      []string  `json:"indicators"`
	TPPercent       float64   `json:"tp_percent"`
	Cycle           bool      `json:"cycle"`
}

// LiveBot represents the live trading bot
type LiveBot struct {
	config      *BotConfig
	bybitClient *bybit.Client
	strategy    *strategy.EnhancedDCAStrategy
	symbol      string
	interval    string
	category    string
	running     bool
	stopChan    chan struct{}
	
	// Trading state
	currentPosition float64
	averagePrice    float64
	totalInvested   float64
	balance         float64
	dcaLevel        int
}

func main() {
	var (
		configFile = flag.String("config", "", "Configuration file (e.g., btc_5.json)")
		demo       = flag.Bool("demo", true, "Use demo trading environment - paper trading (default: true)")
		envFile    = flag.String("env", ".env", "Environment file path (default: .env)")
	)
	flag.Parse()

	if *configFile == "" {
		log.Fatal("Please specify a config file with -config flag")
	}

	// Load environment variables from .env file
	if err := loadEnvFile(*envFile); err != nil {
		log.Printf("Warning: Could not load .env file (%v), checking environment variables...", err)
	}

	// Load configuration
	config, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Extract symbol and interval from config file name and data
	symbol, interval, category := extractTradingParams(*configFile, config)
	
	fmt.Println("🚀 Enhanced DCA Live Bot Starting...")
	fmt.Println()
	printStartupTable(symbol, interval, category, getEnvironmentString(*demo))

	// Create Bybit client
	bybitConfig := bybit.Config{
		APIKey:    os.Getenv("BYBIT_API_KEY"),
		APISecret: os.Getenv("BYBIT_API_SECRET"),
		Testnet:   false, // Always use mainnet infrastructure
		Demo:      *demo,  // Use demo mode for paper trading
	}

	if bybitConfig.APIKey == "" || bybitConfig.APISecret == "" {
		log.Fatal("Please set BYBIT_API_KEY and BYBIT_API_SECRET in .env file or environment variables")
	}



	bybitClient := bybit.NewClient(bybitConfig)

	// Create live bot
	bot := &LiveBot{
		config:      config,
		bybitClient: bybitClient,
		symbol:      symbol,
		interval:    interval,
		category:    category,
		balance:     config.InitialBalance,
		stopChan:    make(chan struct{}),
	}

	// Initialize strategy
	if err := bot.initializeStrategy(); err != nil {
		log.Fatalf("Failed to initialize strategy: %v", err)
	}

	// Sync with real account balance if credentials are provided
	if err := bot.syncAccountBalance(); err != nil {
		log.Printf("⚠️ Could not sync account balance: %v", err)
		log.Printf("💡 Using config balance: $%.2f", bot.balance)
		log.Printf("🔧 Tip: Ensure your Bybit API key has 'Account' permissions enabled")
		
		// Check if it's an authentication error and provide specific guidance
		if bybit.IsAuthenticationError(err) {
			log.Printf("🔑 Authentication issue detected. Please check:")
			log.Printf("   - API key and secret are correct")
			log.Printf("   - API key has 'Account' permissions")
			log.Printf("   - Using the correct environment (demo mode: %v)", *demo)
		}
	}

	// Start the bot
	if err := bot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sigChan:
		fmt.Println("\n🛑 Shutdown signal received...")
	case <-bot.stopChan:
		fmt.Println("\n🛑 Bot stopped...")
	}

	bot.Stop()
	fmt.Println("✅ Bot stopped successfully")
}

// printStartupTable prints the initial startup information in a table format
func printStartupTable(symbol, interval, category, environment string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("BOT INITIALIZATION")
	t.SetStyle(table.StyleRounded)
	
	t.AppendRows([]table.Row{
		{"📊 Symbol", symbol},
		{"⏰ Interval", interval},
		{"🏪 Category", category},
		{"🔧 Environment", environment},
	})
	
	// Configure column settings
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMin: 15, WidthMax: 15, Align: text.AlignLeft},
		{Number: 2, WidthMin: 30, WidthMax: 35, Align: text.AlignLeft},
	})
	
	t.Render()
	fmt.Println()
}

// printBotConfigTable prints the bot configuration in a table format  
func printBotConfigTable(bot *LiveBot, minQtyInfo, minNotionalInfo string, indicators []string, tradingMode string) {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("BOT CONFIGURATION")
	t.SetStyle(table.StyleRounded)
	
	// Trading Parameters Section
	t.AppendRows([]table.Row{
		{"💰 Initial Balance", fmt.Sprintf("$%.2f", bot.balance)},
		{"📈 Base DCA Amount", fmt.Sprintf("$%.2f", bot.config.BaseAmount)},
		{"🔄 Max Multiplier", fmt.Sprintf("%.2f", bot.config.MaxMultiplier)},
		{"📊 Price Threshold", fmt.Sprintf("%.2f%%", bot.config.PriceThreshold*100)},
		{"🎯 Take Profit", fmt.Sprintf("%.2f%%", bot.config.TPPercent*100)},
	})
	
	// Add separator
	t.AppendSeparator()
	
	// Order Constraints Section
	t.AppendRows([]table.Row{
		{"📏 Min Order Qty", minQtyInfo},
		{"💵 Min Notional", minNotionalInfo},
	})
	
	// Add separator
	t.AppendSeparator()
	
	// Technical & Mode Section
	indicatorStr := strings.Join(indicators, ", ")
	t.AppendRows([]table.Row{
		{"📊 Indicators", indicatorStr},
		{"🚨 Trading Mode", tradingMode},
	})
	
	// Configure column settings
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMin: 18, WidthMax: 18, Align: text.AlignLeft},
		{Number: 2, WidthMin: 25, WidthMax: 40, Align: text.AlignLeft},
	})
	
	t.Render()
	fmt.Println()
}

func getEnvironmentString(demo bool) string {
	if demo {
		return "demo (paper trading on mainnet)"
	} else {
		return "live trading on mainnet"
	}
}

func loadEnvFile(envFile string) error {
	// Load .env file if it exists
	if _, err := os.Stat(envFile); err == nil {
		return godotenv.Load(envFile)
	}
	return fmt.Errorf("env file %s not found", envFile)
}

func loadConfig(configFile string) (*BotConfig, error) {
	// If config file doesn't contain path separators, look in configs/ directory
	if !strings.ContainsAny(configFile, "/\\") {
		configFile = filepath.Join("configs", configFile)
	}

	// Add .json extension if not present
	if !strings.HasSuffix(configFile, ".json") {
		configFile += ".json"
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
	}

	var config BotConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func extractTradingParams(configFile string, config *BotConfig) (symbol, interval, category string) {
	// Get base filename without extension
	basename := filepath.Base(configFile)
	if ext := filepath.Ext(basename); ext != "" {
		basename = strings.TrimSuffix(basename, ext)
	}

	// Extract symbol from config
	symbol = config.Symbol

	// Extract interval from filename (e.g., btc_15m.json -> 15m)
	parts := strings.Split(basename, "_")
	if len(parts) >= 2 {
		intervalPart := parts[len(parts)-1] // Last part should be interval
		
		// Convert common intervals to Bybit format
		switch intervalPart {
		case "1m":
			interval = "1"
		case "3m":
			interval = "3"
		case "5m":
			interval = "5"
		case "15m":
			interval = "15"
		case "30m":
			interval = "30"
		case "1h":
			interval = "60"
		case "4h":
			interval = "240"
		case "1d":
			interval = "D"
		default:
			// Try to extract number from string like "5m" -> "5"
			if strings.HasSuffix(intervalPart, "m") {
				if num := strings.TrimSuffix(intervalPart, "m"); num != "" {
					interval = num
				}
			} else if strings.HasSuffix(intervalPart, "h") {
				if num := strings.TrimSuffix(intervalPart, "h"); num != "" {
					if hours, err := strconv.Atoi(num); err == nil {
						interval = strconv.Itoa(hours * 60) // Convert to minutes
					}
				}
			} else {
				interval = intervalPart // Use as-is
			}
		}
	}

	// Default values if extraction fails
	if interval == "" {
		interval = "60" // Default to 1 hour
	}

	// Determine category from data file path or default to linear
	if strings.Contains(config.DataFile, "spot") {
		category = "spot"
	} else if strings.Contains(config.DataFile, "linear") {
		category = "linear"
	} else if strings.Contains(config.DataFile, "inverse") {
		category = "inverse"
	} else {
		category = "linear" // Default to linear futures
	}

	return symbol, interval, category
}

func (bot *LiveBot) syncAccountBalance() error {
	ctx := context.Background()
	
	// Determine the appropriate account type based on trading category
	var accountType bybit.AccountType
	switch bot.category {
	case "spot":
		accountType = bybit.AccountTypeUnified // Spot trading usually uses UNIFIED account
	case "linear", "inverse":
		accountType = bybit.AccountTypeUnified // Futures trading uses UNIFIED account
	default:
		accountType = bybit.AccountTypeUnified // Default to UNIFIED
	}
	
	// Determine the base currency for balance checking
	var baseCurrency string
	switch bot.category {
	case "spot":
		baseCurrency = "USDT" // For spot trading, check USDT balance
	case "linear":
		baseCurrency = "USDT" // For linear futures, check USDT balance
	case "inverse":
		// For inverse futures, we'd check BTC balance for BTCUSD, but let's use USDT for simplicity
		baseCurrency = "USDT"
	default:
		baseCurrency = "USDT"
	}
	
	// Get the tradable balance for the base currency
	realBalance, err := bot.bybitClient.GetTradableBalance(ctx, accountType, baseCurrency)
	if err != nil {
		return fmt.Errorf("failed to fetch account balance: %w", err)
	}
	
	// Update bot balance with real account balance
	bot.balance = realBalance
	
	return nil
}

func (bot *LiveBot) refreshBalance() error {
	ctx := context.Background()
	
	// Determine account type and currency (same logic as syncAccountBalance)
	var accountType bybit.AccountType
	var baseCurrency string
	
	switch bot.category {
	case "spot":
		accountType = bybit.AccountTypeUnified
		baseCurrency = "USDT"
	case "linear":
		accountType = bybit.AccountTypeUnified
		baseCurrency = "USDT"
	case "inverse":
		accountType = bybit.AccountTypeUnified
		baseCurrency = "USDT"
	default:
		accountType = bybit.AccountTypeUnified
		baseCurrency = "USDT"
	}
	
	// Get current balance
	realBalance, err := bot.bybitClient.GetTradableBalance(ctx, accountType, baseCurrency)
	if err != nil {
		return fmt.Errorf("failed to refresh balance: %w", err)
	}
	
	// Update bot balance
	bot.balance = realBalance
	return nil
}

func (bot *LiveBot) initializeStrategy() error {
	// Create strategy
	bot.strategy = strategy.NewEnhancedDCAStrategy(bot.config.BaseAmount)
	bot.strategy.SetPriceThreshold(bot.config.PriceThreshold)

	// Create and add indicators based on config
	for _, indName := range bot.config.Indicators {
		switch strings.ToLower(indName) {
		case "rsi":
			rsi := indicators.NewRSI(bot.config.RSIPeriod)
			bot.strategy.AddIndicator(rsi)
		case "macd":
			macd := indicators.NewMACD(
				bot.config.MACDFast,
				bot.config.MACDSlow,
				bot.config.MACDSignal,
			)
			bot.strategy.AddIndicator(macd)
		case "bb", "bollinger":
			bb := indicators.NewBollingerBands(
				bot.config.BBPeriod,
				bot.config.BBStdDev,
			)
			bot.strategy.AddIndicator(bb)
		case "ema":
			ema := indicators.NewEMA(bot.config.EMAPeriod)
			bot.strategy.AddIndicator(ema)
		case "sma":
			sma := indicators.NewSMA(bot.config.EMAPeriod)
			bot.strategy.AddIndicator(sma)
		}
	}
	
	return nil
}

func (bot *LiveBot) Start() error {
	bot.running = true

	// Check for existing position on startup and sync bot state
	if err := bot.syncExistingPosition(); err != nil {
		log.Printf("⚠️ Could not sync existing position: %v", err)
	}

	// Get minimum lot size info for the table
	var minQtyInfo string
	var minNotionalInfo string
	ctx := context.Background()
	minQty, _, _, err := bot.bybitClient.GetInstrumentManager().GetQuantityConstraints(ctx, bot.category, bot.symbol)
	if err != nil {
		minQtyInfo = "⚠️ Could not fetch"
		minNotionalInfo = "⚠️ Could not fetch"
	} else {
		currentPrice, priceErr := bot.getCurrentPrice()
		if priceErr != nil {
			minQtyInfo = fmt.Sprintf("%.6f %s", minQty, bot.symbol)
			minNotionalInfo = "⚠️ Price unavailable"
		} else {
			minNotionalValue := minQty * currentPrice
			minQtyInfo = fmt.Sprintf("%.6f %s", minQty, bot.symbol)
			minNotionalInfo = fmt.Sprintf("~$%.2f", minNotionalValue)
		}
	}

	// Get active indicators list
	indicators := bot.config.Indicators
	if len(indicators) == 0 {
		indicators = []string{"none"}
	}
	
	// Trading mode
	var tradingMode string
	if bot.bybitClient.IsDemo() {
		tradingMode = "🧪 DEMO MODE (Paper Trading)"
	} else {
		tradingMode = "💰 LIVE TRADING MODE (Real Money!)"
	}

	// Print configuration table
	printBotConfigTable(bot, minQtyInfo, minNotionalInfo, indicators, tradingMode)

	// Start the main trading loop
	go bot.tradingLoop()

	return nil
}

// syncExistingPosition syncs bot state with any existing position on Bybit
func (bot *LiveBot) syncExistingPosition() error {
	position, err := bot.getCurrentPosition()
	if err != nil {
		return fmt.Errorf("failed to get existing position: %w", err)
	}
	
	if position == nil {
		// No existing position - initialize to zero state
		bot.currentPosition = 0
		bot.averagePrice = 0
		bot.totalInvested = 0
		bot.dcaLevel = 0
		fmt.Printf("✅ No existing position found - starting fresh\n")
		return nil
	}
	
	// Sync with existing position
	positionValue := parseFloat(position.PositionValue)
	avgPrice := parseFloat(position.AvgPrice)
	
	if positionValue > 0 && avgPrice > 0 {
		bot.currentPosition = positionValue
		bot.averagePrice = avgPrice
		bot.totalInvested = positionValue // For futures, this tracks notional invested
		bot.dcaLevel = 1 // Assume at least level 1 if position exists
		
		fmt.Printf("🔄 Synced with existing position:\n")
		fmt.Printf("   Position Value: $%.2f\n", positionValue)
		fmt.Printf("   Entry Price: $%.2f\n", avgPrice)
		fmt.Printf("   Position Size: %s %s\n", position.Size, position.Symbol)
		fmt.Printf("   Unrealized P&L: $%s\n", position.UnrealisedPnl)
	}
	
	return nil
}

// syncPositionData syncs bot internal state with real Bybit position data
func (bot *LiveBot) syncPositionData() error {
	position, err := bot.getCurrentPosition()
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}
	
	if position == nil {
		// No position exists - ensure bot state reflects this
		if bot.currentPosition > 0 {
			// Bot thinks it has a position but Bybit says no - reset state
			bot.currentPosition = 0
			bot.averagePrice = 0
			bot.totalInvested = 0
			bot.dcaLevel = 0
		}
		return nil
	}
	
	// Position exists - sync all relevant data
	positionValue := parseFloat(position.PositionValue)
	avgPrice := parseFloat(position.AvgPrice)
	positionSize := parseFloat(position.Size)
	unrealizedPnL := parseFloat(position.UnrealisedPnl)
	markPrice := parseFloat(position.MarkPrice)
	
	// Update bot state with real position data
	if positionValue > 0 && avgPrice > 0 {
		bot.currentPosition = positionValue
		bot.averagePrice = avgPrice
		bot.totalInvested = positionValue // For futures, this tracks notional invested
		
		// Estimate DCA level based on position size vs base amount
		if bot.config.BaseAmount > 0 {
			estimatedLevel := int(positionValue / bot.config.BaseAmount)
			if estimatedLevel > bot.dcaLevel {
				bot.dcaLevel = estimatedLevel
			}
		}
		
		// Store additional position info for potential use
		_ = positionSize    // Position quantity
		_ = unrealizedPnL   // Current P&L
		_ = markPrice       // Current mark price
	}
	
	return nil
}

func (bot *LiveBot) Stop() {
	if bot.running {
		bot.running = false
		
		// Close any open positions before stopping
		if err := bot.closeOpenPositions(); err != nil {
			log.Printf("⚠️ Error closing positions during shutdown: %v", err)
		}
		
		close(bot.stopChan)
	}
}

// closeOpenPositions closes all open positions using real position data from API
func (bot *LiveBot) closeOpenPositions() error {
	fmt.Printf("🔍 Checking for open positions to close...\n")
	
	// Get current position from API
	position, err := bot.getCurrentPosition()
	if err != nil {
		return fmt.Errorf("failed to get current position: %w", err)
	}
	
	if position == nil {
		fmt.Printf("✅ No open positions to close\n")
		bot.currentPosition = 0
		return nil
	}
	
	positionSize := parseFloat(position.Size)
	if positionSize <= 0 {
		fmt.Printf("✅ No open positions to close (size: %s)\n", position.Size)
		bot.currentPosition = 0
		return nil
	}
	
	fmt.Printf("🚨 Closing position: %s %s (value: $%s)\n", position.Size, position.Symbol, position.PositionValue)
	
	// Get current price for P&L calculation
	currentPrice, err := bot.getCurrentPrice()
	if err != nil {
		log.Printf("⚠️ Could not get current price for P&L calculation: %v", err)
		currentPrice = parseFloat(position.MarkPrice) // Use mark price as fallback
	}
	
	// Place market sell order to close entire position
	ctx := context.Background()
	quantityStr := position.Size // Use exact position size from API
	
	order, err := bot.bybitClient.PlaceMarketOrder(ctx, bot.category, bot.symbol, bybit.OrderSideSell, quantityStr)
	if err != nil {
		return fmt.Errorf("failed to place sell order: %w", err)
	}
	
	// Calculate P&L based on real position data
	avgPrice := parseFloat(position.AvgPrice)
	positionValue := parseFloat(position.PositionValue)
	saleValue := positionSize * currentPrice
	profit := saleValue - positionValue
	profitPercent := (profit / positionValue) * 100
	
	fmt.Printf("🔄 POSITION CLOSED (SHUTDOWN)\n")
	fmt.Printf("   Order ID: %s\n", order.OrderID)
	fmt.Printf("   Quantity: %s %s\n", position.Size, bot.symbol)
	fmt.Printf("   Entry Price: $%.2f\n", avgPrice)
	fmt.Printf("   Exit Price: $%.2f\n", currentPrice)
	fmt.Printf("   P&L: $%.2f (%.2f%%)\n", profit, profitPercent)
	fmt.Printf("   Position Value: $%.2f\n", positionValue)
	
	// Reset bot state
	bot.currentPosition = 0
	bot.averagePrice = 0
	bot.totalInvested = 0
	bot.dcaLevel = 0
	
	// Refresh balance after closing position
	if err := bot.refreshBalance(); err != nil {
		log.Printf("⚠️ Could not refresh balance after closing position: %v", err)
	}
	
	// Notify strategy of cycle completion if configured
	if bot.config.Cycle {
		bot.strategy.OnCycleComplete()
	}
	
	return nil
}

func (bot *LiveBot) tradingLoop() {
	// Calculate interval duration
	intervalDuration := bot.getIntervalDuration()
	
	// Calculate time to wait until next candle close
	waitDuration := bot.getTimeUntilNextCandle()
	
	fmt.Printf("⏰ Waiting %.0f seconds for next %s candle close...\n", waitDuration.Seconds(), bot.interval)
	
	// Wait for the next candle close before starting
	time.Sleep(waitDuration)
	
	// Run initial check at candle close
	fmt.Printf("🕐 Candle closed - running initial check\n")
	bot.checkAndTrade()

	// Create ticker for regular checks aligned with candle closes
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

func (bot *LiveBot) checkAndTrade() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Error in trading loop: %v", r)
		}
	}()

	// Refresh account balance to get exact current balance
	if err := bot.refreshBalance(); err != nil {
		log.Printf("⚠️ Could not refresh balance: %v", err)
		// Continue with existing balance - don't stop trading
	}

	// Sync position data with Bybit to keep bot state accurate
	if err := bot.syncPositionData(); err != nil {
		log.Printf("⚠️ Could not sync position data: %v", err)
		// Continue with existing data - don't stop trading
	}

	// Get current market data
	currentPrice, err := bot.getCurrentPrice()
	if err != nil {
		log.Printf("❌ Failed to get current price: %v", err)
		return
	}

	// Get recent klines for indicator analysis
	klines, err := bot.getRecentKlines()
	if err != nil {
		log.Printf("❌ Failed to get recent klines: %v", err)
		return
	}

	if len(klines) < bot.config.WindowSize {
		log.Printf("⚠️ Not enough data points (%d < %d)", len(klines), bot.config.WindowSize)
		return
	}

	// Analyze with strategy
	action := bot.analyzeMarket(klines, currentPrice)

	// Log current status
	bot.logStatus(currentPrice, action)

	// Execute trading action based on demo mode
	if action != "HOLD" {
		if bot.bybitClient.IsDemo() {
			fmt.Printf("🧪 DEMO MODE: Executing %s at $%.2f (paper trading)\n", action, currentPrice)
		} else {
			fmt.Printf("💰 LIVE MODE: Executing %s at $%.2f (real money)\n", action, currentPrice)
		}
		bot.executeTrade(action, currentPrice)
	}
}

func (bot *LiveBot) getCurrentPrice() (float64, error) {
	ctx := context.Background()
	price, err := bot.bybitClient.GetLatestPrice(ctx, bot.category, bot.symbol)
	if err != nil {
		return 0, err
	}
	return price, nil
}

func (bot *LiveBot) getRecentKlines() ([]types.OHLCV, error) {
	// Get recent klines - request more than needed to ensure we have enough after processing
	ctx := context.Background()
	params := bybit.KlineParams{
		Category: bot.category,
		Symbol:   bot.symbol,
		Interval: bybit.KlineInterval(bot.interval),
		Limit:    bot.config.WindowSize + 50,
	}
	
	klines, err := bot.bybitClient.GetKlines(ctx, params)
	if err != nil {
		return nil, err
	}

	// Convert to OHLCV format
	var marketData []types.OHLCV
	for _, kline := range klines {
		marketData = append(marketData, types.OHLCV{
			Timestamp: kline.StartTime,
			Open:      kline.OpenPrice,
			High:      kline.HighPrice,
			Low:       kline.LowPrice,
			Close:     kline.ClosePrice,
			Volume:    kline.Volume,
		})
	}

	// Return the most recent data up to window size
	if len(marketData) > bot.config.WindowSize {
		return marketData[len(marketData)-bot.config.WindowSize:], nil
	}
	return marketData, nil
}

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
		if profitPercent >= bot.config.TPPercent {
			return "SELL"
		}
	}

	return "HOLD"
}

func (bot *LiveBot) executeTrade(action string, price float64) {
	switch action {
	case "BUY":
		bot.executeBuy(price)
	case "SELL":
		bot.executeSell(price)
	}
}

func (bot *LiveBot) executeBuy(price float64) {
	// Calculate DCA amount based on level
	amount := bot.config.BaseAmount
	if bot.dcaLevel > 0 {
		multiplier := 1.0 + float64(bot.dcaLevel)*0.5 // Increase by 50% each level
		if multiplier > bot.config.MaxMultiplier {
			multiplier = bot.config.MaxMultiplier
		}
		amount *= multiplier
	}

	// Get instrument constraints to check minimum order quantity
	ctx := context.Background()
	minQty, _, _, err := bot.bybitClient.GetInstrumentManager().GetQuantityConstraints(ctx, bot.category, bot.symbol)
	if err != nil {
		log.Printf("⚠️ Could not get instrument constraints: %v", err)
		minQty = 0 // Fallback to no minimum
	}

	quantity := amount / price
	originalNotional := amount

	// Apply minimum quantity constraint and step size
	if minQty > 0 {
		// Round to nearest multiple of minQty (step size)
		multiplier := math.Round(quantity / minQty)
		
		// Ensure at least 1 step (minimum quantity)
		if multiplier < 1 {
			multiplier = 1
		}
		
		adjustedQuantity := multiplier * minQty
		
		if adjustedQuantity != quantity {
			quantity = adjustedQuantity
			amount = quantity * price // Recalculate notional based on step-adjusted quantity
			fmt.Printf("📏 Quantity adjusted to step size: %.6f %s (step: %.6f)\n", quantity, bot.symbol, minQty)
		}
	}

	// For 10x leverage, we only need margin = final amount / 10
	marginRequired := amount / 10
	if marginRequired > bot.balance {
		if originalNotional != amount {
			log.Printf("⚠️ Insufficient margin: $%.2f < $%.2f (required for $%.2f position after min lot adjustment)", bot.balance, marginRequired, amount)
		} else {
			log.Printf("⚠️ Insufficient margin: $%.2f < $%.2f (required for $%.2f position)", bot.balance, marginRequired, amount)
		}
		return
	}

	quantityStr := fmt.Sprintf("%.6f", quantity)

	// Execute buy order
	order, err := bot.bybitClient.PlaceMarketOrder(ctx, bot.category, bot.symbol, bybit.OrderSideBuy, quantityStr)
	if err != nil {
		log.Printf("❌ Failed to place buy order: %v", err)
		return
	}

	// Initialize variables for position tracking
	var actualNotionalPosition, actualExecutionPrice float64

	// Get actual position data from Bybit API after order execution
	fmt.Printf("🔍 Fetching current position from Bybit API...\n")
	position, err := bot.getCurrentPosition()
	if err != nil {
		log.Printf("⚠️ Could not fetch position data: %v", err)
		// Fallback to order response data
		executedValue := parseFloat(order.CumExecValue)
		avgExecPrice := parseFloat(order.AvgPrice)
		if executedValue == 0 {
			executedValue = amount
		}
		if avgExecPrice == 0 {
			avgExecPrice = price
		}
		actualNotionalPosition = executedValue
		actualExecutionPrice = avgExecPrice
	} else if position == nil {
		log.Printf("⚠️ No position found - using order data")
		// Use order response as fallback
		executedValue := parseFloat(order.CumExecValue)
		avgExecPrice := parseFloat(order.AvgPrice)
		if executedValue == 0 {
			executedValue = amount
		}
		if avgExecPrice == 0 {
			avgExecPrice = price
		}
		actualNotionalPosition = executedValue
		actualExecutionPrice = avgExecPrice
	} else {
		// Use real position data from API
		actualNotionalPosition = parseFloat(position.PositionValue)
		actualExecutionPrice = parseFloat(position.AvgPrice)
		
		fmt.Printf("✅ Position API Data:\n")
		fmt.Printf("   Position Size: %s %s\n", position.Size, position.Symbol)
		fmt.Printf("   Entry Price: $%.2f\n", actualExecutionPrice)
		fmt.Printf("   Position Value: $%.2f\n", actualNotionalPosition)
		fmt.Printf("   Unrealized P&L: $%s\n", position.UnrealisedPnl)
	}
	
	// Calculate the increment for this order
	var orderIncrement float64
	if position != nil {
		// When we have position API data, use the total position directly
		// The increment is the difference from our previous position
		orderIncrement = actualNotionalPosition - bot.currentPosition
		bot.currentPosition = actualNotionalPosition
		bot.averagePrice = actualExecutionPrice
	} else {
		// When using fallback data, add to existing position
		orderIncrement = actualNotionalPosition
		if bot.currentPosition == 0 {
			bot.averagePrice = actualExecutionPrice
			bot.currentPosition = actualNotionalPosition
		} else {
			// Average based on actual executed notional position sizes
			totalNotional := bot.currentPosition + actualNotionalPosition
			weightedPrice := (bot.currentPosition*bot.averagePrice + actualNotionalPosition*actualExecutionPrice) / totalNotional
			bot.currentPosition = totalNotional
			bot.averagePrice = weightedPrice
		}
	}

	bot.totalInvested += orderIncrement         // Track actual invested amount increment
	bot.balance -= marginRequired               // Only deduct margin requirement  
	bot.dcaLevel++

	fmt.Printf("✅ BUY ORDER EXECUTED\n")
	fmt.Printf("   Order ID: %s\n", order.OrderID)
	fmt.Printf("   Quantity: %s %s\n", order.CumExecQty, bot.symbol)
	fmt.Printf("   Price: $%.2f\n", actualExecutionPrice)
	fmt.Printf("   Notional: $%.2f\n", orderIncrement)
	fmt.Printf("   Margin Used: $%.2f (10x leverage)\n", marginRequired)
	fmt.Printf("   Total Position: $%.2f\n", bot.currentPosition)
	fmt.Printf("   DCA Level: %d\n", bot.dcaLevel)
}

func (bot *LiveBot) executeSell(price float64) {
	if bot.currentPosition <= 0 {
		return
	}

	// Calculate the asset quantity to close the entire notional position
	assetQuantity := bot.currentPosition / price
	quantityStr := fmt.Sprintf("%.6f", assetQuantity)

	// Close entire position
	ctx := context.Background()
	order, err := bot.bybitClient.PlaceMarketOrder(ctx, bot.category, bot.symbol, bybit.OrderSideSell, quantityStr)
	if err != nil {
		log.Printf("❌ Failed to place sell order: %v", err)
		return
	}

	// P&L calculation for futures based on price change
	priceRatio := price / bot.averagePrice
	saleValue := bot.totalInvested * priceRatio
	profit := saleValue - bot.totalInvested
	profitPercent := (price - bot.averagePrice) / bot.averagePrice * 100

	fmt.Printf("✅ SELL ORDER EXECUTED\n")
	fmt.Printf("   Order ID: %s\n", order.OrderID)
	fmt.Printf("   Asset Quantity: %.6f %s\n", assetQuantity, bot.symbol)
	fmt.Printf("   Price: $%.2f\n", price)
	fmt.Printf("   Notional Closed: $%.2f\n", bot.currentPosition)
	fmt.Printf("   Profit: $%.2f (%.2f%%)\n", profit, profitPercent)

	// Reset position - for futures, we get back the margin plus profit/loss
	marginReleased := bot.totalInvested / 10  // Original margin used
	bot.balance += marginReleased + profit    // Margin + P&L
	bot.currentPosition = 0
	bot.averagePrice = 0
	bot.totalInvested = 0
	bot.dcaLevel = 0
	
	// Refresh balance to sync with real account after sale
	if err := bot.refreshBalance(); err != nil {
		log.Printf("⚠️ Could not refresh balance after sale: %v", err)
	}
}

func (bot *LiveBot) logStatus(currentPrice float64, action string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	fmt.Printf("\n[%s] 📊 Market Status\n", timestamp)
	fmt.Printf("💰 Price: $%.2f | Action: %s\n", currentPrice, action)
	fmt.Printf("💼 Balance: $%.2f | Notional Position: $%.2f\n", bot.balance, bot.currentPosition)
	
	if bot.currentPosition > 0 && bot.averagePrice > 0 {
		// For futures: calculate current value based on price change from average
		priceRatio := currentPrice / bot.averagePrice
		currentValue := bot.totalInvested * priceRatio
		unrealizedPnL := currentValue - bot.totalInvested
		unrealizedPercent := (currentPrice - bot.averagePrice) / bot.averagePrice * 100
		
		fmt.Printf("📈 Entry Price: $%.2f | Current Value: $%.2f\n", bot.averagePrice, currentValue)
		fmt.Printf("📊 Unrealized P&L: $%.2f (%.2f%%) | DCA Level: %d\n", unrealizedPnL, unrealizedPercent, bot.dcaLevel)
		fmt.Printf("💰 Margin Used: $%.2f (10x leverage)\n", bot.totalInvested/10)
	} else {
		fmt.Printf("📊 No active position\n")
	}
	fmt.Println(strings.Repeat("-", 50))
}

func (bot *LiveBot) getIntervalDuration() time.Duration {
	switch bot.interval {
	case "1":
		return 1 * time.Minute
	case "3":
		return 3 * time.Minute
	case "5":
		return 5 * time.Minute
	case "15":
		return 15 * time.Minute
	case "30":
		return 30 * time.Minute
	case "60":
		return 1 * time.Hour
	case "240":
		return 4 * time.Hour
	case "D":
		return 24 * time.Hour
	default:
		// Try to parse as minutes
		if minutes, err := strconv.Atoi(bot.interval); err == nil {
			return time.Duration(minutes) * time.Minute
		}
		return 1 * time.Hour // Default
	}
}

// getTimeUntilNextCandle calculates how long to wait until the next candle closes
func (bot *LiveBot) getTimeUntilNextCandle() time.Duration {
	now := time.Now().UTC()
	
	switch bot.interval {
	case "1":
		// Next minute boundary (e.g., 14:23:30 -> wait until 14:24:00)
		next := now.Truncate(time.Minute).Add(time.Minute)
		return next.Sub(now)
		
	case "3":
		// Next 3-minute boundary (e.g., 14:23:30 -> wait until 14:24:00 or 14:27:00)
		minutes := now.Minute()
		nextMinute := ((minutes / 3) + 1) * 3
		if nextMinute >= 60 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
		
	case "5":
		// Next 5-minute boundary (e.g., 14:23:30 -> wait until 14:25:00)
		minutes := now.Minute()
		nextMinute := ((minutes / 5) + 1) * 5
		if nextMinute >= 60 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
		
	case "15":
		// Next 15-minute boundary (e.g., 14:23:30 -> wait until 14:30:00)
		minutes := now.Minute()
		nextMinute := ((minutes / 15) + 1) * 15
		if nextMinute >= 60 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
		
	case "30":
		// Next 30-minute boundary (e.g., 14:23:30 -> wait until 14:30:00 or 15:00:00)
		minutes := now.Minute()
		var nextMinute int
		if minutes < 30 {
			nextMinute = 30
		} else {
			nextMinute = 0 // Next hour
		}
		
		if nextMinute == 0 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
		
	case "60":
		// Next hour boundary (e.g., 14:23:30 -> wait until 15:00:00)
		next := now.Truncate(time.Hour).Add(time.Hour)
		return next.Sub(now)
		
	case "240":
		// Next 4-hour boundary (e.g., 14:23:30 -> wait until 16:00:00 or 20:00:00)
		hours := now.Hour()
		nextHour := ((hours / 4) + 1) * 4
		if nextHour >= 24 {
			next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), nextHour, 0, 0, 0, time.UTC)
		return next.Sub(now)
		
	case "D":
		// Next day boundary (e.g., today 14:23:30 -> wait until tomorrow 00:00:00)
		next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		return next.Sub(now)
		
	default:
		// Try to parse as minutes for custom intervals
		if minutes, err := strconv.Atoi(bot.interval); err == nil {
			intervalMinutes := minutes
			currentMinutes := now.Minute()
			nextMinute := ((currentMinutes / intervalMinutes) + 1) * intervalMinutes
			
			if nextMinute >= 60 {
				next := now.Truncate(time.Hour).Add(time.Hour)
				return next.Sub(now)
			}
			next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
			return next.Sub(now)
		}
		
		// Fallback to 5-minute alignment
		minutes := now.Minute()
		nextMinute := ((minutes / 5) + 1) * 5
		if nextMinute >= 60 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
	}
}

// getCurrentPosition fetches the current position data from Bybit API
func (bot *LiveBot) getCurrentPosition() (*bybit.PositionInfo, error) {
	ctx := context.Background()
	positions, err := bot.bybitClient.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	// Find position for our symbol
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			// Only return positions with size > 0
			if parseFloat(pos.Size) > 0 {
				return &pos, nil
			}
		}
	}

	// No position found or position size is 0
	return nil, nil
}

func parseFloat(s string) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return 0
}


