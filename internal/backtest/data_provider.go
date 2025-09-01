package backtest

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DataProvider interface for loading historical data
type DataProvider interface {
	LoadData(symbol, interval string, params map[string]interface{}) ([]types.OHLCV, error)
	GetCacheStats() CacheStats
	ClearCache()
}

// CacheStats provides cache performance metrics
type CacheStats struct {
	HitCount     int64
	MissCount    int64
	CacheSize    int
	MemoryUsage  int64 // Estimated memory usage in bytes
}

// CSVDataProvider implements DataProvider for CSV files
type CSVDataProvider struct {
	cache       map[string][]types.OHLCV
	cacheMutex  sync.RWMutex
	hitCount    int64
	missCount   int64
	maxCacheSize int
	columnMapping CSVColumnMapping
}

// CSVColumnMapping defines the column positions for different CSV formats
type CSVColumnMapping struct {
	TimestampCol int
	OpenCol      int
	HighCol      int
	LowCol       int
	CloseCol     int
	VolumeCol    int
	MinColumns   int
	DateFormat   string
}

// DefaultCSVMapping provides a standard CSV column mapping
var DefaultCSVMapping = CSVColumnMapping{
	TimestampCol: 0,
	OpenCol:      1,
	HighCol:      2,
	LowCol:       3,
	CloseCol:     4,
	VolumeCol:    5,
	MinColumns:   6,
	DateFormat:   "2006-01-02 15:04:05",
}

// NewCSVDataProvider creates a new CSV data provider with caching
func NewCSVDataProvider(maxCacheSize int) *CSVDataProvider {
	return &CSVDataProvider{
		cache:         make(map[string][]types.OHLCV),
		maxCacheSize:  maxCacheSize,
		columnMapping: DefaultCSVMapping,
	}
}

// SetColumnMapping configures the CSV column mapping
func (p *CSVDataProvider) SetColumnMapping(mapping CSVColumnMapping) {
	p.columnMapping = mapping
}

// LoadData loads OHLCV data from a CSV file with caching
func (p *CSVDataProvider) LoadData(symbol, interval string, params map[string]interface{}) ([]types.OHLCV, error) {
	// Extract file path from params
	filePath, ok := params["file_path"].(string)
	if !ok {
		return nil, fmt.Errorf("file_path parameter required")
	}

	// Create cache key
	cacheKey := fmt.Sprintf("%s:%s:%s", symbol, interval, filePath)

	// Check cache first
	p.cacheMutex.RLock()
	if data, exists := p.cache[cacheKey]; exists {
		p.cacheMutex.RUnlock()
		p.hitCount++
		return data, nil
	}
	p.cacheMutex.RUnlock()

	// Cache miss - load from file
	p.missCount++
	data, err := p.loadFromFile(filePath)
	if err != nil {
		return nil, err
	}

	// Store in cache (with eviction if needed)
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	// Simple LRU: if cache is full, remove oldest entry (first in map)
	if len(p.cache) >= p.maxCacheSize {
		// Remove one entry (Go maps have undefined iteration order, so this is pseudo-LRU)
		for k := range p.cache {
			delete(p.cache, k)
			break
		}
	}

	p.cache[cacheKey] = data
	return data, nil
}

// loadFromFile loads OHLCV data from a CSV file
func (p *CSVDataProvider) loadFromFile(filename string) ([]types.OHLCV, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %v", filename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, fmt.Errorf("failed to read header: %v", err)
	}

	var data []types.OHLCV
	lineNum := 1

	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading CSV at line %d: %v", lineNum, err)
		}
		lineNum++

		// Check minimum columns based on format
		if len(record) < p.columnMapping.MinColumns {
			continue // Skip invalid rows
		}

		// Parse timestamp with configurable format
		timestamp, err := time.Parse(p.columnMapping.DateFormat, record[p.columnMapping.TimestampCol])
		if err != nil {
			continue // Skip invalid timestamps
		}

		// Parse price data using configurable columns
		open, err := strconv.ParseFloat(record[p.columnMapping.OpenCol], 64)
		if err != nil {
			continue
		}

		high, err := strconv.ParseFloat(record[p.columnMapping.HighCol], 64)
		if err != nil {
			continue
		}

		low, err := strconv.ParseFloat(record[p.columnMapping.LowCol], 64)
		if err != nil {
			continue
		}

		close, err := strconv.ParseFloat(record[p.columnMapping.CloseCol], 64)
		if err != nil {
			continue
		}

		volume, err := strconv.ParseFloat(record[p.columnMapping.VolumeCol], 64)
		if err != nil {
			continue
		}

		// Basic data validation
		if open <= 0 || high <= 0 || low <= 0 || close <= 0 {
			continue
		}

		if high < open || high < close || high < low {
			continue
		}

		if low > open || low > close || low > high {
			continue
		}

		data = append(data, types.OHLCV{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}

	return data, nil
}

// GetCacheStats returns cache performance statistics
func (p *CSVDataProvider) GetCacheStats() CacheStats {
	p.cacheMutex.RLock()
	defer p.cacheMutex.RUnlock()

	// Estimate memory usage (rough calculation)
	var memoryUsage int64
	for _, data := range p.cache {
		memoryUsage += int64(len(data) * 64) // Rough estimate: 64 bytes per OHLCV
	}

	return CacheStats{
		HitCount:    p.hitCount,
		MissCount:   p.missCount,
		CacheSize:   len(p.cache),
		MemoryUsage: memoryUsage,
	}
}

// ClearCache clears all cached data
func (p *CSVDataProvider) ClearCache() {
	p.cacheMutex.Lock()
	defer p.cacheMutex.Unlock()

	p.cache = make(map[string][]types.OHLCV)
	p.hitCount = 0
	p.missCount = 0
}

// GetCacheHitRatio returns the cache hit ratio as a percentage
func (p *CSVDataProvider) GetCacheHitRatio() float64 {
	total := p.hitCount + p.missCount
	if total == 0 {
		return 0
	}
	return float64(p.hitCount) / float64(total) * 100
}
