package data

import (
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DataProvider interface for loading historical data from various sources
type DataProvider interface {
	// LoadData loads historical data from the specified source
	LoadData(source string) ([]types.OHLCV, error)
	
	// ValidateData validates the integrity of the loaded data
	ValidateData(data []types.OHLCV) error
	
	// GetName returns the name of the data provider
	GetName() string
}

// DataCache interface for caching loaded data
type DataCache interface {
	// Get retrieves data from cache if available
	Get(key string) ([]types.OHLCV, bool)
	
	// Set stores data in cache
	Set(key string, data []types.OHLCV)
	
	// Clear removes all cached data
	Clear()
	
	// Size returns the number of cached entries
	Size() int
}

// DataFilter interface for filtering and transforming data
type DataFilter interface {
	// FilterByPeriod filters data to the last N period
	FilterByPeriod(data []types.OHLCV, period time.Duration) []types.OHLCV
	
	// FilterByDateRange filters data to a specific date range
	FilterByDateRange(data []types.OHLCV, start, end time.Time) []types.OHLCV
	
	// ValidateTimeSequence ensures data is in chronological order
	ValidateTimeSequence(data []types.OHLCV) error
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

// Predefined CSV formats
var (
	DefaultCSVFormat = CSVColumnMapping{
		TimestampCol: 0,
		OpenCol:      1,
		HighCol:      2,
		LowCol:       3,
		CloseCol:     4,
		VolumeCol:    5,
		MinColumns:   6,
		DateFormat:   "2006-01-02 15:04:05",
	}
	
	BybitCSVFormat = CSVColumnMapping{
		TimestampCol: 0,
		OpenCol:      1,
		HighCol:      2,
		LowCol:       3,
		CloseCol:     4,
		VolumeCol:    5,
		MinColumns:   6,
		DateFormat:   "2006-01-02 15:04:05",
	}
)

// FileLocator interface for finding data files
type FileLocator interface {
	// FindDataFile attempts to locate data files for a specific exchange and symbol
	FindDataFile(dataRoot, exchange, symbol, interval string) string
	
	// ConvertIntervalToMinutes converts interval strings like "5m", "1h", "4h" to minute numbers
	ConvertIntervalToMinutes(interval string) string
}
