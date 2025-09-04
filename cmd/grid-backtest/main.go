package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/cmd/grid-backtest/cli"
	"github.com/ducminhle1904/crypto-dca-bot/internal/grid"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/reporting"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/joho/godotenv"
)

// GridBacktestRunner runs backtests specifically for grid strategies
type GridBacktestRunner struct {
	strategy       strategy.Strategy
	configuration  *config.GridConfig
}

// NewGridBacktestRunner creates a new grid backtest runner
func NewGridBacktestRunner(strategy strategy.Strategy, config *config.GridConfig) *GridBacktestRunner {
	return &GridBacktestRunner{
		strategy:      strategy,
		configuration: config,
	}
}

// RunBacktest executes a complete backtest using the grid strategy
func (gbr *GridBacktestRunner) RunBacktest(marketData []types.OHLCV) (*GridBacktestResults, error) {
	if len(marketData) == 0 {
		return nil, fmt.Errorf("no market data provided")
	}
	
	fmt.Printf("🚀 Starting Grid Backtest: %s\n", gbr.strategy.GetName())
	fmt.Printf("📊 Data Range: %s to %s (%d candles)\n",
		marketData[0].Timestamp.Format(DefaultTimeFormat),
		marketData[len(marketData)-1].Timestamp.Format(DefaultTimeFormat),
		len(marketData))
	
	results := &GridBacktestResults{
		StrategyName:   gbr.strategy.GetName(),
		StartTime:      marketData[0].Timestamp,
		EndTime:        marketData[len(marketData)-1].Timestamp,
		TotalCandles:   len(marketData),
		Decisions:      make([]DecisionRecord, 0),
	}
	
	// Track performance
	startTime := time.Now()
	buyDecisions := 0
	sellDecisions := 0
	holdDecisions := 0
	
	// Process each candle
	for i, candle := range marketData {
		// Create data window (for compatibility with strategy interface)
		window := []types.OHLCV{candle}
		
		// Get strategy decision
		decision, err := gbr.strategy.ShouldExecuteTrade(window)
		if err != nil {
			return nil, fmt.Errorf("strategy error at candle %d: %w", i, err)
		}
		
		// Record decision
		record := DecisionRecord{
			Timestamp:  candle.Timestamp,
			Price:      candle.Close,
			Action:     decision.Action,
			Amount:     decision.Amount,
			Confidence: decision.Confidence,
			Strength:   decision.Strength,
			Reason:     decision.Reason,
		}
		results.Decisions = append(results.Decisions, record)
		
		// Count decisions
		switch decision.Action {
		case strategy.ActionBuy:
			buyDecisions++
		case strategy.ActionSell:
			sellDecisions++
		case strategy.ActionHold:
			holdDecisions++
		}
		
		// Progress reporting
		if i%ProgressReportInterval == 0 || i == len(marketData)-1 {
			fmt.Printf("  📈 Processed %d/%d candles (%.1f%%) - Price: $%.2f, Action: %s\n",
				i+1, len(marketData), float64(i+1)/float64(len(marketData))*100,
				candle.Close, decision.Action)
		}
	}
	
	processingTime := time.Since(startTime)
	
	// Get final statistics from GridStrategy
	if gridStrategy, ok := gbr.strategy.(*strategy.GridStrategy); ok {
		stats := gridStrategy.GetStatistics()
		
		results.FinalBalance = stats["current_balance"].(float64)
		results.InitialBalance = stats["initial_balance"].(float64)
		results.TotalReturn = stats["total_return"].(float64)
		results.TotalTrades = stats["total_trades"].(int)
		results.SuccessfulTrades = stats["successful_trades"].(int)
		results.WinRate = stats["win_rate"].(float64)
		results.MaxConcurrentPositions = stats["max_concurrent_pos"].(int)
		results.ActivePositions = stats["active_positions"].(int)
		results.TotalRealized = stats["total_realized"].(float64)
		results.TotalUnrealized = stats["total_unrealized"].(float64)
		
		// Get position details
		results.ActiveGridPositions = len(gridStrategy.GetActivePositions())
		results.GridLevels = len(gridStrategy.GetGridLevels())
	}
	
	// Set summary statistics
	results.BuyDecisions = buyDecisions
	results.SellDecisions = sellDecisions
	results.HoldDecisions = holdDecisions
	results.ProcessingTime = processingTime
	
	fmt.Printf("✅ Backtest completed in %v\n", processingTime)
	
	return results, nil
}

// GridBacktestResults contains results from a grid backtest
type GridBacktestResults struct {
	// Strategy Info
	StrategyName   string    `json:"strategy_name"`
	StartTime      time.Time `json:"start_time"`
	EndTime        time.Time `json:"end_time"`
	TotalCandles   int       `json:"total_candles"`
	ProcessingTime time.Duration `json:"processing_time"`
	
	// Trading Results
	InitialBalance  float64 `json:"initial_balance"`
	FinalBalance    float64 `json:"final_balance"`
	TotalReturn     float64 `json:"total_return"`
	TotalTrades     int     `json:"total_trades"`
	SuccessfulTrades int    `json:"successful_trades"`
	WinRate         float64 `json:"win_rate"`
	TotalRealized   float64 `json:"total_realized"`
	TotalUnrealized float64 `json:"total_unrealized"`
	
	// Grid-Specific
	MaxConcurrentPositions int `json:"max_concurrent_positions"`
	ActivePositions        int `json:"active_positions"`
	ActiveGridPositions    int `json:"active_grid_positions"`
	GridLevels            int `json:"grid_levels"`
	
	// Decision Summary
	BuyDecisions  int `json:"buy_decisions"`
	SellDecisions int `json:"sell_decisions"`
	HoldDecisions int `json:"hold_decisions"`
	
	// Detailed Records
	Decisions []DecisionRecord `json:"decisions"`
}

// DecisionRecord represents a single strategy decision
type DecisionRecord struct {
	Timestamp  time.Time           `json:"timestamp"`
	Price      float64             `json:"price"`
	Action     strategy.TradeAction `json:"action"`
	Amount     float64             `json:"amount"`
	Confidence float64             `json:"confidence"`
	Strength   float64             `json:"strength"`
	Reason     string              `json:"reason"`
}

// PrintResults displays a comprehensive backtest report
func (results *GridBacktestResults) PrintResults() {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("📊 GRID BACKTEST RESULTS: %s\n", results.StrategyName)
	fmt.Printf(strings.Repeat("=", 60) + "\n")
	
	// Time Information
	fmt.Printf("⏰ Time Period: %s to %s\n",
		results.StartTime.Format("2006-01-02 15:04"),
		results.EndTime.Format("2006-01-02 15:04"))
	fmt.Printf("📈 Total Candles: %d (Processing Time: %v)\n",
		results.TotalCandles, results.ProcessingTime)
	
	// Financial Performance
	fmt.Printf("\n💰 FINANCIAL PERFORMANCE:\n")
	fmt.Printf("   Initial Balance: $%.2f\n", results.InitialBalance)
	fmt.Printf("   Final Balance:   $%.2f\n", results.FinalBalance)
	fmt.Printf("   Total Return:    %.2f%%\n", results.TotalReturn*100)
	fmt.Printf("   Realized P&L:    $%.2f\n", results.TotalRealized)
	fmt.Printf("   Unrealized P&L:  $%.2f\n", results.TotalUnrealized)
	
	// Trading Statistics
	fmt.Printf("\n📊 TRADING STATISTICS:\n")
	fmt.Printf("   Total Trades:      %d\n", results.TotalTrades)
	fmt.Printf("   Successful Trades: %d\n", results.SuccessfulTrades)
	fmt.Printf("   Win Rate:          %.1f%%\n", results.WinRate*100)
	fmt.Printf("   Max Concurrent:    %d positions\n", results.MaxConcurrentPositions)
	fmt.Printf("   Active Positions:  %d\n", results.ActivePositions)
	
	// Grid-Specific Metrics
	fmt.Printf("\n🎯 GRID METRICS:\n")
	fmt.Printf("   Grid Levels:       %d\n", results.GridLevels)
	fmt.Printf("   Active Grids:      %d\n", results.ActiveGridPositions)
	
	// Decision Summary
	totalDecisions := results.BuyDecisions + results.SellDecisions + results.HoldDecisions
	fmt.Printf("\n🤖 DECISION ANALYSIS:\n")
	fmt.Printf("   Buy Decisions:   %d (%.1f%%)\n", results.BuyDecisions, 
		float64(results.BuyDecisions)/float64(totalDecisions)*100)
	fmt.Printf("   Sell Decisions:  %d (%.1f%%)\n", results.SellDecisions,
		float64(results.SellDecisions)/float64(totalDecisions)*100)
	fmt.Printf("   Hold Decisions:  %d (%.1f%%)\n", results.HoldDecisions,
		float64(results.HoldDecisions)/float64(totalDecisions)*100)
	
	// Performance Metrics
	if results.ProcessingTime > 0 {
		candlesPerSecond := float64(results.TotalCandles) / results.ProcessingTime.Seconds()
		fmt.Printf("\n⚡ PERFORMANCE:\n")
		fmt.Printf("   Processing Speed: %.0f candles/second\n", candlesPerSecond)
	}
	
	fmt.Printf(strings.Repeat("=", 60) + "\n")
}

const (
	AppName    = "Grid Backtest"
	AppVersion = "1.0.0"
	
	// Default values
	DefaultDataRoot    = "data"
	DefaultExchange    = "bybit"
	DefaultSymbol      = "BTCUSDT"
	DefaultInterval    = "5m"
	DefaultOutputDir   = "results"
	
	// Progress reporting
	ProgressReportInterval = 1000
	
	// Time formatting
	DefaultTimeFormat     = "2006-01-02 15:04"
	
	// Sample data generation
	MinSampleDataPrice    = 30000.0
	MaxSampleDataPrice    = 70000.0
	SampleDataSpread      = 0.0005
	SampleDataBasePrice   = 50000.0
	SampleDataInterval    = 5 * time.Minute
	SampleDataBaseVolume  = 1000000.0
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Could not load .env file: %v", err)
	}

	// Parse command-line flags
	flags := cli.ParseFlags()
	formatter := cli.NewOutputFormatter()

	// Show version
	if *flags.Version {
		formatter.ShowVersion(AppName, AppVersion)
		return
	}

	// Validate flags
	if err := flags.Validate(); err != nil {
		fmt.Printf("❌ Error: %v\n\n", err)
		formatter.ShowUsage(AppName)
		os.Exit(1)
	}

	// Run the backtest
	if err := runGridBacktest(flags, formatter); err != nil {
		log.Fatalf("❌ Backtest failed: %v", err)
	}
}

// createOutputDirectory creates and returns the output directory path
func createOutputDirectory(baseDir string, gridConfig *config.GridConfig) string {
	outputDir := fmt.Sprintf("%s/grid/%s_%s_%s", 
		baseDir, gridConfig.Symbol, gridConfig.TradingMode, gridConfig.Interval)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("Warning: Could not create output directory: %v", err)
		return baseDir
	}
	return outputDir
}

func runGridBacktest(flags *cli.Flags, formatter *cli.OutputFormatter) error {
	// Show header
	formatter.ShowHeader(AppName, AppVersion, *flags.ConfigFile)

	// Load and parse the config file
	configLoader := cli.NewConfigLoader()
	gridConfig, err := configLoader.LoadGridConfig(*flags.ConfigFile)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Apply overrides
	configLoader.ApplyOverrides(gridConfig, *flags.Symbol, *flags.Interval)
	if *flags.Symbol != "" {
		formatter.ShowOverride("symbol", *flags.Symbol)
	}
	if *flags.Interval != "" {
		formatter.ShowOverride("interval", *flags.Interval)
	}

	// Display configuration summary
	formatter.ShowConfigSummary(gridConfig, *flags.Verbose)

	// Determine data file path
	dataFile := *flags.DataFile
	if dataFile == "" {
		dataFile = getDefaultDataFile(gridConfig.Symbol, gridConfig.Interval)
	}

	fmt.Printf("📊 Loading market data: %s\n", dataFile)

	// Load market data
	marketData, err := loadMarketData(dataFile, *flags.MaxCandles)
	if err != nil {
		return fmt.Errorf("failed to load market data: %w", err)
	}

	formatter.ShowDataInfo(dataFile, len(marketData),
		marketData[0].Timestamp.Format(DefaultTimeFormat),
		marketData[len(marketData)-1].Timestamp.Format(DefaultTimeFormat),
		DefaultTimeFormat)

	// Create grid strategy
	strategyName := fmt.Sprintf("GridBacktest_%s_%s", gridConfig.Symbol, gridConfig.TradingMode)
	gridStrategy, err := strategy.NewGridStrategy(strategyName, gridConfig)
	if err != nil {
		return fmt.Errorf("failed to create grid strategy: %w", err)
	}
	
	// Create and run backtest
	runner := NewGridBacktestRunner(gridStrategy, gridConfig)
	results, err := runner.RunBacktest(marketData)
	if err != nil {
		return fmt.Errorf("backtest execution failed: %w", err)
	}
	
	// Display results
	results.PrintResults()

	// Generate comprehensive report if requested
	if *flags.GenerateReport {
		baseOutputDir := createOutputDirectory(*flags.OutputDir, gridConfig)
		if err := generateComprehensiveReport(results, gridConfig, gridStrategy, baseOutputDir); err != nil {
			log.Printf("⚠️ Report generation failed: %v", err)
		} else {
			formatter.ShowCompletion(baseOutputDir)
		}
	}
	
	return nil
}




func getDefaultDataFile(symbol, interval string) string {
	// Convert interval format from "5m", "15m", "1h", "4h" to directory numbers
	intervalDir := convertIntervalToDirectory(interval)
	// Standard path format: data/{exchange}/linear/{symbol}/{intervalDir}/candles.csv
	return fmt.Sprintf("data/%s/linear/%s/%s/candles.csv", DefaultExchange, symbol, intervalDir)
}

func convertIntervalToDirectory(interval string) string {
	// Map common interval formats to directory names used in your data structure
	intervalMap := map[string]string{
		"1m":  "1",
		"3m":  "3", 
		"5m":  "5",
		"15m": "15",
		"30m": "30",
		"1h":  "60",
		"2h":  "120",
		"4h":  "240",
		"6h":  "360",
		"8h":  "480",
		"12h": "720",
		"1d":  "1440",
		"3d":  "4320",
		"1w":  "10080",
	}
	
	if dir, exists := intervalMap[interval]; exists {
		return dir
	}
	
	// If no mapping found, try to extract just the number part
	// Handle cases like "5m" -> "5", "60m" -> "60"
	if len(interval) > 1 && (interval[len(interval)-1] == 'm' || interval[len(interval)-1] == 'h') {
		numPart := interval[:len(interval)-1]
		if interval[len(interval)-1] == 'h' {
			// Convert hours to minutes for directory name
			if hours, err := strconv.Atoi(numPart); err == nil {
				return strconv.Itoa(hours * 60)
			}
		}
		return numPart
	}
	
	// Return as-is if no conversion needed
	return interval
}

func loadMarketData(dataFile string, maxCandles int) ([]types.OHLCV, error) {
	provider := data.NewCSVProvider()
	
	// Debug information
	fmt.Printf("🔍 Looking for data file: %s\n", dataFile)
	
	// Check if file exists
	if _, err := os.Stat(dataFile); os.IsNotExist(err) {
		fmt.Printf("❌ Historical data file not found: %s\n", dataFile)
		fmt.Printf("💡 Current working directory: %s\n", getWorkingDir())
		
		// Suggest available data files
		suggestAlternativeDataFiles(dataFile)
		
		fmt.Println("⚠️  Generating sample data as fallback...")
		return generateSampleData(600), nil // 600 candles
	}
	
	// Try to load real data
	marketData, err := provider.LoadData(dataFile)
	if err != nil {
		fmt.Printf("⚠️ Could not load data from %s: %v\n", dataFile, err)
		fmt.Println("📊 Generating sample data for testing...")
		
		// Generate sample data as fallback
		return generateSampleData(600), nil // 600 candles
	}
	
	// Apply data limit if specified
	originalCount := len(marketData)
	if maxCandles > 0 && len(marketData) > maxCandles {
		// Use the most recent data (last N candles)
		marketData = marketData[len(marketData)-maxCandles:]
		fmt.Printf("📊 Using last %d candles (from %d total) as requested\n", len(marketData), originalCount)
	}
	
	fmt.Printf("✅ Successfully loaded %d candles from real data\n", len(marketData))
	return marketData, nil
}

func getWorkingDir() string {
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "unknown"
}

func suggestAlternativeDataFiles(requestedFile string) {
	// Extract symbol from requested file path
	parts := strings.Split(requestedFile, string(os.PathSeparator))
	var symbol string
	for i, part := range parts {
		if part == "linear" && i+1 < len(parts) {
			symbol = parts[i+1]
			break
		}
	}
	
	if symbol == "" {
		return
	}
	
	// Check what data is actually available for this symbol
	symbolDir := fmt.Sprintf("data/%s/linear/%s", DefaultExchange, symbol)
	if entries, err := os.ReadDir(symbolDir); err == nil {
		fmt.Printf("💡 Available intervals for %s:\n", symbol)
		for _, entry := range entries {
			if entry.IsDir() {
				dataFile := filepath.Join(symbolDir, entry.Name(), "candles.csv")
				if _, err := os.Stat(dataFile); err == nil {
					fmt.Printf("   - %s minute interval: %s\n", entry.Name(), dataFile)
				}
			}
		}
	} else {
		// Check what symbols are available
		linearDir := fmt.Sprintf("data/%s/linear", DefaultExchange)
		if entries, err := os.ReadDir(linearDir); err == nil {
			fmt.Printf("💡 Available symbols:\n")
			for i, entry := range entries {
				if entry.IsDir() && i < 5 { // Show first 5 symbols
					fmt.Printf("   - %s\n", entry.Name())
				}
			}
			if len(entries) > 5 {
				fmt.Printf("   ... and %d more\n", len(entries)-5)
			}
		}
	}
}

func generateComprehensiveReport(results *GridBacktestResults, gridConfig *config.GridConfig, gridStrategy *strategy.GridStrategy, baseOutputDir string) error {
	// Create the structured output directory: results/grid/{symbol}_{mode}_{timeframe}/
	outputDir := filepath.Join(baseOutputDir, "grid", 
		fmt.Sprintf("%s_%s_%s", gridConfig.Symbol, gridConfig.TradingMode, gridConfig.Interval))
	
	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create comprehensive Excel report with clean filename
	reportFileName := "grid_backtest_report.xlsx"
	excelPath := filepath.Join(outputDir, reportFileName)
	
	fmt.Printf("📊 Generating comprehensive Excel report: %s\n", excelPath)
	
	// Use the advanced grid reporter to create the comprehensive Excel file
	reporter := reporting.NewGridReporter()
	
	// Convert results to GridBacktestResults format
	gridResults := convertToGridBacktestResults(results, gridConfig, gridStrategy)
	
	// Generate the comprehensive Excel report (includes all 5 sheets)
	if err := reporter.WriteGridReportXLSX(gridResults, excelPath); err != nil {
		return fmt.Errorf("Excel report generation failed: %w", err)
	}
	
	// Create a summary text file as well for quick reference
	summaryPath := filepath.Join(outputDir, "backtest_summary.txt")
	if err := createTextSummary(results, gridConfig, summaryPath); err != nil {
		log.Printf("⚠️ Text summary generation failed: %v", err)
	}
	
	fmt.Printf("✅ Comprehensive report generated successfully!\n")
	fmt.Printf("📁 Report directory: %s\n", outputDir)
	fmt.Printf("📊 Excel report: %s\n", reportFileName)
	fmt.Printf("📄 Text summary: backtest_summary.txt\n")
	
	return nil
}

func createTextSummary(results *GridBacktestResults, gridConfig *config.GridConfig, summaryPath string) error {
	file, err := os.Create(summaryPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write comprehensive text summary
	fmt.Fprintf(file, "# Grid Backtest Summary Report\n")
	fmt.Fprintf(file, "Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	// Configuration Summary
	fmt.Fprintf(file, "## Configuration\n")
	fmt.Fprintf(file, "Symbol: %s (%s)\n", gridConfig.Symbol, gridConfig.Category)
	fmt.Fprintf(file, "Trading Mode: %s\n", gridConfig.TradingMode)
	fmt.Fprintf(file, "Interval: %s\n", gridConfig.Interval)
	fmt.Fprintf(file, "Price Range: $%.2f - $%.2f\n", gridConfig.LowerBound, gridConfig.UpperBound)
	fmt.Fprintf(file, "Grid Setup: %d levels, %.1f%% spacing, %.1f%% profit target\n", 
		gridConfig.GridCount, gridConfig.GridSpacing, gridConfig.ProfitPercent*100)
	fmt.Fprintf(file, "Position Size: $%.2f\n", gridConfig.PositionSize)
	fmt.Fprintf(file, "Leverage: %.1fx\n", gridConfig.Leverage)
	fmt.Fprintf(file, "Initial Balance: $%.2f\n", gridConfig.InitialBalance)
	fmt.Fprintf(file, "Commission: %.2f%%\n\n", gridConfig.Commission*100)
	
	// Backtest Period
	fmt.Fprintf(file, "## Backtest Period\n")
	fmt.Fprintf(file, "Start Time: %s\n", results.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "End Time: %s\n", results.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(file, "Duration: %s\n", results.EndTime.Sub(results.StartTime).String())
	fmt.Fprintf(file, "Total Candles: %d\n", results.TotalCandles)
	fmt.Fprintf(file, "Processing Time: %v\n\n", results.ProcessingTime)
	
	// Financial Performance
	fmt.Fprintf(file, "## Financial Performance\n")
	fmt.Fprintf(file, "Initial Balance: $%.2f\n", results.InitialBalance)
	fmt.Fprintf(file, "Final Balance: $%.2f\n", results.FinalBalance)
	fmt.Fprintf(file, "Total Return: %.2f%%\n", results.TotalReturn*100)
	fmt.Fprintf(file, "Absolute P&L: $%.2f\n", results.FinalBalance-results.InitialBalance)
	fmt.Fprintf(file, "Realized P&L: $%.2f\n", results.TotalRealized)
	fmt.Fprintf(file, "Unrealized P&L: $%.2f\n\n", results.TotalUnrealized)
	
	// Trading Statistics
	fmt.Fprintf(file, "## Trading Statistics\n")
	fmt.Fprintf(file, "Total Trades: %d\n", results.TotalTrades)
	fmt.Fprintf(file, "Successful Trades: %d\n", results.SuccessfulTrades)
	fmt.Fprintf(file, "Win Rate: %.1f%%\n", results.WinRate*100)
	fmt.Fprintf(file, "Max Concurrent Positions: %d\n", results.MaxConcurrentPositions)
	fmt.Fprintf(file, "Active Positions: %d\n\n", results.ActivePositions)
	
	// Grid Metrics
	fmt.Fprintf(file, "## Grid Metrics\n")
	fmt.Fprintf(file, "Grid Levels: %d\n", results.GridLevels)
	fmt.Fprintf(file, "Active Grid Positions: %d\n", results.ActiveGridPositions)
	if results.GridLevels > 0 {
		fmt.Fprintf(file, "Grid Utilization: %.1f%%\n", float64(results.ActiveGridPositions)/float64(results.GridLevels)*100)
	}
	fmt.Fprintf(file, "\n")
	
	// Decision Analysis
	totalDecisions := results.BuyDecisions + results.SellDecisions + results.HoldDecisions
	fmt.Fprintf(file, "## Decision Analysis\n")
	fmt.Fprintf(file, "Total Decisions: %d\n", totalDecisions)
	if totalDecisions > 0 {
		fmt.Fprintf(file, "Buy Decisions: %d (%.1f%%)\n", results.BuyDecisions, 
			float64(results.BuyDecisions)/float64(totalDecisions)*100)
		fmt.Fprintf(file, "Sell Decisions: %d (%.1f%%)\n", results.SellDecisions,
			float64(results.SellDecisions)/float64(totalDecisions)*100)
		fmt.Fprintf(file, "Hold Decisions: %d (%.1f%%)\n", results.HoldDecisions,
			float64(results.HoldDecisions)/float64(totalDecisions)*100)
	}
	fmt.Fprintf(file, "\n")
	
	// Performance Metrics
	if results.ProcessingTime > 0 {
		candlesPerSecond := float64(results.TotalCandles) / results.ProcessingTime.Seconds()
		fmt.Fprintf(file, "## Performance Metrics\n")
		fmt.Fprintf(file, "Processing Speed: %.0f candles/second\n", candlesPerSecond)
		fmt.Fprintf(file, "Efficiency: High performance backtesting\n\n")
	}
	
	// Exchange Constraints (if enabled)
	if gridConfig.UseExchangeConstraints {
		fmt.Fprintf(file, "## Exchange Constraints (%s)\n", gridConfig.ExchangeName)
		fmt.Fprintf(file, "Min Order Quantity: %.6f\n", gridConfig.MinOrderQty)
		fmt.Fprintf(file, "Quantity Step Size: %.6f\n", gridConfig.QtyStep)
		fmt.Fprintf(file, "Tick Size: %.6f\n", gridConfig.TickSize)
		fmt.Fprintf(file, "Min Notional Value: $%.2f\n", gridConfig.MinNotional)
		fmt.Fprintf(file, "Max Leverage: %.1fx\n\n", gridConfig.MaxLeverage)
	}
	
	// Footer
	fmt.Fprintf(file, "---\n")
	fmt.Fprintf(file, "Report generated by Enhanced DCA Bot - Grid Backtest System v%s\n", AppVersion)
	fmt.Fprintf(file, "For detailed analysis, open the Excel report: grid_backtest_report.xlsx\n")

	return nil
}

// ResultsConverter handles conversion of grid backtest results to reporting format
type ResultsConverter struct {
	config   *config.GridConfig
	strategy *strategy.GridStrategy
	results  *GridBacktestResults
}

// NewResultsConverter creates a new results converter
func NewResultsConverter(config *config.GridConfig, strategy *strategy.GridStrategy, results *GridBacktestResults) *ResultsConverter {
	return &ResultsConverter{
		config:   config,
		strategy: strategy,
		results:  results,
	}
}

// Convert transforms grid backtest results into reporting format
func (rc *ResultsConverter) Convert() *reporting.GridBacktestResults {
	return &reporting.GridBacktestResults{
		StrategyName:           rc.results.StrategyName,
		StartTime:              rc.results.StartTime,
		EndTime:                rc.results.EndTime,
		TotalCandles:           rc.results.TotalCandles,
		ProcessingTime:         rc.results.ProcessingTime,
		Config:                 rc.config,
		InitialBalance:         rc.results.InitialBalance,
		FinalBalance:           rc.results.FinalBalance,
		TotalReturn:            rc.results.TotalReturn,
		TotalTrades:            rc.results.TotalTrades,
		SuccessfulTrades:       rc.results.SuccessfulTrades,
		WinRate:                rc.results.WinRate,
		TotalRealized:          rc.results.TotalRealized,
		TotalUnrealized:        rc.results.TotalUnrealized,
		MaxConcurrentPositions: rc.results.MaxConcurrentPositions,
		GridLevels:             rc.extractGridLevels(),
		GridPositions:          rc.convertActivePositions(),
		AllPositions:           rc.convertAllPositions(),
		GridPerformance:        rc.buildLevelStatistics(),
	}
}

// extractGridLevels converts grid levels to reporting format
func (rc *ResultsConverter) extractGridLevels() []*grid.GridLevel {
	gridLevels := rc.strategy.GetGridLevels()
	reportingGridLevels := make([]*grid.GridLevel, 0, len(gridLevels))
	
	for _, level := range gridLevels {
		// Create a copy for reporting
		levelCopy := *level
		reportingGridLevels = append(reportingGridLevels, &levelCopy)
	}
	
	return reportingGridLevels
}

// buildLevelStatistics creates performance statistics for each grid level
func (rc *ResultsConverter) buildLevelStatistics() map[int]*reporting.GridLevelStats {
	gridLevels := rc.strategy.GetGridLevels()
	allPositions := rc.strategy.GetAllPositions()
	
	// Initialize stats for all levels
	levelStats := make(map[int]*reporting.GridLevelStats)
	for _, level := range gridLevels {
		levelStats[level.Level] = &reporting.GridLevelStats{
			Level:            level.Level,
			Price:            level.Price,
			Direction:        level.Direction,
			TimesTriggered:   0,
			TotalPnL:         0,
			AvgPnL:          0,
			WinRate:         0,
			TotalVolume:     0,
		}
	}
	
	// Analyze all positions to build level statistics
	for _, position := range allPositions {
		if stats, exists := levelStats[position.GridLevel]; exists {
			stats.TimesTriggered++
			stats.TotalVolume += position.Quantity * position.EntryPrice
			
			// Calculate P&L based on position status
			pnl := rc.calculatePositionPnL(position)
			stats.TotalPnL += pnl
			
			if stats.TimesTriggered > 0 {
				stats.AvgPnL = stats.TotalPnL / float64(stats.TimesTriggered)
			}
		}
	}
	
	return levelStats
}

// calculatePositionPnL determines the P&L for a position based on its status
func (rc *ResultsConverter) calculatePositionPnL(position *grid.GridPosition) float64 {
	if position.Status == "closed" && position.RealizedPnL != nil {
		return *position.RealizedPnL
	}
	return position.UnrealizedPnL
}

// convertActivePositions converts active positions to position records
func (rc *ResultsConverter) convertActivePositions() []*reporting.GridPositionRecord {
	activePositions := rc.strategy.GetActivePositions()
	return convertPositionsToRecords(activePositions, rc.results.EndTime)
}

// convertAllPositions converts all positions (active + closed) to position records
func (rc *ResultsConverter) convertAllPositions() []*reporting.GridPositionRecord {
	allPositions := rc.strategy.GetAllPositions()
	return convertPositionsToRecords(allPositions, rc.results.EndTime)
}

// Legacy function for backward compatibility - now uses ResultsConverter
func convertToGridBacktestResults(results *GridBacktestResults, gridConfig *config.GridConfig, gridStrategy *strategy.GridStrategy) *reporting.GridBacktestResults {
	converter := NewResultsConverter(gridConfig, gridStrategy, results)
	return converter.Convert()
}

// Helper functions to eliminate code duplication

// convertPositionsToRecords converts grid positions to position records for reporting
func convertPositionsToRecords(positions interface{}, endTime time.Time) []*reporting.GridPositionRecord {
	var records []*reporting.GridPositionRecord
	
	// Handle both map[int]*grid.GridPosition (active) and []*grid.GridPosition (all)
	switch pos := positions.(type) {
	case map[int]*grid.GridPosition:
		records = make([]*reporting.GridPositionRecord, 0, len(pos))
		for _, position := range pos {
			record := convertPositionToRecord(position, endTime)
			records = append(records, record)
		}
	case []*grid.GridPosition:
		records = make([]*reporting.GridPositionRecord, 0, len(pos))
		for _, position := range pos {
			record := convertPositionToRecord(position, endTime)
			records = append(records, record)
		}
	}
	
	// Sort by entry time (chronological order)
	sort.Slice(records, func(i, j int) bool {
		return records[i].EntryTime.Before(records[j].EntryTime)
	})
	
	return records
}

// convertPositionToRecord converts a single grid position to a position record
func convertPositionToRecord(position *grid.GridPosition, endTime time.Time) *reporting.GridPositionRecord {
	duration := calculatePositionDuration(position, endTime)
	
	return &reporting.GridPositionRecord{
		GridLevel:      position.GridLevel,
		Direction:      position.Direction,
		EntryTime:      position.EntryTime,
		EntryPrice:     position.EntryPrice,
		Quantity:       position.Quantity,
		ExitTime:       position.ExitTime,
		ExitPrice:      position.ExitPrice,
		Duration:       duration,
		RealizedPnL:    position.RealizedPnL,
		UnrealizedPnL:  position.UnrealizedPnL,
		Commission:     position.Commission,
		Status:         position.Status,
	}
}

// calculatePositionDuration calculates the duration of a position
func calculatePositionDuration(position *grid.GridPosition, endTime time.Time) time.Duration {
	if position.Status == "closed" && position.ExitTime != nil {
		return position.ExitTime.Sub(position.EntryTime)
	} else if endTime.After(position.EntryTime) {
		return endTime.Sub(position.EntryTime)
	}
	return time.Since(position.EntryTime)
}

func generateSampleData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	baseTime := time.Now().Add(-time.Duration(count) * SampleDataInterval)
	basePrice := SampleDataBasePrice
	
	for i := 0; i < count; i++ {
		// Simulate realistic price movement
		change := (float64(i%100) - 50) * 0.002    // ±10% gradual movement
		volatility := (float64(i%13) - 6.5) * 0.001 // Random volatility
		
		price := basePrice * (1.0 + change + volatility)
		
		// Ensure price stays reasonable
		if price < MinSampleDataPrice {
			price = MinSampleDataPrice
		}
		if price > MaxSampleDataPrice {
			price = MaxSampleDataPrice
		}
		
		// Generate OHLCV with some realism
		spread := price * SampleDataSpread
		
		data[i] = types.OHLCV{
			Timestamp: baseTime.Add(time.Duration(i) * SampleDataInterval),
			Open:      price - spread,
			High:      price + spread*2,
			Low:       price - spread*2,
			Close:     price,
			Volume:    SampleDataBaseVolume + float64(i%1000)*500, // Volume variation
		}
	}
	
	return data
}
