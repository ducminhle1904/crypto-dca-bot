package data

import (
	"strconv"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DataManager combines all data operations in a convenient interface
type DataManager struct {
	provider DataProvider
	filter   DataFilter
	locator  FileLocator
}

// NewDataManager creates a new data manager with default components
func NewDataManager() *DataManager {
	return &DataManager{
		provider: NewCachedProvider(NewCSVProvider()),
		filter:   NewDefaultDataFilter(),
		locator:  NewDefaultFileLocator(),
	}
}

// NewDataManagerWithProvider creates a data manager with a custom provider
func NewDataManagerWithProvider(provider DataProvider) *DataManager {
	return &DataManager{
		provider: provider,
		filter:   NewDefaultDataFilter(),
		locator:  NewDefaultFileLocator(),
	}
}

// LoadHistoricalData loads data from a file - convenience function matching original interface
func (dm *DataManager) LoadHistoricalData(filename string) ([]types.OHLCV, error) {
	return dm.provider.LoadData(filename)
}

// LoadHistoricalDataCached loads data with caching - convenience function matching original interface
func (dm *DataManager) LoadHistoricalDataCached(filename string) ([]types.OHLCV, error) {
	return dm.provider.LoadData(filename)
}

// FilterDataByPeriod filters data by time period - convenience function matching original interface
func (dm *DataManager) FilterDataByPeriod(data []types.OHLCV, period time.Duration) []types.OHLCV {
	return dm.filter.FilterByPeriod(data, period)
}

// FindDataFile locates data files - convenience function matching original interface
func (dm *DataManager) FindDataFile(dataRoot, exchange, symbol, interval string) string {
	return dm.locator.FindDataFile(dataRoot, exchange, symbol, interval)
}

// ConvertIntervalToMinutes converts interval to minutes - convenience function matching original interface
func (dm *DataManager) ConvertIntervalToMinutes(interval string) string {
	return dm.locator.ConvertIntervalToMinutes(interval)
}

// ValidateData validates loaded data
func (dm *DataManager) ValidateData(data []types.OHLCV) error {
	return dm.provider.ValidateData(data)
}

// GetProvider returns the underlying data provider
func (dm *DataManager) GetProvider() DataProvider {
	return dm.provider
}

// GetFilter returns the data filter
func (dm *DataManager) GetFilter() DataFilter {
	return dm.filter
}

// GetLocator returns the file locator
func (dm *DataManager) GetLocator() FileLocator {
	return dm.locator
}

// ParseTrailingPeriod parses period strings like "7d", "30d", "180d" - convenience function matching original interface
func ParseTrailingPeriod(s string) (time.Duration, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasSuffix(s, "days") {
		s = strings.TrimSuffix(s, "days") + "d"
	}
	if strings.HasSuffix(s, "d") {
		nStr := strings.TrimSuffix(s, "d")
		if nStr == "" { return 0, false }
		n, err := strconv.Atoi(nStr)
		if err != nil || n <= 0 { return 0, false }
		return time.Duration(n) * 24 * time.Hour, true
	}
	// allow raw durations too (e.g., 168h)
	if d, err := time.ParseDuration(s); err == nil { return d, true }
	return 0, false
}

// Global convenience functions that match the original main.go interface

// DefaultDataManager provides a global instance for backward compatibility
var DefaultDataManager = NewDataManager()

// LoadHistoricalData - global convenience function
func LoadHistoricalData(filename string) ([]types.OHLCV, error) {
	return DefaultDataManager.LoadHistoricalData(filename)
}

// LoadHistoricalDataCached - global convenience function
func LoadHistoricalDataCached(filename string) ([]types.OHLCV, error) {
	return DefaultDataManager.LoadHistoricalDataCached(filename)
}

// FilterDataByPeriod - global convenience function  
func FilterDataByPeriod(data []types.OHLCV, period time.Duration) []types.OHLCV {
	return DefaultDataManager.FilterDataByPeriod(data, period)
}

// FindDataFile - global convenience function
func FindDataFile(dataRoot, exchange, symbol, interval string) string {
	return DefaultDataManager.FindDataFile(dataRoot, exchange, symbol, interval)
}

// ConvertIntervalToMinutes - global convenience function
func ConvertIntervalToMinutes(interval string) string {
	return DefaultDataManager.ConvertIntervalToMinutes(interval)
}
