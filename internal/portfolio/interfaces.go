package portfolio

import (
	"context"
	"time"
)

// PortfolioManager is the main interface for managing a multi-bot portfolio
type PortfolioManager interface {
	// Core Operations
	Initialize(ctx context.Context) error
	RegisterBot(botID string, config BotConfig) error
	UnregisterBot(botID string) error
	
	// Balance Management
	GetAvailableBalance(botID string) (float64, error)
	GetRequiredMargin(botID string, positionValue float64) (float64, error)
	UpdateBotBalance(botID string, newBalance float64) error
	
	// Position Tracking
	UpdatePosition(botID string, positionValue, avgPrice float64, leverage float64) error
	GetTotalPortfolioValue() (float64, error)
	GetBotAllocation(botID string) (*BotAllocation, error)
	
	// Profit Management
	RecordProfit(botID string, profit float64) error
	GetTotalProfit() (float64, error)
	
	// State Management
	SaveState() error
	LoadState() error
	
	// Health & Monitoring
	GetPortfolioHealth() (*PortfolioHealth, error)
	IsHealthy() bool
	
	// Shutdown
	Close() error
}

// StateManager handles portfolio state persistence
type StateManager interface {
	Save(state *PortfolioState) error
	Load() (*PortfolioState, error)
	Lock() error
	Unlock() error
	IsLocked() bool
}

// LeverageCalculator handles margin calculations
type LeverageCalculator interface {
	CalculateRequiredMargin(positionValue, leverage float64) float64
	CalculateMaxPositionSize(availableMargin, leverage float64) float64
	ValidateLeverage(leverage float64) error
	GetEffectiveLeverage(positionValue, margin float64) float64
}

// RiskManager handles portfolio-level risk controls
type RiskManager interface {
	ValidateNewPosition(botID string, positionValue, leverage float64) error
	CheckPortfolioLimits() error
	GetRiskMetrics() (*RiskMetrics, error)
	IsWithinRiskLimits(totalExposure, availableBalance float64) bool
}

// BotConfig represents configuration for a single bot in the portfolio
type BotConfig struct {
	Symbol               string  `json:"symbol"`
	Leverage             float64 `json:"leverage"`
	AllocationPercentage float64 `json:"allocation_percentage"`
	MaxPositionSize      float64 `json:"max_position_size"`
	Category             string  `json:"category"` // linear, spot, inverse
}

// BotAllocation represents current allocation and status for a bot
type BotAllocation struct {
	BotID                string    `json:"bot_id"`
	Symbol               string    `json:"symbol"`
	AllocatedBalance     float64   `json:"allocated_balance"`
	UsedBalance          float64   `json:"used_balance"`
	AvailableBalance     float64   `json:"available_balance"`
	CurrentPosition      float64   `json:"current_position"`
	AveragePrice         float64   `json:"average_price"`
	Leverage             float64   `json:"leverage"`
	UnrealizedPnL        float64   `json:"unrealized_pnl"`
	RealizedPnL          float64   `json:"realized_pnl"`
	LastUpdated          time.Time `json:"last_updated"`
	PositionMarginUsed   float64   `json:"position_margin_used"`
	AllocationPercentage float64   `json:"allocation_percentage"`
}

// PortfolioState represents the complete state of the portfolio
type PortfolioState struct {
	TotalBalance     float64                    `json:"total_balance"`
	TotalProfit      float64                    `json:"total_profit"`
	LastUpdated      time.Time                  `json:"last_updated"`
	Allocations      map[string]*BotAllocation  `json:"allocations"`
	GlobalSettings   *PortfolioConfig          `json:"global_settings"`
	Version          string                     `json:"version"`
	LockHolder       string                     `json:"lock_holder,omitempty"`
	LockTime         *time.Time                 `json:"lock_time,omitempty"`
}

// PortfolioConfig represents global portfolio settings
type PortfolioConfig struct {
	TotalBalance         float64 `json:"total_balance"`
	AllocationStrategy   string  `json:"allocation_strategy"` // equal_weight, performance_based, custom
	SharedStateFile      string  `json:"shared_state_file"`
	MaxTotalExposure     float64 `json:"max_total_exposure"`     // Max total exposure as % of balance
	MaxDrawdownPercent   float64 `json:"max_drawdown_percent"`   // Max allowed drawdown %
	RebalanceFrequency   string  `json:"rebalance_frequency"`    // 1h, 4h, 1d
	RiskLimitPerBot      float64 `json:"risk_limit_per_bot"`     // Max risk per bot as % of allocation
	EmergencyStopEnabled bool    `json:"emergency_stop_enabled"` // Enable emergency stop on high drawdown
}

// PortfolioHealth represents overall portfolio health metrics
type PortfolioHealth struct {
	Status           string    `json:"status"`            // healthy, warning, critical
	TotalValue       float64   `json:"total_value"`
	TotalExposure    float64   `json:"total_exposure"`
	ExposurePercent  float64   `json:"exposure_percent"`  // % of total balance
	TotalPnL         float64   `json:"total_pnl"`
	PnLPercent       float64   `json:"pnl_percent"`
	ActiveBots       int       `json:"active_bots"`
	LastHealthCheck  time.Time `json:"last_health_check"`
	Warnings         []string  `json:"warnings,omitempty"`
	Errors           []string  `json:"errors,omitempty"`
}

// RiskMetrics provides detailed risk analysis
type RiskMetrics struct {
	TotalExposure        float64            `json:"total_exposure"`
	ExposureBySymbol     map[string]float64 `json:"exposure_by_symbol"`
	LeverageWeighted     float64            `json:"leverage_weighted"`    // Weighted average leverage
	ConcentrationRisk    float64            `json:"concentration_risk"`   // Largest single exposure %
	CorrelationRisk      float64            `json:"correlation_risk"`     // Risk from correlated positions
	MarginUtilization    float64            `json:"margin_utilization"`   // % of available margin used
	DrawdownFromPeak     float64            `json:"drawdown_from_peak"`
	VaR95                float64            `json:"var_95"`               // Value at Risk 95%
	WorstCaseDrawdown    float64            `json:"worst_case_drawdown"`
}

// TradeEvent represents a trading event for portfolio tracking
type TradeEvent struct {
	BotID           string    `json:"bot_id"`
	Symbol          string    `json:"symbol"`
	Side            string    `json:"side"`           // BUY, SELL
	Quantity        float64   `json:"quantity"`
	Price           float64   `json:"price"`
	Value           float64   `json:"value"`
	Leverage        float64   `json:"leverage"`
	MarginUsed      float64   `json:"margin_used"`
	PnL             float64   `json:"pnl"`
	Timestamp       time.Time `json:"timestamp"`
	OrderID         string    `json:"order_id"`
	NewPosition     float64   `json:"new_position"`
	NewAveragePrice float64   `json:"new_average_price"`
}

// PortfolioError represents portfolio-specific errors
type PortfolioError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	BotID     string `json:"bot_id,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *PortfolioError) Error() string {
	if e.BotID != "" {
		return e.Code + " [" + e.BotID + "]: " + e.Message
	}
	return e.Code + ": " + e.Message
}

// Common error codes
const (
	ErrInsufficientBalance   = "INSUFFICIENT_BALANCE"
	ErrInsufficientMargin    = "INSUFFICIENT_MARGIN"
	ErrExceedsAllocation     = "EXCEEDS_ALLOCATION"
	ErrExceedsRiskLimit      = "EXCEEDS_RISK_LIMIT"
	ErrBotNotRegistered      = "BOT_NOT_REGISTERED"
	ErrBotAlreadyRegistered  = "BOT_ALREADY_REGISTERED"
	ErrInvalidLeverage       = "INVALID_LEVERAGE"
	ErrPortfolioLocked       = "PORTFOLIO_LOCKED"
	ErrStateCorrupted        = "STATE_CORRUPTED"
	ErrConfigurationInvalid  = "CONFIGURATION_INVALID"
)
