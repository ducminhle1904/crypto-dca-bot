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
	
	// Indicator inclusion
	Indicators     []string `json:"indicators"`

	// Take-profit configuration
	TPPercent      float64 `json:"tp_percent"`
    Cycle         bool    `json:"cycle"`
	
	// Minimum lot size for realistic simulation (e.g., 0.01 for BTCUSDT)
	MinOrderQty    float64 `json:"min_order_qty"`
}

// NestedConfig represents the new nested configuration format for output
type NestedConfig struct {
	Strategy      StrategyConfig      `json:"strategy"`
	Exchange      ExchangeConfig      `json:"exchange"`
	Risk          RiskConfig          `json:"risk"`
	Notifications NotificationsConfig `json:"notifications"`
}

type StrategyConfig struct {
	Symbol         string             `json:"symbol"`
	BaseAmount     float64            `json:"base_amount"`
	MaxMultiplier  float64            `json:"max_multiplier"`
	PriceThreshold float64            `json:"price_threshold"`
	Interval       string             `json:"interval"`
	WindowSize     int                `json:"window_size"`
	TPPercent      float64            `json:"tp_percent"`
	Cycle          bool               `json:"cycle"`
	Indicators     []string           `json:"indicators"`
	RSI            RSIConfig          `json:"rsi"`
	MACD           MACDConfig         `json:"macd"`
	BollingerBands BollingerBandsConfig `json:"bollinger_bands"`
	EMA            EMAConfig          `json:"ema"`
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

// findDataFile attempts to locate data files for a specific exchange
// Structure: data/{exchange}/{category}/{symbol}/{interval}/candles.csv
func findDataFile(dataRoot, exchange, symbol, interval string) string {
	symbol = strings.ToUpper(symbol)
	
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
		path := filepath.Join(dataRoot, exchange, category, symbol, interval, "candles.csv")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// If no existing file found, return the preferred structure (first category for the exchange)
	return filepath.Join(dataRoot, exchange, categories[0], symbol, interval, "candles.csv")
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
		optimize       = flag.Bool("optimize", false, "Run parameter optimization using genetic algorithm")
		allIntervals   = flag.Bool("all-intervals", false, "Scan data root for all intervals for the given symbol and run per-interval backtests/optimizations")
		dataRoot       = flag.String("data-root", DefaultDataRoot, "Root folder containing <EXCHANGE>/<CATEGORY>/<SYMBOL>/<INTERVAL>/candles.csv")
		periodStr      = flag.String("period", "", "Limit data to trailing window (e.g., 7d,30d,180d,365d or Nd)")
		consoleOnly    = flag.Bool("console-only", false, "Only display results in console, do not write files (best.json, trades.xlsx)")
		envFile        = flag.String("env", ".env", "Environment file path for Bybit API credentials")
	)
	
	flag.Parse()

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
		*windowSize, *baseAmount, *maxMultiplier, *priceThreshold)

	// Cycle is always enabled (this system always uses cycle mode)
	cfg.Cycle = true
	// TP will be determined by optimization or use sensible default if not set
	if !*optimize && cfg.TPPercent == 0 {
		cfg.TPPercent = DefaultTPPercent // Default 2% TP when not optimizing and TP not set
	}

	// Capture and parse trailing period window
	var selectedPeriod time.Duration
	if s := strings.TrimSpace(*periodStr); s != "" {
		if d, ok := parseTrailingPeriod(s); ok {
			selectedPeriod = d
		}
	}

	// Set default indicators if not specified in config file
	if len(cfg.Indicators) == 0 {
		cfg.Indicators = []string{"rsi", "macd", "bb", "ema"}
	}

	// Resolve data file from symbol/interval if not explicitly provided and not scanning all intervals
	if !*allIntervals && strings.TrimSpace(cfg.DataFile) == "" {
		sym := strings.ToUpper(cfg.Symbol)
		cfg.DataFile = findDataFile(*dataRoot, *exchange, sym, *intervalFlag)
	}
	
	// Fetch minimum order quantity from Bybit before backtesting
	if err := fetchAndSetMinOrderQty(cfg); err != nil {
		log.Printf("Warning: Could not fetch minimum order quantity from Bybit: %v", err)
		log.Printf("Using default minimum order quantity: %.6f", cfg.MinOrderQty)
	}
	
	if *allIntervals {
		runAcrossIntervals(cfg, *dataRoot, *exchange, *optimize, selectedPeriod, *consoleOnly)
		return
	}
	
	if *optimize {
		var bestResults *backtest.BacktestResults
		var bestConfig BacktestConfig
		
		bestResults, bestConfig = optimizeForInterval(cfg, selectedPeriod)
			fmt.Println("\n\n🏆 OPTIMIZATION RESULTS:")
		
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Best Parameters:\n")
		fmt.Printf("  Indicators:       %s\n", strings.Join(bestConfig.Indicators, ","))
		fmt.Printf("  Base Amount:      $%.2f\n", bestConfig.BaseAmount)
		fmt.Printf("  Max Multiplier:   %.2f\n", bestConfig.MaxMultiplier)
		fmt.Printf("  Price Threshold:  %.2f%%\n", bestConfig.PriceThreshold*100)
		fmt.Printf("  TP Percent:       %.3f%%\n", bestConfig.TPPercent*100)
		fmt.Printf("  Min Order Qty:    %.6f %s (from Bybit)\n", bestConfig.MinOrderQty, bestConfig.Symbol)
		if containsIndicator(bestConfig.Indicators, "rsi") {
			fmt.Printf("  RSI Period:     %d\n", bestConfig.RSIPeriod)
			fmt.Printf("  RSI Oversold:   %.0f\n", bestConfig.RSIOversold)
		}
		if containsIndicator(bestConfig.Indicators, "macd") {
			fmt.Printf("  MACD: fast=%d slow=%d signal=%d\n", bestConfig.MACDFast, bestConfig.MACDSlow, bestConfig.MACDSignal)
		}
		if containsIndicator(bestConfig.Indicators, "bb") {
			fmt.Printf("  BB: period=%d std=%.2f\n", bestConfig.BBPeriod, bestConfig.BBStdDev)
		}
		if containsIndicator(bestConfig.Indicators, "ema") {
			fmt.Printf("  EMA Period:     %d\n", bestConfig.EMAPeriod)
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
	windowSize int, baseAmount, maxMultiplier float64, priceThreshold float64) *BacktestConfig {
	
	cfg := &BacktestConfig{
		DataFile:       dataFile,
		Symbol:         symbol,
		InitialBalance: balance,
		Commission:     commission,
		WindowSize:     windowSize,
		BaseAmount:     baseAmount,
		MaxMultiplier:  maxMultiplier,
		PriceThreshold: priceThreshold,
		RSIPeriod:      DefaultRSIPeriod,
		RSIOversold:    DefaultRSIOversold,
		RSIOverbought:  DefaultRSIOverbought,
		MACDFast:       DefaultMACDFast,
		MACDSlow:       DefaultMACDSlow,
		MACDSignal:     DefaultMACDSignal,
		BBPeriod:       DefaultBBPeriod,
		BBStdDev:       DefaultBBStdDev,
		EMAPeriod:      DefaultEMAPeriod,
		Indicators:     nil,
		TPPercent:      0, // Default to 0 TP, will be set later if needed
		MinOrderQty:    DefaultMinOrderQty, // Default minimum order quantity
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
	
	// Validate configuration parameters
	if err := validateConfig(cfg); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}
	
	return cfg
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
	
	if cfg.MinOrderQty < 0 {
		return fmt.Errorf("minimum order quantity must be non-negative, got: %.6f", cfg.MinOrderQty)
	}
	
	return nil
}

func runAcrossIntervals(cfg *BacktestConfig, dataRoot, exchange string, optimize bool, selectedPeriod time.Duration, consoleOnly bool) {
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
			res, bestCfg = optimizeForInterval(&cfgCopy, selectedPeriod)
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
	fmt.Println("Interval | Return% | Trades | Base$ | MaxMult | TP% | Threshold% | MinQty | RSI(p/ov) | Indicators")
	for _, r := range resultsByInterval {
		c := r.OptimizedCfg
		rsiInfo := "-/-"
		if containsIndicator(c.Indicators, "rsi") {
			rsiInfo = fmt.Sprintf("%d/%.0f", c.RSIPeriod, c.RSIOversold)
		}
		fmt.Printf("%-8s | %7.2f | %6d | %5.0f | %7.2f | %5.2f | %8.2f | %6.3f | %8s | %s\n",
			r.Interval,
			r.Results.TotalReturn*100,
			r.Results.TotalTrades,
			c.BaseAmount,
			c.MaxMultiplier,
			c.TPPercent*100,
			c.PriceThreshold*100,
			c.MinOrderQty,
			rsiInfo,
			strings.Join(c.Indicators, ","),
		)
	}
	best := resultsByInterval[bestIdx]
	fmt.Printf("\nBest interval: %s (Return %.2f%%)\n", best.Interval, best.Results.TotalReturn*100)
	fmt.Printf("Best settings -> Indicators: %s | Base: $%.0f, MaxMult: %.2f, TP: %.2f%%",
		strings.Join(best.OptimizedCfg.Indicators, ","),
		best.OptimizedCfg.BaseAmount,
		best.OptimizedCfg.MaxMultiplier,
		best.OptimizedCfg.TPPercent*100,
	)
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
	engine := backtest.NewBacktestEngine(cfg.InitialBalance, cfg.Commission, strat, tp, cfg.MinOrderQty)
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
	if set["rsi"] {
		parts = append(parts, fmt.Sprintf("rsi(p=%d,ov=%.0f)", cfg.RSIPeriod, cfg.RSIOversold))
	}
	if set["macd"] {
		parts = append(parts, fmt.Sprintf("macd(%d/%d/%d)", cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal))
	}
	if set["bb"] {
		parts = append(parts, fmt.Sprintf("bb(p=%d,sd=%.2f)", cfg.BBPeriod, cfg.BBStdDev))
	}
	if set["ema"] {
		parts = append(parts, fmt.Sprintf("ema(p=%d)", cfg.EMAPeriod))
	}
	
	// Add price threshold info
	if cfg.PriceThreshold > 0 {
		parts = append(parts, fmt.Sprintf("priceThreshold=%.1f%%", cfg.PriceThreshold*100))
	} else {
		parts = append(parts, "priceThreshold=disabled")
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
		
		// TP candidates
		if cfg.Cycle {
			individual.config.TPPercent = randomChoice(OptimizationRanges.TPCandidates, rng)
		} else {
			individual.config.TPPercent = 0
		}
		
		// Always use all indicators - no optimization of indicator combinations
		individual.config.Indicators = []string{"rsi", "macd", "bb", "ema"}
		
		// Always randomize indicator parameters
		individual.config.RSIPeriod = randomChoice(OptimizationRanges.RSIPeriods, rng)
		individual.config.RSIOversold = randomChoice(OptimizationRanges.RSIOversold, rng)
		individual.config.MACDFast = randomChoice(OptimizationRanges.MACDFast, rng)
		individual.config.MACDSlow = randomChoice(OptimizationRanges.MACDSlow, rng)
		individual.config.MACDSignal = randomChoice(OptimizationRanges.MACDSignal, rng)
		individual.config.BBPeriod = randomChoice(OptimizationRanges.BBPeriods, rng)
		individual.config.BBStdDev = randomChoice(OptimizationRanges.BBStdDev, rng)
		individual.config.EMAPeriod = randomChoice(OptimizationRanges.EMAPeriods, rng)
		
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
		if rng.Intn(2) == 0 {
			child.config.Indicators = parent2.config.Indicators
		}
	}
	
	return child
}

// Mutate an individual
func mutate(individual *Individual, rate float64, cfg *BacktestConfig, rng *rand.Rand) {
	if rng.Float64() < rate {
		// Mutate any parameter including indicator parameters
		availableParams := []int{0, 2, 3, 4, 5, 6, 7, 8, 9, 10} // All except TP
		if cfg.Cycle {
			availableParams = append(availableParams, 1) // Add TPPercent if cycle is enabled
		}
		
		switch randomChoice(availableParams, rng) {
		case 0:
			individual.config.MaxMultiplier = randomChoice(OptimizationRanges.Multipliers, rng)
		case 1:
			individual.config.TPPercent = randomChoice(OptimizationRanges.TPCandidates, rng)
		case 2:
			individual.config.PriceThreshold = randomChoice(OptimizationRanges.PriceThresholds, rng)
		case 3:
			individual.config.RSIPeriod = randomChoice(OptimizationRanges.RSIPeriods, rng)
		case 4:
			individual.config.RSIOversold = randomChoice(OptimizationRanges.RSIOversold, rng)
		case 5:
			individual.config.MACDFast = randomChoice(OptimizationRanges.MACDFast, rng)
		case 6:
			individual.config.MACDSlow = randomChoice(OptimizationRanges.MACDSlow, rng)
		case 7:
			individual.config.MACDSignal = randomChoice(OptimizationRanges.MACDSignal, rng)
		case 8:
			individual.config.BBPeriod = randomChoice(OptimizationRanges.BBPeriods, rng)
		case 9:
			individual.config.BBStdDev = randomChoice(OptimizationRanges.BBStdDev, rng)
		case 10:
			individual.config.EMAPeriod = randomChoice(OptimizationRanges.EMAPeriods, rng)
		}
		
		// Ensure indicators always remain the same
		individual.config.Indicators = []string{"rsi", "macd", "bb", "ema"}
		
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
	
	return NestedConfig{
		Strategy: StrategyConfig{
			Symbol:         cfg.Symbol,
			BaseAmount:     cfg.BaseAmount,
			MaxMultiplier:  cfg.MaxMultiplier,
			PriceThreshold: cfg.PriceThreshold,
			Interval:       interval,
			WindowSize:     cfg.WindowSize,
			TPPercent:      cfg.TPPercent,
			Cycle:          cfg.Cycle,
			Indicators:     cfg.Indicators,
			RSI: RSIConfig{
				Period:     cfg.RSIPeriod,
				Oversold:   cfg.RSIOversold,
				Overbought: cfg.RSIOverbought,
			},
			MACD: MACDConfig{
				FastPeriod:   cfg.MACDFast,
				SlowPeriod:   cfg.MACDSlow,
				SignalPeriod: cfg.MACDSignal,
			},
			BollingerBands: BollingerBandsConfig{
				Period: cfg.BBPeriod,
				StdDev: cfg.BBStdDev,
			},
			EMA: EMAConfig{
				Period: cfg.EMAPeriod,
			},
		},
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
		priceDropPct := ((cycleStart - t.EntryPrice) / cycleStart) * 100
		
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

// writeTradesXLSX writes trades and cycles to separate sheets in an Excel file with professional formatting
func writeTradesXLSX(results *backtest.BacktestResults, path string) error {
	fx := excelize.NewFile()
	defer fx.Close()

	// Create sheets
	const tradesSheet = "Trades"
	const cyclesSheet = "Cycles"
	const dashboardSheet = "Dashboard"
	
	// Replace default sheet and create additional sheets
	fx.SetSheetName(fx.GetSheetName(0), tradesSheet)
	fx.NewSheet(cyclesSheet)
	fx.NewSheet(dashboardSheet)

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
	
	// Data style for even rows (light gray background)
	evenRowStyle, _ := fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"F8F8F8"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
	// Win style (green background)
	winStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "006400"},
		Fill: excelize.Fill{
			Type:    "pattern", 
			Color:   []string{"E6FFE6"},
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
	// Loss style (red background)
	lossStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "8B0000"},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"FFE6E6"}, 
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
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
	
	// =========================
	// TRADES SHEET
	// =========================
	
	// Set column widths for better readability
	fx.SetColWidth(tradesSheet, "A", "A", 8)   // Cycle
	fx.SetColWidth(tradesSheet, "B", "C", 18)  // Entry/Exit Time
	fx.SetColWidth(tradesSheet, "D", "D", 12)  // Entry Price
	fx.SetColWidth(tradesSheet, "E", "E", 12)  // Price Drop %
	fx.SetColWidth(tradesSheet, "F", "G", 14)  // Quantity USDT, PnL
	fx.SetColWidth(tradesSheet, "H", "H", 10)  // Win/Loss

	// Write Trades headers
	tradeHeaders := []string{
		"Cycle", "Entry_Time", "Exit_Time", "Entry_Price", 
		"Price_Drop_%", "Quantity_USDT", "Trade_PnL_$", 
		"Win_Loss",
	}
	for i, h := range tradeHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(tradesSheet, cell, h)
		fx.SetCellStyle(tradesSheet, cell, cell, headerStyle)
	}

	// Track cycle start prices for price drop calculation
	cycleData := make(map[int]float64) // cycle -> start price
	
	// Pre-process to find cycle start prices
	for _, t := range results.Trades {
		if _, exists := cycleData[t.Cycle]; !exists {
			cycleData[t.Cycle] = t.EntryPrice
		}
	}

	// Write Trades data with professional formatting
	row := 2
	var cumCost float64
	var totalPnL float64
	for _, t := range results.Trades {
		netCost := t.EntryPrice * t.Quantity
		grossCost := netCost + t.Commission
		cumCost += grossCost
		totalPnL += t.PnL

		// Trade performance calculations
		winLoss := "W"
		if t.PnL < 0 {
			winLoss = "L"
		}
		
		// Strategy context calculations
		cycleStart := cycleData[t.Cycle]
		priceDropPct := ((cycleStart - t.EntryPrice) / cycleStart) * 100

		values := []interface{}{
			t.Cycle,
			t.EntryTime.Format("2006-01-02 15:04:05"),
			t.ExitTime.Format("2006-01-02 15:04:05"),
			t.EntryPrice,
			priceDropPct / 100,      // Convert to decimal for percentage formatting
			math.Ceil(grossCost),    // Rounded up quantity USDT
			math.Ceil(t.PnL),        // Rounded up trade PnL
			winLoss,
		}
		
		// Apply data with appropriate styling
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(tradesSheet, cell, v)
			
			// Apply conditional styling based on content and position
			if i == 7 { // Win/Loss column
				if winLoss == "W" {
					fx.SetCellStyle(tradesSheet, cell, cell, winStyle)
				} else {
					fx.SetCellStyle(tradesSheet, cell, cell, lossStyle)
				}
			} else if i == 4 { // Price Drop % column
				fx.SetCellStyle(tradesSheet, cell, cell, percentStyle)
			} else if i == 5 || i == 6 { // Quantity USDT, PnL columns  
				fx.SetCellStyle(tradesSheet, cell, cell, currencyStyle)
			} else if row%2 == 0 { // Even rows get light gray background
				fx.SetCellStyle(tradesSheet, cell, cell, evenRowStyle)
			}
		}
		row++
	}
	
	// Add AutoFilter to trades data
	if row > 2 {
		fx.AutoFilter(tradesSheet, fmt.Sprintf("A1:H%d", row-1), []excelize.AutoFilterOptions{})
	}
	
	// Add summary row with enhanced styling
	if row > 2 {
		avgTradeReturn := 0.0
		if len(results.Trades) > 0 {
			totalReturn := 0.0
			for _, t := range results.Trades {
				totalReturn += ((t.ExitPrice - t.EntryPrice) / t.EntryPrice) * 100
			}
			avgTradeReturn = totalReturn / float64(len(results.Trades))
		}
		
		// Summary style
		summaryStyle, _ := fx.NewStyle(&excelize.Style{
			Font: &excelize.Font{Bold: true, Size: 10},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{"FFD700"}, // Gold background
				Pattern: 1,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 2},
				{Type: "right", Color: "000000", Style: 2},
				{Type: "top", Color: "000000", Style: 2},
				{Type: "bottom", Color: "000000", Style: 2},
			},
		})
		
		summaryRange := fmt.Sprintf("A%d:H%d", row+1, row+1)
		fx.MergeCell(tradesSheet, summaryRange, "")
		summaryCell, _ := excelize.CoordinatesToCellName(1, row+1)
		fx.SetCellValue(tradesSheet, summaryCell, fmt.Sprintf("SUMMARY: Total PnL: $%.0f | Capital: $%.0f | Avg Return: %.2f%% | Trades: %d", 
			math.Ceil(totalPnL), math.Ceil(cumCost), avgTradeReturn, len(results.Trades)))
		fx.SetCellStyle(tradesSheet, summaryCell, summaryCell, summaryStyle)
	}

	// =========================
	// CYCLES SHEET
	// =========================
	
	// Set column widths for Cycles sheet
	fx.SetColWidth(cyclesSheet, "A", "A", 10)   // Cycle number
	fx.SetColWidth(cyclesSheet, "B", "C", 18)   // Start/End time
	fx.SetColWidth(cyclesSheet, "D", "D", 10)   // Entries
	fx.SetColWidth(cyclesSheet, "E", "G", 12)   // Prices
	fx.SetColWidth(cyclesSheet, "H", "H", 12)   // Duration Hours
	fx.SetColWidth(cyclesSheet, "I", "J", 14)   // PnL, Capital
	
	// Write Cycles headers
	cycleHeaders := []string{"Cycle number", "Start time", "End time", "Entries", "Target price", "Exit price", "Average price", "Duration_Hours", "PnL (USDT)", "Capital (USDT)"}
	for i, h := range cycleHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(cyclesSheet, cell, h)
		fx.SetCellStyle(cyclesSheet, cell, cell, headerStyle)
	}
	
	// Write Cycles data
	row = 2
	if len(results.Cycles) > 0 {
		for _, c := range results.Cycles {
			// compute capital_usdt for this cycle from trades
			capital := 0.0
			for _, t := range results.Trades {
				if t.Cycle == c.CycleNumber {
					capital += t.EntryPrice*t.Quantity + t.Commission
				}
			}
			// Find exit price for this cycle from trades
			exitPrice := 0.0
			for _, t := range results.Trades {
				if t.Cycle == c.CycleNumber {
					exitPrice = t.ExitPrice
					break
				}
			}
			
			// Calculate cycle duration in hours (rounded)
			cycleDuration := c.EndTime.Sub(c.StartTime)
			cycleDurationHours := int(cycleDuration.Hours() + 0.5) // Round up
			
			values := []interface{}{
				c.CycleNumber,
				c.StartTime.Format("2006-01-02 15:04:05"),
				c.EndTime.Format("2006-01-02 15:04:05"),
				c.Entries,
				c.TargetPrice,
				exitPrice,
				c.AvgEntry,
				cycleDurationHours,
				math.Ceil(c.RealizedPnL), // Rounded up PnL
				math.Ceil(capital),       // Rounded up capital
			}
			
			// Apply professional styling to cycles data
			for i, v := range values {
				cell, _ := excelize.CoordinatesToCellName(i+1, row)
				fx.SetCellValue(cyclesSheet, cell, v)
				
				// Apply conditional styling
				if i == 8 { // PnL column - apply win/loss coloring with currency format
					if c.RealizedPnL > 0 { // Use original PnL value for color determination
						// Create a custom style combining currency format with win color
						winCurrencyStyle, _ := fx.NewStyle(&excelize.Style{
							Font: &excelize.Font{Bold: true, Color: "006400"},
							NumFmt: 7, // Currency format
							Fill: excelize.Fill{Type: "pattern", Color: []string{"E6FFE6"}, Pattern: 1},
							Border: []excelize.Border{
								{Type: "left", Color: "E0E0E0", Style: 1},
								{Type: "right", Color: "E0E0E0", Style: 1},
								{Type: "bottom", Color: "E0E0E0", Style: 1},
							},
						})
						fx.SetCellStyle(cyclesSheet, cell, cell, winCurrencyStyle)
					} else {
						// Create a custom style combining currency format with loss color
						lossCurrencyStyle, _ := fx.NewStyle(&excelize.Style{
							Font: &excelize.Font{Bold: true, Color: "8B0000"},
							NumFmt: 7, // Currency format
							Fill: excelize.Fill{Type: "pattern", Color: []string{"FFE6E6"}, Pattern: 1},
							Border: []excelize.Border{
								{Type: "left", Color: "E0E0E0", Style: 1},
								{Type: "right", Color: "E0E0E0", Style: 1},
								{Type: "bottom", Color: "E0E0E0", Style: 1},
							},
						})
						fx.SetCellStyle(cyclesSheet, cell, cell, lossCurrencyStyle)
					}
				} else if i == 9 { // Capital column - apply currency formatting
					fx.SetCellStyle(cyclesSheet, cell, cell, currencyStyle)
				} else if row%2 == 0 { // Even rows
					fx.SetCellStyle(cyclesSheet, cell, cell, evenRowStyle)
				}
			}
			row++
		}
		
		// Add AutoFilter to cycles data
		fx.AutoFilter(cyclesSheet, fmt.Sprintf("A1:J%d", row-1), []excelize.AutoFilterOptions{})
	}

	// =========================
	// DASHBOARD SHEET  
	// =========================
	
	// Dashboard title
	fx.SetColWidth(dashboardSheet, "A", "H", 15)
	fx.SetRowHeight(dashboardSheet, 1, 25)
	
	titleStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   16,
			Color:  "FFFFFF",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"1F4E79"}, // Dark blue
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	
	fx.MergeCell(dashboardSheet, "A1:H1", "")
	fx.SetCellValue(dashboardSheet, "A1", "📊 DCA BACKTESTING DASHBOARD")
	fx.SetCellStyle(dashboardSheet, "A1", "A1", titleStyle)

	// Key Metrics Section
	metricsHeaderStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 12, Color: "2F4F4F"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E8F4FD"}, Pattern: 1},
	})
	
	fx.SetCellValue(dashboardSheet, "A3", "🔑 KEY METRICS")
	fx.SetCellStyle(dashboardSheet, "A3", "A3", metricsHeaderStyle)
	
	// Calculate dashboard metrics
	winTrades := 0
	lossTrades := 0
	totalReturn := 0.0
	for _, t := range results.Trades {
		if t.PnL > 0 {
			winTrades++
		} else {
			lossTrades++
		}
		totalReturn += ((t.ExitPrice - t.EntryPrice) / t.EntryPrice) * 100
	}
	winRate := float64(winTrades) / float64(len(results.Trades)) * 100
	avgReturn := totalReturn / float64(len(results.Trades))
	
	// Metrics data
	metrics := [][]interface{}{
		{"Total Trades:", len(results.Trades)},
		{"Winning Trades:", winTrades},
		{"Win Rate:", fmt.Sprintf("%.1f%%", winRate)},
		{"Total P&L:", fmt.Sprintf("$%.0f", math.Ceil(totalPnL))},
		{"Total Capital:", fmt.Sprintf("$%.0f", math.Ceil(cumCost))},
		{"Avg Return/Trade:", fmt.Sprintf("%.2f%%", avgReturn)},
		{"Total Cycles:", len(results.Cycles)},
		{"ROI:", fmt.Sprintf("%.2f%%", (totalPnL/cumCost)*100)},
	}
	
	metricStyle, _ := fx.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	
	for i, metric := range metrics {
		row := 4 + i
		fx.SetCellValue(dashboardSheet, fmt.Sprintf("A%d", row), metric[0])
		fx.SetCellValue(dashboardSheet, fmt.Sprintf("B%d", row), metric[1])
		fx.SetCellStyle(dashboardSheet, fmt.Sprintf("A%d", row), fmt.Sprintf("B%d", row), metricStyle)
	}
	
	// Create a simple PnL chart using cell formatting (since excelize chart creation is complex)
	fx.SetCellValue(dashboardSheet, "D3", "📈 PERFORMANCE VISUALIZATION")
	fx.SetCellStyle(dashboardSheet, "D3", "D3", metricsHeaderStyle)
	
	// Win/Loss distribution
	fx.SetCellValue(dashboardSheet, "D5", "Wins:")
	fx.SetCellValue(dashboardSheet, "E5", winTrades)
	fx.SetCellValue(dashboardSheet, "D6", "Losses:")  
	fx.SetCellValue(dashboardSheet, "E6", lossTrades)
	
	// Visual representation of win rate
	winBarStyle, _ := fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"00FF00"}, Pattern: 1},
	})
	lossBarStyle, _ := fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FF0000"}, Pattern: 1},
	})
	
	// Create a simple horizontal bar chart representation
	winBars := int(winRate / 10) // Scale to 10 bars max
	lossBars := 10 - winBars
	
	for i := 0; i < winBars; i++ {
		cell := fmt.Sprintf("%c5", 'F'+i)
		fx.SetCellValue(dashboardSheet, cell, "█")
		fx.SetCellStyle(dashboardSheet, cell, cell, winBarStyle)
	}
	for i := 0; i < lossBars; i++ {
		cell := fmt.Sprintf("%c6", 'F'+i)
		fx.SetCellValue(dashboardSheet, cell, "█")
		fx.SetCellStyle(dashboardSheet, cell, cell, lossBarStyle)
	}
	
	// Instructions
	instructStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 9, Color: "666666"},
	})
	
	fx.SetCellValue(dashboardSheet, "A13", "💡 Use filters in Trades and Cycles sheets to analyze specific periods or cycles")
	fx.SetCellStyle(dashboardSheet, "A13", "A13", instructStyle)

	// Save workbook
	return fx.SaveAs(path)
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