package common

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	LogLevelError LogLevel = iota
	LogLevelWarn
	LogLevelInfo
	LogLevelDebug
)

// Logger provides structured logging for CLI applications
type Logger struct {
	Level       LogLevel
	ShowEmojis  bool
	ShowColors  bool
	SilentMode  bool
}

// NewLogger creates a new logger with default settings
func NewLogger() *Logger {
	return &Logger{
		Level:      LogLevelInfo,
		ShowEmojis: true,
		ShowColors: true,
		SilentMode: false,
	}
}

// SetSilentMode enables or disables silent mode
func (l *Logger) SetSilentMode(silent bool) {
	l.SilentMode = silent
}

// Header prints a formatted header
func (l *Logger) Header(title string) {
	if l.SilentMode {
		return
	}
	
	emoji := "🎯"
	if !l.ShowEmojis {
		emoji = "***"
	}
	
	fmt.Printf("\n%s %s\n", emoji, strings.ToUpper(title))
	fmt.Printf("%s\n", strings.Repeat("=", len(title)+5))
}

// Section prints a formatted section header
func (l *Logger) Section(title string) {
	if l.SilentMode {
		return
	}
	
	emoji := "📋"
	if !l.ShowEmojis {
		emoji = "---"
	}
	
	fmt.Printf("\n%s %s\n", emoji, title)
	fmt.Printf("%s\n", strings.Repeat("-", len(title)+5))
}

// Info prints an info message
func (l *Logger) Info(format string, args ...interface{}) {
	if l.SilentMode || l.Level < LogLevelInfo {
		return
	}
	
	emoji := "ℹ️"
	if !l.ShowEmojis {
		emoji = "[INFO]"
	}
	
	fmt.Printf("%s  %s\n", emoji, fmt.Sprintf(format, args...))
}

// Error prints an error message
func (l *Logger) Error(format string, args ...interface{}) {
	emoji := "❌"
	if !l.ShowEmojis {
		emoji = "[ERROR]"
	}
	
	fmt.Printf("%s %s\n", emoji, fmt.Sprintf(format, args...))
}

// Success prints a success message
func (l *Logger) Success(format string, args ...interface{}) {
	if l.SilentMode {
		return
	}
	
	emoji := "✅"
	if !l.ShowEmojis {
		emoji = "[SUCCESS]"
	}
	
	fmt.Printf("%s %s\n", emoji, fmt.Sprintf(format, args...))
}

// Warn prints a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	if l.Level < LogLevelWarn {
		return
	}
	
	emoji := "⚠️"
	if !l.ShowEmojis {
		emoji = "[WARN]"
	}
	
	fmt.Printf("%s  %s\n", emoji, fmt.Sprintf(format, args...))
}

// Debug prints a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	if l.Level < LogLevelDebug {
		return
	}
	
	emoji := "🔍"
	if !l.ShowEmojis {
		emoji = "[DEBUG]"
	}
	
	fmt.Printf("%s %s\n", emoji, fmt.Sprintf(format, args...))
}

// Progress prints a progress message
func (l *Logger) Progress(format string, args ...interface{}) {
	if l.SilentMode {
		return
	}
	
	emoji := "🔄"
	if !l.ShowEmojis {
		emoji = "[PROGRESS]"
	}
	
	fmt.Printf("%s %s\n", emoji, fmt.Sprintf(format, args...))
}

// Quiet prints a quiet message (only when not in silent mode)
func (l *Logger) Quiet(format string, args ...interface{}) {
	if !l.SilentMode {
		fmt.Printf("   %s\n", fmt.Sprintf(format, args...))
	}
}

// FileUtils provides file and path utilities
type FileUtils struct{}

// NewFileUtils creates a new file utilities instance
func NewFileUtils() *FileUtils {
	return &FileUtils{}
}

// FileExists checks if a file exists
func (f *FileUtils) FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// EnsureDir ensures a directory exists, creating it if necessary
func (f *FileUtils) EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// ResolvePath resolves a path with smart defaults
func (f *FileUtils) ResolvePath(path, defaultDir, defaultExt string) string {
	if path == "" {
		return ""
	}
	
	// Add default extension if missing
	if defaultExt != "" && !strings.HasSuffix(strings.ToLower(path), defaultExt) {
		path += defaultExt
	}
	
	// Add default directory if no path separators
	if defaultDir != "" && !strings.ContainsAny(path, "/\\") {
		return filepath.Join(defaultDir, path)
	}
	
	return path
}

// GetFileSize returns the size of a file in bytes
func (f *FileUtils) GetFileSize(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// StringUtils provides string manipulation utilities
type StringUtils struct{}

// NewStringUtils creates a new string utilities instance
func NewStringUtils() *StringUtils {
	return &StringUtils{}
}

// ParseDuration parses duration strings with common suffixes
func (s *StringUtils) ParseDuration(str string) (time.Duration, error) {
	str = strings.ToLower(strings.TrimSpace(str))
	
	// Handle day suffix
	if strings.HasSuffix(str, "d") || strings.HasSuffix(str, "days") {
		str = strings.TrimSuffix(str, "days")
		str = strings.TrimSuffix(str, "d")
		days, err := strconv.Atoi(str)
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	
	// Handle week suffix
	if strings.HasSuffix(str, "w") || strings.HasSuffix(str, "weeks") {
		str = strings.TrimSuffix(str, "weeks")
		str = strings.TrimSuffix(str, "w")
		weeks, err := strconv.Atoi(str)
		if err != nil {
			return 0, err
		}
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}
	
	// Fall back to standard parsing
	return time.ParseDuration(str)
}

// FormatDuration formats a duration in a human-readable way
func (s *StringUtils) FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1fm", d.Minutes())
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%.1fh", d.Hours())
	}
	return fmt.Sprintf("%.1fd", d.Hours()/24)
}

// Truncate truncates a string to a maximum length
func (s *StringUtils) Truncate(str string, maxLen int) string {
	if len(str) <= maxLen {
		return str
	}
	if maxLen <= 3 {
		return str[:maxLen]
	}
	return str[:maxLen-3] + "..."
}

// PadRight pads a string to the right with spaces
func (s *StringUtils) PadRight(str string, length int) string {
	if len(str) >= length {
		return str
	}
	return str + strings.Repeat(" ", length-len(str))
}

// PadLeft pads a string to the left with spaces
func (s *StringUtils) PadLeft(str string, length int) string {
	if len(str) >= length {
		return str
	}
	return strings.Repeat(" ", length-len(str)) + str
}

// EnvLoader provides environment loading utilities
type EnvLoader struct {
	logger *Logger
}

// NewEnvLoader creates a new environment loader
func NewEnvLoader(logger *Logger) *EnvLoader {
	return &EnvLoader{logger: logger}
}

// LoadEnvFile loads environment variables from a file
func (e *EnvLoader) LoadEnvFile(path string) error {
	if path == "" {
		path = ".env"
	}
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		e.logger.Warn("Environment file %s not found, using system environment", path)
		return nil
	}
	
	if err := godotenv.Load(path); err != nil {
		e.logger.Warn("Could not load environment file %s: %v", path, err)
		return err
	}
	
	e.logger.Debug("Environment loaded from %s", path)
	return nil
}

// GetEnvWithDefault gets an environment variable with a default value
func (e *EnvLoader) GetEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// RequireEnv gets a required environment variable or panics
func (e *EnvLoader) RequireEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Required environment variable %s is not set", key)
	}
	return value
}

// ValidateRequiredEnvVars validates that all required environment variables are set
func (e *EnvLoader) ValidateRequiredEnvVars(keys []string) error {
	missing := []string{}
	
	for _, key := range keys {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}
	
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	
	return nil
}

// FormatUtils provides formatting utilities
type FormatUtils struct{}

// NewFormatUtils creates a new format utilities instance
func NewFormatUtils() *FormatUtils {
	return &FormatUtils{}
}

// FormatFloat formats a float64 with appropriate precision
func (f *FormatUtils) FormatFloat(value float64, precision int) string {
	if precision < 0 {
		// Auto-determine precision based on value
		if value >= 100 {
			precision = 2
		} else if value >= 1 {
			precision = 4
		} else {
			precision = 6
		}
	}
	return fmt.Sprintf("%.*f", precision, value)
}

// FormatPercent formats a decimal as a percentage
func (f *FormatUtils) FormatPercent(value float64, precision int) string {
	return fmt.Sprintf("%.*f%%", precision, value*100)
}

// FormatCurrency formats a value as currency
func (f *FormatUtils) FormatCurrency(value float64) string {
	return fmt.Sprintf("$%.2f", value)
}

// FormatFileSize formats a file size in bytes to human-readable format
func (f *FormatUtils) FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// Global instances for convenience
var (
	DefaultLogger    = NewLogger()
	DefaultFileUtils = NewFileUtils()
	DefaultStrUtils  = NewStringUtils()
	DefaultEnvLoader = NewEnvLoader(DefaultLogger)
	DefaultFormatter = NewFormatUtils()
)

// Convenience functions using global instances
func Header(title string)                           { DefaultLogger.Header(title) }
func Section(title string)                          { DefaultLogger.Section(title) }
func Info(format string, args ...interface{})       { DefaultLogger.Info(format, args...) }
func Error(format string, args ...interface{})      { DefaultLogger.Error(format, args...) }
func Success(format string, args ...interface{})    { DefaultLogger.Success(format, args...) }
func Warn(format string, args ...interface{})       { DefaultLogger.Warn(format, args...) }
func Debug(format string, args ...interface{})      { DefaultLogger.Debug(format, args...) }
func Progress(format string, args ...interface{})   { DefaultLogger.Progress(format, args...) }
func Quiet(format string, args ...interface{})      { DefaultLogger.Quiet(format, args...) }
func SetSilentMode(silent bool)                     { DefaultLogger.SetSilentMode(silent) }

func LoadEnvFile(path string) error                 { return DefaultEnvLoader.LoadEnvFile(path) }
func GetEnvWithDefault(key, def string) string      { return DefaultEnvLoader.GetEnvWithDefault(key, def) }
func RequireEnv(key string) string                  { return DefaultEnvLoader.RequireEnv(key) }

func FileExists(path string) bool                   { return DefaultFileUtils.FileExists(path) }
func EnsureDir(path string) error                   { return DefaultFileUtils.EnsureDir(path) }
func ResolvePath(path, dir, ext string) string      { return DefaultFileUtils.ResolvePath(path, dir, ext) }

func FormatFloat(val float64, prec int) string      { return DefaultFormatter.FormatFloat(val, prec) }
func FormatPercent(val float64, prec int) string    { return DefaultFormatter.FormatPercent(val, prec) }
func FormatCurrency(val float64) string             { return DefaultFormatter.FormatCurrency(val) }
