package indicators

import (
	"fmt"
	"strings"
	"time"
)

type PriceData struct {
	Price     float64
	Volume    float64
	Timestamp time.Time
}

// IndicatorType represents the type of technical indicator
type IndicatorType string

const (
	IndicatorTypeSMA              IndicatorType = "SMA"
	IndicatorTypeEMA              IndicatorType = "EMA"
	IndicatorTypeRSI              IndicatorType = "RSI"
	IndicatorTypeMACD             IndicatorType = "MACD"
	IndicatorTypeBollingerBands   IndicatorType = "BOLLINGER_BANDS"
	IndicatorTypeMFI              IndicatorType = "MFI"
	IndicatorTypeWaveTrend        IndicatorType = "WAVETREND"
	IndicatorTypeKeltnerChannels  IndicatorType = "KELTNER_CHANNELS"
	IndicatorTypeHullMA           IndicatorType = "HULL_MA"
)

// IndicatorFactory creates technical indicators based on type and parameters
type IndicatorFactory struct{}

// NewIndicatorFactory creates a new indicator factory
func NewIndicatorFactory() *IndicatorFactory {
	return &IndicatorFactory{}
}

// CreateIndicator creates a technical indicator of the specified type
func (f *IndicatorFactory) CreateIndicator(indicatorType IndicatorType, params map[string]interface{}) (TechnicalIndicator, error) {
	switch indicatorType {
	case IndicatorTypeSMA:
		period, ok := params["period"].(int)
		if !ok {
			return nil, fmt.Errorf("sma requires 'period' parameter")
		}
		return NewSMA(period), nil
		
	case IndicatorTypeEMA:
		period, ok := params["period"].(int)
		if !ok {
			return nil, fmt.Errorf("ema requires 'period' parameter")
		}
		return NewEMA(period), nil
		
	case IndicatorTypeRSI:
		period, ok := params["period"].(int)
		if !ok {
			return nil, fmt.Errorf("rsi requires 'period' parameter")
		}
		return NewRSI(period), nil
		
	case IndicatorTypeMACD:
		fastPeriod := 12
		slowPeriod := 26
		signalPeriod := 9
		
		if fp, ok := params["fast_period"].(int); ok {
			fastPeriod = fp
		}
		if sp, ok := params["slow_period"].(int); ok {
			slowPeriod = sp
		}
		if sgp, ok := params["signal_period"].(int); ok {
			signalPeriod = sgp
		}
		
		return NewMACD(fastPeriod, slowPeriod, signalPeriod), nil
		
	case IndicatorTypeBollingerBands:
		period := 20
		stdDev := 2.0
		
		if p, ok := params["period"].(int); ok {
			period = p
		}
		if sd, ok := params["std_dev"].(float64); ok {
			stdDev = sd
		}
		
		return NewBollingerBands(period, stdDev), nil
		
	case IndicatorTypeMFI:
		period := DefaultMfiPeriod
		if p, ok := params["period"].(int); ok {
			period = p
		}
		return NewMFIWithPeriod(period), nil
		
	case IndicatorTypeWaveTrend:
		channelLength := 10
		averageLength := 21
		
		if cl, ok := params["channel_length"].(int); ok {
			channelLength = cl
		}
		if al, ok := params["average_length"].(int); ok {
			averageLength = al
		}
		
		return NewWaveTrendCustom(channelLength, averageLength), nil
		
	case IndicatorTypeKeltnerChannels:
		period := DefaultKeltnerChannelPeriod
		multiplier := 2.0
		
		if p, ok := params["period"].(int); ok {
			period = p
		}
		if m, ok := params["multiplier"].(float64); ok {
			multiplier = m
		}
		
		return NewKeltnerChannelsCustom(period, multiplier), nil
		
	case IndicatorTypeHullMA:
		period, ok := params["period"].(int)
		if !ok {
			return nil, fmt.Errorf("hull ma requires 'period' parameter")
		}
		return NewHullMA(period), nil
		
	default:
		return nil, fmt.Errorf("unknown indicator type: %s", indicatorType)
	}
}

// GetAvailableIndicators returns a list of available indicator types
func (f *IndicatorFactory) GetAvailableIndicators() []IndicatorType {
	return []IndicatorType{
		IndicatorTypeSMA,
		IndicatorTypeEMA,
		IndicatorTypeRSI,
		IndicatorTypeMACD,
		IndicatorTypeBollingerBands,
		IndicatorTypeMFI,
		IndicatorTypeWaveTrend,
		IndicatorTypeKeltnerChannels,
		IndicatorTypeHullMA,
	}
}

// ParseIndicatorType parses a string into an IndicatorType
func ParseIndicatorType(s string) (IndicatorType, error) {
	switch strings.ToUpper(s) {
	case "SMA":
		return IndicatorTypeSMA, nil
	case "EMA":
		return IndicatorTypeEMA, nil
	case "RSI":
		return IndicatorTypeRSI, nil
	case "MACD":
		return IndicatorTypeMACD, nil
	case "BOLLINGER_BANDS", "BB":
		return IndicatorTypeBollingerBands, nil
	case "MFI":
		return IndicatorTypeMFI, nil
	case "WAVETREND", "WT":
		return IndicatorTypeWaveTrend, nil
	case "KELTNER_CHANNELS", "KC":
		return IndicatorTypeKeltnerChannels, nil
	case "HULL_MA", "HMA":
		return IndicatorTypeHullMA, nil
	default:
		return "", fmt.Errorf("unknown indicator type: %s", s)
	}
}

// IndicatorConfig represents configuration for an indicator
type IndicatorConfig struct {
	Type       IndicatorType          `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	Weight     float64                `json:"weight,omitempty"` // For weighted combinations
}

// MultiIndicatorSignal combines signals from multiple indicators
type MultiIndicatorSignal struct {
	BuySignals    []string  `json:"buy_signals"`
	SellSignals   []string  `json:"sell_signals"`
	OverallSignal SignalType `json:"overall_signal"`
	Confidence    float64   `json:"confidence"`
	Timestamp     time.Time `json:"timestamp"`
}
