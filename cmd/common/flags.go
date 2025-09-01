package common

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CommonFlags contains flags that are shared across multiple commands
type CommonFlags struct {
	// Environment and configuration
	EnvFile      *string
	DataRoot     *string
	ConsoleOnly  *bool
	
	// Logging and output
	Verbose      *bool
	Silent       *bool
	NoEmojis     *bool
	NoColors     *bool
	
	// Help and version
	Version      *bool
	Help         *bool
}

// RegisterCommonFlags registers common flags with the default flag set
func RegisterCommonFlags() *CommonFlags {
	return &CommonFlags{
		EnvFile:     flag.String("env", ".env", "Environment file path"),
		DataRoot:    flag.String("data-root", "data", "Data root directory"),
		ConsoleOnly: flag.Bool("console-only", false, "Console output only (no file output)"),
		
		Verbose:     flag.Bool("verbose", false, "Enable verbose output"),
		Silent:      flag.Bool("silent", false, "Enable silent mode (minimal output)"),
		NoEmojis:    flag.Bool("no-emojis", false, "Disable emoji output"),
		NoColors:    flag.Bool("no-colors", false, "Disable colored output"),
		
		Version:     flag.Bool("version", false, "Show version information"),
		Help:        flag.Bool("help", false, "Show help information"),
	}
}

// RegisterCommonFlagsWithPrefix registers common flags with a prefix
func RegisterCommonFlagsWithPrefix(prefix string) *CommonFlags {
	if prefix != "" && !strings.HasSuffix(prefix, "-") {
		prefix += "-"
	}
	
	return &CommonFlags{
		EnvFile:     flag.String(prefix+"env", ".env", "Environment file path"),
		DataRoot:    flag.String(prefix+"data-root", "data", "Data root directory"),
		ConsoleOnly: flag.Bool(prefix+"console-only", false, "Console output only (no file output)"),
		
		Verbose:     flag.Bool(prefix+"verbose", false, "Enable verbose output"),
		Silent:      flag.Bool(prefix+"silent", false, "Enable silent mode (minimal output)"),
		NoEmojis:    flag.Bool(prefix+"no-emojis", false, "Disable emoji output"),
		NoColors:    flag.Bool(prefix+"no-colors", false, "Disable colored output"),
		
		Version:     flag.Bool(prefix+"version", false, "Show version information"),
		Help:        flag.Bool(prefix+"help", false, "Show help information"),
	}
}

// FlagValidator provides flag validation utilities
type FlagValidator struct {
	errors []string
}

// NewFlagValidator creates a new flag validator
func NewFlagValidator() *FlagValidator {
	return &FlagValidator{
		errors: make([]string, 0),
	}
}

// ValidateFloat validates a float flag value
func (v *FlagValidator) ValidateFloat(name string, value float64, min, max float64) *FlagValidator {
	if value < min || value > max {
		v.errors = append(v.errors, fmt.Sprintf("%s must be between %.4f and %.4f, got: %.4f", name, min, max, value))
	}
	return v
}

// ValidateInt validates an int flag value
func (v *FlagValidator) ValidateInt(name string, value int, min, max int) *FlagValidator {
	if value < min || value > max {
		v.errors = append(v.errors, fmt.Sprintf("%s must be between %d and %d, got: %d", name, min, max, value))
	}
	return v
}

// ValidateString validates a string flag value
func (v *FlagValidator) ValidateString(name, value string, minLen, maxLen int) *FlagValidator {
	if len(value) < minLen || len(value) > maxLen {
		v.errors = append(v.errors, fmt.Sprintf("%s length must be between %d and %d characters, got: %d", name, minLen, maxLen, len(value)))
	}
	return v
}

// ValidateChoice validates that a string is one of the allowed choices
func (v *FlagValidator) ValidateChoice(name, value string, choices []string) *FlagValidator {
	for _, choice := range choices {
		if value == choice {
			return v
		}
	}
	v.errors = append(v.errors, fmt.Sprintf("%s must be one of [%s], got: %s", name, strings.Join(choices, ", "), value))
	return v
}

// ValidateFile validates that a file exists
func (v *FlagValidator) ValidateFile(name, path string, required bool) *FlagValidator {
	if path == "" {
		if required {
			v.errors = append(v.errors, fmt.Sprintf("%s is required", name))
		}
		return v
	}
	
	if _, err := os.Stat(path); os.IsNotExist(err) {
		v.errors = append(v.errors, fmt.Sprintf("%s file does not exist: %s", name, path))
	}
	return v
}

// ValidateDirectory validates that a directory exists
func (v *FlagValidator) ValidateDirectory(name, path string, required bool) *FlagValidator {
	if path == "" {
		if required {
			v.errors = append(v.errors, fmt.Sprintf("%s is required", name))
		}
		return v
	}
	
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		v.errors = append(v.errors, fmt.Sprintf("%s directory does not exist: %s", name, path))
	} else if err == nil && !info.IsDir() {
		v.errors = append(v.errors, fmt.Sprintf("%s is not a directory: %s", name, path))
	}
	return v
}

// AddError adds a custom validation error
func (v *FlagValidator) AddError(message string) *FlagValidator {
	v.errors = append(v.errors, message)
	return v
}

// HasErrors returns true if there are validation errors
func (v *FlagValidator) HasErrors() bool {
	return len(v.errors) > 0
}

// GetErrors returns all validation errors
func (v *FlagValidator) GetErrors() []string {
	return v.errors
}

// GetError returns a formatted error message with all validation errors
func (v *FlagValidator) GetError() error {
	if len(v.errors) == 0 {
		return nil
	}
	
	if len(v.errors) == 1 {
		return fmt.Errorf("validation error: %s", v.errors[0])
	}
	
	return fmt.Errorf("validation errors:\n  - %s", strings.Join(v.errors, "\n  - "))
}

// PrintErrors prints all validation errors
func (v *FlagValidator) PrintErrors() {
	if len(v.errors) == 0 {
		return
	}
	
	fmt.Fprintf(os.Stderr, "❌ Flag validation errors:\n")
	for _, err := range v.errors {
		fmt.Fprintf(os.Stderr, "   • %s\n", err)
	}
}

// UsageFormatter provides utilities for formatting flag usage
type UsageFormatter struct {
	AppName        string
	AppDescription string
	Examples       []UsageExample
}

// UsageExample represents a usage example
type UsageExample struct {
	Command     string
	Description string
}

// NewUsageFormatter creates a new usage formatter
func NewUsageFormatter(appName, description string) *UsageFormatter {
	return &UsageFormatter{
		AppName:        appName,
		AppDescription: description,
		Examples:       make([]UsageExample, 0),
	}
}

// AddExample adds a usage example
func (u *UsageFormatter) AddExample(command, description string) *UsageFormatter {
	u.Examples = append(u.Examples, UsageExample{
		Command:     command,
		Description: description,
	})
	return u
}

// PrintUsage prints formatted usage information
func (u *UsageFormatter) PrintUsage() {
	fmt.Printf("%s - %s\n\n", u.AppName, u.AppDescription)
	
	fmt.Printf("USAGE:\n")
	fmt.Printf("  %s [OPTIONS]\n\n", filepath.Base(os.Args[0]))
	
	if len(u.Examples) > 0 {
		fmt.Printf("EXAMPLES:\n")
		for _, example := range u.Examples {
			fmt.Printf("  # %s\n", example.Description)
			fmt.Printf("  %s\n\n", example.Command)
		}
	}
	
	fmt.Printf("OPTIONS:\n")
	flag.PrintDefaults()
}

// PrintShortUsage prints a short usage line
func (u *UsageFormatter) PrintShortUsage() {
	fmt.Printf("Usage: %s [OPTIONS]\n", filepath.Base(os.Args[0]))
	fmt.Printf("Use --help for more information.\n")
}

// FlagGroupPrinter prints flags organized by groups
type FlagGroupPrinter struct {
	groups map[string][]FlagInfo
}

// FlagInfo contains information about a flag
type FlagInfo struct {
	Name         string
	Usage        string
	DefaultValue string
	Required     bool
}

// NewFlagGroupPrinter creates a new flag group printer
func NewFlagGroupPrinter() *FlagGroupPrinter {
	return &FlagGroupPrinter{
		groups: make(map[string][]FlagInfo),
	}
}

// AddFlag adds a flag to a group
func (p *FlagGroupPrinter) AddFlag(group, name, usage, defaultValue string, required bool) {
	if _, exists := p.groups[group]; !exists {
		p.groups[group] = make([]FlagInfo, 0)
	}
	
	p.groups[group] = append(p.groups[group], FlagInfo{
		Name:         name,
		Usage:        usage,
		DefaultValue: defaultValue,
		Required:     required,
	})
}

// PrintGroups prints all flag groups
func (p *FlagGroupPrinter) PrintGroups() {
	groupOrder := []string{
		"Configuration",
		"Strategy Parameters",
		"Analysis Options",
		"Validation",
		"Output",
		"Common",
	}
	
	for _, groupName := range groupOrder {
		if flags, exists := p.groups[groupName]; exists {
			fmt.Printf("\n%s:\n", strings.ToUpper(groupName))
			for _, flagInfo := range flags {
				required := ""
				if flagInfo.Required {
					required = " (required)"
				}
				
				if flagInfo.DefaultValue != "" {
					fmt.Printf("  -%s%s\n        %s (default: %s)\n", 
						flagInfo.Name, required, flagInfo.Usage, flagInfo.DefaultValue)
				} else {
					fmt.Printf("  -%s%s\n        %s\n", 
						flagInfo.Name, required, flagInfo.Usage)
				}
			}
		}
	}
	
	// Print any remaining groups not in the predefined order
	for groupName, flags := range p.groups {
		inOrder := false
		for _, ordered := range groupOrder {
			if groupName == ordered {
				inOrder = true
				break
			}
		}
		
		if !inOrder {
			fmt.Printf("\n%s:\n", strings.ToUpper(groupName))
			for _, flagInfo := range flags {
				required := ""
				if flagInfo.Required {
					required = " (required)"
				}
				
				if flagInfo.DefaultValue != "" {
					fmt.Printf("  -%s%s\n        %s (default: %s)\n", 
						flagInfo.Name, required, flagInfo.Usage, flagInfo.DefaultValue)
				} else {
					fmt.Printf("  -%s%s\n        %s\n", 
						flagInfo.Name, required, flagInfo.Usage)
				}
			}
		}
	}
}

// Utility functions for common flag operations

// ParseAndValidate parses flags and validates common requirements
func ParseAndValidate(validator *FlagValidator) error {
	flag.Parse()
	
	if validator.HasErrors() {
		validator.PrintErrors()
		return validator.GetError()
	}
	
	return nil
}

// CheckHelpAndVersion checks for help and version flags and handles them
func CheckHelpAndVersion(appName string, commonFlags *CommonFlags, formatter *UsageFormatter) bool {
	if *commonFlags.Version {
		PrintVersion(appName)
		return true
	}
	
	if *commonFlags.Help {
		formatter.PrintUsage()
		return true
	}
	
	return false
}

// SetupLogger configures the default logger based on common flags
func SetupLogger(commonFlags *CommonFlags) {
	logger := DefaultLogger
	
	if *commonFlags.Silent {
		logger.SetSilentMode(true)
	}
	
	if *commonFlags.Verbose {
		logger.Level = LogLevelDebug
	}
	
	if *commonFlags.NoEmojis {
		logger.ShowEmojis = false
	}
	
	if *commonFlags.NoColors {
		logger.ShowColors = false
	}
}
