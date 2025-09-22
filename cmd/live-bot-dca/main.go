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
		demo         = flag.Bool("demo", true, "Use demo/paper trading (default: true). Set to false for LIVE TRADING with real money!")
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
	
	// Apply demo mode override with clear warnings
	if *demo {
		fmt.Println("🧪 DEMO MODE: Running in paper trading mode")
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
	} else {
		fmt.Println("⚠️  LIVE TRADING MODE: Using real money! Double-check your settings.")
		fmt.Printf("💰 Exchange: %s | Symbol: %s | Base Amount: $%.2f\n", 
			botConfig.Exchange.Name, botConfig.Strategy.Symbol, botConfig.Strategy.BaseAmount)
		fmt.Print("   Continue? (type 'yes' to confirm): ")
		var confirmation string
		fmt.Scanln(&confirmation)
		if strings.ToLower(confirmation) != "yes" {
			log.Fatal("🛑 Live trading cancelled by user")
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
		
		// Validate credentials with enhanced checks
		if config.Exchange.Bybit.APIKey == "" || strings.TrimSpace(config.Exchange.Bybit.APIKey) == "" {
			return fmt.Errorf("bybit API key is required (set in environment or config)")
		}
		if config.Exchange.Bybit.APISecret == "" || strings.TrimSpace(config.Exchange.Bybit.APISecret) == "" {
			return fmt.Errorf("bybit API secret is required (set in environment or config)")
		}
		
		// Check for placeholder values that weren't replaced
		if strings.Contains(config.Exchange.Bybit.APIKey, "${") || 
		   strings.Contains(config.Exchange.Bybit.APISecret, "${") {
			return fmt.Errorf("api credentials contain placeholder values - check environment variables")
		}
		
		// Minimum length validation for API credentials (reasonable assumption)
		if len(strings.TrimSpace(config.Exchange.Bybit.APIKey)) < 10 {
			return fmt.Errorf("bybit API key appears to be invalid (too short)")
		}
		if len(strings.TrimSpace(config.Exchange.Bybit.APISecret)) < 10 {
			return fmt.Errorf("bybit API secret appears to be invalid (too short)")
		}
		
	case "binance":
		if config.Exchange.Binance == nil {
			return fmt.Errorf("binance configuration is missing")
		}
		
		// Set from environment if not already set
		if config.Exchange.Binance.APIKey == "" || config.Exchange.Binance.APIKey == "${BINANCE_API_KEY}" {
			config.Exchange.Binance.APIKey = os.Getenv("BINANCE_API_KEY")
		}
		if config.Exchange.Binance.APISecret == "" || config.Exchange.Binance.APISecret == "${BINANCE_API_SECRET}" {
			config.Exchange.Binance.APISecret = os.Getenv("BINANCE_API_SECRET")
		}
		
		// Validate credentials with enhanced checks
		if config.Exchange.Binance.APIKey == "" || strings.TrimSpace(config.Exchange.Binance.APIKey) == "" {
			return fmt.Errorf("binance API key is required (set in environment or config)")
		}
		if config.Exchange.Binance.APISecret == "" || strings.TrimSpace(config.Exchange.Binance.APISecret) == "" {
			return fmt.Errorf("binance API secret is required (set in environment or config)")
		}
		
		// Check for placeholder values that weren't replaced
		if strings.Contains(config.Exchange.Binance.APIKey, "${") || 
		   strings.Contains(config.Exchange.Binance.APISecret, "${") {
			return fmt.Errorf("api credentials contain placeholder values - check environment variables")
		}
		
		// Minimum length validation for API credentials (reasonable assumption)
		if len(strings.TrimSpace(config.Exchange.Binance.APIKey)) < 10 {
			return fmt.Errorf("binance API key appears to be invalid (too short)")
		}
		if len(strings.TrimSpace(config.Exchange.Binance.APISecret)) < 10 {
			return fmt.Errorf("binance API secret appears to be invalid (too short)")
		}
		
	default:
		return fmt.Errorf("unsupported exchange: %s", config.Exchange.Name)
	}
	
	return nil
}


