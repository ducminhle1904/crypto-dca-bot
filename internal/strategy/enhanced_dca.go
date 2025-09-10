package strategy

import (
	"fmt"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// EnhancedDCAStrategy implements a Dollar Cost Averaging strategy with multiple technical indicators
type EnhancedDCAStrategy struct {
	indicatorManager *indicators.IndicatorManager
	baseAmount       float64
	maxMultiplier    float64
	minConfidence    float64
	lastTradeTime    time.Time
	priceThreshold   float64  // Base price drop % for first DCA entry
	priceThresholdMultiplier float64  // Multiplier for progressive DCA spacing (e.g., 1.1x per level)
	lastEntryPrice   float64  // Track last entry price for threshold calculation
	dcaLevel         int      // Current DCA level (0 = first entry, 1+ = subsequent entries)
}

// NewEnhancedDCAStrategy creates a new enhanced DCA strategy instance
func NewEnhancedDCAStrategy(baseAmount float64) *EnhancedDCAStrategy {
	return &EnhancedDCAStrategy{
		indicatorManager: indicators.NewIndicatorManager(),
		baseAmount:       baseAmount,
		maxMultiplier:    3.0,
		minConfidence:    0.5,
		priceThreshold:   0.0, // Default: no threshold (disabled)
		priceThresholdMultiplier: 1.0, // Default: no multiplier (1.0x)
		lastEntryPrice:   0.0, // No previous entry
		dcaLevel:         0,   // Start at level 0
	}
}

// SetPriceThreshold sets the base price drop % required for DCA entries
func (s *EnhancedDCAStrategy) SetPriceThreshold(threshold float64) {
	s.priceThreshold = threshold
}

// SetPriceThresholdMultiplier sets the multiplier for progressive DCA spacing
func (s *EnhancedDCAStrategy) SetPriceThresholdMultiplier(multiplier float64) {
	s.priceThresholdMultiplier = multiplier
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
	buySignals, sellSignals, buyStrength, _ := s.indicatorManager.CountActiveSignals(results)
	
	// Track failed indicators for debugging
	failedCount := 0
	for _, result := range results {
		if result.Error != nil {
			failedCount++
		}
	}
	
	// Calculate active signals (exclude failed indicators)
	totalIndicators := len(results) - failedCount
	activeSignals := buySignals + sellSignals
	
	// If no indicators are giving active signals, hold
	if activeSignals == 0 || totalIndicators == 0 {
		return &TradeDecision{
			Action: ActionHold, 
			Reason: fmt.Sprintf("No active signals from %d indicators", totalIndicators),
		}, nil
	}
	
	// Calculate confidence based on active signals only
	confidence := float64(buySignals) / float64(activeSignals)

	if confidence >= s.minConfidence {
		// Apply price threshold check for DCA entries
		if s.priceThreshold > 0 && s.lastEntryPrice > 0 {
			priceDrop := (s.lastEntryPrice - currentPrice) / s.lastEntryPrice
			requiredThreshold := s.calculateCurrentThreshold()
			
			if priceDrop < requiredThreshold {
				return &TradeDecision{
					Action: ActionHold,
					Reason: fmt.Sprintf("Price threshold not met: %.2f%% < %.2f%% (DCA Level %d)", 
						priceDrop*100, requiredThreshold*100, s.dcaLevel),
				}, nil
			}
		}

		// Calculate net strength for buy decision
		netStrength := 0.0
		if buySignals > 0 {
			netStrength = buyStrength / float64(buySignals)
		}
		
		amount := s.calculatePositionSize(netStrength, confidence)
		
		// Update last entry price, time, and increment DCA level
		s.lastEntryPrice = currentPrice
		s.lastTradeTime = currentCandle.Timestamp
		s.dcaLevel++ // Increment for next threshold calculation
		
		return &TradeDecision{
			Action:     ActionBuy,
			Amount:     amount,
			Confidence: confidence,
			Strength:   netStrength,
			Reason:     fmt.Sprintf("Buy consensus: %d/%d signals (%.1f%% confidence)", 
						buySignals, activeSignals, confidence*100),
		}, nil
	}

	return &TradeDecision{
		Action: ActionHold,
		Reason: fmt.Sprintf("Insufficient buy consensus: %d/%d signals (%.1f%% < %.1f%%)", 
				buySignals, activeSignals, confidence*100, s.minConfidence*100),
	}, nil
}

// calculateCurrentThreshold calculates the progressive price threshold based on DCA level
func (s *EnhancedDCAStrategy) calculateCurrentThreshold() float64 {
	if s.priceThresholdMultiplier <= 1.0 {
		return s.priceThreshold // No multiplier effect
	}
	
	// Progressive threshold: baseThreshold * multiplier^dcaLevel
	// Level 0: 1% × 1.1^0 = 1%
	// Level 1: 1% × 1.1^1 = 1.1% 
	// Level 2: 1% × 1.1^2 = 1.21%
	// Level 3: 1% × 1.1^3 = 1.33%
	threshold := s.priceThreshold
	for i := 0; i < s.dcaLevel; i++ {
		threshold *= s.priceThresholdMultiplier
	}
	
	return threshold
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
	return map[string]interface{}{
		"base_amount":                 s.baseAmount,
		"max_multiplier":              s.maxMultiplier,
		"min_confidence":              s.minConfidence,
		"price_threshold":             s.priceThreshold,
		"price_threshold_multiplier":  s.priceThresholdMultiplier,
		"current_dca_level":           s.dcaLevel,
		"current_threshold":           s.calculateCurrentThreshold(),
		"indicator_count":             s.GetIndicatorCount(),
		"last_entry_price":            s.lastEntryPrice,
		"last_trade_time":             s.lastTradeTime,
	}
}

// ResetForNewPeriod resets strategy state for walk-forward validation periods
func (s *EnhancedDCAStrategy) ResetForNewPeriod() {
	// Reset all indicators and clear cache in one atomic operation
	s.indicatorManager.ResetAllIndicators()
	
	// Reset strategy state
	s.lastEntryPrice = 0.0
	s.lastTradeTime = time.Time{}
	s.dcaLevel = 0
}




