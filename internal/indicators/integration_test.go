package indicators

import (
	"math"
	"testing"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSuite для интеграционного тестирования всех индикаторов
type IndicatorTestSuite struct {
	data []types.OHLCV
}

func TestIndicatorIntegration(t *testing.T) {
	suite := &IndicatorTestSuite{
		data: generateRealisticData(100),
	}

	// Тестируем все индикаторы с одними и теми же данными
	t.Run("RSI_Integration", suite.testRSIIntegration)
	t.Run("SMA_Integration", suite.testSMAIntegration)
	t.Run("MACD_Integration", suite.testMACDIntegration)
	t.Run("BollingerBands_Integration", suite.testBollingerBandsIntegration)
	t.Run("MultiIndicator_Integration", suite.testMultiIndicatorIntegration)
}

func (suite *IndicatorTestSuite) testRSIIntegration(t *testing.T) {
	rsi := NewRSI(14)

	// Тестируем последовательные вычисления
	for i := 14; i < len(suite.data); i++ {
		dataSlice := suite.data[:i+1]
		value, err := rsi.Calculate(dataSlice)
		require.NoError(t, err)

		assert.GreaterOrEqual(t, value, 0.0)
		assert.LessOrEqual(t, value, 100.0)

		// Проверяем сигналы
		currentPrice := dataSlice[len(dataSlice)-1].Close
		shouldBuy, err := rsi.ShouldBuy(currentPrice, dataSlice)
		require.NoError(t, err)

		shouldSell, err := rsi.ShouldSell(currentPrice, dataSlice)
		require.NoError(t, err)

		// Нельзя одновременно покупать и продавать
		assert.False(t, shouldBuy && shouldSell)

		// Проверяем силу сигнала
		strength := rsi.GetSignalStrength()
		assert.GreaterOrEqual(t, strength, 0.0)
		assert.LessOrEqual(t, strength, 1.0)
	}
}

func (suite *IndicatorTestSuite) testSMAIntegration(t *testing.T) {
	sma := NewSMA(20)

	// Тестируем последовательные вычисления
	for i := 19; i < len(suite.data); i++ {
		dataSlice := suite.data[:i+1]
		value, err := sma.Calculate(dataSlice)
		require.NoError(t, err)

		assert.Greater(t, value, 0.0)

		// Проверяем сигналы
		currentPrice := dataSlice[len(dataSlice)-1].Close
		shouldBuy, err := sma.ShouldBuy(currentPrice, dataSlice)
		require.NoError(t, err)

		shouldSell, err := sma.ShouldSell(currentPrice, dataSlice)
		require.NoError(t, err)

		// Логика SMA: если цена выше SMA, то buy=true, sell=false
		// если цена ниже SMA, то buy=false, sell=true
		if currentPrice > value {
			assert.True(t, shouldBuy)
			assert.False(t, shouldSell)
		} else if currentPrice < value {
			assert.False(t, shouldBuy)
			assert.True(t, shouldSell)
		}
	}
}

func (suite *IndicatorTestSuite) testMACDIntegration(t *testing.T) {
	macd := NewMACD(12, 26, 9)

	// Тестируем последовательные вычисления
	for i := 25; i < len(suite.data); i++ {
		dataSlice := suite.data[:i+1]
		_, err := macd.Calculate(dataSlice)
		require.NoError(t, err)

		// Проверяем сигналы
		currentPrice := dataSlice[len(dataSlice)-1].Close
		shouldBuy, err := macd.ShouldBuy(currentPrice, dataSlice)
		require.NoError(t, err)

		shouldSell, err := macd.ShouldSell(currentPrice, dataSlice)
		require.NoError(t, err)

		// Нельзя одновременно покупать и продавать
		assert.False(t, shouldBuy && shouldSell)

		// Проверяем силу сигнала
		strength := macd.GetSignalStrength()
		assert.GreaterOrEqual(t, strength, 0.0)
		assert.LessOrEqual(t, strength, 1.0)
	}
}

func (suite *IndicatorTestSuite) testBollingerBandsIntegration(t *testing.T) {
	bb := NewBollingerBands(20, 2.0)

	// Тестируем последовательные вычисления
	for i := 19; i < len(suite.data); i++ {
		dataSlice := suite.data[:i+1]
		value, err := bb.Calculate(dataSlice)
		require.NoError(t, err)

		assert.Greater(t, value, 0.0)
		assert.Greater(t, bb.lastUpper, bb.lastMiddle)
		assert.Less(t, bb.lastLower, bb.lastMiddle)

		// Проверяем сигналы
		currentPrice := dataSlice[len(dataSlice)-1].Close
		shouldBuy, err := bb.ShouldBuy(currentPrice, dataSlice)
		require.NoError(t, err)

		shouldSell, err := bb.ShouldSell(currentPrice, dataSlice)
		require.NoError(t, err)

		// Нельзя одновременно покупать и продавать
		assert.False(t, shouldBuy && shouldSell)

		// Проверяем логику Bollinger Bands
		if currentPrice <= bb.lastLower*1.01 {
			assert.True(t, shouldBuy)
		}
		if currentPrice >= bb.lastUpper*0.99 {
			assert.True(t, shouldSell)
		}
	}
}

func (suite *IndicatorTestSuite) testMultiIndicatorIntegration(t *testing.T) {
	// Create all indicators
	indicators := []TechnicalIndicator{
		NewRSI(14),
		NewSMA(20),
		NewMACD(12, 26, 9),
		NewBollingerBands(20, 2.0),
	}

	// Test all indicators together
	for i := 25; i < len(suite.data); i++ {
		dataSlice := suite.data[:i+1]
		currentPrice := dataSlice[len(dataSlice)-1].Close

		for _, indicator := range indicators {
			// Check calculation
			value, err := indicator.Calculate(dataSlice)
			require.NoError(t, err)

			// MACD can return negative values, others should be non-negative
			if indicator.GetName() == "MACD" {
				assert.False(t, math.IsNaN(value) || math.IsInf(value, 0), "MACD value should be a valid number")
			} else {
				assert.GreaterOrEqual(t, value, 0.0)
			}

			// Check signals
			shouldBuy, err := indicator.ShouldBuy(currentPrice, dataSlice)
			require.NoError(t, err)

			shouldSell, err := indicator.ShouldSell(currentPrice, dataSlice)
			require.NoError(t, err)

			// Should not buy and sell at the same time
			assert.False(t, shouldBuy && shouldSell)

			// Check signal strength (can be negative for some indicators, e.g. MACD histogram)
			strength := indicator.GetSignalStrength()
			// Only check that it's not NaN or Inf
			assert.False(t, math.IsNaN(strength) || math.IsInf(strength, 0), "Signal strength should be a valid number")

			// Check name and required periods
			name := indicator.GetName()
			assert.NotEmpty(t, name)

			requiredPeriods := indicator.GetRequiredPeriods()
			assert.Greater(t, requiredPeriods, 0)
		}
	}
}

// Тест производительности всех индикаторов
func BenchmarkAllIndicators(b *testing.B) {
	data := generateRealisticData(1000)

	indicators := []TechnicalIndicator{
		NewRSI(14),
		NewSMA(20),
		NewMACD(12, 26, 9),
		NewBollingerBands(20, 2.0),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, indicator := range indicators {
			_, _ = indicator.Calculate(data)
		}
	}
}

// Тест согласованности сигналов
func TestSignalConsistency(t *testing.T) {
	data := generateRealisticData(100)

	indicators := []TechnicalIndicator{
		NewRSI(14),
		NewSMA(20),
		NewMACD(12, 26, 9),
		NewBollingerBands(20, 2.0),
	}

	// Проверяем согласованность сигналов на последних данных
	dataSlice := data[25:]
	currentPrice := dataSlice[len(dataSlice)-1].Close

	buySignals := 0
	sellSignals := 0

	for _, indicator := range indicators {
		shouldBuy, err := indicator.ShouldBuy(currentPrice, dataSlice)
		require.NoError(t, err)

		shouldSell, err := indicator.ShouldSell(currentPrice, dataSlice)
		require.NoError(t, err)

		if shouldBuy {
			buySignals++
		}
		if shouldSell {
			sellSignals++
		}
	}

	// Проверяем, что не все индикаторы дают одинаковые сигналы
	// (это было бы подозрительно)
	assert.Less(t, buySignals, len(indicators))
	assert.Less(t, sellSignals, len(indicators))
}

// Генератор реалистичных данных
func generateRealisticData(count int) []types.OHLCV {
	data := make([]types.OHLCV, count)
	basePrice := 100.0
	volatility := 0.02 // 2% волатильность

	for i := 0; i < count; i++ {
		// Симулируем реалистичные движения цены
		change := (float64(i%7) - 3) * volatility * basePrice
		price := basePrice + change

		// Добавляем случайность
		if i%5 == 0 {
			price *= 1.01 // Случайный всплеск
		} else if i%7 == 0 {
			price *= 0.99 // Случайное падение
		}

		// Обновляем базовую цену
		basePrice = price

		data[i] = types.OHLCV{
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
			Open:      price,
			High:      price * 1.005,
			Low:       price * 0.995,
			Close:     price,
			Volume:    1000.0 + float64(i%100),
		}
	}

	return data
}
