package data

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultFileLocator implements FileLocator for standard file system operations
type DefaultFileLocator struct{}

// NewDefaultFileLocator creates a new default file locator
func NewDefaultFileLocator() *DefaultFileLocator {
	return &DefaultFileLocator{}
}

// ConvertIntervalToMinutes converts interval strings like "5m", "1h", "4h" to minute numbers
func (f *DefaultFileLocator) ConvertIntervalToMinutes(interval string) string {
	// If it's already just a number, return as-is
	if _, err := strconv.Atoi(interval); err == nil {
		return interval
	}
	
	// Parse interval string
	interval = strings.ToLower(strings.TrimSpace(interval))
	
	// Extract number and unit
	if len(interval) < 2 {
		return interval // Invalid format, return as-is
	}
	
	numStr := interval[:len(interval)-1]
	unit := interval[len(interval)-1:]
	
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return interval // Invalid number, return as-is
	}
	
	// Convert to minutes
	switch unit {
	case "m":
		return strconv.Itoa(num)
	case "h":
		return strconv.Itoa(num * 60)
	case "d":
		return strconv.Itoa(num * 24 * 60)
	case "w":
		return strconv.Itoa(num * 7 * 24 * 60)
	default:
		return interval // Unknown unit, return as-is
	}
}

// FindDataFile attempts to locate data files for a specific exchange
// Structure: data/{exchange}/{category}/{symbol}/{interval}/candles.csv
// Returns empty string if no file is found
func (f *DefaultFileLocator) FindDataFile(dataRoot, exchange, symbol, interval string) string {
	symbol = strings.ToUpper(symbol)
	
	// Convert interval to minutes (5m -> 5, 1h -> 60, etc.)
	intervalMinutes := f.ConvertIntervalToMinutes(interval)
	
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
	
	// Check each category for the exchange
	var attemptedPaths []string
	for _, category := range categories {
		path := filepath.Join(dataRoot, exchange, category, symbol, intervalMinutes, "candles.csv")
		attemptedPaths = append(attemptedPaths, path)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// Log attempted paths for debugging
	log.Printf("⚠️ No data file found for %s %s %s in:", exchange, symbol, interval)
	for _, path := range attemptedPaths {
		log.Printf("   - %s", path)
	}
	
	// Return empty string instead of non-existent path
	return ""
}
