package reporting

// Configuration types for JSON output formatting
// NestedConfig represents the new nested configuration format for output
type NestedConfig struct {
	Strategy      StrategyConfig      `json:"strategy"`
	Exchange      ExchangeConfig      `json:"exchange"`
	Risk          RiskConfig          `json:"risk"`
	Notifications NotificationsConfig `json:"notifications"`
}

type StrategyConfig struct {
	Symbol           string                     `json:"symbol"`
	DataFile         string                     `json:"data_file"`           
	BaseAmount       float64                    `json:"base_amount"`
	MaxMultiplier    float64                    `json:"max_multiplier"`
	PriceThreshold   float64                    `json:"price_threshold"`
	PriceThresholdMultiplier float64           `json:"price_threshold_multiplier"`
	Interval         string                     `json:"interval"`
	WindowSize       int                        `json:"window_size"`
	TPPercent        float64                    `json:"tp_percent"`
	UseTPLevels      bool                       `json:"use_tp_levels"`
	Cycle            bool                       `json:"cycle"`
	Indicators       []string                   `json:"indicators"`
	RSI            *RSIConfig                 `json:"rsi,omitempty"`
	MACD           *MACDConfig                `json:"macd,omitempty"`
	BollingerBands *BollingerBandsConfig      `json:"bollinger_bands,omitempty"`
	EMA            *EMAConfig                 `json:"ema,omitempty"`
	HullMA         *HullMAConfig              `json:"hull_ma,omitempty"`
	SuperTrend     *SuperTrendConfig          `json:"supertrend,omitempty"`
	MFI            *MFIConfig                 `json:"mfi,omitempty"`
	KeltnerChannels *KeltnerChannelsConfig    `json:"keltner_channels,omitempty"`
	WaveTrend      *WaveTrendConfig           `json:"wavetrend,omitempty"`
}

type RSIConfig struct {
	Period     int     `json:"period"`
	Oversold   float64 `json:"oversold"`
	Overbought float64 `json:"overbought"`
}

type MACDConfig struct {
	FastPeriod   int `json:"fast_period"`
	SlowPeriod   int `json:"slow_period"`
	SignalPeriod int `json:"signal_period"`
}

type BollingerBandsConfig struct {
	Period int     `json:"period"`
	StdDev float64 `json:"std_dev"`
}

type EMAConfig struct {
	Period int `json:"period"`
}

type ExchangeConfig struct {
	Name  string      `json:"name"`
	Bybit BybitConfig `json:"bybit"`
}

type BybitConfig struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	Testnet   bool   `json:"testnet"`
	Demo      bool   `json:"demo"`
}

type RiskConfig struct {
	InitialBalance float64 `json:"initial_balance"`
	Commission     float64 `json:"commission"`
	MinOrderQty    float64 `json:"min_order_qty"`
}

type NotificationsConfig struct {
	Enabled       bool   `json:"enabled"`
	TelegramToken string `json:"telegram_token"`
	TelegramChat  string `json:"telegram_chat"`
}

type HullMAConfig struct {
	Period int `json:"period"`
}

type SuperTrendConfig struct {
	Period     int     `json:"period"`
	Multiplier float64 `json:"multiplier"`
}

type MFIConfig struct {
	Period     int     `json:"period"`
	Oversold   float64 `json:"oversold"`
	Overbought float64 `json:"overbought"`
}

type KeltnerChannelsConfig struct {
	Period     int     `json:"period"`
	Multiplier float64 `json:"multiplier"`
}

type WaveTrendConfig struct {
	N1         int     `json:"n1"`
	N2         int     `json:"n2"`
	Overbought float64 `json:"overbought"`
	Oversold   float64 `json:"oversold"`
}
