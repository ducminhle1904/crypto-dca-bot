package transition

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/engines"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// TransitionManager orchestrates the entire transition process when regimes change
// Based on the regime_transition_solution.json specifications
type TransitionManager struct {
	// Core dependencies
	logger               *logger.Logger
	exchange             exchange.LiveTradingExchange
	positionEvaluator    *PositionEvaluator
	transitionPolicies   *TransitionPolicies
	transitionExecutor   *TransitionExecutor
	
	// Configuration
	config               *TransitionConfig
	
	// State management
	currentTransition    *ActiveTransition
	transitionHistory    []*TransitionRecord
	transitionMutex      sync.RWMutex
	
	// Performance tracking
	metrics              *TransitionMetrics
	metricsMutex         sync.RWMutex
	
	// Risk controls
	dailyTransitionCount int
	dailyTransitionCost  float64
	lastResetDate        time.Time
	
	// Safety mechanisms
	emergencyStop        bool
	manualOverride       bool
}

// TransitionConfig holds configuration for transition management
type TransitionConfig struct {
	// Daily limits
	MaxDailyTransitions      int       `json:"max_daily_transitions"`       // Max 10 transitions per day
	MaxDailyTransitionCost   float64   `json:"max_daily_transition_cost"`   // Max 1% portfolio per day
	
	// Transition thresholds
	RegimeConfidenceThreshold float64  `json:"regime_confidence_threshold"` // Min 70% confidence for transition
	MinRegimeDuration         time.Duration `json:"min_regime_duration"`     // Min 30 minutes in regime
	TransitionCooldown        time.Duration `json:"transition_cooldown"`     // 5 minutes between transitions
	
	// Risk controls
	MaxTransitionCost         float64  `json:"max_transition_cost"`         // Max 0.1% per transition
	EmergencyExitThreshold    float64  `json:"emergency_exit_threshold"`    // -3% PnL emergency exit
	
	// Policy selection
	DefaultPolicy            string    `json:"default_policy"`              // "conservative", "aggressive", "adaptive"
}

// ActiveTransition represents a transition currently in progress
type ActiveTransition struct {
	ID                  string                    `json:"id"`
	StartTime           time.Time                 `json:"start_time"`
	FromRegime          regime.RegimeType         `json:"from_regime"`
	ToRegime            regime.RegimeType         `json:"to_regime"`
	FromEngine          engines.EngineType        `json:"from_engine"`
	ToEngine            engines.EngineType        `json:"to_engine"`
	Status              TransitionStatus          `json:"status"`
	Progress            float64                   `json:"progress"`           // 0.0 to 1.0
	OriginalPositions   []engines.EnginePosition  `json:"original_positions"`
	TargetPositions     []engines.EnginePosition  `json:"target_positions"`
	TransitionPlan      *TransitionPlan           `json:"transition_plan"`
	ActualCost          float64                   `json:"actual_cost"`
	EstimatedCost       float64                   `json:"estimated_cost"`
	LastUpdate          time.Time                 `json:"last_update"`
}

// TransitionStatus represents the status of a transition
type TransitionStatus string

const (
	TransitionStatusEvaluating   TransitionStatus = "evaluating"
	TransitionStatusPlanned      TransitionStatus = "planned"
	TransitionStatusExecuting    TransitionStatus = "executing"
	TransitionStatusCompleted    TransitionStatus = "completed"
	TransitionStatusFailed       TransitionStatus = "failed"
	TransitionStatusCancelled    TransitionStatus = "cancelled"
)

// TransitionRecord tracks completed transitions for analysis
type TransitionRecord struct {
	ID                  string                    `json:"id"`
	Timestamp           time.Time                 `json:"timestamp"`
	FromRegime          regime.RegimeType         `json:"from_regime"`
	ToRegime            regime.RegimeType         `json:"to_regime"`
	FromEngine          engines.EngineType        `json:"from_engine"`
	ToEngine            engines.EngineType        `json:"to_engine"`
	Duration            time.Duration             `json:"duration"`
	TotalCost           float64                   `json:"total_cost"`
	PositionsAffected   int                       `json:"positions_affected"`
	Success             bool                      `json:"success"`
	PolicyUsed          string                    `json:"policy_used"`
	PnLImpact           float64                   `json:"pnl_impact"`
	EfficiencyScore     float64                   `json:"efficiency_score"`   // 0.0 to 1.0
}

// TransitionMetrics tracks overall transition performance
type TransitionMetrics struct {
	TotalTransitions       int       `json:"total_transitions"`
	SuccessfulTransitions  int       `json:"successful_transitions"`
	FailedTransitions      int       `json:"failed_transitions"`
	AverageTransitionCost  float64   `json:"average_transition_cost"`
	AverageTransitionTime  time.Duration `json:"average_transition_time"`
	TotalTransitionCost    float64   `json:"total_transition_cost"`
	SuccessRate            float64   `json:"success_rate"`
	LastTransitionTime     time.Time `json:"last_transition_time"`
	
	// Daily tracking
	DailyTransitionCount   int       `json:"daily_transition_count"`
	DailyTransitionCost    float64   `json:"daily_transition_cost"`
}

// DefaultTransitionConfig returns default transition configuration
func DefaultTransitionConfig() *TransitionConfig {
	return &TransitionConfig{
		MaxDailyTransitions:       10,                    // Max 10 transitions per day
		MaxDailyTransitionCost:    0.01,                  // Max 1% portfolio per day
		RegimeConfidenceThreshold: 0.7,                   // 70% confidence required
		MinRegimeDuration:         30 * time.Minute,      // 30 minutes minimum
		TransitionCooldown:        5 * time.Minute,       // 5 minutes cooldown
		MaxTransitionCost:         0.001,                 // 0.1% max per transition
		EmergencyExitThreshold:    -0.03,                 // -3% emergency exit
		DefaultPolicy:             "adaptive",            // Adaptive policy
	}
}

// NewTransitionManager creates a new transition manager
func NewTransitionManager(logger *logger.Logger, exchange exchange.LiveTradingExchange) *TransitionManager {
	config := DefaultTransitionConfig()
	
	manager := &TransitionManager{
		logger:               logger,
		exchange:             exchange,
		config:               config,
		transitionHistory:    make([]*TransitionRecord, 0, 100),
		metrics:              &TransitionMetrics{},
		lastResetDate:        time.Now(),
		emergencyStop:        false,
		manualOverride:       false,
	}
	
	// Initialize sub-components
	manager.positionEvaluator = NewPositionEvaluator(logger, exchange)
	manager.transitionPolicies = NewTransitionPolicies(config)
	manager.transitionExecutor = NewTransitionExecutor(logger, exchange)
	
	return manager
}

// EvaluateTransition analyzes current positions vs new regime requirements
// This is the core method that determines if/how to transition engines
func (tm *TransitionManager) EvaluateTransition(ctx context.Context, oldRegime, newRegime regime.RegimeType, 
	fromEngine, toEngine engines.TradingEngine, regimeConfidence float64, marketData []types.OHLCV) (*TransitionDecision, error) {
	
	tm.transitionMutex.Lock()
	defer tm.transitionMutex.Unlock()
	
	// Check if we can perform transitions
	if err := tm.canPerformTransition(regimeConfidence); err != nil {
		return &TransitionDecision{
			Action:     TransitionActionHold,
			Reason:     err.Error(),
			Confidence: 0.0,
		}, nil
	}
	
	// Get current positions from the outgoing engine
	currentPositions := fromEngine.GetCurrentPositions()
	
	// If no positions, transition is safe and cheap
	if len(currentPositions) == 0 {
		return &TransitionDecision{
			Action:        TransitionActionSwitch,
			Reason:        "No open positions - safe to switch engines",
			Confidence:    1.0,
			EstimatedCost: 0.0,
			TransitionPlan: &TransitionPlan{
				Steps: []TransitionStep{
					{
						Type:        "engine_switch",
						Description: "Switch active engine with no position impact",
						EstimatedCost: 0.0,
					},
				},
			},
		}, nil
	}
	
	// Evaluate position compatibility with new regime
	evaluation, err := tm.positionEvaluator.EvaluatePositions(ctx, currentPositions, oldRegime, newRegime, marketData)
	if err != nil {
		return nil, fmt.Errorf("position evaluation failed: %w", err)
	}
	
	// Apply transition decision matrix based on regime change type
	decision := tm.applyTransitionMatrix(oldRegime, newRegime, evaluation, regimeConfidence, marketData)
	
	// Validate decision against risk limits
	if err := tm.validateTransitionDecision(decision); err != nil {
		decision.Action = TransitionActionHold
		decision.Reason = fmt.Sprintf("Risk validation failed: %s", err.Error())
		decision.Confidence = 0.0
	}
	
	tm.logger.Info("🔄 Transition evaluation: %s -> %s | Action: %s (%.1f%% confidence)", 
		oldRegime.String(), newRegime.String(), decision.Action, decision.Confidence*100)
	
	return decision, nil
}

// ExecuteTransition performs the actual position adjustments based on the transition decision
func (tm *TransitionManager) ExecuteTransition(ctx context.Context, decision *TransitionDecision, 
	fromEngine, toEngine engines.TradingEngine, regimeChange *regime.RegimeChange) error {
	
	if decision.Action == TransitionActionHold {
		tm.logger.Info("🔄 Transition execution skipped: %s", decision.Reason)
		return nil
	}
	
	// Create active transition record
	transition := &ActiveTransition{
		ID:                fmt.Sprintf("trans_%d", time.Now().Unix()),
		StartTime:         time.Now(),
		FromRegime:        regimeChange.OldRegime,
		ToRegime:          regimeChange.NewRegime,
		FromEngine:        fromEngine.GetType(),
		ToEngine:          toEngine.GetType(),
		Status:            TransitionStatusExecuting,
		Progress:          0.0,
		OriginalPositions: fromEngine.GetCurrentPositions(),
		TransitionPlan:    decision.TransitionPlan,
		EstimatedCost:     decision.EstimatedCost,
		LastUpdate:        time.Now(),
	}
	
	tm.currentTransition = transition
	
	tm.logger.Trade("🔄 TRANSITION START: %s (%s -> %s) | Est. Cost: $%.2f", 
		transition.ID, 
		transition.FromRegime.String(),
		transition.ToRegime.String(), 
		transition.EstimatedCost)
	
	// Execute the transition plan
	actualCost, err := tm.transitionExecutor.ExecuteTransitionPlan(ctx, transition.TransitionPlan, fromEngine, toEngine)
	if err != nil {
		transition.Status = TransitionStatusFailed
		tm.recordTransition(transition, false, actualCost)
		return fmt.Errorf("transition execution failed: %w", err)
	}
	
	// Update transition record
	transition.Status = TransitionStatusCompleted
	transition.Progress = 1.0
	transition.ActualCost = actualCost
	transition.LastUpdate = time.Now()
	
	// Record successful transition
	tm.recordTransition(transition, true, actualCost)
	
	tm.logger.Trade("🎉 TRANSITION COMPLETE: %s | Actual Cost: $%.2f | Duration: %v", 
		transition.ID, actualCost, time.Since(transition.StartTime))
	
	return nil
}

// MonitorTransition tracks transition progress and handles edge cases
func (tm *TransitionManager) MonitorTransition() {
	tm.transitionMutex.RLock()
	transition := tm.currentTransition
	tm.transitionMutex.RUnlock()
	
	if transition == nil || transition.Status != TransitionStatusExecuting {
		return
	}
	
	// Check for timeout
	if time.Since(transition.StartTime) > 10*time.Minute {
		tm.logger.LogWarning("Transition Timeout", "Transition %s taking too long", transition.ID)
		// Could implement timeout handling here
	}
	
	// Update progress (this would be updated by the executor in a real implementation)
	transition.LastUpdate = time.Now()
}

// applyTransitionMatrix implements the decision logic from regime_transition_solution.json
func (tm *TransitionManager) applyTransitionMatrix(oldRegime, newRegime regime.RegimeType, 
	evaluation *PositionEvaluation, confidence float64, marketData []types.OHLCV) *TransitionDecision {
	
	// Trend to Chop (Ranging) transition
	if oldRegime == regime.RegimeTrending && newRegime == regime.RegimeRanging {
		return tm.handleTrendToChopTransition(evaluation, confidence, marketData)
	}
	
	// Chop (Ranging) to Trend transition  
	if oldRegime == regime.RegimeRanging && newRegime == regime.RegimeTrending {
		return tm.handleChopToTrendTransition(evaluation, confidence, marketData)
	}
	
	// Other regime transitions (use adaptive approach)
	return tm.handleGenericTransition(oldRegime, newRegime, evaluation, confidence)
}

// handleTrendToChopTransition implements the trend_to_chop decision matrix
func (tm *TransitionManager) handleTrendToChopTransition(evaluation *PositionEvaluation, 
	confidence float64, marketData []types.OHLCV) *TransitionDecision {
	
	// Immediate Exit Conditions
	if evaluation.UnrealizedPnLPercent < -0.02 && // < -2%
		confidence > 0.8 &&
		evaluation.AveragePositionAge > 4*time.Hour {
		
		return &TransitionDecision{
			Action:        TransitionActionImmediateExit,
			Reason:        "Immediate exit: losing position in high-confidence regime change",
			Confidence:    0.9,
			EstimatedCost: evaluation.EstimatedExitCost,
			TransitionPlan: tm.createImmediateExitPlan(evaluation.Positions),
		}
	}
	
	// Graceful Migration Conditions  
	if evaluation.UnrealizedPnLPercent > 0 && // Profitable
		evaluation.AveragePositionAge < 2*time.Hour &&
		confidence < 0.7 {
		
		return &TransitionDecision{
			Action:        TransitionActionGracefulMigration,
			Reason:        "Graceful migration: convert profitable trend position to grid anchor",
			Confidence:    0.7,
			EstimatedCost: evaluation.EstimatedMigrationCost,
			TransitionPlan: tm.createGridMigrationPlan(evaluation.Positions),
		}
	}
	
	// Protective Hold Conditions
	if evaluation.UnrealizedPnLPercent > -0.01 { // > -1%
		return &TransitionDecision{
			Action:        TransitionActionProtectiveHold,
			Reason:        "Protective hold: tighten stops and monitor",
			Confidence:    0.5,
			EstimatedCost: 0.0,
			TransitionPlan: tm.createProtectiveHoldPlan(evaluation.Positions),
		}
	}
	
	// Default: Hold position
	return &TransitionDecision{
		Action:     TransitionActionHold,
		Reason:     "No clear transition criteria met",
		Confidence: 0.3,
	}
}

// handleChopToTrendTransition implements the chop_to_trend decision matrix  
func (tm *TransitionManager) handleChopToTrendTransition(evaluation *PositionEvaluation, 
	confidence float64, marketData []types.OHLCV) *TransitionDecision {
	
	// Flatten Hedge Conditions (high trend strength)
	if evaluation.TrendStrength > 25 && // ADX > 25
		confidence > 0.8 &&
		evaluation.NetExposureAgainstTrend {
		
		return &TransitionDecision{
			Action:        TransitionActionFlattenHedge,
			Reason:        "Flatten hedge: strong trend detected, net exposure against trend",
			Confidence:    0.9,
			EstimatedCost: evaluation.EstimatedExitCost,
			TransitionPlan: tm.createHedgeFlattenPlan(evaluation.Positions),
		}
	}
	
	// Convert to Trend Conditions
	if !evaluation.NetExposureAgainstTrend && // Net exposure aligned with trend
		evaluation.UnrealizedPnLPercent > 0 {
		
		return &TransitionDecision{
			Action:        TransitionActionConvertToTrend,
			Reason:        "Convert to trend: net exposure aligned, profitable positions",
			Confidence:    0.8,
			EstimatedCost: evaluation.EstimatedConversionCost,
			TransitionPlan: tm.createTrendConversionPlan(evaluation.Positions),
		}
	}
	
	// Gradual Unwind Conditions
	if evaluation.TrendStrength > 20 && evaluation.TrendStrength <= 25 &&
		evaluation.UnrealizedPnLPercent > 0 {
		
		return &TransitionDecision{
			Action:        TransitionActionGradualUnwind,
			Reason:        "Gradual unwind: moderate trend strength, profitable positions",
			Confidence:    0.6,
			EstimatedCost: evaluation.EstimatedUnwindCost,
			TransitionPlan: tm.createGradualUnwindPlan(evaluation.Positions),
		}
	}
	
	// Default: Hold
	return &TransitionDecision{
		Action:     TransitionActionHold,
		Reason:     "No clear grid-to-trend transition criteria met",
		Confidence: 0.3,
	}
}

// handleGenericTransition handles other regime transitions with adaptive approach
func (tm *TransitionManager) handleGenericTransition(oldRegime, newRegime regime.RegimeType, 
	evaluation *PositionEvaluation, confidence float64) *TransitionDecision {
	
	// High confidence regime change with incompatible positions
	if confidence > 0.8 && evaluation.RegimeCompatibilityScore < 0.3 {
		return &TransitionDecision{
			Action:        TransitionActionImmediateExit,
			Reason:        "High confidence regime change with incompatible positions",
			Confidence:    confidence,
			EstimatedCost: evaluation.EstimatedExitCost,
			TransitionPlan: tm.createImmediateExitPlan(evaluation.Positions),
		}
	}
	
	// Moderate confidence with reasonable compatibility
	if confidence > 0.6 && evaluation.RegimeCompatibilityScore > 0.5 {
		return &TransitionDecision{
			Action:        TransitionActionSwitch,
			Reason:        "Moderate confidence regime change with compatible positions",
			Confidence:    confidence * 0.8,
			EstimatedCost: 0.0,
		}
	}
	
	// Default: Hold and monitor
	return &TransitionDecision{
		Action:     TransitionActionHold,
		Reason:     "Generic transition criteria not met",
		Confidence: 0.3,
	}
}

// Helper methods to create transition plans (simplified implementations)

func (tm *TransitionManager) createImmediateExitPlan(positions []engines.EnginePosition) *TransitionPlan {
	steps := make([]TransitionStep, len(positions))
	totalCost := 0.0
	
	for i, pos := range positions {
		steps[i] = TransitionStep{
			Type:        "immediate_exit",
			PositionID:  pos.GetID(),
			Description: fmt.Sprintf("Immediately close %s position", pos.GetSide()),
			EstimatedCost: pos.GetEntryPrice() * pos.GetSize() * 0.001, // 0.1% slippage
		}
		totalCost += steps[i].EstimatedCost
	}
	
	return &TransitionPlan{
		Type:        "immediate_exit",
		Steps:       steps,
		TotalCost:   totalCost,
		Description: "Close all positions immediately",
	}
}

func (tm *TransitionManager) createGridMigrationPlan(positions []engines.EnginePosition) *TransitionPlan {
	// Implement grid migration logic
	return &TransitionPlan{
		Type:        "grid_migration",
		Steps:       []TransitionStep{},
		TotalCost:   0.0,
		Description: "Convert trend positions to grid anchor points",
	}
}

func (tm *TransitionManager) createProtectiveHoldPlan(positions []engines.EnginePosition) *TransitionPlan {
	// Implement protective hold logic (tighten stops)
	return &TransitionPlan{
		Type:        "protective_hold",
		Steps:       []TransitionStep{},
		TotalCost:   0.0,
		Description: "Tighten stop losses and monitor positions",
	}
}

func (tm *TransitionManager) createHedgeFlattenPlan(positions []engines.EnginePosition) *TransitionPlan {
	// Implement hedge flattening logic
	return &TransitionPlan{
		Type:        "hedge_flatten",
		Steps:       []TransitionStep{},
		TotalCost:   0.0,
		Description: "Close counter-trend hedge positions",
	}
}

func (tm *TransitionManager) createTrendConversionPlan(positions []engines.EnginePosition) *TransitionPlan {
	// Implement trend conversion logic  
	return &TransitionPlan{
		Type:        "trend_conversion",
		Steps:       []TransitionStep{},
		TotalCost:   0.0,
		Description: "Convert grid positions to trend-aligned positions",
	}
}

func (tm *TransitionManager) createGradualUnwindPlan(positions []engines.EnginePosition) *TransitionPlan {
	// Implement gradual unwind logic
	return &TransitionPlan{
		Type:        "gradual_unwind",
		Steps:       []TransitionStep{},
		TotalCost:   0.0,
		Description: "Gradually unwind counter-trend positions",
	}
}

// Risk validation and safety methods

func (tm *TransitionManager) canPerformTransition(confidence float64) error {
	// Check emergency stop
	if tm.emergencyStop {
		return fmt.Errorf("emergency stop activated")
	}
	
	// Check manual override
	if tm.manualOverride {
		return fmt.Errorf("manual override active")
	}
	
	// Check confidence threshold
	if confidence < tm.config.RegimeConfidenceThreshold {
		return fmt.Errorf("regime confidence %.2f below threshold %.2f", confidence, tm.config.RegimeConfidenceThreshold)
	}
	
	// Check daily limits
	if err := tm.checkDailyLimits(); err != nil {
		return err
	}
	
	// Check cooldown
	if tm.metrics.LastTransitionTime.After(time.Now().Add(-tm.config.TransitionCooldown)) {
		return fmt.Errorf("transition cooldown active")
	}
	
	return nil
}

func (tm *TransitionManager) checkDailyLimits() error {
	tm.resetDailyLimitsIfNeeded()
	
	if tm.dailyTransitionCount >= tm.config.MaxDailyTransitions {
		return fmt.Errorf("daily transition limit reached: %d", tm.config.MaxDailyTransitions)
	}
	
	if tm.dailyTransitionCost >= tm.config.MaxDailyTransitionCost {
		return fmt.Errorf("daily transition cost limit reached: $%.2f", tm.dailyTransitionCost)
	}
	
	return nil
}

func (tm *TransitionManager) resetDailyLimitsIfNeeded() {
	now := time.Now()
	if now.Day() != tm.lastResetDate.Day() {
		tm.dailyTransitionCount = 0
		tm.dailyTransitionCost = 0.0
		tm.lastResetDate = now
	}
}

func (tm *TransitionManager) validateTransitionDecision(decision *TransitionDecision) error {
	if decision.EstimatedCost > tm.config.MaxTransitionCost {
		return fmt.Errorf("transition cost $%.4f exceeds limit $%.4f", decision.EstimatedCost, tm.config.MaxTransitionCost)
	}
	
	return nil
}

func (tm *TransitionManager) recordTransition(transition *ActiveTransition, success bool, actualCost float64) {
	duration := time.Since(transition.StartTime)
	
	record := &TransitionRecord{
		ID:                transition.ID,
		Timestamp:         transition.StartTime,
		FromRegime:        transition.FromRegime,
		ToRegime:          transition.ToRegime,
		FromEngine:        transition.FromEngine,
		ToEngine:          transition.ToEngine,
		Duration:          duration,
		TotalCost:         actualCost,
		PositionsAffected: len(transition.OriginalPositions),
		Success:           success,
		PolicyUsed:        tm.config.DefaultPolicy,
		PnLImpact:         actualCost, // Simplified
		EfficiencyScore:   tm.calculateEfficiencyScore(transition, actualCost),
	}
	
	tm.transitionHistory = append(tm.transitionHistory, record)
	if len(tm.transitionHistory) > 100 {
		tm.transitionHistory = tm.transitionHistory[1:]
	}
	
	// Update metrics
	tm.metricsMutex.Lock()
	tm.metrics.TotalTransitions++
	if success {
		tm.metrics.SuccessfulTransitions++
	} else {
		tm.metrics.FailedTransitions++
	}
	tm.metrics.TotalTransitionCost += actualCost
	tm.metrics.LastTransitionTime = time.Now()
	tm.metrics.SuccessRate = float64(tm.metrics.SuccessfulTransitions) / float64(tm.metrics.TotalTransitions)
	tm.metrics.AverageTransitionCost = tm.metrics.TotalTransitionCost / float64(tm.metrics.TotalTransitions)
	
	// Update daily tracking
	tm.dailyTransitionCount++
	tm.dailyTransitionCost += actualCost
	tm.metrics.DailyTransitionCount = tm.dailyTransitionCount
	tm.metrics.DailyTransitionCost = tm.dailyTransitionCost
	
	tm.metricsMutex.Unlock()
	
	// Clear current transition
	tm.currentTransition = nil
}

func (tm *TransitionManager) calculateEfficiencyScore(transition *ActiveTransition, actualCost float64) float64 {
	// Simple efficiency calculation: how close actual cost was to estimated
	if transition.EstimatedCost == 0 {
		return 1.0
	}
	
	ratio := actualCost / transition.EstimatedCost
	if ratio <= 1.0 {
		return 1.0 // Actual cost was same or less than estimated
	}
	
	// Penalize cost overruns
	return 1.0 / ratio
}

// Public API methods

func (tm *TransitionManager) GetCurrentTransition() *ActiveTransition {
	tm.transitionMutex.RLock()
	defer tm.transitionMutex.RUnlock()
	return tm.currentTransition
}

func (tm *TransitionManager) GetTransitionMetrics() *TransitionMetrics {
	tm.metricsMutex.RLock()
	defer tm.metricsMutex.RUnlock()
	
	metricsCopy := *tm.metrics
	return &metricsCopy
}

func (tm *TransitionManager) GetTransitionHistory() []*TransitionRecord {
	tm.transitionMutex.RLock()
	defer tm.transitionMutex.RUnlock()
	
	history := make([]*TransitionRecord, len(tm.transitionHistory))
	copy(history, tm.transitionHistory)
	return history
}

func (tm *TransitionManager) SetEmergencyStop(stop bool) {
	tm.emergencyStop = stop
	if stop {
		tm.logger.LogWarning("Emergency Stop", "Transition emergency stop activated")
	} else {
		tm.logger.Info("Emergency stop deactivated")
	}
}

func (tm *TransitionManager) SetManualOverride(override bool) {
	tm.manualOverride = override
	if override {
		tm.logger.LogWarning("Manual Override", "Transition manual override activated")
	} else {
		tm.logger.Info("Manual override deactivated")
	}
}

func (tm *TransitionManager) IsTransitioning() bool {
	tm.transitionMutex.RLock()
	defer tm.transitionMutex.RUnlock()
	
	return tm.currentTransition != nil && tm.currentTransition.Status == TransitionStatusExecuting
}