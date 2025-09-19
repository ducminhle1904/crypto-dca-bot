package strategy

import (
	"fmt"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// SimpleMarketRegime represents basic market conditions for signal consensus adjustment
type SimpleMarketRegime int

const (
	RegimeFavorable SimpleMarketRegime = iota // 2 indicators needed (Bull + Low Vol)
	RegimeNormal                              // 3 indicators needed (Mixed conditions)
	RegimeHostile                             // 4 indicators needed (Bear + High Vol)
)

func (r SimpleMarketRegime) String() string {
	switch r {
	case RegimeFavorable:
		return "Favorable"
	case RegimeNormal:
		return "Normal"
	case RegimeHostile:
		return "Hostile"
	default:
		return "Unknown"
	}
}

// MarketRegimeConfig holds configuration for market regime detection
type MarketRegimeConfig struct {
	Enabled                    bool    `json:"enabled"`                        // Enable regime-based consensus
	TrendStrengthPeriod        int     `json:"trend_strength_period"`          // Period for trend analysis (default: 20)
	TrendStrengthThreshold     float64 `json:"trend_strength_threshold"`       // Threshold for strong trend (default: 0.75)
	ATRMultiplier              float64 `json:"atr_multiplier"`                 // Dynamic price change threshold multiplier (default: 1.5)
	VolatilityLookback         int     `json:"volatility_lookback"`            // Period for volatility percentile (default: 50)
	LowVolatilityPercentile    float64 `json:"low_volatility_percentile"`      // Low vol threshold (default: 0.3)
	HighVolatilityPercentile   float64 `json:"high_volatility_percentile"`     // High vol threshold (default: 0.8)
	FavorableIndicatorsRequired int    `json:"favorable_indicators_required"`  // Required in favorable conditions (default: 2)
	NormalIndicatorsRequired    int    `json:"normal_indicators_required"`     // Required in normal conditions (default: 3)
	HostileIndicatorsRequired   int    `json:"hostile_indicators_required"`    // Required in hostile conditions (default: 4)
}

// NewDefaultMarketRegimeConfig creates default market regime configuration
func NewDefaultMarketRegimeConfig() *MarketRegimeConfig {
	return &MarketRegimeConfig{
		Enabled:                     false, // Disabled by default for backward compatibility
		TrendStrengthPeriod:         20,
		TrendStrengthThreshold:      0.45, // Reduced from 75% to 65% for crypto markets
		ATRMultiplier:               1.5,  // Dynamic price change threshold (1.5x ATR)
		VolatilityLookback:          50,
		LowVolatilityPercentile:     0.3,
		HighVolatilityPercentile:    0.8,
		FavorableIndicatorsRequired: 2,
		NormalIndicatorsRequired:    3,
		HostileIndicatorsRequired:   4,
	}
}

// MarketRegimeDetector handles market regime detection and signal consensus requirements
type MarketRegimeDetector struct {
	config *MarketRegimeConfig
}

// NewMarketRegimeDetector creates a new market regime detector
func NewMarketRegimeDetector(config *MarketRegimeConfig) *MarketRegimeDetector {
	if config == nil {
		config = NewDefaultMarketRegimeConfig()
	}
	return &MarketRegimeDetector{
		config: config,
	}
}

// DetectRegime detects the current market regime based on trend and volatility
func (d *MarketRegimeDetector) DetectRegime(data []types.OHLCV, currentATR float64) SimpleMarketRegime {
	if !d.config.Enabled {
		return RegimeNormal // Default to normal regime when disabled
	}

	if len(data) < d.config.TrendStrengthPeriod {
		return RegimeNormal // Not enough data, default to normal
	}

	// 1. Check trend strength (now uses dynamic ATR-based threshold)
	trendStrong := d.checkTrendStrength(data, currentATR)
	
	// 2. Check volatility regime
	volatilityLow := d.isLowVolatility(data, currentATR)
	volatilityHigh := d.isHighVolatility(data, currentATR)
	
	// 3. Check for bearish conditions
	bearishConditions := d.checkBearishConditions(data)
	
	// 4. Decision logic
	if trendStrong || volatilityLow {
		fmt.Println("Trend strong")
		return RegimeFavorable // Strong bull trend or low volatility = easier conditions
	}
	
	if volatilityHigh || bearishConditions {
		return RegimeHostile // High volatility or bearish = difficult conditions
	}
	
	return RegimeNormal // Everything else = normal conditions
}

// GetRequiredIndicators returns the number of indicators required for the given regime
func (d *MarketRegimeDetector) GetRequiredIndicators(regime SimpleMarketRegime) int {
	if !d.config.Enabled {
		return d.config.NormalIndicatorsRequired // Default behavior when disabled
	}

	switch regime {
	case RegimeFavorable:
		return d.config.FavorableIndicatorsRequired
	case RegimeNormal:
		return d.config.NormalIndicatorsRequired
	case RegimeHostile:
		return d.config.HostileIndicatorsRequired
	default:
		return d.config.NormalIndicatorsRequired
	}
}

// checkTrendStrength determines if the market is in a strong bullish trend
func (d *MarketRegimeDetector) checkTrendStrength(data []types.OHLCV, currentATR float64) bool {
	if len(data) < d.config.TrendStrengthPeriod {
		return false
	}
	
	recent := data[len(data)-d.config.TrendStrengthPeriod:]
	
	// Calculate price progression over the period
	startPrice := recent[0].Close
	endPrice := recent[len(recent)-1].Close
	currentPrice := endPrice
	
	// Dynamic price change threshold based on current ATR
	// Require price change to be at least config.ATRMultiplier x the current ATR over the period
	dynamicThreshold := (currentATR * d.config.ATRMultiplier) / currentPrice
	
	priceChange := (endPrice - startPrice) / startPrice
	
	// Count how many candles close higher than the previous candle
	higherCloses := 0
	for i := 1; i < len(recent); i++ {
		if recent[i].Close > recent[i-1].Close {
			higherCloses++
		}
	}
	
	trendStrength := float64(higherCloses) / float64(len(recent)-1)
	
	// DEBUG: Show dynamic threshold in action with actual threshold value
	fmt.Printf("DYNAMIC TREND: priceChange=%.4f%% >= dynamicThreshold=%.4f%% (%.1fx ATR=%.4f), trendStrength=%.2f%% >= threshold=%.2f%%, result=%v\n", 
		priceChange*100, dynamicThreshold*100, d.config.ATRMultiplier, currentATR, trendStrength*100, d.config.TrendStrengthThreshold*100,
		(priceChange >= dynamicThreshold && trendStrength >= d.config.TrendStrengthThreshold))
	
	// Dynamic price change requirement based on current market volatility
	if priceChange < dynamicThreshold {
		return false
	}
	
	return trendStrength >= d.config.TrendStrengthThreshold
}

// isLowVolatility checks if current volatility is in the low percentile
func (d *MarketRegimeDetector) isLowVolatility(data []types.OHLCV, currentATR float64) bool {
	if len(data) < d.config.VolatilityLookback {
		return false // Not enough data to determine volatility regime
	}
	
	// Calculate ATR percentile
	atrPercentile := d.calculateATRPercentile(data, currentATR)
	return atrPercentile <= d.config.LowVolatilityPercentile
}

// isHighVolatility checks if current volatility is in the high percentile
func (d *MarketRegimeDetector) isHighVolatility(data []types.OHLCV, currentATR float64) bool {
	if len(data) < d.config.VolatilityLookback {
		return false // Not enough data to determine volatility regime
	}
	
	// Calculate ATR percentile
	atrPercentile := d.calculateATRPercentile(data, currentATR)
	return atrPercentile >= d.config.HighVolatilityPercentile
}

// checkBearishConditions checks for obvious bearish market conditions
func (d *MarketRegimeDetector) checkBearishConditions(data []types.OHLCV) bool {
	if len(data) < d.config.TrendStrengthPeriod {
		return false
	}
	
	recent := data[len(data)-d.config.TrendStrengthPeriod:]
	
	// Check for significant price decline over the period
	startPrice := recent[0].Close
	endPrice := recent[len(recent)-1].Close
	priceChange := (endPrice - startPrice) / startPrice
	
	// Must be at least 5% down to be considered bearish
	if priceChange > -0.05 {
		return false
	}
	
	// Count declining candles
	decliningCandles := 0
	for i := 1; i < len(recent); i++ {
		if recent[i].Close < recent[i-1].Close {
			decliningCandles++
		}
	}
	
	bearishStrength := float64(decliningCandles) / float64(len(recent)-1)
	return bearishStrength > 0.6 // More than 60% declining candles
}

// calculateATRPercentile calculates what percentile the current ATR represents
func (d *MarketRegimeDetector) calculateATRPercentile(data []types.OHLCV, currentATR float64) float64 {
	if len(data) < d.config.VolatilityLookback {
		return 0.5 // Default to median if not enough data
	}
	
	// Calculate ATR for the lookback period
	lookbackData := data[len(data)-d.config.VolatilityLookback:]
	atrValues := make([]float64, 0, len(lookbackData)-1)
	
	for i := 1; i < len(lookbackData); i++ {
		tr := d.calculateTrueRange(lookbackData[i], lookbackData[i-1])
		atrValues = append(atrValues, tr)
	}
	
	if len(atrValues) == 0 {
		return 0.5
	}
	
	// Count how many historical ATR values are below current ATR
	countBelow := 0
	for _, atr := range atrValues {
		if atr < currentATR {
			countBelow++
		}
	}
	
	return float64(countBelow) / float64(len(atrValues))
}

// calculateTrueRange calculates the True Range for two consecutive candles
func (d *MarketRegimeDetector) calculateTrueRange(current, previous types.OHLCV) float64 {
	tr1 := current.High - current.Low
	tr2 := math.Abs(current.High - previous.Close)
	tr3 := math.Abs(current.Low - previous.Close)
	
	return math.Max(tr1, math.Max(tr2, tr3))
}

// Validate validates the market regime configuration
func (c *MarketRegimeConfig) Validate() error {
	if c.TrendStrengthPeriod < 5 {
		return fmt.Errorf("trend strength period must be at least 5, got %d", c.TrendStrengthPeriod)
	}
	
	if c.TrendStrengthThreshold < 0.5 || c.TrendStrengthThreshold > 1.0 {
		return fmt.Errorf("trend strength threshold must be between 0.5 and 1.0, got %.2f", c.TrendStrengthThreshold)
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
