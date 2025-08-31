package reporting

import (
	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
	"github.com/xuri/excelize/v2"
)

// Package reporting provides output generation for trading bot results

// ConsoleReporter defines interface for console output
type ConsoleReporter interface {
	OutputResults(results *backtest.BacktestResults)
	OutputResultsWithContext(results *backtest.BacktestResults, symbol, interval string)
	PrintConfig(config interface{})
	PrintWalkForwardSummary(results interface{})
}

// FileReporter defines interface for file output
type FileReporter interface {
	WriteTradesCSV(results *backtest.BacktestResults, path string) error
	WriteTradesXLSX(results *backtest.BacktestResults, path string) error
	WriteBestConfigJSON(config interface{}, path string) error
}

// ExcelFormatter defines interface for Excel-specific formatting
type ExcelFormatter interface {
	WriteTradeRow(fx *excelize.File, sheet string, row int, values []interface{}, styles ExcelStyles)
	WriteEnhancedTradeRow(fx *excelize.File, sheet string, row int, values []interface{}, styles ExcelStyles, isEntry bool)
	WriteDetailedAnalysis(fx *excelize.File, sheet string, results *backtest.BacktestResults, styles ExcelStyles)
	WriteTimelineAnalysis(fx *excelize.File, sheet string, results *backtest.BacktestResults, styles ExcelStyles)
}

// JSONFormatter defines interface for JSON output
type JSONFormatter interface {
	FormatBestConfig(config interface{}) ([]byte, error)
	PrintBestConfig(config interface{})
	ConvertToNestedConfig(config interface{}) interface{}
}

// PathManager defines interface for output path management
type PathManager interface {
	GetDefaultOutputDir(symbol, interval string) string
	EnsureDirectoryExists(path string) error
}

// Reporter combines all reporting interfaces
type Reporter interface {
	ConsoleReporter
	FileReporter
	JSONFormatter
	PathManager
}

// ExcelStyles holds Excel formatting styles
type ExcelStyles struct {
	HeaderStyle        int
	CurrencyStyle      int
	PercentStyle       int
	BaseStyle          int
	RedPercentStyle    int
	GreenPercentStyle  int
	EntryStyle         int
	ExitStyle          int
	CycleHeaderStyle   int
	SummaryStyle       int
	FinalSummaryStyle  int
}

// ReportingConfig holds configuration for reporting
type ReportingConfig struct {
	EnableConsole    bool
	EnableFiles      bool
	OutputDirectory  string
	ExcelEnabled     bool
	CSVEnabled       bool
	JSONEnabled      bool
}
