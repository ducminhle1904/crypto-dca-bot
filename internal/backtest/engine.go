package backtest

import (
	"fmt"
	"math"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// TPLevel structure for multiple take profit system
type TPLevel struct {
	Level       int        // 1-5
	Percent     float64    // TP percentage (e.g., 0.02 for 2%)
	Quantity    float64    // Always 0.20 (20%)
	Hit         bool       // Whether this level was triggered
	HitTime     *time.Time // When this level was hit
	HitPrice    float64    // Price when hit
	PnL         float64    // PnL for this level (net of sell commission)
	SoldQty     float64    // Actual quantity sold at this TP level
	SellCommission float64 // Actual commission paid on the partial sell
}

type BacktestEngine struct {
	initialBalance float64
	commission     float64
	strategy       strategy.Strategy
	results        *BacktestResults
	// take-profit as a decimal (e.g., 0.01 for 1%). 0 disables TP and behaves like before
	tpPercent      float64

	// Multiple TP level configuration
	useTPLevels    bool       // Enable 5-level TP mode
	tpLevels       []TPLevel  // 5 TP levels configuration

	// Minimum lot size constraints for realistic simulation
	minOrderQty    float64 // Minimum order quantity (e.g., 0.01 for BTCUSDT)
	
	// Current position (units) for equity/drawdown calc
	position       float64
	
	// Current balance tracking
	balance        float64 // Current balance during backtest
	
	// cycle tracking (only meaningful when tpPercent > 0 or useTPLevels is true)
	cycleOpen          bool
	currentCycleNumber int
	cycleEntries       int
	cycleStartTime     time.Time
	cycleQtySum        float64
	cycleCostSum         float64 // sum(entryPrice * qty) - net cost after commission
	cycleGrossCostSum    float64 // sum(entryPrice * gross_qty) - gross cost before commission  
	cycleGrossQtySum     float64 // sum of gross quantities (before commission), for avg gross entry
	cycleCommissionSum   float64 // sum of commission paid in current cycle
	
	// Enhanced cycle tracking for multiple TPs
	cycleTPProgress    map[int]bool  // Track which TP levels are hit
	cycleRemainingQty  float64       // Remaining quantity after partial exits
	cycleUnrealizedPnL float64       // Accumulated unrealized PnL from partial TP exits
	
	// Enhanced drawdown and exposure tracking
	peakEquity         float64       // Peak equity for drawdown calculation
	maxCycleExposure   float64       // Maximum exposure within current cycle
	currentExposure    float64       // Current exposure level
	exposureHistory    []float64     // Historical exposure levels
}

type BacktestResults struct {
	TotalReturn   float64
	MaxDrawdown   float64
	SharpeRatio   float64
	TotalTrades   int
	WinningTrades int
	LosingTrades  int
	ProfitFactor  float64
	StartBalance  float64
	EndBalance    float64
	Trades        []Trade
	// Cycle summaries
	Cycles            []CycleSummary
	CompletedCycles   int
	// Enhanced metrics
	EquityCurve       []EquityPoint
	SortinoRatio      float64
	CalmarRatio       float64
	AnnualizedReturn  float64
	AnnualizedSharpe  float64
	MaxExposure       float64
	AvgExposure       float64
	TotalTurnover     float64
	// Enhanced drawdown metrics
	MaxIntraCycleDD   float64       // Maximum drawdown within a single cycle
	AvgCycleExposure  float64       // Average exposure per cycle
	MaxCycleExposure  float64       // Maximum exposure within any cycle
}

type Trade struct {
	EntryTime  time.Time
	ExitTime   time.Time
	EntryPrice float64
	ExitPrice  float64
	Quantity   float64
	PnL        float64
	Commission float64
	Cycle      int // 0 if no TP cycle tracking (tpPercent==0), otherwise cycle id
}

type CycleSummary struct {
	CycleNumber     int
	StartTime       time.Time
	EndTime         time.Time
	Entries         int
	AvgEntry        float64    // Weighted average entry price (net cost basis)
	AvgGrossEntry   float64    // Weighted average entry price (gross cost basis)
	TargetPrice     float64
	RealizedPnL     float64
	TotalCost       float64    // Total net cost invested (after commission)
	TotalGrossCost  float64    // Total gross cost invested (before commission)
	TotalCommission float64    // Total commission paid in this cycle
	Completed       bool       // true if closed by TP, false if left open at end
	
	// Multiple TP tracking
	TPLevelsHit        int           `json:"tp_levels_hit"`
	PartialExits       []PartialExit `json:"partial_exits"`
	FinalExitPrice     float64       `json:"final_exit_price"`
	TotalRealizedPnL   float64       `json:"total_realized_pnl"`
}

type PartialExit struct {
	TPLevel    int       `json:"tp_level"`
	Quantity   float64   `json:"quantity"`
	Price      float64   `json:"price"`
	Timestamp  time.Time `json:"timestamp"`
	PnL        float64   `json:"pnl"`
	Commission float64   `json:"commission"`
}

// EquityPoint represents a point in the equity curve
type EquityPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Balance   float64   `json:"balance"`
	Position  float64   `json:"position"`
	Price     float64   `json:"price"`
	Equity    float64   `json:"equity"`
	Exposure  float64   `json:"exposure"`  // Position value / total equity
	PnL       float64   `json:"pnl"`      // Running PnL from start
}

func NewBacktestEngine(
	initialBalance float64,
	commission float64,
	strat strategy.Strategy,
	// tpPercent: 0 disables take-profit. Example: 0.02 = 2% above avg entry
	tpPercent float64,
	// minOrderQty: minimum order quantity (e.g., 0.01 for BTCUSDT). 0 disables lot size constraints
	minOrderQty float64,
	// useTPLevels: enable 5-level TP mode using progressive tpPercent levels
	useTPLevels bool,
) *BacktestEngine {
	engine := &BacktestEngine{
		initialBalance: initialBalance,
		commission:     commission,
		strategy:       strat,
		results: &BacktestResults{
			StartBalance: initialBalance,
			Trades:       make([]Trade, 0),
			Cycles:       make([]CycleSummary, 0),
			EquityCurve:  make([]EquityPoint, 0),
		},
		tpPercent:   tpPercent,
		useTPLevels: useTPLevels,
		minOrderQty: minOrderQty,
		balance:     initialBalance,
		position:    0,
		// Initialize enhanced tracking
		peakEquity:       initialBalance,
		maxCycleExposure: 0,
		currentExposure:  0,
		exposureHistory:  make([]float64, 0),
	}
	
	// Initialize TP level tracking
	if useTPLevels {
		// Auto-generate 5 TP levels based on tpPercent
		engine.tpLevels = make([]TPLevel, 5)
		for i := 0; i < 5; i++ {
			engine.tpLevels[i] = TPLevel{
				Level:    i + 1,
				Percent:  tpPercent * float64(i+1) / 5.0, // Progressive: 20%, 40%, 60%, 80%, 100% of tpPercent
				Quantity: 0.20, // Always 20% per level
				Hit:      false,
			}
		}
		
		engine.cycleTPProgress = make(map[int]bool)
		for i := 0; i < 5; i++ {
			engine.cycleTPProgress[i] = false
		}
	}
	
	return engine
}

func (b *BacktestEngine) Run(data []types.OHLCV, windowSize int) *BacktestResults {
	// Handle empty or insufficient data
	if len(data) == 0 {
		b.results.EndBalance = b.initialBalance
		b.results.TotalReturn = 0.0
		b.results.TotalTrades = 0
		return b.results
	}

	// Initialize balance and position
	b.balance = b.initialBalance
	b.position = 0.0
	maxBalance := b.balance
	
	// Initialize enhanced tracking
	b.peakEquity = b.initialBalance
	b.exposureHistory = make([]float64, 0)
	cyclePeakEquity := b.initialBalance

	for i := windowSize; i < len(data); i++ {
		// get a data window for analysis
		window := data[i-windowSize : i+1]
		currentPrice := data[i].Close

		// get a signal from the strategy
		decision, err := b.strategy.ShouldExecuteTrade(window)
		if err == nil && decision.Action == strategy.ActionBuy {
			// Calculate initial quantity and amount
			targetAmount := decision.Amount
			quantity := targetAmount / currentPrice
			actualAmount := targetAmount

			// Apply minimum lot size constraint and step size (simulate real exchange behavior)
			if b.minOrderQty > 0 {
				// Round to nearest multiple of minOrderQty (step size)
				multiplier := math.Round(quantity / b.minOrderQty)
				// Ensure at least 1 step (minimum quantity)
				if multiplier < 1 {
					multiplier = 1
				}
				adjustedQuantity := multiplier * b.minOrderQty
				if adjustedQuantity != quantity {
					quantity = adjustedQuantity
					actualAmount = quantity * currentPrice
				}
			}

			// Calculate commission on executed notional (after lot adjustment)
			commission := actualAmount * b.commission
			totalCost := actualAmount + commission
			
			// Check if we have enough balance for total cost (adjusted amount + commission)
			if b.balance >= totalCost {
				// Execute buy with actual amount, but deduct commission separately
				netAmount := actualAmount // The lot-adjusted amount becomes the net investment
				actualQuantity := netAmount / currentPrice

				b.position += actualQuantity
				b.balance -= totalCost // Deduct both adjusted amount and commission

				// begin a new cycle if needed (only when TP enabled)
				if (b.tpPercent > 0 || b.useTPLevels) && !b.cycleOpen {
					b.currentCycleNumber++
					b.cycleOpen = true
					b.cycleEntries = 0
					b.cycleStartTime = data[i].Timestamp
					b.cycleQtySum = 0
					b.cycleCostSum = 0
					b.cycleGrossCostSum = 0
					b.cycleGrossQtySum = 0
					b.cycleCommissionSum = 0
					b.cycleRemainingQty = 0
					b.cycleUnrealizedPnL = 0
					
					// Reset TP level progress for new cycle
					if b.useTPLevels {
						for i := range b.tpLevels {
							b.cycleTPProgress[i] = false
							b.tpLevels[i].Hit = false
							b.tpLevels[i].HitTime = nil
							b.tpLevels[i].HitPrice = 0
							b.tpLevels[i].PnL = 0
							b.tpLevels[i].SoldQty = 0
							b.tpLevels[i].SellCommission = 0
						}
					}
				}

				//Recording the transaction (input) - use actual executed values
				trade := Trade{
					EntryTime:  data[i].Timestamp,
					EntryPrice: currentPrice,
					Quantity:   actualQuantity, // Use actual quantity after commission
					Commission: commission,
				}
				if b.cycleOpen {
					b.cycleEntries++
					b.cycleQtySum += actualQuantity
					b.cycleRemainingQty += actualQuantity
					// Track net cost (actual quantity after commission deduction)
					b.cycleCostSum += currentPrice * actualQuantity
					// Track gross cost (what we would have bought without commission)
					grossQuantity := actualAmount / currentPrice
					b.cycleGrossCostSum += currentPrice * grossQuantity
					b.cycleGrossQtySum += grossQuantity
					// Track commission for this cycle
					b.cycleCommissionSum += commission
					trade.Cycle = b.currentCycleNumber
					
					// Initialize absolute TP quantities on first entry of cycle
					if b.useTPLevels && b.cycleEntries == 1 {
						b.setTPLevelsQuantities(b.cycleRemainingQty)
					}
					// Reset TP levels when new DCA entry is added (recalculate and start from TP1)
					if b.useTPLevels && b.cycleEntries > 1 {
						b.resetTPLevelsForNewEntry()
					}
				}

				b.results.Trades = append(b.results.Trades, trade)
			}
		}

		// Check and execute take profit orders
		if b.cycleOpen && b.position > 0 {
			if b.useTPLevels {
				b.checkAndExecuteMultipleTP(currentPrice, data[i].Timestamp)
			} else if b.tpPercent > 0 {
				b.checkAndExecuteSingleTP(currentPrice, data[i].Timestamp)
			}
		}

		//Updating metrics (equity tracking)
		currentValue := b.balance + (b.position * currentPrice)
		if currentValue > maxBalance {
			maxBalance = currentValue
		}

		// Enhanced peak tracking
		if currentValue > b.peakEquity {
			b.peakEquity = currentValue
		}
		
		// Track cycle peak equity for intra-cycle drawdown
		if b.cycleOpen {
			if currentValue > cyclePeakEquity {
				cyclePeakEquity = currentValue
			}
			
			// Calculate intra-cycle drawdown
			if cyclePeakEquity > 0 {
				intraCycleDD := (cyclePeakEquity - currentValue) / cyclePeakEquity
				if intraCycleDD > b.results.MaxIntraCycleDD {
					b.results.MaxIntraCycleDD = intraCycleDD
				}
			}
		} else {
			// Reset cycle peak when starting new cycle
			cyclePeakEquity = currentValue
		}

		//Calculating the drawdown
		drawdown := (maxBalance - currentValue) / maxBalance
		if drawdown > b.results.MaxDrawdown {
			b.results.MaxDrawdown = drawdown
		}

		// Track equity curve and exposure
		exposure := 0.0
		if currentValue > 0 {
			exposure = (b.position * currentPrice) / currentValue
		}
		
		// Update exposure tracking
		b.currentExposure = exposure
		b.exposureHistory = append(b.exposureHistory, exposure)
		
		// Track cycle exposure
		if b.cycleOpen && exposure > b.maxCycleExposure {
			b.maxCycleExposure = exposure
		}
		
		runningPnL := currentValue - b.initialBalance
		
		equityPoint := EquityPoint{
			Timestamp: data[i].Timestamp,
			Balance:   b.balance,
			Position:  b.position,
			Price:     currentPrice,
			Equity:    currentValue,
			Exposure:  exposure,
			PnL:       runningPnL,
		}
		b.results.EquityCurve = append(b.results.EquityCurve, equityPoint)
	}

	//Final calculations
	finalPrice := data[len(data)-1].Close
	finalTime := data[len(data)-1].Timestamp

	// Safe finalization logic for remaining trades
	if b.useTPLevels {
		// In multi-TP mode, only create a final exit trade for the actual remaining position
		// Don't auto-close original entry trades since they would show incorrect PnL
		// (partial exits are already handled as synthetic trades)
		if b.position > 0 {
			// Create a single synthetic trade for the remaining position
			avgEntry := 0.0
			if b.cycleOpen && b.cycleQtySum > 0 {
				avgEntry = b.cycleCostSum / b.cycleQtySum
			} else {
				// If no active cycle, calculate from all open entry trades
				totalQty := 0.0
				totalCost := 0.0
				for _, trade := range b.results.Trades {
					if trade.ExitTime.IsZero() && trade.Cycle > 0 { // Original entry trades have Cycle > 0
						totalQty += trade.Quantity
						totalCost += trade.EntryPrice * trade.Quantity
					}
				}
				if totalQty > 0 {
					avgEntry = totalCost / totalQty
				}
			}
			
			// Create final exit trade for remaining position (no commission on mark-to-market)
			finalExitTrade := Trade{
				EntryTime:  finalTime,
				ExitTime:   finalTime,
				EntryPrice: avgEntry,
				ExitPrice:  finalPrice,
				Quantity:   b.position,
				PnL:        (finalPrice - avgEntry) * b.position, // No commission on final mark-to-market
				Commission: 0.0,
				Cycle:      b.currentCycleNumber,
			}
			b.results.Trades = append(b.results.Trades, finalExitTrade)
		}
		
		// Don't auto-close original entry trades in multi-TP mode to avoid PnL inconsistencies
		// The synthetic partial exit trades and final exit trade above represent the actual execution
	} else {
		// Original logic for single-TP or no-TP mode: auto-close remaining open trades
		for i := range b.results.Trades {
			trade := &b.results.Trades[i]
			if trade.ExitTime.IsZero() {
				trade.ExitTime = finalTime
				trade.ExitPrice = finalPrice
				trade.PnL = (finalPrice-trade.EntryPrice)*trade.Quantity - trade.Commission
			}
		}
	}

	// Add mark-to-market value of any remaining position to balance (no extra commission)
	if b.position > 0 {
		b.balance += b.position * finalPrice
		b.position = 0
	}

	// If a cycle is still open at the end, record it as incomplete (both single TP and TP-levels)
	if (b.tpPercent > 0 || b.useTPLevels) && b.cycleOpen && b.cycleQtySum > 0 {
		avgEntry := b.cycleCostSum / b.cycleQtySum
		avgGrossEntry := 0.0
		if b.cycleGrossQtySum > 0 { avgGrossEntry = b.cycleGrossCostSum / b.cycleGrossQtySum }
		// Use accumulated unrealized PnL for incomplete cycles
		realized := b.cycleUnrealizedPnL
		partialExits := make([]PartialExit, 0)
		if b.useTPLevels {
			for i, tp := range b.tpLevels {
				if tp.Hit {
					partialExits = append(partialExits, PartialExit{
						TPLevel:    i + 1,
						Quantity:   tp.SoldQty,
						Price:      tp.HitPrice,
						Timestamp:  *tp.HitTime,
						PnL:        tp.PnL,
						Commission: tp.SellCommission,
					})
				}
			}
		}
		target := 0.0
		if !b.useTPLevels && b.tpPercent > 0 { target = avgEntry * (1.0 + b.tpPercent) }
		b.results.Cycles = append(b.results.Cycles, CycleSummary{
			CycleNumber:       b.currentCycleNumber,
			StartTime:         b.cycleStartTime,
			EndTime:           finalTime,
			Entries:           b.cycleEntries,
			AvgEntry:          avgEntry,          // Net average entry
			AvgGrossEntry:     avgGrossEntry,     // Gross average entry
			TargetPrice:       target,
			RealizedPnL:       realized,
			TotalCost:         b.cycleCostSum,        // Net cost
			TotalGrossCost:    b.cycleGrossCostSum,   // Gross cost
			TotalCommission:   b.cycleCommissionSum,  // Commission
			Completed:         false,
			TPLevelsHit:       len(partialExits),
			PartialExits:      partialExits,
			FinalExitPrice:    0,
			TotalRealizedPnL:  realized,
		})
		// Keep CompletedCycles unchanged
	}

	// Set final results
	b.results.EndBalance = b.balance
	b.results.TotalReturn = (b.balance - b.initialBalance) / b.initialBalance
	b.results.TotalTrades = len(b.results.Trades)
	
	// Calculate enhanced metrics from tracking data
	b.results.MaxCycleExposure = b.maxCycleExposure
	if len(b.exposureHistory) > 0 {
		totalExposure := 0.0
		for _, exp := range b.exposureHistory {
			totalExposure += exp
		}
		b.results.AvgCycleExposure = totalExposure / float64(len(b.exposureHistory))
	}

	return b.results
}



func (b *BacktestResults) PrintSummary() {
	fmt.Printf("=== Backtest Results ===\n")
	fmt.Printf("Initial Balance: $%.2f\n", b.StartBalance)
	fmt.Printf("Final Balance: $%.2f\n", b.EndBalance)
	fmt.Printf("Total Return: %.2f%%\n", b.TotalReturn*100)
	fmt.Printf("Max Drawdown: %.2f%%\n", b.MaxDrawdown*100)
	fmt.Printf("Total Trades: %d\n", b.TotalTrades)
	fmt.Printf("Profit Factor: %.2f\n", b.ProfitFactor)
	if len(b.Cycles) > 0 {
		fmt.Printf("Completed Cycles: %d (Total Cycles: %d)\n", b.CompletedCycles, len(b.Cycles))
		b.PrintCycleDetails()
	}
}

// PrintCycleDetails prints detailed information about each cycle
func (b *BacktestResults) PrintCycleDetails() {
	if len(b.Cycles) == 0 {
		return
	}
	
	fmt.Printf("\n=== Cycle Details ===\n")
	totalCommission := 0.0
	for _, cycle := range b.Cycles {
		status := "✅ Completed"
		if !cycle.Completed {
			status = "⏳ Incomplete"
		}
		
		fmt.Printf("Cycle #%d %s\n", cycle.CycleNumber, status)
		fmt.Printf("  Entries: %d\n", cycle.Entries)
		fmt.Printf("  Net Avg Entry: $%.2f\n", cycle.AvgEntry)
		fmt.Printf("  Gross Avg Entry: $%.2f\n", cycle.AvgGrossEntry) 
		fmt.Printf("  Total Net Cost: $%.2f\n", cycle.TotalCost)
		fmt.Printf("  Total Gross Cost: $%.2f\n", cycle.TotalGrossCost)
		fmt.Printf("  Total Commission: $%.2f (%.3f%%)\n", 
			cycle.TotalCommission, 
			(cycle.TotalCommission/cycle.TotalGrossCost)*100)
		if cycle.Completed {
			fmt.Printf("  Target Price: $%.2f\n", cycle.TargetPrice)
		}
		fmt.Printf("  Realized PnL: $%.2f\n", cycle.RealizedPnL)
		fmt.Printf("  Duration: %s\n", cycle.EndTime.Sub(cycle.StartTime).String())
		fmt.Printf("\n")
		
		totalCommission += cycle.TotalCommission
	}
	fmt.Printf("Total Commission Paid: $%.2f\n", totalCommission)
}
// checkAndExecuteSingleTP handles single TP logic (original behavior)
func (b *BacktestEngine) checkAndExecuteSingleTP(currentPrice float64, timestamp time.Time) {
	// compute weighted average entry price across OPEN trades (of current cycle)
	totalQty := 0.0
	sumEntryCost := 0.0 // sum(entryPrice * qty)
	for _, t := range b.results.Trades {
		if t.ExitTime.IsZero() && t.Cycle == b.currentCycleNumber {
			totalQty += t.Quantity
			sumEntryCost += t.EntryPrice * t.Quantity
		}
	}
	if totalQty > 0 {
		avgEntry := sumEntryCost / totalQty
		target := avgEntry * (1.0 + b.tpPercent)
		if currentPrice >= target {
			// Realize PnL: sell all open quantity
			proceeds := totalQty * currentPrice
			sellCommission := proceeds * b.commission
			b.balance += proceeds - sellCommission
			b.position -= totalQty

			// Proportionally assign sell commission and finalize open trades for this cycle
			realized := 0.0
			for idx := range b.results.Trades {
				if b.results.Trades[idx].ExitTime.IsZero() && b.results.Trades[idx].Cycle == b.currentCycleNumber {
					q := b.results.Trades[idx].Quantity
					share := 0.0
					if totalQty > 0 { share = q / totalQty }
					perTradeSellComm := sellCommission * share
					b.results.Trades[idx].ExitTime = timestamp
					b.results.Trades[idx].ExitPrice = currentPrice
					pnl := (currentPrice-b.results.Trades[idx].EntryPrice)*q - b.results.Trades[idx].Commission - perTradeSellComm
					b.results.Trades[idx].PnL = pnl
					realized += pnl
				}
			}

			// finalize cycle summary
			avgGrossEntry := 0.0
			if b.cycleGrossQtySum > 0 { avgGrossEntry = b.cycleGrossCostSum / b.cycleGrossQtySum }
			b.results.Cycles = append(b.results.Cycles, CycleSummary{
				CycleNumber:     b.currentCycleNumber,
				StartTime:       b.cycleStartTime,
				EndTime:         timestamp,
				Entries:         b.cycleEntries,
				AvgEntry:        avgEntry,          // Net average entry
				AvgGrossEntry:   avgGrossEntry,     // Gross average entry
				TargetPrice:     target,
				RealizedPnL:     realized,
				TotalCost:       b.cycleCostSum,        // Net cost
				TotalGrossCost:  b.cycleGrossCostSum,   // Gross cost
				TotalCommission: b.cycleCommissionSum,  // Commission
				Completed:       true,
			})
			b.results.CompletedCycles++

			// Reset position and cycle state for next DCA cycle
			b.resetCycle()
		}
	}
}

// checkAndExecuteMultipleTP handles 5-level TP logic
func (b *BacktestEngine) checkAndExecuteMultipleTP(currentPrice float64, timestamp time.Time) {
	if b.cycleRemainingQty <= 0 {
		return
	}
	
	// Calculate current average entry price
	avgEntry := b.calculateCurrentAvgEntry()
	
	// Check each TP level
	for i, tpLevel := range b.tpLevels {
		if tpLevel.Hit {
			continue // Skip already hit levels
		}
		
		target := avgEntry * (1.0 + tpLevel.Percent)
		if currentPrice >= target {
			b.executeTPLevel(i, currentPrice, timestamp, avgEntry)
		}
	}
	
	// Check if cycle is complete (all TPs hit)
	if b.isCycleComplete() {
		b.completeCycle(timestamp)
	}
}

func (b *BacktestEngine) executeTPLevel(levelIndex int, currentPrice float64, timestamp time.Time, avgEntry float64) {
    tpLevel := &b.tpLevels[levelIndex]
    
    // Sell fixed absolute quantity defined at TP initialization/reset
    sellQty := tpLevel.Quantity
    if sellQty > b.cycleRemainingQty {
        sellQty = b.cycleRemainingQty
    }
    
    // Execute partial exit
    proceeds := sellQty * currentPrice
    commission := proceeds * b.commission
    pnl := (currentPrice - avgEntry) * sellQty - commission
    
    // Update TP level status
    tpLevel.Hit = true
    tpLevel.HitTime = &timestamp
    tpLevel.HitPrice = currentPrice
    tpLevel.PnL = pnl
    tpLevel.SoldQty = sellQty
    tpLevel.SellCommission = commission
    
    // Update cycle state
    b.cycleRemainingQty -= sellQty
    b.cycleTPProgress[levelIndex] = true
    b.cycleUnrealizedPnL += pnl  // Add PnL to cycle's unrealized PnL
    
    // Update position and balance
    b.position -= sellQty
    b.balance += proceeds - commission
    
    // Create synthetic trade for this partial exit using the provided avgEntry
    b.updateTradeExitsForTPLevel(sellQty, currentPrice, timestamp, commission, avgEntry)
}

// updateTradeExitsForTPLevel creates synthetic trades for TP level partial exits
func (b *BacktestEngine) updateTradeExitsForTPLevel(sellQty float64, currentPrice float64, timestamp time.Time, totalCommission float64, avgEntry float64) {
    // For TP levels, we create a synthetic "exit trade" that represents the partial exit
    // This is cleaner than trying to modify existing entry trades
    
    // Create a synthetic trade representing this partial exit
    partialExitTrade := Trade{
        EntryTime:  timestamp, // Use TP hit time as "entry" for the exit trade
        ExitTime:   timestamp, // Same time for exit
        EntryPrice: avgEntry,  // Use average entry as the "entry" price
        ExitPrice:  currentPrice, // Current price as exit
        Quantity:   sellQty,   // Quantity sold at this TP level
        PnL:        (currentPrice - avgEntry) * sellQty - totalCommission, // PnL for this partial exit
        Commission: totalCommission, // Commission for this partial exit
        Cycle:      b.currentCycleNumber, // Current cycle
    }
    
    // Add this synthetic trade to represent the partial exit
    b.results.Trades = append(b.results.Trades, partialExitTrade)
}

// calculateCurrentAvgEntry calculates the current weighted average entry price
func (b *BacktestEngine) calculateCurrentAvgEntry() float64 {
    // Use net cost basis across the cycle to avoid dependency on trade mutation
    if b.cycleQtySum > 0 {
        return b.cycleCostSum / b.cycleQtySum
    }
    return 0
}

// isCycleComplete checks if all remaining quantity has been sold via TPs
func (b *BacktestEngine) isCycleComplete() bool {
	if !b.useTPLevels {
		return false
	}
	
	// Instead of checking Hit status (which gets reset), check if remaining quantity is near zero
	// This accounts for TP resets during DCA entries within the same cycle
	const tolerance = 1e-8  // Small tolerance for floating point precision
	return b.cycleRemainingQty <= tolerance
}

// completeCycle finalizes the cycle when all TPs are hit
func (b *BacktestEngine) completeCycle(timestamp time.Time) {
    // Create cycle summary with partial exits
    partialExits := make([]PartialExit, 0)
    totalRealizedPnL := 0.0
    
    for i, tp := range b.tpLevels {
        if tp.Hit {
            partialExits = append(partialExits, PartialExit{
                TPLevel:    i + 1,
                Quantity:   tp.SoldQty,
                Price:      tp.HitPrice,
                Timestamp:  *tp.HitTime,
                PnL:        tp.PnL,
                Commission: tp.SellCommission,
            })
            totalRealizedPnL += tp.PnL
        }
    }
    
    avgEntry := 0.0
    if b.cycleQtySum > 0 { avgEntry = b.cycleCostSum / b.cycleQtySum }
    avgGrossEntry := 0.0
    if b.cycleGrossQtySum > 0 { avgGrossEntry = b.cycleGrossCostSum / b.cycleGrossQtySum }
    
    // Determine final exit price safely
    finalExitPrice := 0.0
    if len(partialExits) > 0 {
        finalExitPrice = partialExits[len(partialExits)-1].Price
    }

    b.results.Cycles = append(b.results.Cycles, CycleSummary{
        CycleNumber:       b.currentCycleNumber,
        StartTime:         b.cycleStartTime,
        EndTime:           timestamp,
        Entries:           b.cycleEntries,
        AvgEntry:          avgEntry,
        AvgGrossEntry:     avgGrossEntry,
        TargetPrice:       0, // Not applicable for multiple TPs
        RealizedPnL:       b.cycleUnrealizedPnL, // Use accumulated unrealized PnL
        TotalCost:         b.cycleCostSum,
        TotalGrossCost:    b.cycleGrossCostSum,
        TotalCommission:   b.cycleCommissionSum,
        Completed:         true,
        TPLevelsHit:       len(partialExits),
        PartialExits:      partialExits,
        FinalExitPrice:    finalExitPrice,
        TotalRealizedPnL:  b.cycleUnrealizedPnL, // Use accumulated unrealized PnL
    })
	
	b.results.CompletedCycles++
	
	// Reset cycle state
	b.resetCycle()
}

// resetTPLevelsForNewEntry resets TP levels when new DCA entry is added to existing cycle
func (b *BacktestEngine) resetTPLevelsForNewEntry() {
    if !b.useTPLevels {
        return
    }
    
    // NOTE: We do NOT reverse position/balance changes from previous TP hits
    // The profits remain as unrealized PnL and the partial exits stay in effect
    // The synthetic trades for partial exits remain in the trades list
    // We only reset the TP level status to start checking from TP1 again
    
    // Reset TP level status only (keep the profits as unrealized PnL)
    for i := range b.tpLevels {
        // Reset TP level state to allow hitting them again
        b.cycleTPProgress[i] = false
        b.tpLevels[i].Hit = false
        b.tpLevels[i].HitTime = nil
        b.tpLevels[i].HitPrice = 0
        b.tpLevels[i].PnL = 0
        b.tpLevels[i].SoldQty = 0
        b.tpLevels[i].SellCommission = 0
    }
    
    // Do NOT restore position or balance - the partial exits remain in effect
    // Do NOT change cycleRemainingQty - it reflects the actual remaining position
    // The cycleUnrealizedPnL already contains the profits from previous TP hits
    // The synthetic trades for partial exits remain in the trades list for accurate reporting

    // Recompute absolute TP quantities based on the current remaining quantity
    b.setTPLevelsQuantities(b.cycleRemainingQty)
}

// setTPLevelsQuantities sets each TP level's absolute quantity to 20% of baseQty
func (b *BacktestEngine) setTPLevelsQuantities(baseQty float64) {
    if !b.useTPLevels {
        return
    }
    // Guard against negatives
    if baseQty < 0 {
        baseQty = 0
    }
    for i := range b.tpLevels {
        b.tpLevels[i].Quantity = 0.20 * baseQty
    }
}

// resetCycle resets the cycle state
func (b *BacktestEngine) resetCycle() {
    b.cycleOpen = false
    b.cycleEntries = 0
    b.cycleQtySum = 0
    b.cycleCostSum = 0
    b.cycleGrossCostSum = 0
    b.cycleGrossQtySum = 0
    b.cycleCommissionSum = 0
    b.cycleRemainingQty = 0
    b.cycleUnrealizedPnL = 0
    
    // Reset cycle exposure tracking
    b.maxCycleExposure = 0
    
    // Notify strategy that cycle is complete so it can reset state
    b.strategy.OnCycleComplete()
}
