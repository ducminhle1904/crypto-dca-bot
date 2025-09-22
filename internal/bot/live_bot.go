package bot

import (
	"context"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"sync"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/adapters"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/bands"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/common"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/oscillators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/trend"
	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/volume"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/recovery"
	"github.com/ducminhle1904/crypto-dca-bot/internal/safety"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy/spacing"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// TPOrderInfo holds information about a take profit order
type TPOrderInfo struct {
	Level       int     `json:"level"`        // TP level (1-5)
	Percent     float64 `json:"percent"`      // TP percentage (e.g., 0.004 for 0.4%)
	Quantity    string  `json:"quantity"`     // Order quantity
	Price       string  `json:"price"`        // TP price
	OrderID     string  `json:"order_id"`     // Exchange order ID
	Filled      bool    `json:"filled"`       // Whether this level was filled
	FilledQty   string  `json:"filled_qty"`   // Actual filled quantity
}

// LiveBot represents the live trading bot with exchange interface support
type LiveBot struct {
	config   *config.LiveBotConfig
	exchange exchange.LiveTradingExchange
	strategy *strategy.EnhancedDCAStrategy
	spacingStrategy spacing.DCASpacingStrategy
	logger   *logger.Logger
	
	// Trading parameters extracted from config
	symbol   string
	interval string
	category string
	
	// Bot control
	running  bool
	stopChan chan struct{}
	
	// Trading state - exchange agnostic
	currentPosition float64
	averagePrice    float64
	totalInvested   float64
	balance         float64
	dcaLevel        int
	
	// TP order management - multi-level support
	activeTPOrders map[string]*TPOrderInfo // orderID -> TP order info mapping
	filledTPOrders map[string]*TPOrderInfo // orderID -> filled TP order info mapping
	tpOrderMutex   sync.RWMutex            // Protect TP order map access
	
	// Position synchronization
	positionMutex  sync.RWMutex      // Protect position data access
	
	// Debugging counters
	holdLogCounter int               // Counter for HOLD decision logging
	
	// Safety infrastructure
	validator          *safety.Validator                  // Input validation
	recoveryHandler    *recovery.RecoveryHandler         // Error recovery with backoff
	circuitBreakers    *safety.CircuitBreakerManager     // Circuit breakers for resilience
	rateLimiters       *safety.RateLimiterManager        // Rate limiting for API calls
}

// NewLiveBot creates a new live trading bot instance
func NewLiveBot(config *config.LiveBotConfig) (*LiveBot, error) {
	if config == nil {
		return nil, fmt.Errorf("bot configuration is required")
	}

	// Create exchange instance using factory
	factory := adapters.NewFactory()
	exchangeInstance, err := factory.CreateExchange(config.Exchange)
	if err != nil {
		return nil, fmt.Errorf("failed to create exchange: %w", err)
	}

	// Extract trading parameters
	symbol := config.Strategy.Symbol
	interval := config.Strategy.Interval
	category := config.Strategy.Category



	// Initialize file logger with debug mode (can be controlled via environment variable)
	debugMode := os.Getenv("DCA_BOT_DEBUG") == "true"
	fileLogger, err := logger.NewLoggerWithDebug(symbol, interval, debugMode)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	bot := &LiveBot{
		config:   config,
		exchange: exchangeInstance,
		strategy: nil, // Will be initialized below
		logger:   fileLogger,
		symbol:   symbol,
		interval: interval,
		category: category,
		balance:  config.Risk.InitialBalance,
		stopChan: make(chan struct{}),
		activeTPOrders: make(map[string]*TPOrderInfo),
		filledTPOrders: make(map[string]*TPOrderInfo),
		tpOrderMutex: sync.RWMutex{},
		
		// Initialize safety infrastructure
		validator:       safety.NewValidator(),
		recoveryHandler: recovery.NewRecoveryHandler(fileLogger),
		circuitBreakers: safety.NewCircuitBreakerManager(),
		rateLimiters:    safety.NewRateLimiterManager(),
	}

	// Initialize strategy
	if err := bot.initializeStrategy(); err != nil {
		fileLogger.Close()
		return nil, fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// Initialize circuit breakers and rate limiters for different exchange operations
	bot.initializeCircuitBreakers()
	bot.initializeRateLimiters()

	return bot, nil
}

// initializeCircuitBreakers sets up circuit breakers for different exchange operations
func (bot *LiveBot) initializeCircuitBreakers() {
	// Circuit breaker for trading operations (stricter)
	tradingConfig := safety.CircuitBreakerConfig{
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          2 * time.Minute,
		MaxFailures:      5,
		ResetTimeout:     5 * time.Minute,
	}
	
	// Circuit breaker for data operations (more lenient)
	dataConfig := safety.CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          1 * time.Minute,
		MaxFailures:      10,
		ResetTimeout:     3 * time.Minute,
	}
	
	// Create circuit breakers
	bot.circuitBreakers.GetOrCreate("trading", tradingConfig)
	bot.circuitBreakers.GetOrCreate("market_data", dataConfig)
	bot.circuitBreakers.GetOrCreate("account_data", dataConfig)
	
	// Set up state change callbacks for monitoring
	if tradingCB, exists := bot.circuitBreakers.Get("trading"); exists {
		tradingCB.SetStateChangeCallback(func(from, to safety.CircuitBreakerState) {
			bot.logger.LogWarning("Circuit Breaker", "Trading circuit breaker state changed: %s -> %s", from, to)
		})
	}
}

// initializeRateLimiters sets up rate limiters for different exchange operations
func (bot *LiveBot) initializeRateLimiters() {
	// Rate limiter for trading operations (more restrictive)
	// Most exchanges allow 10-20 orders per second
	bot.rateLimiters.GetOrCreate("trading", 10, 10) // 10 capacity, 10 per second refill
	
	// Rate limiter for market data (more lenient)
	// Market data usually has higher limits
	bot.rateLimiters.GetOrCreate("market_data", 50, 50) // 50 capacity, 50 per second refill
	
	// Rate limiter for account data (moderate)
	bot.rateLimiters.GetOrCreate("account_data", 20, 20) // 20 capacity, 20 per second refill
	
	bot.logger.Info("🚦 Rate limiters initialized for trading, market data, and account operations")
}

// Protected exchange operations with circuit breakers and rate limiting

// protectedPlaceMarketOrder places a market order with rate limiting and circuit breaker protection
func (bot *LiveBot) protectedPlaceMarketOrder(ctx context.Context, params exchange.OrderParams) (*exchange.Order, error) {
	// Apply rate limiting first
	tradingRL, _ := bot.rateLimiters.Get("trading")
	if err := tradingRL.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiting failed: %w", err)
	}
	
	// Then apply circuit breaker protection
	tradingCB, _ := bot.circuitBreakers.Get("trading")
	
	var result *exchange.Order
	err := tradingCB.Call(func() error {
		var orderErr error
		result, orderErr = bot.exchange.PlaceMarketOrder(ctx, params)
		return orderErr
	})
	
	return result, err
}

// protectedPlaceLimitOrder places a limit order with rate limiting and circuit breaker protection
func (bot *LiveBot) protectedPlaceLimitOrder(ctx context.Context, params exchange.OrderParams) (*exchange.Order, error) {
	// Apply rate limiting first
	tradingRL, _ := bot.rateLimiters.Get("trading")
	if err := tradingRL.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiting failed: %w", err)
	}
	
	// Then apply circuit breaker protection
	tradingCB, _ := bot.circuitBreakers.Get("trading")
	
	var result *exchange.Order
	err := tradingCB.Call(func() error {
		var orderErr error
		result, orderErr = bot.exchange.PlaceLimitOrder(ctx, params)
		return orderErr
	})
	
	return result, err
}

// protectedGetLatestPrice gets latest price with rate limiting and circuit breaker protection
func (bot *LiveBot) protectedGetLatestPrice(ctx context.Context, symbol string) (float64, error) {
	// Apply rate limiting first
	marketDataRL, _ := bot.rateLimiters.Get("market_data")
	if err := marketDataRL.Wait(ctx); err != nil {
		return 0, fmt.Errorf("rate limiting failed: %w", err)
	}
	
	// Then apply circuit breaker protection
	marketDataCB, _ := bot.circuitBreakers.Get("market_data")
	
	var result float64
	err := marketDataCB.Call(func() error {
		var priceErr error
		result, priceErr = bot.exchange.GetLatestPrice(ctx, symbol)
		return priceErr
	})
	
	return result, err
}

// protectedGetPositions gets positions with rate limiting and circuit breaker protection
func (bot *LiveBot) protectedGetPositions(ctx context.Context, category, symbol string) ([]exchange.Position, error) {
	// Apply rate limiting first
	accountDataRL, _ := bot.rateLimiters.Get("account_data")
	if err := accountDataRL.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiting failed: %w", err)
	}
	
	// Then apply circuit breaker protection
	accountDataCB, _ := bot.circuitBreakers.Get("account_data")
	
	var result []exchange.Position
	err := accountDataCB.Call(func() error {
		var posErr error
		result, posErr = bot.exchange.GetPositions(ctx, category, symbol)
		return posErr
	})
	
	return result, err
}

// calculateRequiredThreshold calculates the price drop threshold required for the current DCA level using spacing strategy
func (bot *LiveBot) calculateRequiredThreshold(currentPrice float64, recentCandles []types.OHLCV) float64 {
	if bot.spacingStrategy == nil {
		// Fallback to default if no spacing strategy (shouldn't happen)
		bot.logger.LogWarning("Threshold Calculation", "No spacing strategy available, using fallback")
		return 0.02 // 2% fallback
	}
	
	// Get DCA level and last entry price safely with mutex protection
	bot.positionMutex.RLock()
	currentDCALevel := bot.dcaLevel
	lastEntryPrice := bot.averagePrice
	bot.positionMutex.RUnlock()
	
	// Create market context for spacing strategy
	context := &spacing.MarketContext{
		CurrentPrice:   currentPrice,
		LastEntryPrice: lastEntryPrice,
		ATR:           0, // Will be calculated by strategy if needed
		CurrentCandle: types.OHLCV{}, // Will be set if we have current candle
		RecentCandles: recentCandles,
		Timestamp:     time.Now(),
	}
	
	// Set current candle if we have recent data
	if len(recentCandles) > 0 {
		context.CurrentCandle = recentCandles[len(recentCandles)-1]
	}
	
	// Calculate threshold using spacing strategy
	threshold := bot.spacingStrategy.CalculateThreshold(currentDCALevel, context)
	
	return threshold
}

// getCurrentTPOrderCount gets the current number of active TP orders from the exchange
func (bot *LiveBot) getCurrentTPOrderCount() int {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("TP Count", "Failed to get current orders: %v", err)
		// Fallback to memory tracking
		bot.tpOrderMutex.RLock()
		count := len(bot.activeTPOrders)
		bot.tpOrderMutex.RUnlock()
		return count
	}
	
	// Get current average price for TP validation
	positions, posErr := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	var currentAvgPrice float64 = 0
	if posErr == nil {
		for _, pos := range positions {
			if pos.Symbol == bot.symbol && pos.Side == "Buy" {
				if avgPrice, err := parseFloat(pos.AvgPrice); err == nil {
					currentAvgPrice = avgPrice
				}
				break
			}
		}
	}
	
	// Count TP orders on exchange
	tpCount := 0
	for _, order := range orders {
		if bot.isTPOrder(order, currentAvgPrice) {
			tpCount++
		}
	}
	
	return tpCount
}

// syncStrategyState synchronizes the strategy's internal state with the bot's current state
func (bot *LiveBot) syncStrategyState() {
	bot.positionMutex.RLock()
	currentDCALevel := bot.dcaLevel
	avgPrice := bot.averagePrice
	currentPosition := bot.currentPosition
	bot.positionMutex.RUnlock()
	
	if currentPosition > 0 && avgPrice > 0 {
		// Active position - sync strategy state with bot state
		bot.strategy.SetDCALevel(currentDCALevel)
		bot.strategy.SetLastEntryPrice(avgPrice)
	} else {
		// No position - reset strategy state completely
		bot.strategy.OnCycleComplete()
		// Clear filled TP orders tracking for fresh cycle
		bot.clearFilledTPOrders()
	}
}

// Start initializes and starts the bot
func (bot *LiveBot) Start() error {
	bot.running = true

	// Connect to exchange
	ctx := context.Background()
	if err := bot.exchange.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to exchange: %w", err)
	}

	// Sync with real account balance if possible
	if err := bot.syncAccountBalance(); err != nil {
		bot.logger.LogWarning("Could not sync account balance", "Using config balance: $%.2f. Check API key permissions", bot.balance)
		// Still show critical info on console
		fmt.Printf("⚠️ Using config balance: $%.2f (see log for details)\n", bot.balance)
	}

	// Check for existing position and sync bot state
	if err := bot.syncExistingPosition(); err != nil {
		bot.logger.LogWarning("Could not sync existing position", "%v", err)
	}
	
	// Sync existing orders on startup
	if err := bot.syncExistingOrders(); err != nil {
		bot.logger.LogWarning("Could not sync existing orders", "%v", err)
	}

	// Print startup information
	bot.printStartupInfo()
	bot.printBotConfiguration()

	// Show log file location
	fmt.Printf("📝 Trading logs: %s\n", bot.logger.GetLogPath())
	fmt.Printf("🔄 Bot is running... (trading activity logged to file)\n\n")

	// Start the main trading loop
	go bot.tradingLoop()

	return nil
}

// Stop gracefully stops the bot
func (bot *LiveBot) Stop() {
	if !bot.running {
		return // Already stopped
	}
	
	bot.running = false
	
	// Signal trading loop to stop FIRST - this prevents new trading operations
	// Use defer-recover to prevent panic if channel is already closed
	func() {
		defer func() {
			if r := recover(); r != nil {
				// Channel was already closed, which is fine
			}
		}()
		close(bot.stopChan)
	}()
	
	// Give trading loop a moment to exit gracefully
	time.Sleep(2 * time.Second)
	
	// Use timeout for cleanup operations to prevent hanging
	cleanupTimeout := 15 * time.Second
	cleanupDone := make(chan struct{})
	
	go func() {
		defer close(cleanupDone)
		
		// Cancel all active TP orders before closing positions
		fmt.Printf("🧹 Cleaning up TP orders...\n")
		if err := bot.cancelAllTPOrders(); err != nil {
			fmt.Printf("⚠️ Error canceling TP orders: %v\n", err)
			if bot.logger != nil {
				bot.logger.Error("Error canceling TP orders during shutdown: %v", err)
			}
		}
		
		// Close any open positions before stopping
		fmt.Printf("🔄 Closing open positions...\n")
		if err := bot.closeOpenPositions(); err != nil {
			fmt.Printf("⚠️ Error closing positions: %v\n", err)
			if bot.logger != nil {
				bot.logger.Error("Error closing positions during shutdown: %v", err)
			}
		}
		
		// Disconnect from exchange
		fmt.Printf("🔌 Disconnecting from exchange...\n")
		if err := bot.exchange.Disconnect(); err != nil {
			fmt.Printf("⚠️ Error disconnecting: %v\n", err)
			if bot.logger != nil {
				bot.logger.Error("Error disconnecting from exchange: %v", err)
			}
		}
		
		// Close logger
		if bot.logger != nil {
			bot.logger.Close()
		}
	}()
	
	// Wait for cleanup or timeout
	select {
	case <-cleanupDone:
		fmt.Printf("✅ Cleanup completed successfully\n")
	case <-time.After(cleanupTimeout):
		fmt.Printf("⚠️ Cleanup timed out after %v - forcing exit\n", cleanupTimeout)
	}
}

// initializeStrategy sets up the trading strategy with indicators
func (bot *LiveBot) initializeStrategy() error {
	// Create strategy
	bot.strategy = strategy.NewEnhancedDCAStrategy(bot.config.Strategy.BaseAmount)
	
	// Configure DCA spacing strategy
	if bot.config.Strategy.DCASpacing == nil {
		return fmt.Errorf("DCA spacing configuration is required")
	}

	spacingConfig := spacing.SpacingConfig{
		Strategy:   bot.config.Strategy.DCASpacing.Strategy,
		Parameters: bot.config.Strategy.DCASpacing.Parameters,
	}
	
	spacingStrategy, err := spacing.CreateSpacingStrategy(spacingConfig)
	if err != nil {
		return fmt.Errorf("failed to create spacing strategy: %w", err)
	}
	
	// Validate strategy configuration
	if err := spacingStrategy.ValidateConfig(); err != nil {
		return fmt.Errorf("invalid spacing strategy configuration: %w", err)
	}
	
	// Set the spacing strategy in both bot and enhanced strategy
	bot.spacingStrategy = spacingStrategy
	bot.strategy.SetSpacingStrategy(spacingStrategy)
	bot.logger.Info("✅ Using %s spacing strategy", spacingStrategy.GetName())

	// Configure dynamic TP if enabled
	if bot.config.Strategy.DynamicTP != nil {
		bot.strategy.SetDynamicTPConfig(bot.config.Strategy.DynamicTP)
		if bot.strategy.IsDynamicTPEnabled() {
			bot.logger.Info("✅ Dynamic TP enabled: %s strategy", bot.config.Strategy.DynamicTP.Strategy)
			bot.logger.LogDebugOnly("🔍 Dynamic TP Config: Strategy=%s, BaseTP=%.3f%%", 
				bot.config.Strategy.DynamicTP.Strategy, bot.config.Strategy.DynamicTP.BaseTPPercent*100)
			
			// Log strategy-specific parameters
			if bot.config.Strategy.DynamicTP.VolatilityConfig != nil {
				vc := bot.config.Strategy.DynamicTP.VolatilityConfig
				bot.logger.LogDebugOnly("🔍 Volatility Config: Multiplier=%.2f, MinTP=%.2f%%, MaxTP=%.2f%%, ATRPeriod=%d",
					vc.Multiplier, vc.MinTPPercent*100, vc.MaxTPPercent*100, vc.ATRPeriod)
			}
			
			if bot.config.Strategy.DynamicTP.IndicatorConfig != nil {
				ic := bot.config.Strategy.DynamicTP.IndicatorConfig
				bot.logger.LogDebugOnly("🔍 Indicator Config: StrengthMult=%.2f, MinTP=%.2f%%, MaxTP=%.2f%%, Weights=%v",
					ic.StrengthMultiplier, ic.MinTPPercent*100, ic.MaxTPPercent*100, ic.Weights)
			}
		} else {
			bot.logger.Info("🔧 Dynamic TP configured but not enabled (strategy: %s)", bot.config.Strategy.DynamicTP.Strategy)
		}
	} else {
		bot.logger.Info("🔧 Using fixed TP strategy")
	}

	// Add indicators based on configuration
	bot.logger.Info("🔧 Initializing %d indicators: %v", len(bot.config.Strategy.Indicators), bot.config.Strategy.Indicators)
	for _, indName := range bot.config.Strategy.Indicators {
		bot.logger.Info("🔧 Processing indicator: '%s'", indName)
		switch strings.ToLower(indName) {
		case "rsi":
			rsi := oscillators.NewRSI(bot.config.Strategy.RSI.Period)
			rsi.SetOversold(bot.config.Strategy.RSI.Oversold)
			rsi.SetOverbought(bot.config.Strategy.RSI.Overbought)
			bot.strategy.AddIndicator(rsi)
			bot.logger.Info("✅ RSI indicator added successfully")
		case "macd":
			macd := oscillators.NewMACD(
				bot.config.Strategy.MACD.FastPeriod,
				bot.config.Strategy.MACD.SlowPeriod,
				bot.config.Strategy.MACD.SignalPeriod,
			)
			bot.strategy.AddIndicator(macd)
			bot.logger.Info("✅ MACD indicator added successfully")
		case "bb", "bollinger":
			bb := bands.NewBollingerBands(
				bot.config.Strategy.BollingerBands.Period,
				bot.config.Strategy.BollingerBands.StdDev,
			)
			bot.strategy.AddIndicator(bb)
			bot.logger.Info("✅ Bollinger Bands indicator added successfully")
		case "ema":
			ema := common.NewEMA(bot.config.Strategy.EMA.Period)
			bot.strategy.AddIndicator(ema)
			bot.logger.Info("✅ EMA indicator added successfully")
		case "sma":
			sma := common.NewSMA(bot.config.Strategy.EMA.Period)
			bot.strategy.AddIndicator(sma)
			bot.logger.Info("✅ SMA indicator added successfully")
		case "hull_ma", "hullma":
			hullMA := trend.NewHullMA(bot.config.Strategy.HullMA.Period)
			bot.strategy.AddIndicator(hullMA)
			bot.logger.Info("✅ Hull MA indicator added successfully")
		case "mfi":
			mfi := oscillators.NewMFIWithPeriod(bot.config.Strategy.MFI.Period)
			mfi.SetOversold(bot.config.Strategy.MFI.Oversold)
			mfi.SetOverbought(bot.config.Strategy.MFI.Overbought)
			bot.strategy.AddIndicator(mfi)
			bot.logger.Info("✅ MFI indicator added successfully")
		case "keltner_channels", "keltner", "kc":
			keltner := bands.NewKeltnerChannelsCustom(
				bot.config.Strategy.Keltner.Period,
				bot.config.Strategy.Keltner.Multiplier,
			)
			bot.strategy.AddIndicator(keltner)
			bot.logger.Info("✅ Keltner Channels indicator added successfully")
		case "wavetrend", "wt":
			wavetrend := oscillators.NewWaveTrendCustom(
				bot.config.Strategy.WaveTrend.N1,
				bot.config.Strategy.WaveTrend.N2,
			)
			wavetrend.SetOverbought(bot.config.Strategy.WaveTrend.Overbought)
			wavetrend.SetOversold(bot.config.Strategy.WaveTrend.Oversold)
			bot.strategy.AddIndicator(wavetrend)
			bot.logger.Info("✅ WaveTrend indicator added successfully")
		case "supertrend", "st":
			supertrend := trend.NewSuperTrendWithParams(
				bot.config.Strategy.SuperTrend.Period,
				bot.config.Strategy.SuperTrend.Multiplier,
			)
			bot.strategy.AddIndicator(supertrend)
			bot.logger.Info("✅ SuperTrend indicator added successfully")
		case "obv":
			obv := volume.NewOBVWithThreshold(bot.config.Strategy.OBV.TrendThreshold)
			bot.strategy.AddIndicator(obv)
			bot.logger.Info("✅ OBV indicator added successfully")
		case "stochrsi", "stochastic_rsi", "stoch_rsi":
			stochRSI := oscillators.NewStochasticRSIWithThresholds(
				bot.config.Strategy.StochasticRSI.Period,
				bot.config.Strategy.StochasticRSI.Overbought,
				bot.config.Strategy.StochasticRSI.Oversold,
			)
			bot.strategy.AddIndicator(stochRSI)
			bot.logger.Info("✅ Stochastic RSI indicator added successfully")
		default:
			bot.logger.Info("❌ Unknown indicator: '%s'", indName)
		}
	}
	
	// Log final indicator count
	indicatorCount := bot.strategy.GetIndicatorCount()
	bot.logger.Info("🎯 Strategy initialization complete: %d indicators active", indicatorCount)
	
	return nil
}

// syncAccountBalance syncs bot balance with real exchange balance
func (bot *LiveBot) syncAccountBalance() error {
	ctx := context.Background()
	
	// Determine account type and currency based on exchange and category
	accountType := exchange.AccountTypeUnified
	baseCurrency := "USDT"
	
	switch bot.category {
	case "spot":
		accountType = exchange.AccountTypeUnified
		baseCurrency = "USDT"
	case "linear":
		accountType = exchange.AccountTypeUnified
		baseCurrency = "USDT"
	case "inverse":
		accountType = exchange.AccountTypeUnified
		baseCurrency = "USDT" // For simplicity, still check USDT
	}
	
	// Get real balance from exchange
	realBalance, err := bot.exchange.GetTradableBalance(ctx, accountType, baseCurrency)
	if err != nil {
		return fmt.Errorf("failed to fetch account balance: %w", err)
	}
	
	// Update bot balance
	bot.balance = realBalance
	return nil
}

// syncExistingPosition syncs bot state with any existing position
func (bot *LiveBot) syncExistingPosition() error {
	ctx := context.Background()
	
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get existing positions: %w", err)
	}
	
	// Look for our symbol position with enhanced identification
	for _, pos := range positions {
		if pos.Symbol == bot.symbol {
			positionValue, valueErr := parseFloat(pos.PositionValue)
			avgPrice, priceErr := parseFloat(pos.AvgPrice)
			posSize, sizeErr := parseFloat(pos.Size)
			
			// Skip positions with no meaningful data
			if (valueErr != nil || positionValue <= 0) && (sizeErr != nil || posSize <= 0) {
				continue
			}
			
			// Check if position has valid data regardless of side (for futures)
			if (positionValue > 0.01 || posSize > 0.001) && avgPrice > 0 && priceErr == nil {
				// Protect state modifications with mutex
				bot.positionMutex.Lock()
				bot.currentPosition = positionValue
				bot.averagePrice = avgPrice
				bot.totalInvested = positionValue
				// Estimate DCA level based on position size (more accurate than assuming 1)
				if bot.config.Strategy.BaseAmount > 0 {
					bot.dcaLevel = max(1, int(positionValue/bot.config.Strategy.BaseAmount))
				} else {
					bot.dcaLevel = 1 // Fallback to level 1
				}
				bot.positionMutex.Unlock()
				
				bot.logger.LogPositionSync(positionValue, avgPrice, pos.Size, pos.UnrealisedPnl)
				// Show brief console message
				fmt.Printf("🔄 Existing position synced: $%.2f @ $%.4f (see log for details)\n", positionValue, avgPrice)
				
				// Sync strategy state after position sync
				bot.syncStrategyState()
				
				return nil
			}
		}
	}
	
	// No existing position found - protect state reset with mutex
	bot.positionMutex.Lock()
	bot.currentPosition = 0
	bot.averagePrice = 0
	bot.totalInvested = 0
	bot.dcaLevel = 0
	bot.positionMutex.Unlock()
	fmt.Printf("✅ No existing position found - starting fresh\n")
	
	// Sync strategy state after position sync
	bot.syncStrategyState()
	
	return nil
}

// tradingLoop runs the main trading logic
func (bot *LiveBot) tradingLoop() {
	// Calculate interval duration
	intervalDuration := bot.getIntervalDuration()
	bot.logger.Info("Trading interval: %s", bot.interval)
	
	// Wait for next candle close - but make it interruptible
	waitDuration := bot.getTimeUntilNextCandle()
	bot.logger.Info("Waiting %.0f seconds for next %s candle close", waitDuration.Seconds(), bot.interval)
	
	// Use interruptible wait instead of blocking sleep
	waitTimer := time.NewTimer(waitDuration)
	defer waitTimer.Stop()
	
	select {
	case <-waitTimer.C:
		// Timer expired - continue to initial check
		bot.checkAndTrade()
	case <-bot.stopChan:
		bot.logger.Info("Stop signal received during initial wait - ending trading loop")
		return
	}

	// Create ticker for regular checks
	ticker := time.NewTicker(intervalDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Check for stop signal before processing
			if bot.shouldStop() {
				bot.logger.Info("Stop signal detected - ending trading loop")
				return
			}
			bot.checkAndTrade()
		case <-bot.stopChan:
			bot.logger.Info("Stop signal received - ending trading loop")
			return
		}
	}
}

// shouldStop checks if the bot should stop (non-blocking)
func (bot *LiveBot) shouldStop() bool {
	return !bot.running
}

// checkAndTrade performs market analysis and executes trades
func (bot *LiveBot) checkAndTrade() {
	defer func() {
		if r := recover(); r != nil {
			bot.logger.Error("Error in trading loop: %v", r)
		}
	}()

	// Check for stop signal before starting
	if bot.shouldStop() {
		return
	}

	// Use context with timeout to prevent hanging on API calls
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Refresh account balance
	if err := bot.syncAccountBalance(); err != nil {
		bot.logger.LogWarning("Could not refresh balance", "%v", err)
		// Continue despite balance sync failure
	}

	// Check for stop signal after balance sync
	if bot.shouldStop() {
		return
	}

	// Sync position data
	if err := bot.syncPositionData(); err != nil {
		if err.Error() == "STRATEGY_SYNC_REQUIRED" {
			// Position was reset, strategy sync is needed
			bot.syncStrategyState()
		} else {
			bot.logger.LogWarning("Could not sync position data", "%v", err)
		}
		// Continue despite position sync failure
	}

	// CRITICAL: Sync strategy state BEFORE making trade decisions
	// This ensures the strategy has current DCA level and last entry price
	bot.syncStrategyState()

	// Check for stop signal after position sync
	if bot.shouldStop() {
		return
	}

	// Get current market price
	currentPrice, err := bot.exchange.GetLatestPrice(ctx, bot.symbol)
	if err != nil {
		bot.logger.Error("Failed to get current price: %v", err)
		return
	}

	// Get recent klines for analysis
	klines, err := bot.getRecentKlines()
	if err != nil {
		bot.logger.Error("Failed to get recent klines: %v", err)
		return
	}

	// Use a more flexible minimum data requirement
	// Need at least 2x the longest indicator period for basic reliability
	minRequiredDataPoints := 70 // Minimum required for multi-indicator strategy
	if bot.config.Strategy.WindowSize < minRequiredDataPoints {
		minRequiredDataPoints = bot.config.Strategy.WindowSize
	}
	
	if len(klines) < minRequiredDataPoints {
		bot.logger.LogWarning("Insufficient data", "Not enough data points (%d < %d minimum required)", len(klines), minRequiredDataPoints)
		return
	}
	
	// If we have less than the configured window size but more than minimum, proceed with warning
	if len(klines) < bot.config.Strategy.WindowSize {
		bot.logger.LogWarning("Limited data", "Using %d data points (less than configured %d, but sufficient for analysis)", len(klines), bot.config.Strategy.WindowSize)
	}

	// Analyze market conditions with detailed logging
	decision, action := bot.analyzeMarket(klines, currentPrice)
	
	// Log detailed market analysis for debugging
	if decision != nil {
		// Get indicator results for detailed logging
		indicatorResults := bot.strategy.GetLastResults()
		indicatorMap := make(map[string]interface{})
		for name, result := range indicatorResults {
			if result.Error != nil {
				indicatorMap[name] = fmt.Sprintf("ERROR: %v", result.Error)
			} else {
				bias := bot.interpretIndicatorBias(name, result.Value, currentPrice)
				indicatorMap[name] = bias
			}
		}
		
		// Convert klines to interface{} for logging
		klinesInterface := make([]interface{}, len(klines))
		for i, kline := range klines {
			klinesInterface[i] = kline
		}
		
		bot.logger.LogDetailedMarketAnalysis(klinesInterface, indicatorMap, action, decision.Confidence)
		
		// Log additional debug information
		bot.logger.LogDebugOnly("Market Analysis Debug - Decision: %s, Confidence: %.2f%%, Strength: %.2f%%, Amount: $%.2f", 
			action, decision.Confidence*100, decision.Strength*100, decision.Amount)
	}

	// Check for filled TP orders and gather TP information
	filledOrders := bot.detectFilledTPOrders()
	filledTPSummary := bot.getFilledTPSummary()
	
	// Get accurate active TP count from exchange (more reliable than memory tracking)
	activeTPCount := bot.getCurrentTPOrderCount()
	
	// Notify about newly filled orders
	if len(filledOrders) > 0 {
		bot.logger.Info("🎯 TP Orders FILLED: %s", strings.Join(filledOrders, ", "))
		fmt.Printf("🎯 TP Orders Filled: %s\n", strings.Join(filledOrders, ", "))
	}
	
	// Log market status to file with TP information (get state safely with mutex protection)
	bot.positionMutex.RLock()
	safeBalance := bot.balance
	safePosition := bot.currentPosition
	safeAvgPrice := bot.averagePrice
	safeDCALevel := bot.dcaLevel
	bot.positionMutex.RUnlock()
	
	exchangePnL := bot.getExchangePnL()
	bot.logger.LogMarketStatus(currentPrice, action, safeBalance, safePosition, safeAvgPrice, safeDCALevel, exchangePnL, filledTPSummary, activeTPCount)

	// Execute trading action (logging moved to after validation checks)
	if action != "HOLD" {
		bot.executeTrade(decision, action, currentPrice)
	}
}

// getRecentKlines retrieves recent market data with timeout protection
func (bot *LiveBot) getRecentKlines() ([]types.OHLCV, error) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Start with a reasonable request size, but be prepared to accept less
	requestLimit := bot.config.Strategy.WindowSize + 50
	if requestLimit > 200 {
		requestLimit = 200 // Limit max request to avoid exchange limits
	}
	
	params := exchange.KlineParams{
		Category: bot.category,
		Symbol:   bot.symbol,
		Interval: exchange.KlineInterval(bot.interval),
		Limit:    requestLimit,
	}
	
	klines, err := bot.exchange.GetKlines(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	// If we got less data than expected, try with a smaller request
	if len(klines) < 70 && requestLimit > 100 {
		bot.logger.LogWarning("Limited klines", "Got only %d klines, retrying with smaller request", len(klines))
		params.Limit = 100
		klines, err = bot.exchange.GetKlines(ctx, params)
		if err != nil {
			return nil, fmt.Errorf("failed to get klines (retry): %w", err)
		}
	}

	// Return all available data (we'll handle the minimum check in checkAndTrade)
	return klines, nil
}

// getExchangePnL retrieves unrealized PnL from exchange
func (bot *LiveBot) getExchangePnL() string {
	// Get current position safely
	bot.positionMutex.RLock()
	currentPos := bot.currentPosition
	bot.positionMutex.RUnlock()
	
	if currentPos <= 0 {
		return ""
	}
	
	ctx := context.Background()
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return ""
	}
	
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			return pos.UnrealisedPnl
		}
	}
	
	return ""
}

// analyzeMarket performs technical analysis to determine trading action
func (bot *LiveBot) analyzeMarket(klines []types.OHLCV, currentPrice float64) (*strategy.TradeDecision, string) {
	// Use strategy to analyze market conditions
	decision, err := bot.strategy.ShouldExecuteTrade(klines)
	if err != nil {
		bot.logger.Error("Strategy error: %v", err)
		return nil, "HOLD"
	}
	
	
	// Log strategy decision reasoning for better debugging
	if decision.Action == strategy.ActionBuy {
		bot.logger.Info("🎯 BUY Signal: %s (Confidence: %.1f%%, Strength: %.1f%%)", 
			decision.Reason, decision.Confidence*100, decision.Strength*100)
		// Reset hold log counter so next HOLD decision is logged
		bot.holdLogCounter = 0
		return decision, "BUY"
	} else {
		// Log HOLD reasoning every 10th check to avoid spam, but always log immediately after cycle completion
		// Get DCA level safely for logging decision
		bot.positionMutex.RLock()
		currentDCALevel := bot.dcaLevel
		bot.positionMutex.RUnlock()
		
		if currentDCALevel == 0 || bot.holdLogCounter%10 == 0 {
			bot.logger.Info("⏸️ HOLD Decision: %s", decision.Reason)
		}
		bot.holdLogCounter++
		return decision, "HOLD"
	}

	// Take profit is handled by exchange limit orders placed after each buy
	// This provides immediate TP protection without waiting for candle closes
}

// executeTrade executes the trading action
func (bot *LiveBot) executeTrade(decision *strategy.TradeDecision, action string, price float64) {
	switch action {
	case "BUY":
		bot.executeBuy(decision, price)
	case "SELL":
		bot.executeSell(price)
	}
}

// executeBuy executes a buy order using exchange interface
func (bot *LiveBot) executeBuy(decision *strategy.TradeDecision, price float64) {
	ctx := context.Background()

	// Safety validation before executing any trades
	if result := bot.validator.ValidatePrice(price, bot.symbol); !result.Valid {
		bot.logger.LogWarning("Trade Validation", "Invalid price for buy order: %s", result.Message)
		return
	}

	// Additional safety check: Validate price threshold for DCA trades at bot level
	// Get current state safely with mutex protection
	bot.positionMutex.RLock()
	currentDCALevel := bot.dcaLevel
	currentAvgPrice := bot.averagePrice
	bot.positionMutex.RUnlock()
	
	// DCA spacing strategy validation with proper market context
	if currentDCALevel > 0 && currentAvgPrice > 0 {
		// Get recent klines for spacing strategy context
		recentCandles, err := bot.getRecentKlines()
		if err != nil {
			bot.logger.LogWarning("DCA Validation", "Failed to get recent candles for spacing calculation: %v", err)
			recentCandles = []types.OHLCV{} // Use empty slice as fallback
		}
		
		// Calculate price change: positive = price went down (good for DCA), negative = price went up (not ideal for DCA)
		priceChange := (currentAvgPrice - price) / currentAvgPrice
		requiredThreshold := bot.calculateRequiredThreshold(price, recentCandles)
		
		// Log detailed DCA spacing information
		spacingContext := map[string]interface{}{
			"dca_level": currentDCALevel,
			"price_change_percent": priceChange * 100,
			"required_threshold_percent": requiredThreshold * 100,
			"data_points": len(recentCandles),
		}
		
		// Add spacing strategy details if available
		if bot.spacingStrategy != nil {
			spacingContext["strategy_name"] = bot.spacingStrategy.GetName()
			spacingContext["strategy_params"] = bot.spacingStrategy.GetParameters()
		}
		
		bot.logger.LogDCASpacingDetails(currentDCALevel, price, currentAvgPrice, requiredThreshold, 
			bot.spacingStrategy.GetName(), spacingContext)
		
		if priceChange < requiredThreshold {
			// Determine direction for clearer messaging
			if priceChange < 0 {
				bot.logger.Info("🛡️ BOT SAFETY: Blocking DCA buy - price went UP %.2f%% from avg, need DOWN %.2f%% (DCA Level %d)", 
					-priceChange*100, requiredThreshold*100, currentDCALevel)
			} else {
				bot.logger.Info("🛡️ BOT SAFETY: Blocking DCA buy - price drop %.2f%% < required %.2f%% (DCA Level %d)", 
					priceChange*100, requiredThreshold*100, currentDCALevel)
			}
			return
		}
	}

	// Use strategy's calculated amount directly (strategy already handles DCA level scaling)
	amount := decision.Amount
	
	// Get current DCA level for logging only
	bot.positionMutex.RLock()
	currentDCALevelForLogging := bot.dcaLevel
	bot.positionMutex.RUnlock()
	
	// REMOVED: Double multiplier application to prevent 3-6x position size inflation
	// The strategy already calculates appropriate position sizes based on:
	// - Signal confidence and strength
	// - DCA level progression
	// - Max multiplier constraints
	
	// Log position sizing info
	bot.logger.Info("🧠 Strategy Position Sizing: Confidence: %.1f%%, Strength: %.1f%%, DCA Level: %d, Amount: $%.2f", 
		decision.Confidence*100, decision.Strength*100, currentDCALevelForLogging, amount)

	// Get trading constraints
	constraints, err := bot.exchange.GetTradingConstraints(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("Trading constraints", "Could not get trading constraints: %v", err)
		return
	}
	
	// Add defensive check for invalid price
	if price <= 0 {
		bot.logger.LogWarning("Order constraints", "Invalid price: %.4f", price)
		return
	}
	
	// Add defensive check for invalid amount
	if amount <= 0 {
		bot.logger.LogWarning("Order constraints", "Invalid amount: %.4f", amount)
		return
	}
	
	quantity := amount / price
	
	// Validate calculated quantity
	if result := bot.validator.ValidateQuantity(quantity, bot.symbol); !result.Valid {
		bot.logger.LogWarning("Trade Validation", "Invalid quantity for buy order: %s", result.Message)
		return
	}
	
	// Validate order value
	if result := bot.validator.ValidateOrderValue(price, quantity, bot.symbol); !result.Valid {
		bot.logger.LogWarning("Trade Validation", "Invalid order value: %s", result.Message)
		return
	}
	
	// Log detailed order placement information
	constraintsMap := map[string]interface{}{
		"min_order_qty": constraints.MinOrderQty,
		"min_order_value": constraints.MinOrderValue,
		"qty_step": constraints.QtyStep,
		"min_price_step": constraints.MinPriceStep,
	}
	
	bot.logger.LogOrderPlacementDetails("Market", "Buy", bot.symbol, quantity, price, amount, constraintsMap)

	// Apply minimum quantity constraint using floor to avoid overshooting
	if constraints.QtyStep > 0 {
		multiplier := math.Floor(quantity / constraints.QtyStep)
		if multiplier < 1 {
			multiplier = 1
		}
		adjustedQuantity := multiplier * constraints.QtyStep
		
		if adjustedQuantity != quantity {
			quantity = adjustedQuantity
			amount = quantity * price
			bot.logger.Info("📏 Quantity adjusted to step size: %.6f %s", quantity, bot.symbol)
		}
	}

	// Check minimum order constraints
	orderValue := quantity * price
	if quantity < constraints.MinOrderQty {
		bot.logger.LogWarning("Order constraints", "Quantity %.6f < min %.6f %s", quantity, constraints.MinOrderQty, bot.symbol)
		return
	}
	if orderValue < constraints.MinOrderValue {
		bot.logger.LogWarning("Order constraints", "Order value $%.2f < min $%.2f", orderValue, constraints.MinOrderValue)
		return
	}

	// Check available balance for margin (don't assume 10x leverage)
	if amount > bot.balance {
		bot.logger.LogWarning("Insufficient balance", "Balance: $%.2f < Required: $%.2f", bot.balance, amount)
		return
	}

	// Place order with timeout and retry
	orderParams := exchange.OrderParams{
		Category:  bot.category,
		Symbol:    bot.symbol,
		Side:      exchange.OrderSideBuy,
		Quantity:  fmt.Sprintf("%.6f", quantity),
		OrderType: exchange.OrderTypeMarket,
	}

	// Log execution now that all checks have passed
	if bot.exchange.IsDemo() {
		bot.logger.Trade("🧪 DEMO MODE: Executing BUY at $%.2f (paper trading)", price)
	} else {
		bot.logger.Trade("💰 LIVE MODE: Executing BUY at $%.2f (real money)", price)
	}

	order, err := bot.placeOrderWithRetry(orderParams, true) // true for market order
	if err != nil {
		bot.logger.Error("Failed to place buy order after retries: %v", err)
		
		// Log detailed error with context
		errorContext := map[string]interface{}{
			"order_type": "Market",
			"side": "Buy",
			"symbol": bot.symbol,
			"quantity": quantity,
			"price": price,
			"value": amount,
			"dca_level": currentDCALevelForLogging,
		}
		
		bot.logger.LogErrorWithContext("Order Placement", err, errorContext)
		
		// Categorize order placement errors for better debugging
		if strings.Contains(err.Error(), "timeout") {
			bot.logger.Error("Order Error Category: Timeout error - exchange may be slow or overloaded")
		} else if strings.Contains(err.Error(), "insufficient") {
			bot.logger.Error("Order Error Category: Insufficient balance or margin error")
		} else if strings.Contains(err.Error(), "constraint") {
			bot.logger.Error("Order Error Category: Order constraint violation (quantity, price, etc.)")
		} else if strings.Contains(err.Error(), "symbol") {
			bot.logger.Error("Order Error Category: Invalid symbol or trading pair error")
		} else if strings.Contains(err.Error(), "permission") {
			bot.logger.Error("Order Error Category: API permission or authentication error")
		} else {
			bot.logger.Error("Order Error Category: Unknown order placement error: %v", err)
		}
		return
	}

	// Log order placement result (execution details will be synced from exchange)
	bot.logger.Info("📤 Order placed successfully - ID: %s, syncing actual execution from exchange...", order.OrderID)

	// Sync with exchange data first to get actual executed values
	bot.syncAfterTrade(order, "BUY")
	
	// Log sync completion with safely read values
	bot.positionMutex.RLock()
	logPosition := bot.currentPosition
	logAvgPrice := bot.averagePrice
	bot.positionMutex.RUnlock()
	bot.logger.Info("✅ Sync complete - Position: %.6f, AvgPrice: %.4f", logPosition, logAvgPrice)

	// Place multi-level take profit orders for FIRST trade only (DCA level 1)
	// For DCA trades (level 2+), TP orders are updated by syncAfterTrade -> updateMultiLevelTPOrders
	// Get current state safely for TP order decision
	bot.positionMutex.RLock()
	currentDCALevelForTP := bot.dcaLevel
	currentAvgPriceForTP := bot.averagePrice
	currentPositionForTP := bot.currentPosition
	bot.positionMutex.RUnlock()

	if bot.config.Strategy.AutoTPOrders && currentDCALevelForTP <= 1 {
		// Use position data from sync instead of order response (more reliable)
		avgPrice := currentAvgPriceForTP // Updated by syncAfterTrade
		
		if currentPositionForTP > 0 && avgPrice > 0 {
			// Get current position size from exchange for accurate TP sizing
			ctx := context.Background()
			positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
			if err != nil {
				bot.logger.LogWarning("Multi-Level TP Setup", "Could not get position for TP orders: %v", err)
			} else {
				// Find our position and use its size
				for _, pos := range positions {
					if pos.Symbol == bot.symbol && pos.Side == "Buy" {
						bot.logger.Info("🎯 Setting up initial TP orders - Position Size: %s, Avg Price: $%.4f", pos.Size, avgPrice)
						
		// Place TP orders with proper error handling and categorization
		if err := bot.placeMultiLevelTPOrders(pos.Size, avgPrice); err != nil {
			bot.logger.LogWarning("Multi-Level TP Setup", "Could not place TP orders: %v", err)
			// Categorize TP order errors for better debugging
			if strings.Contains(err.Error(), "timeout") {
				bot.logger.LogWarning("TP Error Category", "Timeout error - exchange may be slow")
			} else if strings.Contains(err.Error(), "insufficient") {
				bot.logger.LogWarning("TP Error Category", "Insufficient balance or quantity error")
			} else if strings.Contains(err.Error(), "constraint") {
				bot.logger.LogWarning("TP Error Category", "Order constraint violation")
			} else {
				bot.logger.LogWarning("TP Error Category", "Unknown error type: %v", err)
			}
		}
						break
					}
				}
			}
		} else {
			bot.logger.LogWarning("Multi-Level TP Setup", "No position found after trade execution (position: %.6f, avgPrice: %.4f)", currentPositionForTP, avgPrice)
			
			// For demo mode, provide guidance on likely issue
			if bot.exchange.IsDemo() {
				bot.logger.LogWarning("Multi-Level TP Setup", "This is common in demo mode when order execution details are not properly simulated")
				bot.logger.LogWarning("Multi-Level TP Setup", "The bot will continue trading but TP orders won't be placed until a valid position is detected")
			} else {
				bot.logger.Error("Multi-Level TP Setup: No position found in live mode - this indicates a serious issue with order execution")
			}
		}
	} else if bot.config.Strategy.AutoTPOrders && currentDCALevelForTP > 1 {
		bot.logger.Info("🔄 DCA trade detected - TP orders will be updated by syncAfterTrade process")
	}
}

// placeMultiLevelTPOrders places multiple take profit limit orders after a buy
func (bot *LiveBot) placeMultiLevelTPOrders(totalQuantity string, avgEntryPrice float64) error {
	// Use timeout for placing multiple orders (reduced with rate limiting delays)
	// 5 levels * (10s max per order + 0.5s delay) = ~55s + buffer = 90s
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	
	// First, cancel any existing TP orders to prevent duplicates
	bot.tpOrderMutex.Lock()
	existingOrders := make(map[string]*TPOrderInfo)
	for orderID, tpInfo := range bot.activeTPOrders {
		existingOrders[orderID] = &TPOrderInfo{
			Level:   tpInfo.Level,
			OrderID: orderID,
		}
	}
	// Clear active orders map
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	bot.tpOrderMutex.Unlock()
	
	// Cancel existing orders without holding mutex
	for orderID, tpInfo := range existingOrders {
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("TP Cleanup", "Failed to cancel existing TP Level %d order %s: %v", tpInfo.Level, orderID, err)
		}
	}
	
	// Add a channel to handle timeout gracefully
	done := make(chan error, 1)
	
	go func() {
		done <- bot.placeMultiLevelTPOrdersInternal(ctx, totalQuantity, avgEntryPrice)
	}()
	
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		bot.logger.LogWarning("TP Placement Timeout", "TP order placement timed out after 90 seconds")
		return fmt.Errorf("TP order placement timed out")
	}
}

// placeMultiLevelTPOrdersInternal is the internal implementation
func (bot *LiveBot) placeMultiLevelTPOrdersInternal(ctx context.Context, totalQuantity string, avgEntryPrice float64) error {
	startTime := time.Now()
	
	
	// Parse total quantity
	totalQty, err := parseFloat(totalQuantity)
	if err != nil {
		return fmt.Errorf("invalid quantity: %w", err)
	}
	
	// Get trading constraints for proper rounding
	constraints, err := bot.exchange.GetTradingConstraints(ctx, bot.category, bot.symbol)
	
	if err != nil {
		bot.logger.LogWarning("TP Constraints", "Could not get trading constraints: %v", err)
		// Continue with default constraints
		constraints = &exchange.TradingConstraints{QtyStep: 0.001, MinPriceStep: 0.0001}
	} 
	
	// Pre-calculate quantities for all TP levels to ensure proper distribution
	levelQuantities := bot.calculateTPLevelQuantities(totalQty, constraints)
	
	// Validate TP levels configuration before the loop
	if bot.config.Strategy.TPLevels <= 0 {
		return fmt.Errorf("invalid TP levels configuration: %d", bot.config.Strategy.TPLevels)
	}
	
	bot.logger.Info("🎯 Placing %d-level TP orders from avg entry $%.4f: %.6f %s total", 
		bot.config.Strategy.TPLevels, avgEntryPrice, totalQty, bot.symbol)
	
	successCount := 0
	skippedLevels := 0
	
	// Place TP orders for each level using pre-calculated quantities
	for level := 1; level <= bot.config.Strategy.TPLevels; level++ {
		// Check for context cancellation and timeout safety
		select {
		case <-ctx.Done():
			bot.logger.LogWarning("TP Placement", "Context cancelled during TP level %d placement, stopping", level)
			// Return partial success if we placed some orders
			if successCount > 0 {
				bot.logger.Info("✅ Partial TP success: %d/%d levels placed before timeout", successCount, bot.config.Strategy.TPLevels)
			}
			return fmt.Errorf("context cancelled during TP placement at level %d", level)
		default:
		}
		
		// Additional safety check: if we're close to timeout, stop placing more orders  
		deadline, hasDeadline := ctx.Deadline()
		if hasDeadline && time.Until(deadline) < 20*time.Second {
			bot.logger.LogWarning("TP Placement", "Stopping at level %d - insufficient time remaining (<%ds)", level, 20)
			// Return partial success if we've placed some orders
			if successCount > 0 {
				bot.logger.Info("✅ Partial TP success due to timeout: %d/%d levels placed", successCount, bot.config.Strategy.TPLevels)
			}
			break
		}
		
		// Get pre-calculated quantity for this level with bounds checking
		if level < 1 || level > len(levelQuantities) {
			bot.logger.LogWarning("TP Level %d", "Invalid level index: %d (valid range: 1-%d)", level, level, len(levelQuantities))
			skippedLevels++
			continue
		}
		levelQty := levelQuantities[level-1] // Array is 0-indexed
		if levelQty <= 0 {
			skippedLevels++
			continue
		}
		
		// Calculate TP price for this level based on actual average entry
		
		// Calculate level percentage using dynamic TP if enabled, otherwise use fixed TP
		var levelPercent float64
		if bot.strategy.IsDynamicTPEnabled() {
			bot.logger.LogDebugOnly("🔍 Dynamic TP: Starting calculation for level %d", level)
			
			// Get recent klines for dynamic TP calculation
			recentKlines, err := bot.getRecentKlines()
			if err != nil {
				bot.logger.LogWarning("Dynamic TP", "Failed to get recent klines for dynamic TP calculation: %v, falling back to fixed TP", err)
				levelPercent = bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
				bot.logger.LogDebugOnly("🔍 Dynamic TP: Fallback to fixed TP Level %d: %.3f%%", level, levelPercent*100)
			} else {
				bot.logger.LogDebugOnly("🔍 Dynamic TP: Got %d klines for calculation", len(recentKlines))
				
				// Calculate dynamic TP percentage
				currentCandle := types.OHLCV{Close: avgEntryPrice} // Use avg price as current price for TP calculation
				if len(recentKlines) > 0 {
					currentCandle = recentKlines[len(recentKlines)-1] // Use latest candle
					bot.logger.LogDebugOnly("🔍 Dynamic TP: Using latest candle - Close: $%.2f", currentCandle.Close)
				} else {
					bot.logger.LogDebugOnly("🔍 Dynamic TP: Using avg entry price as current candle - Close: $%.2f", avgEntryPrice)
				}
				
				dynamicTPPercent, err := bot.strategy.GetDynamicTPPercent(currentCandle, recentKlines)
				if err != nil {
					bot.logger.LogWarning("Dynamic TP", "Dynamic TP calculation failed: %v, falling back to fixed TP", err)
					levelPercent = bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
					bot.logger.LogDebugOnly("🔍 Dynamic TP: Fallback to fixed TP Level %d: %.3f%%", level, levelPercent*100)
				} else {
					// Calculate fixed TP for comparison
					fixedTPPercent := bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
					
					// Scale the dynamic TP percentage by level
					levelPercent = dynamicTPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
					
					bot.logger.Info("🎯 Dynamic TP Level %d: %.3f%% (base dynamic: %.3f%%, fixed would be: %.3f%%)", 
						level, levelPercent*100, dynamicTPPercent*100, fixedTPPercent*100)
					bot.logger.LogDebugOnly("🔍 Dynamic TP: Level scaling - Level %d/%d = %.3fx multiplier", 
						level, bot.config.Strategy.TPLevels, float64(level)/float64(bot.config.Strategy.TPLevels))
				}
			}
		} else {
			// Use fixed TP calculation
			levelPercent = bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
			bot.logger.LogDebugOnly("🔍 Fixed TP Level %d: %.3f%% (base: %.3f%%)", 
				level, levelPercent*100, bot.config.Strategy.TPPercent*100)
		}
		
		tpPrice := avgEntryPrice * (1 + levelPercent)
		
		// Log the calculated TP price before rounding
		bot.logger.LogDebugOnly("🔍 TP Level %d: Calculated price $%.4f (entry: $%.4f + %.3f%%)", 
			level, tpPrice, avgEntryPrice, levelPercent*100)
		
		// Round price to exchange tick size
		if constraints.MinPriceStep > 0 {
			originalPrice := tpPrice
			tpPrice = math.Round(tpPrice/constraints.MinPriceStep) * constraints.MinPriceStep
			if originalPrice != tpPrice {
				bot.logger.LogDebugOnly("🔍 TP Level %d: Price rounded from $%.4f to $%.4f (tick size: $%.4f)", 
					level, originalPrice, tpPrice, constraints.MinPriceStep)
			}
		}
		
		// Validate order constraints (should already be satisfied by pre-calculation)
		orderValue := levelQty * tpPrice
		if levelQty < constraints.MinOrderQty || orderValue < constraints.MinOrderValue {
			bot.logger.LogWarning("TP Level %d", "Constraint violation: qty=%.6f, value=$%.2f, skipping", level, levelQty, orderValue)
			skippedLevels++
			continue
		}
		
		// Format values with proper precision
		formattedQty := fmt.Sprintf("%.6f", levelQty)
		formattedPrice := fmt.Sprintf("%.4f", tpPrice)
		
		// Place TP limit order with timeout
		orderParams := exchange.OrderParams{
			Category:  bot.category,
			Symbol:    bot.symbol,
			Side:      exchange.OrderSideSell,
			Quantity:  formattedQty,
			OrderType: exchange.OrderTypeLimit,
			Price:     formattedPrice,
		}
		
		// Place TP limit order with timing
		orderStartTime := time.Now()
		tpOrder, err := bot.placeOrderWithRetry(orderParams, false) // false for limit order
		orderDuration := time.Since(orderStartTime)
		
		if err != nil {
			bot.logger.LogWarning("TP Level %d", "Failed to place TP order after %v: %v", level, orderDuration, err)
			skippedLevels++
			continue
		}
		
		// Track the TP order (using proper defer pattern for safety)
		func() {
			bot.tpOrderMutex.Lock()
			defer bot.tpOrderMutex.Unlock()
			bot.activeTPOrders[tpOrder.OrderID] = &TPOrderInfo{
				Level:     level,
				Percent:   levelPercent,
				Quantity:  formattedQty,
				Price:     formattedPrice,
				OrderID:   tpOrder.OrderID,
				Filled:    false,
				FilledQty: "0",
			}
		}()
		
		// Log detailed TP order information
		quantityFloat, _ := parseFloat(formattedQty)
		priceFloat, _ := parseFloat(formattedPrice)
		bot.logger.LogTPOrderDetails(level, quantityFloat, priceFloat, levelPercent, tpOrder.OrderID, "PLACED")
		
		bot.logger.Info("✅ TP Level %d placed: %s %s at $%s (%.3f%%)", 
			level, formattedQty, bot.symbol, formattedPrice, levelPercent*100)
		
		// Debug log order placement details
		bot.logger.LogDebugOnly("🔍 TP Order Placed: ID=%s, Level=%d, Qty=%s, Price=$%s, Value=$%.2f", 
			tpOrder.OrderID, level, formattedQty, formattedPrice, orderValue)
		
		// Track successful placement
		successCount++
		
		// Add rate limiting delay between orders (except for the last one)
		if level < bot.config.Strategy.TPLevels {
			delayMs := 500 // 500ms delay between orders to avoid rate limiting
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}
	
	if skippedLevels > 0 {
		bot.logger.LogWarning("TP Placement", "%d levels skipped due to constraints", skippedLevels)
	}
	
	// Add comprehensive completion logging
	totalDuration := time.Since(startTime)
	
	// Log summary to console
	if successCount > 0 {
		// Calculate total allocated quantity from pre-calculated levels
		var allocatedQty float64
		for _, qty := range levelQuantities {
			allocatedQty += qty
		}
		fmt.Printf("🎯 Multi-Level TP: %d/%d orders placed successfully (%.6f %s allocated, %.1f%%) in %v\n", 
			successCount, bot.config.Strategy.TPLevels, allocatedQty, bot.symbol, (allocatedQty/totalQty)*100, totalDuration)
		
		if skippedLevels > 0 {
			fmt.Printf("⚠️  %d TP levels skipped due to constraints\n", skippedLevels)
		}
		
		return nil // Don't fail if we placed some orders

	} else {
		return fmt.Errorf("failed to place any TP orders")
	}
}

// updateMultiLevelTPOrders updates existing multi-level TP orders when average price changes
func (bot *LiveBot) updateMultiLevelTPOrders(newAveragePrice float64) error {
	bot.logger.Info("🔄 Updating TP orders for new average price $%.4f", newAveragePrice)
	
	// First, collect order information and clear maps while holding mutex
	bot.tpOrderMutex.Lock()
	memoryOrderCount := len(bot.activeTPOrders)
	
	if memoryOrderCount == 0 {
		bot.tpOrderMutex.Unlock()
		bot.logger.Info("✅ No TP orders in memory to update")
		return nil // No TP orders to update
	}
	
	// Clear filled TP orders from previous cycles to prevent duplicates in display
	for k := range bot.filledTPOrders {
		delete(bot.filledTPOrders, k)
	}
	bot.logger.Info("🧹 Cleared old filled TP orders during TP update to prevent duplicates")
	
	ctx := context.Background()
	
	// First, sync with exchange to get current TP orders (some may have been filled)
	ordersToCancel := make(map[string]*TPOrderInfo)
	for orderID, tpInfo := range bot.activeTPOrders {
		ordersToCancel[orderID] = &TPOrderInfo{
			Level:    tpInfo.Level,
			OrderID:  orderID,
			Quantity: tpInfo.Quantity,
		}
	}
	
	// Clear the map properly while still holding mutex
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	bot.tpOrderMutex.Unlock()
	
	// Now perform sync and cancellation operations without holding mutex
	if err := bot.syncTPOrdersFromExchange(ctx); err != nil {
		bot.logger.LogWarning("TP Sync", "Failed to sync TP orders from exchange: %v", err)
		// Continue with collected order data as fallback
	}
	
	if len(ordersToCancel) == 0 {
		bot.logger.Info("✅ No active TP orders found to cancel after exchange sync")
		return nil
	}
	
	// Cancel orders without holding mutex (to avoid blocking other operations)
	cancelledCount := 0
	failedCancellations := 0
	for orderID, tpInfo := range ordersToCancel {
		
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("TP Order Update", "Failed to cancel TP Level %d order %s: %v", tpInfo.Level, orderID, err)
			failedCancellations++
			continue
		}
		
		cancelledCount++
	}
	
	if failedCancellations > 0 {
		bot.logger.LogWarning("TP Update", "%d TP orders failed to cancel, continuing with new orders", failedCancellations)
	}
	
	// Get current total position size after any TP fills
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("TP Update", "Failed to get position data: %v", err)
		return fmt.Errorf("failed to get position for TP update: %w", err)
	}
	
	var totalPositionSize string
	var exchangeAvgPrice string
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			totalPositionSize = pos.Size
			exchangeAvgPrice = pos.AvgPrice
			break
		}
	}
	
	if totalPositionSize == "" || totalPositionSize == "0" {
		bot.logger.LogWarning("TP Update", "No position found for TP update - possibly all TP orders executed")
		return nil
	}
	
	exchangeAvgPriceFloat, err := parseFloat(exchangeAvgPrice)
	if err != nil {
		bot.logger.LogWarning("TP Update", "Failed to parse exchange avg price '%s': %v", exchangeAvgPrice, err)
		exchangeAvgPriceFloat = newAveragePrice // Use bot's price as fallback
	}
	
	// Use exchange average price if there's a significant difference (safety check)
	priceDiff := math.Abs(newAveragePrice - exchangeAvgPriceFloat)
	if priceDiff > 0.001 {
		bot.logger.LogWarning("TP Update", "Price mismatch: bot $%.4f vs exchange $%.4f (diff: $%.4f)", 
			newAveragePrice, exchangeAvgPriceFloat, priceDiff)
		newAveragePrice = exchangeAvgPriceFloat
	}
	
	// Place new multi-level TP orders based on new average price
	if err := bot.placeMultiLevelTPOrders(totalPositionSize, newAveragePrice); err != nil {
		bot.logger.LogWarning("TP Update", "Failed to place updated multi-level TP orders: %v", err)
		return fmt.Errorf("failed to place updated multi-level TP orders: %w", err)
	}
	
	// Log to console for visibility
	fmt.Printf("🔄 TP Orders Updated: %d levels placed at avg $%.4f\n", 
		bot.config.Strategy.TPLevels, newAveragePrice)
	
	bot.logger.Info("✅ TP order update completed")
	return nil
}

// executeSell executes a sell order to close position
func (bot *LiveBot) executeSell(price float64) {
	if bot.currentPosition <= 0 {
		return
	}

	ctx := context.Background()
	
	// Get current position from exchange to ensure accuracy
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("Position check", "Could not get current position for sell: %v", err)
		return
	}
	
	// Find our position
	var positionSize string
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			positionSize = pos.Size
			break
		}
	}
	
	if positionSize == "" {
		bot.logger.LogWarning("Position close", "No position found to close")
		return
	}
	
	orderParams := exchange.OrderParams{
		Category:  bot.category,
		Symbol:    bot.symbol,
		Side:      exchange.OrderSideSell,
		Quantity:  positionSize, // Use exact position size from exchange
		OrderType: exchange.OrderTypeMarket,
	}

	order, err := bot.exchange.PlaceMarketOrder(ctx, orderParams)
	if err != nil {
		bot.logger.Error("Failed to place sell order: %v", err)
		return
	}

	// Sync with exchange data instead of self-calculating P&L
	bot.syncAfterTrade(order, "SELL")
}

// syncAfterTrade syncs bot state with exchange after trade execution
func (bot *LiveBot) syncAfterTrade(order *exchange.Order, tradeType string) {
	// Wait longer for exchange to settle the trade, especially for market orders
	// Market orders need time for execution data to be available
	if order.OrderType == exchange.OrderTypeMarket {
		time.Sleep(2 * time.Second) // Increased wait time for market orders
	} else {
		time.Sleep(500 * time.Millisecond)
	}
	
	// Sync balance with exchange
	if err := bot.syncAccountBalance(); err != nil {
		bot.logger.LogWarning("Could not refresh balance after trade", "%v", err)
	}
	
	// Capture DCA level BEFORE position sync to avoid interference
	bot.positionMutex.RLock()
	previousDCALevel := bot.dcaLevel
	bot.positionMutex.RUnlock()
	
	// Sync position data with exchange
	if err := bot.syncPositionData(); err != nil {
		if err.Error() == "STRATEGY_SYNC_REQUIRED" {
			// Position was reset, strategy sync is needed
			bot.syncStrategyState()
		} else {
			bot.logger.LogWarning("Could not sync position after trade", "%v", err)
		}
		// Don't return here - continue with the rest of the function
	}
	
	// Update DCA level for buy trades
	if tradeType == "BUY" {
		// Protect DCA level operations with mutex
		bot.positionMutex.Lock()
		// Use the captured previousDCALevel (before position sync interference)
		// This ensures proper DCA level progression regardless of sync side effects
		oldDCALevel := bot.dcaLevel
		if previousDCALevel == 0 {
			bot.dcaLevel = 1 // First trade: 0 → 1
		} else {
			bot.dcaLevel = previousDCALevel + 1 // Subsequent trades: increment
		}
		
		// Log state change for DCA level
		bot.logger.LogStateChange("DCA Level", oldDCALevel, bot.dcaLevel, "Buy trade executed")
		
		// Keep DCA level progression simple and predictable
		// No position-based validation that can cause level jumping
		
		currentDCALevel := bot.dcaLevel
		currentPosition := bot.currentPosition
		avgPrice := bot.averagePrice
		bot.positionMutex.Unlock()
		
		// Log trade execution details with reliable data (don't rely on potentially empty order response fields)
		// Use the calculated values and exchange-synced data instead
		var executedQty string
		if avgPrice > 0 {
			executedQty = fmt.Sprintf("%.6f", currentPosition/avgPrice) // Calculate from position
		} else {
			executedQty = "0.000000" // Safety fallback
		}
		executedPrice := fmt.Sprintf("%.4f", avgPrice)
		executedValue := fmt.Sprintf("%.2f", currentPosition)
		
		bot.logger.LogTradeExecution(tradeType, order.OrderID, executedQty, executedPrice, executedValue, currentDCALevel, currentPosition, avgPrice)
		
		// Update multi-level TP orders for DCA trades (level >= 2) where average price changes
		// Note: We do this in syncAfterTrade to ensure it happens after position sync
		// Level 1 TP orders are placed by executeBuy, levels 2+ need updates due to new average price
		if currentDCALevel >= 2 && bot.config.Strategy.AutoTPOrders {
			fmt.Printf("🔄 DCA Level %d: Updating TP orders for new avg price $%.4f\n", currentDCALevel, avgPrice)
			
		// Update TP orders with proper error handling and timeout protection
		if err := bot.updateMultiLevelTPOrders(avgPrice); err != nil {
			bot.logger.LogWarning("TP Update", "Failed to update TP orders after DCA: %v", err)
			// Categorize TP update errors for better debugging
			if strings.Contains(err.Error(), "timeout") {
				bot.logger.LogWarning("TP Update Error", "Timeout during TP order update - exchange may be slow")
			} else if strings.Contains(err.Error(), "cancel") {
				bot.logger.LogWarning("TP Update Error", "Failed to cancel existing TP orders")
			} else if strings.Contains(err.Error(), "place") {
				bot.logger.LogWarning("TP Update Error", "Failed to place new TP orders")
			} else {
				bot.logger.LogWarning("TP Update Error", "Unknown TP update error: %v", err)
			}
			fmt.Printf("⚠️ TP update failed but bot continues normally: %v\n", err)
			// Continue execution - don't let TP update failure stop the bot
		} else {
			fmt.Printf("✅ TP orders updated successfully for DCA level %d\n", currentDCALevel)
		}
		}
		
		// Sync strategy state after BUY trade to keep DCA level and entry price current
		bot.syncStrategyState()
		
	} else if tradeType == "SELL" {
		// Create dedicated context for cleanup operations
		cleanupCtx := context.Background()
		
		// For sell trades, calculate realized P&L from position data before sync
		if currentPrice, err := bot.exchange.GetLatestPrice(cleanupCtx, bot.symbol); err == nil {
			// Get average price safely with mutex protection
			bot.positionMutex.RLock()
			avgPrice := bot.averagePrice
			bot.positionMutex.RUnlock()
			
	// Add defensive checks to prevent division by zero
	if avgPrice > 0 && currentPrice > 0 {
		profitPercent := (currentPrice - avgPrice) / avgPrice * 100
		
		// Log cycle completion
		bot.logger.LogCycleCompletion(currentPrice, avgPrice, profitPercent)
	} else {
		bot.logger.LogWarning("Cycle Completion", "Cannot calculate profit percentage: avgPrice=%.4f, currentPrice=%.4f", avgPrice, currentPrice)
	}
		}
		
		// Clean up multi-level TP orders since position is closed
		// First, collect order information while holding mutex
		bot.tpOrderMutex.Lock()
		ordersToCancel := make(map[string]*TPOrderInfo)
		for orderID, tpInfo := range bot.activeTPOrders {
			ordersToCancel[orderID] = &TPOrderInfo{
				Level:   tpInfo.Level,
				OrderID: orderID,
			}
		}
		// Clear map while still holding mutex
		for k := range bot.activeTPOrders {
			delete(bot.activeTPOrders, k)
		}
		bot.tpOrderMutex.Unlock()
		
		// Now cancel orders without holding mutex (to avoid blocking other operations)
		for orderID, tpInfo := range ordersToCancel {
			if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
				bot.logger.LogWarning("TP Cleanup", "Failed to cancel TP Level %d order %s: %v", tpInfo.Level, orderID, err)
			} else {
				bot.logger.Info("🧹 Cancelled TP Level %d order %s (position closed)", tpInfo.Level, orderID)
			}
		}
		
		// Log trade execution details
		bot.logger.LogTradeExecution(tradeType, order.OrderID, order.CumExecQty, order.AvgPrice, order.CumExecValue, 0, 0, 0)
		
		// Reset internal counters after sell with mutex protection
		bot.positionMutex.Lock()
		bot.dcaLevel = 0
		bot.positionMutex.Unlock()
		
		// Sync strategy state after position closure (this will call OnCycleComplete)
		bot.syncStrategyState()
		
		// Clear filled TP orders tracking since cycle is complete
		bot.clearFilledTPOrders()
		
		// Reset hold log counter to ensure first HOLD decision after cycle completion is logged
		bot.holdLogCounter = 0
	}
}

// syncExistingOrders syncs existing orders from exchange on startup
func (bot *LiveBot) syncExistingOrders() error {
	ctx := context.Background()
	
	// Get open orders for this symbol
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get existing orders: %w", err)
	}
	
	if len(orders) == 0 {
		fmt.Printf("✅ No existing orders found\n")
		return nil
	}
	
	// Check if we should cancel orphaned orders or sync them
	if bot.config.Strategy.CancelOrphanedOrders {
		return bot.cancelOrphanedOrders(orders)
	}
	
	// Sync existing orders instead of canceling
	tpOrderCount := 0
	otherOrderCount := 0
	
	bot.tpOrderMutex.Lock()
	defer bot.tpOrderMutex.Unlock()
	
	for _, order := range orders {
		// Check if this looks like a TP order (sell side, limit order)
		if order.Side == "Sell" && order.OrderType == "Limit" {
			tpOrderCount++
			
			// Try to reconstruct TP order info
			// Note: We can't perfectly reconstruct level/percent without more context
			// but we can track the order for cancellation purposes
			bot.activeTPOrders[order.OrderID] = &TPOrderInfo{
				Level:     0, // Unknown level
				Percent:   0, // Unknown percent
				Quantity:  order.Quantity,
				Price:     order.Price,
				OrderID:   order.OrderID,
			}
			
		} else {
			otherOrderCount++
		}
	}
	
	if tpOrderCount > 0 {
		fmt.Printf("🎯 Synced %d existing TP orders\n", tpOrderCount)
	}
	if otherOrderCount > 0 {
		fmt.Printf("📋 Found %d other orders (not TP orders)\n", otherOrderCount)
	}
	
	return nil
}

// calculateTPLevelQuantities pre-calculates quantities for all TP levels to ensure proper distribution
func (bot *LiveBot) calculateTPLevelQuantities(totalQty float64, constraints *exchange.TradingConstraints) []float64 {
	numLevels := bot.config.Strategy.TPLevels
	// Add safety check for invalid configuration
	if numLevels <= 0 {
		bot.logger.LogWarning("TP Calculation", "Invalid TP levels configuration: %d", numLevels)
		return []float64{} // Return empty slice
	}
	quantities := make([]float64, numLevels)
	
	// Calculate base quantity per level
	baseQtyPerLevel := totalQty * bot.config.Strategy.TPQuantity
	
	// Apply MinOrderQty constraint to base quantity
	if constraints.MinOrderQty > 0 && baseQtyPerLevel < constraints.MinOrderQty {
		baseQtyPerLevel = constraints.MinOrderQty
	}
	
	// Apply QtyStep constraint to base quantity
	if constraints.QtyStep > 0 {
		multiplier := math.Floor(baseQtyPerLevel / constraints.QtyStep)
		if multiplier < 1 {
			multiplier = 1
		}
		baseQtyPerLevel = multiplier * constraints.QtyStep
	}
	
	// Calculate total quantity needed for all levels with base quantity
	totalNeeded := baseQtyPerLevel * float64(numLevels)
	
	if totalNeeded <= totalQty {
		// Simple case: all levels get the same base quantity
		for i := 0; i < numLevels; i++ {
			quantities[i] = baseQtyPerLevel
		}
		
		// Distribute any remaining quantity to the last level
		remaining := totalQty - totalNeeded
		if remaining > 0 && numLevels > 0 && len(quantities) >= numLevels {
			quantities[numLevels-1] += remaining
		}
	} else {
		// Complex case: need to distribute available quantity across levels
		// Start with minimum quantities and distribute the rest
		remaining := totalQty
		
		// First pass: give each level the minimum required quantity
		minQty := math.Max(constraints.MinOrderQty, constraints.QtyStep)
		if minQty == 0 {
			minQty = 1 // Default minimum
		}
		
		for i := 0; i < numLevels && remaining >= minQty; i++ {
			quantities[i] = minQty
			remaining -= minQty
		}
		
		// Second pass: distribute remaining quantity proportionally
		if remaining > 0 {
			// Find how many levels got the minimum quantity
			activeLevels := 0
			for i := 0; i < numLevels; i++ {
				if quantities[i] > 0 {
					activeLevels++
				}
			}
			
			if activeLevels > 0 {
				extraPerLevel := remaining / float64(activeLevels)
				
				// Apply QtyStep constraint to extra quantity
				if constraints.QtyStep > 0 {
					multiplier := math.Floor(extraPerLevel / constraints.QtyStep)
					extraPerLevel = multiplier * constraints.QtyStep
				}
				
				// Distribute extra quantity
				for i := 0; i < numLevels && extraPerLevel > 0; i++ {
					if quantities[i] > 0 {
						quantities[i] += extraPerLevel
						remaining -= extraPerLevel
					}
				}
				
				// Add any final remaining to the last level
				if remaining > 0 && numLevels > 0 && len(quantities) >= numLevels && quantities[numLevels-1] > 0 {
					quantities[numLevels-1] += remaining
				}
			}
		}
	}
	
	// Only log if there's something notable about the distribution
	if totalNeeded > totalQty {
		bot.logger.LogWarning("TP Calculation", "Insufficient quantity: need %.6f, have %.6f", totalNeeded, totalQty)
	}
	
	return quantities
}

// syncTPOrdersFromExchange syncs internal TP order tracking with exchange state
func (bot *LiveBot) syncTPOrdersFromExchange(ctx context.Context) error {
	// Get current open orders from exchange
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get open orders: %w", err)
	}
	
	// Get current average price for TP order validation
	positions, posErr := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	var currentAvgPrice float64 = 0
	if posErr == nil {
		for _, pos := range positions {
			if pos.Symbol == bot.symbol && pos.Side == "Buy" {
				if avgPrice, err := parseFloat(pos.AvgPrice); err == nil {
					currentAvgPrice = avgPrice
				}
				break
			}
		}
	}
	
	// Create a map of current TP orders on exchange with proper identification
	exchangeTPOrders := make(map[string]*exchange.Order)
	for _, order := range orders {
		if bot.isTPOrder(order, currentAvgPrice) {
			exchangeTPOrders[order.OrderID] = order
		}
	}
	
	// Update TP order tracking with proper mutex protection
	bot.tpOrderMutex.Lock()
	defer bot.tpOrderMutex.Unlock()
	
	// Remove filled/cancelled orders from our tracking
	removedCount := 0
	for orderID := range bot.activeTPOrders {
		if _, exists := exchangeTPOrders[orderID]; !exists {
			// Order no longer exists on exchange (filled or cancelled)
			delete(bot.activeTPOrders, orderID)
			removedCount++
		}
	}
	
	// Add any TP orders we weren't tracking (shouldn't happen, but safety check)
	addedCount := 0
	for orderID, order := range exchangeTPOrders {
		if _, exists := bot.activeTPOrders[orderID]; !exists {
			// Found TP order we weren't tracking
			bot.activeTPOrders[orderID] = &TPOrderInfo{
				Level:     0, // Unknown level
				Percent:   0, // Unknown percent
				Quantity:  order.Quantity,
				Price:     order.Price,
				OrderID:   orderID,
			}
			addedCount++
		}
	}
	
	if removedCount > 0 || addedCount > 0 {
		bot.logger.Info("🔄 TP Sync: %d filled orders removed, %d untracked added", removedCount, addedCount)
	}
	
	return nil
}

// cancelOrphanedOrders cancels all existing orders on startup
func (bot *LiveBot) cancelOrphanedOrders(orders []*exchange.Order) error {
	bot.logger.Info("🧹 Canceling %d orphaned orders on startup...", len(orders))
	fmt.Printf("🧹 Canceling %d orphaned orders...\n", len(orders))
	
	cancelledCount := 0
	failedCount := 0
	
	for _, order := range orders {
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, order.OrderID); err != nil {
			bot.logger.LogWarning("Startup Cleanup", "Failed to cancel order %s (%s %s): %v", 
				order.OrderID, order.Side, order.OrderType, err)
			failedCount++
			continue
		}
		
		bot.logger.Info("❌ Cancelled orphaned order: %s (%s %s %s)", 
			order.OrderID, order.Side, order.OrderType, order.Quantity)
		cancelledCount++
	}
	
	if failedCount > 0 {
		bot.logger.LogWarning("Startup Cleanup", "Cancelled %d orders, %d failed", cancelledCount, failedCount)
		fmt.Printf("⚠️ Cancelled %d orders, %d failed to cancel\n", cancelledCount, failedCount)
	} else {
		fmt.Printf("✅ Cancelled %d orphaned orders successfully\n", cancelledCount)
	}
	
	return nil
}

// cancelAllTPOrders cancels all active TP orders during shutdown
func (bot *LiveBot) cancelAllTPOrders() error {
	// First, sync with exchange to get REAL state (not just memory)
	ctx := context.Background()
	
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("Shutdown Cleanup", "Failed to get open orders from exchange: %v", err)
		fmt.Printf("⚠️ Failed to fetch exchange orders, using memory fallback\n")
		// Fall back to memory-based cancellation
		return bot.cancelTPOrdersFromMemory()
	}
	
	// Find TP orders on exchange with proper identification
	exchangeTPOrders := make(map[string]*exchange.Order)
	
	// Get current position to validate TP order prices
	positions, posErr := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	var currentAvgPrice float64 = 0
	if posErr == nil {
		for _, pos := range positions {
			if pos.Symbol == bot.symbol && pos.Side == "Buy" {
				if avgPrice, err := parseFloat(pos.AvgPrice); err == nil {
					currentAvgPrice = avgPrice
				}
				break
			}
		}
	}
	
	for _, order := range orders {
		if bot.isTPOrder(order, currentAvgPrice) {
			exchangeTPOrders[order.OrderID] = order
		}
	}
	
	orderCount := len(exchangeTPOrders)
	
	if orderCount == 0 {
		bot.logger.Info("✅ No active TP orders found on exchange")
		fmt.Printf("✅ No active TP orders found on exchange\n")
		return nil
	}
	
	bot.logger.Info("🧹 Canceling %d active TP orders from exchange...", orderCount)
	fmt.Printf("🧹 Found %d TP orders on exchange to cancel...\n", orderCount)
	
	// Cancel orders directly from exchange data (no need for memory collection)
	// Clear our internal tracking since we're cancelling everything
	bot.tpOrderMutex.Lock()
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	bot.tpOrderMutex.Unlock()
	
	// Now cancel orders based on exchange data
	cancelledCount := 0
	failedCount := 0
	
	// Create level counter for better logging (since we don't know actual levels from exchange)
	level := 1
	for orderID, order := range exchangeTPOrders {
		fmt.Printf("🔄 Cancelling TP order %d/%d...\n", level, orderCount)
		
		// Use existing method with built-in timeout and retry logic
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("Shutdown Cleanup", "Failed to cancel TP order %s: %v", orderID, err)
			fmt.Printf("❌ Failed to cancel TP order %d: %v\n", level, err)
			failedCount++
			level++
			continue
		}
		
		bot.logger.Info("❌ Cancelled TP order: %s (qty: %s, price: %s)", orderID, order.Quantity, order.Price)
		fmt.Printf("✅ Cancelled TP order %d successfully\n", level)
		cancelledCount++
		level++
	}
	
	if failedCount > 0 {
		bot.logger.LogWarning("Shutdown Cleanup", "Cancelled %d TP orders, %d failed", cancelledCount, failedCount)
		fmt.Printf("⚠️ Cancelled %d TP orders, %d failed to cancel\n", cancelledCount, failedCount)
	} else {
		fmt.Printf("✅ Cancelled %d TP orders successfully\n", cancelledCount)
	}
	
	return nil
}

// clearFilledTPOrders clears the filled TP orders tracking (called when cycle completes)
func (bot *LiveBot) clearFilledTPOrders() {
	bot.tpOrderMutex.Lock()
	defer bot.tpOrderMutex.Unlock()
	
	// Clear all filled TP order tracking
	for k := range bot.filledTPOrders {
		delete(bot.filledTPOrders, k)
	}
	
	bot.logger.Info("🧹 Cleared filled TP orders tracking for new cycle")
}

// cancelTPOrdersFromMemory is a fallback when exchange fetch fails
func (bot *LiveBot) cancelTPOrdersFromMemory() error {
	// First, collect order information while holding mutex
	bot.tpOrderMutex.Lock()
	orderCount := len(bot.activeTPOrders)
	
	if orderCount == 0 {
		bot.tpOrderMutex.Unlock()
		bot.logger.Info("✅ No active TP orders in memory to cancel")
		fmt.Printf("✅ No active TP orders found in memory\n")
		return nil
	}
	
	bot.logger.Info("🧹 Canceling %d TP orders from memory...", orderCount)
	fmt.Printf("🧹 Found %d TP orders in memory to cancel...\n", orderCount)
	
	// Collect order information to cancel
	ordersToCancel := make(map[string]*TPOrderInfo)
	for orderID, tpInfo := range bot.activeTPOrders {
		ordersToCancel[orderID] = &TPOrderInfo{
			Level:    tpInfo.Level,
			OrderID:  orderID,
			Quantity: tpInfo.Quantity,
		}
	}
	
	// Clear the map while still holding mutex
	for k := range bot.activeTPOrders {
		delete(bot.activeTPOrders, k)
	}
	bot.tpOrderMutex.Unlock()
	
	// Now cancel orders without holding mutex
	cancelledCount := 0
	failedCount := 0
	
	for orderID, tpInfo := range ordersToCancel {
		fmt.Printf("🔄 Cancelling TP Level %d order...\n", tpInfo.Level)
		
		// Use existing method with built-in timeout and retry logic
		if err := bot.cancelOrderWithRetry(bot.category, bot.symbol, orderID); err != nil {
			bot.logger.LogWarning("Shutdown Cleanup", "Failed to cancel TP Level %d order %s: %v", tpInfo.Level, orderID, err)
			fmt.Printf("❌ Failed to cancel TP Level %d: %v\n", tpInfo.Level, err)
			failedCount++
			continue
		}
		
		bot.logger.Info("❌ Cancelled TP Level %d order: %s (qty: %s)", tpInfo.Level, orderID, tpInfo.Quantity)
		fmt.Printf("✅ Cancelled TP Level %d successfully\n", tpInfo.Level)
		cancelledCount++
	}
	
	if failedCount > 0 {
		bot.logger.LogWarning("Shutdown Cleanup", "Cancelled %d TP orders, %d failed", cancelledCount, failedCount)
		fmt.Printf("⚠️ Cancelled %d TP orders, %d failed to cancel\n", cancelledCount, failedCount)
	} else {
		fmt.Printf("✅ Cancelled %d TP orders successfully\n", cancelledCount)
	}
	
	return nil
}

// isTPOrder determines if an order is a take profit order placed by this bot
func (bot *LiveBot) isTPOrder(order *exchange.Order, currentAvgPrice float64) bool {
	// Must be a sell limit order
	if order.Side != "Sell" || order.OrderType != "Limit" {
		return false
	}
	
	// Must be for our symbol (should already be filtered, but safety check)
	if order.Symbol != bot.symbol {
		return false
	}
	
	// Parse order price
	orderPrice, err := parseFloat(order.Price)
	if err != nil {
		bot.logger.LogWarning("TP Identification", "Failed to parse order price '%s': %v", order.Price, err)
		return false
	}
	
	// If we don't have current avg price, we can't validate - be conservative
	if currentAvgPrice <= 0 {
		// Check if this order is in our internal tracking
		bot.tpOrderMutex.RLock()
		_, isTracked := bot.activeTPOrders[order.OrderID]
		bot.tpOrderMutex.RUnlock()
		return isTracked
	}
	
	// TP orders should be ABOVE average price (for profit)
	if orderPrice <= currentAvgPrice {
		return false
	}
	
	// TP orders should be within reasonable profit range (0.1% to 15% above avg price)
	profitPercent := (orderPrice - currentAvgPrice) / currentAvgPrice * 100
	minProfitPercent := 0.1  // 0.1% minimum
	maxProfitPercent := 15.0 // 15% maximum (beyond this is likely not a TP order)
	
	if profitPercent < minProfitPercent || profitPercent > maxProfitPercent {
		return false
	}
	
	// Check quantity - TP orders typically have smaller quantities than full position
	orderQty, qtyErr := parseFloat(order.Quantity)
	if qtyErr == nil && currentAvgPrice > 0 {
		// Estimate full position size from our current position value
		bot.positionMutex.RLock()
		fullPositionQty := bot.currentPosition / currentAvgPrice
		bot.positionMutex.RUnlock()
		
		if fullPositionQty > 0 {
			qtyPercent := orderQty / fullPositionQty * 100
			// TP orders are typically 10%-50% of position (not 100%+ which would be full exit)
			if qtyPercent > 70 {
				return false
			}
		}
	}
	
	return true
}

// detectFilledTPOrders detects which TP orders have been filled since last check
func (bot *LiveBot) detectFilledTPOrders() []string {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	// Get current open orders with timeout
	orders, err := bot.exchange.GetOpenOrders(ctx, bot.category, bot.symbol)
	if err != nil {
		bot.logger.LogWarning("TP Fill Detection", "Failed to get open orders: %v", err)
		return nil
	}
	
	// Get current average price for TP validation
	positions, posErr := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	var currentAvgPrice float64 = 0
	if posErr == nil {
		for _, pos := range positions {
			if pos.Symbol == bot.symbol && pos.Side == "Buy" {
				if avgPrice, err := parseFloat(pos.AvgPrice); err == nil {
					currentAvgPrice = avgPrice
				}
				break
			}
		}
	}
	
	// Create map of currently active TP orders on exchange
	activeOnExchange := make(map[string]bool)
	for _, order := range orders {
		if bot.isTPOrder(order, currentAvgPrice) {
			activeOnExchange[order.OrderID] = true
		}
	}
	
	// Find filled TP orders (in our tracking but not on exchange)
	bot.tpOrderMutex.Lock()
	var filledOrderDetails []string
	
	for orderID, tpInfo := range bot.activeTPOrders {
		if !activeOnExchange[orderID] {
			// Validate this is actually a TP order fill, not a system error
			if tpInfo.Level > 0 && tpInfo.Percent > 0 {
				// Order is no longer on exchange - likely filled
				bot.logger.Info("🎯 TP Order FILLED: Level %d at $%s (%.2f%%) - OrderID: %s", 
					tpInfo.Level, tpInfo.Price, tpInfo.Percent*100, orderID)
				
				// Log detailed TP order fill information
				quantityFloat, _ := parseFloat(tpInfo.Quantity)
				priceFloat, _ := parseFloat(tpInfo.Price)
				bot.logger.LogTPOrderDetails(tpInfo.Level, quantityFloat, priceFloat, tpInfo.Percent, orderID, "FILLED")
				
				// Move to filled orders tracking
				tpInfo.Filled = true
				bot.filledTPOrders[orderID] = tpInfo
				
				// Create detailed string for display
				filledDetail := fmt.Sprintf("TP%d@$%s(%.1f%%)", tpInfo.Level, tpInfo.Price, tpInfo.Percent*100)
				filledOrderDetails = append(filledOrderDetails, filledDetail)
			} else {
				// Invalid TP order data - log warning
				bot.logger.LogWarning("TP Fill Detection", "Invalid TP order data detected - Level: %d, Percent: %.4f, OrderID: %s", 
					tpInfo.Level, tpInfo.Percent, orderID)
			}
			
			// Remove from active orders regardless
			delete(bot.activeTPOrders, orderID)
		}
	}
	
	bot.tpOrderMutex.Unlock()
	
	return filledOrderDetails
}

// getFilledTPSummary returns a summary of filled TP orders for display
func (bot *LiveBot) getFilledTPSummary() string {
	bot.tpOrderMutex.RLock()
	defer bot.tpOrderMutex.RUnlock()
	
	if len(bot.filledTPOrders) == 0 {
		return ""
	}
	
	var filledSummary []string
	for _, tpInfo := range bot.filledTPOrders {
		summary := fmt.Sprintf("TP%d@$%s", tpInfo.Level, tpInfo.Price)
		filledSummary = append(filledSummary, summary)
	}
	
	return strings.Join(filledSummary, ", ")
}

// parseFloat safely parses a string to float64 with comprehensive validation
func parseFloat(s string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	// Trim whitespace that might cause parsing errors
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("string contains only whitespace")
	}
	// Check for common invalid values
	if s == "null" || s == "undefined" || s == "NaN" {
		return 0, fmt.Errorf("invalid numeric value: %s", s)
	}
	return strconv.ParseFloat(s, 64)
}

// interpretIndicatorBias converts raw indicator values into meaningful bias descriptions
func (bot *LiveBot) interpretIndicatorBias(name string, value float64, currentPrice float64) string {
	switch strings.ToLower(name) {
	case "rsi":
		if value > 70 {
			return fmt.Sprintf("Overbought (%.1f > 70)", value)
		} else if value < 30 {
			return fmt.Sprintf("Oversold (%.1f < 30)", value)
		} else if value > 50 {
			return fmt.Sprintf("Bullish (%.1f > 50)", value)
		} else {
			return fmt.Sprintf("Bearish (%.1f < 50)", value)
		}
	
	case "mfi", "money flow index":
		if value > 80 {
			return fmt.Sprintf("Overbought (%.1f > 80)", value)
		} else if value < 20 {
			return fmt.Sprintf("Oversold (%.1f < 20)", value)
		} else if value > 50 {
			return fmt.Sprintf("Bullish (%.1f > 50)", value)
		} else {
			return fmt.Sprintf("Bearish (%.1f < 50)", value)
		}
	
	case "wavetrend":
		if value > 50 {
			return fmt.Sprintf("Strong Overbought (%.1f > 50)", value)
		} else if value > 20 {
			return fmt.Sprintf("Overbought (%.1f > 20)", value)
		} else if value < -50 {
			return fmt.Sprintf("Strong Oversold (%.1f < -50)", value)
		} else if value < -20 {
			return fmt.Sprintf("Oversold (%.1f < -20)", value)
		} else if value > 0 {
			return fmt.Sprintf("Bullish (%.1f > 0)", value)
		} else {
			return fmt.Sprintf("Bearish (%.1f < 0)", value)
		}
	
	case "hull_ma", "hull ma":
		if value > currentPrice {
			return fmt.Sprintf("Bullish ($%.2f > $%.2f)", value, currentPrice)
		} else {
			return fmt.Sprintf("Bearish ($%.2f < $%.2f)", value, currentPrice)
		}
	
	case "ema", "sma", "ma":
		if value > currentPrice {
			return fmt.Sprintf("Bullish ($%.2f > $%.2f)", value, currentPrice)
		} else {
			return fmt.Sprintf("Bearish ($%.2f < $%.2f)", value, currentPrice)
		}
	
	case "keltner", "keltner channels":
		// For Keltner Channels, the value is usually the middle line
		diff := ((value - currentPrice) / currentPrice) * 100
		if diff > 1 {
			return fmt.Sprintf("Below Band ($%.2f, +%.1f%%)", value, diff)
		} else if diff < -1 {
			return fmt.Sprintf("Above Band ($%.2f, %.1f%%)", value, diff)
		} else {
			return fmt.Sprintf("Near Middle ($%.2f)", value)
		}
	
	case "bollinger", "bb":
		// Similar to Keltner for middle band
		diff := ((value - currentPrice) / currentPrice) * 100
		if diff > 1 {
			return fmt.Sprintf("Below Band ($%.2f, +%.1f%%)", value, diff)
		} else if diff < -1 {
			return fmt.Sprintf("Above Band ($%.2f, %.1f%%)", value, diff)
		} else {
			return fmt.Sprintf("Near Middle ($%.2f)", value)
		}
	
	case "macd":
		if value > 0 {
			return fmt.Sprintf("Bullish (%.4f > 0)", value)
		} else {
			return fmt.Sprintf("Bearish (%.4f < 0)", value)
		}
	
	default:
		// For unknown indicators, just show the value with some basic interpretation
		return fmt.Sprintf("%.4f", value)
	}
}


// placeOrderWithRetry places an order with timeout and retry logic
func (bot *LiveBot) placeOrderWithRetry(params exchange.OrderParams, isMarket bool) (*exchange.Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	var result *exchange.Order
	
	// Use recovery handler for intelligent retry with backoff and circuit breaker protection
	err := bot.recoveryHandler.ExecuteWithRecovery(ctx, "OrderPlacement", "PlaceOrder", func() error {
		var orderErr error
		
		if isMarket {
			result, orderErr = bot.protectedPlaceMarketOrder(ctx, params)
		} else {
			result, orderErr = bot.protectedPlaceLimitOrder(ctx, params)
		}
		
		return orderErr
	})
	
	if err != nil {
		return nil, fmt.Errorf("order placement failed after retries: %w", err)
	}
	
	return result, nil
}

// cancelOrderWithRetry cancels an order with timeout and retry logic using recovery handler
func (bot *LiveBot) cancelOrderWithRetry(category, symbol, orderID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Use recovery handler for intelligent retry with backoff
	return bot.recoveryHandler.ExecuteWithRecovery(ctx, "OrderCancellation", "CancelOrder", func() error {
		return bot.exchange.CancelOrder(ctx, category, symbol, orderID)
	})
}

// placeFallbackTPOrder places a single TP order for leftover quantity at level 5
func (bot *LiveBot) placeFallbackTPOrder(quantity float64, avgEntryPrice float64, constraints *exchange.TradingConstraints, level int) error {
	// Add zero-value check to prevent division by zero
	if bot.config.Strategy.TPLevels <= 0 {
		return fmt.Errorf("invalid TP levels configuration: %d", bot.config.Strategy.TPLevels)
	}
	
	// Calculate level percentage using dynamic TP if enabled, otherwise use fixed TP
	var levelPercent float64
	if bot.strategy.IsDynamicTPEnabled() {
		bot.logger.LogDebugOnly("🔍 Dynamic TP Fallback: Starting calculation for level %d", level)
		
		// Get recent klines for dynamic TP calculation
		recentKlines, err := bot.getRecentKlines()
		if err != nil {
			bot.logger.LogWarning("Dynamic TP Fallback", "Failed to get recent klines for dynamic TP calculation: %v, falling back to fixed TP", err)
			levelPercent = bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
			bot.logger.LogDebugOnly("🔍 Dynamic TP Fallback: Using fixed TP Level %d: %.3f%%", level, levelPercent*100)
		} else {
			bot.logger.LogDebugOnly("🔍 Dynamic TP Fallback: Got %d klines for calculation", len(recentKlines))
			
			// Calculate dynamic TP percentage
			currentCandle := types.OHLCV{Close: avgEntryPrice} // Use avg price as current price for TP calculation
			if len(recentKlines) > 0 {
				currentCandle = recentKlines[len(recentKlines)-1] // Use latest candle
				bot.logger.LogDebugOnly("🔍 Dynamic TP Fallback: Using latest candle - Close: $%.2f", currentCandle.Close)
			} else {
				bot.logger.LogDebugOnly("🔍 Dynamic TP Fallback: Using avg entry price as current candle - Close: $%.2f", avgEntryPrice)
			}
			
			dynamicTPPercent, err := bot.strategy.GetDynamicTPPercent(currentCandle, recentKlines)
			if err != nil {
				bot.logger.LogWarning("Dynamic TP Fallback", "Dynamic TP calculation failed: %v, falling back to fixed TP", err)
				levelPercent = bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
				bot.logger.LogDebugOnly("🔍 Dynamic TP Fallback: Using fixed TP Level %d: %.3f%%", level, levelPercent*100)
			} else {
				// Calculate fixed TP for comparison
				fixedTPPercent := bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
				
				// Scale the dynamic TP percentage by level
				levelPercent = dynamicTPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
				
				bot.logger.Info("🎯 Dynamic TP Fallback Level %d: %.3f%% (base dynamic: %.3f%%, fixed would be: %.3f%%)", 
					level, levelPercent*100, dynamicTPPercent*100, fixedTPPercent*100)
				bot.logger.LogDebugOnly("🔍 Dynamic TP Fallback: Level scaling - Level %d/%d = %.3fx multiplier", 
					level, bot.config.Strategy.TPLevels, float64(level)/float64(bot.config.Strategy.TPLevels))
			}
		}
	} else {
		// Use fixed TP calculation
		levelPercent = bot.config.Strategy.TPPercent * float64(level) / float64(bot.config.Strategy.TPLevels)
		bot.logger.LogDebugOnly("🔍 Fixed TP Fallback Level %d: %.3f%% (base: %.3f%%)", 
			level, levelPercent*100, bot.config.Strategy.TPPercent*100)
	}
	tpPrice := avgEntryPrice * (1 + levelPercent)
	
	// Round price to exchange tick size
	if constraints.MinPriceStep > 0 {
		tpPrice = math.Round(tpPrice/constraints.MinPriceStep) * constraints.MinPriceStep
	}
	
	// Apply quantity step constraint using floor
	if constraints.QtyStep > 0 {
		multiplier := math.Floor(quantity / constraints.QtyStep)
		if multiplier < 1 {
			return fmt.Errorf("fallback quantity %.6f < step size %.6f", quantity, constraints.QtyStep)
		}
		quantity = multiplier * constraints.QtyStep
	}
	
	// Check minimum order constraints
	orderValue := quantity * tpPrice
	if quantity < constraints.MinOrderQty {
		return fmt.Errorf("fallback quantity %.6f < min %.6f", quantity, constraints.MinOrderQty)
	}
	if orderValue < constraints.MinOrderValue {
		return fmt.Errorf("fallback order value $%.2f < min $%.2f", orderValue, constraints.MinOrderValue)
	}
	
	// Format values
	formattedQty := fmt.Sprintf("%.6f", quantity)
	formattedPrice := fmt.Sprintf("%.4f", tpPrice)
	
	// Place fallback TP limit order
	orderParams := exchange.OrderParams{
		Category:  bot.category,
		Symbol:    bot.symbol,
		Side:      exchange.OrderSideSell,
		Quantity:  formattedQty,
		OrderType: exchange.OrderTypeLimit,
		Price:     formattedPrice,
	}
	
	tpOrder, err := bot.placeOrderWithRetry(orderParams, false) // false for limit order
	if err != nil {
		return fmt.Errorf("failed to place fallback TP order: %w", err)
	}
	
	// Track the fallback TP order
	bot.tpOrderMutex.Lock()
	defer bot.tpOrderMutex.Unlock()
	bot.activeTPOrders[tpOrder.OrderID] = &TPOrderInfo{
		Level:     level, // Use next available level number
		Percent:   levelPercent,
		Quantity:  formattedQty,
		Price:     formattedPrice,
		OrderID:   tpOrder.OrderID,
		Filled:    false,
		FilledQty: "0",
	}
	
	bot.logger.Info("✅ Fallback TP combined with Level %d: %s %s at $%s (%.2f%%) - Order ID: %s", 
		level, formattedQty, bot.symbol, formattedPrice, levelPercent*100, tpOrder.OrderID)
	
	return nil
}
