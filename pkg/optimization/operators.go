package optimization

import (
	"math/rand"
	"strings"
)

// DCAGeneticOperator implements genetic operations for DCA strategy optimization
type DCAGeneticOperator struct {
	ranges *OptimizationRanges
}

// NewDCAGeneticOperator creates a new DCA genetic operator with optimization ranges
func NewDCAGeneticOperator(ranges *OptimizationRanges) *DCAGeneticOperator {
	return &DCAGeneticOperator{
		ranges: ranges,
	}
}

// Crossover creates a child from two parents using genetic crossover
func (op *DCAGeneticOperator) Crossover(parent1, parent2 Individual, rate float64, rng *rand.Rand) Individual {
	// Cast to concrete types for DCA-specific operations
	p1Config := parent1.GetConfig()
	p2Config := parent2.GetConfig()
	
	// Create child starting with parent1's config
	childConfig := op.copyConfig(p1Config)
	child := NewDCAIndividual(childConfig)
	
	// Apply crossover based on rate
	if rng.Float64() < rate {
		op.applyCrossover(childConfig, p1Config, p2Config, rng)
	}
	
	return child
}

// Mutate applies genetic mutation to an individual
func (op *DCAGeneticOperator) Mutate(individual Individual, rate float64, rng *rand.Rand) {
	if rng.Float64() < rate {
		config := individual.GetConfig()
		op.applyMutation(config, rng)
		
		// Reset fitness to force re-evaluation
		individual.SetFitness(0.0)
		individual.SetResults(nil)
	}
}

// Select chooses an individual using tournament selection
func (op *DCAGeneticOperator) Select(population Population, tournamentSize int, rng *rand.Rand) Individual {
	individuals := population.GetIndividuals()
	if len(individuals) == 0 {
		return nil
	}
	
	best := individuals[rng.Intn(len(individuals))]
	
	for i := 1; i < tournamentSize; i++ {
		candidate := individuals[rng.Intn(len(individuals))]
		if candidate.GetFitness() > best.GetFitness() {
			best = candidate
		}
	}
	
	return best
}

// Helper function to copy configuration (simplified for interface compatibility)
func (op *DCAGeneticOperator) copyConfig(config interface{}) interface{} {
	// In a real implementation, this would perform a deep copy
	// For now, we'll assume the config is properly copied elsewhere
	return config
}

// applyCrossover performs the actual crossover between two configs
func (op *DCAGeneticOperator) applyCrossover(child, parent1, parent2 interface{}, rng *rand.Rand) {
	// This is a simplified version - in practice, you'd need to cast to specific config types
	// and perform field-by-field crossover based on the actual config structure
	// For now, this serves as a placeholder for the crossover logic
}

// applyMutation performs the actual mutation on a config
func (op *DCAGeneticOperator) applyMutation(config interface{}, rng *rand.Rand) {
	// This is a simplified version - in practice, you'd need to cast to specific config types
	// and mutate specific fields based on the optimization ranges
	// For now, this serves as a placeholder for the mutation logic
}

// randomChoice selects a random element from a slice
func randomChoice[T any](choices []T, rng *rand.Rand) T {
	if len(choices) == 0 {
		var zero T
		return zero
	}
	idx := rng.Intn(len(choices))
	return choices[idx]
}

// containsIndicator checks if an indicator is in the list
func containsIndicator(indicators []string, name string) bool {
	name = strings.ToLower(name)
	for _, n := range indicators {
		if strings.ToLower(n) == name {
			return true
		}
	}
	return false
}
