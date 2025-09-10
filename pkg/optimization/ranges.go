package optimization

// DefaultOptimizationRanges provides the default parameter ranges for optimization
var DefaultOptimizationRanges = OptimizationRanges{
	Multipliers:     []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0},
	TPCandidates:    []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06},
	PriceThresholds: []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05},
	PriceThresholdMultipliers: []float64{1.0, 1.05, 1.1, 1.15, 1.2, 1.25, 1.3, 1.35, 1.4},
	RSIPeriods:      []int{10, 12, 14, 16, 18, 20, 22, 25},
	RSIOversold:     []float64{20, 25, 30, 35, 40},
	MACDFast:        []int{6, 8, 10, 12, 14, 16, 18},
	MACDSlow:        []int{20, 22, 24, 26, 28, 30, 32, 35},
	MACDSignal:      []int{7, 8, 9, 10, 12, 14},
	BBPeriods:       []int{10, 14, 16, 18, 20, 22, 25, 28, 30},
	BBStdDev:        []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0},
	EMAPeriods:      []int{15, 20, 25, 30, 40, 50, 60, 75, 100, 120},
	// Advanced combo ranges
	HullMAPeriods:      []int{10, 15, 20, 25, 30, 40, 50},
	MFIPeriods:         []int{10, 12, 14, 16, 18, 20, 22},
	MFIOversold:        []float64{15, 20, 25, 30},
	MFIOverbought:      []float64{70, 75, 80, 85},
	KeltnerPeriods:     []int{15, 20, 25, 30, 40, 50},
	KeltnerMultipliers: []float64{1.5, 1.8, 2.0, 2.2, 2.5, 3.0, 3.5},
	WaveTrendN1:        []int{8, 10, 12, 15, 18, 20},
	WaveTrendN2:        []int{18, 21, 24, 28, 32, 35},
	WaveTrendOverbought: []float64{50, 60, 70, 80},
	WaveTrendOversold:   []float64{-80, -70, -60, -50},
}

// GetDefaultOptimizationRanges returns the default optimization ranges
func GetDefaultOptimizationRanges() *OptimizationRanges {
	return &DefaultOptimizationRanges
}

// GetDefaultOptimizationConfig returns the default optimization configuration
func GetDefaultOptimizationConfig() OptimizationConfig {
	return OptimizationConfig{
		PopulationSize: 60,
		Generations:    35,
		MutationRate:   0.1,
		CrossoverRate:  0.8,
		EliteSize:      6,
		TournamentSize: 3,
		MaxWorkers:     4,
	}
}
