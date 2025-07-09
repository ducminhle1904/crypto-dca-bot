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
	"github.com/Zmey56/enhanced-dca-bot/internal/indicators"
	"github.com/Zmey56/enhanced-dca-bot/internal/monitoring"
	"github.com/Zmey56/enhanced-dca-bot/internal/notifications"
	"github.com/Zmey56/enhanced-dca-bot/internal/risk"
	"github.com/Zmey56/enhanced-dca-bot/internal/strategy"
)

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
	strat := strategy.NewEnhancedDCAStrategy(cfg.Strategy.BaseAmount)

	// Add technical indicators
	rsi := indicators.NewRSI(14)
	macd := indicators.NewMACD(12, 26, 9)
	bb := indicators.NewBollingerBands(20, 2.0)
	sma := indicators.NewSMA(50)

	strat.AddIndicator(rsi)
	strat.AddIndicator(macd)
	strat.AddIndicator(bb)
	strat.AddIndicator(sma)

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
