package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/ducminhle1904/crypto-dca-bot/internal/bot"
	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/joho/godotenv"
)

func main() {
	var (
		configFile   = flag.String("config", "", "Configuration file (e.g., btc_5m_bybit.json)")
		exchangeName = flag.String("exchange", "", "Exchange name (bybit, binance) - overrides config")
		demo         = flag.Bool("demo", true, "Use demo trading environment - paper trading (default: true)")
		envFile      = flag.String("env", ".env", "Environment file path (default: .env)")
		legacy       = flag.Bool("legacy", false, "Convert legacy config format to new format")
	)
	flag.Parse()

	if *configFile == "" {
		log.Fatal("Please specify a config file with -config flag")
	}

	// Load environment variables from .env file
	if err := loadEnvFile(*envFile); err != nil {
		log.Printf("Warning: Could not load .env file (%v), checking environment variables...", err)
	}

	fmt.Println("🚀 DCA Bot Starting...")

	var botConfig *config.LiveBotConfig
	var err error

	if *legacy {
		// Convert legacy config to new format
		if *exchangeName == "" {
			*exchangeName = "bybit" // Default for legacy configs
		}
		
		fmt.Printf("🔄 Converting legacy config format for %s...\n", *exchangeName)
		botConfig, err = config.MigrateFromLegacyConfig(*configFile, *exchangeName)
		if err != nil {
			log.Fatalf("Failed to migrate legacy config: %v", err)
		}
		
		// Apply demo mode override
		if *demo {
			switch strings.ToLower(*exchangeName) {
			case "bybit":
				if botConfig.Exchange.Bybit != nil {
					botConfig.Exchange.Bybit.Demo = true
					botConfig.Exchange.Bybit.Testnet = false
				}
			case "binance":
				if botConfig.Exchange.Binance != nil {
					botConfig.Exchange.Binance.Demo = false // Binance doesn't have demo
					botConfig.Exchange.Binance.Testnet = true
				}
			}
		}
	} else {
		// Load new config format
		botConfig, err = config.LoadLiveBotConfig(*configFile)
		if err != nil {
			log.Fatalf("Failed to load config: %v", err)
		}
		
		// Apply exchange override if specified
		if *exchangeName != "" {
			botConfig.Exchange.Name = *exchangeName
			fmt.Printf("🔧 Exchange overridden to: %s\n", *exchangeName)
		}
		
		// Apply demo mode override if specified
		if *demo {
			switch strings.ToLower(botConfig.Exchange.Name) {
			case "bybit":
				if botConfig.Exchange.Bybit != nil {
					botConfig.Exchange.Bybit.Demo = true
					botConfig.Exchange.Bybit.Testnet = false
				}
			case "binance":
				if botConfig.Exchange.Binance != nil {
					botConfig.Exchange.Binance.Demo = false // Binance doesn't have demo
					botConfig.Exchange.Binance.Testnet = true
				}
			}
		}
	}

	// Ensure API credentials are set from environment if not in config
	if err := ensureAPICredentials(botConfig); err != nil {
		log.Fatalf("API credentials validation failed: %v", err)
	}

	// Create the modular live bot
	liveBot, err := bot.NewLiveBot(botConfig)
	if err != nil {
		log.Fatalf("Failed to create live bot: %v", err)
	}

	// Start the bot
	if err := liveBot.Start(); err != nil {
		log.Fatalf("Failed to start bot: %v", err)
	}

	// Wait for shutdown signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("\n🛑 Shutdown signal received...")

	liveBot.Stop()
	fmt.Println("✅ Bot stopped successfully")
}

// loadEnvFile loads environment variables from a file
func loadEnvFile(envFile string) error {
	if _, err := os.Stat(envFile); err == nil {
		return godotenv.Load(envFile)
	}
	return fmt.Errorf("env file %s not found", envFile)
}

// ensureAPICredentials ensures API credentials are set from environment variables
func ensureAPICredentials(config *config.LiveBotConfig) error {
	switch strings.ToLower(config.Exchange.Name) {
	case "bybit":
		if config.Exchange.Bybit == nil {
			return fmt.Errorf("Bybit configuration is missing")
		}
		
		// Set from environment if not already set
		if config.Exchange.Bybit.APIKey == "" || config.Exchange.Bybit.APIKey == "${BYBIT_API_KEY}" {
			config.Exchange.Bybit.APIKey = os.Getenv("BYBIT_API_KEY")
		}
		if config.Exchange.Bybit.APISecret == "" || config.Exchange.Bybit.APISecret == "${BYBIT_API_SECRET}" {
			config.Exchange.Bybit.APISecret = os.Getenv("BYBIT_API_SECRET")
		}
		
		// Validate credentials
		if config.Exchange.Bybit.APIKey == "" {
			return fmt.Errorf("BYBIT_API_KEY is required (set in environment or config)")
		}
		if config.Exchange.Bybit.APISecret == "" {
			return fmt.Errorf("BYBIT_API_SECRET is required (set in environment or config)")
		}
		
	case "binance":
		if config.Exchange.Binance == nil {
			return fmt.Errorf("Binance configuration is missing")
		}
		
		// Set from environment if not already set
		if config.Exchange.Binance.APIKey == "" || config.Exchange.Binance.APIKey == "${BINANCE_API_KEY}" {
			config.Exchange.Binance.APIKey = os.Getenv("BINANCE_API_KEY")
		}
		if config.Exchange.Binance.APISecret == "" || config.Exchange.Binance.APISecret == "${BINANCE_API_SECRET}" {
			config.Exchange.Binance.APISecret = os.Getenv("BINANCE_API_SECRET")
		}
		
		// Validate credentials
		if config.Exchange.Binance.APIKey == "" {
			return fmt.Errorf("BINANCE_API_KEY is required (set in environment or config)")
		}
		if config.Exchange.Binance.APISecret == "" {
			return fmt.Errorf("BINANCE_API_SECRET is required (set in environment or config)")
		}
		
	default:
		return fmt.Errorf("unsupported exchange: %s", config.Exchange.Name)
	}
	
	return nil
}

// extractTradingParams extracts trading parameters from legacy config filename
// This is kept for backward compatibility when using -legacy flag
func extractTradingParams(configFile string) (interval string) {
	// Get base filename without extension
	basename := filepath.Base(configFile)
	if ext := filepath.Ext(basename); ext != "" {
		basename = strings.TrimSuffix(basename, ext)
	}

	// Extract interval from filename (e.g., btc_15m.json -> 15m)
	parts := strings.Split(basename, "_")
	if len(parts) >= 2 {
		intervalPart := parts[len(parts)-1] // Last part should be interval
		
		// Convert common intervals to standard format
		switch intervalPart {
		case "1m", "3m", "5m", "15m", "30m":
			return strings.TrimSuffix(intervalPart, "m")
		case "1h":
			return "60"
		case "4h":
			return "240"
		case "1d":
			return "D"
		default:
			// Try to extract number from string like "5m" -> "5"
			if strings.HasSuffix(intervalPart, "m") {
				return strings.TrimSuffix(intervalPart, "m")
			}
		}
	}

	// Default to 5 minutes
	return "5"
}
