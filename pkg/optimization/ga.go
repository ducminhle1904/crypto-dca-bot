package optimization

import (
	"log"
	"math/rand"
	"sort"
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

// GA Constants - Optimized for Walk-Forward Validation (180-day windows)
const (
	GAPopulationSize    = 24   // Smaller population for faster convergence (was 45)
	GAGenerations       = 15   // Fewer generations - still effective for 180d data (was 30)
	GAMutationRate      = 0.2 // Higher mutation for faster exploration (was 0.1)
	GACrossoverRate     = 0.85 // Higher crossover for better mixing (was 0.8)
	GAEliteSize         = 4    // Keep best 4 individuals (~17% of population)
	TournamentSize      = 2    // Smaller tournament for speed (was 3)
	MaxParallelWorkers  = 6    // More parallel workers for speed (was 4)
	ProgressReportInterval = 3 // More frequent progress reports (was 5)
	DetailReportInterval   = 8 // Less frequent detailed reports (was 10)
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
		
		// Silent generation processing
		
		// Create next generation
		if gen < generations-1 {
			population = CreateNextGeneration(population, eliteSize, crossoverRate, mutationRate, baseConfig, rng)
		}
	}
	
	log.Printf("✅ GA → %.2f%%", bestIndividual.Fitness*100)
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
	dcaConfig.PriceThresholdMultiplier = RandomChoice(ranges.PriceThresholdMultipliers, rng)
	dcaConfig.TPPercent = RandomChoice(ranges.TPCandidates, rng)
	
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
		dcaConfig.HullMAPeriod = RandomChoice(ranges.SuperTrendPeriods, rng)
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
		dcaConfig.PriceThresholdMultiplier = RandomChoice(ranges.PriceThresholdMultipliers, rng)
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
		if rng.Float64() < 0.1 { dcaConfig.HullMAPeriod = RandomChoice(ranges.SuperTrendPeriods, rng) }
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

// createDCAStrategy creates a strategy from DCAConfig (optimized version)
func createDCAStrategy(cfg *configpkg.DCAConfig) strategy.Strategy {
	// Use optimized Enhanced DCA strategy
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)
	dca.SetPriceThreshold(cfg.PriceThreshold)
	
	// Set maximum position multiplier from configuration
	dca.SetMaxMultiplier(cfg.MaxMultiplier)
	
	// Create a set for efficient lookup
	include := make(map[string]bool)
	for _, name := range cfg.Indicators {
		include[strings.ToLower(name)] = true
	}
	
	// Create strategy based on which indicators are present in config
	indicatorSet := make(map[string]bool)
	for _, ind := range cfg.Indicators {
		indicatorSet[strings.ToLower(ind)] = true
	}
	
	// Add indicators based on what's present in the config
	if include["supertrend"] || include["st"] {
		supertrend := indicators.NewSuperTrendWithParams(cfg.SuperTrendPeriod, cfg.SuperTrendMultiplier)
		dca.AddIndicator(supertrend)
	}
	if include["mfi"] {
		mfi := indicators.NewMFIWithPeriod(cfg.MFIPeriod)
		mfi.SetOversold(cfg.MFIOversold)
		mfi.SetOverbought(cfg.MFIOverbought)
		dca.AddIndicator(mfi)
	}
	if include["keltner"] || include["kc"] {
		keltner := indicators.NewKeltnerChannelsCustom(cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
		dca.AddIndicator(keltner)
	}
	if include["wavetrend"] || include["wt"] {
		wavetrend := indicators.NewWaveTrendCustom(cfg.WaveTrendN1, cfg.WaveTrendN2)
		wavetrend.SetOverbought(cfg.WaveTrendOverbought)
		wavetrend.SetOversold(cfg.WaveTrendOversold)
		dca.AddIndicator(wavetrend)
	}
	
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
	if include["bb"] || include["bollinger"] {
		bb := indicators.NewBollingerBandsEMA(cfg.BBPeriod, cfg.BBStdDev)
		dca.AddIndicator(bb)
	}
	if include["ema"] {
		ema := indicators.NewEMA(cfg.EMAPeriod)
		dca.AddIndicator(ema)
	}
	
	if include["hullma"] || include["hull_ma"] {
		hullMA := indicators.NewHullMA(cfg.HullMAPeriod)
		dca.AddIndicator(hullMA)
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


