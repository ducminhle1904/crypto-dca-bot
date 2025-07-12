package indicators

import (
	"testing"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRSI(t *testing.T) {
	rsi := NewRSI(14)

	assert.NotNil(t, rsi)
	assert.Equal(t, 14, rsi.period)
	assert.Equal(t, 70.0, rsi.overbought)
	assert.Equal(t, 30.0, rsi.oversold)
	assert.False(t, rsi.initialized)
}

func TestRSI_Calculate_InsufficientData(t *testing.T) {
	rsi := NewRSI(14)
	data := generateTestData(10) // Less than period + 1

	_, err := rsi.Calculate(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient data points")
}

func TestRSI_Calculate_InitialCalculation(t *testing.T) {
	rsi := NewRSI(14)
	data := generateTestData(20)

	value, err := rsi.Calculate(data)
	require.NoError(t, err)

	assert.True(t, rsi.initialized)
	assert.GreaterOrEqual(t, value, 0.0)
	assert.LessOrEqual(t, value, 100.0)
	assert.Equal(t, value, rsi.lastValue)
}

func TestRSI_Calculate_IncrementalCalculation(t *testing.T) {
	rsi := NewRSI(14)
	data := generateTestData(20)

	// First calculation
	value1, err := rsi.Calculate(data)
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

	value2, err := rsi.Calculate(newData)
	require.NoError(t, err)

	// Values should be different due to incremental calculation
	assert.NotEqual(t, value1, value2)
}

func TestRSI_Calculate_AllGains(t *testing.T) {
	rsi := NewRSI(14)
	data := generateRisingData(20)

	value, err := rsi.Calculate(data)
	require.NoError(t, err)

	// With all gains, RSI should be high (close to 100)
	assert.Greater(t, value, 80.0)
}

func TestRSI_Calculate_AllLosses(t *testing.T) {
	rsi := NewRSI(14)
	data := generateFallingData(20)

	value, err := rsi.Calculate(data)
	require.NoError(t, err)

	// With all losses, RSI should be low (close to 0)
	assert.Less(t, value, 20.0)
}

func TestRSI_Calculate_ZeroLoss(t *testing.T) {
	rsi := NewRSI(14)
	data := generateFlatData(20)

	value, err := rsi.Calculate(data)
	require.NoError(t, err)

	// With zero losses, RSI should be 100
	assert.Equal(t, 100.0, value)
}

func TestRSI_ShouldBuy(t *testing.T) {
	// Test oversold condition
	rsi := NewRSI(14)
	data := generateFallingData(20)
	currentPrice := data[len(data)-1].Close
	shouldBuy, err := rsi.ShouldBuy(currentPrice, data)
	require.NoError(t, err)
	assert.True(t, shouldBuy, "ShouldBuy should be true for oversold RSI on falling data")

	// Test normal condition with fresh RSI instance
	rsi2 := NewRSI(14)
	data2 := generateTestData(20)
	currentPrice2 := data2[len(data2)-1].Close
	shouldBuy2, err := rsi2.ShouldBuy(currentPrice2, data2)
	require.NoError(t, err)
	assert.False(t, shouldBuy2, "ShouldBuy should be false for normal RSI")
}

func TestRSI_ShouldSell(t *testing.T) {
	// Test overbought condition
	rsi := NewRSI(14)
	data := generateRisingData(20)
	currentPrice := data[len(data)-1].Close
	shouldSell, err := rsi.ShouldSell(currentPrice, data)
	require.NoError(t, err)
	assert.True(t, shouldSell, "ShouldSell should be true for overbought RSI on rising data")

	// Test normal condition with fresh RSI instance
	rsi2 := NewRSI(14)
	data2 := generateTestData(20)
	currentPrice2 := data2[len(data2)-1].Close
	shouldSell2, err := rsi2.ShouldSell(currentPrice2, data2)
	require.NoError(t, err)
	assert.False(t, shouldSell2, "ShouldSell should be false for normal RSI")
}

func TestRSI_GetSignalStrength(t *testing.T) {
	rsi := NewRSI(14)

	// Test oversold signal strength
	rsi.lastValue = 20.0 // Well below oversold
	strength := rsi.GetSignalStrength()
	assert.Greater(t, strength, 0.0)
	assert.LessOrEqual(t, strength, 1.0)

	// Test overbought signal strength
	rsi.lastValue = 80.0 // Well above overbought
	strength = rsi.GetSignalStrength()
	assert.Greater(t, strength, 0.0)
	assert.LessOrEqual(t, strength, 1.0)

	// Test neutral signal strength
	rsi.lastValue = 50.0 // Neutral level
	strength = rsi.GetSignalStrength()
	assert.Equal(t, 0.0, strength)
}

func TestRSI_GetName(t *testing.T) {
	rsi := NewRSI(14)
	assert.Equal(t, "RSI", rsi.GetName())
}

func TestRSI_GetRequiredPeriods(t *testing.T) {
	rsi := NewRSI(14)
	assert.Equal(t, 15, rsi.GetRequiredPeriods()) // period + 1
}

func TestRSI_SetOversold(t *testing.T) {
	rsi := NewRSI(14)
	rsi.SetOversold(25.0)
	assert.Equal(t, 25.0, rsi.oversold)
}

func TestRSI_SetOverbought(t *testing.T) {
	rsi := NewRSI(14)
	rsi.SetOverbought(75.0)
	assert.Equal(t, 75.0, rsi.overbought)
}

func TestRSI_InterfaceCompliance(t *testing.T) {
	var _ TechnicalIndicator = NewRSI(14)
}

// Benchmark tests
func BenchmarkRSI_Calculate(b *testing.B) {
	rsi := NewRSI(14)
	data := generateTestData(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rsi.Calculate(data)
	}
}

// Helper functions
func generateTestData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		// Add some randomness to price movements
		change := (float64(i%3) - 1) * 2.0 // -2, 0, or 2
		price := basePrice + change

		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 1.0,
			Low:       price - 1.0,
			Close:     price,
			Volume:    1000.0,
		}
		basePrice = price
	}

	return data
}

func generateRisingData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		price := basePrice + float64(i)*2.0
		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 1.0,
			Low:       price - 1.0,
			Close:     price,
			Volume:    1000.0,
		}
	}

	return data
}

func generateFallingData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		price := basePrice - float64(i)*2.0
		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 1.0,
			Low:       price - 1.0,
			Close:     price,
			Volume:    1000.0,
		}
	}

	return data
}

func generateFlatData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	price := 100.0

	for i := 0; i < count; i++ {
		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 0.1,
			Low:       price - 0.1,
			Close:     price,
			Volume:    1000.0,
		}
	}

	return data
}
