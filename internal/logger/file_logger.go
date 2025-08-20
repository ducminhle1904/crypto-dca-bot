package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Logger represents a file logger for trading activities
type Logger struct {
	symbol     string
	interval   string
	logFile    *os.File
	logger     *log.Logger
	mu         sync.Mutex
	logDir     string
}

// LogLevel represents different types of log entries
type LogLevel string

const (
	LogLevelInfo    LogLevel = "INFO"
	LogLevelWarning LogLevel = "WARN"
	LogLevelError   LogLevel = "ERROR"
	LogLevelTrade   LogLevel = "TRADE"
	LogLevelStatus  LogLevel = "STATUS"
)

// NewLogger creates a new file logger for the specified symbol and interval
func NewLogger(symbol, interval string) (*Logger, error) {
	// Create log directory if it doesn't exist
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create log filename with timestamp
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s_%s_%s.log", symbol, interval, timestamp)
	logPath := filepath.Join(logDir, filename)

	// Open or create log file
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	// Create logger with timestamp and no prefix (we'll add our own formatting)
	logger := log.New(file, "", 0)

	l := &Logger{
		symbol:   symbol,
		interval: interval,
		logFile:  file,
		logger:   logger,
		logDir:   logDir,
	}

	// Write session start header
	l.writeSessionHeader()

	return l, nil
}

// writeSessionHeader writes a session start header to the log
func (l *Logger) writeSessionHeader() {
	l.mu.Lock()
	defer l.mu.Unlock()

	header := fmt.Sprintf(`
================================================================================
🚀 DCA TRADING SESSION STARTED
================================================================================
Symbol: %s | Interval: %s
Started: %s
Log File: %s_%s_%s.log
================================================================================
`, l.symbol, l.interval, time.Now().Format("2006-01-02 15:04:05"), 
	l.symbol, l.interval, time.Now().Format("2006-01-02"))

	l.logger.Print(header)
}

// Log writes a formatted log entry with the specified level
func (l *Logger) Log(level LogLevel, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	message := fmt.Sprintf(format, args...)
	logEntry := fmt.Sprintf("[%s] [%s] %s", timestamp, level, message)
	
	l.logger.Println(logEntry)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.Log(LogLevelInfo, format, args...)
}

// Warning logs a warning message
func (l *Logger) Warning(format string, args ...interface{}) {
	l.Log(LogLevelWarning, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.Log(LogLevelError, format, args...)
}

// Trade logs a trading action
func (l *Logger) Trade(format string, args ...interface{}) {
	l.Log(LogLevelTrade, format, args...)
}

// Status logs market status information
func (l *Logger) Status(format string, args ...interface{}) {
	l.Log(LogLevelStatus, format, args...)
}

// LogMarketStatus logs comprehensive market status
func (l *Logger) LogMarketStatus(currentPrice float64, action string, balance float64, position float64, avgPrice float64, dcaLevel int, exchangePnL string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	statusLog := fmt.Sprintf(`
[%s] [STATUS] ==================== MARKET STATUS ====================
💰 Current Price: $%.2f | Action: %s
💼 Balance: $%.2f | Position Value: $%.2f
📈 Entry Price: $%.2f | DCA Level: %d`, 
		timestamp, currentPrice, action, balance, position, avgPrice, dcaLevel)

	if position > 0 && avgPrice > 0 {
		priceChangePercent := (currentPrice - avgPrice) / avgPrice * 100
		statusLog += fmt.Sprintf(`
📊 Price Change: %.2f%% | Position Status: ACTIVE`, priceChangePercent)
		
		if exchangePnL != "" {
			statusLog += fmt.Sprintf(`
💹 Unrealized P&L: $%s`, exchangePnL)
		} else {
			currentValue := position * (currentPrice / avgPrice)
			unrealizedPnL := currentValue - position
			statusLog += fmt.Sprintf(`
💹 Unrealized P&L: ~$%.2f (calculated)`, unrealizedPnL)
		}
	} else {
		statusLog += "\n📊 Position Status: NO ACTIVE POSITION"
	}

	statusLog += "\n=========================================================="
	
	l.logger.Println(statusLog)
}

// LogTradeExecution logs trade execution details
func (l *Logger) LogTradeExecution(tradeType string, orderID string, quantity string, price string, value string, dcaLevel int, position float64, avgPrice float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	tradeLog := fmt.Sprintf(`
[%s] [TRADE] ==================== %s EXECUTED ====================
✅ Order ID: %s
📦 Quantity: %s %s
💰 Price: $%s
💵 Value: $%s
🔄 DCA Level: %d
📊 Total Position: $%.2f
📈 Average Entry: $%.2f
=============================================================`, 
		timestamp, tradeType, orderID, quantity, l.symbol, price, value, dcaLevel, position, avgPrice)

	l.logger.Println(tradeLog)
}

// LogCycleCompletion logs cycle completion
func (l *Logger) LogCycleCompletion(exitPrice float64, entryPrice float64, profitPercent float64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	cycleLog := fmt.Sprintf(`
[%s] [TRADE] ==================== CYCLE COMPLETED ====================
🎯 Entry Price: $%.2f
🚪 Exit Price: $%.2f  
📊 Price Change: %.2f%%
🔄 Starting fresh cycle...
==============================================================`, 
		timestamp, entryPrice, exitPrice, profitPercent)

	l.logger.Println(cycleLog)
}

// LogPositionSync logs position synchronization
func (l *Logger) LogPositionSync(positionValue float64, entryPrice float64, size string, unrealizedPnL string) {
	l.Info("Position synced - Size: %s, Value: $%.2f, Entry: $%.2f, PnL: %s", size, positionValue, entryPrice, unrealizedPnL)
}

// LogBalanceSync logs balance synchronization
func (l *Logger) LogBalanceSync(oldBalance, newBalance float64) {
	l.Info("Balance synced: $%.2f -> $%.2f", oldBalance, newBalance)
}

// LogError logs error with context
func (l *Logger) LogError(context string, err error) {
	l.Error("%s: %v", context, err)
}

// LogWarning logs warning with context
func (l *Logger) LogWarning(context string, message string, args ...interface{}) {
	fullMessage := fmt.Sprintf(context+": "+message, args...)
	l.Warning("%s", fullMessage)
}

// Close closes the log file
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.logFile != nil {
		// Write session end header
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		footer := fmt.Sprintf(`
================================================================================
🛑 DCA TRADING SESSION ENDED
================================================================================
Ended: %s
================================================================================

`, timestamp)
		l.logger.Print(footer)
		
		return l.logFile.Close()
	}
	return nil
}

// GetLogPath returns the current log file path
func (l *Logger) GetLogPath() string {
	timestamp := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s_%s_%s.log", l.symbol, l.interval, timestamp)
	return filepath.Join(l.logDir, filename)
}
