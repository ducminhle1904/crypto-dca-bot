package oscillators

import (
	"errors"
	"fmt"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

const (
	// DefaultMfiPeriod is the default period of the MFI
	DefaultMfiPeriod = 14
)

// MFI represents the Money Flow Index technical indicator
// MFI combines price and volume to measure buying/selling pressure
//
//	Raw Money Flow = Typical Price * Volume
//	Money Ratio = Positive Money Flow / Negative Money Flow
//	Money Flow Index = 100 - (100 / (1 + Money Ratio))
type MFI struct {
	period         int
	overbought     float64
	oversold       float64
	lastValue      float64
	positiveFlow   float64
	negativeFlow   float64
	initialized    bool
	typicalPrices  []float64 // Rolling window for typical prices
	volumes        []float64 // Rolling window for volumes
	moneyFlows     []float64 // Rolling window for money flows
}

// NewMFI creates a new Money Flow Index indicator with default parameters
func NewMFI() *MFI {
	return NewMFIWithPeriod(DefaultMfiPeriod)
}

// NewMFIWithPeriod creates a new Money Flow Index indicator with given period
func NewMFIWithPeriod(period int) *MFI {
	return &MFI{
		period:        period,
		overbought:    80.0, // MFI typically uses 80/20 instead of RSI's 70/30
		oversold:      20.0,
		typicalPrices: make([]float64, 0, period+1), // Need period+1 for comparison
		volumes:       make([]float64, 0, period+1),
		moneyFlows:    make([]float64, 0, period+1),
	}
}

// Calculate calculates the MFI value
func (m *MFI) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < m.period+1 {
		return 0, errors.New("insufficient data points for MFI calculation")
	}

	if !m.initialized {
		return m.initialCalculation(data)
	}

	return m.incrementalCalculation(data)
}

// initialCalculation calculates the first MFI value
func (m *MFI) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < m.period+1 {
		return 0, errors.New("not enough data for initial MFI calculation")
	}

	// Initialize with the last period+1 values
	recent := data[len(data)-m.period-1:]
	
	// Calculate typical prices and money flows
	for i := 0; i < len(recent); i++ {
		typicalPrice := (recent[i].High + recent[i].Low + recent[i].Close) / 3.0
		m.typicalPrices = append(m.typicalPrices, typicalPrice)
		m.volumes = append(m.volumes, recent[i].Volume)
		
		if i > 0 {
			moneyFlow := typicalPrice * recent[i].Volume
			m.moneyFlows = append(m.moneyFlows, moneyFlow)
		}
	}

	// Calculate positive and negative money flows
	m.positiveFlow = 0.0
	m.negativeFlow = 0.0

	for i := 1; i < len(m.typicalPrices); i++ {
		moneyFlow := m.moneyFlows[i-1]
		
		if m.typicalPrices[i] > m.typicalPrices[i-1] {
			m.positiveFlow += moneyFlow
		} else if m.typicalPrices[i] < m.typicalPrices[i-1] {
			m.negativeFlow += moneyFlow
		}
		// If equal, neither positive nor negative (no change in money flow direction)
	}

	m.initialized = true
	return m.calculateMFI()
}

// incrementalCalculation updates MFI with the latest data point
func (m *MFI) incrementalCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < 2 {
		return m.lastValue, nil
	}

	// Get the latest candle
	latest := data[len(data)-1]
	newTypicalPrice := (latest.High + latest.Low + latest.Close) / 3.0
	newVolume := latest.Volume
	newMoneyFlow := newTypicalPrice * newVolume

	// Remove the oldest values if we have a full window
	if len(m.typicalPrices) >= m.period+1 {
		// Remove oldest money flow contribution
		oldestIndex := 1 // First comparison starts at index 1
		if len(m.moneyFlows) > 0 && oldestIndex < len(m.typicalPrices) {
			oldMoneyFlow := m.moneyFlows[0]
			
			if m.typicalPrices[oldestIndex] > m.typicalPrices[oldestIndex-1] {
				m.positiveFlow -= oldMoneyFlow
			} else if m.typicalPrices[oldestIndex] < m.typicalPrices[oldestIndex-1] {
				m.negativeFlow -= oldMoneyFlow
			}
		}

		// Shift arrays
		copy(m.typicalPrices[:len(m.typicalPrices)-1], m.typicalPrices[1:])
		copy(m.volumes[:len(m.volumes)-1], m.volumes[1:])
		copy(m.moneyFlows[:len(m.moneyFlows)-1], m.moneyFlows[1:])
		
		// Update last elements
		m.typicalPrices[len(m.typicalPrices)-1] = newTypicalPrice
		m.volumes[len(m.volumes)-1] = newVolume
		m.moneyFlows[len(m.moneyFlows)-1] = newMoneyFlow
	} else {
		// Still building up the window
		m.typicalPrices = append(m.typicalPrices, newTypicalPrice)
		m.volumes = append(m.volumes, newVolume)
		m.moneyFlows = append(m.moneyFlows, newMoneyFlow)
	}

	// Update money flows based on the latest comparison
	if len(m.typicalPrices) >= 2 {
		lastIdx := len(m.typicalPrices) - 1
		if m.typicalPrices[lastIdx] > m.typicalPrices[lastIdx-1] {
			m.positiveFlow += newMoneyFlow
		} else if m.typicalPrices[lastIdx] < m.typicalPrices[lastIdx-1] {
			m.negativeFlow += newMoneyFlow
		}
	}

	return m.calculateMFI()
}

// calculateMFI computes the actual MFI value
func (m *MFI) calculateMFI() (float64, error) {
	if m.negativeFlow == 0 {
		m.lastValue = 100.0
		return 100.0, nil
	}

	if m.positiveFlow == 0 {
		m.lastValue = 0.0
		return 0.0, nil
	}

	moneyFlowRatio := m.positiveFlow / m.negativeFlow
	m.lastValue = 100.0 - (100.0 / (1.0 + moneyFlowRatio))

	return m.lastValue, nil
}

// ShouldBuy determines if we should buy based on MFI
func (m *MFI) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	mfiValue, err := m.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy when MFI indicates oversold condition
	return mfiValue < m.oversold, nil
}

// ShouldSell determines if we should sell based on MFI
func (m *MFI) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	mfiValue, err := m.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell when MFI indicates overbought condition
	return mfiValue > m.overbought, nil
}

// GetSignalStrength returns the signal strength based on MFI distance from extremes
func (m *MFI) GetSignalStrength() float64 {
	if m.lastValue < m.oversold {
		// Stronger buy signal the lower the MFI
		return (m.oversold - m.lastValue) / m.oversold
	}
	if m.lastValue > m.overbought {
		// Stronger sell signal the higher the MFI
		return (m.lastValue - m.overbought) / (100 - m.overbought)
	}
	return 0
}

// GetName returns the indicator name
func (m *MFI) GetName() string {
	return "MFI"
}

// String returns the string representation of the MFI (like the reference implementation)
func (m *MFI) String() string {
	return fmt.Sprintf("MFI(%d)", m.period)
}

// GetRequiredPeriods returns the minimum number of periods needed
func (m *MFI) GetRequiredPeriods() int {
	return m.GetIdlePeriod()
}

// GetIdlePeriod returns the initial period that MFI won't yield reliable results
// This matches the reference implementation's IdlePeriod method
func (m *MFI) GetIdlePeriod() int {
	return m.period + 1 // Need extra period for typical price comparison
}

// SetOversold sets the oversold threshold
func (m *MFI) SetOversold(threshold float64) {
	m.oversold = threshold
}

// SetOverbought sets the overbought threshold
func (m *MFI) SetOverbought(threshold float64) {
	m.overbought = threshold
}

// GetLastValue returns the last calculated MFI value
func (m *MFI) GetLastValue() float64 {
	return m.lastValue
}

// GetMoneyFlows returns the current positive and negative money flows
func (m *MFI) GetMoneyFlows() (positive, negative float64) {
	return m.positiveFlow, m.negativeFlow
}

// ResetState resets the MFI internal state for new data periods
func (m *MFI) ResetState() {
	m.lastValue = 0.0
	m.positiveFlow = 0.0
	m.negativeFlow = 0.0
	m.initialized = false
	m.typicalPrices = make([]float64, 0, m.period+1)
	m.volumes = make([]float64, 0, m.period+1)
	m.moneyFlows = make([]float64, 0, m.period+1)
}
