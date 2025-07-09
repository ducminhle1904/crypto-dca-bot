package indicators

import (
	"testing"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

func TestRSI_Calculate(t *testing.T) {
	rsi := NewRSI(14)

	// Create test data
	data := make([]types.OHLCV, 20)
	for i := 0; i < 20; i++ {
		data[i] = types.OHLCV{
			Open:      100.0,
			High:      105.0,
			Low:       95.0,
			Close:     100.0 + float64(i),
			Volume:    1000.0,
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
		}
	}

	// Test calculation
	value, err := rsi.Calculate(data)
	if err != nil {
		t.Fatalf("RSI calculation failed: %v", err)
	}

	if value < 0 || value > 100 {
		t.Errorf("RSI value out of range: %f", value)
	}
}

func TestRSI_ShouldBuy(t *testing.T) {
	rsi := NewRSI(14)

	// Create oversold data (declining prices)
	data := make([]types.OHLCV, 20)
	for i := 0; i < 20; i++ {
		data[i] = types.OHLCV{
			Open:      100.0,
			High:      105.0,
			Low:       95.0,
			Close:     100.0 - float64(i), // Declining prices
			Volume:    1000.0,
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
		}
	}

	// Test buy signal
	shouldBuy, err := rsi.ShouldBuy(90.0, data)
	if err != nil {
		t.Fatalf("ShouldBuy failed: %v", err)
	}

	// Should buy when RSI is oversold (below 30)
	if !shouldBuy {
		t.Error("Expected buy signal for oversold condition")
	}
}

func TestRSI_ShouldSell(t *testing.T) {
	rsi := NewRSI(14)

	// Create overbought data (rising prices)
	data := make([]types.OHLCV, 20)
	for i := 0; i < 20; i++ {
		data[i] = types.OHLCV{
			Open:      100.0,
			High:      105.0,
			Low:       95.0,
			Close:     100.0 + float64(i), // Rising prices
			Volume:    1000.0,
			Timestamp: time.Now().Add(time.Duration(i) * time.Hour),
		}
	}

	// Test sell signal
	shouldSell, err := rsi.ShouldSell(110.0, data)
	if err != nil {
		t.Fatalf("ShouldSell failed: %v", err)
	}

	// Should sell when RSI is overbought (above 70)
	if !shouldSell {
		t.Error("Expected sell signal for overbought condition")
	}
}

func TestRSI_GetSignalStrength(t *testing.T) {
	rsi := NewRSI(14)

	// Test signal strength calculation
	strength := rsi.GetSignalStrength()
	if strength < 0 || strength > 1 {
		t.Errorf("Signal strength out of range: %f", strength)
	}
}

func TestRSI_GetName(t *testing.T) {
	rsi := NewRSI(14)

	name := rsi.GetName()
	if name != "RSI" {
		t.Errorf("Expected name 'RSI', got '%s'", name)
	}
}

func TestRSI_GetRequiredPeriods(t *testing.T) {
	rsi := NewRSI(14)

	periods := rsi.GetRequiredPeriods()
	if periods != 15 { // period + 1
		t.Errorf("Expected 15 periods, got %d", periods)
	}
}
