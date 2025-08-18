package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
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
		dryRun     = flag.Bool("dry-run", true, "Dry run mode - no actual trading (default: true)")
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
	fmt.Printf("📊 Symbol: %s\n", symbol)
	fmt.Printf("⏰ Interval: %s\n", interval)
	fmt.Printf("🏪 Category: %s\n", category)
	fmt.Printf("🔧 Environment: %s\n", getEnvironmentString(*demo))
	fmt.Printf("🧪 Dry Run: %v\n", *dryRun)
	fmt.Println("=" + strings.Repeat("=", 50))

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

	fmt.Printf("🏦 Exchange: Bybit (demo: %v)\n", *demo)
	if *demo {
		fmt.Println("📝 Note: Demo mode uses paper trading - no real money involved")
	} else {
		fmt.Println("⚠️  LIVE TRADING MODE - Real money will be used!")
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
	if err := bot.Start(*dryRun); err != nil {
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
	configBalance := bot.balance
	bot.balance = realBalance
	
	fmt.Printf("💰 Account Balance Sync:\n")
	fmt.Printf("   Config Balance: $%.2f\n", configBalance)
	fmt.Printf("   Real Balance:   $%.2f (%s)\n", realBalance, baseCurrency)
	fmt.Printf("   Account Type:   %s\n", accountType)
	fmt.Printf("   Category:       %s\n", bot.category)
	
	if realBalance < configBalance {
		fmt.Printf("⚠️  Real balance is lower than config balance\n")
	} else if realBalance > configBalance {
		fmt.Printf("✅ Real balance is higher than config balance\n")
	} else {
		fmt.Printf("✅ Real balance matches config balance\n")
	}
	
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

func (bot *LiveBot) Start(dryRun bool) error {
	bot.running = true

	fmt.Printf("🔄 Starting live bot for %s/%s\n", bot.symbol, bot.interval)
	fmt.Printf("💰 Initial Balance: $%.2f\n", bot.balance)
	fmt.Printf("📈 Base DCA Amount: $%.2f\n", bot.config.BaseAmount)
	fmt.Printf("🔄 Max Multiplier: %.2f\n", bot.config.MaxMultiplier)
	fmt.Printf("📊 Price Threshold: %.2f%%\n", bot.config.PriceThreshold*100)
	fmt.Printf("🎯 Take Profit: %.2f%%\n", bot.config.TPPercent*100)
	
	if dryRun {
		fmt.Println("🧪 DRY RUN MODE - No actual trades will be executed")
	}
	fmt.Println(strings.Repeat("=", 60))

	// Start the main trading loop
	go bot.tradingLoop(dryRun)

	return nil
}

func (bot *LiveBot) Stop() {
	if bot.running {
		bot.running = false
		close(bot.stopChan)
	}
}

func (bot *LiveBot) tradingLoop(dryRun bool) {
	// Calculate interval duration
	intervalDuration := bot.getIntervalDuration()
	
	// Run initial check
	bot.checkAndTrade(dryRun)

	// Create ticker for regular checks
	ticker := time.NewTicker(intervalDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bot.checkAndTrade(dryRun)
		case <-bot.stopChan:
			return
		}
	}
}

func (bot *LiveBot) checkAndTrade(dryRun bool) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("❌ Error in trading loop: %v", r)
		}
	}()

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

	// Execute trading action
	if action != "HOLD" && !dryRun {
		bot.executeTrade(action, currentPrice)
	} else if action != "HOLD" && dryRun {
		fmt.Printf("🧪 DRY RUN: Would execute %s at $%.2f\n", action, currentPrice)
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
	// Refresh balance before trading (to ensure we have the latest balance)
	if err := bot.refreshBalance(); err != nil {
		log.Printf("⚠️ Could not refresh balance: %v", err)
	}

	// Calculate DCA amount based on level
	amount := bot.config.BaseAmount
	if bot.dcaLevel > 0 {
		multiplier := 1.0 + float64(bot.dcaLevel)*0.5 // Increase by 50% each level
		if multiplier > bot.config.MaxMultiplier {
			multiplier = bot.config.MaxMultiplier
		}
		amount *= multiplier
	}

	if amount > bot.balance {
		log.Printf("⚠️ Insufficient balance: $%.2f < $%.2f", bot.balance, amount)
		return
	}

	quantity := amount / price
	quantityStr := fmt.Sprintf("%.6f", quantity)

	// Execute buy order
	ctx := context.Background()
	order, err := bot.bybitClient.PlaceMarketOrder(ctx, bot.category, bot.symbol, bybit.OrderSideBuy, quantityStr)
	if err != nil {
		log.Printf("❌ Failed to place buy order: %v", err)
		return
	}

	// Update position
	if bot.currentPosition == 0 {
		bot.averagePrice = price
	} else {
		totalValue := bot.currentPosition*bot.averagePrice + quantity*price
		bot.currentPosition += quantity
		bot.averagePrice = totalValue / bot.currentPosition
	}

	bot.currentPosition += quantity
	bot.totalInvested += amount
	bot.balance -= amount
	bot.dcaLevel++

	fmt.Printf("✅ BUY ORDER EXECUTED\n")
	fmt.Printf("   Order ID: %s\n", order.OrderID)
	fmt.Printf("   Quantity: %.6f %s\n", quantity, bot.symbol)
	fmt.Printf("   Price: $%.2f\n", price)
	fmt.Printf("   Amount: $%.2f\n", amount)
	fmt.Printf("   DCA Level: %d\n", bot.dcaLevel)
}

func (bot *LiveBot) executeSell(price float64) {
	if bot.currentPosition <= 0 {
		return
	}

	quantityStr := fmt.Sprintf("%.6f", bot.currentPosition)

	// Sell entire position
	ctx := context.Background()
	order, err := bot.bybitClient.PlaceMarketOrder(ctx, bot.category, bot.symbol, bybit.OrderSideSell, quantityStr)
	if err != nil {
		log.Printf("❌ Failed to place sell order: %v", err)
		return
	}

	saleValue := bot.currentPosition * price
	profit := saleValue - bot.totalInvested
	profitPercent := (price - bot.averagePrice) / bot.averagePrice * 100

	fmt.Printf("✅ SELL ORDER EXECUTED\n")
	fmt.Printf("   Order ID: %s\n", order.OrderID)
	fmt.Printf("   Quantity: %.6f %s\n", bot.currentPosition, bot.symbol)
	fmt.Printf("   Price: $%.2f\n", price)
	fmt.Printf("   Sale Value: $%.2f\n", saleValue)
	fmt.Printf("   Profit: $%.2f (%.2f%%)\n", profit, profitPercent)

	// Reset position
	bot.balance += saleValue
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
	fmt.Printf("💼 Balance: $%.2f | Position: %.6f %s\n", bot.balance, bot.currentPosition, bot.symbol)
	
	if bot.currentPosition > 0 {
		currentValue := bot.currentPosition * currentPrice
		unrealizedPnL := currentValue - bot.totalInvested
		unrealizedPercent := (currentPrice - bot.averagePrice) / bot.averagePrice * 100
		
		fmt.Printf("📈 Avg Price: $%.2f | Current Value: $%.2f\n", bot.averagePrice, currentValue)
		fmt.Printf("📊 Unrealized P&L: $%.2f (%.2f%%) | DCA Level: %d\n", unrealizedPnL, unrealizedPercent, bot.dcaLevel)
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

func parseFloat(s string) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return 0
}


