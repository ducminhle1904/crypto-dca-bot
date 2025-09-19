package main

import (
	"flag"
	"fmt"
	"path/filepath"
	"strings"
)

// DCAFlags holds all command line flags for the DCA backtest command
type DCAFlags struct {
	// Configuration
	ConfigFile       *string
	DataFile         *string
	Symbol           *string
	Interval         *string
	Exchange         *string
	
	// Account settings
	InitialBalance   *float64
	Commission       *float64
	
	// DCA strategy parameters
	BaseAmount               *float64
	MaxMultiplier            *float64
	
	// DCA spacing strategy parameters
	DCASpacingStrategy       *string  // DCA spacing strategy (fixed, volatility_adaptive)
	SpacingBaseThreshold     *float64 // Base threshold for spacing strategy
	SpacingMultiplier        *float64 // Multiplier for fixed spacing strategy  
	SpacingVolatilitySens    *float64 // Volatility sensitivity for adaptive spacing
	SpacingATRPeriod         *int     // ATR period for adaptive spacing
	
	// Take profit parameters
	TPPercent               *float64 // Base take profit percentage
	UseTPLevels             *bool    // Enable multi-level TP system
	
	// Dynamic take profit parameters
	DynamicTPStrategy       *string  // Dynamic TP strategy (fixed, volatility_adaptive, indicator_based)
	TPVolatilityMult        *float64 // Volatility multiplier for dynamic TP
	TPMinPercent            *float64 // Minimum TP percentage
	TPMaxPercent            *float64 // Maximum TP percentage
	TPStrengthMult          *float64 // Signal strength multiplier for indicator-based TP
	TPIndicatorWeights      *string  // Comma-separated indicator:weight pairs for indicator-based TP
	
	// Indicator selection (flexible system)
	Indicators       *string  // Comma-separated list of indicators
	
	// Individual indicator flags
	UseRSI           *bool
	UseMACD          *bool
	UseBB            *bool
	UseEMA           *bool
	UseHullMA        *bool
	UseSuperTrend    *bool
	UseMFI           *bool
	UseKeltner       *bool
	UseWaveTrend     *bool
	UseOBV           *bool
	UseStochasticRSI *bool
	
	// Market regime flag
	MarketRegime     *bool
	
	// Analysis options
	Optimize         *bool
	AllIntervals     *bool
	Period           *string
	
	// Walk-forward validation
	WFEnable         *bool
	WFSplitRatio     *float64
	WFRolling        *bool
	WFTrainDays      *int
	WFTestDays       *int
	WFRollDays       *int
	
	// Output options
	DataRoot         *string
	ConsoleOnly      *bool
	WindowSize       *int
	EnvFile          *string
	
	// Help and version
	ShowVersion      *bool
	ShowHelp         *bool
}

// NewDCAFlags creates and registers all DCA-specific command line flags
func NewDCAFlags() *DCAFlags {
	flags := &DCAFlags{
		// Configuration
		ConfigFile:       flag.String("config", "", "Path to DCA configuration file"),
		DataFile:         flag.String("data", "", "Path to historical data file"),
		Symbol:           flag.String("symbol", "BTCUSDT", "Trading symbol"),
		Interval:         flag.String("interval", "1h", "Data interval (5m, 15m, 1h, 4h, 1d)"),
		Exchange:         flag.String("exchange", DefaultExchange, "Exchange (bybit, binance)"),
		
		// Account settings
		InitialBalance:   flag.Float64("balance", DefaultInitialBalance, "Initial balance"),
		Commission:       flag.Float64("commission", DefaultCommission, "Trading commission (0.0005 = 0.05%)"),
		
		// DCA strategy parameters
		BaseAmount:               flag.Float64("base-amount", DefaultBaseAmount, "Base DCA amount"),
		MaxMultiplier:            flag.Float64("max-multiplier", DefaultMaxMultiplier, "Maximum position multiplier"),
		
		// DCA spacing strategy parameters
		DCASpacingStrategy:       flag.String("dca-spacing", "fixed", "DCA spacing strategy (fixed, volatility_adaptive)"),
		SpacingBaseThreshold:     flag.Float64("spacing-threshold", 0.01, "Base threshold for DCA spacing (0.01 = 1%)"),
		SpacingMultiplier:        flag.Float64("spacing-multiplier", 1.15, "Multiplier for fixed progressive spacing"),
		SpacingVolatilitySens:    flag.Float64("spacing-sensitivity", 1.8, "Volatility sensitivity for adaptive spacing"),
		SpacingATRPeriod:         flag.Int("spacing-atr-period", 14, "ATR period for adaptive spacing"),
		
		// Take profit parameters
		TPPercent:               flag.Float64("tp-percent", 0.02, "Base take profit percentage (0.02 = 2%)"),
		UseTPLevels:             flag.Bool("use-tp-levels", true, "Enable multi-level TP system (5 levels)"),
		
		// Dynamic take profit parameters
		DynamicTPStrategy:       flag.String("dynamic-tp", "fixed", "Dynamic TP strategy (fixed, volatility_adaptive, indicator_based)"),
		TPVolatilityMult:        flag.Float64("tp-volatility-mult", 0.5, "Volatility multiplier for dynamic TP (0.5 = half of ATR)"),
		TPMinPercent:            flag.Float64("tp-min-percent", 0.01, "Minimum TP percentage (0.01 = 1%)"),
		TPMaxPercent:            flag.Float64("tp-max-percent", 0.05, "Maximum TP percentage (0.05 = 5%)"),
		TPStrengthMult:          flag.Float64("tp-strength-mult", 0.3, "Signal strength multiplier for indicator-based TP"),
		TPIndicatorWeights:      flag.String("tp-indicator-weights", "", "Comma-separated indicator:weight pairs (e.g., rsi:0.3,macd:0.4)"),
		
		// Indicator selection (flexible system)
		Indicators:       flag.String("indicators", "", "Comma-separated list of indicators (e.g., rsi,macd,supertrend)"),
		
		// Individual indicator flags
		UseRSI:           flag.Bool("rsi", false, "Include RSI indicator"),
		UseMACD:          flag.Bool("macd", false, "Include MACD indicator"), 
		UseBB:            flag.Bool("bb", false, "Include Bollinger Bands indicator"),
		UseEMA:           flag.Bool("ema", false, "Include EMA indicator"),
		UseHullMA:        flag.Bool("hullma", false, "Include Hull MA indicator"),
		UseSuperTrend:    flag.Bool("supertrend", false, "Include SuperTrend indicator"),
		UseMFI:           flag.Bool("mfi", false, "Include MFI indicator"),
		UseKeltner:       flag.Bool("keltner", false, "Include Keltner Channels indicator"),
		UseWaveTrend:     flag.Bool("wavetrend", false, "Include WaveTrend indicator"),
		UseOBV:           flag.Bool("obv", false, "Include OBV (On-Balance Volume) indicator"),
		UseStochasticRSI: flag.Bool("stochrsi", false, "Include Stochastic RSI indicator"),
		
		// Market regime flag
		MarketRegime:     flag.Bool("market-regime", false, "Enable market regime-based signal consensus (2/3/4 indicators for favorable/normal/hostile conditions)"),
		
		// Analysis options
		Optimize:         flag.Bool("optimize", false, "Run genetic algorithm optimization"),
		AllIntervals:     flag.Bool("all-intervals", false, "Test all available intervals"),
		Period:           flag.String("period", "", "Limit data to period (7d, 30d, 180d, 365d)"),
		
		// Walk-forward validation
		WFEnable:         flag.Bool("wf-enable", false, "Enable walk-forward validation"),
		WFSplitRatio:     flag.Float64("wf-split-ratio", 0.7, "Train/test split (0.7 = 70% train)"),
		WFRolling:        flag.Bool("wf-rolling", false, "Use rolling walk-forward"),
		WFTrainDays:      flag.Int("wf-train-days", 180, "Training window (days)"),
		WFTestDays:       flag.Int("wf-test-days", 60, "Test window (days)"),
		WFRollDays:       flag.Int("wf-roll-days", 30, "Roll step (days)"),
		
		// Output options
		DataRoot:         flag.String("data-root", DefaultDataRoot, "Data root directory"),
		ConsoleOnly:      flag.Bool("console-only", false, "Console output only (no files)"),
		WindowSize:       flag.Int("window", DefaultWindowSize, "Analysis window size"),
		EnvFile:          flag.String("env", ".env", "Environment file path"),
		
		// Help and version
		ShowVersion:      flag.Bool("version", false, "Show version information"),
		ShowHelp:         flag.Bool("help", false, "Show detailed help"),
	}
	
	return flags
}

// PrintDCAUsageExamples prints usage examples specific to DCA backtesting
func PrintDCAUsageExamples() {
	examples := []struct {
		command     string
		description string
	}{
		{
			"dca-backtest -symbol BTCUSDT -interval 1h",
			"Basic DCA backtest for BTC on 1-hour timeframe",
		},
		{
			"dca-backtest -config configs/bybit/btc_1h.json",
			"Load configuration from file",
		},
		{
			"dca-backtest -symbol ETHUSDT -optimize",
			"Optimize DCA parameters for ETH",
		},
		{
			"dca-backtest -symbol BTCUSDT -all-intervals",
			"Test all available timeframes for BTC",
		},
		{
			"dca-backtest -symbol BTCUSDT -optimize -wf-enable",
			"Optimize with walk-forward validation",
		},
		{
			"dca-backtest -symbol BTCUSDT -advanced-combo -base-amount 50 -max-multiplier 2.5",
			"Use advanced indicators with custom DCA parameters",
		},
		{
			"dca-backtest -symbol ADAUSDT -period 30d -optimize",
			"Optimize ADA strategy using only last 30 days of data",
		},
		{
			"dca-backtest -symbol BTCUSDT -optimize -wf-rolling -wf-train-days 90 -wf-test-days 30",
			"Optimize with rolling walk-forward validation (90-day train, 30-day test)",
		},
		{
			"dca-backtest -symbol BTCUSDT -dca-spacing fixed -spacing-threshold 0.02 -spacing-multiplier 1.2",
			"Use fixed progressive DCA spacing (2% base, 1.2x multiplier)",
		},
		{
			"dca-backtest -symbol ETHUSDT -dca-spacing volatility_adaptive -spacing-sensitivity 2.0 -spacing-atr-period 21",
			"Use volatility-adaptive DCA spacing with high sensitivity",
		},
		{
			"dca-backtest -symbol BTCUSDT -optimize -dca-spacing volatility_adaptive",
			"Optimize with volatility-adaptive DCA spacing strategy",
		},
		{
			"dca-backtest -symbol BTCUSDT -dynamic-tp volatility_adaptive -tp-volatility-mult 0.5",
			"Use volatility-adaptive dynamic TP (0.5x ATR multiplier)",
		},
		{
			"dca-backtest -symbol ETHUSDT -dynamic-tp indicator_based -tp-indicator-weights \"rsi:0.4,macd:0.3,bb:0.3\"",
			"Use indicator-based dynamic TP with custom weights",
		},
		{
			"dca-backtest -symbol ADAUSDT -dynamic-tp volatility_adaptive -tp-min-percent 0.005 -tp-max-percent 0.08",
			"Dynamic TP with custom bounds (0.5% min, 8% max)",
		},
	}
	
	fmt.Printf("\n📚 USAGE EXAMPLES:\n")
	fmt.Printf("%s\n", strings.Repeat("-", 60))
	
	for _, example := range examples {
		fmt.Printf("\n• %s\n", example.description)
		fmt.Printf("  %s\n", example.command)
	}
}

// PrintDCAFlagGroups prints flags organized by category for better readability
func PrintDCAFlagGroups() {
	fmt.Printf(`
📊 CONFIGURATION FLAGS:
  -config FILE          Load configuration from JSON file
  -symbol SYMBOL        Trading symbol (default: BTCUSDT)
  -interval INTERVAL    Time interval: 5m, 15m, 1h, 4h, 1d (default: 1h)
  -exchange EXCHANGE    Exchange: bybit, binance (default: bybit)
  -data FILE            Override data file path

💰 ACCOUNT FLAGS:
  -balance AMOUNT       Initial balance (default: 500)
  -commission RATE      Trading commission (default: 0.0005)

🔄 DCA STRATEGY FLAGS:
  -base-amount AMOUNT   Base DCA amount (default: 40)
  -max-multiplier MULT  Maximum position multiplier (default: 3.0)

📊 DCA SPACING STRATEGY FLAGS:
  -dca-spacing STRATEGY         DCA spacing strategy: fixed, volatility_adaptive (default: fixed)
  -spacing-threshold PCT        Base threshold for DCA spacing (default: 0.01)
  -spacing-multiplier MULT      Multiplier for fixed progressive spacing (default: 1.15)
  -spacing-sensitivity SENS     Volatility sensitivity for adaptive spacing (default: 1.8)
  -spacing-atr-period PERIOD    ATR period for adaptive spacing (default: 14)

🎯 TAKE PROFIT FLAGS:
  -tp-percent PCT               Base take profit percentage (default: 0.02)
  -use-tp-levels                Enable multi-level TP system (default: true)

🚀 DYNAMIC TAKE PROFIT FLAGS:
  -dynamic-tp STRATEGY          Dynamic TP strategy: fixed, volatility_adaptive, indicator_based (default: fixed)
  -tp-volatility-mult MULT      Volatility multiplier for dynamic TP (default: 0.5)
  -tp-min-percent PCT           Minimum TP percentage (default: 0.01)
  -tp-max-percent PCT           Maximum TP percentage (default: 0.05)
  -tp-strength-mult MULT        Signal strength multiplier for indicator-based TP (default: 0.3)
  -tp-indicator-weights PAIRS   Indicator weights for dynamic TP (e.g., rsi:0.3,macd:0.4)

🎯 MARKET REGIME FLAGS:
  -market-regime        Enable market regime-based signal consensus (2/3/4 indicators for favorable/normal/hostile conditions)

🧬 ANALYSIS FLAGS:
  -optimize             Run genetic algorithm optimization
  -all-intervals        Test all available intervals for symbol
  -period PERIOD        Limit data to period (7d, 30d, 180d, 365d)

🔄 WALK-FORWARD VALIDATION FLAGS:
  -wf-enable            Enable walk-forward validation
  -wf-split-ratio RATIO Train/test split ratio (default: 0.7)
  -wf-rolling           Use rolling window instead of simple split
  -wf-train-days DAYS   Training window size (default: 180)
  -wf-test-days DAYS    Test window size (default: 60)
  -wf-roll-days DAYS    Roll forward step (default: 30)

📁 OUTPUT FLAGS:
  -data-root DIR        Data root directory (default: data)
  -console-only         Console output only, no file output
  -window SIZE          Analysis window size (default: 100)
  -env FILE             Environment file path (default: .env)

❓ HELP FLAGS:
  -version              Show version information
  -help                 Show this help message
`)
}

// ValidateDCAFlags performs validation on DCA-specific flag combinations
func ValidateDCAFlags(flags *DCAFlags) error {
	// Validate base amount
	if *flags.BaseAmount <= 0 {
		return fmt.Errorf("base amount must be positive, got: %.2f", *flags.BaseAmount)
	}
	
	// Validate multiplier
	if *flags.MaxMultiplier <= 1.0 {
		return fmt.Errorf("max multiplier must be greater than 1.0, got: %.2f", *flags.MaxMultiplier)
	}
	
	// Price threshold validation is now handled by DCA spacing strategy validation
	
	// Validate walk-forward settings if enabled
	if *flags.WFEnable {
		if *flags.WFSplitRatio <= 0 || *flags.WFSplitRatio >= 1.0 {
			return fmt.Errorf("walk-forward split ratio must be between 0 and 1.0, got: %.2f", *flags.WFSplitRatio)
		}
		
		if *flags.WFRolling {
			if *flags.WFTrainDays <= 0 || *flags.WFTestDays <= 0 || *flags.WFRollDays <= 0 {
				return fmt.Errorf("walk-forward rolling days must be positive")
			}
			
			if *flags.WFTrainDays <= *flags.WFTestDays {
				return fmt.Errorf("training days (%d) should be greater than test days (%d)", 
					*flags.WFTrainDays, *flags.WFTestDays)
			}
		}
	}
	
	// Validate symbol format
	if len(*flags.Symbol) < 3 {
		return fmt.Errorf("symbol must be at least 3 characters, got: %s", *flags.Symbol)
	}
	
	// Validate interval format
	validIntervals := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w", "1M"}
	isValid := false
	for _, valid := range validIntervals {
		if *flags.Interval == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return fmt.Errorf("invalid interval: %s (valid: %s)", *flags.Interval, strings.Join(validIntervals, ", "))
	}
	
	return nil
}

// ResolveConfigPath resolves the configuration file path with smart defaults
func ResolveConfigPath(configFile string) string {
	if configFile == "" {
		return ""
	}
	
	// If no path separators, assume it's in configs/ directory
	if !strings.ContainsAny(configFile, "/\\") {
		// Add .json extension if missing
		if !strings.HasSuffix(strings.ToLower(configFile), ".json") {
			configFile += ".json"
		}
		return filepath.Join("configs", configFile)
	}
	
	return configFile
}


// ResolveIndicators resolves which indicators to use from various input methods
func ResolveIndicators(flags *DCAFlags) ([]string, error) {
	// Check for conflicts first
	hasIndividualFlags := *flags.UseRSI || *flags.UseMACD || *flags.UseBB || *flags.UseEMA ||
		*flags.UseHullMA || *flags.UseSuperTrend || *flags.UseMFI || *flags.UseKeltner ||
		*flags.UseWaveTrend || *flags.UseOBV || *flags.UseStochasticRSI
	hasIndicatorsList := *flags.Indicators != ""
	
	if hasIndividualFlags && hasIndicatorsList {
		return nil, fmt.Errorf("cannot use both individual indicator flags and -indicators list")
	}
	
	var indicators []string
	
	// Priority 1: Indicators list flag
	if *flags.Indicators != "" {
		listIndicators := strings.Split(*flags.Indicators, ",")
		for _, ind := range listIndicators {
			ind = strings.ToLower(strings.TrimSpace(ind))
			if ind == "" {
				continue
			}
			
			// Validate indicator name
			if !isValidIndicator(ind) {
				return nil, fmt.Errorf("invalid indicator: %s", ind)
			}
			
			indicators = append(indicators, ind)
		}
	} else {
		// Priority 2: Individual indicator flags
		if *flags.UseRSI {
			indicators = append(indicators, "rsi")
		}
		if *flags.UseMACD {
			indicators = append(indicators, "macd")
		}
		if *flags.UseBB {
			indicators = append(indicators, "bb")
		}
		if *flags.UseEMA {
			indicators = append(indicators, "ema")
		}
		if *flags.UseHullMA {
			indicators = append(indicators, "hullma")
		}
		if *flags.UseSuperTrend {
			indicators = append(indicators, "supertrend")
		}
		if *flags.UseMFI {
			indicators = append(indicators, "mfi")
		}
		if *flags.UseKeltner {
			indicators = append(indicators, "keltner")
		}
		if *flags.UseWaveTrend {
			indicators = append(indicators, "wavetrend")
		}
		if *flags.UseOBV {
			indicators = append(indicators, "obv")
		}
		if *flags.UseStochasticRSI {
			indicators = append(indicators, "stochrsi")
		}
	}
	
	if len(indicators) == 0 {
		return nil, fmt.Errorf("no indicators specified")
	}
	
	return indicators, nil
}

// isValidIndicator checks if an indicator name is valid
func isValidIndicator(indicator string) bool {
	validIndicators := []string{
		"rsi", "macd", "bb", "bollinger", "ema", "sma",
		"hullma", "hull_ma", "supertrend", "st", "mfi", "keltner", "kc", "wavetrend", "wt", "obv", "stochrsi", "stochastic_rsi", "stoch_rsi",
	}
	
	for _, valid := range validIndicators {
		if indicator == valid {
			return true
		}
	}
	return false
}


// GetIndicatorDescription returns a human-readable description of the indicator selection
func GetIndicatorDescription(indicators []string) string {
	if len(indicators) == 0 {
		return "No indicators"
	}
	
	// Format indicators with proper names
	upperIndicators := make([]string, len(indicators))
	for i, ind := range indicators {
		switch strings.ToLower(ind) {
		case "rsi":
			upperIndicators[i] = "RSI"
		case "macd":
			upperIndicators[i] = "MACD"
		case "bb", "bollinger":
			upperIndicators[i] = "BB"
		case "ema":
			upperIndicators[i] = "EMA"
		case "sma":
			upperIndicators[i] = "SMA"
		case "hullma", "hull_ma":
			upperIndicators[i] = "Hull MA"
		case "supertrend", "st":
			upperIndicators[i] = "SuperTrend"
		case "mfi":
			upperIndicators[i] = "MFI"
		case "keltner", "kc":
			upperIndicators[i] = "Keltner"
		case "wavetrend", "wt":
			upperIndicators[i] = "WaveTrend"
		case "obv":
			upperIndicators[i] = "OBV"
		case "stochrsi", "stochastic_rsi", "stoch_rsi":
			upperIndicators[i] = "Stochastic RSI"
		default:
			upperIndicators[i] = strings.ToUpper(ind)
		}
	}
	
	return "(" + strings.Join(upperIndicators, " + ") + ")"
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
