package orchestrator

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/bybit"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
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
	
	start := time.Now()
	
	// Log configuration summary
	r.logBacktestConfig(cfg, data)
	
	// Create strategy with configured indicators
	strat, err := r.createStrategy(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create strategy: %w", err)
	}
	
	// Create and run backtest engine
	tp := cfg.TPPercent
	if !cfg.Cycle {
		tp = 0
	}
	
	engine := backtest.NewBacktestEngine(cfg.InitialBalance, cfg.Commission, strat, tp, cfg.MinOrderQty, cfg.UseTPLevels)
	results := engine.Run(data, cfg.WindowSize)
	
	// Update all metrics
	results.UpdateMetrics()
	
	log.Printf("⏱️ Backtest completed in: %s", time.Since(start).Truncate(time.Millisecond))
	
	return results, nil
}

// RunWithFile executes a backtest by loading data from file
func (r *DefaultBacktestRunner) RunWithFile(cfg *config.DCAConfig, selectedPeriod time.Duration) (*backtest.BacktestResults, error) {
	// Fetch minimum order quantity from exchange if needed
	if err := r.fetchAndSetMinOrderQty(cfg); err != nil {
		log.Printf("⚠️ Could not fetch minimum order quantity: %v", err)
		log.Printf("ℹ️ Using default minimum order quantity: %.6f", cfg.MinOrderQty)
	}
	
	// Load historical data
	data, err := datamanager.LoadHistoricalDataCached(cfg.DataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
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

// fetchAndSetMinOrderQty fetches minimum order quantity from exchange
func (r *DefaultBacktestRunner) fetchAndSetMinOrderQty(cfg *config.DCAConfig) error {
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

// createStrategy creates a strategy based on configuration
func (r *DefaultBacktestRunner) createStrategy(cfg *config.DCAConfig) (strategy.Strategy, error) {
	// Build Enhanced DCA strategy
	dca := strategy.NewEnhancedDCAStrategy(cfg.BaseAmount)

	// Set price threshold for DCA entry spacing
	dca.SetPriceThreshold(cfg.PriceThreshold)

	// Indicator inclusion map
	include := make(map[string]bool)
	for _, name := range cfg.Indicators {
		include[strings.ToLower(strings.TrimSpace(name))] = true
	}

	if cfg.UseAdvancedCombo {
		// Advanced combo indicators
		if include["hull_ma"] {
			hullMA := indicators.NewHullMA(cfg.HullMAPeriod)
			dca.AddIndicator(hullMA)
		}
		if include["mfi"] {
			mfi := indicators.NewMFIWithPeriod(cfg.MFIPeriod)
			mfi.SetOversold(cfg.MFIOversold)
			mfi.SetOverbought(cfg.MFIOverbought)
			dca.AddIndicator(mfi)
		}
		if include["keltner"] {
			keltner := indicators.NewKeltnerChannelsCustom(cfg.KeltnerPeriod, cfg.KeltnerMultiplier)
			dca.AddIndicator(keltner)
		}
		if include["wavetrend"] {
			wavetrend := indicators.NewWaveTrendCustom(cfg.WaveTrendN1, cfg.WaveTrendN2)
			wavetrend.SetOverbought(cfg.WaveTrendOverbought)
			wavetrend.SetOversold(cfg.WaveTrendOversold)
			dca.AddIndicator(wavetrend)
		}
	} else {
		// Classic combo indicators
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
			bb := indicators.NewBollingerBandsEMA(cfg.BBPeriod, cfg.BBStdDev)
			dca.AddIndicator(bb)
		}
		if include["ema"] {
			ema := indicators.NewEMA(cfg.EMAPeriod)
			dca.AddIndicator(ema)
		}
	}

	return dca, nil
}

// logBacktestConfig logs the backtest configuration
func (r *DefaultBacktestRunner) logBacktestConfig(cfg *config.DCAConfig, data []types.OHLCV) {
	// Combo information
	comboType := "CLASSIC COMBO: RSI + MACD + Bollinger Bands + EMA"
	if cfg.UseAdvancedCombo {
		comboType = "ADVANCED COMBO: Hull MA + MFI + Keltner + WaveTrend"
	}
	log.Printf("🎯 COMBO: %s", comboType)
	log.Printf("⚙️ Params: base=$%.0f, maxMult=%.2f, window=%d, commission=%.4f, minQty=%.6f",
		cfg.BaseAmount, cfg.MaxMultiplier, cfg.WindowSize, cfg.Commission, cfg.MinOrderQty)
	
	// DCA Strategy details
	if cfg.Cycle {
		log.Printf("🔄 DCA Strategy: TP=%.2f%%, PriceThreshold=%.2f%% (cycle mode)", 
			cfg.TPPercent*100, cfg.PriceThreshold*100)
	} else {
		log.Printf("🔄 DCA Strategy: Hold mode (no TP), PriceThreshold=%.2f%%", 
			cfg.PriceThreshold*100)
	}
}

// loadDataForValidation is a helper function for loading data
func loadDataForValidation(dataFile string, selectedPeriod time.Duration) ([]types.OHLCV, error) {
	data, err := datamanager.LoadHistoricalDataCached(dataFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}
	
	if selectedPeriod > 0 {
		data = datamanager.FilterDataByPeriod(data, selectedPeriod)
	}
	
	return data, nil
}
