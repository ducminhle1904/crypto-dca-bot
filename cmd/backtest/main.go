package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	datamanager "github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/orchestrator"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/reporting"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/validation"
	"github.com/joho/godotenv"
)

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
	
	// File and directory constants
	DefaultDataRoot         = "data"
	DefaultExchange         = "bybit"
	BestConfigFile         = "best.json"
	TradesFile             = "trades.xlsx"
)

// Logging functions for better error reporting and debugging
// Enhanced logging functions with consistent formatting
func logHeader(title string) {
	fmt.Printf("\n🎯 %s\n", strings.ToUpper(title))
	fmt.Printf("%s\n", strings.Repeat("=", len(title)+5))
}

func logSection(title string) {
	fmt.Printf("\n📋 %s\n", title)
	fmt.Printf("%s\n", strings.Repeat("-", len(title)+5))
}

func logInfo(format string, args ...interface{}) {
	fmt.Printf("ℹ️  %s\n", fmt.Sprintf(format, args...))
}

func logError(format string, args ...interface{}) {
	fmt.Printf("❌ %s\n", fmt.Sprintf(format, args...))
}

func logSuccess(format string, args ...interface{}) {
	fmt.Printf("✅ %s\n", fmt.Sprintf(format, args...))
}

// Silent mode control
var silentMode = false

func setSilentMode(silent bool) {
	silentMode = silent
}

func logQuiet(format string, args ...interface{}) {
	if !silentMode {
		fmt.Printf("   %s\n", fmt.Sprintf(format, args...))
	}
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
	
	// Set quiet mode as default
	setSilentMode(true)
	
	// Initialize logging
	logHeader("DCA Bot Backtest")

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
	logSection("Trade Configuration")
	logQuiet("Using 5-level TP system (max: %.2f%%)", cfg.TPPercent*100)

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
	
	// Create orchestrator
	orch := orchestrator.NewOrchestrator()
	
	if *allIntervals {
		// Create walk-forward configuration
		wfConfig := &validation.WalkForwardConfig{
			Enable:     *wfEnable,
			Rolling:    *wfRolling,
			SplitRatio: *wfSplitRatio,
			TrainDays:  *wfTrainDays,
			TestDays:   *wfTestDays,
			RollDays:   *wfRollDays,
		}
		
		results, err := orch.RunMultiIntervalAnalysis(cfg, *dataRoot, *exchange, *optimize, selectedPeriod, wfConfig)
		if err != nil {
			log.Fatalf("Multi-interval analysis failed: %v", err)
		}
		
		// Display results and save outputs
		displayIntervalResults(results, *consoleOnly)
		return
	}
	
	if *optimize {
		// Create walk-forward configuration
		var wfConfig *validation.WalkForwardConfig
		if *wfEnable {
			wfConfig = &validation.WalkForwardConfig{
				Enable:     *wfEnable,
				Rolling:    *wfRolling,
				SplitRatio: *wfSplitRatio,
				TrainDays:  *wfTrainDays,
				TestDays:   *wfTestDays,
				RollDays:   *wfRollDays,
			}
		}
		
		// Run optimization workflow
		bestResults, bestConfig, err := orch.RunOptimizedBacktest(cfg, selectedPeriod, wfConfig)
		if err != nil {
			log.Fatalf("Optimization failed: %v", err)
		}
		logHeader("Optimization Results")
		logSection("Best Parameters")
		logQuiet("Combo Type:       %s", getComboTypeName(bestConfig.UseAdvancedCombo))
		logQuiet("Indicators:       %s", strings.Join(bestConfig.Indicators, ","))
		logQuiet("Base Amount:      $%.2f", bestConfig.BaseAmount)
		logQuiet("Max Multiplier:   %.2f", bestConfig.MaxMultiplier)
		logQuiet("Price Threshold:  %.2f%%", bestConfig.PriceThreshold*100)
		logQuiet("TP System:        5-level (%.2f%% max)", bestConfig.TPPercent*100)
		logQuiet("Min Order Qty:    %.6f %s (from Bybit)", bestConfig.MinOrderQty, bestConfig.Symbol)
		
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
		reporting.PrintBacktestConfigJSON(*convertDCAConfigToBacktestConfig(bestConfig))
		fmt.Println(strings.Repeat("-", 50))
		
		// Save to results folder
		stdDir := reporting.DefaultOutputDir(cfg.Symbol, intervalStr)
			stdBestPath := filepath.Join(stdDir, BestConfigFile)
			stdTradesPath := filepath.Join(stdDir, TradesFile)
		
		// Write standard outputs
		if err := reporting.WriteBacktestConfigJSON(*convertDCAConfigToBacktestConfig(bestConfig), stdBestPath); err != nil {
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
			reporting.PrintBacktestConfigJSON(*convertDCAConfigToBacktestConfig(bestConfig))
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("Usage: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
				cfg.Symbol, intervalStr, bestConfig.BaseAmount, bestConfig.MaxMultiplier, bestConfig.PriceThreshold)
		}
		
		fmt.Println("\nBest Results:")
		// Use the results from optimization
		reporting.OutputConsoleWithContext(bestResults, bestConfig.Symbol, intervalStr)
		return
	}
	
	// Run single backtest
	results, err := orch.RunSingleBacktest(cfg, selectedPeriod)
	if err != nil {
		log.Fatalf("Single backtest failed: %v", err)
	}
	
	// Output results to console with context
	intervalStr := guessIntervalFromPath(cfg.DataFile)
	if intervalStr == "" { intervalStr = "unknown" }
	// Use the new context-aware output that includes the interval information
	// that was removed from runBacktestWithData
	reporting.OutputConsoleWithContext(results, cfg.Symbol, intervalStr)
	
	if !*consoleOnly {
		// Save trades to standard path (reuse intervalStr from above)
		stdDir := reporting.DefaultOutputDir(cfg.Symbol, intervalStr)
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

// displayIntervalResults displays the results from multi-interval analysis
func displayIntervalResults(results *orchestrator.IntervalAnalysisResult, consoleOnly bool) {
	fmt.Println("\n================ Interval Comparison ================")
	fmt.Printf("Symbol: %s\n", results.Symbol)
	fmt.Println("Interval | Return% | Trades | Base$ | MaxMult | TP% | Threshold% | MinQty | Combo | Indicators")
	
	for _, r := range results.Results {
		if r.Error != nil {
			fmt.Printf("%-8s | ERROR: %v\n", r.Interval, r.Error)
			continue
		}
		
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
	
	best := results.BestResult
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
	reporting.PrintBacktestConfigJSON(*convertDCAConfigToBacktestConfig(best.OptimizedCfg))
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Usage: go run cmd/backtest/main.go -config best.json\n")
	fmt.Printf("   or: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
		results.Symbol, best.Interval, best.OptimizedCfg.BaseAmount, best.OptimizedCfg.MaxMultiplier, best.OptimizedCfg.PriceThreshold)

	// Optionally print detailed results for best interval
	fmt.Println("\nBest interval detailed results:")
	reporting.OutputConsole(best.Results)
	
	if !consoleOnly {
		// Write standard outputs under results/<SYMBOL>_<INTERVAL>_mode/
		stdDir := reporting.DefaultOutputDir(results.Symbol, best.Interval)
		stdBestPath := filepath.Join(stdDir, BestConfigFile)
		stdTradesPath := filepath.Join(stdDir, TradesFile)
		if err := reporting.WriteBacktestConfigJSON(*convertDCAConfigToBacktestConfig(best.OptimizedCfg), stdBestPath); err != nil {
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

func loadEnvFile(envFile string) error {
	// Load .env file if it exists
	if _, err := os.Stat(envFile); err == nil {
		return godotenv.Load(envFile)
	}
	return fmt.Errorf("env file %s not found", envFile)
}

// fetchAndSetMinOrderQtyDCA function removed - now handled by orchestrator's BacktestRunner

func guessIntervalFromPath(path string) string {
	dir := filepath.Dir(path)
	return filepath.Base(dir)
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

// writeTradesCSV - Note: This function has been moved to pkg/reporting/csv.go
func writeTradesCSV(results *backtest.BacktestResults, path string) error {
	// If the user requests an Excel file, delegate to Excel writer
	if strings.HasSuffix(strings.ToLower(path), ".xlsx") {
		return reporting.WriteTradesXLSX(results, path)
	}
	
	// Use the new CSV reporter
	return reporting.WriteTradesCSV(results, path)
}