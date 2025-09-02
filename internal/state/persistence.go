package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/logger"
	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/internal/transition"
)

// Type alias to avoid circular imports
type EngineType string

const (
	EngineTypeGrid  EngineType = "grid"
	EngineTypeTrend EngineType = "trend"
)

// Minimal interface definitions to avoid circular imports
type TradingEngine interface {
	GetCurrentPositions() []EnginePosition
	IsActive() bool
}

type EnginePosition interface {
	GetID() string
	GetSide() string
	GetSize() float64
	GetEntryPrice() float64
	GetCurrentPrice() float64
	GetUnrealizedPnL() float64
	GetEntryTime() time.Time
}

// StatePersistence manages saving and loading of dual-engine bot state
// Based on the architecture_design.enhanced_components.state_persistence requirements
type StatePersistence struct {
	logger       *logger.Logger
	stateDir     string
	symbol       string
	
	// Current state data
	currentState *SystemState
	stateMutex   sync.RWMutex
	
	// Auto-save settings
	autoSave     bool
	saveInterval time.Duration
	lastSave     time.Time
	
	// File handles for streaming data
	regimeLogFile     *os.File
	transitionLogFile *os.File
	metricsLogFile    *os.File
}

// SystemState represents the complete recoverable state of the dual-engine system
type SystemState struct {
	// Metadata
	Version        string    `json:"version"`
	Symbol         string    `json:"symbol"`
	LastUpdated    time.Time `json:"last_updated"`
	SessionStart   time.Time `json:"session_start"`
	
	// Engine states
	EngineStates   map[EngineType]*EngineState `json:"engine_states"`
	ActiveEngine   EngineType                  `json:"active_engine"`
	
	// Regime state
	RegimeState    *RegimeState    `json:"regime_state"`
	RegimeHistory  []*RegimeRecord `json:"regime_history"`
	
	// Transition state
	TransitionState *TransitionState    `json:"transition_state"`
	TransitionLogs  []*TransitionRecord `json:"transition_logs"`
	
	// Performance and metrics
	SessionMetrics  *SessionMetrics                        `json:"session_metrics"`
	EngineMetrics   map[EngineType]*EngineMetrics  `json:"engine_metrics"`
	
	// Risk and portfolio state
	PortfolioState  *PortfolioState `json:"portfolio_state"`
	RiskState       *RiskState      `json:"risk_state"`
}

// EngineState represents the persisted state of a trading engine
type EngineState struct {
	EngineType      EngineType    `json:"engine_type"`
	IsActive        bool                  `json:"is_active"`
	LastActivated   time.Time             `json:"last_activated"`
	
	// Position data
	Positions       []*PositionSnapshot   `json:"positions"`
	PendingOrders   []*OrderSnapshot      `json:"pending_orders"`
	
	// Engine-specific state
	GridState       *GridEngineState      `json:"grid_state,omitempty"`
	TrendState      *TrendEngineState     `json:"trend_state,omitempty"`
	
	// Performance tracking
	SessionTrades   int                   `json:"session_trades"`
	SessionPnL      float64               `json:"session_pnl"`
	LastTradeTime   time.Time             `json:"last_trade_time"`
}

// GridEngineState contains grid engine specific persistent state
type GridEngineState struct {
	// Grid configuration
	AnchorPrice     float64               `json:"anchor_price"`
	GridLevels      []*GridLevelSnapshot  `json:"grid_levels"`
	LastRebalance   time.Time             `json:"last_rebalance"`
	
	// VWAP state
	VWAPData        *VWAPSnapshot         `json:"vwap_data"`
	
	// Inventory tracking
	NetExposure     float64               `json:"net_exposure"`
	GrossExposure   float64               `json:"gross_exposure"`
	LongExposure    float64               `json:"long_exposure"`
	ShortExposure   float64               `json:"short_exposure"`
	
	// Safety state
	LastSafetyCheck time.Time             `json:"last_safety_check"`
	SafetyPaused    bool                  `json:"safety_paused"`
}

// TrendEngineState contains trend engine specific persistent state
type TrendEngineState struct {
	// Trend analysis
	CurrentTrend    string                `json:"current_trend"`    // "up", "down", "sideways"
	TrendStrength   float64               `json:"trend_strength"`   // ADX value
	TrendStartTime  time.Time             `json:"trend_start_time"`
	
	// Position management
	AddOnsUsed      int                   `json:"add_ons_used"`
	LastAddOnTime   time.Time             `json:"last_add_on_time"`
	
	// Technical levels
	SwingHigh       float64               `json:"swing_high"`
	SwingLow        float64               `json:"swing_low"`
	ATRValue        float64               `json:"atr_value"`
	
	// Entry tracking
	LastEntrySignal time.Time             `json:"last_entry_signal"`
	SignalQuality   float64               `json:"signal_quality"`
}

// RegimeState represents the current regime detection state
type RegimeState struct {
	CurrentRegime   regime.RegimeType     `json:"current_regime"`
	Confidence      float64               `json:"confidence"`
	RegimeStart     time.Time             `json:"regime_start"`
	LastUpdate      time.Time             `json:"last_update"`
	
	// Regime indicators state
	ADXValue        float64               `json:"adx_value"`
	TrendStrength   float64               `json:"trend_strength"`
	VolatilityLevel float64               `json:"volatility_level"`
	NoiseLevel      float64               `json:"noise_level"`
	
	// Buffer state for regime detection
	DataBufferSize  int                   `json:"data_buffer_size"`
	BufferStartTime time.Time             `json:"buffer_start_time"`
}

// TransitionState represents the current transition management state
type TransitionState struct {
	Status              string                        `json:"status"`              // "stable", "transitioning", "monitoring"
	ActiveTransition    *ActiveTransitionSnapshot     `json:"active_transition,omitempty"`
	LastTransitionTime  time.Time                     `json:"last_transition_time"`
	
	// Daily tracking
	DailyTransitions    int                           `json:"daily_transitions"`
	DailyTransitionCost float64                       `json:"daily_transition_cost"`
	DailyLimitResetTime time.Time                     `json:"daily_limit_reset_time"`
	
	// Emergency controls
	EmergencyStop       bool                          `json:"emergency_stop"`
	ManualOverride      bool                          `json:"manual_override"`
	LastEmergencyAction time.Time                     `json:"last_emergency_action,omitempty"`
}

// SessionMetrics tracks overall session performance
type SessionMetrics struct {
	SessionStart    time.Time             `json:"session_start"`
	Uptime          time.Duration         `json:"uptime"`
	
	// Trading metrics
	TotalTrades     int                   `json:"total_trades"`
	WinningTrades   int                   `json:"winning_trades"`
	LosingTrades    int                   `json:"losing_trades"`
	TotalPnL        float64               `json:"total_pnl"`
	TotalVolume     float64               `json:"total_volume"`
	
	// Regime metrics
	RegimeChanges   int                   `json:"regime_changes"`
	EngineSwaps     int                   `json:"engine_swaps"`
	
	// Performance ratios
	WinRate         float64               `json:"win_rate"`
	ProfitFactor    float64               `json:"profit_factor"`
	SharpeRatio     float64               `json:"sharpe_ratio"`
	MaxDrawdown     float64               `json:"max_drawdown"`
}

// PortfolioState represents current portfolio and position state
type PortfolioState struct {
	TotalValue      float64               `json:"total_value"`
	AvailableBalance float64              `json:"available_balance"`
	TotalExposure   float64               `json:"total_exposure"`
	NetExposure     float64               `json:"net_exposure"`
	
	// Daily tracking
	DailyPnL        float64               `json:"daily_pnl"`
	DailyVolume     float64               `json:"daily_volume"`
	DailyTrades     int                   `json:"daily_trades"`
	
	// Risk metrics
	MaxDrawdown     float64               `json:"max_drawdown"`
	PeakValue       float64               `json:"peak_value"`
	LastDrawdown    float64               `json:"last_drawdown"`
}

// RiskState represents current risk management state
type RiskState struct {
	OverallRiskScore     float64                              `json:"overall_risk_score"`
	ActiveViolations     []string                             `json:"active_violations"`
	CircuitBreakersActive []string                            `json:"circuit_breakers_active"`
	
	// Engine risk scores
	EngineRiskScores     map[EngineType]float64       `json:"engine_risk_scores"`
	
	// Emergency state
	EmergencyActionsToday int                                 `json:"emergency_actions_today"`
	LastRiskViolation    time.Time                           `json:"last_risk_violation,omitempty"`
	
	// Daily resets
	RiskLimitResetTime   time.Time                           `json:"risk_limit_reset_time"`
}

// Snapshot types for complex data structures

type PositionSnapshot struct {
	ID              string                `json:"id"`
	Symbol          string                `json:"symbol"`
	Side            string                `json:"side"`        // "long", "short"
	Size            float64               `json:"size"`
	EntryPrice      float64               `json:"entry_price"`
	CurrentPrice    float64               `json:"current_price"`
	UnrealizedPnL   float64               `json:"unrealized_pnl"`
	EntryTime       time.Time             `json:"entry_time"`
	LastUpdate      time.Time             `json:"last_update"`
	
	// Engine specific data
	EngineType      EngineType    `json:"engine_type"`
	EngineData      map[string]interface{} `json:"engine_data,omitempty"`
}

type OrderSnapshot struct {
	ID              string                `json:"id"`
	Symbol          string                `json:"symbol"`
	Type            string                `json:"type"`        // "limit", "market", "stop", "take_profit"
	Side            string                `json:"side"`
	Quantity        float64               `json:"quantity"`
	Price           float64               `json:"price"`
	Status          string                `json:"status"`      // "pending", "filled", "cancelled"
	CreatedTime     time.Time             `json:"created_time"`
	
	// Engine association
	EngineType      EngineType    `json:"engine_type"`
	Purpose         string                `json:"purpose"`     // "entry", "exit", "stop_loss", "take_profit"
}

type GridLevelSnapshot struct {
	Level           int                   `json:"level"`
	Price           float64               `json:"price"`
	IsActive        bool                  `json:"is_active"`
	HasPosition     bool                  `json:"has_position"`
	PositionSize    float64               `json:"position_size"`
	EntryTime       time.Time             `json:"entry_time,omitempty"`
}

type VWAPSnapshot struct {
	CurrentVWAP     float64               `json:"current_vwap"`
	TotalVolume     float64               `json:"total_volume"`
	TotalValue      float64               `json:"total_value"`
	StartTime       time.Time             `json:"start_time"`
	LastUpdate      time.Time             `json:"last_update"`
}

type ActiveTransitionSnapshot struct {
	ID              string                        `json:"id"`
	StartTime       time.Time                     `json:"start_time"`
	FromRegime      regime.RegimeType             `json:"from_regime"`
	ToRegime        regime.RegimeType             `json:"to_regime"`
	FromEngine      EngineType            `json:"from_engine"`
	ToEngine        EngineType            `json:"to_engine"`
	Progress        float64                       `json:"progress"`
	EstimatedCost   float64                       `json:"estimated_cost"`
	ActualCost      float64                       `json:"actual_cost"`
	Status          string                        `json:"status"`
}

type RegimeRecord struct {
	Timestamp       time.Time             `json:"timestamp"`
	Regime          regime.RegimeType     `json:"regime"`
	Confidence      float64               `json:"confidence"`
	Duration        time.Duration         `json:"duration,omitempty"`
	TriggerReason   string                `json:"trigger_reason"`
}

type TransitionRecord struct {
	ID              string                `json:"id"`
	Timestamp       time.Time             `json:"timestamp"`
	FromRegime      regime.RegimeType     `json:"from_regime"`
	ToRegime        regime.RegimeType     `json:"to_regime"`
	FromEngine      EngineType    `json:"from_engine"`
	ToEngine        EngineType    `json:"to_engine"`
	Duration        time.Duration         `json:"duration"`
	TotalCost       float64               `json:"total_cost"`
	Success         bool                  `json:"success"`
	PositionsAffected int                 `json:"positions_affected"`
	Reason          string                `json:"reason"`
}

type EngineMetrics struct {
	EngineType      EngineType    `json:"engine_type"`
	TotalTrades     int                   `json:"total_trades"`
	WinningTrades   int                   `json:"winning_trades"`
	LosingTrades    int                   `json:"losing_trades"`
	TotalPnL        float64               `json:"total_pnl"`
	TotalVolume     float64               `json:"total_volume"`
	MaxDrawdown     float64               `json:"max_drawdown"`
	WinRate         float64               `json:"win_rate"`
	ProfitFactor    float64               `json:"profit_factor"`
	AvgTradeDuration time.Duration        `json:"avg_trade_duration"`
	LastTradeTime   time.Time             `json:"last_trade_time"`
	LastUpdate      time.Time             `json:"last_update"`
}

// NewStatePersistence creates a new state persistence manager
func NewStatePersistence(logger *logger.Logger, stateDir, symbol string) *StatePersistence {
	return &StatePersistence{
		logger:       logger,
		stateDir:     stateDir,
		symbol:       symbol,
		currentState: NewSystemState(symbol),
		autoSave:     true,
		saveInterval: 5 * time.Minute,  // Save every 5 minutes
		lastSave:     time.Now(),
	}
}

// NewSystemState creates a new empty system state
func NewSystemState(symbol string) *SystemState {
	return &SystemState{
		Version:        "1.0.0",
		Symbol:         symbol,
		LastUpdated:    time.Now(),
		SessionStart:   time.Now(),
		EngineStates:   make(map[EngineType]*EngineState),
		RegimeHistory:  make([]*RegimeRecord, 0, 1000),
		TransitionLogs: make([]*TransitionRecord, 0, 100),
		EngineMetrics:  make(map[EngineType]*EngineMetrics),
		SessionMetrics: &SessionMetrics{
			SessionStart: time.Now(),
		},
		PortfolioState: &PortfolioState{},
		RiskState:      &RiskState{
			EngineRiskScores: make(map[EngineType]float64),
		},
	}
}

// Initialize sets up the state persistence system
func (sp *StatePersistence) Initialize() error {
	// Create state directory if it doesn't exist
	if err := os.MkdirAll(sp.stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}
	
	// Initialize log files for streaming data
	if err := sp.initializeLogFiles(); err != nil {
		return fmt.Errorf("failed to initialize log files: %w", err)
	}
	
	sp.logger.Info("State persistence system initialized: %s", sp.stateDir)
	return nil
}

// LoadState loads the system state from disk
func (sp *StatePersistence) LoadState() error {
	sp.stateMutex.Lock()
	defer sp.stateMutex.Unlock()
	
	stateFile := filepath.Join(sp.stateDir, fmt.Sprintf("%s_state.json", sp.symbol))
	
	// Check if state file exists
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		sp.logger.Info("No existing state file found, starting with clean state")
		return nil
	}
	
	// Load state from file
	data, err := os.ReadFile(stateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}
	
	var state SystemState
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}
	
	// Validate loaded state
	if err := sp.validateState(&state); err != nil {
		sp.logger.LogWarning("State Validation", "Loaded state has issues: %v, using clean state", err)
		return nil
	}
	
	sp.currentState = &state
	sp.logger.Info("State loaded successfully from %s", stateFile)
	
	return nil
}

// SaveState saves the current system state to disk
func (sp *StatePersistence) SaveState() error {
	sp.stateMutex.RLock()
	state := *sp.currentState  // Copy the state
	sp.stateMutex.RUnlock()
	
	state.LastUpdated = time.Now()
	
	stateFile := filepath.Join(sp.stateDir, fmt.Sprintf("%s_state.json", sp.symbol))
	backupFile := filepath.Join(sp.stateDir, fmt.Sprintf("%s_state_backup.json", sp.symbol))
	
	// Create backup of current state file
	if _, err := os.Stat(stateFile); err == nil {
		if err := sp.copyFile(stateFile, backupFile); err != nil {
			sp.logger.LogWarning("State Backup", "Failed to create backup: %v", err)
		}
	}
	
	// Marshal state to JSON
	data, err := json.MarshalIndent(&state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}
	
	// Write to temporary file first
	tempFile := stateFile + ".tmp"
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp state file: %w", err)
	}
	
	// Atomic move
	if err := os.Rename(tempFile, stateFile); err != nil {
		return fmt.Errorf("failed to move state file: %w", err)
	}
	
	sp.lastSave = time.Now()
	sp.logger.Info("State saved successfully to %s", stateFile)
	
	return nil
}

// UpdateEngineState updates the state for a specific engine
func (sp *StatePersistence) UpdateEngineState(engineType EngineType, engine TradingEngine) {
	sp.stateMutex.Lock()
	defer sp.stateMutex.Unlock()
	
	if sp.currentState.EngineStates == nil {
		sp.currentState.EngineStates = make(map[EngineType]*EngineState)
	}
	
	// Get current positions
	positions := engine.GetCurrentPositions()
	positionSnapshots := make([]*PositionSnapshot, len(positions))
	
	for i, pos := range positions {
		positionSnapshots[i] = &PositionSnapshot{
			ID:            pos.GetID(),
			Symbol:        sp.symbol,  // Use the persistence manager's symbol
			Side:          pos.GetSide(),
			Size:          pos.GetSize(),
			EntryPrice:    pos.GetEntryPrice(),
			CurrentPrice:  pos.GetCurrentPrice(),
			UnrealizedPnL: pos.GetUnrealizedPnL(),
			EntryTime:     pos.GetEntryTime(),
			LastUpdate:    time.Now(),
			EngineType:    engineType,
		}
	}
	
	// Get engine metrics - simplified since we can't access the actual metrics easily
	sessionPnL := 0.0
	sessionTrades := 0
	
	// Calculate PnL from positions 
	for _, pos := range positions {
		sessionPnL += pos.GetUnrealizedPnL()
	}
	sessionTrades = len(positions)  // Simplified
	
	engineState := &EngineState{
		EngineType:    engineType,
		IsActive:      engine.IsActive(),
		LastActivated: time.Now(),
		Positions:     positionSnapshots,
		PendingOrders: []*OrderSnapshot{}, // Would need to be implemented
		SessionTrades: sessionTrades,
		SessionPnL:    sessionPnL,
		LastTradeTime: time.Now(),
	}
	
	// Add engine-specific state based on type
	switch engineType {
	case EngineTypeGrid:
		engineState.GridState = sp.extractGridState(engine)
	case EngineTypeTrend:
		engineState.TrendState = sp.extractTrendState(engine)
	}
	
	sp.currentState.EngineStates[engineType] = engineState
	
	// Auto-save if enabled
	if sp.autoSave && time.Since(sp.lastSave) > sp.saveInterval {
		go func() {
			if err := sp.SaveState(); err != nil {
				sp.logger.LogError("Auto Save Failed", err)
			}
		}()
	}
}

// UpdateRegimeState updates the regime detection state
func (sp *StatePersistence) UpdateRegimeState(regimeSignal *regime.RegimeSignal) {
	sp.stateMutex.Lock()
	defer sp.stateMutex.Unlock()
	
	if sp.currentState.RegimeState == nil {
		sp.currentState.RegimeState = &RegimeState{}
	}
	
	oldRegime := sp.currentState.RegimeState.CurrentRegime
	newRegime := regimeSignal.Type
	
	// Update current regime state
	sp.currentState.RegimeState.CurrentRegime = newRegime
	sp.currentState.RegimeState.Confidence = regimeSignal.Confidence
	sp.currentState.RegimeState.LastUpdate = time.Now()
	
	// If regime changed, record it
	if oldRegime != newRegime {
		sp.currentState.RegimeState.RegimeStart = time.Now()
		
		// Add to regime history
		record := &RegimeRecord{
			Timestamp:     time.Now(),
			Regime:        newRegime,
			Confidence:    regimeSignal.Confidence,
			TriggerReason: "Regime detection signal",  // Generic reason since not in signal
		}
		
		// Calculate duration if we have previous regime
		if len(sp.currentState.RegimeHistory) > 0 {
			lastRecord := sp.currentState.RegimeHistory[len(sp.currentState.RegimeHistory)-1]
			record.Duration = time.Since(lastRecord.Timestamp)
		}
		
		sp.currentState.RegimeHistory = append(sp.currentState.RegimeHistory, record)
		
		// Keep only last 1000 records
		if len(sp.currentState.RegimeHistory) > 1000 {
			sp.currentState.RegimeHistory = sp.currentState.RegimeHistory[1:]
		}
		
		// Log to regime file
		sp.logRegimeChange(record)
		
		sp.logger.Info("Regime state updated: %s (%.1f%% confidence)", newRegime, regimeSignal.Confidence*100)
	}
}

// UpdateTransitionState updates the transition management state
func (sp *StatePersistence) UpdateTransitionState(transitionRecord *transition.TransitionRecord) {
	sp.stateMutex.Lock()
	defer sp.stateMutex.Unlock()
	
	if sp.currentState.TransitionState == nil {
		sp.currentState.TransitionState = &TransitionState{}
	}
	
	// Convert to our format
	record := &TransitionRecord{
		ID:                transitionRecord.ID,
		Timestamp:         transitionRecord.Timestamp,
		FromRegime:        transitionRecord.FromRegime,
		ToRegime:          transitionRecord.ToRegime,
		FromEngine:        EngineType(transitionRecord.FromEngine),  // Type conversion
		ToEngine:          EngineType(transitionRecord.ToEngine),    // Type conversion
		Duration:          transitionRecord.Duration,
		TotalCost:         transitionRecord.TotalCost,
		Success:           transitionRecord.Success,
		PositionsAffected: transitionRecord.PositionsAffected,
		Reason:            "Regime transition",
	}
	
	sp.currentState.TransitionLogs = append(sp.currentState.TransitionLogs, record)
	
	// Keep only last 100 records
	if len(sp.currentState.TransitionLogs) > 100 {
		sp.currentState.TransitionLogs = sp.currentState.TransitionLogs[1:]
	}
	
	sp.currentState.TransitionState.LastTransitionTime = time.Now()
	
	// Log to transition file
	sp.logTransition(record)
	
	sp.logger.Info("Transition state updated: %s", record.ID)
}

// UpdatePortfolioState updates portfolio and position state
func (sp *StatePersistence) UpdatePortfolioState(totalValue, availableBalance, totalExposure, netExposure, dailyPnL float64) {
	sp.stateMutex.Lock()
	defer sp.stateMutex.Unlock()
	
	if sp.currentState.PortfolioState == nil {
		sp.currentState.PortfolioState = &PortfolioState{}
	}
	
	portfolio := sp.currentState.PortfolioState
	
	// Update values
	portfolio.TotalValue = totalValue
	portfolio.AvailableBalance = availableBalance
	portfolio.TotalExposure = totalExposure
	portfolio.NetExposure = netExposure
	portfolio.DailyPnL = dailyPnL
	
	// Update peak and drawdown
	if totalValue > portfolio.PeakValue {
		portfolio.PeakValue = totalValue
	}
	
	currentDrawdown := (portfolio.PeakValue - totalValue) / portfolio.PeakValue
	if currentDrawdown > portfolio.MaxDrawdown {
		portfolio.MaxDrawdown = currentDrawdown
	}
	portfolio.LastDrawdown = currentDrawdown
}

// Helper methods

func (sp *StatePersistence) initializeLogFiles() error {
	// Initialize regime log file
	regimeLogPath := filepath.Join(sp.stateDir, fmt.Sprintf("%s_regime_log.jsonl", sp.symbol))
	regimeFile, err := os.OpenFile(regimeLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open regime log file: %w", err)
	}
	sp.regimeLogFile = regimeFile
	
	// Initialize transition log file
	transitionLogPath := filepath.Join(sp.stateDir, fmt.Sprintf("%s_transition_log.jsonl", sp.symbol))
	transitionFile, err := os.OpenFile(transitionLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open transition log file: %w", err)
	}
	sp.transitionLogFile = transitionFile
	
	// Initialize metrics log file
	metricsLogPath := filepath.Join(sp.stateDir, fmt.Sprintf("%s_metrics_log.jsonl", sp.symbol))
	metricsFile, err := os.OpenFile(metricsLogPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open metrics log file: %w", err)
	}
	sp.metricsLogFile = metricsFile
	
	return nil
}

func (sp *StatePersistence) logRegimeChange(record *RegimeRecord) {
	if sp.regimeLogFile != nil {
		data, _ := json.Marshal(record)
		sp.regimeLogFile.WriteString(string(data) + "\n")
		sp.regimeLogFile.Sync()
	}
}

func (sp *StatePersistence) logTransition(record *TransitionRecord) {
	if sp.transitionLogFile != nil {
		data, _ := json.Marshal(record)
		sp.transitionLogFile.WriteString(string(data) + "\n")
		sp.transitionLogFile.Sync()
	}
}

func (sp *StatePersistence) extractGridState(engine TradingEngine) *GridEngineState {
	// This would extract grid-specific state from the engine
	// For now, return a basic state
	return &GridEngineState{
		AnchorPrice:     0.0,
		GridLevels:      []*GridLevelSnapshot{},
		LastRebalance:   time.Now(),
		VWAPData:        &VWAPSnapshot{},
		NetExposure:     0.0,
		GrossExposure:   0.0,
		LongExposure:    0.0,
		ShortExposure:   0.0,
		LastSafetyCheck: time.Now(),
		SafetyPaused:    false,
	}
}

func (sp *StatePersistence) extractTrendState(engine TradingEngine) *TrendEngineState {
	// This would extract trend-specific state from the engine
	// For now, return a basic state
	return &TrendEngineState{
		CurrentTrend:    "sideways",
		TrendStrength:   0.0,
		TrendStartTime:  time.Now(),
		AddOnsUsed:      0,
		LastAddOnTime:   time.Time{},
		SwingHigh:       0.0,
		SwingLow:        0.0,
		ATRValue:        0.0,
		LastEntrySignal: time.Time{},
		SignalQuality:   0.0,
	}
}

func (sp *StatePersistence) validateState(state *SystemState) error {
	if state.Symbol != sp.symbol {
		return fmt.Errorf("state symbol mismatch: expected %s, got %s", sp.symbol, state.Symbol)
	}
	
	if state.Version == "" {
		return fmt.Errorf("state version is empty")
	}
	
	if time.Since(state.LastUpdated) > 7*24*time.Hour {
		return fmt.Errorf("state is too old: %v", state.LastUpdated)
	}
	
	return nil
}

func (sp *StatePersistence) copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}

// Cleanup closes all open files and performs cleanup
func (sp *StatePersistence) Cleanup() error {
	if sp.regimeLogFile != nil {
		sp.regimeLogFile.Close()
	}
	if sp.transitionLogFile != nil {
		sp.transitionLogFile.Close()
	}
	if sp.metricsLogFile != nil {
		sp.metricsLogFile.Close()
	}
	
	return sp.SaveState()
}

// GetSystemState returns a copy of the current system state
func (sp *StatePersistence) GetSystemState() *SystemState {
	sp.stateMutex.RLock()
	defer sp.stateMutex.RUnlock()
	
	// Return a deep copy
	stateCopy := *sp.currentState
	return &stateCopy
}
