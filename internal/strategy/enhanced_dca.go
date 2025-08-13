package strategy

import (
	"fmt"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// EnhancedDCAStrategy implements a Dollar Cost Averaging strategy with multiple technical indicators
type EnhancedDCAStrategy struct {
	indicators       []indicators.TechnicalIndicator
	baseAmount       float64
	maxMultiplier    float64
	minConfidence    float64
	lastTradeTime    time.Time
	priceThreshold   float64  // Minimum price drop % for next DCA entry
	lastEntryPrice   float64  // Track last entry price for threshold calculation
}

// NewEnhancedDCAStrategy creates a new enhanced DCA strategy instance
func NewEnhancedDCAStrategy(baseAmount float64) *EnhancedDCAStrategy {
	return &EnhancedDCAStrategy{
		indicators:       make([]indicators.TechnicalIndicator, 0),
		baseAmount:       baseAmount,
		maxMultiplier:    3.0,
		minConfidence:    0.5,
		priceThreshold:   0.0, // Default: no threshold (disabled)
		lastEntryPrice:   0.0, // No previous entry
	}
}

// SetPriceThreshold sets the minimum price drop % required for next DCA entry
func (s *EnhancedDCAStrategy) SetPriceThreshold(threshold float64) {
	s.priceThreshold = threshold
}

// AddIndicator adds a technical indicator to the strategy
func (s *EnhancedDCAStrategy) AddIndicator(indicator indicators.TechnicalIndicator) {
	s.indicators = append(s.indicators, indicator)
}

func (s *EnhancedDCAStrategy) ShouldExecuteTrade(data []types.OHLCV) (*TradeDecision, error) {
	if len(data) == 0 {
		return &TradeDecision{Action: ActionHold}, nil
	}

	// Get current price
	currentPrice := data[len(data)-1].Close

	// Collect signals from all indicators
	buySignals := 0
	sellSignals := 0
	totalStrength := 0.0

	for _, indicator := range s.indicators {
		// check the sufficiency of the data
		if len(data) < indicator.GetRequiredPeriods() {
			continue
		}

		shouldBuy, err := indicator.ShouldBuy(currentPrice, data)
		if err != nil {
			continue
		}

		shouldSell, err := indicator.ShouldSell(currentPrice, data)
		if err != nil {
			continue
		}

		if shouldBuy {
			buySignals++
			totalStrength += indicator.GetSignalStrength()
		} else if shouldSell {
			sellSignals++
			totalStrength -= indicator.GetSignalStrength()
		}
	}

	// make a decision based on consensus
	totalIndicators := len(s.indicators)
	if totalIndicators == 0 {
		return &TradeDecision{Action: ActionHold, Reason: "No indicators configured"}, nil
	}

	confidence := float64(buySignals) / float64(totalIndicators)

	if confidence >= s.minConfidence {
		// Apply price threshold check for DCA entries
		if s.priceThreshold > 0 && s.lastEntryPrice > 0 {
			// Calculate price drop since last entry
			priceDrop := (s.lastEntryPrice - currentPrice) / s.lastEntryPrice
			
			// If price hasn't dropped enough, skip this entry
			if priceDrop < s.priceThreshold {
				return &TradeDecision{
					Action: ActionHold,
					Reason: fmt.Sprintf("Price threshold not met: %.2f%% < %.2f%%", 
						priceDrop*100, s.priceThreshold*100),
				}, nil
			}
		}

		amount := s.calculatePositionSize(totalStrength, confidence)
		
		// Update last entry price when we decide to buy
		s.lastEntryPrice = currentPrice
		s.lastTradeTime = data[len(data)-1].Timestamp
		
		return &TradeDecision{
			Action:     ActionBuy,
			Amount:     amount,
			Confidence: confidence,
			Strength:   totalStrength,
			Reason:     "Buy signal consensus reached",
		}, nil
	}

	return &TradeDecision{
		Action: ActionHold,
		Reason: "Insufficient buy signal consensus",
	}, nil
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
}


