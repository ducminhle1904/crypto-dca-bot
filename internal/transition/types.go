package transition

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/engines"
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
)

// TransitionDecision represents the result of transition evaluation
type TransitionDecision struct {
	Action         TransitionActionType  `json:"action"`
	Reason         string                `json:"reason"`
	Confidence     float64               `json:"confidence"`      // 0.0 to 1.0
	EstimatedCost  float64               `json:"estimated_cost"`  // USD cost estimate
	TransitionPlan *TransitionPlan       `json:"transition_plan,omitempty"`
	RiskFactors    []string              `json:"risk_factors,omitempty"`
	Timestamp      time.Time             `json:"timestamp"`
}

// TransitionActionType represents the type of action to take during transition
type TransitionActionType string

const (
	TransitionActionHold               TransitionActionType = "hold"
	TransitionActionSwitch             TransitionActionType = "switch"
	TransitionActionImmediateExit      TransitionActionType = "immediate_exit"
	TransitionActionGracefulMigration  TransitionActionType = "graceful_migration"
	TransitionActionProtectiveHold     TransitionActionType = "protective_hold"
	TransitionActionFlattenHedge       TransitionActionType = "flatten_hedge"
	TransitionActionConvertToTrend     TransitionActionType = "convert_to_trend"
	TransitionActionGradualUnwind      TransitionActionType = "gradual_unwind"
)

// Recommendation types for transition actions
type RecommendationType string

const (
	RecommendationCloseImmediate RecommendationType = "close_immediate"
	RecommendationScaleOut       RecommendationType = "scale_out"
	RecommendationTightenStops   RecommendationType = "tighten_stops"
	RecommendationConvert        RecommendationType = "convert"
)

// TransitionAction represents a specific action to execute during transition
type TransitionAction struct {
	ID                string                 `json:"id"`
	Type              RecommendationType     `json:"type"`
	Quantity          float64                `json:"quantity"`
	Price             float64                `json:"price"`
	StopLoss          float64                `json:"stop_loss,omitempty"`
	Status            string                 `json:"status"`           // "pending", "executing", "completed", "failed"
	ActualPrice       float64                `json:"actual_price"`
	ActualQuantity    float64                `json:"actual_quantity"`
	TransactionCost   float64                `json:"transaction_cost"`
	Error             string                 `json:"error,omitempty"`
	Timestamp         time.Time              `json:"timestamp"`
}

// TransitionPlan contains the detailed steps for executing a transition
type TransitionPlan struct {
	ID                string                `json:"id"`
	Type              string                `json:"type"`
	Actions           []*TransitionAction   `json:"actions"`
	Steps             []TransitionStep      `json:"steps"`
	TotalCost         float64               `json:"total_cost"`
	Description       string                `json:"description"`
	EstimatedDuration time.Duration         `json:"estimated_duration"`
	RiskLevel         string                `json:"risk_level"`      // "low", "medium", "high"
}

// TransitionStep represents a single step in the transition plan
type TransitionStep struct {
	Type          string    `json:"type"`           // "close_position", "modify_order", "place_order", etc.
	PositionID    string    `json:"position_id,omitempty"`
	OrderID       string    `json:"order_id,omitempty"`
	Description   string    `json:"description"`
	EstimatedCost float64   `json:"estimated_cost"`
	Priority      int       `json:"priority"`       // 1 = highest priority
	Status        string    `json:"status"`         // "pending", "executing", "completed", "failed"
	StartTime     time.Time `json:"start_time,omitempty"`
	CompletedTime time.Time `json:"completed_time,omitempty"`
	ActualCost    float64   `json:"actual_cost"`
	Error         string    `json:"error,omitempty"`
}

// PositionEvaluation contains the assessment of current positions vs new regime
type PositionEvaluation struct {
	Positions                 []engines.EnginePosition `json:"positions"`
	TotalPositions           int                       `json:"total_positions"`
	NetExposure              float64                   `json:"net_exposure"`
	GrossExposure            float64                   `json:"gross_exposure"`
	UnrealizedPnL            float64                   `json:"unrealized_pnl"`
	UnrealizedPnLPercent     float64                   `json:"unrealized_pnl_percent"`
	AveragePositionAge       time.Duration             `json:"average_position_age"`
	RegimeCompatibilityScore float64                   `json:"regime_compatibility_score"` // 0.0 to 1.0
	
	// Risk assessment
	EstimatedExitCost        float64                   `json:"estimated_exit_cost"`
	EstimatedMigrationCost   float64                   `json:"estimated_migration_cost"`
	EstimatedConversionCost  float64                   `json:"estimated_conversion_cost"`
	EstimatedUnwindCost      float64                   `json:"estimated_unwind_cost"`
	
	// Market context
	TrendStrength            float64                   `json:"trend_strength"`           // ADX value
	TrendDirection           int                       `json:"trend_direction"`          // 1, -1, 0
	NetExposureAgainstTrend  bool                     `json:"net_exposure_against_trend"`
	VolatilityLevel          float64                   `json:"volatility_level"`         // ATR or BB width
	SupportResistanceNearby  bool                     `json:"support_resistance_nearby"`
	
	// Position-specific analysis
	ProfitablePositions      int                       `json:"profitable_positions"`
	LosingPositions          int                       `json:"losing_positions"`
	LargestPosition          *PositionRisk             `json:"largest_position,omitempty"`
	RiskiestPosition         *PositionRisk             `json:"riskiest_position,omitempty"`
	
	Timestamp                time.Time                 `json:"timestamp"`
}

// PositionRisk contains risk analysis for a single position
type PositionRisk struct {
	Position          engines.EnginePosition `json:"position"`
	RiskScore         float64                `json:"risk_score"`          // 0.0 to 1.0
	RegimeAlignment   float64                `json:"regime_alignment"`    // -1.0 to 1.0
	TimeAtRisk        time.Duration          `json:"time_at_risk"`
	MaxDrawdown       float64                `json:"max_drawdown"`
	EstimatedSlippage float64                `json:"estimated_slippage"`
	LiquidityRisk     float64                `json:"liquidity_risk"`
}

// RegimeChange represents a regime transition event
type RegimeChange struct {
	Timestamp    time.Time         `json:"timestamp"`
	OldRegime    regime.RegimeType `json:"old_regime"`
	NewRegime    regime.RegimeType `json:"new_regime"`
	Confidence   float64           `json:"confidence"`
	Reason       string            `json:"reason"`
	TriggerPrice float64           `json:"trigger_price"`
	
	// Additional context
	Duration     time.Duration     `json:"duration"`      // How long old regime lasted
	Volatility   float64           `json:"volatility"`    // Market volatility at time of change
	Volume       float64           `json:"volume"`        // Volume at time of change
}

// ExecutorConfig contains configuration for transition execution
type ExecutorConfig struct {
	MaxRetries          int           `json:"max_retries"`
	RetryDelay          time.Duration `json:"retry_delay"`
	TimeoutPerAction    time.Duration `json:"timeout_per_action"`
	MaxSlippagePercent  float64       `json:"max_slippage_percent"`
	MinLiquidityThreshold float64     `json:"min_liquidity_threshold"`
	EmergencyStopEnabled  bool        `json:"emergency_stop_enabled"`
}

// ExecutionResult contains the result of executing a transition plan
type ExecutionResult struct {
	PlanID            string                `json:"plan_id"`
	StartTime         time.Time             `json:"start_time"`
	EndTime           time.Time             `json:"end_time"`
	Duration          time.Duration         `json:"duration"`
	CompletedActions  []*TransitionAction   `json:"completed_actions"`
	FailedActions     []*TransitionAction   `json:"failed_actions"`
	TotalCost         float64               `json:"total_cost"`
	AverageSlippage   float64               `json:"average_slippage"`
	Success           bool                  `json:"success"`
	Errors            []string              `json:"errors"`
	Summary           string                `json:"summary"`
}

// PositionEvaluator interface for evaluating position compatibility with new regimes
type PositionEvaluator interface {
	EvaluatePositions(positions []engines.EnginePosition, newRegime regime.RegimeType) (*PositionEvaluation, error)
	CalculateRegimeCompatibility(position engines.EnginePosition, regime regime.RegimeType) float64
	EstimateTransitionCosts(evaluation *PositionEvaluation, targetRegime regime.RegimeType) (float64, error)
}

// NewPositionEvaluator creates a new position evaluator
func NewPositionEvaluator() PositionEvaluator {
	return &DefaultPositionEvaluator{}
}

// DefaultPositionEvaluator provides default position evaluation logic
type DefaultPositionEvaluator struct{}

// EvaluatePositions evaluates current positions against new regime requirements
func (pe *DefaultPositionEvaluator) EvaluatePositions(positions []engines.EnginePosition, newRegime regime.RegimeType) (*PositionEvaluation, error) {
	// Simplified implementation for now
	evaluation := &PositionEvaluation{
		Positions:                positions,
		TotalPositions:          len(positions),
		RegimeCompatibilityScore: 0.5, // Default moderate compatibility
		Timestamp:               time.Now(),
	}
	
	// Calculate basic metrics
	var totalExposure, unrealizedPnL float64
	for _, pos := range positions {
		// This would need actual position interface implementation
		totalExposure += 1000.0 // Placeholder
		unrealizedPnL += 0.0    // Placeholder  
	}
	
	evaluation.GrossExposure = totalExposure
	evaluation.UnrealizedPnL = unrealizedPnL
	
	return evaluation, nil
}

// CalculateRegimeCompatibility calculates how compatible a position is with a regime
func (pe *DefaultPositionEvaluator) CalculateRegimeCompatibility(position engines.EnginePosition, regimeType regime.RegimeType) float64 {
	// Simplified compatibility scoring
	switch regimeType {
	case regime.RegimeTrending:
		return 0.8 // Positions generally good for trending
	case regime.RegimeRanging:
		return 0.6 // Moderate for ranging
	default:
		return 0.5 // Default moderate compatibility
	}
}

// EstimateTransitionCosts estimates the cost of transitioning positions
func (pe *DefaultPositionEvaluator) EstimateTransitionCosts(evaluation *PositionEvaluation, targetRegime regime.RegimeType) (float64, error) {
	// Simplified cost estimation
	baseCost := float64(len(evaluation.Positions)) * 10.0 // $10 per position
	return baseCost, nil
}
