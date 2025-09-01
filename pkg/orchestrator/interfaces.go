package orchestrator

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/validation"
)

// Orchestrator coordinates all backtest components and workflows
type Orchestrator interface {
	// RunSingleBacktest executes a single backtest with the given configuration
	RunSingleBacktest(cfg *config.DCAConfig, selectedPeriod time.Duration) (*backtest.BacktestResults, error)
	
	// RunOptimizedBacktest executes optimization + backtest workflow
	RunOptimizedBacktest(cfg *config.DCAConfig, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) (*backtest.BacktestResults, *config.DCAConfig, error)
	
	// RunMultiIntervalAnalysis executes backtests across all available intervals
	RunMultiIntervalAnalysis(cfg *config.DCAConfig, dataRoot, exchange string, optimize bool, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) (*IntervalAnalysisResult, error)
}

// Workflow represents different execution workflows
type Workflow interface {
	// Execute runs the workflow and returns results
	Execute() (interface{}, error)
	
	// GetWorkflowType returns the type of workflow
	GetWorkflowType() WorkflowType
}

// WorkflowType represents different types of workflows
type WorkflowType string

const (
	WorkflowTypeSingle       WorkflowType = "single"
	WorkflowTypeOptimization WorkflowType = "optimization"
	WorkflowTypeValidation   WorkflowType = "validation"
	WorkflowTypeInterval     WorkflowType = "interval"
)

// IntervalResult represents results for a single interval
type IntervalResult struct {
	Interval     string
	Results      *backtest.BacktestResults
	OptimizedCfg *config.DCAConfig
	Error        error
}

// IntervalAnalysisResult represents results from multi-interval analysis
type IntervalAnalysisResult struct {
	Results    []IntervalResult
	BestResult *IntervalResult
	Symbol     string
	Exchange   string
}

// BacktestRunner interface for running individual backtests
type BacktestRunner interface {
	// RunWithData executes a backtest with provided data
	RunWithData(cfg *config.DCAConfig, data []types.OHLCV) (*backtest.BacktestResults, error)
	
	// RunWithFile executes a backtest by loading data from file
	RunWithFile(cfg *config.DCAConfig, selectedPeriod time.Duration) (*backtest.BacktestResults, error)
}

// IntervalRunner interface for multi-interval operations
type IntervalRunner interface {
	// FindAvailableIntervals discovers all available intervals for a symbol
	FindAvailableIntervals(dataRoot, exchange, symbol string) ([]string, error)
	
	// RunForInterval executes workflow for a specific interval
	RunForInterval(cfg *config.DCAConfig, dataRoot, exchange, interval string, optimize bool, selectedPeriod time.Duration) (*IntervalResult, error)
}
