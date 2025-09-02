package regime

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// RegimeIndicators provides specialized indicators for regime detection
// These complement the existing indicators in internal/indicators/
type RegimeIndicators struct {
	// TODO: Phase 1 Implementation
	// These will be implemented in Phase 1 according to the plan
}

// DonchianChannels represents Donchian Channel breakout detection
// Used for identifying trend regime transitions
type DonchianChannels struct {
	period      int
	upperValues []float64 // Circular buffer for highest highs
	lowerValues []float64 // Circular buffer for lowest lows
	writeIndex  int
	initialized bool
	lastUpper   float64
	lastLower   float64
}

// NewDonchianChannels creates a new Donchian Channels indicator
func NewDonchianChannels(period int) *DonchianChannels {
	return &DonchianChannels{
		period:      period,
		upperValues: make([]float64, period),
		lowerValues: make([]float64, period),
		writeIndex:  0,
		initialized: false,
	}
}

// Calculate calculates the Donchian Channel values
func (dc *DonchianChannels) Calculate(data []types.OHLCV) (upper, lower float64, err error) {
	if len(data) < dc.period {
		return 0, 0, errors.New("insufficient data for Donchian Channels calculation")
	}

	if !dc.initialized {
		return dc.initialCalculation(data)
	}

	return dc.incrementalCalculation(data[len(data)-1])
}

// initialCalculation performs the initial Donchian Channel calculation
func (dc *DonchianChannels) initialCalculation(data []types.OHLCV) (upper, lower float64, err error) {
	// Initialize buffers with recent data
	startIdx := len(data) - dc.period
	
	dc.lastUpper = data[startIdx].High
	dc.lastLower = data[startIdx].Low
	
	// Find highest high and lowest low over the period
	for i := startIdx; i < len(data); i++ {
		if data[i].High > dc.lastUpper {
			dc.lastUpper = data[i].High
		}
		if data[i].Low < dc.lastLower {
			dc.lastLower = data[i].Low
		}
		
		// Store in circular buffer
		idx := (i - startIdx) % dc.period
		dc.upperValues[idx] = data[i].High
		dc.lowerValues[idx] = data[i].Low
	}
	
	dc.writeIndex = 0
	dc.initialized = true
	
	return dc.lastUpper, dc.lastLower, nil
}

// incrementalCalculation updates Donchian Channels with new data
func (dc *DonchianChannels) incrementalCalculation(newCandle types.OHLCV) (upper, lower float64, err error) {
	// Add new values to circular buffer
	dc.upperValues[dc.writeIndex] = newCandle.High
	dc.lowerValues[dc.writeIndex] = newCandle.Low
	
	// Move write index
	dc.writeIndex = (dc.writeIndex + 1) % dc.period
	
	// Recalculate channels by scanning the entire buffer
	dc.lastUpper = dc.upperValues[0]
	dc.lastLower = dc.lowerValues[0]
	
	for i := 1; i < dc.period; i++ {
		if dc.upperValues[i] > dc.lastUpper {
			dc.lastUpper = dc.upperValues[i]
		}
		if dc.lowerValues[i] < dc.lastLower {
			dc.lastLower = dc.lowerValues[i]
		}
	}
	
	return dc.lastUpper, dc.lastLower, nil
}

// ADRNormalization provides Average Daily Range normalization for volatility assessment
type ADRNormalization struct {
	period      int
	adrValues   []float64
	writeIndex  int
	initialized bool
	lastADR     float64
}

// NewADRNormalization creates a new ADR normalization calculator
func NewADRNormalization(period int) *ADRNormalization {
	return &ADRNormalization{
		period:     period,
		adrValues:  make([]float64, period),
		writeIndex: 0,
	}
}

// Calculate calculates the normalized ADR for current volatility assessment
func (adr *ADRNormalization) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < adr.period {
		return 0, errors.New("insufficient data for ADR calculation")
	}

	if !adr.initialized {
		return adr.initialCalculation(data)
	}

	return adr.incrementalCalculation(data[len(data)-1])
}

// initialCalculation calculates initial ADR average
func (adr *ADRNormalization) initialCalculation(data []types.OHLCV) (float64, error) {
	// Calculate daily ranges for the initial period
	startIdx := len(data) - adr.period
	sum := 0.0
	
	for i := 0; i < adr.period; i++ {
		candle := data[startIdx+i]
		dailyRange := (candle.High - candle.Low) / candle.Close // Normalize by price
		adr.adrValues[i] = dailyRange
		sum += dailyRange
	}
	
	adr.lastADR = sum / float64(adr.period)
	adr.writeIndex = 0
	adr.initialized = true
	
	// Current normalized range
	currentCandle := data[len(data)-1]
	currentRange := (currentCandle.High - currentCandle.Low) / currentCandle.Close
	
	// Return ratio of current range to average range
	if adr.lastADR > 0 {
		return currentRange / adr.lastADR, nil
	}
	
	return 1.0, nil // Default to neutral if no historical average
}

// incrementalCalculation updates ADR with new data
func (adr *ADRNormalization) incrementalCalculation(newCandle types.OHLCV) (float64, error) {
	// Calculate new daily range normalized by price
	newRange := (newCandle.High - newCandle.Low) / newCandle.Close
	
	// Remove oldest value and add new value using circular buffer
	oldValue := adr.adrValues[adr.writeIndex]
	adr.adrValues[adr.writeIndex] = newRange
	adr.writeIndex = (adr.writeIndex + 1) % adr.period
	
	// Update rolling average
	adr.lastADR = adr.lastADR + (newRange - oldValue) / float64(adr.period)
	
	// Return ratio of current range to average range
	if adr.lastADR > 0 {
		return newRange / adr.lastADR, nil
	}
	
	return 1.0, nil
}

// NoiseDetection provides market noise level assessment
type NoiseDetection struct {
	rsiPeriod       int
	noiseThreshold  float64
	noiseBars       int
	consecutiveNoise int
}

// NewNoiseDetection creates a new noise detection indicator
func NewNoiseDetection(rsiPeriod int, noiseThreshold float64) *NoiseDetection {
	return &NoiseDetection{
		rsiPeriod:      rsiPeriod,
		noiseThreshold: noiseThreshold,
		noiseBars:      0,
	}
}

// Calculate calculates the current market noise level
func (nd *NoiseDetection) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < nd.rsiPeriod {
		return 0, errors.New("insufficient data for noise detection")
	}

	// Calculate price efficiency (how much of the price movement was directional)
	if len(data) < 10 {
		return 0.0, nil
	}
	
	// Look at recent price action (last 10 bars)
	lookback := 10
	if len(data) < lookback {
		lookback = len(data)
	}
	
	startIdx := len(data) - lookback
	startPrice := data[startIdx].Close
	endPrice := data[len(data)-1].Close
	
	// Calculate net price movement
	netMovement := math.Abs(endPrice - startPrice)
	
	// Calculate total price movement (sum of individual bar movements)
	totalMovement := 0.0
	for i := startIdx + 1; i < len(data); i++ {
		totalMovement += math.Abs(data[i].Close - data[i-1].Close)
	}
	
	// Price efficiency ratio
	var efficiency float64
	if totalMovement > 0 {
		efficiency = netMovement / totalMovement
	} else {
		efficiency = 1.0 // No movement = perfect efficiency
	}
	
	// Noise level is inverse of efficiency
	// High efficiency (trending) = low noise
	// Low efficiency (choppy) = high noise
	noiseLevel := 1.0 - efficiency
	
	// Count bars in noise range (using simple RSI approximation)
	noiseBars := 0
	recentBars := 5 // Look at last 5 bars
	if len(data) >= recentBars {
		for i := len(data) - recentBars; i < len(data); i++ {
			// Simple momentum check (not full RSI, but similar concept)
			if i > 0 {
				priceChange := (data[i].Close - data[i-1].Close) / data[i-1].Close
				if math.Abs(priceChange) < nd.noiseThreshold {
					noiseBars++
				}
			}
		}
		
		// Boost noise level if many recent bars are in noise range
		if noiseBars >= 3 {
			noiseLevel = math.Min(1.0, noiseLevel * 1.5)
		}
	}
	
	return noiseLevel, nil
}

// BreakoutDetection identifies significant price breakouts that signal regime change
type BreakoutDetection struct {
	donchianPeriod    int
	volumeMultiplier  float64
	priceMultiplier   float64
	lastBreakoutType  int // 1 = bullish, -1 = bearish, 0 = none
}

// NewBreakoutDetection creates a new breakout detection indicator
func NewBreakoutDetection(donchianPeriod int) *BreakoutDetection {
	return &BreakoutDetection{
		donchianPeriod:   donchianPeriod,
		volumeMultiplier: 1.5, // Volume must be 1.5x average
		priceMultiplier:  1.2, // Price must move 1.2x normal range
	}
}

// DetectBreakout analyzes current data for significant breakouts
func (bd *BreakoutDetection) DetectBreakout(data []types.OHLCV) (breakoutType int, strength float64, err error) {
	if len(data) < bd.donchianPeriod {
		return 0, 0, errors.New("insufficient data for breakout detection")
	}

	// Create Donchian Channel for breakout detection
	dc := NewDonchianChannels(bd.donchianPeriod)
	upper, lower, err := dc.Calculate(data)
	if err != nil {
		return 0, 0, err
	}
	
	currentCandle := data[len(data)-1]
	currentPrice := currentCandle.Close
	
	// Check for breakout
	breakoutType = 0 // No breakout
	strength = 0.0
	
	if currentPrice > upper {
		// Bullish breakout
		breakoutType = 1
		channelWidth := upper - lower
		if channelWidth > 0 {
			strength = (currentPrice - upper) / channelWidth
		}
	} else if currentPrice < lower {
		// Bearish breakout
		breakoutType = -1
		channelWidth := upper - lower
		if channelWidth > 0 {
			strength = (lower - currentPrice) / channelWidth
		}
	}
	
	// Enhance strength with volume confirmation if available
	if breakoutType != 0 && len(data) >= 5 {
		// Calculate average volume over last 5 periods
		avgVolume := 0.0
		for i := len(data) - 5; i < len(data) - 1; i++ {
			avgVolume += data[i].Volume
		}
		avgVolume /= 4.0 // Last 4 periods (excluding current)
		
		if avgVolume > 0 && currentCandle.Volume > avgVolume * bd.volumeMultiplier {
			// Volume confirmation - boost strength
			strength *= 1.5
		} else if currentCandle.Volume < avgVolume * 0.5 {
			// Low volume - reduce strength
			strength *= 0.7
		}
	}
	
	// Cap strength at reasonable levels
	if strength > 2.0 {
		strength = 2.0
	}
	
	bd.lastBreakoutType = breakoutType
	
	return breakoutType, strength, nil
}

// RegimeConfidence calculates overall confidence in current regime classification
func CalculateRegimeConfidence(trendStrength, volatility, noiseLevel float64, breakoutStrength float64) float64 {
	// Weighted combination of all regime indicators
	// Higher confidence = more certain about regime classification
	
	baseConfidence := 0.5 // Start neutral
	
	// Trend strength contribution (30% weight)
	trendWeight := 0.3
	if trendStrength > 0.8 {
		baseConfidence += trendWeight * 0.4  // Very strong trend = high confidence
	} else if trendStrength > 0.6 {
		baseConfidence += trendWeight * 0.2  // Strong trend = medium confidence boost
	} else if trendStrength < 0.3 {
		baseConfidence -= trendWeight * 0.2  // Weak trend = reduce confidence
	}
	
	// Noise level contribution (25% weight) - inverse relationship
	noiseWeight := 0.25
	baseConfidence -= noiseWeight * noiseLevel  // Higher noise = lower confidence
	
	// Volatility contribution (20% weight) 
	volatilityWeight := 0.2
	if volatility > 0.8 {
		// Very high volatility might indicate volatile regime with high confidence
		baseConfidence += volatilityWeight * 0.3
	} else if volatility < 0.2 {
		// Very low volatility might indicate ranging regime with high confidence  
		baseConfidence += volatilityWeight * 0.2
	} else {
		// Medium volatility = uncertainty
		baseConfidence -= volatilityWeight * 0.1
	}
	
	// Breakout strength contribution (25% weight)
	breakoutWeight := 0.25
	if breakoutStrength > 0.5 {
		// Strong breakout = high confidence in trend regime
		baseConfidence += breakoutWeight * 0.4
	} else if breakoutStrength > 0.2 {
		// Moderate breakout = some confidence boost
		baseConfidence += breakoutWeight * 0.2
	}
	
	// Consistency bonus: if all indicators align
	if trendStrength > 0.6 && noiseLevel < 0.4 && breakoutStrength > 0.3 {
		baseConfidence += 0.1 // Bonus for consistent trending signals
	} else if trendStrength < 0.4 && noiseLevel > 0.6 && breakoutStrength < 0.2 {
		baseConfidence += 0.1 // Bonus for consistent ranging signals
	}
	
	// Clamp to [0, 1] range
	return math.Max(0.0, math.Min(1.0, baseConfidence))
}
