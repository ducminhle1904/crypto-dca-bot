package engines

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// GridEngine implements mean-reversion trading with hedge grid approach for ranging markets
// Based on the original dual_engine_regime_bot_plan.json specifications
type GridEngine struct {
	// Basic engine info
	engineType    EngineType
	name          string
	symbol        string
	active        bool
	
	// Configuration from plan specifications
	anchorMethods         []string  // [anchored_vwap, ema_100]
	atrMultiplier         float64   // 0.75 for band calculation
	gridSpacingMultiplier float64   // 0.25 for grid level spacing
	maxBands              int       // 4 maximum grid bands
	
	// Hedge management from plan
	symmetricPlacement bool      // True for symmetric grid placement
	maxLongNotional    float64   // 0.4 max long exposure
	maxShortNotional   float64   // 0.4 max short exposure
	maxNetExposure     float64   // 0.1 max net exposure
	
	// Exit conditions from plan
	takeProfitMultiplier float64 // 0.7
	stopLossMultiplier   float64 // 1.5
	timeBasedExit        bool    // True for time-based exits
	maxBarsInTrade       int     // 48 bars maximum trade duration
	
	// Safety mechanisms from plan
	bbWidthExitThreshold float64 // 1.5 for BB width exit
	adxPopupThreshold    float64 // 22 for trend popup detection
	regimeFlipExit       bool    // True to exit on regime change
	
	// Internal state
	indicatorManager     *indicators.IndicatorManager
	gridLevels           []*GridLevel
	activePositions      []*GridPosition
	anchorPrice          float64
	lastRebalance        time.Time
	
	// Performance tracking
	performanceMetrics   *GridEngineMetrics
	status               EngineStatus
	riskLimits           EngineRiskLimits
	
	// Thread safety
	mutex                sync.RWMutex
	positionMutex        sync.RWMutex
	
	// VWAP calculation state
	vwapData            VWAPData
	emaIndicator        *indicators.EMA
}

// GridLevel represents a single level in the grid
type GridLevel struct {
	ID          string    `json:"id"`
	Price       float64   `json:"price"`
	Side        string    `json:"side"`        // "buy" or "sell"
	Quantity    float64   `json:"quantity"`
	Status      string    `json:"status"`      // "pending", "filled", "cancelled"
	OrderID     string    `json:"order_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	FilledAt    *time.Time `json:"filled_at,omitempty"`
}

// GridPosition represents an active grid position
type GridPosition struct {
	ID              string    `json:"id"`
	Side            string    `json:"side"`        // "long" or "short"
	EntryPrice      float64   `json:"entry_price"`
	Quantity        float64   `json:"quantity"`
	CurrentPrice    float64   `json:"current_price"`
	UnrealizedPnL   float64   `json:"unrealized_pnl"`
	EntryTime       time.Time `json:"entry_time"`
	GridLevelID     string    `json:"grid_level_id"`
	ExitTarget      float64   `json:"exit_target"`    // TP level
	StopLoss        float64   `json:"stop_loss"`      // SL level
}

// GridEngineMetrics tracks grid engine performance
type GridEngineMetrics struct {
	TotalTrades       int       `json:"total_trades"`
	WinningTrades     int       `json:"winning_trades"`
	LosingTrades      int       `json:"losing_trades"`
	TotalPnL          float64   `json:"total_pnl"`
	TotalVolume       float64   `json:"total_volume"`
	AverageTradePnL   float64   `json:"average_trade_pnl"`
	LargestWin        float64   `json:"largest_win"`
	LargestLoss       float64   `json:"largest_loss"`
	GridRebalances    int       `json:"grid_rebalances"`
	ActiveLegs        int       `json:"active_legs"`
	NetExposure       float64   `json:"net_exposure"`
	LongExposure      float64   `json:"long_exposure"`
	ShortExposure     float64   `json:"short_exposure"`
}

// VWAPData holds VWAP calculation data
type VWAPData struct {
	PriceVolumeSum    float64
	VolumeSum         float64
	CurrentVWAP       float64
	LastUpdate        time.Time
	SessionStart      time.Time
}

// GridEngineConfig holds configuration for the grid engine
type GridEngineConfig struct {
	// Grid parameters
	MaxBands              int       `json:"max_bands"`               // Maximum grid bands (default: 4)
	GridSpacingMultiplier float64   `json:"grid_spacing_multiplier"` // Grid spacing as ATR multiplier (default: 0.25)
	ATRMultiplier         float64   `json:"atr_multiplier"`          // ATR multiplier for band calculation (default: 0.75)
	
	// Hedge management
	MaxLongNotional       float64   `json:"max_long_notional"`       // Max long exposure (default: 0.4)
	MaxShortNotional      float64   `json:"max_short_notional"`      // Max short exposure (default: 0.4) 
	MaxNetExposure        float64   `json:"max_net_exposure"`        // Max net exposure (default: 0.1)
	
	// Exit conditions
	TakeProfitMultiplier  float64   `json:"take_profit_multiplier"`  // TP multiplier (default: 0.7)
	StopLossMultiplier    float64   `json:"stop_loss_multiplier"`    // SL multiplier (default: 1.5)
	MaxBarsInTrade        int       `json:"max_bars_in_trade"`       // Max trade duration (default: 48)
	
	// Safety mechanisms  
	BBWidthExitThreshold  float64   `json:"bb_width_exit_threshold"` // BB width threshold for exits (default: 1.5)
	ADXPopupThreshold     float64   `json:"adx_popup_threshold"`     // ADX threshold for trend detection (default: 22)
}

// DefaultGridEngineConfig returns default grid engine configuration
func DefaultGridEngineConfig() *GridEngineConfig {
	return &GridEngineConfig{
		MaxBands:              4,      // 4 grid bands maximum
		GridSpacingMultiplier: 0.25,   // 0.25x ATR spacing
		ATRMultiplier:         0.75,   // 0.75x ATR for bands
		MaxLongNotional:       0.4,    // 40% max long exposure
		MaxShortNotional:      0.4,    // 40% max short exposure
		MaxNetExposure:        0.1,    // 10% max net exposure
		TakeProfitMultiplier:  0.7,    // 0.7x grid spacing for TP
		StopLossMultiplier:    1.5,    // 1.5x grid spacing for SL
		MaxBarsInTrade:        48,     // 48 bars max trade duration
		BBWidthExitThreshold:  1.5,    // 1.5x avg BB width for exit
		ADXPopupThreshold:     22,     // ADX > 22 indicates trending
	}
}

// NewGridEngine creates a new grid engine instance
func NewGridEngine(symbol string) (*GridEngine, error) {
	config := DefaultGridEngineConfig()
	
	// Initialize EMA 100 for anchoring
	emaIndicator := indicators.NewEMA(100)
	
	engine := &GridEngine{
		engineType:            EngineTypeGrid,
		name:                 "Grid Hedge Engine",
		symbol:               symbol,
		active:               false,
		anchorMethods:        []string{"anchored_vwap", "ema_100"},
		atrMultiplier:        config.ATRMultiplier,
		gridSpacingMultiplier: config.GridSpacingMultiplier,
		maxBands:             config.MaxBands,
		symmetricPlacement:   true,
		maxLongNotional:      config.MaxLongNotional,
		maxShortNotional:     config.MaxShortNotional,
		maxNetExposure:       config.MaxNetExposure,
		takeProfitMultiplier: config.TakeProfitMultiplier,
		stopLossMultiplier:   config.StopLossMultiplier,
		timeBasedExit:        true,
		maxBarsInTrade:       config.MaxBarsInTrade,
		bbWidthExitThreshold: config.BBWidthExitThreshold,
		adxPopupThreshold:    config.ADXPopupThreshold,
		regimeFlipExit:       true,
		indicatorManager:     indicators.NewIndicatorManager(),
		gridLevels:          make([]*GridLevel, 0),
		activePositions:     make([]*GridPosition, 0),
		performanceMetrics:  &GridEngineMetrics{},
		emaIndicator:        emaIndicator,
		vwapData: VWAPData{
			SessionStart: time.Now(),
		},
	}
	
	return engine, nil
}

// Implement TradingEngine interface

func (g *GridEngine) GetType() EngineType {
	return g.engineType
}

func (g *GridEngine) GetName() string {
	return g.name
}

func (g *GridEngine) GetPreferredRegimes() []regime.RegimeType {
	return []regime.RegimeType{regime.RegimeRanging, regime.RegimeVolatile}
}

func (g *GridEngine) IsCompatibleWithRegime(regimeType regime.RegimeType) bool {
	switch regimeType {
	case regime.RegimeRanging:
		return true  // Perfect for ranging markets
	case regime.RegimeVolatile:
		return true  // Good for volatile markets (can capture swings)
	case regime.RegimeUncertain:
		return true  // Can handle uncertainty with hedged approach
	case regime.RegimeTrending:
		return false // Not good for strong trends (use TrendEngine)
	default:
		return false
	}
}

func (g *GridEngine) GetRegimeCompatibilityScore(regimeType regime.RegimeType) float64 {
	switch regimeType {
	case regime.RegimeRanging:
		return 1.0  // Perfect fit for ranging markets
	case regime.RegimeVolatile:
		return 0.8  // Very good for volatile markets
	case regime.RegimeUncertain:
		return 0.6  // Decent for uncertain markets
	case regime.RegimeTrending:
		return 0.1  // Very poor for trending markets
	default:
		return 0.0
	}
}

func (g *GridEngine) IsActive() bool {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	return g.active
}

func (g *GridEngine) SetActive(active bool) {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	g.active = active
	g.status.IsActive = active
}

func (g *GridEngine) AnalyzeMarket(ctx context.Context, data30m, data5m []types.OHLCV, currentRegime regime.RegimeType) (EngineSignal, error) {
	_ = ctx // Context parameter reserved for future use
	
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	// Check if engine is active and compatible with current regime
	if !g.active || !g.IsCompatibleWithRegime(currentRegime) {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "Grid engine inactive or incompatible with current regime",
		}, nil
	}
	
	// Use 5m data as primary for grid analysis
	analysisData := data5m
	if len(analysisData) < 100 {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "Insufficient data for grid analysis",
		}, nil
	}
	
	// Update VWAP and EMA anchors
	g.updateAnchors(analysisData)
	
	// Check for grid rebalancing opportunities
	signal := g.analyzeGridOpportunity(analysisData, currentRegime)
	
	return signal, nil
}

func (g *GridEngine) AnalyzeMarketWithData(ctx context.Context, data map[string][]types.OHLCV, currentRegime regime.RegimeType) (EngineSignal, error) {
	// Grid primarily uses 5m data, 30m for context
	data5m, has5m := data["5m"]
	data30m, has30m := data["30m"]
	
	if !has5m {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "Missing required 5m timeframe data for Grid",
		}, nil
	}
	
	// Use 30m data if available, otherwise empty slice
	if !has30m {
		data30m = []types.OHLCV{}
	}
	
	return g.AnalyzeMarket(ctx, data30m, data5m, currentRegime)
}

func (g *GridEngine) ManagePositions(ctx context.Context, currentData []types.OHLCV) error {
	_ = ctx // Context parameter reserved for future use
	
	g.positionMutex.Lock()
	defer g.positionMutex.Unlock()
	
	if len(currentData) == 0 {
		return fmt.Errorf("no data provided for position management")
	}
	
	currentPrice := currentData[len(currentData)-1].Close
	
	// Update position PnL and check exit conditions
	for _, position := range g.activePositions {
		g.updatePositionPnL(position, currentPrice)
		
		// Check time-based exit
		if g.timeBasedExit && time.Since(position.EntryTime) > time.Duration(g.maxBarsInTrade)*5*time.Minute {
			// Mark for closure (would normally close via exchange)
			continue
		}
		
		// Check TP/SL levels
		if g.shouldCloseGridPosition(position, currentPrice) {
			// Mark for closure (would normally close via exchange)
			continue
		}
	}
	
	// Update metrics
	g.updateGridMetrics()
	g.status.LastActivity = time.Now()
	
	return nil
}

func (g *GridEngine) ShouldClosePosition(position EnginePosition, currentPrice float64, currentRegime regime.RegimeType) bool {
	// Close grid positions if regime becomes strongly trending
	if currentRegime == regime.RegimeTrending && g.regimeFlipExit {
		return true
	}
	
	// Check if we're at profit target levels
	if position.GetSide() == "long" {
		profitTarget := position.GetEntryPrice() * (1.0 + g.gridSpacingMultiplier*g.takeProfitMultiplier)
		return currentPrice >= profitTarget
	} else {
		profitTarget := position.GetEntryPrice() * (1.0 - g.gridSpacingMultiplier*g.takeProfitMultiplier)
		return currentPrice <= profitTarget
	}
}

func (g *GridEngine) CalculatePositionSize(balance float64, price float64, currentRegime regime.RegimeType) float64 {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	// Base position size calculation for grid levels
	maxExposure := balance * 0.8 // 80% max exposure for grid
	
	// Adjust based on regime compatibility
	regimeScore := g.GetRegimeCompatibilityScore(currentRegime)
	adjustedExposure := maxExposure * regimeScore
	
	// Divide by number of grid levels for balanced exposure
	levelExposure := adjustedExposure / float64(g.maxBands*2) // Both buy and sell sides
	
	return levelExposure / price
}

func (g *GridEngine) ValidateSignal(signal EngineSignal, currentPositions []EnginePosition) error {
	// Check signal confidence
	if signal.GetConfidence() < 0.4 { // Lower threshold for grid since it's market neutral
		return fmt.Errorf("signal confidence too low: %.2f", signal.GetConfidence())
	}
	
	// Check exposure limits
	totalExposure := g.calculateTotalExposure(currentPositions)
	if totalExposure > 0.8 { // 80% max total exposure
		return fmt.Errorf("total exposure exceeds limit: %.2f", totalExposure)
	}
	
	return nil
}

func (g *GridEngine) GetCurrentPositions() []EnginePosition {
	g.positionMutex.RLock()
	defer g.positionMutex.RUnlock()
	
	positions := make([]EnginePosition, len(g.activePositions))
	for i, pos := range g.activePositions {
		positions[i] = pos
	}
	
	return positions
}

func (g *GridEngine) GetPerformanceMetrics() EngineMetrics {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	return &BasicEngineMetrics{
		ActivePositions: len(g.activePositions),
		TotalTrades:     g.performanceMetrics.TotalTrades,
		WinningTrades:   g.performanceMetrics.WinningTrades,
		LosingTrades:    g.performanceMetrics.LosingTrades,
		TotalPnL:        g.performanceMetrics.TotalPnL,
		LastUpdated:     time.Now(),
		EngineSpecificMetrics: map[string]interface{}{
			"largest_win":      g.performanceMetrics.LargestWin,
			"largest_loss":     g.performanceMetrics.LargestLoss,
			"average_trade":    g.performanceMetrics.AverageTradePnL,
			"total_volume":     g.performanceMetrics.TotalVolume,
			"grid_rebalances":  g.performanceMetrics.GridRebalances,
			"net_exposure":     g.performanceMetrics.NetExposure,
		},
	}
}

func (g *GridEngine) GetEngineStatus() EngineStatus {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	status := g.status
	status.ActivePositions = len(g.activePositions)
	status.PendingOrders = len(g.gridLevels)
	status.IsActive = g.active
	status.LastActivity = time.Now()
	
	// Grid-specific status data
	status.EngineSpecificData = map[string]interface{}{
		"grid_levels":      len(g.gridLevels),
		"active_positions": len(g.activePositions),
		"net_exposure":     g.performanceMetrics.NetExposure,
		"long_exposure":    g.performanceMetrics.LongExposure,
		"short_exposure":   g.performanceMetrics.ShortExposure,
		"anchor_price":     g.anchorPrice,
		"current_vwap":     g.vwapData.CurrentVWAP,
	}
	
	return status
}

// Configuration and lifecycle management
func (g *GridEngine) UpdateConfiguration(config map[string]interface{}) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	if val, ok := config["max_bands"]; ok {
		if maxBands, ok := val.(int); ok {
			g.maxBands = maxBands
		}
	}
	
	if val, ok := config["grid_spacing_multiplier"]; ok {
		if spacing, ok := val.(float64); ok {
			g.gridSpacingMultiplier = spacing
		}
	}
	
	if val, ok := config["max_net_exposure"]; ok {
		if exposure, ok := val.(float64); ok {
			g.maxNetExposure = exposure
		}
	}
	
	return nil
}

func (g *GridEngine) GetConfig() map[string]interface{} {
	g.mutex.RLock()
	defer g.mutex.RUnlock()
	
	return map[string]interface{}{
		"max_bands":               g.maxBands,
		"grid_spacing_multiplier": g.gridSpacingMultiplier,
		"max_long_notional":       g.maxLongNotional,
		"max_short_notional":      g.maxShortNotional,
		"max_net_exposure":        g.maxNetExposure,
		"symmetric_placement":     g.symmetricPlacement,
		"anchor_methods":          g.anchorMethods,
	}
}

func (g *GridEngine) SetRiskLimits(limits EngineRiskLimits) error {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	g.riskLimits = limits
	return nil
}

func (g *GridEngine) Initialize() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	g.gridLevels = make([]*GridLevel, 0)
	g.activePositions = make([]*GridPosition, 0)
	g.performanceMetrics = &GridEngineMetrics{}
	g.status = EngineStatus{}
	g.active = false
	g.anchorPrice = 0.0
	g.vwapData = VWAPData{
		SessionStart: time.Now(),
	}
	
	return nil
}

func (g *GridEngine) Start() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	g.active = true
	g.status.IsActive = true
	g.status.IsTrading = true
	
	return nil
}

func (g *GridEngine) Stop() error {
	g.mutex.Lock()
	defer g.mutex.Unlock()
	
	g.active = false
	g.status.IsActive = false
	g.status.IsTrading = false
	
	return nil
}

func (g *GridEngine) Reset() error {
	return g.Initialize()
}

// Grid-specific helper methods

func (g *GridEngine) updateAnchors(data []types.OHLCV) {
	if len(data) == 0 {
		return
	}
	
	latest := data[len(data)-1]
	
	// Update VWAP
	g.updateVWAP(latest)
	
	// Update EMA 100
	emaValue, err := g.emaIndicator.Calculate(data)
	if err != nil {
		// If EMA calculation fails, use previous value
		emaValue = g.emaIndicator.GetLastValue()
	}
	
	// Calculate weighted anchor price
	vwapWeight := 0.6
	emaWeight := 0.4
	
	g.anchorPrice = g.vwapData.CurrentVWAP*vwapWeight + emaValue*emaWeight
}

func (g *GridEngine) updateVWAP(candle types.OHLCV) {
	typicalPrice := (candle.High + candle.Low + candle.Close) / 3.0
	
	g.vwapData.PriceVolumeSum += typicalPrice * candle.Volume
	g.vwapData.VolumeSum += candle.Volume
	
	if g.vwapData.VolumeSum > 0 {
		g.vwapData.CurrentVWAP = g.vwapData.PriceVolumeSum / g.vwapData.VolumeSum
	}
	
	g.vwapData.LastUpdate = time.Now()
}

func (g *GridEngine) analyzeGridOpportunity(data []types.OHLCV, currentRegime regime.RegimeType) EngineSignal {
	if g.anchorPrice == 0 || len(data) < 14 {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "Insufficient anchor data for grid analysis",
		}
	}
	
	currentPrice := data[len(data)-1].Close
	
	// Calculate ATR for grid spacing
	atr := g.calculateATR(data, 14)
	if atr == 0 {
		return &BasicEngineSignal{
			Timestamp:  time.Now(),
			Confidence: 0.0,
			Action:     "HOLD",
			Reason:     "Unable to calculate ATR for grid spacing",
		}
	}
	
	gridSpacing := atr * g.gridSpacingMultiplier
	
	// Check if we need to place new grid levels
	distanceFromAnchor := math.Abs(currentPrice - g.anchorPrice)
	
	// Determine grid action based on price position relative to anchor
	action := "HOLD"
	confidence := 0.3 // Base confidence for grid
	
	if distanceFromAnchor > gridSpacing {
		// We're far enough from anchor to justify grid placement
		action = "PLACE_GRID"
		confidence = 0.7
		
		// Higher confidence in ranging/volatile markets
		if currentRegime == regime.RegimeRanging {
			confidence = 0.9
		} else if currentRegime == regime.RegimeVolatile {
			confidence = 0.8
		}
	}
	
	// Check safety mechanisms
	if g.shouldPauseGridTrading(data, currentRegime) {
		action = "HOLD"
		confidence = 0.0
	}
	
	return &BasicEngineSignal{
		Timestamp:  time.Now(),
		Confidence: confidence,
		Strength:   confidence, // Grid strength equals confidence
		Direction:  0, // Market neutral
		Action:     action,
		Price:      g.anchorPrice,
		Size:       g.calculateGridSize(gridSpacing, currentPrice),
		StopLoss:   0.0, // Grid doesn't use traditional stop loss
		TakeProfit: []float64{}, // Grid uses dynamic TP levels
		Reason:     fmt.Sprintf("Grid analysis - Anchor: %.2f, Current: %.2f, Spacing: %.4f", g.anchorPrice, currentPrice, gridSpacing),
		Metadata: map[string]interface{}{
			"anchor_price":   g.anchorPrice,
			"grid_spacing":   gridSpacing,
			"vwap":          g.vwapData.CurrentVWAP,
			"ema_100":       g.emaIndicator.GetLastValue(),
			"net_exposure":  g.performanceMetrics.NetExposure,
		},
	}
}

func (g *GridEngine) calculateATR(data []types.OHLCV, period int) float64 {
	if len(data) < period+1 {
		return 0.0
	}
	
	trueRanges := make([]float64, len(data)-1)
	for i := 1; i < len(data); i++ {
		high := data[i].High
		low := data[i].Low
		prevClose := data[i-1].Close
		
		tr1 := high - low
		tr2 := math.Abs(high - prevClose)
		tr3 := math.Abs(low - prevClose)
		
		trueRanges[i-1] = math.Max(tr1, math.Max(tr2, tr3))
	}
	
	// Calculate ATR as simple moving average of true ranges
	if len(trueRanges) < period {
		return 0.0
	}
	
	sum := 0.0
	for i := len(trueRanges) - period; i < len(trueRanges); i++ {
		sum += trueRanges[i]
	}
	
	return sum / float64(period)
}

func (g *GridEngine) shouldPauseGridTrading(data []types.OHLCV, currentRegime regime.RegimeType) bool {
	// Pause if regime is trending (ADX safety mechanism)
	if currentRegime == regime.RegimeTrending && g.regimeFlipExit {
		return true
	}
	
	// Check Bollinger Band width (volatility)
	if len(data) >= 20 {
		bb := g.calculateBBWidth(data, 20)
		if bb > g.bbWidthExitThreshold {
			return true // Too volatile for grid
		}
	}
	
	// Check net exposure limits
	if math.Abs(g.performanceMetrics.NetExposure) > g.maxNetExposure {
		return true
	}
	
	return false
}

func (g *GridEngine) calculateBBWidth(data []types.OHLCV, period int) float64 {
	if len(data) < period {
		return 0.0
	}
	
	// Simple BB width calculation
	sum := 0.0
	for i := len(data) - period; i < len(data); i++ {
		sum += data[i].Close
	}
	
	mean := sum / float64(period)
	
	variance := 0.0
	for i := len(data) - period; i < len(data); i++ {
		diff := data[i].Close - mean
		variance += diff * diff
	}
	
	stdDev := math.Sqrt(variance / float64(period))
	upperBand := mean + 2*stdDev
	lowerBand := mean - 2*stdDev
	
	return (upperBand - lowerBand) / mean // Normalized width
}

func (g *GridEngine) calculateGridSize(gridSpacing float64, currentPrice float64) float64 {
	// Base grid size calculation
	baseSize := 100.0 // Base USD amount per grid level
	
	// Adjust based on grid spacing and current exposure
	exposureAdjustment := 1.0 - math.Abs(g.performanceMetrics.NetExposure)/g.maxNetExposure
	
	return (baseSize * exposureAdjustment) / currentPrice
}

func (g *GridEngine) updatePositionPnL(position *GridPosition, currentPrice float64) {
	position.CurrentPrice = currentPrice
	
	if position.Side == "long" {
		position.UnrealizedPnL = (currentPrice - position.EntryPrice) * position.Quantity
	} else {
		position.UnrealizedPnL = (position.EntryPrice - currentPrice) * position.Quantity
	}
}

func (g *GridEngine) shouldCloseGridPosition(position *GridPosition, currentPrice float64) bool {
	// Check profit target
	profitTarget := position.ExitTarget
	
	if position.Side == "long" && currentPrice >= profitTarget {
		return true
	}
	
	if position.Side == "short" && currentPrice <= profitTarget {
		return true
	}
	
	// Check stop loss
	stopLoss := position.StopLoss
	
	if position.Side == "long" && currentPrice <= stopLoss {
		return true
	}
	
	if position.Side == "short" && currentPrice >= stopLoss {
		return true
	}
	
	return false
}

func (g *GridEngine) calculateTotalExposure(positions []EnginePosition) float64 {
	totalExposure := 0.0
	
	for _, pos := range positions {
		totalExposure += math.Abs(pos.GetEntryPrice() * pos.GetSize())
	}
	
	return totalExposure
}

func (g *GridEngine) updateGridMetrics() {
	g.performanceMetrics.ActiveLegs = len(g.activePositions)
	
	// Calculate net exposure
	netExposure := 0.0
	longExposure := 0.0
	shortExposure := 0.0
	
	for _, pos := range g.activePositions {
		exposure := pos.EntryPrice * pos.Quantity
		
		if pos.Side == "long" {
			longExposure += exposure
			netExposure += exposure
		} else {
			shortExposure += exposure
			netExposure -= exposure
		}
	}
	
	g.performanceMetrics.NetExposure = netExposure
	g.performanceMetrics.LongExposure = longExposure
	g.performanceMetrics.ShortExposure = shortExposure
}

// Implement EnginePosition interface for GridPosition
func (pos *GridPosition) GetID() string                        { return pos.ID }
func (pos *GridPosition) GetEngineType() string               { return "grid" }
func (pos *GridPosition) GetSide() string                     { return pos.Side }
func (pos *GridPosition) GetSize() float64                    { return pos.Quantity }
func (pos *GridPosition) GetEntryPrice() float64              { return pos.EntryPrice }
func (pos *GridPosition) GetCurrentPrice() float64            { return pos.CurrentPrice }
func (pos *GridPosition) GetUnrealizedPnL() float64           { return pos.UnrealizedPnL }
func (pos *GridPosition) GetEntryTime() time.Time             { return pos.EntryTime }
func (pos *GridPosition) UpdatePrice(newPrice float64)        { pos.CurrentPrice = newPrice }
func (pos *GridPosition) GetMetadata() map[string]interface{} { 
	return map[string]interface{}{
		"grid_level_id": pos.GridLevelID,
		"exit_target":   pos.ExitTarget,
		"stop_loss":     pos.StopLoss,
	}
}

// Implement EngineMetrics interface for GridEngineMetrics
func (m *GridEngineMetrics) GetTotalTrades() int       { return m.TotalTrades }
func (m *GridEngineMetrics) GetWinningTrades() int     { return m.WinningTrades }
func (m *GridEngineMetrics) GetLosingTrades() int      { return m.LosingTrades }
func (m *GridEngineMetrics) GetTotalPnL() float64      { return m.TotalPnL }
func (m *GridEngineMetrics) GetLargestWin() float64    { return m.LargestWin }
func (m *GridEngineMetrics) GetLargestLoss() float64   { return m.LargestLoss }
func (m *GridEngineMetrics) GetAverageTrade() float64  { return m.AverageTradePnL }
func (m *GridEngineMetrics) GetTotalVolume() float64   { return m.TotalVolume }
func (m *GridEngineMetrics) GetActivePositions() int   { return m.ActiveLegs }
