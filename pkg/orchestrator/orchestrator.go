package orchestrator

import (
	"fmt"
	"log"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	datamanager "github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/optimization"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/validation"
)

// DefaultOrchestrator implements the Orchestrator interface
type DefaultOrchestrator struct {
	backtestRunner BacktestRunner
	intervalRunner IntervalRunner
}

// NewOrchestrator creates a new orchestrator with default components
func NewOrchestrator() Orchestrator {
	return &DefaultOrchestrator{
		backtestRunner: NewDefaultBacktestRunner(),
		intervalRunner: NewDefaultIntervalRunner(),
	}
}

// NewOrchestratorWithComponents creates a new orchestrator with custom components
func NewOrchestratorWithComponents(backtestRunner BacktestRunner, intervalRunner IntervalRunner) Orchestrator {
	return &DefaultOrchestrator{
		backtestRunner: backtestRunner,
		intervalRunner: intervalRunner,
	}
}

// RunSingleBacktest executes a single backtest with the given configuration
func (o *DefaultOrchestrator) RunSingleBacktest(cfg *config.DCAConfig, selectedPeriod time.Duration) (*backtest.BacktestResults, error) {
	log.Println("🚀 Starting Single DCA Bot Backtest")
	log.Printf("📊 Symbol: %s", cfg.Symbol)
	log.Printf("💰 Initial Balance: $%.2f", cfg.InitialBalance)
	log.Printf("📈 Base DCA Amount: $%.2f", cfg.BaseAmount)
	log.Printf("🔄 Max Multiplier: %.2f", cfg.MaxMultiplier)
	
	return o.backtestRunner.RunWithFile(cfg, selectedPeriod)
}

// RunOptimizedBacktest executes optimization + backtest workflow
func (o *DefaultOrchestrator) RunOptimizedBacktest(cfg *config.DCAConfig, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) (*backtest.BacktestResults, *config.DCAConfig, error) {
	log.Println("🚀 Starting DCA Bot Optimization")
	log.Printf("📊 Symbol: %s", cfg.Symbol)
	
	// Check if walk-forward validation is enabled
	if wfConfig != nil && wfConfig.Enable {
		log.Println("🔄 Walk-forward validation enabled")
		
		// Run walk-forward validation first
		err := o.runWalkForwardValidation(cfg, selectedPeriod, wfConfig)
		if err != nil {
			log.Printf("⚠️ Walk-forward validation failed: %v", err)
			// Continue with regular optimization
		}
	}
	
	// Run genetic algorithm optimization
	log.Println("🧬 Running genetic algorithm optimization...")
	bestResults, bestConfigInterface, err := optimization.OptimizeWithGA(cfg, cfg.DataFile, selectedPeriod)
	if err != nil {
		return nil, nil, fmt.Errorf("optimization failed: %w", err)
	}
	
	bestConfig := bestConfigInterface.(*config.DCAConfig)
	log.Printf("✅ Optimization completed - Best return: %.2f%%", bestResults.TotalReturn*100)
	
	return bestResults, bestConfig, nil
}

// RunMultiIntervalAnalysis executes backtests across all available intervals
func (o *DefaultOrchestrator) RunMultiIntervalAnalysis(cfg *config.DCAConfig, dataRoot, exchange string, optimize bool, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) (*IntervalAnalysisResult, error) {
	log.Println("🚀 Starting Multi-Interval Analysis")
	log.Printf("📊 Symbol: %s, Exchange: %s", cfg.Symbol, exchange)
	
	// Find available intervals
	intervals, err := o.intervalRunner.FindAvailableIntervals(dataRoot, exchange, cfg.Symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to find available intervals: %w", err)
	}
	
	if len(intervals) == 0 {
		return nil, fmt.Errorf("no intervals found for symbol %s on exchange %s", cfg.Symbol, exchange)
	}
	
	log.Printf("🔍 Found %d intervals: %v", len(intervals), intervals)
	
	var results []IntervalResult
	var bestResult *IntervalResult
	
	// Process each interval
	for _, interval := range intervals {
		log.Printf("🔄 Processing interval: %s", interval)
		
		result, err := o.intervalRunner.RunForInterval(cfg, dataRoot, exchange, interval, optimize, selectedPeriod, wfConfig)
		if err != nil {
			log.Printf("❌ Failed to process interval %s: %v", interval, err)
			results = append(results, IntervalResult{
				Interval: interval,
				Error:    err,
			})
			continue
		}
		
		results = append(results, *result)
		
		// Track best result by total return
		if bestResult == nil || (result.Results != nil && result.Results.TotalReturn > bestResult.Results.TotalReturn) {
			bestResult = result
		}
	}
	
	if bestResult == nil {
		return nil, fmt.Errorf("no successful results found for any interval")
	}
	
	log.Printf("✅ Multi-interval analysis completed - Best interval: %s (%.2f%%)", 
		bestResult.Interval, bestResult.Results.TotalReturn*100)
	
	return &IntervalAnalysisResult{
		Results:    results,
		BestResult: bestResult,
		Symbol:     cfg.Symbol,
		Exchange:   exchange,
	}, nil
}

// runWalkForwardValidation executes walk-forward validation
func (o *DefaultOrchestrator) runWalkForwardValidation(cfg *config.DCAConfig, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) error {
	// This is a simplified implementation - in practice you'd want more sophisticated WF validation
	log.Println("🔄 Running walk-forward validation...")
	
	// Load data for validation
	data, err := datamanager.LoadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to load data for validation: %w", err)
	}
	
	if selectedPeriod > 0 {
		data = datamanager.FilterDataByPeriod(data, selectedPeriod)
	}
	
	// Run walk-forward validation using the validation package
	_, err = validation.RunWalkForwardValidation(cfg, data, *wfConfig,
		func(config interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error) {
			// Optimization function for training phases
			return optimization.OptimizeWithGA(config, cfg.DataFile, 0)
		},
		func(cfg interface{}, data []types.OHLCV) *backtest.BacktestResults {
			// Evaluation function for test phases
			dcaConfig := cfg.(*config.DCAConfig)
			results, err := o.backtestRunner.RunWithData(dcaConfig, data)
			if err != nil {
				log.Printf("Error in validation backtest: %v", err)
				return nil
			}
			return results
		})
	
	if err != nil {
		return fmt.Errorf("walk-forward validation failed: %w", err)
	}
	
	log.Println("✅ Walk-forward validation completed")
	return nil
}
