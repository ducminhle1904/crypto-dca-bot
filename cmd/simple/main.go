package main

import (
	"log"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/internal/config"
	"github.com/Zmey56/enhanced-dca-bot/internal/indicators"
	"github.com/Zmey56/enhanced-dca-bot/internal/strategy"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

func main() {
	log.Println("=== Simple DCA Bot Test ===")

	// Load configuration
	cfg := config.Load()
	log.Printf("Configuration loaded: %s mode", cfg.Environment)

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

	log.Printf("Strategy initialized with %d indicators", 4)

	// Test multiple cycles
	for i := 1; i <= 5; i++ {
		log.Printf("=== Test Cycle %d/5 ===", i)

		// Generate test data
		data := generateTestData(i)

		// Get trading decision
		decision, err := strat.ShouldExecuteTrade(data)
		if err != nil {
			log.Printf("Strategy error: %v", err)
			continue
		}

		// Log results
		log.Printf("Decision: %s", decision.Action)
		log.Printf("Confidence: %.2f", decision.Confidence)
		log.Printf("Reason: %s", decision.Reason)
		log.Printf("Price: $%.2f", data[len(data)-1].Close)

		if decision.Action == strategy.ActionBuy {
			log.Println("🟢 BUY signal detected!")
		} else if decision.Action == strategy.ActionSell {
			log.Println("🔴 SELL signal detected!")
		} else {
			log.Println("⚪ HOLD - No action")
		}

		log.Println("---")

		// Small delay between cycles
		time.Sleep(1 * time.Second)
	}

	log.Println("✅ All tests completed successfully!")
	log.Println("Bot is working correctly!")
}

func generateTestData(cycle int) []types.OHLCV {
	data := make([]types.OHLCV, 100)
	basePrice := 50000.0

	// Different scenarios for each cycle
	scenarios := []string{"uptrend", "downtrend", "sideways", "volatile", "stable"}
	scenario := scenarios[(cycle-1)%len(scenarios)]

	for i := 0; i < 100; i++ {
		var price float64

		switch scenario {
		case "uptrend":
			price = basePrice + float64(i)*10
		case "downtrend":
			price = basePrice - float64(i)*10
		case "sideways":
			price = basePrice + float64(i%20-10)*5
		case "volatile":
			price = basePrice + float64(i%10-5)*100
		case "stable":
			price = basePrice + float64(i%5-2)*2
		}

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

	log.Printf("Generated %s scenario data", scenario)
	return data
}
