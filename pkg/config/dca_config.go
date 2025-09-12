package config

// DCA-specific configuration constants
const (
	// Default DCA parameter values
	DefaultBaseAmount     = 40.0
	DefaultMaxMultiplier  = 3.0
	DefaultPriceThreshold = 0.02 // 2%
	DefaultPriceThresholdMultiplier = 1.0 // 1.0x = no multiplier (disabled by default)
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
	
	DefaultHullMAPeriod         = 20
	DefaultSuperTrendPeriod     = 14
	DefaultSuperTrendMultiplier = 2.5
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
	MinSuperTrendPeriod    = 2     // Minimum SuperTrend period
	MinMFIPeriod           = 2     // Minimum MFI period
	MinKeltnerPeriod       = 2     // Minimum Keltner period
	MinWaveTrendPeriod     = 2     // Minimum WaveTrend period
)

// DCAConfig holds all configuration for DCA backtesting
type DCAConfig struct {
	DataFile       string  `json:"data_file"`
	Symbol         string  `json:"symbol"`
	Interval       string  `json:"interval"`
	InitialBalance float64 `json:"initial_balance"`
	Commission     float64 `json:"commission"`
	WindowSize     int     `json:"window_size"`
	
	// DCA Strategy parameters
	BaseAmount     float64 `json:"base_amount"`
	MaxMultiplier  float64 `json:"max_multiplier"`
	PriceThreshold float64 `json:"price_threshold"`
	PriceThresholdMultiplier float64 `json:"price_threshold_multiplier"`
	
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
	
	// Indicator inclusion
	Indicators     []string `json:"indicators"`

	// Take-profit configuration
	TPPercent      float64 `json:"tp_percent"`      // Base TP percentage for multi-level TP system
	UseTPLevels    bool    `json:"use_tp_levels"`   // Enable 5-level TP mode
	Cycle          bool    `json:"cycle"`
	
	// Minimum lot size for realistic simulation
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

// Methods required by optimization.BacktestConfig interface
func (c *DCAConfig) GetCycle() bool {
	return c.Cycle
}

func (c *DCAConfig) GetTPPercent() float64 {
	return c.TPPercent
}


// Mutation methods for optimization
func (c *DCAConfig) SetMaxMultiplier(val float64) {
	c.MaxMultiplier = val
}

func (c *DCAConfig) SetTPPercent(val float64) {
	c.TPPercent = val
}

func (c *DCAConfig) SetPriceThreshold(val float64) {
	c.PriceThreshold = val
}

func (c *DCAConfig) SetPriceThresholdMultiplier(val float64) {
	c.PriceThresholdMultiplier = val
}

func (c *DCAConfig) SetHullMAPeriod(val int) {
	c.HullMAPeriod = val
}

func (c *DCAConfig) SetSuperTrendPeriod(val int) {
	c.SuperTrendPeriod = val
}

func (c *DCAConfig) SetSuperTrendMultiplier(val float64) {
	c.SuperTrendMultiplier = val
}

func (c *DCAConfig) SetMFIPeriod(val int) {
	c.MFIPeriod = val
}

func (c *DCAConfig) SetMFIOversold(val float64) {
	c.MFIOversold = val
}

func (c *DCAConfig) SetMFIOverbought(val float64) {
	c.MFIOverbought = val
}

func (c *DCAConfig) SetKeltnerPeriod(val int) {
	c.KeltnerPeriod = val
}

func (c *DCAConfig) SetKeltnerMultiplier(val float64) {
	c.KeltnerMultiplier = val
}

func (c *DCAConfig) SetWaveTrendN1(val int) {
	c.WaveTrendN1 = val
}

func (c *DCAConfig) SetWaveTrendN2(val int) {
	c.WaveTrendN2 = val
}

func (c *DCAConfig) SetWaveTrendOverbought(val float64) {
	c.WaveTrendOverbought = val
}

func (c *DCAConfig) SetWaveTrendOversold(val float64) {
	c.WaveTrendOversold = val
}

func (c *DCAConfig) SetRSIPeriod(val int) {
	c.RSIPeriod = val
}

func (c *DCAConfig) SetRSIOversold(val float64) {
	c.RSIOversold = val
}

func (c *DCAConfig) SetMACDFast(val int) {
	c.MACDFast = val
}

func (c *DCAConfig) SetMACDSlow(val int) {
	c.MACDSlow = val
}

func (c *DCAConfig) SetMACDSignal(val int) {
	c.MACDSignal = val
}

func (c *DCAConfig) SetBBPeriod(val int) {
	c.BBPeriod = val
}

func (c *DCAConfig) SetBBStdDev(val float64) {
	c.BBStdDev = val
}

func (c *DCAConfig) SetEMAPeriod(val int) {
	c.EMAPeriod = val
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
		PriceThresholdMultiplier: DefaultPriceThresholdMultiplier,
		RSIPeriod:      DefaultRSIPeriod,
		RSIOversold:    DefaultRSIOversold,
		RSIOverbought:  DefaultRSIOverbought,
		MACDFast:       DefaultMACDFast,
		MACDSlow:       DefaultMACDSlow,
		MACDSignal:     DefaultMACDSignal,
		BBPeriod:       DefaultBBPeriod,
		BBStdDev:       DefaultBBStdDev,
		EMAPeriod:      DefaultEMAPeriod,
		HullMAPeriod:         DefaultHullMAPeriod,
		SuperTrendPeriod:     DefaultSuperTrendPeriod,
		SuperTrendMultiplier: DefaultSuperTrendMultiplier,
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
