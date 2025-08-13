package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// generateMultiIndicatorCombinations generates all combinations with 2+ indicators
func generateMultiIndicatorCombinations(indicators []string) [][]string {
	var combinations [][]string
	n := len(indicators)
	
	// Generate all combinations from 2 to n indicators
	for mask := 3; mask < (1 << n); mask++ { // Start from 3 (binary 11) to exclude single indicators
		var combo []string
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				combo = append(combo, indicators[i])
			}
		}
		combinations = append(combinations, combo)
	}
	
	return combinations
}

// optimizeExhaustive tests all indicator combinations (2+ indicators) with smaller GA
func optimizeExhaustive(cfg *BacktestConfig, optimizeIndicators bool) (*backtest.BacktestResults, BacktestConfig) {
	// Preload data once for performance
	data, err := loadHistoricalData(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data for optimization: %v", err)
	}
	if selectedPeriod > 0 {
		data = filterDataByPeriod(data, selectedPeriod)
	}

	// Generate all multi-indicator combinations (2+ indicators)
	baseIndicators := []string{"rsi", "macd", "bb", "sma"}
	allCombinations := generateMultiIndicatorCombinations(baseIndicators)
	
	// Setup logging to file
	intervalStr := guessIntervalFromPath(cfg.DataFile)
	if intervalStr == "" { intervalStr = "unknown" }
	logDir := defaultOutputDir(cfg.Symbol, intervalStr)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
	}
	logPath := filepath.Join(logDir, "exhaustive_test.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		logFile = nil
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()
	
	// Helper function to log to both console and file
	logBoth := func(format string, args ...interface{}) {
		message := fmt.Sprintf(format, args...)
		fmt.Print(message)
		if logFile != nil {
			logFile.WriteString(message)
		}
	}
	
	logBoth("🔍 Starting Exhaustive Combination Testing\n")
	logBoth("Testing %d combinations (2+ indicators) with reduced GA\n", len(allCombinations))
	logBoth("Symbol: %s, Interval: %s\n", cfg.Symbol, intervalStr)
	if selectedPeriod > 0 {
		logBoth("Period: %s (%d data points)\n", selectedPeriodRaw, len(data))
	} else {
		logBoth("Data points: %d\n", len(data))
	}
	logBoth("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	logBoth(strings.Repeat("=", 60) + "\n")
	
	var globalBestResults *backtest.BacktestResults
	var globalBestConfig BacktestConfig
	globalBestFitness := -999999.0
	
	for i, combo := range allCombinations {
		logBoth("\n[%d/%d] Testing: %s\n", i+1, len(allCombinations), strings.Join(combo, "+"))
		
		// Create config copy with this indicator combination
		comboCfg := *cfg
		comboCfg.Indicators = combo
		
		// Run smaller GA for this combination
		results, config := optimizeForCombination(&comboCfg, combo, data)
		
		fitness := results.TotalReturn
		logBoth("         Result: %.2f%% return\n", fitness*100)
		logBoth("         Config: base=$%.0f, maxMult=%.1f, tp=%.1f%%, threshold=%.1f%%\n",
			config.BaseAmount, config.MaxMultiplier, config.TPPercent*100, config.PriceThreshold*100)
		
		// Log indicator-specific parameters
		if containsIndicator(config.Indicators, "rsi") {
			logBoth("         RSI: period=%d, oversold=%.0f\n", config.RSIPeriod, config.RSIOversold)
		}
		if containsIndicator(config.Indicators, "macd") {
			logBoth("         MACD: fast=%d, slow=%d, signal=%d\n", config.MACDFast, config.MACDSlow, config.MACDSignal)
		}
		if containsIndicator(config.Indicators, "bb") {
			logBoth("         BB: period=%d, stddev=%.1f\n", config.BBPeriod, config.BBStdDev)
		}
		if containsIndicator(config.Indicators, "sma") {
			logBoth("         SMA: period=%d\n", config.SMAPeriod)
		}
		
		// Track global best
		if fitness > globalBestFitness {
			globalBestFitness = fitness
			globalBestResults = results
			globalBestConfig = config
			logBoth("         🌟 NEW BEST! (%.2f%%)\n", fitness*100)
		}
	}
	
	logBoth("\n" + strings.Repeat("=", 60) + "\n")
	logBoth("🏆 Exhaustive optimization completed!\n")
	logBoth("Best combination: %s (%.2f%% return)\n", 
		strings.Join(globalBestConfig.Indicators, "+"), globalBestFitness*100)
	logBoth("Best config details:\n")
	logBoth("  Base Amount: $%.0f\n", globalBestConfig.BaseAmount)
	logBoth("  Max Multiplier: %.1f\n", globalBestConfig.MaxMultiplier)
	logBoth("  TP Percent: %.1f%%\n", globalBestConfig.TPPercent*100)
	logBoth("  Price Threshold: %.1f%%\n", globalBestConfig.PriceThreshold*100)
	if containsIndicator(globalBestConfig.Indicators, "rsi") {
		logBoth("  RSI: period=%d, oversold=%.0f\n", globalBestConfig.RSIPeriod, globalBestConfig.RSIOversold)
	}
	if containsIndicator(globalBestConfig.Indicators, "macd") {
		logBoth("  MACD: fast=%d, slow=%d, signal=%d\n", globalBestConfig.MACDFast, globalBestConfig.MACDSlow, globalBestConfig.MACDSignal)
	}
	if containsIndicator(globalBestConfig.Indicators, "bb") {
		logBoth("  BB: period=%d, stddev=%.1f\n", globalBestConfig.BBPeriod, globalBestConfig.BBStdDev)
	}
	if containsIndicator(globalBestConfig.Indicators, "sma") {
		logBoth("  SMA: period=%d\n", globalBestConfig.SMAPeriod)
	}
	logBoth("Optimization completed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	
	if logFile != nil {
		fmt.Printf("📝 Detailed log saved to: %s\n", logPath)
	}
	
	return globalBestResults, globalBestConfig
}

// optimizeForCombination runs a smaller GA for a specific indicator combination
func optimizeForCombination(cfg *BacktestConfig, indicators []string, data []types.OHLCV) (*backtest.BacktestResults, BacktestConfig) {
	// Smaller GA parameters for faster execution
	populationSize := 20  // Reduced from 50
	generations := 15     // Reduced from 30
	mutationRate := 0.15
	crossoverRate := 0.8
	eliteSize := 3        // Reduced from 5
	
	// Initialize population for this specific combination
	population := initializePopulationForCombo(cfg, populationSize, indicators)
	
	var bestIndividual *Individual
	var bestResults *backtest.BacktestResults
	
	for gen := 0; gen < generations; gen++ {
		// Evaluate fitness for all individuals
		for i := range population {
			if population[i].fitness == 0 {
				results := runBacktestWithData(&population[i].config, data)
				population[i].fitness = results.TotalReturn
				population[i].results = results
			}
		}
		
		// Sort by fitness
		sortPopulationByFitness(population)
		
		// Track best
		if bestIndividual == nil || population[0].fitness > bestIndividual.fitness {
			bestIndividual = &Individual{
				config:  population[0].config,
				fitness: population[0].fitness,
				results: population[0].results,
			}
			bestResults = population[0].results
		}
		
		// Create next generation
		if gen < generations-1 {
			population = createNextGenerationForCombo(population, eliteSize, crossoverRate, mutationRate, cfg, indicators)
		}
	}
	
	return bestResults, bestIndividual.config
}

// initializePopulationForCombo creates population for a specific indicator combination
func initializePopulationForCombo(cfg *BacktestConfig, size int, indicators []string) []*Individual {
	population := make([]*Individual, size)
	
	// Fixed parameter ranges
	multipliers := []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0}
	tpCandidates := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06}
	priceThresholds := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05}
	rsiPeriods := []int{10, 12, 14, 16, 18, 20, 22, 25}
	rsiOversold := []float64{20, 25, 30, 35, 40}
	macdFast := []int{6, 8, 10, 12, 14, 16, 18}
	macdSlow := []int{20, 22, 24, 26, 28, 30, 32, 35}
	macdSignal := []int{7, 8, 9, 10, 12, 14}
	bbPeriods := []int{10, 14, 16, 18, 20, 22, 25, 28, 30}
	bbStdDev := []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0}
	smaPeriods := []int{15, 20, 25, 30, 40, 50, 60, 75, 100, 120}
	
	for i := 0; i < size; i++ {
		individual := &Individual{
			config: *cfg,
		}
		
		// Fixed indicator combination (no randomization)
		individual.config.Indicators = indicators
		individual.config.BaseAmount = cfg.BaseAmount
		individual.config.MaxMultiplier = randomChoice(multipliers)
		individual.config.PriceThreshold = randomChoice(priceThresholds)
		
		if cfg.Cycle {
			individual.config.TPPercent = randomChoice(tpCandidates)
		} else {
			individual.config.TPPercent = 0
		}
		
		// Randomize indicator parameters
		individual.config.RSIPeriod = randomChoice(rsiPeriods)
		individual.config.RSIOversold = randomChoice(rsiOversold)
		individual.config.MACDFast = randomChoice(macdFast)
		individual.config.MACDSlow = randomChoice(macdSlow)
		individual.config.MACDSignal = randomChoice(macdSignal)
		individual.config.BBPeriod = randomChoice(bbPeriods)
		individual.config.BBStdDev = randomChoice(bbStdDev)
		individual.config.SMAPeriod = randomChoice(smaPeriods)
		
		population[i] = individual
	}
	
	return population
}

// createNextGenerationForCombo creates next generation for a specific combination
func createNextGenerationForCombo(population []*Individual, eliteSize int, crossoverRate, mutationRate float64, cfg *BacktestConfig, indicators []string) []*Individual {
	newPop := make([]*Individual, len(population))
	
	// Elitism
	for i := 0; i < eliteSize; i++ {
		newPop[i] = &Individual{
			config:  population[i].config,
			fitness: population[i].fitness,
			results: population[i].results,
		}
	}
	
	// Fill rest with crossover and mutation
	for i := eliteSize; i < len(population); i++ {
		parent1 := tournamentSelection(population, 3)
		parent2 := tournamentSelection(population, 3)
		
		child := crossover(parent1, parent2, crossoverRate)
		mutateForCombo(child, mutationRate, cfg, indicators)
		
		newPop[i] = child
	}
	
	return newPop
}

// mutateForCombo mutates individual while keeping the indicator combination fixed
func mutateForCombo(individual *Individual, rate float64, cfg *BacktestConfig, indicators []string) {
	if rng.Float64() < rate {
		// Fixed parameter ranges
		multipliers := []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0}
		tpCandidates := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06}
		priceThresholds := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05}
		rsiPeriods := []int{10, 12, 14, 16, 18, 20, 22, 25}
		rsiOversold := []float64{20, 25, 30, 35, 40}
		macdFast := []int{6, 8, 10, 12, 14, 16, 18}
		macdSlow := []int{20, 22, 24, 26, 28, 30, 32, 35}
		macdSignal := []int{7, 8, 9, 10, 12, 14}
		bbPeriods := []int{10, 14, 16, 18, 20, 22, 25, 28, 30}
		bbStdDev := []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0}
		smaPeriods := []int{15, 20, 25, 30, 40, 50, 60, 75, 100, 120}
		
		// Keep indicators fixed, only mutate parameters
		individual.config.Indicators = indicators
		
		// Randomly mutate one parameter (excluding indicator combination)
		switch rng.Intn(11) {
		case 0:
			individual.config.MaxMultiplier = randomChoice(multipliers)
		case 1:
			if cfg.Cycle {
				individual.config.TPPercent = randomChoice(tpCandidates)
			}
		case 2:
			individual.config.PriceThreshold = randomChoice(priceThresholds)
		case 3:
			individual.config.RSIPeriod = randomChoice(rsiPeriods)
		case 4:
			individual.config.RSIOversold = randomChoice(rsiOversold)
		case 5:
			individual.config.MACDFast = randomChoice(macdFast)
		case 6:
			individual.config.MACDSlow = randomChoice(macdSlow)
		case 7:
			individual.config.MACDSignal = randomChoice(macdSignal)
		case 8:
			individual.config.BBPeriod = randomChoice(bbPeriods)
		case 9:
			individual.config.BBStdDev = randomChoice(bbStdDev)
		case 10:
			individual.config.SMAPeriod = randomChoice(smaPeriods)
		}
		
		// Reset fitness
		individual.fitness = 0
		individual.results = nil
	}
} 