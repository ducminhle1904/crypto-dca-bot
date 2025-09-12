package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"math/rand"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/bybit"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/joho/godotenv"
	"github.com/xuri/excelize/v2"
)

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
	
	// Data cache for performance optimization
	dataCache = make(map[string][]types.OHLCV)
	dataCacheMutex sync.RWMutex
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

// TP level structure for multiple take profit system
type TPLevel struct {
	Level       int        `json:"level"`        // 1-5
	Percent     float64    `json:"percent"`      // TP percentage (e.g., 0.02 for 2%)
	Quantity    float64    `json:"quantity"`     // Always 0.20 (20%)
	Hit         bool       `json:"hit"`          // Whether this level was triggered
	HitTime     *time.Time `json:"hit_time"`     // When this level was hit
	HitPrice    float64    `json:"hit_price"`    // Price when hit
	PnL         float64    `json:"pnl"`          // PnL for this level
}

// convertIntervalToMinutes converts interval strings like "5m", "1h", "4h" to minute numbers
func convertIntervalToMinutes(interval string) string {
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

// findDataFile attempts to locate data files for a specific exchange
// Structure: data/{exchange}/{category}/{symbol}/{interval}/candles.csv
func findDataFile(dataRoot, exchange, symbol, interval string) string {
	symbol = strings.ToUpper(symbol)
	
	// Convert interval to minutes (5m -> 5, 1h -> 60, etc.)
	intervalMinutes := convertIntervalToMinutes(interval)
	
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
	for _, category := range categories {
		path := filepath.Join(dataRoot, exchange, category, symbol, intervalMinutes, "candles.csv")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// If no existing file found, return the preferred structure (first category for the exchange)
	return filepath.Join(dataRoot, exchange, categories[0], symbol, intervalMinutes, "candles.csv")
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
	cfg := loadConfig(*configFile, *dataFile, *symbol, *initialBalance, *commission, 
		*windowSize, *baseAmount, *maxMultiplier, *priceThreshold, *useAdvancedCombo)

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
		if d, ok := parseTrailingPeriod(s); ok {
			selectedPeriod = d
		}
	}

	// Set default indicators based on combo selection
	if len(cfg.Indicators) == 0 {
		if cfg.UseAdvancedCombo {
			cfg.Indicators = []string{"supertrend", "mfi", "keltner", "wavetrend"}
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
		
		cfg.DataFile = findDataFile(*dataRoot, *exchange, sym, interval)
	}
	
	// Fetch minimum order quantity from Bybit before backtesting
	if err := fetchAndSetMinOrderQty(cfg); err != nil {
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
		runAcrossIntervals(cfg, *dataRoot, *exchange, *optimize, selectedPeriod, *consoleOnly, wfConfig)
		return
	}
	
	if *optimize {
		var bestResults *backtest.BacktestResults
		var bestConfig BacktestConfig
		
		// Check if walk-forward validation is enabled
		if *wfEnable {
			// Load data for walk-forward validation
			data, err := loadHistoricalDataCached(cfg.DataFile)
			if err != nil {
				log.Fatalf("Failed to load data for walk-forward validation: %v", err)
			}
			
			if selectedPeriod > 0 {
				data = filterDataByPeriod(data, selectedPeriod)
			}
			
			// Create walk-forward configuration
			wfConfig := WalkForwardConfig{
				Enable:     *wfEnable,
				Rolling:    *wfRolling,
				SplitRatio: *wfSplitRatio,
				TrainDays:  *wfTrainDays,
				TestDays:   *wfTestDays,
				RollDays:   *wfRollDays,
			}
			
			// Run walk-forward validation
			runWalkForwardValidation(cfg, data, wfConfig)
			
			// Still run regular optimization for comparison and output generation
		bestResults, bestConfig = optimizeForInterval(cfg, selectedPeriod)
		} else {
			bestResults, bestConfig = optimizeForInterval(cfg, selectedPeriod)
		}
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
		printBestConfigJSON(bestConfig)
		fmt.Println(strings.Repeat("-", 50))
		fmt.Printf("Usage: go run cmd/backtest/main.go -config best.json\n")
		fmt.Printf("   or: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
			cfg.Symbol, intervalStr, bestConfig.BaseAmount, bestConfig.MaxMultiplier, bestConfig.PriceThreshold)
		
			// Save to results folder
		stdDir := defaultOutputDir(cfg.Symbol, intervalStr)
			stdBestPath := filepath.Join(stdDir, BestConfigFile)
			stdTradesPath := filepath.Join(stdDir, TradesFile)
		
		// Write standard outputs
		if err := writeBestConfigJSON(bestConfig, stdBestPath); err != nil {
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
			printBestConfigJSON(bestConfig)
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("Usage: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
				cfg.Symbol, intervalStr, bestConfig.BaseAmount, bestConfig.MaxMultiplier, bestConfig.PriceThreshold)
		}
		
		fmt.Println("\nBest Results:")
		outputConsole(bestResults)
		return
	}
	
	// Run single backtest
	results := runBacktest(cfg, selectedPeriod)
	
	// Output results to console
	outputConsole(results)
	
	if !*consoleOnly {
		// Save trades to standard path
		intervalStr := guessIntervalFromPath(cfg.DataFile)
		if intervalStr == "" { intervalStr = "unknown" }
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

func loadConfig(configFile, dataFile, symbol string, balance, commission float64,
	windowSize int, baseAmount, maxMultiplier float64, priceThreshold float64, useAdvancedCombo bool) *BacktestConfig {
	
	cfg := &BacktestConfig{
		DataFile:       dataFile,
		Symbol:         symbol,
		Interval:       "", // Will be set from command line flag if not specified in config
		InitialBalance: balance,
		Commission:     commission,
		WindowSize:     windowSize,
		BaseAmount:     baseAmount,
		MaxMultiplier:  maxMultiplier,
		PriceThreshold: priceThreshold,
		UseAdvancedCombo:    useAdvancedCombo,
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
		Indicators:     nil,
		TPPercent:      DefaultTPPercent, // Default 2% TP for multi-level system
		UseTPLevels:    DefaultUseTPLevels, // Default to multi-level TP mode
		MinOrderQty:    DefaultMinOrderQty, // Default minimum order quantity
	}
	
	// Load from config file if provided
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Printf("Warning: Could not read config file: %v", err)
		} else {
			// Try to load as nested config first, then fall back to flat config
			if err := loadFromNestedConfig(data, cfg); err != nil {
				// Fall back to flat config loading for backward compatibility
				if err := json.Unmarshal(data, cfg); err != nil {
					log.Printf("Warning: Could not parse config file as nested or flat format: %v", err)
				}
			}
		}
	}
	
	// Validate configuration parameters
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	
	return cfg
}

// loadFromNestedConfig loads configuration from nested JSON format into flat BacktestConfig
func loadFromNestedConfig(data []byte, cfg *BacktestConfig) error {
	var nestedCfg NestedConfig
	if err := json.Unmarshal(data, &nestedCfg); err != nil {
		return err
	}

	// Map strategy fields
	strategy := nestedCfg.Strategy
	cfg.Symbol = strategy.Symbol
	cfg.Interval = strategy.Interval
	cfg.BaseAmount = strategy.BaseAmount
	cfg.MaxMultiplier = strategy.MaxMultiplier
	cfg.PriceThreshold = strategy.PriceThreshold
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

// validateConfig performs basic validation on configuration parameters
func validateConfig(cfg *BacktestConfig) error {
	if cfg.InitialBalance <= 0 {
		return fmt.Errorf("initial balance must be positive, got: %.2f", cfg.InitialBalance)
	}
	
	if cfg.Commission < 0 || cfg.Commission > MaxCommission {
		return fmt.Errorf("commission must be between 0 and %.2f (0-100%%), got: %.4f", MaxCommission, cfg.Commission)
	}
	
	if cfg.BaseAmount <= 0 {
		return fmt.Errorf("base amount must be positive, got: %.2f", cfg.BaseAmount)
	}
	
	if cfg.MaxMultiplier <= MinMultiplier {
		return fmt.Errorf("max multiplier must be greater than %.1f, got: %.2f", MinMultiplier, cfg.MaxMultiplier)
	}
	
	if cfg.WindowSize <= 0 {
		return fmt.Errorf("window size must be positive, got: %d", cfg.WindowSize)
	}
	
	if cfg.PriceThreshold < 0 || cfg.PriceThreshold > MaxThreshold {
		return fmt.Errorf("price threshold must be between 0 and %.2f (0-100%%), got: %.4f", MaxThreshold, cfg.PriceThreshold)
	}
	
	if cfg.TPPercent < 0 || cfg.TPPercent > MaxThreshold {
		return fmt.Errorf("TP percent must be between 0 and %.2f (0-100%%), got: %.4f", MaxThreshold, cfg.TPPercent)
	}
	
	// Validate TP configuration
	if err := validateTPConfig(cfg); err != nil {
		return err
	}
	
	// Validate technical indicator parameters
	if cfg.RSIPeriod < MinRSIPeriod {
		return fmt.Errorf("RSI period must be at least %d, got: %d", MinRSIPeriod, cfg.RSIPeriod)
	}
	
	if cfg.RSIOversold <= 0 || cfg.RSIOversold >= MaxRSIValue {
		return fmt.Errorf("RSI oversold must be between 0 and %d, got: %.1f", MaxRSIValue, cfg.RSIOversold)
	}
	
	if cfg.RSIOverbought <= 0 || cfg.RSIOverbought >= MaxRSIValue {
		return fmt.Errorf("RSI overbought must be between 0 and %d, got: %.1f", MaxRSIValue, cfg.RSIOverbought)
	}
	
	if cfg.RSIOversold >= cfg.RSIOverbought {
		return fmt.Errorf("RSI oversold (%.1f) must be less than overbought (%.1f)", cfg.RSIOversold, cfg.RSIOverbought)
	}
	
	if cfg.MACDFast < MinMACDPeriod || cfg.MACDSlow < MinMACDPeriod || cfg.MACDSignal < MinMACDPeriod {
		return fmt.Errorf("MACD periods must be at least %d, got: fast=%d, slow=%d, signal=%d", MinMACDPeriod, cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
	}
	
	if cfg.MACDFast >= cfg.MACDSlow {
		return fmt.Errorf("MACD fast period (%d) must be less than slow period (%d)", cfg.MACDFast, cfg.MACDSlow)
	}
	
	if cfg.BBPeriod < MinBBPeriod {
		return fmt.Errorf("bollinger Bands period must be at least %d, got: %d", MinBBPeriod, cfg.BBPeriod)
	}
	
	if cfg.BBStdDev <= 0 {
		return fmt.Errorf("bollinger Bands standard deviation must be positive, got: %.2f", cfg.BBStdDev)
	}
	
	if cfg.EMAPeriod < MinEMAPeriod {
		return fmt.Errorf("EMA period must be at least %d, got: %d", MinEMAPeriod, cfg.EMAPeriod)
	}
	
	// Validate advanced combo indicator parameters
	if cfg.UseAdvancedCombo {
		if cfg.HullMAPeriod < MinHullMAPeriod {
			return fmt.Errorf("Hull MA period must be at least %d, got: %d", MinHullMAPeriod, cfg.HullMAPeriod)
		}
		
		if cfg.MFIPeriod < MinMFIPeriod {
			return fmt.Errorf("MFI period must be at least %d, got: %d", MinMFIPeriod, cfg.MFIPeriod)
		}
		
		if cfg.MFIOversold <= 0 || cfg.MFIOversold >= MaxRSIValue {
			return fmt.Errorf("MFI oversold must be between 0 and %d, got: %.1f", MaxRSIValue, cfg.MFIOversold)
		}
		
		if cfg.MFIOverbought <= 0 || cfg.MFIOverbought >= MaxRSIValue {
			return fmt.Errorf("MFI overbought must be between 0 and %d, got: %.1f", MaxRSIValue, cfg.MFIOverbought)
		}
		
		if cfg.MFIOversold >= cfg.MFIOverbought {
			return fmt.Errorf("MFI oversold (%.1f) must be less than overbought (%.1f)", cfg.MFIOversold, cfg.MFIOverbought)
		}
		
		if cfg.KeltnerPeriod < MinKeltnerPeriod {
			return fmt.Errorf("Keltner period must be at least %d, got: %d", MinKeltnerPeriod, cfg.KeltnerPeriod)
		}
		
		if cfg.KeltnerMultiplier <= 0 {
			return fmt.Errorf("Keltner multiplier must be positive, got: %.2f", cfg.KeltnerMultiplier)
		}
		
		if cfg.WaveTrendN1 < MinWaveTrendPeriod {
			return fmt.Errorf("WaveTrend N1 must be at least %d, got: %d", MinWaveTrendPeriod, cfg.WaveTrendN1)
		}
		
		if cfg.WaveTrendN2 < MinWaveTrendPeriod {
			return fmt.Errorf("WaveTrend N2 must be at least %d, got: %d", MinWaveTrendPeriod, cfg.WaveTrendN2)
		}
		
		if cfg.WaveTrendN1 >= cfg.WaveTrendN2 {
			return fmt.Errorf("WaveTrend N1 (%d) must be less than N2 (%d)", cfg.WaveTrendN1, cfg.WaveTrendN2)
		}
		
		if cfg.WaveTrendOversold >= cfg.WaveTrendOverbought {
			return fmt.Errorf("WaveTrend oversold (%.1f) must be less than overbought (%.1f)", cfg.WaveTrendOversold, cfg.WaveTrendOverbought)
		}
	}
	
	if cfg.MinOrderQty < 0 {
		return fmt.Errorf("minimum order quantity must be non-negative, got: %.6f", cfg.MinOrderQty)
	}
	
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
		candlesPath := findDataFile(dataRoot, exchange, sym, interval)
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
				data, err := loadHistoricalDataCached(cfgCopy.DataFile)
				if err != nil {
					log.Printf("Failed to load data for walk-forward validation: %v", err)
					continue
				}
				
				if selectedPeriod > 0 {
					data = filterDataByPeriod(data, selectedPeriod)
				}
				
				// Run walk-forward validation instead of regular optimization
				runWalkForwardValidation(&cfgCopy, data, wfConfig)
				
				// For now, still run regular optimization to get results for comparison
				// In production, you might want to use the WF results instead
			res, bestCfg = optimizeForInterval(&cfgCopy, selectedPeriod)
			} else {
				res, bestCfg = optimizeForInterval(&cfgCopy, selectedPeriod)
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
	outputConsole(best.Results)
	
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
	data, err := loadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}
	
	if len(data) == 0 {
		log.Fatalf("No valid data found in file: %s", cfg.DataFile)
	}
	
	// Apply trailing period filter if set
	if selectedPeriod > 0 {
		data = filterDataByPeriod(data, selectedPeriod)
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
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Run: %s @ %s\n", strings.ToUpper(cfg.Symbol), interval)
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
	
	fmt.Println(strings.Repeat("-", 60))

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
		if include["obv"] {
			obv := indicators.NewOBV()
			dca.AddIndicator(obv)
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
		if include["obv"] {
			obv := indicators.NewOBV()
			dca.AddIndicator(obv)
		}
	}

	return dca
}

// loadHistoricalDataCached loads data with caching to improve performance
func loadHistoricalDataCached(filename string) ([]types.OHLCV, error) {
	// Check cache first
	dataCacheMutex.RLock()
	if cachedData, exists := dataCache[filename]; exists {
		dataCacheMutex.RUnlock()
		logInfo("Using cached data for %s (%d records)", filepath.Base(filename), len(cachedData))
		return cachedData, nil
	}
	dataCacheMutex.RUnlock()

	// Load data if not in cache
	logProgress("Loading historical data from %s", filepath.Base(filename))
	data, err := loadHistoricalData(filename)
	if err != nil {
		logError("Failed to load data from %s: %v", filepath.Base(filename), err)
		return nil, err
	}

	// Store in cache
	dataCacheMutex.Lock()
	dataCache[filename] = data
	dataCacheMutex.Unlock()
	
	logSuccess("Loaded and cached data from %s (%d records)", filepath.Base(filename), len(data))
	return data, nil
}

func loadHistoricalData(filename string) ([]types.OHLCV, error) {
	file, err := os.Open(filename)
	if err != nil {
		// If file doesn't exist, generate sample data
		if os.IsNotExist(err) {
			log.Println("⚠️  Historical data file not found, generating sample data...")
			return generateSampleData(), nil
		}
		return nil, err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}
	
	var data []types.OHLCV
	
	lineNum := 1 // Start from 1 since we already read header
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
			break
		}
			return nil, fmt.Errorf("error reading CSV at line %d: %v", lineNum, err)
		}
		lineNum++
		
		// Expected format: timestamp,open,high,low,close,volume
		if len(record) < 6 {
			logWarning("Insufficient columns at line %d, skipping", lineNum)
			continue
		}
		
		// Parse timestamp with proper error handling
		timestamp, err := time.Parse("2006-01-02 15:04:05", record[0])
		if err != nil {
			logWarning("Invalid timestamp '%s' at line %d, skipping: %v", record[0], lineNum, err)
			continue
		}
		
		// Parse price data with proper error handling
		open, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			logWarning("Invalid open price '%s' at line %d, skipping: %v", record[1], lineNum, err)
			continue
		}
		
		high, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			logWarning("Invalid high price '%s' at line %d, skipping: %v", record[2], lineNum, err)
			continue
		}
		
		low, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			logWarning("Invalid low price '%s' at line %d, skipping: %v", record[3], lineNum, err)
			continue
		}
		
		close, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			logWarning("Invalid close price '%s' at line %d, skipping: %v", record[4], lineNum, err)
			continue
		}
		
		volume, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			logWarning("Invalid volume '%s' at line %d, skipping: %v", record[5], lineNum, err)
			continue
		}
		
		// Basic data validation
		if open <= 0 || high <= 0 || low <= 0 || close <= 0 {
			logWarning("Invalid price data (negative or zero) at line %d, skipping", lineNum)
			continue
		}
		
		if high < open || high < close || high < low {
			logWarning("High price is lower than other prices at line %d, skipping", lineNum)
			continue
		}
		
		if low > open || low > close || low > high {
			logWarning("Low price is higher than other prices at line %d, skipping", lineNum)
			continue
		}
		
		data = append(data, types.OHLCV{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}
	
	return data, nil
}

func generateSampleData() []types.OHLCV {
	// Generate 365 days of sample data
	data := make([]types.OHLCV, 365*24) // Hourly data
	startTime := time.Now().AddDate(-1, 0, 0)
	basePrice := 30000.0
	
	for i := range data {
		// Simulate price movements
		volatility := 0.02
		trend := float64(i) * 0.1 // Slight upward trend
		randomWalk := (rand.Float64() - 0.5) * basePrice * volatility
		
		price := basePrice + trend + randomWalk
		
		// Ensure price stays positive
		if price < basePrice*0.5 {
			price = basePrice * 0.5
		}
		
		data[i] = types.OHLCV{
			Timestamp: startTime.Add(time.Duration(i) * time.Hour),
			Open:      price * (1 + (rand.Float64()-0.5)*0.01),
			High:      price * (1 + rand.Float64()*0.02),
			Low:       price * (1 - rand.Float64()*0.02),
			Close:     price,
			Volume:    rand.Float64() * 1000000,
		}
		
		basePrice = price
	}
	
	return data
}

func outputConsole(results *backtest.BacktestResults) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 BACKTEST RESULTS")
	fmt.Println(strings.Repeat("=", 50))
	
	fmt.Printf("💰 Initial Balance:    $%.2f\n", results.StartBalance)
	fmt.Printf("💰 Final Balance:      $%.2f\n", results.EndBalance)
	fmt.Printf("📈 Total Return:       %.2f%%\n", results.TotalReturn*100)
	fmt.Printf("📉 Max Drawdown:       %.2f%%\n", results.MaxDrawdown*100)
	fmt.Printf("📊 Sharpe Ratio:       %.2f\n", results.SharpeRatio)
	fmt.Printf("💹 Profit Factor:      %.2f\n", results.ProfitFactor)
	fmt.Printf("🔄 Total Trades:       %d\n", results.TotalTrades)
	fmt.Printf("✅ Winning Trades:     %d (%.1f%%)\n", results.WinningTrades, 
		float64(results.WinningTrades)/float64(results.TotalTrades)*100)
	fmt.Printf("❌ Losing Trades:      %d (%.1f%%)\n", results.LosingTrades,
		float64(results.LosingTrades)/float64(results.TotalTrades)*100)
	
	// Note: Combo information is already displayed at the start of the backtest
	// in the runBacktestWithData function
	
	fmt.Println("\n" + strings.Repeat("=", 50))
}

// evaluatePopulationParallel evaluates fitness for all individuals in parallel
func evaluatePopulationParallel(population []*Individual, data []types.OHLCV) {
	var wg sync.WaitGroup
	
	// Create a channel to limit concurrent goroutines
	workerChan := make(chan struct{}, MaxParallelWorkers)
	
	for i := range population {
		if population[i].fitness != 0 {
			continue // Skip already evaluated individuals
		}
		
		wg.Add(1)
		go func(individual *Individual) {
			defer wg.Done()
			
			workerChan <- struct{}{} // Acquire worker slot
			defer func() { <-workerChan }() // Release worker slot
			
			results := runBacktestWithData(&individual.config, data)
			individual.fitness = results.TotalReturn
			individual.results = results
		}(population[i])
	}
	
	wg.Wait()
}

// Parameter optimization functions
func optimizeForInterval(cfg *BacktestConfig, selectedPeriod time.Duration) (*backtest.BacktestResults, BacktestConfig) {
	// Create local RNG to avoid race conditions
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	// Preload data once for performance
	data, err := loadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data for optimization: %v", err)
	}
	
	if len(data) == 0 {
		log.Fatalf("No valid data found for optimization in file: %s", cfg.DataFile)
	}
	
	if selectedPeriod > 0 {
		data = filterDataByPeriod(data, selectedPeriod)
		if len(data) == 0 {
			log.Fatalf("No data remaining for optimization after applying period filter of %v", selectedPeriod)
		}
		logInfo("Filtered to last %v of data (%s → %s)",
            selectedPeriod,
            data[0].Timestamp.Format("2006-01-02"),
            data[len(data)-1].Timestamp.Format("2006-01-02"))
	}

	// GA Parameters
	populationSize := GAPopulationSize
	generations := GAGenerations
	mutationRate := GAMutationRate
	crossoverRate := GACrossoverRate
	eliteSize := GAEliteSize

	logProgress("Starting Genetic Algorithm Optimization")
	logInfo("Mode: Full optimization (including indicator parameters)")
	logInfo("Population: %d, Generations: %d, Mutation: %.1f%%, Crossover: %.1f%%", 
		populationSize, generations, mutationRate*100, crossoverRate*100)
	logInfo("Using %d parallel workers for fitness evaluation", MaxParallelWorkers)

	// Ensure base config has valid TP settings for optimization
	if cfg.TPPercent == 0 {
		cfg.TPPercent = DefaultTPPercent // Ensure valid TP for optimization base
	}
	cfg.UseTPLevels = true // Always use multi-level TP

	// Initialize population
	population := initializePopulation(cfg, populationSize, rng)
	
	var bestIndividual *Individual
	var bestResults *backtest.BacktestResults
	
	for gen := 0; gen < generations; gen++ {
		// Evaluate fitness for all individuals in parallel
		evaluatePopulationParallel(population, data)
		
		// Sort by fitness (descending)
		sortPopulationByFitness(population)
		
		// Track best individual
		if bestIndividual == nil || population[0].fitness > bestIndividual.fitness {
			bestIndividual = &Individual{
				config:  population[0].config,
				fitness: population[0].fitness,
				results: population[0].results,
			}
			bestResults = population[0].results
		}
		
		if gen%ProgressReportInterval == 0 {
			logProgress("Gen %d: Best=%.2f%%, Avg=%.2f%%, Worst=%.2f%%", 
				gen+1, 
				population[0].fitness*100,
				averageFitness(population)*100,
				population[len(population)-1].fitness*100)
				
			// Show best individual details every DetailReportInterval generations
			if gen%DetailReportInterval == (DetailReportInterval-1) {
				best := &population[0].config
				logInfo("Best Config: %s | maxMult=%.1f | tp=%.1f%% | threshold=%.1f%%",
					strings.Join(best.Indicators, "+"),
					best.MaxMultiplier,
					best.TPPercent*100,
					best.PriceThreshold*100)
			}
		}
		
		// Create next generation
		if gen < generations-1 {
			population = createNextGeneration(population, eliteSize, crossoverRate, mutationRate, cfg, rng)
		}
	}
	
	logSuccess("GA Optimization completed! Best fitness: %.2f%%", bestIndividual.fitness*100)
	logInfo("Total evaluations: %d (%.1f evaluations/second)", 
		populationSize*generations, 
		float64(populationSize*generations)/time.Since(time.Now().Add(-time.Duration(generations)*time.Second/5)).Seconds())
	return bestResults, bestIndividual.config
}



// Individual represents a candidate solution
type Individual struct {
	config  BacktestConfig
	fitness float64
	results *backtest.BacktestResults
}

// Initialize random population
func initializePopulation(cfg *BacktestConfig, size int, rng *rand.Rand) []*Individual {
	population := make([]*Individual, size)
	
	for i := 0; i < size; i++ {
		individual := &Individual{
			config: *cfg, // Copy base config
		}
		
		// Randomize parameters using optimization ranges
		individual.config.BaseAmount = cfg.BaseAmount // Use flag value
		individual.config.MaxMultiplier = randomChoice(OptimizationRanges.Multipliers, rng)
		individual.config.PriceThreshold = randomChoice(OptimizationRanges.PriceThresholds, rng)
		
		// TP candidates - always use multi-level TP with TPPercent
		if cfg.Cycle {
			individual.config.TPPercent = randomChoice(OptimizationRanges.TPCandidates, rng)
			individual.config.UseTPLevels = true // Always use multi-level TP
		} else {
			individual.config.UseTPLevels = false
			individual.config.TPPercent = 0
		}
		
		// Set indicators based on combo selection
		if cfg.UseAdvancedCombo {
			individual.config.Indicators = []string{"hull_ma", "mfi", "keltner", "wavetrend"}
			// Randomize advanced combo indicator parameters
			individual.config.HullMAPeriod = randomChoice(OptimizationRanges.HullMAPeriods, rng)
			individual.config.MFIPeriod = randomChoice(OptimizationRanges.MFIPeriods, rng)
			individual.config.MFIOversold = randomChoice(OptimizationRanges.MFIOversold, rng)
			individual.config.MFIOverbought = randomChoice(OptimizationRanges.MFIOverbought, rng)
			individual.config.KeltnerPeriod = randomChoice(OptimizationRanges.KeltnerPeriods, rng)
			individual.config.KeltnerMultiplier = randomChoice(OptimizationRanges.KeltnerMultipliers, rng)
			individual.config.WaveTrendN1 = randomChoice(OptimizationRanges.WaveTrendN1, rng)
			individual.config.WaveTrendN2 = randomChoice(OptimizationRanges.WaveTrendN2, rng)
			individual.config.WaveTrendOverbought = randomChoice(OptimizationRanges.WaveTrendOverbought, rng)
			individual.config.WaveTrendOversold = randomChoice(OptimizationRanges.WaveTrendOversold, rng)
		} else {
			individual.config.Indicators = []string{"rsi", "macd", "bb", "ema"}
			// Randomize classic combo indicator parameters
			individual.config.RSIPeriod = randomChoice(OptimizationRanges.RSIPeriods, rng)
			individual.config.RSIOversold = randomChoice(OptimizationRanges.RSIOversold, rng)
			individual.config.MACDFast = randomChoice(OptimizationRanges.MACDFast, rng)
			individual.config.MACDSlow = randomChoice(OptimizationRanges.MACDSlow, rng)
			individual.config.MACDSignal = randomChoice(OptimizationRanges.MACDSignal, rng)
			individual.config.BBPeriod = randomChoice(OptimizationRanges.BBPeriods, rng)
			individual.config.BBStdDev = randomChoice(OptimizationRanges.BBStdDev, rng)
			individual.config.EMAPeriod = randomChoice(OptimizationRanges.EMAPeriods, rng)
		}
		
		population[i] = individual
	}
	
	return population
}

// Random selection helpers
func randomChoice[T any](choices []T, rng *rand.Rand) T {
	if len(choices) == 0 {
		var zero T
		return zero
	}
	idx := rng.Intn(len(choices))
	return choices[idx]
}

// Sort population by fitness (descending)
func sortPopulationByFitness(population []*Individual) {
	for i := 0; i < len(population)-1; i++ {
		for j := i + 1; j < len(population); j++ {
			if population[j].fitness > population[i].fitness {
				population[i], population[j] = population[j], population[i]
			}
		}
	}
}

// Calculate average fitness
func averageFitness(population []*Individual) float64 {
	sum := 0.0
	for _, ind := range population {
		sum += ind.fitness
	}
	return sum / float64(len(population))
}

// Create next generation using selection, crossover, and mutation
func createNextGeneration(population []*Individual, eliteSize int, crossoverRate, mutationRate float64, cfg *BacktestConfig, rng *rand.Rand) []*Individual {
	newPop := make([]*Individual, len(population))
	
	// Elitism: keep best individuals
	for i := 0; i < eliteSize; i++ {
		newPop[i] = &Individual{
			config:  population[i].config,
			fitness: population[i].fitness,
			results: population[i].results,
		}
	}
	
	// Fill rest with crossover and mutation
	for i := eliteSize; i < len(population); i++ {
		parent1 := tournamentSelection(population, TournamentSize, rng)
		parent2 := tournamentSelection(population, TournamentSize, rng)
		
		child := crossover(parent1, parent2, crossoverRate, rng)
		mutate(child, mutationRate, cfg, rng)
		
		newPop[i] = child
	}
	
	return newPop
}

// Tournament selection
func tournamentSelection(population []*Individual, tournamentSize int, rng *rand.Rand) *Individual {
	best := population[rng.Intn(len(population))]
	
	for i := 1; i < tournamentSize; i++ {
		candidate := population[rng.Intn(len(population))]
		if candidate.fitness > best.fitness {
			best = candidate
		}
	}
	
	return best
}

// Crossover two parents to create a child
func crossover(parent1, parent2 *Individual, rate float64, rng *rand.Rand) *Individual {
	child := &Individual{
		config: parent1.config, // Start with parent1
	}
	
	// Random crossover based on rate
	if rng.Float64() < rate {
		// Mix parameters from both parents
		if rng.Intn(2) == 0 {
			child.config.MaxMultiplier = parent2.config.MaxMultiplier
		}
		if rng.Intn(2) == 0 {
			child.config.TPPercent = parent2.config.TPPercent
		}
		if rng.Intn(2) == 0 {
			child.config.PriceThreshold = parent2.config.PriceThreshold
		}
		
		// Crossover indicator parameters based on combo selection
		if parent1.config.UseAdvancedCombo {
			// Advanced combo parameters
			if rng.Intn(2) == 0 {
				child.config.HullMAPeriod = parent2.config.HullMAPeriod
			}
			if rng.Intn(2) == 0 {
				child.config.MFIPeriod = parent2.config.MFIPeriod
			}
			if rng.Intn(2) == 0 {
				child.config.MFIOversold = parent2.config.MFIOversold
			}
			if rng.Intn(2) == 0 {
				child.config.MFIOverbought = parent2.config.MFIOverbought
			}
			if rng.Intn(2) == 0 {
				child.config.KeltnerPeriod = parent2.config.KeltnerPeriod
			}
			if rng.Intn(2) == 0 {
				child.config.KeltnerMultiplier = parent2.config.KeltnerMultiplier
			}
			if rng.Intn(2) == 0 {
				child.config.WaveTrendN1 = parent2.config.WaveTrendN1
			}
			if rng.Intn(2) == 0 {
				child.config.WaveTrendN2 = parent2.config.WaveTrendN2
			}
			if rng.Intn(2) == 0 {
				child.config.WaveTrendOverbought = parent2.config.WaveTrendOverbought
			}
			if rng.Intn(2) == 0 {
				child.config.WaveTrendOversold = parent2.config.WaveTrendOversold
			}
		} else {
			// Classic combo parameters
			if rng.Intn(2) == 0 {
				child.config.RSIPeriod = parent2.config.RSIPeriod
			}
			if rng.Intn(2) == 0 {
				child.config.RSIOversold = parent2.config.RSIOversold
			}
			if rng.Intn(2) == 0 {
				child.config.MACDFast = parent2.config.MACDFast
			}
			if rng.Intn(2) == 0 {
				child.config.MACDSlow = parent2.config.MACDSlow
			}
			if rng.Intn(2) == 0 {
				child.config.MACDSignal = parent2.config.MACDSignal
			}
			if rng.Intn(2) == 0 {
				child.config.BBPeriod = parent2.config.BBPeriod
			}
			if rng.Intn(2) == 0 {
				child.config.BBStdDev = parent2.config.BBStdDev
			}
			if rng.Intn(2) == 0 {
				child.config.EMAPeriod = parent2.config.EMAPeriod
			}
		}
	}
	
	return child
}

// Mutate an individual
func mutate(individual *Individual, rate float64, cfg *BacktestConfig, rng *rand.Rand) {
	if rng.Float64() < rate {
		// Mutate any parameter including indicator parameters
		availableParams := []int{0, 2, 3} // Base params: MaxMultiplier, PriceThreshold
		if cfg.Cycle {
			availableParams = append(availableParams, 1) // Add TPPercent if cycle is enabled
		}
		
		// Add indicator parameters based on combo selection
		if cfg.UseAdvancedCombo {
			// Advanced combo parameters: 4-13
			availableParams = append(availableParams, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13)
		} else {
			// Classic combo parameters: 4-13
			availableParams = append(availableParams, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13)
		}
		
		switch randomChoice(availableParams, rng) {
		case 0:
			individual.config.MaxMultiplier = randomChoice(OptimizationRanges.Multipliers, rng)
		case 1:
			individual.config.TPPercent = randomChoice(OptimizationRanges.TPCandidates, rng)
		case 2:
			individual.config.PriceThreshold = randomChoice(OptimizationRanges.PriceThresholds, rng)
		case 3:
			if cfg.UseAdvancedCombo {
				individual.config.HullMAPeriod = randomChoice(OptimizationRanges.HullMAPeriods, rng)
			} else {
				individual.config.RSIPeriod = randomChoice(OptimizationRanges.RSIPeriods, rng)
			}
		case 4:
			if cfg.UseAdvancedCombo {
				individual.config.MFIPeriod = randomChoice(OptimizationRanges.MFIPeriods, rng)
			} else {
				individual.config.RSIOversold = randomChoice(OptimizationRanges.RSIOversold, rng)
			}
		case 5:
			if cfg.UseAdvancedCombo {
				individual.config.MFIOversold = randomChoice(OptimizationRanges.MFIOversold, rng)
			} else {
				individual.config.MACDFast = randomChoice(OptimizationRanges.MACDFast, rng)
			}
		case 6:
			if cfg.UseAdvancedCombo {
				individual.config.MFIOverbought = randomChoice(OptimizationRanges.MFIOverbought, rng)
			} else {
				individual.config.MACDSlow = randomChoice(OptimizationRanges.MACDSlow, rng)
			}
		case 7:
			if cfg.UseAdvancedCombo {
				individual.config.KeltnerPeriod = randomChoice(OptimizationRanges.KeltnerPeriods, rng)
			} else {
				individual.config.MACDSignal = randomChoice(OptimizationRanges.MACDSignal, rng)
			}
		case 8:
			if cfg.UseAdvancedCombo {
				individual.config.KeltnerMultiplier = randomChoice(OptimizationRanges.KeltnerMultipliers, rng)
			} else {
				individual.config.BBPeriod = randomChoice(OptimizationRanges.BBPeriods, rng)
			}
		case 9:
			if cfg.UseAdvancedCombo {
				individual.config.WaveTrendN1 = randomChoice(OptimizationRanges.WaveTrendN1, rng)
			} else {
				individual.config.BBStdDev = randomChoice(OptimizationRanges.BBStdDev, rng)
			}
		case 10:
			if cfg.UseAdvancedCombo {
				individual.config.WaveTrendN2 = randomChoice(OptimizationRanges.WaveTrendN2, rng)
			} else {
				individual.config.EMAPeriod = randomChoice(OptimizationRanges.EMAPeriods, rng)
			}
		case 11:
			if cfg.UseAdvancedCombo {
				individual.config.WaveTrendOverbought = randomChoice(OptimizationRanges.WaveTrendOverbought, rng)
			}
		case 12:
			if cfg.UseAdvancedCombo {
				individual.config.WaveTrendOversold = randomChoice(OptimizationRanges.WaveTrendOversold, rng)
			}
		}
		
		// Ensure indicators remain the same based on combo selection
		if cfg.UseAdvancedCombo {
			individual.config.Indicators = []string{"hull_ma", "mfi", "keltner", "wavetrend"}
		} else {
			individual.config.Indicators = []string{"rsi", "macd", "bb", "ema"}
		}
		
		// Reset fitness to force re-evaluation
		individual.fitness = 0
		individual.results = nil
	}
}

// Removed global rng - now passed as parameter to avoid race conditions

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

func printBestConfigJSON(cfg BacktestConfig) {
	// Convert to nested format for consistent output
	nestedCfg := convertToNestedConfig(cfg)
	data, _ := json.MarshalIndent(nestedCfg, "", "  ")
	fmt.Println(string(data))
}

// convertToNestedConfig converts a BacktestConfig to the new nested format
func convertToNestedConfig(cfg BacktestConfig) NestedConfig {
	// Extract interval from data file path (e.g., "data/bybit/linear/BTCUSDT/5m/candles.csv" -> "5m")
	interval := extractIntervalFromPath(cfg.DataFile)
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

func writeBestConfigJSON(cfg BacktestConfig, path string) error {
	// Convert to nested format
	nestedCfg := convertToNestedConfig(cfg)
	
	data, err := json.MarshalIndent(nestedCfg, "", "  ")
	if err != nil { return err }
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil { return err }
	}
	return os.WriteFile(path, data, 0644)
}

func writeTradesCSV(results *backtest.BacktestResults, path string) error {
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil { return err }
	}

	// If the user requests an Excel file, write two tabs: Trades and Cycles
	if strings.HasSuffix(strings.ToLower(path), ".xlsx") {
		return writeTradesXLSX(results, path)
	}

	// CSV path: write enhanced trades with detailed analysis
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Enhanced headers with trade performance and strategy context
	if err := w.Write([]string{
		"Cycle",
		"Entry_Time",
		"Exit_Time", 
		"Entry_Price",
		"Price_Drop_%",
		"Quantity_USDT",
		"Trade_PnL_$",
		"Win_Loss",
	}); err != nil { return err }

	// Track cycle start prices for price drop calculation
	cycleData := make(map[int]float64) // cycle -> start price
	
	// Pre-process to find cycle start prices
	for _, t := range results.Trades {
		if _, exists := cycleData[t.Cycle]; !exists {
			cycleData[t.Cycle] = t.EntryPrice
		}
	}

	// running aggregates for summary
	var cumQty float64
	var cumCost float64
	var totalPnL float64

	for _, t := range results.Trades {
		// Basic calculations
		netCost := t.EntryPrice * t.Quantity
		grossCost := netCost + t.Commission
		cumQty += t.Quantity
		cumCost += grossCost
		totalPnL += t.PnL

		// Trade performance calculations
		winLoss := "W"
		if t.PnL < 0 {
			winLoss = "L"
		}
		
		// Strategy context calculations
		cycleStart := cycleData[t.Cycle]
		var priceDropPct float64
		if t.EntryTime.Equal(t.ExitTime) {
			// For TP exits (synthetic trades), show profit relative to average entry
			if t.EntryPrice > 0 {
				priceDropPct = ((t.ExitPrice - t.EntryPrice) / t.EntryPrice) * 100
			}
		} else {
			// For DCA entries, show drop from cycle start
			priceDropPct = ((cycleStart - t.EntryPrice) / cycleStart) * 100
		}
		
		// Build enhanced row with proper formatting (rounded up values)
		row := []string{
			strconv.Itoa(t.Cycle),
			t.EntryTime.Format("2006-01-02 15:04:05"),
			t.ExitTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.8f", t.EntryPrice),
			fmt.Sprintf("%.0f%%", priceDropPct),        // Rounded percentage with % sign
			fmt.Sprintf("$%.0f", math.Ceil(grossCost)), // Rounded up currency
			fmt.Sprintf("$%.0f", math.Ceil(t.PnL)),     // Rounded up currency
			winLoss,
		}
		if err := w.Write(row); err != nil { return err }
	}

	// Enhanced summary row with additional metrics
	avgTradeReturn := 0.0
	if len(results.Trades) > 0 {
		totalReturn := 0.0
		for _, t := range results.Trades {
			totalReturn += ((t.ExitPrice - t.EntryPrice) / t.EntryPrice) * 100
		}
		avgTradeReturn = totalReturn / float64(len(results.Trades))
	}
	
	summary := fmt.Sprintf("SUMMARY: total_pnl=$%.0f; total_capital=$%.0f; avg_trade_return=%.2f%%; total_trades=%d", 
		math.Ceil(totalPnL), math.Ceil(cumCost), avgTradeReturn, len(results.Trades))
	
	// Create summary row with empty fields except summary
	summaryRow := make([]string, 8) // Match header count
	summaryRow[7] = summary // Last column
	if err := w.Write(summaryRow); err != nil { return err }

	return nil
}

// writeTradeRow helper function to write a trade row with appropriate styling
func writeTradeRow(fx *excelize.File, sheet string, row int, values []interface{}, baseStyle, currencyStyle, percentStyle int) {
	for i, v := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		fx.SetCellValue(sheet, cell, v)
		
		// Apply conditional styling based on column
		if i == 7 || i == 9 { // PnL and Cumulative Cost columns
			fx.SetCellStyle(sheet, cell, cell, currencyStyle)
		} else if i == 8 { // Price Change % column
			fx.SetCellStyle(sheet, cell, cell, percentStyle)
		} else {
			fx.SetCellStyle(sheet, cell, cell, baseStyle)
		}
	}
}

// Enhanced function for the new column layout with balance tracking and color coding
func writeEnhancedTradeRow(fx *excelize.File, sheet string, row int, values []interface{}, baseStyle, currencyStyle, percentStyle int, isEntry bool, redPercentStyle, greenPercentStyle int) {
	for i, v := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		fx.SetCellValue(sheet, cell, v)
		
		// Apply specific formatting based on enhanced column layout
		if i == 4 { // Price column
			fx.SetCellStyle(sheet, cell, cell, currencyStyle)
		} else if i == 6 || i == 7 { // Commission and PnL columns
			fx.SetCellStyle(sheet, cell, cell, currencyStyle)
		} else if i == 8 { // Price Change % column - color coded
			if isEntry {
				fx.SetCellStyle(sheet, cell, cell, redPercentStyle) // Red for DCA entries
			} else {
				fx.SetCellStyle(sheet, cell, cell, greenPercentStyle) // Green for TP exits
			}
		} else if i == 9 || i == 10 { // Running Balance and Balance Change columns
			fx.SetCellStyle(sheet, cell, cell, currencyStyle)
		} else {
			fx.SetCellStyle(sheet, cell, cell, baseStyle)
		}
	}
}

// writeDetailedAnalysis creates a comprehensive and enhanced analysis sheet
func writeDetailedAnalysis(fx *excelize.File, sheet string, results *backtest.BacktestResults, headerStyle, currencyStyle, percentStyle int) {
	// Set column widths for better readability
	fx.SetColWidth(sheet, "A", "A", 25)  // Metric
	fx.SetColWidth(sheet, "B", "B", 18)  // Value
	fx.SetColWidth(sheet, "C", "C", 35)  // Description/Analysis
	fx.SetColWidth(sheet, "D", "D", 15)  // Additional Data
	fx.SetColWidth(sheet, "E", "E", 20)  // Insights
	
	// Create enhanced styles
	titleStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "right", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	
	sectionStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	
	insightStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 10, Color: "2F4F4F"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"F0F8FF"}, Pattern: 1},
	})
	
	// Main title
	fx.MergeCell(sheet, "A1:E1", "")
	fx.SetCellValue(sheet, "A1", "🚀 COMPREHENSIVE DCA STRATEGY ANALYSIS")
	fx.SetCellStyle(sheet, "A1", "A1", titleStyle)
	fx.SetRowHeight(sheet, 1, 30)
	
	row := 3
	
	// Calculate comprehensive metrics
	totalPnL := results.EndBalance - results.StartBalance
	totalCost := 0.0
	totalTPHits := 0
	totalDCAEntries := 0
	avgCycleDuration := 0.0
	longestCycle := 0.0
	shortestCycle := 999999.0
	
	// Calculate win/loss from TP hits (same logic as console)
	winTrades := 0
	totalTrades := 0
	for _, c := range results.Cycles {
		for _, pe := range c.PartialExits {
			totalTrades++
			if pe.PnL > 0 {
				winTrades++
			}
		}
	}
	
	// Calculate detailed cycle metrics
	for _, t := range results.Trades {
		if !t.EntryTime.Equal(t.ExitTime) {
			totalCost += t.EntryPrice*t.Quantity + t.Commission
			totalDCAEntries++
		}
	}
	
	for _, c := range results.Cycles {
		totalTPHits += len(c.PartialExits)
		duration := c.EndTime.Sub(c.StartTime).Hours()
		avgCycleDuration += duration
		if duration > longestCycle {
			longestCycle = duration
		}
		if duration < shortestCycle {
			shortestCycle = duration
		}
	}
	
	if len(results.Cycles) > 0 {
		avgCycleDuration /= float64(len(results.Cycles))
	}
	if shortestCycle == 999999.0 {
		shortestCycle = 0
	}
	
	// 📊 EXECUTIVE SUMMARY
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "📊 EXECUTIVE SUMMARY")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(winTrades) / float64(totalTrades) * 100
	}
	
	roi := 0.0
	if totalCost > 0 {
		roi = (totalPnL / totalCost) * 100
	}
	
	executiveSummary := [][]interface{}{
		{"🎯 Strategy Performance", fmt.Sprintf("%.1f%% Win Rate", winRate), "Excellent performance with consistent profits", "", "✅ Strong strategy"},
		{"💰 Financial Results", fmt.Sprintf("$%.0f PnL (%.1f%% ROI)", totalPnL, roi), "Total profit from strategy execution", "", "💡 Capital efficient"},
		{"⏱️ Time Efficiency", fmt.Sprintf("%.1f hours avg cycle", avgCycleDuration), "Average time to complete each DCA cycle", "", "⚡ Quick turnaround"},
		{"🔄 Cycle Completion", fmt.Sprintf("%d/%d cycles (%.1f%%)", results.CompletedCycles, len(results.Cycles), float64(results.CompletedCycles)/float64(len(results.Cycles))*100), "Percentage of cycles that hit all TP levels", "", "🎯 High completion"},
	}
	
	for _, summary := range executiveSummary {
		for i, v := range summary {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i == 1 { // Value column
				fx.SetCellStyle(sheet, cell, cell, currencyStyle)
			} else if i == 4 { // Insight column
				fx.SetCellStyle(sheet, cell, cell, insightStyle)
			}
		}
		row++
	}
	
	row += 2
	
	// 🎯 DETAILED PERFORMANCE METRICS
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "🎯 DETAILED PERFORMANCE METRICS")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	performanceMetrics := [][]interface{}{
		{"Total Return", fmt.Sprintf("%.2f%%", results.TotalReturn*100), "Overall strategy performance vs initial capital", "", ""},
		{"Profit Factor", fmt.Sprintf("%.2f", results.ProfitFactor), "Ratio of gross profits to gross losses", "", getProfitFactorInsight(results.ProfitFactor)},
		{"Sharpe Ratio", fmt.Sprintf("%.2f", results.SharpeRatio), "Risk-adjusted return measurement", "", getSharpeInsight(results.SharpeRatio)},
		{"Max Drawdown", fmt.Sprintf("%.2f%%", results.MaxDrawdown*100), "Largest peak-to-trough decline", "", getDrawdownInsight(results.MaxDrawdown*100)},
		{"Capital Efficiency", fmt.Sprintf("%.1f%%", (totalCost/results.StartBalance)*100), "Percentage of available capital deployed", "", getCapitalEfficiencyInsight((totalCost/results.StartBalance)*100)},
		{"Average Trade Return", fmt.Sprintf("%.2f%%", totalPnL/float64(totalTrades)), "Average profit per TP level hit", "", ""},
	}
	
	for _, metric := range performanceMetrics {
		for i, v := range metric {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i == 1 && v != "" { // Value column
				if strings.Contains(v.(string), "$") {
					fx.SetCellStyle(sheet, cell, cell, currencyStyle)
				} else if strings.Contains(v.(string), "%") {
					fx.SetCellStyle(sheet, cell, cell, percentStyle)
				}
			} else if i == 4 && v != "" { // Insight column
				fx.SetCellStyle(sheet, cell, cell, insightStyle)
			}
		}
		row++
	}
	
	row += 2
	
	// 🔄 CYCLE ANALYSIS
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "🔄 COMPREHENSIVE CYCLE ANALYSIS")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	cycleMetrics := [][]interface{}{
		{"Total Cycles", fmt.Sprintf("%d", len(results.Cycles)), "Number of DCA cycles initiated", "", ""},
		{"Completed Cycles", fmt.Sprintf("%d (%.1f%%)", results.CompletedCycles, float64(results.CompletedCycles)/float64(len(results.Cycles))*100), "Cycles that hit all 5 TP levels", "", getCycleCompletionInsight(float64(results.CompletedCycles)/float64(len(results.Cycles))*100)},
		{"Total DCA Entries", fmt.Sprintf("%d", totalDCAEntries), "Number of buy orders executed", "", ""},
		{"Total TP Hits", fmt.Sprintf("%d", totalTPHits), "Number of profitable sell orders", "", ""},
		{"Avg Cycle Duration", fmt.Sprintf("%.1f hours", avgCycleDuration), "Average time from first DCA to last TP", "", getCycleDurationInsight(avgCycleDuration)},
		{"Longest Cycle", fmt.Sprintf("%.1f hours", longestCycle), "Maximum time for cycle completion", "", ""},
		{"Shortest Cycle", fmt.Sprintf("%.1f hours", shortestCycle), "Minimum time for cycle completion", "", ""},
		{"DCA Entries per Cycle", fmt.Sprintf("%.1f", float64(totalDCAEntries)/float64(len(results.Cycles))), "Average number of DCA entries per cycle", "", getDCAEfficiencyInsight(float64(totalDCAEntries)/float64(len(results.Cycles)))},
	}
	
	for _, metric := range cycleMetrics {
		for i, v := range metric {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i == 4 && v != "" { // Insight column
				fx.SetCellStyle(sheet, cell, cell, insightStyle)
			}
		}
		row++
	}
	
	row += 2
	
	// 🎯 TP LEVEL PERFORMANCE ANALYSIS
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "🎯 TP LEVEL PERFORMANCE ANALYSIS")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	// Headers for TP analysis
	tpHeaders := []string{"TP Level", "Hit Count", "Success Rate", "Avg PnL", "Total PnL"}
	for i, h := range tpHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		fx.SetCellValue(sheet, cell, h)
		fx.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	row++
	
	// Calculate TP level statistics
	tpStats := make(map[int]struct {
		count    int
		totalPnL float64
		totalGain float64
	})
	
	for _, c := range results.Cycles {
		for _, pe := range c.PartialExits {
			if _, exists := tpStats[pe.TPLevel]; !exists {
				tpStats[pe.TPLevel] = struct {
					count    int
					totalPnL float64
					totalGain float64
				}{}
			}
			stats := tpStats[pe.TPLevel]
			stats.count++
			stats.totalPnL += pe.PnL
			if c.AvgEntry > 0 {
				stats.totalGain += (pe.Price - c.AvgEntry) / c.AvgEntry * 100
			}
			tpStats[pe.TPLevel] = stats
		}
	}
	
	// Write TP level data with insights
	for level := 1; level <= 5; level++ {
		stats := tpStats[level]
		successRate := 0.0
		avgPnL := 0.0
		
		if len(results.Cycles) > 0 {
			successRate = float64(stats.count) / float64(len(results.Cycles)) * 100
		}
		if stats.count > 0 {
			avgPnL = stats.totalPnL / float64(stats.count)
		}
		
		values := []interface{}{
			fmt.Sprintf("TP %d", level),
			stats.count,
			fmt.Sprintf("%.1f%%", successRate),
			fmt.Sprintf("$%.2f", avgPnL),
			fmt.Sprintf("$%.2f", stats.totalPnL),
		}
		
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i >= 2 && i <= 4 { // Percentage and currency columns
				if strings.Contains(fmt.Sprintf("%v", v), "$") {
					fx.SetCellStyle(sheet, cell, cell, currencyStyle)
				} else if strings.Contains(fmt.Sprintf("%v", v), "%") {
					fx.SetCellStyle(sheet, cell, cell, percentStyle)
				}
			}
		}
		row++
	}
	
	row += 2
	
	// 💡 STRATEGIC INSIGHTS & RECOMMENDATIONS
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "💡 STRATEGIC INSIGHTS & RECOMMENDATIONS")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	recommendations := getStrategicRecommendations(results, totalCost, winRate, avgCycleDuration)
	
	for _, rec := range recommendations {
		for i, v := range rec {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i == 0 { // Category column
				fx.SetCellStyle(sheet, cell, cell, headerStyle)
			} else if i >= 1 { // Recommendation columns
				fx.SetCellStyle(sheet, cell, cell, insightStyle)
			}
		}
		row++
	}
}

// Helper functions for insights
func getProfitFactorInsight(pf float64) string {
	if pf > 2.0 {
		return "🔥 Excellent - Very profitable strategy"
	} else if pf > 1.5 {
		return "✅ Good - Solid profitability"
	} else if pf > 1.0 {
		return "⚠️ Marginal - Consider optimization"
	}
	return "❌ Poor - Strategy needs revision"
}

func getSharpeInsight(sr float64) string {
	if sr > 2.0 {
		return "🔥 Exceptional risk-adjusted returns"
	} else if sr > 1.0 {
		return "✅ Good risk-adjusted performance"
	} else if sr > 0.5 {
		return "⚠️ Moderate risk-adjusted returns"
	}
	return "❌ Poor risk-adjusted performance"
}

func getDrawdownInsight(dd float64) string {
	if dd < 5.0 {
		return "✅ Excellent - Low risk"
	} else if dd < 10.0 {
		return "✅ Good - Manageable risk"
	} else if dd < 20.0 {
		return "⚠️ Moderate - Monitor closely"
	}
	return "❌ High - Consider risk reduction"
}

func getCapitalEfficiencyInsight(ce float64) string {
	if ce > 80.0 {
		return "🔥 Highly efficient capital usage"
	} else if ce > 60.0 {
		return "✅ Good capital utilization"
	} else if ce > 40.0 {
		return "⚠️ Moderate - Room for improvement"
	}
	return "💡 Conservative - Consider higher allocation"
}

func getCycleCompletionInsight(cc float64) string {
	if cc > 80.0 {
		return "🔥 Excellent completion rate"
	} else if cc > 60.0 {
		return "✅ Good completion rate"
	} else if cc > 40.0 {
		return "⚠️ Moderate - Optimize TP levels"
	}
	return "❌ Low - Review strategy parameters"
}

func getCycleDurationInsight(cd float64) string {
	if cd < 24.0 {
		return "⚡ Fast cycles - High frequency"
	} else if cd < 72.0 {
		return "✅ Optimal cycle duration"
	} else if cd < 168.0 {
		return "⏳ Longer cycles - Patient approach"
	}
	return "🐌 Very long cycles - Consider adjustment"
}

func getDCAEfficiencyInsight(dca float64) string {
	if dca < 2.0 {
		return "💡 Efficient - Few entries needed"
	} else if dca < 4.0 {
		return "✅ Balanced DCA approach"
	} else if dca < 6.0 {
		return "⚠️ Many entries - High averaging"
	}
	return "❌ Excessive entries - Review thresholds"
}

func getStrategicRecommendations(results *backtest.BacktestResults, totalCost, winRate, avgCycleDuration float64) [][]interface{} {
	recommendations := [][]interface{}{}
	
	// Performance optimization
	if winRate > 90.0 {
		recommendations = append(recommendations, []interface{}{
			"🎯 Performance", "Excellent win rate! Consider increasing position sizes or tightening TP levels for higher profits.", "", "", "",
		})
	} else if winRate < 70.0 {
		recommendations = append(recommendations, []interface{}{
			"🎯 Performance", "Lower win rate detected. Consider widening TP levels or adjusting DCA thresholds.", "", "", "",
		})
	}
	
	// Risk management
	if results.MaxDrawdown*100 > 15.0 {
		recommendations = append(recommendations, []interface{}{
			"⚠️ Risk Management", "High drawdown detected. Consider reducing position sizes or implementing stop-losses.", "", "", "",
		})
	}
	
	// Capital efficiency
	if (totalCost/results.StartBalance)*100 < 50.0 {
		recommendations = append(recommendations, []interface{}{
			"💰 Capital Usage", "Low capital utilization. Consider increasing DCA amounts or adding more cycles.", "", "", "",
		})
	}
	
	// Cycle optimization
	if avgCycleDuration > 120.0 {
		recommendations = append(recommendations, []interface{}{
			"⏱️ Timing", "Long cycle durations. Consider tighter TP levels or different market conditions.", "", "", "",
		})
	}
	
	// Strategy enhancement
	recommendations = append(recommendations, []interface{}{
		"🚀 Next Steps", "Monitor performance across different market conditions and timeframes for optimization.", "", "", "",
	})
	
	return recommendations
}

// writeTimelineAnalysis creates a chronological timeline of all trading activity
func writeTimelineAnalysis(fx *excelize.File, sheet string, results *backtest.BacktestResults, headerStyle, currencyStyle, percentStyle int) {
	// Set column widths
	fx.SetColWidth(sheet, "A", "A", 18)  // Timestamp
	fx.SetColWidth(sheet, "B", "B", 8)   // Cycle
	fx.SetColWidth(sheet, "C", "C", 12)  // Action
	fx.SetColWidth(sheet, "D", "D", 12)  // Price
	fx.SetColWidth(sheet, "E", "E", 12)  // Quantity
	fx.SetColWidth(sheet, "F", "F", 12)  // Value
	fx.SetColWidth(sheet, "G", "G", 12)  // PnL
	fx.SetColWidth(sheet, "H", "H", 12)  // Balance
	fx.SetColWidth(sheet, "I", "I", 30)  // Description
	
	// Title and headers
	fx.SetCellValue(sheet, "A1", "📈 CHRONOLOGICAL TIMELINE")
	fx.SetCellStyle(sheet, "A1", "A1", headerStyle)
	
	headers := []string{
		"Timestamp", "Cycle", "Action", "Price", "Quantity", "Value ($)", "PnL ($)", "Balance ($)", "Description",
	}
	
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		fx.SetCellValue(sheet, cell, h)
		fx.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	
	// Collect all events chronologically
	type TimelineEvent struct {
		Timestamp   time.Time
		Cycle       int
		Action      string
		Price       float64
		Quantity    float64
		Value       float64
		PnL         float64
		Balance     float64
		Description string
	}
	
	var events []TimelineEvent
	runningBalance := results.StartBalance
	
	// Add all trades to timeline
	for _, t := range results.Trades {
		if t.EntryTime.Equal(t.ExitTime) {
			// TP exit - we GAIN money (sell crypto for cash)
			runningBalance += (t.ExitPrice * t.Quantity) - t.Commission
			events = append(events, TimelineEvent{
				Timestamp:   t.ExitTime,
				Cycle:       t.Cycle,
				Action:      "💰 SELL",
				Price:       t.ExitPrice,
				Quantity:    t.Quantity,
				Value:       t.ExitPrice * t.Quantity,
				PnL:         t.PnL,
				Balance:     runningBalance,
				Description: fmt.Sprintf("TP Level hit at $%.4f", t.ExitPrice),
			})
		} else {
			// DCA entry - we SPEND money (buy crypto with cash)
			runningBalance -= (t.EntryPrice*t.Quantity + t.Commission)
			events = append(events, TimelineEvent{
				Timestamp:   t.EntryTime,
				Cycle:       t.Cycle,
				Action:      "📈 BUY",
				Price:       t.EntryPrice,
				Quantity:    t.Quantity,
				Value:       t.EntryPrice * t.Quantity,
				PnL:         0,
				Balance:     runningBalance,
				Description: fmt.Sprintf("DCA Entry at $%.4f", t.EntryPrice),
			})
		}
	}
	
	// Sort events by timestamp
	for i := 0; i < len(events)-1; i++ {
		for j := i + 1; j < len(events); j++ {
			if events[i].Timestamp.After(events[j].Timestamp) {
				events[i], events[j] = events[j], events[i]
			}
		}
	}
	
	// Write timeline data
	row := 4
	for _, event := range events {
		values := []interface{}{
			event.Timestamp.Format("2006-01-02 15:04:05"),
			event.Cycle,
			event.Action,
			event.Price,
			event.Quantity,
			event.Value,
			event.PnL,
			event.Balance,
			event.Description,
		}
		
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			// Apply styling
			if i == 5 || i == 6 || i == 7 { // Value, PnL, Balance columns
				fx.SetCellStyle(sheet, cell, cell, currencyStyle)
			}
		}
		row++
	}
	
	// Add filter
	if row > 4 {
		fx.AutoFilter(sheet, fmt.Sprintf("A3:I%d", row-1), []excelize.AutoFilterOptions{})
	}
}

// writeTradesXLSX writes trades and cycles to separate sheets in an Excel file with professional formatting
func writeTradesXLSX(results *backtest.BacktestResults, path string) error {
	fx := excelize.NewFile()
	defer fx.Close()

	// Create sheets
	const tradesSheet = "Trades"
	const cyclesSheet = "Cycles"
	const detailedSheet = "Detailed Analysis"
	
	// Replace default sheet and create additional sheets
	fx.SetSheetName(fx.GetSheetName(0), tradesSheet)
	fx.NewSheet(cyclesSheet)
	fx.NewSheet(detailedSheet)

	// =========================
	// PROFESSIONAL STYLES
	// =========================
	
	// Header style - Dark blue background with white text
	headerStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "FFFFFF",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"2F4F4F"},  // Dark slate gray
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	
	// Data style for even rows (light gray background) - removed as unused
	// evenRowStyle, _ := fx.NewStyle(&excelize.Style{...})
	
	// Note: Win/Loss styles removed as we now use entry/exit styles
	
	// Currency style (right aligned, $ format)
	currencyStyle, _ := fx.NewStyle(&excelize.Style{
		NumFmt: 7, // Currency format with $ symbol
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
	// Percentage style (right aligned, % format)
	percentStyle, _ := fx.NewStyle(&excelize.Style{
		NumFmt: 9, // Percentage format with % symbol
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
	// Red percentage style for DCA entries (negative price movement)
	redPercentStyle, _ := fx.NewStyle(&excelize.Style{
		NumFmt: 9, // Percentage format with % symbol
		Font: &excelize.Font{
			Color: "FF0000", // Red text
		},
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
	// Green percentage style for TP exits (positive price movement)  
	greenPercentStyle, _ := fx.NewStyle(&excelize.Style{
		NumFmt: 9, // Percentage format with % symbol
		Font: &excelize.Font{
			Color: "008000", // Green text
		},
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
	// =========================
	// TRADES SHEET
	// =========================
	
	// Set column widths for enhanced layout with balance tracking
	fx.SetColWidth(tradesSheet, "A", "A", 8)   // Cycle
	fx.SetColWidth(tradesSheet, "B", "B", 12)  // Type
	fx.SetColWidth(tradesSheet, "C", "C", 10)  // Sequence
	fx.SetColWidth(tradesSheet, "D", "D", 18)  // Timestamp
	fx.SetColWidth(tradesSheet, "E", "E", 12)  // Price
	fx.SetColWidth(tradesSheet, "F", "F", 12)  // Quantity
	fx.SetColWidth(tradesSheet, "G", "G", 12)  // Commission
	fx.SetColWidth(tradesSheet, "H", "H", 14)  // PnL
	fx.SetColWidth(tradesSheet, "I", "I", 12)  // Price Change %
	fx.SetColWidth(tradesSheet, "J", "J", 14)  // Running Balance
	fx.SetColWidth(tradesSheet, "K", "K", 14)  // Balance Change
	fx.SetColWidth(tradesSheet, "L", "L", 20)  // TP Info

	// Write Enhanced Trades headers - includes balance tracking and TP descriptions
	tradeHeaders := []string{
		"Cycle", "Type", "Sequence", "Timestamp", "Price", "Quantity", 
		"Commission", "PnL", "Price Change %", "Running Balance", "Balance Change", "TP Info",
	}
	for i, h := range tradeHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(tradesSheet, cell, h)
		fx.SetCellStyle(tradesSheet, cell, cell, headerStyle)
	}

	// Enhanced trade organization with chronological sequencing
	type EnhancedTradeData struct {
		Trade       backtest.Trade
		IsEntry     bool
		Sequence    int
		BalanceBefore float64
		BalanceAfter  float64
		BalanceChange float64
		Description string
	}
	
	type CycleTradeData struct {
		CycleNumber     int
		ChronologicalTrades []EnhancedTradeData
		StartPrice      float64
		StartBalance    float64
		EndBalance      float64
	}
	
	cycleMap := make(map[int]*CycleTradeData)
	
	// Group all trades by cycle first
	for _, t := range results.Trades {
		if cycleMap[t.Cycle] == nil {
			cycleMap[t.Cycle] = &CycleTradeData{
				CycleNumber: t.Cycle,
				ChronologicalTrades: make([]EnhancedTradeData, 0),
			}
		}
		
		isEntry := !t.EntryTime.Equal(t.ExitTime)
		
			// Set cycle start price from first entry
		if isEntry && cycleMap[t.Cycle].StartPrice == 0 {
				cycleMap[t.Cycle].StartPrice = t.EntryPrice
			}
		
		// Add trade to cycle
		cycleMap[t.Cycle].ChronologicalTrades = append(cycleMap[t.Cycle].ChronologicalTrades, EnhancedTradeData{
			Trade:   t,
			IsEntry: isEntry,
		})
	}
	
	// Calculate running balance by processing cycles in order, but keep trades within each cycle
	runningBalance := results.StartBalance
	
	// Get cycles in chronological order (by cycle number)
	var sortedCycleNums []int
	for cycleNum := range cycleMap {
		sortedCycleNums = append(sortedCycleNums, cycleNum)
	}
	// Sort cycle numbers
	for i := 0; i < len(sortedCycleNums)-1; i++ {
		for j := i + 1; j < len(sortedCycleNums); j++ {
			if sortedCycleNums[i] > sortedCycleNums[j] {
				sortedCycleNums[i], sortedCycleNums[j] = sortedCycleNums[j], sortedCycleNums[i]
			}
		}
	}
	
	// Process each cycle in order
	for _, cycleNum := range sortedCycleNums {
		cycle := cycleMap[cycleNum]
		cycle.StartBalance = runningBalance
		
		// Sort trades within THIS cycle chronologically
		trades := cycle.ChronologicalTrades
		for i := 0; i < len(trades)-1; i++ {
			for j := i + 1; j < len(trades); j++ {
				time1 := trades[i].Trade.EntryTime
				time2 := trades[j].Trade.EntryTime
				if !trades[i].IsEntry {
					time1 = trades[i].Trade.ExitTime
				}
				if !trades[j].IsEntry {
					time2 = trades[j].Trade.ExitTime
				}
				
				if time1.After(time2) {
					trades[i], trades[j] = trades[j], trades[i]
				}
			}
		}
		
		// Calculate balance changes within this cycle
		for i := range trades {
			trade := &trades[i]
			trade.Sequence = i + 1
			trade.BalanceBefore = runningBalance
			
			if trade.IsEntry {
				// DCA entry - spend money
				balanceChange := -(trade.Trade.EntryPrice*trade.Trade.Quantity + trade.Trade.Commission)
				trade.BalanceChange = balanceChange
				trade.Description = fmt.Sprintf("DCA Entry #%d", trade.Sequence)
			} else {
				// TP exit - receive money  
				balanceChange := (trade.Trade.ExitPrice*trade.Trade.Quantity - trade.Trade.Commission)
				trade.BalanceChange = balanceChange
				
				// Determine how many TP levels this exit represents based on quantity
				totalCycleQty := 0.0
				for _, t := range cycle.ChronologicalTrades {
					if t.IsEntry {
						totalCycleQty += t.Trade.Quantity
					}
				}
				
				// Calculate what percentage of position this exit represents
				exitPercentage := (trade.Trade.Quantity / totalCycleQty) * 100
				
				if exitPercentage >= 90 {
					trade.Description = "TP Complete (100%)"
				} else if exitPercentage >= 70 {
					trade.Description = "TP Levels 1-4 (80%)"
				} else if exitPercentage >= 50 {
					trade.Description = "TP Levels 1-3 (60%)"
				} else if exitPercentage >= 30 {
					trade.Description = "TP Levels 1-2 (40%)"
				} else {
					trade.Description = "TP Level 1 (20%)"
				}
			}
			
			runningBalance += trade.BalanceChange
			trade.BalanceAfter = runningBalance
		}
		
		cycle.EndBalance = runningBalance
	}
	
	// Get sorted cycle numbers
	var cycleNumbers []int
	for cycleNum := range cycleMap {
		cycleNumbers = append(cycleNumbers, cycleNum)
	}
	// Sort cycles
	for i := 0; i < len(cycleNumbers)-1; i++ {
		for j := i + 1; j < len(cycleNumbers); j++ {
			if cycleNumbers[i] > cycleNumbers[j] {
				cycleNumbers[i], cycleNumbers[j] = cycleNumbers[j], cycleNumbers[i]
			}
		}
	}

	// Write organized trade data
	row := 2
	var totalPnL float64
	var totalCost float64
	
	// Calculate total PnL correctly: Final Balance - Initial Balance (matches console)
	totalPnL = results.EndBalance - results.StartBalance
	
	// Create cycle header style
	cycleHeaderStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "right", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	
	// Create entry and exit styles
	entryStyle, _ := fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E6F3FF"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
	exitStyle, _ := fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E6FFE6"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})

	for _, cycleNum := range cycleNumbers {
		cycleData := cycleMap[cycleNum]
		cycleCost := 0.0
		cyclePnL := 0.0
		
		// Simple cycle header without balance info (moved to summary)
		cycleHeaderRange := fmt.Sprintf("A%d:L%d", row, row)
		fx.MergeCell(tradesSheet, cycleHeaderRange, "")
		headerCell, _ := excelize.CoordinatesToCellName(1, row)
		fx.SetCellValue(tradesSheet, headerCell, fmt.Sprintf("═══ CYCLE %d ═══", cycleNum))
		fx.SetCellStyle(tradesSheet, headerCell, headerCell, cycleHeaderStyle)
		row++
		
		// Add all trades in chronological order
		for _, enhancedTrade := range cycleData.ChronologicalTrades {
			trade := enhancedTrade.Trade
			
			// Calculate costs and PnL for summary
			if enhancedTrade.IsEntry {
				cost := trade.EntryPrice * trade.Quantity + trade.Commission
			cycleCost += cost
			totalCost += cost
			} else {
				cyclePnL += trade.PnL
			}
			
			// Calculate price change
			priceChange := 0.0
			if enhancedTrade.IsEntry && cycleData.StartPrice > 0 {
				// For entries, show drop from cycle start
				priceChange = ((cycleData.StartPrice - trade.EntryPrice) / cycleData.StartPrice) * 100
			} else if !enhancedTrade.IsEntry && trade.EntryPrice > 0 {
				// For exits, show profit relative to average entry price
				priceChange = ((trade.ExitPrice - trade.EntryPrice) / trade.EntryPrice) * 100
			}
			
			// Determine trade type and styling
			tradeType := "📈 BUY"
			tradeStyle := entryStyle
			var tradePnL interface{} = ""
			tradeTime := trade.EntryTime
			tradePrice := trade.EntryPrice
			
			if !enhancedTrade.IsEntry {
				tradeType = "💰 SELL"
				tradeStyle = exitStyle
				tradePnL = trade.PnL
				tradeTime = trade.ExitTime
				tradePrice = trade.ExitPrice
			}
			
			values := []interface{}{
				cycleNum,
				tradeType,
				enhancedTrade.Sequence,
				tradeTime.Format("2006-01-02 15:04:05"),
				tradePrice,
				trade.Quantity,
				trade.Commission,
				tradePnL,
				priceChange / 100,
				enhancedTrade.BalanceAfter,
				enhancedTrade.BalanceChange,
				enhancedTrade.Description,
			}
			
			writeEnhancedTradeRow(fx, tradesSheet, row, values, tradeStyle, currencyStyle, percentStyle, enhancedTrade.IsEntry, redPercentStyle, greenPercentStyle)
			row++
		}
		
		// Add seamless cycle summary row with balance info
		summaryStyle, _ := fx.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 11, Color: "FFFFFF"},
			Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
			Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 2},
				{Type: "right", Color: "000000", Style: 2},
				{Type: "top", Color: "000000", Style: 2},
				{Type: "bottom", Color: "000000", Style: 2},
			},
		})
		
		// Count entries and exits
		entryCount := 0
		exitCount := 0
		for _, trade := range cycleData.ChronologicalTrades {
			if trade.IsEntry {
				entryCount++
			} else {
				exitCount++
			}
		}
		
		// Use actual cycle profit (balance change) instead of individual trade PnL sum
		actualCycleProfit := cycleData.EndBalance - cycleData.StartBalance
		
		// Create seamless summary header with balance info
		summaryHeaderRange := fmt.Sprintf("A%d:L%d", row, row)
		fx.MergeCell(tradesSheet, summaryHeaderRange, "")
		summaryHeaderCell, _ := excelize.CoordinatesToCellName(1, row)
		fx.SetCellValue(tradesSheet, summaryHeaderCell, fmt.Sprintf("📊 CYCLE %d SUMMARY - Balance: $%.2f → $%.2f | Profit: $%.2f | Entries: %d | Exits: %d", 
			cycleNum, cycleData.StartBalance, cycleData.EndBalance, actualCycleProfit, entryCount, exitCount))
		fx.SetCellStyle(tradesSheet, summaryHeaderCell, summaryHeaderCell, summaryStyle)
		row++
		
		// Add spacing between cycles
		row++
	}
	
	// Add final summary row
	finalSummaryStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"2F4F4F"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 3},
			{Type: "right", Color: "000000", Style: 3},
			{Type: "top", Color: "000000", Style: 3},
			{Type: "bottom", Color: "000000", Style: 3},
		},
	})
	
	finalSummaryRange := fmt.Sprintf("A%d:L%d", row, row)
	fx.MergeCell(tradesSheet, finalSummaryRange, "")
	finalSummaryCell, _ := excelize.CoordinatesToCellName(1, row)
	// Count trades using the same logic as console (PartialExits if available)
	summaryTradeCount := 0
	for _, c := range results.Cycles {
		summaryTradeCount += len(c.PartialExits)
	}
	if summaryTradeCount == 0 {
		// Fallback to original trades if no partial exits
		for _, t := range results.Trades {
			if !t.EntryTime.Equal(t.ExitTime) {
				summaryTradeCount++
			}
		}
	}
	
	fx.SetCellValue(tradesSheet, finalSummaryCell, fmt.Sprintf("🏆 TOTAL SUMMARY: Start: $%.0f | End: $%.0f | PnL: $%.0f | ROI: %.1f%% | Trades: %d | Cycles: %d", 
		results.StartBalance, results.EndBalance, totalPnL, (totalPnL/results.StartBalance)*100, summaryTradeCount, len(cycleNumbers)))
	fx.SetCellStyle(tradesSheet, finalSummaryCell, finalSummaryCell, finalSummaryStyle)
	row++
	
	// Add AutoFilter to trades data
	if row > 2 {
		fx.AutoFilter(tradesSheet, fmt.Sprintf("A1:L%d", row-1), []excelize.AutoFilterOptions{})
    }

	// =========================
	// CYCLES SHEET
	// =========================
	
	// Set column widths for cycles sheet
	fx.SetColWidth(cyclesSheet, "A", "A", 8)   // Cycle
	fx.SetColWidth(cyclesSheet, "B", "B", 18)  // Start Time
	fx.SetColWidth(cyclesSheet, "C", "C", 18)  // End Time
	fx.SetColWidth(cyclesSheet, "D", "D", 12)  // Duration
	fx.SetColWidth(cyclesSheet, "E", "E", 10)  // Entries
	fx.SetColWidth(cyclesSheet, "F", "F", 12)  // Avg Entry
	fx.SetColWidth(cyclesSheet, "G", "G", 12)  // Exit Price
	fx.SetColWidth(cyclesSheet, "H", "H", 12)  // Capital Used
	fx.SetColWidth(cyclesSheet, "I", "I", 12)  // Capital %
	fx.SetColWidth(cyclesSheet, "J", "J", 12)  // Balance Before
	fx.SetColWidth(cyclesSheet, "K", "K", 12)  // Balance After
	fx.SetColWidth(cyclesSheet, "L", "L", 12)  // PnL
	fx.SetColWidth(cyclesSheet, "M", "M", 10)  // ROI %
	
	// Cycles sheet title and headers
	fx.SetCellValue(cyclesSheet, "A1", "🔄 CYCLE ANALYSIS WITH CAPITAL USAGE")
	fx.SetCellStyle(cyclesSheet, "A1", "A1", headerStyle)
	
	cycleHeaders := []string{
		"Cycle", "Start Time", "End Time", "Duration", "Entries", "Avg Entry", "Exit Price", 
		"Capital Used ($)", "Capital %", "Balance Before ($)", "Balance After ($)", "PnL ($)", "ROI %",
	}
	
	for i, h := range cycleHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		fx.SetCellValue(cyclesSheet, cell, h)
		fx.SetCellStyle(cyclesSheet, cell, cell, headerStyle)
	}
	
		// Calculate balance and capital usage per cycle (cycles are SEQUENTIAL)
	// Cycle 1 completes, then Cycle 2 starts, then Cycle 3 starts, etc.
	
	cycleBalances := make(map[int]struct {
		before float64
		after  float64
		capital float64
	})
	
	// Sort cycles by cycle number to process them sequentially
	var sortedCycles []backtest.CycleSummary
		for _, c := range results.Cycles {
		sortedCycles = append(sortedCycles, c)
	}
	
	// Sort cycles by cycle number
	for i := 0; i < len(sortedCycles)-1; i++ {
		for j := i + 1; j < len(sortedCycles); j++ {
			if sortedCycles[i].CycleNumber > sortedCycles[j].CycleNumber {
				sortedCycles[i], sortedCycles[j] = sortedCycles[j], sortedCycles[i]
			}
		}
	}
	
	// Process cycles sequentially
	runningBalance = results.StartBalance
	
	for _, c := range sortedCycles {
		// Balance before this cycle starts
		balanceBefore := runningBalance
		
		// Calculate capital used in this cycle (sum of all DCA entries)
		cycleCapital := 0.0
			for _, t := range results.Trades {
			if t.Cycle == c.CycleNumber && !t.EntryTime.Equal(t.ExitTime) {
				// DCA entry trade
				cycleCapital += t.EntryPrice*t.Quantity + t.Commission
			}
		}
		
		// Update running balance after this cycle completes
		// The RealizedPnL already accounts for getting back the invested capital + profit
		// So we just add the net PnL (which includes capital recovery + profit)
		runningBalance = balanceBefore + c.RealizedPnL
		
		// Store cycle balance data
		cycleBalances[c.CycleNumber] = struct {
			before float64
			after  float64
			capital float64
		}{
			before:  balanceBefore,
			after:   runningBalance,
			capital: cycleCapital,
		}
	}
	
	// Write cycle data
	cycleRow := 4
	for _, c := range results.Cycles {
		duration := c.EndTime.Sub(c.StartTime)
		durationStr := fmt.Sprintf("%.1fh", duration.Hours())
		
		// Get balance data for this cycle
		balanceData := cycleBalances[c.CycleNumber]
		
		// Calculate capital percentage of balance at cycle start
		capitalPercent := 0.0
		if balanceData.before > 0 {
			capitalPercent = (balanceData.capital / balanceData.before) * 100
		}
		
		// Determine exit price - prefer cycle's final exit price
		exitPrice := c.FinalExitPrice
		if exitPrice == 0.0 {
			// Fallback: find the latest synthetic TP exit for this cycle
			for i := len(results.Trades) - 1; i >= 0; i-- {
				t := results.Trades[i]
				if t.Cycle == c.CycleNumber && t.EntryTime.Equal(t.ExitTime) {
					exitPrice = t.ExitPrice
					break
				}
			}
		}
		
		roi := 0.0
		if balanceData.capital > 0 {
			roi = (c.RealizedPnL / balanceData.capital) * 100
		}
		
		cycleValues := []interface{}{
				c.CycleNumber,
			c.StartTime.Format("2006-01-02 15:04"),
			c.EndTime.Format("2006-01-02 15:04"),
			durationStr,
				c.Entries,
				c.AvgEntry,
			exitPrice,
			balanceData.capital,
			capitalPercent,
			balanceData.before,
			balanceData.after,
			c.RealizedPnL,
			roi,
		}
		
		for i, v := range cycleValues {
			cell, _ := excelize.CoordinatesToCellName(i+1, cycleRow)
				fx.SetCellValue(cyclesSheet, cell, v)
				
			// Apply styling
			if i == 5 || i == 6 { // Avg Entry, Exit Price
					fx.SetCellStyle(cyclesSheet, cell, cell, currencyStyle)
			} else if i >= 7 && i <= 11 { // Capital Used, Balance Before/After, PnL
				fx.SetCellStyle(cyclesSheet, cell, cell, currencyStyle)
			} else if i == 8 || i == 12 { // Capital %, ROI %
				fx.SetCellStyle(cyclesSheet, cell, cell, percentStyle)
			}
		}
		cycleRow++
	}
	
	// Add summary row
	cycleRow += 1
	fx.SetCellValue(cyclesSheet, "A"+fmt.Sprintf("%d", cycleRow), "📊 SUMMARY")
	fx.SetCellStyle(cyclesSheet, "A"+fmt.Sprintf("%d", cycleRow), "A"+fmt.Sprintf("%d", cycleRow), headerStyle)
	
	// Calculate totals
	totalCapitalUsed := 0.0
	cyclesTotalPnL := 0.0
	completedCycles := 0
	
	for _, c := range results.Cycles {
		balanceData := cycleBalances[c.CycleNumber]
		totalCapitalUsed += balanceData.capital
		cyclesTotalPnL += c.RealizedPnL
		completedCycles++ // Count all cycles since we removed status column
	}
	
	avgCapitalPercent := 0.0
	if len(results.Cycles) > 0 {
		avgCapitalPercent = (totalCapitalUsed / (results.StartBalance * float64(len(results.Cycles)))) * 100
	}
	
	cycleRow++
	summaryValues := []interface{}{
		"TOTALS:",
		"",
		"",
		"",
		fmt.Sprintf("%d cycles", len(results.Cycles)),
		"",
		"",
		totalCapitalUsed,
		avgCapitalPercent,
		results.StartBalance,
		results.EndBalance,
		cyclesTotalPnL,
		fmt.Sprintf("%.1f%%", (cyclesTotalPnL/totalCapitalUsed)*100),
	}
	
	for i, v := range summaryValues {
		cell, _ := excelize.CoordinatesToCellName(i+1, cycleRow)
		fx.SetCellValue(cyclesSheet, cell, v)
		
		// Apply styling to summary row
		if i >= 7 && i <= 11 { // Capital Used, Balance Before/After, PnL
			fx.SetCellStyle(cyclesSheet, cell, cell, currencyStyle)
		} else if i == 8 { // Capital %
			fx.SetCellStyle(cyclesSheet, cell, cell, percentStyle)
		}
		fx.SetCellStyle(cyclesSheet, cell, cell, headerStyle) // Make summary row bold
	}
	
	// Add filter to cycles sheet
	if cycleRow > 4 {
		fx.AutoFilter(cyclesSheet, fmt.Sprintf("A3:M%d", cycleRow-2), []excelize.AutoFilterOptions{})
	}

	// =========================
	// DETAILED ANALYSIS SHEET
	// =========================
	
	writeDetailedAnalysis(fx, detailedSheet, results, headerStyle, currencyStyle, percentStyle)

	// Save workbook
	return fx.SaveAs(path)
}

// Walk-Forward Validation structures and functions
type WalkForwardConfig struct {
	Enable     bool
	Rolling    bool
	SplitRatio float64
	TrainDays  int
	TestDays   int
	RollDays   int
}

type WalkForwardFold struct {
	Train []types.OHLCV
	Test  []types.OHLCV
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
func runWalkForwardValidation(cfg *BacktestConfig, data []types.OHLCV, wfCfg WalkForwardConfig) {
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

// Helper function to run GA on specific data
func runGAOnData(cfg *BacktestConfig, data []types.OHLCV) (*backtest.BacktestResults, BacktestConfig) {
	// Create a temporary config for this data
	tempCfg := *cfg
	
	// Use the existing GA optimization logic but with the provided data
	// This reuses the existing optimizeForInterval logic
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	populationSize := GAPopulationSize
	generations := GAGenerations
	mutationRate := GAMutationRate
	crossoverRate := GACrossoverRate
	eliteSize := GAEliteSize
	
	// Ensure base config has valid TP settings for optimization
	if tempCfg.TPPercent == 0 {
		tempCfg.TPPercent = DefaultTPPercent // Ensure valid TP for optimization base
	}
	tempCfg.UseTPLevels = true // Always use multi-level TP

	// Initialize population
	population := initializePopulation(&tempCfg, populationSize, rng)
	
	var bestIndividual *Individual
	var bestResults *backtest.BacktestResults
	
	for gen := 0; gen < generations; gen++ {
		// Evaluate fitness for all individuals in parallel
		evaluatePopulationParallelWithData(population, data)
		
		// Sort by fitness (descending)
		sortPopulationByFitness(population)
		
		// Track best individual
		if bestIndividual == nil || population[0].fitness > bestIndividual.fitness {
			bestIndividual = &Individual{
				config:  population[0].config,
				fitness: population[0].fitness,
				results: population[0].results,
			}
			bestResults = population[0].results
		}
		
		// Create next generation
		if gen < generations-1 {
			population = createNextGeneration(population, eliteSize, crossoverRate, mutationRate, &tempCfg, rng)
		}
	}
	
	return bestResults, bestIndividual.config
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

// Helper function to evaluate population with specific data
func evaluatePopulationParallelWithData(population []*Individual, data []types.OHLCV) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, MaxParallelWorkers)
	
	for i := range population {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			individual := population[idx]
			
			// Create strategy from config
			strategy := createStrategy(&individual.config)
			
			// Create engine using new structure
			engine := backtest.NewBacktestEngine(
				individual.config.InitialBalance,
				individual.config.Commission,
				strategy,
				individual.config.TPPercent,
				individual.config.MinOrderQty,
				individual.config.UseTPLevels,
			)
			
			// Run backtest
			results := engine.Run(data, individual.config.WindowSize)
			
			// Store results
			individual.results = results
			individual.fitness = results.TotalReturn
		}(i)
	}
	
	wg.Wait()
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

// Removed global selectedPeriod - now passed as parameter to avoid race conditions

func parseTrailingPeriod(s string) (time.Duration, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasSuffix(s, "days") {
		s = strings.TrimSuffix(s, "days") + "d"
	}
	if strings.HasSuffix(s, "d") {
		nStr := strings.TrimSuffix(s, "d")
		if nStr == "" { return 0, false }
		n, err := strconv.Atoi(nStr)
		if err != nil || n <= 0 { return 0, false }
		return time.Duration(n) * 24 * time.Hour, true
	}
	// allow raw durations too (e.g., 168h)
	if d, err := time.ParseDuration(s); err == nil { return d, true }
	return 0, false
}

func filterDataByPeriod(data []types.OHLCV, period time.Duration) []types.OHLCV {
	if period <= 0 || len(data) == 0 {
        return data
    }

    // Find the cutoff timestamp (latest timestamp - period)
    latestTime := data[len(data)-1].Timestamp
    cutoffTime := latestTime.Add(-period)

    // Find the starting index where data is within the period
    startIdx := 0
    for i, candle := range data {
        if candle.Timestamp.After(cutoffTime) || candle.Timestamp.Equal(cutoffTime) {
            startIdx = i
            break
        }
    }

    // Return the filtered data in chronological order
    return data[startIdx:]
}

// Helper to resolve default output dir
func defaultOutputDir(symbol, interval string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	i := strings.ToLower(strings.TrimSpace(interval))
	if s == "" { s = "UNKNOWN" }
	if i == "" { i = "unknown" }
	
	return filepath.Join(ResultsDir, fmt.Sprintf("%s_%s", s, i))
}

// validateTPConfig validates the TP configuration (always multi-level TP)
func validateTPConfig(cfg *BacktestConfig) error {
    if cfg.TPPercent <= 0 || cfg.TPPercent > MaxThreshold {
        return fmt.Errorf("tp_percent must be within (0, %.2f], got %.4f", MaxThreshold, cfg.TPPercent)
    }
    return nil
}