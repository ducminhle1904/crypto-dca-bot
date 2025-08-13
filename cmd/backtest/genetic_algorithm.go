package main

import (
	"math/rand"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

var (
	rng = rand.New(rand.NewSource(time.Now().UnixNano())) // Random number generator for optimization
)

// Individual represents a candidate solution
type Individual struct {
	config  BacktestConfig
	fitness float64
	results *backtest.BacktestResults
}

// Initialize random population
func initializePopulation(cfg *BacktestConfig, size int, optimizeIndicators bool) []*Individual {
	population := make([]*Individual, size)
	
	// Base indicators for combination generation
	baseIndicators := cfg.Indicators
	if len(baseIndicators) == 0 {
		baseIndicators = []string{"rsi", "macd", "bb", "sma"}
	}
	
	// Fixed parameter ranges for all optimization (no style constraints)
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
			config: *cfg, // Copy base config
		}
		
		// Randomize parameters using fixed ranges
		individual.config.BaseAmount = cfg.BaseAmount // Use flag value
		individual.config.MaxMultiplier = randomChoice(multipliers)
		individual.config.PriceThreshold = randomChoice(priceThresholds)
		
		// TP candidates
		if cfg.Cycle {
			individual.config.TPPercent = randomChoice(tpCandidates)
		} else {
			individual.config.TPPercent = 0
		}
		
		// Indicator selection
		if optimizeIndicators {
			individual.config.Indicators = randomIndicatorCombo(baseIndicators)
		} else {
			individual.config.Indicators = cfg.Indicators
		}
		
		// Indicator parameters using fixed ranges
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

// Random selection helpers
func randomChoice[T any](choices []T) T {
	if len(choices) == 0 {
		var zero T
		return zero
	}
	idx := rng.Intn(len(choices))
	return choices[idx]
}

func randomIndicatorCombo(base []string) []string {
	if len(base) == 0 {
		return []string{}
	}
	
	// Generate random bitmask (at least 1 indicator)
	mask := 1 + (rng.Intn((1 << len(base)) - 1))
	
	var combo []string
	for i := 0; i < len(base); i++ {
		if mask&(1<<i) != 0 {
			combo = append(combo, base[i])
		}
	}
	return combo
}

// Sort population by fitness (descending)
func sortPopulationByFitness(population []*Individual) {
	for i := 0; i < len(population)-1; i++ {
		for j := i + 1; j < len(population); j++ {
			if population[j].fitness > population[i].fitness {
				population[i], population[j] = population[j], population[i]
			}
		}
	}
}

// Calculate average fitness
func averageFitness(population []*Individual) float64 {
	sum := 0.0
	for _, ind := range population {
		sum += ind.fitness
	}
	return sum / float64(len(population))
}

// Create next generation using selection, crossover, and mutation
func createNextGeneration(population []*Individual, eliteSize int, crossoverRate, mutationRate float64, cfg *BacktestConfig, optimizeIndicators bool) []*Individual {
	newPop := make([]*Individual, len(population))
	
	// Elitism: keep best individuals
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
		mutate(child, mutationRate, cfg, optimizeIndicators)
		
		newPop[i] = child
	}
	
	return newPop
}

// Tournament selection
func tournamentSelection(population []*Individual, tournamentSize int) *Individual {
	best := population[rng.Intn(len(population))]
	
	for i := 1; i < tournamentSize; i++ {
		candidate := population[rng.Intn(len(population))]
		if candidate.fitness > best.fitness {
			best = candidate
		}
	}
	
	return best
}

// Crossover two parents to create a child
func crossover(parent1, parent2 *Individual, rate float64) *Individual {
	child := &Individual{
		config: parent1.config, // Start with parent1
	}
	
	// Random crossover based on rate
	if rng.Float64() < rate {
		// Mix parameters from both parents
		if rng.Intn(2) == 0 {
			child.config.MaxMultiplier = parent2.config.MaxMultiplier
		}
		if rng.Intn(2) == 0 {
			child.config.TPPercent = parent2.config.TPPercent
		}
		if rng.Intn(2) == 0 {
			child.config.RSIPeriod = parent2.config.RSIPeriod
		}
		if rng.Intn(2) == 0 {
			child.config.RSIOversold = parent2.config.RSIOversold
		}
		if rng.Intn(2) == 0 {
			child.config.MACDFast = parent2.config.MACDFast
		}
		if rng.Intn(2) == 0 {
			child.config.MACDSlow = parent2.config.MACDSlow
		}
		if rng.Intn(2) == 0 {
			child.config.MACDSignal = parent2.config.MACDSignal
		}
		if rng.Intn(2) == 0 {
			child.config.BBPeriod = parent2.config.BBPeriod
		}
		if rng.Intn(2) == 0 {
			child.config.BBStdDev = parent2.config.BBStdDev
		}
		if rng.Intn(2) == 0 {
			child.config.SMAPeriod = parent2.config.SMAPeriod
		}
		if rng.Intn(2) == 0 {
			child.config.Indicators = parent2.config.Indicators
		}
	}
	
	return child
}

// Mutate an individual
func mutate(individual *Individual, rate float64, cfg *BacktestConfig, optimizeIndicators bool) {
	if rng.Float64() < rate {
		// Fixed parameter ranges for mutation
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
		
		// Randomly mutate one parameter (expanded to include price threshold)
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
		
		// Reset fitness to force re-evaluation
		individual.fitness = 0
		individual.results = nil
	}
} 