package reporting

import (
	"reflect"
	"strings"

	config "github.com/ducminhle1904/crypto-dca-bot/pkg/config"
)

// Configuration conversion functions for JSON output
// Moved from cmd/backtest/main.go to clean up the main file

// MainBacktestConfig represents the flat configuration structure used in main.go
// This is defined here to avoid circular imports when converting to NestedConfig
type MainBacktestConfig struct {
	DataFile       string  `json:"data_file"`
	Symbol         string  `json:"symbol"`
	Interval       string  `json:"interval"`
	InitialBalance float64 `json:"initial_balance"`
	Commission     float64 `json:"commission"`
	WindowSize     int     `json:"window_size"`
	
	// DCA Strategy parameters
	BaseAmount     float64 `json:"base_amount"`
	MaxMultiplier  float64 `json:"max_multiplier"`
	PriceThreshold float64 `json:"price_threshold,omitempty"`
	PriceThresholdMultiplier float64 `json:"price_threshold_multiplier,omitempty"`
	
	DCASpacing     *config.DCASpacingConfig `json:"dca_spacing,omitempty"`
	
	RSIPeriod      int     `json:"rsi_period"`
	RSIOversold    float64 `json:"rsi_oversold"`
	RSIOverbought  float64 `json:"rsi_overbought"`
	
	MACDFast       int     `json:"macd_fast"`
	MACDSlow       int     `json:"macd_slow"`
	MACDSignal     int     `json:"macd_signal"`
	
	BBPeriod       int     `json:"bb_period"`
	BBStdDev       float64 `json:"bb_std_dev"`
	
	EMAPeriod      int     `json:"ema_period"`
	
	HullMAPeriod         int     `json:"hull_ma_period"`
	SuperTrendPeriod     int     `json:"supertrend_period"`
	SuperTrendMultiplier float64 `json:"supertrend_multiplier"`
	MFIPeriod      int     `json:"mfi_period"`
	MFIOversold    float64 `json:"mfi_oversold"`
	MFIOverbought  float64 `json:"mfi_overbought"`
	KeltnerPeriod  int     `json:"keltner_period"`
	KeltnerMultiplier float64 `json:"keltner_multiplier"`
	WaveTrendN1    int     `json:"wavetrend_n1"`
	WaveTrendN2    int     `json:"wavetrend_n2"`
	WaveTrendOverbought float64 `json:"wavetrend_overbought"`
	WaveTrendOversold   float64 `json:"wavetrend_oversold"`
	OBVTrendThreshold   float64 `json:"obv_trend_threshold"`
	
	// Stochastic RSI parameters
	StochasticRSIPeriod     int     `json:"stochastic_rsi_period"`
	StochasticRSIOverbought float64 `json:"stochastic_rsi_overbought"`
	StochasticRSIOversold   float64 `json:"stochastic_rsi_oversold"`
	
	// Indicator inclusion
	Indicators     []string `json:"indicators"`

	// Take-profit configuration
	TPPercent      float64 `json:"tp_percent"`
	UseTPLevels    bool    `json:"use_tp_levels"`
	Cycle          bool    `json:"cycle"`
	
	// Minimum lot size for realistic simulation
	MinOrderQty    float64 `json:"min_order_qty"`
}

// ConvertToNestedConfig converts a MainBacktestConfig to the new nested format
func ConvertToNestedConfig(cfg MainBacktestConfig) NestedConfig {
	interval := ExtractIntervalFromPath(cfg.DataFile)
	if interval == "" {
		interval = "5m" // Default fallback
	}
	
	// Create strategy config with indicators
	strategyConfig := StrategyConfig{
		Symbol:         cfg.Symbol,
		DataFile:       cfg.DataFile,
		BaseAmount:     cfg.BaseAmount,
		MaxMultiplier:  cfg.MaxMultiplier,
		DCASpacing:     cfg.DCASpacing,
		Interval:       interval,
		WindowSize:     cfg.WindowSize,
		TPPercent:      cfg.TPPercent,
		UseTPLevels:    true,
		Cycle:          cfg.Cycle,
		Indicators:     cfg.Indicators,
	}
	
	// Add indicator configs based on which indicators are actually used
	for _, indicator := range cfg.Indicators {
		switch strings.ToLower(indicator) {
		case "rsi":
			strategyConfig.RSI = &RSIConfig{
				Period:     cfg.RSIPeriod,
				Oversold:   cfg.RSIOversold,
				Overbought: cfg.RSIOverbought,
			}
		case "macd":
			strategyConfig.MACD = &MACDConfig{
				FastPeriod:   cfg.MACDFast,
				SlowPeriod:   cfg.MACDSlow,
				SignalPeriod: cfg.MACDSignal,
			}
		case "bb", "bollinger":
			strategyConfig.BollingerBands = &BollingerBandsConfig{
				Period: cfg.BBPeriod,
				StdDev: cfg.BBStdDev,
			}
		case "ema":
			strategyConfig.EMA = &EMAConfig{
				Period: cfg.EMAPeriod,
			}
		case "hullma", "hull_ma":
			strategyConfig.HullMA = &HullMAConfig{
				Period: cfg.HullMAPeriod,
			}
		case "supertrend", "st":
			strategyConfig.SuperTrend = &SuperTrendConfig{
				Period:     cfg.SuperTrendPeriod,
				Multiplier: cfg.SuperTrendMultiplier,
			}
		case "mfi":
			strategyConfig.MFI = &MFIConfig{
				Period:     cfg.MFIPeriod,
				Oversold:   cfg.MFIOversold,
				Overbought: cfg.MFIOverbought,
			}
		case "keltner", "kc":
			strategyConfig.KeltnerChannels = &KeltnerChannelsConfig{
				Period:     cfg.KeltnerPeriod,
				Multiplier: cfg.KeltnerMultiplier,
			}
		case "wavetrend", "wt":
			strategyConfig.WaveTrend = &WaveTrendConfig{
				N1:          cfg.WaveTrendN1,
				N2:          cfg.WaveTrendN2,
				Overbought:  cfg.WaveTrendOverbought,
				Oversold:    cfg.WaveTrendOversold,
			}
		case "obv":
			strategyConfig.OBV = &OBVConfig{
				TrendThreshold: cfg.OBVTrendThreshold,
			}
		case "stochastic_rsi", "stochrsi", "stoch_rsi":
			strategyConfig.StochasticRSI = &StochasticRSIConfig{
				Period:     cfg.StochasticRSIPeriod,
				Overbought: cfg.StochasticRSIOverbought,
				Oversold:   cfg.StochasticRSIOversold,
			}
		}
	}
	
	return NestedConfig{
		Strategy: strategyConfig,
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
			InitialBalance: cfg.InitialBalance,
			Commission:     cfg.Commission,
			MinOrderQty:    cfg.MinOrderQty,
		},
		Notifications: NotificationsConfig{
			Enabled:       false,
			TelegramToken: "${TELEGRAM_TOKEN}",
			TelegramChat:  "${TELEGRAM_CHAT_ID}",
		},
	}
}

// PrintBacktestConfigJSON prints a MainBacktestConfig as nested JSON format
func PrintBacktestConfigJSON(cfg interface{}) {
	// Convert the interface{} to our internal MainBacktestConfig
	// This allows main.go to pass its BacktestConfig without import issues
	mainCfg := convertToMainBacktestConfig(cfg)
	nestedCfg := ConvertToNestedConfig(mainCfg)
	PrintBestConfigJSON(nestedCfg)
}

// WriteBacktestConfigJSON writes a MainBacktestConfig as nested JSON to file
func WriteBacktestConfigJSON(cfg interface{}, path string) error {
	// Convert the interface{} to our internal MainBacktestConfig
	// This allows main.go to pass its BacktestConfig without import issues
	mainCfg := convertToMainBacktestConfig(cfg)
	nestedCfg := ConvertToNestedConfig(mainCfg)
	return WriteBestConfigJSON(nestedCfg, path)
}

// convertToMainBacktestConfig converts an interface{} to MainBacktestConfig
// This uses direct field mapping for better performance (avoiding JSON marshaling overhead)
func convertToMainBacktestConfig(cfg interface{}) MainBacktestConfig {
	// Use type assertion to check if it's already the right type
	if mainCfg, ok := cfg.(MainBacktestConfig); ok {
		return mainCfg
	}
	
	// Use reflection to handle any struct with matching field names
	// This is more efficient than JSON marshaling/unmarshaling
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	
	if v.Kind() != reflect.Struct {
		return MainBacktestConfig{} // Return zero value for non-struct types
	}
	
	result := MainBacktestConfig{}
	resultValue := reflect.ValueOf(&result).Elem()
	resultType := resultValue.Type()
	
	// Map fields by name
	for i := 0; i < resultType.NumField(); i++ {
		field := resultType.Field(i)
		fieldValue := resultValue.Field(i)
		
		if fieldValue.CanSet() {
			// Try to find the same field in the source struct
			sourceField := v.FieldByName(field.Name)
			if sourceField.IsValid() && sourceField.Type() == fieldValue.Type() {
				fieldValue.Set(sourceField)
			}
		}
	}
	
	return result
}
