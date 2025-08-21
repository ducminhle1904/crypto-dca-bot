package backtest

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// HedgeOptimizer implements genetic algorithm optimization for hedge strategies
type HedgeOptimizer struct {
	// GA parameters
	populationSize  int
	generations     int
	mutationRate    float64
	crossoverRate   float64
	eliteSize       int
	tournamentSize  int
	
	// Data for optimization
	data       []types.OHLCV
	windowSize int
	
	// Fixed parameters
	initialBalance float64
	commission     float64
	
	// Exchange constraints
	minOrderQty    float64
	symbol         string
	
	// Optimization bounds
	bounds HedgeParameterBounds
}

// HedgeParameterBounds defines the optimization search space
type HedgeParameterBounds struct {
	// Core strategy parameters
	BaseAmountMin, BaseAmountMax         float64
	HedgeRatioMin, HedgeRatioMax         float64
	StopLossMin, StopLossMax             float64
	TakeProfitMin, TakeProfitMax         float64
	TrailingStopMin, TrailingStopMax     float64
	MaxDrawdownMin, MaxDrawdownMax       float64
	VolatilityThresholdMin, VolatilityThresholdMax float64
	TimeBetweenEntriesMin, TimeBetweenEntriesMax int
	
	// Indicator parameters
	HullMAPeriodMin, HullMAPeriodMax     int
	MFIPeriodMin, MFIPeriodMax           int
	MFIOversoldMin, MFIOversoldMax       float64
	MFIOverboughtMin, MFIOverboughtMax   float64
	KeltnerPeriodMin, KeltnerPeriodMax   int
	KeltnerMultiplierMin, KeltnerMultiplierMax float64
	WaveTrendN1Min, WaveTrendN1Max       int
	WaveTrendN2Min, WaveTrendN2Max       int
	WaveTrendOversoldMin, WaveTrendOversoldMax     float64
	WaveTrendOverboughtMin, WaveTrendOverboughtMax float64
}

// HedgeChromosome represents a candidate solution for hedge optimization
type HedgeChromosome struct {
	// Core strategy genes
	BaseAmount         float64
	HedgeRatio         float64
	StopLoss           float64
	TakeProfit         float64
	TrailingStop       float64
	MaxDrawdown        float64
	VolatilityThreshold float64
	TimeBetweenEntries int
	
	// Indicator genes
	HullMAPeriod        int
	MFIPeriod          int
	MFIOversold        float64
	MFIOverbought      float64
	KeltnerPeriod      int
	KeltnerMultiplier  float64
	WaveTrendN1        int
	WaveTrendN2        int
	WaveTrendOversold  float64
	WaveTrendOverbought float64
	
	// Fitness metrics
	Fitness            float64
	Results            *HedgeBacktestResults
}

// HedgeOptimizationObjective defines what to optimize for
type HedgeOptimizationObjective int

const (
	OptimizeHedgeEfficiency HedgeOptimizationObjective = iota
	OptimizeReturn
	OptimizeSharpe
	OptimizeVolatilityCapture
	OptimizeBalanced // Balanced approach considering multiple metrics
)

// HedgeOptimizationResult contains the results of hedge optimization
type HedgeOptimizationResult struct {
	BestChromosome     *HedgeChromosome
	BestResults        *HedgeBacktestResults
	AllChromosomes     []*HedgeChromosome
	GenerationHistory  []GenerationStats
	OptimizationTime   time.Duration
	Objective          HedgeOptimizationObjective
}

// GenerationStats tracks statistics for each generation
type GenerationStats struct {
	Generation     int
	BestFitness    float64
	AvgFitness     float64
	WorstFitness   float64
	BestReturn     float64
	BestHedgeEff   float64
	BestDrawdown   float64
}

// NewHedgeOptimizer creates a new hedge optimizer with default parameters
func NewHedgeOptimizer(data []types.OHLCV, windowSize int, initialBalance, commission float64) *HedgeOptimizer {
	return &HedgeOptimizer{
		populationSize:  40,  // Smaller population for hedge-specific optimization
		generations:     25,  // Fewer generations but more targeted
		mutationRate:    0.15, // Higher mutation rate for exploration
		crossoverRate:   0.8,
		eliteSize:       4,
		tournamentSize:  3,
		data:           data,
		windowSize:     windowSize,
		initialBalance: initialBalance,
		commission:     commission,
		bounds:         getDefaultHedgeBounds(),
	}
}

// SetPopulationSize sets the population size for optimization
func (ho *HedgeOptimizer) SetPopulationSize(size int) {
	ho.populationSize = size
}

// SetGenerations sets the number of generations for optimization
func (ho *HedgeOptimizer) SetGenerations(generations int) {
	ho.generations = generations
}

// SetMinOrderQuantity sets the minimum order quantity and updates optimization bounds
func (ho *HedgeOptimizer) SetMinOrderQuantity(minOrderQty float64, symbol string) {
	ho.minOrderQty = minOrderQty
	ho.symbol = symbol
	
	// Update bounds based on actual minimum order quantity
	ho.updateBoundsWithMinOrderQty()
}

// updateBoundsWithMinOrderQty adjusts optimization bounds based on actual minimum order quantity
func (ho *HedgeOptimizer) updateBoundsWithMinOrderQty() {
	if ho.minOrderQty <= 0 {
		return // No adjustment needed
	}
	
	// Estimate a reasonable price for the symbol (we'll use a conservative approach)
	// For BTC pairs, assume ~$50,000, for ETH pairs ~$3,000, for others ~$1
	var estimatedPrice float64
	symbolUpper := strings.ToUpper(ho.symbol)
	
	if strings.Contains(symbolUpper, "BTC") {
		estimatedPrice = 50000.0
	} else if strings.Contains(symbolUpper, "ETH") {
		estimatedPrice = 3000.0
	} else if strings.Contains(symbolUpper, "USDT") || strings.Contains(symbolUpper, "USDC") {
		estimatedPrice = 1.0
	} else {
		// For other pairs, use a moderate estimate
		estimatedPrice = 100.0
	}
	
	// Calculate minimum order value
	minOrderValue := ho.minOrderQty * estimatedPrice
	
	// Set BaseAmountMin to be at least 10x the minimum order value to ensure meaningful positions
	minBaseAmount := minOrderValue * 10
	
	// Ensure we don't set it too low (at least $10) or too high
	if minBaseAmount < 10.0 {
		minBaseAmount = 10.0
	}
	if minBaseAmount > ho.bounds.BaseAmountMax {
		minBaseAmount = ho.bounds.BaseAmountMax * 0.1 // 10% of max
	}
	
	// Update the bounds
	ho.bounds.BaseAmountMin = minBaseAmount
	
	fmt.Printf("🔧 Updated BaseAmountMin to $%.2f based on MinOrderQty=%.6f %s (est. price $%.2f)\n", 
		minBaseAmount, ho.minOrderQty, ho.symbol, estimatedPrice)
}

// getDefaultHedgeBounds returns reasonable default bounds for hedge optimization
func getDefaultHedgeBounds() HedgeParameterBounds {
	return HedgeParameterBounds{
		// Core strategy bounds
		BaseAmountMin: 50.0, BaseAmountMax: 500.0,
		HedgeRatioMin: 0.2, HedgeRatioMax: 0.9,
		StopLossMin: 0.02, StopLossMax: 0.08,
		TakeProfitMin: 0.015, TakeProfitMax: 0.05,
		TrailingStopMin: 0.01, TrailingStopMax: 0.04,
		MaxDrawdownMin: 0.05, MaxDrawdownMax: 0.15,
		VolatilityThresholdMin: 0.005, VolatilityThresholdMax: 0.025,
		TimeBetweenEntriesMin: 10, TimeBetweenEntriesMax: 60,
		
		// Indicator bounds
		HullMAPeriodMin: 10, HullMAPeriodMax: 40,
		MFIPeriodMin: 8, MFIPeriodMax: 25,
		MFIOversoldMin: 15.0, MFIOversoldMax: 35.0,
		MFIOverboughtMin: 65.0, MFIOverboughtMax: 85.0,
		KeltnerPeriodMin: 10, KeltnerPeriodMax: 35,
		KeltnerMultiplierMin: 1.5, KeltnerMultiplierMax: 3.0,
		WaveTrendN1Min: 6, WaveTrendN1Max: 15,
		WaveTrendN2Min: 15, WaveTrendN2Max: 30,
		WaveTrendOversoldMin: -80.0, WaveTrendOversoldMax: -40.0,
		WaveTrendOverboughtMin: 40.0, WaveTrendOverboughtMax: 80.0,
	}
}

// Optimize runs the genetic algorithm optimization
func (ho *HedgeOptimizer) Optimize(objective HedgeOptimizationObjective) *HedgeOptimizationResult {
	start := time.Now()
	fmt.Printf("🧬 Starting Hedge Strategy Optimization\n")
	fmt.Printf("📊 Population: %d, Generations: %d, Objective: %s\n", 
		ho.populationSize, ho.generations, ho.getObjectiveName(objective))
	fmt.Printf("📈 Data Points: %d, Window Size: %d\n", len(ho.data), ho.windowSize)
	
	// Initialize random population
	population := ho.initializePopulation()
	
	// Track optimization history
	history := make([]GenerationStats, 0, ho.generations)
	
	// Evolution loop
	for generation := 0; generation < ho.generations; generation++ {
		// Evaluate fitness for all chromosomes
		ho.evaluatePopulation(population, objective)
		
		// Sort by fitness (descending)
		sort.Slice(population, func(i, j int) bool {
			return population[i].Fitness > population[j].Fitness
		})
		
		// Track generation statistics
		stats := ho.calculateGenerationStats(population, generation)
		history = append(history, stats)
		
		// Print progress
		if generation%5 == 0 || generation == ho.generations-1 {
			ho.printProgress(stats)
		}
		
		// Create next generation
		if generation < ho.generations-1 {
			population = ho.createNextGeneration(population)
		}
	}
	
	duration := time.Since(start)
	
	fmt.Printf("\n✅ Optimization completed in %v\n", duration)
	fmt.Printf("🏆 Best fitness: %.4f\n", population[0].Fitness)
	
	return &HedgeOptimizationResult{
		BestChromosome:    population[0],
		BestResults:       population[0].Results,
		AllChromosomes:    population,
		GenerationHistory: history,
		OptimizationTime:  duration,
		Objective:         objective,
	}
}

// initializePopulation creates random initial population
func (ho *HedgeOptimizer) initializePopulation() []*HedgeChromosome {
	population := make([]*HedgeChromosome, ho.populationSize)
	
	for i := 0; i < ho.populationSize; i++ {
		population[i] = ho.createRandomChromosome()
	}
	
	return population
}

// createRandomChromosome creates a random chromosome within bounds
func (ho *HedgeOptimizer) createRandomChromosome() *HedgeChromosome {
	b := ho.bounds
	
	return &HedgeChromosome{
		BaseAmount:         ho.randomFloat(b.BaseAmountMin, b.BaseAmountMax),
		HedgeRatio:         ho.randomFloat(b.HedgeRatioMin, b.HedgeRatioMax),
		StopLoss:           ho.randomFloat(b.StopLossMin, b.StopLossMax),
		TakeProfit:         ho.randomFloat(b.TakeProfitMin, b.TakeProfitMax),
		TrailingStop:       ho.randomFloat(b.TrailingStopMin, b.TrailingStopMax),
		MaxDrawdown:        ho.randomFloat(b.MaxDrawdownMin, b.MaxDrawdownMax),
		VolatilityThreshold: ho.randomFloat(b.VolatilityThresholdMin, b.VolatilityThresholdMax),
		TimeBetweenEntries: ho.randomInt(b.TimeBetweenEntriesMin, b.TimeBetweenEntriesMax),
		
		HullMAPeriod:       ho.randomInt(b.HullMAPeriodMin, b.HullMAPeriodMax),
		MFIPeriod:          ho.randomInt(b.MFIPeriodMin, b.MFIPeriodMax),
		MFIOversold:        ho.randomFloat(b.MFIOversoldMin, b.MFIOversoldMax),
		MFIOverbought:      ho.randomFloat(b.MFIOverboughtMin, b.MFIOverboughtMax),
		KeltnerPeriod:      ho.randomInt(b.KeltnerPeriodMin, b.KeltnerPeriodMax),
		KeltnerMultiplier:  ho.randomFloat(b.KeltnerMultiplierMin, b.KeltnerMultiplierMax),
		WaveTrendN1:        ho.randomInt(b.WaveTrendN1Min, b.WaveTrendN1Max),
		WaveTrendN2:        ho.randomInt(b.WaveTrendN2Min, b.WaveTrendN2Max),
		WaveTrendOversold:  ho.randomFloat(b.WaveTrendOversoldMin, b.WaveTrendOversoldMax),
		WaveTrendOverbought: ho.randomFloat(b.WaveTrendOverboughtMin, b.WaveTrendOverboughtMax),
	}
}

// evaluatePopulation calculates fitness for all chromosomes
func (ho *HedgeOptimizer) evaluatePopulation(population []*HedgeChromosome, objective HedgeOptimizationObjective) {
	for _, chromosome := range population {
		if chromosome.Results == nil {
			ho.evaluateChromosome(chromosome, objective)
		}
	}
}

// evaluateChromosome runs backtest and calculates fitness for a chromosome
func (ho *HedgeOptimizer) evaluateChromosome(chromosome *HedgeChromosome, objective HedgeOptimizationObjective) {
	// Create strategy with chromosome parameters
	strategy := ho.createStrategyFromChromosome(chromosome)
	
	// Run hedge backtest
	engine := NewHedgeBacktestEngine(ho.initialBalance, ho.commission, strategy)
	results := engine.Run(ho.data, ho.windowSize)
	
	// Store results
	chromosome.Results = results
	
	// Calculate fitness based on objective
	chromosome.Fitness = ho.calculateFitness(results, objective)
}

// createStrategyFromChromosome creates a dual position strategy from chromosome
func (ho *HedgeOptimizer) createStrategyFromChromosome(chromosome *HedgeChromosome) strategy.Strategy {
	// Create dual position strategy
	hedge := strategy.NewDualPositionStrategy(chromosome.BaseAmount)
	
	// Configure hedge parameters
	hedge.SetHedgeRatio(chromosome.HedgeRatio)
	hedge.SetRiskParams(chromosome.StopLoss, chromosome.TakeProfit, 
		chromosome.TrailingStop, chromosome.MaxDrawdown)
	hedge.SetVolatilityThreshold(chromosome.VolatilityThreshold)
	hedge.SetTimeBetweenEntries(time.Duration(chromosome.TimeBetweenEntries) * time.Minute)
	
	// Add indicators with chromosome parameters
	// Hull Moving Average
	hullMA := indicators.NewHullMA(chromosome.HullMAPeriod)
	hedge.AddIndicator(hullMA)
	
	// Money Flow Index
	mfi := indicators.NewMFIWithPeriod(chromosome.MFIPeriod)
	mfi.SetOversold(chromosome.MFIOversold)
	mfi.SetOverbought(chromosome.MFIOverbought)
	hedge.AddIndicator(mfi)
	
	// Keltner Channels
	keltner := indicators.NewKeltnerChannelsCustom(chromosome.KeltnerPeriod, chromosome.KeltnerMultiplier)
	hedge.AddIndicator(keltner)
	
	// WaveTrend
	wavetrend := indicators.NewWaveTrendCustom(chromosome.WaveTrendN1, chromosome.WaveTrendN2)
	wavetrend.SetOverbought(chromosome.WaveTrendOverbought)
	wavetrend.SetOversold(chromosome.WaveTrendOversold)
	hedge.AddIndicator(wavetrend)
	
	return hedge
}

// calculateFitness calculates fitness score based on optimization objective
func (ho *HedgeOptimizer) calculateFitness(results *HedgeBacktestResults, objective HedgeOptimizationObjective) float64 {
	// Avoid division by zero and invalid results
	if results.TotalPositions == 0 || results.EndBalance <= 0 {
		return -1000.0
	}
	
	switch objective {
	case OptimizeHedgeEfficiency:
		// Maximize hedge efficiency while maintaining positive return
		efficiency := results.HedgeEfficiency
		if results.TotalReturn < -0.1 { // Penalty for large losses
			efficiency *= 0.1
		}
		return efficiency
		
	case OptimizeReturn:
		// Maximize total return with drawdown penalty
		return results.TotalReturn - (results.MaxDrawdown * 0.5)
		
	case OptimizeSharpe:
		// Maximize risk-adjusted return
		if results.MaxDrawdown == 0 {
			return results.TotalReturn * 10 // Bonus for no drawdown
		}
		return results.TotalReturn / results.MaxDrawdown
		
	case OptimizeVolatilityCapture:
		// Maximize absolute profit from volatility
		return results.VolatilityCapture / ho.initialBalance
		
	case OptimizeBalanced:
		fallthrough
	default:
		// Balanced optimization considering multiple factors
		returnScore := results.TotalReturn
		efficiencyScore := results.HedgeEfficiency * 0.5
		drawdownPenalty := results.MaxDrawdown * 0.8
		volatilityScore := (results.VolatilityCapture / ho.initialBalance) * 0.3
		
		return returnScore + efficiencyScore - drawdownPenalty + volatilityScore
	}
}

// createNextGeneration creates the next generation using GA operators
func (ho *HedgeOptimizer) createNextGeneration(population []*HedgeChromosome) []*HedgeChromosome {
	nextGen := make([]*HedgeChromosome, ho.populationSize)
	
	// Keep elite individuals
	for i := 0; i < ho.eliteSize; i++ {
		nextGen[i] = ho.copyChromosome(population[i])
	}
	
	// Generate rest through crossover and mutation
	for i := ho.eliteSize; i < ho.populationSize; i++ {
		parent1 := ho.tournamentSelection(population)
		parent2 := ho.tournamentSelection(population)
		
		child := ho.crossover(parent1, parent2)
		ho.mutate(child)
		
		nextGen[i] = child
	}
	
	return nextGen
}

// tournamentSelection selects parent through tournament selection
func (ho *HedgeOptimizer) tournamentSelection(population []*HedgeChromosome) *HedgeChromosome {
	best := population[rand.Intn(len(population))]
	
	for i := 1; i < ho.tournamentSize; i++ {
		candidate := population[rand.Intn(len(population))]
		if candidate.Fitness > best.Fitness {
			best = candidate
		}
	}
	
	return best
}

// crossover creates offspring from two parents
func (ho *HedgeOptimizer) crossover(parent1, parent2 *HedgeChromosome) *HedgeChromosome {
	child := &HedgeChromosome{}
	
	// Simple uniform crossover
	if rand.Float64() < 0.5 {
		child.BaseAmount = parent1.BaseAmount
	} else {
		child.BaseAmount = parent2.BaseAmount
	}
	
	if rand.Float64() < 0.5 {
		child.HedgeRatio = parent1.HedgeRatio
	} else {
		child.HedgeRatio = parent2.HedgeRatio
	}
	
	if rand.Float64() < 0.5 {
		child.StopLoss = parent1.StopLoss
	} else {
		child.StopLoss = parent2.StopLoss
	}
	
	if rand.Float64() < 0.5 {
		child.TakeProfit = parent1.TakeProfit
	} else {
		child.TakeProfit = parent2.TakeProfit
	}
	
	if rand.Float64() < 0.5 {
		child.TrailingStop = parent1.TrailingStop
	} else {
		child.TrailingStop = parent2.TrailingStop
	}
	
	if rand.Float64() < 0.5 {
		child.MaxDrawdown = parent1.MaxDrawdown
	} else {
		child.MaxDrawdown = parent2.MaxDrawdown
	}
	
	if rand.Float64() < 0.5 {
		child.VolatilityThreshold = parent1.VolatilityThreshold
	} else {
		child.VolatilityThreshold = parent2.VolatilityThreshold
	}
	
	if rand.Float64() < 0.5 {
		child.TimeBetweenEntries = parent1.TimeBetweenEntries
	} else {
		child.TimeBetweenEntries = parent2.TimeBetweenEntries
	}
	
	// Indicator parameters
	if rand.Float64() < 0.5 {
		child.HullMAPeriod = parent1.HullMAPeriod
	} else {
		child.HullMAPeriod = parent2.HullMAPeriod
	}
	
	if rand.Float64() < 0.5 {
		child.MFIPeriod = parent1.MFIPeriod
	} else {
		child.MFIPeriod = parent2.MFIPeriod
	}
	
	// Continue for other parameters...
	child.MFIOversold = ho.chooseParentValue(parent1.MFIOversold, parent2.MFIOversold)
	child.MFIOverbought = ho.chooseParentValue(parent1.MFIOverbought, parent2.MFIOverbought)
	child.KeltnerPeriod = ho.chooseParentIntValue(parent1.KeltnerPeriod, parent2.KeltnerPeriod)
	child.KeltnerMultiplier = ho.chooseParentValue(parent1.KeltnerMultiplier, parent2.KeltnerMultiplier)
	child.WaveTrendN1 = ho.chooseParentIntValue(parent1.WaveTrendN1, parent2.WaveTrendN1)
	child.WaveTrendN2 = ho.chooseParentIntValue(parent1.WaveTrendN2, parent2.WaveTrendN2)
	child.WaveTrendOversold = ho.chooseParentValue(parent1.WaveTrendOversold, parent2.WaveTrendOversold)
	child.WaveTrendOverbought = ho.chooseParentValue(parent1.WaveTrendOverbought, parent2.WaveTrendOverbought)
	
	return child
}

// chooseParentValue randomly chooses between two parent values
func (ho *HedgeOptimizer) chooseParentValue(val1, val2 float64) float64 {
	if rand.Float64() < 0.5 {
		return val1
	}
	return val2
}

// chooseParentIntValue randomly chooses between two parent int values
func (ho *HedgeOptimizer) chooseParentIntValue(val1, val2 int) int {
	if rand.Float64() < 0.5 {
		return val1
	}
	return val2
}

// mutate applies mutation to a chromosome
func (ho *HedgeOptimizer) mutate(chromosome *HedgeChromosome) {
	b := ho.bounds
	
	if rand.Float64() < ho.mutationRate {
		chromosome.BaseAmount = ho.randomFloat(b.BaseAmountMin, b.BaseAmountMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.HedgeRatio = ho.randomFloat(b.HedgeRatioMin, b.HedgeRatioMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.StopLoss = ho.randomFloat(b.StopLossMin, b.StopLossMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.TakeProfit = ho.randomFloat(b.TakeProfitMin, b.TakeProfitMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.TrailingStop = ho.randomFloat(b.TrailingStopMin, b.TrailingStopMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.MaxDrawdown = ho.randomFloat(b.MaxDrawdownMin, b.MaxDrawdownMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.VolatilityThreshold = ho.randomFloat(b.VolatilityThresholdMin, b.VolatilityThresholdMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.TimeBetweenEntries = ho.randomInt(b.TimeBetweenEntriesMin, b.TimeBetweenEntriesMax)
	}
	
	// Mutate indicator parameters
	if rand.Float64() < ho.mutationRate {
		chromosome.HullMAPeriod = ho.randomInt(b.HullMAPeriodMin, b.HullMAPeriodMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.MFIPeriod = ho.randomInt(b.MFIPeriodMin, b.MFIPeriodMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.MFIOversold = ho.randomFloat(b.MFIOversoldMin, b.MFIOversoldMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.MFIOverbought = ho.randomFloat(b.MFIOverboughtMin, b.MFIOverboughtMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.KeltnerPeriod = ho.randomInt(b.KeltnerPeriodMin, b.KeltnerPeriodMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.KeltnerMultiplier = ho.randomFloat(b.KeltnerMultiplierMin, b.KeltnerMultiplierMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.WaveTrendN1 = ho.randomInt(b.WaveTrendN1Min, b.WaveTrendN1Max)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.WaveTrendN2 = ho.randomInt(b.WaveTrendN2Min, b.WaveTrendN2Max)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.WaveTrendOversold = ho.randomFloat(b.WaveTrendOversoldMin, b.WaveTrendOversoldMax)
	}
	
	if rand.Float64() < ho.mutationRate {
		chromosome.WaveTrendOverbought = ho.randomFloat(b.WaveTrendOverboughtMin, b.WaveTrendOverboughtMax)
	}
	
	// Ensure N1 < N2 for WaveTrend
	if chromosome.WaveTrendN1 >= chromosome.WaveTrendN2 {
		chromosome.WaveTrendN2 = chromosome.WaveTrendN1 + ho.randomInt(3, 10)
		if chromosome.WaveTrendN2 > b.WaveTrendN2Max {
			chromosome.WaveTrendN1 = b.WaveTrendN1Min
			chromosome.WaveTrendN2 = b.WaveTrendN2Max
		}
	}
	
	// Ensure oversold < overbought
	if chromosome.MFIOversold >= chromosome.MFIOverbought {
		chromosome.MFIOversold = b.MFIOversoldMin
		chromosome.MFIOverbought = b.MFIOverboughtMax
	}
	
	if chromosome.WaveTrendOversold >= chromosome.WaveTrendOverbought {
		chromosome.WaveTrendOversold = b.WaveTrendOversoldMin
		chromosome.WaveTrendOverbought = b.WaveTrendOverboughtMax
	}
}

// Helper functions
func (ho *HedgeOptimizer) randomFloat(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func (ho *HedgeOptimizer) randomInt(min, max int) int {
	return min + rand.Intn(max-min+1)
}

func (ho *HedgeOptimizer) copyChromosome(src *HedgeChromosome) *HedgeChromosome {
	return &HedgeChromosome{
		BaseAmount:          src.BaseAmount,
		HedgeRatio:          src.HedgeRatio,
		StopLoss:           src.StopLoss,
		TakeProfit:         src.TakeProfit,
		TrailingStop:       src.TrailingStop,
		MaxDrawdown:        src.MaxDrawdown,
		VolatilityThreshold: src.VolatilityThreshold,
		TimeBetweenEntries: src.TimeBetweenEntries,
		HullMAPeriod:       src.HullMAPeriod,
		MFIPeriod:          src.MFIPeriod,
		MFIOversold:        src.MFIOversold,
		MFIOverbought:      src.MFIOverbought,
		KeltnerPeriod:      src.KeltnerPeriod,
		KeltnerMultiplier:  src.KeltnerMultiplier,
		WaveTrendN1:        src.WaveTrendN1,
		WaveTrendN2:        src.WaveTrendN2,
		WaveTrendOversold:  src.WaveTrendOversold,
		WaveTrendOverbought: src.WaveTrendOverbought,
		Fitness:            src.Fitness,
		Results:            src.Results,
	}
}

func (ho *HedgeOptimizer) calculateGenerationStats(population []*HedgeChromosome, generation int) GenerationStats {
	if len(population) == 0 {
		return GenerationStats{Generation: generation}
	}
	
	best := population[0]
	worst := population[len(population)-1]
	
	var fitnessSum float64
	for _, chromo := range population {
		fitnessSum += chromo.Fitness
	}
	
	return GenerationStats{
		Generation:   generation,
		BestFitness:  best.Fitness,
		AvgFitness:   fitnessSum / float64(len(population)),
		WorstFitness: worst.Fitness,
		BestReturn:   best.Results.TotalReturn,
		BestHedgeEff: best.Results.HedgeEfficiency,
		BestDrawdown: best.Results.MaxDrawdown,
	}
}

func (ho *HedgeOptimizer) printProgress(stats GenerationStats) {
	fmt.Printf("Gen %2d: Fitness=%.4f, Return=%.2f%%, HedgeEff=%.2f%%, DD=%.2f%%\n",
		stats.Generation, stats.BestFitness, stats.BestReturn*100,
		stats.BestHedgeEff*100, stats.BestDrawdown*100)
}

func (ho *HedgeOptimizer) getObjectiveName(objective HedgeOptimizationObjective) string {
	switch objective {
	case OptimizeHedgeEfficiency:
		return "Hedge Efficiency"
	case OptimizeReturn:
		return "Total Return"
	case OptimizeSharpe:
		return "Sharpe Ratio"
	case OptimizeVolatilityCapture:
		return "Volatility Capture"
	case OptimizeBalanced:
		return "Balanced"
	default:
		return "Unknown"
	}
}

// PrintOptimizationSummary prints a summary of optimization results
func (hor *HedgeOptimizationResult) PrintOptimizationSummary() {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("🧬 HEDGE OPTIMIZATION RESULTS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")
	
	fmt.Printf("🎯 Objective: %s\n", hor.getObjectiveName())
	fmt.Printf("⏱️ Duration: %v\n", hor.OptimizationTime)
	fmt.Printf("🏆 Best Fitness: %.4f\n", hor.BestChromosome.Fitness)
	
	fmt.Printf("\n" + strings.Repeat("-", 40) + "\n")
	fmt.Printf("📊 OPTIMAL PARAMETERS\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	
	best := hor.BestChromosome
	fmt.Printf("💰 Base Amount: $%.2f\n", best.BaseAmount)
	fmt.Printf("🔄 Hedge Ratio: %.3f\n", best.HedgeRatio)
	fmt.Printf("🛑 Stop Loss: %.2f%%\n", best.StopLoss*100)
	fmt.Printf("🎯 Take Profit: %.2f%%\n", best.TakeProfit*100)
	fmt.Printf("📈 Trailing Stop: %.2f%%\n", best.TrailingStop*100)
	fmt.Printf("⚠️ Max Drawdown: %.2f%%\n", best.MaxDrawdown*100)
	fmt.Printf("📊 Volatility Threshold: %.2f%%\n", best.VolatilityThreshold*100)
	fmt.Printf("⏰ Time Between Entries: %d min\n", best.TimeBetweenEntries)
	
	fmt.Printf("\n" + strings.Repeat("-", 30) + "\n")
	fmt.Printf("📈 INDICATOR PARAMETERS\n")
	fmt.Printf(strings.Repeat("-", 30) + "\n")
	fmt.Printf("Hull MA Period: %d\n", best.HullMAPeriod)
	fmt.Printf("MFI Period: %d (%.1f/%.1f)\n", best.MFIPeriod, best.MFIOversold, best.MFIOverbought)
	fmt.Printf("Keltner: %d (%.2f)\n", best.KeltnerPeriod, best.KeltnerMultiplier)
	fmt.Printf("WaveTrend: %d/%d (%.1f/%.1f)\n", 
		best.WaveTrendN1, best.WaveTrendN2, best.WaveTrendOversold, best.WaveTrendOverbought)
	
	fmt.Printf("\n" + strings.Repeat("-", 40) + "\n")
	fmt.Printf("📊 PERFORMANCE RESULTS\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	
	if hor.BestResults != nil {
		results := hor.BestResults
		fmt.Printf("💰 Return: %.2f%%\n", results.TotalReturn*100)
		fmt.Printf("📉 Max Drawdown: %.2f%%\n", results.MaxDrawdown*100)
		fmt.Printf("🔄 Hedge Efficiency: %.2f%%\n", results.HedgeEfficiency*100)
		fmt.Printf("🎯 Volatility Capture: $%.2f\n", results.VolatilityCapture)
		fmt.Printf("📊 Total Positions: %d (%d Long, %d Short)\n", 
			results.TotalPositions, results.LongPositions, results.ShortPositions)
		fmt.Printf("✅ Win Rates: Long %.1f%%, Short %.1f%%\n", 
			results.LongWinRate*100, results.ShortWinRate*100)
	}
	
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
}

func (hor *HedgeOptimizationResult) getObjectiveName() string {
	switch hor.Objective {
	case OptimizeHedgeEfficiency:
		return "Hedge Efficiency"
	case OptimizeReturn:
		return "Total Return"
	case OptimizeSharpe:
		return "Sharpe Ratio"
	case OptimizeVolatilityCapture:
		return "Volatility Capture"
	case OptimizeBalanced:
		return "Balanced"
	default:
		return "Unknown"
	}
}
