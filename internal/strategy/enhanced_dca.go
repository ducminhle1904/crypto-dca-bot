package strategy

import (
	"errors"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/internal/indicators"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

type EnhancedDCAStrategy struct {
	indicators    []indicators.TechnicalIndicator
	baseAmount    float64
	maxMultiplier float64
	minConfidence float64
	lastTradeTime time.Time
	minInterval   time.Duration
}

func NewEnhancedDCAStrategy(baseAmount float64) *EnhancedDCAStrategy {
	return &EnhancedDCAStrategy{
		indicators:    make([]indicators.TechnicalIndicator, 0),
		baseAmount:    baseAmount,
		maxMultiplier: 3.0,
		minConfidence: 0.6,
		minInterval:   time.Hour * 4, // Minimum 4 hours between transactions
	}
}

func (s *EnhancedDCAStrategy) AddIndicator(indicator indicators.TechnicalIndicator) {
	s.indicators = append(s.indicators, indicator)
}

func (s *EnhancedDCAStrategy) ShouldExecuteTrade(data []types.OHLCV) (*TradeDecision, error) {
	if len(data) == 0 {
		return nil, errors.New("no market data provided")
	}

	// Checking the time interval
	if time.Since(s.lastTradeTime) < s.minInterval {
		return &TradeDecision{
			Action: ActionHold,
			Reason: "Too soon since last trade",
		}, nil
	}

	// Collect signals from all indicators
	buySignals := 0
	sellSignals := 0
	totalStrength := 0.0

	for _, indicator := range s.indicators {
		// check the sufficiency of the data
		if len(data) < indicator.GetRequiredPeriods() {
			continue
		}

		currentPrice := data[len(data)-1].Close

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
		amount := s.calculatePositionSize(totalStrength, confidence)
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

// EnhancedDCAStrategy implements the Strategy interface
