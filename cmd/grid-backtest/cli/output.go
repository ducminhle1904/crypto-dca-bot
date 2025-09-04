package cli

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
)

// OutputFormatter handles console output formatting
type OutputFormatter struct{}

// NewOutputFormatter creates a new output formatter
func NewOutputFormatter() *OutputFormatter {
	return &OutputFormatter{}
}

// ShowVersion displays version information
func (of *OutputFormatter) ShowVersion(appName, appVersion string) {
	fmt.Printf("%s v%s\n", appName, appVersion)
}

// ShowUsage displays usage information and available config files
func (of *OutputFormatter) ShowUsage(appName string) {
	fmt.Printf("Usage: %s -config <config-file> [options]\n\n", appName)
	fmt.Printf("Required:\n")
	fmt.Printf("  -config <file>    Path to grid configuration file\n\n")
	fmt.Printf("Options:\n")
	fmt.Printf("  -data <file>      Path to historical data file (auto-detected if not provided)\n")
	fmt.Printf("  -output <dir>     Output directory for reports (default: results)\n")
	fmt.Printf("  -symbol <symbol>  Override symbol from config\n")
	fmt.Printf("  -interval <int>   Override interval from config\n")
	fmt.Printf("  -max-candles <n>  Maximum number of candles (0 = all data, default: 0)\n")
	fmt.Printf("  -report           Generate comprehensive Excel report (default: true)\n")
	fmt.Printf("  -verbose          Verbose output\n")
	fmt.Printf("  -version          Show version and exit\n\n")

	// Show available config files
	of.showAvailableConfigs()
}

// showAvailableConfigs displays available configuration files
func (of *OutputFormatter) showAvailableConfigs() {
	configDir := "configs/bybit/grid"
	if files, err := filepath.Glob(filepath.Join(configDir, "*.json")); err == nil && len(files) > 0 {
		fmt.Printf("Available config files in %s/:\n", configDir)
		for _, file := range files {
			fmt.Printf("  - %s\n", file)
		}
	}
}

// ShowHeader displays the application header
func (of *OutputFormatter) ShowHeader(appName, appVersion, configFile string) {
	fmt.Printf("🚀 %s v%s\n", appName, appVersion)
	fmt.Printf("📋 Loading configuration: %s\n", configFile)
}

// ShowConfigSummary displays configuration summary
func (of *OutputFormatter) ShowConfigSummary(gridConfig *config.GridConfig, verbose bool) {
	fmt.Printf("\n📊 Grid Strategy Configuration:\n")
	fmt.Printf("   Symbol: %s (%s)\n", gridConfig.Symbol, gridConfig.Category)
	fmt.Printf("   Trading Mode: %s\n", gridConfig.TradingMode)
	fmt.Printf("   Price Range: $%.2f - $%.2f\n", gridConfig.LowerBound, gridConfig.UpperBound)
	fmt.Printf("   Grid Setup: %d levels, %.1f%% spacing, %.1f%% profit\n",
		gridConfig.GridCount, gridConfig.GridSpacing, gridConfig.ProfitPercent)
	fmt.Printf("   Position: $%.2f size, %.1fx leverage\n", gridConfig.PositionSize, gridConfig.Leverage)
	fmt.Printf("   Risk: $%.2f balance, %.2f%% commission\n", gridConfig.InitialBalance, gridConfig.Commission*100)
	
	if gridConfig.UseExchangeConstraints {
		fmt.Printf("   Exchange Constraints: %s (min: %f, step: %f)\n",
			gridConfig.ExchangeName, gridConfig.MinOrderQty, gridConfig.QtyStep)
	}

	if verbose {
		of.showVerboseConfig(gridConfig)
	}
}

// showVerboseConfig displays detailed configuration information
func (of *OutputFormatter) showVerboseConfig(gridConfig *config.GridConfig) {
	fmt.Printf("\n🔍 Detailed Configuration:\n")
	fmt.Printf("   Grid Levels: %d\n", gridConfig.GridCount)
	fmt.Printf("   Spacing Type: Percentage-based (%.2f%%)\n", gridConfig.GridSpacing)
	fmt.Printf("   Price Bounds: $%.2f to $%.2f\n", gridConfig.LowerBound, gridConfig.UpperBound)
	fmt.Printf("   Position Sizing: Fixed $%.2f per grid\n", gridConfig.PositionSize)
	fmt.Printf("   Profit Strategy: %.2f%% target per position\n", gridConfig.ProfitPercent)
	fmt.Printf("   Risk Management: %.2f%% commission, %.1fx leverage\n", gridConfig.Commission*100, gridConfig.Leverage)
	
	if gridConfig.UseExchangeConstraints {
		fmt.Printf("   Exchange: %s\n", gridConfig.ExchangeName)
		fmt.Printf("   Min Order Qty: %.6f\n", gridConfig.MinOrderQty)
		fmt.Printf("   Qty Step: %.6f\n", gridConfig.QtyStep)
		fmt.Printf("   Tick Size: %.6f\n", gridConfig.TickSize)
		fmt.Printf("   Min Notional: $%.2f\n", gridConfig.MinNotional)
		fmt.Printf("   Max Leverage: %.1fx\n", gridConfig.MaxLeverage)
	}
}

// ShowOverride displays configuration override information
func (of *OutputFormatter) ShowOverride(field, value string) {
	fmt.Printf("🔄 %s overridden: %s\n", strings.Title(field), value)
}

// ShowDataInfo displays data loading information
func (of *OutputFormatter) ShowDataInfo(dataFile string, candleCount int, startTime, endTime, timeFormat string) {
	fmt.Printf("📊 Loading market data: %s\n", dataFile)
	fmt.Printf("✅ Loaded %d candles (%s to %s)\n", candleCount, startTime, endTime)
}

// ShowCompletion displays completion message with report information
func (of *OutputFormatter) ShowCompletion(baseOutputDir string) {
	fmt.Printf("📁 Report directory: %s\n", baseOutputDir)
	fmt.Printf("📊 Excel report: grid_backtest_report.xlsx\n")
	fmt.Printf("📄 Text summary: backtest_summary.txt\n")
	fmt.Printf("✅ Grid backtest completed successfully!\n")
}
