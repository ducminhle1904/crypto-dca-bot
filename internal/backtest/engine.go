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
	PnL         float64    // PnL for this level
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
	cycleCommissionSum   float64 // sum of commission paid in current cycle
	
	// Enhanced cycle tracking for multiple TPs
	cycleTPProgress    map[int]bool  // Track which TP levels are hit
	cycleRemainingQty  float64       // Remaining quantity after partial exits
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

func NewBacktestEngine(
	initialBalance float64,
	commission float64,
	strat strategy.Strategy,
	// tpPercent: 0 disables take-profit. Example: 0.02 = 2% above avg entry
	tpPercent float64,
	// minOrderQty: minimum order quantity (e.g., 0.01 for BTCUSDT). 0 disables lot size constraints
	minOrderQty float64,
	// useTPLevels: enable 5-level TP mode
	useTPLevels bool,
	// tpLevels: 5 TP levels configuration
	tpLevels []TPLevel,
) *BacktestEngine {
	engine := &BacktestEngine{
		initialBalance: initialBalance,
		commission:     commission,
		strategy:       strat,
		results: &BacktestResults{
			StartBalance: initialBalance,
			Trades:       make([]Trade, 0),
			Cycles:       make([]CycleSummary, 0),
		},
		tpPercent:   tpPercent,
		useTPLevels: useTPLevels,
		tpLevels:    tpLevels,
		minOrderQty: minOrderQty,
		balance:     initialBalance,
	}
	
	// Initialize TP level tracking
	if useTPLevels {
		engine.cycleTPProgress = make(map[int]bool)
		for i := range tpLevels {
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
	position := 0.0
	maxBalance := b.balance

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

			// Calculate commission on original target amount (what strategy intended to invest)
			commission := targetAmount * b.commission
			totalCost := actualAmount + commission // Adjusted amount + commission on original
			
			// Check if we have enough balance for total cost (adjusted amount + commission)
			if b.balance >= totalCost {
				// Execute buy with actual amount, but deduct commission separately
				netAmount := actualAmount // The lot-adjusted amount becomes the net investment
				actualQuantity := netAmount / currentPrice

				position += actualQuantity
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
					b.cycleCommissionSum = 0
					b.cycleRemainingQty = 0
					
					// Reset TP level progress for new cycle
					if b.useTPLevels {
						for i := range b.tpLevels {
							b.cycleTPProgress[i] = false
							b.tpLevels[i].Hit = false
							b.tpLevels[i].HitTime = nil
							b.tpLevels[i].HitPrice = 0
							b.tpLevels[i].PnL = 0
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
					// Track commission for this cycle
					b.cycleCommissionSum += commission
					trade.Cycle = b.currentCycleNumber
				}

				b.results.Trades = append(b.results.Trades, trade)
			}
		}

		// Check and execute take profit orders
		if b.cycleOpen && position > 0 {
			if b.useTPLevels {
				b.checkAndExecuteMultipleTP(currentPrice, data[i].Timestamp)
			} else if b.tpPercent > 0 {
				b.checkAndExecuteSingleTP(currentPrice, data[i].Timestamp)
			}
		}

		//Updating metrics (equity tracking)
		currentValue := b.balance + (position * currentPrice)
		if currentValue > maxBalance {
			maxBalance = currentValue
		}

		//Calculating the drawdown
		drawdown := (maxBalance - currentValue) / maxBalance
		if drawdown > b.results.MaxDrawdown {
			b.results.MaxDrawdown = drawdown
		}
	}

	//Final calculations
	finalPrice := data[len(data)-1].Close
	finalTime := data[len(data)-1].Timestamp

	// For any remaining OPEN trades, compute unrealized PnL at final price
	for i := range b.results.Trades {
		trade := &b.results.Trades[i]
		if trade.ExitTime.IsZero() {
			trade.ExitTime = finalTime
			trade.ExitPrice = finalPrice
			trade.PnL = (finalPrice-trade.EntryPrice)*trade.Quantity - trade.Commission
		}
	}

	// If a cycle is still open at the end, record it as incomplete
	if b.tpPercent > 0 && b.cycleOpen && b.cycleQtySum > 0 {
		avgEntry := b.cycleCostSum / b.cycleQtySum
		target := avgEntry * (1.0 + b.tpPercent)
		// Sum PnL for trades in current cycle (already set to final price above)
		realized := 0.0
		for _, t := range b.results.Trades {
			if t.Cycle == b.currentCycleNumber {
				realized += t.PnL
			}
		}
		avgGrossEntry := b.cycleGrossCostSum / (b.cycleGrossCostSum / finalPrice) // gross cost / gross qty
		b.results.Cycles = append(b.results.Cycles, CycleSummary{
			CycleNumber:     b.currentCycleNumber,
			StartTime:       b.cycleStartTime,
			EndTime:         finalTime,
			Entries:         b.cycleEntries,
			AvgEntry:        avgEntry,          // Net average entry
			AvgGrossEntry:   avgGrossEntry,     // Gross average entry
			TargetPrice:     target,
			RealizedPnL:     realized,
			TotalCost:       b.cycleCostSum,        // Net cost
			TotalGrossCost:  b.cycleGrossCostSum,   // Gross cost
			TotalCommission: b.cycleCommissionSum,  // Commission
			Completed:       false,
		})
		// Keep CompletedCycles unchanged
	}

	// Set final results
	b.results.EndBalance = b.balance
	b.results.TotalReturn = (b.balance - b.initialBalance) / b.initialBalance
	b.results.TotalTrades = len(b.results.Trades)

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
			avgGrossEntry := b.cycleGrossCostSum / (b.cycleGrossCostSum / currentPrice) // gross cost / gross qty
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

// executeTPLevel executes a single TP level
func (b *BacktestEngine) executeTPLevel(levelIndex int, currentPrice float64, timestamp time.Time, avgEntry float64) {
	tpLevel := &b.tpLevels[levelIndex]
	
	// Calculate quantity to sell (20% of original position)
	sellQty := b.cycleQtySum * 0.20 // DefaultTPQuantity
	
	// Ensure we don't sell more than remaining
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
	
	// Update cycle state
	b.cycleRemainingQty -= sellQty
	b.cycleTPProgress[levelIndex] = true
	
	b.balance += proceeds - commission
	
	// Record partial exit in trades
	for idx := range b.results.Trades {
		if b.results.Trades[idx].ExitTime.IsZero() && b.results.Trades[idx].Cycle == b.currentCycleNumber {
			// Mark this trade as partially exited
			b.results.Trades[idx].ExitTime = timestamp
			b.results.Trades[idx].ExitPrice = currentPrice
			b.results.Trades[idx].PnL = (currentPrice - b.results.Trades[idx].EntryPrice) * b.results.Trades[idx].Quantity - b.results.Trades[idx].Commission
			break // Only mark one trade per TP level for now
		}
	}
}

// calculateCurrentAvgEntry calculates the current weighted average entry price
func (b *BacktestEngine) calculateCurrentAvgEntry() float64 {
	totalQty := 0.0
	sumEntryCost := 0.0
	for _, t := range b.results.Trades {
		if t.ExitTime.IsZero() && t.Cycle == b.currentCycleNumber {
			totalQty += t.Quantity
			sumEntryCost += t.EntryPrice * t.Quantity
		}
	}
	if totalQty > 0 {
		return sumEntryCost / totalQty
	}
	return 0
}

// isCycleComplete checks if all TP levels are hit
func (b *BacktestEngine) isCycleComplete() bool {
	if !b.useTPLevels {
		return false
	}
	
	// Check if all 5 TP levels are hit
	for i := 0; i < len(b.tpLevels); i++ {
		if !b.tpLevels[i].Hit {
			return false
		}
	}
	return true
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
				Quantity:   b.cycleQtySum * 0.20,
				Price:      tp.HitPrice,
				Timestamp:  *tp.HitTime,
				PnL:        tp.PnL,
				Commission: tp.PnL * b.commission, // Approximate commission
			})
			totalRealizedPnL += tp.PnL
		}
	}
	
	avgEntry := b.calculateCurrentAvgEntry()
	avgGrossEntry := b.cycleGrossCostSum / (b.cycleGrossCostSum / avgEntry)
	
	b.results.Cycles = append(b.results.Cycles, CycleSummary{
		CycleNumber:       b.currentCycleNumber,
		StartTime:         b.cycleStartTime,
		EndTime:           timestamp,
		Entries:           b.cycleEntries,
		AvgEntry:          avgEntry,
		AvgGrossEntry:     avgGrossEntry,
		TargetPrice:       0, // Not applicable for multiple TPs
		RealizedPnL:       totalRealizedPnL,
		TotalCost:         b.cycleCostSum,
		TotalGrossCost:    b.cycleGrossCostSum,
		TotalCommission:   b.cycleCommissionSum,
		Completed:         true,
		TPLevelsHit:       len(partialExits),
		PartialExits:      partialExits,
		FinalExitPrice:    partialExits[len(partialExits)-1].Price,
		TotalRealizedPnL:  totalRealizedPnL,
	})
	
	b.results.CompletedCycles++
	
	// Reset cycle state
	b.resetCycle()
}

// resetCycle resets the cycle state
func (b *BacktestEngine) resetCycle() {
	b.cycleOpen = false
	b.cycleEntries = 0
	b.cycleQtySum = 0
	b.cycleCostSum = 0
	b.cycleGrossCostSum = 0
	b.cycleCommissionSum = 0
	b.cycleRemainingQty = 0
	
	// Notify strategy that cycle is complete so it can reset state
	b.strategy.OnCycleComplete()
}
