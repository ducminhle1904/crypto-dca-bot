package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
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
	
	// Validate flags before proceeding
	if err := ValidateDCAFlags(flags); err != nil {
		log.Fatalf("❌ Flag validation error: %v", err)
	}
	
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
		// Validate period format before parsing
		period := strings.TrimSpace(*flags.Period)
		if period == "" {
			log.Fatalf("❌ Empty period specified")
		}
		if d, ok := datamanager.ParseTrailingPeriod(period); ok {
			selectedPeriod = d
		} else {
			log.Fatalf("❌ Invalid period format: %s (use 7d, 30d, 180d, 365d)", period)
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
	fmt.Printf("%s v%s - Enhanced DCA Strategy Backtesting\n\n", AppName, AppVersion)
	fmt.Printf("USAGE:\n  %s [OPTIONS]\n\n", filepath.Base(flag.CommandLine.Name()))
	
	// Print examples using the DCA examples function
	PrintDCAUsageExamples()
	
	// Print flag groups to show all available flags including dynamic TP
	PrintDCAFlagGroups()
	
	fmt.Printf("\nFor more information, see the README or documentation.\n")
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
	
	// Safe type assertion with nil check and error handling
	if cfgInterface == nil {
		return nil, fmt.Errorf("configuration manager returned nil config")
	}
	
	cfg, ok := cfgInterface.(*config.DCAConfig)
	if !ok {
		return nil, fmt.Errorf("invalid configuration type: expected *config.DCAConfig, got %T", cfgInterface)
	}
	
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
		// log.Printf("📋 Using indicators from config file: %s", strings.Join(cfg.Indicators, ", "))
	} else {
		// No config file indicators - check command line
		indicators, err := ResolveIndicators(flags)
		if err != nil {
			return nil, fmt.Errorf("indicator configuration error: %w", err)
		}
		
		if len(indicators) > 0 {
			// Use command line specified indicators
			cfg.Indicators = indicators
			// log.Printf("📋 Using indicators from command line: %s", strings.Join(indicators, ", "))
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
	}
	
	// Validate that DCA spacing is now present
	if cfg.DCASpacing == nil {
		return nil, fmt.Errorf("DCA spacing configuration is required - please specify dca_spacing in config file or use DCA spacing flags")
	}
	
	if *flags.DynamicTPStrategy != "fixed" {
		// Validate dynamic TP parameters first
		if err := validateDynamicTPFlags(flags); err != nil {
			return nil, fmt.Errorf("dynamic TP validation error: %w", err)
		}
	}
	
	// Configure Dynamic TP from command line flags if not present in config
	if cfg.DynamicTP == nil && *flags.DynamicTPStrategy != "fixed" {
		// Create Dynamic TP configuration from flags
		dynamicTPConfig, err := createDynamicTPFromFlags(flags)
		if err != nil {
			return nil, fmt.Errorf("failed to create Dynamic TP configuration: %w", err)
		}
		cfg.DynamicTP = dynamicTPConfig
	}
	
	// Set TP parameters from flags if not in config
	if cfg.TPPercent == 0 {
		cfg.TPPercent = *flags.TPPercent
	}
	cfg.UseTPLevels = *flags.UseTPLevels
	
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
	
	// Comprehensive business logic validation to prevent trading disasters
	if err := validateBusinessLogicBounds(cfg); err != nil {
		return nil, fmt.Errorf("business logic validation failed: %w", err)
	}
	
	// Ensure no invalid dynamic TP configs can slip through
	if cfg.DynamicTP != nil {
		if cfg.DynamicTP.VolatilityConfig != nil {
			min := cfg.DynamicTP.VolatilityConfig.MinTPPercent
			max := cfg.DynamicTP.VolatilityConfig.MaxTPPercent
			if min >= max {
				return nil, fmt.Errorf("CRITICAL VALIDATION ERROR: Dynamic TP min (%.3f) >= max (%.3f) - this should never happen", min, max)
			}
		}
		if cfg.DynamicTP.IndicatorConfig != nil {
			min := cfg.DynamicTP.IndicatorConfig.MinTPPercent
			max := cfg.DynamicTP.IndicatorConfig.MaxTPPercent
			if min >= max {
				return nil, fmt.Errorf("CRITICAL VALIDATION ERROR: Dynamic TP min (%.3f) >= max (%.3f) - this should never happen", min, max)
			}
		}
	}

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
	
	// Display TP system information
	if cfg.UseTPLevels && cfg.DynamicTP != nil {
		fmt.Printf("   TP System: Multi-level Dynamic %s (5 levels, base: %.2f%%", cfg.DynamicTP.Strategy, cfg.TPPercent*100)
		if cfg.DynamicTP.VolatilityConfig != nil {
			fmt.Printf(", range: %.2f%%-%.2f%%", 
				cfg.DynamicTP.VolatilityConfig.MinTPPercent*100,
				cfg.DynamicTP.VolatilityConfig.MaxTPPercent*100)
		} else if cfg.DynamicTP.IndicatorConfig != nil {
			fmt.Printf(", range: %.2f%%-%.2f%%", 
				cfg.DynamicTP.IndicatorConfig.MinTPPercent*100,
				cfg.DynamicTP.IndicatorConfig.MaxTPPercent*100)
		}
		fmt.Printf(")\n")
	} else if cfg.UseTPLevels {
		fmt.Printf("   TP System: Multi-level Fixed (5 levels, %.2f%% max)\n", cfg.TPPercent*100)
	} else if cfg.DynamicTP != nil {
		fmt.Printf("   TP System: Single-level Dynamic %s (base: %.2f%%", cfg.DynamicTP.Strategy, cfg.TPPercent*100)
		if cfg.DynamicTP.VolatilityConfig != nil {
			fmt.Printf(", range: %.2f%%-%.2f%%", 
				cfg.DynamicTP.VolatilityConfig.MinTPPercent*100,
				cfg.DynamicTP.VolatilityConfig.MaxTPPercent*100)
		} else if cfg.DynamicTP.IndicatorConfig != nil {
			fmt.Printf(", range: %.2f%%-%.2f%%", 
				cfg.DynamicTP.IndicatorConfig.MinTPPercent*100,
				cfg.DynamicTP.IndicatorConfig.MaxTPPercent*100)
		}
		fmt.Printf(")\n")
	} else {
		fmt.Printf("   TP System: Single-level Fixed (%.2f%%)\n", cfg.TPPercent*100)
	}
	
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
	
	// Display TP system information
	if bestConfig.UseTPLevels {
		fmt.Printf("   TP System: Multi-level (5 levels, %.2f%% max)\n", bestConfig.TPPercent*100)
	} else if bestConfig.DynamicTP != nil {
		fmt.Printf("   TP System: Dynamic %s (base: %.2f%%", bestConfig.DynamicTP.Strategy, bestConfig.TPPercent*100)
		if bestConfig.DynamicTP.VolatilityConfig != nil {
			fmt.Printf(", range: %.2f%%-%.2f%%", 
				bestConfig.DynamicTP.VolatilityConfig.MinTPPercent*100,
				bestConfig.DynamicTP.VolatilityConfig.MaxTPPercent*100)
		} else if bestConfig.DynamicTP.IndicatorConfig != nil {
			fmt.Printf(", range: %.2f%%-%.2f%%", 
				bestConfig.DynamicTP.IndicatorConfig.MinTPPercent*100,
				bestConfig.DynamicTP.IndicatorConfig.MaxTPPercent*100)
		}
		fmt.Printf(")\n")
	} else {
		fmt.Printf("   TP System: Fixed single-level (%.2f%%)\n", bestConfig.TPPercent*100)
	}
	
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
	
	// Clean the path first
	path = filepath.Clean(path)
	dir := filepath.Dir(path)
	
	// Handle edge cases
	if dir == "." || dir == "/" || dir == "\\" {
		return "unknown"
	}
	
	interval := filepath.Base(dir)
	// Validate that it looks like a time interval
	if interval == "" || interval == "." {
		return "unknown"
	}
	
	return interval
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
		DynamicTP:           cfg.DynamicTP,
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

// createDynamicTPFromFlags creates Dynamic TP configuration from command line flags
func createDynamicTPFromFlags(flags *DCAFlags) (*config.DynamicTPConfig, error) {
	strategy := strings.ToLower(strings.TrimSpace(*flags.DynamicTPStrategy))
	
	switch strategy {
	case "volatility_adaptive", "volatility", "atr":
		return &config.DynamicTPConfig{
			Strategy:      "volatility_adaptive",
			BaseTPPercent: *flags.TPPercent,
			VolatilityConfig: &config.DynamicTPVolatilityConfig{
				Multiplier:   *flags.TPVolatilityMult,
				MinTPPercent: *flags.TPMinPercent,
				MaxTPPercent: *flags.TPMaxPercent,
				ATRPeriod:    14, // Default ATR period
			},
		}, nil
		
	case "indicator_based", "indicator", "signals":
		indicatorConfig := &config.DynamicTPIndicatorConfig{
			Weights:          make(map[string]float64),
			StrengthMultiplier: *flags.TPStrengthMult,
			MinTPPercent:     *flags.TPMinPercent,
			MaxTPPercent:     *flags.TPMaxPercent,
		}
		
		// Parse indicator weights if provided
		if *flags.TPIndicatorWeights != "" {
			weights, err := parseIndicatorWeights(*flags.TPIndicatorWeights)
			if err != nil {
				return nil, fmt.Errorf("failed to parse TP indicator weights: %w", err)
			}
			indicatorConfig.Weights = weights
		} else {
			// Default equal weights for common indicators
			indicatorConfig.Weights = map[string]float64{
				"rsi":  0.25,
				"macd": 0.25,
				"bb":   0.25,
				"ema":  0.25,
			}
		}
		
		return &config.DynamicTPConfig{
			Strategy:        "indicator_based",
			BaseTPPercent:   *flags.TPPercent,
			IndicatorConfig: indicatorConfig,
		}, nil
		
	case "fixed":
		// Fixed TP mode - no dynamic TP config needed
		return nil, nil
		
	default:
		return nil, fmt.Errorf("unsupported Dynamic TP strategy: %s (supported: fixed, volatility_adaptive, indicator_based)", strategy)
	}
}

// parseIndicatorWeights parses comma-separated indicator:weight pairs
func parseIndicatorWeights(weightsStr string) (map[string]float64, error) {
	// Validate input
	weightsStr = strings.TrimSpace(weightsStr)
	if weightsStr == "" {
		return nil, fmt.Errorf("empty weights string provided")
	}
	
	weights := make(map[string]float64)
	pairs := strings.Split(weightsStr, ",")
	
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue // Skip empty pairs
		}
		
		parts := strings.Split(pair, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid weight format: %s (expected indicator:weight)", pair)
		}
		
		indicator := strings.TrimSpace(parts[0])
		weightStr := strings.TrimSpace(parts[1])
		
		// Validate indicator name
		if indicator == "" {
			return nil, fmt.Errorf("empty indicator name in pair: %s", pair)
		}
		
		// Validate weight string
		if weightStr == "" {
			return nil, fmt.Errorf("empty weight value for indicator %s", indicator)
		}
		
		weight, err := strconv.ParseFloat(weightStr, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid weight value for %s: %s", indicator, weightStr)
		}
		
		// Allow weight of 0 but warn about it in validation
		if weight < 0 || weight > 1 {
			return nil, fmt.Errorf("weight for %s must be between 0 and 1, got %f", indicator, weight)
		}
		
		// Check for duplicate indicators
		if _, exists := weights[indicator]; exists {
			return nil, fmt.Errorf("duplicate indicator: %s", indicator)
		}
		
		weights[indicator] = weight
	}
	
	return weights, nil
}

// validateDynamicTPFlags validates dynamic TP command line flag parameters
func validateDynamicTPFlags(flags *DCAFlags) error {
	// Validate TP percentages are within valid range (0 < value <= 1.0)
	if *flags.TPPercent <= 0 || *flags.TPPercent > 1.0 {
		return fmt.Errorf("base TP percentage must be between 0 and 1.0, got %.3f", *flags.TPPercent)
	}
	
	if *flags.TPMinPercent <= 0 || *flags.TPMinPercent > 1.0 {
		return fmt.Errorf("minimum TP percentage must be between 0 and 1.0, got %.3f", *flags.TPMinPercent)
	}
	
	if *flags.TPMaxPercent <= 0 || *flags.TPMaxPercent > 1.0 {
		return fmt.Errorf("maximum TP percentage must be between 0 and 1.0, got %.3f", *flags.TPMaxPercent)
	}
	
	if *flags.TPMinPercent >= *flags.TPMaxPercent {
		return fmt.Errorf("minimum TP percentage (%.3f) must be less than maximum TP percentage (%.3f)", 
			*flags.TPMinPercent, *flags.TPMaxPercent)
	}
	
	if *flags.TPPercent < *flags.TPMinPercent || *flags.TPPercent > *flags.TPMaxPercent {
		return fmt.Errorf("base TP percentage (%f) must be between min (%f) and max (%f)", 
			*flags.TPPercent, *flags.TPMinPercent, *flags.TPMaxPercent)
	}
	
	// Strategy-specific validation
	strategy := strings.ToLower(strings.TrimSpace(*flags.DynamicTPStrategy))
	switch strategy {
	case "volatility_adaptive", "volatility", "atr":
		if *flags.TPVolatilityMult <= 0 || *flags.TPVolatilityMult > 5.0 {
			return fmt.Errorf("volatility multiplier must be between 0 and 5.0, got %f", *flags.TPVolatilityMult)
		}
		
	case "indicator_based", "indicator", "signals":
		if *flags.TPStrengthMult <= 0 || *flags.TPStrengthMult > 2.0 {
			return fmt.Errorf("strength multiplier must be between 0 and 2.0, got %f", *flags.TPStrengthMult)
		}
		
		// Validate indicator weights if provided
		if *flags.TPIndicatorWeights != "" {
			weights, err := parseIndicatorWeights(*flags.TPIndicatorWeights)
			if err != nil {
				return fmt.Errorf("invalid indicator weights: %w", err)
			}
			
			// Check that total weight doesn't exceed reasonable bounds
			totalWeight := 0.0
			for _, weight := range weights {
				totalWeight += weight
			}
			if totalWeight > 2.0 {
				return fmt.Errorf("total indicator weights (%f) should not exceed 2.0 for reasonable results", totalWeight)
			}
		}
		
	case "fixed":
		// No additional validation needed for fixed mode
		
	default:
		return fmt.Errorf("unsupported Dynamic TP strategy: %s (supported: fixed, volatility_adaptive, indicator_based)", strategy)
	}
	
	return nil
}

// validateBusinessLogicBounds performs comprehensive validation of parameter combinations
// to prevent trading disasters and unreasonable configurations
func validateBusinessLogicBounds(cfg *config.DCAConfig) error {
	// 1. Validate position sizing bounds to prevent balance exhaustion
	if cfg.MaxMultiplier > 5.0 {
		return fmt.Errorf("max multiplier %.2f is dangerously high (>5.0x) - risk of rapid balance depletion", cfg.MaxMultiplier)
	}
	
	// 2. Validate base amount vs balance ratio - allow up to 50% for aggressive strategies
	if cfg.BaseAmount > cfg.InitialBalance*0.5 {
		return fmt.Errorf("base amount $%.2f exceeds 50%% of balance $%.2f - insufficient reserves for DCA strategy", 
			cfg.BaseAmount, cfg.InitialBalance)
	}
	
	// 3. Validate DCA spacing parameters to ensure entries are possible
	if cfg.DCASpacing != nil {
		if err := validateDCASpacingBounds(cfg.DCASpacing); err != nil {
			return fmt.Errorf("DCA spacing validation: %w", err)
		}
	}
	
	// 4. Validate take profit bounds - allow wider range for different market conditions
	if cfg.TPPercent > 0.25 { // 25%
		return fmt.Errorf("take profit %.2f%% is unrealistically high (>25%%) - may never be reached", cfg.TPPercent*100)
	}
	if cfg.TPPercent < 0.002 { // 0.2%
		return fmt.Errorf("take profit %.2f%% is too low (<0.2%%) - insufficient profit margin after fees", cfg.TPPercent*100)
	}
	
	// 5. Validate commission doesn't negate small profits - use more flexible multiplier
	minProfitAfterFees := cfg.Commission * 2.5 // Need 2.5x commission to be reasonably profitable
	if cfg.TPPercent < minProfitAfterFees {
		return fmt.Errorf("take profit %.3f%% insufficient for commission %.3f%% - need at least %.3f%% to be reasonably profitable", 
			cfg.TPPercent*100, cfg.Commission*100, minProfitAfterFees*100)
	}
	
	// 6. Validate indicator parameters are reasonable
	if err := validateIndicatorBounds(cfg); err != nil {
		return fmt.Errorf("indicator validation: %w", err)
	}
	
	// 7. Validate combined risk exposure to prevent total balance exhaustion
	// maxPossiblePosition := cfg.BaseAmount * cfg.MaxMultiplier * 6 // Assume max 6 DCA levels in extreme cases
	// if maxPossiblePosition > cfg.InitialBalance*0.9 {
	// 	return fmt.Errorf("maximum possible position $%.2f exceeds 90%% of balance $%.2f - excessive risk exposure", 
	// 		maxPossiblePosition, cfg.InitialBalance)
	// }
	
	return nil
}

// validateDCASpacingBounds validates DCA spacing strategy parameters
func validateDCASpacingBounds(spacing *config.DCASpacingConfig) error {
	if spacing == nil {
		return nil
	}
	
	switch spacing.Strategy {
	case "fixed":
		if baseThresh, ok := spacing.Parameters["base_threshold"].(float64); ok {
			if baseThresh > 0.08 { // 8%
				return fmt.Errorf("fixed base threshold %.2f%% is too high (>8%%) - DCA entries may never trigger", baseThresh*100)
			}
			if mult, ok := spacing.Parameters["threshold_multiplier"].(float64); ok && mult > 2.0 {
				return fmt.Errorf("threshold multiplier %.2fx is too aggressive (>2.0x) - later DCA levels unreachable", mult)
			}
		}
		
	case "volatility_adaptive":
		if sens, ok := spacing.Parameters["volatility_sensitivity"].(float64); ok && sens > 5.0 {
			return fmt.Errorf("volatility sensitivity %.1fx is too high (>5.0x) - may create unreachable thresholds", sens)
		}
	}
	
	return nil
}

// validateIndicatorBounds validates indicator parameter ranges
func validateIndicatorBounds(cfg *config.DCAConfig) error {
	// Validate RSI parameters
	if cfg.RSIOversold >= cfg.RSIOverbought {
		return fmt.Errorf("RSI oversold %.0f >= overbought %.0f - invalid range", cfg.RSIOversold, cfg.RSIOverbought)
	}
	if cfg.RSIOversold > 40 || cfg.RSIOverbought < 60 {
		return fmt.Errorf("RSI thresholds (%.0f/%.0f) too narrow - may generate excessive signals", cfg.RSIOversold, cfg.RSIOverbought)
	}
	
	// Validate MACD parameters
	if cfg.MACDFast >= cfg.MACDSlow {
		return fmt.Errorf("MACD fast %d >= slow %d - invalid configuration", cfg.MACDFast, cfg.MACDSlow)
	}
	
	// Validate Bollinger Bands
	if cfg.BBStdDev > 4.0 {
		return fmt.Errorf("Bollinger Bands standard deviation %.1f is too high (>4.0) - bands too wide", cfg.BBStdDev)
	}
	if cfg.BBStdDev < 1.0 {
		return fmt.Errorf("Bollinger Bands standard deviation %.1f is too low (<1.0) - bands too narrow", cfg.BBStdDev)
	}
	
	return nil
}
