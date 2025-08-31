package optimization

import (
	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
)

// DCAIndividual represents a candidate DCA configuration solution
type DCAIndividual struct {
	config  interface{} // Will be BacktestConfig in practice
	fitness float64
	results *backtest.BacktestResults
}

// NewDCAIndividual creates a new DCA individual with the given config
func NewDCAIndividual(config interface{}) *DCAIndividual {
	return &DCAIndividual{
		config:  config,
		fitness: 0.0,
		results: nil,
	}
}

// GetConfig returns the configuration for this individual
func (i *DCAIndividual) GetConfig() interface{} {
	return i.config
}

// SetConfig sets the configuration for this individual
func (i *DCAIndividual) SetConfig(config interface{}) {
	i.config = config
}

// GetFitness returns the fitness score for this individual
func (i *DCAIndividual) GetFitness() float64 {
	return i.fitness
}

// SetFitness sets the fitness score for this individual
func (i *DCAIndividual) SetFitness(fitness float64) {
	i.fitness = fitness
}

// GetResults returns the backtest results for this individual
func (i *DCAIndividual) GetResults() *backtest.BacktestResults {
	return i.results
}

// SetResults sets the backtest results for this individual
func (i *DCAIndividual) SetResults(results *backtest.BacktestResults) {
	i.results = results
}

// Copy creates a deep copy of this individual
func (i *DCAIndividual) Copy() Individual {
	return &DCAIndividual{
		config:  i.config, // Note: config should be copied by caller if needed
		fitness: i.fitness,
		results: i.results,
	}
}

// Reset resets the fitness and results for re-evaluation
func (i *DCAIndividual) Reset() {
	i.fitness = 0.0
	i.results = nil
}
