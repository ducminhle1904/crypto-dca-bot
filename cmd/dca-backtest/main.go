package main

import (
	"flag"
	"fmt"
	"log"
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

const (
	AppName    = "DCA Backtest"
	AppVersion = "2.0.0"
	
	// Default values
	DefaultInitialBalance = 500.0
	DefaultCommission     = 0.0005 // 0.05%
	DefaultWindowSize     = 100
	DefaultBaseAmount     = 40.0
	DefaultMaxMultiplier  = 3.0
	DefaultPriceThreshold = 0.02 // 2%
	DefaultPriceThresholdMultiplier = 1.0 // 1.0x = no multiplier
	DefaultTPPercent      = 0.02 // 2%
	DefaultDataRoot       = "data"
	DefaultExchange       = "bybit"
)

func main() {
	// Command line flags
	var (
		configFile       = flag.String("config", "", "Path to DCA configuration file")
		dataFile         = flag.String("data", "", "Path to historical data file")
		symbol           = flag.String("symbol", "BTCUSDT", "Trading symbol")
		interval         = flag.String("interval", "1h", "Data interval (5m, 15m, 1h, 4h, 1d)")
		exchange         = flag.String("exchange", DefaultExchange, "Exchange (bybit, binance)")
		
		// Account settings
		initialBalance   = flag.Float64("balance", DefaultInitialBalance, "Initial balance")
		commission       = flag.Float64("commission", DefaultCommission, "Trading commission (0.0005 = 0.05%)")
		
		// DCA strategy parameters
		baseAmount       = flag.Float64("base-amount", DefaultBaseAmount, "Base DCA amount")
		maxMultiplier    = flag.Float64("max-multiplier", DefaultMaxMultiplier, "Maximum position multiplier")
		priceThreshold   = flag.Float64("price-threshold", DefaultPriceThreshold, "Price drop % for next DCA (0.02 = 2%)")
		priceThresholdMultiplier = flag.Float64("price-threshold-multiplier", DefaultPriceThresholdMultiplier, "Progressive multiplier for DCA spacing (1.1 = 10% increase per level)")
		useAdvancedCombo = flag.Bool("advanced-combo", false, "Use advanced indicators (Hull MA, MFI, Keltner, WaveTrend)")
		
		// Analysis options
		optimize         = flag.Bool("optimize", false, "Run genetic algorithm optimization")
		allIntervals     = flag.Bool("all-intervals", false, "Test all available intervals")
		period           = flag.String("period", "", "Limit data to period (7d, 30d, 180d, 365d)")
		
		// Walk-forward validation
		wfEnable         = flag.Bool("wf-enable", false, "Enable walk-forward validation")
		wfSplitRatio     = flag.Float64("wf-split-ratio", 0.7, "Train/test split (0.7 = 70% train)")
		wfRolling        = flag.Bool("wf-rolling", false, "Use rolling walk-forward")
		wfTrainDays      = flag.Int("wf-train-days", 180, "Training window (days)")
		wfTestDays       = flag.Int("wf-test-days", 60, "Test window (days)")
		wfRollDays       = flag.Int("wf-roll-days", 30, "Roll step (days)")
		
		// Output options
		dataRoot         = flag.String("data-root", DefaultDataRoot, "Data root directory")
		consoleOnly      = flag.Bool("console-only", false, "Console output only (no files)")
		windowSize       = flag.Int("window", DefaultWindowSize, "Analysis window size")
		envFile          = flag.String("env", ".env", "Environment file path")
		
		// Help
		showVersion      = flag.Bool("version", false, "Show version information")
		showHelp         = flag.Bool("help", false, "Show detailed help")
	)
	
	flag.Parse()
	
	// Version and help
	if *showVersion {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		return
	}
	
	if *showHelp {
		printUsageHelp()
		return
	}
	
	// Header
	printHeader()
	
	// Load environment
	loadEnvironment(*envFile)
	
	// Load configuration
	cfg, err := loadDCAConfiguration(*configFile, *dataFile, *symbol, *interval, 
		*initialBalance, *commission, *windowSize, *baseAmount, *maxMultiplier, 
		*priceThreshold, *priceThresholdMultiplier, *useAdvancedCombo)
	if err != nil {
		log.Fatalf("❌ Configuration error: %v", err)
	}
	
	// Parse period filter
	var selectedPeriod time.Duration
	if *period != "" {
		if d, ok := datamanager.ParseTrailingPeriod(*period); ok {
			selectedPeriod = d
		} else {
			log.Fatalf("❌ Invalid period format: %s (use 7d, 30d, 180d, 365d)", *period)
		}
	}
	
	// Create orchestrator
	orch := orchestrator.NewOrchestrator()
	
	// Execute based on options
	if *allIntervals {
		runMultiIntervalAnalysis(orch, cfg, *dataRoot, *exchange, *optimize, selectedPeriod, 
			*wfEnable, *wfSplitRatio, *wfRolling, *wfTrainDays, *wfTestDays, *wfRollDays, *consoleOnly)
	} else if *optimize {
		runOptimization(orch, cfg, selectedPeriod, *wfEnable, *wfSplitRatio, *wfRolling, 
			*wfTrainDays, *wfTestDays, *wfRollDays, *consoleOnly)
	} else {
		runSingleBacktest(orch, cfg, selectedPeriod, *consoleOnly)
	}
}

func printHeader() {
	fmt.Printf("🎯 %s v%s\n", strings.ToUpper(AppName), AppVersion)
	fmt.Printf("%s\n\n", strings.Repeat("=", 50))
}

func printUsageHelp() {
	fmt.Printf(`%s v%s - Enhanced DCA Strategy Backtesting

USAGE:
  %s [OPTIONS]

EXAMPLES:
  # Basic backtest
  dca-backtest -symbol BTCUSDT -interval 1h
  
  # Load from config file
  dca-backtest -config configs/bybit/btc_1h.json
  
  # Optimize parameters
  dca-backtest -symbol BTCUSDT -optimize
  
  # Test all intervals
  dca-backtest -symbol BTCUSDT -all-intervals
  
  # Walk-forward validation
  dca-backtest -symbol BTCUSDT -optimize -wf-enable
  
  # Advanced combo with custom parameters
  dca-backtest -symbol BTCUSDT -advanced-combo -base-amount 50 -max-multiplier 2.5
  
  # Progressive DCA spacing (1%->1.1%->1.21%->1.33%...)
  dca-backtest -symbol BTCUSDT -price-threshold 0.01 -price-threshold-multiplier 1.1

CONFIGURATION:
  -config FILE          Load configuration from JSON file
  -symbol SYMBOL        Trading symbol (default: BTCUSDT)
  -interval INTERVAL    Time interval: 5m, 15m, 1h, 4h, 1d (default: 1h)
  -exchange EXCHANGE    Exchange: bybit, binance (default: bybit)

ACCOUNT SETTINGS:
  -balance AMOUNT       Initial balance (default: 500)
  -commission RATE      Trading commission (default: 0.0005)

DCA STRATEGY:
  -base-amount AMOUNT           Base DCA amount (default: 40)
  -max-multiplier MULT          Maximum position multiplier (default: 3.0)
  -price-threshold PCT          Price drop %% for next DCA (default: 0.02)
  -price-threshold-multiplier X Progressive multiplier for DCA spacing (default: 1.0)
  -advanced-combo               Use advanced indicators instead of classic

ANALYSIS:
  -optimize             Run genetic algorithm optimization
  -all-intervals        Test all available intervals for symbol
  -period PERIOD        Limit data to period (7d, 30d, 180d, 365d)

WALK-FORWARD VALIDATION:
  -wf-enable            Enable walk-forward validation
  -wf-split-ratio RATIO Train/test split ratio (default: 0.7)
  -wf-rolling           Use rolling window instead of simple split
  -wf-train-days DAYS   Training window size (default: 180)
  -wf-test-days DAYS    Test window size (default: 60)
  -wf-roll-days DAYS    Roll forward step (default: 30)

OUTPUT:
  -data-root DIR        Data root directory (default: data)
  -console-only         Console output only, no file output
  -window SIZE          Analysis window size (default: 100)

OTHER:
  -env FILE             Environment file path (default: .env)
  -version              Show version information
  -help                 Show this help message

For more information, see the README or documentation.
`, AppName, AppVersion, filepath.Base(flag.CommandLine.Name()))
}

func loadEnvironment(envFile string) {
	if err := godotenv.Load(envFile); err != nil {
		log.Printf("⚠️  Could not load %s (%v)", envFile, err)
	}
}

func loadDCAConfiguration(configFile, dataFile, symbol, interval string, 
	initialBalance, commission float64, windowSize int, baseAmount, maxMultiplier, 
	priceThreshold, priceThresholdMultiplier float64, useAdvancedCombo bool) (*config.DCAConfig, error) {
	
	// Resolve config file path
	if configFile != "" && !strings.ContainsAny(configFile, "/\\") {
		configFile = filepath.Join("configs", configFile+".json")
	}
	
	// Load configuration using the config manager
	configManager := config.NewDCAConfigManager()
	params := map[string]interface{}{
		"base_amount":                 baseAmount,
		"max_multiplier":              maxMultiplier,
		"price_threshold":             priceThreshold,
		"price_threshold_multiplier":  priceThresholdMultiplier,
		"use_advanced_combo":          useAdvancedCombo,
	}
	
	cfgInterface, err := configManager.LoadConfig(configFile, dataFile, symbol, 
		initialBalance, commission, windowSize, params)
	if err != nil {
		return nil, err
	}
	
	cfg := cfgInterface.(*config.DCAConfig)
	
	// DCA-specific defaults
	cfg.Cycle = true        // Always use cycle mode for DCA
	cfg.UseTPLevels = true  // Always use 5-level TP system
	
	// Set default TP if not configured
	if cfg.TPPercent == 0 {
		cfg.TPPercent = DefaultTPPercent
	}
	
	// Set default indicators based on combo selection
	if len(cfg.Indicators) == 0 {
		if cfg.UseAdvancedCombo {
			cfg.Indicators = []string{"hull_ma", "mfi", "keltner", "wavetrend"}
		} else {
			cfg.Indicators = []string{"rsi", "macd", "bb", "ema"}
		}
	}
	
	// Preserve interval from config file, or use command line value if no config file
	effectiveInterval := interval
	if cfg.Interval != "" {
		effectiveInterval = cfg.Interval
		log.Printf("📊 Using interval from config file: %s", effectiveInterval)
	} else {
		cfg.Interval = interval
		log.Printf("📊 Using interval from command line: %s", effectiveInterval)
	}
	
	// Resolve data file if not set and not scanning all intervals
	if strings.TrimSpace(cfg.DataFile) == "" {
		dataFile := datamanager.FindDataFile("data", "bybit", strings.ToUpper(cfg.Symbol), effectiveInterval)
		if dataFile == "" {
			return nil, fmt.Errorf("no data file found for symbol %s with interval %s\n"+
				"💡 Expected data structure: data/bybit/{category}/%s/%s/candles.csv\n"+
				"   Where {category} is one of: spot, linear, inverse\n"+
				"   Please ensure data files exist or provide explicit -data flag", 
				cfg.Symbol, effectiveInterval, strings.ToUpper(cfg.Symbol), effectiveInterval)
		}
		cfg.DataFile = dataFile
		log.Printf("📁 Auto-detected data file: %s", filepath.Base(dataFile))
	}
	
	// Log configuration summary
	printConfigSummary(cfg)
	
	return cfg, nil
}

func printConfigSummary(cfg *config.DCAConfig) {
	fmt.Printf("📊 DCA Strategy Configuration\n")
	fmt.Printf("   Symbol: %s\n", cfg.Symbol)
	fmt.Printf("   Balance: $%.2f\n", cfg.InitialBalance)
	fmt.Printf("   Base Amount: $%.2f\n", cfg.BaseAmount)
	fmt.Printf("   Max Multiplier: %.2fx\n", cfg.MaxMultiplier)
	fmt.Printf("   Price Threshold: %.2f%%", cfg.PriceThreshold*100)
	if cfg.PriceThresholdMultiplier > 1.0 {
		fmt.Printf(" (Progressive: %.2fx per level)\n", cfg.PriceThresholdMultiplier)
	} else {
		fmt.Printf(" (Fixed)\n")
	}
	fmt.Printf("   TP System: 5-level (%.2f%% max)\n", cfg.TPPercent*100)
	
	comboName := "Classic (RSI, MACD, BB, EMA)"
	if cfg.UseAdvancedCombo {
		comboName = "Advanced (Hull MA, MFI, Keltner, WaveTrend)"
	}
	fmt.Printf("   Indicators: %s\n", comboName)
	fmt.Printf("   Data: %s\n\n", filepath.Base(cfg.DataFile))
}

func runSingleBacktest(orch orchestrator.Orchestrator, cfg *config.DCAConfig, 
	selectedPeriod time.Duration, consoleOnly bool) {
	
	fmt.Printf("🚀 Starting DCA Backtest\n\n")
	
	results, err := orch.RunSingleBacktest(cfg, selectedPeriod)
	if err != nil {
		log.Fatalf("❌ Backtest failed: %v", err)
	}
	
	// Output results
	interval := guessIntervalFromPath(cfg.DataFile)
	reporting.OutputConsoleWithContext(results, cfg.Symbol, interval)
	
	if !consoleOnly {
		saveResults(results, cfg.Symbol, interval, "optimized_trades.xlsx")
	}
}

func runOptimization(orch orchestrator.Orchestrator, cfg *config.DCAConfig, 
	selectedPeriod time.Duration, wfEnable bool, wfSplitRatio float64, wfRolling bool,
	wfTrainDays, wfTestDays, wfRollDays int, consoleOnly bool) {
	
	fmt.Printf("🧬 Starting DCA Optimization\n\n")
	
	// Create walk-forward config if enabled
	var wfConfig *validation.WalkForwardConfig
	if wfEnable {
		wfConfig = &validation.WalkForwardConfig{
			Enable:     true,
			Rolling:    wfRolling,
			SplitRatio: wfSplitRatio,
			TrainDays:  wfTrainDays,
			TestDays:   wfTestDays,
			RollDays:   wfRollDays,
		}
	}
	
	bestResults, bestConfig, err := orch.RunOptimizedBacktest(cfg, selectedPeriod, wfConfig)
	if err != nil {
		log.Fatalf("❌ Optimization failed: %v", err)
	}
	
	// Display results
	printOptimizationResults(bestConfig, bestResults)
	
	interval := guessIntervalFromPath(bestConfig.DataFile)
	reporting.OutputConsoleWithContext(bestResults, bestConfig.Symbol, interval)
	
	if !consoleOnly {
		saveResults(bestResults, bestConfig.Symbol, interval, "optimized_trades.xlsx")
		saveOptimizedConfig(bestConfig, bestConfig.Symbol, interval)
	}
}

func runMultiIntervalAnalysis(orch orchestrator.Orchestrator, cfg *config.DCAConfig,
	dataRoot, exchange string, optimize bool, selectedPeriod time.Duration,
	wfEnable bool, wfSplitRatio float64, wfRolling bool, wfTrainDays, wfTestDays, wfRollDays int,
	consoleOnly bool) {
	
	fmt.Printf("📊 Starting Multi-Interval Analysis\n\n")
	
	// Create walk-forward config if enabled
	var wfConfig *validation.WalkForwardConfig
	if wfEnable {
		wfConfig = &validation.WalkForwardConfig{
			Enable:     wfEnable,
			Rolling:    wfRolling,
			SplitRatio: wfSplitRatio,
			TrainDays:  wfTrainDays,
			TestDays:   wfTestDays,
			RollDays:   wfRollDays,
		}
	}
	
	results, err := orch.RunMultiIntervalAnalysis(cfg, dataRoot, exchange, optimize, selectedPeriod, wfConfig)
	if err != nil {
		log.Fatalf("❌ Multi-interval analysis failed: %v", err)
	}
	
	displayIntervalResults(results, consoleOnly)
}

func displayIntervalResults(results *orchestrator.IntervalAnalysisResult, consoleOnly bool) {
	fmt.Printf("\n📈 INTERVAL COMPARISON - %s\n", results.Symbol)
	fmt.Printf("%s\n", strings.Repeat("=", 80))
	fmt.Printf("%-8s | %7s | %6s | %5s | %7s | %5s | %8s | %s\n",
		"Interval", "Return%", "Trades", "Base$", "MaxMult", "TP%", "Thresh%", "Combo")
	fmt.Printf("%s\n", strings.Repeat("-", 80))
	
	for _, r := range results.Results {
		if r.Error != nil {
			fmt.Printf("%-8s | ERROR: %v\n", r.Interval, r.Error)
			continue
		}
		
		c := r.OptimizedCfg
		comboInfo := "Classic"
		if c.UseAdvancedCombo {
			comboInfo = "Advanced"
		}
		
		fmt.Printf("%-8s | %7.2f | %6d | %5.0f | %7.2f | %5.2f | %8.2f | %s\n",
			r.Interval,
			r.Results.TotalReturn*100,
			r.Results.TotalTrades,
			c.BaseAmount,
			c.MaxMultiplier,
			c.TPPercent*100,
			c.PriceThreshold*100,
			comboInfo,
		)
	}
	
	best := results.BestResult
	fmt.Printf("\n🏆 BEST: %s (%.2f%% return)\n", best.Interval, best.Results.TotalReturn*100)
	thresholdInfo := fmt.Sprintf("%.2f%%", best.OptimizedCfg.PriceThreshold*100)
	if best.OptimizedCfg.PriceThresholdMultiplier > 1.0 {
		thresholdInfo += fmt.Sprintf(" (%.2fx)", best.OptimizedCfg.PriceThresholdMultiplier)
	}
	fmt.Printf("Settings: Base=$%.0f, Mult=%.2fx, TP=%.2f%%, Thresh=%s\n\n",
		best.OptimizedCfg.BaseAmount, best.OptimizedCfg.MaxMultiplier,
		best.OptimizedCfg.TPPercent*100, thresholdInfo)
	
	// Show detailed results for best interval
	reporting.OutputConsole(best.Results)
	
	if !consoleOnly {
		saveResults(best.Results, results.Symbol, best.Interval, "optimized_trades.xlsx")
		saveOptimizedConfig(best.OptimizedCfg, results.Symbol, best.Interval)
	}
}

func printOptimizationResults(bestConfig *config.DCAConfig, bestResults *backtest.BacktestResults) {
	fmt.Printf("\n🎯 OPTIMIZATION RESULTS\n")
	fmt.Printf("%s\n", strings.Repeat("=", 50))
	fmt.Printf("Best Return: %.2f%%\n", bestResults.TotalReturn*100)
	fmt.Printf("Combo: %s\n", getComboName(bestConfig.UseAdvancedCombo))
	fmt.Printf("Base Amount: $%.2f\n", bestConfig.BaseAmount)
	fmt.Printf("Max Multiplier: %.2fx\n", bestConfig.MaxMultiplier)
	fmt.Printf("Price Threshold: %.2f%%", bestConfig.PriceThreshold*100)
	if bestConfig.PriceThresholdMultiplier > 1.0 {
		fmt.Printf(" (Progressive: %.2fx per level)\n", bestConfig.PriceThresholdMultiplier)
	} else {
		fmt.Printf(" (Fixed)\n")
	}
	fmt.Printf("TP System: 5-level (%.2f%% max)\n", bestConfig.TPPercent*100)
	fmt.Printf("Indicators: %s\n\n", strings.Join(bestConfig.Indicators, ", "))
}

func getComboName(useAdvanced bool) string {
	if useAdvanced {
		return "Advanced (Hull MA + MFI + Keltner + WaveTrend)"
	}
	return "Classic (RSI + MACD + BB + EMA)"
}

func guessIntervalFromPath(path string) string {
	if path == "" {
		return "unknown"
	}
	return filepath.Base(filepath.Dir(path))
}

func saveResults(results *backtest.BacktestResults, symbol, interval, filename string) {
	outputDir := reporting.DefaultOutputDir(symbol, interval)
	filePath := filepath.Join(outputDir, filename)
	
	if err := reporting.WriteTradesXLSX(results, filePath); err != nil {
		log.Printf("⚠️  Failed to save results: %v", err)
	} else {
		fmt.Printf("💾 Results saved: %s\n", filePath)
	}
}

func saveOptimizedConfig(cfg *config.DCAConfig, symbol, interval string) {
	outputDir := reporting.DefaultOutputDir(symbol, interval)
	filePath := filepath.Join(outputDir, "best_config.json")
	
	if err := reporting.WriteBacktestConfigJSON(convertDCAConfig(cfg), filePath); err != nil {
		log.Printf("⚠️  Failed to save config: %v", err)
	} else {
		fmt.Printf("💾 Config saved: %s\n", filePath)
	}
}

// Helper function to convert DCAConfig for JSON output
func convertDCAConfig(cfg *config.DCAConfig) reporting.MainBacktestConfig {
	return reporting.MainBacktestConfig{
		DataFile:            cfg.DataFile,         // ✅ PRESERVE DATA FILE
		Symbol:              cfg.Symbol,
		Interval:            cfg.Interval,         // ✅ PRESERVE INTERVAL
		InitialBalance:      cfg.InitialBalance,
		Commission:          cfg.Commission,
		WindowSize:          cfg.WindowSize,
		BaseAmount:          cfg.BaseAmount,
		MaxMultiplier:       cfg.MaxMultiplier,
		PriceThreshold:      cfg.PriceThreshold,
		PriceThresholdMultiplier: cfg.PriceThresholdMultiplier,
		UseAdvancedCombo:    cfg.UseAdvancedCombo,
		RSIPeriod:           cfg.RSIPeriod,
		RSIOversold:         cfg.RSIOversold,
		RSIOverbought:       cfg.RSIOverbought,
		MACDFast:            cfg.MACDFast,
		MACDSlow:            cfg.MACDSlow,
		MACDSignal:          cfg.MACDSignal,
		BBPeriod:            cfg.BBPeriod,
		BBStdDev:            cfg.BBStdDev,
		EMAPeriod:           cfg.EMAPeriod,
		HullMAPeriod:        cfg.HullMAPeriod,
		MFIPeriod:           cfg.MFIPeriod,
		MFIOversold:         cfg.MFIOversold,
		MFIOverbought:       cfg.MFIOverbought,
		KeltnerPeriod:       cfg.KeltnerPeriod,
		KeltnerMultiplier:   cfg.KeltnerMultiplier,
		WaveTrendN1:         cfg.WaveTrendN1,
		WaveTrendN2:         cfg.WaveTrendN2,
		WaveTrendOverbought: cfg.WaveTrendOverbought,
		WaveTrendOversold:   cfg.WaveTrendOversold,
		Indicators:          cfg.Indicators,
		TPPercent:           cfg.TPPercent,
		UseTPLevels:         cfg.UseTPLevels,
		Cycle:               cfg.Cycle,
		MinOrderQty:         cfg.MinOrderQty,
	}
}
