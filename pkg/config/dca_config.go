package config

// DCA-specific configuration constants
const (
	// Default DCA parameter values
	DefaultBaseAmount     = 40.0
	DefaultMaxMultiplier  = 3.0
	DefaultPriceThreshold = 0.02 // 2%
	DefaultTPPercent      = 0.02 // 2%
	
	// Multiple TP configuration
	DefaultUseTPLevels = true            // Default to multi-level TP mode
	DefaultTPLevels    = 5               // Number of TP levels
	DefaultTPQuantity  = 0.20            // 20% per level (1.0 / 5 levels)
	
	// Default indicator parameters
	DefaultRSIPeriod      = 14
	DefaultRSIOversold    = 30
	DefaultRSIOverbought  = 70
	DefaultMACDFast       = 12
	DefaultMACDSlow       = 26
	DefaultMACDSignal     = 9
	DefaultBBPeriod       = 20
	DefaultBBStdDev       = 2.0
	DefaultEMAPeriod      = 50
	
	// Advanced combo indicator parameters
	DefaultHullMAPeriod   = 20
	DefaultMFIPeriod      = 14
	DefaultMFIOversold    = 20
	DefaultMFIOverbought  = 80
	DefaultKeltnerPeriod  = 20
	DefaultKeltnerMultiplier = 2.0
	DefaultWaveTrendN1    = 10
	DefaultWaveTrendN2    = 21
	DefaultWaveTrendOverbought = 60
	DefaultWaveTrendOversold = -60
	
	// Technical indicator validation constants
	MinRSIPeriod           = 2     // Minimum RSI period
	MaxRSIValue            = 100   // Maximum RSI value
	MinMACDPeriod          = 1     // Minimum MACD period
	MinBBPeriod            = 2     // Minimum Bollinger Bands period
	MinEMAPeriod           = 1     // Minimum EMA period
	
	// Advanced combo validation constants
	MinHullMAPeriod        = 2     // Minimum Hull MA period
	MinMFIPeriod           = 2     // Minimum MFI period
	MinKeltnerPeriod       = 2     // Minimum Keltner period
	MinWaveTrendPeriod     = 2     // Minimum WaveTrend period
)

// DCAConfig holds all configuration for DCA backtesting
type DCAConfig struct {
	DataFile       string  `json:"data_file"`
	Symbol         string  `json:"symbol"`
	Interval       string  `json:"interval"`        // Trading interval (5m, 1h, etc.)
	InitialBalance float64 `json:"initial_balance"`
	Commission     float64 `json:"commission"`
	WindowSize     int     `json:"window_size"`
	
	// DCA Strategy parameters
	BaseAmount     float64 `json:"base_amount"`
	MaxMultiplier  float64 `json:"max_multiplier"`
	PriceThreshold float64 `json:"price_threshold"`
	
	// Combo selection  
	UseAdvancedCombo bool  `json:"use_advanced_combo"` // true = advanced combo (Hull MA, MFI, Keltner, WaveTrend), false = classic combo (RSI, MACD, BB, EMA)
	
	// Classic combo indicator parameters
	RSIPeriod      int     `json:"rsi_period"`
	RSIOversold    float64 `json:"rsi_oversold"`
	RSIOverbought  float64 `json:"rsi_overbought"`
	
	MACDFast       int     `json:"macd_fast"`
	MACDSlow       int     `json:"macd_slow"`
	MACDSignal     int     `json:"macd_signal"`
	
	BBPeriod       int     `json:"bb_period"`
	BBStdDev       float64 `json:"bb_std_dev"`
	
	EMAPeriod      int     `json:"ema_period"`
	
	// Advanced combo indicator parameters
	HullMAPeriod   int     `json:"hull_ma_period"`
	MFIPeriod      int     `json:"mfi_period"`
	MFIOversold    float64 `json:"mfi_oversold"`
	MFIOverbought  float64 `json:"mfi_overbought"`
	KeltnerPeriod  int     `json:"keltner_period"`
	KeltnerMultiplier float64 `json:"keltner_multiplier"`
	WaveTrendN1    int     `json:"wavetrend_n1"`
	WaveTrendN2    int     `json:"wavetrend_n2"`
	WaveTrendOverbought float64 `json:"wavetrend_overbought"`
	WaveTrendOversold   float64 `json:"wavetrend_oversold"`
	
	// Indicator inclusion
	Indicators     []string `json:"indicators"`

	// Take-profit configuration
	TPPercent      float64 `json:"tp_percent"`      // Base TP percentage for multi-level TP system
	UseTPLevels    bool    `json:"use_tp_levels"`   // Enable 5-level TP mode
	Cycle          bool    `json:"cycle"`
	
	// Minimum lot size for realistic simulation (e.g., 0.01 for BTCUSDT)
	MinOrderQty    float64 `json:"min_order_qty"`
}

// Implement Config interface
func (c *DCAConfig) GetSymbol() string {
	return c.Symbol
}

func (c *DCAConfig) GetInitialBalance() float64 {
	return c.InitialBalance
}

func (c *DCAConfig) GetCommission() float64 {
	return c.Commission
}

func (c *DCAConfig) GetWindowSize() int {
	return c.WindowSize
}

func (c *DCAConfig) GetMinOrderQty() float64 {
	return c.MinOrderQty
}

func (c *DCAConfig) GetInterval() string {
	return c.Interval
}

func (c *DCAConfig) GetDataFile() string {
	return c.DataFile
}

func (c *DCAConfig) SetDataFile(dataFile string) {
	c.DataFile = dataFile
}

func (c *DCAConfig) GetIndicators() []string {
	return c.Indicators
}

func (c *DCAConfig) SetIndicators(indicators []string) {
	c.Indicators = indicators
}

// NewDefaultDCAConfig creates a new DCA configuration with default values
func NewDefaultDCAConfig() *DCAConfig {
	return &DCAConfig{
		InitialBalance: DefaultInitialBalance,
		Commission:     DefaultCommission,
		WindowSize:     DefaultWindowSize,
		BaseAmount:     DefaultBaseAmount,
		MaxMultiplier:  DefaultMaxMultiplier,
		PriceThreshold: DefaultPriceThreshold,
		UseAdvancedCombo: false,
		// Classic combo defaults
		RSIPeriod:      DefaultRSIPeriod,
		RSIOversold:    DefaultRSIOversold,
		RSIOverbought:  DefaultRSIOverbought,
		MACDFast:       DefaultMACDFast,
		MACDSlow:       DefaultMACDSlow,
		MACDSignal:     DefaultMACDSignal,
		BBPeriod:       DefaultBBPeriod,
		BBStdDev:       DefaultBBStdDev,
		EMAPeriod:      DefaultEMAPeriod,
		// Advanced combo defaults
		HullMAPeriod:   DefaultHullMAPeriod,
		MFIPeriod:      DefaultMFIPeriod,
		MFIOversold:    DefaultMFIOversold,
		MFIOverbought:  DefaultMFIOverbought,
		KeltnerPeriod:  DefaultKeltnerPeriod,
		KeltnerMultiplier: DefaultKeltnerMultiplier,
		WaveTrendN1:    DefaultWaveTrendN1,
		WaveTrendN2:    DefaultWaveTrendN2,
		WaveTrendOverbought: DefaultWaveTrendOverbought,
		WaveTrendOversold:   DefaultWaveTrendOversold,
		TPPercent:      DefaultTPPercent,
		UseTPLevels:    DefaultUseTPLevels,
		MinOrderQty:    DefaultMinOrderQty,
	}
}
