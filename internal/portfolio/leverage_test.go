package portfolio

import (
	"testing"
)

func TestLeverageCalculator(t *testing.T) {
	calc := NewLeverageCalculator()
	
	tests := []struct {
		name           string
		positionValue  float64
		leverage       float64
		expectedMargin float64
	}{
		{
			name:           "10x leverage on $100 position",
			positionValue:  100.0,
			leverage:       10.0,
			expectedMargin: 10.0,
		},
		{
			name:           "50x leverage on $1000 position",
			positionValue:  1000.0,
			leverage:       50.0,
			expectedMargin: 20.0,
		},
		{
			name:           "1x leverage (spot) on $500 position",
			positionValue:  500.0,
			leverage:       1.0,
			expectedMargin: 500.0,
		},
		{
			name:           "Zero leverage should return full amount",
			positionValue:  200.0,
			leverage:       0.0,
			expectedMargin: 200.0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			margin := calc.CalculateRequiredMargin(tt.positionValue, tt.leverage)
			if margin != tt.expectedMargin {
				t.Errorf("CalculateRequiredMargin() = %.2f, want %.2f", margin, tt.expectedMargin)
			}
		})
	}
}

func TestLeverageValidation(t *testing.T) {
	calc := NewLeverageCalculator()
	
	tests := []struct {
		name      string
		leverage  float64
		shouldErr bool
	}{
		{"Valid 10x leverage", 10.0, false},
		{"Valid 1x leverage", 1.0, false},
		{"Valid max leverage 125x", 125.0, false},
		{"Invalid zero leverage", 0.0, true},
		{"Invalid negative leverage", -5.0, true},
		{"Invalid excessive leverage", 200.0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := calc.ValidateLeverage(tt.leverage)
			if (err != nil) != tt.shouldErr {
				t.Errorf("ValidateLeverage() error = %v, shouldErr %v", err, tt.shouldErr)
			}
		})
	}
}

func TestMaxPositionSize(t *testing.T) {
	calc := NewLeverageCalculator()
	
	tests := []struct {
		name              string
		availableMargin   float64
		leverage          float64
		expectedMaxSize   float64
	}{
		{
			name:            "$50 margin with 10x leverage",
			availableMargin: 50.0,
			leverage:        10.0,
			expectedMaxSize: 500.0,
		},
		{
			name:            "$100 margin with 1x leverage",
			availableMargin: 100.0,
			leverage:        1.0,
			expectedMaxSize: 100.0,
		},
		{
			name:            "$20 margin with 50x leverage",
			availableMargin: 20.0,
			leverage:        50.0,
			expectedMaxSize: 1000.0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxSize := calc.CalculateMaxPositionSize(tt.availableMargin, tt.leverage)
			if maxSize != tt.expectedMaxSize {
				t.Errorf("CalculateMaxPositionSize() = %.2f, want %.2f", maxSize, tt.expectedMaxSize)
			}
		})
	}
}

func TestLeverageHelper(t *testing.T) {
	helper := NewLeverageHelper(nil)
	
	t.Run("Margin percentage calculation", func(t *testing.T) {
		tests := []struct {
			leverage        float64
			expectedPercent float64
		}{
			{10.0, 0.10}, // 10x leverage = 10% margin
			{20.0, 0.05}, // 20x leverage = 5% margin
			{1.0, 1.00},  // 1x leverage = 100% margin
		}
		
		for _, tt := range tests {
			percent := helper.CalculateMarginPercent(tt.leverage)
			if percent != tt.expectedPercent {
				t.Errorf("CalculateMarginPercent(%.1f) = %.3f, want %.3f", 
					tt.leverage, percent, tt.expectedPercent)
			}
		}
	})
	
	t.Run("Liquidation price calculation", func(t *testing.T) {
		entryPrice := 100.0
		leverage := 10.0
		
		longLiqPrice := helper.CalculateLiquidationPrice(entryPrice, leverage, true)
		shortLiqPrice := helper.CalculateLiquidationPrice(entryPrice, leverage, false)
		
		// Long liquidation should be below entry price
		if longLiqPrice >= entryPrice {
			t.Errorf("Long liquidation price %.2f should be below entry price %.2f", 
				longLiqPrice, entryPrice)
		}
		
		// Short liquidation should be above entry price
		if shortLiqPrice <= entryPrice {
			t.Errorf("Short liquidation price %.2f should be above entry price %.2f", 
				shortLiqPrice, entryPrice)
		}
	})
}

func TestPositionSafety(t *testing.T) {
	helper := NewLeverageHelper(nil)
	
	t.Run("Safe position validation", func(t *testing.T) {
		safety := helper.ValidatePositionSafety(
			100.0, // position value
			50.0,  // entry price
			10.0,  // leverage
			20.0,  // available margin (enough for 10.0 required)
			true,  // long position
		)
		
		if !safety.IsValid {
			t.Error("Position should be valid")
		}
		
		if safety.RequiredMargin != 10.0 {
			t.Errorf("Required margin should be 10.0, got %.2f", safety.RequiredMargin)
		}
		
		if safety.RiskLevel == "CRITICAL" {
			t.Error("Risk level should not be CRITICAL for safe position")
		}
	})
	
	t.Run("Unsafe position validation", func(t *testing.T) {
		safety := helper.ValidatePositionSafety(
			100.0, // position value
			50.0,  // entry price
			10.0,  // leverage
			5.0,   // available margin (insufficient for 10.0 required)
			true,  // long position
		)
		
		if safety.IsValid {
			t.Error("Position should be invalid due to insufficient margin")
		}
		
		if len(safety.Errors) == 0 {
			t.Error("Should have error messages for insufficient margin")
		}
		
		if safety.RiskLevel != "CRITICAL" {
			t.Error("Risk level should be CRITICAL for invalid position")
		}
	})
}
