package main

/*
Interactive Examples for Enhanced DCA Bot
=========================================

This package provides interactive examples demonstrating the bot's capabilities
using real Bitcoin data. It includes:

1. Data Analysis Examples
2. Strategy Testing Examples
3. Backtesting Examples
4. Interactive Trading Simulations
5. Performance Visualization

All examples use simulated trading for educational purposes.
*/

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

func main() {
	fmt.Println("🚀 Enhanced DCA Bot - Interactive Examples")
	fmt.Println("==========================================")
	fmt.Println()

	for {
		fmt.Println("Available Examples:")
		fmt.Println("1. 📊 Data Analysis - Analyze BTC market data")
		fmt.Println("2. 🎯 Strategy Testing - Test different strategies")
		fmt.Println("3. 📈 Backtesting - Run historical backtests")
		fmt.Println("4. 🎮 Interactive Trading - Simulate live trading")
		fmt.Println("5. 📋 Performance Comparison - Compare strategies")
		fmt.Println("6. 🔧 Configuration Examples - Show config options")
		fmt.Println("7. 📄 Excel Data Loading - Test Excel file loading")
		fmt.Println("0. Exit")
		fmt.Println()

		choice := getUserChoice("Select an example (0-7): ", 0, 7)
		if choice == 0 {
			break
		}

		fmt.Println()
		switch choice {
		case 1:
			runDataAnalysisExample()
		case 2:
			runStrategyTestingExample()
		case 3:
			runBacktestingExample()
		case 4:
			runInteractiveTradingExample()
		case 5:
			runPerformanceComparisonExample()
		case 6:
			runConfigurationExamples()
		case 7:
			runExcelDataLoadingExample()
		}

		fmt.Println()
		fmt.Println("Press Enter to continue...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		fmt.Println("\n" + strings.Repeat("=", 50) + "\n")
	}

	fmt.Println("Thank you for using Enhanced DCA Bot Examples! 👋")
}

func runDataAnalysisExample() {
	fmt.Println("📊 === Data Analysis Example ===")
	fmt.Println()

	// Load BTC data
	data, err := loadBTCData()
	if err != nil {
		log.Printf("Failed to load BTC data: %v", err)
		return
	}

	fmt.Printf("📈 Loaded %d data points\n", len(data))
	if len(data) > 0 {
		first := data[0]
		last := data[len(data)-1]
		fmt.Printf("📅 Date Range: %s to %s\n",
			first.Timestamp.Format("2006-01-02"),
			last.Timestamp.Format("2006-01-02"))
		fmt.Printf("💰 Price Range: $%.2f - $%.2f\n",
			findMinPrice(data), findMaxPrice(data))
	}

	// Calculate basic statistics
	stats := calculateMarketStats(data)
	fmt.Println()
	fmt.Println("📊 Market Statistics:")
	fmt.Printf("  Average Price: $%.2f\n", stats.AveragePrice)
	fmt.Printf("  Price Volatility: %.2f%%\n", stats.Volatility*100)
	fmt.Printf("  Total Volume: %.2f BTC\n", stats.TotalVolume)
	fmt.Printf("  Average Volume: %.2f BTC/day\n", stats.AverageVolume)

	// Show recent price action
	fmt.Println()
	fmt.Println("📈 Recent Price Action (Last 10 periods):")
	for i := len(data) - 10; i < len(data); i++ {
		if i >= 0 {
			change := (data[i].Close - data[i-1].Close) / data[i-1].Close * 100
			trend := "➡️"
			if change > 0 {
				trend = "📈"
			} else if change < 0 {
				trend = "📉"
			}
			fmt.Printf("  %s %s: $%.2f (%+.2f%%)\n",
				data[i].Timestamp.Format("01-02 15:04"),
				trend, data[i].Close, change)
		}
	}
}

func runStrategyTestingExample() {
	fmt.Println("🎯 === Strategy Testing Example ===")
	fmt.Println()

	data, err := loadBTCData()
	if err != nil {
		log.Printf("Failed to load BTC data: %v", err)
		return
	}

	// Test different strategies
	strategies := []struct {
		name     string
		strategy strategy.Strategy
	}{
		{"Multi-Indicator Strategy", strategy.NewMultiIndicatorStrategy()},
		{"Enhanced DCA Strategy", strategy.NewEnhancedDCAStrategy(100.0)},
	}

	for _, s := range strategies {
		fmt.Printf("🧪 Testing %s:\n", s.name)

		// Test on recent data
		recentData := data[len(data)-100:]
		decision, err := s.strategy.ShouldExecuteTrade(recentData)
		if err != nil {
			fmt.Printf("  ❌ Error: %v\n", err)
			continue
		}

		fmt.Printf("  📊 Decision: %s\n", decision.Action)
		fmt.Printf("  🎯 Confidence: %.2f%%\n", decision.Confidence*100)
		fmt.Printf("  💪 Strength: %.2f%%\n", decision.Strength*100)
		fmt.Printf("  📝 Reason: %s\n", decision.Reason)
		fmt.Println()
	}
}

func runBacktestingExample() {
	fmt.Println("📈 === Backtesting Example ===")
	fmt.Println()

	data, err := loadBTCData()
	if err != nil {
		log.Printf("Failed to load BTC data: %v", err)
		return
	}

	// Get backtest parameters
	initialBalance := getUserFloat("Enter initial balance ($): ", 10000.0)
	commission := getUserFloat("Enter commission rate (0.001 = 0.1%): ", 0.001)
	windowSize := getUserInt("Enter analysis window size (50-200): ", 100)

	// Create backtest engine
	engine := backtest.NewBacktestEngine(
		initialBalance,
		commission,
		strategy.NewMultiIndicatorStrategy(),
		0, // tpPercent disabled by default in examples
	)

	fmt.Println()
	fmt.Println("🔄 Running backtest...")
	results := engine.Run(data, windowSize)

	fmt.Println()
	fmt.Println("📊 === Backtest Results ===")
	results.PrintSummary()

	// Show trade details
	if len(results.Trades) > 0 {
		fmt.Println()
		fmt.Println("📋 Recent Trades:")
		start := len(results.Trades) - 5
		if start < 0 {
			start = 0
		}
		for i := start; i < len(results.Trades); i++ {
			trade := results.Trades[i]
			fmt.Printf("  %s: Buy %.6f BTC @ $%.2f\n",
				trade.EntryTime.Format("01-02 15:04"),
				trade.Quantity,
				trade.EntryPrice)
		}
	}
}

func runInteractiveTradingExample() {
	fmt.Println("🎮 === Interactive Trading Simulation ===")
	fmt.Println()

	data, err := loadBTCData()
	if err != nil {
		log.Printf("Failed to load BTC data: %v", err)
		return
	}

	// Initialize simulation
	balance := getUserFloat("Enter starting balance ($): ", 10000.0)
	position := 0.0
	strategy := strategy.NewMultiIndicatorStrategy()

	fmt.Println()
	fmt.Println("🎯 Starting interactive trading simulation...")
	fmt.Println("Press Enter to advance to next period, 'q' to quit")

	// Start from recent data
	startIndex := len(data) - 50
	for i := startIndex; i < len(data); i++ {
		currentData := data[:i+1]
		currentPrice := data[i].Close

		fmt.Printf("\n📅 %s - Price: $%.2f\n",
			data[i].Timestamp.Format("2006-01-02 15:04"), currentPrice)
		fmt.Printf("💰 Balance: $%.2f | Position: %.6f BTC | Total Value: $%.2f\n",
			balance, position, balance+position*currentPrice)

		// Get strategy decision
		decision, err := strategy.ShouldExecuteTrade(currentData)
		if err != nil {
			fmt.Printf("❌ Strategy error: %v\n", err)
			continue
		}

		fmt.Printf("🎯 Strategy: %s (Confidence: %.1f%%)\n",
			decision.Action, decision.Confidence*100)

		if decision.Action == 1 && balance > 100 { // ActionBuy = 1
			amount := 100.0 // Fixed DCA amount
			quantity := amount / currentPrice
			balance -= amount
			position += quantity
			fmt.Printf("✅ Bought %.6f BTC for $%.2f\n", quantity, amount)
		}

		fmt.Print("Press Enter to continue, 'q' to quit: ")
		input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		if strings.TrimSpace(input) == "q" {
			break
		}
	}

	// Final summary
	finalPrice := data[len(data)-1].Close
	finalValue := balance + position*finalPrice
	totalReturn := (finalValue - getUserFloat("Enter initial balance ($): ", 10000.0)) / getUserFloat("Enter initial balance ($): ", 10000.0) * 100

	fmt.Println()
	fmt.Println("📊 === Simulation Summary ===")
	fmt.Printf("Final Balance: $%.2f\n", balance)
	fmt.Printf("Final Position: %.6f BTC\n", position)
	fmt.Printf("Total Value: $%.2f\n", finalValue)
	fmt.Printf("Total Return: %.2f%%\n", totalReturn)
}

func runPerformanceComparisonExample() {
	fmt.Println("📋 === Performance Comparison Example ===")
	fmt.Println()

	data, err := loadBTCData()
	if err != nil {
		log.Printf("Failed to load BTC data: %v", err)
		return
	}

	strategies := []struct {
		name     string
		strategy strategy.Strategy
	}{
		{"Multi-Indicator", strategy.NewMultiIndicatorStrategy()},
		{"Enhanced DCA", strategy.NewEnhancedDCAStrategy(100.0)},
	}

	initialBalance := 10000.0
	commission := 0.001

	fmt.Println("🔄 Running performance comparison...")
	fmt.Println()

	for _, s := range strategies {
		engine := backtest.NewBacktestEngine(initialBalance, commission, s.strategy, 0)
		results := engine.Run(data, 100)

		fmt.Printf("📊 %s Results:\n", s.name)
		fmt.Printf("  Total Return: %.2f%%\n", results.TotalReturn*100)
		fmt.Printf("  Max Drawdown: %.2f%%\n", results.MaxDrawdown*100)
		fmt.Printf("  Total Trades: %d\n", results.TotalTrades)
		fmt.Printf("  Final Balance: $%.2f\n", results.EndBalance)
		fmt.Println()
	}
}

func runConfigurationExamples() {
	fmt.Println("🔧 === Configuration Examples ===")
	fmt.Println()

	// Show different configuration examples
	configs := []struct {
		name   string
		config *config.Config
	}{
		{
			"Conservative DCA",
			&config.Config{
				Strategy: struct {
					Symbol        string
					BaseAmount    float64
					MaxMultiplier float64
					Interval      time.Duration
				}{
					Symbol:        "BTCUSDT",
					BaseAmount:    50.0,
					MaxMultiplier: 2.0,
					Interval:      24 * time.Hour,
				},
			},
		},
		{
			"Aggressive DCA",
			&config.Config{
				Strategy: struct {
					Symbol        string
					BaseAmount    float64
					MaxMultiplier float64
					Interval      time.Duration
				}{
					Symbol:        "BTCUSDT",
					BaseAmount:    100.0,
					MaxMultiplier: 3.0,
					Interval:      1 * time.Hour,
				},
			},
		},
		{
			"Balanced DCA",
			&config.Config{
				Strategy: struct {
					Symbol        string
					BaseAmount    float64
					MaxMultiplier float64
					Interval      time.Duration
				}{
					Symbol:        "BTCUSDT",
					BaseAmount:    75.0,
					MaxMultiplier: 2.5,
					Interval:      6 * time.Hour,
				},
			},
		},
	}

	for _, cfg := range configs {
		fmt.Printf("📋 %s Configuration:\n", cfg.name)
		fmt.Printf("  Symbol: %s\n", cfg.config.Strategy.Symbol)
		fmt.Printf("  Interval: %v\n", cfg.config.Strategy.Interval)
		fmt.Printf("  Base Amount: $%.2f\n", cfg.config.Strategy.BaseAmount)
		fmt.Printf("  Max Multiplier: %.1fx\n", cfg.config.Strategy.MaxMultiplier)
		fmt.Println()
	}
}

func runExcelDataLoadingExample() {
	fmt.Println("📄 === Excel Data Loading Example ===")
	fmt.Println()

	// Test Excel data loading
	loader := NewDataLoader("examples/data")

	data, err := loader.LoadBTCData()
	if err != nil {
		log.Printf("Failed to load data: %v", err)
		return
	}

	fmt.Printf("✅ Successfully loaded %d data points\n", len(data))

	// Validate data
	if err := loader.ValidateData(data); err != nil {
		log.Printf("Data validation failed: %v", err)
		return
	}

	// Get summary
	summary := loader.GetDataSummary(data)
	fmt.Printf("📊 Data Summary: %s\n", summary.String())

	// Show first few records
	fmt.Println("\n📈 First 5 records:")
	for i := 0; i < 5 && i < len(data); i++ {
		record := data[i]
		fmt.Printf("  %s: O=%.2f H=%.2f L=%.2f C=%.2f V=%.2f\n",
			record.Timestamp.Format("2006-01-02 15:04"),
			record.Open, record.High, record.Low, record.Close, record.Volume)
	}

	fmt.Println("\n🎉 Excel data loading test completed successfully!")
}

// Helper functions
func getUserChoice(prompt string, min, max int) int {
	for {
		fmt.Print(prompt)
		var input string
		fmt.Scanln(&input)

		choice, err := strconv.Atoi(input)
		if err == nil && choice >= min && choice <= max {
			return choice
		}
		fmt.Printf("Please enter a number between %d and %d\n", min, max)
	}
}

func getUserFloat(prompt string, defaultValue float64) float64 {
	fmt.Print(prompt)
	var input string
	fmt.Scanln(&input)

	if input == "" {
		return defaultValue
	}

	value, err := strconv.ParseFloat(input, 64)
	if err != nil {
		return defaultValue
	}
	return value
}

func getUserInt(prompt string, defaultValue int) int {
	fmt.Print(prompt)
	var input string
	fmt.Scanln(&input)

	if input == "" {
		return defaultValue
	}

	value, err := strconv.Atoi(input)
	if err != nil {
		return defaultValue
	}
	return value
}

// Data loading and analysis functions
func getDataPath() string {
	// 1. Check command-line flag
	dataFlag := flag.String("data", "", "Path to data directory (default: examples/data or $DATA_PATH)")
	flag.Parse()
	if *dataFlag != "" {
		return *dataFlag
	}
	// 2. Check environment variable
	if env := os.Getenv("DATA_PATH"); env != "" {
		return env
	}
	// 3. Default
	return "examples/data"
}

func loadBTCData() ([]types.OHLCV, error) {
	dataPath := getDataPath()
	log.Printf("📁 Using data path: %s", dataPath)
	// Create data loader
	loader := NewDataLoader(dataPath)

	// Load data from various sources
	data, err := loader.LoadBTCData()
	if err != nil {
		return nil, fmt.Errorf("failed to load BTC data: %w", err)
	}

	// Validate data
	if err := loader.ValidateData(data); err != nil {
		return nil, fmt.Errorf("data validation failed: %w", err)
	}

	// Print data summary
	summary := loader.GetDataSummary(data)
	log.Printf("📊 Data Summary: %s", summary.String())

	return data, nil
}

type MarketStats struct {
	AveragePrice  float64
	Volatility    float64
	TotalVolume   float64
	AverageVolume float64
}

func calculateMarketStats(data []types.OHLCV) MarketStats {
	if len(data) == 0 {
		return MarketStats{}
	}

	totalPrice := 0.0
	totalVolume := 0.0
	var returns []float64

	for i, candle := range data {
		totalPrice += candle.Close
		totalVolume += candle.Volume

		if i > 0 {
			ret := (candle.Close - data[i-1].Close) / data[i-1].Close
			returns = append(returns, ret)
		}
	}

	avgPrice := totalPrice / float64(len(data))
	avgVolume := totalVolume / float64(len(data))

	// Calculate volatility (standard deviation of returns)
	variance := 0.0
	for _, r := range returns {
		variance += r * r
	}
	volatility := math.Sqrt(variance / float64(len(returns)))

	return MarketStats{
		AveragePrice:  avgPrice,
		Volatility:    volatility,
		TotalVolume:   totalVolume,
		AverageVolume: avgVolume,
	}
}

func findMinPrice(data []types.OHLCV) float64 {
	if len(data) == 0 {
		return 0
	}
	min := data[0].Low
	for _, candle := range data {
		if candle.Low < min {
			min = candle.Low
		}
	}
	return min
}

func findMaxPrice(data []types.OHLCV) float64 {
	if len(data) == 0 {
		return 0
	}
	max := data[0].High
	for _, candle := range data {
		if candle.High > max {
			max = candle.High
		}
	}
	return max
}
