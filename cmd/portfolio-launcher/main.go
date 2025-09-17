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
	"github.com/joho/godotenv"
)

func main() {
	var (
		portfolioConfigFile = flag.String("portfolio", "", "Portfolio configuration file (e.g., aave_hype_equal_weight.json)")
		exchangeName       = flag.String("exchange", "", "Exchange name (bybit, binance) - overrides individual bot configs")
		demo               = flag.Bool("demo", true, "Use demo trading environment - paper trading (default: true)")
		envFile            = flag.String("env", ".env", "Environment file path (default: .env)")
		statusOnly         = flag.Bool("status", false, "Show status only (don't start bots)")
		detailedStatus     = flag.Bool("detailed", false, "Show detailed status report")
		monitorInterval    = flag.Duration("monitor", 30*time.Second, "Status monitoring interval (e.g., 30s, 1m)")
	)
	flag.Parse()

	if *portfolioConfigFile == "" {
		fmt.Println("❌ Please specify a portfolio config file with -portfolio flag")
		fmt.Println("\nUsage examples:")
		fmt.Println("  ./portfolio-launcher -portfolio aave_hype_equal_weight.json")
		fmt.Println("  ./portfolio-launcher -portfolio configs/portfolio/aave_hype_equal_weight.json -demo=false")
		fmt.Println("  ./portfolio-launcher -portfolio aave_hype_equal_weight.json -status")
		os.Exit(1)
	}

	// Load environment variables from .env file
	if err := loadEnvFile(*envFile); err != nil {
		log.Printf("Warning: Could not load .env file (%v), checking environment variables...", err)
	}

	fmt.Println("🚀 Multi-Bot Portfolio Launcher Starting...")
	fmt.Printf("📁 Portfolio Config: %s\n", *portfolioConfigFile)

	// Load portfolio configuration
	portfolioConfig, err := LoadPortfolioConfig(*portfolioConfigFile)
	if err != nil {
		log.Fatalf("Failed to load portfolio config: %v", err)
	}

	fmt.Printf("📋 Portfolio: %s\n", portfolioConfig.Description)
	fmt.Printf("🤖 Enabled Bots: %d\n", len(portfolioConfig.GetEnabledBots()))

	// If status-only mode, load and show status then exit
	if *statusOnly {
		showPortfolioStatus(portfolioConfig)
		return
	}

	// Create bot manager
	botManager := NewBotManager(portfolioConfig)

	// Apply global overrides to all bot configs
	if err := applyGlobalOverrides(portfolioConfig, *exchangeName, *demo); err != nil {
		log.Fatalf("Failed to apply global overrides: %v", err)
	}

	// Ensure API credentials for all bots
	if err := ensureAllAPICredentials(portfolioConfig); err != nil {
		log.Fatalf("API credentials validation failed: %v", err)
	}

	// Create portfolio monitor
	monitor := NewPortfolioMonitor(botManager, portfolioConfig)

	// Start all bots
	if err := botManager.StartAllBots(); err != nil {
		log.Fatalf("Failed to start portfolio: %v", err)
	}

	// Start monitoring
	monitor.Start()

	// Show initial detailed status if requested
	if *detailedStatus {
		time.Sleep(3 * time.Second) // Give bots time to initialize
		monitor.PrintDetailedStatus()
	}

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Status monitoring loop
	go func() {
		ticker := time.NewTicker(*monitorInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if *detailedStatus {
					monitor.PrintDetailedStatus()
				} else {
					// Just show a simple status line
					summary := monitor.GetPortfolioSummary()
					fmt.Printf("📊 Portfolio: %d/%d bots running | Uptime: %v | Alerts: %d\n",
						summary.RunningBots, summary.TotalBots,
						summary.Uptime.Truncate(time.Second),
						len(summary.Alerts))
				}
			case <-sigChan:
				return
			}
		}
	}()

	// Wait for shutdown signal
	fmt.Println("📡 Portfolio is ready. Commands:")
	fmt.Println("  • Ctrl+C: Graceful shutdown")
	fmt.Println("  • Send SIGTERM for graceful shutdown")
	if !*detailedStatus {
		fmt.Printf("  • Use -detailed flag for detailed monitoring\n")
	}
	fmt.Println()

	select {
	case sig := <-sigChan:
		fmt.Printf("\n🛑 Shutdown signal (%v) received...\n", sig)
	}

	// Graceful shutdown sequence
	fmt.Println("🔄 Initiating graceful shutdown...")

	// Stop monitoring first
	monitor.Stop()

	// Stop all bots
	botManager.StopAllBots()

	fmt.Println("✅ Portfolio stopped successfully")
}

// showPortfolioStatus displays the current status of a portfolio
func showPortfolioStatus(portfolioConfig *PortfolioConfig) {
	fmt.Printf("\n")
	fmt.Printf("╔══════════════════════════════════════════════════════════════╗\n")
	fmt.Printf("║                    PORTFOLIO CONFIGURATION                   ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║ Description: %s\n", portfolioConfig.Description)
	fmt.Printf("║ Total Balance: $%.2f\n", portfolioConfig.Portfolio.TotalBalance)
	fmt.Printf("║ Strategy: %s\n", portfolioConfig.Portfolio.AllocationStrategy)
	fmt.Printf("║ Max Exposure: %.1fx\n", portfolioConfig.Portfolio.MaxTotalExposure)
	fmt.Printf("║ Max Drawdown: %.1f%%\n", portfolioConfig.Portfolio.MaxDrawdownPercent)
	fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")
	fmt.Printf("║                           BOTS                               ║\n")
	fmt.Printf("╠══════════════════════════════════════════════════════════════╣\n")

	for i, bot := range portfolioConfig.Bots {
		status := "🔴 DISABLED"
		if bot.Enabled {
			status = "🟢 ENABLED"
		}

		fmt.Printf("║ %d. %-15s │ %s\n", i+1, bot.BotID, status)
		fmt.Printf("║    Config: %s\n", bot.ConfigFile)

		// Try to load and show basic config info
		if bot.Enabled {
			if botConfig, err := config.LoadLiveBotConfig(bot.ConfigFile); err == nil {
				fmt.Printf("║    Symbol: %s │ Leverage: %.1fx │ Interval: %s\n",
					botConfig.Strategy.Symbol,
					botConfig.Strategy.Leverage,
					botConfig.Strategy.Interval)
			}
		}
		fmt.Printf("║\n")
	}

	fmt.Printf("╚══════════════════════════════════════════════════════════════╝\n")
	fmt.Printf("\nTo start the portfolio, run without -status flag\n")
}

// applyGlobalOverrides applies global settings to all bot configurations
func applyGlobalOverrides(portfolioConfig *PortfolioConfig, exchangeName string, demo bool) error {
	if exchangeName == "" && !demo {
		return nil // No overrides to apply
	}

	fmt.Printf("🔧 Applying global overrides...\n")

	for _, botConfig := range portfolioConfig.GetEnabledBots() {
		// Load bot configuration
		liveBotConfig, err := config.LoadLiveBotConfig(botConfig.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load config for bot %s: %w", botConfig.BotID, err)
		}

		// Apply exchange override if specified
		if exchangeName != "" {
			liveBotConfig.Exchange.Name = exchangeName
			fmt.Printf("   🔄 Bot %s: Exchange -> %s\n", botConfig.BotID, exchangeName)
		}

		// Apply demo mode override
		if demo {
			switch strings.ToLower(liveBotConfig.Exchange.Name) {
			case "bybit":
				if liveBotConfig.Exchange.Bybit != nil {
					liveBotConfig.Exchange.Bybit.Demo = true
					liveBotConfig.Exchange.Bybit.Testnet = false
					fmt.Printf("   🧪 Bot %s: Demo mode enabled (Bybit)\n", botConfig.BotID)
				}
			case "binance":
				if liveBotConfig.Exchange.Binance != nil {
					liveBotConfig.Exchange.Binance.Demo = false // Binance doesn't have demo
					liveBotConfig.Exchange.Binance.Testnet = true
					fmt.Printf("   🧪 Bot %s: Testnet enabled (Binance)\n", botConfig.BotID)
				}
			}
		}

		// Note: We don't save the config back to file, just keep overrides in memory
	}

	return nil
}

// ensureAllAPICredentials ensures API credentials are set for all enabled bots
func ensureAllAPICredentials(portfolioConfig *PortfolioConfig) error {
	fmt.Printf("🔐 Validating API credentials for all bots...\n")

	for _, botConfig := range portfolioConfig.GetEnabledBots() {
		// Load bot configuration
		liveBotConfig, err := config.LoadLiveBotConfig(botConfig.ConfigFile)
		if err != nil {
			return fmt.Errorf("failed to load config for bot %s: %w", botConfig.BotID, err)
		}

		// Ensure API credentials
		if err := ensureAPICredentials(liveBotConfig); err != nil {
			return fmt.Errorf("bot %s: %w", botConfig.BotID, err)
		}

		fmt.Printf("   ✅ Bot %s: API credentials OK\n", botConfig.BotID)
	}

	return nil
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
