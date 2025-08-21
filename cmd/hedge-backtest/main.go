package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/bybit"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/joho/godotenv"
)

// Constants for default configuration values
const (
	DefaultInitialBalance = 1000.0
	DefaultCommission     = 0.0005 // 0.05%
	DefaultWindowSize     = 100
	DefaultBaseAmount     = 100.0
	
	// Dual Position Strategy defaults
	DefaultHedgeRatio           = 0.5  // Equal long/short positions
	DefaultStopLossPct          = 0.05 // 5% stop loss
	DefaultTakeProfitPct        = 0.03 // 3% take profit
	DefaultTrailingStopPct      = 0.02 // 2% trailing stop
	DefaultMaxDrawdownPct       = 0.10 // 10% max drawdown
	DefaultVolatilityThreshold  = 0.015 // 1.5% minimum volatility
	DefaultTimeBetweenEntries   = 15   // 15 minutes between entries
	
	// Default minimum order quantity (will be overridden by exchange data)
	DefaultMinOrderQty          = 0.01 // Fallback minimum order quantity
	
	// File and directory constants
	DefaultDataRoot             = "data"
	DefaultExchange             = "bybit"
	
	// Advanced combo indicator parameters
	DefaultHullMAPeriod   = 20
	DefaultMFIPeriod      = 14
	DefaultMFIOversold    = 20
	DefaultMFIOverbought  = 80
	DefaultKeltnerPeriod  = 20
	DefaultKeltnerMultiplier = 2.0
	DefaultWaveTrendN1    = 10
	DefaultWaveTrendN2    = 21
	DefaultWaveTrendOverbought = 60
	DefaultWaveTrendOversold = -60
)

// HedgeConfig holds configuration for hedge backtests
type HedgeConfig struct {
	Symbol         string  `json:"symbol"`
	DataFile       string  `json:"data_file"`
	InitialBalance float64 `json:"initial_balance"`
	Commission     float64 `json:"commission"`
	WindowSize     int     `json:"window_size"`
	
	// Strategy parameters
	BaseAmount           float64 `json:"base_amount"`
	HedgeRatio           float64 `json:"hedge_ratio"`
	StopLossPct          float64 `json:"stop_loss_pct"`
	TakeProfitPct        float64 `json:"take_profit_pct"`
	TrailingStopPct      float64 `json:"trailing_stop_pct"`
	MaxDrawdownPct       float64 `json:"max_drawdown_pct"`
	VolatilityThreshold  float64 `json:"volatility_threshold"`
	TimeBetweenEntries   int     `json:"time_between_entries"`
	
	// Indicator parameters (using advanced combo)
	HullMAPeriod         int     `json:"hull_ma_period"`
	MFIPeriod           int     `json:"mfi_period"`
	MFIOversold         float64 `json:"mfi_oversold"`
	MFIOverbought       float64 `json:"mfi_overbought"`
	KeltnerPeriod       int     `json:"keltner_period"`
	KeltnerMultiplier   float64 `json:"keltner_multiplier"`
	WaveTrendN1         int     `json:"wavetrend_n1"`
	WaveTrendN2         int     `json:"wavetrend_n2"`
	WaveTrendOverbought float64 `json:"wavetrend_overbought"`
	WaveTrendOversold   float64 `json:"wavetrend_oversold"`
	
	// Exchange constraints
	MinOrderQty         float64 `json:"min_order_qty"`
}

func main() {
	var (
		symbol         = flag.String("symbol", "BTCUSDT", "Trading symbol")
		dataFile       = flag.String("data", "", "Path to historical data file")
		initialBalance = flag.Float64("balance", DefaultInitialBalance, "Initial balance")
		commission     = flag.Float64("commission", DefaultCommission, "Trading commission (0.001 = 0.1%)")
		windowSize     = flag.Int("window", DefaultWindowSize, "Data window size for analysis")
		baseAmount     = flag.Float64("base-amount", DefaultBaseAmount, "Base hedge amount")
		
		// Hedge strategy parameters
		hedgeRatio      = flag.Float64("hedge-ratio", DefaultHedgeRatio, "Hedge ratio (0.0=only long, 0.5=equal, 1.0=only short)")
		stopLossPct     = flag.Float64("stop-loss", DefaultStopLossPct, "Stop loss percentage")
		takeProfitPct   = flag.Float64("take-profit", DefaultTakeProfitPct, "Take profit percentage")
		trailingStopPct = flag.Float64("trailing-stop", DefaultTrailingStopPct, "Trailing stop percentage")
		maxDrawdownPct  = flag.Float64("max-drawdown", DefaultMaxDrawdownPct, "Maximum drawdown percentage per position")
		volatilityThreshold = flag.Float64("volatility-threshold", DefaultVolatilityThreshold, "Minimum volatility threshold")
		timeBetweenEntries = flag.Int("time-between-entries", DefaultTimeBetweenEntries, "Minutes between position entries")
		
		// Advanced indicator parameters
		hullMAPeriod    = flag.Int("hull-ma-period", DefaultHullMAPeriod, "Hull MA period")
		mfiPeriod       = flag.Int("mfi-period", DefaultMFIPeriod, "MFI period")
		mfiOversold     = flag.Float64("mfi-oversold", DefaultMFIOversold, "MFI oversold level")
		mfiOverbought   = flag.Float64("mfi-overbought", DefaultMFIOverbought, "MFI overbought level")
		keltnerPeriod   = flag.Int("keltner-period", DefaultKeltnerPeriod, "Keltner Channels period")
		keltnerMultiplier = flag.Float64("keltner-multiplier", DefaultKeltnerMultiplier, "Keltner Channels multiplier")
		waveTrendN1     = flag.Int("wavetrend-n1", DefaultWaveTrendN1, "WaveTrend N1 parameter")
		waveTrendN2     = flag.Int("wavetrend-n2", DefaultWaveTrendN2, "WaveTrend N2 parameter")
		waveTrendOverbought = flag.Float64("wavetrend-overbought", DefaultWaveTrendOverbought, "WaveTrend overbought level")
		waveTrendOversold = flag.Float64("wavetrend-oversold", DefaultWaveTrendOversold, "WaveTrend oversold level")
		
		// Optimization parameters
		optimize        = flag.Bool("optimize", false, "Run parameter optimization using genetic algorithm")
		optimizeObjective = flag.String("objective", "balanced", "Optimization objective: hedge-efficiency, return, sharpe, volatility-capture, balanced")
		generations     = flag.Int("generations", 25, "Number of generations for optimization")
		populationSize  = flag.Int("population", 40, "Population size for optimization")
		
		// File and exchange parameters
		interval       = flag.String("interval", "5m", "Data interval (e.g., 5m, 1h, 4h, 1d)")
		dataRoot       = flag.String("data-root", DefaultDataRoot, "Root folder containing exchange data")
		exchange       = flag.String("exchange", DefaultExchange, "Exchange to use (bybit, binance)")
		envFile        = flag.String("env", ".env", "Environment file path for API credentials")
	)
	
	flag.Parse()

	// Create configuration
	cfg := &HedgeConfig{
		Symbol:         *symbol,
		DataFile:       *dataFile,
		InitialBalance: *initialBalance,
		Commission:     *commission,
		WindowSize:     *windowSize,
		BaseAmount:     *baseAmount,
		HedgeRatio:     *hedgeRatio,
		StopLossPct:    *stopLossPct,
		TakeProfitPct:  *takeProfitPct,
		TrailingStopPct: *trailingStopPct,
		MaxDrawdownPct: *maxDrawdownPct,
		VolatilityThreshold: *volatilityThreshold,
		TimeBetweenEntries: *timeBetweenEntries,
		HullMAPeriod:    *hullMAPeriod,
		MFIPeriod:       *mfiPeriod,
		MFIOversold:     *mfiOversold,
		MFIOverbought:   *mfiOverbought,
		KeltnerPeriod:   *keltnerPeriod,
		KeltnerMultiplier: *keltnerMultiplier,
		WaveTrendN1:     *waveTrendN1,
		WaveTrendN2:     *waveTrendN2,
		WaveTrendOverbought: *waveTrendOverbought,
		WaveTrendOversold: *waveTrendOversold,
		MinOrderQty: DefaultMinOrderQty,
	}

	// Load environment variables from .env file
	if err := loadEnvFile(*envFile); err != nil {
		log.Printf("Warning: Could not load .env file (%v), checking environment variables...", err)
	}

	// Resolve data file from symbol/interval if not explicitly provided
	if cfg.DataFile == "" {
		sym := strings.ToUpper(cfg.Symbol)
		cfg.DataFile = findDataFile(*dataRoot, *exchange, sym, *interval)
		
		// Check if the resolved file exists
		if _, err := os.Stat(cfg.DataFile); err != nil {
			log.Printf("⚠️  Data file not found: %s", cfg.DataFile)
			log.Printf("Expected structure: %s/%s/{linear,spot,inverse}/%s/%s/candles.csv", 
				*dataRoot, *exchange, sym, convertIntervalToMinutes(*interval))
			log.Printf("Generating sample data for testing...")
			cfg.DataFile = generateSampleDataFile()
		} else {
			log.Printf("📁 Using data file: %s", cfg.DataFile)
		}
	}

	// Fetch minimum order quantity from exchange to set realistic optimization bounds
	if err := fetchAndSetMinOrderQty(cfg); err != nil {
		log.Printf("Warning: Could not fetch minimum order quantity: %v", err)
		log.Printf("Using default minimum order quantity: %.6f", DefaultMinOrderQty)
		cfg.MinOrderQty = DefaultMinOrderQty
	} else {
		log.Printf("✅ Using minimum order quantity from exchange: %.6f %s", cfg.MinOrderQty, cfg.Symbol)
	}

	if *optimize {
		// Run optimization
		objectiveType := parseObjective(*optimizeObjective)
		optimizationResult := runHedgeOptimization(cfg, objectiveType, *generations, *populationSize)
		
		// Print optimization results
		optimizationResult.PrintOptimizationSummary()
	} else {
		// Run single hedge backtest
		results := runHedgeBacktest(cfg)
		
		// Print results
		results.PrintSummary()
	}
}

func runHedgeBacktest(cfg *HedgeConfig) *backtest.HedgeBacktestResults {
	log.Println("🚀 Starting Hedge Strategy Backtest")
	log.Printf("📊 Symbol: %s", cfg.Symbol)
	log.Printf("💰 Initial Balance: $%.2f", cfg.InitialBalance)
	log.Printf("🔄 Hedge Ratio: %.2f", cfg.HedgeRatio)
	log.Printf("🛡️ Stop Loss: %.2f%%, Take Profit: %.2f%%", cfg.StopLossPct*100, cfg.TakeProfitPct*100)
	log.Println("=" + strings.Repeat("=", 50))

	// Load historical data
	data, err := loadHistoricalData(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}

	if len(data) == 0 {
		log.Fatalf("No valid data found")
	}

	log.Printf("📈 Loaded %d data points", len(data))

	// Create dual position strategy
	strategy := createHedgeStrategy(cfg)

	// Create hedge backtest engine
	engine := backtest.NewHedgeBacktestEngine(cfg.InitialBalance, cfg.Commission, strategy)

	// Run backtest
	start := time.Now()
	results := engine.Run(data, cfg.WindowSize)
	
	log.Printf("⏱️ Backtest completed in %v", time.Since(start))
	
	return results
}

func createHedgeStrategy(cfg *HedgeConfig) strategy.Strategy {
	// Create dual position strategy
	hedge := strategy.NewDualPositionStrategy(cfg.BaseAmount)
	
	// Configure hedge parameters
	hedge.SetHedgeRatio(cfg.HedgeRatio)
	hedge.SetRiskParams(cfg.StopLossPct, cfg.TakeProfitPct, cfg.TrailingStopPct, cfg.MaxDrawdownPct)
	hedge.SetVolatilityThreshold(cfg.VolatilityThreshold)
	hedge.SetTimeBetweenEntries(time.Duration(cfg.TimeBetweenEntries) * time.Minute)

	// Add advanced combo indicators
	log.Println("🎯 Adding Advanced Combo Indicators:")
	
	// Hull Moving Average
	hullMA := indicators.NewHullMA(cfg.HullMAPeriod)
	hedge.AddIndicator(hullMA)
	log.Printf("  ✅ Hull MA (period: %d)", cfg.HullMAPeriod)

	// Money Flow Index
	mfi := indicators.NewMFIWithPeriod(cfg.MFIPeriod)
	mfi.SetOversold(cfg.MFIOversold)
	mfi.SetOverbought(cfg.MFIOverbought)
	hedge.AddIndicator(mfi)
	log.Printf("  ✅ MFI (period: %d, oversold: %.1f, overbought: %.1f)", 
		cfg.MFIPeriod, cfg.MFIOversold, cfg.MFIOverbought)

	// Keltner Channels
	keltner := indicators.NewKeltnerChannelsCustom(cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
	hedge.AddIndicator(keltner)
	log.Printf("  ✅ Keltner Channels (period: %d, multiplier: %.1f)", 
		cfg.KeltnerPeriod, cfg.KeltnerMultiplier)

	// WaveTrend
	wavetrend := indicators.NewWaveTrendCustom(cfg.WaveTrendN1, cfg.WaveTrendN2)
	wavetrend.SetOverbought(cfg.WaveTrendOverbought)
	wavetrend.SetOversold(cfg.WaveTrendOversold)
	hedge.AddIndicator(wavetrend)
	log.Printf("  ✅ WaveTrend (N1: %d, N2: %d, overbought: %.1f, oversold: %.1f)", 
		cfg.WaveTrendN1, cfg.WaveTrendN2, cfg.WaveTrendOverbought, cfg.WaveTrendOversold)

	return hedge
}

func loadHistoricalData(filename string) ([]types.OHLCV, error) {
	if filename == "" {
		return generateSampleData(), nil
	}
	
	file, err := os.Open(filename)
	if err != nil {
		// If file doesn't exist, generate sample data
		if os.IsNotExist(err) {
			log.Printf("⚠️  Historical data file not found: %s, generating sample data...", filename)
			return generateSampleData(), nil
		}
		return nil, err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}
	
	var data []types.OHLCV
	
	lineNum := 1 // Start from 1 since we already read header
	for {
		record, err := reader.Read()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			return nil, fmt.Errorf("error reading CSV at line %d: %v", lineNum, err)
		}
		lineNum++
		
		// Expected format: timestamp,open,high,low,close,volume
		if len(record) < 6 {
			log.Printf("Warning: Insufficient columns at line %d, skipping", lineNum)
			continue
		}
		
		// Parse timestamp with proper error handling
		timestamp, err := time.Parse("2006-01-02 15:04:05", record[0])
		if err != nil {
			log.Printf("Warning: Invalid timestamp '%s' at line %d, skipping: %v", record[0], lineNum, err)
			continue
		}
		
		// Parse price data with proper error handling
		open, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			log.Printf("Warning: Invalid open price '%s' at line %d, skipping: %v", record[1], lineNum, err)
			continue
		}
		
		high, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			log.Printf("Warning: Invalid high price '%s' at line %d, skipping: %v", record[2], lineNum, err)
			continue
		}
		
		low, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			log.Printf("Warning: Invalid low price '%s' at line %d, skipping: %v", record[3], lineNum, err)
			continue
		}
		
		close, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			log.Printf("Warning: Invalid close price '%s' at line %d, skipping: %v", record[4], lineNum, err)
			continue
		}
		
		volume, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			log.Printf("Warning: Invalid volume '%s' at line %d, skipping: %v", record[5], lineNum, err)
			continue
		}
		
		// Validate price data
		if open <= 0 || high <= 0 || low <= 0 || close <= 0 {
			log.Printf("Warning: Invalid price data at line %d (prices must be > 0), skipping", lineNum)
			continue
		}
		
		if low > high {
			log.Printf("Warning: Low price %.8f > High price %.8f at line %d, skipping", low, high, lineNum)
			continue
		}
		
		if open > high || open < low || close > high || close < low {
			log.Printf("Warning: Open/Close prices outside High/Low range at line %d, skipping", lineNum)
			continue
		}
		
		data = append(data, types.OHLCV{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}
	
	if len(data) == 0 {
		log.Printf("⚠️  No valid data loaded from %s, generating sample data...", filename)
		return generateSampleData(), nil
	}
	
	log.Printf("✅ Loaded %d candles from %s", len(data), filename)
	return data, nil
}

func generateSampleData() []types.OHLCV {
	// Generate realistic sample OHLCV data for testing
	data := make([]types.OHLCV, 1000)
	basePrice := 45000.0 // Starting price around $45,000
	baseTime := time.Now().Add(-time.Duration(len(data)) * 5 * time.Minute)
	
	for i := range data {
		// Simulate price movement with some volatility
		priceChange := (float64(i%20) - 10) * 100 // +/- $1000 range
		volatility := float64(i%5) * 50 // Additional volatility
		
		price := basePrice + priceChange + volatility
		high := price + float64(i%3)*20
		low := price - float64(i%3)*20
		
		data[i] = types.OHLCV{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - float64(i%2)*10,
			High:      high,
			Low:       low,
			Close:     price,
			Volume:    1000000 + float64(i%100)*10000, // Sample volume
		}
	}
	
	return data
}

func generateSampleDataFile() string {
	return "" // Returns empty string to trigger sample data generation
}

func parseObjective(objectiveStr string) backtest.HedgeOptimizationObjective {
	switch strings.ToLower(objectiveStr) {
	case "hedge-efficiency":
		return backtest.OptimizeHedgeEfficiency
	case "return":
		return backtest.OptimizeReturn
	case "sharpe":
		return backtest.OptimizeSharpe
	case "volatility-capture":
		return backtest.OptimizeVolatilityCapture
	case "balanced":
		return backtest.OptimizeBalanced
	default:
		log.Printf("Unknown objective '%s', using balanced", objectiveStr)
		return backtest.OptimizeBalanced
	}
}

func runHedgeOptimization(cfg *HedgeConfig, objective backtest.HedgeOptimizationObjective, generations, populationSize int) *backtest.HedgeOptimizationResult {
	log.Println("🧬 Starting Hedge Strategy Optimization")
	log.Printf("📊 Symbol: %s", cfg.Symbol)
	log.Printf("🎯 Objective: %s", getObjectiveName(objective))
	log.Printf("👥 Population: %d, Generations: %d", populationSize, generations)
	log.Println("=" + strings.Repeat("=", 50))

	// Load historical data
	data, err := loadHistoricalData(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}

	if len(data) == 0 {
		log.Fatalf("No valid data found")
	}

	log.Printf("📈 Loaded %d data points for optimization", len(data))

	// Create optimizer with minimum order quantity constraint
	optimizer := backtest.NewHedgeOptimizer(data, cfg.WindowSize, cfg.InitialBalance, cfg.Commission)
	
	// Set minimum order quantity for realistic bounds
	optimizer.SetMinOrderQuantity(cfg.MinOrderQty, cfg.Symbol)
	
	// Set custom GA parameters if provided
	if populationSize > 0 {
		optimizer.SetPopulationSize(populationSize)
	}
	if generations > 0 {
		optimizer.SetGenerations(generations)
	}

	// Run optimization
	return optimizer.Optimize(objective)
}

func getObjectiveName(objective backtest.HedgeOptimizationObjective) string {
	switch objective {
	case backtest.OptimizeHedgeEfficiency:
		return "Hedge Efficiency"
	case backtest.OptimizeReturn:
		return "Total Return"
	case backtest.OptimizeSharpe:
		return "Sharpe Ratio"
	case backtest.OptimizeVolatilityCapture:
		return "Volatility Capture"
	case backtest.OptimizeBalanced:
		return "Balanced"
	default:
		return "Unknown"
	}
}

func fetchAndSetMinOrderQty(cfg *HedgeConfig) error {
	// Create Bybit client to fetch instrument info
	bybitConfig := bybit.Config{
		APIKey:    os.Getenv("BYBIT_API_KEY"),
		APISecret: os.Getenv("BYBIT_API_SECRET"),
		Demo:      true, // Use demo mode for fetching instrument info (safer)
	}

	// Skip if no API credentials (use default)
	if bybitConfig.APIKey == "" || bybitConfig.APISecret == "" {
		log.Printf("No Bybit API credentials found, using default min_order_qty: %.6f", cfg.MinOrderQty)
		return nil
	}

	bybitClient := bybit.NewClient(bybitConfig)
	
	// Determine category - for hedge backtest we'll assume linear derivatives
	category := "linear" // Most common for hedging strategies
	if strings.Contains(strings.ToLower(cfg.Symbol), "usdt") || strings.Contains(strings.ToLower(cfg.Symbol), "usdc") {
		// For USDT/USDC pairs, check if spot or linear
		category = "linear" // Default to linear for derivatives trading
	}

	// Fetch instrument constraints
	ctx := context.Background()
	minQty, _, _, err := bybitClient.GetInstrumentManager().GetQuantityConstraints(ctx, category, cfg.Symbol)
	if err != nil {
		return fmt.Errorf("failed to fetch instrument constraints for %s: %w", cfg.Symbol, err)
	}

	// Update config with fetched minimum order quantity
	cfg.MinOrderQty = minQty
	log.Printf("✅ Fetched minimum order quantity for %s: %.6f %s", cfg.Symbol, minQty, cfg.Symbol)
	
	return nil
}

func loadEnvFile(envFile string) error {
	// Load .env file if it exists
	if _, err := os.Stat(envFile); err == nil {
		return godotenv.Load(envFile)
	}
	return fmt.Errorf("env file %s not found", envFile)
}

// convertIntervalToMinutes converts interval strings like "5m", "1h", "4h" to minute numbers
func convertIntervalToMinutes(interval string) string {
	// If it's already just a number, return as-is
	if _, err := strconv.Atoi(interval); err == nil {
		return interval
	}
	
	// Parse interval string
	interval = strings.ToLower(strings.TrimSpace(interval))
	
	// Extract number and unit
	if len(interval) < 2 {
		return interval // Invalid format, return as-is
	}
	
	numStr := interval[:len(interval)-1]
	unit := interval[len(interval)-1:]
	
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return interval // Invalid number, return as-is
	}
	
	// Convert to minutes
	switch unit {
	case "m":
		return strconv.Itoa(num)
	case "h":
		return strconv.Itoa(num * 60)
	case "d":
		return strconv.Itoa(num * 24 * 60)
	case "w":
		return strconv.Itoa(num * 7 * 24 * 60)
	default:
		return interval // Unknown unit, return as-is
	}
}

// findDataFile attempts to locate data files for a specific exchange
// Structure: data/{exchange}/{category}/{symbol}/{interval}/candles.csv
func findDataFile(dataRoot, exchange, symbol, interval string) string {
	symbol = strings.ToUpper(symbol)
	
	// Convert interval to minutes (5m -> 5, 1h -> 60, etc.)
	intervalMinutes := convertIntervalToMinutes(interval)
	
	// Define categories by exchange
	var categories []string
	switch strings.ToLower(exchange) {
	case "bybit":
		categories = []string{"linear", "spot", "inverse"} // Prefer linear for hedge trading
	case "binance":
		categories = []string{"futures", "spot"}
	default:
		categories = []string{"linear", "spot", "futures", "inverse"}
	}
	
	// Check each category for the exchange
	for _, category := range categories {
		path := filepath.Join(dataRoot, exchange, category, symbol, intervalMinutes, "candles.csv")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	
	// If no existing file found, return the preferred structure (first category for the exchange)
	return filepath.Join(dataRoot, exchange, categories[0], symbol, intervalMinutes, "candles.csv")
}
