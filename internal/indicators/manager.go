package indicators

import (
	"fmt"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// IndicatorResult holds all calculation results for a single indicator
type IndicatorResult struct {
	Value      float64
	ShouldBuy  bool
	ShouldSell bool
	Strength   float64
	Timestamp  time.Time
	Error      error
}

// IndicatorManager efficiently manages multiple indicators with caching
type IndicatorManager struct {
	indicators    []TechnicalIndicator
	cache         map[string]*IndicatorResult
	lastTimestamp time.Time
	mutex         sync.RWMutex // Thread-safe caching
}

// NewIndicatorManager creates a new indicator manager
func NewIndicatorManager(indicators ...TechnicalIndicator) *IndicatorManager {
	return &IndicatorManager{
		indicators: indicators,
		cache:      make(map[string]*IndicatorResult),
	}
}

// AddIndicator adds an indicator to the manager
func (m *IndicatorManager) AddIndicator(indicator TechnicalIndicator) {
	m.indicators = append(m.indicators, indicator)
}

// ProcessCandle efficiently processes all indicators for a single candle
func (m *IndicatorManager) ProcessCandle(candle types.OHLCV, data []types.OHLCV) map[string]*IndicatorResult {
	m.mutex.RLock()
	// Early exit if same timestamp (cache hit)
	if candle.Timestamp.Equal(m.lastTimestamp) && len(m.cache) > 0 {
		cacheCopy := make(map[string]*IndicatorResult, len(m.cache))
		for k, v := range m.cache {
			cacheCopy[k] = v
		}
		m.mutex.RUnlock()
		return cacheCopy
	}
	m.mutex.RUnlock()

	// Process all indicators in batch
	results := make(map[string]*IndicatorResult, len(m.indicators))
	
	for _, indicator := range m.indicators {
		name := indicator.GetName()
		result := &IndicatorResult{Timestamp: candle.Timestamp}
		
		// Skip if insufficient data
		if len(data) < indicator.GetRequiredPeriods() {
			result.Error = NewInsufficientDataError(name, len(data), indicator.GetRequiredPeriods())
			results[name] = result
			continue
		}
		
		// Single calculation per indicator (most expensive operation)
		value, err := indicator.Calculate(data)
		if err != nil {
			result.Error = err
			results[name] = result
			continue
		}
		result.Value = value
		
		// Efficient signal calculation using cached value and current price
		shouldBuy, err := indicator.ShouldBuy(candle.Close, data)
		if err != nil {
			result.Error = err
			results[name] = result
			continue
		}
		result.ShouldBuy = shouldBuy
		
		shouldSell, err := indicator.ShouldSell(candle.Close, data)
		if err != nil {
			result.Error = err
			results[name] = result
			continue
		}
		result.ShouldSell = shouldSell
		
		// Get signal strength (usually lightweight)
		result.Strength = indicator.GetSignalStrength()
		
		results[name] = result
	}
	
	// Update cache atomically
	m.mutex.Lock()
	m.cache = results
	m.lastTimestamp = candle.Timestamp
	m.mutex.Unlock()
	
	return results
}

// GetCachedResults returns cached results without processing
func (m *IndicatorManager) GetCachedResults() map[string]*IndicatorResult {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	if len(m.cache) == 0 {
		return nil
	}
	
	// Return defensive copy
	cacheCopy := make(map[string]*IndicatorResult, len(m.cache))
	for k, v := range m.cache {
		cacheCopy[k] = v
	}
	return cacheCopy
}

// ClearCache clears the internal cache (useful for new time periods)
func (m *IndicatorManager) ClearCache() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.cache = make(map[string]*IndicatorResult)
	m.lastTimestamp = time.Time{}
}

// GetIndicators returns all managed indicators
func (m *IndicatorManager) GetIndicators() []TechnicalIndicator {
	return m.indicators
}

// CountActiveSignals efficiently counts buy/sell signals from cached results
func (m *IndicatorManager) CountActiveSignals(results map[string]*IndicatorResult) (buySignals, sellSignals int, buyStrength, sellStrength float64) {
	for _, result := range results {
		if result.Error != nil {
			continue // Skip failed indicators
		}
		
		if result.ShouldBuy {
			buySignals++
			buyStrength += result.Strength
		} else if result.ShouldSell {
			sellSignals++
			sellStrength += result.Strength
		}
	}
	return
}

// InsufficientDataError represents an error when there's not enough data for calculation
type InsufficientDataError struct {
	Indicator string
	Available int
	Required  int
}

func (e InsufficientDataError) Error() string {
	return fmt.Sprintf("insufficient data for %s: have %d, need %d", 
		e.Indicator, e.Available, e.Required)
}

func NewInsufficientDataError(indicator string, available, required int) *InsufficientDataError {
	return &InsufficientDataError{
		Indicator: indicator,
		Available: available,
		Required:  required,
	}
}

// BatchIndicatorConfig holds configuration for batch processing
type BatchIndicatorConfig struct {
	EnableCaching     bool          // Enable result caching
	CacheTimeout      time.Duration // Cache invalidation timeout
	MaxCacheSize      int           // Maximum cache entries
	EnableParallel    bool          // Enable parallel processing (for CPU-intensive indicators)
}

// DefaultBatchConfig returns sensible default configuration
func DefaultBatchConfig() BatchIndicatorConfig {
	return BatchIndicatorConfig{
		EnableCaching:  true,
		CacheTimeout:   time.Minute,     // Cache for 1 minute
		MaxCacheSize:   1000,            // Reasonable cache size
		EnableParallel: false,           // Usually not needed for single candle
	}
}
