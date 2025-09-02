package regime

import "time"

// RegimeChange represents a regime transition event
type RegimeChange struct {
	Timestamp    time.Time   `json:"timestamp"`
	OldRegime    RegimeType  `json:"old_regime"`
	NewRegime    RegimeType  `json:"new_regime"`
	Confidence   float64     `json:"confidence"`
	Reason       string      `json:"reason"`       // Human-readable reason for change
	TriggerPrice float64     `json:"trigger_price"` // Price when change occurred
}

// RegimeMetrics holds detailed metrics for regime analysis
type RegimeMetrics struct {
	// Trend indicators
	EMADistance      float64 `json:"ema_distance"`       // Distance between EMAs
	ADXValue         float64 `json:"adx_value"`          // Trend strength
	TrendDirection   int     `json:"trend_direction"`    // 1=up, -1=down, 0=sideways
	TrendStrength    float64 `json:"trend_strength"`     // Combined trend strength (0-1)
	
	// Volatility indicators  
	ATRNormalized    float64 `json:"atr_normalized"`     // Normalized ATR
	BBWidth          float64 `json:"bb_width"`           // Bollinger Band width
	VolatilityRank   float64 `json:"volatility_rank"`    // 0-1 volatility ranking
	Volatility       float64 `json:"volatility"`         // Combined volatility measure (0-1)
	
	// Noise indicators
	RSINoiseScore    float64 `json:"rsi_noise_score"`    // RSI-based noise measure
	PriceEfficiency  float64 `json:"price_efficiency"`   // Directional movement efficiency
	ChoppinessIndex  float64 `json:"choppiness_index"`   // Market choppiness measure
	NoiseLevel       float64 `json:"noise_level"`        // Combined noise level (0-1)
	
	// Breakout indicators
	DonchianBreakout bool    `json:"donchian_breakout"`  // Breakout above/below channels
	VolumeSpike      float64 `json:"volume_spike"`       // Volume relative to average
	MomentumShift    float64 `json:"momentum_shift"`     // Rate of momentum change
	DonchianBreakoutStrength float64 `json:"donchian_breakout_strength"` // Strength of breakout
}

// RegimeConfig holds configuration parameters for regime detection
type RegimeConfig struct {
	// Trend detection parameters from plan
	EmaPeriods           []int   `json:"ema_periods"`            // [50, 200]
	AdxPeriod            int     `json:"adx_period"`             // 14
	AdxTrendThreshold    float64 `json:"adx_trend_threshold"`    // 20
	EmaDistanceThreshold float64 `json:"ema_distance_threshold"` // 0.005
	DonchianPeriod       int     `json:"donchian_period"`        // 20
	
	// Volatility assessment parameters
	AtrPeriod            int     `json:"atr_period"`             // 14
	BbPeriod             int     `json:"bb_period"`              // 20
	BbStdDev             float64 `json:"bb_std_dev"`             // 2.0
	VolatilityNormalization string `json:"volatility_normalization"` // "price_percentage"
	
	// Noise detection parameters
	RsiPeriod            int       `json:"rsi_period"`            // 14
	RsiNoiseRange        [2]float64 `json:"rsi_noise_range"`      // [45, 55]
	NoiseBarsThreshold   int       `json:"noise_bars_threshold"`  // 8
	
	// Hysteresis parameters
	ConfirmationBars     int       `json:"confirmation_bars"`     // 3
	RegimeSwitchCooldown int       `json:"regime_switch_cooldown"` // 2
}

// DefaultRegimeConfig returns default configuration matching the plan
func DefaultRegimeConfig() *RegimeConfig {
	return &RegimeConfig{
		EmaPeriods:              []int{50, 200},
		AdxPeriod:               14,
		AdxTrendThreshold:       20.0,
		EmaDistanceThreshold:    0.005,
		DonchianPeriod:          20,
		AtrPeriod:               14,
		BbPeriod:                20,
		BbStdDev:                2.0,
		VolatilityNormalization: "price_percentage",
		RsiPeriod:               14,
		RsiNoiseRange:           [2]float64{45.0, 55.0},
		NoiseBarsThreshold:      8,
		ConfirmationBars:        3,
		RegimeSwitchCooldown:    2,
	}
}

// RegimeAnalytics provides performance analytics for regime detection
type RegimeAnalytics struct {
	TotalRegimeChanges   int                    `json:"total_regime_changes"`
	RegimeAccuracy       float64                `json:"regime_accuracy"`        // % of correct classifications
	AverageConfidence    float64                `json:"average_confidence"`
	FalseSignalRate      float64                `json:"false_signal_rate"`      // % of false regime switches
	RegimeDistribution   map[RegimeType]float64 `json:"regime_distribution"`    // % time in each regime
	AverageRegimeDuration time.Duration         `json:"average_regime_duration"`
	LastAnalyzed         time.Time              `json:"last_analyzed"`
}

// RegimeCallback defines the interface for regime change notifications
type RegimeCallback interface {
	OnRegimeChange(change *RegimeChange) error
}

// RegimeSubscriber allows components to subscribe to regime changes
type RegimeSubscriber struct {
	ID       string
	Callback RegimeCallback
}

// RegimeEventBus manages regime change notifications
type RegimeEventBus struct {
	subscribers map[string]*RegimeSubscriber
}

// NewRegimeEventBus creates a new event bus for regime notifications
func NewRegimeEventBus() *RegimeEventBus {
	return &RegimeEventBus{
		subscribers: make(map[string]*RegimeSubscriber),
	}
}

// Subscribe adds a new subscriber for regime changes
func (bus *RegimeEventBus) Subscribe(id string, callback RegimeCallback) {
	bus.subscribers[id] = &RegimeSubscriber{
		ID:       id,
		Callback: callback,
	}
}

// Unsubscribe removes a subscriber
func (bus *RegimeEventBus) Unsubscribe(id string) {
	delete(bus.subscribers, id)
}

// PublishRegimeChange notifies all subscribers of a regime change
func (bus *RegimeEventBus) PublishRegimeChange(change *RegimeChange) {
	for _, subscriber := range bus.subscribers {
		// Fire and forget - don't block on subscriber errors
		go func(sub *RegimeSubscriber) {
			if err := sub.Callback.OnRegimeChange(change); err != nil {
				// TODO: Add logging for subscriber errors
				// For now, just continue
			}
		}(subscriber)
	}
}
