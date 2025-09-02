package transition

import (
	"fmt"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
)

// TransitionPolicies manages configurable policies for different transition scenarios
// Based on the regime_transition_solution.json policy configurations
type TransitionPolicies struct {
	policies        map[string]*TransitionPolicy
	activePolicy    string
	config          *TransitionConfig
}

// NewTransitionPolicies creates transition policies with default configurations
func NewTransitionPolicies(config *TransitionConfig) *TransitionPolicies {
	tp := &TransitionPolicies{
		policies:     make(map[string]*TransitionPolicy),
		activePolicy: config.DefaultPolicy,
		config:       config,
	}
	
	// Initialize default policies
	tp.initializeDefaultPolicies()
	
	return tp
}

// initializeDefaultPolicies creates the three standard policy types from the plan
func (tp *TransitionPolicies) initializeDefaultPolicies() {
	// Conservative Policy - Minimize transition costs, accept some regime mismatch
	conservative := &TransitionPolicy{
		Name:                    "conservative",
		Description:             "Minimize transition costs, accept some regime mismatch",
		FromRegimes:            []regime.RegimeType{regime.RegimeTrending, regime.RegimeRanging, regime.RegimeVolatile, regime.RegimeUncertain},
		ToRegimes:              []regime.RegimeType{regime.RegimeTrending, regime.RegimeRanging, regime.RegimeVolatile, regime.RegimeUncertain},
		MinConfidence:          0.8,                        // High confidence required
		PreferredAction:        TransitionActionHold,       // Default to holding positions
		FallbackAction:         TransitionActionProtectiveHold,
		MaxCostThreshold:       0.005,                      // 0.5% max cost
		RequireConfirmation:    true,
		MaxDailyApplications:   5,                          // Conservative limit
		CooldownPeriod:         15 * time.Minute,           // Longer cooldown
	}
	
	// Set specific thresholds for conservative policy
	conservative.MaxPnLThreshold = ptrFloat64(-0.03)  // Only exit if losing >3%
	
	tp.policies["conservative"] = conservative
	
	// Aggressive Policy - Optimize for regime alignment, accept higher costs
	aggressive := &TransitionPolicy{
		Name:                    "aggressive",
		Description:             "Optimize for regime alignment, accept higher transition costs",
		FromRegimes:            []regime.RegimeType{regime.RegimeTrending, regime.RegimeRanging, regime.RegimeVolatile, regime.RegimeUncertain},
		ToRegimes:              []regime.RegimeType{regime.RegimeTrending, regime.RegimeRanging, regime.RegimeVolatile, regime.RegimeUncertain},
		MinConfidence:          0.6,                        // Lower confidence threshold
		PreferredAction:        TransitionActionImmediateExit,
		FallbackAction:         TransitionActionGracefulMigration,
		MaxCostThreshold:       0.015,                      // 1.5% max cost
		RequireConfirmation:    false,
		MaxDailyApplications:   15,                         // Higher limit
		CooldownPeriod:         5 * time.Minute,            // Shorter cooldown
	}
	
	// Set specific thresholds for aggressive policy
	aggressive.MinPnLThreshold = ptrFloat64(0.0)  // Any counter-regime position
	
	tp.policies["aggressive"] = aggressive
	
	// Adaptive Policy - Dynamic based on market conditions and performance
	adaptive := &TransitionPolicy{
		Name:                    "adaptive",
		Description:             "Dynamic policy based on market conditions and performance",
		FromRegimes:            []regime.RegimeType{regime.RegimeTrending, regime.RegimeRanging, regime.RegimeVolatile, regime.RegimeUncertain},
		ToRegimes:              []regime.RegimeType{regime.RegimeTrending, regime.RegimeRanging, regime.RegimeVolatile, regime.RegimeUncertain},
		MinConfidence:          0.7,                        // Medium confidence
		PreferredAction:        TransitionActionGracefulMigration,
		FallbackAction:         TransitionActionProtectiveHold,
		MaxCostThreshold:       0.01,                       // 1.0% max cost
		RequireConfirmation:    false,
		MaxDailyApplications:   10,                         // Moderate limit
		CooldownPeriod:         7 * time.Minute,            // Medium cooldown
	}
	
	tp.policies["adaptive"] = adaptive
}

// GetActivePolicy returns the currently active policy
func (tp *TransitionPolicies) GetActivePolicy() *TransitionPolicy {
	if policy, exists := tp.policies[tp.activePolicy]; exists {
		return policy
	}
	// Fallback to conservative if active policy not found
	return tp.policies["conservative"]
}

// SetActivePolicy changes the active policy
func (tp *TransitionPolicies) SetActivePolicy(policyName string) error {
	if _, exists := tp.policies[policyName]; !exists {
		return fmt.Errorf("policy '%s' does not exist", policyName)
	}
	
	tp.activePolicy = policyName
	return nil
}

// EvaluatePolicy determines which action to take based on the active policy
func (tp *TransitionPolicies) EvaluatePolicy(evaluation *PositionEvaluation, 
	oldRegime, newRegime regime.RegimeType, confidence float64) TransitionAction {
	
	policy := tp.GetActivePolicy()
	
	// Check if policy applies to this regime transition
	if !tp.policyApplies(policy, oldRegime, newRegime) {
		return TransitionActionHold
	}
	
	// Check confidence threshold
	if confidence < policy.MinConfidence {
		return TransitionActionHold
	}
	
	// Check daily application limits
	if policy.ApplicationCount >= policy.MaxDailyApplications {
		return TransitionActionHold
	}
	
	// Check cooldown period
	if time.Since(policy.LastApplied) < policy.CooldownPeriod {
		return TransitionActionHold
	}
	
	// Determine action based on position evaluation and policy rules
	return tp.determineActionFromPolicy(policy, evaluation, oldRegime, newRegime, confidence)
}

// policyApplies checks if a policy applies to the given regime transition
func (tp *TransitionPolicies) policyApplies(policy *TransitionPolicy, 
	oldRegime, newRegime regime.RegimeType) bool {
	
	// Check if old regime is in the FromRegimes list
	fromMatch := false
	for _, regime := range policy.FromRegimes {
		if regime == oldRegime {
			fromMatch = true
			break
		}
	}
	
	if !fromMatch {
		return false
	}
	
	// Check if new regime is in the ToRegimes list
	toMatch := false
	for _, regime := range policy.ToRegimes {
		if regime == newRegime {
			toMatch = true
			break
		}
	}
	
	return toMatch
}

// determineActionFromPolicy determines the specific action based on policy rules
func (tp *TransitionPolicies) determineActionFromPolicy(policy *TransitionPolicy,
	evaluation *PositionEvaluation, oldRegime, newRegime regime.RegimeType, confidence float64) TransitionAction {
	
	// Apply policy-specific rules based on position characteristics
	
	// Check PnL thresholds
	if policy.MinPnLThreshold != nil && evaluation.UnrealizedPnLPercent < *policy.MinPnLThreshold {
		return policy.PreferredAction
	}
	
	if policy.MaxPnLThreshold != nil && evaluation.UnrealizedPnLPercent > *policy.MaxPnLThreshold {
		return policy.FallbackAction
	}
	
	// Check position age
	if policy.MaxPositionAge != nil && evaluation.AveragePositionAge > *policy.MaxPositionAge {
		return policy.PreferredAction
	}
	
	// Apply regime-specific logic
	switch {
	case oldRegime == regime.RegimeTrending && newRegime == regime.RegimeRanging:
		return tp.handleTrendToRangePolicy(policy, evaluation, confidence)
		
	case oldRegime == regime.RegimeRanging && newRegime == regime.RegimeTrending:
		return tp.handleRangeToTrendPolicy(policy, evaluation, confidence)
		
	default:
		// For other transitions, use general policy rules
		if evaluation.RegimeCompatibilityScore < 0.5 {
			return policy.PreferredAction
		}
		return policy.FallbackAction
	}
}

// handleTrendToRangePolicy implements trend-to-range transition logic from the plan
func (tp *TransitionPolicies) handleTrendToRangePolicy(policy *TransitionPolicy,
	evaluation *PositionEvaluation, confidence float64) TransitionAction {
	
	// Immediate exit conditions (from regime_transition_solution.json)
	if evaluation.UnrealizedPnLPercent < -0.02 && // < -2%
		confidence > 0.8 &&
		evaluation.AveragePositionAge > 4*time.Hour {
		return TransitionActionImmediateExit
	}
	
	// Graceful migration conditions
	if evaluation.UnrealizedPnLPercent > 0 && // Profitable
		evaluation.AveragePositionAge < 2*time.Hour &&
		confidence < 0.7 {
		return TransitionActionGracefulMigration
	}
	
	// Protective hold conditions
	if evaluation.UnrealizedPnLPercent > -0.01 { // > -1%
		return TransitionActionProtectiveHold
	}
	
	return policy.FallbackAction
}

// handleRangeToTrendPolicy implements range-to-trend transition logic from the plan
func (tp *TransitionPolicies) handleRangeToTrendPolicy(policy *TransitionPolicy,
	evaluation *PositionEvaluation, confidence float64) TransitionAction {
	
	// Flatten hedge conditions
	if evaluation.TrendStrength > 25 && // ADX > 25
		confidence > 0.8 &&
		evaluation.NetExposureAgainstTrend {
		return TransitionActionFlattenHedge
	}
	
	// Convert to trend conditions
	if !evaluation.NetExposureAgainstTrend && // Net exposure aligned with trend
		evaluation.UnrealizedPnLPercent > 0 {
		return TransitionActionConvertToTrend
	}
	
	// Gradual unwind conditions
	if evaluation.TrendStrength > 20 && evaluation.TrendStrength <= 25 &&
		evaluation.UnrealizedPnLPercent > 0 {
		return TransitionActionGradualUnwind
	}
	
	return policy.FallbackAction
}

// RecordPolicyApplication tracks policy usage for performance analysis
func (tp *TransitionPolicies) RecordPolicyApplication(policyName string, success bool, cost float64) {
	if policy, exists := tp.policies[policyName]; exists {
		policy.ApplicationCount++
		if success {
			policy.SuccessCount++
		}
		
		// Update average cost (running average)
		if policy.ApplicationCount == 1 {
			policy.AverageCost = cost
		} else {
			policy.AverageCost = (policy.AverageCost*float64(policy.ApplicationCount-1) + cost) / float64(policy.ApplicationCount)
		}
		
		policy.LastApplied = time.Now()
	}
}

// GetPolicyPerformance returns performance statistics for all policies
func (tp *TransitionPolicies) GetPolicyPerformance() map[string]*PolicyPerformance {
	performance := make(map[string]*PolicyPerformance)
	
	for name, policy := range tp.policies {
		successRate := 0.0
		if policy.ApplicationCount > 0 {
			successRate = float64(policy.SuccessCount) / float64(policy.ApplicationCount)
		}
		
		performance[name] = &PolicyPerformance{
			Name:               name,
			ApplicationCount:   policy.ApplicationCount,
			SuccessCount:       policy.SuccessCount,
			SuccessRate:        successRate,
			AverageCost:        policy.AverageCost,
			LastApplied:        policy.LastApplied,
		}
	}
	
	return performance
}

// ResetDailyLimits resets daily application counts for all policies
func (tp *TransitionPolicies) ResetDailyLimits() {
	for _, policy := range tp.policies {
		policy.ApplicationCount = 0
	}
}

// PolicyPerformance contains performance metrics for a policy
type PolicyPerformance struct {
	Name               string    `json:"name"`
	ApplicationCount   int       `json:"application_count"`
	SuccessCount      int       `json:"success_count"`
	SuccessRate       float64   `json:"success_rate"`
	AverageCost       float64   `json:"average_cost"`
	LastApplied       time.Time `json:"last_applied"`
}

// Helper function to create float64 pointer
func ptrFloat64(f float64) *float64 {
	return &f
}

// NewTransitionPolicies creates a new policy engine
func NewTransitionPolicies(policyType string) *TransitionPolicies {
	switch policyType {
	case "conservative":
		return newConservativePolicy()
	case "aggressive":
		return newAggressivePolicy()
	case "adaptive":
		return newAdaptivePolicy()
	default:
		return newAdaptivePolicy() // Default to adaptive
	}
}

// newConservativePolicy creates conservative policy from transition solution
func newConservativePolicy() *TransitionPolicies {
	return &TransitionPolicies{
		policyType:                 "conservative",
		regimeConfidenceThreshold:  0.8,    // High confidence required
		maxTransitionCost:          0.005,  // 0.5% of portfolio
		positionHoldPreference:     true,   // Prefer to let positions run
		emergencyExitThreshold:     -0.03,  // Only exit if loss > 3%
	}
}

// newAggressivePolicy creates aggressive policy from transition solution
func newAggressivePolicy() *TransitionPolicies {
	return &TransitionPolicies{
		policyType:                 "aggressive",
		regimeConfidenceThreshold:  0.6,    // Lower confidence threshold
		maxTransitionCost:          0.015,  // 1.5% of portfolio
		positionHoldPreference:     false,  // Immediate alignment preferred
		emergencyExitThreshold:     0.0,    // Exit any counter-regime position
	}
}

// newAdaptivePolicy creates adaptive policy from transition solution
func newAdaptivePolicy() *TransitionPolicies {
	return &TransitionPolicies{
		policyType:                 "adaptive",
		regimeConfidenceThreshold:  0.7,    // Dynamic based on conditions
		maxTransitionCost:          0.01,   // 1% of portfolio (adjustable)
		positionHoldPreference:     false,  // Adapts to conditions
		emergencyExitThreshold:     -0.02,  // Dynamic based on volatility
		recentSuccessRate:          0.7,    // Track recent performance
		volatilityAdjustment:       1.0,    // Volatility adjustment factor
		timeBasedAdjustment:        1.0,    // Time-based adjustment factor
	}
}

// GenerateTransitionPlan creates a transition plan based on policy and evaluations
func (tp *TransitionPolicies) GenerateTransitionPlan(regimeChange *regime.RegimeChange, evaluations []*PositionEvaluation) (*TransitionPlan, error) {
	if regimeChange == nil || len(evaluations) == 0 {
		return nil, fmt.Errorf("invalid inputs: regimeChange and evaluations required")
	}
	
	plan := &TransitionPlan{
		ID:                   fmt.Sprintf("transition_%d", time.Now().Unix()),
		TransitionType:       tp.getTransitionType(regimeChange.OldRegime, regimeChange.NewRegime),
		RegimeChange:         regimeChange,
		Actions:              make([]*TransitionAction, 0),
		EstimatedCost:        0.0,
		EstimatedDuration:    0,
		MaxRiskTolerance:     tp.maxTransitionCost,
		Priority:             tp.calculatePriority(regimeChange),
		RequiresConfirmation: tp.shouldRequireConfirmation(regimeChange),
		CreatedAt:            time.Now(),
	}
	
	// Apply policy-specific decision logic
	switch tp.policyType {
	case "conservative":
		return tp.applyConservativePolicy(plan, evaluations)
	case "aggressive":
		return tp.applyAggressivePolicy(plan, evaluations)
	case "adaptive":
		return tp.applyAdaptivePolicy(plan, evaluations)
	default:
		return tp.applyAdaptivePolicy(plan, evaluations)
	}
}

// applyConservativePolicy implements conservative transition decisions
func (tp *TransitionPolicies) applyConservativePolicy(plan *TransitionPlan, evaluations []*PositionEvaluation) (*TransitionPlan, error) {
	// Conservative policy: minimize transition costs, accept some regime mismatch
	for _, eval := range evaluations {
		// Only act on high-confidence, high-risk situations
		if eval.RegimeConfidence < tp.regimeConfidenceThreshold {
			continue
		}
		
		// Emergency exit only for significant losses
		if eval.UnrealizedPnLPercent < tp.emergencyExitThreshold {
			action := &TransitionAction{
				ID:         fmt.Sprintf("action_%d_%s", time.Now().Unix(), eval.PositionID),
				Type:       RecommendationCloseImmediate,
				PositionID: eval.PositionID,
				Quantity:   1.0, // Close 100%
				TimeLimit:  5 * time.Minute,
				Status:     "pending",
			}
			plan.Actions = append(plan.Actions, action)
			plan.EstimatedCost += 0.001 // Estimate 0.1% cost per close
		} else if eval.RiskScore > 0.8 {
			// Tighten stops for high-risk positions
			action := &TransitionAction{
				ID:         fmt.Sprintf("action_%d_%s", time.Now().Unix(), eval.PositionID),
				Type:       RecommendationTightenStops,
				PositionID: eval.PositionID,
				TimeLimit:  1 * time.Minute,
				Status:     "pending",
			}
			plan.Actions = append(plan.Actions, action)
		}
	}
	
	plan.Priority = 3 // Medium priority for conservative approach
	return plan, nil
}

// applyAggressivePolicy implements aggressive transition decisions
func (tp *TransitionPolicies) applyAggressivePolicy(plan *TransitionPlan, evaluations []*PositionEvaluation) (*TransitionPlan, error) {
	// Aggressive policy: optimize for regime alignment, accept higher transition costs
	for _, eval := range evaluations {
		if eval.CompatibilityScore < 0.5 {
			// Close or convert positions with poor compatibility
			var actionType RecommendationType
			if eval.RiskScore > 0.6 {
				actionType = RecommendationCloseImmediate
			} else if eval.UnrealizedPnLPercent > 0 {
				actionType = RecommendationConvert
			} else {
				actionType = RecommendationScaleOut
			}
			
			action := &TransitionAction{
				ID:         fmt.Sprintf("action_%d_%s", time.Now().Unix(), eval.PositionID),
				Type:       actionType,
				PositionID: eval.PositionID,
				Quantity:   tp.calculateActionQuantity(actionType, eval),
				TimeLimit:  3 * time.Minute,
				Status:     "pending",
			}
			plan.Actions = append(plan.Actions, action)
			plan.EstimatedCost += tp.estimateActionCost(action)
		}
	}
	
	plan.Priority = 1 // High priority for aggressive approach
	return plan, nil
}

// applyAdaptivePolicy implements dynamic transition decisions
func (tp *TransitionPolicies) applyAdaptivePolicy(plan *TransitionPlan, evaluations []*PositionEvaluation) (*TransitionPlan, error) {
	// Adaptive policy: adjust based on market conditions and recent performance
	
	// Adjust thresholds based on recent success rate
	confidenceThreshold := tp.regimeConfidenceThreshold
	if tp.recentSuccessRate > 0.8 {
		confidenceThreshold *= 0.9 // Be more aggressive if recent success
	} else if tp.recentSuccessRate < 0.6 {
		confidenceThreshold *= 1.1 // Be more conservative if poor recent performance
	}
	
	// Adjust based on volatility
	volatilityAdjustment := tp.volatilityAdjustment
	if volatilityAdjustment > 1.5 {
		// High volatility: be more conservative
		confidenceThreshold *= 1.2
	}
	
	for _, eval := range evaluations {
		action := tp.determineOptimalAction(eval, confidenceThreshold)
		if action != nil {
			plan.Actions = append(plan.Actions, action)
			plan.EstimatedCost += tp.estimateActionCost(action)
		}
	}
	
	// Adjust priority based on regime confidence and urgency
	if len(evaluations) > 0 {
		avgConfidence := 0.0
		for _, eval := range evaluations {
			avgConfidence += eval.RegimeConfidence
		}
		avgConfidence /= float64(len(evaluations))
		
		if avgConfidence > 0.8 {
			plan.Priority = 1 // High confidence = high priority
		} else if avgConfidence > 0.6 {
			plan.Priority = 2 // Medium confidence = medium priority
		} else {
			plan.Priority = 3 // Low confidence = low priority
		}
	}
	
	return plan, nil
}

// determineOptimalAction determines the best action for a position evaluation
func (tp *TransitionPolicies) determineOptimalAction(eval *PositionEvaluation, confidenceThreshold float64) *TransitionAction {
	// Use evaluation recommendation but apply policy filters
	if eval.Confidence < 0.5 {
		return nil // Skip low-confidence recommendations
	}
	
	if eval.RegimeConfidence < confidenceThreshold {
		// Don't act on low-confidence regime changes
		if eval.Recommendation == RecommendationCloseImmediate {
			// Downgrade to scale out
			eval.Recommendation = RecommendationScaleOut
		}
	}
	
	action := &TransitionAction{
		ID:         fmt.Sprintf("action_%d_%s", time.Now().Unix(), eval.PositionID),
		Type:       eval.Recommendation,
		PositionID: eval.PositionID,
		Quantity:   tp.calculateActionQuantity(eval.Recommendation, eval),
		TimeLimit:  tp.getActionTimeLimit(eval.Recommendation),
		Status:     "pending",
	}
	
	return action
}

// Helper methods

func (tp *TransitionPolicies) getTransitionType(oldRegime, newRegime regime.RegimeType) TransitionType {
	if oldRegime == regime.RegimeTrending && newRegime == regime.RegimeRanging {
		return TransitionTrendToChop
	} else if oldRegime == regime.RegimeRanging && newRegime == regime.RegimeTrending {
		return TransitionChopToTrend
	} else if newRegime == regime.RegimeVolatile {
		return TransitionToVolatile
	} else if oldRegime == regime.RegimeVolatile {
		return TransitionFromVolatile
	}
	return TransitionUncertain
}

func (tp *TransitionPolicies) calculatePriority(regimeChange *regime.RegimeChange) int {
	if regimeChange.Confidence > 0.9 {
		return 1 // Immediate
	} else if regimeChange.Confidence > 0.7 {
		return 2 // High
	} else if regimeChange.Confidence > 0.5 {
		return 3 // Medium
	}
	return 4 // Low
}

func (tp *TransitionPolicies) shouldRequireConfirmation(regimeChange *regime.RegimeChange) bool {
	// Require confirmation for uncertain regime changes in conservative policy
	return tp.policyType == "conservative" && regimeChange.Confidence < 0.8
}

func (tp *TransitionPolicies) calculateActionQuantity(actionType RecommendationType, eval *PositionEvaluation) float64 {
	switch actionType {
	case RecommendationCloseImmediate:
		return 1.0 // Close 100%
	case RecommendationScaleOut:
		if eval.RiskScore > 0.8 {
			return 0.75 // Close 75% if high risk
		}
		return 0.5 // Close 50% otherwise
	case RecommendationTightenStops:
		return 0.0 // No quantity for stop adjustments
	case RecommendationConvert:
		return 1.0 // Convert entire position
	default:
		return 0.0
	}
}

func (tp *TransitionPolicies) getActionTimeLimit(actionType RecommendationType) time.Duration {
	switch actionType {
	case RecommendationCloseImmediate:
		return 2 * time.Minute
	case RecommendationScaleOut:
		return 5 * time.Minute
	case RecommendationTightenStops:
		return 1 * time.Minute
	case RecommendationConvert:
		return 10 * time.Minute
	default:
		return 5 * time.Minute
	}
}

func (tp *TransitionPolicies) estimateActionCost(action *TransitionAction) float64 {
	// Rough cost estimates for different action types
	switch action.Type {
	case RecommendationCloseImmediate:
		return 0.001 * action.Quantity // 0.1% per position closed
	case RecommendationScaleOut:
		return 0.0008 * action.Quantity // 0.08% per position scaled
	case RecommendationTightenStops:
		return 0.0001 // Minimal cost for stop adjustments
	case RecommendationConvert:
		return 0.002 // Higher cost for position conversion
	default:
		return 0.0005
	}
}

// UpdatePolicyParameters allows dynamic adjustment of policy parameters
func (tp *TransitionPolicies) UpdatePolicyParameters(params map[string]interface{}) error {
	if confidenceThreshold, ok := params["regime_confidence_threshold"].(float64); ok {
		tp.regimeConfidenceThreshold = confidenceThreshold
	}
	if maxCost, ok := params["max_transition_cost"].(float64); ok {
		tp.maxTransitionCost = maxCost
	}
	if successRate, ok := params["recent_success_rate"].(float64); ok {
		tp.recentSuccessRate = successRate
	}
	if volatility, ok := params["volatility_adjustment"].(float64); ok {
		tp.volatilityAdjustment = volatility
	}
	
	return nil
}
