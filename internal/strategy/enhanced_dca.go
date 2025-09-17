package strategy

import (
	"fmt"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/base"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy/spacing"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// EnhancedDCAStrategy implements a Dollar Cost Averaging strategy with multiple technical indicators
type EnhancedDCAStrategy struct {
	indicatorManager *indicators.IndicatorManager
	baseAmount       float64
	maxMultiplier    float64
	minConfidence    float64
	lastTradeTime    time.Time
	lastEntryPrice   float64  // Track last entry price for threshold calculation
	dcaLevel         int      // Current DCA level (0 = first entry, 1+ = subsequent entries)
	
	// DCA spacing strategy
	spacingStrategy  spacing.DCASpacingStrategy // Pluggable spacing strategy
	atrCalculator    *base.ATR                  // ATR calculator for market context
	
	// Dynamic TP configuration
	dynamicTPConfig  *config.DynamicTPConfig    // Dynamic TP configuration
}

// NewEnhancedDCAStrategy creates a new enhanced DCA strategy instance
func NewEnhancedDCAStrategy(baseAmount float64) *EnhancedDCAStrategy {
	return &EnhancedDCAStrategy{
		indicatorManager: indicators.NewIndicatorManager(),
		baseAmount:       baseAmount,
		maxMultiplier:    3.0,
		minConfidence:    0.5,
		lastEntryPrice:   0.0, // No previous entry
		dcaLevel:         0,   // Start at level 0
		spacingStrategy:  nil, // Will be set by orchestrator
		atrCalculator:    base.NewATR(14), // 14-period ATR for market context
		dynamicTPConfig:  nil, // Will be set by orchestrator if dynamic TP is enabled
	}
}


// SetSpacingStrategy sets the advanced spacing strategy (new feature)
func (s *EnhancedDCAStrategy) SetSpacingStrategy(spacingStrategy spacing.DCASpacingStrategy) {
	s.spacingStrategy = spacingStrategy
}

// GetSpacingStrategy returns the current spacing strategy
func (s *EnhancedDCAStrategy) GetSpacingStrategy() spacing.DCASpacingStrategy {
	return s.spacingStrategy
}

// SetDynamicTPConfig sets the dynamic TP configuration
func (s *EnhancedDCAStrategy) SetDynamicTPConfig(dynamicTPConfig *config.DynamicTPConfig) {
	s.dynamicTPConfig = dynamicTPConfig
}

// GetDynamicTPConfig returns the current dynamic TP configuration
func (s *EnhancedDCAStrategy) GetDynamicTPConfig() *config.DynamicTPConfig {
	return s.dynamicTPConfig
}

// AddIndicator adds a technical indicator to the strategy
func (s *EnhancedDCAStrategy) AddIndicator(indicator indicators.TechnicalIndicator) {
	s.indicatorManager.AddIndicator(indicator)
}

func (s *EnhancedDCAStrategy) ShouldExecuteTrade(data []types.OHLCV) (*TradeDecision, error) {
	if len(data) == 0 {
		return &TradeDecision{Action: ActionHold}, nil
	}

	currentCandle := data[len(data)-1]
	currentPrice := currentCandle.Close

	// Process all indicators in batch (major optimization)
	results := s.indicatorManager.ProcessCandle(currentCandle, data)
	
	// Check if we have any indicators configured
	if len(results) == 0 {
		return &TradeDecision{Action: ActionHold, Reason: "No indicators configured"}, nil
	}

	// Efficiently count signals using batch results
	buySignals, sellSignals, _, _ := s.indicatorManager.CountActiveSignals(results)
	
	// Track failed indicators for debugging
	failedCount := 0
	workingCount := 0
	failedIndicators := []string{}
	workingIndicators := []string{}
	
	for name, result := range results {
		if result.Error != nil {
			failedCount++
			failedIndicators = append(failedIndicators, fmt.Sprintf("%s: %v", name, result.Error))
		} else {
			workingCount++
			workingIndicators = append(workingIndicators, name)
		}
	}
	
	// Use the appended slices to avoid linter warnings (they're used for debugging)
	_ = failedIndicators
	_ = workingIndicators

	
	// Get total configured indicators (always base calculations on this)
	totalConfiguredIndicators := s.GetIndicatorCount()
	workingIndicatorsCount := len(results) - failedCount
	activeSignals := buySignals + sellSignals
	
	// If no indicators are working, hold
	if workingIndicatorsCount == 0 || totalConfiguredIndicators == 0 {
		return &TradeDecision{
			Action: ActionHold, 
			Reason: fmt.Sprintf("No working indicators (%d failed out of %d total)", failedCount, totalConfiguredIndicators),
		}, nil
	}
	
	// Calculate confidence based on ALL configured indicators (not just active signals)
	confidence := float64(buySignals) / float64(totalConfiguredIndicators)

	if confidence >= s.minConfidence {
	// Apply price threshold check for DCA entries with defensive checks
	if s.spacingStrategy != nil && s.lastEntryPrice > 0 && currentPrice > 0 {
		// Add defensive check to prevent division by zero
		if s.lastEntryPrice <= 0 {
			return &TradeDecision{
				Action: ActionHold,
				Reason: "Invalid last entry price for threshold calculation",
			}, nil
		}
		
		priceDrop := (s.lastEntryPrice - currentPrice) / s.lastEntryPrice
		requiredThreshold := s.calculateCurrentThreshold(currentCandle, data)
		
		if priceDrop < requiredThreshold {
			strategyInfo := "Fixed Progressive"
			if s.spacingStrategy != nil {
				strategyInfo = s.spacingStrategy.GetName()
			}
			
			return &TradeDecision{
				Action: ActionHold,
				Reason: fmt.Sprintf("Price threshold not met: %.2f%% < %.2f%% (DCA Level %d, Strategy: %s)", 
					priceDrop*100, requiredThreshold*100, s.dcaLevel, strategyInfo),
			}, nil
		}
	}

		// Calculate net strength based on buy signals across ALL indicators  
		netStrength := float64(buySignals) / float64(totalConfiguredIndicators)
		
		amount := s.calculatePositionSize(netStrength, confidence)
		
		// Update last entry price, time, and increment DCA level
		s.lastEntryPrice = currentPrice
		s.lastTradeTime = currentCandle.Timestamp
		s.dcaLevel++ // Increment DCA level for next entry
		
		return &TradeDecision{
			Action:     ActionBuy,
			Amount:     amount,
			Confidence: confidence,
			Strength:   netStrength,
			Reason:     fmt.Sprintf("Buy consensus: %d/%d active", buySignals, activeSignals),
		}, nil
	}

	return &TradeDecision{
		Action: ActionHold,
		Reason: fmt.Sprintf("Insufficient buy consensus: %d/%d active (%.1f%% < %.1f%%)", 
				buySignals, activeSignals, confidence*100, s.minConfidence*100),
	}, nil
}

// calculateCurrentThreshold calculates the price threshold based on DCA level using the configured spacing strategy
func (s *EnhancedDCAStrategy) calculateCurrentThreshold(currentCandle types.OHLCV, recentCandles []types.OHLCV) float64 {
	if s.spacingStrategy == nil {
		// This should not happen if configuration is properly validated
		return 0.01 // Fallback to 1%
	}
	
	return s.calculateWithSpacingStrategy(currentCandle, recentCandles)
}

// calculateWithSpacingStrategy uses the configured spacing strategy
func (s *EnhancedDCAStrategy) calculateWithSpacingStrategy(currentCandle types.OHLCV, recentCandles []types.OHLCV) float64 {
	// Calculate ATR with recent data using strategy's calculator
	atrValue := 0.0
	if atr, err := s.atrCalculator.Calculate(recentCandles); err != nil {
		// Log ATR calculation failure but continue with base threshold
		fmt.Printf("Warning: ATR calculation failed: %v, using base threshold for DCA level %d\n", err, s.dcaLevel)
		atrValue = 0
	} else {
		atrValue = atr
	}
	
	// Create market context
	context := &spacing.MarketContext{
		CurrentPrice:   currentCandle.Close,
		LastEntryPrice: s.lastEntryPrice,
		ATR:           atrValue,
		CurrentCandle: currentCandle,
		RecentCandles: recentCandles,
		Timestamp:     currentCandle.Timestamp,
	}
	
	// Calculate threshold using spacing strategy
	return s.spacingStrategy.CalculateThreshold(s.dcaLevel, context)
}


func (s *EnhancedDCAStrategy) calculatePositionSize(strength, confidence float64) float64 {
	// The base amount is multiplied by the confidence and strength of the signal
	multiplier := 1.0 + (confidence * strength)

	// limit it to the maximum multiplier
	if multiplier > s.maxMultiplier {
		multiplier = s.maxMultiplier
	}

	return s.baseAmount * multiplier
}

func (s *EnhancedDCAStrategy) GetName() string {
	return "Enhanced DCA Strategy"
}

// OnCycleComplete resets strategy state when a take-profit cycle is completed
func (s *EnhancedDCAStrategy) OnCycleComplete() {
	// Reset the last entry price so the next cycle starts fresh
	s.lastEntryPrice = 0.0
	// Reset DCA level for next cycle
	s.dcaLevel = 0
	// Clear indicator cache to start fresh for next cycle
	s.indicatorManager.ClearCache()
	// Reset spacing strategy state
	if s.spacingStrategy != nil {
		s.spacingStrategy.Reset()
	}
	// Reset ATR calculator
	s.atrCalculator = base.NewATR(14)
}

// GetIndicatorManager returns the indicator manager (useful for advanced configuration)
func (s *EnhancedDCAStrategy) GetIndicatorManager() *indicators.IndicatorManager {
	return s.indicatorManager
}

// GetIndicatorCount returns the number of configured indicators
func (s *EnhancedDCAStrategy) GetIndicatorCount() int {
	return len(s.indicatorManager.GetIndicators())
}

// GetLastResults returns the most recent indicator results (useful for debugging)
func (s *EnhancedDCAStrategy) GetLastResults() map[string]*indicators.IndicatorResult {
	return s.indicatorManager.GetCachedResults()
}

// SetMinConfidence sets the minimum confidence threshold for buy signals
func (s *EnhancedDCAStrategy) SetMinConfidence(confidence float64) {
	s.minConfidence = confidence
}

// SetMaxMultiplier sets the maximum position size multiplier
func (s *EnhancedDCAStrategy) SetMaxMultiplier(multiplier float64) {
	s.maxMultiplier = multiplier
}

// GetConfiguration returns current strategy configuration
func (s *EnhancedDCAStrategy) GetConfiguration() map[string]interface{} {
	config := map[string]interface{}{
		"base_amount":                 s.baseAmount,
		"max_multiplier":              s.maxMultiplier,
		"min_confidence":              s.minConfidence,
		"current_dca_level":           s.dcaLevel,
		"indicator_count":             s.GetIndicatorCount(),
		"last_entry_price":            s.lastEntryPrice,
		"last_trade_time":             s.lastTradeTime,
	}
	
	// Add spacing strategy info if available
	if s.spacingStrategy != nil {
		config["spacing_strategy"] = s.spacingStrategy.GetName()
		config["spacing_parameters"] = s.spacingStrategy.GetParameters()
	}
	
	return config
}

// ResetForNewPeriod resets strategy state for walk-forward validation periods
func (s *EnhancedDCAStrategy) ResetForNewPeriod() {
	// Reset all indicators and clear cache in one atomic operation
	s.indicatorManager.ResetAllIndicators()
	
	// Reset strategy state
	s.lastEntryPrice = 0.0
	s.lastTradeTime = time.Time{}
	s.dcaLevel = 0
	
	// Reset spacing strategy state
	if s.spacingStrategy != nil {
		s.spacingStrategy.Reset()
	}
	
	// Reset ATR calculator
	s.atrCalculator = base.NewATR(14)
}

// SetDCALevel sets the current DCA level (for live bot state synchronization)
func (s *EnhancedDCAStrategy) SetDCALevel(level int) {
	s.dcaLevel = level
}

// SetLastEntryPrice sets the last entry price (for live bot state synchronization)
func (s *EnhancedDCAStrategy) SetLastEntryPrice(price float64) {
	s.lastEntryPrice = price
}

// IsDynamicTPEnabled returns true if dynamic TP is configured and enabled
func (s *EnhancedDCAStrategy) IsDynamicTPEnabled() bool {
	return s.dynamicTPConfig != nil && 
		   s.dynamicTPConfig.Strategy != "" && 
		   s.dynamicTPConfig.Strategy != "fixed"
}

// GetDynamicTPPercent calculates dynamic TP percentage based on current market conditions
func (s *EnhancedDCAStrategy) GetDynamicTPPercent(currentCandle types.OHLCV, data []types.OHLCV) (float64, error) {
	if !s.IsDynamicTPEnabled() {
		return 0, nil // Return 0 if dynamic TP is not enabled
	}

	return s.calculateDynamicTP(currentCandle, data)
}

// calculateDynamicTP calculates the dynamic TP based on the configured strategy
func (s *EnhancedDCAStrategy) calculateDynamicTP(currentCandle types.OHLCV, data []types.OHLCV) (float64, error) {
	if err := s.validateDynamicTPConfig(); err != nil {
		return s.dynamicTPConfig.BaseTPPercent, err // Fallback to base TP
	}

	switch s.dynamicTPConfig.Strategy {
	case "volatility_adaptive":
		return s.calculateVolatilityBasedTP(currentCandle, data)
	case "indicator_based":
		return s.calculateIndicatorBasedTP(currentCandle, data)
	default:
		return s.dynamicTPConfig.BaseTPPercent, nil // Fixed fallback
	}
}

// calculateVolatilityBasedTP calculates TP based on market volatility (ATR)
func (s *EnhancedDCAStrategy) calculateVolatilityBasedTP(currentCandle types.OHLCV, data []types.OHLCV) (float64, error) {
	volatilityConfig := s.dynamicTPConfig.VolatilityConfig
	if volatilityConfig == nil {
		return s.dynamicTPConfig.BaseTPPercent, fmt.Errorf("volatility config is nil")
	}

	// Calculate ATR using existing infrastructure
	atrValue, err := s.atrCalculator.Calculate(data)
	if err != nil {
		return s.dynamicTPConfig.BaseTPPercent, fmt.Errorf("ATR calculation failed: %w", err)
	}

	// Normalize ATR as percentage of current price
	normalizedVolatility := atrValue / currentCandle.Close

	// Formula: Higher volatility = Higher TP target
	// TP = BaseTP * (1 + normalizedVolatility * multiplier)
	dynamicTP := s.dynamicTPConfig.BaseTPPercent * (1 + normalizedVolatility*volatilityConfig.Multiplier)

	// Apply bounds
	if dynamicTP < volatilityConfig.MinTPPercent {
		dynamicTP = volatilityConfig.MinTPPercent
	}
	if dynamicTP > volatilityConfig.MaxTPPercent {
		dynamicTP = volatilityConfig.MaxTPPercent
	}

	return dynamicTP, nil
}

// calculateIndicatorBasedTP calculates TP based on indicator signal strength
func (s *EnhancedDCAStrategy) calculateIndicatorBasedTP(currentCandle types.OHLCV, data []types.OHLCV) (float64, error) {
	indicatorConfig := s.dynamicTPConfig.IndicatorConfig
	if indicatorConfig == nil {
		return s.dynamicTPConfig.BaseTPPercent, fmt.Errorf("indicator config is nil")
	}

	// Process all indicators to get current signals
	results := s.indicatorManager.ProcessCandle(currentCandle, data)
	
	totalStrength := 0.0
	totalWeight := 0.0

	// Aggregate signal strength from all active indicators
	for name, result := range results {
		if result.Error != nil {
			continue // Skip failed indicators
		}

		weight := indicatorConfig.Weights[name]
		if weight > 0 {
			// Create signal from result and convert to strength
			signal := s.createSignalFromResult(result)
			strength := s.convertSignalToTPStrength(signal)
			totalStrength += strength * weight
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		return s.dynamicTPConfig.BaseTPPercent, nil
	}

	avgStrength := totalStrength / totalWeight

	// Formula: Stronger signals = Higher TP targets
	// TP = BaseTP * (0.7 + avgStrength * strengthMultiplier)
	// Range: 70% to 130% of base TP (when avgStrength is -1 to 1)
	dynamicTP := s.dynamicTPConfig.BaseTPPercent * (0.7 + avgStrength*indicatorConfig.StrengthMultiplier)

	// Apply bounds
	if dynamicTP < indicatorConfig.MinTPPercent {
		dynamicTP = indicatorConfig.MinTPPercent
	}
	if dynamicTP > indicatorConfig.MaxTPPercent {
		dynamicTP = indicatorConfig.MaxTPPercent
	}

	return dynamicTP, nil
}

// createSignalFromResult creates a signal from indicator result
func (s *EnhancedDCAStrategy) createSignalFromResult(result *indicators.IndicatorResult) indicators.Signal {
	if result.ShouldBuy {
		return indicators.Signal{
			Type:     indicators.SignalBuy,
			Strength: result.Strength,
		}
	} else if result.ShouldSell {
		return indicators.Signal{
			Type:     indicators.SignalSell,
			Strength: result.Strength,
		}
	} else {
		return indicators.Signal{
			Type:     indicators.SignalHold,
			Strength: result.Strength,
		}
	}
}

// convertSignalToTPStrength converts indicator signal to TP strength value (-1 to 1)
func (s *EnhancedDCAStrategy) convertSignalToTPStrength(signal indicators.Signal) float64 {
	switch signal.Type {
	case indicators.SignalBuy:
		// Buy signals get positive strength (0.2 to 1.0 based on signal strength)
		return 0.2 + (signal.Strength * 0.8)
	case indicators.SignalSell:
		// Sell signals get negative strength (-0.2 to -1.0 based on signal strength)
		return -(0.2 + (signal.Strength * 0.8))
	case indicators.SignalHold:
		return 0.0
	default:
		return 0.0
	}
}

// validateDynamicTPConfig validates the dynamic TP configuration
func (s *EnhancedDCAStrategy) validateDynamicTPConfig() error {
	if s.dynamicTPConfig == nil {
		return fmt.Errorf("dynamic TP config is nil")
	}

	if s.dynamicTPConfig.BaseTPPercent <= 0 {
		return fmt.Errorf("base TP percent must be positive, got: %.4f", s.dynamicTPConfig.BaseTPPercent)
	}

	switch s.dynamicTPConfig.Strategy {
	case "volatility_adaptive":
		if s.dynamicTPConfig.VolatilityConfig == nil {
			return fmt.Errorf("volatility config is required for volatility_adaptive strategy")
		}
		vc := s.dynamicTPConfig.VolatilityConfig
		if vc.Multiplier < 0 || vc.Multiplier > 10 {
			return fmt.Errorf("volatility multiplier must be between 0 and 10, got: %.2f", vc.Multiplier)
		}
		if vc.MinTPPercent >= vc.MaxTPPercent {
			return fmt.Errorf("min TP percent (%.4f) must be less than max TP percent (%.4f)", 
				vc.MinTPPercent, vc.MaxTPPercent)
		}

	case "indicator_based":
		if s.dynamicTPConfig.IndicatorConfig == nil {
			return fmt.Errorf("indicator config is required for indicator_based strategy")
		}
		ic := s.dynamicTPConfig.IndicatorConfig
		if ic.StrengthMultiplier < 0 || ic.StrengthMultiplier > 2 {
			return fmt.Errorf("strength multiplier must be between 0 and 2, got: %.2f", ic.StrengthMultiplier)
		}
		if ic.MinTPPercent >= ic.MaxTPPercent {
			return fmt.Errorf("min TP percent (%.4f) must be less than max TP percent (%.4f)", 
				ic.MinTPPercent, ic.MaxTPPercent)
		}
	}

	return nil
}
