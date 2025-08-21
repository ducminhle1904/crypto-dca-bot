package strategy

import (
	"fmt"
	"math"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// PositionSide represents the direction of a position
type PositionSide int

const (
	PositionLong PositionSide = iota
	PositionShort
)

func (ps PositionSide) String() string {
	switch ps {
	case PositionLong:
		return "LONG"
	case PositionShort:
		return "SHORT"
	default:
		return "UNKNOWN"
	}
}

// Position represents an individual trading position
type Position struct {
	Side            PositionSide
	EntryPrice      float64
	Quantity        float64
	UnrealizedPnL   float64
	RealizedPnL     float64
	StopLoss        float64
	TakeProfit      float64
	IsActive        bool
	EntryTime       time.Time
	ExitTime        time.Time
	MaxProfit       float64  // Track maximum profit reached
	MaxLoss         float64  // Track maximum loss reached
	Commission      float64  // Total commission paid
	TrailingStopPct float64  // Trailing stop percentage
}

// DualPositionTradeDecision represents a decision that can involve both positions
type DualPositionTradeDecision struct {
	LongAction   TradeAction
	ShortAction  TradeAction
	LongAmount   float64
	ShortAmount  float64
	Confidence   float64
	Strength     float64
	Reason       string
	Timestamp    time.Time
}

// DualPositionStrategy implements a strategy that opens both long and short positions
type DualPositionStrategy struct {
	// Core configuration
	baseAmount           float64
	hedgeRatio           float64  // 0.0 = only long, 0.5 = equal positions, 1.0 = only short
	maxMultiplier        float64
	minConfidence        float64
	
	// Advanced risk management
	stopLossPct         float64  // Stop loss percentage
	takeProfitPct       float64  // Take profit percentage
	trailingStopPct     float64  // Trailing stop percentage
	maxDrawdownPct      float64  // Maximum allowed drawdown per position
	positionSizeLimit   float64  // Maximum position size as % of total balance
	
	// Volatility and timing controls
	volatilityThreshold float64  // Minimum volatility to open positions
	timeBetweenEntries  time.Duration // Minimum time between new entries
	lastTradeTime       time.Time
	
	// Position tracking
	longPosition        *Position
	shortPosition       *Position
	
	// Indicators
	indicators          []indicators.TechnicalIndicator
	
	// Market regime detection
	regimeDetector      *MarketRegimeDetector
	
	// Performance tracking
	totalPnL            float64
	totalCommission     float64
	maxEquity           float64
	currentDrawdown     float64
}

// MarketRegimeDetector detects market conditions for strategy adaptation
type MarketRegimeDetector struct {
	atrPeriod           int
	trendPeriod         int
	volatilityThreshold float64
}

// MarketCondition represents the current market state
type MarketCondition int

const (
	MarketTrending MarketCondition = iota
	MarketRanging
	MarketVolatile
)

func (mc MarketCondition) String() string {
	switch mc {
	case MarketTrending:
		return "TRENDING"
	case MarketRanging:
		return "RANGING"
	case MarketVolatile:
		return "VOLATILE"
	default:
		return "UNKNOWN"
	}
}

// NewDualPositionStrategy creates a new dual position strategy
func NewDualPositionStrategy(baseAmount float64) *DualPositionStrategy {
	return &DualPositionStrategy{
		baseAmount:           baseAmount,
		hedgeRatio:           0.5, // Equal long/short by default
		maxMultiplier:        3.0,
		minConfidence:        0.4, // Reduced from 0.6 to be less restrictive
		
		// Risk management defaults
		stopLossPct:         0.05,  // 5% stop loss
		takeProfitPct:       0.03,  // 3% take profit
		trailingStopPct:     0.02,  // 2% trailing stop
		maxDrawdownPct:      0.10,  // 10% max drawdown
		positionSizeLimit:   0.25,  // 25% of balance per position max
		
		// Volatility and timing (more permissive for backtesting)
		volatilityThreshold: 0.002, // 0.2% minimum volatility (very permissive)
		timeBetweenEntries:  time.Second * 30, // 30 seconds between entries
		
		indicators:          make([]indicators.TechnicalIndicator, 0),
		regimeDetector:      &MarketRegimeDetector{
			atrPeriod:           14,
			trendPeriod:        50,
			volatilityThreshold: 0.02,
		},
	}
}

// Configuration setters
func (s *DualPositionStrategy) SetHedgeRatio(ratio float64) {
	s.hedgeRatio = math.Max(0.0, math.Min(1.0, ratio))
}

func (s *DualPositionStrategy) SetRiskParams(stopLoss, takeProfit, trailingStop, maxDrawdown float64) {
	s.stopLossPct = stopLoss
	s.takeProfitPct = takeProfit
	s.trailingStopPct = trailingStop
	s.maxDrawdownPct = maxDrawdown
}

func (s *DualPositionStrategy) SetVolatilityThreshold(threshold float64) {
	s.volatilityThreshold = threshold
}

func (s *DualPositionStrategy) SetTimeBetweenEntries(duration time.Duration) {
	s.timeBetweenEntries = duration
}

// AddIndicator adds a technical indicator to the strategy
func (s *DualPositionStrategy) AddIndicator(indicator indicators.TechnicalIndicator) {
	s.indicators = append(s.indicators, indicator)
}

// ShouldExecuteTrade implements the Strategy interface for dual position logic
func (s *DualPositionStrategy) ShouldExecuteTrade(data []types.OHLCV) (*TradeDecision, error) {
	if len(data) == 0 {
		return &TradeDecision{Action: ActionHold}, nil
	}

	currentTime := data[len(data)-1].Timestamp
	currentPrice := data[len(data)-1].Close

	// Update existing positions and check for exits
	s.updatePositions(currentPrice, currentTime)
	
	// Check if we should close positions based on risk management
	if decision := s.checkRiskManagement(currentPrice, currentTime); decision != nil {
		return decision, nil
	}
	
	// Time between entries check removed - enter whenever conditions are met
	
	// Collect signals from all indicators (no market condition checks)
	dualDecision := s.analyzeIndicators(data, currentPrice)
	
	// Convert to single TradeDecision for compatibility
	return s.convertDualDecisionToSingle(dualDecision, currentPrice, currentTime), nil
}

// analyzeIndicators collects and weights signals from all indicators
func (s *DualPositionStrategy) analyzeIndicators(data []types.OHLCV, currentPrice float64) *DualPositionTradeDecision {
	if len(s.indicators) == 0 {
		return &DualPositionTradeDecision{
			LongAction:  ActionHold,
			ShortAction: ActionHold,
			Reason:      "No indicators configured",
		}
	}

	buySignals := 0
	sellSignals := 0
	totalStrength := 0.0
	validIndicators := 0

	fmt.Printf("🔍 Analyzing %d indicators for dual position signals\n", len(s.indicators))
	
	for _, indicator := range s.indicators {
		// Check data sufficiency
		if len(data) < indicator.GetRequiredPeriods() {
			continue
		}

		shouldBuy, err := indicator.ShouldBuy(currentPrice, data)
		if err != nil {
			continue
		}

		shouldSell, err := indicator.ShouldSell(currentPrice, data)
		if err != nil {
			continue
		}

		strength := indicator.GetSignalStrength()
		validIndicators++

		signal := "HOLD"
		if shouldBuy {
			buySignals++
			totalStrength += strength
			signal = "BUY"
		} else if shouldSell {
			sellSignals++
			totalStrength -= strength
			signal = "SELL"
		}

		fmt.Printf("  %s: %s (Strength: %.2f%%)\n", 
			indicator.GetName(), signal, strength*100)
	}

	if validIndicators == 0 {
		return &DualPositionTradeDecision{
			LongAction:  ActionHold,
			ShortAction: ActionHold,
			Reason:      "No valid indicator signals",
		}
	}

	// Calculate confidence and determine actions
	buyConfidence := float64(buySignals) / float64(validIndicators)
	sellConfidence := float64(sellSignals) / float64(validIndicators)
	avgStrength := math.Abs(totalStrength) / float64(validIndicators)

	fmt.Printf("📊 Signal Summary: Buy=%.1f%%, Sell=%.1f%%, AvgStrength=%.1f%%, MinConf=%.1f%%\n", 
		buyConfidence*100, sellConfidence*100, avgStrength*100, s.minConfidence*100)

	// Make decisions based purely on indicator strength
	return s.makeDecisionFromIndicators(buyConfidence, sellConfidence, avgStrength, currentPrice)
}

// makeDecisionFromIndicators makes decisions based purely on indicator strength
func (s *DualPositionStrategy) makeDecisionFromIndicators(buyConf, sellConf, strength float64, price float64) *DualPositionTradeDecision {
	decision := &DualPositionTradeDecision{
		Confidence: math.Max(buyConf, sellConf),
		Strength:   strength,
		Timestamp:  time.Now(),
	}

	// Simple logic: if we have sufficient confidence, open positions
	if buyConf > s.minConfidence || sellConf > s.minConfidence {
		// Calculate position sizes based on strength
		longSize := s.calculatePositionSize(strength, buyConf, PositionLong)
		shortSize := s.calculatePositionSize(strength, sellConf, PositionShort)
		
		// Always use the configured hedge ratio
		if buyConf >= sellConf {
			// Bullish signal dominant - larger long position
			decision.LongAction = ActionBuy
			decision.LongAmount = longSize
			decision.ShortAction = ActionBuy
			decision.ShortAmount = longSize * s.hedgeRatio
			decision.Reason = fmt.Sprintf("Buy signal dominant (%.1f%% vs %.1f%%) - hedge ratio %.2f", 
				buyConf*100, sellConf*100, s.hedgeRatio)
		} else {
			// Bearish signal dominant - larger short position
			decision.ShortAction = ActionBuy
			decision.ShortAmount = shortSize
			decision.LongAction = ActionBuy
			decision.LongAmount = shortSize * s.hedgeRatio
			decision.Reason = fmt.Sprintf("Sell signal dominant (%.1f%% vs %.1f%%) - hedge ratio %.2f", 
				sellConf*100, buyConf*100, s.hedgeRatio)
		}
		
		fmt.Printf("🎯 Decision: Long=$%.2f, Short=$%.2f (Confidence: %.1f%%, Strength: %.1f%%)\n",
			decision.LongAmount, decision.ShortAmount, decision.Confidence*100, strength*100)
	} else {
		decision.LongAction = ActionHold
		decision.ShortAction = ActionHold
		decision.Reason = fmt.Sprintf("Insufficient confidence: Buy=%.1f%%, Sell=%.1f%% < Min=%.1f%%", 
			buyConf*100, sellConf*100, s.minConfidence*100)
		fmt.Printf("⏸️  Holding: %s\n", decision.Reason)
	}

	return decision
}

// calculatePositionSize calculates the appropriate position size
func (s *DualPositionStrategy) calculatePositionSize(strength, confidence float64, side PositionSide) float64 {
	// Base calculation similar to Enhanced DCA
	multiplier := 1.0 + (confidence * strength)
	if multiplier > s.maxMultiplier {
		multiplier = s.maxMultiplier
	}

	baseSize := s.baseAmount * multiplier

	// Adjust based on existing positions to manage risk
	if s.longPosition != nil && s.longPosition.IsActive && side == PositionLong {
		baseSize *= 0.7 // Reduce size if we already have a long position
	}
	if s.shortPosition != nil && s.shortPosition.IsActive && side == PositionShort {
		baseSize *= 0.7 // Reduce size if we already have a short position
	}

	return baseSize
}

// convertDualDecisionToSingle converts dual position decision to single TradeDecision for compatibility
func (s *DualPositionStrategy) convertDualDecisionToSingle(dual *DualPositionTradeDecision, price float64, timestamp time.Time) *TradeDecision {
	// Priority: if both actions are buy, execute the larger position first
	if dual.LongAction == ActionBuy && dual.ShortAction == ActionBuy {
		if dual.LongAmount >= dual.ShortAmount {
			// Execute long position first
			s.executeDualPosition(dual, price, timestamp, true)
			return &TradeDecision{
				Action:     ActionBuy,
				Amount:     dual.LongAmount,
				Confidence: dual.Confidence,
				Strength:   dual.Strength,
				Reason:     fmt.Sprintf("%s (LONG first)", dual.Reason),
				Timestamp:  timestamp,
			}
		} else {
			// Execute short position first
			s.executeDualPosition(dual, price, timestamp, false)
			return &TradeDecision{
				Action:     ActionBuy,
				Amount:     dual.ShortAmount,
				Confidence: dual.Confidence,
				Strength:   dual.Strength,
				Reason:     fmt.Sprintf("%s (SHORT first)", dual.Reason),
				Timestamp:  timestamp,
			}
		}
	} else if dual.LongAction == ActionBuy {
		return &TradeDecision{
			Action:     ActionBuy,
			Amount:     dual.LongAmount,
			Confidence: dual.Confidence,
			Strength:   dual.Strength,
			Reason:     fmt.Sprintf("%s (LONG only)", dual.Reason),
			Timestamp:  timestamp,
		}
	} else if dual.ShortAction == ActionBuy {
		return &TradeDecision{
			Action:     ActionBuy,
			Amount:     dual.ShortAmount,
			Confidence: dual.Confidence,
			Strength:   dual.Strength,
			Reason:     fmt.Sprintf("%s (SHORT only)", dual.Reason),
			Timestamp:  timestamp,
		}
	}

	return &TradeDecision{
		Action:     ActionHold,
		Confidence: dual.Confidence,
		Strength:   dual.Strength,
		Reason:     dual.Reason,
		Timestamp:  timestamp,
	}
}

// executeDualPosition handles the execution logic for dual positions
func (s *DualPositionStrategy) executeDualPosition(decision *DualPositionTradeDecision, price float64, timestamp time.Time, longFirst bool) {
	if longFirst && decision.LongAction == ActionBuy {
		s.openPosition(PositionLong, price, decision.LongAmount, timestamp)
		// Queue short position for next iteration
		s.lastTradeTime = timestamp
	} else if !longFirst && decision.ShortAction == ActionBuy {
		s.openPosition(PositionShort, price, decision.ShortAmount, timestamp)
		// Queue long position for next iteration
		s.lastTradeTime = timestamp
	}
}

// openPosition opens a new position with risk management setup
func (s *DualPositionStrategy) openPosition(side PositionSide, price, amount float64, timestamp time.Time) {
	quantity := amount / price
	
	position := &Position{
		Side:            side,
		EntryPrice:     price,
		Quantity:       quantity,
		IsActive:       true,
		EntryTime:      timestamp,
		TrailingStopPct: s.trailingStopPct,
	}

	// Set stop loss and take profit levels
	if side == PositionLong {
		position.StopLoss = price * (1 - s.stopLossPct)
		position.TakeProfit = price * (1 + s.takeProfitPct)
	} else {
		position.StopLoss = price * (1 + s.stopLossPct)
		position.TakeProfit = price * (1 - s.takeProfitPct)
	}

	// Store position
	if side == PositionLong {
		s.longPosition = position
	} else {
		s.shortPosition = position
	}

	fmt.Printf("🎯 Opened %s position: %.6f @ $%.2f (SL: $%.2f, TP: $%.2f)\n", 
		side, quantity, price, position.StopLoss, position.TakeProfit)
}

// updatePositions updates P&L and trailing stops for active positions
func (s *DualPositionStrategy) updatePositions(currentPrice float64, currentTime time.Time) {
	if s.longPosition != nil && s.longPosition.IsActive {
		s.updatePosition(s.longPosition, currentPrice, currentTime)
	}
	if s.shortPosition != nil && s.shortPosition.IsActive {
		s.updatePosition(s.shortPosition, currentPrice, currentTime)
	}
}

// updatePosition updates a single position's P&L and risk levels
func (s *DualPositionStrategy) updatePosition(pos *Position, currentPrice float64, currentTime time.Time) {
	var pnl float64
	
	if pos.Side == PositionLong {
		pnl = (currentPrice - pos.EntryPrice) * pos.Quantity
	} else {
		pnl = (pos.EntryPrice - currentPrice) * pos.Quantity
	}
	
	pos.UnrealizedPnL = pnl
	
	// Update max profit/loss tracking
	if pnl > pos.MaxProfit {
		pos.MaxProfit = pnl
		// Update trailing stop when we reach new profit highs
		if pos.Side == PositionLong {
			pos.StopLoss = math.Max(pos.StopLoss, currentPrice * (1 - pos.TrailingStopPct))
		} else {
			pos.StopLoss = math.Min(pos.StopLoss, currentPrice * (1 + pos.TrailingStopPct))
		}
	}
	
	if pnl < pos.MaxLoss {
		pos.MaxLoss = pnl
	}
}

// checkRiskManagement checks if any positions need to be closed due to risk rules
func (s *DualPositionStrategy) checkRiskManagement(currentPrice float64, currentTime time.Time) *TradeDecision {
	// Check long position
	if s.longPosition != nil && s.longPosition.IsActive {
		if shouldClose, reason := s.shouldClosePosition(s.longPosition, currentPrice); shouldClose {
			s.closePosition(s.longPosition, currentPrice, currentTime, reason)
			return &TradeDecision{
				Action:    ActionSell,
				Amount:    s.longPosition.Quantity * currentPrice,
				Reason:    fmt.Sprintf("Closed LONG: %s", reason),
				Timestamp: currentTime,
			}
		}
	}
	
	// Check short position
	if s.shortPosition != nil && s.shortPosition.IsActive {
		if shouldClose, reason := s.shouldClosePosition(s.shortPosition, currentPrice); shouldClose {
			s.closePosition(s.shortPosition, currentPrice, currentTime, reason)
			return &TradeDecision{
				Action:    ActionSell,
				Amount:    s.shortPosition.Quantity * currentPrice,
				Reason:    fmt.Sprintf("Closed SHORT: %s", reason),
				Timestamp: currentTime,
			}
		}
	}
	
	return nil
}

// shouldClosePosition determines if a position should be closed
func (s *DualPositionStrategy) shouldClosePosition(pos *Position, currentPrice float64) (bool, string) {
	if pos.Side == PositionLong {
		if currentPrice <= pos.StopLoss {
			return true, fmt.Sprintf("Stop Loss hit @ $%.2f", pos.StopLoss)
		}
		if currentPrice >= pos.TakeProfit {
			return true, fmt.Sprintf("Take Profit hit @ $%.2f", pos.TakeProfit)
		}
	} else {
		if currentPrice >= pos.StopLoss {
			return true, fmt.Sprintf("Stop Loss hit @ $%.2f", pos.StopLoss)
		}
		if currentPrice <= pos.TakeProfit {
			return true, fmt.Sprintf("Take Profit hit @ $%.2f", pos.TakeProfit)
		}
	}
	
	// Check maximum drawdown
	drawdownPct := math.Abs(pos.UnrealizedPnL) / (pos.EntryPrice * pos.Quantity)
	if drawdownPct > s.maxDrawdownPct {
		return true, fmt.Sprintf("Max Drawdown exceeded: %.2f%%", drawdownPct*100)
	}
	
	return false, ""
}

// closePosition closes a position and updates P&L tracking
func (s *DualPositionStrategy) closePosition(pos *Position, exitPrice float64, exitTime time.Time, reason string) {
	pos.IsActive = false
	pos.ExitTime = exitTime
	
	// Calculate realized P&L
	if pos.Side == PositionLong {
		pos.RealizedPnL = (exitPrice - pos.EntryPrice) * pos.Quantity
	} else {
		pos.RealizedPnL = (pos.EntryPrice - exitPrice) * pos.Quantity
	}
	
	// Update strategy totals
	s.totalPnL += pos.RealizedPnL
	
	fmt.Printf("🏁 Closed %s position: %.6f @ $%.2f | P&L: $%.2f | Reason: %s\n", 
		pos.Side, pos.Quantity, exitPrice, pos.RealizedPnL, reason)
}

// detectMarketRegime analyzes market conditions
func (s *DualPositionStrategy) detectMarketRegime(data []types.OHLCV) MarketCondition {
	if len(data) < s.regimeDetector.trendPeriod {
		return MarketRanging
	}

	// Calculate ATR for volatility
	atr := s.calculateATR(data, s.regimeDetector.atrPeriod)
	avgPrice := s.calculateAvgPrice(data, 20)
	volatility := atr / avgPrice

	// Calculate trend strength using SMAs
	sma20 := s.calculateSMA(data, 20)
	sma50 := s.calculateSMA(data, 50)
	trendStrength := math.Abs(sma20-sma50) / sma50

	if volatility > s.regimeDetector.volatilityThreshold*1.5 {
		return MarketVolatile
	} else if trendStrength > 0.02 {
		return MarketTrending
	}
	
	return MarketRanging
}

// hasEnoughVolatility checks if market volatility meets minimum threshold
func (s *DualPositionStrategy) hasEnoughVolatility(data []types.OHLCV) bool {
	// At start of dataset, use smaller periods for volatility calculation
	minPeriod := 3  // Minimum 3 candles needed for basic volatility
	if len(data) < minPeriod {
		// Not enough data even for basic volatility - assume volatile to allow trades
		fmt.Printf("📊 Early dataset: Insufficient data (%d candles), assuming volatile\n", len(data))
		return true
	}
	
	// Use adaptive period based on available data
	period := 14
	if len(data) < 14 {
		period = len(data) - 1  // Use all available data minus 1 for ATR calculation
		fmt.Printf("📊 Early dataset: Using %d-period ATR instead of 14\n", period)
	}
	
	atr := s.calculateATR(data, period)
	avgPrice := s.calculateAvgPrice(data, period)
	
	if avgPrice == 0 {
		// Fallback: assume volatile if can't calculate properly
		return true
	}
	
	volatility := atr / avgPrice
	
	return volatility >= s.volatilityThreshold
}

// Helper functions for calculations
func (s *DualPositionStrategy) calculateATR(data []types.OHLCV, period int) float64 {
	if len(data) < 2 {
		return 0  // Need at least 2 candles for True Range
	}
	
	// Adjust period if not enough data available
	availablePeriod := len(data) - 1  // -1 because we need previous candle for TR
	if period > availablePeriod {
		period = availablePeriod
	}
	
	if period <= 0 {
		return 0
	}
	
	atr := 0.0
	for i := len(data) - period; i < len(data); i++ {
		tr := math.Max(
			data[i].High-data[i].Low,
			math.Max(
				math.Abs(data[i].High-data[i-1].Close),
				math.Abs(data[i].Low-data[i-1].Close),
			),
		)
		atr += tr
	}
	return atr / float64(period)
}

func (s *DualPositionStrategy) calculateAvgPrice(data []types.OHLCV, period int) float64 {
	if len(data) == 0 {
		return 0
	}
	
	// Adjust period if not enough data available
	if period > len(data) {
		period = len(data)
	}
	
	if period <= 0 {
		return 0
	}
	
	sum := 0.0
	for i := len(data) - period; i < len(data); i++ {
		sum += data[i].Close
	}
	return sum / float64(period)
}

func (s *DualPositionStrategy) calculateSMA(data []types.OHLCV, period int) float64 {
	if len(data) < period {
		return 0
	}
	
	sum := 0.0
	for i := len(data) - period; i < len(data); i++ {
		sum += data[i].Close
	}
	return sum / float64(period)
}

// GetName returns the strategy name
func (s *DualPositionStrategy) GetName() string {
	return "Dual Position Strategy"
}

// OnCycleComplete is called when a take-profit cycle is completed
func (s *DualPositionStrategy) OnCycleComplete() {
	// Reset positions for next cycle
	if s.longPosition != nil {
		s.longPosition = nil
	}
	if s.shortPosition != nil {
		s.shortPosition = nil
	}
	
	fmt.Printf("🔄 Cycle complete - Total P&L: $%.2f, Total Commission: $%.2f\n", 
		s.totalPnL, s.totalCommission)
}

// GetPositionSummary returns current position information
func (s *DualPositionStrategy) GetPositionSummary() map[string]interface{} {
	summary := make(map[string]interface{})
	
	// Include hedge ratio and risk parameters in summary
	summary["hedge_ratio"] = s.hedgeRatio
	summary["stop_loss_pct"] = s.stopLossPct
	summary["take_profit_pct"] = s.takeProfitPct
	summary["trailing_stop_pct"] = s.trailingStopPct
	summary["max_drawdown_pct"] = s.maxDrawdownPct
	
	if s.longPosition != nil {
		summary["long"] = map[string]interface{}{
			"active":        s.longPosition.IsActive,
			"entry_price":   s.longPosition.EntryPrice,
			"quantity":      s.longPosition.Quantity,
			"unrealized_pnl": s.longPosition.UnrealizedPnL,
			"realized_pnl":   s.longPosition.RealizedPnL,
			"stop_loss":     s.longPosition.StopLoss,
			"take_profit":   s.longPosition.TakeProfit,
		}
	}
	
	if s.shortPosition != nil {
		summary["short"] = map[string]interface{}{
			"active":        s.shortPosition.IsActive,
			"entry_price":   s.shortPosition.EntryPrice,
			"quantity":      s.shortPosition.Quantity,
			"unrealized_pnl": s.shortPosition.UnrealizedPnL,
			"realized_pnl":   s.shortPosition.RealizedPnL,
			"stop_loss":     s.shortPosition.StopLoss,
			"take_profit":   s.shortPosition.TakeProfit,
		}
	}
	
	summary["total_pnl"] = s.totalPnL
	summary["total_commission"] = s.totalCommission
	
	return summary
}
