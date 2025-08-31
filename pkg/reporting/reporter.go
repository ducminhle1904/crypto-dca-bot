package reporting

import (
	"github.com/ducminhle1904/crypto-dca-bot/internal/backtest"
)

// DefaultReporter implements the complete Reporter interface
type DefaultReporter struct {
	console  *DefaultConsoleReporter
	csv      *DefaultCSVReporter
	excel    *DefaultExcelReporter
	json     *DefaultJSONFormatter
	paths    *DefaultPathManager
}

// NewDefaultReporter creates a new default reporter with all functionality
func NewDefaultReporter() *DefaultReporter {
	return &DefaultReporter{
		console: NewDefaultConsoleReporter(),
		csv:     NewDefaultCSVReporter(),
		excel:   NewDefaultExcelReporter(),
		json:    NewDefaultJSONFormatter(),
		paths:   NewDefaultPathManager(),
	}
}

// Console output methods
func (r *DefaultReporter) OutputResults(results *backtest.BacktestResults) {
	r.console.OutputResults(results)
}

func (r *DefaultReporter) OutputResultsWithContext(results *backtest.BacktestResults, symbol, interval string) {
	r.console.OutputResultsWithContext(results, symbol, interval)
}

func (r *DefaultReporter) PrintConfig(config interface{}) {
	r.console.PrintConfig(config)
}

func (r *DefaultReporter) PrintWalkForwardSummary(results interface{}) {
	r.console.PrintWalkForwardSummary(results)
}

// File output methods
func (r *DefaultReporter) WriteTradesCSV(results *backtest.BacktestResults, path string) error {
	return r.csv.WriteTradesCSV(results, path)
}

func (r *DefaultReporter) WriteTradesXLSX(results *backtest.BacktestResults, path string) error {
	return r.excel.WriteTradesXLSX(results, path)
}

func (r *DefaultReporter) WriteBestConfigJSON(config interface{}, path string) error {
	return WriteBestConfigJSON(config, path)
}

// JSON methods
func (r *DefaultReporter) FormatBestConfig(config interface{}) ([]byte, error) {
	return r.json.FormatBestConfig(config)
}

func (r *DefaultReporter) PrintBestConfig(config interface{}) {
	r.json.PrintBestConfig(config)
}

func (r *DefaultReporter) ConvertToNestedConfig(config interface{}) interface{} {
	return r.json.ConvertToNestedConfig(config)
}

// Path management methods
func (r *DefaultReporter) GetDefaultOutputDir(symbol, interval string) string {
	return r.paths.GetDefaultOutputDir(symbol, interval)
}

func (r *DefaultReporter) EnsureDirectoryExists(path string) error {
	return r.paths.EnsureDirectoryExists(path)
}

// ReportingManager provides a high-level interface for all reporting needs
type ReportingManager struct {
	reporter *DefaultReporter
	config   ReportingConfig
}

// NewReportingManager creates a new reporting manager with configuration
func NewReportingManager(config ReportingConfig) *ReportingManager {
	return &ReportingManager{
		reporter: NewDefaultReporter(),
		config:   config,
	}
}

// ReportResults outputs results according to configuration
func (m *ReportingManager) ReportResults(results *backtest.BacktestResults, symbol, interval string) error {
	// Console output
	if m.config.EnableConsole {
		m.reporter.OutputResults(results)
	}

	// File outputs
	if m.config.EnableFiles {
		outputDir := m.reporter.GetDefaultOutputDir(symbol, interval)
		
		if m.config.CSVEnabled {
			csvPath := outputDir + "/trades.csv"
			if err := m.reporter.WriteTradesCSV(results, csvPath); err != nil {
				return err
			}
		}
		
		if m.config.ExcelEnabled {
			xlsxPath := outputDir + "/trades.xlsx"
			if err := m.reporter.WriteTradesXLSX(results, xlsxPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// ReportConfig outputs configuration according to settings
func (m *ReportingManager) ReportConfig(config interface{}, symbol, interval string) error {
	// Console output
	if m.config.EnableConsole {
		m.reporter.PrintBestConfig(config)
	}

	// File output
	if m.config.EnableFiles && m.config.JSONEnabled {
		outputDir := m.reporter.GetDefaultOutputDir(symbol, interval)
		jsonPath := outputDir + "/best.json"
		if err := m.reporter.WriteBestConfigJSON(config, jsonPath); err != nil {
			return err
		}
	}

	return nil
}
