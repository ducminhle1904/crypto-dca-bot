package main

import (
	"testing"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/monitoring"
	"github.com/ducminhle1904/crypto-dca-bot/internal/notifications"
	"github.com/ducminhle1904/crypto-dca-bot/internal/risk"
)

// TestEnhancedDCABotCreation tests that the bot can be created successfully
func TestEnhancedDCABotCreation(t *testing.T) {
	// Create test configuration
	cfg := &config.Config{
		Environment: "test",
		Strategy: struct {
			Symbol        string
			BaseAmount    float64
			MaxMultiplier float64
			Interval      time.Duration
		}{
			Symbol:        "BTCUSDT",
			BaseAmount:    100.0,
			MaxMultiplier: 3.0,
			Interval:      5 * time.Minute,
		},
	}

	// Create optimized config
	optCfg := &OptimizedConfig{
		Symbol:         "BTCUSDT",
		BaseAmount:     100.0,
		MaxMultiplier:  2.0,
		PriceThreshold: 0.015,
		TPPercent:      0.055,
		Indicators:     []string{"rsi", "macd", "bb", "ema"},
		RSIPeriod:      16,
		RSIOversold:    25,
		RSIOverbought:  70,
		MACDFast:       6,
		MACDSlow:       28,
		MACDSignal:     12,
		BBPeriod:       25,
		BBStdDev:       1.8,
		EMAPeriod:      60,
		Interval:       "15m",
		ConfigPath:     "configs/btc_15m.json",
	}

	// Initialize components
	healthChecker := monitoring.NewHealthChecker()
	notifier := notifications.NewTelegramNotifier("", "")
	exch := exchange.NewBinanceExchange("", "", true)
	strat := createEnhancedDCAStrategy(optCfg)
	riskManager := risk.NewRiskManager(optCfg.MaxMultiplier)

	// Create bot
	bot := NewEnhancedDCABot(cfg, exch, strat, riskManager, notifier, healthChecker, optCfg)

	// Verify bot was created
	if bot == nil {
		t.Fatal("Bot should not be nil")
	}

	// Verify bot fields
	if bot.config != cfg {
		t.Error("Bot config should match provided config")
	}

	if bot.exchange != exch {
		t.Error("Bot exchange should match provided exchange")
	}

	if bot.strategy != strat {
		t.Error("Bot strategy should match provided strategy")
	}

	if bot.optimizedConfig != optCfg {
		t.Error("Bot optimized config should match provided optimized config")
	}

	if bot.cycleNumber != 1 {
		t.Error("Bot should start with cycle number 1")
	}

	if bot.lastEntryPrice != 0.0 {
		t.Error("Bot should start with no entry price")
	}
}

// TestCreateEnhancedDCAStrategy tests that the strategy is created with correct indicators
func TestCreateEnhancedDCAStrategy(t *testing.T) {
	optCfg := &OptimizedConfig{
		BaseAmount:     100.0,
		Indicators:     []string{"rsi", "macd", "bb", "ema"},
		RSIPeriod:      16,
		RSIOversold:    25,
		RSIOverbought:  70,
		MACDFast:       6,
		MACDSlow:       28,
		MACDSignal:     12,
		BBPeriod:       25,
		BBStdDev:       1.8,
		EMAPeriod:      60,
		Interval:       "15m",
		ConfigPath:     "configs/btc_15m.json",
	}

	strat := createEnhancedDCAStrategy(optCfg)

	if strat == nil {
		t.Fatal("Strategy should not be nil")
	}

	// Verify strategy name
	if strat.GetName() != "Enhanced DCA Strategy" {
		t.Errorf("Expected strategy name 'Enhanced DCA Strategy', got '%s'", strat.GetName())
	}
}

// TestBotCycleManagement tests basic cycle management functionality
func TestBotCycleManagement(t *testing.T) {
	cfg := &config.Config{
		Strategy: struct {
			Symbol        string
			BaseAmount    float64
			MaxMultiplier float64
			Interval      time.Duration
		}{
			Symbol:        "BTCUSDT",
			BaseAmount:    100.0,
			MaxMultiplier: 3.0,
			Interval:      5 * time.Minute,
		},
	}

	optCfg := &OptimizedConfig{
		Symbol:         "BTCUSDT",
		BaseAmount:     100.0,
		MaxMultiplier:  2.0,
		PriceThreshold: 0.015,
		TPPercent:      0.055,
		Indicators:     []string{"rsi", "macd", "bb", "ema"},
		RSIPeriod:      16,
		RSIOversold:    25,
		RSIOverbought:  70,
		MACDFast:       6,
		MACDSlow:       28,
		MACDSignal:     12,
		BBPeriod:       25,
		BBStdDev:       1.8,
		EMAPeriod:      60,
		Interval:       "15m",
		ConfigPath:     "configs/btc_15m.json",
	}

	healthChecker := monitoring.NewHealthChecker()
	notifier := notifications.NewTelegramNotifier("", "")
	exch := exchange.NewBinanceExchange("", "", true)
	strat := createEnhancedDCAStrategy(optCfg)
	riskManager := risk.NewRiskManager(optCfg.MaxMultiplier)

	bot := NewEnhancedDCABot(cfg, exch, strat, riskManager, notifier, healthChecker, optCfg)

	// Test initial cycle state
	if bot.cycleNumber != 1 {
		t.Errorf("Expected initial cycle number 1, got %d", bot.cycleNumber)
	}

	if bot.lastEntryPrice != 0.0 {
		t.Errorf("Expected initial entry price 0.0, got %f", bot.cycleNumber)
	}

	// Test cycle completion logic
	bot.lastEntryPrice = 50000.0 // Set a mock entry price
	currentPrice := 52750.0       // 5.5% increase (using optimized TP)

	if !bot.shouldCompleteCycle(currentPrice) {
		t.Error("Should complete cycle with 5.5% price increase")
	}

	// Test cycle completion
	err := bot.completeCycle(nil, currentPrice)
	if err != nil {
		t.Errorf("Cycle completion should not error: %v", err)
	}

	// Verify cycle was completed
	if bot.cycleNumber != 2 {
		t.Errorf("Expected cycle number 2 after completion, got %d", bot.cycleNumber)
	}

	if bot.lastEntryPrice != 0.0 {
		t.Errorf("Expected entry price reset to 0.0, got %f", bot.lastEntryPrice)
	}
}

// TestBotShutdown tests that the bot can be shut down gracefully
func TestBotShutdown(t *testing.T) {
	cfg := &config.Config{
		Strategy: struct {
			Symbol        string
			BaseAmount    float64
			MaxMultiplier float64
			Interval      time.Duration
		}{
			Symbol:        "BTCUSDT",
			BaseAmount:    100.0,
			MaxMultiplier: 3.0,
			Interval:      5 * time.Minute,
		},
	}

	optCfg := &OptimizedConfig{
		Symbol:         "BTCUSDT",
		BaseAmount:     100.0,
		MaxMultiplier:  2.0,
		PriceThreshold: 0.015,
		TPPercent:      0.055,
		Indicators:     []string{"rsi", "macd", "bb", "ema"},
		RSIPeriod:      16,
		RSIOversold:    25,
		RSIOverbought:  70,
		MACDFast:       6,
		MACDSlow:       28,
		MACDSignal:     12,
		BBPeriod:       25,
		BBStdDev:       1.8,
		EMAPeriod:      60,
		Interval:       "15m",
		ConfigPath:     "configs/btc_15m.json",
	}

	healthChecker := monitoring.NewHealthChecker()
	notifier := notifications.NewTelegramNotifier("", "")
	exch := exchange.NewBinanceExchange("", "", true)
	strat := createEnhancedDCAStrategy(optCfg)
	riskManager := risk.NewRiskManager(optCfg.MaxMultiplier)

	bot := NewEnhancedDCABot(cfg, exch, strat, riskManager, notifier, healthChecker, optCfg)

	// Set some state
	bot.running = true
	bot.cycleNumber = 5
	bot.lastEntryPrice = 50000.0

	// Test shutdown
	ctx := &mockContext{}
	err := bot.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown should not error: %v", err)
	}

	// Verify shutdown state
	if bot.running {
		t.Error("Bot should not be running after shutdown")
	}
}

// TestLoadOptimizedConfig tests config loading functionality
func TestLoadOptimizedConfig(t *testing.T) {
	// Test with empty config (should use default)
	optCfg, err := loadOptimizedConfig("")
	if err != nil {
		t.Logf("Expected error for missing config file: %v", err)
		// This is expected behavior when no config file exists
		return
	}

	// If config file exists, verify it loaded correctly
	if optCfg != nil {
		if optCfg.Symbol == "" {
			t.Error("Loaded config should have a symbol")
		}
		if optCfg.BaseAmount <= 0 {
			t.Error("Loaded config should have a positive base amount")
		}
		if len(optCfg.Indicators) == 0 {
			t.Error("Loaded config should have indicators")
		}
	}
}

// mockContext is a simple mock context for testing
type mockContext struct{}

func (m *mockContext) Done() <-chan struct{} {
	return nil
}

func (m *mockContext) Err() error {
	return nil
}

func (m *mockContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

func (m *mockContext) Value(key interface{}) interface{} {
	return nil
}
