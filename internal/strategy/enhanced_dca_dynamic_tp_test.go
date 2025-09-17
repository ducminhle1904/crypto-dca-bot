package strategy

import (
	"testing"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/oscillators"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

func createTestData() []types.OHLCV {
	baseTime := time.Now()
	data := make([]types.OHLCV, 50)
	
	for i := 0; i < 50; i++ {
		price := 50000.0 + float64(i*100) // Trending upward
		volatility := 200.0 // Basic volatility
		
		data[i] = types.OHLCV{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Open:      price - volatility/2,
			High:      price + volatility,
			Low:       price - volatility,
			Close:     price,
			Volume:    1000000,
		}
	}
	
	return data
}

func createVolatileTestData() []types.OHLCV {
	baseTime := time.Now()
	data := make([]types.OHLCV, 50)
	
	for i := 0; i < 50; i++ {
		price := 50000.0 + float64(i*50)
		volatility := 1000.0 // High volatility
		
		data[i] = types.OHLCV{
			Timestamp: baseTime.Add(time.Duration(i) * time.Minute),
			Open:      price - volatility/2,
			High:      price + volatility*1.5,
			Low:       price - volatility*1.5,
			Close:     price,
			Volume:    1000000,
		}
	}
	
	return data
}

func TestDynamicTPConfig_Disabled(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Test when no dynamic TP config is set
	if strategy.IsDynamicTPEnabled() {
		t.Error("Dynamic TP should be disabled when no config is set")
	}
	
	data := createTestData()
	currentCandle := data[len(data)-1]
	
	tpPercent, err := strategy.GetDynamicTPPercent(currentCandle, data)
	if err != nil {
		t.Errorf("GetDynamicTPPercent should not error when disabled: %v", err)
	}
	if tpPercent != 0 {
		t.Errorf("Expected TP percent to be 0 when disabled, got: %.4f", tpPercent)
	}
}

func TestDynamicTPConfig_FixedStrategy(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Set up fixed strategy (should be treated as disabled)
	dynamicTPConfig := &config.DynamicTPConfig{
		Strategy:      "fixed",
		BaseTPPercent: 0.02,
	}
	strategy.SetDynamicTPConfig(dynamicTPConfig)
	
	if strategy.IsDynamicTPEnabled() {
		t.Error("Dynamic TP should be disabled for fixed strategy")
	}
	
	data := createTestData()
	currentCandle := data[len(data)-1]
	
	tpPercent, err := strategy.GetDynamicTPPercent(currentCandle, data)
	if err != nil {
		t.Errorf("GetDynamicTPPercent should not error for fixed strategy: %v", err)
	}
	if tpPercent != 0 {
		t.Errorf("Expected TP percent to be 0 for fixed strategy, got: %.4f", tpPercent)
	}
}

func TestVolatilityBasedTP_Basic(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Set up volatility-adaptive TP config
	dynamicTPConfig := &config.DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.02, // 2%
		VolatilityConfig: &config.DynamicTPVolatilityConfig{
			Multiplier:   0.5,   // 0.5x volatility sensitivity
			MinTPPercent: 0.01,  // 1% minimum
			MaxTPPercent: 0.05,  // 5% maximum
			ATRPeriod:    14,
		},
	}
	strategy.SetDynamicTPConfig(dynamicTPConfig)
	
	if !strategy.IsDynamicTPEnabled() {
		t.Error("Dynamic TP should be enabled for volatility_adaptive strategy")
	}
	
	// Test with normal volatility data
	data := createTestData()
	currentCandle := data[len(data)-1]
	
	tpPercent, err := strategy.GetDynamicTPPercent(currentCandle, data)
	if err != nil {
		t.Errorf("GetDynamicTPPercent failed: %v", err)
	}
	
	// Should be between min and max bounds
	if tpPercent < 0.01 || tpPercent > 0.05 {
		t.Errorf("TP percent %.4f is outside bounds [0.01, 0.05]", tpPercent)
	}
	
	// Should be different from base TP due to volatility adjustment
	if tpPercent == 0.02 {
		t.Log("TP percent equals base TP, which is possible but unlikely with volatility adjustment")
	}
}

func TestVolatilityBasedTP_HighVolatility(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Set up volatility-adaptive TP config
	dynamicTPConfig := &config.DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.02, // 2%
		VolatilityConfig: &config.DynamicTPVolatilityConfig{
			Multiplier:   1.0,   // 1.0x volatility sensitivity (higher)
			MinTPPercent: 0.01,  // 1% minimum
			MaxTPPercent: 0.08,  // 8% maximum
			ATRPeriod:    14,
		},
	}
	strategy.SetDynamicTPConfig(dynamicTPConfig)
	
	// Test with high volatility data
	highVolatilityData := createVolatileTestData()
	currentCandle := highVolatilityData[len(highVolatilityData)-1]
	
	tpPercentHighVol, err := strategy.GetDynamicTPPercent(currentCandle, highVolatilityData)
	if err != nil {
		t.Errorf("GetDynamicTPPercent failed for high volatility: %v", err)
	}
	
	// Test with normal volatility data
	normalData := createTestData()
	currentCandle = normalData[len(normalData)-1]
	
	tpPercentNormalVol, err := strategy.GetDynamicTPPercent(currentCandle, normalData)
	if err != nil {
		t.Errorf("GetDynamicTPPercent failed for normal volatility: %v", err)
	}
	
	// High volatility should result in higher TP target
	if tpPercentHighVol <= tpPercentNormalVol {
		t.Errorf("High volatility TP (%.4f) should be higher than normal volatility TP (%.4f)", 
			tpPercentHighVol, tpPercentNormalVol)
	}
}

func TestIndicatorBasedTP_Basic(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Add an RSI indicator to test with
	rsiIndicator := oscillators.NewRSI(14)
	strategy.AddIndicator(rsiIndicator)
	
	// Set up indicator-based TP config
	dynamicTPConfig := &config.DynamicTPConfig{
		Strategy:      "indicator_based",
		BaseTPPercent: 0.02, // 2%
		IndicatorConfig: &config.DynamicTPIndicatorConfig{
			Weights: map[string]float64{
				"RSI": 1.0, // Full weight on RSI
			},
			StrengthMultiplier: 0.3,   // 30% strength adjustment
			MinTPPercent:       0.01,  // 1% minimum
			MaxTPPercent:       0.04,  // 4% maximum
		},
	}
	strategy.SetDynamicTPConfig(dynamicTPConfig)
	
	if !strategy.IsDynamicTPEnabled() {
		t.Error("Dynamic TP should be enabled for indicator_based strategy")
	}
	
	data := createTestData()
	currentCandle := data[len(data)-1]
	
	tpPercent, err := strategy.GetDynamicTPPercent(currentCandle, data)
	if err != nil {
		t.Errorf("GetDynamicTPPercent failed: %v", err)
	}
	
	// Should be between min and max bounds
	if tpPercent < 0.01 || tpPercent > 0.04 {
		t.Errorf("TP percent %.4f is outside bounds [0.01, 0.04]", tpPercent)
	}
}

func TestIndicatorBasedTP_MultipleIndicators(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Add multiple indicators
	rsiIndicator := oscillators.NewRSI(14)
	strategy.AddIndicator(rsiIndicator)
	
	// Set up indicator-based TP config with multiple weights
	dynamicTPConfig := &config.DynamicTPConfig{
		Strategy:      "indicator_based",
		BaseTPPercent: 0.025, // 2.5%
		IndicatorConfig: &config.DynamicTPIndicatorConfig{
			Weights: map[string]float64{
				"RSI":  0.6, // 60% weight on RSI
				"MACD": 0.4, // 40% weight on MACD (even if not present)
			},
			StrengthMultiplier: 0.4,   // 40% strength adjustment
			MinTPPercent:       0.015, // 1.5% minimum
			MaxTPPercent:       0.04,  // 4% maximum
		},
	}
	strategy.SetDynamicTPConfig(dynamicTPConfig)
	
	data := createTestData()
	currentCandle := data[len(data)-1]
	
	tpPercent, err := strategy.GetDynamicTPPercent(currentCandle, data)
	if err != nil {
		t.Errorf("GetDynamicTPPercent failed: %v", err)
	}
	
	// Should be between min and max bounds
	if tpPercent < 0.015 || tpPercent > 0.04 {
		t.Errorf("TP percent %.4f is outside bounds [0.015, 0.04]", tpPercent)
	}
}

func TestConvertSignalToTPStrength(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	tests := []struct {
		signalType     indicators.SignalType
		signalStrength float64
		expectedMin    float64
		expectedMax    float64
	}{
		{indicators.SignalBuy, 0.0, 0.2, 0.2},   // Minimum buy strength
		{indicators.SignalBuy, 1.0, 1.0, 1.0},   // Maximum buy strength
		{indicators.SignalSell, 0.0, -0.2, -0.2}, // Minimum sell strength
		{indicators.SignalSell, 1.0, -1.0, -1.0}, // Maximum sell strength
		{indicators.SignalHold, 0.5, 0.0, 0.0},   // Hold always 0
	}
	
	for _, test := range tests {
		signal := indicators.Signal{
			Type:     test.signalType,
			Strength: test.signalStrength,
		}
		strength := strategy.convertSignalToTPStrength(signal)
		if strength < test.expectedMin || strength > test.expectedMax {
			t.Errorf("Expected strength between %.1f and %.1f for signal %v (strength=%.1f), got %.1f", 
				test.expectedMin, test.expectedMax, test.signalType, test.signalStrength, strength)
		}
	}
}

func TestValidateDynamicTPConfig(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Test nil config
	err := strategy.validateDynamicTPConfig()
	if err == nil {
		t.Error("Should error with nil config")
	}
	
	// Test invalid base TP percent
	invalidConfig := &config.DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: -0.01, // Invalid negative
	}
	strategy.SetDynamicTPConfig(invalidConfig)
	
	err = strategy.validateDynamicTPConfig()
	if err == nil {
		t.Error("Should error with negative base TP percent")
	}
	
	// Test valid volatility config
	validVolatilityConfig := &config.DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.02,
		VolatilityConfig: &config.DynamicTPVolatilityConfig{
			Multiplier:   0.5,
			MinTPPercent: 0.01,
			MaxTPPercent: 0.05,
			ATRPeriod:    14,
		},
	}
	strategy.SetDynamicTPConfig(validVolatilityConfig)
	
	err = strategy.validateDynamicTPConfig()
	if err != nil {
		t.Errorf("Valid volatility config should not error: %v", err)
	}
	
	// Test invalid volatility config (min >= max)
	invalidVolatilityConfig := &config.DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.02,
		VolatilityConfig: &config.DynamicTPVolatilityConfig{
			Multiplier:   0.5,
			MinTPPercent: 0.05, // Min > Max
			MaxTPPercent: 0.03,
			ATRPeriod:    14,
		},
	}
	strategy.SetDynamicTPConfig(invalidVolatilityConfig)
	
	err = strategy.validateDynamicTPConfig()
	if err == nil {
		t.Error("Should error when min TP >= max TP")
	}
	
	// Test valid indicator config
	validIndicatorConfig := &config.DynamicTPConfig{
		Strategy:      "indicator_based",
		BaseTPPercent: 0.02,
		IndicatorConfig: &config.DynamicTPIndicatorConfig{
			Weights: map[string]float64{
				"RSI": 1.0,
			},
			StrengthMultiplier: 0.3,
			MinTPPercent:       0.01,
			MaxTPPercent:       0.04,
		},
	}
	strategy.SetDynamicTPConfig(validIndicatorConfig)
	
	err = strategy.validateDynamicTPConfig()
	if err != nil {
		t.Errorf("Valid indicator config should not error: %v", err)
	}
}

func TestDynamicTPBounds(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Test volatility-based TP bounds enforcement
	dynamicTPConfig := &config.DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.02,
		VolatilityConfig: &config.DynamicTPVolatilityConfig{
			Multiplier:   10.0,  // Very high multiplier to test bounds
			MinTPPercent: 0.015, // 1.5% minimum
			MaxTPPercent: 0.025, // 2.5% maximum (tight bounds)
			ATRPeriod:    14,
		},
	}
	strategy.SetDynamicTPConfig(dynamicTPConfig)
	
	// Use high volatility data that would normally result in TP > max
	highVolData := createVolatileTestData()
	currentCandle := highVolData[len(highVolData)-1]
	
	tpPercent, err := strategy.GetDynamicTPPercent(currentCandle, highVolData)
	if err != nil {
		t.Errorf("GetDynamicTPPercent failed: %v", err)
	}
	
	// Should be clamped to max bound
	if tpPercent > 0.025 {
		t.Errorf("TP percent %.4f exceeds maximum bound 0.025", tpPercent)
	}
	if tpPercent < 0.015 {
		t.Errorf("TP percent %.4f below minimum bound 0.015", tpPercent)
	}
}

func TestDynamicTPConfigGettersSetters(t *testing.T) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Test initial state
	if strategy.GetDynamicTPConfig() != nil {
		t.Error("Initial dynamic TP config should be nil")
	}
	
	// Test setter and getter
	config := &config.DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.03,
	}
	
	strategy.SetDynamicTPConfig(config)
	
	retrievedConfig := strategy.GetDynamicTPConfig()
	if retrievedConfig == nil {
		t.Error("Retrieved config should not be nil")
	}
	if retrievedConfig.Strategy != "volatility_adaptive" {
		t.Errorf("Expected strategy 'volatility_adaptive', got '%s'", retrievedConfig.Strategy)
	}
	if retrievedConfig.BaseTPPercent != 0.03 {
		t.Errorf("Expected base TP percent 0.03, got %.4f", retrievedConfig.BaseTPPercent)
	}
}

// Benchmark tests for performance validation
func BenchmarkVolatilityBasedTP(b *testing.B) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	dynamicTPConfig := &config.DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.02,
		VolatilityConfig: &config.DynamicTPVolatilityConfig{
			Multiplier:   0.5,
			MinTPPercent: 0.01,
			MaxTPPercent: 0.05,
			ATRPeriod:    14,
		},
	}
	strategy.SetDynamicTPConfig(dynamicTPConfig)
	
	data := createTestData()
	currentCandle := data[len(data)-1]
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := strategy.GetDynamicTPPercent(currentCandle, data)
		if err != nil {
			b.Errorf("Benchmark failed: %v", err)
		}
	}
}

func BenchmarkIndicatorBasedTP(b *testing.B) {
	strategy := NewEnhancedDCAStrategy(40.0)
	
	// Add RSI indicator
	rsiIndicator := oscillators.NewRSI(14)
	strategy.AddIndicator(rsiIndicator)
	
	dynamicTPConfig := &config.DynamicTPConfig{
		Strategy:      "indicator_based",
		BaseTPPercent: 0.02,
		IndicatorConfig: &config.DynamicTPIndicatorConfig{
			Weights: map[string]float64{
				"RSI": 1.0,
			},
			StrengthMultiplier: 0.3,
			MinTPPercent:       0.01,
			MaxTPPercent:       0.04,
		},
	}
	strategy.SetDynamicTPConfig(dynamicTPConfig)
	
	data := createTestData()
	currentCandle := data[len(data)-1]
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := strategy.GetDynamicTPPercent(currentCandle, data)
		if err != nil {
			b.Errorf("Benchmark failed: %v", err)
		}
	}
}
