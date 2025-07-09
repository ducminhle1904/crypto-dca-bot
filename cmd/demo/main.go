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
	"github.com/Zmey56/enhanced-dca-bot/internal/indicators"
	"github.com/Zmey56/enhanced-dca-bot/internal/monitoring"
	"github.com/Zmey56/enhanced-dca-bot/internal/notifications"
	"github.com/Zmey56/enhanced-dca-bot/internal/risk"
	"github.com/Zmey56/enhanced-dca-bot/internal/strategy"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

func main() {
	log.Println("=== Enhanced DCA Bot Demo ===")

	// Load configuration
	cfg := config.Load()
	log.Printf("Configuration loaded: %s mode", cfg.Environment)

	// Initialize components
	healthChecker := monitoring.NewHealthChecker()
	notifier := notifications.NewTelegramNotifier(cfg.Notifications.TelegramToken, cfg.Notifications.TelegramChatID)

	// Initialize strategy with multiple indicators
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

	log.Printf("Strategy initialized with %d indicators", 4)

	// Initialize risk manager
	riskManager := risk.NewRiskManager(cfg.Strategy.MaxMultiplier)
	log.Println("Risk manager initialized")

	// Setup monitoring servers
	setupMonitoringServers(cfg, healthChecker)

	// Demo trading loop with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute) // Run for 2 minutes
	defer cancel()

	// Start demo loop
	go runDemoLoop(ctx, cfg, strat, riskManager, notifier, healthChecker)

	log.Println("Demo bot started successfully!")
	log.Println("Will run for 2 minutes, then exit automatically")

	// Wait for timeout or interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-ctx.Done():
		log.Println("Demo completed (timeout)")
	case <-sigChan:
		log.Println("Demo interrupted by user")
	}

	log.Println("Demo finished successfully!")
}

func runDemoLoop(ctx context.Context, cfg *config.Config, strat strategy.Strategy, riskManager risk.RiskManager,
	notifier notifications.Notifier, healthChecker *monitoring.HealthChecker) {

	ticker := time.NewTicker(10 * time.Second) // Demo cycle every 10 seconds
	defer ticker.Stop()

	cycle := 0
	maxCycles := 12 // Run for 2 minutes (12 * 10 seconds)

	for {
		select {
		case <-ctx.Done():
			log.Println("Demo loop stopped")
			return
		case <-ticker.C:
			cycle++
			log.Printf("=== Demo Cycle %d/%d ===", cycle, maxCycles)

			// Generate mock market data
			data := generateMockData()

			// Get trading decision
			decision, err := strat.ShouldExecuteTrade(data)
			if err != nil {
				log.Printf("Strategy error: %v", err)
				continue
			}

			// Update health checker
			healthChecker.UpdatePrice(data[len(data)-1].Close)

			// Log decision
			log.Printf("Strategy Decision: %s", decision.Action)
			log.Printf("Confidence: %.2f", decision.Confidence)
			log.Printf("Reason: %s", decision.Reason)
			log.Printf("Current Price: $%.2f", data[len(data)-1].Close)

			// Simulate trade execution
			if decision.Action == strategy.ActionBuy {
				log.Println("🟢 Simulating BUY order...")

				// Update health checker
				healthChecker.UpdateLastTrade(time.Now())

				// Send notification (if configured)
				if cfg.Notifications.TelegramToken != "" {
					message := fmt.Sprintf("Demo BUY signal\nConfidence: %.2f%%\nReason: %s",
						decision.Confidence*100, decision.Reason)
					if err := notifier.SendAlert("success", message); err != nil {
						log.Printf("Failed to send notification: %v", err)
					}
				}

				// Record metrics
				monitoring.RecordTrade("BTCUSDT", "buy", 100.0)
				monitoring.UpdatePrice("BTCUSDT", data[len(data)-1].Close)
				monitoring.UpdateStrategyConfidence("Enhanced DCA", decision.Confidence)

			} else if decision.Action == strategy.ActionSell {
				log.Println("🔴 Simulating SELL order...")

				// Update health checker
				healthChecker.UpdateLastTrade(time.Now())

				// Send notification (if configured)
				if cfg.Notifications.TelegramToken != "" {
					message := fmt.Sprintf("Demo SELL signal\nConfidence: %.2f%%\nReason: %s",
						decision.Confidence*100, decision.Reason)
					if err := notifier.SendAlert("warning", message); err != nil {
						log.Printf("Failed to send notification: %v", err)
					}
				}

				// Record metrics
				monitoring.RecordTrade("BTCUSDT", "sell", 100.0)
				monitoring.UpdatePrice("BTCUSDT", data[len(data)-1].Close)
				monitoring.UpdateStrategyConfidence("Enhanced DCA", decision.Confidence)

			} else {
				log.Println("⚪ HOLD - No action taken")

				// Update metrics
				monitoring.UpdatePrice("BTCUSDT", data[len(data)-1].Close)
				monitoring.UpdateStrategyConfidence("Enhanced DCA", decision.Confidence)
			}

			log.Println("---")

			// Exit after max cycles
			if cycle >= maxCycles {
				log.Println("Demo completed all cycles")
				return
			}
		}
	}
}

func generateMockData() []types.OHLCV {
	// Generate realistic mock data with more variation
	data := make([]types.OHLCV, 100)
	basePrice := 50000.0
	volatility := 0.02

	// Add some trend to make it more interesting
	trend := float64(time.Now().Unix()%100) / 100.0

	for i := 0; i < 100; i++ {
		// Add trend and randomness
		trendComponent := trend * float64(i) * 10
		randomComponent := (float64(i%20) - 10) * volatility * basePrice
		price := basePrice + trendComponent + randomComponent

		// Ensure price is positive
		if price < 1000 {
			price = 1000
		}

		data[i] = types.OHLCV{
			Open:      price * 0.999,
			High:      price * 1.002,
			Low:       price * 0.998,
			Close:     price,
			Volume:    1000.0 + float64(i%100),
			Timestamp: time.Now().Add(time.Duration(-100+i) * time.Hour),
		}
	}

	return data
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
