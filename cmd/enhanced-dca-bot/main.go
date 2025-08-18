package main

/*
Enhanced DCA Bot - Live Trading Version
=======================================

This bot implements the Enhanced DCA strategy with multiple technical indicators
for live trading. It uses the same logic as the backtest system and loads
optimized configuration files for optimal performance.

Features:
- Enhanced DCA strategy with price threshold
- Multiple technical indicators (RSI, MACD, Bollinger Bands, EMA)
- Risk management and position sizing
- Telegram notifications
- Health monitoring and metrics
- Real exchange integration (configurable for testnet/production)
- Optimized parameters from backtest results

The Enhanced DCA strategy:
- Buys when multiple indicators show consensus
- Uses price threshold to space out DCA entries
- Adjusts position size based on signal strength
- Implements take-profit cycles
*/

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/monitoring"
	"github.com/ducminhle1904/crypto-dca-bot/internal/notifications"
	"github.com/ducminhle1904/crypto-dca-bot/internal/risk"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
)

// OptimizedConfig represents the optimized configuration from backtest results
type OptimizedConfig struct {
	DataFile       string    `json:"data_file"`
	Symbol         string    `json:"symbol"`
	InitialBalance float64   `json:"initial_balance"`
	Commission     float64   `json:"commission"`
	WindowSize     int       `json:"window_size"`
	BaseAmount     float64   `json:"base_amount"`
	MaxMultiplier  float64   `json:"max_multiplier"`
	PriceThreshold float64   `json:"price_threshold"`
	RSIPeriod      int       `json:"rsi_period"`
	RSIOversold    float64   `json:"rsi_oversold"`
	RSIOverbought  float64   `json:"rsi_overbought"`
	MACDFast       int       `json:"macd_fast"`
	MACDSlow       int       `json:"macd_slow"`
	MACDSignal     int       `json:"macd_signal"`
	BBPeriod       int       `json:"bb_period"`
	BBStdDev       float64   `json:"bb_std_dev"`
	EMAPeriod      int       `json:"ema_period"`
	Indicators     []string  `json:"indicators"`
	TPPercent      float64   `json:"tp_percent"`
	Cycle          bool      `json:"cycle"`
	
	// Derived fields from config filename
	Interval       string    `json:"-"` // e.g., "5m", "15m", "1h", "4h"
	ConfigPath     string    `json:"-"` // Full path to config file
}

// EnhancedDCABot represents the Enhanced DCA trading bot
type EnhancedDCABot struct {
	config        *config.Config
	exchange      exchange.Exchange
	strategy      strategy.Strategy
	riskManager   risk.RiskManager
	notifier      notifications.Notifier
	healthChecker *monitoring.HealthChecker
	running       bool
	stopChan      chan struct{}
	
	// Enhanced DCA specific fields
	lastEntryPrice   float64
	cycleStartTime   time.Time
	cycleNumber      int
	
	// Optimized configuration
	optimizedConfig  *OptimizedConfig
}

// NewEnhancedDCABot creates a new Enhanced DCA bot instance
func NewEnhancedDCABot(
	cfg *config.Config,
	exch exchange.Exchange,
	strat strategy.Strategy,
	riskMgr risk.RiskManager,
	notif notifications.Notifier,
	health *monitoring.HealthChecker,
	optCfg *OptimizedConfig,
) *EnhancedDCABot {
	return &EnhancedDCABot{
		config:        cfg,
		exchange:      exch,
		strategy:      strat,
		riskManager:   riskMgr,
		notifier:      notif,
		healthChecker: health,
		stopChan:      make(chan struct{}),
		cycleNumber:   1,
		optimizedConfig: optCfg,
	}
}

// Start initializes and starts the bot
func (b *EnhancedDCABot) Start(ctx context.Context) error {
	log.Println("🚀 Starting Enhanced DCA Bot...")

	// Connect to exchange
	if err := b.exchange.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to exchange: %w", err)
	}

	b.healthChecker.SetConnected(true)
	b.running = true
	b.cycleStartTime = time.Now()

	// Send startup notification
	if b.config.Notifications.TelegramToken != "" {
		message := fmt.Sprintf(
			"🚀 *Enhanced DCA Bot Started*\n\n"+
				"Symbol: %s\n"+
				"Interval: %s\n"+
				"Base Amount: $%.2f\n"+
				"Max Multiplier: %.2fx\n"+
				"Trading Interval: %v\n"+
				"Price Threshold: %.2f%%\n"+
				"Take Profit: %.2f%%\n"+
				"Cycle #%d Started\n"+
				"📊 Using Optimized Config: %s",
			b.optimizedConfig.Symbol,
			b.optimizedConfig.Interval,
			b.optimizedConfig.BaseAmount,
			b.optimizedConfig.MaxMultiplier,
			b.config.Strategy.Interval,
			b.optimizedConfig.PriceThreshold*100,
			b.optimizedConfig.TPPercent*100,
			b.cycleNumber,
			filepath.Base(b.optimizedConfig.ConfigPath),
		)

		if err := b.notifier.SendAlert("info", message); err != nil {
			log.Printf("Failed to send startup notification: %v", err)
		} else {
			log.Println("📱 Telegram startup notification sent")
		}
	} else {
		log.Println("Telegram notifications disabled (no token configured)")
	}

	// Start trading loop
	go b.tradingLoop(ctx)

	log.Printf("✅ Enhanced DCA Bot started successfully - Cycle #%d", b.cycleNumber)
	return nil
}

// tradingLoop runs the main trading cycle
func (b *EnhancedDCABot) tradingLoop(ctx context.Context) {
	ticker := time.NewTicker(b.config.Strategy.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Trading loop stopped")
			return
		case <-b.stopChan:
			log.Println("Trading loop stopped")
			return
		case <-ticker.C:
			if err := b.executeTradingCycle(ctx); err != nil {
				log.Printf("Trading cycle error: %v", err)
				b.healthChecker.AddError(err.Error())
			}
		}
	}
}

// executeTradingCycle performs one trading cycle
func (b *EnhancedDCABot) executeTradingCycle(ctx context.Context) error {
	log.Printf("🔄 === Starting Trading Cycle #%d ===", b.cycleNumber)

	// Get current market data
	klines, err := b.exchange.GetKlines(b.optimizedConfig.Symbol, b.optimizedConfig.Interval, 100)
	if err != nil {
		return fmt.Errorf("failed to get market data: %w", err)
	}

	// Get current ticker
	ticker, err := b.exchange.GetTicker(b.optimizedConfig.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get ticker: %w", err)
	}

	b.healthChecker.UpdatePrice(ticker.Price)

	// Log current market conditions
	currentPrice := ticker.Price
	log.Printf("📊 Current Price: $%.2f", currentPrice)
	log.Printf("📈 24h Volume: %.2f", ticker.Volume)
	log.Printf("⏰ Timestamp: %s", ticker.Timestamp.Format("2006-01-02 15:04:05"))
	
	if b.lastEntryPrice > 0 {
		priceChange := ((currentPrice - b.lastEntryPrice) / b.lastEntryPrice) * 100
		log.Printf("📊 Price Change Since Last Entry: %.2f%%", priceChange)
	}

	// Get trading decision from Enhanced DCA strategy
	decision, err := b.strategy.ShouldExecuteTrade(klines)
	if err != nil {
		return fmt.Errorf("strategy error: %w", err)
	}

	// Log strategy decision
	b.logStrategyDecision(decision, currentPrice)

	// Execute trade if needed
	if decision.Action == strategy.ActionBuy {
		if err := b.executeBuyOrder(ctx, decision, ticker.Price); err != nil {
			return fmt.Errorf("failed to execute buy order: %w", err)
		}
	} else {
		log.Println("⏸️  No trade executed - holding position")
	}

	// Check if we should complete the cycle (take profit)
	if b.shouldCompleteCycle(currentPrice) {
		if err := b.completeCycle(ctx, currentPrice); err != nil {
			log.Printf("Failed to complete cycle: %v", err)
		}
	}

	log.Printf("✅ === Trading Cycle #%d Complete ===", b.cycleNumber)
	return nil
}

// shouldCompleteCycle determines if the current cycle should be completed
func (b *EnhancedDCABot) shouldCompleteCycle(currentPrice float64) bool {
	if b.lastEntryPrice <= 0 {
		return false
	}
	
	// Use optimized take profit percentage from config
	tpPercent := b.optimizedConfig.TPPercent
	priceIncrease := (currentPrice - b.lastEntryPrice) / b.lastEntryPrice
	
	return priceIncrease >= tpPercent
}

// completeCycle completes the current DCA cycle and starts a new one
func (b *EnhancedDCABot) completeCycle(ctx context.Context, currentPrice float64) error {
	log.Printf("🎯 === Completing Cycle #%d ===", b.cycleNumber)
	
	// Calculate cycle performance
	cycleDuration := time.Since(b.cycleStartTime)
	priceChange := ((currentPrice - b.lastEntryPrice) / b.lastEntryPrice) * 100
	
	log.Printf("📈 Cycle #%d Performance:", b.cycleNumber)
	log.Printf("   Duration: %v", cycleDuration)
	log.Printf("   Entry Price: $%.2f", b.lastEntryPrice)
	log.Printf("   Exit Price: $%.2f", currentPrice)
	log.Printf("   Price Change: %.2f%%", priceChange)
	
	// Reset strategy state for new cycle
	b.strategy.OnCycleComplete()
	
	// Reset cycle variables
	b.lastEntryPrice = 0.0
	b.cycleNumber++
	b.cycleStartTime = time.Now()
	
	// Send cycle completion notification
	if b.config.Notifications.TelegramToken != "" {
		message := fmt.Sprintf(
			"🎯 *Cycle #%d Completed*\n\n"+
				"Symbol: %s\n"+
				"Entry Price: $%.2f\n"+
				"Exit Price: $%.2f\n"+
				"Profit: %.2f%%\n"+
				"Duration: %v\n"+
				"Starting Cycle #%d",
			b.optimizedConfig.Symbol,
			b.lastEntryPrice,
			currentPrice,
			priceChange,
			cycleDuration,
			b.cycleNumber,
		)
		
		if err := b.notifier.SendAlert("success", message); err != nil {
			log.Printf("Failed to send cycle completion notification: %v", err)
		}
	}
	
	log.Printf("🔄 === Starting New Cycle #%d ===", b.cycleNumber)
	return nil
}

// executeBuyOrder executes a buy order
func (b *EnhancedDCABot) executeBuyOrder(ctx context.Context, decision *strategy.TradeDecision, price float64) error {
	log.Println("💼 === Executing Enhanced DCA Buy Order ===")

	// Get current balance
	balance, err := b.exchange.GetBalance("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	log.Printf("💰 Current Balance: $%.2f USDT", balance.Free)

	// Validate order with risk manager
	order := &risk.Order{
		Symbol: b.optimizedConfig.Symbol,
		Side:   exchange.OrderBuy,
		Amount: decision.Amount,
		Price:  price,
	}

	portfolio := &risk.Portfolio{
		Balance: balance.Free,
		Symbol:  b.optimizedConfig.Symbol,
	}

	log.Println("🛡️  === Risk Management Check ===")
	if err := b.riskManager.ValidateOrder(order, portfolio); err != nil {
		log.Printf("❌ Risk validation failed: %v", err)
		log.Println("⏸️  Trade cancelled due to risk management")
		return nil // Don't treat as error, just skip the trade
	}
	log.Println("✅ Risk validation passed")

	// Place the order
	log.Println("📤 === Placing Market Order ===")
	orderResult, err := b.exchange.PlaceMarketOrder(
		b.optimizedConfig.Symbol,
		exchange.OrderBuy,
		decision.Amount,
	)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	// Update last entry price for this cycle
	b.lastEntryPrice = price

	// Log order details
	log.Printf("🆔 Order ID: %s", orderResult.ID)
	log.Printf("📊 Symbol: %s", orderResult.Symbol)
	log.Printf("💰 Amount: $%.2f", orderResult.Quantity)
	log.Printf("📈 Price: $%.2f", price)
	log.Printf("📦 Quantity: %.6f %s", orderResult.Quantity/price, b.optimizedConfig.Symbol[:3])
	log.Printf("📋 Status: %s", orderResult.Status)
	log.Printf("⏰ Timestamp: %s", orderResult.Timestamp.Format("2006-01-02 15:04:05"))
	log.Printf("🎯 Cycle #%d Entry Price: $%.2f", b.cycleNumber, b.lastEntryPrice)

	// Update health checker
	b.healthChecker.UpdateLastTrade(time.Now())

	// Send notification
	if b.config.Notifications.TelegramToken != "" {
		message := fmt.Sprintf(
			"🟢 *Enhanced DCA Trade Executed*\n\n"+
				"Symbol: %s\n"+
				"Action: BUY\n"+
				"Amount: $%.2f\n"+
				"Price: $%.2f\n"+
				"Quantity: %.6f %s\n"+
				"Confidence: %.2f%%\n"+
				"Cycle: #%d\n"+
				"Order ID: %s",
			b.optimizedConfig.Symbol,
			decision.Amount,
			price,
			orderResult.Quantity/price,
			b.optimizedConfig.Symbol[:3],
			decision.Confidence*100,
			b.cycleNumber,
			orderResult.ID,
		)

		if err := b.notifier.SendAlert("success", message); err != nil {
			log.Printf("⚠️  Failed to send trade notification: %v", err)
		} else {
			log.Println("📱 Telegram notification sent")
		}
	}

	log.Println("✅ === Enhanced DCA Buy Order Executed Successfully ===")
	return nil
}

// Shutdown gracefully shuts down the bot
func (b *EnhancedDCABot) Shutdown(ctx context.Context) error {
	log.Println("Shutting down Enhanced DCA Bot...")

	b.running = false
	close(b.stopChan)

	// Disconnect from exchange
	if err := b.exchange.Disconnect(); err != nil {
		log.Printf("Error disconnecting from exchange: %v", err)
	}

	b.healthChecker.SetConnected(false)

	// Send shutdown notification
	if b.config.Notifications.TelegramToken != "" {
		message := fmt.Sprintf(
			"🛑 *Enhanced DCA Bot Stopped*\n\n"+
				"Final Status:\n"+
				"Cycles Completed: %d\n"+
				"Current Cycle: #%d\n"+
				"Last Entry Price: $%.2f",
			b.cycleNumber-1,
			b.cycleNumber,
			b.lastEntryPrice,
		)
		
		if err := b.notifier.SendAlert("info", message); err != nil {
			log.Printf("Failed to send shutdown notification: %v", err)
		}
	}

	log.Println("Enhanced DCA Bot shutdown complete")
	return nil
}

// logStrategyDecision logs detailed information about the strategy decision
func (b *EnhancedDCABot) logStrategyDecision(decision *strategy.TradeDecision, currentPrice float64) {
	log.Println("🧠 === Enhanced DCA Strategy Analysis ===")

	// Log decision details
	actionEmoji := "⏸️"
	actionText := "HOLD"
	switch decision.Action {
	case strategy.ActionBuy:
		actionEmoji = "🟢"
		actionText = "BUY"
	case strategy.ActionSell:
		actionEmoji = "🔴"
		actionText = "SELL"
	}

	log.Printf("%s Decision: %s", actionEmoji, actionText)
	log.Printf("📊 Confidence: %.2f%%", decision.Confidence*100)
	log.Printf("💪 Signal Strength: %.2f%%", decision.Strength*100)
	log.Printf("💡 Reason: %s", decision.Reason)

	if decision.Action == strategy.ActionBuy {
		log.Printf("💰 Amount: $%.2f", decision.Amount)
		log.Printf("📈 Price: $%.2f", currentPrice)
		log.Printf("🪙 Quantity: %.6f %s", decision.Amount/currentPrice, b.optimizedConfig.Symbol[:3])
		log.Printf("🎯 Cycle #%d Entry", b.cycleNumber)
	}

	log.Println("📋 === End Strategy Analysis ===")
}

// createEnhancedDCAStrategy creates and configures the Enhanced DCA strategy with optimized parameters
func createEnhancedDCAStrategy(optCfg *OptimizedConfig) strategy.Strategy {
	// Create Enhanced DCA strategy with optimized base amount
	dca := strategy.NewEnhancedDCAStrategy(optCfg.BaseAmount)
	
	// Set optimized price threshold
	dca.SetPriceThreshold(optCfg.PriceThreshold)
	
	// Add technical indicators with optimized parameters
	for _, indicatorName := range optCfg.Indicators {
		switch indicatorName {
		case "rsi":
			rsi := indicators.NewRSI(optCfg.RSIPeriod)
			rsi.SetOversold(optCfg.RSIOversold)
			rsi.SetOverbought(optCfg.RSIOverbought)
			dca.AddIndicator(rsi)
			log.Printf("📊 Added RSI: Period=%d, Oversold=%.1f, Overbought=%.1f", 
				optCfg.RSIPeriod, optCfg.RSIOversold, optCfg.RSIOverbought)
			
		case "macd":
			macd := indicators.NewMACD(optCfg.MACDFast, optCfg.MACDSlow, optCfg.MACDSignal)
			dca.AddIndicator(macd)
			log.Printf("📊 Added MACD: Fast=%d, Slow=%d, Signal=%d", 
				optCfg.MACDFast, optCfg.MACDSlow, optCfg.MACDSignal)
			
		case "bb":
			bb := indicators.NewBollingerBands(optCfg.BBPeriod, optCfg.BBStdDev)
			dca.AddIndicator(bb)
			log.Printf("📊 Added Bollinger Bands: Period=%d, StdDev=%.1f", 
				optCfg.BBPeriod, optCfg.BBStdDev)
			
		case "ema":
			ema := indicators.NewEMA(optCfg.EMAPeriod)
			dca.AddIndicator(ema)
			log.Printf("📊 Added EMA: Period=%d", optCfg.EMAPeriod)
		}
	}
	
	return dca
}

// loadOptimizedConfig loads the optimized configuration from a JSON file
func loadOptimizedConfig(configPath string) (*OptimizedConfig, error) {
	// If no config path provided, try to find one based on environment
	if configPath == "" {
		// Default to BTC 15m if no config specified
		configPath = "configs/btc_15m.json"
		log.Printf("📁 No config specified, using default: %s", configPath)
	}
	
	// Ensure the path is relative to configs directory if not absolute
	if !filepath.IsAbs(configPath) && !filepath.HasPrefix(configPath, "configs/") {
		configPath = filepath.Join("configs", configPath)
	}
	
	// Add .json extension if not present
	if filepath.Ext(configPath) == "" {
		configPath += ".json"
	}
	
	log.Printf("📁 Loading optimized configuration from: %s", configPath)
	
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file %s: %w", configPath, err)
	}
	defer file.Close()
	
	var optCfg OptimizedConfig
	if err := json.NewDecoder(file).Decode(&optCfg); err != nil {
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}
	
	// Extract interval from filename (e.g., "btc_15m.json" -> "15m")
	filename := filepath.Base(configPath)
	// Remove .json extension
	filenameWithoutExt := filename[:len(filename)-len(filepath.Ext(filename))]
	// Split by underscore and get the interval part
	parts := strings.Split(filenameWithoutExt, "_")
	if len(parts) >= 2 {
		optCfg.Interval = parts[len(parts)-1] // Last part is the interval
	} else {
		// Fallback to default if filename format is unexpected
		optCfg.Interval = "1h"
		log.Printf("⚠️  Could not extract interval from filename, using default: %s", optCfg.Interval)
	}
	optCfg.ConfigPath = configPath
	
	log.Printf("✅ Loaded optimized config for %s", optCfg.Symbol)
	log.Printf("   Data Interval: %s", optCfg.Interval)
	log.Printf("   Base Amount: $%.2f", optCfg.BaseAmount)
	log.Printf("   Max Multiplier: %.2fx", optCfg.MaxMultiplier)
	log.Printf("   Price Threshold: %.2f%%", optCfg.PriceThreshold*100)
	log.Printf("   Take Profit: %.2f%%", optCfg.TPPercent*100)
	log.Printf("   Indicators: %v", optCfg.Indicators)
	
	return &optCfg, nil
}

func main() {
	// Parse command line flags
	var (
		configFile = flag.String("config", "", "Path to optimized configuration file (e.g., btc_15m, eth_30m)")
	)
	flag.Parse()

	// Load optimized configuration
	optCfg, err := loadOptimizedConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load optimized config: %v", err)
	}

	// Load main configuration
	cfg := config.Load()

	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("🚀 === Enhanced DCA Bot - Live Trading with Optimized Config ===")
	log.Printf("🌍 Environment: %s", cfg.Environment)
	log.Printf("📊 Trading Symbol: %s", optCfg.Symbol)
	log.Printf("⏰ Data Interval: %s", optCfg.Interval)
	log.Printf("⏰ Trading Interval: %v", cfg.Strategy.Interval)
	log.Printf("💰 Base Amount: $%.2f (Optimized)", optCfg.BaseAmount)
	log.Printf("🛡️  Max Multiplier: %.2fx (Optimized)", optCfg.MaxMultiplier)
	log.Printf("📉 Price Threshold: %.2f%% (Optimized)", optCfg.PriceThreshold*100)
	log.Printf("📈 Take Profit: %.2f%% (Optimized)", optCfg.TPPercent*100)
	log.Printf("📊 Indicators: %v (Optimized)", optCfg.Indicators)
	log.Println("📈 Strategy: Enhanced DCA with Optimized Parameters")
	log.Println("🎯 Features: Price Threshold, Take-Profit Cycles, Risk Management")
	log.Println("=")

	// Initialize components
	healthChecker := monitoring.NewHealthChecker()
	notifier := notifications.NewTelegramNotifier(cfg.Notifications.TelegramToken, cfg.Notifications.TelegramChatID)

	// Initialize exchange
	exch := exchange.NewBinanceExchange(cfg.Exchange.APIKey, cfg.Exchange.Secret, cfg.Exchange.Testnet)

	// Initialize Enhanced DCA strategy with optimized parameters
	strat := createEnhancedDCAStrategy(optCfg)

	// Initialize risk manager with optimized multiplier
	riskManager := risk.NewRiskManager(optCfg.MaxMultiplier)

	// Initialize bot
	bot := NewEnhancedDCABot(cfg, exch, strat, riskManager, notifier, healthChecker, optCfg)

	// Setup HTTP servers
	go setupMonitoringServers(cfg, healthChecker)

	// Start the bot
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := bot.Start(ctx); err != nil {
			log.Printf("Bot error: %v", err)
			cancel()
		}
	}()

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down...")
	cancel()

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := bot.Shutdown(shutdownCtx); err != nil {
		log.Printf("Error during shutdown: %v", err)
	}

	log.Println("Enhanced DCA Bot stopped successfully")
}

func setupMonitoringServers(cfg *config.Config, healthChecker *monitoring.HealthChecker) {
	// Create separate mux for health server
	healthMux := http.NewServeMux()
	healthMux.Handle("/health", healthChecker)

	// Start health server
	go func() {
		log.Printf("Starting health server on port %d", cfg.Monitoring.HealthPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Monitoring.HealthPort), healthMux); err != nil {
			log.Printf("Health server error: %v", err)
		}
	}()

	// Start Prometheus metrics server
	go func() {
		log.Printf("Starting Prometheus server on port %d", cfg.Monitoring.PrometheusPort)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", cfg.Monitoring.PrometheusPort), monitoring.NewMetricsHandler()); err != nil {
			log.Printf("Prometheus server error: %v", err)
		}
	}()
}
