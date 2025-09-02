package engines

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// TrendEngine implements directional trading for trending markets
// This corresponds to the trend engine specifications from the plan
type TrendEngine struct {
	// Configuration from plan specifications
	biasTimeframe      string    // 30m for trend bias
	executionTimeframes []string // [5m, 1m] for execution
	
	// Entry conditions from plan
	pullbackLevels     []float64 // [0.382, 0.618] Fibonacci levels
	momentumIndicators []string  // [macd_histogram, rsi_5]
	entryMethods       []string  // [limit_first, market_fallback]
	
	// Risk management from plan
	stopLossMethod     string    // swing_low_or_atr
	atrMultiplier      float64   // 1.2
	takeProfitScaling  []float64 // [0.5, 1.0]
	trailingMethod     string    // atr_or_chandelier
	chandelierMultiplier float64 // 3.0
	
	// Position management from plan
	maxAddOns          int       // 2
	addOnConditions    []string  // [unrealized_pnl_positive, adx_rising]
	
	// Internal state
	indicatorManager   *indicators.IndicatorManager
	currentPositions   []*TrendPosition
	lastSignal         *TrendSignal
	performanceMetrics *TrendEngineMetrics
	
	// Interface compatibility
	active             bool
	engineStatus       EngineStatus
	riskLimits         EngineRiskLimits
	mutex              sync.RWMutex
}

// TrendPosition represents a position managed by the trend engine
type TrendPosition struct {
	ID                 string              `json:"id"`
	EntryPrice         float64             `json:"entry_price"`
	CurrentPrice       float64             `json:"current_price"`
	Size               float64             `json:"size"`
	EntryTime          time.Time           `json:"entry_time"`
	Direction          int                 `json:"direction"`        // 1 = long, -1 = short
	StopLoss           float64             `json:"stop_loss"`
	TakeProfit         []float64           `json:"take_profit"`
	TrailingStop       float64             `json:"trailing_stop,omitempty"`
	AddOnCount         int                 `json:"add_on_count"`
	UnrealizedPnL      float64             `json:"unrealized_pnl"`
	
	// Trend-specific data
	TrendStrength      float64             `json:"trend_strength"`
	PullbackLevel      float64             `json:"pullback_level"`   // Fib level used for entry
	MomentumScore      float64             `json:"momentum_score"`
}

// TrendSignal represents a trend trading signal
type TrendSignal struct {
	Timestamp          time.Time           `json:"timestamp"`
	Direction          int                 `json:"direction"`        // 1 = long, -1 = short, 0 = no signal
	Confidence         float64             `json:"confidence"`       // 0-1
	Strength           float64             `json:"strength"`         // Trend strength
	EntryMethod        string              `json:"entry_method"`     // limit or market
	SuggestedEntry     float64             `json:"suggested_entry"`
	SuggestedStopLoss  float64             `json:"suggested_stop_loss"`
	SuggestedTakeProfit []float64          `json:"suggested_take_profit"`
	
	// Supporting data
	PullbackLevel      float64             `json:"pullback_level"`
	MomentumConfirmation bool              `json:"momentum_confirmation"`
	TrendBias          int                 `json:"trend_bias"`       // From 30m timeframe
}

// TrendEngineMetrics holds performance metrics for the trend engine
type TrendEngineMetrics struct {
	ActivePositions    int                 `json:"active_positions"`
	TotalTrades        int                 `json:"total_trades"`
	WinRate            float64             `json:"win_rate"`
	AverageWin         float64             `json:"average_win"`
	AverageLoss        float64             `json:"average_loss"`
	ProfitFactor       float64             `json:"profit_factor"`
	TotalPnL           float64             `json:"total_pnl"`
	MaxDrawdown        float64             `json:"max_drawdown"`
	SharpeRatio        float64             `json:"sharpe_ratio"`
	
	// Trend-specific metrics
	AverageTrendDuration time.Duration     `json:"average_trend_duration"`
	PullbackSuccessRate  float64           `json:"pullback_success_rate"`
	AddOnSuccessRate     float64           `json:"add_on_success_rate"`
}

// NewTrendEngine creates a new trend engine with default parameters from the plan
func NewTrendEngine() *TrendEngine {
	return &TrendEngine{
		biasTimeframe:        "30m",
		executionTimeframes:  []string{"5m", "1m"},
		pullbackLevels:       []float64{0.382, 0.618},
		momentumIndicators:   []string{"macd_histogram", "rsi_5"},
		entryMethods:         []string{"limit_first", "market_fallback"},
		stopLossMethod:       "swing_low_or_atr",
		atrMultiplier:        1.2,
		takeProfitScaling:    []float64{0.5, 1.0},
		trailingMethod:       "atr_or_chandelier",
		chandelierMultiplier: 3.0,
		maxAddOns:            2,
		addOnConditions:      []string{"unrealized_pnl_positive", "adx_rising"},
		indicatorManager:     indicators.NewIndicatorManager(),
		currentPositions:     make([]*TrendPosition, 0),
		performanceMetrics:   &TrendEngineMetrics{},
	}
}

// AnalyzeMarket analyzes market conditions for trending opportunities
func (te *TrendEngine) AnalyzeMarket(data30m, data5m, data1m []types.OHLCV) (*TrendSignal, error) {
	if len(data30m) < 50 || len(data5m) < 20 || len(data1m) < 10 {
		return nil, fmt.Errorf("insufficient data for trend analysis")
	}
	
	// TODO: Phase 2 Implementation
	// This will be implemented in Phase 2 according to the plan
	
	signal := &TrendSignal{
		Timestamp:           data5m[len(data5m)-1].Timestamp,
		Direction:           0,
		Confidence:          0.0,
		Strength:            0.0,
		EntryMethod:         "limit_first",
		SuggestedEntry:      data5m[len(data5m)-1].Close,
		SuggestedStopLoss:   0.0,
		SuggestedTakeProfit: []float64{},
		PullbackLevel:       0.0,
		MomentumConfirmation: false,
		TrendBias:           0,
	}
	
	te.lastSignal = signal
	return signal, nil
}

// GenerateEntry generates entry signals based on pullback detection
func (te *TrendEngine) GenerateEntry(signal *TrendSignal) (*TrendEntryOrder, error) {
	if signal.Direction == 0 {
		return nil, fmt.Errorf("no trend direction identified")
	}
	
	// TODO: Phase 2 Implementation
	// Implement pullback detection and entry logic
	
	order := &TrendEntryOrder{
		Type:        "limit",
		Direction:   signal.Direction,
		Price:       signal.SuggestedEntry,
		Quantity:    te.calculatePositionSize(signal),
		StopLoss:    signal.SuggestedStopLoss,
		TakeProfit:  signal.SuggestedTakeProfit,
		TimeLimit:   5 * time.Minute,
		Pullback:    signal.PullbackLevel,
	}
	
	return order, nil
}

// ManagePositions handles position management for active trend positions
func (te *TrendEngine) ManagePositions(currentData []types.OHLCV) error {
	if len(currentData) == 0 {
		return fmt.Errorf("no market data provided")
	}
	
	currentPrice := currentData[len(currentData)-1].Close
	
	for _, position := range te.currentPositions {
		// Update current price and unrealized PnL
		position.CurrentPrice = currentPrice
		position.UnrealizedPnL = te.calculateUnrealizedPnL(position, currentPrice)
		
		// Check for trailing stop updates
		if err := te.updateTrailingStop(position, currentData); err != nil {
			// Log error but continue with other positions
			continue
		}
		
		// Check for add-on opportunities
		if position.AddOnCount < te.maxAddOns {
			if te.shouldAddToPosition(position, currentData) {
				// TODO: Phase 2 - Implement add-on logic
			}
		}
		
		// Check for exit conditions
		if te.shouldExitPosition(position, currentData) {
			// TODO: Phase 2 - Implement exit logic
		}
	}
	
	return nil
}

// IsCompatibleWithRegime checks if trend engine is suitable for current regime
func (te *TrendEngine) IsCompatibleWithRegime(regimeType regime.RegimeType) bool {
	// Trend engine is optimized for trending markets
	return regimeType == regime.RegimeTrending
}

// GetCurrentPositions returns all active trend positions
func (te *TrendEngine) GetCurrentPositions() []*TrendPosition {
	return te.currentPositions
}

// GetPerformanceMetrics returns current performance metrics
func (te *TrendEngine) GetPerformanceMetrics() *TrendEngineMetrics {
	return te.performanceMetrics
}

// GetLastSignal returns the most recent trend signal
func (te *TrendEngine) GetLastSignal() *TrendSignal {
	return te.lastSignal
}

// Helper methods

func (te *TrendEngine) calculatePositionSize(signal *TrendSignal) float64 {
	// TODO: Phase 2 Implementation
	// Calculate position size based on trend strength, volatility, and risk parameters
	baseSize := 100.0 // Placeholder base size
	
	// Adjust based on trend strength
	strengthMultiplier := 0.5 + (signal.Strength * 0.5) // 0.5 to 1.0 range
	
	// Adjust based on confidence
	confidenceMultiplier := 0.3 + (signal.Confidence * 0.7) // 0.3 to 1.0 range
	
	return baseSize * strengthMultiplier * confidenceMultiplier
}

func (te *TrendEngine) calculateUnrealizedPnL(position *TrendPosition, currentPrice float64) float64 {
	priceDiff := currentPrice - position.EntryPrice
	if position.Direction == -1 {
		priceDiff = -priceDiff
	}
	return (priceDiff / position.EntryPrice) * position.Size
}

func (te *TrendEngine) updateTrailingStop(position *TrendPosition, data []types.OHLCV) error {
	// TODO: Phase 2 Implementation
	// Implement ATR or Chandelier trailing stop logic
	return nil
}

func (te *TrendEngine) shouldAddToPosition(position *TrendPosition, data []types.OHLCV) bool {
	// TODO: Phase 2 Implementation
	// Check add-on conditions: unrealized PnL positive and ADX rising
	return false
}

func (te *TrendEngine) shouldExitPosition(position *TrendPosition, data []types.OHLCV) bool {
	// TODO: Phase 2 Implementation
	// Check stop loss, take profit, and trend reversal conditions
	return false
}

// TrendEntryOrder represents an entry order for the trend engine
type TrendEntryOrder struct {
	Type        string    `json:"type"`         // limit or market
	Direction   int       `json:"direction"`    // 1 = long, -1 = short
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	StopLoss    float64   `json:"stop_loss"`
	TakeProfit  []float64 `json:"take_profit"`
	TimeLimit   time.Duration `json:"time_limit"`
	
	// Trend-specific
	Pullback    float64   `json:"pullback"`     // Fibonacci pullback level
}

// TradingEngine interface implementation for TrendEngine

func (te *TrendEngine) GetType() EngineType {
	return EngineTypeTrend
}

func (te *TrendEngine) GetName() string {
	return "Multi-Timeframe Trend Engine"
}

func (te *TrendEngine) GetPreferredRegimes() []regime.RegimeType {
	return []regime.RegimeType{regime.RegimeTrending}
}

func (te *TrendEngine) GetRegimeCompatibilityScore(regimeType regime.RegimeType) float64 {
	switch regimeType {
	case regime.RegimeTrending:
		return 1.0  // Perfect fit
	case regime.RegimeVolatile:
		return 0.3  // Some compatibility with volatile trending markets
	case regime.RegimeRanging:
		return 0.1  // Poor fit
	case regime.RegimeUncertain:
		return 0.0  // No fit
	default:
		return 0.0
	}
}

// AnalyzeMarketForEngine implements the main analysis method required by TradingEngine interface
func (te *TrendEngine) AnalyzeMarketForEngine(ctx context.Context, data30m, data5m []types.OHLCV, currentRegime regime.RegimeType) (EngineSignal, error) {
	_ = ctx // Context parameter reserved for future use (timeout handling, cancellation)
	
	te.mutex.Lock()
	defer te.mutex.Unlock()
	
	// Check if engine is active and compatible with current regime
	if !te.active || !te.IsCompatibleWithRegime(currentRegime) {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "Engine inactive or incompatible with current regime",
		}, nil
	}
	
	// Ensure we have enough data for analysis
	if len(data30m) < 50 || len(data5m) < 20 {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "Insufficient data for trend analysis",
		}, nil
	}
	
	// Use existing AnalyzeMarket method and convert result
	trendSignal, err := te.AnalyzeMarket(data30m, data5m, []types.OHLCV{}) // Pass empty 1m data
	if err != nil {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     fmt.Sprintf("Analysis error: %v", err),
		}, nil
	}
	
	if trendSignal == nil {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "No trend signal generated",
		}, nil
	}
	
	// Convert TrendSignal to BasicEngineSignal
	action := "HOLD"
	if trendSignal.Direction == 1 {
		action = "BUY"
	} else if trendSignal.Direction == -1 {
		action = "SELL"
	}
	
	return &BasicEngineSignal{
		Timestamp:  trendSignal.Timestamp,
		Confidence: trendSignal.Confidence,
		Strength:   trendSignal.Strength,
		Direction:  trendSignal.Direction,
		Action:     action,
		Price:      0.0, // Use market price
		Size:       te.calculatePositionSize(trendSignal),
		StopLoss:   0.0,  // TODO: Calculate from ATR
		TakeProfit: []float64{}, // TODO: Calculate TP levels
		Reason:     "Trend signal generated",
		Metadata:   map[string]interface{}{
			"timeframe_30m": "bias",
			"timeframe_5m":  "execution",
		},
	}, nil
}

// AnalyzeMarketWithData implements multi-timeframe analysis with custom data map
func (te *TrendEngine) AnalyzeMarketWithData(ctx context.Context, data map[string][]types.OHLCV, currentRegime regime.RegimeType) (EngineSignal, error) {
	_ = ctx // Context parameter reserved for future use
	// Extract required timeframes
	data30m, has30m := data["30m"]
	data5m, has5m := data["5m"]
	
	if !has30m || !has5m {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "Missing required timeframe data (30m, 5m)",
		}, nil
	}
	
	return te.AnalyzeMarketForEngine(ctx, data30m, data5m, currentRegime)
}

func (te *TrendEngine) ManagePositionsForEngine(ctx context.Context, currentData []types.OHLCV) error {
	_ = ctx // Context parameter reserved for future use
	
	te.mutex.Lock()
	defer te.mutex.Unlock()
	
	if len(currentData) == 0 {
		return fmt.Errorf("no data provided for position management")
	}
	
	// Use existing ManagePositions method
	return te.ManagePositions(currentData)
}

func (te *TrendEngine) ShouldClosePosition(position EnginePosition, currentPrice float64, currentRegime regime.RegimeType) bool {
	// If regime changes away from trending, consider closing
	if !te.IsCompatibleWithRegime(currentRegime) {
		return true
	}
	
	// TODO: Implement specific close logic based on trend engine criteria
	// For now, use basic stop loss check
	if basicPos, ok := position.(*BasicEnginePosition); ok {
		if basicPos.Side == "long" && currentPrice < basicPos.EntryPrice*0.98 { // 2% stop loss
			return true
		}
		if basicPos.Side == "short" && currentPrice > basicPos.EntryPrice*1.02 { // 2% stop loss
			return true
		}
	}
	
	return false
}

func (te *TrendEngine) CalculatePositionSize(balance float64, price float64, currentRegime regime.RegimeType) float64 {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	
	// Base position size as percentage of balance
	basePercent := 0.02 // 2% of balance
	
	// Adjust based on regime compatibility
	regimeScore := te.GetRegimeCompatibilityScore(currentRegime)
	adjustedPercent := basePercent * regimeScore
	
	// Calculate position size in base units
	positionValue := balance * adjustedPercent
	positionSize := positionValue / price
	
	// Apply maximum limits
	maxSize := balance * 0.1 / price // Never more than 10% of balance
	if positionSize > maxSize {
		positionSize = maxSize
	}
	
	return positionSize
}

func (te *TrendEngine) ValidateSignal(signal EngineSignal, currentPositions []EnginePosition) error {
	// Check signal confidence
	if signal.GetConfidence() < 0.5 {
		return fmt.Errorf("signal confidence too low: %.2f", signal.GetConfidence())
	}
	
	// Check if we already have too many positions
	if len(currentPositions) >= te.maxAddOns+1 {
		return fmt.Errorf("maximum positions reached: %d", len(currentPositions))
	}
	
	return nil
}

func (te *TrendEngine) GetEngineStatus() EngineStatus {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	
	status := te.engineStatus
	status.ActivePositions = len(te.currentPositions)
	status.IsActive = te.active
	status.LastActivity = time.Now()
	
	return status
}

func (te *TrendEngine) UpdateConfiguration(config map[string]interface{}) error {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	
	// Update configuration fields from map
	if val, ok := config["atr_multiplier"]; ok {
		if atrMult, ok := val.(float64); ok {
			te.atrMultiplier = atrMult
		}
	}
	
	if val, ok := config["max_add_ons"]; ok {
		if maxAddOns, ok := val.(int); ok {
			te.maxAddOns = maxAddOns
		}
	}
	
	// Add more configuration updates as needed
	
	return nil
}

func (te *TrendEngine) GetConfig() map[string]interface{} {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	
	return map[string]interface{}{
		"bias_timeframe":        te.biasTimeframe,
		"execution_timeframes":  te.executionTimeframes,
		"atr_multiplier":        te.atrMultiplier,
		"max_add_ons":          te.maxAddOns,
		"pullback_levels":      te.pullbackLevels,
		"momentum_indicators":  te.momentumIndicators,
		"stop_loss_method":     te.stopLossMethod,
		"trailing_method":      te.trailingMethod,
	}
}

func (te *TrendEngine) SetRiskLimits(limits EngineRiskLimits) error {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	
	te.riskLimits = limits
	return nil
}

func (te *TrendEngine) Initialize() error {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	
	te.currentPositions = []*TrendPosition{}
	te.performanceMetrics = &TrendEngineMetrics{}
	te.engineStatus = EngineStatus{}
	te.active = false
	
	return nil
}

func (te *TrendEngine) Start() error {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	
	te.active = true
	te.engineStatus.IsActive = true
	te.engineStatus.IsTrading = true
	
	return nil
}

func (te *TrendEngine) Stop() error {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	
	te.active = false
	te.engineStatus.IsActive = false
	te.engineStatus.IsTrading = false
	
	return nil
}

func (te *TrendEngine) Reset() error {
	return te.Initialize()
}

func (te *TrendEngine) IsActive() bool {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	return te.active
}

func (te *TrendEngine) SetActive(active bool) {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	te.active = active
	te.engineStatus.IsActive = active
}

// Helper methods for the new interface implementation

func (te *TrendEngine) updateIndicators(data30m, data5m []types.OHLCV) error {
	// TODO: Update all indicators with latest data
	// This would update EMAs, MACD, ADX, RSI, etc. with the new candle data
	return nil
}

func (te *TrendEngine) analyze30mBias(data []types.OHLCV) *TrendSignal {
	// TODO: Implement 30m bias analysis
	// Check EMA alignment, MACD direction, ADX strength
	return nil
}

func (te *TrendEngine) analyze5mExecution(data []types.OHLCV) *TrendSignal {
	// TODO: Implement 5m execution analysis
	// Look for pullbacks, RSI oversold/overbought, entry timing
	return nil
}

func (te *TrendEngine) combineTimeframeSignals(bias30m, signal5m *TrendSignal, currentPrice float64) *TrendSignal {
	// TODO: Combine 30m bias with 5m execution signal
	// Only take trades that align with higher timeframe bias
	return nil
}

func (te *TrendEngine) applySignalFilters(signal *TrendSignal, data []types.OHLCV) *TrendSignal {
	// TODO: Apply volume filters, momentum filters, etc.
	return signal
}

// Convert TrendPosition to EnginePosition interface
func (tp *TrendPosition) ToEnginePosition() EnginePosition {
	side := "long"
	if tp.Direction == -1 {
		side = "short"
	}
	
	return &BasicEnginePosition{
		ID:            tp.ID,
		EngineType:    "trend",
		Symbol:        "", // TODO: Add symbol to TrendPosition
		Side:          side,
		Size:          tp.Size,
		EntryPrice:    tp.EntryPrice,
		CurrentPrice:  tp.CurrentPrice,
		UnrealizedPnL: tp.UnrealizedPnL,
		EntryTime:     tp.EntryTime,
		Metadata:      map[string]interface{}{}, // Empty metadata for now
	}
}

// GetCurrentPositionsForEngine returns interface types
func (te *TrendEngine) GetCurrentPositionsForEngine() []EnginePosition {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	
	positions := make([]EnginePosition, len(te.currentPositions))
	for i, pos := range te.currentPositions {
		positions[i] = pos.ToEnginePosition()
	}
	
	return positions
}

func (te *TrendEngine) GetPerformanceMetricsForEngine() EngineMetrics {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	
	// Convert TrendEngineMetrics to BasicEngineMetrics
	// Since TrendEngineMetrics might not have all fields, use defaults
	return &BasicEngineMetrics{
		ActivePositions: len(te.currentPositions),
		TotalTrades:    te.performanceMetrics.TotalTrades,
		WinningTrades:  0, // TODO: Add these fields to TrendEngineMetrics
		LosingTrades:   0, // TODO: Add these fields to TrendEngineMetrics  
		WinRate:       te.performanceMetrics.WinRate,
		ProfitFactor:  te.performanceMetrics.ProfitFactor,
		TotalPnL:     te.performanceMetrics.TotalPnL,
		MaxDrawdown:  te.performanceMetrics.MaxDrawdown,
		SharpeRatio:  te.performanceMetrics.SharpeRatio,
		LastUpdated:  time.Now(),
	}
}

// TrendEngineAdapter wraps TrendEngine to implement TradingEngine interface
type TrendEngineAdapter struct {
	*TrendEngine
}

// NewTrendEngineAdapter creates a new adapter around the TrendEngine
func NewTrendEngineAdapter(symbol string) (*TrendEngineAdapter, error) {
	trendEngine := &TrendEngine{
		biasTimeframe:        "30m",
		executionTimeframes:  []string{"5m", "1m"},
		pullbackLevels:      []float64{0.382, 0.618},
		momentumIndicators:  []string{"macd_histogram", "rsi_5"},
		entryMethods:        []string{"limit_first", "market_fallback"},
		stopLossMethod:      "swing_low_or_atr", 
		atrMultiplier:       1.2,
		takeProfitScaling:   []float64{0.5, 1.0},
		trailingMethod:      "atr_or_chandelier",
		chandelierMultiplier: 3.0,
		maxAddOns:           2,
		addOnConditions:     []string{"unrealized_pnl_positive", "adx_rising"},
		currentPositions:    []*TrendPosition{},
		performanceMetrics:  &TrendEngineMetrics{},
		active:              false,
		engineStatus:        EngineStatus{},
		riskLimits:          EngineRiskLimits{},
	}
	
	return &TrendEngineAdapter{TrendEngine: trendEngine}, nil
}

// Implement TradingEngine interface methods by delegating to TrendEngine

func (tea *TrendEngineAdapter) AnalyzeMarket(ctx context.Context, data30m, data5m []types.OHLCV, currentRegime regime.RegimeType) (EngineSignal, error) {
	return tea.AnalyzeMarketForEngine(ctx, data30m, data5m, currentRegime)
}

func (tea *TrendEngineAdapter) ManagePositions(ctx context.Context, currentData []types.OHLCV) error {
	return tea.ManagePositionsForEngine(ctx, currentData)
}

func (tea *TrendEngineAdapter) GetCurrentPositions() []EnginePosition {
	return tea.GetCurrentPositionsForEngine()
}

func (tea *TrendEngineAdapter) GetPerformanceMetrics() EngineMetrics {
	return tea.GetPerformanceMetricsForEngine()
}
