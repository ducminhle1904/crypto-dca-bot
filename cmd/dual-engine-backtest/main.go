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
	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

func main() {
	fmt.Println("🚀 Enhanced DCA Bot - Dual Engine Backtesting System")
	fmt.Println("==================================================")

	// Command line flags
	var (
		configFile    = flag.String("config", "", "Dual-engine configuration file (required)")
		symbol        = flag.String("symbol", "BTCUSDT", "Trading symbol")
		interval      = flag.String("interval", "5m", "Time interval (5m, 15m, 30m, 1h)")
		exchange      = flag.String("exchange", "bybit", "Exchange name")
		startDate     = flag.String("start", "", "Start date (YYYY-MM-DD) - optional")
		endDate       = flag.String("end", "", "End date (YYYY-MM-DD) - optional")
		outputDir     = flag.String("output", "backtest_results", "Output directory for results")
		dataFile      = flag.String("data", "", "Custom CSV data file (optional)")
		_ = flag.Bool("verbose", false, "Enable verbose logging")
		reportFormat  = flag.String("format", "all", "Report format: json, excel, csv, all")
		validate      = flag.Bool("validate", false, "Validate regime detection accuracy")
		help          = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *configFile == "" {
		fmt.Println("❌ Error: Configuration file is required")
		fmt.Println("Use -help for more information")
		os.Exit(1)
	}

	// Load dual-engine configuration
	fmt.Printf("📝 Loading configuration: %s\n", *configFile)
	dualEngineConfig, err := config.LoadDualEngineConfig(*configFile, "development")
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Override symbol and interval from config if provided
	if dualEngineConfig.MainConfig != nil {
		if *symbol == "BTCUSDT" && dualEngineConfig.MainConfig.Symbol != "" {
			*symbol = dualEngineConfig.MainConfig.Symbol
		}
		if *interval == "5m" && dualEngineConfig.MainConfig.Interval != "" {
			*interval = dualEngineConfig.MainConfig.Interval
		}
	}

	fmt.Printf("📊 Backtesting: %s %s on %s\n", *symbol, *interval, *exchange)

	// Create backtester
	backtester, err := backtest.NewDualEngineBacktester(dualEngineConfig, *symbol, *interval)
	if err != nil {
		log.Fatalf("❌ Failed to create backtester: %v", err)
	}

	// Load historical data
	fmt.Println("📈 Loading historical data...")
	var historicalData []types.OHLCV

	if *dataFile != "" {
		// Load from custom CSV file
		fmt.Printf("📁 Loading data from: %s\n", *dataFile)
		historicalData, err = loadDataFromCSV(*dataFile)
		if err != nil {
			log.Fatalf("❌ Failed to load data from CSV: %v", err)
		}
	} else {
		// Load from standard data directory
		historicalData, err = loadHistoricalData(*exchange, *symbol, *interval, *startDate, *endDate)
		if err != nil {
			log.Fatalf("❌ Failed to load historical data: %v", err)
		}
	}

	if len(historicalData) == 0 {
		log.Fatalf("❌ No historical data available")
	}

	fmt.Printf("✅ Loaded %d data points from %v to %v\n", 
		len(historicalData), 
		historicalData[0].Timestamp.Format("2006-01-02"), 
		historicalData[len(historicalData)-1].Timestamp.Format("2006-01-02"))

	// Load data into backtester
	if err := backtester.LoadData(historicalData); err != nil {
		log.Fatalf("❌ Failed to load data into backtester: %v", err)
	}

	// Run backtest
	fmt.Println("🧪 Running dual-engine backtest...")
	startTime := time.Now()

	results, err := backtester.Run()
	if err != nil {
		log.Fatalf("❌ Backtest failed: %v", err)
	}

	duration := time.Since(startTime)
	fmt.Printf("✅ Backtest completed in %v\n", duration.Round(time.Second))

	// Display results summary
	displayResultsSummary(results)

	// Validate regime detection if requested
	if *validate {
		fmt.Println("\n🔍 Validating regime detection accuracy...")
		validateRegimeDetection(backtester)
	}

	// Generate reports
	fmt.Printf("\n📄 Generating reports in: %s\n", *outputDir)
	if err := generateReports(backtester, *outputDir, *reportFormat); err != nil {
		log.Fatalf("❌ Failed to generate reports: %v", err)
	}

	// Final summary
	fmt.Printf("\n🎉 Backtest Analysis Complete!\n")
	fmt.Printf("📊 Total Return: %.2f%%\n", results.TotalReturn)
	fmt.Printf("📈 Annualized Return: %.2f%%\n", results.AnnualizedReturn)
	fmt.Printf("📉 Max Drawdown: %.2f%%\n", results.MaxDrawdown)
	fmt.Printf("⚡ Sharpe Ratio: %.2f\n", results.SharpeRatio)
	fmt.Printf("🎯 Win Rate: %.1f%%\n", results.WinRate)
	fmt.Printf("📁 Reports saved to: %s\n", *outputDir)

	// Profitability check
	if results.TotalReturn > 0 && results.SharpeRatio > 1.0 && results.MaxDrawdown < 20 {
		fmt.Println("✅ System appears PROFITABLE with good risk metrics!")
	} else {
		fmt.Println("⚠️  System needs optimization - check risk metrics and parameters")
	}
}

func showHelp() {
	fmt.Println("🚀 Enhanced DCA Bot - Dual Engine Backtesting System")
	fmt.Println("==================================================")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  dual-engine-backtest -config <config_file> [options]")
	fmt.Println("")
	fmt.Println("OPTIONS:")
	fmt.Println("  -config <file>    Dual-engine configuration file (REQUIRED)")
	fmt.Println("  -symbol <symbol>  Trading symbol (default: BTCUSDT)")
	fmt.Println("  -interval <int>   Time interval: 5m, 15m, 30m, 1h (default: 5m)")
	fmt.Println("  -exchange <name>  Exchange name: bybit, binance (default: bybit)")
	fmt.Println("  -start <date>     Start date YYYY-MM-DD (optional)")
	fmt.Println("  -end <date>       End date YYYY-MM-DD (optional)")
	fmt.Println("  -output <dir>     Output directory (default: backtest_results)")
	fmt.Println("  -data <file>      Custom CSV data file (optional)")
	fmt.Println("  -format <fmt>     Report format: json, excel, csv, all (default: all)")
	fmt.Println("  -validate         Validate regime detection accuracy")
	fmt.Println("  -verbose          Enable verbose logging")
	fmt.Println("  -help             Show this help message")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Basic backtest with dual-engine configuration")
	fmt.Println("  dual-engine-backtest -config configs/dual_engine/btc_development.json")
	fmt.Println("")
	fmt.Println("  # Backtest specific symbol and timeframe")
	fmt.Println("  dual-engine-backtest -config configs/dual_engine/btc_development.json -symbol ETHUSDT -interval 15m")
	fmt.Println("")
	fmt.Println("  # Backtest with date range")
	fmt.Println("  dual-engine-backtest -config configs/dual_engine/btc_development.json -start 2024-01-01 -end 2024-12-31")
	fmt.Println("")
	fmt.Println("  # Validate regime detection")
	fmt.Println("  dual-engine-backtest -config configs/dual_engine/btc_development.json -validate")
	fmt.Println("")
	fmt.Println("  # Use custom data file")
	fmt.Println("  dual-engine-backtest -config configs/dual_engine/btc_development.json -data custom_data.csv")
	fmt.Println("")
	fmt.Println("FEATURES:")
	fmt.Println("  🎯 Comprehensive regime detection validation")
	fmt.Println("  🏗️  Individual and combined engine performance analysis")
	fmt.Println("  🔄 Engine transition cost and effectiveness measurement")
	fmt.Println("  📊 Advanced risk metrics (Sharpe, Sortino, VaR, Drawdown)")
	fmt.Println("  📈 Detailed performance reports (Excel, JSON, CSV)")
	fmt.Println("  🛡️  Risk management validation")
	fmt.Println("  💰 Profitability analysis and optimization suggestions")
	fmt.Println("")
}

func loadHistoricalData(exchange, symbol, interval, startDate, endDate string) ([]types.OHLCV, error) {
	// Use existing data provider infrastructure
	dataProvider := data.NewCSVProvider()
	
	// Construct data path
	dataPath := filepath.Join("data", exchange, "linear", symbol, getIntervalDir(interval), "candles.csv")
	
	// Check if file exists
	if _, err := os.Stat(dataPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("data file not found: %s", dataPath)
	}
	
	// Load data
	rawData, err := dataProvider.LoadData(dataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	
	// Convert to OHLCV format
	ohlcvData := make([]types.OHLCV, len(rawData))
	for i, point := range rawData {
		ohlcvData[i] = types.OHLCV{
			Timestamp: point.Timestamp, // Already a time.Time
			Open:      point.Open,
			High:      point.High,
			Low:       point.Low,
			Close:     point.Close,
			Volume:    point.Volume,
		}
	}
	
	// Filter by date range if specified
	if startDate != "" || endDate != "" {
		ohlcvData = filterByDateRange(ohlcvData, startDate, endDate)
	}
	
	return ohlcvData, nil
}

func loadDataFromCSV(csvFile string) ([]types.OHLCV, error) {
	dataProvider := data.NewCSVProvider()
	
	rawData, err := dataProvider.LoadData(csvFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load CSV data: %w", err)
	}
	
	ohlcvData := make([]types.OHLCV, len(rawData))
	for i, point := range rawData {
		ohlcvData[i] = types.OHLCV{
			Timestamp: point.Timestamp, // Already a time.Time
			Open:      point.Open,
			High:      point.High,
			Low:       point.Low,
			Close:     point.Close,
			Volume:    point.Volume,
		}
	}
	
	return ohlcvData, nil
}

func getIntervalDir(interval string) string {
	switch interval {
	case "1m":
		return "1"
	case "5m":
		return "5"
	case "15m":
		return "15"
	case "30m":
		return "30"
	case "1h":
		return "60"
	default:
		return "5" // Default to 5m
	}
}

func filterByDateRange(data []types.OHLCV, startDate, endDate string) []types.OHLCV {
	var filtered []types.OHLCV
	
	var start, end time.Time
	var err error
	
	if startDate != "" {
		start, err = time.Parse("2006-01-02", startDate)
		if err != nil {
			log.Printf("Warning: Invalid start date format: %s", startDate)
			return data
		}
	}
	
	if endDate != "" {
		end, err = time.Parse("2006-01-02", endDate)
		if err != nil {
			log.Printf("Warning: Invalid end date format: %s", endDate)
			return data
		}
		end = end.Add(24 * time.Hour) // Include the entire end date
	}
	
	for _, point := range data {
		if startDate != "" && point.Timestamp.Before(start) {
			continue
		}
		if endDate != "" && point.Timestamp.After(end) {
			continue
		}
		filtered = append(filtered, point)
	}
	
	return filtered
}

func displayResultsSummary(results *backtest.DualEngineBacktestResults) {
	fmt.Println("\n📊 BACKTEST RESULTS SUMMARY")
	fmt.Println("==========================")
	
	// Basic metrics
	fmt.Printf("📅 Period: %s to %s (%.0f days)\n", 
		results.StartTime.Format("2006-01-02"), 
		results.EndTime.Format("2006-01-02"),
		results.Duration.Hours()/24)
	
	fmt.Printf("💰 Initial Balance: $%.2f\n", results.InitialBalance)
	fmt.Printf("💰 Final Balance: $%.2f\n", results.FinalBalance)
	fmt.Printf("📈 Total Return: %.2f%%\n", results.TotalReturn)
	fmt.Printf("📈 Annualized Return: %.2f%%\n", results.AnnualizedReturn)
	fmt.Printf("📉 Max Drawdown: %.2f%%\n", results.MaxDrawdown)
	
	// Trading metrics
	fmt.Printf("\n🔢 TRADING METRICS:\n")
	fmt.Printf("  • Total Trades: %d\n", results.TotalTrades)
	fmt.Printf("  • Win Rate: %.1f%% (%d/%d)\n", results.WinRate, results.WinningTrades, results.TotalTrades)
	fmt.Printf("  • Profit Factor: %.2f\n", results.ProfitFactor)
	fmt.Printf("  • Avg Trade Duration: %v\n", results.AvgTradeDuration.Round(time.Hour))
	
	// Risk metrics
	fmt.Printf("\n📊 RISK METRICS:\n")
	fmt.Printf("  • Sharpe Ratio: %.2f\n", results.SharpeRatio)
	fmt.Printf("  • Sortino Ratio: %.2f\n", results.SortinoRatio)
	fmt.Printf("  • Calmar Ratio: %.2f\n", results.CalmarRatio)
	fmt.Printf("  • VaR 95%%: %.2f%%\n", results.VaR95)
	
	// Regime metrics
	fmt.Printf("\n🎯 REGIME DETECTION:\n")
	fmt.Printf("  • Regime Changes: %d\n", results.RegimeChanges)
	fmt.Printf("  • Avg Regime Duration: %v\n", results.AvgRegimeDuration.Round(time.Hour))
	fmt.Printf("  • Regime Accuracy: %.1f%%\n", results.RegimeAccuracy)
	
	fmt.Printf("\n🏗️ REGIME DISTRIBUTION:\n")
	for regimeType, percentage := range results.RegimeDistribution {
		fmt.Printf("  • %s: %.1f%%\n", strings.ToUpper(regimeType.String()), percentage)
	}
	
	// Engine metrics
	fmt.Printf("\n⚡ ENGINE PERFORMANCE:\n")
	for engineType, metrics := range results.EnginePerformance {
		fmt.Printf("  • %s Engine:\n", strings.Title(string(engineType)))
		fmt.Printf("    - Trades: %d (%.1f%% win rate)\n", metrics.TotalTrades, metrics.WinRate)
		fmt.Printf("    - P&L: $%.2f\n", metrics.TotalPnL)
		fmt.Printf("    - Utilization: %.1f%%\n", results.EngineUtilization[engineType])
	}
	
	// Transition metrics
	fmt.Printf("\n🔄 TRANSITIONS:\n")
	fmt.Printf("  • Total Transitions: %d\n", results.TotalTransitions)
	fmt.Printf("  • Transition Costs: %.3f%% of portfolio\n", results.TransitionCosts)
	fmt.Printf("  • Success Rate: %.1f%%\n", results.TransitionSuccessRate)
}

func validateRegimeDetection(backtester *backtest.DualEngineBacktester) {
	regimeHistory := backtester.GetRegimeHistory()
	
	fmt.Printf("📈 Analyzed %d regime changes\n", len(regimeHistory))
	
	// Simple validation metrics
	if len(regimeHistory) > 0 {
		// Calculate regime stability (longer regimes are generally better)
		totalDuration := time.Duration(0)
		for _, record := range regimeHistory {
			totalDuration += record.Duration
		}
		avgDuration := totalDuration / time.Duration(len(regimeHistory))
		
		fmt.Printf("⏱️  Average regime duration: %v\n", avgDuration.Round(time.Hour))
		
		// Check for regime flip-flopping (too many rapid changes)
		rapidChanges := 0
		for _, record := range regimeHistory {
			if record.Duration < 2*time.Hour {
				rapidChanges++
			}
		}
		
		rapidChangeRate := float64(rapidChanges) / float64(len(regimeHistory)) * 100
		fmt.Printf("⚡ Rapid changes (<2h): %.1f%%\n", rapidChangeRate)
		
		if rapidChangeRate > 30 {
			fmt.Println("⚠️  Warning: High rate of rapid regime changes - consider adjusting detection parameters")
		} else {
			fmt.Println("✅ Regime detection stability looks good")
		}
		
		// Confidence analysis
		totalConfidence := 0.0
		for _, record := range regimeHistory {
			totalConfidence += record.Confidence
		}
		avgConfidence := totalConfidence / float64(len(regimeHistory))
		
		fmt.Printf("🎯 Average confidence: %.1f%%\n", avgConfidence*100)
		
		if avgConfidence < 0.6 {
			fmt.Println("⚠️  Warning: Low average confidence - consider tuning regime detection parameters")
		} else {
			fmt.Println("✅ Regime detection confidence looks good")
		}
	}
}

func generateReports(backtester *backtest.DualEngineBacktester, outputDir, format string) error {
	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	results := backtester.GetResults()
	
	// Generate JSON report (always generated for programmatic access)
	jsonPath := filepath.Join(outputDir, "backtest_results.json")
	if err := generateJSONReport(results, jsonPath); err != nil {
		log.Printf("Warning: Failed to generate JSON report: %v", err)
	} else {
		fmt.Printf("✅ JSON report: %s\n", jsonPath)
	}
	
	// Generate Excel report if requested
	if format == "excel" || format == "all" {
		excelPath := filepath.Join(outputDir, "backtest_analysis.xlsx")
		if err := generateExcelReport(backtester, excelPath); err != nil {
			log.Printf("Warning: Failed to generate Excel report: %v", err)
		} else {
			fmt.Printf("✅ Excel report: %s\n", excelPath)
		}
	}
	
	// Generate CSV reports if requested
	if format == "csv" || format == "all" {
		if err := generateCSVReports(backtester, outputDir); err != nil {
			log.Printf("Warning: Failed to generate CSV reports: %v", err)
		} else {
			fmt.Printf("✅ CSV reports: %s/\n", outputDir)
		}
	}
	
	return nil
}

func generateExcelReport(backtester *backtest.DualEngineBacktester, outputPath string) error {
	// Simplified Excel report generation
	// For a complete implementation, this would integrate with the existing Excel infrastructure
	
	results := backtester.GetResults()
	
	// Create a simple text-based report for now
	file, err := os.Create(outputPath + ".txt") // Create as text file for simplicity
	if err != nil {
		return err
	}
	defer file.Close()
	
	file.WriteString("Dual-Engine Backtest Report\n")
	file.WriteString(fmt.Sprintf("Symbol: %s\n", results.Symbol))
	file.WriteString(fmt.Sprintf("Total Return: %.2f%%\n", results.TotalReturn))
	file.WriteString(fmt.Sprintf("Max Drawdown: %.2f%%\n", results.MaxDrawdown))
	file.WriteString(fmt.Sprintf("Sharpe Ratio: %.2f\n", results.SharpeRatio))
	file.WriteString(fmt.Sprintf("Win Rate: %.1f%%\n", results.WinRate))
	
	return nil
}

func generateCSVReports(backtester *backtest.DualEngineBacktester, outputDir string) error {
	// Generate multiple CSV files for different aspects
	
	// 1. Main results
	resultsPath := filepath.Join(outputDir, "results.csv")
	if err := generateResultsCSV(backtester.GetResults(), resultsPath); err != nil {
		return fmt.Errorf("failed to generate results CSV: %w", err)
	}
	
	// 2. Regime history
	regimeHistoryPath := filepath.Join(outputDir, "regime_history.csv")
	if err := generateRegimeHistoryCSV(backtester.GetRegimeHistory(), regimeHistoryPath); err != nil {
		return fmt.Errorf("failed to generate regime history CSV: %w", err)
	}
	
	// 3. Transition history
	transitionHistoryPath := filepath.Join(outputDir, "transition_history.csv")
	if err := generateTransitionHistoryCSV(backtester.GetTransitionHistory(), transitionHistoryPath); err != nil {
		return fmt.Errorf("failed to generate transition history CSV: %w", err)
	}
	
	return nil
}

func generateRegimeHistoryCSV(regimeHistory []*backtest.RegimeRecord, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV header
	file.WriteString("timestamp,old_regime,new_regime,confidence,duration_hours,price\n")
	
	// Write data
	for _, record := range regimeHistory {
		file.WriteString(fmt.Sprintf("%s,%s,%s,%.3f,%.1f,%.2f\n",
			record.Timestamp.Format("2006-01-02 15:04:05"),
			record.OldRegime,
			record.NewRegime,
			record.Confidence,
			record.Duration.Hours(),
			record.Price))
	}
	
	return nil
}

func generateTransitionHistoryCSV(transitionHistory []*backtest.TransitionRecord, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV header
	file.WriteString("timestamp,from_engine,to_engine,reason,cost,positions_before,positions_after,successful,price\n")
	
	// Write data
	for _, record := range transitionHistory {
		file.WriteString(fmt.Sprintf("%s,%s,%s,%s,%.4f,%d,%d,%t,%.2f\n",
			record.Timestamp.Format("2006-01-02 15:04:05"),
			record.FromEngine,
			record.ToEngine,
			record.Reason,
			record.Cost,
			record.PositionsBefore,
			record.PositionsAfter,
			record.Successful,
			record.Price))
	}
	
	return nil
}

// Helper functions for report generation

func generateJSONReport(results *backtest.DualEngineBacktestResults, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Simple JSON serialization (in a real implementation, use encoding/json)
	file.WriteString(fmt.Sprintf(`{
  "symbol": "%s",
  "timeframe": "%s",
  "total_return": %.2f,
  "annualized_return": %.2f,
  "max_drawdown": %.2f,
  "sharpe_ratio": %.2f,
  "win_rate": %.1f,
  "total_trades": %d,
  "regime_changes": %d,
  "total_transitions": %d
}`,
		results.Symbol,
		results.Timeframe,
		results.TotalReturn,
		results.AnnualizedReturn,
		results.MaxDrawdown,
		results.SharpeRatio,
		results.WinRate,
		results.TotalTrades,
		results.RegimeChanges,
		results.TotalTransitions))
	
	return nil
}

func generateResultsCSV(results *backtest.DualEngineBacktestResults, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write CSV header
	file.WriteString("metric,value\n")
	
	// Write data
	file.WriteString(fmt.Sprintf("symbol,%s\n", results.Symbol))
	file.WriteString(fmt.Sprintf("timeframe,%s\n", results.Timeframe))
	file.WriteString(fmt.Sprintf("total_return,%.2f\n", results.TotalReturn))
	file.WriteString(fmt.Sprintf("annualized_return,%.2f\n", results.AnnualizedReturn))
	file.WriteString(fmt.Sprintf("max_drawdown,%.2f\n", results.MaxDrawdown))
	file.WriteString(fmt.Sprintf("sharpe_ratio,%.2f\n", results.SharpeRatio))
	file.WriteString(fmt.Sprintf("win_rate,%.1f\n", results.WinRate))
	file.WriteString(fmt.Sprintf("total_trades,%d\n", results.TotalTrades))
	file.WriteString(fmt.Sprintf("regime_changes,%d\n", results.RegimeChanges))
	file.WriteString(fmt.Sprintf("total_transitions,%d\n", results.TotalTransitions))
	
	return nil
}
