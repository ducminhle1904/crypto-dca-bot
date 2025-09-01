package reporting

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/xuri/excelize/v2"
)

// DefaultExcelReporter implements Excel output functionality
type DefaultExcelReporter struct{}

// NewDefaultExcelReporter creates a new Excel reporter
func NewDefaultExcelReporter() *DefaultExcelReporter {
	return &DefaultExcelReporter{}
}

// WriteTradesXLSX writes trades to Excel file - extracted from main.go writeTradesXLSX
func (r *DefaultExcelReporter) WriteTradesXLSX(results *backtest.BacktestResults, path string) error {
	// Ensure directory exists before creating file
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	fx := excelize.NewFile()
	defer fx.Close()

	// Create sheets
	const tradesSheet = "Trades"
	const cyclesSheet = "Cycles"
	const detailedSheet = "Detailed Analysis"
	
	// Replace default sheet and create additional sheets
	fx.SetSheetName(fx.GetSheetName(0), tradesSheet)
	fx.NewSheet(cyclesSheet)
	fx.NewSheet(detailedSheet)

	// Create professional styles
	styles, err := r.createExcelStyles(fx)
	if err != nil {
		return err
	}

	// Write all sheets
	if err := r.writeTradesSheet(fx, tradesSheet, results, styles); err != nil {
		return err
	}
	
	if err := r.writeCyclesSheet(fx, cyclesSheet, results, styles); err != nil {
		return err
	}
	
	if err := r.writeDetailedAnalysisSheet(fx, detailedSheet, results, styles); err != nil {
		return err
	}

	// Save workbook
	return fx.SaveAs(path)
}

// createExcelStyles creates all Excel styles - extracted from main.go
func (r *DefaultExcelReporter) createExcelStyles(fx *excelize.File) (ExcelStyles, error) {
	var styles ExcelStyles
	var err error

	// Header style - Dark blue background with white text
	styles.HeaderStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "FFFFFF",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"2F4F4F"}, // Dark slate gray
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	if err != nil {
		return styles, err
	}

	// Currency style (right aligned, $ format)
	styles.CurrencyStyle, err = fx.NewStyle(&excelize.Style{
		NumFmt: 7, // Currency format with $ symbol
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return styles, err
	}

	// Percentage style (right aligned, % format)
	styles.PercentStyle, err = fx.NewStyle(&excelize.Style{
		NumFmt: 9, // Percentage format with % symbol
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return styles, err
	}

	// Red percentage style for DCA entries
	styles.RedPercentStyle, err = fx.NewStyle(&excelize.Style{
		NumFmt: 9, // Percentage format with % symbol
		Font: &excelize.Font{
			Color: "FF0000", // Red text
		},
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return styles, err
	}

	// Green percentage style for TP exits
	styles.GreenPercentStyle, err = fx.NewStyle(&excelize.Style{
		NumFmt: 9, // Percentage format with % symbol
		Font: &excelize.Font{
			Color: "008000", // Green text
		},
		Alignment: &excelize.Alignment{
			Horizontal: "right",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return styles, err
	}

	// Base style (light borders)
	styles.BaseStyle, err = fx.NewStyle(&excelize.Style{
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return styles, err
	}

	// Entry style (light blue background)
	styles.EntryStyle, err = fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"E6F3FF"}, // Light blue
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return styles, err
	}

	// Exit style (light green background)
	styles.ExitStyle, err = fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"E6FFE6"}, // Light green
			Pattern: 1,
		},
		Border: []excelize.Border{
			{Type: "left", Color: "E0E0E0", Style: 1},
			{Type: "right", Color: "E0E0E0", Style: 1},
			{Type: "bottom", Color: "E0E0E0", Style: 1},
		},
	})
	if err != nil {
		return styles, err
	}

	// Cycle header style (blue background)
	styles.CycleHeaderStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "FFFFFF",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"4472C4"}, // Blue
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "right", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	if err != nil {
		return styles, err
	}

	// Summary style (same as cycle header)
	styles.SummaryStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   11,
			Color:  "FFFFFF",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"4472C4"}, // Blue
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "right", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	if err != nil {
		return styles, err
	}

	// Final summary style (dark gray background)
	styles.FinalSummaryStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold:   true,
			Size:   12,
			Color:  "FFFFFF",
			Family: "Calibri",
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"2F4F4F"}, // Dark slate gray
			Pattern: 1,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 3},
			{Type: "right", Color: "000000", Style: 3},
			{Type: "top", Color: "000000", Style: 3},
			{Type: "bottom", Color: "000000", Style: 3},
		},
	})
	if err != nil {
		return styles, err
	}

	return styles, nil
}

// writeTradesSheet writes the trades sheet with enhanced format matching original implementation
func (r *DefaultExcelReporter) writeTradesSheet(fx *excelize.File, sheet string, results *backtest.BacktestResults, styles ExcelStyles) error {
	// Set column widths for enhanced layout with balance tracking
	fx.SetColWidth(sheet, "A", "A", 8)   // Cycle
	fx.SetColWidth(sheet, "B", "B", 12)  // Type
	fx.SetColWidth(sheet, "C", "C", 10)  // Sequence
	fx.SetColWidth(sheet, "D", "D", 18)  // Timestamp
	fx.SetColWidth(sheet, "E", "E", 12)  // Price
	fx.SetColWidth(sheet, "F", "F", 12)  // Quantity
	fx.SetColWidth(sheet, "G", "G", 12)  // Commission
	fx.SetColWidth(sheet, "H", "H", 14)  // PnL
	fx.SetColWidth(sheet, "I", "I", 12)  // Price Change %
	fx.SetColWidth(sheet, "J", "J", 14)  // Running Balance
	fx.SetColWidth(sheet, "K", "K", 14)  // Balance Change
	fx.SetColWidth(sheet, "L", "L", 20)  // TP Info

	// Write Enhanced Trades headers - includes balance tracking and TP descriptions
	tradeHeaders := []string{
		"Cycle", "Type", "Sequence", "Timestamp", "Price", "Quantity", 
		"Commission", "PnL", "Price Change %", "Running Balance", "Balance Change", "TP Info",
	}
	for i, h := range tradeHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(sheet, cell, h)
		fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle)
	}

	// Enhanced trade organization with chronological sequencing
	type EnhancedTradeData struct {
		Trade         backtest.Trade
		IsEntry       bool
		Sequence      int
		BalanceBefore float64
		BalanceAfter  float64
		BalanceChange float64
		Description   string
	}
	
	type CycleTradeData struct {
		CycleNumber         int
		ChronologicalTrades []EnhancedTradeData
		StartPrice          float64
		StartBalance        float64
		EndBalance          float64
	}
	
	cycleMap := make(map[int]*CycleTradeData)
	
	// Group all trades by cycle first
	for _, t := range results.Trades {
		if cycleMap[t.Cycle] == nil {
			cycleMap[t.Cycle] = &CycleTradeData{
				CycleNumber:         t.Cycle,
				ChronologicalTrades: make([]EnhancedTradeData, 0),
			}
		}
		
		isEntry := !t.EntryTime.Equal(t.ExitTime)
		
		// Set cycle start price from first entry
		if isEntry && cycleMap[t.Cycle].StartPrice == 0 {
			cycleMap[t.Cycle].StartPrice = t.EntryPrice
		}
		
		// Add trade to cycle
		cycleMap[t.Cycle].ChronologicalTrades = append(cycleMap[t.Cycle].ChronologicalTrades, EnhancedTradeData{
			Trade:   t,
			IsEntry: isEntry,
		})
	}
	
	// Calculate running balance by processing cycles in order
	runningBalance := results.StartBalance
	
	// Get cycles in chronological order (by cycle number)
	var sortedCycleNums []int
	for cycleNum := range cycleMap {
		sortedCycleNums = append(sortedCycleNums, cycleNum)
	}
	// Sort cycle numbers
	for i := 0; i < len(sortedCycleNums)-1; i++ {
		for j := i + 1; j < len(sortedCycleNums); j++ {
			if sortedCycleNums[i] > sortedCycleNums[j] {
				sortedCycleNums[i], sortedCycleNums[j] = sortedCycleNums[j], sortedCycleNums[i]
			}
		}
	}
	
	// Process each cycle in order
	for _, cycleNum := range sortedCycleNums {
		cycle := cycleMap[cycleNum]
		cycle.StartBalance = runningBalance
		
		// Sort trades within THIS cycle chronologically
		trades := cycle.ChronologicalTrades
		for i := 0; i < len(trades)-1; i++ {
			for j := i + 1; j < len(trades); j++ {
				time1 := trades[i].Trade.EntryTime
				time2 := trades[j].Trade.EntryTime
				if !trades[i].IsEntry {
					time1 = trades[i].Trade.ExitTime
				}
				if !trades[j].IsEntry {
					time2 = trades[j].Trade.ExitTime
				}
				
				if time1.After(time2) {
					trades[i], trades[j] = trades[j], trades[i]
				}
			}
		}
		
		// Calculate balance changes within this cycle
		for i := range trades {
			trade := &trades[i]
			trade.Sequence = i + 1
			trade.BalanceBefore = runningBalance
			
			if trade.IsEntry {
				// DCA entry - spend money
				balanceChange := -(trade.Trade.EntryPrice*trade.Trade.Quantity + trade.Trade.Commission)
				trade.BalanceChange = balanceChange
				trade.Description = fmt.Sprintf("DCA Entry #%d", trade.Sequence)
			} else {
				// TP exit - receive money  
				balanceChange := (trade.Trade.ExitPrice*trade.Trade.Quantity - trade.Trade.Commission)
				trade.BalanceChange = balanceChange
				
				// Determine how many TP levels this exit represents based on quantity
				totalCycleQty := 0.0
				for _, t := range cycle.ChronologicalTrades {
					if t.IsEntry {
						totalCycleQty += t.Trade.Quantity
					}
				}
				
				// Calculate what percentage of position this exit represents
				exitPercentage := (trade.Trade.Quantity / totalCycleQty) * 100
				
				if exitPercentage >= 90 {
					trade.Description = "TP Complete (100%)"
				} else if exitPercentage >= 70 {
					trade.Description = "TP Levels 1-4 (80%)"
				} else if exitPercentage >= 50 {
					trade.Description = "TP Levels 1-3 (60%)"
				} else if exitPercentage >= 30 {
					trade.Description = "TP Levels 1-2 (40%)"
				} else {
					trade.Description = "TP Level 1 (20%)"
				}
			}
			
			runningBalance += trade.BalanceChange
			trade.BalanceAfter = runningBalance
		}
		
		cycle.EndBalance = runningBalance
	}

	// Write organized trade data
	row := 2
	var totalPnL float64
	var totalCost float64
	
	// Calculate total PnL correctly: Final Balance - Initial Balance
	totalPnL = results.EndBalance - results.StartBalance

	for _, cycleNum := range sortedCycleNums {
		cycleData := cycleMap[cycleNum]
		cycleCost := 0.0
		cyclePnL := 0.0
		
		// Simple cycle header without balance info
		cycleHeaderRange := fmt.Sprintf("A%d:L%d", row, row)
		fx.MergeCell(sheet, cycleHeaderRange, "")
		headerCell, _ := excelize.CoordinatesToCellName(1, row)
		fx.SetCellValue(sheet, headerCell, fmt.Sprintf("═══ CYCLE %d ═══", cycleNum))
		fx.SetCellStyle(sheet, headerCell, headerCell, styles.CycleHeaderStyle)
		row++
		
		// Add all trades in chronological order
		for _, enhancedTrade := range cycleData.ChronologicalTrades {
			trade := enhancedTrade.Trade
			
			// Calculate costs and PnL for summary
			if enhancedTrade.IsEntry {
				cost := trade.EntryPrice * trade.Quantity + trade.Commission
				cycleCost += cost
				totalCost += cost
			} else {
				cyclePnL += trade.PnL
			}
			
			// Calculate price change
			priceChange := 0.0
			if enhancedTrade.IsEntry && cycleData.StartPrice > 0 {
				// For entries, show drop from cycle start
				priceChange = ((cycleData.StartPrice - trade.EntryPrice) / cycleData.StartPrice) * 100
			} else if !enhancedTrade.IsEntry && trade.EntryPrice > 0 {
				// For exits, show profit relative to average entry price
				priceChange = ((trade.ExitPrice - trade.EntryPrice) / trade.EntryPrice) * 100
			}
			
			// Determine trade type and styling
			tradeType := "📈 BUY"
			tradeStyle := styles.EntryStyle
			var tradePnL interface{} = ""
			tradeTime := trade.EntryTime
			tradePrice := trade.EntryPrice
			
			if !enhancedTrade.IsEntry {
				tradeType = "💰 SELL"
				tradeStyle = styles.ExitStyle
				tradePnL = trade.PnL
				tradeTime = trade.ExitTime
				tradePrice = trade.ExitPrice
			}
			
			values := []interface{}{
				cycleNum,
				tradeType,
				enhancedTrade.Sequence,
				tradeTime.Format("2006-01-02 15:04:05"),
				tradePrice,
				trade.Quantity,
				trade.Commission,
				tradePnL,
				priceChange / 100,
				enhancedTrade.BalanceAfter,
				enhancedTrade.BalanceChange,
				enhancedTrade.Description,
			}
			
			r.writeEnhancedTradeRow(fx, sheet, row, values, styles, enhancedTrade.IsEntry, tradeStyle)
			row++
		}
		
		// Add cycle summary row
		entryCount := 0
		exitCount := 0
		for _, trade := range cycleData.ChronologicalTrades {
			if trade.IsEntry {
				entryCount++
			} else {
				exitCount++
			}
		}
		
		// Use actual cycle profit (balance change) instead of individual trade PnL sum
		actualCycleProfit := cycleData.EndBalance - cycleData.StartBalance
		
		// Create cycle summary header
		summaryHeaderRange := fmt.Sprintf("A%d:L%d", row, row)
		fx.MergeCell(sheet, summaryHeaderRange, "")
		summaryHeaderCell, _ := excelize.CoordinatesToCellName(1, row)
		fx.SetCellValue(sheet, summaryHeaderCell, fmt.Sprintf("📊 CYCLE %d SUMMARY - Balance: $%.2f → $%.2f | Profit: $%.2f | Entries: %d | Exits: %d", 
			cycleNum, cycleData.StartBalance, cycleData.EndBalance, actualCycleProfit, entryCount, exitCount))
		fx.SetCellStyle(sheet, summaryHeaderCell, summaryHeaderCell, styles.SummaryStyle)
		row++
		
		// Add spacing between cycles
		row++
	}
	
	// Add final summary row
	finalSummaryRange := fmt.Sprintf("A%d:L%d", row, row)
	fx.MergeCell(sheet, finalSummaryRange, "")
	finalSummaryCell, _ := excelize.CoordinatesToCellName(1, row)
	fx.SetCellValue(sheet, finalSummaryCell, fmt.Sprintf("🎯 FINAL SUMMARY - Start: $%.2f | End: $%.2f | Total Profit: $%.2f | Total Return: %.2f%%", 
		results.StartBalance, results.EndBalance, totalPnL, (totalPnL/results.StartBalance)*100))
	fx.SetCellStyle(sheet, finalSummaryCell, finalSummaryCell, styles.FinalSummaryStyle)

	return nil
}

// writeCyclesSheet writes the cycles sheet with comprehensive format matching original implementation
func (r *DefaultExcelReporter) writeCyclesSheet(fx *excelize.File, sheet string, results *backtest.BacktestResults, styles ExcelStyles) error {
	// Set column widths for cycles sheet (matching original)
	fx.SetColWidth(sheet, "A", "A", 8)   // Cycle
	fx.SetColWidth(sheet, "B", "B", 18)  // Start Time
	fx.SetColWidth(sheet, "C", "C", 18)  // End Time
	fx.SetColWidth(sheet, "D", "D", 12)  // Duration
	fx.SetColWidth(sheet, "E", "E", 10)  // Entries
	fx.SetColWidth(sheet, "F", "F", 12)  // Avg Entry
	fx.SetColWidth(sheet, "G", "G", 12)  // Exit Price
	fx.SetColWidth(sheet, "H", "H", 12)  // Capital Used
	fx.SetColWidth(sheet, "I", "I", 12)  // Capital %
	fx.SetColWidth(sheet, "J", "J", 12)  // Balance Before
	fx.SetColWidth(sheet, "K", "K", 12)  // Balance After
	fx.SetColWidth(sheet, "L", "L", 12)  // PnL
	fx.SetColWidth(sheet, "M", "M", 10)  // ROI %
	
	// Cycles sheet title and headers
	fx.SetCellValue(sheet, "A1", "🔄 CYCLE ANALYSIS WITH CAPITAL USAGE")
	fx.SetCellStyle(sheet, "A1", "A1", styles.HeaderStyle)
	
	cycleHeaders := []string{
		"Cycle", "Start Time", "End Time", "Duration", "Entries", "Avg Entry", "Exit Price", 
		"Capital Used ($)", "Capital %", "Balance Before ($)", "Balance After ($)", "PnL ($)", "ROI %",
	}
	
	for i, h := range cycleHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		fx.SetCellValue(sheet, cell, h)
		fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle)
	}
	
	// Calculate balance and capital usage per cycle (cycles are SEQUENTIAL)
	cycleBalances := make(map[int]struct {
		before float64
		after  float64
		capital float64
	})
	
	// Sort cycles by cycle number to process them sequentially
	var sortedCycles []backtest.CycleSummary
	sortedCycles = append(sortedCycles, results.Cycles...)
	
	// Sort cycles by cycle number
	for i := 0; i < len(sortedCycles)-1; i++ {
		for j := i + 1; j < len(sortedCycles); j++ {
			if sortedCycles[i].CycleNumber > sortedCycles[j].CycleNumber {
				sortedCycles[i], sortedCycles[j] = sortedCycles[j], sortedCycles[i]
			}
		}
	}
	
	// Process cycles sequentially
	runningBalance := results.StartBalance
	
	for _, c := range sortedCycles {
		// Balance before this cycle starts
		balanceBefore := runningBalance
		
		// Calculate capital used in this cycle (sum of all DCA entries)
		cycleCapital := 0.0
		for _, t := range results.Trades {
			if t.Cycle == c.CycleNumber && !t.EntryTime.Equal(t.ExitTime) {
				// DCA entry trade
				cycleCapital += t.EntryPrice*t.Quantity + t.Commission
			}
		}
		
		// Update running balance after this cycle completes
		runningBalance = balanceBefore + c.RealizedPnL
		
		// Store cycle balance data
		cycleBalances[c.CycleNumber] = struct {
			before float64
			after  float64
			capital float64
		}{
			before:  balanceBefore,
			after:   runningBalance,
			capital: cycleCapital,
		}
	}
	
	// Write cycle data
	cycleRow := 4
	for _, c := range results.Cycles {
		duration := c.EndTime.Sub(c.StartTime)
		durationStr := fmt.Sprintf("%.1fh", duration.Hours())
		
		// Get balance data for this cycle
		balanceData := cycleBalances[c.CycleNumber]
		
		// Calculate capital percentage of balance at cycle start
		capitalPercent := 0.0
		if balanceData.before > 0 {
			capitalPercent = (balanceData.capital / balanceData.before) * 100
		}
		
		// Determine exit price - prefer cycle's final exit price
		exitPrice := c.FinalExitPrice
		if exitPrice == 0.0 {
			// Fallback: find the latest synthetic TP exit for this cycle
			for i := len(results.Trades) - 1; i >= 0; i-- {
				t := results.Trades[i]
				if t.Cycle == c.CycleNumber && t.EntryTime.Equal(t.ExitTime) {
					exitPrice = t.ExitPrice
					break
				}
			}
		}
		
		roi := 0.0
		if balanceData.capital > 0 {
			roi = (c.RealizedPnL / balanceData.capital) * 100
		}
		
		cycleValues := []interface{}{
			c.CycleNumber,
			c.StartTime.Format("2006-01-02 15:04"),
			c.EndTime.Format("2006-01-02 15:04"),
			durationStr,
			c.Entries,
			c.AvgEntry,
			exitPrice,
			balanceData.capital,
			capitalPercent,
			balanceData.before,
			balanceData.after,
			c.RealizedPnL,
			roi,
		}
		
		for i, v := range cycleValues {
			cell, _ := excelize.CoordinatesToCellName(i+1, cycleRow)
			fx.SetCellValue(sheet, cell, v)
			
			// Apply styling
			if i == 5 || i == 6 { // Avg Entry, Exit Price
				fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
			} else if i >= 7 && i <= 11 { // Capital Used, Balance Before/After, PnL
				fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
			} else if i == 8 || i == 12 { // Capital %, ROI %
				fx.SetCellStyle(sheet, cell, cell, styles.PercentStyle)
			}
		}
		cycleRow++
	}
	
	// Add summary row
	cycleRow += 1
	fx.SetCellValue(sheet, "A"+fmt.Sprintf("%d", cycleRow), "📊 SUMMARY")
	fx.SetCellStyle(sheet, "A"+fmt.Sprintf("%d", cycleRow), "A"+fmt.Sprintf("%d", cycleRow), styles.HeaderStyle)
	
	// Calculate totals
	totalCapitalUsed := 0.0
	cyclesTotalPnL := 0.0
	completedCycles := 0
	
	for _, c := range results.Cycles {
		balanceData := cycleBalances[c.CycleNumber]
		totalCapitalUsed += balanceData.capital
		cyclesTotalPnL += c.RealizedPnL
		completedCycles++ // Count all cycles since we removed status column
	}
	
	avgCapitalPercent := 0.0
	if len(results.Cycles) > 0 {
		avgCapitalPercent = (totalCapitalUsed / (results.StartBalance * float64(len(results.Cycles)))) * 100
	}
	
	cycleRow++
	summaryValues := []interface{}{
		"TOTALS:",
		"",
		"",
		"",
		fmt.Sprintf("%d cycles", len(results.Cycles)),
		"",
		"",
		totalCapitalUsed,
		avgCapitalPercent,
		results.StartBalance,
		results.EndBalance,
		cyclesTotalPnL,
		fmt.Sprintf("%.1f%%", (cyclesTotalPnL/totalCapitalUsed)*100),
	}
	
	for i, v := range summaryValues {
		cell, _ := excelize.CoordinatesToCellName(i+1, cycleRow)
		fx.SetCellValue(sheet, cell, v)
		
		// Apply styling to summary row
		if i >= 7 && i <= 11 { // Capital Used, Balance Before/After, PnL
			fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
		} else if i == 8 { // Capital %
			fx.SetCellStyle(sheet, cell, cell, styles.PercentStyle)
		}
		fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle) // Make summary row bold
	}
	
	// Add filter to cycles sheet
	if cycleRow > 4 {
		fx.AutoFilter(sheet, fmt.Sprintf("A3:M%d", cycleRow-2), []excelize.AutoFilterOptions{})
	}

	return nil
}

// writeDetailedAnalysisSheet writes the detailed analysis sheet with comprehensive format matching original implementation
func (r *DefaultExcelReporter) writeDetailedAnalysisSheet(fx *excelize.File, sheet string, results *backtest.BacktestResults, styles ExcelStyles) error {
	// Set column widths for better readability
	fx.SetColWidth(sheet, "A", "A", 25)  // Metric
	fx.SetColWidth(sheet, "B", "B", 18)  // Value
	fx.SetColWidth(sheet, "C", "C", 35)  // Description/Analysis
	fx.SetColWidth(sheet, "D", "D", 15)  // Additional Data
	fx.SetColWidth(sheet, "E", "E", 20)  // Insights
	
	// Create enhanced styles
	titleStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "right", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
		},
	})
	
	sectionStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 14, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 1},
			{Type: "right", Color: "000000", Style: 1},
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})
	
	insightStyle, _ := fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 10, Color: "2F4F4F"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"F0F8FF"}, Pattern: 1},
	})
	
	// Main title
	fx.MergeCell(sheet, "A1:E1", "")
	fx.SetCellValue(sheet, "A1", "🚀 COMPREHENSIVE DCA STRATEGY ANALYSIS")
	fx.SetCellStyle(sheet, "A1", "A1", titleStyle)
	fx.SetRowHeight(sheet, 1, 30)
	
	row := 3
	
	// Calculate comprehensive metrics
	totalPnL := results.EndBalance - results.StartBalance
	totalCost := 0.0
	totalTPHits := 0
	totalDCAEntries := 0
	avgCycleDuration := 0.0
	longestCycle := 0.0
	shortestCycle := 999999.0
	
	// Calculate win/loss from TP hits (same logic as console)
	winTrades := 0
	totalTrades := 0
	for _, c := range results.Cycles {
		for _, pe := range c.PartialExits {
			totalTrades++
			if pe.PnL > 0 {
				winTrades++
			}
		}
	}
	
	// Calculate detailed cycle metrics
	for _, t := range results.Trades {
		if !t.EntryTime.Equal(t.ExitTime) {
			totalCost += t.EntryPrice*t.Quantity + t.Commission
			totalDCAEntries++
		}
	}
	
	for _, c := range results.Cycles {
		totalTPHits += len(c.PartialExits)
		duration := c.EndTime.Sub(c.StartTime).Hours()
		avgCycleDuration += duration
		if duration > longestCycle {
			longestCycle = duration
		}
		if duration < shortestCycle {
			shortestCycle = duration
		}
	}
	
	if len(results.Cycles) > 0 {
		avgCycleDuration /= float64(len(results.Cycles))
	}
	if shortestCycle == 999999.0 {
		shortestCycle = 0
	}
	
	// 📊 EXECUTIVE SUMMARY
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "📊 EXECUTIVE SUMMARY")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(winTrades) / float64(totalTrades) * 100
	}
	
	roi := 0.0
	if totalCost > 0 {
		roi = (totalPnL / totalCost) * 100
	}
	
	executiveSummary := [][]interface{}{
		{"🎯 Strategy Performance", fmt.Sprintf("%.1f%% Win Rate", winRate), "Excellent performance with consistent profits", "", "✅ Strong strategy"},
		{"💰 Financial Results", fmt.Sprintf("$%.0f PnL (%.1f%% ROI)", totalPnL, roi), "Total profit from strategy execution", "", "💡 Capital efficient"},
		{"⏱️ Time Efficiency", fmt.Sprintf("%.1f hours avg cycle", avgCycleDuration), "Average time to complete each DCA cycle", "", "⚡ Quick turnaround"},
		{"🔄 Cycle Completion", fmt.Sprintf("%d/%d cycles (%.1f%%)", results.CompletedCycles, len(results.Cycles), float64(results.CompletedCycles)/float64(len(results.Cycles))*100), "Percentage of cycles that hit all TP levels", "", "🎯 High completion"},
	}
	
	for _, summary := range executiveSummary {
		for i, v := range summary {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i == 1 { // Value column
				fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
			} else if i == 4 { // Insight column
				fx.SetCellStyle(sheet, cell, cell, insightStyle)
			}
		}
		row++
	}
	
	row += 2
	
	// 🎯 DETAILED PERFORMANCE METRICS
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "🎯 DETAILED PERFORMANCE METRICS")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	performanceMetrics := [][]interface{}{
		{"Total Return", fmt.Sprintf("%.2f%%", results.TotalReturn*100), "Overall strategy performance vs initial capital", "", ""},
		{"Profit Factor", fmt.Sprintf("%.2f", results.ProfitFactor), "Ratio of gross profits to gross losses", "", r.getProfitFactorInsight(results.ProfitFactor)},
		{"Sharpe Ratio", fmt.Sprintf("%.2f", results.SharpeRatio), "Risk-adjusted return measurement", "", r.getSharpeInsight(results.SharpeRatio)},
		{"Max Drawdown", fmt.Sprintf("%.2f%%", results.MaxDrawdown*100), "Largest peak-to-trough decline", "", r.getDrawdownInsight(results.MaxDrawdown*100)},
		{"Capital Efficiency", fmt.Sprintf("%.1f%%", (totalCost/results.StartBalance)*100), "Percentage of available capital deployed", "", r.getCapitalEfficiencyInsight((totalCost/results.StartBalance)*100)},
		{"Average Trade Return", fmt.Sprintf("%.2f%%", totalPnL/float64(totalTrades)), "Average profit per TP level hit", "", ""},
	}
	
	for _, metric := range performanceMetrics {
		for i, v := range metric {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i == 1 && v != "" { // Value column
				if strings.Contains(v.(string), "$") {
					fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
				} else if strings.Contains(v.(string), "%") {
					fx.SetCellStyle(sheet, cell, cell, styles.PercentStyle)
				}
			} else if i == 4 && v != "" { // Insight column
				fx.SetCellStyle(sheet, cell, cell, insightStyle)
			}
		}
		row++
	}
	
	row += 2
	
	// 🔄 CYCLE ANALYSIS
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "🔄 COMPREHENSIVE CYCLE ANALYSIS")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	cycleMetrics := [][]interface{}{
		{"Total Cycles", fmt.Sprintf("%d", len(results.Cycles)), "Number of DCA cycles initiated", "", ""},
		{"Completed Cycles", fmt.Sprintf("%d (%.1f%%)", results.CompletedCycles, float64(results.CompletedCycles)/float64(len(results.Cycles))*100), "Cycles that hit all 5 TP levels", "", r.getCycleCompletionInsight(float64(results.CompletedCycles)/float64(len(results.Cycles))*100)},
		{"Total DCA Entries", fmt.Sprintf("%d", totalDCAEntries), "Number of buy orders executed", "", ""},
		{"Total TP Hits", fmt.Sprintf("%d", totalTPHits), "Number of profitable sell orders", "", ""},
		{"Avg Cycle Duration", fmt.Sprintf("%.1f hours", avgCycleDuration), "Average time from first DCA to last TP", "", r.getCycleDurationInsight(avgCycleDuration)},
		{"Longest Cycle", fmt.Sprintf("%.1f hours", longestCycle), "Maximum time for cycle completion", "", ""},
		{"Shortest Cycle", fmt.Sprintf("%.1f hours", shortestCycle), "Minimum time for cycle completion", "", ""},
		{"DCA Entries per Cycle", fmt.Sprintf("%.1f", float64(totalDCAEntries)/float64(len(results.Cycles))), "Average number of DCA entries per cycle", "", r.getDCAEfficiencyInsight(float64(totalDCAEntries)/float64(len(results.Cycles)))},
	}
	
	for _, metric := range cycleMetrics {
		for i, v := range metric {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i == 4 && v != "" { // Insight column
				fx.SetCellStyle(sheet, cell, cell, insightStyle)
			}
		}
		row++
	}
	
	row += 2
	
	// 🎯 TP LEVEL PERFORMANCE ANALYSIS
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "🎯 TP LEVEL PERFORMANCE ANALYSIS")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	// Headers for TP analysis
	tpHeaders := []string{"TP Level", "Hit Count", "Success Rate", "Avg PnL", "Total PnL"}
	for i, h := range tpHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		fx.SetCellValue(sheet, cell, h)
		fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle)
	}
	row++
	
	// Calculate TP level statistics
	tpStats := make(map[int]struct {
		count    int
		totalPnL float64
		totalGain float64
	})
	
	for _, c := range results.Cycles {
		for _, pe := range c.PartialExits {
			if _, exists := tpStats[pe.TPLevel]; !exists {
				tpStats[pe.TPLevel] = struct {
					count    int
					totalPnL float64
					totalGain float64
				}{}
			}
			stats := tpStats[pe.TPLevel]
			stats.count++
			stats.totalPnL += pe.PnL
			if c.AvgEntry > 0 {
				stats.totalGain += (pe.Price - c.AvgEntry) / c.AvgEntry * 100
			}
			tpStats[pe.TPLevel] = stats
		}
	}
	
	// Write TP level data with insights
	for level := 1; level <= 5; level++ {
		stats := tpStats[level]
		successRate := 0.0
		avgPnL := 0.0
		
		if len(results.Cycles) > 0 {
			successRate = float64(stats.count) / float64(len(results.Cycles)) * 100
		}
		if stats.count > 0 {
			avgPnL = stats.totalPnL / float64(stats.count)
		}
		
		values := []interface{}{
			fmt.Sprintf("TP %d", level),
			stats.count,
			fmt.Sprintf("%.1f%%", successRate),
			fmt.Sprintf("$%.2f", avgPnL),
			fmt.Sprintf("$%.2f", stats.totalPnL),
		}
		
		for i, v := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i >= 2 && i <= 4 { // Percentage and currency columns
				if strings.Contains(fmt.Sprintf("%v", v), "$") {
					fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
				} else if strings.Contains(fmt.Sprintf("%v", v), "%") {
					fx.SetCellStyle(sheet, cell, cell, styles.PercentStyle)
				}
			}
		}
		row++
	}
	
	row += 2
	
	// 💡 STRATEGIC INSIGHTS & RECOMMENDATIONS
	fx.MergeCell(sheet, fmt.Sprintf("A%d:E%d", row, row), "")
	fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), "💡 STRATEGIC INSIGHTS & RECOMMENDATIONS")
	fx.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), sectionStyle)
	row += 2
	
	recommendations := r.getStrategicRecommendations(results, totalCost, winRate, avgCycleDuration)
	
	for _, rec := range recommendations {
		for i, v := range rec {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			fx.SetCellValue(sheet, cell, v)
			
			if i == 0 { // Category column
				fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle)
			} else if i >= 1 { // Recommendation columns
				fx.SetCellStyle(sheet, cell, cell, insightStyle)
			}
		}
		row++
	}

	return nil
}

// WriteTradeRow writes a single trade row with styling - extracted from main.go
func (r *DefaultExcelReporter) WriteTradeRow(fx *excelize.File, sheet string, row int, values []interface{}, styles ExcelStyles) {
	for i, v := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		fx.SetCellValue(sheet, cell, v)
		
		// Apply conditional styling based on column
		if i == 7 || i == 9 { // PnL and Cumulative Cost columns
			fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
		} else if i == 8 { // Price Change % column
			fx.SetCellStyle(sheet, cell, cell, styles.PercentStyle)
		} else {
			fx.SetCellStyle(sheet, cell, cell, styles.BaseStyle)
		}
	}
}

// WriteEnhancedTradeRow writes enhanced trade row with color coding - extracted from main.go
func (r *DefaultExcelReporter) WriteEnhancedTradeRow(fx *excelize.File, sheet string, row int, values []interface{}, styles ExcelStyles, isEntry bool) {
	for i, v := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		fx.SetCellValue(sheet, cell, v)
		
		// Apply specific formatting based on enhanced column layout
		if i == 4 { // Price column
			fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
		} else if i == 6 || i == 7 { // Commission and PnL columns
			fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
		} else if i == 8 { // Price Change % column - color coded
			if isEntry {
				fx.SetCellStyle(sheet, cell, cell, styles.RedPercentStyle) // Red for DCA entries
			} else {
				fx.SetCellStyle(sheet, cell, cell, styles.GreenPercentStyle) // Green for TP exits
			}
		} else if i == 9 || i == 10 { // Running Balance and Balance Change columns
			fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
		} else {
			fx.SetCellStyle(sheet, cell, cell, styles.BaseStyle)
		}
	}
}

// writeEnhancedTradeRow writes enhanced trade row with specific styling - internal method
func (r *DefaultExcelReporter) writeEnhancedTradeRow(fx *excelize.File, sheet string, row int, values []interface{}, styles ExcelStyles, isEntry bool, tradeStyle int) {
	for i, v := range values {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		fx.SetCellValue(sheet, cell, v)
		
		// Apply specific formatting based on column type
		if i == 4 { // Price column
			fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
		} else if i == 6 || i == 7 { // Commission and PnL columns
			fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
		} else if i == 8 { // Price Change % column - color coded
			if isEntry {
				fx.SetCellStyle(sheet, cell, cell, styles.RedPercentStyle) // Red for DCA entries
			} else {
				fx.SetCellStyle(sheet, cell, cell, styles.GreenPercentStyle) // Green for TP exits
			}
		} else if i == 9 || i == 10 { // Running Balance and Balance Change columns
			fx.SetCellStyle(sheet, cell, cell, styles.CurrencyStyle)
		} else {
			// Apply the trade style (entry/exit background color) for other columns
			fx.SetCellStyle(sheet, cell, cell, tradeStyle)
		}
	}
}

// Helper methods for detailed analysis insights

func (r *DefaultExcelReporter) getProfitFactorInsight(profitFactor float64) string {
	if profitFactor >= 2.0 {
		return "🌟 Excellent profitability"
	} else if profitFactor >= 1.5 {
		return "✅ Good performance"
	} else if profitFactor >= 1.2 {
		return "👌 Acceptable"
	} else {
		return "⚠️ Needs improvement"
	}
}

func (r *DefaultExcelReporter) getSharpeInsight(sharpeRatio float64) string {
	if sharpeRatio >= 1.5 {
		return "🏆 Outstanding risk-adjusted returns"
	} else if sharpeRatio >= 1.0 {
		return "✅ Good risk management"
	} else if sharpeRatio >= 0.5 {
		return "👌 Moderate performance"
	} else {
		return "⚠️ High risk relative to return"
	}
}

func (r *DefaultExcelReporter) getDrawdownInsight(drawdown float64) string {
	if drawdown <= 5.0 {
		return "🛡️ Very low risk"
	} else if drawdown <= 10.0 {
		return "✅ Acceptable risk"
	} else if drawdown <= 20.0 {
		return "⚠️ Moderate risk"
	} else {
		return "🚨 High risk"
	}
}

func (r *DefaultExcelReporter) getCapitalEfficiencyInsight(efficiency float64) string {
	if efficiency >= 80.0 {
		return "💪 Highly efficient capital use"
	} else if efficiency >= 60.0 {
		return "✅ Good capital utilization"
	} else if efficiency >= 40.0 {
		return "👌 Moderate efficiency"
	} else {
		return "💡 Room for more capital deployment"
	}
}

func (r *DefaultExcelReporter) getCycleCompletionInsight(completion float64) string {
	if completion >= 80.0 {
		return "🎯 Excellent cycle completion"
	} else if completion >= 60.0 {
		return "✅ Good TP targeting"
	} else if completion >= 40.0 {
		return "👌 Moderate success"
	} else {
		return "⚠️ Consider TP level adjustments"
	}
}

func (r *DefaultExcelReporter) getCycleDurationInsight(duration float64) string {
	if duration <= 12.0 {
		return "⚡ Very fast cycles"
	} else if duration <= 24.0 {
		return "✅ Efficient timing"
	} else if duration <= 48.0 {
		return "👌 Reasonable duration"
	} else {
		return "⏰ Consider strategy adjustments"
	}
}

func (r *DefaultExcelReporter) getDCAEfficiencyInsight(entriesPerCycle float64) string {
	if entriesPerCycle <= 2.0 {
		return "🎯 Highly efficient entries"
	} else if entriesPerCycle <= 3.0 {
		return "✅ Good entry efficiency"
	} else if entriesPerCycle <= 4.0 {
		return "👌 Moderate DCA usage"
	} else {
		return "💡 Consider entry threshold adjustments"
	}
}

func (r *DefaultExcelReporter) getStrategicRecommendations(results *backtest.BacktestResults, totalCost float64, winRate float64, avgCycleDuration float64) [][]interface{} {
	recommendations := [][]interface{}{
		{"📈 Performance", "Strategy shows consistent profitability", "Maintain current parameters", "Monitor for market regime changes", ""},
		{"⏰ Timing", fmt.Sprintf("Avg %.1fh cycles suggest good timing", avgCycleDuration), "Current TP levels are well-calibrated", "Consider market volatility adjustments", ""},
		{"💰 Capital", fmt.Sprintf("%.1f%% capital efficiency", (totalCost/results.StartBalance)*100), "Good balance between deployment and reserves", "Monitor for position sizing opportunities", ""},
		{"🎯 Risk Management", fmt.Sprintf("%.1f%% max drawdown is controlled", results.MaxDrawdown*100), "DCA approach effectively manages entry risk", "Continue monitoring stop-loss triggers", ""},
	}
	
	if winRate >= 70.0 {
		recommendations = append(recommendations, []interface{}{"🌟 Optimization", "High win rate suggests room for position sizing", "Consider gradually increasing base amounts", "Monitor for over-optimization", ""})
	} else if winRate < 50.0 {
		recommendations = append(recommendations, []interface{}{"⚠️ Warning", "Low win rate requires strategy review", "Consider tightening entry conditions", "Review TP level spacing", ""})
	}
	
	return recommendations
}

// Package-level convenience function
func WriteTradesXLSX(results *backtest.BacktestResults, path string) error {
	reporter := NewDefaultExcelReporter()
	return reporter.WriteTradesXLSX(results, path)
}
