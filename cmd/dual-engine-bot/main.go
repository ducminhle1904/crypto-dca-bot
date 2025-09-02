package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/config"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange/adapters"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/orchestrator"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("🚀 Enhanced DCA Bot - Dual Engine System")
	fmt.Println("======================================")

	// Command line flags
	var (
		configFile = flag.String("config", "", "Path to configuration file (required)")
		demoMode   = flag.Bool("demo", false, "Run in demo mode (paper trading)")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		envFile    = flag.String("env", ".env", "Environment file path (default: .env)")
		help       = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	if *configFile == "" {
		fmt.Println("❌ Error: Configuration file is required")
		fmt.Println("Use -help for more information")
		os.Exit(1)
	}

	// Load environment variables from .env file
	if err := loadEnvFile(*envFile); err != nil {
		log.Printf("Warning: Could not load .env file (%v), checking environment variables...", err)
	}

	// Load dual-engine configuration
	fmt.Printf("📝 Loading configuration: %s\n", *configFile)
	dualEngineConfig, err := config.LoadDualEngineConfig(*configFile, "development")
	if err != nil {
		log.Fatalf("❌ Failed to load configuration: %v", err)
	}

	// Extract symbol from dual-engine config
	symbol := dualEngineConfig.MainConfig.Symbol
	if symbol == "" {
		log.Fatalf("❌ Symbol not specified in configuration")
	}

	fmt.Printf("💱 Trading Symbol: %s\n", symbol)
	fmt.Printf("⚙️  Exchange: %s\n", dualEngineConfig.MainConfig.Exchange)
	
	if *demoMode || dualEngineConfig.MainConfig.DemoMode {
		fmt.Println("🧪 Running in DEMO MODE (Paper Trading)")
		// Override demo mode setting
		dualEngineConfig.MainConfig.DemoMode = true
	} else {
		fmt.Println("💰 Running in LIVE MODE (Real Money)")
	}

	// Create logger
	fileLogger, err := logger.NewLogger(symbol, dualEngineConfig.MainConfig.Interval)
	if err != nil {
		log.Fatalf("❌ Failed to create logger: %v", err)
	}
	defer fileLogger.Close()

	// Create exchange adapter from dual-engine config
	exchangeConfig := exchange.ExchangeConfig{
		Name: dualEngineConfig.MainConfig.Exchange,
	}
	
	// Add exchange-specific config if available, otherwise use demo placeholders
	if dualEngineConfig.MainConfig.Exchange == "bybit" {
		bybitConfig := &exchange.BybitConfig{
			Demo:    dualEngineConfig.MainConfig.DemoMode,
			Testnet: false, // Use demo mode instead of testnet for development
		}
		
		// Check if we have exchange config in the dual engine config
		if dualEngineConfig.ExchangeConfig != nil && dualEngineConfig.ExchangeConfig.Bybit != nil {
			bybitConfig.APIKey = dualEngineConfig.ExchangeConfig.Bybit.APIKey
			bybitConfig.APISecret = dualEngineConfig.ExchangeConfig.Bybit.APISecret
			bybitConfig.Demo = dualEngineConfig.ExchangeConfig.Bybit.Demo
			bybitConfig.Testnet = dualEngineConfig.ExchangeConfig.Bybit.Testnet
		} else {
			// Use demo placeholders
			bybitConfig.APIKey = "demo-key-placeholder"
			bybitConfig.APISecret = "demo-secret-placeholder"
		}
		
		exchangeConfig.Bybit = bybitConfig
	}
	
	// Create legacy config for compatibility
	legacyConfig := &config.LiveBotConfig{
		Exchange: exchangeConfig,
		Strategy: config.StrategyConfig{
			Symbol:   dualEngineConfig.MainConfig.Symbol,
			Interval: dualEngineConfig.MainConfig.Interval,
		},
	}
	
	// Ensure API credentials are set from environment variables
	if err := ensureAPICredentials(legacyConfig); err != nil {
		log.Fatalf("❌ Failed to set API credentials: %v", err)
	}

	factory := adapters.NewFactory()
	exchangeInstance, err := factory.CreateExchange(legacyConfig.Exchange)
	if err != nil {
		log.Fatalf("❌ Failed to create exchange: %v", err)
	}

	// Create dual engine bot
	fmt.Println("🎼 Initializing Dual Engine Bot...")
	dualEngineBot, err := orchestrator.NewDualEngineBot(symbol, legacyConfig, exchangeInstance, fileLogger)
	if err != nil {
		log.Fatalf("❌ Failed to create dual engine bot: %v", err)
	}

	// Start the bot
	fmt.Println("🚀 Starting Dual Engine Bot...")
	if err := dualEngineBot.Start(); err != nil {
		log.Fatalf("❌ Failed to start dual engine bot: %v", err)
	}

	// Setup graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Display initial status
	showBotStatus(dualEngineBot)

	// Start status monitoring in background
	go monitorBotStatus(dualEngineBot, *verbose)

	// Wait for shutdown signal
	fmt.Println("✅ Dual Engine Bot is running. Press Ctrl+C to stop.")
	<-signalChan

	// Graceful shutdown
	fmt.Println("\n🛑 Shutdown signal received...")
	if err := dualEngineBot.Stop(); err != nil {
		log.Printf("⚠️  Error during shutdown: %v", err)
	}

	fmt.Println("👋 Dual Engine Bot stopped successfully")
}

func showHelp() {
	fmt.Println("🚀 Enhanced DCA Bot - Dual Engine System")
	fmt.Println("======================================")
	fmt.Println("")
	fmt.Println("USAGE:")
	fmt.Println("  dual-engine-bot -config <config_file> [options]")
	fmt.Println("")
	fmt.Println("OPTIONS:")
	fmt.Println("  -config <file>    Configuration file path (REQUIRED)")
	fmt.Println("  -demo             Run in demo mode (paper trading)")
	fmt.Println("  -verbose          Enable verbose status logging")
	fmt.Println("  -env <file>       Environment file path (default: .env)")
	fmt.Println("  -help             Show this help message")
	fmt.Println("")
	fmt.Println("EXAMPLES:")
	fmt.Println("  # Run with dual-engine configuration")
	fmt.Println("  dual-engine-bot -config configs/dual_engine/btc_development.json")
	fmt.Println("")
	fmt.Println("  # Run in demo mode")
	fmt.Println("  dual-engine-bot -config configs/dual_engine/btc_development.json -demo")
	fmt.Println("")
	fmt.Println("  # Run with verbose logging")
	fmt.Println("  dual-engine-bot -config configs/dual_engine/btc_development.json -verbose")
	fmt.Println("")
	fmt.Println("  # Use custom environment file")
	fmt.Println("  dual-engine-bot -config configs/dual_engine/btc_development.json -env production.env")
	fmt.Println("")
	fmt.Println("FEATURES:")
	fmt.Println("  🎯 Automatic regime detection (TRENDING/RANGING/VOLATILE/UNCERTAIN)")
	fmt.Println("  🏗️  Dual engine system:")
	fmt.Println("     • Grid Engine - VWAP/EMA anchored hedge grid for ranging/volatile markets")
	fmt.Println("     • Trend Engine - Multi-timeframe trend following for trending markets")  
	fmt.Println("  🔄 Automatic engine switching based on market conditions")
	fmt.Println("  📊 Real-time performance monitoring")
	fmt.Println("  🛡️  Advanced risk management")
	fmt.Println("  📈 Multi-timeframe analysis")
	fmt.Println("")
}

func showBotStatus(bot *orchestrator.DualEngineBot) {
	fmt.Println("\n📊 DUAL ENGINE BOT STATUS")
	fmt.Println("========================")
	
	// Current regime and engine
	currentRegime := bot.GetCurrentRegime()
	activeEngine := bot.GetActiveEngine()
	
	fmt.Printf("🎯 Current Regime: %s\n", currentRegime.String())
	fmt.Printf("🏗️  Active Engine: %s\n", activeEngine.GetName())
	
	// Engine statuses
	fmt.Println("\n🔧 ENGINE STATUS:")
	engineStatuses := bot.GetEngineStatuses()
	for engineType, status := range engineStatuses {
		activeIndicator := "⏸️"
		if status.IsActive {
			activeIndicator = "▶️"
		}
		
		fmt.Printf("  %s %s: %s (Positions: %d)\n",
			activeIndicator,
			engineType.String(),
			getStatusEmoji(status.IsActive, status.IsTrading),
			status.ActivePositions,
		)
	}
	
	// Orchestration metrics
	metrics := bot.GetOrchestrationMetrics()
	fmt.Printf("\n📈 SESSION METRICS:\n")
	fmt.Printf("  • Regime Changes: %d\n", metrics.RegimeChanges)
	fmt.Printf("  • Total Signals: %d\n", metrics.TotalSignals)
	fmt.Printf("  • Session Duration: %v\n", time.Since(metrics.SessionStartTime).Round(time.Second))
	fmt.Printf("  • Total P&L: $%.2f\n", metrics.TotalPnL)
	
	fmt.Println("")
}

func monitorBotStatus(bot *orchestrator.DualEngineBot, verbose bool) {
	ticker := time.NewTicker(30 * time.Second) // Update every 30 seconds
	defer ticker.Stop()
	
	lastRegime := bot.GetCurrentRegime()
	lastActiveEngine := bot.GetActiveEngine().GetType()
	
	for {
		select {
		case <-ticker.C:
			if !bot.IsRunning() {
				return
			}
			
			currentRegime := bot.GetCurrentRegime()
			currentActiveEngine := bot.GetActiveEngine().GetType()
			
			// Only show status if something changed or verbose mode
			if verbose || currentRegime != lastRegime || currentActiveEngine != lastActiveEngine {
				fmt.Printf("\r🎯 Regime: %s | 🏗️ Engine: %s | ⏰ %s",
					currentRegime.String(),
					currentActiveEngine.String(),
					time.Now().Format("15:04:05"),
				)
				
				if currentRegime != lastRegime {
					fmt.Printf(" | 🔄 REGIME CHANGE: %s → %s", 
						lastRegime.String(), 
						currentRegime.String())
				}
				
				if currentActiveEngine != lastActiveEngine {
					fmt.Printf(" | ⚡ ENGINE SWITCH: %s → %s", 
						lastActiveEngine.String(), 
						currentActiveEngine.String())
				}
				
				fmt.Println()
			}
			
			lastRegime = currentRegime
			lastActiveEngine = currentActiveEngine
		}
	}
}

func getStatusEmoji(isActive, isTrading bool) string {
	if isActive && isTrading {
		return "🟢 ACTIVE"
	} else if isActive {
		return "🟡 STANDBY"
	}
	return "🔴 INACTIVE"
}

// loadEnvFile loads environment variables from .env file
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
		if config.Exchange.Bybit.APIKey == "" || config.Exchange.Bybit.APIKey == "demo-key-placeholder" {
			config.Exchange.Bybit.APIKey = os.Getenv("BYBIT_API_KEY")
		}
		if config.Exchange.Bybit.APISecret == "" || config.Exchange.Bybit.APISecret == "demo-secret-placeholder" {
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
		if config.Exchange.Binance.APIKey == "" || config.Exchange.Binance.APIKey == "demo-key-placeholder" {
			config.Exchange.Binance.APIKey = os.Getenv("BINANCE_API_KEY")
		}
		if config.Exchange.Binance.APISecret == "" || config.Exchange.Binance.APISecret == "demo-secret-placeholder" {
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
