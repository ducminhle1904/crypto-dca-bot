package transition

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/engines"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
)

// TransitionExecutor handles the actual execution of transition plans
// This component performs position adjustments, order modifications, and executions
type TransitionExecutor struct {
	logger     *logger.Logger
	exchange   exchange.LiveTradingExchange
	config     *ExecutorConfig
	
	// Execution state
	activeExecutions map[string]*ExecutionContext
	executionMutex   sync.RWMutex
	
	// Performance tracking
	totalExecutions    int
	successfulExecutions int
	totalCost         float64
	totalTime         time.Duration
}

// ExecutionContext tracks the context of an active transition execution
type ExecutionContext struct {
	PlanID        string                `json:"plan_id"`
	StartTime     time.Time            `json:"start_time"`
	Steps         []TransitionStep     `json:"steps"`
	CurrentStep   int                  `json:"current_step"`
	CompletedSteps int                 `json:"completed_steps"`
	TotalCost     float64              `json:"total_cost"`
	Status        string               `json:"status"`     // "executing", "completed", "failed"
	Error         error                `json:"error,omitempty"`
	
	// Execution control
	ctx           context.Context
	cancel        context.CancelFunc
	resultChannel chan ExecutionResult
}

// ExecutionResult represents the result of executing a transition plan
type ExecutionResult struct {
	Success      bool          `json:"success"`
	TotalCost    float64       `json:"total_cost"`
	Duration     time.Duration `json:"duration"`
	StepsCompleted int         `json:"steps_completed"`
	Error        error         `json:"error,omitempty"`
	Details      []StepResult  `json:"details"`
}

// StepResult represents the result of executing a single transition step
type StepResult struct {
	StepIndex   int           `json:"step_index"`
	Success     bool          `json:"success"`
	ActualCost  float64       `json:"actual_cost"`
	Duration    time.Duration `json:"duration"`
	Error       error         `json:"error,omitempty"`
	Details     string        `json:"details,omitempty"`
}

// NewTransitionExecutor creates a new transition executor
func NewTransitionExecutor(logger *logger.Logger, exchange exchange.LiveTradingExchange) *TransitionExecutor {
	return &TransitionExecutor{
		logger:           logger,
		exchange:         exchange,
		config:           DefaultExecutorConfig(),
		activeExecutions: make(map[string]*ExecutionContext),
	}
}

// DefaultExecutorConfig returns default executor configuration
func DefaultExecutorConfig() *ExecutorConfig {
	return &ExecutorConfig{
		MaxSimultaneousSteps: 3,                  // Max 3 parallel steps
		StepTimeoutSeconds:   30,                 // 30 seconds per step
		RetryAttempts:       3,                   // 3 retry attempts
		RetryDelaySeconds:   5,                   // 5 seconds between retries
		RequireConfirmation: false,               // No confirmation required by default
		DryRun:              false,               // Real execution by default
		MaxStepCost:         1000.0,              // Max $1000 per step
		RealtimeUpdates:     true,                // Enable real-time updates
		DetailedLogging:     true,                // Enable detailed logging
	}
}

// ExecuteTransitionPlan executes a complete transition plan
func (te *TransitionExecutor) ExecuteTransitionPlan(ctx context.Context, plan *TransitionPlan, 
	fromEngine, toEngine engines.TradingEngine) (float64, error) {
	
	if plan == nil || len(plan.Steps) == 0 {
		return 0, fmt.Errorf("transition plan is empty")
	}
	
	// Create execution context
	executionCtx, cancel := context.WithTimeout(ctx, time.Duration(len(plan.Steps)*te.config.StepTimeoutSeconds)*time.Second)
	defer cancel()
	
	execution := &ExecutionContext{
		PlanID:        fmt.Sprintf("exec_%d", time.Now().Unix()),
		StartTime:     time.Now(),
		Steps:         plan.Steps,
		CurrentStep:   0,
		CompletedSteps: 0,
		TotalCost:     0,
		Status:        "executing",
		ctx:           executionCtx,
		cancel:        cancel,
		resultChannel: make(chan ExecutionResult, 1),
	}
	
	// Register active execution
	te.executionMutex.Lock()
	te.activeExecutions[execution.PlanID] = execution
	te.executionMutex.Unlock()
	
	defer func() {
		// Clean up active execution
		te.executionMutex.Lock()
		delete(te.activeExecutions, execution.PlanID)
		te.executionMutex.Unlock()
	}()
	
	te.logger.Trade("🔄 EXECUTING TRANSITION PLAN: %s (%d steps)", execution.PlanID, len(plan.Steps))
	
	// Execute plan
	result := te.executeSteps(execution, fromEngine, toEngine)
	
	// Update statistics
	te.totalExecutions++
	if result.Success {
		te.successfulExecutions++
	}
	te.totalCost += result.TotalCost
	te.totalTime += result.Duration
	
	if result.Success {
		te.logger.Trade("✅ TRANSITION PLAN COMPLETED: %s | Cost: $%.2f | Duration: %v", 
			execution.PlanID, result.TotalCost, result.Duration)
	} else {
		te.logger.LogError("Transition Execution Failed", "Plan %s failed: %v", execution.PlanID, result.Error)
	}
	
	return result.TotalCost, result.Error
}

// executeSteps executes all steps in the transition plan
func (te *TransitionExecutor) executeSteps(execution *ExecutionContext, 
	fromEngine, toEngine engines.TradingEngine) ExecutionResult {
	
	startTime := time.Now()
	var stepResults []StepResult
	totalCost := 0.0
	
	// Execute steps sequentially (or in parallel based on priority)
	for i, step := range execution.Steps {
		execution.CurrentStep = i
		
		// Check if execution was cancelled
		select {
		case <-execution.ctx.Done():
			return ExecutionResult{
				Success:        false,
				TotalCost:      totalCost,
				Duration:       time.Since(startTime),
				StepsCompleted: execution.CompletedSteps,
				Error:          fmt.Errorf("execution cancelled"),
				Details:        stepResults,
			}
		default:
		}
		
		// Execute step with retries
		stepResult := te.executeStepWithRetries(step, execution, fromEngine, toEngine)
		stepResults = append(stepResults, stepResult)
		
		if stepResult.Success {
			execution.CompletedSteps++
			totalCost += stepResult.ActualCost
			
			// Update step status
			execution.Steps[i].Status = "completed"
			execution.Steps[i].CompletedTime = time.Now()
			execution.Steps[i].ActualCost = stepResult.ActualCost
			
		} else {
			// Step failed - decide whether to continue or abort
			execution.Steps[i].Status = "failed"
			execution.Steps[i].Error = stepResult.Error.Error()
			
			// For critical steps, abort the entire plan
			if step.Priority == 1 {
				te.logger.LogError("Critical Step Failed", "Critical step %d failed, aborting plan: %v", i, stepResult.Error)
				return ExecutionResult{
					Success:        false,
					TotalCost:      totalCost,
					Duration:       time.Since(startTime),
					StepsCompleted: execution.CompletedSteps,
					Error:          fmt.Errorf("critical step %d failed: %w", i, stepResult.Error),
					Details:        stepResults,
				}
			}
			
			// For non-critical steps, continue but log the failure
			te.logger.LogWarning("Step Failed", "Non-critical step %d failed, continuing: %v", i, stepResult.Error)
		}
		
		// Small delay between steps to avoid overwhelming the exchange
		time.Sleep(100 * time.Millisecond)
	}
	
	execution.Status = "completed"
	execution.TotalCost = totalCost
	
	return ExecutionResult{
		Success:        true,
		TotalCost:      totalCost,
		Duration:       time.Since(startTime),
		StepsCompleted: execution.CompletedSteps,
		Details:        stepResults,
	}
}

// executeStepWithRetries executes a single step with retry logic
func (te *TransitionExecutor) executeStepWithRetries(step TransitionStep, execution *ExecutionContext,
	fromEngine, toEngine engines.TradingEngine) StepResult {
	
	var lastError error
	stepStartTime := time.Now()
	
	for attempt := 0; attempt <= te.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			// Wait before retry
			select {
			case <-execution.ctx.Done():
				return StepResult{
					Success: false,
					Error:   fmt.Errorf("execution cancelled during retry"),
					Duration: time.Since(stepStartTime),
				}
			case <-time.After(time.Duration(te.config.RetryDelaySeconds) * time.Second):
			}
		}
		
		// Execute the step
		cost, err := te.executeSingleStep(step, execution, fromEngine, toEngine)
		if err == nil {
			return StepResult{
				Success:    true,
				ActualCost: cost,
				Duration:   time.Since(stepStartTime),
				Details:    fmt.Sprintf("Completed on attempt %d", attempt+1),
			}
		}
		
		lastError = err
		te.logger.LogWarning("Step Execution Failed", "Step attempt %d failed: %v", attempt+1, err)
	}
	
	return StepResult{
		Success:    false,
		ActualCost: 0,
		Duration:   time.Since(stepStartTime),
		Error:      lastError,
	}
}

// executeSingleStep executes a single transition step
func (te *TransitionExecutor) executeSingleStep(step TransitionStep, execution *ExecutionContext,
	fromEngine, toEngine engines.TradingEngine) (float64, error) {
	
	if te.config.DryRun {
		te.logger.Info("DRY RUN: Would execute step: %s", step.Description)
		return step.EstimatedCost, nil
	}
	
	te.logger.Info("Executing step: %s", step.Description)
	
	switch step.Type {
	case "immediate_exit":
		return te.executeImmediateExit(step, fromEngine)
		
	case "close_position":
		return te.executeClosePosition(step, fromEngine)
		
	case "modify_order":
		return te.executeModifyOrder(step, fromEngine)
		
	case "place_order":
		return te.executePlaceOrder(step, toEngine)
		
	case "engine_switch":
		return te.executeEngineSwitch(step, fromEngine, toEngine)
		
	case "protective_stop":
		return te.executeProtectiveStop(step, fromEngine)
		
	default:
		return 0, fmt.Errorf("unknown step type: %s", step.Type)
	}
}

// Step execution methods

func (te *TransitionExecutor) executeImmediateExit(step TransitionStep, engine engines.TradingEngine) (float64, error) {
	// Find and close the specified position immediately
	positions := engine.GetCurrentPositions()
	
	for _, pos := range positions {
		if pos.GetID() == step.PositionID {
			// Execute market order to close position
			cost := pos.GetEntryPrice() * pos.GetSize() * 0.001 // Estimated slippage cost
			
			te.logger.Trade("🔴 IMMEDIATE EXIT: Closing position %s (%s %.4f at %.4f)", 
				pos.GetID(), pos.GetSide(), pos.GetSize(), pos.GetEntryPrice())
			
			// In a real implementation, this would call the exchange API
			// For now, we'll simulate the cost
			return cost, nil
		}
	}
	
	return 0, fmt.Errorf("position %s not found for immediate exit", step.PositionID)
}

func (te *TransitionExecutor) executeClosePosition(step TransitionStep, engine engines.TradingEngine) (float64, error) {
	// Similar to immediate exit but potentially with limit orders
	positions := engine.GetCurrentPositions()
	
	for _, pos := range positions {
		if pos.GetID() == step.PositionID {
			cost := pos.GetEntryPrice() * pos.GetSize() * 0.0005 // Lower cost for limit orders
			
			te.logger.Trade("📉 CLOSE POSITION: Closing position %s", pos.GetID())
			return cost, nil
		}
	}
	
	return 0, fmt.Errorf("position %s not found for closing", step.PositionID)
}

func (te *TransitionExecutor) executeModifyOrder(step TransitionStep, engine engines.TradingEngine) (float64, error) {
	// Modify existing orders (stop losses, take profits)
	te.logger.Trade("🔧 MODIFY ORDER: Modifying order %s", step.OrderID)
	
	// Minimal cost for order modifications
	return 0.1, nil
}

func (te *TransitionExecutor) executePlaceOrder(step TransitionStep, engine engines.TradingEngine) (float64, error) {
	// Place new orders for the target engine
	te.logger.Trade("📈 PLACE ORDER: Placing new order for engine %s", engine.GetType())
	
	// Small cost for placing orders
	return 0.5, nil
}

func (te *TransitionExecutor) executeEngineSwitch(step TransitionStep, fromEngine, toEngine engines.TradingEngine) (float64, error) {
	// Switch active engines - this is more of a logical switch
	te.logger.Trade("🔄 ENGINE SWITCH: %s -> %s", fromEngine.GetType(), toEngine.GetType())
	
	// No direct cost for engine switching
	return 0, nil
}

func (te *TransitionExecutor) executeProtectiveStop(step TransitionStep, engine engines.TradingEngine) (float64, error) {
	// Tighten stop losses for protection
	te.logger.Trade("🛡️ PROTECTIVE STOP: Tightening stops for position %s", step.PositionID)
	
	// Minimal cost for stop modifications
	return 0.05, nil
}

// Status and monitoring methods

func (te *TransitionExecutor) GetActiveExecutions() map[string]*ExecutionContext {
	te.executionMutex.RLock()
	defer te.executionMutex.RUnlock()
	
	result := make(map[string]*ExecutionContext)
	for id, execution := range te.activeExecutions {
		result[id] = execution
	}
	
	return result
}

func (te *TransitionExecutor) GetExecutorStats() *ExecutorStats {
	successRate := 0.0
	if te.totalExecutions > 0 {
		successRate = float64(te.successfulExecutions) / float64(te.totalExecutions)
	}
	
	avgCost := 0.0
	if te.totalExecutions > 0 {
		avgCost = te.totalCost / float64(te.totalExecutions)
	}
	
	avgTime := time.Duration(0)
	if te.totalExecutions > 0 {
		avgTime = time.Duration(int64(te.totalTime) / int64(te.totalExecutions))
	}
	
	return &ExecutorStats{
		TotalExecutions:      te.totalExecutions,
		SuccessfulExecutions: te.successfulExecutions,
		SuccessRate:          successRate,
		TotalCost:            te.totalCost,
		AverageCost:          avgCost,
		TotalTime:            te.totalTime,
		AverageTime:          avgTime,
		ActiveExecutions:     len(te.activeExecutions),
	}
}

// ExecutorStats contains statistics about transition execution performance
type ExecutorStats struct {
	TotalExecutions      int           `json:"total_executions"`
	SuccessfulExecutions int           `json:"successful_executions"`
	SuccessRate          float64       `json:"success_rate"`
	TotalCost            float64       `json:"total_cost"`
	AverageCost          float64       `json:"average_cost"`
	TotalTime            time.Duration `json:"total_time"`
	AverageTime          time.Duration `json:"average_time"`
	ActiveExecutions     int           `json:"active_executions"`
}

// ExecutionCoordinator manages position adjustments during transitions
// This handles the actual execution of transition plans
type ExecutionCoordinator struct {
	// Execution settings
	maxConcurrentActions int
	defaultSlippageLimit float64
	executionTimeout     time.Duration
	
	// State tracking
	activeExecutions     map[string]*ExecutionStatus
	executionHistory     []*ExecutionRecord
}

// ExecutionStatus tracks the status of an individual action execution
type ExecutionStatus struct {
	ActionID      string        `json:"action_id"`
	Status        string        `json:"status"`        // pending, executing, completed, failed, timeout
	StartTime     time.Time     `json:"start_time"`
	CompletionTime time.Time    `json:"completion_time,omitempty"`
	Progress      float64       `json:"progress"`      // 0-1
	Error         string        `json:"error,omitempty"`
	
	// Execution details
	TargetQuantity  float64     `json:"target_quantity"`
	ExecutedQuantity float64    `json:"executed_quantity"`
	AveragePrice     float64    `json:"average_price"`
	Slippage         float64    `json:"slippage"`
	TransactionCost  float64    `json:"transaction_cost"`
}

// ExecutionRecord stores completed execution details
type ExecutionRecord struct {
	TransitionID  string              `json:"transition_id"`
	Actions       []*ExecutionStatus  `json:"actions"`
	TotalCost     float64             `json:"total_cost"`
	Success       bool                `json:"success"`
	Duration      time.Duration       `json:"duration"`
	Timestamp     time.Time           `json:"timestamp"`
}

// ExecutionResult contains the overall result of executing a transition plan
type ExecutionResult struct {
	Success           bool                `json:"success"`
	TotalCost         float64             `json:"total_cost"`
	Duration          time.Duration       `json:"duration"`
	CompletedActions  []*TransitionAction `json:"completed_actions"`
	FailedActions     []*TransitionAction `json:"failed_actions"`
	AverageSlippage   float64             `json:"average_slippage"`
	Errors            []string            `json:"errors"`
}

// NewExecutionCoordinator creates a new execution coordinator
func NewExecutionCoordinator() *ExecutionCoordinator {
	return &ExecutionCoordinator{
		maxConcurrentActions: 5,             // Max 5 concurrent executions
		defaultSlippageLimit: 0.002,         // 0.2% slippage limit
		executionTimeout:     10 * time.Minute,
		activeExecutions:     make(map[string]*ExecutionStatus),
		executionHistory:     make([]*ExecutionRecord, 0, 100),
	}
}

// ExecuteTransitionPlan executes all actions in a transition plan
func (ec *ExecutionCoordinator) ExecuteTransitionPlan(plan *TransitionPlan) (*ExecutionResult, error) {
	if plan == nil || len(plan.Actions) == 0 {
		return nil, fmt.Errorf("invalid transition plan: no actions to execute")
	}
	
	startTime := time.Now()
	result := &ExecutionResult{
		Success:           true,
		CompletedActions:  make([]*TransitionAction, 0),
		FailedActions:     make([]*TransitionAction, 0),
		Errors:            make([]string, 0),
	}
	
	// Sort actions by priority (immediate actions first)
	sortedActions := ec.sortActionsByPriority(plan.Actions)
	
	// Execute actions in batches to respect concurrency limits
	for i := 0; i < len(sortedActions); i += ec.maxConcurrentActions {
		end := i + ec.maxConcurrentActions
		if end > len(sortedActions) {
			end = len(sortedActions)
		}
		
		batch := sortedActions[i:end]
		batchResults, err := ec.executeBatch(batch)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.Success = false
		}
		
		// Aggregate batch results
		for _, batchResult := range batchResults {
			if batchResult.Status == "completed" {
				// Find the corresponding action
				for _, action := range batch {
					if action.ID == batchResult.ActionID {
						action.Status = "completed"
						action.ActualPrice = batchResult.AveragePrice
						action.ActualQuantity = batchResult.ExecutedQuantity
						action.TransactionCost = batchResult.TransactionCost
						result.CompletedActions = append(result.CompletedActions, action)
						break
					}
				}
			} else {
				// Handle failed action
				for _, action := range batch {
					if action.ID == batchResult.ActionID {
						action.Status = "failed"
						action.Error = batchResult.Error
						result.FailedActions = append(result.FailedActions, action)
						result.Success = false
						break
					}
				}
			}
		}
	}
	
	// Calculate final metrics
	result.Duration = time.Since(startTime)
	result.TotalCost = ec.calculateTotalCost(result.CompletedActions)
	result.AverageSlippage = ec.calculateAverageSlippage(result.CompletedActions)
	
	// Record execution in history
	ec.recordExecution(plan.ID, result)
	
	return result, nil
}

// executeBatch executes a batch of actions concurrently
func (ec *ExecutionCoordinator) executeBatch(actions []*TransitionAction) ([]*ExecutionStatus, error) {
	results := make([]*ExecutionStatus, len(actions))
	
	// TODO: Phase 4 Implementation
	// This is where actual exchange integration happens
	// For now, simulate execution
	
	for i, action := range actions {
		status := &ExecutionStatus{
			ActionID:    action.ID,
			Status:      "executing",
			StartTime:   time.Now(),
			Progress:    0.0,
		}
		
		// Simulate execution based on action type
		executionResult, err := ec.simulateActionExecution(action)
		if err != nil {
			status.Status = "failed"
			status.Error = err.Error()
		} else {
			status.Status = "completed"
			status.CompletionTime = time.Now()
			status.Progress = 1.0
			status.TargetQuantity = action.Quantity
			status.ExecutedQuantity = executionResult.ExecutedQuantity
			status.AveragePrice = executionResult.AveragePrice
			status.Slippage = executionResult.Slippage
			status.TransactionCost = executionResult.TransactionCost
		}
		
		results[i] = status
	}
	
	return results, nil
}

// simulateActionExecution simulates the execution of a single action
// TODO: Phase 4 - Replace with actual exchange integration
func (ec *ExecutionCoordinator) simulateActionExecution(action *TransitionAction) (*ActionExecutionResult, error) {
	// Simulate different execution scenarios based on action type
	switch action.Type {
	case RecommendationCloseImmediate:
		return ec.simulateCloseExecution(action)
	case RecommendationScaleOut:
		return ec.simulateScaleOutExecution(action)
	case RecommendationTightenStops:
		return ec.simulateStopAdjustment(action)
	case RecommendationConvert:
		return ec.simulateConversionExecution(action)
	default:
		return nil, fmt.Errorf("unsupported action type: %s", action.Type.String())
	}
}

// Action execution simulators - TODO: Replace with real exchange calls

func (ec *ExecutionCoordinator) simulateCloseExecution(action *TransitionAction) (*ActionExecutionResult, error) {
	// Simulate market order execution with some slippage
	slippage := 0.001 // 0.1% typical slippage
	if action.Quantity > 0.8 {
		slippage = 0.002 // Higher slippage for large orders
	}
	
	return &ActionExecutionResult{
		ExecutedQuantity: action.Quantity,
		AveragePrice:     action.Price * (1 + slippage), // Assuming sell order
		Slippage:         slippage,
		TransactionCost:  action.Quantity * action.Price * 0.0005, // 0.05% fee
	}, nil
}

func (ec *ExecutionCoordinator) simulateScaleOutExecution(action *TransitionAction) (*ActionExecutionResult, error) {
	// Simulate partial position closure
	slippage := 0.0008 // Lower slippage for smaller orders
	
	return &ActionExecutionResult{
		ExecutedQuantity: action.Quantity,
		AveragePrice:     action.Price * (1 + slippage),
		Slippage:         slippage,
		TransactionCost:  action.Quantity * action.Price * 0.0005,
	}, nil
}

func (ec *ExecutionCoordinator) simulateStopAdjustment(action *TransitionAction) (*ActionExecutionResult, error) {
	// Simulate stop loss adjustment (minimal cost)
	return &ActionExecutionResult{
		ExecutedQuantity: 0, // No quantity for stop adjustments
		AveragePrice:     action.StopLoss,
		Slippage:         0,
		TransactionCost:  0.1, // Minimal fixed cost
	}, nil
}

func (ec *ExecutionCoordinator) simulateConversionExecution(action *TransitionAction) (*ActionExecutionResult, error) {
	// Simulate position conversion (higher cost due to multiple transactions)
	slippage := 0.0015 // Higher slippage for conversion
	
	return &ActionExecutionResult{
		ExecutedQuantity: action.Quantity,
		AveragePrice:     action.Price * (1 + slippage),
		Slippage:         slippage,
		TransactionCost:  action.Quantity * action.Price * 0.001, // 0.1% fee for conversion
	}, nil
}

// Helper methods

func (ec *ExecutionCoordinator) sortActionsByPriority(actions []*TransitionAction) []*TransitionAction {
	// Create a copy to avoid modifying original
	sorted := make([]*TransitionAction, len(actions))
	copy(sorted, actions)
	
	// Simple priority sort: immediate actions first
	// TODO: Implement more sophisticated priority sorting
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if ec.getActionPriority(sorted[i]) < ec.getActionPriority(sorted[j]) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

func (ec *ExecutionCoordinator) getActionPriority(action *TransitionAction) int {
	switch action.Type {
	case RecommendationCloseImmediate:
		return 1 // Highest priority
	case RecommendationTightenStops:
		return 2 // High priority (safety)
	case RecommendationScaleOut:
		return 3 // Medium priority
	case RecommendationConvert:
		return 4 // Lower priority (complex operation)
	default:
		return 5 // Lowest priority
	}
}

func (ec *ExecutionCoordinator) calculateTotalCost(actions []*TransitionAction) float64 {
	totalCost := 0.0
	for _, action := range actions {
		totalCost += action.TransactionCost
	}
	return totalCost
}

func (ec *ExecutionCoordinator) calculateAverageSlippage(actions []*TransitionAction) float64 {
	if len(actions) == 0 {
		return 0.0
	}
	
	totalSlippage := 0.0
	count := 0
	
	for _, action := range actions {
		if action.ActualPrice > 0 && action.Price > 0 {
			slippage := (action.ActualPrice - action.Price) / action.Price
			totalSlippage += slippage
			count++
		}
	}
	
	if count == 0 {
		return 0.0
	}
	
	return totalSlippage / float64(count)
}

func (ec *ExecutionCoordinator) recordExecution(transitionID string, result *ExecutionResult) {
	record := &ExecutionRecord{
		TransitionID: transitionID,
		Actions:      make([]*ExecutionStatus, 0),
		TotalCost:    result.TotalCost,
		Success:      result.Success,
		Duration:     result.Duration,
		Timestamp:    time.Now(),
	}
	
	ec.executionHistory = append(ec.executionHistory, record)
	
	// Keep only last 100 records
	if len(ec.executionHistory) > 100 {
		ec.executionHistory = ec.executionHistory[1:]
	}
}

// ActionExecutionResult holds the result of executing a single action
type ActionExecutionResult struct {
	ExecutedQuantity float64 `json:"executed_quantity"`
	AveragePrice     float64 `json:"average_price"`
	Slippage         float64 `json:"slippage"`
	TransactionCost  float64 `json:"transaction_cost"`
}
