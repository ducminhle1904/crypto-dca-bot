package reporting

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/grid"
	"github.com/ducminhle1904/crypto-dca-bot/internal/strategy"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/config"
	"github.com/xuri/excelize/v2"
)

// GridBacktestResults represents comprehensive results from a grid backtest
type GridBacktestResults struct {
	// Basic Info
	StrategyName   string                        `json:"strategy_name"`
	StartTime      time.Time                     `json:"start_time"`
	EndTime        time.Time                     `json:"end_time"`
	TotalCandles   int                           `json:"total_candles"`
	ProcessingTime time.Duration                 `json:"processing_time"`
	
	// Configuration
	Config         *config.GridConfig            `json:"config"`
	
	// Grid-Specific Data
	GridLevels     []*grid.GridLevel             `json:"grid_levels"`
	GridPositions  []*GridPositionRecord         `json:"grid_positions"`       // Active positions only (for Grid Positions sheet)
	AllPositions   []*GridPositionRecord         `json:"all_positions"`        // All positions (for Trading Timeline)
	DecisionLog    []GridDecisionRecord          `json:"decision_log"`
	
	// Performance Metrics
	InitialBalance      float64 `json:"initial_balance"`
	FinalBalance        float64 `json:"final_balance"`
	TotalReturn         float64 `json:"total_return"`
	TotalTrades         int     `json:"total_trades"`
	SuccessfulTrades    int     `json:"successful_trades"`
	WinRate            float64 `json:"win_rate"`
	TotalRealized      float64 `json:"total_realized"`
	TotalUnrealized    float64 `json:"total_unrealized"`
	MaxConcurrentPositions int `json:"max_concurrent_positions"`
	
	// Grid Analytics
	GridUtilization    float64                   `json:"grid_utilization"`     // Percentage of grids used
	AvgPositionDuration time.Duration           `json:"avg_position_duration"`
	GridPerformance    map[int]*GridLevelStats  `json:"grid_performance"`     // Performance per grid level
	PriceRangeStats    *PriceRangeAnalysis      `json:"price_range_stats"`
}

// GridPositionRecord represents a single grid position for reporting
type GridPositionRecord struct {
	GridLevel      int       `json:"grid_level"`
	Direction      string    `json:"direction"`
	EntryTime      time.Time `json:"entry_time"`
	EntryPrice     float64   `json:"entry_price"`
	Quantity       float64   `json:"quantity"`
	ExitTime       *time.Time `json:"exit_time,omitempty"`
	ExitPrice      *float64  `json:"exit_price,omitempty"`
	RealizedPnL    *float64  `json:"realized_pnl,omitempty"`
	UnrealizedPnL  float64   `json:"unrealized_pnl"`
	Commission     float64   `json:"commission"`
	Status         string    `json:"status"`
	Duration       time.Duration `json:"duration"`
}

// GridDecisionRecord represents a strategy decision for analysis
type GridDecisionRecord struct {
	Timestamp   time.Time              `json:"timestamp"`
	Price       float64                `json:"price"`
	Action      strategy.TradeAction   `json:"action"`
	Amount      float64                `json:"amount"`
	Reason      string                 `json:"reason"`
	GridLevel   int                    `json:"grid_level,omitempty"`
}

// GridLevelStats represents performance statistics for a specific grid level
type GridLevelStats struct {
	Level            int     `json:"level"`
	Price            float64 `json:"price"`
	Direction        string  `json:"direction"`
	TimesTriggered   int     `json:"times_triggered"`
	TotalPnL         float64 `json:"total_pnl"`
	AvgPnL           float64 `json:"avg_pnl"`
	WinCount         int     `json:"win_count"`
	LossCount        int     `json:"loss_count"`
	WinRate          float64 `json:"win_rate"`
	TotalVolume      float64 `json:"total_volume"`
	AvgHoldDuration  time.Duration `json:"avg_hold_duration"`
}

// PriceRangeAnalysis provides analysis of price behavior within the grid
type PriceRangeAnalysis struct {
	MinPriceReached    float64   `json:"min_price_reached"`
	MaxPriceReached    float64   `json:"max_price_reached"`
	PriceRangeUsed     float64   `json:"price_range_used"`     // Percentage of total range used
	TimeAtGridLevels   map[int]time.Duration `json:"time_at_grid_levels"`
	PriceDistribution  map[string]int        `json:"price_distribution"` // Price bins -> count
}

// GridReporter handles grid-specific reporting functionality
type GridReporter struct {
	*DefaultExcelReporter
}

// NewGridReporter creates a new grid reporter
func NewGridReporter() *GridReporter {
	return &GridReporter{
		DefaultExcelReporter: NewDefaultExcelReporter(),
	}
}

// WriteGridReportXLSX creates a comprehensive Excel report for grid trading results
func (r *GridReporter) WriteGridReportXLSX(results *GridBacktestResults, path string) error {
	// Ensure directory exists
	if dir := filepath.Dir(path); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	fx := excelize.NewFile()
	defer fx.Close()

	// Create grid-specific sheets
	const summarySheet = "Grid Summary"
	const levelsSheet = "Grid Levels"
	const positionsSheet = "Grid Positions"
	const heatMapSheet = "Performance Heat Map"
	const timelineSheet = "Trading Timeline"
	
	// Replace default sheet and create additional sheets
	fx.SetSheetName(fx.GetSheetName(0), summarySheet)
	fx.NewSheet(levelsSheet)
	fx.NewSheet(positionsSheet)
	fx.NewSheet(heatMapSheet)
	fx.NewSheet(timelineSheet)

	// Create professional styles
	styles, err := r.createGridExcelStyles(fx)
	if err != nil {
		return fmt.Errorf("failed to create Excel styles: %w", err)
	}

	// Write all sheets
	if err := r.writeGridSummarySheet(fx, summarySheet, results, styles); err != nil {
		return fmt.Errorf("failed to write summary sheet: %w", err)
	}
	
	if err := r.writeGridLevelsSheet(fx, levelsSheet, results, styles); err != nil {
		return fmt.Errorf("failed to write levels sheet: %w", err)
	}
	
	if err := r.writeGridPositionsSheet(fx, positionsSheet, results, styles); err != nil {
		return fmt.Errorf("failed to write positions sheet: %w", err)
	}
	
	if err := r.writeGridHeatMapSheet(fx, heatMapSheet, results, styles); err != nil {
		return fmt.Errorf("failed to write heat map sheet: %w", err)
	}
	
	if err := r.writeGridTimelineSheet(fx, timelineSheet, results, styles); err != nil {
		return fmt.Errorf("failed to write timeline sheet: %w", err)
	}

	// Save workbook
	return fx.SaveAs(path)
}

// createGridExcelStyles creates Excel styles specifically for grid reporting
func (r *GridReporter) createGridExcelStyles(fx *excelize.File) (GridExcelStyles, error) {
	var styles GridExcelStyles
	var err error

	// Header style - blue background
	if styles.HeaderStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"366092"}, Pattern: 1},
		Border: []excelize.Border{
			{Type: "left", Color: "000000", Style: 2},
			{Type: "top", Color: "000000", Style: 2},
			{Type: "bottom", Color: "000000", Style: 2},
			{Type: "right", Color: "000000", Style: 2},
		},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	}); err != nil {
		return styles, err
	}

	// Summary style - green for positive values
	if styles.SummaryPositiveStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"C6EFCE"}, Pattern: 1},
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
		NumFmt: 164, // Currency format
	}); err != nil {
		return styles, err
	}

	// Summary style - red for negative values
	if styles.SummaryNegativeStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "9C0006"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFC7CE"}, Pattern: 1},
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
		NumFmt: 164, // Currency format
	}); err != nil {
		return styles, err
	}

	// Grid level styles
	if styles.LongGridStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "006100"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E8F5E8"}, Pattern: 1},
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
	}); err != nil {
		return styles, err
	}

	if styles.ShortGridStyle, err = fx.NewStyle(&excelize.Style{
		Font: &excelize.Font{Color: "9C0006"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFE8E8"}, Pattern: 1},
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
	}); err != nil {
		return styles, err
	}

	// Heat map styles (for different performance levels)
	if styles.HeatMapColdStyle, err = fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E6F3FF"}, Pattern: 1},
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
	}); err != nil {
		return styles, err
	}

	if styles.HeatMapWarmStyle, err = fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FFE6CC"}, Pattern: 1},
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
	}); err != nil {
		return styles, err
	}

	if styles.HeatMapHotStyle, err = fx.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Color: []string{"FF9999"}, Pattern: 1},
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
	}); err != nil {
		return styles, err
	}

	// Basic styles
	if styles.BaseStyle, err = fx.NewStyle(&excelize.Style{
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
	}); err != nil {
		return styles, err
	}

	if styles.CurrencyStyle, err = fx.NewStyle(&excelize.Style{
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
		NumFmt: 7, // USD Currency format ($1,234.56)
		Alignment: &excelize.Alignment{Horizontal: "right"},
	}); err != nil {
		return styles, err
	}

	if styles.PercentStyle, err = fx.NewStyle(&excelize.Style{
		Border: []excelize.Border{{Type: "left", Color: "000000", Style: 1}, {Type: "top", Color: "000000", Style: 1}, {Type: "bottom", Color: "000000", Style: 1}, {Type: "right", Color: "000000", Style: 1}},
		NumFmt: 10, // Percentage format
	}); err != nil {
		return styles, err
	}

	return styles, nil
}

// GridExcelStyles holds Excel formatting styles for grid reporting
type GridExcelStyles struct {
	HeaderStyle           int
	SummaryPositiveStyle  int
	SummaryNegativeStyle  int
	LongGridStyle        int
	ShortGridStyle       int
	HeatMapColdStyle     int
	HeatMapWarmStyle     int
	HeatMapHotStyle      int
	BaseStyle            int
	CurrencyStyle        int
	PercentStyle         int
}

// AnalyzeGridResults analyzes a GridStrategy and creates comprehensive results
func (r *GridReporter) AnalyzeGridResults(gridStrategy *strategy.GridStrategy, startTime, endTime time.Time, totalCandles int, processingTime time.Duration) *GridBacktestResults {
	stats := gridStrategy.GetStatistics()
	config := gridStrategy.GetConfiguration()
	levels := gridStrategy.GetGridLevels()
	activePositions := gridStrategy.GetActivePositions()
	
	results := &GridBacktestResults{
		StrategyName:   gridStrategy.GetName(),
		StartTime:      startTime,
		EndTime:        endTime,
		TotalCandles:   totalCandles,
		ProcessingTime: processingTime,
		Config:         config,
		GridLevels:     levels,
		
		// Convert statistics
		InitialBalance:         stats["initial_balance"].(float64),
		FinalBalance:           stats["current_balance"].(float64),
		TotalReturn:            stats["total_return"].(float64),
		TotalTrades:            stats["total_trades"].(int),
		SuccessfulTrades:       stats["successful_trades"].(int),
		WinRate:               stats["win_rate"].(float64),
		TotalRealized:         stats["total_realized"].(float64),
		TotalUnrealized:       stats["total_unrealized"].(float64),
		MaxConcurrentPositions: stats["max_concurrent_pos"].(int),
	}
	
	// Convert active positions to records
	results.GridPositions = make([]*GridPositionRecord, 0, len(activePositions))
	for level, pos := range activePositions {
		record := &GridPositionRecord{
			GridLevel:     level,
			Direction:     pos.Direction,
			EntryTime:     pos.EntryTime,
			EntryPrice:    pos.EntryPrice,
			Quantity:      pos.Quantity,
			UnrealizedPnL: pos.UnrealizedPnL,
			Commission:    pos.Commission,
			Status:        pos.Status,
			Duration:      time.Since(pos.EntryTime),
		}
		
		if pos.Status == "closed" && pos.ExitTime != nil {
			record.ExitTime = pos.ExitTime
			record.ExitPrice = pos.ExitPrice
			record.RealizedPnL = pos.RealizedPnL
			record.Duration = pos.ExitTime.Sub(pos.EntryTime)
		}
		
		results.GridPositions = append(results.GridPositions, record)
	}
	
	// Calculate grid utilization
	if len(levels) > 0 {
		results.GridUtilization = float64(len(activePositions)) / float64(len(levels))
	}
	
	// Calculate grid performance statistics
	results.GridPerformance = r.calculateGridPerformance(results.GridPositions, levels)
	
	// Calculate price range analysis
	results.PriceRangeStats = r.calculatePriceRangeStats(config, results.GridPositions)
	
	return results
}

// calculateGridPerformance calculates performance statistics for each grid level
func (r *GridReporter) calculateGridPerformance(positions []*GridPositionRecord, levels []*grid.GridLevel) map[int]*GridLevelStats {
	performance := make(map[int]*GridLevelStats)
	
	// Initialize stats for all levels
	for _, level := range levels {
		performance[level.Level] = &GridLevelStats{
			Level:     level.Level,
			Price:     level.Price,
			Direction: level.Direction,
		}
	}
	
	// Calculate stats from positions
	for _, pos := range positions {
		stats := performance[pos.GridLevel]
		if stats == nil {
			continue
		}
		
		stats.TimesTriggered++
		stats.TotalVolume += pos.Quantity * pos.EntryPrice
		
		if pos.RealizedPnL != nil {
			stats.TotalPnL += *pos.RealizedPnL
			if *pos.RealizedPnL > 0 {
				stats.WinCount++
			} else {
				stats.LossCount++
			}
		}
		
		// Add unrealized PnL for open positions
		stats.TotalPnL += pos.UnrealizedPnL
	}
	
	// Calculate derived metrics
	for _, stats := range performance {
		if stats.TimesTriggered > 0 {
			stats.AvgPnL = stats.TotalPnL / float64(stats.TimesTriggered)
			if stats.WinCount+stats.LossCount > 0 {
				stats.WinRate = float64(stats.WinCount) / float64(stats.WinCount+stats.LossCount)
			}
		}
	}
	
	return performance
}

// calculatePriceRangeStats analyzes price behavior within the grid range
func (r *GridReporter) calculatePriceRangeStats(config *config.GridConfig, positions []*GridPositionRecord) *PriceRangeAnalysis {
	analysis := &PriceRangeAnalysis{
		MinPriceReached: config.UpperBound,
		MaxPriceReached: config.LowerBound,
		TimeAtGridLevels: make(map[int]time.Duration),
		PriceDistribution: make(map[string]int),
	}
	
	// Find actual price range from positions
	for _, pos := range positions {
		if pos.EntryPrice < analysis.MinPriceReached {
			analysis.MinPriceReached = pos.EntryPrice
		}
		if pos.EntryPrice > analysis.MaxPriceReached {
			analysis.MaxPriceReached = pos.EntryPrice
		}
		
		// Track time at grid levels (approximate)
		analysis.TimeAtGridLevels[pos.GridLevel] += pos.Duration
	}
	
	// Calculate percentage of range used
	totalRange := config.UpperBound - config.LowerBound
	usedRange := analysis.MaxPriceReached - analysis.MinPriceReached
	if totalRange > 0 {
		analysis.PriceRangeUsed = usedRange / totalRange
	}
	
	return analysis
}

// writeGridSummarySheet writes the main summary sheet with key metrics
func (r *GridReporter) writeGridSummarySheet(fx *excelize.File, sheet string, results *GridBacktestResults, styles GridExcelStyles) error {
	// Set column widths
	fx.SetColWidth(sheet, "A", "A", 20)  // Metric names
	fx.SetColWidth(sheet, "B", "B", 15)  // Values
	fx.SetColWidth(sheet, "C", "C", 3)   // Spacer
	fx.SetColWidth(sheet, "D", "D", 20)  // Second column metrics
	fx.SetColWidth(sheet, "E", "E", 15)  // Second column values
	
	// Main title
	fx.SetCellValue(sheet, "A1", "🎯 GRID TRADING BACKTEST SUMMARY")
	fx.SetCellStyle(sheet, "A1", "E1", styles.HeaderStyle)
	fx.MergeCell(sheet, "A1", "E1")
	
	// Strategy information
	row := 3
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Strategy Name:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), results.StrategyName)
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Symbol:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), results.Config.Symbol)
	
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Trading Mode:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), results.Config.TradingMode)
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Grid Count:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), len(results.GridLevels))
	
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Price Range:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), fmt.Sprintf("$%.0f - $%.0f", results.Config.LowerBound, results.Config.UpperBound))
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Grid Spacing:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), fmt.Sprintf("%.1f%%", results.Config.GridSpacing))
	
	// Performance section
	row += 2
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "💰 FINANCIAL PERFORMANCE")
	fx.SetCellStyle(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row), styles.HeaderStyle)
	fx.MergeCell(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row))
	
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Initial Balance:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), results.InitialBalance)
	fx.SetCellStyle(sheet, "B"+strconv.Itoa(row), "B"+strconv.Itoa(row), styles.CurrencyStyle)
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Final Balance:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), results.FinalBalance)
	fx.SetCellStyle(sheet, "E"+strconv.Itoa(row), "E"+strconv.Itoa(row), styles.CurrencyStyle)
	
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Total Return:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), results.TotalReturn)
	returnStyle := styles.SummaryPositiveStyle
	if results.TotalReturn < 0 {
		returnStyle = styles.SummaryNegativeStyle
	}
	fx.SetCellStyle(sheet, "B"+strconv.Itoa(row), "B"+strconv.Itoa(row), returnStyle)
	
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Total Realized P&L:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), results.TotalRealized)
	pnlStyle := styles.SummaryPositiveStyle
	if results.TotalRealized < 0 {
		pnlStyle = styles.SummaryNegativeStyle
	}
	fx.SetCellStyle(sheet, "E"+strconv.Itoa(row), "E"+strconv.Itoa(row), pnlStyle)
	
	// Trading statistics
	row += 2
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "📊 TRADING STATISTICS")
	fx.SetCellStyle(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row), styles.HeaderStyle)
	fx.MergeCell(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row))
	
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Total Trades:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), results.TotalTrades)
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Successful Trades:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), results.SuccessfulTrades)
	
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Win Rate:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), results.WinRate)
	fx.SetCellStyle(sheet, "B"+strconv.Itoa(row), "B"+strconv.Itoa(row), styles.PercentStyle)
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Max Concurrent:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), results.MaxConcurrentPositions)
	
	// Grid-specific metrics
	row += 2
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "🎯 GRID METRICS")
	fx.SetCellStyle(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row), styles.HeaderStyle)
	fx.MergeCell(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row))
	
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Grid Levels:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), len(results.GridLevels))
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Grid Utilization:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), results.GridUtilization)
	fx.SetCellStyle(sheet, "E"+strconv.Itoa(row), "E"+strconv.Itoa(row), styles.PercentStyle)
	
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Active Positions:")
	fx.SetCellValue(sheet, "B"+strconv.Itoa(row), len(results.GridPositions))
	fx.SetCellValue(sheet, "D"+strconv.Itoa(row), "Total Candles:")
	fx.SetCellValue(sheet, "E"+strconv.Itoa(row), results.TotalCandles)
	
	// Exchange constraints info if available
	if results.Config.UseExchangeConstraints {
		row += 2
		fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "🔗 EXCHANGE CONSTRAINTS")
		fx.SetCellStyle(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row), styles.HeaderStyle)
		fx.MergeCell(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row))
		
		row++
		fx.SetCellValue(sheet, "A"+strconv.Itoa(row), results.Config.GetExchangeInfo())
		fx.MergeCell(sheet, "A"+strconv.Itoa(row), "E"+strconv.Itoa(row))
	}
	
	return nil
}

// writeGridLevelsSheet writes detailed information about each grid level
func (r *GridReporter) writeGridLevelsSheet(fx *excelize.File, sheet string, results *GridBacktestResults, styles GridExcelStyles) error {
	// Set column widths
	fx.SetColWidth(sheet, "A", "A", 8)   // Level
	fx.SetColWidth(sheet, "B", "B", 12)  // Price
	fx.SetColWidth(sheet, "C", "C", 10)  // Direction
	fx.SetColWidth(sheet, "D", "D", 12)  // Times Triggered
	fx.SetColWidth(sheet, "E", "E", 12)  // Total P&L
	fx.SetColWidth(sheet, "F", "F", 12)  // Avg P&L
	fx.SetColWidth(sheet, "G", "G", 10)  // Win Rate
	fx.SetColWidth(sheet, "H", "H", 12)  // Total Volume
	fx.SetColWidth(sheet, "I", "I", 10)  // Status
	
	// Headers
	headers := []string{
		"Level", "Price", "Direction", "Times Triggered", "Total P&L", "Avg P&L", "Win Rate", "Total Volume", "Status",
	}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(sheet, cell, header)
		fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle)
	}
	
	// Sort grid levels by level number
	sortedLevels := make([]*grid.GridLevel, len(results.GridLevels))
	copy(sortedLevels, results.GridLevels)
	sort.Slice(sortedLevels, func(i, j int) bool {
		return sortedLevels[i].Level < sortedLevels[j].Level
	})
	
	// Write grid level data
	for row, level := range sortedLevels {
		rowNum := row + 2
		
		fx.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), level.Level)
		fx.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), level.Price)
		fx.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), level.Direction)
		
		// Get performance data
		if perf, exists := results.GridPerformance[level.Level]; exists {
			fx.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), perf.TimesTriggered)
			fx.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), perf.TotalPnL)
			fx.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), perf.AvgPnL)
			fx.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), perf.WinRate)
			fx.SetCellValue(sheet, fmt.Sprintf("H%d", rowNum), perf.TotalVolume)
		}
		
		// Status and styling
		status := "Unused"
		rowStyle := styles.BaseStyle
		if level.IsActive {
			status = "Active"
			if level.Direction == "long" {
				rowStyle = styles.LongGridStyle
			} else {
				rowStyle = styles.ShortGridStyle
			}
		} else if perf, exists := results.GridPerformance[level.Level]; exists && perf.TimesTriggered > 0 {
			status = "Used"
		}
		
		fx.SetCellValue(sheet, fmt.Sprintf("I%d", rowNum), status)
		
		// Apply row styling
		for col := 1; col <= 9; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowNum)
			fx.SetCellStyle(sheet, cell, cell, rowStyle)
		}
		
		// Apply currency formatting to P&L columns
		fx.SetCellStyle(sheet, fmt.Sprintf("E%d", rowNum), fmt.Sprintf("E%d", rowNum), styles.CurrencyStyle)
		fx.SetCellStyle(sheet, fmt.Sprintf("F%d", rowNum), fmt.Sprintf("F%d", rowNum), styles.CurrencyStyle)
		fx.SetCellStyle(sheet, fmt.Sprintf("G%d", rowNum), fmt.Sprintf("G%d", rowNum), styles.PercentStyle)
		fx.SetCellStyle(sheet, fmt.Sprintf("H%d", rowNum), fmt.Sprintf("H%d", rowNum), styles.CurrencyStyle)
	}
	
	return nil
}

// writeGridPositionsSheet writes detailed information about individual positions
func (r *GridReporter) writeGridPositionsSheet(fx *excelize.File, sheet string, results *GridBacktestResults, styles GridExcelStyles) error {
	// Set column widths
	fx.SetColWidth(sheet, "A", "A", 8)   // Grid Level
	fx.SetColWidth(sheet, "B", "B", 10)  // Direction
	fx.SetColWidth(sheet, "C", "C", 18)  // Entry Time
	fx.SetColWidth(sheet, "D", "D", 12)  // Entry Price
	fx.SetColWidth(sheet, "E", "E", 10)  // Quantity
	fx.SetColWidth(sheet, "F", "F", 18)  // Exit Time
	fx.SetColWidth(sheet, "G", "G", 12)  // Exit Price
	fx.SetColWidth(sheet, "H", "H", 12)  // Duration
	fx.SetColWidth(sheet, "I", "I", 12)  // P&L
	fx.SetColWidth(sheet, "J", "J", 12)  // Commission
	fx.SetColWidth(sheet, "K", "K", 10)  // Status
	
	// Headers
	headers := []string{
		"Grid Level", "Direction", "Entry Time", "Entry Price", "Quantity", 
		"Exit Time", "Exit Price", "Duration", "P&L", "Commission", "Status",
	}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		fx.SetCellValue(sheet, cell, header)
		fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle)
	}
	
	// Sort positions by entry time
	sortedPositions := make([]*GridPositionRecord, len(results.GridPositions))
	copy(sortedPositions, results.GridPositions)
	sort.Slice(sortedPositions, func(i, j int) bool {
		return sortedPositions[i].EntryTime.Before(sortedPositions[j].EntryTime)
	})
	
	// Write position data
	for row, pos := range sortedPositions {
		rowNum := row + 2
		
		fx.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), pos.GridLevel)
		fx.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), pos.Direction)
		fx.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), pos.EntryTime.Format("2006-01-02 15:04:05"))
		fx.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), pos.EntryPrice)
		fx.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), pos.Quantity)
		
		// Exit information
		if pos.ExitTime != nil {
			fx.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), pos.ExitTime.Format("2006-01-02 15:04:05"))
		} else {
			fx.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), "-")
		}
		
		if pos.ExitPrice != nil {
			fx.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), *pos.ExitPrice)
		} else {
			fx.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), "-")
		}
		
		// Duration
		durationHours := pos.Duration.Hours()
		fx.SetCellValue(sheet, fmt.Sprintf("H%d", rowNum), fmt.Sprintf("%.1fh", durationHours))
		
		// P&L
		pnl := pos.UnrealizedPnL
		if pos.RealizedPnL != nil {
			pnl = *pos.RealizedPnL
		}
		fx.SetCellValue(sheet, fmt.Sprintf("I%d", rowNum), pnl)
		
		fx.SetCellValue(sheet, fmt.Sprintf("J%d", rowNum), pos.Commission)
		fx.SetCellValue(sheet, fmt.Sprintf("K%d", rowNum), pos.Status)
		
		// Apply styling based on direction and P&L
		rowStyle := styles.BaseStyle
		if pos.Direction == "long" {
			rowStyle = styles.LongGridStyle
		} else if pos.Direction == "short" {
			rowStyle = styles.ShortGridStyle
		}
		
		// Apply row styling
		for col := 1; col <= 11; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowNum)
			fx.SetCellStyle(sheet, cell, cell, rowStyle)
		}
		
		// Currency formatting
		fx.SetCellStyle(sheet, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("D%d", rowNum), styles.CurrencyStyle)
		fx.SetCellStyle(sheet, fmt.Sprintf("G%d", rowNum), fmt.Sprintf("G%d", rowNum), styles.CurrencyStyle)
		fx.SetCellStyle(sheet, fmt.Sprintf("I%d", rowNum), fmt.Sprintf("I%d", rowNum), styles.CurrencyStyle)
		fx.SetCellStyle(sheet, fmt.Sprintf("J%d", rowNum), fmt.Sprintf("J%d", rowNum), styles.CurrencyStyle)
	}
	
	return nil
}

// writeGridHeatMapSheet creates a performance heat map visualization
func (r *GridReporter) writeGridHeatMapSheet(fx *excelize.File, sheet string, results *GridBacktestResults, styles GridExcelStyles) error {
	fx.SetCellValue(sheet, "A1", "🔥 GRID PERFORMANCE HEAT MAP")
	fx.SetCellStyle(sheet, "A1", "F1", styles.HeaderStyle)
	fx.MergeCell(sheet, "A1", "F1")
	
	// Create a simple heat map showing grid level performance
	headers := []string{"Grid Level", "Price", "Direction", "Times Used", "Total P&L", "Performance Score"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		fx.SetCellValue(sheet, cell, header)
		fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle)
	}
	
	// Calculate performance scores and apply heat map colors
	maxPnL := -math.MaxFloat64
	minPnL := math.MaxFloat64
	
	// Find min/max P&L for normalization
	for _, perf := range results.GridPerformance {
		if perf.TotalPnL > maxPnL {
			maxPnL = perf.TotalPnL
		}
		if perf.TotalPnL < minPnL {
			minPnL = perf.TotalPnL
		}
	}
	
	// Sort levels by level number
	var sortedLevels []int
	for level := range results.GridPerformance {
		sortedLevels = append(sortedLevels, level)
	}
	sort.Ints(sortedLevels)
	
	row := 4
	for _, level := range sortedLevels {
		perf := results.GridPerformance[level]
		
		fx.SetCellValue(sheet, fmt.Sprintf("A%d", row), perf.Level)
		fx.SetCellValue(sheet, fmt.Sprintf("B%d", row), perf.Price)
		fx.SetCellValue(sheet, fmt.Sprintf("C%d", row), perf.Direction)
		fx.SetCellValue(sheet, fmt.Sprintf("D%d", row), perf.TimesTriggered)
		fx.SetCellValue(sheet, fmt.Sprintf("E%d", row), perf.TotalPnL)
		
		// Calculate performance score (0-100)
		var score float64
		if maxPnL > minPnL {
			score = ((perf.TotalPnL - minPnL) / (maxPnL - minPnL)) * 100
		}
		fx.SetCellValue(sheet, fmt.Sprintf("F%d", row), score)
		
		// Apply heat map coloring based on performance
		heatStyle := styles.HeatMapColdStyle
		if score > 66 {
			heatStyle = styles.HeatMapHotStyle
		} else if score > 33 {
			heatStyle = styles.HeatMapWarmStyle
		}
		
		// Apply heat map style to the entire row
		for col := 1; col <= 6; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			fx.SetCellStyle(sheet, cell, cell, heatStyle)
		}
		
		row++
	}
	
	// Add legend
	row += 2
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Legend:")
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Cold (Low Performance)")
	fx.SetCellStyle(sheet, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), styles.HeatMapColdStyle)
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Warm (Medium Performance)")
	fx.SetCellStyle(sheet, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), styles.HeatMapWarmStyle)
	row++
	fx.SetCellValue(sheet, "A"+strconv.Itoa(row), "Hot (High Performance)")
	fx.SetCellStyle(sheet, "A"+strconv.Itoa(row), "B"+strconv.Itoa(row), styles.HeatMapHotStyle)
	
	return nil
}

// writeGridTimelineSheet creates a timeline view of grid trading activity
func (r *GridReporter) writeGridTimelineSheet(fx *excelize.File, sheet string, results *GridBacktestResults, styles GridExcelStyles) error {
	fx.SetCellValue(sheet, "A1", "📈 TRADING TIMELINE")
	fx.SetCellStyle(sheet, "A1", "H1", styles.HeaderStyle)
	fx.MergeCell(sheet, "A1", "H1")
	
	// Create chronological timeline of all grid activities
	headers := []string{"Time", "Event Type", "Grid Level", "Price", "Direction", "Amount", "P&L", "Note"}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 3)
		fx.SetCellValue(sheet, cell, header)
		fx.SetCellStyle(sheet, cell, cell, styles.HeaderStyle)
	}
	
	// Create timeline events
	type TimelineEvent struct {
		Time      time.Time
		EventType string
		Level     int
		Price     float64
		Direction string
		Amount    float64
		PnL       float64
		Note      string
	}
	
	var events []TimelineEvent
	
	// Add position events from ALL positions (both active and closed)
	positionsForTimeline := results.AllPositions
	if len(positionsForTimeline) == 0 {
		// Fallback to GridPositions if AllPositions is not populated
		positionsForTimeline = results.GridPositions
	}
	
	for _, pos := range positionsForTimeline {
		// Entry event
		events = append(events, TimelineEvent{
			Time:      pos.EntryTime,
			EventType: "Position Open",
			Level:     pos.GridLevel,
			Price:     pos.EntryPrice,
			Direction: pos.Direction,
			Amount:    pos.Quantity * pos.EntryPrice,
			PnL:       -pos.Commission, // Commission paid on entry
			Note:      fmt.Sprintf("Opened %s position", pos.Direction),
		})
		
		// Exit event (if closed)
		if pos.ExitTime != nil && pos.ExitPrice != nil && pos.RealizedPnL != nil {
			events = append(events, TimelineEvent{
				Time:      *pos.ExitTime,
				EventType: "Position Close",
				Level:     pos.GridLevel,
				Price:     *pos.ExitPrice,
				Direction: pos.Direction,
				Amount:    pos.Quantity * (*pos.ExitPrice),
				PnL:       *pos.RealizedPnL,
				Note:      "Position closed at profit target",
			})
		}
	}
	
	// Sort events chronologically
	sort.Slice(events, func(i, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})
	
	// Write timeline events
	for row, event := range events {
		rowNum := row + 4
		
		fx.SetCellValue(sheet, fmt.Sprintf("A%d", rowNum), event.Time.Format("2006-01-02 15:04"))
		fx.SetCellValue(sheet, fmt.Sprintf("B%d", rowNum), event.EventType)
		fx.SetCellValue(sheet, fmt.Sprintf("C%d", rowNum), event.Level)
		fx.SetCellValue(sheet, fmt.Sprintf("D%d", rowNum), event.Price)
		fx.SetCellValue(sheet, fmt.Sprintf("E%d", rowNum), event.Direction)
		fx.SetCellValue(sheet, fmt.Sprintf("F%d", rowNum), event.Amount)
		fx.SetCellValue(sheet, fmt.Sprintf("G%d", rowNum), event.PnL)
		fx.SetCellValue(sheet, fmt.Sprintf("H%d", rowNum), event.Note)
		
		// Style based on event type
		rowStyle := styles.BaseStyle
		if event.EventType == "Position Open" {
			if event.Direction == "long" {
				rowStyle = styles.LongGridStyle
			} else {
				rowStyle = styles.ShortGridStyle
			}
		}
		
		// Apply row styling
		for col := 1; col <= 8; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, rowNum)
			fx.SetCellStyle(sheet, cell, cell, rowStyle)
		}
		
		// Currency formatting
		fx.SetCellStyle(sheet, fmt.Sprintf("D%d", rowNum), fmt.Sprintf("D%d", rowNum), styles.CurrencyStyle)
		fx.SetCellStyle(sheet, fmt.Sprintf("F%d", rowNum), fmt.Sprintf("F%d", rowNum), styles.CurrencyStyle)
		fx.SetCellStyle(sheet, fmt.Sprintf("G%d", rowNum), fmt.Sprintf("G%d", rowNum), styles.CurrencyStyle)
	}
	
	return nil
}

// WriteGridReportCSV creates detailed CSV exports for grid trading results
func (r *GridReporter) WriteGridReportCSV(results *GridBacktestResults, basePath string) error {
	// Ensure directory exists
	if dir := filepath.Dir(basePath); dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	
	// Create different CSV files for different aspects
	if err := r.writeGridSummaryCSV(results, basePath+"_summary.csv"); err != nil {
		return fmt.Errorf("failed to write summary CSV: %w", err)
	}
	
	if err := r.writeGridPositionsCSV(results, basePath+"_positions.csv"); err != nil {
		return fmt.Errorf("failed to write positions CSV: %w", err)
	}
	
	if err := r.writeGridLevelsCSV(results, basePath+"_levels.csv"); err != nil {
		return fmt.Errorf("failed to write levels CSV: %w", err)
	}
	
	return nil
}

// writeGridSummaryCSV writes a summary CSV with key metrics
func (r *GridReporter) writeGridSummaryCSV(results *GridBacktestResults, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Write summary data as key-value pairs
	data := [][]string{
		{"Metric", "Value"},
		{"Strategy Name", results.StrategyName},
		{"Symbol", results.Config.Symbol},
		{"Trading Mode", results.Config.TradingMode},
		{"Start Time", results.StartTime.Format("2006-01-02 15:04:05")},
		{"End Time", results.EndTime.Format("2006-01-02 15:04:05")},
		{"Total Candles", fmt.Sprintf("%d", results.TotalCandles)},
		{"Processing Time", results.ProcessingTime.String()},
		{"Initial Balance", fmt.Sprintf("%.2f", results.InitialBalance)},
		{"Final Balance", fmt.Sprintf("%.2f", results.FinalBalance)},
		{"Total Return", fmt.Sprintf("%.4f", results.TotalReturn)},
		{"Total Return %", fmt.Sprintf("%.2f%%", results.TotalReturn*100)},
		{"Total Trades", fmt.Sprintf("%d", results.TotalTrades)},
		{"Successful Trades", fmt.Sprintf("%d", results.SuccessfulTrades)},
		{"Win Rate", fmt.Sprintf("%.2f%%", results.WinRate*100)},
		{"Total Realized P&L", fmt.Sprintf("%.2f", results.TotalRealized)},
		{"Total Unrealized P&L", fmt.Sprintf("%.2f", results.TotalUnrealized)},
		{"Max Concurrent Positions", fmt.Sprintf("%d", results.MaxConcurrentPositions)},
		{"Grid Levels", fmt.Sprintf("%d", len(results.GridLevels))},
		{"Grid Utilization", fmt.Sprintf("%.2f%%", results.GridUtilization*100)},
		{"Active Positions", fmt.Sprintf("%d", len(results.GridPositions))},
	}
	
	// Add exchange constraints if available
	if results.Config.UseExchangeConstraints {
		data = append(data, []string{"Exchange", results.Config.ExchangeName})
		data = append(data, []string{"Min Order Qty", fmt.Sprintf("%.6f", results.Config.MinOrderQty)})
		data = append(data, []string{"Qty Step", fmt.Sprintf("%.6f", results.Config.QtyStep)})
		data = append(data, []string{"Min Notional", fmt.Sprintf("%.2f", results.Config.MinNotional)})
		data = append(data, []string{"Max Leverage", fmt.Sprintf("%.1f", results.Config.MaxLeverage)})
	}
	
	return writer.WriteAll(data)
}

// writeGridPositionsCSV writes detailed position data to CSV
func (r *GridReporter) writeGridPositionsCSV(results *GridBacktestResults, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Headers
	headers := []string{
		"Grid Level", "Direction", "Entry Time", "Entry Price", "Quantity",
		"Exit Time", "Exit Price", "Duration Hours", "Realized P&L", 
		"Unrealized P&L", "Commission", "Status", "ROI %",
	}
	
	if err := writer.Write(headers); err != nil {
		return err
	}
	
	// Sort positions by entry time
	sortedPositions := make([]*GridPositionRecord, len(results.GridPositions))
	copy(sortedPositions, results.GridPositions)
	sort.Slice(sortedPositions, func(i, j int) bool {
		return sortedPositions[i].EntryTime.Before(sortedPositions[j].EntryTime)
	})
	
	// Write position data
	for _, pos := range sortedPositions {
		exitTime := "-"
		if pos.ExitTime != nil {
			exitTime = pos.ExitTime.Format("2006-01-02 15:04:05")
		}
		
		exitPrice := "-"
		if pos.ExitPrice != nil {
			exitPrice = fmt.Sprintf("%.2f", *pos.ExitPrice)
		}
		
		realizedPnL := "-"
		if pos.RealizedPnL != nil {
			realizedPnL = fmt.Sprintf("%.2f", *pos.RealizedPnL)
		}
		
		// Calculate ROI
		investment := pos.Quantity * pos.EntryPrice
		totalPnL := pos.UnrealizedPnL
		if pos.RealizedPnL != nil {
			totalPnL = *pos.RealizedPnL
		}
		roi := (totalPnL / investment) * 100
		
		record := []string{
			fmt.Sprintf("%d", pos.GridLevel),
			pos.Direction,
			pos.EntryTime.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.2f", pos.EntryPrice),
			fmt.Sprintf("%.6f", pos.Quantity),
			exitTime,
			exitPrice,
			fmt.Sprintf("%.1f", pos.Duration.Hours()),
			realizedPnL,
			fmt.Sprintf("%.2f", pos.UnrealizedPnL),
			fmt.Sprintf("%.2f", pos.Commission),
			pos.Status,
			fmt.Sprintf("%.2f%%", roi),
		}
		
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	
	return nil
}

// writeGridLevelsCSV writes grid level performance data to CSV
func (r *GridReporter) writeGridLevelsCSV(results *GridBacktestResults, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	writer := csv.NewWriter(file)
	defer writer.Flush()
	
	// Headers
	headers := []string{
		"Level", "Price", "Direction", "Times Triggered", "Total P&L", 
		"Avg P&L", "Win Count", "Loss Count", "Win Rate %", "Total Volume", "Status",
	}
	
	if err := writer.Write(headers); err != nil {
		return err
	}
	
	// Sort grid levels by level number
	sortedLevels := make([]*grid.GridLevel, len(results.GridLevels))
	copy(sortedLevels, results.GridLevels)
	sort.Slice(sortedLevels, func(i, j int) bool {
		return sortedLevels[i].Level < sortedLevels[j].Level
	})
	
	// Write level data
	for _, level := range sortedLevels {
		perf := results.GridPerformance[level.Level]
		
		status := "Unused"
		if level.IsActive {
			status = "Active"
		} else if perf != nil && perf.TimesTriggered > 0 {
			status = "Used"
		}
		
		// Default values if no performance data
		timesTriggered := 0
		totalPnL := 0.0
		avgPnL := 0.0
		winCount := 0
		lossCount := 0
		winRate := 0.0
		totalVolume := 0.0
		
		if perf != nil {
			timesTriggered = perf.TimesTriggered
			totalPnL = perf.TotalPnL
			avgPnL = perf.AvgPnL
			winCount = perf.WinCount
			lossCount = perf.LossCount
			winRate = perf.WinRate * 100
			totalVolume = perf.TotalVolume
		}
		
		record := []string{
			fmt.Sprintf("%d", level.Level),
			fmt.Sprintf("%.2f", level.Price),
			level.Direction,
			fmt.Sprintf("%d", timesTriggered),
			fmt.Sprintf("%.2f", totalPnL),
			fmt.Sprintf("%.2f", avgPnL),
			fmt.Sprintf("%d", winCount),
			fmt.Sprintf("%d", lossCount),
			fmt.Sprintf("%.2f%%", winRate),
			fmt.Sprintf("%.2f", totalVolume),
			status,
		}
		
		if err := writer.Write(record); err != nil {
			return err
		}
	}
	
	return nil
}
