package regime

import (
	"fmt"
	"math"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// RegimeType represents different market regimes
type RegimeType int

const (
	RegimeTrending RegimeType = iota
	RegimeRanging
	RegimeVolatile
	RegimeUncertain
)

func (r RegimeType) String() string {
	switch r {
	case RegimeTrending:
		return "TRENDING"
	case RegimeRanging:
		return "RANGING"
	case RegimeVolatile:
		return "VOLATILE"
	case RegimeUncertain:
		return "UNCERTAIN"
	default:
		return "UNKNOWN"
	}
}

// RegimeSignal represents the output of regime detection
type RegimeSignal struct {
	Type           RegimeType    `json:"type"`
	Confidence     float64       `json:"confidence"`     // 0.0 to 1.0
	Timestamp      time.Time     `json:"timestamp"`
	TrendStrength  float64       `json:"trend_strength"` // For trending regimes
	Volatility     float64       `json:"volatility"`     // Normalized volatility
	NoiseLevel     float64       `json:"noise_level"`    // Market noise assessment
	TransitionFlag bool          `json:"transition_flag"` // True if regime is transitioning
}

// RegimeDetector analyzes market conditions and classifies regimes
type RegimeDetector struct {
	// Configuration parameters from plan
	emaPeriods           []int     // [50, 200] for trend detection
	adxPeriod            int       // 14 for trend strength
	adxTrendThreshold    float64   // 20 for trend detection
	emaDistanceThreshold float64   // 0.005 for trend confirmation
	donchianPeriod       int       // 20 for breakout detection
	
	// Volatility assessment
	atrPeriod            int       // 14 for volatility
	bbPeriod             int       // 20 for volatility bands
	bbStdDev             float64   // 2.0 for volatility bands
	
	// Noise detection
	rsiPeriod            int       // 14 for noise assessment
	rsiNoiseRange        [2]float64 // [45, 55] for noise range
	noiseBarsThreshold   int       // 8 bars in noise range
	
	// Hysteresis for regime stability
	confirmationBars     int       // 3 bars confirmation
	regimeSwitchCooldown int       // 2 bar cooldown
	
	// State tracking
	lastRegime           RegimeType
	lastSignal           *RegimeSignal
	confirmationCounter  int
	cooldownCounter      int
	regimeHistory        []*RegimeSignal // For analysis
}

// NewRegimeDetector creates a new regime detector with default parameters from plan
func NewRegimeDetector() *RegimeDetector {
	return &RegimeDetector{
		emaPeriods:           []int{50, 200},
		adxPeriod:            14,
		adxTrendThreshold:    20.0,
		emaDistanceThreshold: 0.005,
		donchianPeriod:       20,
		atrPeriod:            14,
		bbPeriod:             20,
		bbStdDev:             2.0,
		rsiPeriod:            14,
		rsiNoiseRange:        [2]float64{45.0, 55.0},
		noiseBarsThreshold:   8,
		confirmationBars:     3,
		regimeSwitchCooldown: 2,
		lastRegime:           RegimeUncertain,
		regimeHistory:        make([]*RegimeSignal, 0, 100),
	}
}

// DetectRegime analyzes market data and returns regime classification
func (rd *RegimeDetector) DetectRegime(data []types.OHLCV) (*RegimeSignal, error) {
	if len(data) < rd.getMinRequiredPeriods() {
		return nil, fmt.Errorf("insufficient data: need at least %d periods", rd.getMinRequiredPeriods())
	}

	// Calculate all regime detection indicators
	metrics, err := rd.calculateRegimeMetrics(data)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate regime metrics: %w", err)
	}
	
	// Classify regime based on metrics
	regimeType := rd.classifyRegime(metrics)
	
	// Calculate confidence in the classification
	confidence := rd.calculateConfidence(metrics, regimeType)
	
	// Apply hysteresis to prevent rapid regime switching
	finalRegime, isTransition := rd.applyHysteresis(regimeType, confidence)
	
	signal := &RegimeSignal{
		Type:           finalRegime,
		Confidence:     confidence,
		Timestamp:      data[len(data)-1].Timestamp,
		TrendStrength:  metrics.TrendStrength,
		Volatility:     metrics.Volatility,
		NoiseLevel:     metrics.NoiseLevel,
		TransitionFlag: isTransition,
	}
	
	// Update state
	rd.lastRegime = finalRegime
	rd.lastSignal = signal
	rd.regimeHistory = append(rd.regimeHistory, signal)
	
	// Keep history manageable
	if len(rd.regimeHistory) > 1000 {
		rd.regimeHistory = rd.regimeHistory[100:]
	}
	
	return signal, nil
}

// getMinRequiredPeriods returns minimum periods needed for regime detection
func (rd *RegimeDetector) getMinRequiredPeriods() int {
	maxPeriod := rd.emaPeriods[1] // 200 EMA is typically the longest
	if rd.donchianPeriod > maxPeriod {
		maxPeriod = rd.donchianPeriod
	}
	return maxPeriod + rd.confirmationBars
}

// GetCurrentRegime returns the last detected regime
func (rd *RegimeDetector) GetCurrentRegime() RegimeType {
	return rd.lastRegime
}

// GetLastSignal returns the most recent regime signal
func (rd *RegimeDetector) GetLastSignal() *RegimeSignal {
	return rd.lastSignal
}

// GetRegimeHistory returns the recent regime history
func (rd *RegimeDetector) GetRegimeHistory() []*RegimeSignal {
	return rd.regimeHistory
}

// IsRegimeStable checks if the current regime has been stable
func (rd *RegimeDetector) IsRegimeStable() bool {
	if rd.lastSignal == nil {
		return false
	}
	return rd.confirmationCounter >= rd.confirmationBars && rd.lastSignal.Confidence > 0.7
}

// Note: RegimeMetrics is defined in types.go

// calculateRegimeMetrics calculates all indicators needed for regime detection
func (rd *RegimeDetector) calculateRegimeMetrics(data []types.OHLCV) (*RegimeMetrics, error) {
	metrics := &RegimeMetrics{}
	
	// Create indicator instances using the indicators package
	adx := indicators.NewADX(rd.adxPeriod)
	ema50 := indicators.NewEMA(rd.emaPeriods[0])   // 50 EMA
	ema200 := indicators.NewEMA(rd.emaPeriods[1])  // 200 EMA
	donchian := indicators.NewDonchianChannels(rd.donchianPeriod)
	atr := indicators.NewATR(rd.atrPeriod)
	bb := indicators.NewBollingerBands(rd.bbPeriod, rd.bbStdDev)
	rsi := indicators.NewRSI(rd.rsiPeriod)
	
	// Calculate ADX for trend strength
	adxValue, err := adx.Calculate(data)
	if err != nil {
		return nil, fmt.Errorf("ADX calculation failed: %w", err)
	}
	metrics.ADXValue = adxValue
	
	// Calculate EMAs for trend direction and strength
	ema50Value, err := ema50.Calculate(data)
	if err != nil {
		return nil, fmt.Errorf("EMA50 calculation failed: %w", err)
	}
	
	ema200Value, err := ema200.Calculate(data)
	if err != nil {
		return nil, fmt.Errorf("EMA200 calculation failed: %w", err)
	}
	
	// Calculate EMA distance (normalized by price)
	currentPrice := data[len(data)-1].Close
	metrics.EMADistance = (ema50Value - ema200Value) / currentPrice
	
	// Determine trend direction
	if ema50Value > ema200Value*1.005 { // 0.5% threshold
		metrics.TrendDirection = 1
	} else if ema50Value < ema200Value*0.995 {
		metrics.TrendDirection = -1
	} else {
		metrics.TrendDirection = 0
	}
	
	// Calculate combined trend strength
	trendFromADX := adxValue / 100.0 // Normalize ADX to 0-1
	trendFromEMA := math.Abs(metrics.EMADistance) / rd.emaDistanceThreshold
	if trendFromEMA > 1.0 {
		trendFromEMA = 1.0
	}
	metrics.TrendStrength = (trendFromADX + trendFromEMA) / 2.0
	
	// Calculate Donchian channels for breakout detection
	_, err = donchian.Calculate(data)
	if err != nil {
		return nil, fmt.Errorf("Donchian calculation failed: %w", err)
	}
	
	metrics.DonchianBreakout = donchian.IsBreakoutAbove(currentPrice) || donchian.IsBreakoutBelow(currentPrice)
	metrics.DonchianBreakoutStrength = donchian.GetBreakoutStrength(currentPrice)
	
	// Calculate ATR for volatility
	atrValue, err := atr.Calculate(data)
	if err != nil {
		return nil, fmt.Errorf("ATR calculation failed: %w", err)
	}
	metrics.ATRNormalized = atrValue / currentPrice // Normalize by price
	
	// Calculate Bollinger Bands for volatility
	_, err = bb.Calculate(data)
	if err != nil {
		return nil, fmt.Errorf("Bollinger Bands calculation failed: %w", err)
	}
	
	upper, middle, lower := bb.GetBands()
	if middle > 0 {
		metrics.BBWidth = (upper - lower) / middle // Normalized BB width
	}
	
	// Combined volatility measure
	volatilityFromATR := metrics.ATRNormalized / 0.03 // Assuming 3% is high volatility
	if volatilityFromATR > 1.0 {
		volatilityFromATR = 1.0
	}
	
	volatilityFromBB := metrics.BBWidth / 0.10 // Assuming 10% BB width is high volatility
	if volatilityFromBB > 1.0 {
		volatilityFromBB = 1.0
	}
	
	metrics.Volatility = (volatilityFromATR + volatilityFromBB) / 2.0
	
	// Calculate RSI for noise detection
	rsiValue, err := rsi.Calculate(data)
	if err != nil {
		return nil, fmt.Errorf("RSI calculation failed: %w", err)
	}
	
	// RSI noise score: higher when RSI is in the "noise range" [45-55]
	if rsiValue >= rd.rsiNoiseRange[0] && rsiValue <= rd.rsiNoiseRange[1] {
		// RSI in noise range - calculate how centered it is
		center := (rd.rsiNoiseRange[0] + rd.rsiNoiseRange[1]) / 2.0
		distance := math.Abs(rsiValue - center)
		maxDistance := (rd.rsiNoiseRange[1] - rd.rsiNoiseRange[0]) / 2.0
		metrics.RSINoiseScore = 1.0 - (distance / maxDistance) // Higher score = more centered = more noise
	} else {
		metrics.RSINoiseScore = 0.0 // RSI outside noise range
	}
	
	// Combined noise level (higher = more choppy/noisy)
	metrics.NoiseLevel = metrics.RSINoiseScore
	
	return metrics, nil
}

// classifyRegime determines the regime type based on calculated metrics
func (rd *RegimeDetector) classifyRegime(metrics *RegimeMetrics) RegimeType {
	// Primary classification based on trend strength and ADX
	if metrics.ADXValue > rd.adxTrendThreshold && metrics.TrendStrength > 0.6 {
		// Strong trend detected
		if metrics.DonchianBreakout && metrics.DonchianBreakoutStrength > 0.3 {
			return RegimeTrending // Confirmed breakout trend
		} else if math.Abs(metrics.EMADistance) > rd.emaDistanceThreshold {
			return RegimeTrending // EMAs diverged significantly
		}
	}
	
	// Check for ranging/choppy market
	if metrics.ADXValue < rd.adxTrendThreshold*0.8 && metrics.NoiseLevel > 0.6 {
		// Low trend strength with high noise
		if metrics.Volatility < 0.4 {
			return RegimeRanging // Low volatility ranging
		} else {
			return RegimeVolatile // High volatility chop
		}
	}
	
	// Check for high volatility regime
	if metrics.Volatility > 0.7 {
		return RegimeVolatile
	}
	
	// Check for clear ranging market
	if metrics.ADXValue < rd.adxTrendThreshold && 
		!metrics.DonchianBreakout && 
		metrics.TrendStrength < 0.4 {
		return RegimeRanging
	}
	
	// Default to uncertain if no clear regime identified
	return RegimeUncertain
}

// calculateConfidence determines confidence in regime classification
func (rd *RegimeDetector) calculateConfidence(metrics *RegimeMetrics, regimeType RegimeType) float64 {
	baseConfidence := 0.5
	
	switch regimeType {
	case RegimeTrending:
		// High confidence if strong ADX and clear EMA separation
		if metrics.ADXValue > rd.adxTrendThreshold*1.5 {
			baseConfidence += 0.3
		} else if metrics.ADXValue > rd.adxTrendThreshold {
			baseConfidence += 0.2
		}
		
		if math.Abs(metrics.EMADistance) > rd.emaDistanceThreshold*2 {
			baseConfidence += 0.2
		} else if math.Abs(metrics.EMADistance) > rd.emaDistanceThreshold {
			baseConfidence += 0.1
		}
		
		if metrics.DonchianBreakout && metrics.DonchianBreakoutStrength > 0.5 {
			baseConfidence += 0.2
		}
		
	case RegimeRanging:
		// High confidence if low ADX and high noise with low volatility
		if metrics.ADXValue < rd.adxTrendThreshold*0.5 {
			baseConfidence += 0.2
		}
		
		if metrics.NoiseLevel > 0.7 {
			baseConfidence += 0.2
		}
		
		if metrics.Volatility < 0.3 {
			baseConfidence += 0.2
		}
		
		if !metrics.DonchianBreakout {
			baseConfidence += 0.1
		}
		
	case RegimeVolatile:
		// High confidence if high volatility with unclear trend
		if metrics.Volatility > 0.8 {
			baseConfidence += 0.3
		} else if metrics.Volatility > 0.6 {
			baseConfidence += 0.2
		}
		
		if metrics.ADXValue < rd.adxTrendThreshold {
			baseConfidence += 0.1 // Volatile but not trending
		}
		
	case RegimeUncertain:
		// Lower confidence by definition
		baseConfidence = 0.3
	}
	
	// Cap confidence at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}
	
	return baseConfidence
}

// applyHysteresis prevents rapid regime switching using confirmation logic
func (rd *RegimeDetector) applyHysteresis(newRegime RegimeType, confidence float64) (RegimeType, bool) {
	// If this is the first detection, accept the new regime
	if rd.lastRegime == RegimeUncertain {
		rd.confirmationCounter = 1
		return newRegime, false
	}
	
	// Check if we're in cooldown period
	if rd.cooldownCounter > 0 {
		rd.cooldownCounter--
		return rd.lastRegime, false // Stay in current regime during cooldown
	}
	
	// If new regime matches current regime, we're stable
	if newRegime == rd.lastRegime {
		rd.confirmationCounter = 0 // Reset counter
		return rd.lastRegime, false
	}
	
	// New regime detected - check if we have enough confirmation
	if confidence > 0.8 {
		// High confidence - require fewer confirmation bars
		rd.confirmationCounter++
		if rd.confirmationCounter >= rd.confirmationBars-1 {
			// Regime change confirmed
			rd.confirmationCounter = 0
			rd.cooldownCounter = rd.regimeSwitchCooldown
			return newRegime, true // Transition detected
		}
	} else if confidence > 0.6 {
		// Medium confidence - require normal confirmation
		rd.confirmationCounter++
		if rd.confirmationCounter >= rd.confirmationBars {
			rd.confirmationCounter = 0
			rd.cooldownCounter = rd.regimeSwitchCooldown
			return newRegime, true
		}
	} else {
		// Low confidence - reset counter and stay in current regime
		rd.confirmationCounter = 0
	}
	
	// Not enough confirmation yet - stay in current regime
	return rd.lastRegime, false
}



// UpdateConfiguration allows updating detector parameters
func (rd *RegimeDetector) UpdateConfiguration(params map[string]interface{}) error {
	if emaPeriods, ok := params["ema_periods"].([]int); ok && len(emaPeriods) == 2 {
		rd.emaPeriods = emaPeriods
	}
	if adxPeriod, ok := params["adx_period"].(int); ok {
		rd.adxPeriod = adxPeriod
	}
	if adxThreshold, ok := params["adx_trend_threshold"].(float64); ok {
		rd.adxTrendThreshold = adxThreshold
	}
	if emaThreshold, ok := params["ema_distance_threshold"].(float64); ok {
		rd.emaDistanceThreshold = emaThreshold
	}
	if donchianPeriod, ok := params["donchian_period"].(int); ok {
		rd.donchianPeriod = donchianPeriod
	}
	return nil
}
