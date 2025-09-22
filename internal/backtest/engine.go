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

// DynamicTPRecord tracks dynamic TP calculation data for analysis
type DynamicTPRecord struct {
	Timestamp        time.Time // When TP was calculated
	Price            float64   // Market price at calculation
	BaseTPPercent    float64   // Base TP percentage
	CalculatedTP     float64   // Calculated dynamic TP percentage
	Strategy         string    // TP strategy used ("volatility_adaptive", "indicator_based")
	MarketVolatility float64   // ATR/Price ratio at time of calculation
	SignalStrength   float64   // Average signal strength from indicators
	BoundsApplied    bool      // Whether min/max bounds were applied
}

// DynamicTPMetrics contains performance analysis for dynamic TP
type DynamicTPMetrics struct {
	Enabled               bool    // Whether dynamic TP was used
	Strategy              string  // Primary TP strategy used
	AvgTPPercent          float64 // Average TP percentage used across all trades
	TPRangeUtilization    float64 // Percentage of min/max range utilized (0-1)
	VolatilityTPCorrelation float64 // Correlation between market volatility and TP targets
	DynamicTPHitRate      float64 // Hit rate for dynamic TP vs fixed TP baseline
	MinTPUsed             float64 // Minimum TP percentage used
	MaxTPUsed             float64 // Maximum TP percentage used
	BoundsHitCount        int     // Number of times min/max bounds were applied
	TotalCalculations     int     // Total number of dynamic TP calculations
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

	// Dynamic TP configuration
	dynamicTPEnabled bool     // Enable dynamic TP calculation
	dynamicTPHistory []DynamicTPRecord // Historical dynamic TP data for analysis

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
	// Memory-efficient exposure tracking
	exposureSamples    int           // Number of exposure samples taken
	exposureSum        float64       // Sum of all exposure values for average calculation
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
	
	// Dynamic TP metrics
	DynamicTPMetrics  *DynamicTPMetrics // Dynamic TP performance analysis
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
	
	// Dynamic TP tracking fields
	TPTarget         float64 // Calculated TP target for this trade
	TPStrategy       string  // TP strategy used ("fixed", "volatility_adaptive", "indicator_based")
	MarketVolatility float64 // ATR/Price ratio at time of entry
	SignalStrength   float64 // Average indicator strength at entry
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
			DynamicTPMetrics: &DynamicTPMetrics{
				Enabled: strat.IsDynamicTPEnabled(),
			},
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
		exposureSamples:  0,
		exposureSum:      0,
		// Initialize dynamic TP tracking
		dynamicTPEnabled: strat.IsDynamicTPEnabled(),
		dynamicTPHistory: make([]DynamicTPRecord, 0),
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
	b.exposureSamples = 0
	b.exposureSum = 0
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
				
				// Add dynamic TP tracking for the trade
				if b.dynamicTPEnabled && decision != nil {
					// Calculate what the TP target would be for this trade
					historyData := data[:i+1]
					avgEntryEstimate := currentPrice // For new trades, average entry is current price
					_, dynamicRecord, err := b.calculateCurrentTPTarget(data[i], historyData, avgEntryEstimate)
					if err == nil && dynamicRecord != nil {
						trade.TPTarget = dynamicRecord.CalculatedTP
						trade.TPStrategy = dynamicRecord.Strategy
						trade.MarketVolatility = dynamicRecord.MarketVolatility
						trade.SignalStrength = decision.Strength
					}
				} else {
					// Fixed TP mode
					trade.TPTarget = b.tpPercent
					trade.TPStrategy = "fixed"
					trade.MarketVolatility = 0
					trade.SignalStrength = 0
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

		// Check and execute take profit orders using High price for realistic TP execution
		if b.cycleOpen && b.position > 0 {
			if b.useTPLevels {
				// Use High price to check which TP levels were hit during the candle
				b.checkAndExecuteMultipleTPWithHigh(data[i].High, data[i].Timestamp, data, i)
			} else if b.tpPercent > 0 || b.dynamicTPEnabled {
				// For single TP (fixed or dynamic), use High price to check if target was reached
				b.checkAndExecuteSingleTPWithHigh(data[i].High, data[i].Timestamp, data, i)
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
		
		// Update exposure tracking with memory-efficient running average
		b.currentExposure = exposure
		b.exposureSamples++
		b.exposureSum += exposure
		
		// Track cycle exposure
		if b.cycleOpen && exposure > b.maxCycleExposure {
			b.maxCycleExposure = exposure
		}
		
		runningPnL := currentValue - b.initialBalance
		
		// Sample equity curve every 100 data points to reduce memory usage for large datasets
		// For smaller datasets, keep more points for precision
		sampleInterval := 1
		if len(data) > 10000 {
			sampleInterval = len(data) / 5000 // Keep ~5000 points maximum
		}
		
		if i%sampleInterval == 0 || i == len(data)-1 { // Always keep the last point
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
	if b.exposureSamples > 0 {
		b.results.AvgCycleExposure = b.exposureSum / float64(b.exposureSamples)
	}

	// Finalize dynamic TP metrics
	b.finalizeDynamicTPMetrics()

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
// checkAndExecuteSingleTPWithHigh handles single TP logic using High price
func (b *BacktestEngine) checkAndExecuteSingleTPWithHigh(highPrice float64, timestamp time.Time, data []types.OHLCV, currentIndex int) {
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
		
		// Calculate TP target using dynamic TP if enabled
		currentCandle := data[currentIndex]
		// Use data up to current index for proper dynamic TP calculation
		historyData := data[:currentIndex+1]
		target, dynamicRecord, err := b.calculateCurrentTPTarget(currentCandle, historyData, avgEntry)
		if err != nil {
			// Log error but continue with calculated target (fallback already applied)
			// In production, you might want to log this error
		}
		
		// Add dynamic TP record if available
		if dynamicRecord != nil {
			b.addDynamicTPRecord(dynamicRecord)
		}
		
		if highPrice >= target {
			// Execute at target price, not current price for realistic simulation
			exitPrice := target
			
			// Realize PnL: sell all open quantity at target price
			proceeds := totalQty * exitPrice
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
					b.results.Trades[idx].ExitPrice = exitPrice
					pnl := (exitPrice-b.results.Trades[idx].EntryPrice)*q - b.results.Trades[idx].Commission - perTradeSellComm
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

// checkAndExecuteMultipleTPWithHigh executes multi-level take profit strategy with 5 progressive levels.
// Each level takes 20% of the position at incrementally higher prices (20%, 40%, 60%, 80%, 100% of base TP).
// Supports both fixed and dynamic TP base percentages, with realistic execution using High price.
func (b *BacktestEngine) checkAndExecuteMultipleTPWithHigh(highPrice float64, timestamp time.Time, data []types.OHLCV, currentIndex int) {
	if b.cycleRemainingQty <= 0 {
		return
	}
	
	// Calculate weighted average entry price for TP target calculation
	avgEntry := b.calculateCurrentAvgEntry()
	
	// Determine base TP percentage (fixed or dynamic based on market conditions)
	var baseTPPercent float64
	var dynamicRecord *DynamicTPRecord
	if b.dynamicTPEnabled && data != nil && currentIndex > 0 {
		currentCandle := data[currentIndex]
		historyData := data[:currentIndex+1]
		_, dynamicRecord, err := b.calculateCurrentTPTarget(currentCandle, historyData, avgEntry)
		if err != nil {
			baseTPPercent = b.tpPercent // Fallback to fixed TP on calculation error
		} else if dynamicRecord != nil {
			baseTPPercent = dynamicRecord.CalculatedTP
			b.addDynamicTPRecord(dynamicRecord) // Track for performance analysis
		} else {
			baseTPPercent = b.tpPercent
		}
	} else {
		baseTPPercent = b.tpPercent // Use configured fixed TP percentage
	}
	
	// Check each TP level sequentially using High price to determine if reached
	for i, tpLevel := range b.tpLevels {
		if tpLevel.Hit {
			continue // Skip already hit levels
		}
		
		// Calculate TP target for this level using base TP percentage
		// Level multipliers: 40%, 60%, 80%, 100%, 120% of base TP (more aggressive progression)
		// This ensures first level is achievable while maintaining reasonable spread
		levelMultiplier := 0.4 + float64(i)*0.2 // 0.4, 0.6, 0.8, 1.0, 1.2
		levelTPPercent := baseTPPercent * levelMultiplier
		target := avgEntry * (1.0 + levelTPPercent)
		
		if highPrice >= target {
			// Execute at exact target price for realistic simulation
			// Pass dynamic TP info for tracking
			b.executeTPLevelWithDynamicInfo(i, target, timestamp, avgEntry, levelTPPercent, dynamicRecord)
		}
	}
	
	// Check if cycle is complete (all TPs hit)
	if b.isCycleComplete() {
		b.completeCycle(timestamp)
	}
}

// Legacy function for backward compatibility (now uses High price internally)
func (b *BacktestEngine) checkAndExecuteMultipleTP(currentPrice float64, timestamp time.Time) {
	// This is kept for compatibility but should not be used in new code
	// Use checkAndExecuteMultipleTPWithHigh instead
	// Note: This legacy method doesn't support dynamic TP due to missing data context
	b.checkAndExecuteMultipleTPWithHigh(currentPrice, timestamp, nil, 0)
}

// executeTPLevelWithDynamicInfo executes a TP level with dynamic TP information tracking
func (b *BacktestEngine) executeTPLevelWithDynamicInfo(levelIndex int, currentPrice float64, timestamp time.Time, avgEntry float64, levelTPPercent float64, dynamicRecord *DynamicTPRecord) {
    tpLevel := &b.tpLevels[levelIndex]
    
    // Sell fixed absolute quantity defined at TP initialization/reset
    sellQty := tpLevel.Quantity
    if sellQty > b.cycleRemainingQty {
        sellQty = b.cycleRemainingQty
    }
    
    // Execute partial exit
    proceeds := sellQty * currentPrice
    commission := proceeds * b.commission
    
    // Calculate proportional cost for the quantity being sold
    proportionalCost := 0.0
    if b.cycleGrossQtySum > 0 {
        proportionalCost = (b.cycleCostSum / b.cycleGrossQtySum) * sellQty
    }
    
    // PnL = (exit proceeds - commission) - proportional entry cost
    pnl := (proceeds - commission) - proportionalCost
    
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
    
    // Create synthetic trade for this partial exit with dynamic TP info
    b.updateTradeExitsForTPLevelWithDynamicInfo(sellQty, currentPrice, timestamp, commission, avgEntry, levelTPPercent, dynamicRecord)
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
    
    // Calculate proportional cost for the quantity being sold
    // Use the actual cost basis (total cost / total quantity) instead of simple average entry
    proportionalCost := 0.0
    if b.cycleGrossQtySum > 0 {
        proportionalCost = (b.cycleCostSum / b.cycleGrossQtySum) * sellQty
    }
    
    // PnL = (exit proceeds - commission) - proportional entry cost
    pnl := (proceeds - commission) - proportionalCost
    
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
    // Use the enhanced function with no dynamic TP info for backward compatibility
    b.updateTradeExitsForTPLevelWithDynamicInfo(sellQty, currentPrice, timestamp, commission, avgEntry, b.tpPercent, nil)
}

// updateTradeExitsForTPLevelWithDynamicInfo creates synthetic trades for TP level partial exits with dynamic TP info
func (b *BacktestEngine) updateTradeExitsForTPLevelWithDynamicInfo(sellQty float64, currentPrice float64, timestamp time.Time, totalCommission float64, avgEntry float64, levelTPPercent float64, dynamicRecord *DynamicTPRecord) {
    // Calculate proportional cost for consistency
    proportionalCost := 0.0
    if b.cycleGrossQtySum > 0 {
        proportionalCost = (b.cycleCostSum / b.cycleGrossQtySum) * sellQty
    }
    
    // Calculate PnL
    proceeds := sellQty * currentPrice
    pnl := (proceeds - totalCommission) - proportionalCost
    
    // Create a synthetic trade representing this partial exit with dynamic TP info
    partialExitTrade := Trade{
        EntryTime:  timestamp, // Use TP hit time as "entry" for the exit trade
        ExitTime:   timestamp, // Same time for exit
        EntryPrice: avgEntry,  // Use average entry as the "entry" price (for display purposes)
        ExitPrice:  currentPrice, // Current price as exit
        Quantity:   sellQty,   // Quantity sold at this TP level
        PnL:        pnl,       // PnL calculated using proportional cost basis
        Commission: totalCommission, // Commission for this partial exit
        Cycle:      b.currentCycleNumber, // Current cycle
        
        // Dynamic TP information
        TPTarget:        levelTPPercent, // The specific level TP percentage used
        TPStrategy:      "fixed", // Default fallback
        MarketVolatility: 0,
        SignalStrength:   0,
    }
    
    // Fill in dynamic TP details if available
    if dynamicRecord != nil {
        partialExitTrade.TPStrategy = dynamicRecord.Strategy
        partialExitTrade.MarketVolatility = dynamicRecord.MarketVolatility
        partialExitTrade.SignalStrength = dynamicRecord.SignalStrength
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

// resetTPLevelsForNewEntry recalculates take profit levels when a new DCA entry is added.
// This enables dynamic TP recalculation: when market conditions change and new DCA entries occur,
// TP targets are recalculated based on the new average entry price and current market conditions.
// 
// Key behaviors:
// - Preserves all previous TP profits (they remain as unrealized PnL)
// - Maintains partial position exits that already occurred
// - Recalculates TP targets from level 1 based on new average price
// - Adjusts TP quantities based on remaining position size
func (b *BacktestEngine) resetTPLevelsForNewEntry() {
    if !b.useTPLevels {
        return
    }
    
    // Reset TP level tracking to enable fresh calculation from level 1
    // Previous TP profits are preserved in cycleUnrealizedPnL
    for i := range b.tpLevels {
        b.cycleTPProgress[i] = false
        b.tpLevels[i].Hit = false
        b.tpLevels[i].HitTime = nil
        b.tpLevels[i].HitPrice = 0
        b.tpLevels[i].PnL = 0
        b.tpLevels[i].SoldQty = 0
        b.tpLevels[i].SellCommission = 0
    }

    // Recalculate TP quantities based on current remaining position
    // This ensures proper 20% allocation per level of the remaining position
    b.setTPLevelsQuantities(b.cycleRemainingQty)
}

// setTPLevelsQuantities sets each TP level's absolute quantity to 20% of baseQty
// with validation to prevent over-selling
func (b *BacktestEngine) setTPLevelsQuantities(baseQty float64) {
    if !b.useTPLevels {
        return
    }
    // Guard against negatives
    if baseQty < 0 {
        baseQty = 0
    }
    
    // Calculate individual level quantity (20% each)
    levelQty := 0.20 * baseQty
    
    // Ensure total TP quantities don't exceed available position
    totalTPQty := levelQty * 5.0 // 5 levels × 20% each = 100%
    if totalTPQty > baseQty {
        // Adjust to prevent over-selling: reduce each level proportionally
        levelQty = baseQty / 5.0
    }
    
    // Set quantities for each level
    for i := range b.tpLevels {
        b.tpLevels[i].Quantity = levelQty
    }
    
    // Additional safety check: verify total doesn't exceed position
    totalAllocated := 0.0
    for i := range b.tpLevels {
        totalAllocated += b.tpLevels[i].Quantity
    }
    
    if totalAllocated > baseQty {
        // Emergency fix: scale down all quantities proportionally
        scaleFactor := baseQty / totalAllocated
        for i := range b.tpLevels {
            b.tpLevels[i].Quantity *= scaleFactor
        }
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

// calculateCurrentTPTarget calculates the TP target using dynamic TP if enabled
func (b *BacktestEngine) calculateCurrentTPTarget(currentCandle types.OHLCV, data []types.OHLCV, avgEntry float64) (float64, *DynamicTPRecord, error) {
	if !b.dynamicTPEnabled {
		// Use fixed TP
		target := avgEntry * (1.0 + b.tpPercent)
		return target, nil, nil
	}

	// Calculate dynamic TP percentage
	dynamicTPPercent, err := b.strategy.GetDynamicTPPercent(currentCandle, data)
	if err != nil {
		// Fallback to fixed TP on error
		target := avgEntry * (1.0 + b.tpPercent)
		return target, nil, fmt.Errorf("dynamic TP calculation failed, using fixed TP: %w", err)
	}

	// If dynamic TP returns 0, use fixed TP
	if dynamicTPPercent == 0 {
		target := avgEntry * (1.0 + b.tpPercent)
		return target, nil, nil
	}

	// Calculate target price using dynamic TP
	target := avgEntry * (1.0 + dynamicTPPercent)

	// Create dynamic TP record for analysis
	record := &DynamicTPRecord{
		Timestamp:        currentCandle.Timestamp,
		Price:            currentCandle.Close,
		BaseTPPercent:    b.tpPercent,
		CalculatedTP:     dynamicTPPercent,
		Strategy:         "unknown", // Will be set by caller based on strategy config
		MarketVolatility: 0,         // Will be calculated by caller
		SignalStrength:   0,         // Will be calculated by caller
		BoundsApplied:    false,     // Will be set by caller
	}

	return target, record, nil
}

// addDynamicTPRecord adds a dynamic TP record to the history and updates metrics
func (b *BacktestEngine) addDynamicTPRecord(record *DynamicTPRecord) {
	if record == nil {
		return
	}

	// Add to history
	b.dynamicTPHistory = append(b.dynamicTPHistory, *record)

	// Update metrics
	metrics := b.results.DynamicTPMetrics
	metrics.TotalCalculations++

	// Update min/max TP used
	if metrics.TotalCalculations == 1 {
		metrics.MinTPUsed = record.CalculatedTP
		metrics.MaxTPUsed = record.CalculatedTP
	} else {
		if record.CalculatedTP < metrics.MinTPUsed {
			metrics.MinTPUsed = record.CalculatedTP
		}
		if record.CalculatedTP > metrics.MaxTPUsed {
			metrics.MaxTPUsed = record.CalculatedTP
		}
	}

	// Track bounds application
	if record.BoundsApplied {
		metrics.BoundsHitCount++
	}

	// Update strategy if not set
	if metrics.Strategy == "" {
		metrics.Strategy = record.Strategy
	}
}

// finalizeDynamicTPMetrics calculates final dynamic TP metrics
func (b *BacktestEngine) finalizeDynamicTPMetrics() {
	metrics := b.results.DynamicTPMetrics
	if !metrics.Enabled || len(b.dynamicTPHistory) == 0 {
		return
	}

	// Calculate average TP percent
	totalTP := 0.0
	totalVolatility := 0.0
	validVolatilityCount := 0

	for _, record := range b.dynamicTPHistory {
		totalTP += record.CalculatedTP
		if record.MarketVolatility > 0 {
			totalVolatility += record.MarketVolatility
			validVolatilityCount++
		}
	}

	metrics.AvgTPPercent = totalTP / float64(len(b.dynamicTPHistory))

	// Calculate TP range utilization (requires knowing min/max bounds from strategy)
	if metrics.MaxTPUsed > metrics.MinTPUsed {
		// This is a simplified calculation - in practice, we'd need to know the actual configured bounds
		metrics.TPRangeUtilization = (metrics.MaxTPUsed - metrics.MinTPUsed) / metrics.MaxTPUsed
	}

	// Calculate volatility-TP correlation (simplified)
	if validVolatilityCount > 1 {
		// This would need a proper correlation calculation
		// For now, we'll set a placeholder value
		metrics.VolatilityTPCorrelation = 0.0
	}

	// Calculate dynamic TP hit rate (would need comparison with fixed TP baseline)
	// This would require running a parallel simulation with fixed TP
	metrics.DynamicTPHitRate = 0.0 // Placeholder
}
