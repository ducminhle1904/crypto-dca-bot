package validation

import (
	"fmt"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DefaultWalkForwardValidator implements walk-forward validation
type DefaultWalkForwardValidator struct {
	splitter  DataSplitter
	optimizer func(config interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error)
	backtester func(config interface{}, data []types.OHLCV) *backtest.BacktestResults
}

// NewDefaultWalkForwardValidator creates a new walk-forward validator
func NewDefaultWalkForwardValidator() *DefaultWalkForwardValidator {
	return &DefaultWalkForwardValidator{
		splitter: NewDefaultDataSplitter(),
	}
}

// SetOptimizer sets the optimization function to use
func (v *DefaultWalkForwardValidator) SetOptimizer(optimizer func(config interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error)) {
	v.optimizer = optimizer
}

// SetBacktester sets the backtest function to use
func (v *DefaultWalkForwardValidator) SetBacktester(backtester func(config interface{}, data []types.OHLCV) *backtest.BacktestResults) {
	v.backtester = backtester
}

// Validate performs walk-forward validation - extracted from main.go runWalkForwardValidation
func (v *DefaultWalkForwardValidator) Validate(config interface{}, data []types.OHLCV, wfConfig WalkForwardConfig) (*WalkForwardSummary, error) {
	fmt.Println("\n🔄 ================ WALK-FORWARD VALIDATION ================")
	
	if wfConfig.Rolling {
		return v.validateRolling(config, data, wfConfig)
	} else {
		return v.validateHoldout(config, data, wfConfig)
	}
}

// validateRolling performs rolling walk-forward validation
func (v *DefaultWalkForwardValidator) validateRolling(config interface{}, data []types.OHLCV, wfConfig WalkForwardConfig) (*WalkForwardSummary, error) {
	// Rolling walk-forward
	fmt.Printf("Mode: Rolling Walk-Forward\n")
	fmt.Printf("Train: %d days, Test: %d days, Roll: %d days\n", wfConfig.TrainDays, wfConfig.TestDays, wfConfig.RollDays)
	
	folds := v.splitter.CreateRollingFolds(data, wfConfig.TrainDays, wfConfig.TestDays, wfConfig.RollDays)
	if len(folds) == 0 {
		return nil, fmt.Errorf("not enough data for rolling walk-forward validation")
	}
	
	fmt.Printf("Created %d folds\n\n", len(folds))
	
	var allResults []WalkForwardResults
	
	for i, fold := range folds {
		fmt.Printf("📊 Fold %d/%d: Train %s → %s, Test %s → %s\n", 
			i+1, len(folds),
			fold.TrainStart.Format("2006-01-02"),
			fold.TrainEnd.Format("2006-01-02"),
			fold.TestStart.Format("2006-01-02"),
			fold.TestEnd.Format("2006-01-02"))
		
		// Run optimization on training data
		trainResults, bestConfig, err := v.optimizer(config, fold.Train)
		if err != nil {
			return nil, fmt.Errorf("optimization failed for fold %d: %v", i+1, err)
		}
		
		// Test on out-of-sample data
		testResults := v.backtester(bestConfig, fold.Test)
		
		result := WalkForwardResults{
			TrainResults: trainResults,
			TestResults:  testResults,
			BestConfig:   bestConfig,
			Fold:         i + 1,
		}
		
		allResults = append(allResults, result)
		
		fmt.Printf("  Train: %.2f%% return, %.2f%% drawdown\n", 
			trainResults.TotalReturn*100, trainResults.MaxDrawdown*100)
		fmt.Printf("  Test:  %.2f%% return, %.2f%% drawdown\n\n", 
			testResults.TotalReturn*100, testResults.MaxDrawdown*100)
	}
	
	// Calculate summary
	summary := v.calculateSummary(allResults)
	v.printRollingSummary(summary)
	
	return summary, nil
}

// validateHoldout performs simple holdout validation
func (v *DefaultWalkForwardValidator) validateHoldout(config interface{}, data []types.OHLCV, wfConfig WalkForwardConfig) (*WalkForwardSummary, error) {
	// Simple holdout validation
	fmt.Printf("Mode: Simple Holdout\n")
	fmt.Printf("Split: %.0f%% train, %.0f%% test\n", wfConfig.SplitRatio*100, (1-wfConfig.SplitRatio)*100)
	
	trainData, testData := v.splitter.SplitByRatio(data, wfConfig.SplitRatio)
	if len(testData) < 50 {
		return nil, fmt.Errorf("not enough test data for validation")
	}
	
	fmt.Printf("Train: %d candles (%s → %s)\n", 
		len(trainData),
		trainData[0].Timestamp.Format("2006-01-02"),
		trainData[len(trainData)-1].Timestamp.Format("2006-01-02"))
	fmt.Printf("Test:  %d candles (%s → %s)\n\n", 
		len(testData),
		testData[0].Timestamp.Format("2006-01-02"),
		testData[len(testData)-1].Timestamp.Format("2006-01-02"))
	
	// Run optimization on training data
	fmt.Println("🧬 Running GA optimization on training data...")
	trainResults, bestConfig, err := v.optimizer(config, trainData)
	if err != nil {
		return nil, fmt.Errorf("optimization failed: %v", err)
	}
	
	// Test on out-of-sample data
	fmt.Println("🧪 Testing optimized parameters on test data...")
	testResults := v.backtester(bestConfig, testData)
	
	// Create single result for holdout validation
	result := WalkForwardResults{
		TrainResults: trainResults,
		TestResults:  testResults,
		BestConfig:   bestConfig,
		Fold:         1,
	}
	
	// Calculate summary
	summary := v.calculateSummary([]WalkForwardResults{result})
	v.printHoldoutResults(trainResults, testResults, summary.ReturnDegradation)
	
	return summary, nil
}

// calculateSummary calculates summary statistics from all results
func (v *DefaultWalkForwardValidator) calculateSummary(results []WalkForwardResults) *WalkForwardSummary {
	if len(results) == 0 {
		return &WalkForwardSummary{}
	}
	
	var trainReturns, testReturns []float64
	var trainDrawdowns, testDrawdowns []float64
	
	for _, r := range results {
		trainReturns = append(trainReturns, r.TrainResults.TotalReturn*100)
		testReturns = append(testReturns, r.TestResults.TotalReturn*100)
		trainDrawdowns = append(trainDrawdowns, r.TrainResults.MaxDrawdown*100)
		testDrawdowns = append(testDrawdowns, r.TestResults.MaxDrawdown*100)
	}
	
	// Calculate averages
	avgTrainReturn := average(trainReturns)
	avgTestReturn := average(testReturns)
	avgTrainDD := average(trainDrawdowns)
	avgTestDD := average(testDrawdowns)
	
	// Calculate return degradation
	returnDegradation := ((avgTrainReturn - avgTestReturn) / math.Max(0.01, math.Abs(avgTrainReturn))) * 100
	
	// Determine robustness
	isRobust := returnDegradation <= 30
	var overfittingRisk string
	if returnDegradation > 30 {
		overfittingRisk = "HIGH"
	} else if returnDegradation > 15 {
		overfittingRisk = "MODERATE"
	} else {
		overfittingRisk = "LOW"
	}
	
	return &WalkForwardSummary{
		Results:              results,
		AverageTrainReturn:   avgTrainReturn,
		AverageTestReturn:    avgTestReturn,
		AverageTrainDrawdown: avgTrainDD,
		AverageTestDrawdown:  avgTestDD,
		ReturnDegradation:    returnDegradation,
		IsRobust:             isRobust,
		OverfittingRisk:      overfittingRisk,
	}
}

// printRollingSummary prints summary for rolling validation
func (v *DefaultWalkForwardValidator) printRollingSummary(summary *WalkForwardSummary) {
	fmt.Println("📊 ================ WALK-FORWARD SUMMARY ================")
	
	trainStdDev := stdDev([]float64{})
	testStdDev := stdDev([]float64{})
	
	// Extract returns for std dev calculation
	var trainReturns, testReturns []float64
	for _, r := range summary.Results {
		trainReturns = append(trainReturns, r.TrainResults.TotalReturn*100)
		testReturns = append(testReturns, r.TestResults.TotalReturn*100)
	}
	trainStdDev = stdDev(trainReturns)
	testStdDev = stdDev(testReturns)
	
	fmt.Printf("AVERAGE PERFORMANCE ACROSS %d FOLDS:\n", len(summary.Results))
	fmt.Printf("  Train Return:    %.2f%% ± %.2f%%\n", summary.AverageTrainReturn, trainStdDev)
	fmt.Printf("  Test Return:     %.2f%% ± %.2f%%\n", summary.AverageTestReturn, testStdDev)
	fmt.Printf("  Train Drawdown:  %.2f%% ± %.2f%%\n", summary.AverageTrainDrawdown, stdDev([]float64{}))
	fmt.Printf("  Test Drawdown:   %.2f%% ± %.2f%%\n", summary.AverageTestDrawdown, stdDev([]float64{}))
	
	// Consistency analysis
	fmt.Printf("\nCONSISTENCY ANALYSIS:\n")
	fmt.Printf("  Return Degradation: %.1f%%\n", summary.ReturnDegradation)
	
	if summary.ReturnDegradation > 30 {
		fmt.Printf("  ⚠️  HIGH OVERFITTING RISK - Strategy may not generalize well\n")
	} else if summary.ReturnDegradation > 15 {
		fmt.Printf("  ⚠️  MODERATE OVERFITTING - Some performance degradation\n")
	} else {
		fmt.Printf("  ✅ ROBUST STRATEGY - Good generalization across time periods\n")
	}
}

// printHoldoutResults prints results for holdout validation
func (v *DefaultWalkForwardValidator) printHoldoutResults(trainResults, testResults *backtest.BacktestResults, returnDegradation float64) {
	fmt.Println("\n📈 ================ WALK-FORWARD RESULTS ================")
	fmt.Printf("TRAIN RESULTS:\n")
	fmt.Printf("  Return:    %.2f%%\n", trainResults.TotalReturn*100)
	fmt.Printf("  Drawdown:  %.2f%%\n", trainResults.MaxDrawdown*100)
	fmt.Printf("  Trades:    %d\n", trainResults.TotalTrades)
	
	trainResults.UpdateMetrics()
	fmt.Printf("  Sharpe:    %.2f\n", trainResults.SharpeRatio)
	fmt.Printf("  ProfitFactor: %.2f\n", trainResults.ProfitFactor)
	
	fmt.Printf("\nTEST RESULTS (Out-of-Sample):\n")
	fmt.Printf("  Return:    %.2f%%\n", testResults.TotalReturn*100)
	fmt.Printf("  Drawdown:  %.2f%%\n", testResults.MaxDrawdown*100)
	fmt.Printf("  Trades:    %d\n", testResults.TotalTrades)
	
	testResults.UpdateMetrics()
	fmt.Printf("  Sharpe:    %.2f\n", testResults.SharpeRatio)
	fmt.Printf("  ProfitFactor: %.2f\n", testResults.ProfitFactor)
	
	// Performance degradation analysis
	fmt.Printf("\n📊 ANALYSIS:\n")
	fmt.Printf("  Return Degradation: %.1f%%\n", returnDegradation)
	
	if returnDegradation > 50 {
		fmt.Printf("  ⚠️  HIGH OVERFITTING RISK - Test performance much worse than train\n")
	} else if returnDegradation > 20 {
		fmt.Printf("  ⚠️  MODERATE OVERFITTING - Some performance degradation\n")
	} else if returnDegradation < -10 {
		fmt.Printf("  ✅ ROBUST STRATEGY - Test performance better than train\n")
	} else {
		fmt.Printf("  ✅ GOOD GENERALIZATION - Consistent train/test performance\n")
	}
}

// Helper functions

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	avg := average(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - avg
		sumSquares += diff * diff
	}
	
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

// Package-level convenience functions

// RunWalkForwardValidation is a convenience function using the default validator
func RunWalkForwardValidation(config interface{}, data []types.OHLCV, wfConfig WalkForwardConfig, 
	optimizer func(interface{}, []types.OHLCV) (*backtest.BacktestResults, interface{}, error),
	backtester func(interface{}, []types.OHLCV) *backtest.BacktestResults) (*WalkForwardSummary, error) {
	
	validator := NewDefaultWalkForwardValidator()
	validator.SetOptimizer(optimizer)
	validator.SetBacktester(backtester)
	
	return validator.Validate(config, data, wfConfig)
}
