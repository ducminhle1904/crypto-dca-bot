package risk

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/engines"
	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// Overseer implements global risk management and emergency controls
// Based on the dual_engine_regime_bot_plan.json risk management framework
type Overseer struct {
	logger     *logger.Logger
	exchange   exchange.LiveTradingExchange
	config     *OverseerConfig
	
	// Risk tracking
	dailyPnL           float64
	maxDrawdown        float64
	peakValue          float64
	portfolioValue     float64
	totalExposure      float64
	
	// State management
	emergencyStop      bool
	manualOverride     bool
	systemPaused       bool
	lastResetDate      time.Time
	
	// Circuit breakers
	circuitBreakers    map[string]*CircuitBreaker
	
	// Engine monitoring
	engineMetrics      map[engines.EngineType]*EngineRiskMetrics
	
	// Thread safety
	mutex              sync.RWMutex
	
	// Emergency actions
	emergencyActions   []EmergencyAction
	actionMutex        sync.Mutex
}

// OverseerConfig contains all risk management configuration
type OverseerConfig struct {
	// Global risk limits (from plan)
	MaxPortfolioRisk    float64   `json:"max_portfolio_risk"`     // 5% max portfolio risk
	MaxDailyLoss        float64   `json:"max_daily_loss"`         // 3% max daily loss
	MaxDrawdown         float64   `json:"max_drawdown"`           // 8% max drawdown
	
	// Engine-level limits
	TrendEngineConfig   *EngineRiskConfig `json:"trend_engine_config"`
	GridEngineConfig    *EngineRiskConfig `json:"grid_engine_config"`
	
	// Circuit breaker thresholds
	VolatilitySpikeThreshold    float64  `json:"volatility_spike_threshold"`     // 3x daily ATR
	RegimeUncertaintyThreshold  float64  `json:"regime_uncertainty_threshold"`   // Confidence below 60%
	
	// Correlation limits
	MaxEngineCorrelation        float64  `json:"max_engine_correlation"`         // 0.6 max correlation
	PositionConcentration       float64  `json:"position_concentration"`         // 0.4 max concentration
	
	// Emergency settings
	AutoFlattenOnEmergency     bool     `json:"auto_flatten_on_emergency"`
	EmergencyNotifications     bool     `json:"emergency_notifications"`
	RequireManualConfirmation  bool     `json:"require_manual_confirmation"`
}

// EngineRiskConfig contains risk limits for individual engines
type EngineRiskConfig struct {
	MaxPositionSize     float64  `json:"max_position_size"`       // Max position size ratio
	MaxDailyTrades      int      `json:"max_daily_trades"`        // Max trades per day
	MaxDailyLoss        float64  `json:"max_daily_loss"`          // Max daily loss ratio
	MaxTotalExposure    float64  `json:"max_total_exposure"`      // Max total exposure ratio
}

// CircuitBreaker represents a specific risk control mechanism
type CircuitBreaker struct {
	Name                string          `json:"name"`
	Type                string          `json:"type"`                // "volatility_spike", "regime_uncertainty", "system_error"
	Threshold           float64         `json:"threshold"`
	Action              string          `json:"action"`              // "reduce_positions", "pause_trading", "emergency_flatten"
	CooldownPeriod      time.Duration   `json:"cooldown_period"`
	ActivationCount     int             `json:"activation_count"`
	LastActivated       time.Time       `json:"last_activated"`
	IsActive            bool            `json:"is_active"`
}

// EngineRiskMetrics tracks risk metrics for individual engines
type EngineRiskMetrics struct {
	EngineType          engines.EngineType  `json:"engine_type"`
	DailyTrades         int                 `json:"daily_trades"`
	DailyPnL            float64             `json:"daily_pnl"`
	TotalExposure       float64             `json:"total_exposure"`
	MaxPositionSize     float64             `json:"max_position_size"`
	RiskScore           float64             `json:"risk_score"`           // 0.0 to 1.0
	LastUpdate          time.Time           `json:"last_update"`
}

// EmergencyAction represents an emergency action taken by the overseer
type EmergencyAction struct {
	Timestamp           time.Time           `json:"timestamp"`
	Trigger             string              `json:"trigger"`
	Action              string              `json:"action"`
	EngineAffected      engines.EngineType  `json:"engine_affected"`
	PositionsAffected   int                 `json:"positions_affected"`
	Success             bool                `json:"success"`
	Details             string              `json:"details"`
}

// DefaultOverseerConfig returns default risk management configuration
func DefaultOverseerConfig() *OverseerConfig {
	return &OverseerConfig{
		// Global limits (from plan)
		MaxPortfolioRisk:           0.05,    // 5%
		MaxDailyLoss:               0.03,    // 3%
		MaxDrawdown:                0.08,    // 8%
		
		// Engine-specific limits
		TrendEngineConfig: &EngineRiskConfig{
			MaxPositionSize:    0.3,         // 30%
			MaxDailyTrades:     10,          // 10 trades/day
			MaxDailyLoss:       0.02,        // 2%
			MaxTotalExposure:   0.5,         // 50%
		},
		GridEngineConfig: &EngineRiskConfig{
			MaxPositionSize:    0.4,         // 40%
			MaxDailyTrades:     20,          // 20 trades/day (grid can be more active)
			MaxDailyLoss:       0.015,       // 1.5%
			MaxTotalExposure:   0.8,         // 80%
		},
		
		// Circuit breaker thresholds
		VolatilitySpikeThreshold:   3.0,     // 3x daily ATR
		RegimeUncertaintyThreshold: 0.6,     // 60% confidence
		
		// Correlation limits
		MaxEngineCorrelation:       0.6,     // 60%
		PositionConcentration:      0.4,     // 40%
		
		// Emergency settings
		AutoFlattenOnEmergency:     true,    // Auto-flatten on emergency
		EmergencyNotifications:     true,    // Send notifications
		RequireManualConfirmation:  false,   // No manual confirmation by default
	}
}

// NewOverseer creates a new risk management overseer
func NewOverseer(logger *logger.Logger, exchange exchange.LiveTradingExchange) *Overseer {
	overseer := &Overseer{
		logger:            logger,
		exchange:          exchange,
		config:            DefaultOverseerConfig(),
		circuitBreakers:   make(map[string]*CircuitBreaker),
		engineMetrics:     make(map[engines.EngineType]*EngineRiskMetrics),
		emergencyActions:  make([]EmergencyAction, 0, 100),
		lastResetDate:     time.Now(),
		peakValue:         10000.0, // Starting value - should be configured
		portfolioValue:    10000.0,
	}
	
	overseer.initializeCircuitBreakers()
	overseer.initializeEngineMetrics()
	
	return overseer
}

// initializeCircuitBreakers sets up the circuit breaker mechanisms
func (o *Overseer) initializeCircuitBreakers() {
	// Volatility spike circuit breaker
	o.circuitBreakers["volatility_spike"] = &CircuitBreaker{
		Name:           "Volatility Spike Protection",
		Type:           "volatility_spike",
		Threshold:      o.config.VolatilitySpikeThreshold,
		Action:         "reduce_positions",
		CooldownPeriod: 30 * time.Minute,
	}
	
	// Regime uncertainty circuit breaker
	o.circuitBreakers["regime_uncertainty"] = &CircuitBreaker{
		Name:           "Regime Uncertainty Protection", 
		Type:           "regime_uncertainty",
		Threshold:      o.config.RegimeUncertaintyThreshold,
		Action:         "pause_new_entries",
		CooldownPeriod: 15 * time.Minute,
	}
	
	// System error circuit breaker
	o.circuitBreakers["system_error"] = &CircuitBreaker{
		Name:           "System Error Protection",
		Type:           "system_error",
		Threshold:      1.0, // Any system error
		Action:         "alert_and_hold",
		CooldownPeriod: 5 * time.Minute,
	}
	
	// Exchange disconnection circuit breaker
	o.circuitBreakers["exchange_disconnect"] = &CircuitBreaker{
		Name:           "Exchange Disconnection Protection",
		Type:           "exchange_disconnect",
		Threshold:      1.0, // Any disconnection
		Action:         "emergency_flatten",
		CooldownPeriod: 1 * time.Minute,
	}
	
	// Data feed issues circuit breaker
	o.circuitBreakers["data_feed"] = &CircuitBreaker{
		Name:           "Data Feed Protection",
		Type:           "data_feed",
		Threshold:      1.0, // Any data feed issue
		Action:         "pause_trading",
		CooldownPeriod: 10 * time.Minute,
	}
}

// initializeEngineMetrics sets up monitoring for each engine type
func (o *Overseer) initializeEngineMetrics() {
	o.engineMetrics[engines.EngineTypeGrid] = &EngineRiskMetrics{
		EngineType:      engines.EngineTypeGrid,
		DailyTrades:     0,
		DailyPnL:        0.0,
		TotalExposure:   0.0,
		MaxPositionSize: 0.0,
		RiskScore:       0.0,
		LastUpdate:      time.Now(),
	}
	
	o.engineMetrics[engines.EngineTypeTrend] = &EngineRiskMetrics{
		EngineType:      engines.EngineTypeTrend,
		DailyTrades:     0,
		DailyPnL:        0.0,
		TotalExposure:   0.0,
		MaxPositionSize: 0.0,
		RiskScore:       0.0,
		LastUpdate:      time.Now(),
	}
}

// MonitorRisk performs comprehensive risk monitoring and enforcement
func (o *Overseer) MonitorRisk(ctx context.Context, engines map[engines.EngineType]engines.TradingEngine, 
	regimeSignal *regime.RegimeSignal, marketData []types.OHLCV) *RiskAssessment {
	
	o.mutex.Lock()
	defer o.mutex.Unlock()
	
	// Reset daily limits if needed
	o.resetDailyLimitsIfNeeded()
	
	// Update portfolio metrics
	o.updatePortfolioMetrics(engines)
	
	// Update engine metrics
	o.updateEngineMetrics(engines)
	
	// Check global risk limits
	globalViolations := o.checkGlobalRiskLimits()
	
	// Check engine-specific limits
	engineViolations := o.checkEngineRiskLimits(engines)
	
	// Check circuit breakers
	circuitBreakerAlerts := o.checkCircuitBreakers(regimeSignal, marketData)
	
	// Calculate overall risk score
	overallRiskScore := o.calculateOverallRiskScore()
	
	assessment := &RiskAssessment{
		Timestamp:            time.Now(),
		OverallRiskScore:     overallRiskScore,
		PortfolioValue:       o.portfolioValue,
		DailyPnL:             o.dailyPnL,
		MaxDrawdown:          o.maxDrawdown,
		TotalExposure:        o.totalExposure,
		GlobalViolations:     globalViolations,
		EngineViolations:     engineViolations,
		CircuitBreakerAlerts: circuitBreakerAlerts,
		SystemStatus:         o.getSystemStatus(),
		EmergencyStop:        o.emergencyStop,
		ManualOverride:       o.manualOverride,
	}
	
	// Take emergency actions if needed
	if len(globalViolations) > 0 || len(circuitBreakerAlerts) > 0 {
		o.handleEmergencyConditions(assessment, engines)
	}
	
	return assessment
}

// checkGlobalRiskLimits verifies global portfolio risk limits
func (o *Overseer) checkGlobalRiskLimits() []RiskViolation {
	var violations []RiskViolation
	
	// Check daily loss limit
	dailyLossPercent := o.dailyPnL / o.portfolioValue
	if dailyLossPercent < -o.config.MaxDailyLoss {
		violations = append(violations, RiskViolation{
			Type:        "daily_loss_limit",
			Description: fmt.Sprintf("Daily loss %.2f%% exceeds limit %.2f%%", dailyLossPercent*100, o.config.MaxDailyLoss*100),
			Severity:    "critical",
			Current:     dailyLossPercent,
			Limit:       -o.config.MaxDailyLoss,
			Action:      "emergency_flatten",
		})
	}
	
	// Check maximum drawdown limit
	drawdownPercent := o.maxDrawdown / o.peakValue
	if drawdownPercent > o.config.MaxDrawdown {
		violations = append(violations, RiskViolation{
			Type:        "max_drawdown_limit",
			Description: fmt.Sprintf("Drawdown %.2f%% exceeds limit %.2f%%", drawdownPercent*100, o.config.MaxDrawdown*100),
			Severity:    "critical",
			Current:     drawdownPercent,
			Limit:       o.config.MaxDrawdown,
			Action:      "reduce_positions",
		})
	}
	
	// Check portfolio risk limit
	portfolioRisk := o.totalExposure / o.portfolioValue
	if portfolioRisk > o.config.MaxPortfolioRisk {
		violations = append(violations, RiskViolation{
			Type:        "portfolio_risk_limit",
			Description: fmt.Sprintf("Portfolio risk %.2f%% exceeds limit %.2f%%", portfolioRisk*100, o.config.MaxPortfolioRisk*100),
			Severity:    "high",
			Current:     portfolioRisk,
			Limit:       o.config.MaxPortfolioRisk,
			Action:      "reduce_positions",
		})
	}
	
	return violations
}

// checkEngineRiskLimits verifies engine-specific risk limits
func (o *Overseer) checkEngineRiskLimits(engines map[engines.EngineType]engines.TradingEngine) []RiskViolation {
	var violations []RiskViolation
	
	for engineType, _ := range engines {
		metrics := o.engineMetrics[engineType]
		var config *EngineRiskConfig
		
		// Get engine-specific config
		switch engineType {
		case engines.EngineTypeTrend:
			config = o.config.TrendEngineConfig
		case engines.EngineTypeGrid:
			config = o.config.GridEngineConfig
		default:
			continue
		}
		
		// Check daily trades limit
		if metrics.DailyTrades > config.MaxDailyTrades {
			violations = append(violations, RiskViolation{
				Type:        "engine_daily_trades",
				Description: fmt.Sprintf("%s engine exceeded daily trade limit: %d > %d", engineType, metrics.DailyTrades, config.MaxDailyTrades),
				Severity:    "medium",
				Current:     float64(metrics.DailyTrades),
				Limit:       float64(config.MaxDailyTrades),
				Action:      "pause_engine_trading",
				EngineType:  engineType,
			})
		}
		
		// Check daily loss limit
		dailyLossPercent := metrics.DailyPnL / o.portfolioValue
		if dailyLossPercent < -config.MaxDailyLoss {
			violations = append(violations, RiskViolation{
				Type:        "engine_daily_loss",
				Description: fmt.Sprintf("%s engine daily loss %.2f%% exceeds limit %.2f%%", engineType, dailyLossPercent*100, config.MaxDailyLoss*100),
				Severity:    "high",
				Current:     dailyLossPercent,
				Limit:       -config.MaxDailyLoss,
				Action:      "reduce_engine_positions",
				EngineType:  engineType,
			})
		}
		
		// Check total exposure limit
		exposurePercent := metrics.TotalExposure / o.portfolioValue
		if exposurePercent > config.MaxTotalExposure {
			violations = append(violations, RiskViolation{
				Type:        "engine_exposure_limit",
				Description: fmt.Sprintf("%s engine exposure %.2f%% exceeds limit %.2f%%", engineType, exposurePercent*100, config.MaxTotalExposure*100),
				Severity:    "medium",
				Current:     exposurePercent,
				Limit:       config.MaxTotalExposure,
				Action:      "reduce_engine_positions",
				EngineType:  engineType,
			})
		}
	}
	
	return violations
}

// checkCircuitBreakers evaluates all circuit breaker conditions
func (o *Overseer) checkCircuitBreakers(regimeSignal *regime.RegimeSignal, marketData []types.OHLCV) []CircuitBreakerAlert {
	var alerts []CircuitBreakerAlert
	
	// Check volatility spike
	if len(marketData) >= 14 {
		currentVolatility := o.calculateVolatility(marketData)
		dailyATR := o.calculateDailyATR(marketData)
		
		if currentVolatility > dailyATR*o.config.VolatilitySpikeThreshold {
			alert := o.triggerCircuitBreaker("volatility_spike", fmt.Sprintf("Volatility spike detected: %.2fx daily ATR", currentVolatility/dailyATR))
			if alert != nil {
				alerts = append(alerts, *alert)
			}
		}
	}
	
	// Check regime uncertainty
	if regimeSignal != nil && regimeSignal.Confidence < o.config.RegimeUncertaintyThreshold {
		alert := o.triggerCircuitBreaker("regime_uncertainty", fmt.Sprintf("Low regime confidence: %.1f%%", regimeSignal.Confidence*100))
		if alert != nil {
			alerts = append(alerts, *alert)
		}
	}
	
	return alerts
}

// triggerCircuitBreaker activates a circuit breaker if conditions are met
func (o *Overseer) triggerCircuitBreaker(breakerName, reason string) *CircuitBreakerAlert {
	breaker := o.circuitBreakers[breakerName]
	if breaker == nil {
		return nil
	}
	
	// Check if breaker is in cooldown period
	if breaker.IsActive && time.Since(breaker.LastActivated) < breaker.CooldownPeriod {
		return nil
	}
	
	// Activate circuit breaker
	breaker.IsActive = true
	breaker.LastActivated = time.Now()
	breaker.ActivationCount++
	
	o.logger.LogWarning("Circuit Breaker Activated", "%s: %s", breaker.Name, reason)
	
	return &CircuitBreakerAlert{
		Name:        breaker.Name,
		Type:        breaker.Type,
		Action:      breaker.Action,
		Reason:      reason,
		Timestamp:   time.Now(),
		Severity:    o.getCircuitBreakerSeverity(breaker.Type),
	}
}

// Helper methods for calculations

func (o *Overseer) updatePortfolioMetrics(engines map[engines.EngineType]engines.TradingEngine) {
	// Calculate total portfolio value and PnL
	totalValue := 0.0
	totalUnrealizedPnL := 0.0
	totalExposure := 0.0
	
	for _, engine := range engines {
		positions := engine.GetCurrentPositions()
		for _, pos := range positions {
			notional := pos.GetEntryPrice() * pos.GetSize()
			totalValue += notional
			totalUnrealizedPnL += pos.GetUnrealizedPnL()
			totalExposure += notional
		}
	}
	
	o.portfolioValue = totalValue
	o.totalExposure = totalExposure
	
	// Update peak value and drawdown
	if o.portfolioValue > o.peakValue {
		o.peakValue = o.portfolioValue
	}
	
	currentDrawdown := o.peakValue - o.portfolioValue
	if currentDrawdown > o.maxDrawdown {
		o.maxDrawdown = currentDrawdown
	}
	
	// Update daily PnL (simplified - should be calculated from start of day)
	o.dailyPnL = totalUnrealizedPnL
}

func (o *Overseer) updateEngineMetrics(engines map[engines.EngineType]engines.TradingEngine) {
	for engineType, engine := range engines {
		metrics := o.engineMetrics[engineType]
		if metrics == nil {
			continue
		}
		
		// Update engine-specific metrics
		positions := engine.GetCurrentPositions()
		totalExposure := 0.0
		totalPnL := 0.0
		maxPositionSize := 0.0
		
		for _, pos := range positions {
			notional := pos.GetEntryPrice() * pos.GetSize()
			totalExposure += notional
			totalPnL += pos.GetUnrealizedPnL()
			
			if notional > maxPositionSize {
				maxPositionSize = notional
			}
		}
		
		metrics.TotalExposure = totalExposure
		metrics.DailyPnL = totalPnL
		metrics.MaxPositionSize = maxPositionSize
		metrics.RiskScore = o.calculateEngineRiskScore(metrics)
		metrics.LastUpdate = time.Now()
	}
}

func (o *Overseer) calculateOverallRiskScore() float64 {
	// Weighted risk score calculation
	score := 0.0
	
	// Daily PnL impact (30% weight)
	if o.portfolioValue > 0 {
		dailyLossRatio := math.Abs(o.dailyPnL) / o.portfolioValue / o.config.MaxDailyLoss
		score += 0.3 * math.Min(dailyLossRatio, 1.0)
	}
	
	// Drawdown impact (30% weight)
	if o.peakValue > 0 {
		drawdownRatio := o.maxDrawdown / o.peakValue / o.config.MaxDrawdown
		score += 0.3 * math.Min(drawdownRatio, 1.0)
	}
	
	// Exposure impact (25% weight)
	if o.portfolioValue > 0 {
		exposureRatio := o.totalExposure / o.portfolioValue / o.config.MaxPortfolioRisk
		score += 0.25 * math.Min(exposureRatio, 1.0)
	}
	
	// Engine risk scores (15% weight)
	engineRiskSum := 0.0
	engineCount := 0
	for _, metrics := range o.engineMetrics {
		engineRiskSum += metrics.RiskScore
		engineCount++
	}
	if engineCount > 0 {
		score += 0.15 * (engineRiskSum / float64(engineCount))
	}
	
	return math.Min(score, 1.0)
}

func (o *Overseer) calculateEngineRiskScore(metrics *EngineRiskMetrics) float64 {
	score := 0.0
	
	// Get engine config
	var config *EngineRiskConfig
	switch metrics.EngineType {
	case engines.EngineTypeTrend:
		config = o.config.TrendEngineConfig
	case engines.EngineTypeGrid:
		config = o.config.GridEngineConfig
	default:
		return 0.0
	}
	
	// Daily trades impact
	if config.MaxDailyTrades > 0 {
		tradeRatio := float64(metrics.DailyTrades) / float64(config.MaxDailyTrades)
		score += 0.2 * math.Min(tradeRatio, 1.0)
	}
	
	// Daily loss impact
	if o.portfolioValue > 0 {
		lossRatio := math.Abs(metrics.DailyPnL) / o.portfolioValue / config.MaxDailyLoss
		score += 0.4 * math.Min(lossRatio, 1.0)
	}
	
	// Exposure impact
	if o.portfolioValue > 0 {
		exposureRatio := metrics.TotalExposure / o.portfolioValue / config.MaxTotalExposure
		score += 0.4 * math.Min(exposureRatio, 1.0)
	}
	
	return math.Min(score, 1.0)
}

func (o *Overseer) calculateVolatility(data []types.OHLCV) float64 {
	if len(data) < 14 {
		return 0
	}
	
	// Simple ATR calculation
	var atrSum float64
	for i := 1; i < min(len(data), 15); i++ {
		high := data[i].High
		low := data[i].Low
		prevClose := data[i-1].Close
		
		tr := math.Max(high-low, math.Max(math.Abs(high-prevClose), math.Abs(low-prevClose)))
		atrSum += tr
	}
	
	return atrSum / 14.0
}

func (o *Overseer) calculateDailyATR(data []types.OHLCV) float64 {
	// Daily ATR calculation (simplified - should use daily timeframe)
	return o.calculateVolatility(data) * 24 // Approximate daily from hourly
}

// Emergency action methods

func (o *Overseer) handleEmergencyConditions(assessment *RiskAssessment, engines map[engines.EngineType]engines.TradingEngine) {
	for _, violation := range assessment.GlobalViolations {
		o.executeEmergencyAction(violation.Action, "", violation.Description, engines)
	}
	
	for _, violation := range assessment.EngineViolations {
		o.executeEmergencyAction(violation.Action, violation.EngineType, violation.Description, engines)
	}
	
	for _, alert := range assessment.CircuitBreakerAlerts {
		o.executeEmergencyAction(alert.Action, "", alert.Reason, engines)
	}
}

func (o *Overseer) executeEmergencyAction(action string, engineType string, reason string, engines map[engines.EngineType]engines.TradingEngine) {
	o.actionMutex.Lock()
	defer o.actionMutex.Unlock()
	
	emergencyAction := EmergencyAction{
		Timestamp:      time.Now(),
		Trigger:        reason,
		Action:         action,
		EngineAffected: engines.EngineType(engineType),
		Success:        false,
	}
	
	switch action {
	case "emergency_flatten":
		emergencyAction.Success = o.emergencyFlattenAll(engines)
		
	case "reduce_positions":
		emergencyAction.Success = o.reducePositions(engines, 0.5) // Reduce by 50%
		
	case "pause_new_entries":
		emergencyAction.Success = o.pauseNewEntries(engines)
		
	case "reduce_engine_positions":
		if engine, exists := engines[engines.EngineType(engineType)]; exists {
			emergencyAction.Success = o.reduceEnginePositions(engine, 0.3) // Reduce by 30%
		}
		
	case "pause_engine_trading":
		if engine, exists := engines[engines.EngineType(engineType)]; exists {
			engine.SetActive(false)
			emergencyAction.Success = true
		}
		
	default:
		o.logger.LogWarning("Unknown Emergency Action", "Unknown action: %s", action)
		return
	}
	
	// Log emergency action
	o.emergencyActions = append(o.emergencyActions, emergencyAction)
	if len(o.emergencyActions) > 100 {
		o.emergencyActions = o.emergencyActions[1:] // Keep last 100
	}
	
	if emergencyAction.Success {
		o.logger.LogError("Emergency Action Executed", fmt.Errorf("Action: %s, Trigger: %s", action, reason))
	} else {
		o.logger.LogError("Emergency Action Failed", fmt.Errorf("Failed to execute %s: %s", action, reason))
	}
}

func (o *Overseer) emergencyFlattenAll(engines map[engines.EngineType]engines.TradingEngine) bool {
	success := true
	for _, engine := range engines {
		if !o.flattenEngine(engine) {
			success = false
		}
	}
	
	if success {
		o.emergencyStop = true
	}
	
	return success
}

func (o *Overseer) reducePositions(engines map[engines.EngineType]engines.TradingEngine, reduction float64) bool {
	success := true
	for _, engine := range engines {
		if !o.reduceEnginePositions(engine, reduction) {
			success = false
		}
	}
	return success
}

func (o *Overseer) pauseNewEntries(engines map[engines.EngineType]engines.TradingEngine) bool {
	// This would set a flag to prevent new entries
	// Implementation depends on engine design
	o.systemPaused = true
	return true
}

func (o *Overseer) reduceEnginePositions(engine engines.TradingEngine, reduction float64) bool {
	// Reduce positions by the specified percentage
	// This is a simplified implementation
	positions := engine.GetCurrentPositions()
	
	for _, pos := range positions {
		// In a real implementation, this would place orders to reduce position size
		o.logger.Info("Would reduce position %s by %.1f%%", pos.GetID(), reduction*100)
	}
	
	return true
}

func (o *Overseer) flattenEngine(engine engines.TradingEngine) bool {
	// Close all positions for the engine
	positions := engine.GetCurrentPositions()
	
	for _, pos := range positions {
		// In a real implementation, this would place market orders to close positions
		o.logger.Trade("Emergency flatten: closing position %s", pos.GetID())
	}
	
	return true
}

// Status and utility methods

func (o *Overseer) getSystemStatus() string {
	if o.emergencyStop {
		return "emergency_stop"
	}
	if o.systemPaused {
		return "paused"
	}
	if o.manualOverride {
		return "manual_override"
	}
	return "normal"
}

func (o *Overseer) getCircuitBreakerSeverity(breakerType string) string {
	switch breakerType {
	case "exchange_disconnect", "system_error":
		return "critical"
	case "volatility_spike":
		return "high"
	case "regime_uncertainty", "data_feed":
		return "medium"
	default:
		return "low"
	}
}

func (o *Overseer) resetDailyLimitsIfNeeded() {
	now := time.Now()
	if now.Day() != o.lastResetDate.Day() {
		// Reset daily counters
		o.dailyPnL = 0
		for _, metrics := range o.engineMetrics {
			metrics.DailyTrades = 0
			metrics.DailyPnL = 0
		}
		o.lastResetDate = now
		
		o.logger.Info("Daily risk limits reset")
	}
}

// Public API methods

func (o *Overseer) SetEmergencyStop(stop bool) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	
	o.emergencyStop = stop
	if stop {
		o.logger.LogError("Emergency Stop Activated", fmt.Errorf("Manual emergency stop activated"))
	} else {
		o.logger.Info("Emergency stop deactivated")
	}
}

func (o *Overseer) SetManualOverride(override bool) {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	
	o.manualOverride = override
	if override {
		o.logger.LogWarning("Manual Override", "Manual override activated - risk limits suspended")
	} else {
		o.logger.Info("Manual override deactivated")
	}
}

func (o *Overseer) GetRiskStatus() *RiskStatus {
	o.mutex.RLock()
	defer o.mutex.RUnlock()
	
	return &RiskStatus{
		SystemStatus:       o.getSystemStatus(),
		OverallRiskScore:   o.calculateOverallRiskScore(),
		PortfolioValue:     o.portfolioValue,
		DailyPnL:           o.dailyPnL,
		MaxDrawdown:        o.maxDrawdown,
		TotalExposure:      o.totalExposure,
		EmergencyStop:      o.emergencyStop,
		ManualOverride:     o.manualOverride,
		SystemPaused:       o.systemPaused,
		ActiveCircuitBreakers: o.getActiveCircuitBreakers(),
		LastUpdate:         time.Now(),
	}
}

func (o *Overseer) getActiveCircuitBreakers() []string {
	var active []string
	for name, breaker := range o.circuitBreakers {
		if breaker.IsActive && time.Since(breaker.LastActivated) < breaker.CooldownPeriod {
			active = append(active, name)
		}
	}
	return active
}

func (o *Overseer) GetEmergencyHistory() []EmergencyAction {
	o.actionMutex.Lock()
	defer o.actionMutex.Unlock()
	
	// Return copy of emergency actions
	history := make([]EmergencyAction, len(o.emergencyActions))
	copy(history, o.emergencyActions)
	return history
}

// Utility function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
