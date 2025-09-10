package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DCAConfigManager implements ConfigManager for DCA configurations
type DCAConfigManager struct {
	validator Validator
}

// NewDCAConfigManager creates a new DCA configuration manager
func NewDCAConfigManager() *DCAConfigManager {
	return &DCAConfigManager{
		validator: NewDCAValidator(),
	}
}

// LoadConfig loads configuration from file and command line parameters
func (m *DCAConfigManager) LoadConfig(configFile, dataFile, symbol string, balance, commission float64,
	windowSize int, params map[string]interface{}) (Config, error) {
	
	// Start with default DCA configuration
	cfg := NewDefaultDCAConfig()
	
	// Set basic parameters (config file will override these if provided)
	cfg.Symbol = symbol
	cfg.InitialBalance = balance
	cfg.Commission = commission
	cfg.WindowSize = windowSize
	
	// Only set DataFile from command line if not loading from config file
	// This allows config file to override with the correct data source
	if configFile == "" {
		cfg.DataFile = dataFile
	}
	
	// Extract DCA-specific parameters from params map
	if baseAmount, ok := params["base_amount"].(float64); ok {
		cfg.BaseAmount = baseAmount
	}
	if maxMultiplier, ok := params["max_multiplier"].(float64); ok {
		cfg.MaxMultiplier = maxMultiplier
	}
	if priceThreshold, ok := params["price_threshold"].(float64); ok {
		cfg.PriceThreshold = priceThreshold
	}
	if priceThresholdMultiplier, ok := params["price_threshold_multiplier"].(float64); ok {
		cfg.PriceThresholdMultiplier = priceThresholdMultiplier
	}
	if useAdvancedCombo, ok := params["use_advanced_combo"].(bool); ok {
		cfg.UseAdvancedCombo = useAdvancedCombo
	}
	
	// Load from config file if provided
	if configFile != "" {
		if err := m.loadFromFile(configFile, cfg); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}
	
	// Validate configuration
	if err := m.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return cfg, nil
}

// loadFromFile loads configuration from a JSON file
func (m *DCAConfigManager) loadFromFile(configFile string, cfg *DCAConfig) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("could not read config file: %w", err)
	}
	
	// Try to load as nested config first, then fall back to flat config
	if err := m.loadFromNestedConfig(data, cfg); err != nil {
		// Fall back to flat config loading for backward compatibility
		if err := json.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("could not parse config file as nested or flat format: %w", err)
		}
	}
	
	return nil
}

// loadFromNestedConfig loads configuration from nested JSON format into flat DCAConfig
func (m *DCAConfigManager) loadFromNestedConfig(data []byte, cfg *DCAConfig) error {
	var nestedCfg NestedConfig
	if err := json.Unmarshal(data, &nestedCfg); err != nil {
		return err
	}

	// Map strategy fields
	strategy := nestedCfg.Strategy
	cfg.Symbol = strategy.Symbol
	cfg.DataFile = strategy.DataFile        
	cfg.Interval = strategy.Interval
	cfg.BaseAmount = strategy.BaseAmount
	cfg.MaxMultiplier = strategy.MaxMultiplier
	cfg.PriceThreshold = strategy.PriceThreshold
	cfg.PriceThresholdMultiplier = strategy.PriceThresholdMultiplier
	cfg.WindowSize = strategy.WindowSize
	cfg.TPPercent = strategy.TPPercent
	cfg.UseTPLevels = strategy.UseTPLevels
	cfg.Cycle = strategy.Cycle
	cfg.Indicators = strategy.Indicators
	cfg.UseAdvancedCombo = strategy.UseAdvancedCombo

	// Map indicator-specific configurations
	if strategy.UseAdvancedCombo {
		// Advanced combo parameters
		if strategy.HullMA != nil {
			cfg.HullMAPeriod = strategy.HullMA.Period
		}
		if strategy.MFI != nil {
			cfg.MFIPeriod = strategy.MFI.Period
			cfg.MFIOversold = strategy.MFI.Oversold
			cfg.MFIOverbought = strategy.MFI.Overbought
		}
		if strategy.KeltnerChannels != nil {
			cfg.KeltnerPeriod = strategy.KeltnerChannels.Period
			cfg.KeltnerMultiplier = strategy.KeltnerChannels.Multiplier
		}
		if strategy.WaveTrend != nil {
			cfg.WaveTrendN1 = strategy.WaveTrend.N1
			cfg.WaveTrendN2 = strategy.WaveTrend.N2
			cfg.WaveTrendOverbought = strategy.WaveTrend.Overbought
			cfg.WaveTrendOversold = strategy.WaveTrend.Oversold
		}
	} else {
		// Classic combo parameters
		if strategy.RSI != nil {
			cfg.RSIPeriod = strategy.RSI.Period
			cfg.RSIOversold = strategy.RSI.Oversold
			cfg.RSIOverbought = strategy.RSI.Overbought
		}
		if strategy.MACD != nil {
			cfg.MACDFast = strategy.MACD.FastPeriod
			cfg.MACDSlow = strategy.MACD.SlowPeriod
			cfg.MACDSignal = strategy.MACD.SignalPeriod
		}
		if strategy.BollingerBands != nil {
			cfg.BBPeriod = strategy.BollingerBands.Period
			cfg.BBStdDev = strategy.BollingerBands.StdDev
		}
		if strategy.EMA != nil {
			cfg.EMAPeriod = strategy.EMA.Period
		}
	}

	// Map risk parameters
	if nestedCfg.Risk.InitialBalance > 0 {
		cfg.InitialBalance = nestedCfg.Risk.InitialBalance
	}
	if nestedCfg.Risk.Commission > 0 {
		cfg.Commission = nestedCfg.Risk.Commission
	}
	if nestedCfg.Risk.MinOrderQty > 0 {
		cfg.MinOrderQty = nestedCfg.Risk.MinOrderQty
	}

	return nil
}

// ValidateConfig validates a configuration using the validator
func (m *DCAConfigManager) ValidateConfig(cfg Config) error {
	return m.validator.Validate(cfg)
}

// ConvertToNested converts a flat DCA config to nested format for output
func (m *DCAConfigManager) ConvertToNested(cfg Config) (NestedConfig, error) {
	dcaCfg, ok := cfg.(*DCAConfig)
	if !ok {
		return NestedConfig{}, fmt.Errorf("expected *DCAConfig, got %T", cfg)
	}
	
	// Extract interval from data file path (e.g., "data/bybit/linear/BTCUSDT/5m/candles.csv" -> "5m")
	interval := extractIntervalFromPath(dcaCfg.DataFile)
	if interval == "" {
		interval = "5m" // Default fallback
	}
	
	// Only include the combo that was actually used
	strategyConfig := StrategyConfig{
		Symbol:         dcaCfg.Symbol,
		DataFile:       dcaCfg.DataFile,
		BaseAmount:     dcaCfg.BaseAmount,
		MaxMultiplier:  dcaCfg.MaxMultiplier,
		PriceThreshold: dcaCfg.PriceThreshold,
		PriceThresholdMultiplier: dcaCfg.PriceThresholdMultiplier,
		Interval:       interval,
		WindowSize:     dcaCfg.WindowSize,
		TPPercent:      dcaCfg.TPPercent,
		UseTPLevels:    true, // Always use multi-level TP
		Cycle:          dcaCfg.Cycle,
		Indicators:     dcaCfg.Indicators,
		UseAdvancedCombo:    dcaCfg.UseAdvancedCombo,
	}
	
	// Add combo-specific configurations based on what was used
	if dcaCfg.UseAdvancedCombo {
		// Only include advanced combo parameters
		strategyConfig.HullMA = &HullMAConfig{
			Period: dcaCfg.HullMAPeriod,
		}
		strategyConfig.MFI = &MFIConfig{
			Period:     dcaCfg.MFIPeriod,
			Oversold:   dcaCfg.MFIOversold,
			Overbought: dcaCfg.MFIOverbought,
		}
		strategyConfig.KeltnerChannels = &KeltnerChannelsConfig{
			Period:     dcaCfg.KeltnerPeriod,
			Multiplier: dcaCfg.KeltnerMultiplier,
		}
		strategyConfig.WaveTrend = &WaveTrendConfig{
			N1:          dcaCfg.WaveTrendN1,
			N2:          dcaCfg.WaveTrendN2,
			Overbought:  dcaCfg.WaveTrendOverbought,
			Oversold:    dcaCfg.WaveTrendOversold,
		}
	} else {
		// Only include classic combo parameters
		strategyConfig.RSI = &RSIConfig{
			Period:     dcaCfg.RSIPeriod,
			Oversold:   dcaCfg.RSIOversold,
			Overbought: dcaCfg.RSIOverbought,
		}
		strategyConfig.MACD = &MACDConfig{
			FastPeriod:   dcaCfg.MACDFast,
			SlowPeriod:   dcaCfg.MACDSlow,
			SignalPeriod: dcaCfg.MACDSignal,
		}
		strategyConfig.BollingerBands = &BollingerBandsConfig{
			Period: dcaCfg.BBPeriod,
			StdDev: dcaCfg.BBStdDev,
		}
		strategyConfig.EMA = &EMAConfig{
			Period: dcaCfg.EMAPeriod,
		}
	}
	
	return NestedConfig{
		Strategy:         strategyConfig,
		Exchange: ExchangeConfig{
			Name: "bybit",
			Bybit: BybitConfig{
				APIKey:    "${BYBIT_API_KEY}",
				APISecret: "${BYBIT_API_SECRET}",
				Testnet:   false,
				Demo:      true,
			},
		},
		Risk: RiskConfig{
			InitialBalance: dcaCfg.InitialBalance,
			Commission:     dcaCfg.Commission,
			MinOrderQty:    dcaCfg.MinOrderQty,
		},
		Notifications: NotificationsConfig{
			Enabled:       false,
			TelegramToken: "${TELEGRAM_TOKEN}",
			TelegramChat:  "${TELEGRAM_CHAT_ID}",
		},
	}, nil
}

// SaveConfig saves configuration to file
func (m *DCAConfigManager) SaveConfig(cfg Config, path string) error {
	nestedCfg, err := m.ConvertToNested(cfg)
	if err != nil {
		return fmt.Errorf("failed to convert config to nested format: %w", err)
	}
	
	data, err := json.MarshalIndent(nestedCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}
	}
	
	return os.WriteFile(path, data, 0644)
}

// extractIntervalFromPath extracts interval from data file path
// Example: "data/bybit/linear/BTCUSDT/5m/candles.csv" -> "5m"
func extractIntervalFromPath(dataPath string) string {
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
