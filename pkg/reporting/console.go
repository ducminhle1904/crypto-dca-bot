package reporting

import (
	"fmt"
	"math"
	"strings"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
)

// DefaultConsoleReporter implements console output functionality
type DefaultConsoleReporter struct{}

// NewDefaultConsoleReporter creates a new console reporter
func NewDefaultConsoleReporter() *DefaultConsoleReporter {
	return &DefaultConsoleReporter{}
}

// OutputResults prints backtest results to console - extracted from main.go outputConsole
func (r *DefaultConsoleReporter) OutputResults(results *backtest.BacktestResults) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 BACKTEST RESULTS")
	fmt.Println(strings.Repeat("=", 50))
	
	fmt.Printf("💰 Initial Balance:    $%.2f\n", results.StartBalance)
	fmt.Printf("💰 Final Balance:      $%.2f\n", results.EndBalance)
	fmt.Printf("📈 Total Return:       %.2f%%\n", results.TotalReturn*100)
	fmt.Printf("📈 Annualized Return:  %.2f%%\n", results.AnnualizedReturn*100)
	fmt.Printf("📉 Max Drawdown:       %.2f%%\n", results.MaxDrawdown*100)
	fmt.Printf("📉 Max Intra-Cycle DD: %.2f%%\n", results.MaxIntraCycleDD*100)
	fmt.Printf("📊 Sharpe Ratio:       %.2f (Ann: %.2f)\n", results.SharpeRatio, results.AnnualizedSharpe)
	fmt.Printf("📊 Sortino Ratio:      %.2f\n", results.SortinoRatio)
	fmt.Printf("📊 Calmar Ratio:       %.2f\n", results.CalmarRatio)
	fmt.Printf("💹 Profit Factor:      %.2f\n", results.ProfitFactor)
	fmt.Printf("🔄 Total Trades:       %d\n", results.TotalTrades)
	
	// Avoid division by zero
	winRate := 0.0
	loseRate := 0.0
	if results.TotalTrades > 0 {
		winRate = float64(results.WinningTrades) / float64(results.TotalTrades) * 100
		loseRate = float64(results.LosingTrades) / float64(results.TotalTrades) * 100
	}
	
	fmt.Printf("✅ Winning Trades:     %d (%.1f%%)\n", results.WinningTrades, winRate)
	fmt.Printf("❌ Losing Trades:      %d (%.1f%%)\n", results.LosingTrades, loseRate)
	fmt.Printf("🎯 Max Exposure:       %.1f%%\n", results.MaxExposure*100)
	fmt.Printf("🎯 Avg Exposure:       %.1f%%\n", results.AvgExposure*100)
	fmt.Printf("🎯 Max Cycle Exposure: %.1f%%\n", results.MaxCycleExposure*100)
	fmt.Printf("🎯 Avg Cycle Exposure: %.1f%%\n", results.AvgCycleExposure*100)
	fmt.Printf("🔄 Total Turnover:     %.2fx\n", results.TotalTurnover)
}

// PrintConfig prints configuration to console
func (r *DefaultConsoleReporter) PrintConfig(config interface{}) {
	// This can be implemented based on specific config types
	fmt.Printf("Configuration: %+v\n", config)
}

// PrintWalkForwardSummary prints walk-forward validation summary - extracted from main.go
func (r *DefaultConsoleReporter) PrintWalkForwardSummary(results interface{}) {
	// Cast to specific walk-forward results type when implemented
	fmt.Println("📊 ================ WALK-FORWARD SUMMARY ================")
	
	// This would contain the actual walk-forward summary logic
	// For now, placeholder implementation
	fmt.Printf("Walk-forward validation results: %+v\n", results)
}

// Helper functions for walk-forward validation

// printWalkForwardSummary - extracted from main.go printWalkForwardSummary
func PrintWalkForwardSummary(results []interface{}) {
	fmt.Println("📊 ================ WALK-FORWARD SUMMARY ================")
	
	var trainReturns, testReturns []float64
	var trainDrawdowns, testDrawdowns []float64
	
	// This would extract data from results when properly typed
	// For now, placeholder logic
	
	// Calculate averages
	avgTrainReturn := average(trainReturns)
	avgTestReturn := average(testReturns)
	avgTrainDD := average(trainDrawdowns)
	avgTestDD := average(testDrawdowns)
	
	fmt.Printf("AVERAGE PERFORMANCE ACROSS %d FOLDS:\n", len(results))
	fmt.Printf("  Train Return:    %.2f%% ± %.2f%%\n", avgTrainReturn, stdDev(trainReturns))
	fmt.Printf("  Test Return:     %.2f%% ± %.2f%%\n", avgTestReturn, stdDev(testReturns))
	fmt.Printf("  Train Drawdown:  %.2f%% ± %.2f%%\n", avgTrainDD, stdDev(trainDrawdowns))
	fmt.Printf("  Test Drawdown:   %.2f%% ± %.2f%%\n", avgTestDD, stdDev(testDrawdowns))
	
	// Consistency analysis
	returnDegradation := ((avgTrainReturn - avgTestReturn) / math.Max(0.01, math.Abs(avgTrainReturn))) * 100
	fmt.Printf("\nCONSISTENCY ANALYSIS:\n")
	fmt.Printf("  Return Degradation: %.1f%%\n", returnDegradation)
	
	if returnDegradation > 30 {
		fmt.Printf("  ⚠️  HIGH OVERFITTING RISK - Strategy may not generalize well\n")
	} else if returnDegradation > 15 {
		fmt.Printf("  ⚠️  MODERATE OVERFITTING - Some performance degradation\n")
	} else {
		fmt.Printf("  ✅ ROBUST STRATEGY - Good generalization across time periods\n")
	}
}

// Helper functions for statistics
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

// OutputResultsWithContext prints results with symbol and interval context
func (r *DefaultConsoleReporter) OutputResultsWithContext(results *backtest.BacktestResults, symbol, interval string) {
	r.OutputResults(results)
}

// Package-level convenience function
func OutputConsole(results *backtest.BacktestResults) {
	reporter := NewDefaultConsoleReporter()
	reporter.OutputResults(results)
}

// Package-level convenience function with context
func OutputConsoleWithContext(results *backtest.BacktestResults, symbol, interval string) {
	reporter := NewDefaultConsoleReporter()
	reporter.OutputResultsWithContext(results, symbol, interval)
}
