package indicators

import (
	"errors"
	"fmt"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

const (
	// DefaultKeltnerChannelPeriod is the default period for the Keltner Channel
	DefaultKeltnerChannelPeriod = 20
)

// KeltnerChannels represents the Keltner Channels technical indicator
// Uses EMA for the middle line and ATR (Average True Range) for the channel width
//
//	Middle Line = EMA(period, closings)
//	Upper Band = EMA(period, closings) + multiplier * ATR(period, highs, lows, closings)
//	Lower Band = EMA(period, closings) - multiplier * ATR(period, highs, lows, closings)
type KeltnerChannels struct {
	period     int     // Period for EMA and ATR calculation
	multiplier float64 // Multiplier for ATR to create the bands
	
	// Internal components
	emaIndicator *EMA // EMA for middle line calculation (closing prices)
	atrIndicator *ATR // ATR for volatility calculation
	
	// State tracking
	lastUpper      float64
	lastMiddle     float64
	lastLower      float64
	initialized    bool
}

// NewKeltnerChannels creates a new Keltner Channels indicator with default parameters
func NewKeltnerChannels() *KeltnerChannels {
	return NewKeltnerChannelsWithPeriod(DefaultKeltnerChannelPeriod)
}

// NewKeltnerChannelsWithPeriod creates a new Keltner Channels indicator with given period
func NewKeltnerChannelsWithPeriod(period int) *KeltnerChannels {
	return NewKeltnerChannelsCustom(period, 2.0) // Default multiplier of 2.0
}

// NewKeltnerChannelsCustom creates a new Keltner Channels indicator with custom parameters
func NewKeltnerChannelsCustom(period int, multiplier float64) *KeltnerChannels {
	return &KeltnerChannels{
		period:       period,
		multiplier:   multiplier,
		emaIndicator: NewEMA(period),      // EMA for closing prices
		atrIndicator: NewATR(period),      // ATR for volatility
	}
}

// Calculate calculates the Keltner Channels values (returns middle line)
func (kc *KeltnerChannels) Calculate(data []types.OHLCV) (float64, error) {
	minRequired := kc.GetRequiredPeriods()
	if len(data) < minRequired {
		return 0, errors.New("insufficient data points for Keltner Channels calculation")
	}

	if !kc.initialized {
		return kc.initialCalculation(data)
	}

	return kc.incrementalCalculation(data)
}

// initialCalculation calculates the initial Keltner Channels values
func (kc *KeltnerChannels) initialCalculation(data []types.OHLCV) (float64, error) {
	minRequired := kc.GetRequiredPeriods()
	if len(data) < minRequired {
		return 0, errors.New("not enough data for initial Keltner Channels calculation")
	}

	// Calculate EMA of closing prices (middle line)
	middleLine, err := kc.emaIndicator.Calculate(data)
	if err != nil {
		return 0, err
	}
	
	// Calculate ATR for volatility
	atr, err := kc.atrIndicator.Calculate(data)
	if err != nil {
		return 0, err
	}
	
	// Calculate bands
	kc.lastMiddle = middleLine
	kc.lastUpper = middleLine + (kc.multiplier * atr)
	kc.lastLower = middleLine - (kc.multiplier * atr)

	kc.initialized = true
	return kc.lastMiddle, nil
}

// incrementalCalculation updates Keltner Channels with the latest data point
func (kc *KeltnerChannels) incrementalCalculation(data []types.OHLCV) (float64, error) {
	if len(data) == 0 {
		return kc.lastMiddle, nil
	}

	// Calculate EMA of closing prices (middle line)
	middleLine, err := kc.emaIndicator.Calculate(data)
	if err != nil {
		return kc.lastMiddle, err
	}

	// Calculate ATR for volatility
	atr, err := kc.atrIndicator.Calculate(data)
	if err != nil {
		return kc.lastMiddle, err
	}

	// Calculate bands
	kc.lastMiddle = middleLine
	kc.lastUpper = middleLine + (kc.multiplier * atr)
	kc.lastLower = middleLine - (kc.multiplier * atr)

	return kc.lastMiddle, nil
}

// No longer needed - ATR indicator handles True Range calculation

// ShouldBuy determines if we should buy based on Keltner Channels
func (kc *KeltnerChannels) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	_, err := kc.Calculate(data)
	if err != nil {
		return false, err
	}

	// Buy signals:
	// 1. Price touches or goes below the lower band (oversold)
	// 2. Price moves back above the lower band (reversal)
	touchLowerBand := current <= kc.lastLower*1.005 // Small tolerance (0.5%)
	
	return touchLowerBand, nil
}

// ShouldSell determines if we should sell based on Keltner Channels
func (kc *KeltnerChannels) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	_, err := kc.Calculate(data)
	if err != nil {
		return false, err
	}

	// Sell signals:
	// 1. Price touches or goes above the upper band (overbought)
	// 2. Price moves back below the upper band (reversal)
	touchUpperBand := current >= kc.lastUpper*0.995 // Small tolerance (0.5%)
	
	return touchUpperBand, nil
}

// GetSignalStrength returns the signal strength based on current price position within channels
func (kc *KeltnerChannels) GetSignalStrength() float64 {
	// This is a simplified calculation since we don't track the current price internally
	// In practice, you would call GetChannelPosition(currentPrice) for accurate strength
	return 0.5 // Default moderate strength
}

// GetSignalStrengthForPrice returns the signal strength based on given price position within channels
func (kc *KeltnerChannels) GetSignalStrengthForPrice(currentPrice float64) float64 {
	if kc.lastUpper == kc.lastLower {
		return 0 // No channel width
	}
	
	// Calculate position within the channel (0 = lower band, 1 = upper band)
	position := kc.GetChannelPosition(currentPrice)
	
	// Signal strength increases as price approaches the bands
	if position <= 0.2 {
		// Near lower band - buy signal strength
		return (0.2 - position) / 0.2
	} else if position >= 0.8 {
		// Near upper band - sell signal strength
		return (position - 0.8) / 0.2
	}
	
	return 0 // Neutral zone
}

// GetName returns the indicator name
func (kc *KeltnerChannels) GetName() string {
	return "Keltner Channels"
}

// String returns the string representation of the Keltner Channels (like the reference implementation)
func (kc *KeltnerChannels) String() string {
	return fmt.Sprintf("KC(%d)", kc.period)
}

// GetRequiredPeriods returns the minimum number of periods needed
func (kc *KeltnerChannels) GetRequiredPeriods() int {
	// Match reference implementation: use ATR's idle period
	return kc.atrIndicator.GetIdlePeriod()
}

// GetIdlePeriod returns the initial period that Keltner Channel won't yield reliable results
// This matches the reference implementation's IdlePeriod method
func (kc *KeltnerChannels) GetIdlePeriod() int {
	return kc.atrIndicator.GetIdlePeriod()
}

// GetChannels returns the current channel values
func (kc *KeltnerChannels) GetChannels() (upper, middle, lower float64) {
	return kc.lastUpper, kc.lastMiddle, kc.lastLower
}

// GetMiddleLine returns the middle line (EMA) value
func (kc *KeltnerChannels) GetMiddleLine() float64 {
	return kc.lastMiddle
}

// GetUpperBand returns the upper band value
func (kc *KeltnerChannels) GetUpperBand() float64 {
	return kc.lastUpper
}

// GetLowerBand returns the lower band value
func (kc *KeltnerChannels) GetLowerBand() float64 {
	return kc.lastLower
}

// SetMultiplier sets the ATR multiplier for band calculation
func (kc *KeltnerChannels) SetMultiplier(multiplier float64) {
	kc.multiplier = multiplier
}

// GetMultiplier returns the current ATR multiplier
func (kc *KeltnerChannels) GetMultiplier() float64 {
	return kc.multiplier
}

// IsAboveUpperBand returns true if current price is above upper band
func (kc *KeltnerChannels) IsAboveUpperBand(price float64) bool {
	return price > kc.lastUpper
}

// IsBelowLowerBand returns true if current price is below lower band
func (kc *KeltnerChannels) IsBelowLowerBand(price float64) bool {
	return price < kc.lastLower
}

// IsWithinChannels returns true if price is within the channels
func (kc *KeltnerChannels) IsWithinChannels(price float64) bool {
	return price >= kc.lastLower && price <= kc.lastUpper
}

// GetChannelWidth returns the current width of the channel
func (kc *KeltnerChannels) GetChannelWidth() float64 {
	return kc.lastUpper - kc.lastLower
}

// GetChannelPosition returns the position of price within channel (0-1 range)
func (kc *KeltnerChannels) GetChannelPosition(price float64) float64 {
	if kc.lastUpper == kc.lastLower {
		return 0.5 // Default middle if no width
	}
	
	position := (price - kc.lastLower) / (kc.lastUpper - kc.lastLower)
	return math.Max(0, math.Min(1, position)) // Clamp to 0-1 range
}
