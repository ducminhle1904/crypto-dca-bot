package backtest

import (
	"fmt"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/internal/strategy"
	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

type BacktestEngine struct {
	initialBalance float64
	commission     float64
	strategy       strategy.Strategy
	results        *BacktestResults
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
}

type Trade struct {
	EntryTime  time.Time
	ExitTime   time.Time
	EntryPrice float64
	ExitPrice  float64
	Quantity   float64
	PnL        float64
	Commission float64
}

func NewBacktestEngine(
	initialBalance float64,
	commission float64,
	strat strategy.Strategy,
) *BacktestEngine {
	return &BacktestEngine{
		initialBalance: initialBalance,
		commission:     commission,
		strategy:       strat,
		results: &BacktestResults{
			StartBalance: initialBalance,
			Trades:       make([]Trade, 0),
		},
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
		if err != nil || decision.Action == strategy.ActionHold {
			continue
		}

		if decision.Action == strategy.ActionBuy && balance > decision.Amount {
			// Buy
			commission := decision.Amount * b.commission
			netAmount := decision.Amount - commission
			quantity := netAmount / currentPrice

			position += quantity
			balance -= decision.Amount

			//Recording the transaction (input)
			trade := Trade{
				EntryTime:  data[i].Timestamp,
				EntryPrice: currentPrice,
				Quantity:   quantity,
				Commission: commission,
			}

			b.results.Trades = append(b.results.Trades, trade)
		}

		//Updating metrics
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
	finalValue := balance + (position * finalPrice)

	// Compute unrealized PnL per trade at the final price so wins/losses are meaningful
	finalTime := data[len(data)-1].Timestamp
	for i := range b.results.Trades {
		trade := &b.results.Trades[i]
		trade.ExitTime = finalTime
		trade.ExitPrice = finalPrice
		trade.PnL = (finalPrice-trade.EntryPrice)*trade.Quantity - trade.Commission
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
}
