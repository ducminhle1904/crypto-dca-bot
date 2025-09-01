package orchestrator

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/validation"
)

// SingleBacktestWorkflow represents a single backtest workflow
type SingleBacktestWorkflow struct {
	orchestrator   Orchestrator
	config         *config.DCAConfig
	selectedPeriod time.Duration
}

// NewSingleBacktestWorkflow creates a new single backtest workflow
func NewSingleBacktestWorkflow(orchestrator Orchestrator, config *config.DCAConfig, selectedPeriod time.Duration) Workflow {
	return &SingleBacktestWorkflow{
		orchestrator:   orchestrator,
		config:         config,
		selectedPeriod: selectedPeriod,
	}
}

// Execute runs the single backtest workflow
func (w *SingleBacktestWorkflow) Execute() (interface{}, error) {
	return w.orchestrator.RunSingleBacktest(w.config, w.selectedPeriod)
}

// GetWorkflowType returns the workflow type
func (w *SingleBacktestWorkflow) GetWorkflowType() WorkflowType {
	return WorkflowTypeSingle
}

// OptimizationWorkflow represents an optimization workflow
type OptimizationWorkflow struct {
	orchestrator   Orchestrator
	config         *config.DCAConfig
	selectedPeriod time.Duration
	wfConfig       *validation.WalkForwardConfig
}

// NewOptimizationWorkflow creates a new optimization workflow
func NewOptimizationWorkflow(orchestrator Orchestrator, config *config.DCAConfig, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) Workflow {
	return &OptimizationWorkflow{
		orchestrator:   orchestrator,
		config:         config,
		selectedPeriod: selectedPeriod,
		wfConfig:       wfConfig,
	}
}

// Execute runs the optimization workflow
func (w *OptimizationWorkflow) Execute() (interface{}, error) {
	results, bestConfig, err := w.orchestrator.RunOptimizedBacktest(w.config, w.selectedPeriod, w.wfConfig)
	if err != nil {
		return nil, err
	}
	
	return &OptimizationResult{
		Results:    results,
		BestConfig: bestConfig,
	}, nil
}

// GetWorkflowType returns the workflow type
func (w *OptimizationWorkflow) GetWorkflowType() WorkflowType {
	return WorkflowTypeOptimization
}

// IntervalAnalysisWorkflow represents a multi-interval analysis workflow
type IntervalAnalysisWorkflow struct {
	orchestrator   Orchestrator
	config         *config.DCAConfig
	dataRoot       string
	exchange       string
	optimize       bool
	selectedPeriod time.Duration
	wfConfig       *validation.WalkForwardConfig
}

// NewIntervalAnalysisWorkflow creates a new interval analysis workflow
func NewIntervalAnalysisWorkflow(orchestrator Orchestrator, config *config.DCAConfig, dataRoot, exchange string, optimize bool, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) Workflow {
	return &IntervalAnalysisWorkflow{
		orchestrator:   orchestrator,
		config:         config,
		dataRoot:       dataRoot,
		exchange:       exchange,
		optimize:       optimize,
		selectedPeriod: selectedPeriod,
		wfConfig:       wfConfig,
	}
}

// Execute runs the interval analysis workflow
func (w *IntervalAnalysisWorkflow) Execute() (interface{}, error) {
	return w.orchestrator.RunMultiIntervalAnalysis(w.config, w.dataRoot, w.exchange, w.optimize, w.selectedPeriod, w.wfConfig)
}

// GetWorkflowType returns the workflow type
func (w *IntervalAnalysisWorkflow) GetWorkflowType() WorkflowType {
	return WorkflowTypeInterval
}

// OptimizationResult represents the result of an optimization workflow
type OptimizationResult struct {
	Results    *backtest.BacktestResults
	BestConfig *config.DCAConfig
}
