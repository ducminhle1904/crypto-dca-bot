package backtest

import (
	"fmt"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

type BacktestEngine struct {
	initialBalance float64
	commission     float64
	strategy       strategy.Strategy
	results        *BacktestResults
	// take-profit as a decimal (e.g., 0.01 for 1%). 0 disables TP and behaves like before
	tpPercent      float64

	// cycle tracking (only meaningful when tpPercent > 0)
	cycleOpen          bool
	currentCycleNumber int
	cycleEntries       int
	cycleStartTime     time.Time
	cycleQtySum        float64
	cycleCostSum       float64 // sum(entryPrice * qty)
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
	CycleNumber int
	StartTime   time.Time
	EndTime     time.Time
	Entries     int
	AvgEntry    float64
	TargetPrice float64
	RealizedPnL float64
	Completed   bool // true if closed by TP, false if left open at end
}

func NewBacktestEngine(
	initialBalance float64,
	commission float64,
	strat strategy.Strategy,
	// tpPercent: 0 disables take-profit. Example: 0.02 = 2% above avg entry
	tpPercent float64,
) *BacktestEngine {
	return &BacktestEngine{
		initialBalance: initialBalance,
		commission:     commission,
		strategy:       strat,
		results: &BacktestResults{
			StartBalance: initialBalance,
			Trades:       make([]Trade, 0),
			Cycles:       make([]CycleSummary, 0),
		},
		tpPercent: tpPercent,
	}
}

func (b *BacktestEngine) Run(data []types.OHLCV, windowSize int) *BacktestResults {
	// Handle empty or insufficient data
	if len(data) == 0 {
		b.results.EndBalance = b.initialBalance
		b.results.TotalReturn = 0.0
		b.results.TotalTrades = 0
		return b.results
	}

	balance := b.initialBalance
	position := 0.0
	maxBalance := balance

	for i := windowSize; i < len(data); i++ {
		// get a data window for analysis
		window := data[i-windowSize : i+1]
		currentPrice := data[i].Close

		// get a signal from the strategy
		decision, err := b.strategy.ShouldExecuteTrade(window)
		if err == nil && decision.Action == strategy.ActionBuy && balance > decision.Amount {
			// Buy
			commission := decision.Amount * b.commission
			netAmount := decision.Amount - commission
			quantity := netAmount / currentPrice

			position += quantity
			balance -= decision.Amount

			// begin a new cycle if needed (only when TP enabled)
			if b.tpPercent > 0 && !b.cycleOpen {
				b.currentCycleNumber++
				b.cycleOpen = true
				b.cycleEntries = 0
				b.cycleStartTime = data[i].Timestamp
				b.cycleQtySum = 0
				b.cycleCostSum = 0
			}

			//Recording the transaction (input)
			trade := Trade{
				EntryTime:  data[i].Timestamp,
				EntryPrice: currentPrice,
				Quantity:   quantity,
				Commission: commission,
			}
			if b.cycleOpen {
				b.cycleEntries++
				b.cycleQtySum += quantity
				b.cycleCostSum += currentPrice * quantity
				trade.Cycle = b.currentCycleNumber
			}

			b.results.Trades = append(b.results.Trades, trade)
		}

		// Optional take-profit: if enabled and we have a position, check if price >= avg entry * (1 + tp)
		if b.tpPercent > 0 && b.cycleOpen && position > 0 {
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
					proceeds := position * currentPrice
					sellCommission := proceeds * b.commission
					balance += proceeds - sellCommission

					// Proportionally assign sell commission and finalize open trades for this cycle
					realized := 0.0
					for idx := range b.results.Trades {
						if b.results.Trades[idx].ExitTime.IsZero() && b.results.Trades[idx].Cycle == b.currentCycleNumber {
							q := b.results.Trades[idx].Quantity
							share := 0.0
							if totalQty > 0 { share = q / totalQty }
							perTradeSellComm := sellCommission * share
							b.results.Trades[idx].ExitTime = data[i].Timestamp
							b.results.Trades[idx].ExitPrice = currentPrice
							pnl := (currentPrice-b.results.Trades[idx].EntryPrice)*q - b.results.Trades[idx].Commission - perTradeSellComm
							b.results.Trades[idx].PnL = pnl
							realized += pnl
						}
					}

					// finalize cycle summary
					b.results.Cycles = append(b.results.Cycles, CycleSummary{
						CycleNumber: b.currentCycleNumber,
						StartTime:   b.cycleStartTime,
						EndTime:     data[i].Timestamp,
						Entries:     b.cycleEntries,
						AvgEntry:    avgEntry,
						TargetPrice: target,
						RealizedPnL: realized,
						Completed:   true,
					})
					b.results.CompletedCycles++

					// Reset position and cycle state for next DCA cycle
					position = 0
					b.cycleOpen = false
					b.cycleEntries = 0
					b.cycleQtySum = 0
					b.cycleCostSum = 0
				}
			}
		}

		//Updating metrics (equity tracking)
		currentValue := balance + (position * currentPrice)
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
	finalValue := balance + (position * finalPrice)

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
		b.results.Cycles = append(b.results.Cycles, CycleSummary{
			CycleNumber: b.currentCycleNumber,
			StartTime:   b.cycleStartTime,
			EndTime:     finalTime,
			Entries:     b.cycleEntries,
			AvgEntry:    avgEntry,
			TargetPrice: target,
			RealizedPnL: realized,
			Completed:   false,
		})
		// Keep CompletedCycles unchanged
	}

	b.results.EndBalance = finalValue
	b.results.TotalReturn = (finalValue - b.initialBalance) / b.initialBalance
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
	}
}
