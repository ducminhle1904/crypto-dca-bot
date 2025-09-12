package oscillators

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/indicators/common"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// WaveTrend represents the WaveTrend oscillator technical indicator
// WaveTrend combines multiple smoothing techniques to create a momentum oscillator
type WaveTrend struct {
	n1           int     // First EMA length for channel calculation (default: 10)
	n2           int     // Second EMA length for average calculation (default: 21)
	overBought   float64 // Overbought level (default: 60)
	overSold     float64 // Oversold level (default: -60)
	
	// Internal components
	esaEMA     *common.EMA     // ESA (Exponential Simple Average) calculator
	dEMA       *common.EMA     // D (smoothed absolute difference) calculator
	wt1EMA     *common.EMA     // WT1 (WaveTrend) calculator
	wt2Values  []float64 // Rolling window for WT2 calculation (SMA of WT1, length 4)
	
	// State tracking
	lastWT1        float64
	lastWT2        float64
	lastHLC3       float64
	initialized    bool
	dataPoints     int
}

// NewWaveTrend creates a new WaveTrend indicator with default parameters
func NewWaveTrend() *WaveTrend {
	return NewWaveTrendCustom(10, 21)
}

// NewWaveTrendCustom creates a new WaveTrend indicator with custom parameters
func NewWaveTrendCustom(n1, n2 int) *WaveTrend {
	return &WaveTrend{
		n1:           n1,
		n2:           n2,
		overBought:    60.0,
		overSold:      -60.0,
		esaEMA:        common.NewEMA(n1),
		dEMA:          common.NewEMA(n1),
		wt1EMA:        common.NewEMA(n2),
		wt2Values:     make([]float64, 0, 4), // WT2 is SMA(WT1, 4)
	}
}

// Calculate calculates the WaveTrend values (returns WT1)
func (wt *WaveTrend) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < wt.getMinRequiredPeriods() {
		return 0, errors.New("insufficient data points for WaveTrend calculation")
	}

	if !wt.initialized {
		return wt.initialCalculation(data)
	}

	return wt.incrementalCalculation(data)
}

// initialCalculation calculates the initial WaveTrend values
func (wt *WaveTrend) initialCalculation(data []types.OHLCV) (float64, error) {
	minRequired := wt.getMinRequiredPeriods()
	if len(data) < minRequired {
		return 0, errors.New("not enough data for initial WaveTrend calculation")
	}

	// Process all data points to build up the EMAs
	for i := 0; i < len(data); i++ {
		candle := data[i]
		hlc3 := (candle.High + candle.Low + candle.Close) / 3.0
		
		// Update ESA (Exponential Simple Average of Typical Price)
		esa := wt.esaEMA.UpdateSingle(hlc3)
		
		// Calculate absolute difference and update D
		absDiff := math.Abs(hlc3 - esa)
		d := wt.dEMA.UpdateSingle(absDiff)
		
		// Calculate CI (Channel Index)  
		var ci float64
		if d != 0 {
			ci = (hlc3 - esa) / (0.015 * d)
		}
		
		// Update WT1 (WaveTrend = EMA of CI)
		wt1 := wt.wt1EMA.UpdateSingle(ci)
		
		// Only use results after enough data points for proper EMA calculation
		if i < wt.n1 {
			continue // Skip early values where EMAs aren't stable yet
		}
		
		// WT1 = EMA(CI, n2)
		wt.lastWT1 = wt1
		
		// Update WT2 rolling window (SMA of WT1, length 4)
		wt.wt2Values = append(wt.wt2Values, wt1)
		if len(wt.wt2Values) > 4 {
			wt.wt2Values = wt.wt2Values[1:] // Remove oldest, keep only 4 values
		}
		
		wt.lastHLC3 = hlc3
		wt.dataPoints = i + 1
	}

	// Calculate WT2 (SMA of last 4 WT1 values)
	if len(wt.wt2Values) >= 4 {
		sum := 0.0
		for _, val := range wt.wt2Values {
			sum += val
		}
		wt.lastWT2 = sum / 4.0
	}

	wt.initialized = true
	return wt.lastWT1, nil
}

// incrementalCalculation updates WaveTrend with the latest data point
func (wt *WaveTrend) incrementalCalculation(data []types.OHLCV) (float64, error) {
	if len(data) == 0 {
		return wt.lastWT1, nil
	}

	// Get the latest candle
	latest := data[len(data)-1]
	hlc3 := (latest.High + latest.Low + latest.Close) / 3.0

	// Update ESA (Exponential Simple Average of Typical Price)
	esa := wt.esaEMA.UpdateSingle(hlc3)

	// Calculate absolute difference and update D
	absDiff := math.Abs(hlc3 - esa)
	d := wt.dEMA.UpdateSingle(absDiff)

	// Calculate CI (Channel Index)
	var ci float64
	if d != 0 {
		ci = (hlc3 - esa) / (0.015 * d)
	}

	// Update WT1 (WaveTrend = EMA of CI)
	wt1 := wt.wt1EMA.UpdateSingle(ci)

	// WT1 = EMA(CI, n2)
	wt.lastWT1 = wt1

	// Update WT2 rolling window (SMA of WT1, length 4)
	wt.wt2Values = append(wt.wt2Values, wt1)
	if len(wt.wt2Values) > 4 {
		wt.wt2Values = wt.wt2Values[1:] // Remove oldest, keep only 4 values
	}

	// Calculate WT2 (SMA of last 4 WT1 values)
	if len(wt.wt2Values) >= 4 {
		sum := 0.0
		for _, val := range wt.wt2Values {
			sum += val
		}
		wt.lastWT2 = sum / 4.0
	}

	wt.lastHLC3 = hlc3
	return wt.lastWT1, nil
}

// ShouldBuy determines if we should buy based on WaveTrend
func (wt *WaveTrend) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	_, err := wt.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy signals:
	// 1. WT1 crosses above WT2 (bullish crossover)
	// 2. WT1 is in oversold territory and trending up
	crossoverBuy := wt.lastWT1 > wt.lastWT2 && wt.lastWT1 > wt.overSold
	oversoldBuy := wt.lastWT1 < wt.overSold && wt.lastWT1 > wt.lastWT2

	return crossoverBuy || oversoldBuy, nil
}

// ShouldSell determines if we should sell based on WaveTrend
func (wt *WaveTrend) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	_, err := wt.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell signals:
	// 1. WT1 crosses below WT2 (bearish crossover)
	// 2. WT1 is in overbought territory and trending down
	crossoverSell := wt.lastWT1 < wt.lastWT2 && wt.lastWT1 < wt.overBought
	overboughtSell := wt.lastWT1 > wt.overBought && wt.lastWT1 < wt.lastWT2

	return crossoverSell || overboughtSell, nil
}

// GetSignalStrength returns the signal strength based on WaveTrend position and momentum
func (wt *WaveTrend) GetSignalStrength() float64 {
	// Calculate signal strength based on:
	// 1. Distance from overbought/oversold levels
	// 2. Momentum (difference between WT1 and WT2)
	
	momentum := math.Abs(wt.lastWT1 - wt.lastWT2)
	maxMomentum := 20.0 // Typical max momentum observed
	momentumStrength := math.Min(momentum/maxMomentum, 1.0)

	var positionStrength float64
	if wt.lastWT1 > wt.overBought {
		// Overbought - sell signal strength
		positionStrength = (wt.lastWT1 - wt.overBought) / (100 - wt.overBought)
	} else if wt.lastWT1 < wt.overSold {
		// Oversold - buy signal strength
		positionStrength = (wt.overSold - wt.lastWT1) / math.Abs(wt.overSold)
	}

	// Combine momentum and position strength
	return math.Min((momentumStrength+positionStrength)/2.0, 1.0)
}

// GetName returns the indicator name
func (wt *WaveTrend) GetName() string {
	return "WaveTrend"
}

// GetRequiredPeriods returns the minimum number of periods needed
func (wt *WaveTrend) GetRequiredPeriods() int {
	return wt.getMinRequiredPeriods()
}

// getMinRequiredPeriods calculates the minimum required periods
func (wt *WaveTrend) getMinRequiredPeriods() int {
	// Need enough data for all EMAs to initialize
	return wt.n1 + wt.n2
}

// GetWT1 returns the last calculated WT1 value
func (wt *WaveTrend) GetWT1() float64 {
	return wt.lastWT1
}

// GetWT2 returns the last calculated WT2 value
func (wt *WaveTrend) GetWT2() float64 {
	return wt.lastWT2
}

// GetWaves returns both WT1 and WT2 values
func (wt *WaveTrend) GetWaves() (wt1, wt2 float64) {
	return wt.lastWT1, wt.lastWT2
}

// SetOverbought sets the overbought threshold
func (wt *WaveTrend) SetOverbought(threshold float64) {
	wt.overBought = threshold
}

// SetOversold sets the oversold threshold
func (wt *WaveTrend) SetOversold(threshold float64) {
	wt.overSold = threshold
}

// IsOverbought returns true if WT1 is above overbought level
func (wt *WaveTrend) IsOverbought() bool {
	return wt.lastWT1 > wt.overBought
}

// IsOversold returns true if WT1 is below oversold level
func (wt *WaveTrend) IsOversold() bool {
	return wt.lastWT1 < wt.overSold
}

// ResetState resets the WaveTrend internal state for new data periods
func (wt *WaveTrend) ResetState() {
	// Reset EMA components
	wt.esaEMA.ResetState()
	wt.dEMA.ResetState()
	wt.wt1EMA.ResetState()
	
	// Reset rolling windows and values
	wt.wt2Values = make([]float64, 0, 4)
	wt.lastWT1 = 0.0
	wt.lastWT2 = 0.0
	wt.lastHLC3 = 0.0
	wt.initialized = false
	wt.dataPoints = 0
}
