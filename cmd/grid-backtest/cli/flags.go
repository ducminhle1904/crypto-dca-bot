package cli

import "flag"

// Flags holds all command-line flag values
type Flags struct {
	ConfigFile     *string
	DataFile       *string
	OutputDir      *string
	Symbol         *string
	Interval       *string
	MaxCandles     *int
	GenerateReport *bool
	Verbose        *bool
	Version        *bool
}

// ParseFlags defines and parses command-line flags
func ParseFlags() *Flags {
	flags := &Flags{
		ConfigFile:     flag.String("config", "", "Path to grid configuration file (required)"),
		DataFile:       flag.String("data", "", "Path to historical data file (optional, auto-detected if not provided)"),
		OutputDir:      flag.String("output", "results", "Output directory for reports"),
		Symbol:         flag.String("symbol", "", "Override symbol from config"),
		Interval:       flag.String("interval", "", "Override interval from config"),
		MaxCandles:     flag.Int("max-candles", 0, "Maximum number of candles to use (0 = use all available data)"),
		GenerateReport: flag.Bool("report", true, "Generate comprehensive Excel report"),
		Verbose:        flag.Bool("verbose", false, "Verbose output"),
		Version:        flag.Bool("version", false, "Show version and exit"),
	}

	flag.Parse()
	return flags
}

// Validate checks if required flags are provided
func (f *Flags) Validate() error {
	if *f.ConfigFile == "" {
		return &ValidationError{
			Field:   "config",
			Message: "config file path is required",
		}
	}
	return nil
}

// ValidationError represents a flag validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
