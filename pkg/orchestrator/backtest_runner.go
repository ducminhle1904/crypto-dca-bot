package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/bybit"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/bands"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/common"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/oscillators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/trend"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/volume"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy/spacing"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	datamanager "github.com/ducminhle1904/crypto-dca-bot/pkg/data"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DefaultBacktestRunner implements the BacktestRunner interface
type DefaultBacktestRunner struct{}

// NewDefaultBacktestRunner creates a new default backtest runner
func NewDefaultBacktestRunner() BacktestRunner {
	return &DefaultBacktestRunner{}
}

// RunWithData executes a backtest with provided data
func (r *DefaultBacktestRunner) RunWithData(cfg *config.DCAConfig, data []types.OHLCV) (*backtest.BacktestResults, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}
	
	// Log configuration summary
	r.logBacktestConfig(cfg, data)
	
	// Create strategy with configured indicators
	strat, err := r.createStrategy(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create strategy: %w", err)
	}
	
	// Reset strategy state to prevent contamination from previous runs
	// This is crucial for walk-forward validation accuracy
	strat.ResetForNewPeriod()
	
	// Create and run backtest engine
	tp := cfg.TPPercent
	if !cfg.Cycle {
		tp = 0
	}
	
	engine := backtest.NewBacktestEngine(cfg.InitialBalance, cfg.Commission, strat, tp, cfg.MinOrderQty, cfg.UseTPLevels)
	results := engine.Run(data, cfg.WindowSize)
	
	// Update all metrics
	results.UpdateMetrics()
	
	return results, nil
}

// RunWithFile executes a backtest by loading data from file
func (r *DefaultBacktestRunner) RunWithFile(cfg *config.DCAConfig, selectedPeriod time.Duration) (*backtest.BacktestResults, error) {
	// Only fetch minimum order quantity if not already set (preserve optimized values)
	// This prevents API overrides when using saved configs from optimization
	if cfg.MinOrderQty <= 0.000001 { // Use small threshold to account for floating point precision
		if err := r.FetchAndSetMinOrderQty(cfg); err != nil {
			log.Printf("⚠️ Could not fetch minimum order quantity: %v", err)
			// Set a sensible default
			cfg.MinOrderQty = 0.001
			log.Printf("ℹ️ Using default minimum order quantity: %.6f", cfg.MinOrderQty)
		}
	} else {
		log.Printf("✅ Using configured minimum order quantity: %.6f %s", cfg.MinOrderQty, cfg.Symbol)
	}
	
	// Validate data file exists before attempting to load
	if err := validateDataFile(cfg.DataFile); err != nil {
		return nil, err
	}
	
	// Load historical data
	data, err := datamanager.LoadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load data from '%s': %w", cfg.DataFile, err)
	}
	
	if len(data) == 0 {
		return nil, fmt.Errorf("no valid data found in file: %s", cfg.DataFile)
	}
	
	// Apply trailing period filter if set
	if selectedPeriod > 0 {
		data = datamanager.FilterDataByPeriod(data, selectedPeriod)
		if len(data) == 0 {
			return nil, fmt.Errorf("no data remaining after applying period filter of %v", selectedPeriod)
		}
		log.Printf("ℹ️ Filtered to last %v of data (%s → %s)",
			selectedPeriod,
			data[0].Timestamp.Format("2006-01-02"),
			data[len(data)-1].Timestamp.Format("2006-01-02"))
	}
	
	return r.RunWithData(cfg, data)
}

// FetchAndSetMinOrderQty fetches minimum order quantity from exchange
func (r *DefaultBacktestRunner) FetchAndSetMinOrderQty(cfg *config.DCAConfig) error {
	// Create Bybit client to fetch instrument info
	bybitConfig := bybit.Config{
		APIKey:    os.Getenv("BYBIT_API_KEY"),
		APISecret: os.Getenv("BYBIT_API_SECRET"),
		Demo:      true, // Use demo mode for fetching instrument info (safer)
	}

	// Skip if no API credentials (use default)
	if bybitConfig.APIKey == "" || bybitConfig.APISecret == "" {
		return fmt.Errorf("no Bybit API credentials found")
	}

	bybitClient := bybit.NewClient(bybitConfig)
	
	// Determine category from data file path
	category := "linear" // Default
	if strings.Contains(cfg.DataFile, "spot") {
		category = "spot"
	} else if strings.Contains(cfg.DataFile, "inverse") {
		category = "inverse"
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

// createStrategy creates and configures a DCA strategy from the provided configuration
func (r *DefaultBacktestRunner) createStrategy(cfg *config.DCAConfig) (strategy.Strategy, error) {
	// Initialize Enhanced DCA strategy with base trading amount
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)
	dca.SetMaxMultiplier(cfg.MaxMultiplier)

	// Configure DCA spacing strategy (required for all configurations)
	if cfg.DCASpacing == nil {
		return nil, fmt.Errorf("no DCA spacing strategy configured - please specify dca_spacing in your configuration")
	}

	spacingConfig := spacing.SpacingConfig{
		Strategy:   cfg.DCASpacing.Strategy,
		Parameters: cfg.DCASpacing.Parameters,
	}
	
	spacingStrategy, err := spacing.CreateSpacingStrategy(spacingConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create spacing strategy: %w", err)
	}
	
	if err := spacingStrategy.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid spacing strategy configuration: %w", err)
	}
	
	dca.SetSpacingStrategy(spacingStrategy)

	// Configure dynamic take profit if specified
	if cfg.DynamicTP != nil {
		dca.SetDynamicTPConfig(cfg.DynamicTP)
	}

	// Indicator inclusion map
	include := make(map[string]bool)
	for _, name := range cfg.Indicators {
		include[strings.ToLower(strings.TrimSpace(name))] = true
	}

	// Instantiate indicators based on what's actually in the indicators list
	if include["rsi"] {
		rsi := oscillators.NewRSI(cfg.RSIPeriod)
		rsi.SetOversold(cfg.RSIOversold)
		rsi.SetOverbought(cfg.RSIOverbought)
		dca.AddIndicator(rsi)
	}
	if include["macd"] {
		macd := oscillators.NewMACD(cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
		dca.AddIndicator(macd)
	}
	if include["bb"] || include["bollinger"] {
		bb := bands.NewBollingerBandsEMA(cfg.BBPeriod, cfg.BBStdDev)
		dca.AddIndicator(bb)
	}
	if include["ema"] {
		ema := common.NewEMA(cfg.EMAPeriod)
		dca.AddIndicator(ema)
	}
	if include["hullma"] || include["hull_ma"] {
		hullMA := trend.NewHullMA(cfg.HullMAPeriod)
		dca.AddIndicator(hullMA)
	}
	if include["supertrend"] || include["st"] {
		supertrend := trend.NewSuperTrendWithParams(cfg.SuperTrendPeriod, cfg.SuperTrendMultiplier)
		dca.AddIndicator(supertrend)
	}
	if include["mfi"] {
		mfi := oscillators.NewMFIWithPeriod(cfg.MFIPeriod)
		mfi.SetOversold(cfg.MFIOversold)
		mfi.SetOverbought(cfg.MFIOverbought)
		dca.AddIndicator(mfi)
	}
	if include["keltner"] || include["kc"] {
		keltner := bands.NewKeltnerChannelsCustom(cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
		dca.AddIndicator(keltner)
	}
	if include["wavetrend"] || include["wt"] {
		wavetrend := oscillators.NewWaveTrendCustom(cfg.WaveTrendN1, cfg.WaveTrendN2)
		wavetrend.SetOverbought(cfg.WaveTrendOverbought)
		wavetrend.SetOversold(cfg.WaveTrendOversold)
		dca.AddIndicator(wavetrend)
	}
	if include["obv"] {
		obv := volume.NewOBV()
		dca.AddIndicator(obv)
	}
	if include["stochrsi"] || include["stochastic_rsi"] || include["stoch_rsi"] {
		stochRSI := oscillators.NewStochasticRSI()
		dca.AddIndicator(stochRSI)
	}

	return dca, nil
}

// getMinTPPercent extracts minimum TP percentage from dynamic TP config
func getMinTPPercent(cfg *config.DynamicTPConfig) float64 {
	if cfg.VolatilityConfig != nil {
		return cfg.VolatilityConfig.MinTPPercent
	}
	if cfg.IndicatorConfig != nil {
		return cfg.IndicatorConfig.MinTPPercent
	}
	return cfg.BaseTPPercent // Fallback
}

// getMaxTPPercent extracts maximum TP percentage from dynamic TP config
func getMaxTPPercent(cfg *config.DynamicTPConfig) float64 {
	if cfg.VolatilityConfig != nil {
		return cfg.VolatilityConfig.MaxTPPercent
	}
	if cfg.IndicatorConfig != nil {
		return cfg.IndicatorConfig.MaxTPPercent
	}
	return cfg.BaseTPPercent // Fallback
}

// logBacktestConfig logs the backtest configuration
func (r *DefaultBacktestRunner) logBacktestConfig(cfg *config.DCAConfig, data []types.OHLCV) {
	// Indicator information - use actual indicators
	indicatorDescription := r.getIndicatorDescription(cfg.Indicators)
	log.Printf("🎯 INDICATORS: %s", indicatorDescription)
	log.Printf("⚙️ Params: base=$%.0f, maxMult=%.2f, window=%d, commission=%.4f, minQty=%.6f, TP=%.2f%%",
		cfg.BaseAmount, cfg.MaxMultiplier, cfg.WindowSize, cfg.Commission, cfg.MinOrderQty, cfg.TPPercent*100)
}

// getIndicatorDescription returns a description based on the actual indicators used
func (r *DefaultBacktestRunner) getIndicatorDescription(indicators []string) string {
	if len(indicators) == 0 {
		return "No indicators"
	}
	
	// Format the indicators nicely
	var formattedIndicators []string
	for _, ind := range indicators {
		switch strings.ToLower(ind) {
		case "rsi":
			formattedIndicators = append(formattedIndicators, "RSI")
		case "macd":
			formattedIndicators = append(formattedIndicators, "MACD")
		case "bb", "bollinger":
			formattedIndicators = append(formattedIndicators, "Bollinger Bands")
		case "ema":
			formattedIndicators = append(formattedIndicators, "EMA")
		case "hullma", "hull_ma":
			formattedIndicators = append(formattedIndicators, "Hull MA")
		case "supertrend", "st":
			formattedIndicators = append(formattedIndicators, "SuperTrend")
		case "mfi":
			formattedIndicators = append(formattedIndicators, "MFI")
		case "keltner", "kc":
			formattedIndicators = append(formattedIndicators, "Keltner")
		case "wavetrend", "wt":
			formattedIndicators = append(formattedIndicators, "WaveTrend")
		default:
			formattedIndicators = append(formattedIndicators, strings.ToUpper(ind))
		}
	}
	
	return strings.Join(formattedIndicators, " + ")
}

// validateDataFile validates that the data file exists and is accessible
func validateDataFile(dataFile string) error {
	if strings.TrimSpace(dataFile) == "" {
		return fmt.Errorf("data file path is empty")
	}
	
	// Get absolute path for better error reporting
	absPath, err := filepath.Abs(dataFile)
	if err != nil {
		absPath = dataFile // fallback to original if absolute path fails
	}
	
	// Check if file exists
	if _, err := os.Stat(dataFile); err != nil {
		if os.IsNotExist(err) {
			// Get current working directory for context
			wd := "unknown"
			if currentWd, wdErr := os.Getwd(); wdErr == nil {
				wd = currentWd
			}
			
			return fmt.Errorf("data file not found: %s\n"+
				"  Absolute path: %s\n"+
				"  Current working directory: %s\n"+
				"  💡 Hint: Check if the file path in your config is correct", 
				dataFile, absPath, wd)
		}
		return fmt.Errorf("cannot access data file '%s': %w", absPath, err)
	}
	
	return nil
}
