package optimization

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	configpkg "github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	datamanager "github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// GA Constants
const (
	GAPopulationSize    = 60   // Population size for optimization
	GAGenerations       = 35   // Number of generations
	GAMutationRate      = 0.1  // Mutation rate
	GACrossoverRate     = 0.8  // Crossover rate
	GAEliteSize         = 6    // Elite size
	TournamentSize      = 3    // Tournament selection size
	MaxParallelWorkers  = 4    // Maximum concurrent GA evaluations
	ProgressReportInterval = 5 // Report progress every N generations
	DetailReportInterval   = 10 // Show detailed config every N generations
)

// Individual represents a candidate solution - extracted from main.go
type GAIndividual struct {
	Config  interface{} // BacktestConfig in practice
	Fitness float64
	Results *backtest.BacktestResults
}

// OptimizeWithGA runs genetic algorithm optimization - extracted from main.go optimizeForInterval
func OptimizeWithGA(baseConfig interface{}, dataFile string, selectedPeriod time.Duration) (*backtest.BacktestResults, interface{}, error) {
	// Create local RNG to avoid race conditions
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	
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

	log.Printf("🔄 Starting Genetic Algorithm Optimization")
	log.Printf("Mode: Full optimization (including indicator parameters)")
	log.Printf("Population: %d, Generations: %d, Mutation: %.1f%%, Crossover: %.1f%%", 
		populationSize, generations, mutationRate*100, crossoverRate*100)
	log.Printf("Using %d parallel workers for fitness evaluation", MaxParallelWorkers)

	// Initialize population
	population := InitializePopulation(baseConfig, populationSize, rng)
	
	var bestIndividual *GAIndividual
	var bestResults *backtest.BacktestResults
	
	for gen := 0; gen < generations; gen++ {
		// Evaluate fitness for all individuals in parallel
		EvaluatePopulationParallel(population, data)
		
		// Sort by fitness (descending)
		SortPopulationByFitness(population)
		
		// Track best individual
		if bestIndividual == nil || population[0].Fitness > bestIndividual.Fitness {
			bestIndividual = &GAIndividual{
				Config:  population[0].Config,
				Fitness: population[0].Fitness,
				Results: population[0].Results,
			}
			bestResults = population[0].Results
		}
		
		if gen%ProgressReportInterval == 0 {
			log.Printf("🔄 Gen %d: Best=%.2f%%, Avg=%.2f%%, Worst=%.2f%%", 
				gen+1, 
				population[0].Fitness*100,
				AverageFitness(population)*100,
				population[len(population)-1].Fitness*100)
				
			// Show best individual details every DetailReportInterval generations
			if gen%DetailReportInterval == (DetailReportInterval-1) {
				log.Printf("ℹ️ Best Config: maxMult=%.1f | tp=%.1f%% | threshold=%.1f%%",
					getConfigField(population[0].Config, "MaxMultiplier"),
					getConfigField(population[0].Config, "TPPercent")*100,
					getConfigField(population[0].Config, "PriceThreshold")*100)
			}
		}
		
		// Create next generation
		if gen < generations-1 {
			population = CreateNextGeneration(population, eliteSize, crossoverRate, mutationRate, baseConfig, rng)
		}
	}
	
	log.Printf("✅ GA Optimization completed! Best fitness: %.2f%%", bestIndividual.Fitness*100)
	return bestResults, bestIndividual.Config, nil
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

// SortPopulationByFitness sorts population by fitness (descending) - extracted from main.go
func SortPopulationByFitness(population []*GAIndividual) {
	for i := 0; i < len(population)-1; i++ {
		for j := i + 1; j < len(population); j++ {
			if population[j].Fitness > population[i].Fitness {
				population[i], population[j] = population[j], population[i]
			}
		}
	}
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
	dcaConfig.PriceThreshold = RandomChoice(ranges.PriceThresholds, rng)
	dcaConfig.TPPercent = RandomChoice(ranges.TPCandidates, rng)
	
	// Set indicators based on combo type
	if dcaConfig.UseAdvancedCombo {
		dcaConfig.Indicators = []string{"hull_ma", "mfi", "keltner", "wavetrend"}
		
		// Randomize advanced combo parameters using predefined ranges
		dcaConfig.HullMAPeriod = RandomChoice(ranges.HullMAPeriods, rng)
		dcaConfig.MFIPeriod = RandomChoice(ranges.MFIPeriods, rng)
		dcaConfig.MFIOversold = RandomChoice(ranges.MFIOversold, rng)
		dcaConfig.MFIOverbought = RandomChoice(ranges.MFIOverbought, rng)
		dcaConfig.KeltnerPeriod = RandomChoice(ranges.KeltnerPeriods, rng)
		dcaConfig.KeltnerMultiplier = RandomChoice(ranges.KeltnerMultipliers, rng)
		dcaConfig.WaveTrendN1 = RandomChoice(ranges.WaveTrendN1, rng)
		dcaConfig.WaveTrendN2 = RandomChoice(ranges.WaveTrendN2, rng)
		dcaConfig.WaveTrendOverbought = RandomChoice(ranges.WaveTrendOverbought, rng)
		dcaConfig.WaveTrendOversold = RandomChoice(ranges.WaveTrendOversold, rng)
	} else {
		dcaConfig.Indicators = []string{"rsi", "macd", "bb", "ema"}
		
		// Randomize classic combo parameters using predefined ranges
		dcaConfig.RSIPeriod = RandomChoice(ranges.RSIPeriods, rng)
		dcaConfig.RSIOversold = RandomChoice(ranges.RSIOversold, rng)
		dcaConfig.RSIOverbought = 100.0 - RandomChoice(ranges.RSIOversold, rng)  // Complement for overbought
		dcaConfig.MACDFast = RandomChoice(ranges.MACDFast, rng)
		dcaConfig.MACDSlow = RandomChoice(ranges.MACDSlow, rng)
		dcaConfig.MACDSignal = RandomChoice(ranges.MACDSignal, rng)
		dcaConfig.BBPeriod = RandomChoice(ranges.BBPeriods, rng)
		dcaConfig.BBStdDev = RandomChoice(ranges.BBStdDev, rng)
		dcaConfig.EMAPeriod = RandomChoice(ranges.EMAPeriods, rng)
	}
}

func RunBacktestWithData(config interface{}, data []types.OHLCV) *backtest.BacktestResults {
	// Convert interface{} to config.DCAConfig which implements BacktestConfig interface
	dcaConfig, ok := config.(*configpkg.DCAConfig)
	if !ok {
		// Fallback - this shouldn't happen in normal operation
		return &backtest.BacktestResults{TotalReturn: 0.0}
	}
	
	// Run backtest using the real strategy and backtest engine
	// This is similar to runBacktestWithData in main.go but adapted for DCAConfig
	return runDCABacktestWithData(dcaConfig, data)
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
	if rng.Float64() < 0.5 {
		childConfig.PriceThreshold = parent2Config.PriceThreshold
	}
	if rng.Float64() < 0.5 {
		childConfig.TPPercent = parent2Config.TPPercent
	}
	
	if childConfig.UseAdvancedCombo {
		// Crossover advanced parameters from parent2
		if rng.Float64() < 0.5 { childConfig.HullMAPeriod = parent2Config.HullMAPeriod }
		if rng.Float64() < 0.5 { childConfig.MFIPeriod = parent2Config.MFIPeriod }
		if rng.Float64() < 0.5 { childConfig.MFIOversold = parent2Config.MFIOversold }
		if rng.Float64() < 0.5 { childConfig.MFIOverbought = parent2Config.MFIOverbought }
		if rng.Float64() < 0.5 { childConfig.KeltnerPeriod = parent2Config.KeltnerPeriod }
		if rng.Float64() < 0.5 { childConfig.KeltnerMultiplier = parent2Config.KeltnerMultiplier }
		if rng.Float64() < 0.5 { childConfig.WaveTrendN1 = parent2Config.WaveTrendN1 }
		if rng.Float64() < 0.5 { childConfig.WaveTrendN2 = parent2Config.WaveTrendN2 }
		if rng.Float64() < 0.5 { childConfig.WaveTrendOverbought = parent2Config.WaveTrendOverbought }
		if rng.Float64() < 0.5 { childConfig.WaveTrendOversold = parent2Config.WaveTrendOversold }
	} else {
		// Crossover classic parameters from parent2
		if rng.Float64() < 0.5 { childConfig.RSIPeriod = parent2Config.RSIPeriod }
		if rng.Float64() < 0.5 { childConfig.RSIOversold = parent2Config.RSIOversold }
		if rng.Float64() < 0.5 { childConfig.RSIOverbought = parent2Config.RSIOverbought }
		if rng.Float64() < 0.5 { childConfig.MACDFast = parent2Config.MACDFast }
		if rng.Float64() < 0.5 { childConfig.MACDSlow = parent2Config.MACDSlow }
		if rng.Float64() < 0.5 { childConfig.MACDSignal = parent2Config.MACDSignal }
		if rng.Float64() < 0.5 { childConfig.BBPeriod = parent2Config.BBPeriod }
		if rng.Float64() < 0.5 { childConfig.BBStdDev = parent2Config.BBStdDev }
		if rng.Float64() < 0.5 { childConfig.EMAPeriod = parent2Config.EMAPeriod }
	}
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
	if rng.Float64() < 0.1 {
		dcaConfig.PriceThreshold = RandomChoice(ranges.PriceThresholds, rng)
	}
	if rng.Float64() < 0.1 {
		dcaConfig.TPPercent = RandomChoice(ranges.TPCandidates, rng)
	}
	
	// Mutate indicator parameters based on current combo type using predefined ranges
	if dcaConfig.UseAdvancedCombo {
		if rng.Float64() < 0.1 { dcaConfig.HullMAPeriod = RandomChoice(ranges.HullMAPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MFIPeriod = RandomChoice(ranges.MFIPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MFIOversold = RandomChoice(ranges.MFIOversold, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MFIOverbought = RandomChoice(ranges.MFIOverbought, rng) }
		if rng.Float64() < 0.1 { dcaConfig.KeltnerPeriod = RandomChoice(ranges.KeltnerPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.KeltnerMultiplier = RandomChoice(ranges.KeltnerMultipliers, rng) }
		if rng.Float64() < 0.1 { dcaConfig.WaveTrendN1 = RandomChoice(ranges.WaveTrendN1, rng) }
		if rng.Float64() < 0.1 { dcaConfig.WaveTrendN2 = RandomChoice(ranges.WaveTrendN2, rng) }
		if rng.Float64() < 0.1 { dcaConfig.WaveTrendOverbought = RandomChoice(ranges.WaveTrendOverbought, rng) }
		if rng.Float64() < 0.1 { dcaConfig.WaveTrendOversold = RandomChoice(ranges.WaveTrendOversold, rng) }
	} else {
		if rng.Float64() < 0.1 { dcaConfig.RSIPeriod = RandomChoice(ranges.RSIPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.RSIOversold = RandomChoice(ranges.RSIOversold, rng) }
		if rng.Float64() < 0.1 { dcaConfig.RSIOverbought = 100.0 - RandomChoice(ranges.RSIOversold, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MACDFast = RandomChoice(ranges.MACDFast, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MACDSlow = RandomChoice(ranges.MACDSlow, rng) }
		if rng.Float64() < 0.1 { dcaConfig.MACDSignal = RandomChoice(ranges.MACDSignal, rng) }
		if rng.Float64() < 0.1 { dcaConfig.BBPeriod = RandomChoice(ranges.BBPeriods, rng) }
		if rng.Float64() < 0.1 { dcaConfig.BBStdDev = RandomChoice(ranges.BBStdDev, rng) }
		if rng.Float64() < 0.1 { dcaConfig.EMAPeriod = RandomChoice(ranges.EMAPeriods, rng) }
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
	case "PriceThreshold":
		return dcaConfig.PriceThreshold
	default:
		return 0.0
	}
}

// runDCABacktestWithData runs a backtest with DCAConfig and data
func runDCABacktestWithData(cfg *configpkg.DCAConfig, data []types.OHLCV) *backtest.BacktestResults {
	start := time.Now()
	
	// Create strategy with configured indicators
	strat := createDCAStrategy(cfg)
	
	// Create and run backtest engine
	tp := cfg.TPPercent
	if !cfg.Cycle { 
		tp = 0 
	}
	engine := backtest.NewBacktestEngine(cfg.InitialBalance, cfg.Commission, strat, tp, cfg.MinOrderQty, cfg.UseTPLevels)
	results := engine.Run(data, cfg.WindowSize)
	
	// Update all metrics
	results.UpdateMetrics()
	
	_ = time.Since(start) // Suppress unused variable warning
	
	return results
}

// createDCAStrategy creates a strategy from DCAConfig (adapted from main.go)
func createDCAStrategy(cfg *configpkg.DCAConfig) strategy.Strategy {
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)
	dca.SetPriceThreshold(cfg.PriceThreshold)
	
	// Create a set for efficient lookup
	include := make(map[string]bool)
	for _, name := range cfg.Indicators {
		include[strings.ToLower(name)] = true
	}
	
	if cfg.UseAdvancedCombo {
		// Advanced combo indicators
		if include["hull_ma"] {
			hullMA := indicators.NewHullMA(cfg.HullMAPeriod)
			dca.AddIndicator(hullMA)
		}
		if include["mfi"] {
			mfi := indicators.NewMFIWithPeriod(cfg.MFIPeriod)
			mfi.SetOversold(cfg.MFIOversold)
			mfi.SetOverbought(cfg.MFIOverbought)
			dca.AddIndicator(mfi)
		}
		if include["keltner"] {
			keltner := indicators.NewKeltnerChannelsCustom(cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
			dca.AddIndicator(keltner)
		}
		if include["wavetrend"] {
			wavetrend := indicators.NewWaveTrendCustom(cfg.WaveTrendN1, cfg.WaveTrendN2)
			wavetrend.SetOverbought(cfg.WaveTrendOverbought)
			wavetrend.SetOversold(cfg.WaveTrendOversold)
			dca.AddIndicator(wavetrend)
		}
	} else {
		// Classic combo indicators
		if include["rsi"] {
			rsi := indicators.NewRSI(cfg.RSIPeriod)
			rsi.SetOversold(cfg.RSIOversold)
			rsi.SetOverbought(cfg.RSIOverbought)
			dca.AddIndicator(rsi)
		}
		if include["macd"] {
			macd := indicators.NewMACD(cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
			dca.AddIndicator(macd)
		}
		if include["bb"] {
			bb := indicators.NewBollingerBandsEMA(cfg.BBPeriod, cfg.BBStdDev)
			dca.AddIndicator(bb)
		}
		if include["ema"] {
			ema := indicators.NewEMA(cfg.EMAPeriod)
			dca.AddIndicator(ema)
		}
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
