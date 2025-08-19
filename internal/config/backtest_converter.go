package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// BacktestConfig represents the configuration structure expected by the backtest system
type BacktestConfig struct {
	DataFile       string  `json:"data_file"`
	Symbol         string  `json:"symbol"`
	InitialBalance float64 `json:"initial_balance"`
	Commission     float64 `json:"commission"`
	WindowSize     int     `json:"window_size"`
	
	// DCA Strategy parameters
	BaseAmount     float64 `json:"base_amount"`
	MaxMultiplier  float64 `json:"max_multiplier"`
	PriceThreshold float64 `json:"price_threshold"`
	
	// Technical indicator parameters
	RSIPeriod      int     `json:"rsi_period"`
	RSIOversold    float64 `json:"rsi_oversold"`
	RSIOverbought  float64 `json:"rsi_overbought"`
	
	MACDFast       int     `json:"macd_fast"`
	MACDSlow       int     `json:"macd_slow"`
	MACDSignal     int     `json:"macd_signal"`
	
	BBPeriod       int     `json:"bb_period"`
	BBStdDev       float64 `json:"bb_std_dev"`
	
	EMAPeriod      int     `json:"ema_period"`
	
	// Additional backtest-specific parameters
	Indicators     []string `json:"indicators"`
	TPPercent      float64  `json:"tp_percent"`
	Cycle          bool     `json:"cycle"`
}

// ConvertLiveBotConfigToBacktest converts the new LiveBotConfig format to BacktestConfig format
func ConvertLiveBotConfigToBacktest(liveBotConfig *LiveBotConfig, dataFile string) *BacktestConfig {
	if liveBotConfig == nil {
		return nil
	}

	// If no data file specified, try to infer from symbol and interval
	if dataFile == "" {
		dataFile = inferDataFile(liveBotConfig.Strategy.Symbol, liveBotConfig.Strategy.Interval)
	}

	return &BacktestConfig{
		DataFile:       dataFile,
		Symbol:         liveBotConfig.Strategy.Symbol,
		InitialBalance: liveBotConfig.Risk.InitialBalance,
		Commission:     liveBotConfig.Risk.Commission,
		WindowSize:     liveBotConfig.Strategy.WindowSize,
		
		// DCA Strategy parameters
		BaseAmount:     liveBotConfig.Strategy.BaseAmount,
		MaxMultiplier:  liveBotConfig.Strategy.MaxMultiplier,
		PriceThreshold: liveBotConfig.Strategy.PriceThreshold,
		
		// Technical indicator parameters
		RSIPeriod:      liveBotConfig.Strategy.RSI.Period,
		RSIOversold:    liveBotConfig.Strategy.RSI.Oversold,
		RSIOverbought:  liveBotConfig.Strategy.RSI.Overbought,
		
		MACDFast:       liveBotConfig.Strategy.MACD.FastPeriod,
		MACDSlow:       liveBotConfig.Strategy.MACD.SlowPeriod,
		MACDSignal:     liveBotConfig.Strategy.MACD.SignalPeriod,
		
		BBPeriod:       liveBotConfig.Strategy.BollingerBands.Period,
		BBStdDev:       liveBotConfig.Strategy.BollingerBands.StdDev,
		
		EMAPeriod:      liveBotConfig.Strategy.EMA.Period,
		
		// Additional parameters
		Indicators:     liveBotConfig.Strategy.Indicators,
		TPPercent:      liveBotConfig.Strategy.TPPercent,
		Cycle:          liveBotConfig.Strategy.Cycle,
	}
}

// LoadConfigForBacktest loads a config file and converts it to BacktestConfig format
// It auto-detects whether the file is in the new or legacy format
func LoadConfigForBacktest(configFile string, dataFile string) (*BacktestConfig, error) {
	// Try loading as new format first
	liveBotConfig, err := LoadLiveBotConfig(configFile)
	if err == nil {
		// Successfully loaded as new format, convert to backtest format
		backtestConfig := ConvertLiveBotConfigToBacktest(liveBotConfig, dataFile)
		if backtestConfig == nil {
			return nil, fmt.Errorf("failed to convert LiveBotConfig to BacktestConfig")
		}
		return backtestConfig, nil
	}

	// If new format failed, try loading as legacy BacktestConfig format
	data, readErr := readConfigFile(configFile)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configFile, readErr)
	}

	// Try to unmarshal as legacy BacktestConfig
	var backtestConfig BacktestConfig
	if unmarshallErr := unmarshallJSON(data, &backtestConfig); unmarshallErr != nil {
		return nil, fmt.Errorf("config file is neither new LiveBotConfig nor legacy BacktestConfig format: %w", unmarshallErr)
	}

	// Apply data file if provided
	if dataFile != "" {
		backtestConfig.DataFile = dataFile
	}

	// Apply defaults for missing values
	applyBacktestDefaults(&backtestConfig)

	return &backtestConfig, nil
}

// inferDataFile tries to infer the data file path from symbol and interval
func inferDataFile(symbol, interval string) string {
	// Convert symbol to lowercase and create file pattern
	symbolLower := strings.ToLower(symbol)
	
	// Remove USDT/USD suffix if present to get base symbol
	symbolLower = strings.TrimSuffix(symbolLower, "usdt")
	symbolLower = strings.TrimSuffix(symbolLower, "usd")

	// Convert interval to common format
	intervalStr := interval
	switch interval {
	case "1":
		intervalStr = "1m"
	case "3":
		intervalStr = "3m"
	case "5":
		intervalStr = "5m"
	case "15":
		intervalStr = "15m"
	case "30":
		intervalStr = "30m"
	case "60":
		intervalStr = "1h"
	case "240":
		intervalStr = "4h"
	case "D":
		intervalStr = "1d"
	}

	// Create potential file paths
	possiblePaths := []string{
		filepath.Join("data", fmt.Sprintf("%s_%s.csv", symbolLower, intervalStr)),
		filepath.Join("data", fmt.Sprintf("%s_linear_%s.csv", symbolLower, intervalStr)),
		filepath.Join("data", fmt.Sprintf("%s_spot_%s.csv", symbolLower, intervalStr)),
		filepath.Join("data", fmt.Sprintf("%sUSDT_%s.csv", strings.ToUpper(symbolLower), intervalStr)),
		filepath.Join("data", fmt.Sprintf("%sUSD_%s.csv", strings.ToUpper(symbolLower), intervalStr)),
	}

	// Return the first path that exists, or the first one as default
	for _, path := range possiblePaths {
		if fileExists(path) {
			return path
		}
	}

	// Return default pattern if no files found
	return filepath.Join("data", fmt.Sprintf("%s_%s.csv", symbolLower, intervalStr))
}

// applyBacktestDefaults applies default values to backtest config
func applyBacktestDefaults(config *BacktestConfig) {
	if config.InitialBalance == 0 {
		config.InitialBalance = 1000.0
	}
	if config.Commission == 0 {
		config.Commission = 0.001
	}
	if config.WindowSize == 0 {
		config.WindowSize = 100
	}
	if config.BaseAmount == 0 {
		config.BaseAmount = 50.0
	}
	if config.MaxMultiplier == 0 {
		config.MaxMultiplier = 3.0
	}
	if config.PriceThreshold == 0 {
		config.PriceThreshold = 0.05
	}
	if config.TPPercent == 0 {
		config.TPPercent = 0.08
	}
	
	// Indicator defaults
	if config.RSIPeriod == 0 {
		config.RSIPeriod = 14
	}
	if config.RSIOversold == 0 {
		config.RSIOversold = 30
	}
	if config.RSIOverbought == 0 {
		config.RSIOverbought = 70
	}
	if config.MACDFast == 0 {
		config.MACDFast = 12
	}
	if config.MACDSlow == 0 {
		config.MACDSlow = 26
	}
	if config.MACDSignal == 0 {
		config.MACDSignal = 9
	}
	if config.BBPeriod == 0 {
		config.BBPeriod = 20
	}
	if config.BBStdDev == 0 {
		config.BBStdDev = 2.0
	}
	if config.EMAPeriod == 0 {
		config.EMAPeriod = 21
	}
}

// Helper functions to avoid circular imports

func readConfigFile(configFile string) ([]byte, error) {
	// If config file doesn't contain path separators, look in configs/ directory
	if !strings.ContainsAny(configFile, "/\\") {
		configFile = filepath.Join("configs", configFile)
	}

	// Add .json extension if not present
	if !strings.HasSuffix(configFile, ".json") {
		configFile += ".json"
	}

	return os.ReadFile(configFile)
}

func unmarshallJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
