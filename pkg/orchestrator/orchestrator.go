package orchestrator

import (
	"fmt"
	"log"
	"strings"
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
	start := time.Now()
	
	log.Println("🚀 Starting Single DCA Bot Backtest")
	log.Printf("📊 Symbol: %s", cfg.Symbol)
	log.Printf("💰 Initial Balance: $%.2f", cfg.InitialBalance)
	log.Printf("📈 Base DCA Amount: $%.2f", cfg.BaseAmount)
	log.Printf("🔄 Max Multiplier: %.2f", cfg.MaxMultiplier)
	
	results, err := o.backtestRunner.RunWithFile(cfg, selectedPeriod)
	if err != nil {
		return nil, err
	}
	
	log.Printf("⚡ Performance: Single backtest completed in %s", time.Since(start).Truncate(time.Millisecond))
	return results, nil
}

// RunOptimizedBacktest executes optimization + backtest workflow
func (o *DefaultOrchestrator) RunOptimizedBacktest(cfg *config.DCAConfig, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) (*backtest.BacktestResults, *config.DCAConfig, error) {
	start := time.Now()
	
	log.Println("🚀 Starting DCA Bot Optimization")
	log.Printf("📊 Symbol: %s", cfg.Symbol)
	
	// Check if walk-forward validation is enabled
	if wfConfig != nil && wfConfig.Enable {
		// Run walk-forward validation first
		err := o.runWalkForwardValidation(cfg, selectedPeriod, wfConfig)
		if err != nil {
			log.Printf("⚠️ Walk-forward validation failed: %v", err)
			// Continue with regular optimization
		}
	}
	
	// Run genetic algorithm optimization
	log.Println("🧬 Running genetic algorithm optimization...")
	optimizationStart := time.Now()
	
	bestResults, bestConfigInterface, err := optimization.OptimizeWithGA(cfg, cfg.DataFile, selectedPeriod)
	if err != nil {
		return nil, nil, fmt.Errorf("optimization failed: %w", err)
	}
	
	bestConfig := bestConfigInterface.(*config.DCAConfig)
	
	optimizationTime := time.Since(optimizationStart)
	totalTime := time.Since(start)
	
	log.Printf("✅ Optimization completed - Best return: %.2f%%", bestResults.TotalReturn*100)
	log.Printf("⚡ Performance: Optimization took %s (Total: %s)", 
		optimizationTime.Truncate(time.Millisecond), 
		totalTime.Truncate(time.Millisecond))
	
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

// runWalkForwardValidation executes walk-forward validation with clean, concise logging
func (o *DefaultOrchestrator) runWalkForwardValidation(cfg *config.DCAConfig, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) error {
	log.Printf("🔄 Walk-Forward Validation: %s", cfg.Symbol)
	log.Printf("   Mode: %s", func() string {
		if wfConfig.Rolling {
			return fmt.Sprintf("Rolling (Train=%dd, Test=%dd, Roll=%dd)", wfConfig.TrainDays, wfConfig.TestDays, wfConfig.RollDays)
		}
		return fmt.Sprintf("Holdout (Split=%.0f%%)", wfConfig.SplitRatio*100)
	}())
	
	// Load data for validation
	data, err := datamanager.LoadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to load data for validation: %w", err)
	}
	
	if selectedPeriod > 0 {
		data = datamanager.FilterDataByPeriod(data, selectedPeriod)
	}
	
	log.Printf("   Data: %d candles (%s → %s)", 
		len(data), 
		data[0].Timestamp.Format("2006-01-02"), 
		data[len(data)-1].Timestamp.Format("2006-01-02"))
	
	// Run walk-forward validation in quiet mode
	summary, err := validation.RunQuietWalkForwardValidation(cfg, data, *wfConfig,
		func(config interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error) {
			// Quiet optimization - no per-fold logging
			return optimization.OptimizeWithGA(config, cfg.DataFile, 0)
		},
		func(cfg interface{}, data []types.OHLCV) *backtest.BacktestResults {
			// Quiet testing - no per-fold logging
			dcaConfig := cfg.(*config.DCAConfig)
			results, err := o.backtestRunner.RunWithData(dcaConfig, data)
			if err != nil {
				return nil
			}
			return results
		})
	
	if err != nil {
		return fmt.Errorf("walk-forward validation failed: %w", err)
	}
	
	// Display clean summary results
	o.printCleanWalkForwardSummary(summary)
	
	return nil
}

// printCleanWalkForwardSummary displays a concise, table-formatted walk-forward validation summary
func (o *DefaultOrchestrator) printCleanWalkForwardSummary(summary *validation.WalkForwardSummary) {
	log.Printf("\n📊 Walk-Forward Validation Results (%d folds)", len(summary.Results))
	log.Println(strings.Repeat("-", 65))
	
	// Clean table header
	if len(summary.Results) > 1 {
		log.Println("Fold | Train Return | Test Return  | Degradation | Status")
		log.Println("-----|--------------|--------------|-------------|-------")
		
		for _, result := range summary.Results {
			degradation := (result.TrainResults.TotalReturn - result.TestResults.TotalReturn) * 100
			status := "✅"
			if degradation > 50 {
				status = "🚨"
			} else if degradation > 15 {
				status = "⚠️"
			}
			
			log.Printf("%3d  | %+10.1f%% | %+10.1f%% | %+9.1f%% | %s",
				result.Fold,
				result.TrainResults.TotalReturn*100,
				result.TestResults.TotalReturn*100,
				degradation,
				status)
		}
		log.Println(strings.Repeat("-", 65))
	}
	
	// Concise summary
	log.Printf("Summary: Train=%.1f%%, Test=%.1f%%, Degradation=%.1f%%", 
		summary.AverageTrainReturn*100, 
		summary.AverageTestReturn*100, 
		summary.ReturnDegradation*100)
	
	// Quick assessment
	assessment := "🚨 HIGH RISK"
	if summary.ReturnDegradation < 0.05 {
		assessment = "✅ ROBUST"
	} else if summary.ReturnDegradation < 0.15 {
		assessment = "⚠️  MODERATE"
	}
	
	profitable := ""
	if summary.AverageTestReturn > 0 {
		profitable = "Profitable"
	} else {
		profitable = "Unprofitable"
	}
	
	log.Printf("Assessment: %s | %s | Risk: %s", assessment, profitable, summary.OverfittingRisk)
	log.Println(strings.Repeat("-", 65) + "\n")
}
