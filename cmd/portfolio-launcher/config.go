package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// PortfolioConfig represents the main portfolio configuration
type PortfolioConfig struct {
	Description string      `json:"description"`
	Bots        []BotConfig `json:"bots"`
	Portfolio   struct {
		TotalBalance        float64 `json:"total_balance"`
		AllocationStrategy  string  `json:"allocation_strategy"`
		SharedStateFile     string  `json:"shared_state_file"`
		MaxTotalExposure    float64 `json:"max_total_exposure"`
		MaxDrawdownPercent  float64 `json:"max_drawdown_percent"`
		RebalanceFrequency  string  `json:"rebalance_frequency"`
		RiskLimitPerBot     float64 `json:"risk_limit_per_bot"`
		EmergencyStopEnabled bool   `json:"emergency_stop_enabled"`
		ProfitSharing       struct {
			Enabled             bool    `json:"enabled"`
			Method              string  `json:"method"`
			RebalanceThreshold  float64 `json:"rebalance_threshold"`
		} `json:"profit_sharing"`
	} `json:"portfolio"`
	Monitoring struct {
		HeartbeatInterval   string `json:"heartbeat_interval"`
		SyncInterval        string `json:"sync_interval"`
		HealthCheckInterval string `json:"health_check_interval"`
	} `json:"monitoring"`
	Alerts struct {
		PortfolioDrawdownThreshold    float64 `json:"portfolio_drawdown_threshold"`
		IndividualBotLossThreshold    float64 `json:"individual_bot_loss_threshold"`
		CorrelationSpikeThreshold     float64 `json:"correlation_spike_threshold"`
	} `json:"alerts"`
}

// BotConfig represents individual bot configuration within portfolio
type BotConfig struct {
	BotID      string `json:"bot_id"`
	ConfigFile string `json:"config_file"`
	Enabled    bool   `json:"enabled"`
}

// LoadPortfolioConfig loads and validates a portfolio configuration file
func LoadPortfolioConfig(configFile string) (*PortfolioConfig, error) {
	// Resolve relative paths
	if !filepath.IsAbs(configFile) {
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			// Try configs/portfolio/ directory
			altPath := filepath.Join("configs", "portfolio", configFile)
			if _, err := os.Stat(altPath); err == nil {
				configFile = altPath
			}
		}
	}

	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read portfolio config file %s: %w", configFile, err)
	}

	var config PortfolioConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse portfolio config: %w", err)
	}

	// Validate configuration
	if err := validatePortfolioConfig(&config); err != nil {
		return nil, fmt.Errorf("portfolio config validation failed: %w", err)
	}

	return &config, nil
}

// validatePortfolioConfig validates the portfolio configuration
func validatePortfolioConfig(config *PortfolioConfig) error {
	if len(config.Bots) == 0 {
		return fmt.Errorf("no bots defined in portfolio")
	}

	enabledBots := 0
	for i, bot := range config.Bots {
		if bot.BotID == "" {
			return fmt.Errorf("bot[%d]: bot_id is required", i)
		}
		if bot.ConfigFile == "" {
			return fmt.Errorf("bot[%d]: config_file is required", i)
		}
		
		// Check if config file exists - try multiple path resolutions
		configPath := bot.ConfigFile
		if !filepath.IsAbs(configPath) {
			// Try relative to current directory first
			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				// Try relative to project root (go up two levels from cmd/portfolio-launcher)
				rootPath := filepath.Join("..", "..", configPath)
				if _, err := os.Stat(rootPath); err == nil {
					// Update the config path to the working one
					config.Bots[i].ConfigFile = rootPath
				} else {
					return fmt.Errorf("bot[%d]: config file %s not found (tried: %s, %s)", 
						i, bot.ConfigFile, configPath, rootPath)
				}
			}
		} else {
			if _, err := os.Stat(configPath); err != nil {
				return fmt.Errorf("bot[%d]: config file %s not accessible: %w", i, configPath, err)
			}
		}

		if bot.Enabled {
			enabledBots++
		}
	}

	if enabledBots == 0 {
		return fmt.Errorf("no enabled bots in portfolio")
	}

	// Validate portfolio settings
	if config.Portfolio.TotalBalance <= 0 {
		return fmt.Errorf("total_balance must be positive")
	}

	if config.Portfolio.MaxTotalExposure <= 0 {
		return fmt.Errorf("max_total_exposure must be positive")
	}

	if config.Portfolio.MaxDrawdownPercent <= 0 || config.Portfolio.MaxDrawdownPercent > 100 {
		return fmt.Errorf("max_drawdown_percent must be between 0 and 100")
	}

	validStrategies := map[string]bool{
		"equal_weight":     true,
		"performance_based": true,
		"risk_weighted":    true,
		"fixed_percentage": true,
		"custom":           true,
	}

	if !validStrategies[config.Portfolio.AllocationStrategy] {
		return fmt.Errorf("invalid allocation_strategy: %s", config.Portfolio.AllocationStrategy)
	}

	return nil
}

// GetEnabledBots returns only the enabled bots from the configuration
func (config *PortfolioConfig) GetEnabledBots() []BotConfig {
	var enabled []BotConfig
	for _, bot := range config.Bots {
		if bot.Enabled {
			enabled = append(enabled, bot)
		}
	}
	return enabled
}

// GetBotByID returns a bot configuration by its ID
func (config *PortfolioConfig) GetBotByID(botID string) (*BotConfig, error) {
	for _, bot := range config.Bots {
		if bot.BotID == botID {
			return &bot, nil
		}
	}
	return nil, fmt.Errorf("bot with ID %s not found", botID)
}
