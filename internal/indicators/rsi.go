package indicators

import (
	"errors"
	"math"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

type RSI struct {
	period      int
	overbought  float64
	oversold    float64
	lastValue   float64
	avgGain     float64
	avgLoss     float64
	initialized bool
	dataPoints  int
}

func NewRSI(period int) *RSI {
	return &RSI{
		period:     period,
		overbought: 70.0,
		oversold:   30.0,
	}
}

func (r *RSI) Calculate(data []types.OHLCV) (float64, error) {
	if len(data) < r.period+1 {
		return 0, errors.New("insufficient data points for RSI calculation")
	}

	// For the first calculation, we use SMA
	if !r.initialized {
		return r.initialCalculation(data)
	}

	// For subsequent calculations, we use EMA for optimization
	return r.incrementalCalculation(data)
}

func (r *RSI) initialCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < r.period+1 {
		return 0, errors.New("not enough data for initial RSI calculation")
	}

	gains := 0.0
	losses := 0.0

	// We take the last period+1 values
	recent := data[len(data)-r.period-1:]

	for i := 1; i < len(recent); i++ {
		change := recent[i].Close - recent[i-1].Close
		if change > 0 {
			gains += change
		} else {
			losses += math.Abs(change)
		}
	}

	r.avgGain = gains / float64(r.period)
	r.avgLoss = losses / float64(r.period)

	if r.avgLoss == 0 {
		r.lastValue = 100
		r.initialized = true
		return 100, nil
	}

	rs := r.avgGain / r.avgLoss
	r.lastValue = 100 - (100 / (1 + rs))
	r.initialized = true

	return r.lastValue, nil
}

func (r *RSI) incrementalCalculation(data []types.OHLCV) (float64, error) {
	if len(data) < 2 {
		return r.lastValue, nil
	}

	// We take only the last change for the incremental calculation.
	lastTwo := data[len(data)-2:]
	change := lastTwo[1].Close - lastTwo[0].Close

	gain := 0.0
	loss := 0.0

	if change > 0 {
		gain = change
	} else {
		loss = math.Abs(change)
	}

	// Wilder's smoothing (Modified EMA)
	alpha := 1.0 / float64(r.period)
	r.avgGain = (r.avgGain * (1 - alpha)) + (gain * alpha)
	r.avgLoss = (r.avgLoss * (1 - alpha)) + (loss * alpha)

	if r.avgLoss == 0 {
		r.lastValue = 100
		return 100, nil
	}

	rs := r.avgGain / r.avgLoss
	r.lastValue = 100 - (100 / (1 + rs))

	return r.lastValue, nil
}

func (r *RSI) ShouldBuy(current float64, data []types.OHLCV) (bool, error) {
	rsiValue, err := r.Calculate(data)
	if err != nil {
		return false, err
	}

	return rsiValue < r.oversold, nil
}

func (r *RSI) ShouldSell(current float64, data []types.OHLCV) (bool, error) {
	rsiValue, err := r.Calculate(data)
	if err != nil {
		return false, err
	}

	return rsiValue > r.overbought, nil
}

func (r *RSI) GetSignalStrength() float64 {
	if r.lastValue < r.oversold {
		return (r.oversold - r.lastValue) / r.oversold
	}
	if r.lastValue > r.overbought {
		return (r.lastValue - r.overbought) / (100 - r.overbought)
	}
	return 0
}

func (r *RSI) GetName() string {
	return "RSI"
}

func (r *RSI) GetRequiredPeriods() int {
	return r.period + 1
}

// SetOversold sets the oversold threshold
func (r *RSI) SetOversold(threshold float64) {
	r.oversold = threshold
}

// SetOverbought sets the overbought threshold
func (r *RSI) SetOverbought(threshold float64) {
	r.overbought = threshold
}
