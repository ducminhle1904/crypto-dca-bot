package backtest

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// Type aliases to avoid circular imports
type EngineType string

const (
	EngineTypeGrid  EngineType = "grid"
	EngineTypeTrend EngineType = "trend"
)

// DualEngineBacktester performs comprehensive backtesting of the dual-engine regime system
type DualEngineBacktester struct {
	// Configuration
	config     *config.DualEngineConfig
	symbol     string
	timeframe  string
	
	// Core components
	regimeDetector *regime.RegimeDetector
	logger         *logger.Logger
	
	// Backtesting state
	data           []types.OHLCV
	currentIndex   int
	startTime      time.Time
	endTime        time.Time
	
	// Trading state
	balance            float64
	totalInvested      float64
	activeEngine       EngineType
	enginePositions    map[EngineType][]*BacktestPosition
	
	// Performance tracking
	results            *DualEngineBacktestResults
	regimeHistory      []*RegimeRecord
	transitionHistory  []*TransitionRecord
	engineMetrics      map[EngineType]*EngineBacktestMetrics
	
	// Risk management
	maxDrawdown        float64
	peakBalance        float64
	dailyReturns       []float64
	
	// Transition tracking
	transitionCosts    float64
	totalTransitions   int
	lastTransitionTime time.Time
}

// BacktestPosition represents a position during backtesting
type BacktestPosition struct {
	ID            string
	Symbol        string
	Side          string    // "long" or "short"
	Size          float64
	EntryPrice    float64
	EntryTime     time.Time
	ExitPrice     float64
	ExitTime      time.Time
	PnL           float64
	Duration      time.Duration
	EngineType    EngineType
	RegimeAtEntry regime.RegimeType
	RegimeAtExit  regime.RegimeType
	IsOpen        bool
}

// DualEngineBacktestResults contains comprehensive backtesting results
type DualEngineBacktestResults struct {
	// Meta information
	Symbol        string    `json:"symbol"`
	Timeframe     string    `json:"timeframe"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Duration      time.Duration `json:"duration"`
	TotalBars     int       `json:"total_bars"`
	
	// Financial metrics
	InitialBalance    float64 `json:"initial_balance"`
	FinalBalance      float64 `json:"final_balance"`
	TotalReturn       float64 `json:"total_return"`       // %
	AnnualizedReturn  float64 `json:"annualized_return"`  // %
	MaxDrawdown       float64 `json:"max_drawdown"`       // %
	
	// Trading metrics
	TotalTrades       int     `json:"total_trades"`
	WinningTrades     int     `json:"winning_trades"`
	LosingTrades      int     `json:"losing_trades"`
	WinRate           float64 `json:"win_rate"`           // %
	ProfitFactor      float64 `json:"profit_factor"`
	AvgTradeDuration  time.Duration `json:"avg_trade_duration"`
	
	// Risk metrics
	SharpeRatio       float64 `json:"sharpe_ratio"`
	SortinoRatio      float64 `json:"sortino_ratio"`
	CalmarRatio       float64 `json:"calmar_ratio"`
	VaR95             float64 `json:"var_95"`             // Value at Risk 95%
	
	// Regime-specific metrics
	RegimeAccuracy    float64 `json:"regime_accuracy"`    // %
	RegimeChanges     int     `json:"regime_changes"`
	AvgRegimeDuration time.Duration `json:"avg_regime_duration"`
	RegimeDistribution map[regime.RegimeType]float64 `json:"regime_distribution"`
	
	// Engine-specific metrics
	EngineUtilization map[EngineType]float64 `json:"engine_utilization"`  // % time active
	EnginePerformance map[EngineType]*EngineBacktestMetrics `json:"engine_performance"`
	
	// Transition metrics
	TotalTransitions  int     `json:"total_transitions"`
	TransitionCosts   float64 `json:"transition_costs"`   // Total cost in %
	AvgTransitionCost float64 `json:"avg_transition_cost"` // Per transition
	TransitionSuccessRate float64 `json:"transition_success_rate"` // %
	
	// Market condition performance
	TrendingPerformance float64 `json:"trending_performance"`  // Return in trending markets
	RangingPerformance  float64 `json:"ranging_performance"`   // Return in ranging markets
	VolatilePerformance float64 `json:"volatile_performance"`  // Return in volatile markets
}

// EngineBacktestMetrics contains engine-specific backtesting metrics
type EngineBacktestMetrics struct {
	EngineType        EngineType `json:"engine_type"`
	TotalTrades       int                `json:"total_trades"`
	WinningTrades     int                `json:"winning_trades"`
	LosingTrades      int                `json:"losing_trades"`
	WinRate           float64            `json:"win_rate"`
	TotalPnL          float64            `json:"total_pnl"`
	ProfitFactor      float64            `json:"profit_factor"`
	AvgWin            float64            `json:"avg_win"`
	AvgLoss           float64            `json:"avg_loss"`
	MaxWin            float64            `json:"max_win"`
	MaxLoss           float64            `json:"max_loss"`
	AvgTradeDuration  time.Duration      `json:"avg_trade_duration"`
	
	// Engine-specific metrics
	TimeActive        time.Duration      `json:"time_active"`
	UtilizationRate   float64            `json:"utilization_rate"`
	
	// Regime compatibility
	RegimePerformance map[regime.RegimeType]float64 `json:"regime_performance"`
}

// RegimeRecord represents a regime change during backtesting
type RegimeRecord struct {
	Timestamp       time.Time         `json:"timestamp"`
	OldRegime       regime.RegimeType `json:"old_regime"`
	NewRegime       regime.RegimeType `json:"new_regime"`
	Confidence      float64           `json:"confidence"`
	Duration        time.Duration     `json:"duration"`
	Price           float64           `json:"price"`
	ActualCorrect   bool              `json:"actual_correct"`   // Manual validation
}

// TransitionRecord represents an engine transition during backtesting
type TransitionRecord struct {
	Timestamp       time.Time             `json:"timestamp"`
	FromEngine      EngineType            `json:"from_engine"`
	ToEngine        EngineType            `json:"to_engine"`
	Reason          string                `json:"reason"`
	Cost            float64               `json:"cost"`
	PositionsBefore int                   `json:"positions_before"`
	PositionsAfter  int                   `json:"positions_after"`
	Successful      bool                  `json:"successful"`
	Price           float64               `json:"price"`
}

// NewDualEngineBacktester creates a new dual-engine backtester
func NewDualEngineBacktester(config *config.DualEngineConfig, symbol, timeframe string) (*DualEngineBacktester, error) {
	// Create logger
	logger, err := logger.NewLogger(fmt.Sprintf("backtest_%s_%s", symbol, timeframe), "backtest")
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}
	
	// Initialize regime detector
	regimeDetector := regime.NewRegimeDetector()
	
	backtester := &DualEngineBacktester{
		config:         config,
		symbol:         symbol,
		timeframe:      timeframe,
		regimeDetector: regimeDetector,
		logger:         logger,
		
		// Initialize state
		balance:         config.MainConfig.BaseAmount,
		activeEngine:    EngineTypeGrid, // Start with grid engine
		enginePositions: make(map[EngineType][]*BacktestPosition),
		engineMetrics:   make(map[EngineType]*EngineBacktestMetrics),
		
		// Initialize tracking
		regimeHistory:      make([]*RegimeRecord, 0),
		transitionHistory:  make([]*TransitionRecord, 0),
		dailyReturns:       make([]float64, 0),
		peakBalance:        config.MainConfig.BaseAmount,
		
		results: &DualEngineBacktestResults{
			Symbol:             symbol,
			Timeframe:          timeframe,
			InitialBalance:     config.MainConfig.BaseAmount,
			RegimeDistribution: make(map[regime.RegimeType]float64),
			EngineUtilization:  make(map[EngineType]float64),
			EnginePerformance:  make(map[EngineType]*EngineBacktestMetrics),
		},
	}
	
	// Initialize engine metrics
	backtester.engineMetrics[EngineTypeGrid] = &EngineBacktestMetrics{
		EngineType:        EngineTypeGrid,
		RegimePerformance: make(map[regime.RegimeType]float64),
	}
	backtester.engineMetrics[EngineTypeTrend] = &EngineBacktestMetrics{
		EngineType:        EngineTypeTrend,
		RegimePerformance: make(map[regime.RegimeType]float64),
	}
	
	return backtester, nil
}

// LoadData loads historical market data for backtesting
func (bt *DualEngineBacktester) LoadData(data []types.OHLCV) error {
	if len(data) == 0 {
		return fmt.Errorf("no data provided")
	}
	
	// Sort data by timestamp
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})
	
	bt.data = data
	bt.startTime = data[0].Timestamp
	bt.endTime = data[len(data)-1].Timestamp
	bt.results.StartTime = bt.startTime
	bt.results.EndTime = bt.endTime
	bt.results.Duration = bt.endTime.Sub(bt.startTime)
	bt.results.TotalBars = len(data)
	
	bt.logger.Info("Loaded %d bars from %v to %v", len(data), bt.startTime, bt.endTime)
	return nil
}

// Run executes the complete backtesting process
func (bt *DualEngineBacktester) Run() (*DualEngineBacktestResults, error) {
	bt.logger.Info("Starting dual-engine backtesting for %s %s", bt.symbol, bt.timeframe)
	
	if len(bt.data) == 0 {
		return nil, fmt.Errorf("no data loaded")
	}
	
	// Process each bar
	for bt.currentIndex = 0; bt.currentIndex < len(bt.data); bt.currentIndex++ {
		currentBar := bt.data[bt.currentIndex]
		
		// Update regime detection
		if err := bt.updateRegimeDetection(currentBar); err != nil {
			bt.logger.LogError("Regime Detection Error", err)
			continue
		}
		
		// Process engine logic
		if err := bt.processEngineLogic(currentBar); err != nil {
			bt.logger.LogError("Engine Logic Error", err)
			continue
		}
		
		// Update performance metrics
		bt.updatePerformanceMetrics(currentBar)
		
		// Process end-of-day calculations
		if bt.isEndOfDay(currentBar) {
			bt.processEndOfDay(currentBar)
		}
		
		// Progress reporting
		if bt.currentIndex%1000 == 0 {
			progress := float64(bt.currentIndex) / float64(len(bt.data)) * 100
			bt.logger.Info("Backtesting progress: %.1f%% (%d/%d)", progress, bt.currentIndex, len(bt.data))
		}
	}
	
	// Finalize results
	bt.finalizeResults()
	
	bt.logger.Info("Backtesting completed. Final balance: $%.2f (%.2f%% return)", 
		bt.results.FinalBalance, bt.results.TotalReturn)
	
	return bt.results, nil
}

// updateRegimeDetection processes regime detection for the current bar
func (bt *DualEngineBacktester) updateRegimeDetection(bar types.OHLCV) error {
	// Get historical data window for regime detection
	windowSize := 200 // Minimum window for meaningful analysis
	if bt.currentIndex < windowSize {
		return nil // Not enough data yet
	}
	
	// Create data window
	startIdx := max(0, bt.currentIndex-windowSize)
	dataWindow := bt.data[startIdx:bt.currentIndex+1]
	
	// Detect regime
	regimeSignal, err := bt.regimeDetector.DetectRegime(dataWindow)
	if err != nil || regimeSignal == nil {
		return nil
	}
	
	// Check for regime change
	currentRegime := regimeSignal.Type
	var oldRegime regime.RegimeType
	
	if len(bt.regimeHistory) > 0 {
		oldRegime = bt.regimeHistory[len(bt.regimeHistory)-1].NewRegime
	}
	
	// Record regime change
	if currentRegime != oldRegime {
		record := &RegimeRecord{
			Timestamp:  bar.Timestamp,
			OldRegime:  oldRegime,
			NewRegime:  currentRegime,
			Confidence: regimeSignal.Confidence,
			Price:      bar.Close,
		}
		
		// Calculate duration if we have previous regime
		if len(bt.regimeHistory) > 0 {
			record.Duration = bar.Timestamp.Sub(bt.regimeHistory[len(bt.regimeHistory)-1].Timestamp)
		}
		
		bt.regimeHistory = append(bt.regimeHistory, record)
		
		// Evaluate engine switching
		bt.evaluateEngineSwitch(currentRegime, bar)
	}
	
	return nil
}

// evaluateEngineSwitch determines if engine should be switched based on regime
func (bt *DualEngineBacktester) evaluateEngineSwitch(newRegime regime.RegimeType, bar types.OHLCV) {
	var preferredEngine EngineType
	
	// Determine preferred engine based on regime
	switch newRegime {
	case regime.RegimeTrending:
		preferredEngine = EngineTypeTrend
	case regime.RegimeRanging, regime.RegimeVolatile, regime.RegimeUncertain:
		preferredEngine = EngineTypeGrid
	default:
		return // No change
	}
	
	// Check if switch is needed
	if preferredEngine == bt.activeEngine {
		return // Already using correct engine
	}
	
	// Check transition cooldown
	cooldownPeriod := 30 * time.Minute
	if time.Since(bt.lastTransitionTime) < cooldownPeriod {
		return // Too soon to switch again
	}
	
	// Execute engine switch
	bt.executeEngineSwitch(preferredEngine, fmt.Sprintf("Regime change to %s", newRegime), bar)
}

// executeEngineSwitch performs the actual engine transition
func (bt *DualEngineBacktester) executeEngineSwitch(toEngine EngineType, reason string, bar types.OHLCV) {
	fromEngine := bt.activeEngine
	
	// Count positions before transition
	positionsBefore := len(bt.enginePositions[fromEngine])
	
	// Calculate transition cost (simplified)
	transitionCost := bt.calculateTransitionCost(fromEngine, toEngine)
	bt.transitionCosts += transitionCost
	bt.balance -= transitionCost
	
	// Close positions from old engine (simplified approach)
	bt.closeEnginePositions(fromEngine, bar, "Engine transition")
	
	// Switch active engine
	bt.activeEngine = toEngine
	bt.lastTransitionTime = bar.Timestamp
	bt.totalTransitions++
	
	// Count positions after transition
	positionsAfter := len(bt.enginePositions[toEngine])
	
	// Record transition
	transition := &TransitionRecord{
		Timestamp:       bar.Timestamp,
		FromEngine:      fromEngine,
		ToEngine:        toEngine,
		Reason:          reason,
		Cost:            transitionCost,
		PositionsBefore: positionsBefore,
		PositionsAfter:  positionsAfter,
		Successful:      true,
		Price:           bar.Close,
	}
	
	bt.transitionHistory = append(bt.transitionHistory, transition)
	
	bt.logger.Info("Engine transition: %s → %s (Cost: $%.2f, Reason: %s)", 
		fromEngine, toEngine, transitionCost, reason)
}

// processEngineLogic processes trading logic for the active engine
func (bt *DualEngineBacktester) processEngineLogic(bar types.OHLCV) error {
	switch bt.activeEngine {
	case EngineTypeGrid:
		return bt.processGridEngine(bar)
	case EngineTypeTrend:
		return bt.processTrendEngine(bar)
	default:
		return fmt.Errorf("unknown engine type: %s", bt.activeEngine)
	}
}

// processGridEngine simulates grid engine trading logic
func (bt *DualEngineBacktester) processGridEngine(bar types.OHLCV) error {
	// Simplified grid engine logic for backtesting
	// In reality, this would use the actual grid engine
	
	// Check for grid trading opportunities
	if bt.shouldCreateGridPosition(bar) {
		bt.createGridPosition(bar)
	}
	
	// Check existing positions for exits
	bt.checkGridExits(bar)
	
	return nil
}

// processTrendEngine simulates trend engine trading logic
func (bt *DualEngineBacktester) processTrendEngine(bar types.OHLCV) error {
	// Simplified trend engine logic for backtesting
	// In reality, this would use the actual trend engine
	
	// Check for trend signals
	if bt.shouldCreateTrendPosition(bar) {
		bt.createTrendPosition(bar)
	}
	
	// Check existing positions for exits
	bt.checkTrendExits(bar)
	
	return nil
}

// Helper methods for position management

func (bt *DualEngineBacktester) shouldCreateGridPosition(bar types.OHLCV) bool {
	// Simplified logic - in practice would use grid levels, VWAP, etc.
	return len(bt.enginePositions[EngineTypeGrid]) < 3 // Max 3 grid positions
}

func (bt *DualEngineBacktester) shouldCreateTrendPosition(bar types.OHLCV) bool {
	// Simplified logic - in practice would use trend indicators
	return len(bt.enginePositions[EngineTypeTrend]) == 0 // Max 1 trend position
}

func (bt *DualEngineBacktester) createGridPosition(bar types.OHLCV) {
	// Calculate position size (simplified)
	positionSize := bt.balance * 0.1 // 10% of balance
	
	position := &BacktestPosition{
		ID:            fmt.Sprintf("grid_%d_%d", bt.currentIndex, time.Now().UnixNano()),
		Symbol:        bt.symbol,
		Side:          "long", // Simplified - alternating long/short in real grid
		Size:          positionSize / bar.Close,
		EntryPrice:    bar.Close,
		EntryTime:     bar.Timestamp,
		EngineType:    EngineTypeGrid,
		RegimeAtEntry: bt.getCurrentRegime(),
		IsOpen:        true,
	}
	
	bt.enginePositions[EngineTypeGrid] = append(bt.enginePositions[EngineTypeGrid], position)
	bt.totalInvested += positionSize
	
	bt.logger.Info("Created grid position: %.4f @ $%.2f", position.Size, position.EntryPrice)
}

func (bt *DualEngineBacktester) createTrendPosition(bar types.OHLCV) {
	// Calculate position size (simplified)
	positionSize := bt.balance * 0.2 // 20% of balance
	
	position := &BacktestPosition{
		ID:            fmt.Sprintf("trend_%d_%d", bt.currentIndex, time.Now().UnixNano()),
		Symbol:        bt.symbol,
		Side:          "long", // Simplified - would determine based on trend direction
		Size:          positionSize / bar.Close,
		EntryPrice:    bar.Close,
		EntryTime:     bar.Timestamp,
		EngineType:    EngineTypeTrend,
		RegimeAtEntry: bt.getCurrentRegime(),
		IsOpen:        true,
	}
	
	bt.enginePositions[EngineTypeTrend] = append(bt.enginePositions[EngineTypeTrend], position)
	bt.totalInvested += positionSize
	
	bt.logger.Info("Created trend position: %.4f @ $%.2f", position.Size, position.EntryPrice)
}

func (bt *DualEngineBacktester) checkGridExits(bar types.OHLCV) {
	for _, position := range bt.enginePositions[EngineTypeGrid] {
		if !position.IsOpen {
			continue
		}
		
		// Simple exit logic - 2% profit or 1% loss
		pnlPercent := (bar.Close - position.EntryPrice) / position.EntryPrice
		
		if pnlPercent >= 0.02 || pnlPercent <= -0.01 {
			bt.closePosition(position, bar, "Grid exit condition")
		}
	}
}

func (bt *DualEngineBacktester) checkTrendExits(bar types.OHLCV) {
	for _, position := range bt.enginePositions[EngineTypeTrend] {
		if !position.IsOpen {
			continue
		}
		
		// Simple exit logic - 5% profit or 3% loss
		pnlPercent := (bar.Close - position.EntryPrice) / position.EntryPrice
		
		if pnlPercent >= 0.05 || pnlPercent <= -0.03 {
			bt.closePosition(position, bar, "Trend exit condition")
		}
	}
}

func (bt *DualEngineBacktester) closePosition(position *BacktestPosition, bar types.OHLCV, reason string) {
	position.ExitPrice = bar.Close
	position.ExitTime = bar.Timestamp
	position.RegimeAtExit = bt.getCurrentRegime()
	position.Duration = bar.Timestamp.Sub(position.EntryTime)
	position.IsOpen = false
	
	// Calculate P&L
	if position.Side == "long" {
		position.PnL = (position.ExitPrice - position.EntryPrice) * position.Size
	} else {
		position.PnL = (position.EntryPrice - position.ExitPrice) * position.Size
	}
	
	bt.balance += position.PnL
	bt.totalInvested -= position.EntryPrice * position.Size
	
	// Update engine metrics
	metrics := bt.engineMetrics[position.EngineType]
	metrics.TotalTrades++
	metrics.TotalPnL += position.PnL
	
	if position.PnL > 0 {
		metrics.WinningTrades++
	} else {
		metrics.LosingTrades++
	}
	
	bt.logger.Info("Closed %s position: P&L $%.2f (%.2f%%) - %s", 
		position.EngineType, position.PnL, (position.PnL/(position.EntryPrice*position.Size))*100, reason)
}

func (bt *DualEngineBacktester) closeEnginePositions(engineType EngineType, bar types.OHLCV, reason string) {
	for _, position := range bt.enginePositions[engineType] {
		if position.IsOpen {
			bt.closePosition(position, bar, reason)
		}
	}
}

// Utility methods

func (bt *DualEngineBacktester) getCurrentRegime() regime.RegimeType {
	if len(bt.regimeHistory) == 0 {
		return regime.RegimeUncertain
	}
	return bt.regimeHistory[len(bt.regimeHistory)-1].NewRegime
}

func (bt *DualEngineBacktester) calculateTransitionCost(from, to EngineType) float64 {
	// Simplified transition cost calculation
	// In practice, this would consider slippage, fees, spread impact
	return bt.balance * 0.001 // 0.1% of balance
}

func (bt *DualEngineBacktester) updatePerformanceMetrics(bar types.OHLCV) {
	// Update peak balance and drawdown
	if bt.balance > bt.peakBalance {
		bt.peakBalance = bt.balance
	}
	
	currentDrawdown := (bt.peakBalance - bt.balance) / bt.peakBalance
	if currentDrawdown > bt.maxDrawdown {
		bt.maxDrawdown = currentDrawdown
	}
}

func (bt *DualEngineBacktester) isEndOfDay(bar types.OHLCV) bool {
	// Simplified - check if it's a new day
	if bt.currentIndex == 0 {
		return false
	}
	
	prevBar := bt.data[bt.currentIndex-1]
	return bar.Timestamp.Day() != prevBar.Timestamp.Day()
}

func (bt *DualEngineBacktester) processEndOfDay(bar types.OHLCV) {
	// Calculate daily return
	if bt.currentIndex > 0 {
		prevBalance := bt.results.InitialBalance // Simplified
		if len(bt.dailyReturns) > 0 {
			// Would need to track daily balances properly
		}
		
		dailyReturn := (bt.balance - prevBalance) / prevBalance
		bt.dailyReturns = append(bt.dailyReturns, dailyReturn)
	}
}

func (bt *DualEngineBacktester) finalizeResults() {
	// Basic financial metrics
	bt.results.FinalBalance = bt.balance
	bt.results.TotalReturn = (bt.balance - bt.results.InitialBalance) / bt.results.InitialBalance * 100
	bt.results.MaxDrawdown = bt.maxDrawdown * 100
	
	// Annualized return
	years := bt.results.Duration.Hours() / (24 * 365.25)
	if years > 0 {
		bt.results.AnnualizedReturn = (math.Pow(bt.balance/bt.results.InitialBalance, 1/years) - 1) * 100
	}
	
	// Trading metrics
	bt.results.TotalTrades = bt.getTotalTrades()
	bt.results.WinningTrades = bt.getWinningTrades()
	bt.results.LosingTrades = bt.results.TotalTrades - bt.results.WinningTrades
	
	if bt.results.TotalTrades > 0 {
		bt.results.WinRate = float64(bt.results.WinningTrades) / float64(bt.results.TotalTrades) * 100
	}
	
	// Risk metrics
	bt.calculateRiskMetrics()
	
	// Regime metrics
	bt.calculateRegimeMetrics()
	
	// Engine metrics
	bt.finalizeEngineMetrics()
	
	// Transition metrics
	bt.calculateTransitionMetrics()
}

func (bt *DualEngineBacktester) getTotalTrades() int {
	total := 0
	for _, positions := range bt.enginePositions {
		for _, position := range positions {
			if !position.IsOpen {
				total++
			}
		}
	}
	return total
}

func (bt *DualEngineBacktester) getWinningTrades() int {
	winning := 0
	for _, positions := range bt.enginePositions {
		for _, position := range positions {
			if !position.IsOpen && position.PnL > 0 {
				winning++
			}
		}
	}
	return winning
}

func (bt *DualEngineBacktester) calculateRiskMetrics() {
	if len(bt.dailyReturns) == 0 {
		return
	}
	
	// Calculate Sharpe ratio (simplified)
	avgReturn := bt.average(bt.dailyReturns)
	stdDev := bt.standardDeviation(bt.dailyReturns, avgReturn)
	
	if stdDev > 0 {
		bt.results.SharpeRatio = (avgReturn - 0.02/365) / stdDev * math.Sqrt(365) // Assuming 2% risk-free rate
	}
	
	// Calculate VaR 95%
	sort.Float64s(bt.dailyReturns)
	var95Index := int(float64(len(bt.dailyReturns)) * 0.05)
	if var95Index < len(bt.dailyReturns) {
		bt.results.VaR95 = bt.dailyReturns[var95Index] * 100
	}
}

func (bt *DualEngineBacktester) calculateRegimeMetrics() {
	bt.results.RegimeChanges = len(bt.regimeHistory)
	
	if len(bt.regimeHistory) > 0 {
		// Calculate average regime duration
		totalDuration := time.Duration(0)
		for _, record := range bt.regimeHistory {
			totalDuration += record.Duration
		}
		bt.results.AvgRegimeDuration = totalDuration / time.Duration(len(bt.regimeHistory))
		
		// Calculate regime distribution
		regimeCounts := make(map[regime.RegimeType]int)
		for _, record := range bt.regimeHistory {
			regimeCounts[record.NewRegime]++
		}
		
		total := len(bt.regimeHistory)
		for regimeType, count := range regimeCounts {
			bt.results.RegimeDistribution[regimeType] = float64(count) / float64(total) * 100
		}
	}
}

func (bt *DualEngineBacktester) finalizeEngineMetrics() {
	for engineType, metrics := range bt.engineMetrics {
		if metrics.TotalTrades > 0 {
			metrics.WinRate = float64(metrics.WinningTrades) / float64(metrics.TotalTrades) * 100
		}
		
		bt.results.EnginePerformance[engineType] = metrics
	}
}

func (bt *DualEngineBacktester) calculateTransitionMetrics() {
	bt.results.TotalTransitions = bt.totalTransitions
	bt.results.TransitionCosts = bt.transitionCosts / bt.results.InitialBalance * 100
	
	if bt.totalTransitions > 0 {
		bt.results.AvgTransitionCost = bt.results.TransitionCosts / float64(bt.totalTransitions)
		
		// Calculate transition success rate (simplified)
		successfulTransitions := 0
		for _, transition := range bt.transitionHistory {
			if transition.Successful {
				successfulTransitions++
			}
		}
		bt.results.TransitionSuccessRate = float64(successfulTransitions) / float64(bt.totalTransitions) * 100
	}
}

// Utility functions

func (bt *DualEngineBacktester) average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (bt *DualEngineBacktester) standardDeviation(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sumSquaredDiffs := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquaredDiffs += diff * diff
	}
	
	variance := sumSquaredDiffs / float64(len(values))
	return math.Sqrt(variance)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetResults returns the current backtest results
func (bt *DualEngineBacktester) GetResults() *DualEngineBacktestResults {
	return bt.results
}

// GetRegimeHistory returns the regime change history
func (bt *DualEngineBacktester) GetRegimeHistory() []*RegimeRecord {
	return bt.regimeHistory
}

// GetTransitionHistory returns the engine transition history
func (bt *DualEngineBacktester) GetTransitionHistory() []*TransitionRecord {
	return bt.transitionHistory
}

// GetPositions returns all positions for analysis
func (bt *DualEngineBacktester) GetPositions() map[EngineType][]*BacktestPosition {
	return bt.enginePositions
}
