package portfolio

import (
	"fmt"
	"time"
)

// BasicRiskManager implements the RiskManager interface
type BasicRiskManager struct {
	config *PortfolioConfig
}

// NewBasicRiskManager creates a new basic risk manager
func NewBasicRiskManager(config *PortfolioConfig) RiskManager {
	return &BasicRiskManager{
		config: config,
	}
}

// ValidateNewPosition validates if a new position is within risk limits
func (r *BasicRiskManager) ValidateNewPosition(botID string, positionValue, leverage float64) error {
	// Basic validation - position size should be reasonable
	if positionValue <= 0 {
		return &PortfolioError{
			Code:      ErrConfigurationInvalid,
			Message:   "Position value must be greater than 0",
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	// Check leverage limits
	if leverage > 125 { // Bybit max leverage
		return &PortfolioError{
			Code:      ErrInvalidLeverage,
			Message:   fmt.Sprintf("Leverage %.1fx exceeds maximum allowed 125x", leverage),
			BotID:     botID,
			Timestamp: time.Now(),
		}
	}
	
	return nil
}

// CheckPortfolioLimits checks if portfolio is within overall limits
func (r *BasicRiskManager) CheckPortfolioLimits() error {
	// This would typically check against current portfolio state
	// For now, return nil (basic implementation)
	return nil
}

// GetRiskMetrics returns current risk metrics
func (r *BasicRiskManager) GetRiskMetrics() (*RiskMetrics, error) {
	// Basic risk metrics implementation
	return &RiskMetrics{
		TotalExposure:     0,
		ExposureBySymbol:  make(map[string]float64),
		LeverageWeighted:  1.0,
		ConcentrationRisk: 0,
		CorrelationRisk:   0,
		MarginUtilization: 0,
		DrawdownFromPeak:  0,
		VaR95:            0,
		WorstCaseDrawdown: 0,
	}, nil
}

// IsWithinRiskLimits checks if exposure is within limits
func (r *BasicRiskManager) IsWithinRiskLimits(totalExposure, availableBalance float64) bool {
	if availableBalance <= 0 {
		return false
	}
	
	exposureRatio := totalExposure / availableBalance
	return exposureRatio <= r.config.MaxTotalExposure
}
