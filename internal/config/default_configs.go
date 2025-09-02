package config

import (
	"time"
)

// Note: EngineType is defined in dual_engine_config.go

// DefaultRegimeDetectionConfig returns the default regime detection configuration
// Based on the technical_specifications.regime_detection_parameters from the plan
func DefaultRegimeDetectionConfig() *RegimeDetectionConfig {
	config := &RegimeDetectionConfig{}
	
	// Trend detection parameters
	config.TrendDetection.EMAPeriods = []int{50, 200}
	config.TrendDetection.ADXPeriod = 14
	config.TrendDetection.ADXTrendThreshold = 20
	config.TrendDetection.EMADistanceThreshold = 0.005
	config.TrendDetection.DonchianPeriod = 20
	
	// Volatility assessment
	config.VolatilityAssessment.ATRPeriod = 14
	config.VolatilityAssessment.BBPeriod = 20
	config.VolatilityAssessment.BBStdDev = 2
	config.VolatilityAssessment.VolatilityNormalization = "price_percentage"
	
	// Noise detection
	config.NoiseDetection.RSIPeriod = 14
	config.NoiseDetection.RSINoiseRange = []float64{45, 55}
	config.NoiseDetection.NoiseBarsThreshold = 8
	
	// Hysteresis
	config.Hysteresis.ConfirmationBars = 3
	config.Hysteresis.RegimeSwitchCooldown = 2
	
	// Confidence thresholds for each regime (using string keys for JSON compatibility)
	config.ConfidenceThresholds = map[string]float64{
		"trending":  0.75,
		"ranging":   0.70,
		"volatile":  0.65,
		"uncertain": 0.60,
	}
	
	return config
}

// DevelopmentTrendEngineConfig returns trend engine config optimized for development
func DevelopmentTrendEngineConfig() *TrendEngineConfig {
	config := &TrendEngineConfig{}
	
	// Timeframe hierarchy
	config.TimeframeHierarchy.BiasTimeframe = "30m"
	config.TimeframeHierarchy.ExecutionTimeframes = []string{"5m", "1m"}
	
	// Entry conditions
	config.EntryConditions.PullbackLevels = []float64{0.382, 0.618}
	config.EntryConditions.MomentumIndicators = []string{"macd_histogram", "rsi_5"}
	config.EntryConditions.EntryMethods = []string{"limit_first", "market_fallback"}
	
	// Risk management
	config.RiskManagement.StopLossMethod = "swing_low_or_atr"
	config.RiskManagement.ATRMultiplier = 1.2
	config.RiskManagement.TakeProfitScaling = []float64{0.5, 1.0}
	config.RiskManagement.TrailingMethod = "atr_or_chandelier"
	config.RiskManagement.ChandelierMultiplier = 3
	
	// Position management
	config.PositionManagement.MaxAddOns = 2
	config.PositionManagement.AddOnConditions = []string{"unrealized_pnl_positive", "adx_rising"}
	
	// Conservative limits for development
	config.MaxPositionSize = 0.2  // 20% instead of 30%
	config.MaxDailyTrades = 5     // 5 instead of 10
	config.MaxDailyLoss = 0.01    // 1% instead of 2%
	
	return config
}

// DevelopmentGridEngineConfig returns grid engine config optimized for development
func DevelopmentGridEngineConfig() *GridEngineConfig {
	config := &GridEngineConfig{}
	
	// Anchor methods
	config.AnchorMethods = []string{"anchored_vwap", "ema_100"}
	
	// Band calculation
	config.BandCalculation.ATRMultiplier = 0.75
	config.BandCalculation.GridSpacingMultiplier = 0.25
	config.BandCalculation.MaxBands = 3  // 3 instead of 4 for development
	
	// Hedge management
	config.HedgeManagement.SymmetricPlacement = true
	config.HedgeManagement.InventoryLimits.MaxLongNotional = 0.3   // 30% instead of 40%
	config.HedgeManagement.InventoryLimits.MaxShortNotional = 0.3  // 30% instead of 40%
	config.HedgeManagement.InventoryLimits.MaxNetExposure = 0.05   // 5% instead of 10%
	
	// Exit conditions
	config.ExitConditions.TakeProfitMultiplier = 0.7
	config.ExitConditions.StopLossMultiplier = 1.5
	config.ExitConditions.TimeBasedExit = true
	config.ExitConditions.MaxBarsInTrade = 24  // 24 instead of 48 for faster testing
	
	// Safety mechanisms
	config.SafetyMechanisms.BBWidthExitThreshold = 1.5
	config.SafetyMechanisms.ADXPopupThreshold = 22
	config.SafetyMechanisms.RegimeFlipExit = true
	
	// Conservative limits for development
	config.MaxTotalExposure = 0.6   // 60% instead of 80%
	config.MaxLegsPerSide = 4       // 4 instead of 6
	config.MaxDailyLoss = 0.01      // 1% instead of 1.5%
	
	return config
}

// DevelopmentRiskManagementConfig returns risk management config for development
func DevelopmentRiskManagementConfig() *RiskManagementConfig {
	config := &RiskManagementConfig{}
	
	// Global limits - more conservative for development
	config.GlobalLimits.MaxPortfolioRisk = 0.03     // 3% instead of 5%
	config.GlobalLimits.MaxDailyLoss = 0.02         // 2% instead of 3%
	config.GlobalLimits.MaxDrawdown = 0.05          // 5% instead of 8%
	config.GlobalLimits.CorrelationLimits.MaxEngineCorrelation = 0.5  // 50% instead of 60%
	config.GlobalLimits.CorrelationLimits.PositionConcentration = 0.3 // 30% instead of 40%
	
	// Circuit breakers
	config.CircuitBreakers.VolatilitySpike.Threshold = "2x_daily_atr"  // 2x instead of 3x
	config.CircuitBreakers.VolatilitySpike.Action = "reduce_positions"
	config.CircuitBreakers.RegimeUncertainty.Threshold = "confidence_below_70%"  // 70% instead of 60%
	config.CircuitBreakers.RegimeUncertainty.Action = "pause_new_entries"
	config.CircuitBreakers.SystemErrors.ExchangeDisconnection = "emergency_flatten"
	config.CircuitBreakers.SystemErrors.DataFeedIssues = "pause_trading"
	config.CircuitBreakers.SystemErrors.CalculationErrors = "alert_and_hold"
	
	// Emergency settings
	config.AutoFlattenOnEmergency = true
	config.RequireManualConfirmation = true   // Require confirmation in development
	config.EmergencyNotifications = true
	
	return config
}

// DevelopmentTransitionRulesConfig returns transition rules for development
func DevelopmentTransitionRulesConfig() *TransitionRulesConfig {
	config := &TransitionRulesConfig{}
	
	// Daily limits - more conservative
	config.MaxDailyTransitions = 5                    // 5 instead of 10
	config.MaxDailyTransitionCost = 0.005             // 0.5% instead of 1%
	
	// Transition thresholds
	config.RegimeConfidenceThreshold = 0.75           // 75% instead of 70%
	config.MinRegimeDuration = 15 * time.Minute       // 15 minutes instead of 30
	config.TransitionCooldown = 10 * time.Minute      // 10 minutes instead of 5
	
	// Cost controls
	config.MaxTransitionCost = 0.0005                 // 0.05% instead of 0.1%
	config.EmergencyExitThreshold = -0.02             // -2% instead of -3%
	
	// Policy settings
	config.DefaultPolicy = "conservative"
	
	// Trend to chop rules
	config.TrendToChopRules.ImmediateExitThreshold = -0.015   // -1.5% instead of -2%
	config.TrendToChopRules.GracefulMigrationAge = 1 * time.Hour  // 1 hour instead of 2
	config.TrendToChopRules.ProtectiveHoldThreshold = -0.005  // -0.5% instead of -1%
	
	// Chop to trend rules
	config.ChopToTrendRules.FlattenHedgeADXThreshold = 23     // 23 instead of 25
	config.ChopToTrendRules.ConvertToTrendThreshold = 0.005   // 0.5% instead of 0%
	config.ChopToTrendRules.GradualUnwindADXRange.Min = 18    // 18 instead of 20
	config.ChopToTrendRules.GradualUnwindADXRange.Max = 23    // 23 instead of 25
	
	return config
}

// DefaultOrchestrationConfig returns the default orchestration configuration
func DefaultOrchestrationConfig() *OrchestrationConfig {
	config := &OrchestrationConfig{}
	
	// Engine selection
	config.DefaultEngine = EngineTypeGrid  // Start with grid engine
	config.EngineSelectionMode = "regime_based"
	
	// Regime compatibility matrix (using string keys for JSON compatibility)
	config.RegimeCompatibility = map[string][]EnginePreference{
		"trending": {
			{Engine: EngineTypeTrend, Compatibility: 1.0, Priority: 1},
			{Engine: EngineTypeGrid, Compatibility: 0.3, Priority: 2},
		},
		"ranging": {
			{Engine: EngineTypeGrid, Compatibility: 1.0, Priority: 1},
			{Engine: EngineTypeTrend, Compatibility: 0.2, Priority: 2},
		},
		"volatile": {
			{Engine: EngineTypeGrid, Compatibility: 0.8, Priority: 1},
			{Engine: EngineTypeTrend, Compatibility: 0.4, Priority: 2},
		},
		"uncertain": {
			{Engine: EngineTypeGrid, Compatibility: 0.6, Priority: 1},
			{Engine: EngineTypeTrend, Compatibility: 0.3, Priority: 2},
		},
	}
	
	// Switching logic
	config.SwitchingEnabled = true
	config.MinEngineRuntime = 30 * time.Minute      // Min 30 minutes before switching
	config.SwitchingCooldown = 15 * time.Minute     // 15 minutes cooldown
	
	// Performance monitoring
	config.PerformanceWindow = 24 * time.Hour       // 24 hour performance window
	config.MinPerformanceDiff = 0.02                // 2% minimum difference
	
	return config
}

// DevelopmentMonitoringConfig returns monitoring config for development
func DevelopmentMonitoringConfig() *MonitoringConfig {
	config := &MonitoringConfig{}
	
	// Metrics collection
	config.EnableMetrics = true
	config.MetricsInterval = 30 * time.Second
	config.MetricsRetention = 7 * 24 * time.Hour  // 7 days
	
	// Performance tracking
	config.TrackPerformance = true
	config.PerformanceMetrics = []string{"pnl", "sharpe", "drawdown", "win_rate", "avg_trade"}
	
	// Health monitoring
	config.HealthChecks = []string{"exchange", "data_feed", "memory", "disk"}
	config.HealthCheckInterval = 1 * time.Minute
	
	// Alert thresholds
	config.AlertThresholds = map[string]float64{
		"daily_loss":      -0.02,  // -2%
		"drawdown":        0.05,   // 5%
		"memory_usage":    0.80,   // 80%
		"disk_usage":      0.85,   // 85%
		"error_rate":      0.05,   // 5%
	}
	
	return config
}

// DevelopmentDualEngineNotificationConfig returns notification config for development
func DevelopmentDualEngineNotificationConfig() *DualEngineNotificationConfig {
	config := &DualEngineNotificationConfig{}
	
	// Channels - disabled by default in development
	config.EnableTelegram = false
	config.EnableEmail = false
	config.EnableWebhook = false
	
	// Notification rules
	config.NotifyRegimeChanges = true
	config.NotifyEngineSwitch = true
	config.NotifyRiskViolations = true
	config.NotifyEmergencyActions = true
	config.NotifyPerformance = false  // Don't spam with performance notifications in dev
	
	// Frequency controls
	config.MaxNotificationsPerHour = 10
	config.NotificationCooldown = 5 * time.Minute
	
	return config
}

// Production configuration variants

// ProductionRiskManagementConfig returns production-optimized risk management
func ProductionRiskManagementConfig() *RiskManagementConfig {
	config := DevelopmentRiskManagementConfig()
	
	// More aggressive limits for production
	config.GlobalLimits.MaxPortfolioRisk = 0.05      // 5%
	config.GlobalLimits.MaxDailyLoss = 0.03          // 3%
	config.GlobalLimits.MaxDrawdown = 0.08           // 8%
	config.GlobalLimits.CorrelationLimits.MaxEngineCorrelation = 0.6   // 60%
	config.GlobalLimits.CorrelationLimits.PositionConcentration = 0.4  // 40%
	
	// Circuit breakers
	config.CircuitBreakers.VolatilitySpike.Threshold = "3x_daily_atr"
	config.CircuitBreakers.RegimeUncertainty.Threshold = "confidence_below_60%"
	
	// Emergency settings
	config.RequireManualConfirmation = false  // No manual confirmation in production
	
	return config
}

// ProductionTransitionRulesConfig returns production-optimized transition rules
func ProductionTransitionRulesConfig() *TransitionRulesConfig {
	config := DevelopmentTransitionRulesConfig()
	
	// Production limits
	config.MaxDailyTransitions = 10                   // 10 transitions per day
	config.MaxDailyTransitionCost = 0.01              // 1% of portfolio
	
	// Transition thresholds
	config.RegimeConfidenceThreshold = 0.70           // 70%
	config.MinRegimeDuration = 30 * time.Minute       // 30 minutes
	config.TransitionCooldown = 5 * time.Minute       // 5 minutes
	
	// Cost controls
	config.MaxTransitionCost = 0.001                  // 0.1%
	config.EmergencyExitThreshold = -0.03             // -3%
	
	// Policy settings
	config.DefaultPolicy = "adaptive"  // Use adaptive policy in production
	
	// Restore original thresholds from plan
	config.TrendToChopRules.ImmediateExitThreshold = -0.02      // -2%
	config.TrendToChopRules.GracefulMigrationAge = 2 * time.Hour // 2 hours
	config.TrendToChopRules.ProtectiveHoldThreshold = -0.01     // -1%
	
	config.ChopToTrendRules.FlattenHedgeADXThreshold = 25       // 25
	config.ChopToTrendRules.ConvertToTrendThreshold = 0         // 0%
	config.ChopToTrendRules.GradualUnwindADXRange.Min = 20      // 20
	config.ChopToTrendRules.GradualUnwindADXRange.Max = 25      // 25
	
	return config
}

// ProductionMonitoringConfig returns production-optimized monitoring
func ProductionMonitoringConfig() *MonitoringConfig {
	config := DevelopmentMonitoringConfig()
	
	// Production settings
	config.MetricsInterval = 1 * time.Minute         // More frequent in production
	config.MetricsRetention = 30 * 24 * time.Hour    // 30 days retention
	config.HealthCheckInterval = 30 * time.Second     // More frequent health checks
	
	// Enhanced performance tracking
	config.PerformanceMetrics = []string{
		"pnl", "sharpe", "sortino", "calmar", "drawdown", "var", 
		"win_rate", "avg_trade", "profit_factor", "recovery_factor",
	}
	
	// Production alert thresholds
	config.AlertThresholds = map[string]float64{
		"daily_loss":      -0.03,  // -3%
		"drawdown":        0.08,   // 8%
		"memory_usage":    0.85,   // 85%
		"disk_usage":      0.90,   // 90%
		"error_rate":      0.02,   // 2%
		"latency":         1.0,    // 1 second
	}
	
	return config
}

// ProductionDualEngineNotificationConfig returns production-optimized notifications
func ProductionDualEngineNotificationConfig() *DualEngineNotificationConfig {
	config := DevelopmentDualEngineNotificationConfig()
	
	// Enable all channels for production
	config.EnableTelegram = true
	config.EnableEmail = true
	config.EnableWebhook = true
	
	// Enable all notifications for production
	config.NotifyRegimeChanges = true
	config.NotifyEngineSwitch = true
	config.NotifyRiskViolations = true
	config.NotifyEmergencyActions = true
	config.NotifyPerformance = true
	
	// Production frequency controls
	config.MaxNotificationsPerHour = 20
	config.NotificationCooldown = 2 * time.Minute
	
	return config
}
