package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/xuri/excelize/v2"
)

// Output directory helpers
func defaultOutputDir(symbol, interval string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	i := strings.ToLower(strings.TrimSpace(interval))
	if s == "" { s = "UNKNOWN" }
	if i == "" { i = "unknown" }
	return filepath.Join("results", fmt.Sprintf("%s_%s", s, i))
}

func defaultTradesCsvPath(symbol, interval string) string {
	s := strings.ToUpper(strings.TrimSpace(symbol))
	i := strings.ToLower(strings.TrimSpace(interval))
	if s == "" { s = "UNKNOWN" }
	if i == "" { i = "unknown" }
	return filepath.Join("results", fmt.Sprintf("trades_%s_%s.xlsx", s, i))
}

// Output functions
func outputResults(results *backtest.BacktestResults, cfg *BacktestConfig) {
	switch cfg.OutputFormat {
	case "json":
		outputJSON(results, cfg.OutputFile)
	case "csv":
		outputCSV(results, cfg.OutputFile)
	default:
		outputConsole(results, cfg.Verbose)
	}
}

func outputConsole(results *backtest.BacktestResults, verbose bool) {
	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("📊 BACKTEST RESULTS")
	fmt.Println(strings.Repeat("=", 50))
	
	fmt.Printf("💰 Initial Balance:    $%.2f\n", results.StartBalance)
	fmt.Printf("💰 Final Balance:      $%.2f\n", results.EndBalance)
	fmt.Printf("📈 Total Return:       %.2f%%\n", results.TotalReturn*100)
	fmt.Printf("📉 Max Drawdown:       %.2f%%\n", results.MaxDrawdown*100)
	fmt.Printf("📊 Sharpe Ratio:       %.2f\n", results.SharpeRatio)
	fmt.Printf("💹 Profit Factor:      %.2f\n", results.ProfitFactor)
	fmt.Printf("🔄 Total Trades:       %d\n", results.TotalTrades)
	fmt.Printf("✅ Winning Trades:     %d (%.1f%%)\n", results.WinningTrades, 
		float64(results.WinningTrades)/float64(results.TotalTrades)*100)
	fmt.Printf("❌ Losing Trades:      %d (%.1f%%)\n", results.LosingTrades,
		float64(results.LosingTrades)/float64(results.TotalTrades)*100)
	
	if verbose && len(results.Trades) > 0 {
		fmt.Println("\n📋 TRADE HISTORY:")
		fmt.Println(strings.Repeat("-", 80))
		fmt.Printf("%-20s %-10s %-10s %-10s %-10s\n", 
			"Entry Time", "Entry Price", "Quantity", "PnL", "Commission")
		fmt.Println(strings.Repeat("-", 80))
		
		for _, trade := range results.Trades {
			fmt.Printf("%-20s $%-9.2f %-10.6f $%-9.2f $%-9.2f\n",
				trade.EntryTime.Format("2006-01-02 15:04"),
				trade.EntryPrice,
				trade.Quantity,
				trade.PnL,
				trade.Commission)
		}
	}
	
	fmt.Println("\n" + strings.Repeat("=", 50))
}

func outputJSON(results *backtest.BacktestResults, outputFile string) {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal results: %v", err)
	}
	
	if outputFile == "" {
		fmt.Println(string(data))
	} else {
		if err := os.WriteFile(outputFile, data, 0644); err != nil {
			log.Fatalf("Failed to write output file: %v", err)
		}
		fmt.Printf("Results saved to: %s\n", outputFile)
	}
}

func outputCSV(results *backtest.BacktestResults, outputFile string) {
	var w *csv.Writer
	
	if outputFile == "" {
		w = csv.NewWriter(os.Stdout)
	} else {
		file, err := os.Create(outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer file.Close()
		w = csv.NewWriter(file)
	}
	
	// Write summary
	w.Write([]string{"Metric", "Value"})
	w.Write([]string{"Initial Balance", fmt.Sprintf("%.2f", results.StartBalance)})
	w.Write([]string{"Final Balance", fmt.Sprintf("%.2f", results.EndBalance)})
	w.Write([]string{"Total Return %", fmt.Sprintf("%.2f", results.TotalReturn*100)})
	w.Write([]string{"Max Drawdown %", fmt.Sprintf("%.2f", results.MaxDrawdown*100)})
	w.Write([]string{"Sharpe Ratio", fmt.Sprintf("%.2f", results.SharpeRatio)})
	w.Write([]string{"Total Trades", fmt.Sprintf("%d", results.TotalTrades)})
	
	w.Flush()
	
	if outputFile != "" {
		fmt.Printf("Results saved to: %s\n", outputFile)
	}
}

// writeTradesCSV writes trades to CSV or Excel format
func writeTradesCSV(results *backtest.BacktestResults, path string) error {
	// ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil { return err }
	}

	// If the user requests an Excel file, write two tabs: Trades and Cycles
	if strings.HasSuffix(strings.ToLower(path), ".xlsx") {
		return writeTradesXLSX(results, path)
	}

	// CSV path: write only trades with improved headers and order
	f, err := os.Create(path)
	if err != nil { return err }
	defer f.Close()

	w := csv.NewWriter(f)
	defer w.Flush()

	// header with improved names and order (Cycle first)
	if err := w.Write([]string{
		"Cycle",
		"Entry price",
		"Exit price",
		"Entry time",
		"Exit time",
		"Quantity (USDT)",
		"Summary",
	}); err != nil { return err }

	// running aggregates for summary
	var cumQty float64
	var cumCost float64
	var totalPnL float64

	for _, t := range results.Trades {
		netCost := t.EntryPrice * t.Quantity
		grossCost := netCost + t.Commission
		cumQty += t.Quantity
		cumCost += grossCost
		totalPnL += t.PnL

		qtyUSDT := grossCost

		row := []string{
			strconv.Itoa(t.Cycle),
			fmt.Sprintf("%.8f", t.EntryPrice),
			fmt.Sprintf("%.8f", t.ExitPrice),
			t.EntryTime.Format("2006-01-02 15:04:05"),
			t.ExitTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.3f", qtyUSDT),
			"",
		}
		if err := w.Write(row); err != nil { return err }
	}

	// final summary row: only pnl_usdt and capital_usdt in the summary column
	summary := fmt.Sprintf("pnl_usdt=%.3f; capital_usdt=%.3f", totalPnL, cumCost)
	if err := w.Write([]string{"", "", "", "", "", "", summary}); err != nil { return err }

	return nil
}

// writeTradesXLSX writes trades and cycles to separate sheets in an Excel file
func writeTradesXLSX(results *backtest.BacktestResults, path string) error {
	// lazy import
	// NOTE: excelize is a direct dependency in go.mod
	fx := excelize.NewFile()
	defer fx.Close()

	// Create sheets
	const tradesSheet = "Trades"
	const cyclesSheet = "Cycles"
	// Replace default sheet
	fx.SetSheetName(fx.GetSheetName(0), tradesSheet)
	fx.NewSheet(cyclesSheet)

	// Header styles (optional bold)
	headStyle, _ := fx.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})

	// Write Trades header
	tradeHeaders := []string{"Cycle", "Entry price", "Exit price", "Entry time", "Exit time", "Quantity (USDT)", "Summary"}
	for i, h := range tradeHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(tradesSheet, cell, h)
		fx.SetCellStyle(tradesSheet, cell, cell, headStyle)
	}

	// Write Trades rows
	row := 2
	var cumCost float64
	var totalPnL float64
	for _, t := range results.Trades {
		netCost := t.EntryPrice * t.Quantity
		grossCost := netCost + t.Commission
		cumCost += grossCost
		totalPnL += t.PnL

		values := []interface{}{
			t.Cycle,
			fmt.Sprintf("%.8f", t.EntryPrice),
			fmt.Sprintf("%.8f", t.ExitPrice),
			t.EntryTime.Format("2006-01-02 15:04:05"),
			t.ExitTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.3f", grossCost),
			"",
		}
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(tradesSheet, cell, v)
		}
		row++
	}
	// Final summary cell in the last row, last column
	if row > 2 {
		cell, _ := excelize.CoordinatesToCellName(len(tradeHeaders), row)
		fx.SetCellValue(tradesSheet, cell, fmt.Sprintf("pnl_usdt=%.3f; capital_usdt=%.3f", totalPnL, cumCost))
	}

	// Write Cycles sheet if any
	cycleHeaders := []string{"Cycle number", "Start time", "End time", "Entries", "Target price", "Average price", "PnL (USDT)", "Capital (USDT)"}
	for i, h := range cycleHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(cyclesSheet, cell, h)
		fx.SetCellStyle(cyclesSheet, cell, cell, headStyle)
	}
	row = 2
	if len(results.Cycles) > 0 {
		for _, c := range results.Cycles {
			// compute capital_usdt for this cycle from trades
			capital := 0.0
			for _, t := range results.Trades {
				if t.Cycle == c.CycleNumber {
					capital += t.EntryPrice*t.Quantity + t.Commission
				}
			}
			values := []interface{}{
				c.CycleNumber,
				c.StartTime.Format("2006-01-02 15:04:05"),
				c.EndTime.Format("2006-01-02 15:04:05"),
				c.Entries,
				fmt.Sprintf("%.8f", c.TargetPrice),
				fmt.Sprintf("%.8f", c.AvgEntry),
				fmt.Sprintf("%.3f", c.RealizedPnL),
				fmt.Sprintf("%.3f", capital),
			}
			for i, v := range values {
				cell, _ := excelize.CoordinatesToCellName(i+1, row)
				fx.SetCellValue(cyclesSheet, cell, v)
			}
			row++
		}
	}

	// Save workbook
	return fx.SaveAs(path)
} 