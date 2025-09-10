package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
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

	// Load config file
	botConfig, err := config.LoadLiveBotConfig(*configFile)
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

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for shutdown signal with immediate response
	fmt.Println("📡 Bot is ready. Press Ctrl+C to stop...")
	
	select {
	case sig := <-sigChan:
		fmt.Printf("\n🛑 Shutdown signal (%v) received...\n", sig)
	}

	// Stop the bot gracefully
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


