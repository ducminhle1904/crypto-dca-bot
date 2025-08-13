package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// BacktestConfig holds all configuration for backtesting
type BacktestConfig struct {
	DataFile       string  `json:"data_file"`
	Symbol         string  `json:"symbol"`
	InitialBalance float64 `json:"initial_balance"`
	Commission     float64 `json:"commission"`
	WindowSize     int     `json:"window_size"`
	
	// DCA Strategy parameters
	BaseAmount     float64 `json:"base_amount"`
	MaxMultiplier  float64 `json:"max_multiplier"`
	PriceThreshold float64 `json:"price_threshold"` // Minimum price drop % for next DCA entry
	
	// Technical indicator parameters
	RSIPeriod      int     `json:"rsi_period"`
	RSIOversold    float64 `json:"rsi_oversold"`
	RSIOverbought  float64 `json:"rsi_overbought"`
	
	MACDFast       int     `json:"macd_fast"`
	MACDSlow       int     `json:"macd_slow"`
	MACDSignal     int     `json:"macd_signal"`
	
	BBPeriod       int     `json:"bb_period"`
	BBStdDev       float64 `json:"bb_std_dev"`
	
	SMAPeriod      int     `json:"sma_period"`
	
	// Indicator inclusion
	Indicators     []string `json:"indicators"`
	
	// Output settings
	OutputFormat   string  `json:"output_format"` // "console", "json", "csv"
	OutputFile     string  `json:"output_file"`
	Verbose        bool    `json:"verbose"`

	// Take-profit configuration
	TPPercent      float64 `json:"tp_percent"`
    Cycle         bool    `json:"cycle"`
}

func loadConfig(configFile, dataFile, symbol string, balance, commission float64,
	windowSize int, baseAmount, maxMultiplier float64, priceThreshold float64, outputFormat, outputFile string, verbose bool) *BacktestConfig {
	
	cfg := &BacktestConfig{
		DataFile:       dataFile,
		Symbol:         symbol,
		InitialBalance: balance,
		Commission:     commission,
		WindowSize:     windowSize,
		BaseAmount:     baseAmount,
		MaxMultiplier:  maxMultiplier,
		PriceThreshold: priceThreshold, // Initialize PriceThreshold
		RSIPeriod:      14,
		RSIOversold:    30,
		RSIOverbought:  70,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		BBPeriod:       20,
		BBStdDev:       2,
		SMAPeriod:      50,
		Indicators:     nil,
		OutputFormat:   outputFormat,
		OutputFile:     outputFile,
		Verbose:        verbose,
		TPPercent:     0, // Default to 0 TP
	}
	
	// Load from config file if provided
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Printf("Warning: Could not read config file: %v", err)
		} else {
			if err := json.Unmarshal(data, cfg); err != nil {
				log.Printf("Warning: Could not parse config file: %v", err)
			}
		}
	}
	
	return cfg
}

func parseIndicatorsList(list string) []string {
	parts := strings.Split(list, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		n := strings.ToLower(strings.TrimSpace(p))
		if n == "rsi" || n == "macd" || n == "bb" || n == "sma" {
			res = append(res, n)
		}
	}
	return res
}

func containsIndicator(indicators []string, name string) bool {
	name = strings.ToLower(name)
	for _, n := range indicators {
		if strings.ToLower(n) == name { return true }
	}
	return false
}

func printBestConfigJSON(cfg BacktestConfig) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(data))
}

func writeBestConfigJSON(cfg BacktestConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil { return err }
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil { return err }
	}
	return os.WriteFile(path, data, 0644)
}

func defaultBestConfigPath(symbol, interval string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	i := strings.ToLower(strings.TrimSpace(interval))
	if s == "" { s = "UNKNOWN" }
	if i == "" { i = "unknown" }
	return filepath.Join("results", fmt.Sprintf("best_%s_%s.json", s, i))
} 