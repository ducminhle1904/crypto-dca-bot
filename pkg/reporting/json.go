package reporting

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultJSONFormatter implements JSON output functionality
type DefaultJSONFormatter struct{}

// NewDefaultJSONFormatter creates a new JSON formatter
func NewDefaultJSONFormatter() *DefaultJSONFormatter {
	return &DefaultJSONFormatter{}
}

// FormatBestConfig formats configuration as JSON bytes
func (f *DefaultJSONFormatter) FormatBestConfig(config interface{}) ([]byte, error) {
	// Convert to nested format
	nestedCfg := f.ConvertToNestedConfig(config)
	
	return json.MarshalIndent(nestedCfg, "", "  ")
}

// PrintBestConfig prints configuration as JSON to console - extracted from main.go printBestConfigJSON
func (f *DefaultJSONFormatter) PrintBestConfig(config interface{}) {
	// Convert to nested format for consistent output
	nestedCfg := f.ConvertToNestedConfig(config)
	data, _ := json.MarshalIndent(nestedCfg, "", "  ")
	fmt.Println(string(data))
}

// ConvertToNestedConfig converts a config to nested format (correct best.json format)
func (f *DefaultJSONFormatter) ConvertToNestedConfig(config interface{}) interface{} {
	// The config should already be in nested format when passed to this function
	// from main.go after conversion, so we just return it as-is
	return config
}

// WriteBestConfigJSON writes configuration to JSON file - extracted from main.go writeBestConfigJSON
func WriteBestConfigJSON(config interface{}, path string) error {
	formatter := NewDefaultJSONFormatter()
	
	// Convert to nested format
	nestedCfg := formatter.ConvertToNestedConfig(config)
	
	data, err := json.MarshalIndent(nestedCfg, "", "  ")
	if err != nil {
		return err
	}
	
	// Ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	
	return os.WriteFile(path, data, 0644)
}

// ExtractIntervalFromPath extracts interval from data file path - extracted from main.go
// Example: "data/bybit/linear/BTCUSDT/5m/candles.csv" -> "5m"
func ExtractIntervalFromPath(dataPath string) string {
	if dataPath == "" {
		return ""
	}
	
	// Normalize path separators
	dataPath = filepath.ToSlash(dataPath)
	parts := strings.Split(dataPath, "/")
	
	// Look for interval pattern (number followed by m,h,d)
	for i := len(parts) - 1; i >= 0; i-- {
		part := parts[i]
		if len(part) >= 2 {
			// Check if it matches interval pattern (e.g., "5m", "1h", "4h", "1d")
			lastChar := part[len(part)-1]
			if lastChar == 'm' || lastChar == 'h' || lastChar == 'd' {
				// Check if the rest is numeric
				numPart := part[:len(part)-1]
				if _, err := strconv.Atoi(numPart); err == nil {
					return part
				}
			}
		}
	}
	
	return ""
}

// Package-level convenience functions

// PrintBestConfigJSON is a convenience function using the default formatter
func PrintBestConfigJSON(config interface{}) {
	formatter := NewDefaultJSONFormatter()
	formatter.PrintBestConfig(config)
}
