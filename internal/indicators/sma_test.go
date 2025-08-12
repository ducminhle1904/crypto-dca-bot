package indicators

import (
	"testing"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSMA(t *testing.T) {
	sma := NewSMA(20)

	assert.NotNil(t, sma)
	assert.Equal(t, 20, sma.period)
	assert.Equal(t, 0.0, sma.lastValue)
}

func TestSMA_Calculate_InsufficientData(t *testing.T) {
	sma := NewSMA(20)
	data := generateTestData(10) // Less than period

	_, err := sma.Calculate(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient data")
}

func TestSMA_Calculate_SufficientData(t *testing.T) {
	sma := NewSMA(5)
	data := generateTestData(10)

	value, err := sma.Calculate(data)
	require.NoError(t, err)

	assert.Greater(t, value, 0.0)
	assert.Equal(t, value, sma.lastValue)
}

func TestSMA_Calculate_ExactPeriod(t *testing.T) {
	sma := NewSMA(5)
	data := generateTestData(5)

	value, err := sma.Calculate(data)
	require.NoError(t, err)

	// Calculate expected SMA manually
	expectedSum := 0.0
	for _, d := range data {
		expectedSum += d.Close
	}
	expectedSMA := expectedSum / 5.0

	assert.InDelta(t, expectedSMA, value, 0.01)
}

func TestSMA_Calculate_MoreThanPeriod(t *testing.T) {
	sma := NewSMA(5)
	data := generateTestData(10)

	value, err := sma.Calculate(data)
	require.NoError(t, err)

	// Should use only the last 5 values
	expectedSum := 0.0
	for i := 5; i < 10; i++ {
		expectedSum += data[i].Close
	}
	expectedSMA := expectedSum / 5.0

	assert.InDelta(t, expectedSMA, value, 0.01)
}

func TestSMA_Calculate_ConsistentValues(t *testing.T) {
	sma := NewSMA(5)
	data := generateFlatData(10) // All values are 100.0

	value, err := sma.Calculate(data)
	require.NoError(t, err)

	assert.Equal(t, 100.0, value)
}

func TestSMA_ShouldBuy_AboveSMA(t *testing.T) {
	sma := NewSMA(5)
	sma.lastValue = 100.0
	currentPrice := 110.0

	shouldBuy, err := sma.ShouldBuy(currentPrice, generateTestData(10))
	require.NoError(t, err)
	assert.True(t, shouldBuy)
}

func TestSMA_ShouldBuy_BelowSMA(t *testing.T) {
	sma := NewSMA(5)
	sma.lastValue = 100.0
	currentPrice := 90.0

	shouldBuy, err := sma.ShouldBuy(currentPrice, generateTestData(10))
	require.NoError(t, err)
	assert.False(t, shouldBuy)
}

func TestSMA_ShouldBuy_EqualSMA(t *testing.T) {
	sma := NewSMA(5)
	// Use flat data so currentPrice == SMA
	data := generateFlatData(10)
	currentPrice := data[len(data)-1].Close
	shouldBuy, err := sma.ShouldBuy(currentPrice, data)
	require.NoError(t, err)
	assert.False(t, shouldBuy, "ShouldBuy should be false when price equals SMA")
}

func TestSMA_ShouldSell_BelowSMA(t *testing.T) {
	sma := NewSMA(5)
	sma.lastValue = 100.0
	currentPrice := 90.0

	shouldSell, err := sma.ShouldSell(currentPrice, generateTestData(10))
	require.NoError(t, err)
	assert.True(t, shouldSell)
}

func TestSMA_ShouldSell_AboveSMA(t *testing.T) {
	sma := NewSMA(5)
	sma.lastValue = 100.0
	currentPrice := 110.0

	shouldSell, err := sma.ShouldSell(currentPrice, generateTestData(10))
	require.NoError(t, err)
	assert.False(t, shouldSell)
}

func TestSMA_ShouldSell_EqualSMA(t *testing.T) {
	sma := NewSMA(5)
	sma.lastValue = 100.0
	currentPrice := 100.0

	shouldSell, err := sma.ShouldSell(currentPrice, generateTestData(10))
	require.NoError(t, err)
	assert.False(t, shouldSell) // Price equal to SMA is not a sell signal
}

func TestSMA_GetSignalStrength(t *testing.T) {
	sma := NewSMA(5)

	strength := sma.GetSignalStrength()
	assert.Equal(t, 0.3, strength) // Default moderate strength
}

func TestSMA_GetName(t *testing.T) {
	sma := NewSMA(5)
	assert.Equal(t, "SMA", sma.GetName())
}

func TestSMA_GetRequiredPeriods(t *testing.T) {
	sma := NewSMA(5)
	assert.Equal(t, 5, sma.GetRequiredPeriods())
}

func TestSMA_InterfaceCompliance(t *testing.T) {
	var _ TechnicalIndicator = NewSMA(5)
}

func TestSMA_Calculate_EdgeCases(t *testing.T) {
	sma := NewSMA(1)
	data := generateTestData(5)

	value, err := sma.Calculate(data)
	require.NoError(t, err)

	// With period 1, should return the last close price
	assert.Equal(t, data[len(data)-1].Close, value)
}

func TestSMA_Calculate_ZeroValues(t *testing.T) {
	sma := NewSMA(5)
	data := make([]types.OHLCV, 5)

	for i := 0; i < 5; i++ {
		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      0.0,
			High:      0.0,
			Low:       0.0,
			Close:     0.0,
			Volume:    0.0,
		}
	}

	value, err := sma.Calculate(data)
	require.NoError(t, err)
	assert.Equal(t, 0.0, value)
}

func TestSMA_Calculate_NegativeValues(t *testing.T) {
	sma := NewSMA(5)
	data := make([]types.OHLCV, 5)

	for i := 0; i < 5; i++ {
		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      -10.0,
			High:      -5.0,
			Low:       -15.0,
			Close:     -10.0,
			Volume:    1000.0,
		}
	}

	value, err := sma.Calculate(data)
	require.NoError(t, err)
	assert.Equal(t, -10.0, value)
}

// Benchmark tests
func BenchmarkSMA_Calculate(b *testing.B) {
	sma := NewSMA(20)
	data := generateTestData(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sma.Calculate(data)
	}
}

func BenchmarkSMA_ShouldBuy(b *testing.B) {
	sma := NewSMA(20)
	data := generateTestData(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sma.ShouldBuy(100.0, data)
	}
}
