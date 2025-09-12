package optimization

import (
	"math/rand"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// Package optimization provides genetic algorithm optimization for trading strategies

// Individual represents a candidate solution in the genetic algorithm
type Individual interface {
	GetConfig() interface{}
	GetFitness() float64
	SetFitness(fitness float64)
	GetResults() *backtest.BacktestResults
	SetResults(results *backtest.BacktestResults)
}

// Population represents a collection of individuals
type Population interface {
	GetIndividuals() []Individual
	Size() int
	GetBest() Individual
	SetIndividuals(individuals []Individual)
}

// GeneticOperator defines interface for genetic algorithm operations
type GeneticOperator interface {
	Crossover(parent1, parent2 Individual, rate float64, rng *rand.Rand) Individual
	Mutate(individual Individual, rate float64, rng *rand.Rand)
	Select(population Population, tournamentSize int, rng *rand.Rand) Individual
}

// FitnessEvaluator evaluates the fitness of individuals
type FitnessEvaluator interface {
	Evaluate(individual Individual, data []types.OHLCV) error
	EvaluatePopulation(population Population, data []types.OHLCV) error
}

// Optimizer defines the main optimization interface
type Optimizer interface {
	Optimize(baseConfig interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error)
	SetParameters(populationSize, generations int, mutationRate, crossoverRate float64, eliteSize int)
	GetBestIndividual() Individual
}

// OptimizationConfig holds the configuration for the genetic algorithm
type OptimizationConfig struct {
	PopulationSize int
	Generations    int
	MutationRate   float64
	CrossoverRate  float64
	EliteSize      int
	TournamentSize int
	MaxWorkers     int
}

// OptimizationRanges defines the parameter ranges for optimization
type OptimizationRanges struct {
	Multipliers        []float64
	TPCandidates       []float64
	PriceThresholds    []float64
	PriceThresholdMultipliers []float64 // Progressive DCA spacing multipliers
	RSIPeriods         []int
	RSIOversold        []float64
	MACDFast           []int
	MACDSlow           []int
	MACDSignal         []int
	BBPeriods          []int
	BBStdDev           []float64
	EMAPeriods         []int
	SuperTrendPeriods     []int
	SuperTrendMultipliers []float64
	MFIPeriods         []int
	MFIOversold        []float64
	MFIOverbought      []float64
	KeltnerPeriods     []int
	KeltnerMultipliers []float64
	WaveTrendN1        []int
	WaveTrendN2        []int
	WaveTrendOverbought []float64
	WaveTrendOversold   []float64
}
