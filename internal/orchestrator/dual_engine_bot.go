package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/engines"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// DualEngineBot orchestrates multiple trading engines based on market regime
// This is the main coordinator that decides which engine to use and manages transitions
type DualEngineBot struct {
	// Basic configuration
	symbol        string
	config        *config.LiveBotConfig
	logger        *logger.Logger
	exchange      exchange.LiveTradingExchange
	
	// Engine management
	engines       map[engines.EngineType]engines.TradingEngine
	activeEngine  engines.TradingEngine
	engineMutex   sync.RWMutex
	
	// Regime detection integration
	regimeDetector     *regime.RegimeDetector
	currentRegime      regime.RegimeType
	lastRegimeSignal   *regime.RegimeSignal
	regimeHistory      []*regime.RegimeChange
	regimeMutex        sync.RWMutex
	
	// Market data management
	marketDataBuffer   []types.OHLCV
	bufferSize         int
	bufferMutex        sync.RWMutex
	
	// Bot control and state
	running            bool
	stopChan           chan struct{}
	controlMutex       sync.RWMutex
	
	// Performance tracking
	orchestratorMetrics *OrchestratorMetrics
	metricsMutex        sync.RWMutex
	
	// Engine transition management
	lastTransition     time.Time
	transitionCooldown time.Duration
	minRegimeDuration  time.Duration
}

// OrchestratorMetrics tracks orchestrator performance
type OrchestratorMetrics struct {
	TotalSignals       int                              `json:"total_signals"`
	EngineActivations  map[engines.EngineType]int       `json:"engine_activations"`
	RegimeChanges      int                              `json:"regime_changes"`
	TransitionCosts    float64                          `json:"total_transition_costs"`
	LastTransition     time.Time                        `json:"last_transition"`
	SessionStartTime   time.Time                        `json:"session_start_time"`
	TotalPnL           float64                          `json:"total_pnl"`
	
	// Engine-specific performance
	EnginePerformance  map[engines.EngineType]EnginePerformanceSummary `json:"engine_performance"`
}

// EnginePerformanceSummary summarizes individual engine performance
type EnginePerformanceSummary struct {
	ActivationCount    int       `json:"activation_count"`
	TotalActiveTime    time.Duration `json:"total_active_time"`
	LastActivation     time.Time     `json:"last_activation"`
	TotalPnL          float64       `json:"total_pnl"`
	WinRate           float64       `json:"win_rate"`
	SignalCount       int           `json:"signal_count"`
}

// DualEngineBotConfig holds configuration for the orchestrator
type DualEngineBotConfig struct {
	// Engine selection parameters
	RegimeConfidenceThreshold  float64       `json:"regime_confidence_threshold"`  // Min confidence to switch engines
	TransitionCooldownMinutes  int           `json:"transition_cooldown_minutes"`  // Minutes between engine switches
	MinRegimeDurationMinutes   int           `json:"min_regime_duration_minutes"`  // Min time in regime before switching
	
	// Signal filtering
	MinSignalConfidence        float64       `json:"min_signal_confidence"`        // Min confidence for engine signals
	MaxDailyTransitions        int           `json:"max_daily_transitions"`        // Max engine switches per day
	
	// Risk management
	GlobalMaxPositionSize      float64       `json:"global_max_position_size"`     // Max total position size
	GlobalMaxDailyLoss         float64       `json:"global_max_daily_loss"`        // Max daily loss across all engines
	EmergencyStopLoss          float64       `json:"emergency_stop_loss"`          // Emergency portfolio stop loss
	
	// Data management
	MarketDataBufferSize       int           `json:"market_data_buffer_size"`      // Size of market data buffer
	RegimeDetectionMinData     int           `json:"regime_detection_min_data"`    // Min data points for regime detection
}

// DefaultDualEngineBotConfig returns default configuration
func DefaultDualEngineBotConfig() *DualEngineBotConfig {
	return &DualEngineBotConfig{
		RegimeConfidenceThreshold:  0.7,   // 70% confidence to switch engines
		TransitionCooldownMinutes:  15,    // 15 minutes between switches
		MinRegimeDurationMinutes:   30,    // 30 minutes minimum regime duration
		MinSignalConfidence:        0.5,   // 50% minimum signal confidence
		MaxDailyTransitions:        10,    // Max 10 engine switches per day
		GlobalMaxPositionSize:      0.8,   // 80% max position size
		GlobalMaxDailyLoss:         0.05,  // 5% max daily loss
		EmergencyStopLoss:          0.15,  // 15% emergency stop loss
		MarketDataBufferSize:       500,   // 500 candles buffer
		RegimeDetectionMinData:     250,   // 250 candles for regime detection
	}
}

// NewDualEngineBot creates a new dual engine bot orchestrator
func NewDualEngineBot(symbol string, config *config.LiveBotConfig, exchangeInstance exchange.LiveTradingExchange, fileLogger *logger.Logger) (*DualEngineBot, error) {
	if config == nil {
		return nil, fmt.Errorf("bot configuration is required")
	}
	
	bot := &DualEngineBot{
		symbol:              symbol,
		config:              config,
		logger:              fileLogger,
		exchange:            exchangeInstance,
		engines:             make(map[engines.EngineType]engines.TradingEngine),
		regimeDetector:      regime.NewRegimeDetector(),
		currentRegime:       regime.RegimeUncertain,
		regimeHistory:       make([]*regime.RegimeChange, 0, 100),
		marketDataBuffer:    make([]types.OHLCV, 0, 500),
		bufferSize:          500,
		running:             false,
		stopChan:            make(chan struct{}),
		orchestratorMetrics: &OrchestratorMetrics{
			EngineActivations: make(map[engines.EngineType]int),
			EnginePerformance: make(map[engines.EngineType]EnginePerformanceSummary),
			SessionStartTime:  time.Now(),
		},
		transitionCooldown:  15 * time.Minute,
		minRegimeDuration:   30 * time.Minute,
	}
	
	// Initialize engines
	if err := bot.initializeEngines(); err != nil {
		return nil, fmt.Errorf("failed to initialize engines: %w", err)
	}
	
	return bot, nil
}

// initializeEngines creates and configures all trading engines
func (bot *DualEngineBot) initializeEngines() error {
	// Create Grid Engine for ranging markets
	gridEngine, err := engines.NewGridEngine(bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to create grid engine: %w", err)
	}
	
	// Create Trend Engine for trending markets
	trendEngineAdapter, err := engines.NewTrendEngineAdapter(bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to create trend engine: %w", err)
	}
	
	// Register engines
	bot.engines[engines.EngineTypeGrid] = gridEngine
	bot.engines[engines.EngineTypeTrend] = trendEngineAdapter
	
	// Set default active engine (Grid for uncertain regime)
	bot.activeEngine = gridEngine
	gridEngine.SetActive(true)
	
	// Initialize performance tracking for each engine
	for engineType := range bot.engines {
		bot.orchestratorMetrics.EngineActivations[engineType] = 0
		bot.orchestratorMetrics.EnginePerformance[engineType] = EnginePerformanceSummary{
			LastActivation: time.Time{},
		}
	}
	
	bot.logger.Info("🚀 Dual Engine Bot initialized with %d engines (Grid + Trend)", len(bot.engines))
	
	return nil
}

// Start begins the dual engine bot orchestration
func (bot *DualEngineBot) Start() error {
	bot.controlMutex.Lock()
	defer bot.controlMutex.Unlock()
	
	if bot.running {
		return fmt.Errorf("bot is already running")
	}
	
	// Start all engines
	for engineType, engine := range bot.engines {
		if err := engine.Initialize(); err != nil {
			return fmt.Errorf("failed to initialize %s engine: %w", engineType, err)
		}
		if err := engine.Start(); err != nil {
			return fmt.Errorf("failed to start %s engine: %w", engineType, err)
		}
	}
	
	bot.running = true
	bot.orchestratorMetrics.SessionStartTime = time.Now()
	
	bot.logger.Info("🚀 Dual Engine Bot started - Active Engine: %s", bot.activeEngine.GetName())
	
	// Start the main orchestration loop in a separate goroutine
	go bot.orchestrationLoop()
	
	return nil
}

// Stop stops the dual engine bot
func (bot *DualEngineBot) Stop() error {
	bot.controlMutex.Lock()
	defer bot.controlMutex.Unlock()
	
	if !bot.running {
		return nil
	}
	
	// Signal stop
	close(bot.stopChan)
	bot.running = false
	
	// Stop all engines
	for engineType, engine := range bot.engines {
		if err := engine.Stop(); err != nil {
			bot.logger.Error("Failed to stop %s engine: %v", engineType, err)
		}
	}
	
	bot.logger.Info("🛑 Dual Engine Bot stopped")
	
	return nil
}

// orchestrationLoop is the main control loop that coordinates engines and regime detection
func (bot *DualEngineBot) orchestrationLoop() {
	ticker := time.NewTicker(5 * time.Minute) // Check every 5 minutes
	defer ticker.Stop()
	
	bot.logger.Info("🎼 Orchestration loop started")
	
	for {
		select {
		case <-ticker.C:
			bot.performOrchestrationCycle()
		case <-bot.stopChan:
			bot.logger.Info("🎼 Orchestration loop stopped")
			return
		}
	}
}

// performOrchestrationCycle executes one orchestration cycle
func (bot *DualEngineBot) performOrchestrationCycle() {
	defer func() {
		if r := recover(); r != nil {
			bot.logger.Error("Orchestration cycle panic: %v", r)
		}
	}()
	
	ctx := context.Background()
	
	// 1. Get current market data
	klines, err := bot.getRecentKlines()
	if err != nil {
		bot.logger.Error("Failed to get market data: %v", err)
		return
	}
	
	if len(klines) < 50 {
		bot.logger.LogWarning("Insufficient data", "Only %d data points available", len(klines))
		return
	}
	
	// 2. Update market data buffer
	bot.updateMarketDataBuffer(klines)
	
	// 3. Detect current market regime
	bot.detectAndUpdateRegime()
	
	// 4. Select optimal engine based on regime
	bot.selectOptimalEngine()
	
	// 5. Get current market price
	currentPrice, err := bot.exchange.GetLatestPrice(ctx, bot.symbol)
	if err != nil {
		bot.logger.Error("Failed to get current price: %v", err)
		return
	}
	
	// 6. Analyze market with active engine
	signal, err := bot.analyzeWithActiveEngine(klines, currentPrice)
	if err != nil {
		bot.logger.Error("Failed to analyze market: %v", err)
		return
	}
	
	// 7. Execute trading decision if signal is strong enough
	bot.executeSignal(signal, currentPrice)
	
	// 8. Manage positions across all engines
	bot.manageAllPositions(klines)
	
	// 9. Update performance metrics
	bot.updateMetrics()
}

// selectOptimalEngine chooses the best engine based on current regime
func (bot *DualEngineBot) selectOptimalEngine() {
	bot.engineMutex.Lock()
	defer bot.engineMutex.Unlock()
	
	bot.regimeMutex.RLock()
	currentRegime := bot.currentRegime
	lastRegimeSignal := bot.lastRegimeSignal
	bot.regimeMutex.RUnlock()
	
	if lastRegimeSignal == nil {
		return // No regime data yet
	}
	
	// Check if we should consider switching engines
	if time.Since(bot.lastTransition) < bot.transitionCooldown {
		return // Still in cooldown period
	}
	
	// Find the engine with the highest compatibility score for current regime
	var bestEngine engines.TradingEngine
	var bestScore float64 = -1
	var bestEngineType engines.EngineType
	
	for engineType, engine := range bot.engines {
		score := engine.GetRegimeCompatibilityScore(currentRegime)
		
		// Bonus for currently active engine to prevent unnecessary switches
		if engine == bot.activeEngine {
			score += 0.1 // 10% bonus for stability
		}
		
		if score > bestScore {
			bestScore = score
			bestEngine = engine
			bestEngineType = engineType
		}
	}
	
	// Switch engines if the best engine is different and confidence is high enough
	if bestEngine != bot.activeEngine && lastRegimeSignal.Confidence >= 0.7 && bestScore >= 0.5 {
		bot.switchEngine(bestEngine, bestEngineType, currentRegime)
	}
}

// switchEngine performs engine transition
func (bot *DualEngineBot) switchEngine(newEngine engines.TradingEngine, engineType engines.EngineType, regime regime.RegimeType) {
	oldEngine := bot.activeEngine
	oldEngineType := oldEngine.GetType()
	
	bot.logger.Trade("🔄 ENGINE SWITCH: %s → %s (Regime: %s, Compatibility: %.1f%%)",
		oldEngine.GetName(),
		newEngine.GetName(),
		regime.String(),
		newEngine.GetRegimeCompatibilityScore(regime)*100,
	)
	
	// Deactivate old engine
	oldEngine.SetActive(false)
	
	// Activate new engine
	newEngine.SetActive(true)
	bot.activeEngine = newEngine
	
	// Update metrics
	bot.metricsMutex.Lock()
	bot.orchestratorMetrics.EngineActivations[engineType]++
	bot.orchestratorMetrics.LastTransition = time.Now()
	bot.lastTransition = time.Now()
	
	// Update engine performance summary
	if summary, exists := bot.orchestratorMetrics.EnginePerformance[engineType]; exists {
		summary.ActivationCount++
		summary.LastActivation = time.Now()
		bot.orchestratorMetrics.EnginePerformance[engineType] = summary
	}
	
	bot.metricsMutex.Unlock()
	
	// Log to console for visibility
	fmt.Printf("🔄 ENGINE SWITCH: %s → %s (%s regime)\n", 
		oldEngineType.String(), engineType.String(), regime.String())
}

// analyzeWithActiveEngine gets analysis from the currently active engine
func (bot *DualEngineBot) analyzeWithActiveEngine(klines []types.OHLCV, currentPrice float64) (engines.EngineSignal, error) {
	bot.engineMutex.RLock()
	activeEngine := bot.activeEngine
	currentRegime := bot.currentRegime
	bot.engineMutex.RUnlock()
	
	ctx := context.Background()
	
	// Convert klines to 30m and 5m data (simplified for now)
	data30m := klines // TODO: Implement proper timeframe separation
	data5m := klines
	
	signal, err := activeEngine.AnalyzeMarket(ctx, data30m, data5m, currentRegime)
	if err != nil {
		return nil, fmt.Errorf("engine analysis failed: %w", err)
	}
	
	return signal, nil
}

// executeSignal executes trading signal if it meets criteria
func (bot *DualEngineBot) executeSignal(signal engines.EngineSignal, currentPrice float64) {
	if signal == nil || signal.GetAction() == "HOLD" || signal.GetConfidence() < 0.5 {
		return
	}
	
	bot.metricsMutex.Lock()
	bot.orchestratorMetrics.TotalSignals++
	bot.metricsMutex.Unlock()
	
	bot.logger.Trade("📊 SIGNAL: %s (Confidence: %.1f%%, Strength: %.1f%%) from %s Engine",
		signal.GetAction(),
		signal.GetConfidence()*100,
		signal.GetStrength()*100,
		bot.activeEngine.GetName(),
	)
	
	// TODO: Implement actual trade execution via exchange
	// For now, just log the signal
}

// manageAllPositions manages positions across all engines
func (bot *DualEngineBot) manageAllPositions(currentData []types.OHLCV) {
	ctx := context.Background()
	
	for engineType, engine := range bot.engines {
		if err := engine.ManagePositions(ctx, currentData); err != nil {
			bot.logger.Error("Failed to manage positions for %s engine: %v", engineType, err)
		}
	}
}

// Helper methods from original LiveBot (adapted)

func (bot *DualEngineBot) getRecentKlines() ([]types.OHLCV, error) {
	// TODO: Implement based on original LiveBot logic
	// This would fetch recent klines from the exchange
	return []types.OHLCV{}, nil
}

func (bot *DualEngineBot) updateMarketDataBuffer(klines []types.OHLCV) {
	bot.bufferMutex.Lock()
	defer bot.bufferMutex.Unlock()
	
	if len(klines) > 0 {
		latestCandle := klines[len(klines)-1]
		bot.marketDataBuffer = append(bot.marketDataBuffer, latestCandle)
		
		if len(bot.marketDataBuffer) > bot.bufferSize {
			excess := len(bot.marketDataBuffer) - bot.bufferSize
			bot.marketDataBuffer = bot.marketDataBuffer[excess:]
		}
	}
}

func (bot *DualEngineBot) detectAndUpdateRegime() {
	bot.bufferMutex.RLock()
	dataBuffer := make([]types.OHLCV, len(bot.marketDataBuffer))
	copy(dataBuffer, bot.marketDataBuffer)
	bot.bufferMutex.RUnlock()
	
	if len(dataBuffer) < 250 {
		return // Not enough data yet
	}
	
	signal, err := bot.regimeDetector.DetectRegime(dataBuffer)
	if err != nil {
		bot.logger.LogWarning("Regime Detection Error", "%v", err)
		return
	}
	
	bot.regimeMutex.Lock()
	defer bot.regimeMutex.Unlock()
	
	oldRegime := bot.currentRegime
	bot.currentRegime = signal.Type
	bot.lastRegimeSignal = signal
	
	if oldRegime != signal.Type && oldRegime != regime.RegimeUncertain {
		regimeChange := &regime.RegimeChange{
			Timestamp:    signal.Timestamp,
			OldRegime:    oldRegime,
			NewRegime:    signal.Type,
			Confidence:   signal.Confidence,
			Reason:       "Orchestrator detected regime change",
			TriggerPrice: dataBuffer[len(dataBuffer)-1].Close,
		}
		
		bot.regimeHistory = append(bot.regimeHistory, regimeChange)
		if len(bot.regimeHistory) > 50 {
			bot.regimeHistory = bot.regimeHistory[1:]
		}
		
		bot.orchestratorMetrics.RegimeChanges++
		
		bot.logger.Trade("🔄 REGIME CHANGE: %s → %s (Confidence: %.1f%%)",
			oldRegime.String(),
			signal.Type.String(),
			signal.Confidence*100,
		)
	}
}

func (bot *DualEngineBot) updateMetrics() {
	bot.metricsMutex.Lock()
	defer bot.metricsMutex.Unlock()
	
	// Update total PnL from all engines
	totalPnL := 0.0
	for _, engine := range bot.engines {
		metrics := engine.GetPerformanceMetrics()
		totalPnL += metrics.GetTotalPnL()
	}
	
	bot.orchestratorMetrics.TotalPnL = totalPnL
}

// Public API methods

func (bot *DualEngineBot) GetCurrentRegime() regime.RegimeType {
	bot.regimeMutex.RLock()
	defer bot.regimeMutex.RUnlock()
	return bot.currentRegime
}

func (bot *DualEngineBot) GetActiveEngine() engines.TradingEngine {
	bot.engineMutex.RLock()
	defer bot.engineMutex.RUnlock()
	return bot.activeEngine
}

func (bot *DualEngineBot) GetOrchestrationMetrics() *OrchestratorMetrics {
	bot.metricsMutex.RLock()
	defer bot.metricsMutex.RUnlock()
	
	// Return a copy to prevent concurrent access issues
	metricsCopy := *bot.orchestratorMetrics
	return &metricsCopy
}

func (bot *DualEngineBot) GetEngineStatuses() map[engines.EngineType]engines.EngineStatus {
	statuses := make(map[engines.EngineType]engines.EngineStatus)
	
	for engineType, engine := range bot.engines {
		statuses[engineType] = engine.GetEngineStatus()
	}
	
	return statuses
}

func (bot *DualEngineBot) IsRunning() bool {
	bot.controlMutex.RLock()
	defer bot.controlMutex.RUnlock()
	return bot.running
}
