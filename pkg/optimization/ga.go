package optimization

import (
	"fmt"
	"log"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/bands"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/common"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/oscillators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/trend"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/volume"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy/spacing"
	configpkg "github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	datamanager "github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// GA Constants - Balanced for large datasets
const (
	GAPopulationSize    = 40   // Moderate population for good exploration without being too slow
	GAGenerations       = 25   // Moderate generations for convergence on large data
	GAMutationRate      = 0.18 // Slightly higher mutation for exploration
	GACrossoverRate     = 0.8  // Good crossover rate for mixing
	GAEliteSize         = 6    // Keep best 6 individuals (~15% of population)
	TournamentSize      = 3    // Standard tournament size
	MaxParallelWorkers  = 6    // Balanced parallel workers
	ProgressReportInterval = 5 // Progress reports every 5 generations
	DetailReportInterval   = 10 // Detailed reports every 10 generations
)

// Individual represents a candidate solution - extracted from main.go
type GAIndividual struct {
	Config  interface{} // BacktestConfig in practice
	Fitness float64
	Results *backtest.BacktestResults
}

// OptimizeWithGA runs genetic algorithm optimization - extracted from main.go optimizeForInterval
func OptimizeWithGA(baseConfig interface{}, dataFile string, selectedPeriod time.Duration) (*backtest.BacktestResults, interface{}, error) {
	// Create local RNG with random seed for non-deterministic optimization
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	// MinOrderQty should be fetched at orchestrator level before calling OptimizeWithGA
	// to ensure consistency with single backtest runs
	
	// Preload data once for performance
	data, err := datamanager.LoadHistoricalDataCached(dataFile)
	if err != nil {
		log.Fatalf("Failed to load data for optimization: %v", err)
	}
	
	if len(data) == 0 {
		log.Fatalf("No valid data found for optimization in file: %s", dataFile)
	}
	
	if selectedPeriod > 0 {
		data = datamanager.FilterDataByPeriod(data, selectedPeriod)
		if len(data) == 0 {
			log.Fatalf("No data remaining for optimization after applying period filter of %v", selectedPeriod)
		}
		log.Printf("ℹ️ Filtered to last %v of data (%s → %s)",
			selectedPeriod,
			data[0].Timestamp.Format("2006-01-02"),
			data[len(data)-1].Timestamp.Format("2006-01-02"))
	}

	// GA Parameters
	populationSize := GAPopulationSize
	generations := GAGenerations
	mutationRate := GAMutationRate
	crossoverRate := GACrossoverRate
	eliteSize := GAEliteSize

	// Silent GA optimization - only show final result

	// Initialize population
	population := InitializePopulation(baseConfig, populationSize, rng)
	
	var bestIndividual *GAIndividual
	
	for gen := 0; gen < generations; gen++ {
		// Evaluate fitness for all individuals in parallel
		EvaluatePopulationParallel(population, data)
		
		// Sort by fitness (descending)
		SortPopulationByFitness(population)
		
		// Track best individual
		if bestIndividual == nil || population[0].Fitness > bestIndividual.Fitness {
			bestIndividual = &GAIndividual{
				Config:  copyConfig(population[0].Config), // Deep copy to prevent mutation
				Fitness: population[0].Fitness,
				Results: population[0].Results,
			}
		}
		
		// Silent generation processing
		
		// Create next generation
		if gen < generations-1 {
			population = CreateNextGeneration(population, eliteSize, crossoverRate, mutationRate, baseConfig, rng)
		}
	}
	
	// Re-run the best configuration to ensure consistency with standalone runs
	// The GA cached results may have strategy state contamination from parallel execution
	// This ensures the final result matches exactly what would be produced by a regular backtest
	finalResults := RunBacktestWithData(bestIndividual.Config, data)
	return finalResults, bestIndividual.Config, nil
}

// InitializePopulation creates initial random population - extracted from main.go
func InitializePopulation(baseConfig interface{}, size int, rng *rand.Rand) []*GAIndividual {
	population := make([]*GAIndividual, size)
	
	for i := 0; i < size; i++ {
		individual := &GAIndividual{
			Config: copyConfig(baseConfig), // Copy base config
		}
		
		// Randomize parameters using optimization ranges
		RandomizeConfig(individual.Config, rng)
		
		population[i] = individual
	}
	
	return population
}

// EvaluatePopulationParallel evaluates fitness for all individuals in parallel - extracted from main.go
func EvaluatePopulationParallel(population []*GAIndividual, data []types.OHLCV) {
	var wg sync.WaitGroup
	
	// Create a channel to limit concurrent goroutines
	workerChan := make(chan struct{}, MaxParallelWorkers)
	
	for i := range population {
		if population[i].Fitness != 0 {
			continue // Skip already evaluated individuals
		}
		
		wg.Add(1)
		go func(individual *GAIndividual) {
			defer wg.Done()
			
			workerChan <- struct{}{} // Acquire worker slot
			defer func() { <-workerChan }() // Release worker slot
			
			results := RunBacktestWithData(individual.Config, data)
			individual.Fitness = results.TotalReturn
			individual.Results = results
		}(population[i])
	}
	
	wg.Wait()
}

// SortPopulationByFitness sorts population by fitness (descending) - optimized with O(n log n) complexity
func SortPopulationByFitness(population []*GAIndividual) {
	sort.Slice(population, func(i, j int) bool {
		return population[i].Fitness > population[j].Fitness // Descending order
	})
}

// AverageFitness calculates average fitness - extracted from main.go
func AverageFitness(population []*GAIndividual) float64 {
	sum := 0.0
	for _, ind := range population {
		sum += ind.Fitness
	}
	return sum / float64(len(population))
}

// CreateNextGeneration creates next generation using selection, crossover, and mutation - extracted from main.go
func CreateNextGeneration(population []*GAIndividual, eliteSize int, crossoverRate, mutationRate float64, baseConfig interface{}, rng *rand.Rand) []*GAIndividual {
	newPop := make([]*GAIndividual, len(population))
	
	// Elitism: keep best individuals
	for i := 0; i < eliteSize; i++ {
		newPop[i] = &GAIndividual{
			Config:  copyConfig(population[i].Config),
			Fitness: population[i].Fitness,
			Results: population[i].Results,
		}
	}
	
	// Fill rest with crossover and mutation
	for i := eliteSize; i < len(population); i++ {
		parent1 := TournamentSelection(population, TournamentSize, rng)
		parent2 := TournamentSelection(population, TournamentSize, rng)
		
		child := Crossover(parent1, parent2, crossoverRate, rng)
		Mutate(child, mutationRate, baseConfig, rng)
		
		newPop[i] = child
	}
	
	return newPop
}

// TournamentSelection performs tournament selection - extracted from main.go
func TournamentSelection(population []*GAIndividual, tournamentSize int, rng *rand.Rand) *GAIndividual {
	best := population[rng.Intn(len(population))]
	
	for i := 1; i < tournamentSize; i++ {
		candidate := population[rng.Intn(len(population))]
		if candidate.Fitness > best.Fitness {
			best = candidate
		}
	}
	
	return best
}

// Crossover creates child from two parents - extracted from main.go
func Crossover(parent1, parent2 *GAIndividual, rate float64, rng *rand.Rand) *GAIndividual {
	child := &GAIndividual{
		Config: copyConfig(parent1.Config), // Start with parent1
	}
	
	// Random crossover based on rate
	if rng.Float64() < rate {
		// Mix parameters from both parents
		CrossoverConfigs(child.Config, parent1.Config, parent2.Config, rng)
	}
	
	return child
}

// Mutate applies mutation to an individual - extracted from main.go
func Mutate(individual *GAIndividual, rate float64, baseConfig interface{}, rng *rand.Rand) {
	if rng.Float64() < rate {
		// Mutate configuration
		MutateConfig(individual.Config, baseConfig, rng)
		
		// Reset fitness to force re-evaluation
		individual.Fitness = 0
		individual.Results = nil
	}
}

// copyConfig creates a deep copy of DCAConfig
func copyConfig(config interface{}) interface{} {
	dcaConfig, ok := config.(*configpkg.DCAConfig)
	if !ok {
		return config // Fallback
	}
	
	// Create a deep copy
	copied := *dcaConfig
	
	// Copy slice fields
	if dcaConfig.Indicators != nil {
		copied.Indicators = make([]string, len(dcaConfig.Indicators))
		copy(copied.Indicators, dcaConfig.Indicators)
	}
	
	if dcaConfig.DCASpacing != nil {
		spacingCopy := *dcaConfig.DCASpacing
		
		// Deep copy the Parameters map
		if dcaConfig.DCASpacing.Parameters != nil {
			spacingCopy.Parameters = make(map[string]interface{})
			for k, v := range dcaConfig.DCASpacing.Parameters {
				spacingCopy.Parameters[k] = v
			}
		}
		
		copied.DCASpacing = &spacingCopy
	}
	
	// Deep copy Dynamic TP configuration
	if dcaConfig.DynamicTP != nil {
		dynamicTPCopy := *dcaConfig.DynamicTP
		
		// Deep copy VolatilityConfig if present
		if dcaConfig.DynamicTP.VolatilityConfig != nil {
			volatilityConfigCopy := *dcaConfig.DynamicTP.VolatilityConfig
			dynamicTPCopy.VolatilityConfig = &volatilityConfigCopy
		}
		
		// Deep copy IndicatorConfig if present
		if dcaConfig.DynamicTP.IndicatorConfig != nil {
			indicatorConfigCopy := *dcaConfig.DynamicTP.IndicatorConfig
			
			// Deep copy the Weights map
			if dcaConfig.DynamicTP.IndicatorConfig.Weights != nil {
				indicatorConfigCopy.Weights = make(map[string]float64)
				for k, v := range dcaConfig.DynamicTP.IndicatorConfig.Weights {
					indicatorConfigCopy.Weights[k] = v
				}
			}
			
			dynamicTPCopy.IndicatorConfig = &indicatorConfigCopy
		}
		
		copied.DynamicTP = &dynamicTPCopy
	}
	
	return &copied
}


func RandomizeConfig(config interface{}, rng *rand.Rand) {
	dcaConfig, ok := config.(*configpkg.DCAConfig)
	if !ok {
		return
	}
	
	// Get optimization ranges
	ranges := GetDefaultOptimizationRanges()
	
	dcaConfig.BaseAmount = 40.0
	
	// Randomize strategy parameters using predefined ranges
	dcaConfig.MaxMultiplier = RandomChoice(ranges.Multipliers, rng)
	dcaConfig.TPPercent = RandomChoice(ranges.TPCandidates, rng)
	
	// Preserve existing DCA spacing strategy or default to fixed
	originalStrategy := "fixed"
	if dcaConfig.DCASpacing != nil {
		originalStrategy = dcaConfig.DCASpacing.Strategy
	}
	
	// Create DCA spacing configuration with randomized parameters based on strategy
	switch originalStrategy {
	case "volatility_adaptive":
		dcaConfig.DCASpacing = &configpkg.DCASpacingConfig{
			Strategy: "volatility_adaptive",
			Parameters: map[string]interface{}{
				"base_threshold":        RandomChoice(ranges.PriceThresholds, rng),
				"volatility_sensitivity": RandomChoice(ranges.VolatilitySensitivity, rng),
				"atr_period":            RandomChoice(ranges.ATRPeriods, rng),
				"level_multiplier":      RandomChoice(ranges.LevelMultipliers, rng),
				"max_threshold":         0.05,  // 5% safety limit for adaptive
				"min_threshold":         0.003, // 0.3% safety limit
			},
		}
	default: // "fixed"
		dcaConfig.DCASpacing = &configpkg.DCASpacingConfig{
			Strategy: "fixed",
			Parameters: map[string]interface{}{
				"base_threshold":       RandomChoice(ranges.PriceThresholds, rng),
				"threshold_multiplier": RandomChoice(ranges.PriceThresholdMultipliers, rng),
				"max_threshold":        0.10, // 10% safety limit
				"min_threshold":        0.003, // 0.3% safety limit
			},
		}
	}
	
	// Default to classic indicators for genetic algorithm optimization if none specified
	if len(dcaConfig.Indicators) == 0 {
		dcaConfig.Indicators = []string{"rsi", "macd", "bb", "ema"}
	}
	
	// Randomize parameters for each indicator that's present
	indicatorSet := make(map[string]bool)
	for _, ind := range dcaConfig.Indicators {
		indicatorSet[strings.ToLower(ind)] = true
	}
	
	// Randomize classic indicator parameters if present
	if indicatorSet["rsi"] {
		dcaConfig.RSIPeriod = RandomChoice(ranges.RSIPeriods, rng)
		dcaConfig.RSIOversold = RandomChoice(ranges.RSIOversold, rng)
		dcaConfig.RSIOverbought = 100.0 - RandomChoice(ranges.RSIOversold, rng)
	}
	if indicatorSet["macd"] {
		dcaConfig.MACDFast = RandomChoice(ranges.MACDFast, rng)
		dcaConfig.MACDSlow = RandomChoice(ranges.MACDSlow, rng)
		dcaConfig.MACDSignal = RandomChoice(ranges.MACDSignal, rng)
	}
	if indicatorSet["bb"] || indicatorSet["bollinger"] {
		dcaConfig.BBPeriod = RandomChoice(ranges.BBPeriods, rng)
		dcaConfig.BBStdDev = RandomChoice(ranges.BBStdDev, rng)
	}
	if indicatorSet["ema"] {
		dcaConfig.EMAPeriod = RandomChoice(ranges.EMAPeriods, rng)
	}
	
	// Randomize advanced indicator parameters if present
	if indicatorSet["hullma"] || indicatorSet["hull_ma"] {
		dcaConfig.HullMAPeriod = RandomChoice(ranges.HullMAPeriods, rng)
	}
	if indicatorSet["supertrend"] || indicatorSet["st"] {
		dcaConfig.SuperTrendPeriod = RandomChoice(ranges.SuperTrendPeriods, rng)
		dcaConfig.SuperTrendMultiplier = RandomChoice(ranges.SuperTrendMultipliers, rng)
	}
	if indicatorSet["mfi"] {
		dcaConfig.MFIPeriod = RandomChoice(ranges.MFIPeriods, rng)
		dcaConfig.MFIOversold = RandomChoice(ranges.MFIOversold, rng)
		dcaConfig.MFIOverbought = RandomChoice(ranges.MFIOverbought, rng)
	}
	if indicatorSet["keltner"] || indicatorSet["kc"] {
		dcaConfig.KeltnerPeriod = RandomChoice(ranges.KeltnerPeriods, rng)
		dcaConfig.KeltnerMultiplier = RandomChoice(ranges.KeltnerMultipliers, rng)
	}
	if indicatorSet["wavetrend"] || indicatorSet["wt"] {
		dcaConfig.WaveTrendN1 = RandomChoice(ranges.WaveTrendN1, rng)
		dcaConfig.WaveTrendN2 = RandomChoice(ranges.WaveTrendN2, rng)
		dcaConfig.WaveTrendOverbought = RandomChoice(ranges.WaveTrendOverbought, rng)
		dcaConfig.WaveTrendOversold = RandomChoice(ranges.WaveTrendOversold, rng)
	}
	if indicatorSet["obv"] {
		dcaConfig.OBVTrendThreshold = RandomChoice(ranges.OBVTrendThresholds, rng)
	}
	if indicatorSet["stochrsi"] || indicatorSet["stochastic_rsi"] || indicatorSet["stoch_rsi"] {
		dcaConfig.StochasticRSIPeriod = RandomChoice(ranges.StochasticRSIPeriods, rng)
		dcaConfig.StochasticRSIOverbought = RandomChoice(ranges.StochasticRSIOverboughts, rng)
		dcaConfig.StochasticRSIOversold = RandomChoice(ranges.StochasticRSIOversolds, rng)
	}
	
	// Randomize Dynamic TP parameters if dynamic TP is configured
	if dcaConfig.DynamicTP != nil {
		// Randomize BaseTPPercent - this is the core TP percentage for dynamic calculations
		dcaConfig.DynamicTP.BaseTPPercent = RandomChoice(ranges.TPCandidates, rng)
		
		switch dcaConfig.DynamicTP.Strategy {
		case "volatility_adaptive":
			if dcaConfig.DynamicTP.VolatilityConfig != nil {
				dcaConfig.DynamicTP.VolatilityConfig.Multiplier = RandomChoice(ranges.TPVolatilityMultipliers, rng)
				dcaConfig.DynamicTP.VolatilityConfig.MinTPPercent = RandomChoice(ranges.TPMinPercents, rng)
				dcaConfig.DynamicTP.VolatilityConfig.MaxTPPercent = RandomChoice(ranges.TPMaxPercents, rng)
				// ATRPeriod will be synchronized with DCA spacing - randomized in DCA spacing parameters
			}
		case "indicator_based":
			if dcaConfig.DynamicTP.IndicatorConfig != nil {
				dcaConfig.DynamicTP.IndicatorConfig.StrengthMultiplier = RandomChoice(ranges.TPStrengthMultipliers, rng)
				dcaConfig.DynamicTP.IndicatorConfig.MinTPPercent = RandomChoice(ranges.TPMinPercents, rng)
				dcaConfig.DynamicTP.IndicatorConfig.MaxTPPercent = RandomChoice(ranges.TPMaxPercents, rng)
				// Weights are preserved from original config for indicator-based TP
			}
		}
	}
	
	// Fix parameter ordering constraints after randomization
	validateAndFixParameterOrdering(dcaConfig)
	
	// Synchronize ATR periods between DCA spacing and Dynamic TP
	synchronizeATRPeriodsInConfig(dcaConfig)
}

func RunBacktestWithData(config interface{}, data []types.OHLCV) *backtest.BacktestResults {
	// Convert interface{} to config.DCAConfig which implements BacktestConfig interface
	dcaConfig, ok := config.(*configpkg.DCAConfig)
	if !ok {
		// Fallback - this shouldn't happen in normal operation
		return &backtest.BacktestResults{TotalReturn: 0.0}
	}
	
	// CRITICAL FIX: Use the SAME strategy creation logic as regular backtest
	// This ensures 100% consistency between optimization and regular backtest runs
	strat, err := createDCAStrategyFromConfig(dcaConfig)
	if err != nil {
		log.Printf("⚠️ GA: Failed to create strategy: %v", err)
		return &backtest.BacktestResults{TotalReturn: 0.0}
	}
	
	// Reset strategy state to prevent contamination from previous runs
	strat.ResetForNewPeriod()
	
	tp := dcaConfig.TPPercent
	if !dcaConfig.Cycle {
		tp = 0
	}
	
	engine := backtest.NewBacktestEngine(dcaConfig.InitialBalance, dcaConfig.Commission, strat, tp, dcaConfig.MinOrderQty, dcaConfig.UseTPLevels)
	results := engine.Run(data, dcaConfig.WindowSize)
	results.UpdateMetrics()
	
	return results
}

func CrossoverConfigs(child, parent1, parent2 interface{}, rng *rand.Rand) {
	childConfig, ok1 := child.(*configpkg.DCAConfig)
	_, ok2 := parent1.(*configpkg.DCAConfig)  // Not used but needed for validation
	parent2Config, ok3 := parent2.(*configpkg.DCAConfig)
	
	if !ok1 || !ok2 || !ok3 {
		return
	}
	
	// Crossover strategy parameters (base amount is not crossed over - it's fixed to min order qty)
	if rng.Float64() < 0.5 {
		childConfig.MaxMultiplier = parent2Config.MaxMultiplier
	}
	// DCA spacing crossover - inherit parent2's spacing parameters
	if parent2Config.DCASpacing != nil && childConfig.DCASpacing != nil && 
	   parent2Config.DCASpacing.Strategy == childConfig.DCASpacing.Strategy {
		// Only crossover if both parents have the same strategy
		for param, value := range parent2Config.DCASpacing.Parameters {
			if rng.Float64() < 0.5 {
				childConfig.DCASpacing.Parameters[param] = value
			}
		}
	}
	
	if rng.Float64() < 0.5 {
		childConfig.TPPercent = parent2Config.TPPercent
	}
	
	// Crossover parameters for each indicator that's present
	indicatorSet := make(map[string]bool)
	for _, ind := range childConfig.Indicators {
		indicatorSet[strings.ToLower(ind)] = true
	}
	
	// Crossover classic indicator parameters if present
	if indicatorSet["rsi"] {
		if rng.Float64() < 0.5 { childConfig.RSIPeriod = parent2Config.RSIPeriod }
		if rng.Float64() < 0.5 { childConfig.RSIOversold = parent2Config.RSIOversold }
		if rng.Float64() < 0.5 { childConfig.RSIOverbought = parent2Config.RSIOverbought }
	}
	if indicatorSet["macd"] {
		if rng.Float64() < 0.5 { childConfig.MACDFast = parent2Config.MACDFast }
		if rng.Float64() < 0.5 { childConfig.MACDSlow = parent2Config.MACDSlow }
		if rng.Float64() < 0.5 { childConfig.MACDSignal = parent2Config.MACDSignal }
	}
	if indicatorSet["bb"] || indicatorSet["bollinger"] {
		if rng.Float64() < 0.5 { childConfig.BBPeriod = parent2Config.BBPeriod }
		if rng.Float64() < 0.5 { childConfig.BBStdDev = parent2Config.BBStdDev }
	}
	if indicatorSet["ema"] {
		if rng.Float64() < 0.5 { childConfig.EMAPeriod = parent2Config.EMAPeriod }
	}
	
	// Crossover advanced indicator parameters if present
	if indicatorSet["hullma"] || indicatorSet["hull_ma"] {
		if rng.Float64() < 0.5 { childConfig.HullMAPeriod = parent2Config.HullMAPeriod }
	}
	if indicatorSet["supertrend"] || indicatorSet["st"] {
		if rng.Float64() < 0.5 { childConfig.SuperTrendPeriod = parent2Config.SuperTrendPeriod }
		if rng.Float64() < 0.5 { childConfig.SuperTrendMultiplier = parent2Config.SuperTrendMultiplier }
	}
	if indicatorSet["mfi"] {
		if rng.Float64() < 0.5 { childConfig.MFIPeriod = parent2Config.MFIPeriod }
		if rng.Float64() < 0.5 { childConfig.MFIOversold = parent2Config.MFIOversold }
		if rng.Float64() < 0.5 { childConfig.MFIOverbought = parent2Config.MFIOverbought }
	}
	if indicatorSet["keltner"] || indicatorSet["kc"] {
		if rng.Float64() < 0.5 { childConfig.KeltnerPeriod = parent2Config.KeltnerPeriod }
		if rng.Float64() < 0.5 { childConfig.KeltnerMultiplier = parent2Config.KeltnerMultiplier }
	}
	if indicatorSet["wavetrend"] || indicatorSet["wt"] {
		if rng.Float64() < 0.5 { childConfig.WaveTrendN1 = parent2Config.WaveTrendN1 }
		if rng.Float64() < 0.5 { childConfig.WaveTrendN2 = parent2Config.WaveTrendN2 }
		if rng.Float64() < 0.5 { childConfig.WaveTrendOverbought = parent2Config.WaveTrendOverbought }
		if rng.Float64() < 0.5 { childConfig.WaveTrendOversold = parent2Config.WaveTrendOversold }
	}
	if indicatorSet["obv"] {
		if rng.Float64() < 0.5 { childConfig.OBVTrendThreshold = parent2Config.OBVTrendThreshold }
	}
	if indicatorSet["stochrsi"] || indicatorSet["stochastic_rsi"] || indicatorSet["stoch_rsi"] {
		if rng.Float64() < 0.5 { childConfig.StochasticRSIPeriod = parent2Config.StochasticRSIPeriod }
		if rng.Float64() < 0.5 { childConfig.StochasticRSIOverbought = parent2Config.StochasticRSIOverbought }
		if rng.Float64() < 0.5 { childConfig.StochasticRSIOversold = parent2Config.StochasticRSIOversold }
	}
	
	// Crossover Dynamic TP parameters if both parents have dynamic TP configured
	if childConfig.DynamicTP != nil && parent2Config.DynamicTP != nil {
		// Crossover BaseTPPercent - critical for dynamic TP optimization
		if rng.Float64() < 0.5 {
			childConfig.DynamicTP.BaseTPPercent = parent2Config.DynamicTP.BaseTPPercent
		}
		
		switch childConfig.DynamicTP.Strategy {
		case "volatility_adaptive":
			if childConfig.DynamicTP.VolatilityConfig != nil && parent2Config.DynamicTP.VolatilityConfig != nil {
				if rng.Float64() < 0.5 {
					childConfig.DynamicTP.VolatilityConfig.Multiplier = parent2Config.DynamicTP.VolatilityConfig.Multiplier
				}
				if rng.Float64() < 0.5 {
					childConfig.DynamicTP.VolatilityConfig.MinTPPercent = parent2Config.DynamicTP.VolatilityConfig.MinTPPercent
				}
				if rng.Float64() < 0.5 {
					childConfig.DynamicTP.VolatilityConfig.MaxTPPercent = parent2Config.DynamicTP.VolatilityConfig.MaxTPPercent
				}
			}
		case "indicator_based":
			if childConfig.DynamicTP.IndicatorConfig != nil && parent2Config.DynamicTP.IndicatorConfig != nil {
				if rng.Float64() < 0.5 {
					childConfig.DynamicTP.IndicatorConfig.StrengthMultiplier = parent2Config.DynamicTP.IndicatorConfig.StrengthMultiplier
				}
				if rng.Float64() < 0.5 {
					childConfig.DynamicTP.IndicatorConfig.MinTPPercent = parent2Config.DynamicTP.IndicatorConfig.MinTPPercent
				}
				if rng.Float64() < 0.5 {
					childConfig.DynamicTP.IndicatorConfig.MaxTPPercent = parent2Config.DynamicTP.IndicatorConfig.MaxTPPercent
				}
			}
		}
	}
	
	// Fix parameter ordering constraints after crossover
	validateAndFixParameterOrdering(childConfig)
	
	// Synchronize ATR periods between DCA spacing and Dynamic TP
	synchronizeATRPeriodsInConfig(childConfig)
}

func MutateConfig(config, baseConfig interface{}, rng *rand.Rand) {
	dcaConfig, ok := config.(*configpkg.DCAConfig)
	if !ok {
		return
	}
	
	// Get optimization ranges
	ranges := GetDefaultOptimizationRanges()
	
	// Mutate strategy parameters (10% chance each) using predefined ranges
	// Base amount is not mutated - it's set to min order quantity
	if rng.Float64() < 0.1 {
		dcaConfig.MaxMultiplier = RandomChoice(ranges.Multipliers, rng)
	}
	// Mutate DCA spacing parameters based on strategy
	if dcaConfig.DCASpacing != nil {
		// Always mutate base_threshold for both strategies
		if rng.Float64() < 0.1 {
			dcaConfig.DCASpacing.Parameters["base_threshold"] = RandomChoice(ranges.PriceThresholds, rng)
		}
		
		// Strategy-specific parameter mutations
		switch dcaConfig.DCASpacing.Strategy {
		case "volatility_adaptive":
			if rng.Float64() < 0.1 {
				dcaConfig.DCASpacing.Parameters["volatility_sensitivity"] = RandomChoice(ranges.VolatilitySensitivity, rng)
			}
			if rng.Float64() < 0.1 {
				dcaConfig.DCASpacing.Parameters["atr_period"] = RandomChoice(ranges.ATRPeriods, rng)
			}
			if rng.Float64() < 0.1 {
				dcaConfig.DCASpacing.Parameters["level_multiplier"] = RandomChoice(ranges.LevelMultipliers, rng)
			}
		case "fixed":
			if rng.Float64() < 0.1 {
				dcaConfig.DCASpacing.Parameters["threshold_multiplier"] = RandomChoice(ranges.PriceThresholdMultipliers, rng)
			}
		}
	}
	if rng.Float64() < 0.1 {
		dcaConfig.TPPercent = RandomChoice(ranges.TPCandidates, rng)
	}
	
	// Mutate indicator parameters based on what indicators are present using predefined ranges
	indicatorSet := make(map[string]bool)
	for _, ind := range dcaConfig.Indicators {
		indicatorSet[strings.ToLower(ind)] = true
	}
	
	// Mutate classic indicator parameters if present
	if indicatorSet["rsi"] {
		if rng.Float64() < 0.1 { dcaConfig.RSIPeriod = RandomChoice(ranges.RSIPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.RSIOversold = RandomChoice(ranges.RSIOversold, rng) }
		if rng.Float64() < 0.1 { dcaConfig.RSIOverbought = 100.0 - RandomChoice(ranges.RSIOversold, rng) }
	}
	if indicatorSet["macd"] {
		if rng.Float64() < 0.1 { dcaConfig.MACDFast = RandomChoice(ranges.MACDFast, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MACDSlow = RandomChoice(ranges.MACDSlow, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MACDSignal = RandomChoice(ranges.MACDSignal, rng) }
	}
	if indicatorSet["bb"] || indicatorSet["bollinger"] {
		if rng.Float64() < 0.1 { dcaConfig.BBPeriod = RandomChoice(ranges.BBPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.BBStdDev = RandomChoice(ranges.BBStdDev, rng) }
	}
	if indicatorSet["ema"] {
		if rng.Float64() < 0.1 { dcaConfig.EMAPeriod = RandomChoice(ranges.EMAPeriods, rng) }
	}
	
	// Mutate advanced indicator parameters if present
	if indicatorSet["hullma"] || indicatorSet["hull_ma"] {
		if rng.Float64() < 0.1 { dcaConfig.HullMAPeriod = RandomChoice(ranges.HullMAPeriods, rng) }
	}
	if indicatorSet["supertrend"] || indicatorSet["st"] {
		if rng.Float64() < 0.1 { dcaConfig.SuperTrendPeriod = RandomChoice(ranges.SuperTrendPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.SuperTrendMultiplier = RandomChoice(ranges.SuperTrendMultipliers, rng) }
	}
	if indicatorSet["mfi"] {
		if rng.Float64() < 0.1 { dcaConfig.MFIPeriod = RandomChoice(ranges.MFIPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MFIOversold = RandomChoice(ranges.MFIOversold, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MFIOverbought = RandomChoice(ranges.MFIOverbought, rng) }
	}
	if indicatorSet["keltner"] || indicatorSet["kc"] {
		if rng.Float64() < 0.1 { dcaConfig.KeltnerPeriod = RandomChoice(ranges.KeltnerPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.KeltnerMultiplier = RandomChoice(ranges.KeltnerMultipliers, rng) }
	}
	if indicatorSet["wavetrend"] || indicatorSet["wt"] {
		if rng.Float64() < 0.1 { dcaConfig.WaveTrendN1 = RandomChoice(ranges.WaveTrendN1, rng) }
		if rng.Float64() < 0.1 { dcaConfig.WaveTrendN2 = RandomChoice(ranges.WaveTrendN2, rng) }
		if rng.Float64() < 0.1 { dcaConfig.WaveTrendOverbought = RandomChoice(ranges.WaveTrendOverbought, rng) }
		if rng.Float64() < 0.1 { dcaConfig.WaveTrendOversold = RandomChoice(ranges.WaveTrendOversold, rng) }
	}
	if indicatorSet["obv"] {
		if rng.Float64() < 0.1 { dcaConfig.OBVTrendThreshold = RandomChoice(ranges.OBVTrendThresholds, rng) }
	}
	if indicatorSet["stochrsi"] || indicatorSet["stochastic_rsi"] || indicatorSet["stoch_rsi"] {
		if rng.Float64() < 0.1 { dcaConfig.StochasticRSIPeriod = RandomChoice(ranges.StochasticRSIPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.StochasticRSIOverbought = RandomChoice(ranges.StochasticRSIOverboughts, rng) }
		if rng.Float64() < 0.1 { dcaConfig.StochasticRSIOversold = RandomChoice(ranges.StochasticRSIOversolds, rng) }
	}
	
	// Mutate Dynamic TP parameters if configured (10% chance each)
	if dcaConfig.DynamicTP != nil {
		// Mutate BaseTPPercent - critical for dynamic TP optimization
		if rng.Float64() < 0.1 {
			dcaConfig.DynamicTP.BaseTPPercent = RandomChoice(ranges.TPCandidates, rng)
		}
		
		switch dcaConfig.DynamicTP.Strategy {
		case "volatility_adaptive":
			if dcaConfig.DynamicTP.VolatilityConfig != nil {
				if rng.Float64() < 0.1 {
					dcaConfig.DynamicTP.VolatilityConfig.Multiplier = RandomChoice(ranges.TPVolatilityMultipliers, rng)
				}
				if rng.Float64() < 0.1 {
					dcaConfig.DynamicTP.VolatilityConfig.MinTPPercent = RandomChoice(ranges.TPMinPercents, rng)
				}
				if rng.Float64() < 0.1 {
					dcaConfig.DynamicTP.VolatilityConfig.MaxTPPercent = RandomChoice(ranges.TPMaxPercents, rng)
				}
			}
		case "indicator_based":
			if dcaConfig.DynamicTP.IndicatorConfig != nil {
				if rng.Float64() < 0.1 {
					dcaConfig.DynamicTP.IndicatorConfig.StrengthMultiplier = RandomChoice(ranges.TPStrengthMultipliers, rng)
				}
				if rng.Float64() < 0.1 {
					dcaConfig.DynamicTP.IndicatorConfig.MinTPPercent = RandomChoice(ranges.TPMinPercents, rng)
				}
				if rng.Float64() < 0.1 {
					dcaConfig.DynamicTP.IndicatorConfig.MaxTPPercent = RandomChoice(ranges.TPMaxPercents, rng)
				}
			}
		}
	}
	
	// Fix parameter ordering constraints after mutation
	validateAndFixParameterOrdering(dcaConfig)
	
	// Synchronize ATR periods between DCA spacing and Dynamic TP
	synchronizeATRPeriodsInConfig(dcaConfig)
}

// validateAndFixParameterOrdering ensures all parameter ordering constraints are satisfied
func validateAndFixParameterOrdering(dcaConfig *configpkg.DCAConfig) {
	// Fix WaveTrend N1 < N2 constraint
	if dcaConfig.WaveTrendN1 >= dcaConfig.WaveTrendN2 {
		// Swap if N1 >= N2, ensuring N1 < N2
		if dcaConfig.WaveTrendN1 == dcaConfig.WaveTrendN2 {
			// If equal, increment N2
			dcaConfig.WaveTrendN2++
		} else {
			// If N1 > N2, swap them
			dcaConfig.WaveTrendN1, dcaConfig.WaveTrendN2 = dcaConfig.WaveTrendN2, dcaConfig.WaveTrendN1
		}
	}
	
	// Fix MACD Fast < Slow constraint
	if dcaConfig.MACDFast >= dcaConfig.MACDSlow {
		// Swap if Fast >= Slow, ensuring Fast < Slow
		if dcaConfig.MACDFast == dcaConfig.MACDSlow {
			// If equal, increment Slow
			dcaConfig.MACDSlow++
		} else {
			// If Fast > Slow, swap them
			dcaConfig.MACDFast, dcaConfig.MACDSlow = dcaConfig.MACDSlow, dcaConfig.MACDFast
		}
	}
	
	// Fix RSI Oversold < Overbought constraint
	if dcaConfig.RSIOversold >= dcaConfig.RSIOverbought {
		// Swap if Oversold >= Overbought
		if dcaConfig.RSIOversold == dcaConfig.RSIOverbought {
			// If equal, adjust to maintain spread
			dcaConfig.RSIOversold = dcaConfig.RSIOverbought - 10
			if dcaConfig.RSIOversold < 10 {
				dcaConfig.RSIOversold = 10
				dcaConfig.RSIOverbought = 90
			}
		} else {
			dcaConfig.RSIOversold, dcaConfig.RSIOverbought = dcaConfig.RSIOverbought, dcaConfig.RSIOversold
		}
	}
	
	// Fix MFI Oversold < Overbought constraint
	if dcaConfig.MFIOversold >= dcaConfig.MFIOverbought {
		// Swap if Oversold >= Overbought
		if dcaConfig.MFIOversold == dcaConfig.MFIOverbought {
			// If equal, adjust to maintain spread
			dcaConfig.MFIOversold = dcaConfig.MFIOverbought - 10
			if dcaConfig.MFIOversold < 10 {
				dcaConfig.MFIOversold = 10
				dcaConfig.MFIOverbought = 90
			}
		} else {
			dcaConfig.MFIOversold, dcaConfig.MFIOverbought = dcaConfig.MFIOverbought, dcaConfig.MFIOversold
		}
	}
	
	// Fix StochasticRSI Oversold < Overbought constraint
	if dcaConfig.StochasticRSIOversold >= dcaConfig.StochasticRSIOverbought {
		// Swap if Oversold >= Overbought
		if dcaConfig.StochasticRSIOversold == dcaConfig.StochasticRSIOverbought {
			// If equal, adjust to maintain spread
			dcaConfig.StochasticRSIOversold = dcaConfig.StochasticRSIOverbought - 10
			if dcaConfig.StochasticRSIOversold < 10 {
				dcaConfig.StochasticRSIOversold = 10
				dcaConfig.StochasticRSIOverbought = 90
			}
		} else {
			dcaConfig.StochasticRSIOversold, dcaConfig.StochasticRSIOverbought = dcaConfig.StochasticRSIOverbought, dcaConfig.StochasticRSIOversold
		}
	}
	
	// Fix Dynamic TP Min < Max constraints
	if dcaConfig.DynamicTP != nil {
		switch dcaConfig.DynamicTP.Strategy {
		case "volatility_adaptive":
			if dcaConfig.DynamicTP.VolatilityConfig != nil {
				if dcaConfig.DynamicTP.VolatilityConfig.MinTPPercent >= dcaConfig.DynamicTP.VolatilityConfig.MaxTPPercent {
					// Swap if Min >= Max
					if dcaConfig.DynamicTP.VolatilityConfig.MinTPPercent == dcaConfig.DynamicTP.VolatilityConfig.MaxTPPercent {
						// If equal, adjust Max to maintain spread
						dcaConfig.DynamicTP.VolatilityConfig.MaxTPPercent = dcaConfig.DynamicTP.VolatilityConfig.MinTPPercent + 0.01
					} else {
						dcaConfig.DynamicTP.VolatilityConfig.MinTPPercent, dcaConfig.DynamicTP.VolatilityConfig.MaxTPPercent = 
							dcaConfig.DynamicTP.VolatilityConfig.MaxTPPercent, dcaConfig.DynamicTP.VolatilityConfig.MinTPPercent
					}
				}
			}
		case "indicator_based":
			if dcaConfig.DynamicTP.IndicatorConfig != nil {
				if dcaConfig.DynamicTP.IndicatorConfig.MinTPPercent >= dcaConfig.DynamicTP.IndicatorConfig.MaxTPPercent {
					// Swap if Min >= Max
					if dcaConfig.DynamicTP.IndicatorConfig.MinTPPercent == dcaConfig.DynamicTP.IndicatorConfig.MaxTPPercent {
						// If equal, adjust Max to maintain spread
						dcaConfig.DynamicTP.IndicatorConfig.MaxTPPercent = dcaConfig.DynamicTP.IndicatorConfig.MinTPPercent + 0.01
					} else {
						dcaConfig.DynamicTP.IndicatorConfig.MinTPPercent, dcaConfig.DynamicTP.IndicatorConfig.MaxTPPercent = 
							dcaConfig.DynamicTP.IndicatorConfig.MaxTPPercent, dcaConfig.DynamicTP.IndicatorConfig.MinTPPercent
					}
				}
			}
		}
	}
}

// synchronizeATRPeriodsInConfig ensures DCA spacing and Dynamic TP use the same ATR period in the configuration
// Priority: DCA spacing atr_period > Dynamic TP ATRPeriod > Default 14
func synchronizeATRPeriodsInConfig(dcaConfig *configpkg.DCAConfig) {
	var targetATRPeriod int = 14 // Default fallback
	
	// Priority 1: Get ATR period from DCA spacing configuration
	if dcaConfig.DCASpacing != nil && dcaConfig.DCASpacing.Parameters != nil {
		if atrPeriod, ok := dcaConfig.DCASpacing.Parameters["atr_period"].(int); ok && atrPeriod > 0 {
			targetATRPeriod = atrPeriod
		}
	}
	
	// Priority 2: If no DCA spacing ATR period, use Dynamic TP ATR period
	if targetATRPeriod == 14 && dcaConfig.DynamicTP != nil && dcaConfig.DynamicTP.VolatilityConfig != nil {
		if dcaConfig.DynamicTP.VolatilityConfig.ATRPeriod > 0 {
			targetATRPeriod = dcaConfig.DynamicTP.VolatilityConfig.ATRPeriod
		}
	}
	
	// Synchronize both configurations to use the same ATR period
	if dcaConfig.DCASpacing != nil && dcaConfig.DCASpacing.Parameters != nil {
		dcaConfig.DCASpacing.Parameters["atr_period"] = targetATRPeriod
	}
	
	if dcaConfig.DynamicTP != nil && dcaConfig.DynamicTP.VolatilityConfig != nil {
		dcaConfig.DynamicTP.VolatilityConfig.ATRPeriod = targetATRPeriod
	}
}

func getConfigField(config interface{}, field string) float64 {
	// Cast to DCAConfig and extract the field
	dcaConfig, ok := config.(*configpkg.DCAConfig)
	if !ok {
		return 0.0
	}
	
	switch field {
	case "MaxMultiplier":
		return dcaConfig.MaxMultiplier
	case "TPPercent":
		return dcaConfig.TPPercent
	case "BaseThreshold":
		if dcaConfig.DCASpacing != nil {
			if val, ok := dcaConfig.DCASpacing.Parameters["base_threshold"].(float64); ok {
				return val
			}
		}
		return 0.01 // Default fallback
	case "ThresholdMultiplier":
		if dcaConfig.DCASpacing != nil {
			if val, ok := dcaConfig.DCASpacing.Parameters["threshold_multiplier"].(float64); ok {
				return val
			}
		}
		return 1.15 // Default fallback
	case "VolatilitySensitivity":
		if dcaConfig.DCASpacing != nil {
			if val, ok := dcaConfig.DCASpacing.Parameters["volatility_sensitivity"].(float64); ok {
				return val
			}
		}
		return 1.8 // Default fallback
	case "ATRPeriod":
		if dcaConfig.DCASpacing != nil {
			if val, ok := dcaConfig.DCASpacing.Parameters["atr_period"].(int); ok {
				return float64(val)
			}
		}
		return 14.0 // Default fallback
	case "LevelMultiplier":
		if dcaConfig.DCASpacing != nil {
			if val, ok := dcaConfig.DCASpacing.Parameters["level_multiplier"].(float64); ok {
				return val
			}
		}
		return 1.1 // Default fallback
	default:
		return 0.0
	}
}

// runDCABacktestWithData runs a backtest with DCAConfig and data (DEPRECATED - use RunBacktestWithData instead)
func runDCABacktestWithData(cfg *configpkg.DCAConfig, data []types.OHLCV) *backtest.BacktestResults {
	// Use the consistent strategy creation function
	return RunBacktestWithData(cfg, data)
}

// createDCAStrategyFromConfig creates a strategy from DCAConfig using the SAME logic as backtest runner
// This ensures 100% consistency between optimization and regular backtest runs
func createDCAStrategyFromConfig(cfg *configpkg.DCAConfig) (strategy.Strategy, error) {
	// Initialize Enhanced DCA strategy with base trading amount
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)
	dca.SetMaxMultiplier(cfg.MaxMultiplier)

	// Configure DCA spacing strategy (required for all configurations)
	if cfg.DCASpacing == nil {
		return nil, fmt.Errorf("no DCA spacing strategy configured - please specify dca_spacing in your configuration")
	}

	spacingConfig := spacing.SpacingConfig{
		Strategy:   cfg.DCASpacing.Strategy,
		Parameters: cfg.DCASpacing.Parameters,
	}
	
	spacingStrategy, err := spacing.CreateSpacingStrategy(spacingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create spacing strategy: %w", err)
	}
	
	if err := spacingStrategy.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid spacing strategy configuration: %w", err)
	}
	
	dca.SetSpacingStrategy(spacingStrategy)

	// Configure dynamic take profit if specified
	if cfg.DynamicTP != nil {
		dca.SetDynamicTPConfig(cfg.DynamicTP)
	}

	// Indicator inclusion map
	include := make(map[string]bool)
	for _, name := range cfg.Indicators {
		include[strings.ToLower(strings.TrimSpace(name))] = true
	}

	// Instantiate indicators based on what's actually in the indicators list
	if include["rsi"] {
		rsi := oscillators.NewRSI(cfg.RSIPeriod)
		rsi.SetOversold(cfg.RSIOversold)
		rsi.SetOverbought(cfg.RSIOverbought)
		dca.AddIndicator(rsi)
	}
	if include["macd"] {
		macd := oscillators.NewMACD(cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
		dca.AddIndicator(macd)
	}
	if include["bb"] || include["bollinger"] {
		bb := bands.NewBollingerBandsEMA(cfg.BBPeriod, cfg.BBStdDev)
		dca.AddIndicator(bb)
	}
	if include["ema"] {
		ema := common.NewEMA(cfg.EMAPeriod)
		dca.AddIndicator(ema)
	}
	if include["hullma"] || include["hull_ma"] {
		hullMA := trend.NewHullMA(cfg.HullMAPeriod)
		dca.AddIndicator(hullMA)
	}
	if include["supertrend"] || include["st"] {
		supertrend := trend.NewSuperTrendWithParams(cfg.SuperTrendPeriod, cfg.SuperTrendMultiplier)
		dca.AddIndicator(supertrend)
	}
	if include["mfi"] {
		mfi := oscillators.NewMFIWithPeriod(cfg.MFIPeriod)
		mfi.SetOversold(cfg.MFIOversold)
		mfi.SetOverbought(cfg.MFIOverbought)
		dca.AddIndicator(mfi)
	}
	if include["keltner"] || include["kc"] {
		keltner := bands.NewKeltnerChannelsCustom(cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
		dca.AddIndicator(keltner)
	}
	if include["wavetrend"] || include["wt"] {
		wavetrend := oscillators.NewWaveTrendCustom(cfg.WaveTrendN1, cfg.WaveTrendN2)
		wavetrend.SetOverbought(cfg.WaveTrendOverbought)
		wavetrend.SetOversold(cfg.WaveTrendOversold)
		dca.AddIndicator(wavetrend)
	}
	if include["obv"] {
		obv := volume.NewOBV()
		dca.AddIndicator(obv)
	}
	if include["stochrsi"] || include["stochastic_rsi"] || include["stoch_rsi"] {
		stochRSI := oscillators.NewStochasticRSI()
		dca.AddIndicator(stochRSI)
	}

	return dca, nil
}

// getMinTPPercent extracts minimum TP percentage from dynamic TP config
func getMinTPPercent(cfg *configpkg.DynamicTPConfig) float64 {
	if cfg.VolatilityConfig != nil {
		return cfg.VolatilityConfig.MinTPPercent
	}
	if cfg.IndicatorConfig != nil {
		return cfg.IndicatorConfig.MinTPPercent
	}
	return cfg.BaseTPPercent // Fallback
}

// getMaxTPPercent extracts maximum TP percentage from dynamic TP config
func getMaxTPPercent(cfg *configpkg.DynamicTPConfig) float64 {
	if cfg.VolatilityConfig != nil {
		return cfg.VolatilityConfig.MaxTPPercent
	}
	if cfg.IndicatorConfig != nil {
		return cfg.IndicatorConfig.MaxTPPercent
	}
	return cfg.BaseTPPercent // Fallback
}

// createDCAStrategy creates a strategy from DCAConfig (DEPRECATED - use createDCAStrategyFromConfig instead)
func createDCAStrategy(cfg *configpkg.DCAConfig) strategy.Strategy {
	// Use optimized Enhanced DCA strategy
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)
	
	// Set maximum position multiplier from configuration
	dca.SetMaxMultiplier(cfg.MaxMultiplier)
	
	// Configure spacing strategy - required for consistency
	if cfg.DCASpacing == nil {
		// This should not happen in GA optimization, but handle gracefully
		return dca
	}
	
	spacingConfig := spacing.SpacingConfig{
		Strategy:   cfg.DCASpacing.Strategy,
		Parameters: cfg.DCASpacing.Parameters,
	}
	
	spacingStrategy, err := spacing.CreateSpacingStrategy(spacingConfig)
	if err != nil {
		// Log error but continue with default fixed spacing for GA robustness
		log.Printf("⚠️ GA: Failed to create spacing strategy: %v, using default fixed spacing", err)
		defaultConfig := spacing.SpacingConfig{
			Strategy: "fixed",
			Parameters: map[string]interface{}{
				"base_threshold":       0.01,  // 1%
				"threshold_multiplier": 1.15,  // 1.15x
			},
		}
		spacingStrategy, _ = spacing.CreateSpacingStrategy(defaultConfig)
	}
	
	if err := spacingStrategy.ValidateConfig(); err != nil {
		// Log error but continue with default fixed spacing for GA robustness
		log.Printf("⚠️ GA: Spacing strategy validation failed: %v, using default fixed spacing", err)
		defaultConfig := spacing.SpacingConfig{
			Strategy: "fixed",
			Parameters: map[string]interface{}{
				"base_threshold":       0.01,  // 1%
				"threshold_multiplier": 1.15,  // 1.15x
			},
		}
		spacingStrategy, _ = spacing.CreateSpacingStrategy(defaultConfig)
	}
	
	dca.SetSpacingStrategy(spacingStrategy)
	
	// Create a set for efficient lookup
	include := make(map[string]bool)
	for _, name := range cfg.Indicators {
		include[strings.ToLower(strings.TrimSpace(name))] = true
	}
	
	// Add indicators in EXACT same order as orchestrator for deterministic results
	if include["rsi"] {
		rsi := oscillators.NewRSI(cfg.RSIPeriod)
		rsi.SetOversold(cfg.RSIOversold)
		rsi.SetOverbought(cfg.RSIOverbought)
		dca.AddIndicator(rsi)
	}
	if include["macd"] {
		macd := oscillators.NewMACD(cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
		dca.AddIndicator(macd)
	}
	if include["bb"] || include["bollinger"] {
		bb := bands.NewBollingerBandsEMA(cfg.BBPeriod, cfg.BBStdDev)
		dca.AddIndicator(bb)
	}
	if include["ema"] {
		ema := common.NewEMA(cfg.EMAPeriod)
		dca.AddIndicator(ema)
	}
	if include["hullma"] || include["hull_ma"] {
		hullMA := trend.NewHullMA(cfg.HullMAPeriod)
		dca.AddIndicator(hullMA)
	}
	if include["supertrend"] || include["st"] {
		supertrend := trend.NewSuperTrendWithParams(cfg.SuperTrendPeriod, cfg.SuperTrendMultiplier)
		dca.AddIndicator(supertrend)
	}
	if include["mfi"] {
		mfi := oscillators.NewMFIWithPeriod(cfg.MFIPeriod)
		mfi.SetOversold(cfg.MFIOversold)
		mfi.SetOverbought(cfg.MFIOverbought)
		dca.AddIndicator(mfi)
	}
	if include["keltner"] || include["kc"] {
		keltner := bands.NewKeltnerChannelsCustom(cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
		dca.AddIndicator(keltner)
	}
	if include["wavetrend"] || include["wt"] {
		wavetrend := oscillators.NewWaveTrendCustom(cfg.WaveTrendN1, cfg.WaveTrendN2)
		wavetrend.SetOverbought(cfg.WaveTrendOverbought)
		wavetrend.SetOversold(cfg.WaveTrendOversold)
		dca.AddIndicator(wavetrend)
	}
	if include["obv"] {
		obv := volume.NewOBV()
		dca.AddIndicator(obv)
	}
	if include["stochrsi"] || include["stochastic_rsi"] || include["stoch_rsi"] {
		stochRSI := oscillators.NewStochasticRSI()
		dca.AddIndicator(stochRSI)
	}
	
	// CRITICAL FIX: Configure dynamic TP if present in the config
	if cfg.DynamicTP != nil {
		dca.SetDynamicTPConfig(cfg.DynamicTP)
	}
	
	return dca
}

// RandomChoice selects a random element from a slice - extracted from main.go
func RandomChoice[T any](choices []T, rng *rand.Rand) T {
	if len(choices) == 0 {
		var zero T
		return zero
	}
	idx := rng.Intn(len(choices))
	return choices[idx]
}

// ContainsIndicator checks if an indicator is in the list - extracted from main.go
func ContainsIndicator(indicators []string, name string) bool {
	name = strings.ToLower(name)
	for _, n := range indicators {
		if strings.ToLower(n) == name {
			return true
		}
	}
	return false
}


