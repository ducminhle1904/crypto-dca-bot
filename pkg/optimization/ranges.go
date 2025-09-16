package optimization

// DefaultOptimizationRanges provides the default parameter ranges for optimization
var DefaultOptimizationRanges = OptimizationRanges{
	Multipliers:     []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0, 4.5},
	TPCandidates:    []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06, 0.07, 0.08},
	PriceThresholds: []float64{0.008, 0.01, 0.012, 0.015, 0.018, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05},
	PriceThresholdMultipliers: []float64{1.0, 1.05, 1.1, 1.15, 1.2, 1.25, 1.3, 1.35, 1.4, 1.45, 1.5},
	RSIPeriods:      []int{10, 12, 14, 16, 18, 20, 22, 25},
	RSIOversold:     []float64{20, 25, 30, 35, 40},
	MACDFast:        []int{8, 10, 12, 14, 16, 18},
	MACDSlow:        []int{20, 22, 24, 26, 28, 30, 32, 35},
	MACDSignal:      []int{7, 8, 9, 10, 12, 14},
	BBPeriods:       []int{14, 16, 18, 20, 22, 25, 28, 30},
	BBStdDev:        []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0},
	EMAPeriods:      []int{15, 20, 25, 30, 40, 50, 60, 75, 100},
	SuperTrendPeriods:     []int{10, 12, 14, 16, 18, 20, 25},
	SuperTrendMultipliers: []float64{1.5, 2.0, 2.5, 3.0, 3.5, 4.0},
	HullMAPeriods:         []int{8, 10, 12, 14, 16, 18, 20, 22, 25, 30},
	MFIPeriods:         []int{10, 12, 14, 16, 18, 20, 22},
	MFIOversold:        []float64{15, 20, 25, 30},
	MFIOverbought:      []float64{70, 75, 80, 85},
	KeltnerPeriods:     []int{15, 20, 25, 30, 40, 50},
	KeltnerMultipliers: []float64{1.5, 1.8, 2.0, 2.2, 2.5, 3.0, 3.5},
	WaveTrendN1:        []int{8, 10, 12, 15, 18, 20},
	WaveTrendN2:        []int{18, 21, 24, 28, 32, 35},
	WaveTrendOverbought: []float64{50, 60, 70, 80},
	WaveTrendOversold:   []float64{-80, -70, -60, -50},
	OBVTrendThresholds: []float64{0.005, 0.008, 0.01, 0.012, 0.015, 0.018, 0.02, 0.025, 0.03},
	StochasticRSIPeriods: []int{10, 12, 14, 16, 18, 20, 22},
	StochasticRSIOverboughts: []float64{75.0, 80.0, 85.0, 90.0},
	StochasticRSIOversolds: []float64{10.0, 15.0, 20.0, 25.0},
	VolatilitySensitivity: []float64{1.0, 1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0},
	ATRPeriods:           []int{10, 12, 14, 16, 18, 21, 24, 28},
	LevelMultipliers:     []float64{1.05, 1.1, 1.15, 1.2, 1.25, 1.3, 1.35, 1.4},
}

// GetDefaultOptimizationRanges returns the default optimization ranges
func GetDefaultOptimizationRanges() *OptimizationRanges {
	return &DefaultOptimizationRanges
}

// GetDefaultOptimizationConfig returns the default optimization configuration
func GetDefaultOptimizationConfig() OptimizationConfig {
	return OptimizationConfig{
		PopulationSize: 40,  // Balanced for large datasets
		Generations:    25,  // Moderate generations for convergence
		MutationRate:   0.18, // Slightly higher mutation for exploration
		CrossoverRate:  0.8,  // Good crossover rate
		EliteSize:      6,    // Keep best individuals
		TournamentSize: 3,    // Standard tournament size
		MaxWorkers:     6,    // Balanced parallel workers
	}
}
