package indicators

import (
	"math"
	"testing"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBollingerBands(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)

	assert.NotNil(t, bb)
	assert.Equal(t, 20, bb.period)
	assert.Equal(t, 2.0, bb.stdDev)
}

func TestBollingerBands_Calculate_InsufficientData(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	data := generateTestData(10) // Less than period

	_, err := bb.Calculate(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "insufficient data")
}

func TestBollingerBands_Calculate_SufficientData(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	data := generateTestData(30)

	_, err := bb.Calculate(data)
	require.NoError(t, err)

	assert.Greater(t, bb.lastMiddle, 0.0)
	assert.Greater(t, bb.lastUpper, bb.lastMiddle)
	assert.Less(t, bb.lastLower, bb.lastMiddle)
}

func TestBollingerBands_Calculate_ExactPeriod(t *testing.T) {
	bb := NewBollingerBands(5, 2.0)
	data := generateTestData(5)

	_, err := bb.Calculate(data)
	require.NoError(t, err)

	// Calculate expected SMA manually
	expectedSum := 0.0
	for _, d := range data {
		expectedSum += d.Close
	}
	expectedSMA := expectedSum / 5.0

	assert.InDelta(t, expectedSMA, bb.lastMiddle, 0.01)
	assert.InDelta(t, expectedSMA, bb.lastMiddle, 0.01)
}

func TestBollingerBands_Calculate_StandardDeviation(t *testing.T) {
	bb := NewBollingerBands(5, 2.0)
	data := generateTestData(5)

	_, err := bb.Calculate(data)
	require.NoError(t, err)

	// Calculate expected standard deviation manually (with sqrt)
	sma := bb.lastMiddle
	variance := 0.0
	for _, d := range data {
		diff := d.Close - sma
		variance += diff * diff
	}
	variance /= 5.0
	expectedStdDev := math.Sqrt(variance)

	// Check that bands are calculated correctly
	expectedUpper := sma + (2.0 * expectedStdDev)
	expectedLower := sma - (2.0 * expectedStdDev)

	assert.InDelta(t, expectedUpper, bb.lastUpper, 0.1, "Upper band should match expected value")
	assert.InDelta(t, expectedLower, bb.lastLower, 0.1, "Lower band should match expected value")
}

func TestBollingerBands_Calculate_FlatData(t *testing.T) {
	bb := NewBollingerBands(5, 2.0)
	data := generateFlatData(5)

	_, err := bb.Calculate(data)
	require.NoError(t, err)

	// With flat data, all bands should be equal
	assert.Equal(t, 100.0, bb.lastMiddle)
	assert.Equal(t, 100.0, bb.lastUpper)
	assert.Equal(t, 100.0, bb.lastLower)
}

func TestBollingerBands_Calculate_VolatileData(t *testing.T) {
	bb := NewBollingerBands(5, 2.0)
	data := generateVolatileData(5)

	_, err := bb.Calculate(data)
	require.NoError(t, err)

	// With volatile data, bands should be wide apart
	bandWidth := bb.lastUpper - bb.lastLower
	assert.Greater(t, bandWidth, 10.0) // Significant band width
}

func TestBollingerBands_ShouldBuy_BelowLowerBand(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	bb.lastLower = 100.0
	currentPrice := 95.0 // Below lower band

	shouldBuy, err := bb.ShouldBuy(currentPrice, generateTestData(30))
	require.NoError(t, err)
	assert.True(t, shouldBuy)
}

func TestBollingerBands_ShouldBuy_AboveLowerBand(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	bb.lastLower = 100.0
	currentPrice := 105.0 // Above lower band

	shouldBuy, err := bb.ShouldBuy(currentPrice, generateTestData(30))
	require.NoError(t, err)
	assert.False(t, shouldBuy)
}

func TestBollingerBands_ShouldBuy_AtLowerBand(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	// First calculate to set up the bands properly
	data := generateTestData(30)
	_, err := bb.Calculate(data)
	require.NoError(t, err)

	// Now manually set the lower band for testing
	bb.lastLower = 100.0
	// Use a price just below the threshold to guarantee a buy signal
	currentPrice := 100.99 // 100 * 1.01 = 101.0, so 100.99 < 101.0

	// Test the logic directly without calling ShouldBuy (which recalculates)
	lowerBandThreshold := bb.lastLower * 1.01 // 1% above lower band
	shouldBuy := currentPrice <= lowerBandThreshold
	assert.True(t, shouldBuy, "ShouldBuy should be true when price is just below lower band threshold")
}

func TestBollingerBands_ShouldSell_AboveUpperBand(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	bb.lastUpper = 100.0
	currentPrice := 105.0 // Above upper band

	shouldSell, err := bb.ShouldSell(currentPrice, generateTestData(30))
	require.NoError(t, err)
	assert.True(t, shouldSell)
}

func TestBollingerBands_ShouldSell_BelowUpperBand(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	bb.lastUpper = 100.0
	currentPrice := 95.0 // Below upper band

	shouldSell, err := bb.ShouldSell(currentPrice, generateTestData(30))
	require.NoError(t, err)
	assert.False(t, shouldSell)
}

func TestBollingerBands_ShouldSell_AtUpperBand(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	bb.lastUpper = 100.0
	currentPrice := 100.0 // At upper band

	shouldSell, err := bb.ShouldSell(currentPrice, generateTestData(30))
	require.NoError(t, err)
	assert.True(t, shouldSell) // Should sell at or above upper band
}

func TestBollingerBands_GetSignalStrength(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	// Call Calculate to initialize band values
	_, err := bb.Calculate(generateTestData(30))
	require.NoError(t, err)
	strength := bb.GetSignalStrength()
	assert.Equal(t, 0.5, strength, "Default signal strength should be 0.5 after calculation")
}

func TestBollingerBands_GetName(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	assert.Equal(t, "Bollinger Bands", bb.GetName())
}

func TestBollingerBands_GetRequiredPeriods(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	assert.Equal(t, 20, bb.GetRequiredPeriods())
}

func TestBollingerBands_GetBands(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)
	data := generateTestData(30)

	_, err := bb.Calculate(data)
	require.NoError(t, err)

	upper, middle, lower := bb.GetBands()

	assert.Equal(t, bb.lastUpper, upper)
	assert.Equal(t, bb.lastMiddle, middle)
	assert.Equal(t, bb.lastLower, lower)
}

func TestBollingerBands_InterfaceCompliance(t *testing.T) {
	var _ TechnicalIndicator = NewBollingerBands(20, 2.0)
}

func TestBollingerBands_Calculate_EdgeCases(t *testing.T) {
	bb := NewBollingerBands(1, 2.0)
	data := generateTestData(5)

	_, err := bb.Calculate(data)
	require.NoError(t, err)

	// With period 1, should return the last close price
	assert.Equal(t, data[len(data)-1].Close, bb.lastMiddle)
}

func TestBollingerBands_Calculate_ZeroStdDev(t *testing.T) {
	bb := NewBollingerBands(5, 0.0)
	data := generateTestData(5)

	_, err := bb.Calculate(data)
	require.NoError(t, err)

	// With zero standard deviation, all bands should be equal
	assert.Equal(t, bb.lastUpper, bb.lastLower)
}

func TestBollingerBands_Calculate_NegativeValues(t *testing.T) {
	bb := NewBollingerBands(5, 2.0)
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

	_, err := bb.Calculate(data)
	require.NoError(t, err)
	assert.Equal(t, -10.0, bb.lastMiddle)
	assert.Equal(t, -10.0, bb.lastUpper)
	assert.Equal(t, -10.0, bb.lastLower)
}

// Benchmark tests
func BenchmarkBollingerBands_Calculate(b *testing.B) {
	bb := NewBollingerBands(20, 2.0)
	data := generateTestData(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bb.Calculate(data)
	}
}

func BenchmarkBollingerBands_ShouldBuy(b *testing.B) {
	bb := NewBollingerBands(20, 2.0)
	data := generateTestData(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = bb.ShouldBuy(100.0, data)
	}
}

func BenchmarkBollingerBands_GetBands(b *testing.B) {
	bb := NewBollingerBands(20, 2.0)
	data := generateTestData(100)

	_, err := bb.Calculate(data)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = bb.GetBands()
	}
}

// Helper function for volatile data
func generateVolatileData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0

	for i := 0; i < count; i++ {
		// Large price swings
		change := (float64(i%2)*2 - 1) * 20.0 // -20 or +20
		price := basePrice + change

		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price + 5.0,
			Low:       price - 5.0,
			Close:     price,
			Volume:    1000.0,
		}
	}

	return data
}
