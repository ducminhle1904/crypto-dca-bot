package optimization

import (
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// BacktestConfig represents the configuration for DCA backtesting
// This is a temporary interface until we can refactor to use the config package
type BacktestConfig interface {
	GetDataFile() string
	GetCycle() bool
	GetTPPercent() float64
	GetUseAdvancedCombo() bool
	GetIndicators() []string
	SetIndicators(indicators []string)
	
	// Mutation methods
	SetMaxMultiplier(val float64)
	SetTPPercent(val float64)
	SetPriceThreshold(val float64)
	SetPriceThresholdMultiplier(val float64) // Progressive DCA spacing multiplier
	SetHullMAPeriod(val int)
	SetMFIPeriod(val int)
	SetMFIOversold(val float64)
	SetMFIOverbought(val float64)
	SetKeltnerPeriod(val int)
	SetKeltnerMultiplier(val float64)
	SetWaveTrendN1(val int)
	SetWaveTrendN2(val int)
	SetWaveTrendOverbought(val float64)
	SetWaveTrendOversold(val float64)
	SetRSIPeriod(val int)
	SetRSIOversold(val float64)
	SetMACDFast(val int)
	SetMACDSlow(val int)
	SetMACDSignal(val int)
	SetBBPeriod(val int)
	SetBBStdDev(val float64)
	SetEMAPeriod(val int)
}

// DCAOptimizer implements genetic algorithm optimization for DCA strategies
type DCAOptimizer struct {
	config         OptimizationConfig
	ranges         *OptimizationRanges
	bestIndividual Individual
	rng            *rand.Rand
}

// NewDCAOptimizer creates a new DCA optimizer
func NewDCAOptimizer(config OptimizationConfig, ranges *OptimizationRanges) *DCAOptimizer {
	return &DCAOptimizer{
		config: config,
		ranges: ranges,
		rng:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Optimize runs the genetic algorithm optimization
func (opt *DCAOptimizer) Optimize(baseConfig interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error) {
	cfg := baseConfig.(BacktestConfig)
	
	log.Printf("🧬 Starting Genetic Algorithm Optimization")
	log.Printf("Population: %d, Generations: %d, Mutation: %.1f%%, Crossover: %.1f%%", 
		opt.config.PopulationSize, opt.config.Generations, opt.config.MutationRate*100, opt.config.CrossoverRate*100)

	// Initialize population
	population := opt.initializePopulation(cfg, opt.config.PopulationSize)
	
	var bestResults *backtest.BacktestResults
	
	for gen := 0; gen < opt.config.Generations; gen++ {
		// Evaluate fitness for all individuals in parallel
		opt.evaluatePopulationParallel(population, data)
		
		// Sort by fitness (descending)
		opt.sortPopulationByFitness(population)
		
		// Track best individual
		if opt.bestIndividual == nil || population[0].GetFitness() > opt.bestIndividual.GetFitness() {
			opt.bestIndividual = population[0]
			bestResults = population[0].GetResults()
		}
		
		if gen%5 == 0 { // Progress report every 5 generations
			log.Printf("🔄 Gen %d: Best=%.2f%%, Avg=%.2f%%, Worst=%.2f%%", 
				gen+1, 
				population[0].GetFitness()*100,
				opt.averageFitness(population)*100,
				population[len(population)-1].GetFitness()*100)
		}
		
		// Create next generation
		if gen < opt.config.Generations-1 {
			population = opt.createNextGeneration(population, cfg)
		}
	}
	
	log.Printf("✅ GA Optimization completed! Best fitness: %.2f%%", opt.bestIndividual.GetFitness()*100)
	return bestResults, opt.bestIndividual.GetConfig(), nil
}

// SetParameters updates the optimization parameters
func (opt *DCAOptimizer) SetParameters(populationSize, generations int, mutationRate, crossoverRate float64, eliteSize int) {
	opt.config.PopulationSize = populationSize
	opt.config.Generations = generations
	opt.config.MutationRate = mutationRate
	opt.config.CrossoverRate = crossoverRate
	opt.config.EliteSize = eliteSize
}

// GetBestIndividual returns the best individual found
func (opt *DCAOptimizer) GetBestIndividual() Individual {
	return opt.bestIndividual
}

// Private helper methods

func (opt *DCAOptimizer) initializePopulation(cfg BacktestConfig, size int) []Individual {
	population := make([]Individual, size)
	
	for i := 0; i < size; i++ {
		// Copy base config and randomize parameters
		individual := NewDCAIndividual(cfg)
		opt.randomizeConfig(individual.GetConfig().(BacktestConfig))
		population[i] = individual
	}
	
	return population
}

func (opt *DCAOptimizer) randomizeConfig(cfg BacktestConfig) {
	// This would contain the actual randomization logic from main.go
	// For now, this is a placeholder
}

func (opt *DCAOptimizer) evaluatePopulationParallel(population []Individual, data []types.OHLCV) {
	var wg sync.WaitGroup
	workerChan := make(chan struct{}, opt.config.MaxWorkers)
	
	for i := range population {
		if population[i].GetFitness() != 0 {
			continue // Skip already evaluated individuals
		}
		
		wg.Add(1)
		go func(individual Individual) {
			defer wg.Done()
			
			workerChan <- struct{}{} // Acquire worker slot
			defer func() { <-workerChan }() // Release worker slot
			
			// This would contain the actual evaluation logic
			// For now, we'll set a placeholder fitness
			individual.SetFitness(opt.rng.Float64())
		}(population[i])
	}
	
	wg.Wait()
}

func (opt *DCAOptimizer) sortPopulationByFitness(population []Individual) {
	// Simple bubble sort by fitness (descending)
	for i := 0; i < len(population)-1; i++ {
		for j := i + 1; j < len(population); j++ {
			if population[j].GetFitness() > population[i].GetFitness() {
				population[i], population[j] = population[j], population[i]
			}
		}
	}
}

func (opt *DCAOptimizer) averageFitness(population []Individual) float64 {
	sum := 0.0
	for _, ind := range population {
		sum += ind.GetFitness()
	}
	return sum / float64(len(population))
}

func (opt *DCAOptimizer) createNextGeneration(population []Individual, cfg BacktestConfig) []Individual {
	newPop := make([]Individual, len(population))
	
	// Elitism: keep best individuals
	for i := 0; i < opt.config.EliteSize; i++ {
		newPop[i] = population[i] // Already sorted by fitness
	}
	
	// Fill rest with crossover and mutation
	for i := opt.config.EliteSize; i < len(population); i++ {
		parent1 := opt.tournamentSelection(population)
		parent2 := opt.tournamentSelection(population)
		
		child := opt.crossover(parent1, parent2)
		opt.mutate(child, cfg)
		
		newPop[i] = child
	}
	
	return newPop
}

func (opt *DCAOptimizer) tournamentSelection(population []Individual) Individual {
	best := population[opt.rng.Intn(len(population))]
	
	for i := 1; i < opt.config.TournamentSize; i++ {
		candidate := population[opt.rng.Intn(len(population))]
		if candidate.GetFitness() > best.GetFitness() {
			best = candidate
		}
	}
	
	return best
}

func (opt *DCAOptimizer) crossover(parent1, parent2 Individual) Individual {
	// Create child starting with parent1
	child := NewDCAIndividual(parent1.GetConfig())
	
	// Apply crossover logic here
	if opt.rng.Float64() < opt.config.CrossoverRate {
		// Crossover logic would go here
	}
	
	return child
}

func (opt *DCAOptimizer) mutate(individual Individual, cfg BacktestConfig) {
	if opt.rng.Float64() < opt.config.MutationRate {
		// Mutation logic would go here
		
		// Reset fitness to force re-evaluation
		individual.SetFitness(0.0)
		individual.SetResults(nil)
	}
}
