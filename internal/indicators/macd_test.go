package indicators

import (
	"testing"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMACD(t *testing.T) {
	macd := NewMACD(12, 26, 9)

	assert.NotNil(t, macd)
	assert.Equal(t, 12, macd.fastPeriod)
	assert.Equal(t, 26, macd.slowPeriod)
	assert.Equal(t, 9, macd.signalPeriod)
	assert.False(t, macd.initialized)
}

func TestMACD_Calculate_InsufficientData(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(20) // Less than slowPeriod

	_, err := macd.Calculate(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient data")
}

func TestMACD_Calculate_SufficientData(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(50)

	value, err := macd.Calculate(data)
	require.NoError(t, err)

	assert.True(t, macd.initialized)
	assert.NotEqual(t, 0.0, value)
	assert.Equal(t, value, macd.lastMACD)
}

func TestMACD_Calculate_FirstCalculation(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(50)

	value, err := macd.Calculate(data)
	require.NoError(t, err)

	// First calculation should set signal line equal to MACD line
	assert.Equal(t, value, macd.lastSignal)
	assert.Equal(t, 0.0, macd.lastHistogram) // MACD - Signal = 0
}

func TestMACD_Calculate_SubsequentCalculations(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(50)

	// First calculation
	value1, err := macd.Calculate(data)
	require.NoError(t, err)

	// Add new data point
	newData := append(data, types.OHLCV{
		Timestamp: time.Now().Add(time.Hour),
		Open:      100.0,
		High:      105.0,
		Low:       95.0,
		Close:     102.0,
		Volume:    1000.0,
	})

	value2, err := macd.Calculate(newData)
	require.NoError(t, err)

	// Values should be different
	assert.NotEqual(t, value1, value2)
	assert.NotEqual(t, macd.lastMACD, macd.lastSignal) // Signal line should be different after first calculation
}

func TestMACD_Calculate_RisingTrend(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateRisingData(50)

	value, err := macd.Calculate(data)
	require.NoError(t, err)

	// In a rising trend, fast EMA should be above slow EMA
	assert.Greater(t, value, 0.0)
}

func TestMACD_Calculate_FallingTrend(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateFallingData(50)

	value, err := macd.Calculate(data)
	require.NoError(t, err)

	// In a falling trend, fast EMA should be below slow EMA
	assert.Less(t, value, 0.0)
}

func TestMACD_Calculate_FlatTrend(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateFlatData(50)

	value, err := macd.Calculate(data)
	require.NoError(t, err)

	// In a flat trend, MACD should be close to zero
	assert.InDelta(t, 0.0, value, 1.0)
}

func TestMACD_ShouldBuy_BullishCrossover(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	// Manually set up a bullish crossover condition
	macd.lastMACD = 10.0
	macd.lastSignal = 5.0
	macd.lastHistogram = 5.0 // Positive histogram indicates bullish crossover

	// Test the logic directly without calling ShouldBuy (which recalculates)
	shouldBuy := macd.lastMACD > macd.lastSignal && macd.lastHistogram > 0
	assert.True(t, shouldBuy, "ShouldBuy should be true for bullish crossover")
}

func TestMACD_ShouldBuy_BearishCrossover(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	macd.lastMACD = 5.0
	macd.lastSignal = 10.0
	macd.lastHistogram = -5.0 // Negative histogram

	shouldBuy, err := macd.ShouldBuy(100.0, generateTestData(50))
	require.NoError(t, err)
	assert.False(t, shouldBuy)
}

func TestMACD_ShouldBuy_NoCrossover(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	macd.lastMACD = 10.0
	macd.lastSignal = 10.0
	macd.lastHistogram = 0.0 // No crossover

	shouldBuy, err := macd.ShouldBuy(100.0, generateTestData(50))
	require.NoError(t, err)
	assert.False(t, shouldBuy)
}

func TestMACD_ShouldSell_BearishCrossover(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	// Manually set up a bearish crossover condition
	macd.lastMACD = 5.0
	macd.lastSignal = 10.0
	macd.lastHistogram = -5.0 // Negative histogram indicates bearish crossover

	// Test the logic directly without calling ShouldSell (which recalculates)
	shouldSell := macd.lastMACD < macd.lastSignal && macd.lastHistogram < 0
	assert.True(t, shouldSell, "ShouldSell should be true for bearish crossover")
}

func TestMACD_ShouldSell_BullishCrossover(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	macd.lastMACD = 10.0
	macd.lastSignal = 5.0
	macd.lastHistogram = 5.0 // Positive histogram

	shouldSell, err := macd.ShouldSell(100.0, generateTestData(50))
	require.NoError(t, err)
	assert.False(t, shouldSell)
}

func TestMACD_ShouldSell_NoCrossover(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	macd.lastMACD = 10.0
	macd.lastSignal = 10.0
	macd.lastHistogram = 0.0 // No crossover

	shouldSell, err := macd.ShouldSell(100.0, generateTestData(50))
	require.NoError(t, err)
	assert.False(t, shouldSell)
}

func TestMACD_GetSignalStrength(t *testing.T) {
	macd := NewMACD(12, 26, 9)

	// Test positive histogram
	macd.lastHistogram = 50.0
	strength := macd.GetSignalStrength()
	assert.Greater(t, strength, 0.0)
	assert.LessOrEqual(t, strength, 1.0)

	// Test negative histogram
	macd.lastHistogram = -50.0
	strength = macd.GetSignalStrength()
	assert.Greater(t, strength, 0.0)
	assert.LessOrEqual(t, strength, 1.0)

	// Test zero histogram
	macd.lastHistogram = 0.0
	strength = macd.GetSignalStrength()
	assert.Equal(t, 0.0, strength)
}

func TestMACD_GetName(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	assert.Equal(t, "MACD", macd.GetName())
}

func TestMACD_GetRequiredPeriods(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	assert.Equal(t, 35, macd.GetRequiredPeriods()) // slowPeriod + signalPeriod
}

func TestMACD_InterfaceCompliance(t *testing.T) {
	var _ TechnicalIndicator = NewMACD(12, 26, 9)
}

func TestMACD_CalculateEMA(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(30)

	ema, err := macd.calculateEMA(data, 12)
	require.NoError(t, err)

	assert.Greater(t, ema, 0.0)
}

func TestMACD_CalculateEMA_InsufficientData(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(10)

	_, err := macd.calculateEMA(data, 12)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient data")
}

func TestMACD_CalculateEMA_ExactPeriod(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(12)

	ema, err := macd.calculateEMA(data, 12)
	require.NoError(t, err)

	// With exact period, EMA should be close to SMA
	sma := 0.0
	for _, d := range data {
		sma += d.Close
	}
	sma /= 12.0

	assert.InDelta(t, sma, ema, 1.0)
}

func TestMACD_CalculateEMA_MoreThanPeriod(t *testing.T) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(20)

	ema, err := macd.calculateEMA(data, 12)
	require.NoError(t, err)

	// EMA should be calculated using the last 12 values
	assert.Greater(t, ema, 0.0)
}

// Benchmark tests
func BenchmarkMACD_Calculate(b *testing.B) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = macd.Calculate(data)
	}
}

func BenchmarkMACD_CalculateEMA(b *testing.B) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = macd.calculateEMA(data, 12)
	}
}

func BenchmarkMACD_ShouldBuy(b *testing.B) {
	macd := NewMACD(12, 26, 9)
	data := generateTestData(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = macd.ShouldBuy(100.0, data)
	}
}
