package main

import (
	"testing"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/monitoring"
	"github.com/ducminhle1904/crypto-dca-bot/internal/notifications"
	"github.com/ducminhle1904/crypto-dca-bot/internal/risk"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

func TestBotComponents(t *testing.T) {
	// Test configuration loading
	cfg := config.Load()
	if cfg == nil {
		t.Fatal("Failed to load configuration")
	}

	// Test health checker
	healthChecker := monitoring.NewHealthChecker()
	if healthChecker == nil {
		t.Fatal("Failed to create health checker")
	}

	// Test notifier (with empty token)
	notifier := notifications.NewTelegramNotifier("", "")
	if notifier == nil {
		t.Fatal("Failed to create notifier")
	}

	// Test strategy
	strat := strategy.NewEnhancedDCAStrategy(100.0)
	if strat == nil {
		t.Fatal("Failed to create strategy")
	}

	// Test indicators
	rsi := indicators.NewRSI(14)
	macd := indicators.NewMACD(12, 26, 9)
	bb := indicators.NewBollingerBands(20, 2.0)
	sma := indicators.NewSMA(50)

	strat.AddIndicator(rsi)
	strat.AddIndicator(macd)
	strat.AddIndicator(bb)
	strat.AddIndicator(sma)

	// Test risk manager
	riskManager := risk.NewRiskManager(3.0)
	if riskManager == nil {
		t.Fatal("Failed to create risk manager")
	}

	t.Log("All components initialized successfully")
}

func TestStrategyDecision(t *testing.T) {
	strat := strategy.NewEnhancedDCAStrategy(100.0)

	// Add indicators
	rsi := indicators.NewRSI(14)
	strat.AddIndicator(rsi)

	// Create test data
	data := make([]types.OHLCV, 50)
	for i := 0; i < 50; i++ {
		data[i] = types.OHLCV{
			Open:      100.0,
			High:      105.0,
			Low:       95.0,
			Close:     100.0 + float64(i%10),
			Volume:    1000.0,
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
		}
	}

	// Test strategy decision
	decision, err := strat.ShouldExecuteTrade(data)
	if err != nil {
		t.Fatalf("Strategy decision failed: %v", err)
	}

	if decision == nil {
		t.Fatal("Strategy returned nil decision")
	}

	t.Logf("Strategy decision: %s, confidence: %.2f, reason: %s",
		decision.Action, decision.Confidence, decision.Reason)
}
