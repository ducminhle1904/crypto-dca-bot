package engines

import (
	"context"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// TradingEngine defines the common interface for all trading engines
// This allows the orchestrator to manage different engines uniformly
type TradingEngine interface {
	// Basic identification
	GetType() EngineType
	GetName() string
	
	// Market Analysis
	AnalyzeMarket(ctx context.Context, data30m, data5m []types.OHLCV, currentRegime regime.RegimeType) (EngineSignal, error)
	AnalyzeMarketWithData(ctx context.Context, data map[string][]types.OHLCV, currentRegime regime.RegimeType) (EngineSignal, error) // For multi-timeframe analysis
	
	// Position Management  
	GetCurrentPositions() []EnginePosition
	ManagePositions(ctx context.Context, currentData []types.OHLCV) error
	ShouldClosePosition(position EnginePosition, currentPrice float64, regime regime.RegimeType) bool
	
	// Regime Compatibility
	IsCompatibleWithRegime(regimeType regime.RegimeType) bool
	GetRegimeCompatibilityScore(regimeType regime.RegimeType) float64
	GetPreferredRegimes() []regime.RegimeType
	
	// Risk Management
	CalculatePositionSize(balance float64, price float64, currentRegime regime.RegimeType) float64
	ValidateSignal(signal EngineSignal, currentPositions []EnginePosition) error
	
	// Performance and Metrics
	GetPerformanceMetrics() EngineMetrics
	GetEngineStatus() EngineStatus
	
	// Configuration and Control
	UpdateConfiguration(config map[string]interface{}) error
	SetRiskLimits(limits EngineRiskLimits) error
	
	// Lifecycle Management
	Initialize() error
	Start() error
	Stop() error
	Reset() error
	
	// State management
	IsActive() bool
	SetActive(active bool)
}

// EngineSignal represents a generic trading signal from any engine
type EngineSignal interface {
	GetTimestamp() time.Time
	GetConfidence() float64
	GetStrength() float64
	GetDirection() int // 1 = bullish, -1 = bearish, 0 = neutral
	GetSignalType() string
	GetAction() string // "BUY", "SELL", "HOLD", "SCALE_IN", "SCALE_OUT", "CLOSE"
	GetMetadata() map[string]interface{}
}

// EnginePosition represents a generic position from any engine
type EnginePosition interface {
	GetID() string
	GetEngineType() string
	GetSide() string // "long" or "short"
	GetSize() float64
	GetEntryPrice() float64
	GetCurrentPrice() float64
	GetUnrealizedPnL() float64
	GetEntryTime() time.Time
	GetMetadata() map[string]interface{}
}

// EngineMetrics represents performance metrics for any engine
type EngineMetrics interface {
	GetActivePositions() int
	GetTotalTrades() int
	GetWinRate() float64
	GetProfitFactor() float64
	GetTotalPnL() float64
	GetMaxDrawdown() float64
	GetSharpeRatio() float64
	GetLastUpdated() time.Time
	GetEngineSpecificMetrics() map[string]interface{}
}

// EngineStatus represents the current operational status of an engine
type EngineStatus struct {
	IsActive           bool              `json:"is_active"`
	IsTrading          bool              `json:"is_trading"`
	LastActivity       time.Time         `json:"last_activity"`
	ErrorCount         int               `json:"error_count"`
	LastError          string            `json:"last_error,omitempty"`
	ActivePositions    int               `json:"active_positions"`
	PendingOrders      int               `json:"pending_orders"`
	CurrentExposure    float64           `json:"current_exposure"`
	RiskUtilization    float64           `json:"risk_utilization"`    // % of risk limits used
	
	// Engine-specific status
	EngineSpecificData map[string]interface{} `json:"engine_specific_data,omitempty"`
}

// EngineRiskLimits defines risk management parameters for engines
type EngineRiskLimits struct {
	MaxPositionSize    float64           `json:"max_position_size"`
	MaxDailyTrades     int               `json:"max_daily_trades"`
	MaxDailyLoss       float64           `json:"max_daily_loss"`
	MaxExposure        float64           `json:"max_exposure"`
	MaxCorrelation     float64           `json:"max_correlation,omitempty"`
	
	// Engine-specific limits
	CustomLimits       map[string]interface{} `json:"custom_limits,omitempty"`
}

// EngineFactory creates trading engine instances
type EngineFactory interface {
	CreateTrendEngine(config map[string]interface{}) (TradingEngine, error)
	CreateGridEngine(config map[string]interface{}) (TradingEngine, error)
	GetAvailableEngines() []string
}

// EngineType represents different types of trading engines
type EngineType string

const (
	EngineTypeTrend EngineType = "trend"
	EngineTypeGrid  EngineType = "grid"
	EngineTypeDCA   EngineType = "dca"   // For compatibility with existing system
)

func (e EngineType) String() string {
	return string(e)
}

// EngineConfiguration holds configuration for any engine
type EngineConfiguration struct {
	Type               EngineType        `json:"type"`
	Name               string            `json:"name"`
	Enabled            bool              `json:"enabled"`
	Priority           int               `json:"priority"`           // For engine selection
	RiskLimits         EngineRiskLimits  `json:"risk_limits"`
	
	// Regime preferences
	PreferredRegimes   []regime.RegimeType `json:"preferred_regimes"`
	RegimeWeights      map[regime.RegimeType]float64 `json:"regime_weights"`
	
	// Engine-specific configuration
	Parameters         map[string]interface{} `json:"parameters"`
}

// DefaultEngineConfiguration returns default configuration for an engine type
func DefaultEngineConfiguration(engineType EngineType) *EngineConfiguration {
	switch engineType {
	case EngineTypeTrend:
		return &EngineConfiguration{
			Type:     EngineTypeTrend,
			Name:     "Trend Following Engine",
			Enabled:  true,
			Priority: 1,
			RiskLimits: EngineRiskLimits{
				MaxPositionSize: 0.3,
				MaxDailyTrades:  10,
				MaxDailyLoss:    0.02,
				MaxExposure:     0.5,
			},
			PreferredRegimes: []regime.RegimeType{regime.RegimeTrending},
			RegimeWeights: map[regime.RegimeType]float64{
				regime.RegimeTrending: 1.0,
				regime.RegimeRanging:  0.2,
				regime.RegimeVolatile: 0.1,
			},
			Parameters: map[string]interface{}{
				"bias_timeframe":        "30m",
				"execution_timeframes":  []string{"5m", "1m"},
				"atr_multiplier":        1.2,
				"max_add_ons":           2,
			},
		}
		
	case EngineTypeGrid:
		return &EngineConfiguration{
			Type:     EngineTypeGrid,
			Name:     "Grid Trading Engine",
			Enabled:  true,
			Priority: 2,
			RiskLimits: EngineRiskLimits{
				MaxPositionSize: 0.8,
				MaxDailyTrades:  50,
				MaxDailyLoss:    0.015,
				MaxExposure:     0.8,
			},
			PreferredRegimes: []regime.RegimeType{regime.RegimeRanging},
			RegimeWeights: map[regime.RegimeType]float64{
				regime.RegimeTrending: 0.1,
				regime.RegimeRanging:  1.0,
				regime.RegimeVolatile: 0.3,
			},
			Parameters: map[string]interface{}{
				"max_bands":             4,
				"grid_spacing_multiplier": 0.25,
				"max_net_exposure":      0.1,
				"symmetric_placement":   true,
			},
		}
		
	default:
		return nil
	}
}

// EngineTransitionInfo contains information needed for engine transitions
type EngineTransitionInfo struct {
	FromEngine         EngineType        `json:"from_engine"`
	ToEngine           EngineType        `json:"to_engine"`
	RegimeChange       *regime.RegimeChange `json:"regime_change"`
	CurrentPositions   []EnginePosition  `json:"current_positions"`
	TransitionCost     float64           `json:"estimated_transition_cost"`
	TransitionTime     time.Duration     `json:"estimated_transition_time"`
	PositionCompatibility []PositionCompatibility `json:"position_compatibility"`
}

// PositionCompatibility indicates how well a position fits with the new engine
type PositionCompatibility struct {
	PositionID         string            `json:"position_id"`
	CompatibilityScore float64           `json:"compatibility_score"` // 0-1
	RecommendedAction  string            `json:"recommended_action"`  // "keep", "close", "convert"
	ConversionPlan     *PositionConversionPlan `json:"conversion_plan,omitempty"`
}

// PositionConversionPlan describes how to convert a position from one engine to another
type PositionConversionPlan struct {
	OriginalPosition   EnginePosition    `json:"original_position"`
	TargetConfiguration map[string]interface{} `json:"target_configuration"`
	ConversionSteps    []ConversionStep  `json:"conversion_steps"`
	EstimatedCost      float64           `json:"estimated_cost"`
	EstimatedTime      time.Duration     `json:"estimated_time"`
}

// ConversionStep represents a single step in position conversion
type ConversionStep struct {
	Action             string            `json:"action"`      // "close", "modify", "create"
	Parameters         map[string]interface{} `json:"parameters"`
	Order              int               `json:"order"`       // Execution order
	Dependencies       []string          `json:"dependencies,omitempty"` // Other steps that must complete first
}

// Concrete implementations for easier development

// BasicEngineSignal is a concrete implementation of EngineSignal
type BasicEngineSignal struct {
	Timestamp    time.Time              `json:"timestamp"`
	Confidence   float64                `json:"confidence"`
	Strength     float64                `json:"strength"`
	Direction    int                    `json:"direction"`    // 1 = bullish, -1 = bearish, 0 = neutral
	SignalType   string                 `json:"signal_type"`
	Action       string                 `json:"action"`       // "BUY", "SELL", "HOLD", "SCALE_IN", "SCALE_OUT", "CLOSE"
	Price        float64                `json:"price"`        // Suggested price (0 = market)
	Size         float64                `json:"size"`         // Position size
	StopLoss     float64                `json:"stop_loss"`
	TakeProfit   []float64              `json:"take_profit"`
	Reason       string                 `json:"reason"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// Implement EngineSignal interface
func (s *BasicEngineSignal) GetTimestamp() time.Time { return s.Timestamp }
func (s *BasicEngineSignal) GetConfidence() float64 { return s.Confidence }
func (s *BasicEngineSignal) GetStrength() float64 { return s.Strength }
func (s *BasicEngineSignal) GetDirection() int { return s.Direction }
func (s *BasicEngineSignal) GetSignalType() string { return s.SignalType }
func (s *BasicEngineSignal) GetAction() string { return s.Action }
func (s *BasicEngineSignal) GetMetadata() map[string]interface{} { return s.Metadata }

// BasicEnginePosition is a concrete implementation of EnginePosition
type BasicEnginePosition struct {
	ID            string                 `json:"id"`
	EngineType    string                 `json:"engine_type"`
	Symbol        string                 `json:"symbol"`
	Side          string                 `json:"side"`          // "long" or "short"
	Size          float64                `json:"size"`
	EntryPrice    float64                `json:"entry_price"`
	CurrentPrice  float64                `json:"current_price"`
	UnrealizedPnL float64                `json:"unrealized_pnl"`
	EntryTime     time.Time              `json:"entry_time"`
	LastUpdate    time.Time              `json:"last_update"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// Implement EnginePosition interface
func (p *BasicEnginePosition) GetID() string { return p.ID }
func (p *BasicEnginePosition) GetEngineType() string { return p.EngineType }
func (p *BasicEnginePosition) GetSide() string { return p.Side }
func (p *BasicEnginePosition) GetSize() float64 { return p.Size }
func (p *BasicEnginePosition) GetEntryPrice() float64 { return p.EntryPrice }
func (p *BasicEnginePosition) GetCurrentPrice() float64 { return p.CurrentPrice }
func (p *BasicEnginePosition) GetUnrealizedPnL() float64 { return p.UnrealizedPnL }
func (p *BasicEnginePosition) GetEntryTime() time.Time { return p.EntryTime }
func (p *BasicEnginePosition) GetMetadata() map[string]interface{} { return p.Metadata }

// Update current price and unrealized PnL
func (p *BasicEnginePosition) UpdatePrice(newPrice float64) {
	p.CurrentPrice = newPrice
	p.LastUpdate = time.Now()
	
	// Calculate unrealized PnL
	if p.Side == "long" {
		p.UnrealizedPnL = (newPrice - p.EntryPrice) * p.Size
	} else {
		p.UnrealizedPnL = (p.EntryPrice - newPrice) * p.Size
	}
}

// BasicEngineMetrics is a concrete implementation of EngineMetrics
type BasicEngineMetrics struct {
	ActivePositions        int                    `json:"active_positions"`
	TotalTrades           int                    `json:"total_trades"`
	WinningTrades         int                    `json:"winning_trades"`
	LosingTrades          int                    `json:"losing_trades"`
	WinRate               float64                `json:"win_rate"`
	ProfitFactor          float64                `json:"profit_factor"`
	TotalPnL              float64                `json:"total_pnl"`
	MaxDrawdown           float64                `json:"max_drawdown"`
	SharpeRatio           float64                `json:"sharpe_ratio"`
	LastUpdated           time.Time              `json:"last_updated"`
	EngineSpecificMetrics map[string]interface{} `json:"engine_specific_metrics"`
}

// Implement EngineMetrics interface
func (m *BasicEngineMetrics) GetActivePositions() int { return m.ActivePositions }
func (m *BasicEngineMetrics) GetTotalTrades() int { return m.TotalTrades }
func (m *BasicEngineMetrics) GetWinRate() float64 { return m.WinRate }
func (m *BasicEngineMetrics) GetProfitFactor() float64 { return m.ProfitFactor }
func (m *BasicEngineMetrics) GetTotalPnL() float64 { return m.TotalPnL }
func (m *BasicEngineMetrics) GetMaxDrawdown() float64 { return m.MaxDrawdown }
func (m *BasicEngineMetrics) GetSharpeRatio() float64 { return m.SharpeRatio }
func (m *BasicEngineMetrics) GetLastUpdated() time.Time { return m.LastUpdated }
func (m *BasicEngineMetrics) GetEngineSpecificMetrics() map[string]interface{} { return m.EngineSpecificMetrics }

// Update metrics with new trade data
func (m *BasicEngineMetrics) UpdateTrade(pnl float64, wasWinning bool) {
	m.TotalTrades++
	m.TotalPnL += pnl
	
	if wasWinning {
		m.WinningTrades++
	} else {
		m.LosingTrades++
	}
	
	if m.TotalTrades > 0 {
		m.WinRate = float64(m.WinningTrades) / float64(m.TotalTrades)
	}
	
	m.LastUpdated = time.Now()
}
