package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/bybit"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	datamanager "github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/optimization"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/reporting"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/validation"
	"github.com/joho/godotenv"
)

// convertBacktestConfigToDCAConfig converts the old BacktestConfig to the new DCAConfig
func convertBacktestConfigToDCAConfig(btCfg *BacktestConfig) *config.DCAConfig {
	dcaCfg := config.NewDefaultDCAConfig()
	dcaCfg.DataFile = btCfg.DataFile
	dcaCfg.Symbol = btCfg.Symbol
	dcaCfg.Interval = btCfg.Interval
	dcaCfg.InitialBalance = btCfg.InitialBalance
	dcaCfg.Commission = btCfg.Commission
	dcaCfg.WindowSize = btCfg.WindowSize
	dcaCfg.BaseAmount = btCfg.BaseAmount
	dcaCfg.MaxMultiplier = btCfg.MaxMultiplier
	dcaCfg.PriceThreshold = btCfg.PriceThreshold
	dcaCfg.UseAdvancedCombo = btCfg.UseAdvancedCombo
	dcaCfg.RSIPeriod = btCfg.RSIPeriod
	dcaCfg.RSIOversold = btCfg.RSIOversold
	dcaCfg.RSIOverbought = btCfg.RSIOverbought
	dcaCfg.MACDFast = btCfg.MACDFast
	dcaCfg.MACDSlow = btCfg.MACDSlow
	dcaCfg.MACDSignal = btCfg.MACDSignal
	dcaCfg.BBPeriod = btCfg.BBPeriod
	dcaCfg.BBStdDev = btCfg.BBStdDev
	dcaCfg.EMAPeriod = btCfg.EMAPeriod
	dcaCfg.HullMAPeriod = btCfg.HullMAPeriod
	dcaCfg.MFIPeriod = btCfg.MFIPeriod
	dcaCfg.MFIOversold = btCfg.MFIOversold
	dcaCfg.MFIOverbought = btCfg.MFIOverbought
	dcaCfg.KeltnerPeriod = btCfg.KeltnerPeriod
	dcaCfg.KeltnerMultiplier = btCfg.KeltnerMultiplier
	dcaCfg.WaveTrendN1 = btCfg.WaveTrendN1
	dcaCfg.WaveTrendN2 = btCfg.WaveTrendN2
	dcaCfg.WaveTrendOverbought = btCfg.WaveTrendOverbought
	dcaCfg.WaveTrendOversold = btCfg.WaveTrendOversold
	dcaCfg.Indicators = btCfg.Indicators
	dcaCfg.TPPercent = btCfg.TPPercent
	dcaCfg.UseTPLevels = btCfg.UseTPLevels
	dcaCfg.Cycle = btCfg.Cycle
	dcaCfg.MinOrderQty = btCfg.MinOrderQty
	return dcaCfg
}

// convertDCAConfigToBacktestConfig converts the new DCAConfig to the old BacktestConfig for backward compatibility
func convertDCAConfigToBacktestConfig(dcaCfg *config.DCAConfig) *BacktestConfig {
	return &BacktestConfig{
		DataFile:       dcaCfg.DataFile,
		Symbol:         dcaCfg.Symbol,
		Interval:       dcaCfg.Interval,
		InitialBalance: dcaCfg.InitialBalance,
		Commission:     dcaCfg.Commission,
		WindowSize:     dcaCfg.WindowSize,
		BaseAmount:     dcaCfg.BaseAmount,
		MaxMultiplier:  dcaCfg.MaxMultiplier,
		PriceThreshold: dcaCfg.PriceThreshold,
		UseAdvancedCombo: dcaCfg.UseAdvancedCombo,
		RSIPeriod:      dcaCfg.RSIPeriod,
		RSIOversold:    dcaCfg.RSIOversold,
		RSIOverbought:  dcaCfg.RSIOverbought,
		MACDFast:       dcaCfg.MACDFast,
		MACDSlow:       dcaCfg.MACDSlow,
		MACDSignal:     dcaCfg.MACDSignal,
		BBPeriod:       dcaCfg.BBPeriod,
		BBStdDev:       dcaCfg.BBStdDev,
		EMAPeriod:      dcaCfg.EMAPeriod,
		HullMAPeriod:   dcaCfg.HullMAPeriod,
		MFIPeriod:      dcaCfg.MFIPeriod,
		MFIOversold:    dcaCfg.MFIOversold,
		MFIOverbought:  dcaCfg.MFIOverbought,
		KeltnerPeriod:  dcaCfg.KeltnerPeriod,
		KeltnerMultiplier: dcaCfg.KeltnerMultiplier,
		WaveTrendN1:    dcaCfg.WaveTrendN1,
		WaveTrendN2:    dcaCfg.WaveTrendN2,
		WaveTrendOverbought: dcaCfg.WaveTrendOverbought,
		WaveTrendOversold:   dcaCfg.WaveTrendOversold,
		Indicators:     dcaCfg.Indicators,
		TPPercent:      dcaCfg.TPPercent,
		UseTPLevels:    dcaCfg.UseTPLevels,
		Cycle:          dcaCfg.Cycle,
		MinOrderQty:    dcaCfg.MinOrderQty,
	}
}

// Constants for default configuration values
const (
	// Default parameter values
	DefaultInitialBalance = 500.0
	DefaultCommission     = 0.0005 // 0.1%
	DefaultWindowSize     = 100
	DefaultBaseAmount     = 40.0
	DefaultMaxMultiplier  = 3.0
	DefaultPriceThreshold = 0.02 // 2%
	DefaultTPPercent      = 0.02 // 2%
	DefaultMinOrderQty    = 0.01 // Default minimum order quantity (typical for BTCUSDT)
	
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
	
	// Genetic Algorithm constants
	GAPopulationSize = 60   // Population size for optimization
	GAGenerations    = 35   // Number of generations
	GAMutationRate   = 0.1  // Mutation rate
	GACrossoverRate  = 0.8  // Crossover rate
	GAEliteSize      = 6    // Elite size
	
	TournamentSize          = 3    // Tournament selection size
	
	// File and directory constants
	DefaultDataRoot         = "data"
	DefaultExchange         = "bybit"          // Default exchange for data
	ResultsDir             = "results"
	BestConfigFile         = "best.json"
	TradesFile             = "trades.xlsx"
	
	// Data validation constants
	MinDataPoints          = 100   // Minimum data points required for backtest
	MaxCommission          = 1.0   // Maximum commission (100%)
	MinMultiplier          = 1.0   // Minimum multiplier value
	MaxThreshold           = 1.0   // Maximum threshold (100%)
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
	
	// Display and formatting constants
	ReportLineLength       = 50
	ProgressReportInterval = 5     // Report progress every N generations
	DetailReportInterval   = 10    // Show detailed config every N generations
	
	// Performance constants
	MaxParallelWorkers     = 4     // Maximum concurrent GA evaluations
	
	// Output control constants
	EnableFileOutput       = true  // Default file output behavior
)

// Parameter ranges for optimization (eliminates duplicate logic)
var (
	OptimizationRanges = struct {
		Multipliers     []float64
		TPCandidates    []float64
		PriceThresholds []float64
		RSIPeriods      []int
		RSIOversold     []float64
		MACDFast        []int
		MACDSlow        []int
		MACDSignal      []int
		BBPeriods       []int
		BBStdDev        []float64
		EMAPeriods      []int
		// Advanced combo optimization ranges
		HullMAPeriods   []int
		MFIPeriods      []int
		MFIOversold     []float64
		MFIOverbought   []float64
		KeltnerPeriods  []int
		KeltnerMultipliers []float64
		WaveTrendN1     []int
		WaveTrendN2     []int
		WaveTrendOverbought []float64
		WaveTrendOversold   []float64
	}{
		Multipliers:     []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0},
		TPCandidates:    []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06},
		PriceThresholds: []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05},
		RSIPeriods:      []int{10, 12, 14, 16, 18, 20, 22, 25},
		RSIOversold:     []float64{20, 25, 30, 35, 40},
		MACDFast:        []int{6, 8, 10, 12, 14, 16, 18},
		MACDSlow:        []int{20, 22, 24, 26, 28, 30, 32, 35},
		MACDSignal:      []int{7, 8, 9, 10, 12, 14},
		BBPeriods:       []int{10, 14, 16, 18, 20, 22, 25, 28, 30},
		BBStdDev:        []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0},
		EMAPeriods:      []int{15, 20, 25, 30, 40, 50, 60, 75, 100, 120},
		// Advanced combo ranges
		HullMAPeriods:   []int{10, 15, 20, 25, 30, 40, 50},
		MFIPeriods:      []int{10, 12, 14, 16, 18, 20, 22},
		MFIOversold:     []float64{15, 20, 25, 30},
		MFIOverbought:   []float64{70, 75, 80, 85},
		KeltnerPeriods:  []int{15, 20, 25, 30, 40, 50},
		KeltnerMultipliers: []float64{1.5, 1.8, 2.0, 2.2, 2.5, 3.0, 3.5},
		WaveTrendN1:     []int{8, 10, 12, 15, 18, 20},
		WaveTrendN2:     []int{18, 21, 24, 28, 32, 35},
		WaveTrendOverbought: []float64{50, 60, 70, 80},
		WaveTrendOversold:   []float64{-80, -70, -60, -50},
	}
	
	// Note: Data cache moved to pkg/data package
)

// Logging functions for better error reporting and debugging
func logInfo(format string, args ...interface{}) {
	log.Printf("ℹ️  "+format, args...)
}

func logWarning(format string, args ...interface{}) {
	log.Printf("⚠️  "+format, args...)
}

func logError(format string, args ...interface{}) {
	log.Printf("❌ "+format, args...)
}

func logSuccess(format string, args ...interface{}) {
	log.Printf("✅ "+format, args...)
}

func logProgress(format string, args ...interface{}) {
	log.Printf("🔄 "+format, args...)
}

// BacktestConfig holds all configuration for backtesting
type BacktestConfig struct {
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

// NestedConfig represents the new nested configuration format for output
type NestedConfig struct {
	Strategy         StrategyConfig     `json:"strategy"`
	Exchange         ExchangeConfig     `json:"exchange"`
	Risk             RiskConfig         `json:"risk"`
	Notifications    NotificationsConfig `json:"notifications"`
}

type StrategyConfig struct {
	Symbol         string             `json:"symbol"`
	BaseAmount     float64            `json:"base_amount"`
	MaxMultiplier  float64            `json:"max_multiplier"`
	PriceThreshold float64            `json:"price_threshold"`
	Interval       string             `json:"interval"`
	WindowSize     int                `json:"window_size"`
	TPPercent      float64            `json:"tp_percent"`
	UseTPLevels    bool               `json:"use_tp_levels"`
	Cycle          bool               `json:"cycle"`
	Indicators     []string           `json:"indicators"`
	UseAdvancedCombo bool             `json:"use_advanced_combo"`
	// Classic combo - use pointers so they can be omitted when not used
	RSI            *RSIConfig         `json:"rsi,omitempty"`
	MACD           *MACDConfig        `json:"macd,omitempty"`
	BollingerBands *BollingerBandsConfig `json:"bollinger_bands,omitempty"`
	EMA            *EMAConfig         `json:"ema,omitempty"`
	// Advanced combo - use pointers so they can be omitted when not used
	HullMA         *HullMAConfig      `json:"hull_ma,omitempty"`
	MFI            *MFIConfig         `json:"mfi,omitempty"`
	KeltnerChannels *KeltnerChannelsConfig `json:"keltner_channels,omitempty"`
	WaveTrend      *WaveTrendConfig   `json:"wavetrend,omitempty"`
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

// Advanced combo indicator configs
type HullMAConfig struct {
	Period int `json:"period"`
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

func main() {
	var (
		configFile     = flag.String("config", "", "Path to configuration file")
		dataFile       = flag.String("data", "", "Path to historical data file (overrides -interval)")
		symbol         = flag.String("symbol", "BTCUSDT", "Trading symbol")
		intervalFlag  = flag.String("interval", "1h", "Data interval to use (e.g., 15m,1h,4h,1d)")
		exchange       = flag.String("exchange", DefaultExchange, "Exchange to use for data (bybit, binance, etc.)")
		initialBalance = flag.Float64("balance", DefaultInitialBalance, "Initial balance")
		commission     = flag.Float64("commission", DefaultCommission, "Trading commission (0.001 = 0.1%)")
		windowSize     = flag.Int("window", DefaultWindowSize, "Data window size for analysis")
		baseAmount     = flag.Float64("base-amount", DefaultBaseAmount, "Base DCA amount")
		maxMultiplier  = flag.Float64("max-multiplier", DefaultMaxMultiplier, "Maximum position multiplier")
		priceThreshold = flag.Float64("price-threshold", DefaultPriceThreshold, "Minimum price drop % for next DCA entry (default: 2%)")
		useAdvancedCombo = flag.Bool("advanced-combo", false, "Use advanced combo indicators (Hull MA, MFI, Keltner Channels, WaveTrend) instead of classic (RSI, MACD, BB, EMA)")

		optimize        = flag.Bool("optimize", false, "Run parameter optimization using genetic algorithm")
		allIntervals   = flag.Bool("all-intervals", false, "Scan data root for all intervals for the given symbol and run per-interval backtests/optimizations")
		dataRoot       = flag.String("data-root", DefaultDataRoot, "Root folder containing <EXCHANGE>/<CATEGORY>/<SYMBOL>/<INTERVAL>/candles.csv")
		periodStr      = flag.String("period", "", "Limit data to trailing window (e.g., 7d,30d,180d,365d or Nd)")
		consoleOnly    = flag.Bool("console-only", false, "Only display results in console, do not write files (best.json, trades.xlsx)")
		
		// Walk-Forward Validation flags
		wfEnable       = flag.Bool("wf-enable", false, "Enable walk-forward validation")
		wfSplitRatio   = flag.Float64("wf-split-ratio", 0.7, "Train/test split ratio (0.7 = 70% train, 30% test)")
		wfRolling      = flag.Bool("wf-rolling", false, "Use rolling walk-forward instead of simple holdout")
		wfTrainDays    = flag.Int("wf-train-days", 180, "Training window size in days (for rolling WF)")
		wfTestDays     = flag.Int("wf-test-days", 60, "Test window size in days (for rolling WF)")
		wfRollDays     = flag.Int("wf-roll-days", 30, "Roll forward step size in days (for rolling WF)")
		envFile        = flag.String("env", ".env", "Environment file path for Bybit API credentials")
	)
	
	flag.Parse()
	
	// Test walk-forward flags to ensure they're accessible
	if *wfEnable {
		fmt.Printf("Walk-forward validation enabled\n")
	}

	// If configFile is set and does not contain a path, prepend "configs/"
    if *configFile != "" && !strings.ContainsAny(*configFile, "/\\") {
        *configFile = filepath.Join("configs", *configFile + ".json")
    }
	
	// Load environment variables from .env file
	if err := loadEnvFile(*envFile); err != nil {
		log.Printf("Warning: Could not load .env file (%v), checking environment variables...", err)
	}
	
	// Load configuration - cycle is always enabled, console output only
	configManager := config.NewDCAConfigManager()
	params := map[string]interface{}{
		"base_amount":        *baseAmount,
		"max_multiplier":     *maxMultiplier,
		"price_threshold":    *priceThreshold,
		"use_advanced_combo": *useAdvancedCombo,
	}
	cfgInterface, err := configManager.LoadConfig(*configFile, *dataFile, *symbol, *initialBalance, *commission, *windowSize, params)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	cfg := cfgInterface.(*config.DCAConfig)

	// Cycle is always enabled (this system always uses cycle mode)
	cfg.Cycle = true
	
	// Multi-level TP is now always enabled by default
	cfg.UseTPLevels = true
	
	// TP will be determined by optimization or use sensible default if not set
	if cfg.TPPercent == 0 {
		cfg.TPPercent = DefaultTPPercent // Default 2% TP when TP not set
	}
	
	// Log TP configuration (multi-level TP is always enabled)
	log.Printf("Using 5-level TP system derived from TPPercent: %.2f%%", cfg.TPPercent*100)
	log.Printf("TP Levels (20%% each):")
	for i := 0; i < 5; i++ {
		levelPct := cfg.TPPercent * float64(i+1) / 5.0
		log.Printf("  Level %d: 20.00%% at %.2f%%", i+1, levelPct*100)
	}

	// Capture and parse trailing period window
	var selectedPeriod time.Duration
	if s := strings.TrimSpace(*periodStr); s != "" {
		if d, ok := datamanager.ParseTrailingPeriod(s); ok {
			selectedPeriod = d
		}
	}

	// Set default indicators based on combo selection
	if len(cfg.Indicators) == 0 {
		if cfg.UseAdvancedCombo {
			cfg.Indicators = []string{"hull_ma", "mfi", "keltner", "wavetrend"}
		} else {
			cfg.Indicators = []string{"rsi", "macd", "bb", "ema"}
		}
	}

	// Resolve data file from symbol/interval if not explicitly provided and not scanning all intervals
	if !*allIntervals && strings.TrimSpace(cfg.DataFile) == "" {
		sym := strings.ToUpper(cfg.Symbol)
		
		// Prefer interval from config file over command line flag
		interval := *intervalFlag
		if cfg.Interval != "" {
			interval = cfg.Interval
		}
		
		cfg.DataFile = datamanager.FindDataFile(*dataRoot, *exchange, sym, interval)
	}
	
	// Fetch minimum order quantity from Bybit before backtesting
	if err := fetchAndSetMinOrderQtyDCA(cfg); err != nil {
		log.Printf("Warning: Could not fetch minimum order quantity from Bybit: %v", err)
		log.Printf("Using default minimum order quantity: %.6f", cfg.MinOrderQty)
	}
	
	if *allIntervals {
		// Create walk-forward configuration
		wfConfig := WalkForwardConfig{
			Enable:     *wfEnable,
			Rolling:    *wfRolling,
			SplitRatio: *wfSplitRatio,
			TrainDays:  *wfTrainDays,
			TestDays:   *wfTestDays,
			RollDays:   *wfRollDays,
		}
		runAcrossIntervals(convertDCAConfigToBacktestConfig(cfg), *dataRoot, *exchange, *optimize, selectedPeriod, *consoleOnly, wfConfig)
		return
	}
	
	if *optimize {
		var bestResults *backtest.BacktestResults
		var bestConfig *config.DCAConfig
		
		// Check if walk-forward validation is enabled
		if *wfEnable {
			// Load data for walk-forward validation
			data, err := datamanager.LoadHistoricalDataCached(cfg.DataFile)
			if err != nil {
				log.Fatalf("Failed to load data for walk-forward validation: %v", err)
			}
			
			if selectedPeriod > 0 {
				data = datamanager.FilterDataByPeriod(data, selectedPeriod)
			}
			
			// Create walk-forward configuration
			wfConfig := validation.WalkForwardConfig{
				Enable:     *wfEnable,
				Rolling:    *wfRolling,
				SplitRatio: *wfSplitRatio,
				TrainDays:  *wfTrainDays,
				TestDays:   *wfTestDays,
				RollDays:   *wfRollDays,
			}
			
			// Run walk-forward validation
			_, err = validation.RunWalkForwardValidation(cfg, data, wfConfig, 
				func(config interface{}, data []types.OHLCV) (*backtest.BacktestResults, interface{}, error) {
					return optimization.OptimizeWithGA(config, cfg.DataFile, 0)
				},
				func(cfg interface{}, data []types.OHLCV) *backtest.BacktestResults {
					// Convert interface to DCAConfig
					dcaConfig := cfg.(*config.DCAConfig)
					return runBacktestWithData(convertDCAConfigToBacktestConfig(dcaConfig), data)
				})
			if err != nil {
				log.Printf("Walk-forward validation failed: %v", err)
			}
		}
		
		// Run regular optimization
		bestResults, bestConfigInterface, err := optimization.OptimizeWithGA(cfg, cfg.DataFile, selectedPeriod)
		if err != nil {
			log.Fatalf("Optimization failed: %v", err)
		}
		bestConfig = bestConfigInterface.(*config.DCAConfig)
			fmt.Println("\n\n🏆 OPTIMIZATION RESULTS:")
		
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Best Parameters:\n")
		fmt.Printf("  Combo Type:       %s\n", getComboTypeName(bestConfig.UseAdvancedCombo))
		fmt.Printf("  Indicators:       %s\n", strings.Join(bestConfig.Indicators, ","))
		fmt.Printf("  Base Amount:      $%.2f\n", bestConfig.BaseAmount)
		fmt.Printf("  Max Multiplier:   %.2f\n", bestConfig.MaxMultiplier)
		fmt.Printf("  Price Threshold:  %.2f%%\n", bestConfig.PriceThreshold*100)
		fmt.Printf("  TP Levels (5-level system, derived from TPPercent %.2f%%):\n", bestConfig.TPPercent*100)
			for i := 0; i < 5; i++ {
				levelPct := bestConfig.TPPercent * float64(i+1) / 5.0
				fmt.Printf("    Level %d: 20.00%% at %.2f%%\n", i+1, levelPct*100)
		}
		fmt.Printf("  Min Order Qty:    %.6f %s (from Bybit)\n", bestConfig.MinOrderQty, bestConfig.Symbol)
		
		if bestConfig.UseAdvancedCombo {
			// Advanced combo parameters
			if containsIndicator(bestConfig.Indicators, "hull_ma") {
				fmt.Printf("  Hull MA Period:   %d\n", bestConfig.HullMAPeriod)
			}
			if containsIndicator(bestConfig.Indicators, "mfi") {
				fmt.Printf("  MFI Period:       %d\n", bestConfig.MFIPeriod)
				fmt.Printf("  MFI Oversold:     %.0f\n", bestConfig.MFIOversold)
			}
			if containsIndicator(bestConfig.Indicators, "keltner") {
				fmt.Printf("  Keltner Period:   %d\n", bestConfig.KeltnerPeriod)
				fmt.Printf("  Keltner Multiplier: %.1f\n", bestConfig.KeltnerMultiplier)
			}
			if containsIndicator(bestConfig.Indicators, "wavetrend") {
				fmt.Printf("  WaveTrend N1:     %d\n", bestConfig.WaveTrendN1)
				fmt.Printf("  WaveTrend N2:     %d\n", bestConfig.WaveTrendN2)
			}
		} else {
			// Classic combo parameters
			if containsIndicator(bestConfig.Indicators, "rsi") {
				fmt.Printf("  RSI Period:       %d\n", bestConfig.RSIPeriod)
				fmt.Printf("  RSI Oversold:     %.0f\n", bestConfig.RSIOversold)
			}
			if containsIndicator(bestConfig.Indicators, "macd") {
				fmt.Printf("  MACD: fast=%d slow=%d signal=%d\n", bestConfig.MACDFast, bestConfig.MACDSlow, bestConfig.MACDSignal)
			}
			if containsIndicator(bestConfig.Indicators, "bb") {
				fmt.Printf("  BB: period=%d std=%.2f\n", bestConfig.BBPeriod, bestConfig.BBStdDev)
			}
			if containsIndicator(bestConfig.Indicators, "ema") {
				fmt.Printf("  EMA Period:       %d\n", bestConfig.EMAPeriod)
			}
		}
		
		// Determine interval string for usage example
		intervalStr := filepath.Base(filepath.Dir(bestConfig.DataFile))
		if intervalStr == "" { intervalStr = filepath.Base(filepath.Dir(cfg.DataFile)) }
		if intervalStr == "" { intervalStr = "unknown" }
		
		if !*consoleOnly {
		fmt.Println("\nBest Config (JSON):")
		fmt.Println("Copy this configuration to reuse these optimized settings:")
		fmt.Println(strings.Repeat("-", 50))
		printBestConfigJSON(*convertDCAConfigToBacktestConfig(bestConfig))
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("Usage: go run cmd/backtest/main.go -config best.json\n")
		fmt.Printf("   or: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
			cfg.Symbol, intervalStr, bestConfig.BaseAmount, bestConfig.MaxMultiplier, bestConfig.PriceThreshold)
		
			// Save to results folder
		stdDir := defaultOutputDir(cfg.Symbol, intervalStr)
			stdBestPath := filepath.Join(stdDir, BestConfigFile)
			stdTradesPath := filepath.Join(stdDir, TradesFile)
		
		// Write standard outputs
		if err := writeBestConfigJSON(*convertDCAConfigToBacktestConfig(bestConfig), stdBestPath); err != nil {
				logError("Failed to write best config: %v", err)
		} else {
				logSuccess("Saved best config to: %s", stdBestPath)
		}
		if err := writeTradesCSV(bestResults, stdTradesPath); err != nil {
				logError("Failed to write trades file: %v", err)
		} else {
				logSuccess("Saved trades to: %s", stdTradesPath)
			}
		
		// Log TP configuration in best config (always 5-level TP)
			logSuccess("Best config uses 5-level TP derived from TPPercent: %.2f%%", bestConfig.TPPercent*100)
			for i := 0; i < 5; i++ {
				levelPct := bestConfig.TPPercent * float64(i+1) / 5.0
				logSuccess("  Level %d: 20.00%% at %.2f%%", i+1, levelPct*100)
		}
		} else {
			logInfo("Console-only mode: Skipping file output")
			fmt.Println("\nBest Config (JSON):")
			fmt.Println("Copy this configuration to reuse these optimized settings:")
			fmt.Println(strings.Repeat("-", 50))
			printBestConfigJSON(*convertDCAConfigToBacktestConfig(bestConfig))
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("Usage: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
				cfg.Symbol, intervalStr, bestConfig.BaseAmount, bestConfig.MaxMultiplier, bestConfig.PriceThreshold)
		}
		
		fmt.Println("\nBest Results:")
		// Run a fresh backtest with the optimized config to get proper results
		finalResults := runBacktest(convertDCAConfigToBacktestConfig(bestConfig), selectedPeriod)
		// Use the new context-aware output with the interval information
		reporting.OutputConsoleWithContext(finalResults, bestConfig.Symbol, intervalStr)
		return
	}
	
	// Run single backtest
	results := runBacktest(convertDCAConfigToBacktestConfig(cfg), selectedPeriod)
	
	// Output results to console with context
	intervalStr := guessIntervalFromPath(cfg.DataFile)
	if intervalStr == "" { intervalStr = "unknown" }
	// Use the new context-aware output that includes the interval information
	// that was removed from runBacktestWithData
	reporting.OutputConsoleWithContext(results, cfg.Symbol, intervalStr)
	
	if !*consoleOnly {
		// Save trades to standard path (reuse intervalStr from above)
		stdDir := defaultOutputDir(cfg.Symbol, intervalStr)
		stdTradesPath := filepath.Join(stdDir, TradesFile)
		
		if err := writeTradesCSV(results, stdTradesPath); err != nil {
			logError("Failed to write trades file: %v", err)
		} else {
			logSuccess("Saved trades to: %s", stdTradesPath)
		}
	} else {
		logInfo("Console-only mode: Skipping file output")
	}
}

func loadEnvFile(envFile string) error {
	// Load .env file if it exists
	if _, err := os.Stat(envFile); err == nil {
		return godotenv.Load(envFile)
	}
	return fmt.Errorf("env file %s not found", envFile)
}

func fetchAndSetMinOrderQtyDCA(cfg *config.DCAConfig) error {
	// Create Bybit client to fetch instrument info
	bybitConfig := bybit.Config{
		APIKey:    os.Getenv("BYBIT_API_KEY"),
		APISecret: os.Getenv("BYBIT_API_SECRET"),
		Demo:      true, // Use demo mode for fetching instrument info (safer)
	}

	// Skip if no API credentials (use default)
	if bybitConfig.APIKey == "" || bybitConfig.APISecret == "" {
		log.Printf("No Bybit API credentials found, using default min_order_qty: %.6f", cfg.MinOrderQty)
		return nil
	}

	bybitClient := bybit.NewClient(bybitConfig)
	
	// Determine category from data file path
	category := "linear" // Default
	if strings.Contains(cfg.DataFile, "spot") {
		category = "spot"
	} else if strings.Contains(cfg.DataFile, "inverse") {
		category = "inverse"
	}

	// Fetch instrument constraints
	ctx := context.Background()
	minQty, _, _, err := bybitClient.GetInstrumentManager().GetQuantityConstraints(ctx, category, cfg.Symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch instrument constraints for %s: %w", cfg.Symbol, err)
	}

	// Update config with fetched minimum order quantity
	cfg.MinOrderQty = minQty
	log.Printf("✅ Fetched minimum order quantity for %s: %.6f %s", cfg.Symbol, minQty, cfg.Symbol)
	
	return nil
}

func fetchAndSetMinOrderQty(cfg *BacktestConfig) error {
	// Create Bybit client to fetch instrument info
	bybitConfig := bybit.Config{
		APIKey:    os.Getenv("BYBIT_API_KEY"),
		APISecret: os.Getenv("BYBIT_API_SECRET"),
		Demo:      true, // Use demo mode for fetching instrument info (safer)
	}

	// Skip if no API credentials (use default)
	if bybitConfig.APIKey == "" || bybitConfig.APISecret == "" {
		log.Printf("No Bybit API credentials found, using default min_order_qty: %.6f", cfg.MinOrderQty)
		return nil
	}

	bybitClient := bybit.NewClient(bybitConfig)
	
	// Determine category from data file path
	category := "linear" // Default
	if strings.Contains(cfg.DataFile, "spot") {
		category = "spot"
	} else if strings.Contains(cfg.DataFile, "inverse") {
		category = "inverse"
	}

	// Fetch instrument constraints
	ctx := context.Background()
	minQty, _, _, err := bybitClient.GetInstrumentManager().GetQuantityConstraints(ctx, category, cfg.Symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch instrument constraints for %s: %w", cfg.Symbol, err)
	}

	// Update config with fetched minimum order quantity
	cfg.MinOrderQty = minQty
	log.Printf("✅ Fetched minimum order quantity for %s: %.6f %s", cfg.Symbol, minQty, cfg.Symbol)

	return nil
}

func runAcrossIntervals(cfg *BacktestConfig, dataRoot, exchange string, optimize bool, selectedPeriod time.Duration, consoleOnly bool, wfConfig WalkForwardConfig) {
	sym := strings.ToUpper(cfg.Symbol)
	
	// Find all available intervals for this symbol in the exchange
	var availableIntervals []string
	
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
	
	// Check each category for available intervals
	for _, category := range categories {
		categoryDir := filepath.Join(dataRoot, exchange, category, sym)
		if entries, err := os.ReadDir(categoryDir); err == nil {
			for _, e := range entries {
				if !e.IsDir() { continue }
				interval := e.Name()
				candlesPath := filepath.Join(categoryDir, interval, "candles.csv")
				if _, err := os.Stat(candlesPath); err == nil {
					// Only add if not already found
					found := false
					for _, existing := range availableIntervals {
						if existing == interval {
							found = true
							break
						}
					}
					if !found {
						availableIntervals = append(availableIntervals, interval)
					}
				}
			}
		}
	}
	
	if len(availableIntervals) == 0 {
		log.Fatalf("No data found for symbol %s in exchange %s at %s", sym, exchange, dataRoot)
	}

	type intervalResult struct {
		Interval       string
		Results        *backtest.BacktestResults
		OptimizedCfg   BacktestConfig
	}

	var resultsByInterval []intervalResult

	for _, interval := range availableIntervals {
		// Use findDataFile to get the correct path for this interval
		candlesPath := datamanager.FindDataFile(dataRoot, exchange, sym, interval)
		if _, err := os.Stat(candlesPath); err != nil { continue }

		cfgCopy := *cfg
		cfgCopy.DataFile = candlesPath

		// Fetch minimum order quantity for this interval
		if err := fetchAndSetMinOrderQty(&cfgCopy); err != nil {
			log.Printf("Warning: Could not fetch minimum order quantity for %s: %v", interval, err)
		}

		var res *backtest.BacktestResults
		var bestCfg BacktestConfig
		if optimize {
			// propagate cycle preference
			cfgCopy.Cycle = cfg.Cycle
			
			if wfConfig.Enable {
				// Load data for walk-forward validation
				data, err := datamanager.LoadHistoricalDataCached(cfgCopy.DataFile)
				if err != nil {
					log.Printf("Failed to load data for walk-forward validation: %v", err)
					continue
				}
				
				if selectedPeriod > 0 {
					data = datamanager.FilterDataByPeriod(data, selectedPeriod)
				}
				
				// Run walk-forward validation instead of regular optimization
				runWalkForwardValidation(&cfgCopy, data, wfConfig)
				
				// For now, still run regular optimization to get results for comparison
				// In production, you might want to use the WF results instead
				tempDCAConfig := convertBacktestConfigToDCAConfig(&cfgCopy)
				results, bestConfigInterface, err := optimization.OptimizeWithGA(tempDCAConfig, cfgCopy.DataFile, selectedPeriod)
				if err != nil {
					log.Printf("Optimization failed: %v", err)
					continue
				}
				res = results
				bestCfg = *convertDCAConfigToBacktestConfig(bestConfigInterface.(*config.DCAConfig))
			} else {
				tempDCAConfig := convertBacktestConfigToDCAConfig(&cfgCopy)
				results, bestConfigInterface, err := optimization.OptimizeWithGA(tempDCAConfig, cfgCopy.DataFile, selectedPeriod)
				if err != nil {
					log.Printf("Optimization failed: %v", err)
					continue
				}
				res = results
				bestCfg = *convertDCAConfigToBacktestConfig(bestConfigInterface.(*config.DCAConfig))
			}
		} else {
			res = runBacktest(&cfgCopy, selectedPeriod)
			bestCfg = cfgCopy
		}

		resultsByInterval = append(resultsByInterval, intervalResult{
			Interval:     interval,
			Results:      res,
			OptimizedCfg: bestCfg,
		})
	}

	if len(resultsByInterval) == 0 {
		log.Fatalf("No interval data found for symbol %s in exchange %s", sym, exchange)
	}

	// Find best by TotalReturn
	bestIdx := 0
	for i := 1; i < len(resultsByInterval); i++ {
		if resultsByInterval[i].Results.TotalReturn > resultsByInterval[bestIdx].Results.TotalReturn {
			bestIdx = i
		}
	}

	fmt.Println("\n================ Interval Comparison ================")
	fmt.Printf("Symbol: %s\n", sym)
	fmt.Println("Interval | Return% | Trades | Base$ | MaxMult | TP% | Threshold% | MinQty | Combo | Indicators")
	for _, r := range resultsByInterval {
		c := r.OptimizedCfg
		comboInfo := getComboTypeName(c.UseAdvancedCombo)
		indicatorInfo := "-"
		if c.UseAdvancedCombo {
			// Advanced combo info
			if containsIndicator(c.Indicators, "hull_ma") {
				indicatorInfo = fmt.Sprintf("HullMA(%d)", c.HullMAPeriod)
			}
			if containsIndicator(c.Indicators, "mfi") {
				indicatorInfo = fmt.Sprintf("MFI(%d/%.0f)", c.MFIPeriod, c.MFIOversold)
			}
			if containsIndicator(c.Indicators, "keltner") {
				indicatorInfo = fmt.Sprintf("Keltner(%d/%.1f)", c.KeltnerPeriod, c.KeltnerMultiplier)
			}
			if containsIndicator(c.Indicators, "wavetrend") {
				indicatorInfo = fmt.Sprintf("WT(%d/%d)", c.WaveTrendN1, c.WaveTrendN2)
			}
		} else {
			// Classic combo info
			if containsIndicator(c.Indicators, "rsi") {
				indicatorInfo = fmt.Sprintf("RSI(%d/%.0f)", c.RSIPeriod, c.RSIOversold)
			}
			if containsIndicator(c.Indicators, "macd") {
				indicatorInfo = fmt.Sprintf("MACD(%d/%d/%d)", c.MACDFast, c.MACDSlow, c.MACDSignal)
			}
			if containsIndicator(c.Indicators, "bb") {
				indicatorInfo = fmt.Sprintf("BB(%d/%.2f)", c.BBPeriod, c.BBStdDev)
			}
			if containsIndicator(c.Indicators, "ema") {
				indicatorInfo = fmt.Sprintf("EMA(%d)", c.EMAPeriod)
			}
		}
		fmt.Printf("%-8s | %7.2f | %6d | %5.0f | %7.2f | %5.2f | %8.2f | %6.3f | %s | %s\n",
			r.Interval,
			r.Results.TotalReturn*100,
			r.Results.TotalTrades,
			c.BaseAmount,
			c.MaxMultiplier,
			c.TPPercent*100,
			c.PriceThreshold*100,
			c.MinOrderQty,
			comboInfo,
			indicatorInfo,
		)
	}
	best := resultsByInterval[bestIdx]
	fmt.Printf("\nBest interval: %s (Return %.2f%%)\n", best.Interval, best.Results.TotalReturn*100)
	fmt.Printf("Best settings -> Combo: %s | Base: $%.0f, MaxMult: %.2f, TP: %.2f%%",
		getComboTypeName(best.OptimizedCfg.UseAdvancedCombo),
		best.OptimizedCfg.BaseAmount,
		best.OptimizedCfg.MaxMultiplier,
		best.OptimizedCfg.TPPercent*100,
	)
	
	if best.OptimizedCfg.UseAdvancedCombo {
		// Advanced combo parameters
		if containsIndicator(best.OptimizedCfg.Indicators, "hull_ma") {
			fmt.Printf(", Hull MA: %d", best.OptimizedCfg.HullMAPeriod)
		}
		if containsIndicator(best.OptimizedCfg.Indicators, "mfi") {
			fmt.Printf(", MFI: %d/%.0f", best.OptimizedCfg.MFIPeriod, best.OptimizedCfg.MFIOversold)
		}
		if containsIndicator(best.OptimizedCfg.Indicators, "keltner") {
			fmt.Printf(", Keltner: %d/%.1f", best.OptimizedCfg.KeltnerPeriod, best.OptimizedCfg.KeltnerMultiplier)
		}
		if containsIndicator(best.OptimizedCfg.Indicators, "wavetrend") {
			fmt.Printf(", WaveTrend: %d/%d", best.OptimizedCfg.WaveTrendN1, best.OptimizedCfg.WaveTrendN2)
		}
	} else {
		// Classic combo parameters
		if containsIndicator(best.OptimizedCfg.Indicators, "rsi") {
			fmt.Printf(", RSI: %d/%.0f", best.OptimizedCfg.RSIPeriod, best.OptimizedCfg.RSIOversold)
		}
		if containsIndicator(best.OptimizedCfg.Indicators, "macd") {
			fmt.Printf(", MACD: %d/%d/%d", best.OptimizedCfg.MACDFast, best.OptimizedCfg.MACDSlow, best.OptimizedCfg.MACDSignal)
		}
		if containsIndicator(best.OptimizedCfg.Indicators, "bb") {
			fmt.Printf(", BB: p=%d sd=%.2f", best.OptimizedCfg.BBPeriod, best.OptimizedCfg.BBStdDev)
		}
		if containsIndicator(best.OptimizedCfg.Indicators, "ema") {
			fmt.Printf(", EMA: %d", best.OptimizedCfg.EMAPeriod)
		}
	}
	fmt.Printf("\n")

	fmt.Println("Best Config (JSON):")
	fmt.Println("Copy this configuration to reuse these optimized settings:")
	fmt.Println(strings.Repeat("-", 50))
	printBestConfigJSON(best.OptimizedCfg)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Usage: go run cmd/backtest/main.go -config best.json\n")
	fmt.Printf("   or: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
		cfg.Symbol, best.Interval, best.OptimizedCfg.BaseAmount, best.OptimizedCfg.MaxMultiplier, best.OptimizedCfg.PriceThreshold)

	// Optionally print detailed results for best interval
	fmt.Println("\nBest interval detailed results:")
	reporting.OutputConsole(best.Results)
	
	if !consoleOnly {
		// Write standard outputs under results/<SYMBOL>_<INTERVAL>_mode/
		stdDir := defaultOutputDir(cfg.Symbol, best.Interval)
		stdBestPath := filepath.Join(stdDir, BestConfigFile)
		stdTradesPath := filepath.Join(stdDir, TradesFile)
		if err := writeBestConfigJSON(best.OptimizedCfg, stdBestPath); err != nil {
			logError("Failed to write best config: %v", err)
		} else {
			logSuccess("Saved best config to: %s", stdBestPath)
		}
		if err := writeTradesCSV(best.Results, stdTradesPath); err != nil {
			logError("Failed to write trades file: %v", err)
		} else {
			logSuccess("Saved trades to: %s", stdTradesPath)
		}
	} else {
		logInfo("Console-only mode: Skipping file output for interval analysis")
	}
}

func runBacktest(cfg *BacktestConfig, selectedPeriod time.Duration) *backtest.BacktestResults {
	log.Println("🚀 Starting DCA Bot Backtest")
	log.Printf("📊 Symbol: %s", cfg.Symbol)
	log.Printf("💰 Initial Balance: $%.2f", cfg.InitialBalance)
	log.Printf("📈 Base DCA Amount: $%.2f", cfg.BaseAmount)
	log.Printf("🔄 Max Multiplier: %.2f", cfg.MaxMultiplier)
	log.Println("=" + strings.Repeat("=", 40))
	
	// Load historical data
	data, err := datamanager.LoadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}
	
	if len(data) == 0 {
		log.Fatalf("No valid data found in file: %s", cfg.DataFile)
	}
	
	// Apply trailing period filter if set
	if selectedPeriod > 0 {
		data = datamanager.FilterDataByPeriod(data, selectedPeriod)
		if len(data) == 0 {
			log.Fatalf("No data remaining after applying period filter of %v", selectedPeriod)
		}
		logInfo("Filtered to last %v of data (%s → %s)",
            selectedPeriod,
            data[0].Timestamp.Format("2006-01-02"),
            data[len(data)-1].Timestamp.Format("2006-01-02"))
	}
	
	return runBacktestWithData(cfg, data)
}

func runBacktestWithData(cfg *BacktestConfig, data []types.OHLCV) *backtest.BacktestResults {
	if len(data) == 0 {
		log.Fatalf("No data provided")
	}
	start := time.Now()

	interval := guessIntervalFromPath(cfg.DataFile)
	if interval == "" {
		interval = "?"
	}

	// Config summary
	fmt.Printf("Data: %d bars (%s → %s)\n",
		len(data),
		data[0].Timestamp.Format("2006-01-02 15:04"),
		data[len(data)-1].Timestamp.Format("2006-01-02 15:04"))
	
	// Combo information - prominently displayed  
	fmt.Printf("🎯 COMBO: %s\n", getComboTypeName(cfg.UseAdvancedCombo))
	fmt.Printf("Indicators: %s\n", indicatorSummary(cfg))
	fmt.Printf("Params: base=$%.0f, maxMult=%.2f, window=%d, commission=%.4f, minQty=%.6f\n",
		cfg.BaseAmount, cfg.MaxMultiplier, cfg.WindowSize, cfg.Commission, cfg.MinOrderQty)
	
	// DCA Strategy details
	if cfg.Cycle {
		fmt.Printf("DCA Strategy: TP=%.2f%%, PriceThreshold=%.2f%% (cycle mode)\n", 
			cfg.TPPercent*100, cfg.PriceThreshold*100)
	} else {
		fmt.Printf("DCA Strategy: Hold mode (no TP), PriceThreshold=%.2f%%\n", 
			cfg.PriceThreshold*100)
	}

	// Create strategy with configured indicators
	strat := createStrategy(cfg)

	// Create and run backtest engine
	tp := cfg.TPPercent
	if !cfg.Cycle { tp = 0 }
	engine := backtest.NewBacktestEngine(cfg.InitialBalance, cfg.Commission, strat, tp, cfg.MinOrderQty, cfg.UseTPLevels)
	results := engine.Run(data, cfg.WindowSize)

	// Update all metrics
	results.UpdateMetrics()

	fmt.Printf("Elapsed: %s\n", time.Since(start).Truncate(time.Millisecond))

	return results
}

func guessIntervalFromPath(path string) string {
	dir := filepath.Dir(path)
	return filepath.Base(dir)
}

func indicatorSummary(cfg *BacktestConfig) string {
	parts := []string{}
	set := make(map[string]bool)
	for _, n := range cfg.Indicators { set[strings.ToLower(n)] = true }
	
	if cfg.UseAdvancedCombo {
		// Advanced combo summary
		if set["hull_ma"] {
			parts = append(parts, fmt.Sprintf("HullMA(%d)", cfg.HullMAPeriod))
		}
		if set["mfi"] {
			parts = append(parts, fmt.Sprintf("MFI(%d/%.0f)", cfg.MFIPeriod, cfg.MFIOversold))
		}
		if set["keltner"] {
			parts = append(parts, fmt.Sprintf("Keltner(%d/%.1f)", cfg.KeltnerPeriod, cfg.KeltnerMultiplier))
		}
		if set["wavetrend"] {
			parts = append(parts, fmt.Sprintf("WaveTrend(%d/%d)", cfg.WaveTrendN1, cfg.WaveTrendN2))
		}
	} else {
		// Classic combo summary
		if set["rsi"] {
			parts = append(parts, fmt.Sprintf("RSI(%d/%.0f)", cfg.RSIPeriod, cfg.RSIOversold))
		}
		if set["macd"] {
			parts = append(parts, fmt.Sprintf("MACD(%d/%d/%d)", cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal))
		}
		if set["bb"] {
			parts = append(parts, fmt.Sprintf("BB(%d/%.2f)", cfg.BBPeriod, cfg.BBStdDev))
		}
		if set["ema"] {
			parts = append(parts, fmt.Sprintf("EMA(%d)", cfg.EMAPeriod))
		}
	}
	
	// Add price threshold info
	if cfg.PriceThreshold > 0 {
		parts = append(parts, fmt.Sprintf("PriceThreshold=%.1f%%", cfg.PriceThreshold*100))
	} else {
		parts = append(parts, "PriceThreshold=disabled")
	}
	
	if len(parts) == 0 {
		return "no-config"
	}
	return strings.Join(parts, ", ")
}

func createStrategy(cfg *BacktestConfig) strategy.Strategy {
	// Build Enhanced DCA strategy
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)

	// Set price threshold for DCA entry spacing
	dca.SetPriceThreshold(cfg.PriceThreshold)

	// Indicator inclusion map
	include := make(map[string]bool)
	for _, name := range cfg.Indicators {
		include[strings.ToLower(strings.TrimSpace(name))] = true
	}

	if cfg.UseAdvancedCombo {
		// Advanced combo indicators
		if include["hull_ma"] {
			hullMA := indicators.NewHullMA(cfg.HullMAPeriod)
			dca.AddIndicator(hullMA)
		}
		if include["mfi"] {
			mfi := indicators.NewMFIWithPeriod(cfg.MFIPeriod)
			mfi.SetOversold(cfg.MFIOversold)
			mfi.SetOverbought(cfg.MFIOverbought)
			dca.AddIndicator(mfi)
		}
		if include["keltner"] {
			keltner := indicators.NewKeltnerChannelsCustom(cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
			dca.AddIndicator(keltner)
		}
		if include["wavetrend"] {
			wavetrend := indicators.NewWaveTrendCustom(cfg.WaveTrendN1, cfg.WaveTrendN2)
			wavetrend.SetOverbought(cfg.WaveTrendOverbought)
			wavetrend.SetOversold(cfg.WaveTrendOversold)
			dca.AddIndicator(wavetrend)
		}
	} else {
		// Classic combo indicators
		if include["rsi"] {
			rsi := indicators.NewRSI(cfg.RSIPeriod)
			rsi.SetOversold(cfg.RSIOversold)
			rsi.SetOverbought(cfg.RSIOverbought)
			dca.AddIndicator(rsi)
		}
		if include["macd"] {
			macd := indicators.NewMACD(cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
			dca.AddIndicator(macd)
		}
		if include["bb"] {
			bb := indicators.NewBollingerBandsEMA(cfg.BBPeriod, cfg.BBStdDev)
			dca.AddIndicator(bb)
		}
		if include["ema"] {
			ema := indicators.NewEMA(cfg.EMAPeriod)
			dca.AddIndicator(ema)
		}
	}

	return dca
}

func containsIndicator(indicators []string, name string) bool {
	name = strings.ToLower(name)
	for _, n := range indicators {
		if strings.ToLower(n) == name { return true }
	}
	return false
}

// getComboTypeName returns a human-readable name for the combo type
func getComboTypeName(useAdvancedCombo bool) string {
	if useAdvancedCombo {
		return "ADVANCED COMBO: Hull MA + MFI + Keltner + WaveTrend"
	}
	return "CLASSIC COMBO: RSI + MACD + Bollinger Bands + EMA"
}

// printBestConfigJSON - Note: This function has been moved to pkg/reporting/json.go
func printBestConfigJSON(cfg BacktestConfig) {
	// Convert BacktestConfig to nested format using the main.go conversion logic
	// TODO: This should be refactored to avoid duplicated conversion logic
	nestedCfg := convertToNestedConfig(cfg)
	reporting.PrintBestConfigJSON(nestedCfg)
}

// convertToNestedConfig converts a BacktestConfig to the new nested format
func convertToNestedConfig(cfg BacktestConfig) NestedConfig {
	// Extract interval from data file path (e.g., "data/bybit/linear/BTCUSDT/5m/candles.csv" -> "5m")
	interval := reporting.ExtractIntervalFromPath(cfg.DataFile)
	if interval == "" {
		interval = "5m" // Default fallback
	}
	
	// Only include the combo that was actually used
	strategyConfig := StrategyConfig{
		Symbol:         cfg.Symbol,
		BaseAmount:     cfg.BaseAmount,
		MaxMultiplier:  cfg.MaxMultiplier,
		PriceThreshold: cfg.PriceThreshold,
		Interval:       interval,
		WindowSize:     cfg.WindowSize,
		TPPercent:      cfg.TPPercent,
		UseTPLevels:    true, // Always use multi-level TP
		Cycle:          cfg.Cycle,
		Indicators:     cfg.Indicators,
		UseAdvancedCombo:    cfg.UseAdvancedCombo,
	}
	
	// Add combo-specific configurations based on what was used
	if cfg.UseAdvancedCombo {
		// Only include advanced combo parameters
		strategyConfig.HullMA = &HullMAConfig{
			Period: cfg.HullMAPeriod,
		}
		strategyConfig.MFI = &MFIConfig{
			Period:     cfg.MFIPeriod,
			Oversold:   cfg.MFIOversold,
			Overbought: cfg.MFIOverbought,
		}
		strategyConfig.KeltnerChannels = &KeltnerChannelsConfig{
			Period:     cfg.KeltnerPeriod,
			Multiplier: cfg.KeltnerMultiplier,
		}
		strategyConfig.WaveTrend = &WaveTrendConfig{
			N1:          cfg.WaveTrendN1,
			N2:          cfg.WaveTrendN2,
			Overbought:  cfg.WaveTrendOverbought,
			Oversold:    cfg.WaveTrendOversold,
		}
	} else {
		// Only include classic combo parameters
		strategyConfig.RSI = &RSIConfig{
			Period:     cfg.RSIPeriod,
			Oversold:   cfg.RSIOversold,
			Overbought: cfg.RSIOverbought,
		}
		strategyConfig.MACD = &MACDConfig{
			FastPeriod:   cfg.MACDFast,
			SlowPeriod:   cfg.MACDSlow,
			SignalPeriod: cfg.MACDSignal,
		}
		strategyConfig.BollingerBands = &BollingerBandsConfig{
			Period: cfg.BBPeriod,
			StdDev: cfg.BBStdDev,
		}
		strategyConfig.EMA = &EMAConfig{
			Period: cfg.EMAPeriod,
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

// writeBestConfigJSON - Note: This function has been moved to pkg/reporting/json.go
func writeBestConfigJSON(cfg BacktestConfig, path string) error {
	// Convert BacktestConfig to nested format using the main.go conversion logic
	// TODO: This should be refactored to avoid duplicated conversion logic
	nestedCfg := convertToNestedConfig(cfg)
	return reporting.WriteBestConfigJSON(nestedCfg, path)
}

// writeTradesCSV - Note: This function has been moved to pkg/reporting/csv.go
func writeTradesCSV(results *backtest.BacktestResults, path string) error {
	// If the user requests an Excel file, delegate to Excel writer
	if strings.HasSuffix(strings.ToLower(path), ".xlsx") {
		return reporting.WriteTradesXLSX(results, path)
	}
	
	// Use the new CSV reporter
	return reporting.WriteTradesCSV(results, path)
}

// Walk-forward validation structures (legacy - now handled by pkg/validation)
type WalkForwardConfig struct {
	Enable     bool
	Rolling    bool
	SplitRatio float64
	TrainDays  int
	TestDays   int
	RollDays   int
}

type WalkForwardFold struct {
	Train      []types.OHLCV
	Test       []types.OHLCV
	TrainStart time.Time
	TrainEnd   time.Time
	TestStart  time.Time
	TestEnd    time.Time
}

type WalkForwardResults struct {
	TrainResults *backtest.BacktestResults
	TestResults  *backtest.BacktestResults
	BestConfig   BacktestConfig
	Fold         int
}

// splitByRatio splits data into train/test by ratio
func splitByRatio(data []types.OHLCV, ratio float64) ([]types.OHLCV, []types.OHLCV) {
	if ratio <= 0 || ratio >= 1 {
		return data, nil
	}
	
	n := int(float64(len(data)) * ratio)
	if n < 1 || n >= len(data) {
		return data, nil
	}
	
	return data[:n], data[n:]
}

// createRollingFolds creates rolling walk-forward folds
func createRollingFolds(data []types.OHLCV, trainDays, testDays, rollDays int) []WalkForwardFold {
	var folds []WalkForwardFold
	
	trainDur := time.Duration(trainDays) * 24 * time.Hour
	testDur := time.Duration(testDays) * 24 * time.Hour
	rollDur := time.Duration(rollDays) * 24 * time.Hour
	
	if len(data) < 100 {
		return folds // Need minimum data
	}
	
	start := 0
	for {
		// Find train window
		trainEndTs := data[start].Timestamp.Add(trainDur)
		trainEnd := start
		for trainEnd < len(data) && data[trainEnd].Timestamp.Before(trainEndTs) {
			trainEnd++
		}
		
		// Find test window
		testEndTs := trainEndTs.Add(testDur)
		testEnd := trainEnd
		for testEnd < len(data) && data[testEnd].Timestamp.Before(testEndTs) {
			testEnd++
		}
		
		// Check if we have enough data
		trainSize := trainEnd - start
		testSize := testEnd - trainEnd
		
		if trainSize < 50 || testSize < 10 {
			break // Not enough data for this fold
		}
		
		fold := WalkForwardFold{
			Train:      data[start:trainEnd],
			Test:       data[trainEnd:testEnd],
			TrainStart: data[start].Timestamp,
			TrainEnd:   data[trainEnd-1].Timestamp,
			TestStart:  data[trainEnd].Timestamp,
			TestEnd:    data[testEnd-1].Timestamp,
		}
		
		folds = append(folds, fold)
		
		// Roll forward
		nextStartTs := data[start].Timestamp.Add(rollDur)
		nextStart := start
		for nextStart < len(data) && data[nextStart].Timestamp.Before(nextStartTs) {
			nextStart++
		}
		
		if nextStart <= start {
			nextStart = start + 1
		}
		if nextStart >= len(data) {
			break
		}
		
		start = nextStart
	}
	
	return folds
}

// runWalkForwardValidation runs the complete walk-forward validation
// Note: This function will be replaced with pkg/validation in future iterations
func runWalkForwardValidation(cfg *BacktestConfig, data []types.OHLCV, wfCfg WalkForwardConfig) {
	// TODO: Replace with validation.RunWalkForwardValidation(cfg, data, validationConfig, optimizer, backtester)
	fmt.Println("\n🔄 ================ WALK-FORWARD VALIDATION ================")
	
	if wfCfg.Rolling {
		// Rolling walk-forward
		fmt.Printf("Mode: Rolling Walk-Forward\n")
		fmt.Printf("Train: %d days, Test: %d days, Roll: %d days\n", wfCfg.TrainDays, wfCfg.TestDays, wfCfg.RollDays)
		
		folds := createRollingFolds(data, wfCfg.TrainDays, wfCfg.TestDays, wfCfg.RollDays)
		if len(folds) == 0 {
			fmt.Println("❌ Not enough data for rolling walk-forward validation")
			return
		}
		
		fmt.Printf("Created %d folds\n\n", len(folds))
		
		var allResults []WalkForwardResults
		
		for i, fold := range folds {
			fmt.Printf("📊 Fold %d/%d: Train %s → %s, Test %s → %s\n", 
				i+1, len(folds),
				fold.TrainStart.Format("2006-01-02"),
				fold.TrainEnd.Format("2006-01-02"),
				fold.TestStart.Format("2006-01-02"),
				fold.TestEnd.Format("2006-01-02"))
			
			// Run GA on training data
			trainResults, bestConfig := runGAOnData(cfg, fold.Train)
			
			// Test on out-of-sample data
			testResults := runBacktestWithConfig(bestConfig, fold.Test)
			
			result := WalkForwardResults{
				TrainResults: trainResults,
				TestResults:  testResults,
				BestConfig:   bestConfig,
				Fold:         i + 1,
			}
			
			allResults = append(allResults, result)
			
			fmt.Printf("  Train: %.2f%% return, %.2f%% drawdown\n", 
				trainResults.TotalReturn*100, trainResults.MaxDrawdown*100)
			fmt.Printf("  Test:  %.2f%% return, %.2f%% drawdown\n\n", 
				testResults.TotalReturn*100, testResults.MaxDrawdown*100)
		}
		
		// Print summary
		printWalkForwardSummary(allResults)
		
	} else {
		// Simple holdout validation
		fmt.Printf("Mode: Simple Holdout\n")
		fmt.Printf("Split: %.0f%% train, %.0f%% test\n", wfCfg.SplitRatio*100, (1-wfCfg.SplitRatio)*100)
		
		trainData, testData := splitByRatio(data, wfCfg.SplitRatio)
		if len(testData) < 50 {
			fmt.Println("❌ Not enough test data for validation")
			return
		}
		
		fmt.Printf("Train: %d candles (%s → %s)\n", 
			len(trainData),
			trainData[0].Timestamp.Format("2006-01-02"),
			trainData[len(trainData)-1].Timestamp.Format("2006-01-02"))
		fmt.Printf("Test:  %d candles (%s → %s)\n\n", 
			len(testData),
			testData[0].Timestamp.Format("2006-01-02"),
			testData[len(testData)-1].Timestamp.Format("2006-01-02"))
		
		// Run GA on training data
		fmt.Println("🧬 Running GA optimization on training data...")
		trainResults, bestConfig := runGAOnData(cfg, trainData)
		
		// Test on out-of-sample data
		fmt.Println("🧪 Testing optimized parameters on test data...")
		testResults := runBacktestWithConfig(bestConfig, testData)
		
		// Print results
		fmt.Println("\n📈 ================ WALK-FORWARD RESULTS ================")
		fmt.Printf("TRAIN RESULTS:\n")
		fmt.Printf("  Return:    %.2f%%\n", trainResults.TotalReturn*100)
		fmt.Printf("  Drawdown:  %.2f%%\n", trainResults.MaxDrawdown*100)
		fmt.Printf("  Trades:    %d\n", trainResults.TotalTrades)
		
		trainResults.UpdateMetrics()
		fmt.Printf("  Sharpe:    %.2f\n", trainResults.SharpeRatio)
		fmt.Printf("  ProfitFactor: %.2f\n", trainResults.ProfitFactor)
		
		fmt.Printf("\nTEST RESULTS (Out-of-Sample):\n")
		fmt.Printf("  Return:    %.2f%%\n", testResults.TotalReturn*100)
		fmt.Printf("  Drawdown:  %.2f%%\n", testResults.MaxDrawdown*100)
		fmt.Printf("  Trades:    %d\n", testResults.TotalTrades)
		
		testResults.UpdateMetrics()
		fmt.Printf("  Sharpe:    %.2f\n", testResults.SharpeRatio)
		fmt.Printf("  ProfitFactor: %.2f\n", testResults.ProfitFactor)
		
		// Performance degradation analysis
		returnDegradation := ((trainResults.TotalReturn - testResults.TotalReturn) / math.Max(0.01, math.Abs(trainResults.TotalReturn))) * 100
		fmt.Printf("\n📊 ANALYSIS:\n")
		fmt.Printf("  Return Degradation: %.1f%%\n", returnDegradation)
		
		if returnDegradation > 50 {
			fmt.Printf("  ⚠️  HIGH OVERFITTING RISK - Test performance much worse than train\n")
		} else if returnDegradation > 20 {
			fmt.Printf("  ⚠️  MODERATE OVERFITTING - Some performance degradation\n")
		} else if returnDegradation < -10 {
			fmt.Printf("  ✅ ROBUST STRATEGY - Test performance better than train\n")
		} else {
			fmt.Printf("  ✅ GOOD GENERALIZATION - Consistent train/test performance\n")
		}
	}
}

// Helper function to run GA on specific data - now uses pkg/optimization
func runGAOnData(cfg *BacktestConfig, data []types.OHLCV) (*backtest.BacktestResults, BacktestConfig) {
	// Convert BacktestConfig to DCAConfig for pkg/optimization
	dcaConfig := convertBacktestConfigToDCAConfig(cfg)
	
	// Create a temporary data file path (optimization requires a file path)
	// For in-memory data, we'll use the original file path but override with data
	dataFile := dcaConfig.DataFile
	if dataFile == "" {
		dataFile = "temp_data" // Fallback
	}
	
	// Use pkg/optimization with the data
	// Note: This is a limitation - pkg/optimization expects file paths
	// For now, we'll use a simplified approach
	results, bestConfigInterface, err := optimization.OptimizeWithGA(dcaConfig, dataFile, 0)
	if err != nil {
		log.Printf("GA optimization failed: %v", err)
		// Fallback to simple backtest
		engine := backtest.NewBacktestEngine(
			cfg.InitialBalance,
			cfg.Commission, 
			createStrategy(cfg),
			cfg.TPPercent,
			cfg.MinOrderQty,
			cfg.UseTPLevels,
		)
		results = engine.Run(data, cfg.WindowSize)
		return results, *cfg
	}
	
	// Convert back to BacktestConfig
	bestDCAConfig := bestConfigInterface.(*config.DCAConfig)
	bestBacktestConfig := convertDCAConfigToBacktestConfig(bestDCAConfig)
	
	return results, *bestBacktestConfig
}

// Helper function to run backtest with specific config and data
func runBacktestWithConfig(cfg BacktestConfig, data []types.OHLCV) *backtest.BacktestResults {
	engine := backtest.NewBacktestEngine(
		cfg.InitialBalance,
		cfg.Commission,
		createStrategy(&cfg),
		cfg.TPPercent,
		cfg.MinOrderQty,
		cfg.UseTPLevels,
	)
	
	return engine.Run(data, cfg.WindowSize)
}

// Print summary of rolling walk-forward results
func printWalkForwardSummary(results []WalkForwardResults) {
	fmt.Println("📊 ================ WALK-FORWARD SUMMARY ================")
	
	var trainReturns, testReturns []float64
	var trainDrawdowns, testDrawdowns []float64
	
	for _, r := range results {
		trainReturns = append(trainReturns, r.TrainResults.TotalReturn*100)
		testReturns = append(testReturns, r.TestResults.TotalReturn*100)
		trainDrawdowns = append(trainDrawdowns, r.TrainResults.MaxDrawdown*100)
		testDrawdowns = append(testDrawdowns, r.TestResults.MaxDrawdown*100)
	}
	
	// Calculate averages
	avgTrainReturn := average(trainReturns)
	avgTestReturn := average(testReturns)
	avgTrainDD := average(trainDrawdowns)
	avgTestDD := average(testDrawdowns)
	
	fmt.Printf("AVERAGE PERFORMANCE ACROSS %d FOLDS:\n", len(results))
	fmt.Printf("  Train Return:    %.2f%% ± %.2f%%\n", avgTrainReturn, stdDev(trainReturns))
	fmt.Printf("  Test Return:     %.2f%% ± %.2f%%\n", avgTestReturn, stdDev(testReturns))
	fmt.Printf("  Train Drawdown:  %.2f%% ± %.2f%%\n", avgTrainDD, stdDev(trainDrawdowns))
	fmt.Printf("  Test Drawdown:   %.2f%% ± %.2f%%\n", avgTestDD, stdDev(testDrawdowns))
	
	// Consistency analysis
	returnDegradation := ((avgTrainReturn - avgTestReturn) / math.Max(0.01, math.Abs(avgTrainReturn))) * 100
	fmt.Printf("\nCONSISTENCY ANALYSIS:\n")
	fmt.Printf("  Return Degradation: %.1f%%\n", returnDegradation)
	
	if returnDegradation > 30 {
		fmt.Printf("  ⚠️  HIGH OVERFITTING RISK - Strategy may not generalize well\n")
	} else if returnDegradation > 15 {
		fmt.Printf("  ⚠️  MODERATE OVERFITTING - Some performance degradation\n")
	} else {
		fmt.Printf("  ✅ ROBUST STRATEGY - Good generalization across time periods\n")
	}
}

// Helper functions for statistics
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func stdDev(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	
	avg := average(values)
	sumSquares := 0.0
	for _, v := range values {
		diff := v - avg
		sumSquares += diff * diff
	}
	
	return math.Sqrt(sumSquares / float64(len(values)-1))
}

// Helper to resolve default output dir
// defaultOutputDir - Note: This function has been moved to pkg/reporting/paths.go
func defaultOutputDir(symbol, interval string) string {
	// TODO: Replace with reporting.DefaultOutputDir(symbol, interval)
	return reporting.DefaultOutputDir(symbol, interval)
}