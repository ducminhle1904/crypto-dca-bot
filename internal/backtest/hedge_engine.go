package backtest

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// HedgeBacktestEngine is specialized for dual position strategies
type HedgeBacktestEngine struct {
	initialBalance float64
	commission     float64
	strategy       strategy.Strategy
	results        *HedgeBacktestResults
	
	// Current state
	balance        float64
	longPositions  []HedgePosition
	shortPositions []HedgePosition
	maxBalance     float64
	
	// Position tracking
	nextPositionID int
}

// HedgeBacktestResults contains results specific to hedging strategies
type HedgeBacktestResults struct {
	// Overall performance
	StartBalance      float64
	EndBalance        float64
	TotalReturn       float64
	MaxDrawdown       float64
	SharpeRatio       float64
	
	// Position statistics
	TotalPositions    int
	LongPositions     int
	ShortPositions    int
	
	// Performance by side
	LongProfit        float64
	ShortProfit       float64
	LongWinRate       float64
	ShortWinRate      float64
	
	// Risk metrics
	MaxConcurrentLongs  int
	MaxConcurrentShorts int
	LargestLongLoss     float64
	LargestShortLoss    float64
	LargestLongWin      float64
	LargestShortWin     float64
	
	// Detailed records
	Positions         []HedgePosition
	EquityCurve       []EquityPoint
	
	// Hedge-specific metrics
	HedgeEfficiency   float64  // How well the hedge protected against losses
	NetExposure       float64  // Average net exposure (long - short)
	VolatilityCapture float64  // Profit from volatility regardless of direction
}

// HedgePosition represents a single position in the hedge strategy
type HedgePosition struct {
	ID            int
	Side          PositionSide
	EntryTime     time.Time
	ExitTime      time.Time
	EntryPrice    float64
	ExitPrice     float64
	Quantity      float64
	EntryValue    float64
	ExitValue     float64
	PnL           float64
	Commission    float64
	IsOpen        bool
	StopLoss      float64
	TakeProfit    float64
	ExitReason    string
	MaxProfit     float64
	MaxLoss       float64
	Duration      time.Duration
}

// PositionSide represents long or short
type PositionSide int

const (
	SideLong PositionSide = iota
	SideShort
)

func (ps PositionSide) String() string {
	switch ps {
	case SideLong:
		return "LONG"
	case SideShort:
		return "SHORT"
	default:
		return "UNKNOWN"
	}
}

// EquityPoint represents a point in the equity curve
type EquityPoint struct {
	Time   time.Time
	Equity float64
	Price  float64
}

// NewHedgeBacktestEngine creates a new hedge-specific backtest engine
func NewHedgeBacktestEngine(initialBalance, commission float64, strat strategy.Strategy) *HedgeBacktestEngine {
	return &HedgeBacktestEngine{
		initialBalance: initialBalance,
		commission:     commission,
		strategy:       strat,
		balance:        initialBalance,
		maxBalance:     initialBalance,
		results: &HedgeBacktestResults{
			StartBalance: initialBalance,
			Positions:    make([]HedgePosition, 0),
			EquityCurve:  make([]EquityPoint, 0),
		},
	}
}

// Run executes the hedge backtest
func (h *HedgeBacktestEngine) Run(data []types.OHLCV, windowSize int) *HedgeBacktestResults {
	if len(data) == 0 {
		h.results.EndBalance = h.initialBalance
		h.results.TotalReturn = 0.0
		return h.results
	}

	for i := windowSize; i < len(data); i++ {
		window := data[i-windowSize : i+1]
		currentPrice := data[i].Close
		currentTime := data[i].Timestamp

		// Update existing positions
		h.updatePositions(currentPrice, currentTime)
		
		// Get strategy decision
		decision, err := h.strategy.ShouldExecuteTrade(window)
		if err == nil && decision.Action == strategy.ActionBuy {
			h.executeHedgeEntry(decision, currentPrice, currentTime)
		}
		
		// Update equity curve
		currentEquity := h.calculateTotalEquity(currentPrice)
		h.results.EquityCurve = append(h.results.EquityCurve, EquityPoint{
			Time:   currentTime,
			Equity: currentEquity,
			Price:  currentPrice,
		})
		
		// Update max balance and drawdown
		if currentEquity > h.maxBalance {
			h.maxBalance = currentEquity
		}
		
		drawdown := (h.maxBalance - currentEquity) / h.maxBalance
		if drawdown > h.results.MaxDrawdown {
			h.results.MaxDrawdown = drawdown
		}
	}
	
	// Final calculations
	finalPrice := data[len(data)-1].Close
	finalTime := data[len(data)-1].Timestamp
	h.closeAllPositions(finalPrice, finalTime, "End of backtest")
	
	h.calculateFinalResults()
	
	return h.results
}

// executeHedgeEntry handles entering hedge positions based on the dual position strategy
func (h *HedgeBacktestEngine) executeHedgeEntry(decision *strategy.TradeDecision, price float64, timestamp time.Time) {
	// For dual position strategy, we need to check what type of entry this is
	// The strategy should indicate whether this is a long, short, or dual entry
	
	// Get position summary from the strategy to understand current state
	if dualStrat, ok := h.strategy.(*strategy.DualPositionStrategy); ok {
		positionSummary := dualStrat.GetPositionSummary()
		
		// Get hedge ratio from strategy with safety check
		var hedgeRatio float64
		hedgeRatioInterface, exists := positionSummary["hedge_ratio"]
		if !exists || hedgeRatioInterface == nil {
			hedgeRatio = 0.5 // Default hedge ratio
		} else {
			hedgeRatio = hedgeRatioInterface.(float64)
		}
		
		// Check what positions we can open
		canOpenLong := h.shouldOpenPosition(SideLong, positionSummary)
		canOpenShort := h.shouldOpenPosition(SideShort, positionSummary)
		
		// If we can't open any new positions, return early
		if !canOpenLong && !canOpenShort {
			fmt.Printf("⏸️  Skipping entry: positions already open\n")
			return
		}
		
		// Determine what positions to open based on strategy state and market conditions
		longAmount := decision.Amount
		shortAmount := decision.Amount * hedgeRatio
		
		// Calculate required balance based on what positions we can open
		var totalRequired float64
		if canOpenLong && canOpenShort {
			totalRequired = (longAmount + shortAmount) * (1 + h.commission)
			fmt.Printf("💼 Entry Signal: Long=$%.2f, Short=$%.2f (ratio=%.2f)\n", 
				longAmount, shortAmount, hedgeRatio)
		} else if canOpenLong {
			totalRequired = longAmount * (1 + h.commission)
			fmt.Printf("📈 Entry Signal: Long=$%.2f (short position already open)\n", longAmount)
		} else if canOpenShort {
			totalRequired = shortAmount * (1 + h.commission)
			fmt.Printf("📉 Entry Signal: Short=$%.2f (long position already open)\n", shortAmount)
		}
		
		// Check if we have enough balance
		if h.balance >= totalRequired {
			// Open long position if allowed
			if canOpenLong {
				fmt.Printf("📈 Opening LONG position: $%.2f @ $%.2f\n", longAmount, price)
				h.openPosition(SideLong, price, longAmount, timestamp)
			}
			
			// Open short position if allowed
			if canOpenShort {
				fmt.Printf("📉 Opening SHORT position: $%.2f @ $%.2f\n", shortAmount, price)
				h.openPosition(SideShort, price, shortAmount, timestamp)
			}
		} else {
			fmt.Printf("⚠️ Insufficient balance: Need $%.2f, Have $%.2f\n", totalRequired, h.balance)
		}
	}
}

// shouldOpenPosition determines if we should open a position of the given side
func (h *HedgeBacktestEngine) shouldOpenPosition(side PositionSide, positionSummary map[string]interface{}) bool {
	// For dual position strategy, limit to 1 position per side at a time for simplicity
	activeCount := 0
	if side == SideLong {
		for _, pos := range h.longPositions {
			if pos.IsOpen {
				activeCount++
			}
		}
		return activeCount == 0 // Only allow 1 concurrent long position
	} else {
		for _, pos := range h.shortPositions {
			if pos.IsOpen {
				activeCount++
			}
		}
		return activeCount == 0 // Only allow 1 concurrent short position
	}
}

// openPosition opens a new hedge position
func (h *HedgeBacktestEngine) openPosition(side PositionSide, price, amount float64, timestamp time.Time) {
	quantity := amount / price
	commission := amount * h.commission
	totalCost := amount + commission
	
	// Check if we have enough balance
	if h.balance < totalCost {
		return
	}
	
	h.nextPositionID++
	position := HedgePosition{
		ID:         h.nextPositionID,
		Side:       side,
		EntryTime:  timestamp,
		EntryPrice: price,
		Quantity:   quantity,
		EntryValue: amount,
		Commission: commission,
		IsOpen:     true,
	}
	
	// Get risk parameters from strategy if it's a dual position strategy
	stopLossPct := 0.05   // Default 5%
	takeProfitPct := 0.03 // Default 3%
	
	if dualStrat, ok := h.strategy.(*strategy.DualPositionStrategy); ok {
		summary := dualStrat.GetPositionSummary()
		
		// Get risk parameters from strategy summary
		if sl, ok := summary["stop_loss_pct"].(float64); ok {
			stopLossPct = sl
		}
		if tp, ok := summary["take_profit_pct"].(float64); ok {
			takeProfitPct = tp
		}
	}
	
	// Set stop loss and take profit based on position side
	if side == SideLong {
		position.StopLoss = price * (1 - stopLossPct)
		position.TakeProfit = price * (1 + takeProfitPct)
	} else {
		position.StopLoss = price * (1 + stopLossPct)
		position.TakeProfit = price * (1 - takeProfitPct)
	}
	
	h.balance -= totalCost
	
	// Add to appropriate position slice
	if side == SideLong {
		h.longPositions = append(h.longPositions, position)
	} else {
		h.shortPositions = append(h.shortPositions, position)
	}
	
	fmt.Printf("🎯 Opened %s position #%d: %.6f @ $%.2f (SL: $%.2f, TP: $%.2f)\n", 
		side, position.ID, quantity, price, position.StopLoss, position.TakeProfit)
}

// updatePositions updates all open positions and checks for exits
func (h *HedgeBacktestEngine) updatePositions(currentPrice float64, currentTime time.Time) {
	// Update long positions
	for i := range h.longPositions {
		if h.longPositions[i].IsOpen {
			h.updateSinglePosition(&h.longPositions[i], currentPrice, currentTime)
		}
	}
	
	// Update short positions
	for i := range h.shortPositions {
		if h.shortPositions[i].IsOpen {
			h.updateSinglePosition(&h.shortPositions[i], currentPrice, currentTime)
		}
	}
}

// updateSinglePosition updates a single position and checks for exit conditions
func (h *HedgeBacktestEngine) updateSinglePosition(pos *HedgePosition, currentPrice float64, currentTime time.Time) {
	var unrealizedPnL float64
	
	if pos.Side == SideLong {
		unrealizedPnL = (currentPrice - pos.EntryPrice) * pos.Quantity
	} else {
		unrealizedPnL = (pos.EntryPrice - currentPrice) * pos.Quantity
	}
	
	// Update max profit/loss tracking
	if unrealizedPnL > pos.MaxProfit {
		pos.MaxProfit = unrealizedPnL
	}
	if unrealizedPnL < pos.MaxLoss {
		pos.MaxLoss = unrealizedPnL
	}
	
	// Check exit conditions
	shouldExit, reason := h.shouldExitPosition(pos, currentPrice)
	if shouldExit {
		h.closePosition(pos, currentPrice, currentTime, reason)
	}
}

// shouldExitPosition determines if a position should be closed
func (h *HedgeBacktestEngine) shouldExitPosition(pos *HedgePosition, currentPrice float64) (bool, string) {
	if pos.Side == SideLong {
		if currentPrice <= pos.StopLoss {
			return true, "Stop Loss"
		}
		if currentPrice >= pos.TakeProfit {
			return true, "Take Profit"
		}
	} else {
		if currentPrice >= pos.StopLoss {
			return true, "Stop Loss"
		}
		if currentPrice <= pos.TakeProfit {
			return true, "Take Profit"
		}
	}
	
	return false, ""
}

// closePosition closes a position and updates balance
func (h *HedgeBacktestEngine) closePosition(pos *HedgePosition, exitPrice float64, exitTime time.Time, reason string) {
	pos.IsOpen = false
	pos.ExitTime = exitTime
	pos.ExitPrice = exitPrice
	pos.ExitReason = reason
	pos.Duration = exitTime.Sub(pos.EntryTime)
	
	// Calculate exit value and commission
	pos.ExitValue = exitPrice * pos.Quantity
	exitCommission := pos.ExitValue * h.commission
	pos.Commission += exitCommission
	
	// Calculate P&L (gross profit/loss before commission)
	if pos.Side == SideLong {
		pos.PnL = (exitPrice - pos.EntryPrice) * pos.Quantity
	} else {
		pos.PnL = (pos.EntryPrice - exitPrice) * pos.Quantity
	}
	
	// Subtract total commission from P&L (entry + exit)
	pos.PnL -= pos.Commission
	
	// Update balance: we get back the exit value minus exit commission
	proceeds := pos.ExitValue - exitCommission
	h.balance += proceeds
	
	fmt.Printf("🏁 Closed %s position #%d: %.6f @ $%.2f | P&L: $%.2f | Balance: $%.2f\n", 
		pos.Side, pos.ID, pos.Quantity, exitPrice, pos.PnL, h.balance)
}

// closeAllPositions closes all open positions (used at end of backtest)
func (h *HedgeBacktestEngine) closeAllPositions(price float64, timestamp time.Time, reason string) {
	// Close long positions
	for i := range h.longPositions {
		if h.longPositions[i].IsOpen {
			h.closePosition(&h.longPositions[i], price, timestamp, reason)
		}
	}
	
	// Close short positions
	for i := range h.shortPositions {
		if h.shortPositions[i].IsOpen {
			h.closePosition(&h.shortPositions[i], price, timestamp, reason)
		}
	}
}

// calculateTotalEquity calculates current total equity including unrealized P&L
func (h *HedgeBacktestEngine) calculateTotalEquity(currentPrice float64) float64 {
	equity := h.balance
	
	// Add unrealized P&L from long positions
	for _, pos := range h.longPositions {
		if pos.IsOpen {
			unrealizedPnL := (currentPrice - pos.EntryPrice) * pos.Quantity
			equity += unrealizedPnL
		}
	}
	
	// Add unrealized P&L from short positions
	for _, pos := range h.shortPositions {
		if pos.IsOpen {
			unrealizedPnL := (pos.EntryPrice - currentPrice) * pos.Quantity
			equity += unrealizedPnL
		}
	}
	
	return equity
}

// calculateFinalResults computes final statistics
func (h *HedgeBacktestEngine) calculateFinalResults() {
	allPositions := append(h.longPositions, h.shortPositions...)
	h.results.Positions = allPositions
	h.results.EndBalance = h.balance
	h.results.TotalReturn = (h.balance - h.initialBalance) / h.initialBalance
	
	// Count positions and calculate profits
	longWins, shortWins := 0, 0
	longCount, shortCount := 0, 0
	
	for _, pos := range allPositions {
		if pos.Side == SideLong {
			longCount++
			h.results.LongProfit += pos.PnL
			if pos.PnL > 0 {
				longWins++
			}
			if pos.PnL > h.results.LargestLongWin {
				h.results.LargestLongWin = pos.PnL
			}
			if pos.PnL < h.results.LargestLongLoss {
				h.results.LargestLongLoss = pos.PnL
			}
		} else {
			shortCount++
			h.results.ShortProfit += pos.PnL
			if pos.PnL > 0 {
				shortWins++
			}
			if pos.PnL > h.results.LargestShortWin {
				h.results.LargestShortWin = pos.PnL
			}
			if pos.PnL < h.results.LargestShortLoss {
				h.results.LargestShortLoss = pos.PnL
			}
		}
	}
	
	h.results.TotalPositions = len(allPositions)
	h.results.LongPositions = longCount
	h.results.ShortPositions = shortCount
	
	if longCount > 0 {
		h.results.LongWinRate = float64(longWins) / float64(longCount)
	}
	if shortCount > 0 {
		h.results.ShortWinRate = float64(shortWins) / float64(shortCount)
	}
	
	// Calculate hedge-specific metrics
	h.calculateHedgeMetrics()
}

// calculateHedgeMetrics calculates metrics specific to hedge strategies
func (h *HedgeBacktestEngine) calculateHedgeMetrics() {
	if len(h.results.Positions) == 0 {
		return
	}
	
	// Calculate hedge efficiency (how well short positions offset long losses)
	longLosses := 0.0
	shortGainsFromLongLosses := 0.0
	
	for _, pos := range h.results.Positions {
		if pos.Side == SideLong && pos.PnL < 0 {
			longLosses += math.Abs(pos.PnL)
		}
		if pos.Side == SideShort && pos.PnL > 0 {
			shortGainsFromLongLosses += pos.PnL
		}
	}
	
	if longLosses > 0 {
		h.results.HedgeEfficiency = shortGainsFromLongLosses / longLosses
	}
	
	// Calculate average net exposure
	netExposureSum := 0.0
	exposurePoints := 0
	
	for _, point := range h.results.EquityCurve {
		longExposure := 0.0
		shortExposure := 0.0
		
		for _, pos := range h.results.Positions {
			if pos.EntryTime.Before(point.Time) && (pos.ExitTime.IsZero() || pos.ExitTime.After(point.Time)) {
				if pos.Side == SideLong {
					longExposure += pos.EntryValue
				} else {
					shortExposure += pos.EntryValue
				}
			}
		}
		
		netExposure := longExposure - shortExposure
		netExposureSum += math.Abs(netExposure)
		exposurePoints++
	}
	
	if exposurePoints > 0 {
		h.results.NetExposure = netExposureSum / float64(exposurePoints)
	}
	
	// Volatility capture (total profit from both sides)
	h.results.VolatilityCapture = h.results.LongProfit + h.results.ShortProfit
}

// PrintSummary prints a detailed summary of the hedge backtest results
func (h *HedgeBacktestResults) PrintSummary() {
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
	fmt.Printf("🔄 HEDGE STRATEGY BACKTEST RESULTS\n")
	fmt.Printf(strings.Repeat("=", 60) + "\n")
	
	fmt.Printf("💰 Initial Balance:    $%.2f\n", h.StartBalance)
	fmt.Printf("💰 Final Balance:      $%.2f\n", h.EndBalance)
	fmt.Printf("📈 Total Return:       %.2f%%\n", h.TotalReturn*100)
	fmt.Printf("📉 Max Drawdown:       %.2f%%\n", h.MaxDrawdown*100)
	
	fmt.Printf("\n" + strings.Repeat("-", 40) + "\n")
	fmt.Printf("📊 POSITION STATISTICS\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	fmt.Printf("🎯 Total Positions:    %d\n", h.TotalPositions)
	fmt.Printf("📈 Long Positions:     %d (Win Rate: %.1f%%)\n", h.LongPositions, h.LongWinRate*100)
	fmt.Printf("📉 Short Positions:    %d (Win Rate: %.1f%%)\n", h.ShortPositions, h.ShortWinRate*100)
	
	fmt.Printf("\n" + strings.Repeat("-", 40) + "\n")
	fmt.Printf("💹 PROFIT BREAKDOWN\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	fmt.Printf("📈 Long P&L:           $%.2f\n", h.LongProfit)
	fmt.Printf("📉 Short P&L:          $%.2f\n", h.ShortProfit)
	fmt.Printf("🎯 Volatility Capture: $%.2f\n", h.VolatilityCapture)
	
	fmt.Printf("\n" + strings.Repeat("-", 40) + "\n")
	fmt.Printf("🛡️ HEDGE METRICS\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	fmt.Printf("🔄 Hedge Efficiency:   %.2f%% (how well shorts offset long losses)\n", h.HedgeEfficiency*100)
	fmt.Printf("⚖️  Avg Net Exposure:   $%.2f\n", h.NetExposure)
	
	fmt.Printf("\n" + strings.Repeat("-", 40) + "\n")
	fmt.Printf("🎯 RISK ANALYSIS\n")
	fmt.Printf(strings.Repeat("-", 40) + "\n")
	fmt.Printf("📈 Largest Long Win:   $%.2f\n", h.LargestLongWin)
	fmt.Printf("📈 Largest Long Loss:  $%.2f\n", h.LargestLongLoss)
	fmt.Printf("📉 Largest Short Win:  $%.2f\n", h.LargestShortWin)
	fmt.Printf("📉 Largest Short Loss: $%.2f\n", h.LargestShortLoss)
	
	fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
}
