package config

import (
	"encoding/json"
	"testing"
)

func TestDynamicTPConfig_JSONSerialization(t *testing.T) {
	// Test volatility-adaptive configuration
	volatilityConfig := &DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.02,
		VolatilityConfig: &DynamicTPVolatilityConfig{
			Multiplier:   0.5,
			MinTPPercent: 0.01,
			MaxTPPercent: 0.05,
			ATRPeriod:    14,
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(volatilityConfig)
	if err != nil {
		t.Errorf("Failed to marshal volatility config: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledConfig DynamicTPConfig
	err = json.Unmarshal(jsonData, &unmarshaledConfig)
	if err != nil {
		t.Errorf("Failed to unmarshal volatility config: %v", err)
	}

	// Verify values
	if unmarshaledConfig.Strategy != "volatility_adaptive" {
		t.Errorf("Expected strategy 'volatility_adaptive', got '%s'", unmarshaledConfig.Strategy)
	}
	if unmarshaledConfig.BaseTPPercent != 0.02 {
		t.Errorf("Expected base TP percent 0.02, got %.4f", unmarshaledConfig.BaseTPPercent)
	}
	if unmarshaledConfig.VolatilityConfig == nil {
		t.Error("Volatility config should not be nil")
	} else {
		vc := unmarshaledConfig.VolatilityConfig
		if vc.Multiplier != 0.5 {
			t.Errorf("Expected multiplier 0.5, got %.2f", vc.Multiplier)
		}
		if vc.MinTPPercent != 0.01 {
			t.Errorf("Expected min TP percent 0.01, got %.4f", vc.MinTPPercent)
		}
		if vc.MaxTPPercent != 0.05 {
			t.Errorf("Expected max TP percent 0.05, got %.4f", vc.MaxTPPercent)
		}
		if vc.ATRPeriod != 14 {
			t.Errorf("Expected ATR period 14, got %d", vc.ATRPeriod)
		}
	}
}

func TestDynamicTPConfig_IndicatorBasedSerialization(t *testing.T) {
	// Test indicator-based configuration
	indicatorConfig := &DynamicTPConfig{
		Strategy:      "indicator_based",
		BaseTPPercent: 0.025,
		IndicatorConfig: &DynamicTPIndicatorConfig{
			Weights: map[string]float64{
				"rsi":  0.4,
				"macd": 0.3,
				"bb":   0.3,
			},
			StrengthMultiplier: 0.4,
			MinTPPercent:       0.015,
			MaxTPPercent:       0.04,
		},
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(indicatorConfig)
	if err != nil {
		t.Errorf("Failed to marshal indicator config: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledConfig DynamicTPConfig
	err = json.Unmarshal(jsonData, &unmarshaledConfig)
	if err != nil {
		t.Errorf("Failed to unmarshal indicator config: %v", err)
	}

	// Verify values
	if unmarshaledConfig.Strategy != "indicator_based" {
		t.Errorf("Expected strategy 'indicator_based', got '%s'", unmarshaledConfig.Strategy)
	}
	if unmarshaledConfig.BaseTPPercent != 0.025 {
		t.Errorf("Expected base TP percent 0.025, got %.4f", unmarshaledConfig.BaseTPPercent)
	}
	if unmarshaledConfig.IndicatorConfig == nil {
		t.Error("Indicator config should not be nil")
	} else {
		ic := unmarshaledConfig.IndicatorConfig
		if ic.StrengthMultiplier != 0.4 {
			t.Errorf("Expected strength multiplier 0.4, got %.2f", ic.StrengthMultiplier)
		}
		if ic.MinTPPercent != 0.015 {
			t.Errorf("Expected min TP percent 0.015, got %.4f", ic.MinTPPercent)
		}
		if ic.MaxTPPercent != 0.04 {
			t.Errorf("Expected max TP percent 0.04, got %.4f", ic.MaxTPPercent)
		}
		
		// Check weights
		if len(ic.Weights) != 3 {
			t.Errorf("Expected 3 indicator weights, got %d", len(ic.Weights))
		}
		if ic.Weights["rsi"] != 0.4 {
			t.Errorf("Expected RSI weight 0.4, got %.2f", ic.Weights["rsi"])
		}
		if ic.Weights["macd"] != 0.3 {
			t.Errorf("Expected MACD weight 0.3, got %.2f", ic.Weights["macd"])
		}
		if ic.Weights["bb"] != 0.3 {
			t.Errorf("Expected BB weight 0.3, got %.2f", ic.Weights["bb"])
		}
	}
}

func TestDynamicTPConfig_FixedStrategy(t *testing.T) {
	// Test fixed strategy (minimal config)
	fixedConfig := &DynamicTPConfig{
		Strategy:      "fixed",
		BaseTPPercent: 0.02,
	}

	// Test JSON marshaling
	jsonData, err := json.Marshal(fixedConfig)
	if err != nil {
		t.Errorf("Failed to marshal fixed config: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaledConfig DynamicTPConfig
	err = json.Unmarshal(jsonData, &unmarshaledConfig)
	if err != nil {
		t.Errorf("Failed to unmarshal fixed config: %v", err)
	}

	// Verify values
	if unmarshaledConfig.Strategy != "fixed" {
		t.Errorf("Expected strategy 'fixed', got '%s'", unmarshaledConfig.Strategy)
	}
	if unmarshaledConfig.BaseTPPercent != 0.02 {
		t.Errorf("Expected base TP percent 0.02, got %.4f", unmarshaledConfig.BaseTPPercent)
	}
	
	// Both configs should be nil for fixed strategy
	if unmarshaledConfig.VolatilityConfig != nil {
		t.Error("Volatility config should be nil for fixed strategy")
	}
	if unmarshaledConfig.IndicatorConfig != nil {
		t.Error("Indicator config should be nil for fixed strategy")
	}
}

func TestDCAConfig_DynamicTPHelpers(t *testing.T) {
	config := NewDefaultDCAConfig()
	
	// Test initial state
	if config.HasDynamicTP() {
		t.Error("Default config should not have dynamic TP")
	}
	if config.IsDynamicTPEnabled() {
		t.Error("Default config should not have dynamic TP enabled")
	}
	if config.GetDynamicTPConfig() != nil {
		t.Error("Default config should have nil dynamic TP config")
	}
	
	// Test setting dynamic TP config
	dynamicTPConfig := &DynamicTPConfig{
		Strategy:      "volatility_adaptive",
		BaseTPPercent: 0.03,
		VolatilityConfig: &DynamicTPVolatilityConfig{
			Multiplier:   0.8,
			MinTPPercent: 0.015,
			MaxTPPercent: 0.06,
			ATRPeriod:    21,
		},
	}
	
	config.SetDynamicTPConfig(dynamicTPConfig)
	
	// Test enabled state
	if !config.HasDynamicTP() {
		t.Error("Config should have dynamic TP after setting")
	}
	if !config.IsDynamicTPEnabled() {
		t.Error("Config should have dynamic TP enabled after setting")
	}
	
	// Test getter
	retrievedConfig := config.GetDynamicTPConfig()
	if retrievedConfig == nil {
		t.Error("Retrieved config should not be nil")
	}
	if retrievedConfig.Strategy != "volatility_adaptive" {
		t.Errorf("Expected strategy 'volatility_adaptive', got '%s'", retrievedConfig.Strategy)
	}
	if retrievedConfig.BaseTPPercent != 0.03 {
		t.Errorf("Expected base TP percent 0.03, got %.4f", retrievedConfig.BaseTPPercent)
	}
	
	// Test fixed strategy (should not be considered enabled)
	fixedConfig := &DynamicTPConfig{
		Strategy:      "fixed",
		BaseTPPercent: 0.02,
	}
	config.SetDynamicTPConfig(fixedConfig)
	
	if config.HasDynamicTP() {
		t.Error("Fixed strategy should not be considered dynamic TP")
	}
	if config.IsDynamicTPEnabled() {
		t.Error("Fixed strategy should not be considered enabled")
	}
}

func TestDynamicTPConfig_CompleteJSONExample(t *testing.T) {
	// Test complete configuration as it would appear in a JSON file
	jsonString := `{
		"strategy": "volatility_adaptive",
		"base_tp_percent": 0.025,
		"volatility_config": {
			"multiplier": 0.6,
			"min_tp_percent": 0.012,
			"max_tp_percent": 0.055,
			"atr_period": 20
		}
	}`
	
	var config DynamicTPConfig
	err := json.Unmarshal([]byte(jsonString), &config)
	if err != nil {
		t.Errorf("Failed to unmarshal complete JSON: %v", err)
	}
	
	// Verify all values
	if config.Strategy != "volatility_adaptive" {
		t.Errorf("Expected strategy 'volatility_adaptive', got '%s'", config.Strategy)
	}
	if config.BaseTPPercent != 0.025 {
		t.Errorf("Expected base TP percent 0.025, got %.4f", config.BaseTPPercent)
	}
	if config.VolatilityConfig == nil {
		t.Error("Volatility config should not be nil")
	} else {
		vc := config.VolatilityConfig
		if vc.Multiplier != 0.6 {
			t.Errorf("Expected multiplier 0.6, got %.2f", vc.Multiplier)
		}
		if vc.MinTPPercent != 0.012 {
			t.Errorf("Expected min TP percent 0.012, got %.4f", vc.MinTPPercent)
		}
		if vc.MaxTPPercent != 0.055 {
			t.Errorf("Expected max TP percent 0.055, got %.4f", vc.MaxTPPercent)
		}
		if vc.ATRPeriod != 20 {
			t.Errorf("Expected ATR period 20, got %d", vc.ATRPeriod)
		}
	}
}

func TestDynamicTPConfig_IndicatorCompleteJSON(t *testing.T) {
	// Test complete indicator-based configuration
	jsonString := `{
		"strategy": "indicator_based",
		"base_tp_percent": 0.03,
		"indicator_config": {
			"weights": {
				"rsi": 0.25,
				"macd": 0.25,
				"bb": 0.2,
				"hull_ma": 0.15,
				"stochrsi": 0.15
			},
			"strength_multiplier": 0.35,
			"min_tp_percent": 0.018,
			"max_tp_percent": 0.045
		}
	}`
	
	var config DynamicTPConfig
	err := json.Unmarshal([]byte(jsonString), &config)
	if err != nil {
		t.Errorf("Failed to unmarshal indicator JSON: %v", err)
	}
	
	// Verify all values
	if config.Strategy != "indicator_based" {
		t.Errorf("Expected strategy 'indicator_based', got '%s'", config.Strategy)
	}
	if config.BaseTPPercent != 0.03 {
		t.Errorf("Expected base TP percent 0.03, got %.4f", config.BaseTPPercent)
	}
	if config.IndicatorConfig == nil {
		t.Error("Indicator config should not be nil")
	} else {
		ic := config.IndicatorConfig
		if ic.StrengthMultiplier != 0.35 {
			t.Errorf("Expected strength multiplier 0.35, got %.2f", ic.StrengthMultiplier)
		}
		if ic.MinTPPercent != 0.018 {
			t.Errorf("Expected min TP percent 0.018, got %.4f", ic.MinTPPercent)
		}
		if ic.MaxTPPercent != 0.045 {
			t.Errorf("Expected max TP percent 0.045, got %.4f", ic.MaxTPPercent)
		}
		
		// Check all weights
		expectedWeights := map[string]float64{
			"rsi":      0.25,
			"macd":     0.25,
			"bb":       0.2,
			"hull_ma":  0.15,
			"stochrsi": 0.15,
		}
		
		if len(ic.Weights) != len(expectedWeights) {
			t.Errorf("Expected %d indicator weights, got %d", len(expectedWeights), len(ic.Weights))
		}
		
		for name, expectedWeight := range expectedWeights {
			if actualWeight, exists := ic.Weights[name]; !exists {
				t.Errorf("Expected weight for %s not found", name)
			} else if actualWeight != expectedWeight {
				t.Errorf("Expected weight %.2f for %s, got %.2f", expectedWeight, name, actualWeight)
			}
		}
	}
}

func TestDynamicTPConfig_OmitEmptyFields(t *testing.T) {
	// Test that omitempty works correctly
	config := &DynamicTPConfig{
		Strategy:      "fixed",
		BaseTPPercent: 0.02,
		// Both volatility and indicator configs are nil (should be omitted)
	}
	
	jsonData, err := json.Marshal(config)
	if err != nil {
		t.Errorf("Failed to marshal config with nil sub-configs: %v", err)
	}
	
	jsonString := string(jsonData)
	
	// Should not contain the omitted fields
	if contains(jsonString, "volatility_config") {
		t.Error("JSON should not contain volatility_config when nil")
	}
	if contains(jsonString, "indicator_config") {
		t.Error("JSON should not contain indicator_config when nil")
	}
	
	// Should contain the non-nil fields
	if !contains(jsonString, "strategy") {
		t.Error("JSON should contain strategy field")
	}
	if !contains(jsonString, "base_tp_percent") {
		t.Error("JSON should contain base_tp_percent field")
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr, 1)))
}

func containsAt(s, substr string, start int) bool {
	if start >= len(s) {
		return false
	}
	if len(s)-start < len(substr) {
		return false
	}
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
