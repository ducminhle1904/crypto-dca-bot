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
	DefaultTPPercent      = 0.02 // 2%
	DefaultDataRoot       = "data"
	DefaultExchange       = "bybit"
)

func main() {
	// Create and parse command line flags
	flags := NewDCAFlags()
	flag.Parse()
	
	// Version and help
	if *flags.ShowVersion {
		fmt.Printf("%s v%s\n", AppName, AppVersion)
		return
	}
	
	if *flags.ShowHelp {
		printUsageHelp()
		return
	}
	
	// Header
	printHeader()
	
	// Load environment
	loadEnvironment(*flags.EnvFile)
	
	// Load configuration first to see if it has indicators
	cfg, err := loadDCAConfiguration(*flags.ConfigFile, *flags.DataFile, *flags.Symbol, *flags.Interval, 
		*flags.InitialBalance, *flags.Commission, *flags.WindowSize, *flags.BaseAmount, *flags.MaxMultiplier, flags)
	if err != nil {
		log.Fatalf("❌ Configuration error: %v", err)
	}
	
	// Parse period filter
	var selectedPeriod time.Duration
	if *flags.Period != "" {
		if d, ok := datamanager.ParseTrailingPeriod(*flags.Period); ok {
			selectedPeriod = d
		} else {
			log.Fatalf("❌ Invalid period format: %s (use 7d, 30d, 180d, 365d)", *flags.Period)
		}
	}
	
	// Create orchestrator
	orch := orchestrator.NewOrchestrator()
	
	// Execute based on options
	if *flags.AllIntervals {
		runMultiIntervalAnalysis(orch, cfg, *flags.DataRoot, *flags.Exchange, *flags.Optimize, selectedPeriod, 
			*flags.WFEnable, *flags.WFSplitRatio, *flags.WFRolling, *flags.WFTrainDays, *flags.WFTestDays, *flags.WFRollDays, *flags.ConsoleOnly)
	} else if *flags.Optimize {
		runOptimization(orch, cfg, selectedPeriod, *flags.WFEnable, *flags.WFSplitRatio, *flags.WFRolling, 
			*flags.WFTrainDays, *flags.WFTestDays, *flags.WFRollDays, *flags.ConsoleOnly)
	} else {
		runSingleBacktest(orch, cfg, selectedPeriod, *flags.ConsoleOnly)
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
  
  # Custom indicators with parameters  
  dca-backtest -symbol BTCUSDT -rsi -supertrend -mfi -base-amount 50 -max-multiplier 2.5
  
  # Progressive DCA spacing (1%->1.1%->1.21%->1.33%...)
  dca-backtest -symbol BTCUSDT -price-threshold 0.01 -price-threshold-multiplier 1.1
  
  # Custom indicator combinations
  dca-backtest -symbol BTCUSDT -supertrend -mfi
  dca-backtest -symbol BTCUSDT -indicators "rsi,supertrend,ema"

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

INDICATOR SELECTION (flexible):
  -indicators LIST              Comma-separated indicators (e.g., rsi,macd,supertrend)
  
INDIVIDUAL INDICATORS:
  -rsi                          Include RSI indicator
  -macd                         Include MACD indicator  
  -bb                           Include Bollinger Bands indicator
  -ema                          Include EMA indicator
  -hullma                       Include Hull MA indicator
  -supertrend                   Include SuperTrend indicator
  -mfi                          Include MFI indicator
  -keltner                      Include Keltner Channels indicator
  -wavetrend                    Include WaveTrend indicator
  -obv                          Include OBV (On-Balance Volume) indicator

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
	initialBalance, commission float64, windowSize int, baseAmount, maxMultiplier float64, flags *DCAFlags) (*config.DCAConfig, error) {
	
	// Resolve config file path
	if configFile != "" && !strings.ContainsAny(configFile, "/\\") {
		configFile = filepath.Join("configs", configFile+".json")
	}
	
	// Load configuration using the config manager
	configManager := config.NewDCAConfigManager()
	params := map[string]interface{}{
		"base_amount":                 baseAmount,
		"max_multiplier":              maxMultiplier,
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
	
	// Priority: Config file indicators > Command line indicators > ERROR (no defaults)
	if configFile != "" && len(cfg.Indicators) > 0 {
		// Config file has indicators - use them (highest priority)
		log.Printf("📋 Using indicators from config file: %s", strings.Join(cfg.Indicators, ", "))
	} else {
		// No config file indicators - check command line
		indicators, err := ResolveIndicators(flags)
		if err != nil {
			return nil, fmt.Errorf("indicator configuration error: %w", err)
		}
		
		if len(indicators) > 0 {
			// Use command line specified indicators
			cfg.Indicators = indicators
			log.Printf("📋 Using indicators from command line: %s", strings.Join(indicators, ", "))
		} else {
			// No indicators specified - require explicit specification
			return nil, fmt.Errorf("no indicators specified. Please use one of:\n" +
				"  • Individual flags: -rsi -macd -bb -ema\n" +
				"  • Indicator list: -indicators \"rsi,macd,bb,ema\"\n" +
				"  • Config file with indicators specified\n" +
				"\nAvailable indicators: rsi, macd, bb, ema, hullma, supertrend, mfi, keltner, wavetrend, obv, stochrsi")
		}
	}
	
	// Configure DCA spacing strategy from command line flags if not present in config
	if cfg.DCASpacing == nil {
		// Create DCA spacing configuration from flags
		dcaSpacingConfig, err := createDCASpacingFromFlags(flags)
		if err != nil {
			return nil, fmt.Errorf("failed to create DCA spacing configuration: %w", err)
		}
		cfg.DCASpacing = dcaSpacingConfig
		log.Printf("📊 Using DCA spacing from command line: %s strategy", cfg.DCASpacing.Strategy)
	} else {
		log.Printf("📊 Using DCA spacing from config file: %s strategy", cfg.DCASpacing.Strategy)
	}
	
	// Validate that DCA spacing is now present
	if cfg.DCASpacing == nil {
		return nil, fmt.Errorf("DCA spacing configuration is required - please specify dca_spacing in config file or use DCA spacing flags")
	}
	
	// Preserve interval from config file, or use command line value if no config file
	effectiveInterval := interval
	if cfg.Interval != "" {
		effectiveInterval = cfg.Interval
	} else {
		cfg.Interval = interval
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
	}
	
	// Log configuration summary with sources
	printConfigSummary(cfg)
	
	return cfg, nil
}

func printConfigSummary(cfg *config.DCAConfig) {
	fmt.Printf("📊 DCA Strategy Configuration\n")
	fmt.Printf("   Symbol: %s\n", cfg.Symbol)
	fmt.Printf("   Interval: %s\n", cfg.Interval)
	fmt.Printf("   Balance: $%.2f\n", cfg.InitialBalance)
	fmt.Printf("   Base Amount: $%.2f\n", cfg.BaseAmount)
	fmt.Printf("   Max Multiplier: %.2fx\n", cfg.MaxMultiplier)
	
	// DCA Spacing Strategy display
	if cfg.DCASpacing != nil {
		fmt.Printf("   DCA Spacing: %s\n", cfg.DCASpacing.Strategy)
		if cfg.DCASpacing.Strategy == "fixed" {
			if baseThreshold, ok := cfg.DCASpacing.Parameters["base_threshold"].(float64); ok {
				if multiplier, ok := cfg.DCASpacing.Parameters["threshold_multiplier"].(float64); ok && multiplier > 1.0 {
					fmt.Printf("   Progression: %.2f%% → %.2f%% → %.2f%% → %.2f%% → %.2f%%\n",
						baseThreshold*100,
						baseThreshold*multiplier*100,
						baseThreshold*multiplier*multiplier*100,
						baseThreshold*multiplier*multiplier*multiplier*100,
						baseThreshold*multiplier*multiplier*multiplier*multiplier*100)
				} else {
					fmt.Printf("   Threshold: %.2f%% (Fixed)\n", baseThreshold*100)
				}
			}
		} else if cfg.DCASpacing.Strategy == "volatility_adaptive" {
			if baseThreshold, ok := cfg.DCASpacing.Parameters["base_threshold"].(float64); ok {
				if sensitivity, ok := cfg.DCASpacing.Parameters["volatility_sensitivity"].(float64); ok {
					fmt.Printf("   ATR-based: %.2f%% base, %.1fx sensitivity\n", baseThreshold*100, sensitivity)
				}
			}
		}
	} else {
		fmt.Printf("   DCA Spacing: Not configured\n")
	}
	
	fmt.Printf("   TP System: 5-level (%.2f%% max)\n", cfg.TPPercent*100)
	
	indicatorDescription := GetIndicatorDescription(cfg.Indicators)
	fmt.Printf("   Indicators: %s\n", indicatorDescription)
	
	// Print detailed indicator settings
	printIndicatorSettings(cfg)
	fmt.Printf("\n")
}

func printIndicatorSettings(cfg *config.DCAConfig) {
	fmt.Printf("   📈 Indicator Settings:\n")
	
	for _, indicator := range cfg.Indicators {
		switch strings.ToLower(indicator) {
		case "rsi":
			fmt.Printf("      • RSI: period=%d, oversold=%.0f, overbought=%.0f\n", 
				cfg.RSIPeriod, cfg.RSIOversold, cfg.RSIOverbought)
		case "macd":
			fmt.Printf("      • MACD: fast=%d, slow=%d, signal=%d\n", 
				cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
		case "bb", "bollinger":
			fmt.Printf("      • Bollinger Bands: period=%d, stddev=%.1f\n", 
				cfg.BBPeriod, cfg.BBStdDev)
		case "ema":
			fmt.Printf("      • EMA: period=%d\n", cfg.EMAPeriod)
		case "hullma", "hull_ma":
			fmt.Printf("      • Hull MA: period=%d\n", cfg.HullMAPeriod)
		case "supertrend", "st":
			fmt.Printf("      • SuperTrend: period=%d, multiplier=%.1f\n", 
				cfg.SuperTrendPeriod, cfg.SuperTrendMultiplier)
		case "mfi":
			fmt.Printf("      • MFI: period=%d, oversold=%.0f, overbought=%.0f\n", 
				cfg.MFIPeriod, cfg.MFIOversold, cfg.MFIOverbought)
		case "keltner", "kc":
			fmt.Printf("      • Keltner: period=%d, multiplier=%.1f\n", 
				cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
		case "wavetrend", "wt":
			fmt.Printf("      • WaveTrend: n1=%d, n2=%d, overbought=%.0f, oversold=%.0f\n", 
				cfg.WaveTrendN1, cfg.WaveTrendN2, cfg.WaveTrendOverbought, cfg.WaveTrendOversold)
		case "obv":
			fmt.Printf("      • OBV: trend_threshold=%.3f\n", cfg.OBVTrendThreshold)
		case "stochrsi", "stochastic_rsi", "stoch_rsi":
			fmt.Printf("      • Stochastic RSI: period=%d, overbought=%.1f, oversold=%.1f\n", 
				cfg.StochasticRSIPeriod, cfg.StochasticRSIOverbought, cfg.StochasticRSIOversold)
		}
	}
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
	fmt.Printf("%s\n", strings.Repeat("=", 90))
	fmt.Printf("%-8s | %7s | %6s | %5s | %7s | %5s | %8s | %6s | %s\n",
		"Interval", "Return%", "Trades", "Base$", "MaxMult", "TP%", "Thresh%", "Spacing", "Indicators")
	fmt.Printf("%s\n", strings.Repeat("-", 90))
	
	for _, r := range results.Results {
		if r.Error != nil {
			fmt.Printf("%-8s | ERROR: %v\n", r.Interval, r.Error)
			continue
		}
		
		c := r.OptimizedCfg
		comboInfo := GetIndicatorDescription(c.Indicators)
		
		// Format spacing display
		spacingDisplay := "N/A"
		threshDisplay := "N/A"
		if c.DCASpacing != nil {
			spacingDisplay = c.DCASpacing.Strategy
			if baseThresh, ok := c.DCASpacing.Parameters["base_threshold"].(float64); ok {
				threshDisplay = fmt.Sprintf("%.2f", baseThresh*100)
			}
		}
		
		fmt.Printf("%-8s | %7.2f | %6d | %5.0f | %7.2f | %5.2f | %8s | %6s | %s\n",
			r.Interval,
			r.Results.TotalReturn*100,
			r.Results.TotalTrades,
			c.BaseAmount,
			c.MaxMultiplier,
			c.TPPercent*100,
			threshDisplay+"%",
			spacingDisplay,
			comboInfo,
		)
	}
	
	best := results.BestResult
	fmt.Printf("\n🏆 BEST: %s (%.2f%% return)\n", best.Interval, best.Results.TotalReturn*100)
	
	spacingInfo := "N/A"
	if best.OptimizedCfg.DCASpacing != nil {
		if baseThresh, ok := best.OptimizedCfg.DCASpacing.Parameters["base_threshold"].(float64); ok {
			spacingInfo = fmt.Sprintf("%.2f%% (%s)", baseThresh*100, best.OptimizedCfg.DCASpacing.Strategy)
		} else {
			spacingInfo = best.OptimizedCfg.DCASpacing.Strategy
		}
	}
	
	fmt.Printf("Settings: Base=$%.0f, Mult=%.2fx, TP=%.2f%%, Spacing=%s\n\n",
		best.OptimizedCfg.BaseAmount, best.OptimizedCfg.MaxMultiplier,
		best.OptimizedCfg.TPPercent*100, spacingInfo)
	
	// Show detailed results for best interval
	reporting.OutputConsole(best.Results)
	
	if !consoleOnly {
		saveResults(best.Results, results.Symbol, best.Interval, "optimized_trades.xlsx")
		saveOptimizedConfig(best.OptimizedCfg, results.Symbol, best.Interval)
	}
}

func printOptimizationResults(bestConfig *config.DCAConfig, bestResults *backtest.BacktestResults) {
	fmt.Printf("\n🎯 OPTIMIZED DCA STRATEGY CONFIGURATION\n")
	fmt.Printf("%s\n", strings.Repeat("=", 50))
	fmt.Printf("   Best Return: %.2f%%\n", bestResults.TotalReturn*100)
	fmt.Printf("   Base Amount: $%.2f\n", bestConfig.BaseAmount)
	fmt.Printf("   Max Multiplier: %.2fx\n", bestConfig.MaxMultiplier)
	
	// DCA Spacing Strategy display (matching initial format)
	if bestConfig.DCASpacing != nil {
		fmt.Printf("   DCA Spacing: %s Strategy\n", bestConfig.DCASpacing.Strategy)
		if bestConfig.DCASpacing.Strategy == "fixed" {
			if baseThreshold, ok := bestConfig.DCASpacing.Parameters["base_threshold"].(float64); ok {
				if multiplier, ok := bestConfig.DCASpacing.Parameters["threshold_multiplier"].(float64); ok && multiplier > 1.0 {
					fmt.Printf("   Fixed: %.3f%% base, %.3fx multiplier\n", baseThreshold*100, multiplier)
					fmt.Printf("   Progression: %.3f%% → %.3f%% → %.3f%% → %.3f%% → %.3f%%\n",
						baseThreshold*100,
						baseThreshold*multiplier*100,
						baseThreshold*multiplier*multiplier*100,
						baseThreshold*multiplier*multiplier*multiplier*100,
						baseThreshold*multiplier*multiplier*multiplier*multiplier*100)
				} else {
					fmt.Printf("   Fixed: %.3f%% base (no multiplier)\n", baseThreshold*100)
				}
			}
		} else if bestConfig.DCASpacing.Strategy == "volatility_adaptive" {
			if baseThreshold, ok := bestConfig.DCASpacing.Parameters["base_threshold"].(float64); ok {
				if sensitivity, ok := bestConfig.DCASpacing.Parameters["volatility_sensitivity"].(float64); ok {
					fmt.Printf("   ATR-based: %.2f%% base, %.1fx sensitivity", baseThreshold*100, sensitivity)
					if atrPeriod, ok := bestConfig.DCASpacing.Parameters["atr_period"].(int); ok {
						fmt.Printf(", %d-period", atrPeriod)
					}
					if levelMultiplier, ok := bestConfig.DCASpacing.Parameters["level_multiplier"].(float64); ok && levelMultiplier > 1.0 {
						fmt.Printf(", %.2fx level progression", levelMultiplier)
					}
					fmt.Printf("\n")
				}
			}
		}
	} else {
		fmt.Printf("   DCA Spacing: Not configured\n")
	}
	
	fmt.Printf("   TP System: 5-level (%.2f%% max)\n", bestConfig.TPPercent*100)
	
	indicatorDescription := GetIndicatorDescription(bestConfig.Indicators)
	fmt.Printf("   Indicators: %s\n", indicatorDescription)
	
	// Print optimized indicator settings
	printOptimizedIndicatorSettings(bestConfig)
	fmt.Printf("\n")
}

func printOptimizedIndicatorSettings(cfg *config.DCAConfig) {
	fmt.Printf("   📈 Optimized Indicator Settings:\n")
	
	for _, indicator := range cfg.Indicators {
		switch strings.ToLower(indicator) {
		case "rsi":
			fmt.Printf("      • RSI: period=%d, oversold=%.0f, overbought=%.0f\n", 
				cfg.RSIPeriod, cfg.RSIOversold, cfg.RSIOverbought)
		case "macd":
			fmt.Printf("      • MACD: fast=%d, slow=%d, signal=%d\n", 
				cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
		case "bb", "bollinger":
			fmt.Printf("      • Bollinger Bands: period=%d, stddev=%.1f\n", 
				cfg.BBPeriod, cfg.BBStdDev)
		case "ema":
			fmt.Printf("      • EMA: period=%d\n", cfg.EMAPeriod)
		case "hullma", "hull_ma":
			fmt.Printf("      • Hull MA: period=%d\n", cfg.HullMAPeriod)
		case "supertrend", "st":
			fmt.Printf("      • SuperTrend: period=%d, multiplier=%.1f\n", 
				cfg.SuperTrendPeriod, cfg.SuperTrendMultiplier)
		case "mfi":
			fmt.Printf("      • MFI: period=%d, oversold=%.0f, overbought=%.0f\n", 
				cfg.MFIPeriod, cfg.MFIOversold, cfg.MFIOverbought)
		case "keltner", "kc":
			fmt.Printf("      • Keltner: period=%d, multiplier=%.1f\n", 
				cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
		case "wavetrend", "wt":
			fmt.Printf("      • WaveTrend: n1=%d, n2=%d, overbought=%.0f, oversold=%.0f\n", 
				cfg.WaveTrendN1, cfg.WaveTrendN2, cfg.WaveTrendOverbought, cfg.WaveTrendOversold)
		case "obv":
			fmt.Printf("      • OBV: trend_threshold=%.3f\n", cfg.OBVTrendThreshold)
		case "stochrsi", "stochastic_rsi", "stoch_rsi":
			fmt.Printf("      • Stochastic RSI: period=%d, overbought=%.1f, oversold=%.1f\n", 
				cfg.StochasticRSIPeriod, cfg.StochasticRSIOverbought, cfg.StochasticRSIOversold)
		}
	}
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
		DataFile:            cfg.DataFile,
		Symbol:              cfg.Symbol,
		Interval:            cfg.Interval,
		InitialBalance:      cfg.InitialBalance,
		Commission:          cfg.Commission,
		WindowSize:          cfg.WindowSize,
		BaseAmount:          cfg.BaseAmount,
		MaxMultiplier:       cfg.MaxMultiplier,
		RSIPeriod:           cfg.RSIPeriod,
		RSIOversold:         cfg.RSIOversold,
		RSIOverbought:       cfg.RSIOverbought,
		MACDFast:            cfg.MACDFast,
		MACDSlow:            cfg.MACDSlow,
		MACDSignal:          cfg.MACDSignal,
		BBPeriod:            cfg.BBPeriod,
		BBStdDev:            cfg.BBStdDev,
		EMAPeriod:           cfg.EMAPeriod,
		SuperTrendPeriod:     cfg.SuperTrendPeriod,
		SuperTrendMultiplier: cfg.SuperTrendMultiplier,
		MFIPeriod:           cfg.MFIPeriod,
		MFIOversold:         cfg.MFIOversold,
		MFIOverbought:       cfg.MFIOverbought,
		KeltnerPeriod:       cfg.KeltnerPeriod,
		KeltnerMultiplier:   cfg.KeltnerMultiplier,
		WaveTrendN1:         cfg.WaveTrendN1,
		WaveTrendN2:         cfg.WaveTrendN2,
		WaveTrendOverbought: cfg.WaveTrendOverbought,
		WaveTrendOversold:   cfg.WaveTrendOversold,
		OBVTrendThreshold:   cfg.OBVTrendThreshold,
		HullMAPeriod:        cfg.HullMAPeriod,
		StochasticRSIPeriod:     cfg.StochasticRSIPeriod,
		StochasticRSIOverbought: cfg.StochasticRSIOverbought,
		StochasticRSIOversold:   cfg.StochasticRSIOversold,
		Indicators:          cfg.Indicators,
		TPPercent:           cfg.TPPercent,
		UseTPLevels:         cfg.UseTPLevels,
		Cycle:               cfg.Cycle,
		MinOrderQty:         cfg.MinOrderQty,
		DCASpacing:          cfg.DCASpacing,
	}
}

// createDCASpacingFromFlags creates DCA spacing configuration from command line flags
func createDCASpacingFromFlags(flags *DCAFlags) (*config.DCASpacingConfig, error) {
	strategy := strings.ToLower(strings.TrimSpace(*flags.DCASpacingStrategy))
	
	switch strategy {
	case "fixed":
		return &config.DCASpacingConfig{
			Strategy: "fixed",
			Parameters: map[string]interface{}{
				"base_threshold":       *flags.SpacingBaseThreshold,
				"threshold_multiplier": *flags.SpacingMultiplier,
				"max_threshold":        0.10, // 10% safety limit
				"min_threshold":        0.003, // 0.3% safety limit
			},
		}, nil
		
	case "volatility_adaptive", "adaptive", "atr":
		return &config.DCASpacingConfig{
			Strategy: "volatility_adaptive",
			Parameters: map[string]interface{}{
				"base_threshold":        *flags.SpacingBaseThreshold,
				"volatility_sensitivity": *flags.SpacingVolatilitySens,
				"atr_period":            *flags.SpacingATRPeriod,
				"max_threshold":         0.05, // 5% safety limit for adaptive
				"min_threshold":         0.003, // 0.3% safety limit
				"level_multiplier":      1.1,  // Default level multiplier
			},
		}, nil
		
	default:
		return nil, fmt.Errorf("unsupported DCA spacing strategy: %s (supported: fixed, volatility_adaptive)", strategy)
	}
}
