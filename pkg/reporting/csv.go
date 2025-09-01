package reporting

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
)

// DefaultCSVReporter implements CSV output functionality
type DefaultCSVReporter struct{}

// NewDefaultCSVReporter creates a new CSV reporter
func NewDefaultCSVReporter() *DefaultCSVReporter {
	return &DefaultCSVReporter{}
}

// WriteTradesCSV writes trades to CSV file - extracted from main.go writeTradesCSV
func (r *DefaultCSVReporter) WriteTradesCSV(results *backtest.BacktestResults, path string) error {
	// Ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// If the user requests an Excel file, delegate to Excel writer
	if strings.HasSuffix(strings.ToLower(path), ".xlsx") {
		// Delegate to Excel writer
		return WriteTradesXLSX(results, path)
	}

	// CSV path: write enhanced trades with detailed analysis
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// Enhanced headers with trade performance and strategy context
	if err := w.Write([]string{
		"Cycle",
		"Entry_Time",
		"Exit_Time",
		"Entry_Price",
		"Price_Drop_%",
		"Quantity_USDT",
		"Trade_PnL_$",
		"Win_Loss",
	}); err != nil {
		return err
	}

	// Track cycle start prices for price drop calculation
	cycleData := make(map[int]float64) // cycle -> start price
	
	// Pre-process to find cycle start prices
	for _, t := range results.Trades {
		if _, exists := cycleData[t.Cycle]; !exists {
			cycleData[t.Cycle] = t.EntryPrice
		}
	}

	// Running aggregates for summary
	var cumQty float64
	var cumCost float64
	var totalPnL float64

	for _, t := range results.Trades {
		// Basic calculations
		netCost := t.EntryPrice * t.Quantity
		grossCost := netCost + t.Commission
		cumQty += t.Quantity
		cumCost += grossCost
		totalPnL += t.PnL

		// Trade performance calculations
		winLoss := "W"
		if t.PnL < 0 {
			winLoss = "L"
		}
		
		// Strategy context calculations
		cycleStart := cycleData[t.Cycle]
		var priceDropPct float64
		if t.EntryTime.Equal(t.ExitTime) {
			// For TP exits (synthetic trades), show profit relative to average entry
			if t.EntryPrice > 0 {
				priceDropPct = ((t.ExitPrice - t.EntryPrice) / t.EntryPrice) * 100
			}
		} else {
			// For DCA entries, show drop from cycle start
			priceDropPct = ((cycleStart - t.EntryPrice) / cycleStart) * 100
		}
		
		// Build enhanced row with proper formatting (rounded up values)
		row := []string{
			strconv.Itoa(t.Cycle),
			t.EntryTime.Format("2006-01-02 15:04:05"),
			t.ExitTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.8f", t.EntryPrice),
			fmt.Sprintf("%.0f%%", priceDropPct),        // Rounded percentage with % sign
			fmt.Sprintf("$%.0f", math.Ceil(grossCost)), // Rounded up currency
			fmt.Sprintf("$%.0f", math.Ceil(t.PnL)),     // Rounded up currency
			winLoss,
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}

	// Enhanced summary row with additional metrics
	avgTradeReturn := 0.0
	if len(results.Trades) > 0 {
		totalReturn := 0.0
		for _, t := range results.Trades {
			totalReturn += ((t.ExitPrice - t.EntryPrice) / t.EntryPrice) * 100
		}
		avgTradeReturn = totalReturn / float64(len(results.Trades))
	}
	
	summary := fmt.Sprintf("SUMMARY: total_pnl=$%.0f; total_capital=$%.0f; avg_trade_return=%.2f%%; total_trades=%d", 
		math.Ceil(totalPnL), math.Ceil(cumCost), avgTradeReturn, len(results.Trades))
	
	// Create summary row with empty fields except summary
	summaryRow := make([]string, 8) // Match header count
	summaryRow[7] = summary // Last column
	if err := w.Write(summaryRow); err != nil {
		return err
	}

	return nil
}

// Package-level convenience function
func WriteTradesCSV(results *backtest.BacktestResults, path string) error {
	reporter := NewDefaultCSVReporter()
	return reporter.WriteTradesCSV(results, path)
}
