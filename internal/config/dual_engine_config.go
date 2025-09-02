package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Type aliases to avoid circular imports
type EngineType string

const (
	EngineTypeGrid  EngineType = "grid"
	EngineTypeTrend EngineType = "trend"
)

// DualEngineConfig represents the complete configuration for the dual-engine system
// Based on the dual_engine_regime_bot_plan.json configuration management structure
type DualEngineConfig struct {
	// Metadata
	Version     string    `json:"version"`
	Environment string    `json:"environment"`    // "development", "staging", "production"
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	
	// Core configuration
	MainConfig        *MainConfig               `json:"main_config"`
	ExchangeConfig    *ExchangeConfiguration    `json:"exchange_config,omitempty"`
	RegimeDetection   *RegimeDetectionConfig    `json:"regime_detection"`
	TrendEngine       *TrendEngineConfig        `json:"trend_engine"`
	GridEngine        *GridEngineConfig         `json:"grid_engine"`
	RiskManagement    *RiskManagementConfig     `json:"risk_management"`
	TransitionRules   *TransitionRulesConfig    `json:"transition_rules"`
	
	// Engine orchestration
	Orchestration     *OrchestrationConfig      `json:"orchestration"`
	
	// Monitoring and notifications
	Monitoring        *MonitoringConfig                 `json:"monitoring"`
	Notifications     *DualEngineNotificationConfig     `json:"notifications"`
}

// MainConfig contains core system configuration
type MainConfig struct {
	// Trading configuration
	Symbol          string  `json:"symbol"`
	Exchange        string  `json:"exchange"`
	Interval        string  `json:"interval"`
	BaseAmount      float64 `json:"base_amount"`
	
	// System settings
	LogLevel        string  `json:"log_level"`        // "debug", "info", "warn", "error"
	DataDirectory   string  `json:"data_directory"`
	StateDirectory  string  `json:"state_directory"`
	
	// Performance settings
	MaxConcurrency  int     `json:"max_concurrency"`
	UpdateInterval  string  `json:"update_interval"`  // "30s", "1m", "5m"
	
	// Safety settings
	DemoMode        bool    `json:"demo_mode"`
	SafeMode        bool    `json:"safe_mode"`        // Extra conservative mode
	ManualOverride  bool    `json:"manual_override"`  // Require manual confirmation
}

// ExchangeConfiguration contains exchange-specific configurations
type ExchangeConfiguration struct {
	Bybit   *BybitConfiguration   `json:"bybit,omitempty"`
	Binance *BinanceConfiguration `json:"binance,omitempty"`
}

// BybitConfiguration contains Bybit-specific settings
type BybitConfiguration struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	Demo      bool   `json:"demo"`
	Testnet   bool   `json:"testnet"`
}

// BinanceConfiguration contains Binance-specific settings
type BinanceConfiguration struct {
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
	Demo      bool   `json:"demo"`
	Testnet   bool   `json:"testnet"`
}

// RegimeDetectionConfig contains all regime detection parameters
type RegimeDetectionConfig struct {
	// From plan's regime_detection_parameters
	TrendDetection struct {
		EMAPeriods           []int   `json:"ema_periods"`             // [50, 200]
		ADXPeriod           int     `json:"adx_period"`              // 14
		ADXTrendThreshold   float64 `json:"adx_trend_threshold"`     // 20
		EMADistanceThreshold float64 `json:"ema_distance_threshold"` // 0.005
		DonchianPeriod      int     `json:"donchian_period"`         // 20
	} `json:"trend_detection"`
	
	VolatilityAssessment struct {
		ATRPeriod              int     `json:"atr_period"`                // 14
		BBPeriod               int     `json:"bb_period"`                 // 20
		BBStdDev               float64 `json:"bb_std_dev"`                // 2
		VolatilityNormalization string  `json:"volatility_normalization"` // "price_percentage"
	} `json:"volatility_assessment"`
	
	NoiseDetection struct {
		RSIPeriod           int       `json:"rsi_period"`             // 14
		RSINoiseRange       []float64 `json:"rsi_noise_range"`        // [45, 55]
		NoiseBarsThreshold  int       `json:"noise_bars_threshold"`   // 8
	} `json:"noise_detection"`
	
	Hysteresis struct {
		ConfirmationBars      int `json:"confirmation_bars"`       // 3
		RegimeSwitchCooldown  int `json:"regime_switch_cooldown"`  // 2
	} `json:"hysteresis"`
	
	// Confidence thresholds (JSON keys are strings)
	ConfidenceThresholds map[string]float64 `json:"confidence_thresholds"`
}

// TrendEngineConfig contains trend engine parameters from the plan
type TrendEngineConfig struct {
	// From plan's trend_engine_parameters
	TimeframeHierarchy struct {
		BiasTimeframe       string   `json:"bias_timeframe"`        // "30m"
		ExecutionTimeframes []string `json:"execution_timeframes"`  // ["5m", "1m"]
	} `json:"timeframe_hierarchy"`
	
	EntryConditions struct {
		PullbackLevels    []float64 `json:"pullback_levels"`      // [0.382, 0.618]
		MomentumIndicators []string  `json:"momentum_indicators"`  // ["macd_histogram", "rsi_5"]
		EntryMethods      []string  `json:"entry_methods"`        // ["limit_first", "market_fallback"]
	} `json:"entry_conditions"`
	
	RiskManagement struct {
		StopLossMethod      string    `json:"stop_loss_method"`       // "swing_low_or_atr"
		ATRMultiplier       float64   `json:"atr_multiplier"`         // 1.2
		TakeProfitScaling   []float64 `json:"take_profit_scaling"`    // [0.5, 1.0]
		TrailingMethod      string    `json:"trailing_method"`        // "atr_or_chandelier"
		ChandelierMultiplier float64  `json:"chandelier_multiplier"`  // 3
	} `json:"risk_management"`
	
	PositionManagement struct {
		MaxAddOns      int      `json:"max_add_ons"`       // 2
		AddOnConditions []string `json:"add_on_conditions"` // ["unrealized_pnl_positive", "adx_rising"]
	} `json:"position_management"`
	
	// Engine limits
	MaxPositionSize float64 `json:"max_position_size"`   // 0.3 (30%)
	MaxDailyTrades  int     `json:"max_daily_trades"`    // 10
	MaxDailyLoss    float64 `json:"max_daily_loss"`      // 0.02 (2%)
}

// GridEngineConfig contains grid engine parameters from the plan
type GridEngineConfig struct {
	// From plan's grid_engine_parameters
	AnchorMethods []string `json:"anchor_methods"`  // ["anchored_vwap", "ema_100"]
	
	BandCalculation struct {
		ATRMultiplier         float64 `json:"atr_multiplier"`          // 0.75
		GridSpacingMultiplier float64 `json:"grid_spacing_multiplier"` // 0.25
		MaxBands             int     `json:"max_bands"`               // 4
	} `json:"band_calculation"`
	
	HedgeManagement struct {
		SymmetricPlacement bool `json:"symmetric_placement"`  // true
		InventoryLimits struct {
			MaxLongNotional  float64 `json:"max_long_notional"`   // 0.4
			MaxShortNotional float64 `json:"max_short_notional"`  // 0.4
			MaxNetExposure   float64 `json:"max_net_exposure"`    // 0.1
		} `json:"inventory_limits"`
	} `json:"hedge_management"`
	
	ExitConditions struct {
		TakeProfitMultiplier float64 `json:"take_profit_multiplier"` // 0.7
		StopLossMultiplier   float64 `json:"stop_loss_multiplier"`   // 1.5
		TimeBasedExit        bool    `json:"time_based_exit"`        // true
		MaxBarsInTrade       int     `json:"max_bars_in_trade"`      // 48
	} `json:"exit_conditions"`
	
	SafetyMechanisms struct {
		BBWidthExitThreshold float64 `json:"bb_width_exit_threshold"` // 1.5
		ADXPopupThreshold    float64 `json:"adx_popup_threshold"`     // 22
		RegimeFlipExit       bool    `json:"regime_flip_exit"`        // true
	} `json:"safety_mechanisms"`
	
	// Engine limits
	MaxTotalExposure float64 `json:"max_total_exposure"`  // 0.8 (80%)
	MaxLegsPerSide   int     `json:"max_legs_per_side"`   // 6
	MaxDailyLoss     float64 `json:"max_daily_loss"`      // 0.015 (1.5%)
}

// RiskManagementConfig contains global risk management configuration
type RiskManagementConfig struct {
	// From plan's risk_management_framework
	GlobalLimits struct {
		MaxPortfolioRisk    float64 `json:"max_portfolio_risk"`    // 0.05 (5%)
		MaxDailyLoss        float64 `json:"max_daily_loss"`        // 0.03 (3%)
		MaxDrawdown         float64 `json:"max_drawdown"`          // 0.08 (8%)
		CorrelationLimits struct {
			MaxEngineCorrelation   float64 `json:"max_engine_correlation"`   // 0.6
			PositionConcentration  float64 `json:"position_concentration"`   // 0.4
		} `json:"correlation_limits"`
	} `json:"global_limits"`
	
	CircuitBreakers struct {
		VolatilitySpike struct {
			Threshold string `json:"threshold"`  // "3x_daily_atr"
			Action    string `json:"action"`     // "reduce_positions"
		} `json:"volatility_spike"`
		
		RegimeUncertainty struct {
			Threshold string `json:"threshold"`  // "confidence_below_60%"
			Action    string `json:"action"`     // "pause_new_entries"
		} `json:"regime_uncertainty"`
		
		SystemErrors struct {
			ExchangeDisconnection string `json:"exchange_disconnection"` // "emergency_flatten"
			DataFeedIssues       string `json:"data_feed_issues"`       // "pause_trading"
			CalculationErrors    string `json:"calculation_errors"`     // "alert_and_hold"
		} `json:"system_errors"`
	} `json:"circuit_breakers"`
	
	// Emergency settings
	AutoFlattenOnEmergency    bool `json:"auto_flatten_on_emergency"`
	RequireManualConfirmation bool `json:"require_manual_confirmation"`
	EmergencyNotifications    bool `json:"emergency_notifications"`
}

// TransitionRulesConfig contains transition management configuration
type TransitionRulesConfig struct {
	// Daily limits
	MaxDailyTransitions     int     `json:"max_daily_transitions"`      // Max 10 per day
	MaxDailyTransitionCost  float64 `json:"max_daily_transition_cost"`  // Max 1% portfolio
	
	// Transition thresholds
	RegimeConfidenceThreshold float64       `json:"regime_confidence_threshold"` // 70%
	MinRegimeDuration         time.Duration `json:"min_regime_duration"`         // 30 minutes
	TransitionCooldown        time.Duration `json:"transition_cooldown"`         // 5 minutes
	
	// Cost controls
	MaxTransitionCost      float64 `json:"max_transition_cost"`       // 0.1% per transition
	EmergencyExitThreshold float64 `json:"emergency_exit_threshold"`  // -3% PnL
	
	// Policy settings
	DefaultPolicy string `json:"default_policy"`  // "conservative", "aggressive", "adaptive"
	
	// Transition decision matrix configuration
	TrendToChopRules struct {
		ImmediateExitThreshold   float64       `json:"immediate_exit_threshold"`    // -2%
		GracefulMigrationAge     time.Duration `json:"graceful_migration_age"`      // 2 hours
		ProtectiveHoldThreshold  float64       `json:"protective_hold_threshold"`   // -1%
	} `json:"trend_to_chop_rules"`
	
	ChopToTrendRules struct {
		FlattenHedgeADXThreshold float64 `json:"flatten_hedge_adx_threshold"` // 25
		ConvertToTrendThreshold  float64 `json:"convert_to_trend_threshold"`  // 0% PnL
		GradualUnwindADXRange    struct {
			Min float64 `json:"min"` // 20
			Max float64 `json:"max"` // 25
		} `json:"gradual_unwind_adx_range"`
	} `json:"chop_to_trend_rules"`
}

// OrchestrationConfig contains engine orchestration settings
type OrchestrationConfig struct {
	// Engine selection
	DefaultEngine       EngineType          `json:"default_engine"`        // "grid" or "trend"
	EngineSelectionMode string             `json:"engine_selection_mode"` // "regime_based", "manual", "hybrid"
	
	// Regime compatibility matrix (JSON keys are strings)
	RegimeCompatibility map[string][]EnginePreference `json:"regime_compatibility"`
	
	// Switching logic
	SwitchingEnabled    bool          `json:"switching_enabled"`
	MinEngineRuntime    time.Duration `json:"min_engine_runtime"`    // Min time before switching
	SwitchingCooldown   time.Duration `json:"switching_cooldown"`    // Cooldown between switches
	
	// Performance monitoring
	PerformanceWindow   time.Duration `json:"performance_window"`    // Window for performance comparison
	MinPerformanceDiff  float64       `json:"min_performance_diff"`  // Min difference to trigger switch
}

// EnginePreference represents engine preference for a regime
type EnginePreference struct {
	Engine         EngineType          `json:"engine"`
	Compatibility  float64            `json:"compatibility"`  // 0.0 to 1.0
	Priority       int                `json:"priority"`       // 1 = highest
}

// MonitoringConfig contains monitoring and metrics configuration
type MonitoringConfig struct {
	// Metrics collection
	EnableMetrics      bool          `json:"enable_metrics"`
	MetricsInterval    time.Duration `json:"metrics_interval"`     // How often to collect
	MetricsRetention   time.Duration `json:"metrics_retention"`    // How long to keep
	
	// Performance tracking
	TrackPerformance   bool `json:"track_performance"`
	PerformanceMetrics []string `json:"performance_metrics"` // ["pnl", "sharpe", "drawdown", etc.]
	
	// Health monitoring
	HealthChecks       []string      `json:"health_checks"`       // ["exchange", "data_feed", "memory", etc.]
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	
	// Alerting thresholds
	AlertThresholds map[string]float64 `json:"alert_thresholds"`
}

// DualEngineNotificationConfig contains notification settings for dual engine system
type DualEngineNotificationConfig struct {
	// Channels
	EnableTelegram bool   `json:"enable_telegram"`
	TelegramToken  string `json:"telegram_token,omitempty"`
	TelegramChatID string `json:"telegram_chat_id,omitempty"`
	
	EnableEmail    bool   `json:"enable_email"`
	EmailSMTP      string `json:"email_smtp,omitempty"`
	EmailUser      string `json:"email_user,omitempty"`
	EmailPassword  string `json:"email_password,omitempty"`
	EmailTo        string `json:"email_to,omitempty"`
	
	EnableWebhook  bool   `json:"enable_webhook"`
	WebhookURL     string `json:"webhook_url,omitempty"`
	
	// Notification rules
	NotifyRegimeChanges   bool `json:"notify_regime_changes"`
	NotifyEngineSwitch    bool `json:"notify_engine_switch"`
	NotifyRiskViolations  bool `json:"notify_risk_violations"`
	NotifyEmergencyActions bool `json:"notify_emergency_actions"`
	NotifyPerformance     bool `json:"notify_performance"`
	
	// Frequency controls
	MaxNotificationsPerHour int           `json:"max_notifications_per_hour"`
	NotificationCooldown    time.Duration `json:"notification_cooldown"`
}

// Environment-specific configuration presets

// DevelopmentConfig returns configuration optimized for development/testing
func DevelopmentConfig() *DualEngineConfig {
	return &DualEngineConfig{
		Version:     "1.0.0-dev",
		Environment: "development",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		
		MainConfig: &MainConfig{
			Symbol:         "BTCUSDT",
			Exchange:       "bybit",
			Interval:       "5m",
			BaseAmount:     100.0,  // Small amount for testing
			LogLevel:       "debug",
			DataDirectory:  "data",
			StateDirectory: "state",
			MaxConcurrency: 10,
			UpdateInterval: "30s",
			DemoMode:       true,   // Always demo in dev
			SafeMode:       true,
			ManualOverride: false,
		},
		
		RegimeDetection: DefaultRegimeDetectionConfig(),
		TrendEngine:     DevelopmentTrendEngineConfig(),
		GridEngine:      DevelopmentGridEngineConfig(),
		RiskManagement:  DevelopmentRiskManagementConfig(),
		TransitionRules: DevelopmentTransitionRulesConfig(),
		Orchestration:   DefaultOrchestrationConfig(),
		Monitoring:      DevelopmentMonitoringConfig(),
		Notifications:   DevelopmentDualEngineNotificationConfig(),
	}
}

// StagingConfig returns configuration optimized for staging/validation
func StagingConfig() *DualEngineConfig {
	config := DevelopmentConfig()
	config.Version = "1.0.0-staging"
	config.Environment = "staging"
	
	// More conservative settings
	config.MainConfig.BaseAmount = 500.0
	config.MainConfig.LogLevel = "info"
	config.MainConfig.SafeMode = true
	config.MainConfig.ManualOverride = true  // Require confirmation in staging
	
	return config
}

// ProductionConfig returns configuration optimized for production
func ProductionConfig() *DualEngineConfig {
	config := DevelopmentConfig()
	config.Version = "1.0.0-prod"
	config.Environment = "production"
	
	// Production settings
	config.MainConfig.BaseAmount = 1000.0
	config.MainConfig.LogLevel = "info"
	config.MainConfig.DemoMode = false     // Real trading
	config.MainConfig.SafeMode = false
	config.MainConfig.ManualOverride = false
	config.MainConfig.MaxConcurrency = 20
	config.MainConfig.UpdateInterval = "60s"
	
	// More aggressive risk limits for production
	config.RiskManagement = ProductionRiskManagementConfig()
	config.TransitionRules = ProductionTransitionRulesConfig()
	config.Monitoring = ProductionMonitoringConfig()
	config.Notifications = ProductionDualEngineNotificationConfig()
	
	return config
}

// Configuration loader and management

// LoadDualEngineConfig loads configuration from file or returns default for environment
func LoadDualEngineConfig(configPath, environment string) (*DualEngineConfig, error) {
	// If specific config file provided, load it
	if configPath != "" && configPath != "default" {
		return LoadConfigFromFile(configPath)
	}
	
	// Otherwise return environment default
	switch environment {
	case "development", "dev":
		return DevelopmentConfig(), nil
	case "staging", "stage":
		return StagingConfig(), nil
	case "production", "prod":
		return ProductionConfig(), nil
	default:
		return DevelopmentConfig(), nil  // Default to development
	}
}

// LoadConfigFromFile loads configuration from a JSON file
func LoadConfigFromFile(configPath string) (*DualEngineConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}
	
	var config DualEngineConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", configPath, err)
	}
	
	config.UpdatedAt = time.Now()
	
	// Validate configuration
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	
	return &config, nil
}

// SaveConfigToFile saves configuration to a JSON file
func SaveConfigToFile(config *DualEngineConfig, configPath string) error {
	config.UpdatedAt = time.Now()
	
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// ValidateConfig validates the configuration for consistency and correctness
func ValidateConfig(config *DualEngineConfig) error {
	if config.MainConfig == nil {
		return fmt.Errorf("main_config is required")
	}
	
	if config.MainConfig.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}
	
	if config.MainConfig.Exchange == "" {
		return fmt.Errorf("exchange is required")
	}
	
	if config.MainConfig.BaseAmount <= 0 {
		return fmt.Errorf("base_amount must be positive")
	}
	
	// Validate risk management limits
	if config.RiskManagement != nil {
		rm := config.RiskManagement
		if rm.GlobalLimits.MaxPortfolioRisk <= 0 || rm.GlobalLimits.MaxPortfolioRisk > 1 {
			return fmt.Errorf("max_portfolio_risk must be between 0 and 1")
		}
		if rm.GlobalLimits.MaxDailyLoss <= 0 || rm.GlobalLimits.MaxDailyLoss > 1 {
			return fmt.Errorf("max_daily_loss must be between 0 and 1")
		}
		if rm.GlobalLimits.MaxDrawdown <= 0 || rm.GlobalLimits.MaxDrawdown > 1 {
			return fmt.Errorf("max_drawdown must be between 0 and 1")
		}
	}
	
	// Validate transition rules
	if config.TransitionRules != nil {
		tr := config.TransitionRules
		if tr.MaxDailyTransitions <= 0 {
			return fmt.Errorf("max_daily_transitions must be positive")
		}
		if tr.RegimeConfidenceThreshold <= 0 || tr.RegimeConfidenceThreshold > 1 {
			return fmt.Errorf("regime_confidence_threshold must be between 0 and 1")
		}
	}
	
	return nil
}

// GetConfigSummary returns a summary of the configuration for logging
func GetConfigSummary(config *DualEngineConfig) string {
	return fmt.Sprintf("Dual Engine Config v%s (%s): %s/%s, Base: $%.2f, Demo: %v", 
		config.Version, 
		config.Environment,
		config.MainConfig.Exchange,
		config.MainConfig.Symbol,
		config.MainConfig.BaseAmount,
		config.MainConfig.DemoMode)
}
