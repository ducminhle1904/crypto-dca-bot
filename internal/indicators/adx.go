package indicators

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// ADX represents the Average Directional Index technical indicator
// ADX measures trend strength regardless of direction (0-100 scale)
// Values > 20 indicate trending market, > 40 indicate strong trend
type ADX struct {
	period      int
	
	// Internal components for calculation
	trValues    []float64 // True Range values
	plusDI      []float64 // +DI values  
	minusDI     []float64 // -DI values
	dx          []float64 // DX values
	adxValues   []float64 // ADX values (smoothed DX)
	
	// Previous values for incremental calculation
	prevHigh    float64
	prevLow     float64
	prevClose   float64
	
	// Smoothing components (Wilder's smoothing)
	trSum       float64
	plusDMSum   float64
	minusDMSum  float64
	adxSum      float64
	
	// State tracking
	writeIndex  int
	count       int
	initialized bool
	lastADX     float64
}

// NewADX creates a new ADX indicator
func NewADX(period int) *ADX {
	return &ADX{
		period:     period,
		trValues:   make([]float64, period),
		plusDI:     make([]float64, period),
		minusDI:    make([]float64, period),
		dx:         make([]float64, period),
		adxValues:  make([]float64, period),
		writeIndex: 0,
		count:      0,
		initialized: false,
	}
}

// Calculate calculates the ADX value
func (adx *ADX) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < adx.period*3 { // Need extra periods for proper ADX calculation
		return 0, errors.New("insufficient data for ADX calculation")
	}

	if !adx.initialized {
		return adx.initialCalculation(data)
	}

	return adx.incrementalCalculation(data[len(data)-1])
}

// initialCalculation performs the initial ADX calculation
func (adx *ADX) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < adx.period*3 {
		return 0, errors.New("insufficient data for ADX initialization")
	}
	
	// Calculate initial True Range, +DM, -DM for the period
	adx.trSum = 0
	adx.plusDMSum = 0
	adx.minusDMSum = 0
	
	startIdx := len(data) - (adx.period * 2) // Start from enough back for proper calculation
	if startIdx < 1 {
		startIdx = 1
	}
	
	// Calculate TR, +DM, -DM for initial period
	for i := startIdx; i < startIdx+adx.period && i < len(data); i++ {
		if i == 0 {
			continue
		}
		
		current := data[i]
		previous := data[i-1]
		
		// True Range calculation
		tr := math.Max(current.High-current.Low, 
			math.Max(math.Abs(current.High-previous.Close),
				math.Abs(current.Low-previous.Close)))
		adx.trSum += tr
		
		// Directional Movement calculation
		plusDM := 0.0
		minusDM := 0.0
		
		highDiff := current.High - previous.High
		lowDiff := previous.Low - current.Low
		
		if highDiff > lowDiff && highDiff > 0 {
			plusDM = highDiff
		}
		if lowDiff > highDiff && lowDiff > 0 {
			minusDM = lowDiff
		}
		
		adx.plusDMSum += plusDM
		adx.minusDMSum += minusDM
	}
	
	// Calculate initial DI values
	plusDI14 := (adx.plusDMSum / adx.trSum) * 100
	minusDI14 := (adx.minusDMSum / adx.trSum) * 100
	
	// Calculate DX
	diSum := plusDI14 + minusDI14
	var dx float64
	if diSum != 0 {
		dx = (math.Abs(plusDI14-minusDI14) / diSum) * 100
	}
	
	// Start building ADX (need more DX values for proper smoothing)
	dxValues := make([]float64, 0)
	dxValues = append(dxValues, dx)
	
	// Calculate more DX values to get initial ADX
	for i := startIdx + adx.period; i < len(data); i++ {
		current := data[i]
		previous := data[i-1]
		
		// True Range
		tr := math.Max(current.High-current.Low, 
			math.Max(math.Abs(current.High-previous.Close),
				math.Abs(current.Low-previous.Close)))
		
		// Smooth TR using Wilder's method
		adx.trSum = adx.trSum - (adx.trSum / float64(adx.period)) + tr
		
		// Directional Movement
		plusDM := 0.0
		minusDM := 0.0
		
		highDiff := current.High - previous.High
		lowDiff := previous.Low - current.Low
		
		if highDiff > lowDiff && highDiff > 0 {
			plusDM = highDiff
		}
		if lowDiff > highDiff && lowDiff > 0 {
			minusDM = lowDiff
		}
		
		// Smooth DM using Wilder's method
		adx.plusDMSum = adx.plusDMSum - (adx.plusDMSum / float64(adx.period)) + plusDM
		adx.minusDMSum = adx.minusDMSum - (adx.minusDMSum / float64(adx.period)) + minusDM
		
		// Calculate DI
		plusDI := (adx.plusDMSum / adx.trSum) * 100
		minusDI := (adx.minusDMSum / adx.trSum) * 100
		
		// Calculate DX
		diSum := plusDI + minusDI
		if diSum != 0 {
			dx = (math.Abs(plusDI-minusDI) / diSum) * 100
		} else {
			dx = 0
		}
		
		dxValues = append(dxValues, dx)
	}
	
	// Calculate initial ADX as simple average of first period DX values
	if len(dxValues) >= adx.period {
		adxSum := 0.0
		for i := 0; i < adx.period; i++ {
			adxSum += dxValues[i]
		}
		adx.lastADX = adxSum / float64(adx.period)
		adx.adxSum = adx.lastADX * float64(adx.period) // Set up for Wilder's smoothing
	} else {
		adx.lastADX = 0
		adx.adxSum = 0
	}
	
	// Store last values for incremental calculation
	lastCandle := data[len(data)-1]
	adx.prevHigh = lastCandle.High
	adx.prevLow = lastCandle.Low
	adx.prevClose = lastCandle.Close
	
	adx.initialized = true
	adx.count = adx.period
	
	return adx.lastADX, nil
}

// incrementalCalculation updates ADX with new price data
func (adx *ADX) incrementalCalculation(newCandle types.OHLCV) (float64, error) {
	// True Range calculation
	tr := math.Max(newCandle.High-newCandle.Low, 
		math.Max(math.Abs(newCandle.High-adx.prevClose),
			math.Abs(newCandle.Low-adx.prevClose)))
	
	// Smooth TR using Wilder's method
	adx.trSum = adx.trSum - (adx.trSum / float64(adx.period)) + tr
	
	// Directional Movement calculation
	plusDM := 0.0
	minusDM := 0.0
	
	highDiff := newCandle.High - adx.prevHigh
	lowDiff := adx.prevLow - newCandle.Low
	
	if highDiff > lowDiff && highDiff > 0 {
		plusDM = highDiff
	}
	if lowDiff > highDiff && lowDiff > 0 {
		minusDM = lowDiff
	}
	
	// Smooth DM using Wilder's method
	adx.plusDMSum = adx.plusDMSum - (adx.plusDMSum / float64(adx.period)) + plusDM
	adx.minusDMSum = adx.minusDMSum - (adx.minusDMSum / float64(adx.period)) + minusDM
	
	// Calculate DI
	plusDI := (adx.plusDMSum / adx.trSum) * 100
	minusDI := (adx.minusDMSum / adx.trSum) * 100
	
	// Calculate DX
	diSum := plusDI + minusDI
	var dx float64
	if diSum != 0 {
		dx = (math.Abs(plusDI-minusDI) / diSum) * 100
	}
	
	// Smooth ADX using Wilder's method
	adx.adxSum = adx.adxSum - (adx.adxSum / float64(adx.period)) + dx
	adx.lastADX = adx.adxSum / float64(adx.period)
	
	// Update previous values
	adx.prevHigh = newCandle.High
	adx.prevLow = newCandle.Low
	adx.prevClose = newCandle.Close
	
	return adx.lastADX, nil
}

// ShouldBuy determines if ADX indicates a trending market suitable for trend following
func (adx *ADX) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	// ADX doesn't give buy/sell signals directly, but indicates trend strength
	// For regime detection, we use it to identify trending vs ranging markets
	// ADX > 20 typically indicates trending market
	return current > 20.0, nil
}

// ShouldSell determines if ADX indicates weakening trend
func (adx *ADX) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	// ADX falling below 20 might indicate trend weakness
	// But for regime detection, we're more interested in the absolute value
	return current < 15.0, nil // Very low ADX might indicate ranging market
}

// GetSignalStrength returns the strength of the trend signal (0-1 scale)
func (adx *ADX) GetSignalStrength() float64 {
	// Convert ADX (0-100) to 0-1 scale
	// ADX > 40 is considered very strong
	strength := adx.lastADX / 40.0
	if strength > 1.0 {
		strength = 1.0
	}
	return strength
}

// GetName returns the indicator name
func (adx *ADX) GetName() string {
	return "ADX"
}

// GetRequiredPeriods returns minimum periods needed for calculation
func (adx *ADX) GetRequiredPeriods() int {
	return adx.period * 3 // Need extra periods for proper ADX calculation
}

// ResetState resets internal state for new data periods
func (adx *ADX) ResetState() {
	adx.trSum = 0
	adx.plusDMSum = 0
	adx.minusDMSum = 0
	adx.adxSum = 0
	adx.prevHigh = 0
	adx.prevLow = 0
	adx.prevClose = 0
	adx.writeIndex = 0
	adx.count = 0
	adx.initialized = false
	adx.lastADX = 0
}

// GetDirectionalIndex returns both +DI and -DI for additional analysis
func (adx *ADX) GetDirectionalIndex() (plusDI, minusDI float64) {
	if adx.trSum > 0 {
		plusDI = (adx.plusDMSum / adx.trSum) * 100
		minusDI = (adx.minusDMSum / adx.trSum) * 100
	}
	return plusDI, minusDI
}

// IsTrending returns true if ADX indicates a trending market
func (adx *ADX) IsTrending() bool {
	return adx.lastADX > 20.0
}

// GetTrendStrength returns trend strength classification
func (adx *ADX) GetTrendStrength() string {
	if adx.lastADX < 20 {
		return "weak_or_ranging"
	} else if adx.lastADX < 40 {
		return "trending"
	} else {
		return "strong_trending"
	}
}
