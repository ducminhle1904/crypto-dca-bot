package config

import (
	"fmt"
)

// MarketRegimeConfig holds configuration for market regime detection
type MarketRegimeConfig struct {
	Enabled                     bool    `json:"enabled"`                        // Enable regime-based consensus
	TrendStrengthPeriod         int     `json:"trend_strength_period"`          // Period for trend analysis (default: 20)
	TrendStrengthThreshold      float64 `json:"trend_strength_threshold"`       // Threshold for strong trend (default: 0.75)
	ATRMultiplier               float64 `json:"atr_multiplier"`                 // Dynamic price change threshold multiplier (default: 1.5)
	VolatilityLookback          int     `json:"volatility_lookback"`            // Period for volatility percentile (default: 50)
	LowVolatilityPercentile     float64 `json:"low_volatility_percentile"`      // Low vol threshold (default: 0.3)
	HighVolatilityPercentile    float64 `json:"high_volatility_percentile"`     // High vol threshold (default: 0.8)
	FavorableIndicatorsRequired int     `json:"favorable_indicators_required"`  // Required in favorable conditions (default: 2)
	NormalIndicatorsRequired    int     `json:"normal_indicators_required"`     // Required in normal conditions (default: 3)
	HostileIndicatorsRequired   int     `json:"hostile_indicators_required"`    // Required in hostile conditions (default: 4)
}

// NewDefaultMarketRegimeConfig creates default market regime configuration
func NewDefaultMarketRegimeConfig() *MarketRegimeConfig {
	return &MarketRegimeConfig{
		Enabled:                     false, // Disabled by default for backward compatibility
		TrendStrengthPeriod:         20,
		TrendStrengthThreshold:      0.35, // Reduced from 75% to 65% for crypto markets
		ATRMultiplier:               1.5,  // Dynamic price change threshold (1.5x ATR)
		VolatilityLookback:          50,
		LowVolatilityPercentile:     0.3,
		HighVolatilityPercentile:    0.8,
		FavorableIndicatorsRequired: 2,
		NormalIndicatorsRequired:    3,
		HostileIndicatorsRequired:   4,
	}
}

// Validate validates the market regime configuration
func (c *MarketRegimeConfig) Validate() error {
	if c.TrendStrengthPeriod < 5 {
		return fmt.Errorf("trend strength period must be at least 5, got %d", c.TrendStrengthPeriod)
	}
	
	if c.TrendStrengthThreshold < 0.5 || c.TrendStrengthThreshold > 1.0 {
		return fmt.Errorf("trend strength threshold must be between 0.5 and 1.0, got %.2f", c.TrendStrengthThreshold)
	}
	
	if c.ATRMultiplier < 0.1 || c.ATRMultiplier > 10.0 {
		return fmt.Errorf("ATR multiplier must be between 0.1 and 10.0, got %.2f", c.ATRMultiplier)
	}
	
	if c.VolatilityLookback < 10 {
		return fmt.Errorf("volatility lookback must be at least 10, got %d", c.VolatilityLookback)
	}
	
	if c.LowVolatilityPercentile < 0 || c.LowVolatilityPercentile > 1 {
		return fmt.Errorf("low volatility percentile must be between 0 and 1, got %.2f", c.LowVolatilityPercentile)
	}
	
	if c.HighVolatilityPercentile < 0 || c.HighVolatilityPercentile > 1 {
		return fmt.Errorf("high volatility percentile must be between 0 and 1, got %.2f", c.HighVolatilityPercentile)
	}
	
	if c.LowVolatilityPercentile >= c.HighVolatilityPercentile {
		return fmt.Errorf("low volatility percentile (%.2f) must be less than high volatility percentile (%.2f)", 
			c.LowVolatilityPercentile, c.HighVolatilityPercentile)
	}
	
	if c.FavorableIndicatorsRequired < 1 || c.FavorableIndicatorsRequired > 10 {
		return fmt.Errorf("favorable indicators required must be between 1 and 10, got %d", c.FavorableIndicatorsRequired)
	}
	
	if c.NormalIndicatorsRequired < 1 || c.NormalIndicatorsRequired > 10 {
		return fmt.Errorf("normal indicators required must be between 1 and 10, got %d", c.NormalIndicatorsRequired)
	}
	
	if c.HostileIndicatorsRequired < 1 || c.HostileIndicatorsRequired > 10 {
		return fmt.Errorf("hostile indicators required must be between 1 and 10, got %d", c.HostileIndicatorsRequired)
	}
	
	return nil
}
