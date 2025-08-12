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
		baseAmount     = flag.Float64("base-amount", 100, "Base DCA amount")
		maxMultiplier  = flag.Float64("max-multiplier", 3, "Maximum position multiplier")
		outputFormat   = flag.String("output", "console", "Output format: console, json, csv")
		outputFile     = flag.String("output-file", "", "Output file path")
		verbose        = flag.Bool("verbose", false, "Verbose output")
		optimize       = flag.Bool("optimize", false, "Run parameter optimization")
		allIntervals   = flag.Bool("all-intervals", false, "Scan data root for all intervals for the given symbol and run per-interval backtests/optimizations")
		dataRoot       = flag.String("data-root", "data/historical", "Root folder containing <SYMBOL>/<INTERVAL>/candles.csv")
		indicatorsFlag = flag.String("indicators", "", "Comma-separated indicators to include (rsi,macd,bb,sma). Empty = use config or default all")
		optiIndicators = flag.Bool("optimize-indicators", false, "When optimizing, also search over indicator combinations")
		bestConfigOut  = flag.String("best-config-out", "", "If set, write the best optimized config to this JSON file")
		periodStr      = flag.String("period", "", "Limit data to trailing window (e.g., 7d,30d,180d,365d or Nd)")
		tradesCsvOut   = flag.String("trades-csv-out", "", "If set, write detailed trade history CSV to this file")
		tpPercentFlag  = flag.Float64("tp", 0, "Take-profit percent as decimal (e.g., 0.02). Applies only when -cycle is true")
        cycleFlag      = flag.Bool("cycle", true, "Enable TP cycle (DCA -> TP -> reset). Set false to hold and ignore -tp")
	)
	
	flag.Parse()
	
	// Load configuration
	cfg := loadConfig(*configFile, *dataFile, *symbol, *initialBalance, *commission, 
		*windowSize, *baseAmount, *maxMultiplier, *outputFormat, *outputFile, *verbose)

	// apply cycle and tp from flags
	cfg.Cycle = *cycleFlag
	if !cfg.Cycle {
		// hold mode: ignore tp
		cfg.TPPercent = 0
	} else if *tpPercentFlag > 0 {
		cfg.TPPercent = *tpPercentFlag
	}
	// If cycle enabled but no TP provided and not optimizing, use a sensible default (2%)
	if cfg.Cycle && cfg.TPPercent == 0 && !*optimize {
		cfg.TPPercent = 0.02
	}

	// Capture best-config output path flag into package variable
	bestConfigOutPath = strings.TrimSpace(*bestConfigOut)

	// Capture and parse trailing period window
	if s := strings.TrimSpace(*periodStr); s != "" {
		if d, ok := parseTrailingPeriod(s); ok {
			selectedPeriod = d
			selectedPeriodRaw = s
		}
	}

	// Capture trades CSV output path into package variable
	tradesCsvOutPath = strings.TrimSpace(*tradesCsvOut)

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
		bestResults, bestConfig := optimizeForInterval(cfg, *optiIndicators)
		fmt.Println("\n\n🏆 OPTIMIZATION RESULTS:")
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("Best Parameters:\n")
		fmt.Printf("  Indicators:     %s\n", strings.Join(bestConfig.Indicators, ","))
		fmt.Printf("  Base Amount:    $%.2f\n", bestConfig.BaseAmount)
		fmt.Printf("  Max Multiplier: %.2f\n", bestConfig.MaxMultiplier)
		fmt.Printf("  TP Percent:     %.3f%%\n", bestConfig.TPPercent*100)
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
		printBestConfigJSON(bestConfig)
		// Determine standard output dir based on symbol/interval
		intervalStr := filepath.Base(filepath.Dir(bestConfig.DataFile))
		if intervalStr == "" { intervalStr = filepath.Base(filepath.Dir(cfg.DataFile)) }
		if intervalStr == "" { intervalStr = "unknown" }
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
			fmt.Printf("Failed to write trades CSV: %v\n", err)
		} else {
			fmt.Printf("Saved trades CSV to: %s\n", stdTradesPath)
		}
		// Optional additional copies based on user flags
		if bestConfigOutPath != "" {
			if err := writeBestConfigJSON(bestConfig, bestConfigOutPath); err != nil {
				fmt.Printf("Failed to write best config: %v\n", err)
			} else {
				fmt.Printf("Saved best config to: %s\n", bestConfigOutPath)
			}
		}
		if tradesCsvOutPath != "" {
			if err := writeTradesCSV(bestResults, tradesCsvOutPath); err != nil {
				fmt.Printf("Failed to write trades CSV: %v\n", err)
			} else {
				fmt.Printf("Saved trades CSV to: %s\n", tradesCsvOutPath)
			}
		}
		fmt.Println("\nBest Results:")
		outputConsole(bestResults, false)
		return
	}
	
	// Run single backtest
	results := runBacktest(cfg)
	
	// Output results
	outputResults(results, cfg)
	
	// Optionally write trades CSV (single run)
	if path := tradesCsvOutPath; path != "" {
		if err := writeTradesCSV(results, path); err != nil {
			fmt.Printf("Failed to write trades CSV: %v\n", err)
		} else {
			fmt.Printf("Saved trades CSV to: %s\n", path)
		}
	}
}

func loadConfig(configFile, dataFile, symbol string, balance, commission float64,
	windowSize int, baseAmount, maxMultiplier float64, outputFormat, outputFile string, verbose bool) *BacktestConfig {
	
	cfg := &BacktestConfig{
		DataFile:       dataFile,
		Symbol:         symbol,
		InitialBalance: balance,
		Commission:     commission,
		WindowSize:     windowSize,
		BaseAmount:     baseAmount,
		MaxMultiplier:  maxMultiplier,
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
	fmt.Println("Interval | Return% | Trades | Base$ | MaxMult | TP% | RSI(p/ov) | Indicators")
	for _, r := range resultsByInterval {
		c := r.OptimizedCfg
		rsiInfo := "-/-"
		if containsIndicator(c.Indicators, "rsi") {
			rsiInfo = fmt.Sprintf("%d/%.0f", c.RSIPeriod, c.RSIOversold)
		}
		fmt.Printf("%-8s | %7.2f | %6d | %5.0f | %7.2f | %5.2f | %8s | %s\n",
			r.Interval,
			r.Results.TotalReturn*100,
			r.Results.TotalTrades,
			c.BaseAmount,
			c.MaxMultiplier,
			c.TPPercent*100,
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
	printBestConfigJSON(best.OptimizedCfg)
	outPath := bestConfigOutPath
	if outPath == "" {
		outPath = defaultBestConfigPath(cfg.Symbol, best.Interval)
	}
	if err := writeBestConfigJSON(best.OptimizedCfg, outPath); err != nil {
		fmt.Printf("Failed to write best config: %v\n", err)
	} else {
		fmt.Printf("Saved best config to: %s\n", outPath)
	}

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
		fmt.Printf("Failed to write trades CSV: %v\n", err)
	} else {
		fmt.Printf("Saved trades CSV to: %s\n", stdTradesPath)
	}
	// Optional additional copies based on user flags
	if bestConfigOutPath != "" {
		if err := writeBestConfigJSON(best.OptimizedCfg, bestConfigOutPath); err != nil {
			fmt.Printf("Failed to write best config: %v\n", err)
		} else {
			fmt.Printf("Saved best config to: %s\n", bestConfigOutPath)
		}
	}
	if tradesCsvOutPath != "" {
		if err := writeTradesCSV(best.Results, tradesCsvOutPath); err != nil {
			fmt.Printf("Failed to write trades CSV: %v\n", err)
		} else {
			fmt.Printf("Saved trades CSV to: %s\n", tradesCsvOutPath)
		}
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
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, ", ")
}

func createStrategy(cfg *BacktestConfig) strategy.Strategy {
	// Build Enhanced DCA strategy
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)

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
	// Candidate indicator names
	baseIndicators := cfg.Indicators
	if len(baseIndicators) == 0 {
		baseIndicators = []string{"rsi", "macd", "bb", "sma"}
	}

	// Preload data once for performance
	data, err := loadHistoricalData(cfg.DataFile)
	if err != nil {
		log.Fatalf("Failed to load data for optimization: %v", err)
	}
	// Apply trailing period filter if set
	if selectedPeriod > 0 {
		data = filterDataByPeriod(data, selectedPeriod)
	}

	// Helper to generate indicator combos
	genCombos := func(base []string) [][]string {
		var combos [][]string
		n := len(base)
		for mask := 1; mask < (1 << n); mask++ { // skip empty set
			var combo []string
			for i := 0; i < n; i++ {
				if mask&(1<<i) != 0 {
					combo = append(combo, base[i])
				}
			}
			combos = append(combos, combo)
		}
		return combos
	}

	// Determine which combos to evaluate
	combos := [][]string{cfg.Indicators}
	if optimizeIndicators {
		combos = genCombos(baseIndicators)
	}

	// TP candidates default when optimizing; if cycle disabled, force hold (0)
	tpCandidates := []float64{0}
	if cfg.Cycle {
		tpCandidates = []float64{0.01, 0.015, 0.02, 0.025, 0.03, 0.035, 0.04, 0.045, 0.05}
	}

	// Parameter ranges
	baseAmounts := []float64{50, 100, 200}
	multipliers := []float64{1.5, 2.0, 3.0}
	rsiPeriods := []int{12, 14, 20}
	rsiOversold := []float64{25, 30, 35}
	macdFastSet := []int{8, 12, 16}
	macdSlowSet := []int{20, 26, 30}
	macdSignalSet := []int{9, 12}
	bbPeriodSet := []int{14, 20, 28}
	bbStdSet := []float64{2.0, 2.5}
	smaPeriodSet := []int{20, 50, 100}

	var bestResults *backtest.BacktestResults
	var bestConfig BacktestConfig
	bestReturn := -1000.0

	for _, combo := range combos {
		// Build a set for quick checks
		set := make(map[string]bool)
		for _, n := range combo { set[n] = true }

		// If an indicator not included, lock its params to current cfg
		curRSIPeriods := rsiPeriods
		curRSIOversold := rsiOversold
		if !set["rsi"] {
			curRSIPeriods = []int{cfg.RSIPeriod}
			curRSIOversold = []float64{cfg.RSIOversold}
		}
		curMACDFast := macdFastSet
		curMACDSlow := macdSlowSet
		curMACDSignal := macdSignalSet
		if !set["macd"] {
			curMACDFast = []int{cfg.MACDFast}
			curMACDSlow = []int{cfg.MACDSlow}
			curMACDSignal = []int{cfg.MACDSignal}
		}
		curBBPeriod := bbPeriodSet
		curBBStd := bbStdSet
		if !set["bb"] {
			curBBPeriod = []int{cfg.BBPeriod}
			curBBStd = []float64{cfg.BBStdDev}
		}
		curSMAPeriod := smaPeriodSet
		if !set["sma"] {
			curSMAPeriod = []int{cfg.SMAPeriod}
		}

		for _, baseAmount := range baseAmounts {
			for _, multiplier := range multipliers {
				for _, rsiPeriod := range curRSIPeriods {
					for _, oversold := range curRSIOversold {
						for _, mf := range curMACDFast {
							for _, ms := range curMACDSlow {
								for _, msi := range curMACDSignal {
									for _, bbp := range curBBPeriod {
										for _, bbs := range curBBStd {
											for _, smap := range curSMAPeriod {
												for _, tp := range tpCandidates {
													// Create test configuration
													testCfg := *cfg
													testCfg.Indicators = combo
													testCfg.BaseAmount = baseAmount
													testCfg.MaxMultiplier = multiplier
													testCfg.RSIPeriod = rsiPeriod
													testCfg.RSIOversold = oversold
													testCfg.MACDFast = mf
													testCfg.MACDSlow = ms
													testCfg.MACDSignal = msi
													testCfg.BBPeriod = bbp
													testCfg.BBStdDev = bbs
													testCfg.SMAPeriod = smap
													if cfg.Cycle {
														testCfg.TPPercent = tp
													} else {
														testCfg.TPPercent = 0
													}

													// Run backtest with preloaded data
													results := runBacktestWithData(&testCfg, data)

													// Track best
													if results.TotalReturn > bestReturn {
														bestReturn = results.TotalReturn
														bestResults = results
														bestConfig = testCfg
														if cfg.Verbose {
															fmt.Printf("best-so-far: %.2f%% | %s | base=%.0f max=%.2f tp=%.2f%% | %s\n",
																results.TotalReturn*100,
																strings.Join(testCfg.Indicators, ","),
																testCfg.BaseAmount,
																testCfg.MaxMultiplier,
																testCfg.TPPercent*100,
																indicatorSummary(&testCfg),
															)
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return bestResults, bestConfig
}

var (
	rand = struct {
		Float64 func() float64
	}{
		Float64: func() float64 {
			return float64(time.Now().UnixNano()%1000) / 1000
		},
	}
)

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