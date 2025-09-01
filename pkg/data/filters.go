package data

import (
	"fmt"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DefaultDataFilter implements DataFilter for common filtering operations
type DefaultDataFilter struct{}

// NewDefaultDataFilter creates a new default data filter
func NewDefaultDataFilter() *DefaultDataFilter {
	return &DefaultDataFilter{}
}

// FilterByPeriod filters data to the last N period
func (f *DefaultDataFilter) FilterByPeriod(data []types.OHLCV, period time.Duration) []types.OHLCV {
	if period <= 0 || len(data) == 0 {
        return data
    }

    // Find the cutoff timestamp (latest timestamp - period)
    latestTime := data[len(data)-1].Timestamp
    cutoffTime := latestTime.Add(-period)

    // Find the starting index where data is within the period
    startIdx := 0
    for i, candle := range data {
        if candle.Timestamp.After(cutoffTime) || candle.Timestamp.Equal(cutoffTime) {
            startIdx = i
            break
        }
    }

    // Return the filtered data in chronological order
    return data[startIdx:]
}

// FilterByDateRange filters data to a specific date range
func (f *DefaultDataFilter) FilterByDateRange(data []types.OHLCV, start, end time.Time) []types.OHLCV {
	if len(data) == 0 {
		return data
	}
	
	var filtered []types.OHLCV
	
	for _, candle := range data {
		if (candle.Timestamp.After(start) || candle.Timestamp.Equal(start)) &&
		   (candle.Timestamp.Before(end) || candle.Timestamp.Equal(end)) {
			filtered = append(filtered, candle)
		}
	}
	
	return filtered
}

// ValidateTimeSequence ensures data is in chronological order
func (f *DefaultDataFilter) ValidateTimeSequence(data []types.OHLCV) error {
	if len(data) <= 1 {
		return nil // Single item or empty is always valid
	}
	
	for i := 1; i < len(data); i++ {
		if data[i].Timestamp.Before(data[i-1].Timestamp) {
			return fmt.Errorf("data not in chronological order at index %d: %s comes after %s", 
				i, data[i].Timestamp.Format(time.RFC3339), data[i-1].Timestamp.Format(time.RFC3339))
		}
		
		// Check for duplicate timestamps
		if data[i].Timestamp.Equal(data[i-1].Timestamp) {
			return fmt.Errorf("duplicate timestamp at index %d: %s", 
				i, data[i].Timestamp.Format(time.RFC3339))
		}
	}
	
	return nil
}

// SortByTimestamp sorts data by timestamp (ascending order)
func (f *DefaultDataFilter) SortByTimestamp(data []types.OHLCV) []types.OHLCV {
	if len(data) <= 1 {
		return data
	}
	
	// Create a copy to avoid modifying original
	sorted := make([]types.OHLCV, len(data))
	copy(sorted, data)
	
	// Simple bubble sort for chronological ordering
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Timestamp.After(sorted[j].Timestamp) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	return sorted
}

// RemoveDuplicates removes duplicate timestamps, keeping the first occurrence
func (f *DefaultDataFilter) RemoveDuplicates(data []types.OHLCV) []types.OHLCV {
	if len(data) <= 1 {
		return data
	}
	
	var filtered []types.OHLCV
	seen := make(map[int64]bool)
	
	for _, candle := range data {
		timestamp := candle.Timestamp.Unix()
		if !seen[timestamp] {
			seen[timestamp] = true
			filtered = append(filtered, candle)
		}
	}
	
	return filtered
}

// FilterOutliers removes price data that seems unrealistic (basic outlier detection)
func (f *DefaultDataFilter) FilterOutliers(data []types.OHLCV, maxPercentChange float64) []types.OHLCV {
	if len(data) <= 1 || maxPercentChange <= 0 {
		return data
	}
	
	var filtered []types.OHLCV
	
	for i, candle := range data {
		if i == 0 {
			// Always include first candle
			filtered = append(filtered, candle)
			continue
		}
		
		prevClose := data[i-1].Close
		currentOpen := candle.Open
		
		// Calculate percentage change from previous close to current open
		percentChange := ((currentOpen - prevClose) / prevClose) * 100
		
		// If change is within acceptable range, include the candle
		if percentChange <= maxPercentChange && percentChange >= -maxPercentChange {
			filtered = append(filtered, candle)
		}
		// Otherwise skip this candle (outlier)
	}
	
	return filtered
}
