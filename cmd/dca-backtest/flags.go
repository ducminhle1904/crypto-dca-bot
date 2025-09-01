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
	BaseAmount       *float64
	MaxMultiplier    *float64
	PriceThreshold   *float64
	UseAdvancedCombo *bool
	
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
		BaseAmount:       flag.Float64("base-amount", DefaultBaseAmount, "Base DCA amount"),
		MaxMultiplier:    flag.Float64("max-multiplier", DefaultMaxMultiplier, "Maximum position multiplier"),
		PriceThreshold:   flag.Float64("price-threshold", DefaultPriceThreshold, "Price drop % for next DCA (0.02 = 2%)"),
		UseAdvancedCombo: flag.Bool("advanced-combo", false, "Use advanced indicators (Hull MA, MFI, Keltner, WaveTrend)"),
		
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
  -price-threshold PCT  Price drop %% for next DCA (default: 0.02)
  -advanced-combo       Use advanced indicators instead of classic

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
	
	// Validate price threshold
	if *flags.PriceThreshold <= 0 || *flags.PriceThreshold >= 1.0 {
		return fmt.Errorf("price threshold must be between 0 and 1.0, got: %.4f", *flags.PriceThreshold)
	}
	
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

// GetDCAStrategyDescription returns a description of the DCA strategy configuration
func GetDCAStrategyDescription(useAdvanced bool, baseAmount, maxMultiplier, priceThreshold float64) string {
	combo := "Classic (RSI + MACD + Bollinger Bands + EMA)"
	if useAdvanced {
		combo = "Advanced (Hull MA + MFI + Keltner Channels + WaveTrend)"
	}
	
	return fmt.Sprintf(`
DCA Strategy Configuration:
• Base Amount: $%.2f per entry
• Max Multiplier: %.2fx (maximum position size)
• Price Threshold: %.2f%% (minimum drop for next entry)
• Indicator Combo: %s
• Take Profit: 5-level system (automatically optimized)
• Cycle Mode: Enabled (reinvest profits)
`, baseAmount, maxMultiplier, priceThreshold*100, combo)
}
