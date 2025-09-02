package risk

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/engines"
)

// RiskAssessment contains comprehensive risk analysis results
type RiskAssessment struct {
	Timestamp            time.Time                 `json:"timestamp"`
	OverallRiskScore     float64                   `json:"overall_risk_score"`     // 0.0 to 1.0
	PortfolioValue       float64                   `json:"portfolio_value"`
	DailyPnL             float64                   `json:"daily_pnl"`
	MaxDrawdown          float64                   `json:"max_drawdown"`
	TotalExposure        float64                   `json:"total_exposure"`
	GlobalViolations     []RiskViolation           `json:"global_violations"`
	EngineViolations     []RiskViolation           `json:"engine_violations"`
	CircuitBreakerAlerts []CircuitBreakerAlert     `json:"circuit_breaker_alerts"`
	SystemStatus         string                    `json:"system_status"`
	EmergencyStop        bool                      `json:"emergency_stop"`
	ManualOverride       bool                      `json:"manual_override"`
}

// RiskViolation represents a violation of risk limits
type RiskViolation struct {
	Type        string                `json:"type"`
	Description string                `json:"description"`
	Severity    string                `json:"severity"`
	Current     float64               `json:"current"`
	Limit       float64               `json:"limit"`
	Action      string                `json:"action"`
	EngineType  engines.EngineType    `json:"engine_type,omitempty"`
	Timestamp   time.Time             `json:"timestamp"`
}

// CircuitBreakerAlert represents an activated circuit breaker
type CircuitBreakerAlert struct {
	Name         string     `json:"name"`
	Type         string     `json:"type"`
	Action       string     `json:"action"`
	Reason       string     `json:"reason"`
	Timestamp    time.Time  `json:"timestamp"`
	Severity     string     `json:"severity"`
}

// RiskStatus represents the current risk status of the system
type RiskStatus struct {
	SystemStatus          string    `json:"system_status"`
	OverallRiskScore      float64   `json:"overall_risk_score"`
	PortfolioValue        float64   `json:"portfolio_value"`
	DailyPnL              float64   `json:"daily_pnl"`
	MaxDrawdown           float64   `json:"max_drawdown"`
	TotalExposure         float64   `json:"total_exposure"`
	EmergencyStop         bool      `json:"emergency_stop"`
	ManualOverride        bool      `json:"manual_override"`
	SystemPaused          bool      `json:"system_paused"`
	ActiveCircuitBreakers []string  `json:"active_circuit_breakers"`
	LastUpdate            time.Time `json:"last_update"`
}