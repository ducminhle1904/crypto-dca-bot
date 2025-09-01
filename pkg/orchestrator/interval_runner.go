package orchestrator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	datamanager "github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/optimization"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/validation"
)

// DefaultIntervalRunner implements the IntervalRunner interface
type DefaultIntervalRunner struct {
	backtestRunner BacktestRunner
}

// NewDefaultIntervalRunner creates a new default interval runner
func NewDefaultIntervalRunner() IntervalRunner {
	return &DefaultIntervalRunner{
		backtestRunner: NewDefaultBacktestRunner(),
	}
}

// FindAvailableIntervals discovers all available intervals for a symbol
func (r *DefaultIntervalRunner) FindAvailableIntervals(dataRoot, exchange, symbol string) ([]string, error) {
	sym := strings.ToUpper(symbol)
	var availableIntervals []string
	
	// Define categories by exchange
	var categories []string
	switch strings.ToLower(exchange) {
	case "bybit":
		categories = []string{"spot", "linear", "inverse"}
	case "binance":
		categories = []string{"spot", "futures"}
	default:
		categories = []string{"spot", "futures", "linear", "inverse"}
	}
	
	// Check each category for available intervals
	for _, category := range categories {
		categoryDir := filepath.Join(dataRoot, exchange, category, sym)
		entries, err := os.ReadDir(categoryDir)
		if err != nil {
			continue // Skip if category doesn't exist
		}
		
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			
			interval := e.Name()
			candlesPath := filepath.Join(categoryDir, interval, "candles.csv")
			if _, err := os.Stat(candlesPath); err == nil {
				// Only add if not already found
				found := false
				for _, existing := range availableIntervals {
					if existing == interval {
						found = true
						break
					}
				}
				if !found {
					availableIntervals = append(availableIntervals, interval)
				}
			}
		}
	}
	
	if len(availableIntervals) == 0 {
		return nil, fmt.Errorf("no data found for symbol %s in exchange %s at %s", sym, exchange, dataRoot)
	}
	
	return availableIntervals, nil
}

// RunForInterval executes workflow for a specific interval
func (r *DefaultIntervalRunner) RunForInterval(cfg *config.DCAConfig, dataRoot, exchange, interval string, optimize bool, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) (*IntervalResult, error) {
	// Create a copy of the config for this interval
	cfgCopy := *cfg
	
	// Find data file for this interval
	// Note: interval is already in minutes format from FindAvailableIntervals
	dataFile := datamanager.FindDataFile(dataRoot, exchange, cfg.Symbol, interval+"m")
	if _, err := os.Stat(dataFile); err != nil {
		return nil, fmt.Errorf("data file not found for interval %s: %w", interval, err)
	}
	
	cfgCopy.DataFile = dataFile
	cfgCopy.Interval = interval
	
	// Fetch minimum order quantity for this interval
	backtestRunner := NewDefaultBacktestRunner()
	if err := backtestRunner.(*DefaultBacktestRunner).fetchAndSetMinOrderQty(&cfgCopy); err != nil {
		log.Printf("⚠️ Could not fetch minimum order quantity for %s: %v", interval, err)
	}
	
	var optimizedCfg *config.DCAConfig
	var results *backtest.BacktestResults
	var err error
	
	if optimize {
		log.Printf("🧬 Optimizing for interval: %s", interval)
		
		if wfConfig != nil && wfConfig.Enable {
			// Run walk-forward validation
			if wfErr := r.runWalkForwardForInterval(&cfgCopy, selectedPeriod, wfConfig); wfErr != nil {
				log.Printf("⚠️ Walk-forward validation failed for %s: %v", interval, wfErr)
				// Continue with regular optimization
			}
		}
		
		// Run optimization
		var optimizedCfgInterface interface{}
		results, optimizedCfgInterface, err = optimization.OptimizeWithGA(&cfgCopy, cfgCopy.DataFile, selectedPeriod)
		if err != nil {
			return nil, fmt.Errorf("optimization failed for interval %s: %w", interval, err)
		}
		optimizedCfg = optimizedCfgInterface.(*config.DCAConfig)
	} else {
		log.Printf("📊 Running backtest for interval: %s", interval)
		results, err = backtestRunner.RunWithFile(&cfgCopy, selectedPeriod)
		if err != nil {
			return nil, fmt.Errorf("backtest failed for interval %s: %w", interval, err)
		}
		optimizedCfg = &cfgCopy
	}
	
	log.Printf("✅ Completed interval %s: %.2f%% return", interval, results.TotalReturn*100)
	
	return &IntervalResult{
		Interval:     interval,
		Results:      results,
		OptimizedCfg: optimizedCfg,
	}, nil
}

// runWalkForwardForInterval runs walk-forward validation for a specific interval
func (r *DefaultIntervalRunner) runWalkForwardForInterval(cfg *config.DCAConfig, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) error {
	// Load data for walk-forward validation
	data, err := datamanager.LoadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to load data for walk-forward validation: %w", err)
	}
	
	if selectedPeriod > 0 {
		data = datamanager.FilterDataByPeriod(data, selectedPeriod)
	}
	
	// Run walk-forward validation using pkg/validation
	_, err = validation.RunWalkForwardValidation(cfg, data, *wfConfig,
		func(config interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error) {
			return optimization.OptimizeWithGA(config, cfg.DataFile, 0)
		},
		func(cfg interface{}, data []types.OHLCV) *backtest.BacktestResults {
			dcaConfig := cfg.(*config.DCAConfig)
			results, err := r.backtestRunner.RunWithData(dcaConfig, data)
			if err != nil {
				log.Printf("Error in validation backtest: %v", err)
				return nil
			}
			return results
		})
	
	return err
}

// Helper functions removed - dataRoot and exchange are now passed directly
