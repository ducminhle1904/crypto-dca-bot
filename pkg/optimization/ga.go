package optimization

import (
	"log"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
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

// Placeholder functions that need to be implemented based on actual config types
func copyConfig(config interface{}) interface{} {
	// This needs to be implemented based on the actual config type
	return config
}

func RandomizeConfig(config interface{}, rng *rand.Rand) {
	// This needs to be implemented based on the actual config type
}

func RunBacktestWithData(config interface{}, data []types.OHLCV) *backtest.BacktestResults {
	// This needs to be implemented to run actual backtest
	return &backtest.BacktestResults{TotalReturn: 0.5} // Placeholder
}

func CrossoverConfigs(child, parent1, parent2 interface{}, rng *rand.Rand) {
	// This needs to be implemented based on the actual config type
}

func MutateConfig(config, baseConfig interface{}, rng *rand.Rand) {
	// This needs to be implemented based on the actual config type
}

func getConfigField(config interface{}, field string) float64 {
	// This needs to be implemented based on the actual config type
	return 0.0
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
