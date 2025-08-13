package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/xuri/excelize/v2"
	"math/rand"
)

// BacktestConfig holds all configuration for backtesting
type BacktestConfig struct {
	DataFile       string  `json:"data_file"`
	Symbol         string  `json:"symbol"`
	InitialBalance float64 `json:"initial_balance"`
	Commission     float64 `json:"commission"`
	WindowSize     int     `json:"window_size"`
	
	// DCA Strategy parameters
	BaseAmount     float64 `json:"base_amount"`
	MaxMultiplier  float64 `json:"max_multiplier"`
	PriceThreshold float64 `json:"price_threshold"` // Minimum price drop % for next DCA entry
	
	// Technical indicator parameters
	RSIPeriod      int     `json:"rsi_period"`
	RSIOversold    float64 `json:"rsi_oversold"`
	RSIOverbought  float64 `json:"rsi_overbought"`
	
	MACDFast       int     `json:"macd_fast"`
	MACDSlow       int     `json:"macd_slow"`
	MACDSignal     int     `json:"macd_signal"`
	
	BBPeriod       int     `json:"bb_period"`
	BBStdDev       float64 `json:"bb_std_dev"`
	
	SMAPeriod      int     `json:"sma_period"`
	
	// Indicator inclusion
	Indicators     []string `json:"indicators"`
	
	// Output settings
	OutputFormat   string  `json:"output_format"` // "console", "json", "csv"
	OutputFile     string  `json:"output_file"`
	Verbose        bool    `json:"verbose"`

	// Take-profit configuration
	TPPercent      float64 `json:"tp_percent"`
    Cycle         bool    `json:"cycle"`
}

func main() {
	var (
		configFile     = flag.String("config", "", "Path to configuration file")
		dataFile       = flag.String("data", "", "Path to historical data file (overrides -interval)")
		symbol         = flag.String("symbol", "BTCUSDT", "Trading symbol")
		intervalFlag  = flag.String("interval", "1h", "Data interval to use (e.g., 15m,1h,4h,1d)")
		initialBalance = flag.Float64("balance", 500, "Initial balance")
		commission     = flag.Float64("commission", 0.001, "Trading commission (0.001 = 0.1%)")
		windowSize     = flag.Int("window", 100, "Data window size for analysis")
		baseAmount     = flag.Float64("base-amount", 20, "Base DCA amount")
		maxMultiplier  = flag.Float64("max-multiplier", 3, "Maximum position multiplier")
		priceThreshold = flag.Float64("price-threshold", 0.02, "Minimum price drop % for next DCA entry (default: 2%)")
		optimize       = flag.Bool("optimize", false, "Run parameter optimization using genetic algorithm")
		allIntervals   = flag.Bool("all-intervals", false, "Scan data root for all intervals for the given symbol and run per-interval backtests/optimizations")
		dataRoot       = flag.String("data-root", "data/historical", "Root folder containing <SYMBOL>/<INTERVAL>/candles.csv")
		indicatorsFlag = flag.String("indicators", "", "Comma-separated indicators to include (rsi,macd,bb,sma). Empty = use config or default all")
		optiIndicators = flag.Bool("optimize-indicators", false, "When optimizing, also search over indicator combinations")
		periodStr      = flag.String("period", "", "Limit data to trailing window (e.g., 7d,30d,180d,365d or Nd)")
        exhaustiveFlag = flag.Bool("exhaustive", false, "Test all indicator combinations (2+ indicators) with smaller GA for faster results")
	)
	
	flag.Parse()
	
	// Load configuration - cycle is always enabled, console output only
	cfg := loadConfig(*configFile, *dataFile, *symbol, *initialBalance, *commission, 
		*windowSize, *baseAmount, *maxMultiplier, *priceThreshold, "console", "", false)

	// Cycle is always enabled (our main logic)
	cfg.Cycle = true
	// TP will be determined by optimization or use sensible default
	if !*optimize {
		cfg.TPPercent = 0.02 // Default 2% TP when not optimizing
	}

	// Capture and parse trailing period window
	if s := strings.TrimSpace(*periodStr); s != "" {
		if d, ok := parseTrailingPeriod(s); ok {
			selectedPeriod = d
			selectedPeriodRaw = s
		}
	}

	// If optimize is requested, also optimize indicators by default
	// but respect explicit user setting (-optimize-indicators=false)
	if *optimize {
		if !flagProvided("optimize-indicators") {
			*optiIndicators = true
		}
	}

	// Apply indicators from flag if provided
	if strings.TrimSpace(*indicatorsFlag) != "" {
		cfg.Indicators = parseIndicatorsList(*indicatorsFlag)
	}
	// Default to all indicators if none specified anywhere
	if len(cfg.Indicators) == 0 {
		cfg.Indicators = []string{"rsi", "macd", "bb", "sma"}
	}

	// Resolve data file from symbol/interval if not explicitly provided and not scanning all intervals
	if !*allIntervals && strings.TrimSpace(cfg.DataFile) == "" {
		sym := strings.ToUpper(cfg.Symbol)
		cfg.DataFile = filepath.Join(*dataRoot, sym, *intervalFlag, "candles.csv")
	}
	
	if *allIntervals {
		runAcrossIntervals(cfg, *dataRoot, *optimize, *optiIndicators)
		return
	}
	
	if *optimize {
		var bestResults *backtest.BacktestResults
		var bestConfig BacktestConfig
		
		if *exhaustiveFlag {
			bestResults, bestConfig = optimizeExhaustive(cfg, *optiIndicators)
			fmt.Println("\n\n🏆 EXHAUSTIVE OPTIMIZATION RESULTS:")
		} else {
			bestResults, bestConfig = optimizeForInterval(cfg, *optiIndicators)
			fmt.Println("\n\n🏆 OPTIMIZATION RESULTS:")
		}
		
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Best Parameters:\n")
		fmt.Printf("  Indicators:       %s\n", strings.Join(bestConfig.Indicators, ","))
		fmt.Printf("  Base Amount:      $%.2f\n", bestConfig.BaseAmount)
		fmt.Printf("  Max Multiplier:   %.2f\n", bestConfig.MaxMultiplier)
		fmt.Printf("  Price Threshold:  %.2f%%\n", bestConfig.PriceThreshold*100)
		fmt.Printf("  TP Percent:       %.3f%%\n", bestConfig.TPPercent*100)
		if containsIndicator(bestConfig.Indicators, "rsi") {
			fmt.Printf("  RSI Period:     %d\n", bestConfig.RSIPeriod)
			fmt.Printf("  RSI Oversold:   %.0f\n", bestConfig.RSIOversold)
		}
		if containsIndicator(bestConfig.Indicators, "macd") {
			fmt.Printf("  MACD: fast=%d slow=%d signal=%d\n", bestConfig.MACDFast, bestConfig.MACDSlow, bestConfig.MACDSignal)
		}
		if containsIndicator(bestConfig.Indicators, "bb") {
			fmt.Printf("  BB: period=%d std=%.2f\n", bestConfig.BBPeriod, bestConfig.BBStdDev)
		}
		if containsIndicator(bestConfig.Indicators, "sma") {
			fmt.Printf("  SMA Period:     %d\n", bestConfig.SMAPeriod)
		}
		fmt.Println("\nBest Config (JSON):")
		fmt.Println("Copy this configuration to reuse these optimized settings:")
		fmt.Println(strings.Repeat("-", 50))
		printBestConfigJSON(bestConfig)
		fmt.Println(strings.Repeat("-", 50))
		// Determine interval string for usage example
		intervalStr := filepath.Base(filepath.Dir(bestConfig.DataFile))
		if intervalStr == "" { intervalStr = filepath.Base(filepath.Dir(cfg.DataFile)) }
		if intervalStr == "" { intervalStr = "unknown" }
		fmt.Printf("Usage: go run cmd/backtest/main.go -config best.json\n")
		fmt.Printf("   or: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
			cfg.Symbol, intervalStr, bestConfig.BaseAmount, bestConfig.MaxMultiplier, bestConfig.PriceThreshold)
		
		// Always save to standard paths in results folder
		stdDir := defaultOutputDir(cfg.Symbol, intervalStr)
		stdBestPath := filepath.Join(stdDir, "best.json")
		stdTradesPath := filepath.Join(stdDir, "trades.xlsx")
		
		// Write standard outputs
		if err := writeBestConfigJSON(bestConfig, stdBestPath); err != nil {
			fmt.Printf("Failed to write best config: %v\n", err)
		} else {
			fmt.Printf("Saved best config to: %s\n", stdBestPath)
		}
		if err := writeTradesCSV(bestResults, stdTradesPath); err != nil {
			fmt.Printf("Failed to write trades file: %v\n", err)
		} else {
			fmt.Printf("Saved trades to: %s\n", stdTradesPath)
		}
		
		fmt.Println("\nBest Results:")
		outputConsole(bestResults, false)
		return
	}
	
	// Run single backtest
	results := runBacktest(cfg)
	
	// Output results to console
	outputConsole(results, false)
	
	// Always save trades to standard path
	intervalStr := guessIntervalFromPath(cfg.DataFile)
	if intervalStr == "" { intervalStr = "unknown" }
	stdDir := defaultOutputDir(cfg.Symbol, intervalStr)
	stdTradesPath := filepath.Join(stdDir, "trades.xlsx")
	
	if err := writeTradesCSV(results, stdTradesPath); err != nil {
		fmt.Printf("Failed to write trades file: %v\n", err)
	} else {
		fmt.Printf("Saved trades to: %s\n", stdTradesPath)
	}
}

func loadConfig(configFile, dataFile, symbol string, balance, commission float64,
	windowSize int, baseAmount, maxMultiplier float64, priceThreshold float64, outputFormat, outputFile string, verbose bool) *BacktestConfig {
	
	cfg := &BacktestConfig{
		DataFile:       dataFile,
		Symbol:         symbol,
		InitialBalance: balance,
		Commission:     commission,
		WindowSize:     windowSize,
		BaseAmount:     baseAmount,
		MaxMultiplier:  maxMultiplier,
		PriceThreshold: priceThreshold, // Initialize PriceThreshold
		RSIPeriod:      14,
		RSIOversold:    30,
		RSIOverbought:  70,
		MACDFast:       12,
		MACDSlow:       26,
		MACDSignal:     9,
		BBPeriod:       20,
		BBStdDev:       2,
		SMAPeriod:      50,
		Indicators:     nil,
		OutputFormat:   outputFormat,
		OutputFile:     outputFile,
		Verbose:        verbose,
		TPPercent:     0, // Default to 0 TP
	}
	
	// Load from config file if provided
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			log.Printf("Warning: Could not read config file: %v", err)
		} else {
			if err := json.Unmarshal(data, cfg); err != nil {
				log.Printf("Warning: Could not parse config file: %v", err)
			}
		}
	}
	
	return cfg
}

func runAcrossIntervals(cfg *BacktestConfig, dataRoot string, optimize bool, optimizeIndicators bool) {
	sym := strings.ToUpper(cfg.Symbol)
	symDir := filepath.Join(dataRoot, sym)
	entries, err := os.ReadDir(symDir)
	if err != nil {
		log.Fatalf("Failed to read symbol directory %s: %v", symDir, err)
	}

	type intervalResult struct {
		Interval       string
		Results        *backtest.BacktestResults
		OptimizedCfg   BacktestConfig
	}

	var resultsByInterval []intervalResult

	for _, e := range entries {
		if !e.IsDir() { continue }
		interval := e.Name()
		candlesPath := filepath.Join(symDir, interval, "candles.csv")
		if _, err := os.Stat(candlesPath); err != nil { continue }

		cfgCopy := *cfg
		cfgCopy.DataFile = candlesPath

		var res *backtest.BacktestResults
		var bestCfg BacktestConfig
		if optimize {
			// propagate cycle preference
			cfgCopy.Cycle = cfg.Cycle
			res, bestCfg = optimizeForInterval(&cfgCopy, optimizeIndicators)
		} else {
			res = runBacktest(&cfgCopy)
			bestCfg = cfgCopy
		}

		resultsByInterval = append(resultsByInterval, intervalResult{
			Interval:     interval,
			Results:      res,
			OptimizedCfg: bestCfg,
		})
	}

	if len(resultsByInterval) == 0 {
		log.Fatalf("No interval data found under %s", symDir)
	}

	// Find best by TotalReturn
	bestIdx := 0
	for i := 1; i < len(resultsByInterval); i++ {
		if resultsByInterval[i].Results.TotalReturn > resultsByInterval[bestIdx].Results.TotalReturn {
			bestIdx = i
		}
	}

	fmt.Println("\n================ Interval Comparison ================")
	fmt.Printf("Symbol: %s\n", sym)
	fmt.Println("Interval | Return% | Trades | Base$ | MaxMult | TP% | Threshold% | RSI(p/ov) | Indicators")
	for _, r := range resultsByInterval {
		c := r.OptimizedCfg
		rsiInfo := "-/-"
		if containsIndicator(c.Indicators, "rsi") {
			rsiInfo = fmt.Sprintf("%d/%.0f", c.RSIPeriod, c.RSIOversold)
		}
		fmt.Printf("%-8s | %7.2f | %6d | %5.0f | %7.2f | %5.2f | %8.2f | %8s | %s\n",
			r.Interval,
			r.Results.TotalReturn*100,
			r.Results.TotalTrades,
			c.BaseAmount,
			c.MaxMultiplier,
			c.TPPercent*100,
			c.PriceThreshold*100,
			rsiInfo,
			strings.Join(c.Indicators, ","),
		)
	}
	best := resultsByInterval[bestIdx]
	fmt.Printf("\nBest interval: %s (Return %.2f%%)\n", best.Interval, best.Results.TotalReturn*100)
	fmt.Printf("Best settings -> Indicators: %s | Base: $%.0f, MaxMult: %.2f, TP: %.2f%%",
		strings.Join(best.OptimizedCfg.Indicators, ","),
		best.OptimizedCfg.BaseAmount,
		best.OptimizedCfg.MaxMultiplier,
		best.OptimizedCfg.TPPercent*100,
	)
	if containsIndicator(best.OptimizedCfg.Indicators, "rsi") {
		fmt.Printf(", RSI: %d/%.0f", best.OptimizedCfg.RSIPeriod, best.OptimizedCfg.RSIOversold)
	}
	if containsIndicator(best.OptimizedCfg.Indicators, "macd") {
		fmt.Printf(", MACD: %d/%d/%d", best.OptimizedCfg.MACDFast, best.OptimizedCfg.MACDSlow, best.OptimizedCfg.MACDSignal)
	}
	if containsIndicator(best.OptimizedCfg.Indicators, "bb") {
		fmt.Printf(", BB: p=%d sd=%.2f", best.OptimizedCfg.BBPeriod, best.OptimizedCfg.BBStdDev)
	}
	if containsIndicator(best.OptimizedCfg.Indicators, "sma") {
		fmt.Printf(", SMA: %d", best.OptimizedCfg.SMAPeriod)
	}
	fmt.Printf("\n")

	fmt.Println("Best Config (JSON):")
	fmt.Println("Copy this configuration to reuse these optimized settings:")
	fmt.Println(strings.Repeat("-", 50))
	printBestConfigJSON(best.OptimizedCfg)
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Usage: go run cmd/backtest/main.go -config best.json\n")
	fmt.Printf("   or: go run cmd/backtest/main.go -symbol %s -interval %s -base-amount %.0f -max-multiplier %.1f -price-threshold %.3f\n",
		cfg.Symbol, best.Interval, best.OptimizedCfg.BaseAmount, best.OptimizedCfg.MaxMultiplier, best.OptimizedCfg.PriceThreshold)

	// Optionally print detailed results for best interval
	fmt.Println("\nBest interval detailed results:")
	outputConsole(best.Results, false)
	
	// Write standard outputs under results/<SYMBOL>_<INTERVAL>/
	stdDir := defaultOutputDir(cfg.Symbol, best.Interval)
	stdBestPath := filepath.Join(stdDir, "best.json")
	stdTradesPath := filepath.Join(stdDir, "trades.xlsx")
	if err := writeBestConfigJSON(best.OptimizedCfg, stdBestPath); err != nil {
		fmt.Printf("Failed to write best config: %v\n", err)
	} else {
		fmt.Printf("Saved best config to: %s\n", stdBestPath)
	}
	if err := writeTradesCSV(best.Results, stdTradesPath); err != nil {
		fmt.Printf("Failed to write trades file: %v\n", err)
	} else {
		fmt.Printf("Saved trades to: %s\n", stdTradesPath)
	}
}

func runBacktest(cfg *BacktestConfig) *backtest.BacktestResults {
	log.Println("🚀 Starting DCA Bot Backtest")
	log.Printf("📊 Symbol: %s", cfg.Symbol)
	log.Printf("💰 Initial Balance: $%.2f", cfg.InitialBalance)
	log.Printf("📈 Base DCA Amount: $%.2f", cfg.BaseAmount)
	log.Printf("🔄 Max Multiplier: %.2f", cfg.MaxMultiplier)
	log.Println("=" + strings.Repeat("=", 40))
	
	// Load historical data
	data, err := loadHistoricalData(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}
	
	// Apply trailing period filter if set
	if selectedPeriod > 0 {
		data = filterDataByPeriod(data, selectedPeriod)
	}
	
	return runBacktestWithData(cfg, data)
}

func runBacktestWithData(cfg *BacktestConfig, data []types.OHLCV) *backtest.BacktestResults {
	if len(data) == 0 {
		log.Fatalf("No data provided")
	}
	start := time.Now()

	interval := guessIntervalFromPath(cfg.DataFile)
	if interval == "" {
		interval = "?"
	}

	// Config summary
	fmt.Println(strings.Repeat("-", 60))
	fmt.Printf("Run: %s @ %s\n", strings.ToUpper(cfg.Symbol), interval)
	fmt.Printf("Data: %d bars (%s → %s)\n",
		len(data),
		data[0].Timestamp.Format("2006-01-02 15:04"),
		data[len(data)-1].Timestamp.Format("2006-01-02 15:04"))
	fmt.Printf("Indicators: %s\n", indicatorSummary(cfg))
	fmt.Printf("Params: base=$%.0f, maxMult=%.2f, window=%d, commission=%.4f\n",
		cfg.BaseAmount, cfg.MaxMultiplier, cfg.WindowSize, cfg.Commission)
	
	// DCA Strategy details
	if cfg.Cycle {
		fmt.Printf("DCA Strategy: TP=%.2f%%, PriceThreshold=%.2f%% (cycle mode)\n", 
			cfg.TPPercent*100, cfg.PriceThreshold*100)
	} else {
		fmt.Printf("DCA Strategy: Hold mode (no TP), PriceThreshold=%.2f%%\n", 
			cfg.PriceThreshold*100)
	}
	
	fmt.Println(strings.Repeat("-", 60))

	// Create strategy with configured indicators
	strat := createStrategy(cfg)

	// Create and run backtest engine
	tp := cfg.TPPercent
	if !cfg.Cycle { tp = 0 }
	engine := backtest.NewBacktestEngine(cfg.InitialBalance, cfg.Commission, strat, tp)
	results := engine.Run(data, cfg.WindowSize)

	// Update all metrics
	results.UpdateMetrics()

	fmt.Printf("Elapsed: %s\n", time.Since(start).Truncate(time.Millisecond))

	return results
}

func guessIntervalFromPath(path string) string {
	dir := filepath.Dir(path)
	return filepath.Base(dir)
}

func indicatorSummary(cfg *BacktestConfig) string {
	parts := []string{}
	set := make(map[string]bool)
	for _, n := range cfg.Indicators { set[strings.ToLower(n)] = true }
	if set["rsi"] {
		parts = append(parts, fmt.Sprintf("rsi(p=%d,ov=%.0f)", cfg.RSIPeriod, cfg.RSIOversold))
	}
	if set["macd"] {
		parts = append(parts, fmt.Sprintf("macd(%d/%d/%d)", cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal))
	}
	if set["bb"] {
		parts = append(parts, fmt.Sprintf("bb(p=%d,sd=%.2f)", cfg.BBPeriod, cfg.BBStdDev))
	}
	if set["sma"] {
		parts = append(parts, fmt.Sprintf("sma(p=%d)", cfg.SMAPeriod))
	}
	
	// Add price threshold info
	if cfg.PriceThreshold > 0 {
		parts = append(parts, fmt.Sprintf("priceThreshold=%.1f%%", cfg.PriceThreshold*100))
	} else {
		parts = append(parts, "priceThreshold=disabled")
	}
	
	if len(parts) == 0 {
		return "no-config"
	}
	return strings.Join(parts, ", ")
}

func createStrategy(cfg *BacktestConfig) strategy.Strategy {
	// Build Enhanced DCA strategy
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)

	// Set price threshold for DCA entry spacing
	dca.SetPriceThreshold(cfg.PriceThreshold)

	// Indicator inclusion map
	include := make(map[string]bool)
	for _, name := range cfg.Indicators {
		include[strings.ToLower(strings.TrimSpace(name))] = true
	}

	if include["rsi"] {
		rsi := indicators.NewRSI(cfg.RSIPeriod)
		rsi.SetOversold(cfg.RSIOversold)
		rsi.SetOverbought(cfg.RSIOverbought)
		dca.AddIndicator(rsi)
	}
	if include["macd"] {
		macd := indicators.NewMACD(cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
		dca.AddIndicator(macd)
	}
	if include["bb"] {
		bb := indicators.NewBollingerBands(cfg.BBPeriod, cfg.BBStdDev)
		dca.AddIndicator(bb)
	}
	if include["sma"] {
		sma := indicators.NewSMA(cfg.SMAPeriod)
		dca.AddIndicator(sma)
	}

	// No min interval spacing; enter on closed candle when consensus reached

	return dca
}
 
var bestConfigOutPath string
var tradesCsvOutPath string

func parseDurationSafe(s string) (time.Duration, bool) {
	d, err := time.ParseDuration(s)
	if err != nil { return 0, false }
	return d, true
}

func loadHistoricalData(filename string) ([]types.OHLCV, error) {
	file, err := os.Open(filename)
	if err != nil {
		// If file doesn't exist, generate sample data
		if os.IsNotExist(err) {
			log.Println("⚠️  Historical data file not found, generating sample data...")
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
	
	for {
		record, err := reader.Read()
		if err != nil {
			break
		}
		
		// Expected format: timestamp,open,high,low,close,volume
		if len(record) < 6 {
			continue
		}
		
		timestamp, _ := time.Parse("2006-01-02 15:04:05", record[0])
		open, _ := strconv.ParseFloat(record[1], 64)
		high, _ := strconv.ParseFloat(record[2], 64)
		low, _ := strconv.ParseFloat(record[3], 64)
		close, _ := strconv.ParseFloat(record[4], 64)
		volume, _ := strconv.ParseFloat(record[5], 64)
		
		data = append(data, types.OHLCV{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}
	
	return data, nil
}

func generateSampleData() []types.OHLCV {
	// Generate 365 days of sample data
	data := make([]types.OHLCV, 365*24) // Hourly data
	startTime := time.Now().AddDate(-1, 0, 0)
	basePrice := 30000.0
	
	for i := range data {
		// Simulate price movements
		volatility := 0.02
		trend := float64(i) * 0.1 // Slight upward trend
		randomWalk := (rand.Float64() - 0.5) * basePrice * volatility
		
		price := basePrice + trend + randomWalk
		
		// Ensure price stays positive
		if price < basePrice*0.5 {
			price = basePrice * 0.5
		}
		
		data[i] = types.OHLCV{
			Timestamp: startTime.Add(time.Duration(i) * time.Hour),
			Open:      price * (1 + (rand.Float64()-0.5)*0.01),
			High:      price * (1 + rand.Float64()*0.02),
			Low:       price * (1 - rand.Float64()*0.02),
			Close:     price,
			Volume:    rand.Float64() * 1000000,
		}
		
		basePrice = price
	}
	
	return data
}

func outputResults(results *backtest.BacktestResults, cfg *BacktestConfig) {
	switch cfg.OutputFormat {
	case "json":
		outputJSON(results, cfg.OutputFile)
	case "csv":
		outputCSV(results, cfg.OutputFile)
	default:
		outputConsole(results, cfg.Verbose)
	}
}

func outputConsole(results *backtest.BacktestResults, verbose bool) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 BACKTEST RESULTS")
	fmt.Println(strings.Repeat("=", 50))
	
	fmt.Printf("💰 Initial Balance:    $%.2f\n", results.StartBalance)
	fmt.Printf("💰 Final Balance:      $%.2f\n", results.EndBalance)
	fmt.Printf("📈 Total Return:       %.2f%%\n", results.TotalReturn*100)
	fmt.Printf("📉 Max Drawdown:       %.2f%%\n", results.MaxDrawdown*100)
	fmt.Printf("📊 Sharpe Ratio:       %.2f\n", results.SharpeRatio)
	fmt.Printf("💹 Profit Factor:      %.2f\n", results.ProfitFactor)
	fmt.Printf("🔄 Total Trades:       %d\n", results.TotalTrades)
	fmt.Printf("✅ Winning Trades:     %d (%.1f%%)\n", results.WinningTrades, 
		float64(results.WinningTrades)/float64(results.TotalTrades)*100)
	fmt.Printf("❌ Losing Trades:      %d (%.1f%%)\n", results.LosingTrades,
		float64(results.LosingTrades)/float64(results.TotalTrades)*100)
	
	if verbose && len(results.Trades) > 0 {
		fmt.Println("\n📋 TRADE HISTORY:")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("%-20s %-10s %-10s %-10s %-10s\n", 
			"Entry Time", "Entry Price", "Quantity", "PnL", "Commission")
		fmt.Println(strings.Repeat("-", 80))
		
		for _, trade := range results.Trades {
			fmt.Printf("%-20s $%-9.2f %-10.6f $%-9.2f $%-9.2f\n",
				trade.EntryTime.Format("2006-01-02 15:04"),
				trade.EntryPrice,
				trade.Quantity,
				trade.PnL,
				trade.Commission)
		}
	}
	
	fmt.Println("\n" + strings.Repeat("=", 50))
}

func outputJSON(results *backtest.BacktestResults, outputFile string) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal results: %v", err)
	}
	
	if outputFile == "" {
		fmt.Println(string(data))
	} else {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Printf("Results saved to: %s\n", outputFile)
	}
}

func outputCSV(results *backtest.BacktestResults, outputFile string) {
	var w *csv.Writer
	
	if outputFile == "" {
		w = csv.NewWriter(os.Stdout)
	} else {
		file, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()
		w = csv.NewWriter(file)
	}
	
	// Write summary
	w.Write([]string{"Metric", "Value"})
	w.Write([]string{"Initial Balance", fmt.Sprintf("%.2f", results.StartBalance)})
	w.Write([]string{"Final Balance", fmt.Sprintf("%.2f", results.EndBalance)})
	w.Write([]string{"Total Return %", fmt.Sprintf("%.2f", results.TotalReturn*100)})
	w.Write([]string{"Max Drawdown %", fmt.Sprintf("%.2f", results.MaxDrawdown*100)})
	w.Write([]string{"Sharpe Ratio", fmt.Sprintf("%.2f", results.SharpeRatio)})
	w.Write([]string{"Total Trades", fmt.Sprintf("%d", results.TotalTrades)})
	
	w.Flush()
	
	if outputFile != "" {
		fmt.Printf("Results saved to: %s\n", outputFile)
	}
}

// Parameter optimization functions
func optimizeForInterval(cfg *BacktestConfig, optimizeIndicators bool) (*backtest.BacktestResults, BacktestConfig) {
	// Preload data once for performance
	data, err := loadHistoricalData(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data for optimization: %v", err)
	}
	if selectedPeriod > 0 {
		data = filterDataByPeriod(data, selectedPeriod)
	}

	// GA Parameters
	populationSize := 50
	generations := 30
	mutationRate := 0.1
	crossoverRate := 0.8
	eliteSize := 5

	fmt.Printf("🧬 Starting Genetic Algorithm Optimization\n")
	fmt.Printf("Population: %d, Generations: %d, Mutation: %.1f%%, Crossover: %.1f%%\n", 
		populationSize, generations, mutationRate*100, crossoverRate*100)

	// Initialize population
	population := initializePopulation(cfg, populationSize, optimizeIndicators)
	
	var bestIndividual *Individual
	var bestResults *backtest.BacktestResults
	
	for gen := 0; gen < generations; gen++ {
		// Evaluate fitness for all individuals
		for i := range population {
			if population[i].fitness == 0 { // Only evaluate if not already done
				results := runBacktestWithData(&population[i].config, data)
				population[i].fitness = results.TotalReturn
				population[i].results = results
			}
		}
		
		// Sort by fitness (descending)
		sortPopulationByFitness(population)
		
		// Track best individual
		if bestIndividual == nil || population[0].fitness > bestIndividual.fitness {
			bestIndividual = &Individual{
				config:  population[0].config,
				fitness: population[0].fitness,
				results: population[0].results,
			}
			bestResults = population[0].results
		}
		
		if gen%5 == 0 {
			fmt.Printf("Gen %d: Best=%.2f%%, Avg=%.2f%%, Worst=%.2f%%\n", 
				gen+1, 
				population[0].fitness*100,
				averageFitness(population)*100,
				population[len(population)-1].fitness*100)
				
			// Show best individual details every 10 generations
			if gen%10 == 9 {
				best := &population[0].config
				fmt.Printf("      Best Config: %s | maxMult=%.1f | tp=%.1f%% | threshold=%.1f%%\n",
					strings.Join(best.Indicators, "+"),
					best.MaxMultiplier,
					best.TPPercent*100,
					best.PriceThreshold*100)
			}
		}
		
		// Create next generation
		if gen < generations-1 {
			population = createNextGeneration(population, eliteSize, crossoverRate, mutationRate, cfg, optimizeIndicators)
		}
	}
	
	fmt.Printf("🏆 GA Optimization completed! Best fitness: %.2f%%\n", bestIndividual.fitness*100)
	return bestResults, bestIndividual.config
}

// optimizeExhaustive tests all indicator combinations (2+ indicators) with smaller GA
func optimizeExhaustive(cfg *BacktestConfig, optimizeIndicators bool) (*backtest.BacktestResults, BacktestConfig) {
	// Preload data once for performance
	data, err := loadHistoricalData(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data for optimization: %v", err)
	}
	if selectedPeriod > 0 {
		data = filterDataByPeriod(data, selectedPeriod)
	}

	// Generate all multi-indicator combinations (2+ indicators)
	baseIndicators := []string{"rsi", "macd", "bb", "sma"}
	allCombinations := generateMultiIndicatorCombinations(baseIndicators)
	
	// Setup logging to file
	intervalStr := guessIntervalFromPath(cfg.DataFile)
	if intervalStr == "" { intervalStr = "unknown" }
	logDir := defaultOutputDir(cfg.Symbol, intervalStr)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
	}
	logPath := filepath.Join(logDir, "exhaustive_test.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		log.Printf("Failed to create log file: %v", err)
		logFile = nil
	}
	defer func() {
		if logFile != nil {
			logFile.Close()
		}
	}()
	
	// Helper function to log to both console and file
	logBoth := func(format string, args ...interface{}) {
		message := fmt.Sprintf(format, args...)
		fmt.Print(message)
		if logFile != nil {
			logFile.WriteString(message)
		}
	}
	
	logBoth("🔍 Starting Exhaustive Combination Testing\n")
	logBoth("Testing %d combinations (2+ indicators) with reduced GA\n", len(allCombinations))
	logBoth("Symbol: %s, Interval: %s\n", cfg.Symbol, intervalStr)
	if selectedPeriod > 0 {
		logBoth("Period: %s (%d data points)\n", selectedPeriodRaw, len(data))
	} else {
		logBoth("Data points: %d\n", len(data))
	}
	logBoth("Timestamp: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	logBoth(strings.Repeat("=", 60) + "\n")
	
	var globalBestResults *backtest.BacktestResults
	var globalBestConfig BacktestConfig
	globalBestFitness := -999999.0
	
	for i, combo := range allCombinations {
		logBoth("\n[%d/%d] Testing: %s\n", i+1, len(allCombinations), strings.Join(combo, "+"))
		
		// Create config copy with this indicator combination
		comboCfg := *cfg
		comboCfg.Indicators = combo
		
		// Run smaller GA for this combination
		results, config := optimizeForCombination(&comboCfg, combo, data)
		
		fitness := results.TotalReturn
		logBoth("         Result: %.2f%% return\n", fitness*100)
		logBoth("         Config: base=$%.0f, maxMult=%.1f, tp=%.1f%%, threshold=%.1f%%\n",
			config.BaseAmount, config.MaxMultiplier, config.TPPercent*100, config.PriceThreshold*100)
		
		// Log indicator-specific parameters
		if containsIndicator(config.Indicators, "rsi") {
			logBoth("         RSI: period=%d, oversold=%.0f\n", config.RSIPeriod, config.RSIOversold)
		}
		if containsIndicator(config.Indicators, "macd") {
			logBoth("         MACD: fast=%d, slow=%d, signal=%d\n", config.MACDFast, config.MACDSlow, config.MACDSignal)
		}
		if containsIndicator(config.Indicators, "bb") {
			logBoth("         BB: period=%d, stddev=%.1f\n", config.BBPeriod, config.BBStdDev)
		}
		if containsIndicator(config.Indicators, "sma") {
			logBoth("         SMA: period=%d\n", config.SMAPeriod)
		}
		
		// Track global best
		if fitness > globalBestFitness {
			globalBestFitness = fitness
			globalBestResults = results
			globalBestConfig = config
			logBoth("         🌟 NEW BEST! (%.2f%%)\n", fitness*100)
		}
	}
	
	logBoth("\n" + strings.Repeat("=", 60) + "\n")
	logBoth("🏆 Exhaustive optimization completed!\n")
	logBoth("Best combination: %s (%.2f%% return)\n", 
		strings.Join(globalBestConfig.Indicators, "+"), globalBestFitness*100)
	logBoth("Best config details:\n")
	logBoth("  Base Amount: $%.0f\n", globalBestConfig.BaseAmount)
	logBoth("  Max Multiplier: %.1f\n", globalBestConfig.MaxMultiplier)
	logBoth("  TP Percent: %.1f%%\n", globalBestConfig.TPPercent*100)
	logBoth("  Price Threshold: %.1f%%\n", globalBestConfig.PriceThreshold*100)
	if containsIndicator(globalBestConfig.Indicators, "rsi") {
		logBoth("  RSI: period=%d, oversold=%.0f\n", globalBestConfig.RSIPeriod, globalBestConfig.RSIOversold)
	}
	if containsIndicator(globalBestConfig.Indicators, "macd") {
		logBoth("  MACD: fast=%d, slow=%d, signal=%d\n", globalBestConfig.MACDFast, globalBestConfig.MACDSlow, globalBestConfig.MACDSignal)
	}
	if containsIndicator(globalBestConfig.Indicators, "bb") {
		logBoth("  BB: period=%d, stddev=%.1f\n", globalBestConfig.BBPeriod, globalBestConfig.BBStdDev)
	}
	if containsIndicator(globalBestConfig.Indicators, "sma") {
		logBoth("  SMA: period=%d\n", globalBestConfig.SMAPeriod)
	}
	logBoth("Optimization completed at: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	
	if logFile != nil {
		fmt.Printf("📝 Detailed log saved to: %s\n", logPath)
	}
	
	return globalBestResults, globalBestConfig
}

// Individual represents a candidate solution
type Individual struct {
	config  BacktestConfig
	fitness float64
	results *backtest.BacktestResults
}

// Initialize random population
func initializePopulation(cfg *BacktestConfig, size int, optimizeIndicators bool) []*Individual {
	population := make([]*Individual, size)
	
	// Base indicators for combination generation
	baseIndicators := cfg.Indicators
	if len(baseIndicators) == 0 {
		baseIndicators = []string{"rsi", "macd", "bb", "sma"}
	}
	
	// Fixed parameter ranges for all optimization (no style constraints)
	multipliers := []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0}
	tpCandidates := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06}
	priceThresholds := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05}
	rsiPeriods := []int{10, 12, 14, 16, 18, 20, 22, 25}
	rsiOversold := []float64{20, 25, 30, 35, 40}
	macdFast := []int{6, 8, 10, 12, 14, 16, 18}
	macdSlow := []int{20, 22, 24, 26, 28, 30, 32, 35}
	macdSignal := []int{7, 8, 9, 10, 12, 14}
	bbPeriods := []int{10, 14, 16, 18, 20, 22, 25, 28, 30}
	bbStdDev := []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0}
	smaPeriods := []int{15, 20, 25, 30, 40, 50, 60, 75, 100, 120}
	
	for i := 0; i < size; i++ {
		individual := &Individual{
			config: *cfg, // Copy base config
		}
		
		// Randomize parameters using fixed ranges
		individual.config.BaseAmount = cfg.BaseAmount // Use flag value
		individual.config.MaxMultiplier = randomChoice(multipliers)
		individual.config.PriceThreshold = randomChoice(priceThresholds)
		
		// TP candidates
		if cfg.Cycle {
			individual.config.TPPercent = randomChoice(tpCandidates)
		} else {
			individual.config.TPPercent = 0
		}
		
		// Indicator selection
		if optimizeIndicators {
			individual.config.Indicators = randomIndicatorCombo(baseIndicators)
		} else {
			individual.config.Indicators = cfg.Indicators
		}
		
		// Indicator parameters using fixed ranges
		individual.config.RSIPeriod = randomChoice(rsiPeriods)
		individual.config.RSIOversold = randomChoice(rsiOversold)
		individual.config.MACDFast = randomChoice(macdFast)
		individual.config.MACDSlow = randomChoice(macdSlow)
		individual.config.MACDSignal = randomChoice(macdSignal)
		individual.config.BBPeriod = randomChoice(bbPeriods)
		individual.config.BBStdDev = randomChoice(bbStdDev)
		individual.config.SMAPeriod = randomChoice(smaPeriods)
		
		population[i] = individual
	}
	
	return population
}

// Random selection helpers
func randomChoice[T any](choices []T) T {
	if len(choices) == 0 {
		var zero T
		return zero
	}
	idx := rng.Intn(len(choices))
	return choices[idx]
}

func randomIndicatorCombo(base []string) []string {
	if len(base) == 0 {
		return []string{}
	}
	
	// Generate random bitmask (at least 1 indicator)
	mask := 1 + (rng.Intn((1 << len(base)) - 1))
	
	var combo []string
	for i := 0; i < len(base); i++ {
		if mask&(1<<i) != 0 {
			combo = append(combo, base[i])
		}
	}
	return combo
}

// Sort population by fitness (descending)
func sortPopulationByFitness(population []*Individual) {
	for i := 0; i < len(population)-1; i++ {
		for j := i + 1; j < len(population); j++ {
			if population[j].fitness > population[i].fitness {
				population[i], population[j] = population[j], population[i]
			}
		}
	}
}

// Calculate average fitness
func averageFitness(population []*Individual) float64 {
	sum := 0.0
	for _, ind := range population {
		sum += ind.fitness
	}
	return sum / float64(len(population))
}

// Create next generation using selection, crossover, and mutation
func createNextGeneration(population []*Individual, eliteSize int, crossoverRate, mutationRate float64, cfg *BacktestConfig, optimizeIndicators bool) []*Individual {
	newPop := make([]*Individual, len(population))
	
	// Elitism: keep best individuals
	for i := 0; i < eliteSize; i++ {
		newPop[i] = &Individual{
			config:  population[i].config,
			fitness: population[i].fitness,
			results: population[i].results,
		}
	}
	
	// Fill rest with crossover and mutation
	for i := eliteSize; i < len(population); i++ {
		parent1 := tournamentSelection(population, 3)
		parent2 := tournamentSelection(population, 3)
		
		child := crossover(parent1, parent2, crossoverRate)
		mutate(child, mutationRate, cfg, optimizeIndicators)
		
		newPop[i] = child
	}
	
	return newPop
}

// Tournament selection
func tournamentSelection(population []*Individual, tournamentSize int) *Individual {
	best := population[rng.Intn(len(population))]
	
	for i := 1; i < tournamentSize; i++ {
		candidate := population[rng.Intn(len(population))]
		if candidate.fitness > best.fitness {
			best = candidate
		}
	}
	
	return best
}

// Crossover two parents to create a child
func crossover(parent1, parent2 *Individual, rate float64) *Individual {
	child := &Individual{
		config: parent1.config, // Start with parent1
	}
	
	// Random crossover based on rate
	if rng.Float64() < rate {
		// Mix parameters from both parents
		if rng.Intn(2) == 0 {
			child.config.MaxMultiplier = parent2.config.MaxMultiplier
		}
		if rng.Intn(2) == 0 {
			child.config.TPPercent = parent2.config.TPPercent
		}
		if rng.Intn(2) == 0 {
			child.config.RSIPeriod = parent2.config.RSIPeriod
		}
		if rng.Intn(2) == 0 {
			child.config.RSIOversold = parent2.config.RSIOversold
		}
		if rng.Intn(2) == 0 {
			child.config.MACDFast = parent2.config.MACDFast
		}
		if rng.Intn(2) == 0 {
			child.config.MACDSlow = parent2.config.MACDSlow
		}
		if rng.Intn(2) == 0 {
			child.config.MACDSignal = parent2.config.MACDSignal
		}
		if rng.Intn(2) == 0 {
			child.config.BBPeriod = parent2.config.BBPeriod
		}
		if rng.Intn(2) == 0 {
			child.config.BBStdDev = parent2.config.BBStdDev
		}
		if rng.Intn(2) == 0 {
			child.config.SMAPeriod = parent2.config.SMAPeriod
		}
		if rng.Intn(2) == 0 {
			child.config.Indicators = parent2.config.Indicators
		}
	}
	
	return child
}

// Mutate an individual
func mutate(individual *Individual, rate float64, cfg *BacktestConfig, optimizeIndicators bool) {
	if rng.Float64() < rate {
		// Fixed parameter ranges for mutation
		multipliers := []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0}
		tpCandidates := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06}
		priceThresholds := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05}
		rsiPeriods := []int{10, 12, 14, 16, 18, 20, 22, 25}
		rsiOversold := []float64{20, 25, 30, 35, 40}
		macdFast := []int{6, 8, 10, 12, 14, 16, 18}
		macdSlow := []int{20, 22, 24, 26, 28, 30, 32, 35}
		macdSignal := []int{7, 8, 9, 10, 12, 14}
		bbPeriods := []int{10, 14, 16, 18, 20, 22, 25, 28, 30}
		bbStdDev := []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0}
		smaPeriods := []int{15, 20, 25, 30, 40, 50, 60, 75, 100, 120}
		
		// Randomly mutate one parameter (expanded to include price threshold)
		switch rng.Intn(11) {
		case 0:
			individual.config.MaxMultiplier = randomChoice(multipliers)
		case 1:
			if cfg.Cycle {
				individual.config.TPPercent = randomChoice(tpCandidates)
			}
		case 2:
			individual.config.PriceThreshold = randomChoice(priceThresholds)
		case 3:
			individual.config.RSIPeriod = randomChoice(rsiPeriods)
		case 4:
			individual.config.RSIOversold = randomChoice(rsiOversold)
		case 5:
			individual.config.MACDFast = randomChoice(macdFast)
		case 6:
			individual.config.MACDSlow = randomChoice(macdSlow)
		case 7:
			individual.config.MACDSignal = randomChoice(macdSignal)
		case 8:
			individual.config.BBPeriod = randomChoice(bbPeriods)
		case 9:
			individual.config.BBStdDev = randomChoice(bbStdDev)
		case 10:
			individual.config.SMAPeriod = randomChoice(smaPeriods)
		}
		
		// Reset fitness to force re-evaluation
		individual.fitness = 0
		individual.results = nil
	}
}

var (
	rng = rand.New(rand.NewSource(time.Now().UnixNano())) // Random number generator for optimization
)

// generateMultiIndicatorCombinations generates all combinations with 2+ indicators
func generateMultiIndicatorCombinations(indicators []string) [][]string {
	var combinations [][]string
	n := len(indicators)
	
	// Generate all combinations from 2 to n indicators
	for mask := 3; mask < (1 << n); mask++ { // Start from 3 (binary 11) to exclude single indicators
		var combo []string
		for i := 0; i < n; i++ {
			if mask&(1<<i) != 0 {
				combo = append(combo, indicators[i])
			}
		}
		combinations = append(combinations, combo)
	}
	
	return combinations
}

// optimizeForCombination runs a smaller GA for a specific indicator combination
func optimizeForCombination(cfg *BacktestConfig, indicators []string, data []types.OHLCV) (*backtest.BacktestResults, BacktestConfig) {
	// Smaller GA parameters for faster execution
	populationSize := 20  // Reduced from 50
	generations := 15     // Reduced from 30
	mutationRate := 0.15
	crossoverRate := 0.8
	eliteSize := 3        // Reduced from 5
	
	// Initialize population for this specific combination
	population := initializePopulationForCombo(cfg, populationSize, indicators)
	
	var bestIndividual *Individual
	var bestResults *backtest.BacktestResults
	
	for gen := 0; gen < generations; gen++ {
		// Evaluate fitness for all individuals
		for i := range population {
			if population[i].fitness == 0 {
				results := runBacktestWithData(&population[i].config, data)
				population[i].fitness = results.TotalReturn
				population[i].results = results
			}
		}
		
		// Sort by fitness
		sortPopulationByFitness(population)
		
		// Track best
		if bestIndividual == nil || population[0].fitness > bestIndividual.fitness {
			bestIndividual = &Individual{
				config:  population[0].config,
				fitness: population[0].fitness,
				results: population[0].results,
			}
			bestResults = population[0].results
		}
		
		// Create next generation
		if gen < generations-1 {
			population = createNextGenerationForCombo(population, eliteSize, crossoverRate, mutationRate, cfg, indicators)
		}
	}
	
	return bestResults, bestIndividual.config
}

// initializePopulationForCombo creates population for a specific indicator combination
func initializePopulationForCombo(cfg *BacktestConfig, size int, indicators []string) []*Individual {
	population := make([]*Individual, size)
	
	// Fixed parameter ranges
	multipliers := []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0}
	tpCandidates := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06}
	priceThresholds := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05}
	rsiPeriods := []int{10, 12, 14, 16, 18, 20, 22, 25}
	rsiOversold := []float64{20, 25, 30, 35, 40}
	macdFast := []int{6, 8, 10, 12, 14, 16, 18}
	macdSlow := []int{20, 22, 24, 26, 28, 30, 32, 35}
	macdSignal := []int{7, 8, 9, 10, 12, 14}
	bbPeriods := []int{10, 14, 16, 18, 20, 22, 25, 28, 30}
	bbStdDev := []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0}
	smaPeriods := []int{15, 20, 25, 30, 40, 50, 60, 75, 100, 120}
	
	for i := 0; i < size; i++ {
		individual := &Individual{
			config: *cfg,
		}
		
		// Fixed indicator combination (no randomization)
		individual.config.Indicators = indicators
		individual.config.BaseAmount = cfg.BaseAmount
		individual.config.MaxMultiplier = randomChoice(multipliers)
		individual.config.PriceThreshold = randomChoice(priceThresholds)
		
		if cfg.Cycle {
			individual.config.TPPercent = randomChoice(tpCandidates)
		} else {
			individual.config.TPPercent = 0
		}
		
		// Randomize indicator parameters
		individual.config.RSIPeriod = randomChoice(rsiPeriods)
		individual.config.RSIOversold = randomChoice(rsiOversold)
		individual.config.MACDFast = randomChoice(macdFast)
		individual.config.MACDSlow = randomChoice(macdSlow)
		individual.config.MACDSignal = randomChoice(macdSignal)
		individual.config.BBPeriod = randomChoice(bbPeriods)
		individual.config.BBStdDev = randomChoice(bbStdDev)
		individual.config.SMAPeriod = randomChoice(smaPeriods)
		
		population[i] = individual
	}
	
	return population
}

// createNextGenerationForCombo creates next generation for a specific combination
func createNextGenerationForCombo(population []*Individual, eliteSize int, crossoverRate, mutationRate float64, cfg *BacktestConfig, indicators []string) []*Individual {
	newPop := make([]*Individual, len(population))
	
	// Elitism
	for i := 0; i < eliteSize; i++ {
		newPop[i] = &Individual{
			config:  population[i].config,
			fitness: population[i].fitness,
			results: population[i].results,
		}
	}
	
	// Fill rest with crossover and mutation
	for i := eliteSize; i < len(population); i++ {
		parent1 := tournamentSelection(population, 3)
		parent2 := tournamentSelection(population, 3)
		
		child := crossover(parent1, parent2, crossoverRate)
		mutateForCombo(child, mutationRate, cfg, indicators)
		
		newPop[i] = child
	}
	
	return newPop
}

// mutateForCombo mutates individual while keeping the indicator combination fixed
func mutateForCombo(individual *Individual, rate float64, cfg *BacktestConfig, indicators []string) {
	if rng.Float64() < rate {
		// Fixed parameter ranges
		multipliers := []float64{1.2, 1.5, 1.8, 2.0, 2.5, 3.0, 3.5, 4.0}
		tpCandidates := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05, 0.055, 0.06}
		priceThresholds := []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05}
		rsiPeriods := []int{10, 12, 14, 16, 18, 20, 22, 25}
		rsiOversold := []float64{20, 25, 30, 35, 40}
		macdFast := []int{6, 8, 10, 12, 14, 16, 18}
		macdSlow := []int{20, 22, 24, 26, 28, 30, 32, 35}
		macdSignal := []int{7, 8, 9, 10, 12, 14}
		bbPeriods := []int{10, 14, 16, 18, 20, 22, 25, 28, 30}
		bbStdDev := []float64{1.5, 1.8, 2.0, 2.2, 2.5, 2.8, 3.0}
		smaPeriods := []int{15, 20, 25, 30, 40, 50, 60, 75, 100, 120}
		
		// Keep indicators fixed, only mutate parameters
		individual.config.Indicators = indicators
		
		// Randomly mutate one parameter (excluding indicator combination)
		switch rng.Intn(11) {
		case 0:
			individual.config.MaxMultiplier = randomChoice(multipliers)
		case 1:
			if cfg.Cycle {
				individual.config.TPPercent = randomChoice(tpCandidates)
			}
		case 2:
			individual.config.PriceThreshold = randomChoice(priceThresholds)
		case 3:
			individual.config.RSIPeriod = randomChoice(rsiPeriods)
		case 4:
			individual.config.RSIOversold = randomChoice(rsiOversold)
		case 5:
			individual.config.MACDFast = randomChoice(macdFast)
		case 6:
			individual.config.MACDSlow = randomChoice(macdSlow)
		case 7:
			individual.config.MACDSignal = randomChoice(macdSignal)
		case 8:
			individual.config.BBPeriod = randomChoice(bbPeriods)
		case 9:
			individual.config.BBStdDev = randomChoice(bbStdDev)
		case 10:
			individual.config.SMAPeriod = randomChoice(smaPeriods)
		}
		
		// Reset fitness
		individual.fitness = 0
		individual.results = nil
	}
}

func parseIndicatorsList(list string) []string {
	parts := strings.Split(list, ",")
	res := make([]string, 0, len(parts))
	for _, p := range parts {
		n := strings.ToLower(strings.TrimSpace(p))
		if n == "rsi" || n == "macd" || n == "bb" || n == "sma" {
			res = append(res, n)
		}
	}
	return res
}

func containsIndicator(indicators []string, name string) bool {
	name = strings.ToLower(name)
	for _, n := range indicators {
		if strings.ToLower(n) == name { return true }
	}
	return false
}

func printBestConfigJSON(cfg BacktestConfig) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println(string(data))
}

func writeBestConfigJSON(cfg BacktestConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil { return err }
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil { return err }
	}
	return os.WriteFile(path, data, 0644)
}

func defaultBestConfigPath(symbol, interval string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	i := strings.ToLower(strings.TrimSpace(interval))
	if s == "" { s = "UNKNOWN" }
	if i == "" { i = "unknown" }
	return filepath.Join("results", fmt.Sprintf("best_%s_%s.json", s, i))
}

// trades CSV writer
var bestTradesCsvOutPath string

func writeTradesCSV(results *backtest.BacktestResults, path string) error {
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil { return err }
	}

	// If the user requests an Excel file, write two tabs: Trades and Cycles
	if strings.HasSuffix(strings.ToLower(path), ".xlsx") {
		return writeTradesXLSX(results, path)
	}

	// CSV path: write only trades with improved headers and order
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// header with improved names and order (Cycle first)
	if err := w.Write([]string{
		"Cycle",
		"Entry price",
		"Exit price",
		"Entry time",
		"Exit time",
		"Quantity (USDT)",
		"Summary",
	}); err != nil { return err }

	// running aggregates for summary
	var cumQty float64
	var cumCost float64
	var totalPnL float64

	for _, t := range results.Trades {
		netCost := t.EntryPrice * t.Quantity
		grossCost := netCost + t.Commission
		cumQty += t.Quantity
		cumCost += grossCost
		totalPnL += t.PnL

		qtyUSDT := grossCost

		row := []string{
			strconv.Itoa(t.Cycle),
			fmt.Sprintf("%.8f", t.EntryPrice),
			fmt.Sprintf("%.8f", t.ExitPrice),
			t.EntryTime.Format("2006-01-02 15:04:05"),
			t.ExitTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.3f", qtyUSDT),
			"",
		}
		if err := w.Write(row); err != nil { return err }
	}

	// final summary row: only pnl_usdt and capital_usdt in the summary column
	summary := fmt.Sprintf("pnl_usdt=%.3f; capital_usdt=%.3f", totalPnL, cumCost)
	if err := w.Write([]string{"", "", "", "", "", "", summary}); err != nil { return err }

	return nil
}

// writeTradesXLSX writes trades and cycles to separate sheets in an Excel file
func writeTradesXLSX(results *backtest.BacktestResults, path string) error {
	// lazy import
	// NOTE: excelize is a direct dependency in go.mod
	fx := excelize.NewFile()
	defer fx.Close()

	// Create sheets
	const tradesSheet = "Trades"
	const cyclesSheet = "Cycles"
	// Replace default sheet
	fx.SetSheetName(fx.GetSheetName(0), tradesSheet)
	fx.NewSheet(cyclesSheet)

	// Header styles (optional bold)
	headStyle, _ := fx.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})

	// Write Trades header
	tradeHeaders := []string{"Cycle", "Entry price", "Exit price", "Entry time", "Exit time", "Quantity (USDT)", "Summary"}
	for i, h := range tradeHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(tradesSheet, cell, h)
		fx.SetCellStyle(tradesSheet, cell, cell, headStyle)
	}

	// Write Trades rows
	row := 2
	var cumCost float64
	var totalPnL float64
	for _, t := range results.Trades {
		netCost := t.EntryPrice * t.Quantity
		grossCost := netCost + t.Commission
		cumCost += grossCost
		totalPnL += t.PnL

		values := []interface{}{
			t.Cycle,
			fmt.Sprintf("%.8f", t.EntryPrice),
			fmt.Sprintf("%.8f", t.ExitPrice),
			t.EntryTime.Format("2006-01-02 15:04:05"),
			t.ExitTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.3f", grossCost),
			"",
		}
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(tradesSheet, cell, v)
		}
		row++
	}
	// Final summary cell in the last row, last column
	if row > 2 {
		cell, _ := excelize.CoordinatesToCellName(len(tradeHeaders), row)
		fx.SetCellValue(tradesSheet, cell, fmt.Sprintf("pnl_usdt=%.3f; capital_usdt=%.3f", totalPnL, cumCost))
	}

	// Write Cycles sheet if any
	cycleHeaders := []string{"Cycle number", "Start time", "End time", "Entries", "Target price", "Average price", "PnL (USDT)", "Capital (USDT)"}
	for i, h := range cycleHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(cyclesSheet, cell, h)
		fx.SetCellStyle(cyclesSheet, cell, cell, headStyle)
	}
	row = 2
	if len(results.Cycles) > 0 {
		for _, c := range results.Cycles {
			// compute capital_usdt for this cycle from trades
			capital := 0.0
			for _, t := range results.Trades {
				if t.Cycle == c.CycleNumber {
					capital += t.EntryPrice*t.Quantity + t.Commission
				}
			}
			values := []interface{}{
				c.CycleNumber,
				c.StartTime.Format("2006-01-02 15:04:05"),
				c.EndTime.Format("2006-01-02 15:04:05"),
				c.Entries,
				fmt.Sprintf("%.8f", c.TargetPrice),
				fmt.Sprintf("%.8f", c.AvgEntry),
				fmt.Sprintf("%.3f", c.RealizedPnL),
				fmt.Sprintf("%.3f", capital),
			}
			for i, v := range values {
				cell, _ := excelize.CoordinatesToCellName(i+1, row)
				fx.SetCellValue(cyclesSheet, cell, v)
			}
			row++
		}
	}

	// Save workbook
	return fx.SaveAs(path)
}

func defaultTradesCsvPath(symbol, interval string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	i := strings.ToLower(strings.TrimSpace(interval))
	if s == "" { s = "UNKNOWN" }
	if i == "" { i = "unknown" }
	return filepath.Join("results", fmt.Sprintf("trades_%s_%s.xlsx", s, i))
}

// trailing period selection
var selectedPeriod time.Duration
var selectedPeriodRaw string

func parseTrailingPeriod(s string) (time.Duration, bool) {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasSuffix(s, "days") {
		s = strings.TrimSuffix(s, "days") + "d"
	}
	if strings.HasSuffix(s, "d") {
		nStr := strings.TrimSuffix(s, "d")
		if nStr == "" { return 0, false }
		n, err := strconv.Atoi(nStr)
		if err != nil || n <= 0 { return 0, false }
		return time.Duration(n) * 24 * time.Hour, true
	}
	// allow raw durations too (e.g., 168h)
	if d, err := time.ParseDuration(s); err == nil { return d, true }
	return 0, false
}

func filterDataByPeriod(data []types.OHLCV, period time.Duration) []types.OHLCV {
	if period <= 0 || len(data) == 0 { return data }
	end := data[len(data)-1].Timestamp
	cut := end.Add(-period)
	// find first index >= cut
	lo, hi := 0, len(data)-1
	idx := 0
	for lo <= hi {
		mid := (lo + hi) / 2
		if data[mid].Timestamp.Before(cut) {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	idx = lo
	if idx < 0 { idx = 0 }
	if idx >= len(data) { return []types.OHLCV{} }
	return data[idx:]
}

// Helper to resolve default output dir
func defaultOutputDir(symbol, interval string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	i := strings.ToLower(strings.TrimSpace(interval))
	if s == "" { s = "UNKNOWN" }
	if i == "" { i = "unknown" }
	return filepath.Join("results", fmt.Sprintf("%s_%s", s, i))
}

// flagProvided returns true if the named flag appears in os.Args
func flagProvided(name string) bool {
    // Accept forms: -name, -name=value, --name, --name=value
    dash := "-" + name
    ddash := "--" + name
    for _, arg := range os.Args[1:] {
        if strings.HasPrefix(arg, dash) || strings.HasPrefix(arg, ddash) {
            return true
        }
    }
    return false
}