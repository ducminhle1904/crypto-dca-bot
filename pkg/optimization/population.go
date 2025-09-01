package optimization

import (
	"sort"
)

// DCAPopulation represents a collection of DCA individuals
type DCAPopulation struct {
	individuals []Individual
}

// NewDCAPopulation creates a new population with the given individuals
func NewDCAPopulation(individuals []Individual) *DCAPopulation {
	return &DCAPopulation{
		individuals: individuals,
	}
}

// GetIndividuals returns all individuals in the population
func (p *DCAPopulation) GetIndividuals() []Individual {
	return p.individuals
}

// SetIndividuals sets the individuals in the population
func (p *DCAPopulation) SetIndividuals(individuals []Individual) {
	p.individuals = individuals
}

// Size returns the number of individuals in the population
func (p *DCAPopulation) Size() int {
	return len(p.individuals)
}

// GetBest returns the individual with the highest fitness
func (p *DCAPopulation) GetBest() Individual {
	if len(p.individuals) == 0 {
		return nil
	}
	
	best := p.individuals[0]
	for _, individual := range p.individuals[1:] {
		if individual.GetFitness() > best.GetFitness() {
			best = individual
		}
	}
	return best
}

// GetWorst returns the individual with the lowest fitness
func (p *DCAPopulation) GetWorst() Individual {
	if len(p.individuals) == 0 {
		return nil
	}
	
	worst := p.individuals[0]
	for _, individual := range p.individuals[1:] {
		if individual.GetFitness() < worst.GetFitness() {
			worst = individual
		}
	}
	return worst
}

// SortByFitness sorts the population by fitness in descending order (best first)
func (p *DCAPopulation) SortByFitness() {
	sort.Slice(p.individuals, func(i, j int) bool {
		return p.individuals[i].GetFitness() > p.individuals[j].GetFitness()
	})
}

// AverageFitness calculates the average fitness of all individuals
func (p *DCAPopulation) AverageFitness() float64 {
	if len(p.individuals) == 0 {
		return 0.0
	}
	
	sum := 0.0
	for _, individual := range p.individuals {
		sum += individual.GetFitness()
	}
	return sum / float64(len(p.individuals))
}

// GetElite returns the top n individuals by fitness
func (p *DCAPopulation) GetElite(n int) []Individual {
	if n >= len(p.individuals) {
		return p.individuals
	}
	
	// Ensure population is sorted
	p.SortByFitness()
	
	elite := make([]Individual, n)
	copy(elite, p.individuals[:n])
	return elite
}

// Add adds an individual to the population
func (p *DCAPopulation) Add(individual Individual) {
	p.individuals = append(p.individuals, individual)
}

// Clear removes all individuals from the population
func (p *DCAPopulation) Clear() {
	p.individuals = p.individuals[:0]
}
