package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/internal/config"
	"github.com/Zmey56/enhanced-dca-bot/internal/exchange"
	"github.com/Zmey56/enhanced-dca-bot/internal/monitoring"
	"github.com/Zmey56/enhanced-dca-bot/internal/notifications"
	"github.com/Zmey56/enhanced-dca-bot/internal/risk"
	"github.com/Zmey56/enhanced-dca-bot/internal/strategy"
)

// DCABot represents the main trading bot
type DCABot struct {
	config        *config.Config
	exchange      exchange.Exchange
	strategy      strategy.Strategy
	riskManager   risk.RiskManager
	notifier      notifications.Notifier
	healthChecker *monitoring.HealthChecker
	running       bool
	stopChan      chan struct{}
}

// NewDCABot creates a new DCA bot instance
func NewDCABot(
	cfg *config.Config,
	exch exchange.Exchange,
	strat strategy.Strategy,
	riskMgr risk.RiskManager,
	notif notifications.Notifier,
	health *monitoring.HealthChecker,
) *DCABot {
	return &DCABot{
		config:        cfg,
		exchange:      exch,
		strategy:      strat,
		riskManager:   riskMgr,
		notifier:      notif,
		healthChecker: health,
		stopChan:      make(chan struct{}),
	}
}

// Start initializes and starts the bot
func (b *DCABot) Start(ctx context.Context) error {
	log.Println("Starting DCA Bot...")

	// Connect to exchange
	if err := b.exchange.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to exchange: %w", err)
	}

	b.healthChecker.SetConnected(true)
	b.running = true

	// Send startup notification (optional)
	if b.config.Notifications.TelegramToken != "" {
		if err := b.notifier.SendAlert("info", "DCA Bot started successfully"); err != nil {
			log.Printf("Failed to send startup notification: %v", err)
		}
	} else {
		log.Println("Telegram notifications disabled (no token configured)")
	}

	// Start trading loop
	go b.tradingLoop(ctx)

	log.Println("DCA Bot started successfully")
	return nil
}

// tradingLoop runs the main trading cycle
func (b *DCABot) tradingLoop(ctx context.Context) {
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
func (b *DCABot) executeTradingCycle(ctx context.Context) error {
	// Get current market data
	klines, err := b.exchange.GetKlines(b.config.Strategy.Symbol, "1h", 100)
	if err != nil {
		return fmt.Errorf("failed to get market data: %w", err)
	}

	// Get current ticker
	ticker, err := b.exchange.GetTicker(b.config.Strategy.Symbol)
	if err != nil {
		return fmt.Errorf("failed to get ticker: %w", err)
	}

	b.healthChecker.UpdatePrice(ticker.Price)

	// Get trading decision from strategy
	decision, err := b.strategy.ShouldExecuteTrade(klines)
	if err != nil {
		return fmt.Errorf("strategy error: %w", err)
	}

	// Execute trade if needed
	if decision.Action == strategy.ActionBuy {
		if err := b.executeBuyOrder(ctx, decision, ticker.Price); err != nil {
			return fmt.Errorf("failed to execute buy order: %w", err)
		}
	}

	return nil
}

// executeBuyOrder executes a buy order
func (b *DCABot) executeBuyOrder(ctx context.Context, decision *strategy.TradeDecision, price float64) error {
	// Get current balance
	balance, err := b.exchange.GetBalance("USDT")
	if err != nil {
		return fmt.Errorf("failed to get balance: %w", err)
	}

	// Validate order with risk manager
	order := &risk.Order{
		Symbol: b.config.Strategy.Symbol,
		Side:   exchange.OrderBuy,
		Amount: decision.Amount,
		Price:  price,
	}

	portfolio := &risk.Portfolio{
		Balance: balance.Free,
		Symbol:  b.config.Strategy.Symbol,
	}

	if err := b.riskManager.ValidateOrder(order, portfolio); err != nil {
		log.Printf("Risk validation failed: %v", err)
		return nil // Don't treat as error, just skip the trade
	}

	// Place the order
	orderResult, err := b.exchange.PlaceMarketOrder(
		b.config.Strategy.Symbol,
		exchange.OrderBuy,
		decision.Amount,
	)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	// Update health checker
	b.healthChecker.UpdateLastTrade(time.Now())

	// Send notification
	message := fmt.Sprintf(
		"Buy order executed\nSymbol: %s\nAmount: $%.2f\nPrice: $%.2f\nConfidence: %.2f%%",
		b.config.Strategy.Symbol,
		decision.Amount,
		price,
		decision.Confidence*100,
	)

	if err := b.notifier.SendAlert("success", message); err != nil {
		log.Printf("Failed to send trade notification: %v", err)
	}

	log.Printf("Buy order executed: %+v", orderResult)
	return nil
}

// Shutdown gracefully shuts down the bot
func (b *DCABot) Shutdown(ctx context.Context) error {
	log.Println("Shutting down DCA Bot...")

	b.running = false
	close(b.stopChan)

	// Disconnect from exchange
	if err := b.exchange.Disconnect(); err != nil {
		log.Printf("Error disconnecting from exchange: %v", err)
	}

	b.healthChecker.SetConnected(false)

	// Send shutdown notification
	if err := b.notifier.SendAlert("info", "DCA Bot stopped"); err != nil {
		log.Printf("Failed to send shutdown notification: %v", err)
	}

	log.Println("DCA Bot shutdown complete")
	return nil
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Setup logging
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("Starting Enhanced DCA Bot in %s mode", cfg.Environment)

	// Initialize components
	healthChecker := monitoring.NewHealthChecker()
	notifier := notifications.NewTelegramNotifier(cfg.Notifications.TelegramToken, cfg.Notifications.TelegramChatID)

	// Initialize exchange
	exch := exchange.NewBinanceExchange(cfg.Exchange.APIKey, cfg.Exchange.Secret, cfg.Exchange.Testnet)

	// Initialize strategy
	strat := strategy.NewMultiIndicatorStrategy()

	// Initialize risk manager
	riskManager := risk.NewRiskManager(cfg.Strategy.MaxMultiplier)

	// Initialize bot
	bot := NewDCABot(cfg, exch, strat, riskManager, notifier, healthChecker)

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

	log.Println("Bot stopped successfully")
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
