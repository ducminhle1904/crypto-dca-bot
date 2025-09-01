package orchestrator

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	datamanager "github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/optimization"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/validation"
)

// DefaultIntervalRunner implements the IntervalRunner interface with performance optimizations
type DefaultIntervalRunner struct {
	backtestRunner BacktestRunner
	minQtyCache    map[string]float64 // Cache minimum order quantities to avoid API calls
	cacheMutex     sync.RWMutex
}

// NewDefaultIntervalRunner creates a new default interval runner with performance optimizations
func NewDefaultIntervalRunner() IntervalRunner {
	return &DefaultIntervalRunner{
		backtestRunner: NewDefaultBacktestRunner(),
		minQtyCache:    make(map[string]float64),
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
	
	// PERFORMANCE FIX: Use cached minimum order quantity to avoid API calls
	if err := r.fetchAndSetMinOrderQtyCached(&cfgCopy); err != nil {
		log.Printf("⚠️ Could not fetch minimum order quantity for %s: %v", interval, err)
		// Set a sensible default instead of failing
		cfgCopy.MinOrderQty = 0.001 // Default for most USDT pairs
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
		// PERFORMANCE FIX: Reuse existing backtest runner
		results, err = r.backtestRunner.RunWithFile(&cfgCopy, selectedPeriod)
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

// runWalkForwardForInterval runs walk-forward validation for a specific interval with detailed logging
func (r *DefaultIntervalRunner) runWalkForwardForInterval(cfg *config.DCAConfig, selectedPeriod time.Duration, wfConfig *validation.WalkForwardConfig) error {
	log.Printf("🔄 [%s] Walk-Forward Validation starting...", cfg.Interval)
	
	// Load data for walk-forward validation
	data, err := datamanager.LoadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		return fmt.Errorf("failed to load data for walk-forward validation: %w", err)
	}
	
	if selectedPeriod > 0 {
		data = datamanager.FilterDataByPeriod(data, selectedPeriod)
	}
	
	// Run walk-forward validation in quiet mode
	summary, err := validation.RunQuietWalkForwardValidation(cfg, data, *wfConfig,
		func(config interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error) {
			// Quiet optimization for intervals
			return optimization.OptimizeWithGA(config, cfg.DataFile, 0)
		},
		func(cfg interface{}, data []types.OHLCV) *backtest.BacktestResults {
			// Quiet testing for intervals
			dcaConfig := cfg.(*config.DCAConfig)
			results, err := r.backtestRunner.RunWithData(dcaConfig, data)
			if err != nil {
				return nil
			}
			return results
		})
		
	if err != nil {
		return fmt.Errorf("walk-forward validation failed for interval %s: %w", cfg.Interval, err)
	}
	
	// Display clean interval-specific summary
	r.printCleanIntervalWalkForwardSummary(cfg.Interval, summary)
	
	return nil
}

// fetchAndSetMinOrderQtyCached fetches minimum order quantity with caching to avoid redundant API calls
func (r *DefaultIntervalRunner) fetchAndSetMinOrderQtyCached(cfg *config.DCAConfig) error {
	// Create cache key
	cacheKey := cfg.Symbol
	
	// Check cache first (read lock)
	r.cacheMutex.RLock()
	if minQty, exists := r.minQtyCache[cacheKey]; exists {
		r.cacheMutex.RUnlock()
		cfg.MinOrderQty = minQty
		log.Printf("ℹ️ Using cached minimum order quantity for %s: %.6f", cfg.Symbol, minQty)
		return nil
	}
	r.cacheMutex.RUnlock()
	
	// If not in cache, fetch once and cache it (write lock)
	r.cacheMutex.Lock()
	defer r.cacheMutex.Unlock()
	
	// Double-check in case another goroutine filled the cache
	if minQty, exists := r.minQtyCache[cacheKey]; exists {
		cfg.MinOrderQty = minQty
		log.Printf("ℹ️ Using cached minimum order quantity for %s: %.6f", cfg.Symbol, minQty)
		return nil
	}
	
	// Fetch from API (this is the expensive call)
	runner := r.backtestRunner.(*DefaultBacktestRunner)
	if err := runner.fetchAndSetMinOrderQty(cfg); err != nil {
		return err
	}
	
	// Cache the result
	r.minQtyCache[cacheKey] = cfg.MinOrderQty
	log.Printf("✅ Fetched and cached minimum order quantity for %s: %.6f", cfg.Symbol, cfg.MinOrderQty)
	
	return nil
}

// printCleanIntervalWalkForwardSummary displays a concise walk-forward validation summary for a specific interval
func (r *DefaultIntervalRunner) printCleanIntervalWalkForwardSummary(interval string, summary *validation.WalkForwardSummary) {
	// Quick assessment
	assessment := "🚨"
	if summary.ReturnDegradation < 0.05 {
		assessment = "✅"
	} else if summary.ReturnDegradation < 0.15 {
		assessment = "⚠️"
	}
	
	profitable := ""
	if summary.AverageTestReturn > 0 {
		profitable = "Profitable"
	} else {
		profitable = "Unprofitable"
	}
	
	log.Printf("   [%s] Train=%.1f%%, Test=%.1f%%, Degradation=%.1f%% | %s %s", 
		strings.ToUpper(interval),
		summary.AverageTrainReturn*100, 
		summary.AverageTestReturn*100, 
		summary.ReturnDegradation*100,
		assessment,
		profitable)
}

// Helper functions removed - dataRoot and exchange are now passed directly
