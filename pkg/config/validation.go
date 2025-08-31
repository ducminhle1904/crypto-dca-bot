package config

import (
	"fmt"
)

// DCAValidator implements validation for DCA configurations
type DCAValidator struct{}

// NewDCAValidator creates a new DCA validator
func NewDCAValidator() *DCAValidator {
	return &DCAValidator{}
}

// Validate performs comprehensive validation on DCA configuration parameters
func (v *DCAValidator) Validate(cfg Config) error {
	dcaCfg, ok := cfg.(*DCAConfig)
	if !ok {
		return fmt.Errorf("expected *DCAConfig, got %T", cfg)
	}
	
	return v.validateDCAConfig(dcaCfg)
}

// validateDCAConfig performs basic validation on configuration parameters
func (v *DCAValidator) validateDCAConfig(cfg *DCAConfig) error {
	if cfg.InitialBalance <= 0 {
		return fmt.Errorf("initial balance must be positive, got: %.2f", cfg.InitialBalance)
	}
	
	if cfg.Commission < 0 || cfg.Commission > MaxCommission {
		return fmt.Errorf("commission must be between 0 and %.2f (0-100%%), got: %.4f", MaxCommission, cfg.Commission)
	}
	
	if cfg.BaseAmount <= 0 {
		return fmt.Errorf("base amount must be positive, got: %.2f", cfg.BaseAmount)
	}
	
	if cfg.MaxMultiplier <= MinMultiplier {
		return fmt.Errorf("max multiplier must be greater than %.1f, got: %.2f", MinMultiplier, cfg.MaxMultiplier)
	}
	
	if cfg.WindowSize <= 0 {
		return fmt.Errorf("window size must be positive, got: %d", cfg.WindowSize)
	}
	
	if cfg.PriceThreshold < 0 || cfg.PriceThreshold > MaxThreshold {
		return fmt.Errorf("price threshold must be between 0 and %.2f (0-100%%), got: %.4f", MaxThreshold, cfg.PriceThreshold)
	}
	
	if cfg.TPPercent < 0 || cfg.TPPercent > MaxThreshold {
		return fmt.Errorf("TP percent must be between 0 and %.2f (0-100%%), got: %.4f", MaxThreshold, cfg.TPPercent)
	}
	
	// Validate TP configuration
	if err := v.validateTPConfig(cfg); err != nil {
		return err
	}
	
	// Validate technical indicator parameters
	if err := v.validateClassicIndicators(cfg); err != nil {
		return err
	}
	
	// Validate advanced combo indicator parameters
	if cfg.UseAdvancedCombo {
		if err := v.validateAdvancedIndicators(cfg); err != nil {
			return err
		}
	}
	
	if cfg.MinOrderQty < 0 {
		return fmt.Errorf("minimum order quantity must be non-negative, got: %.6f", cfg.MinOrderQty)
	}
	
	return nil
}

// validateTPConfig validates the TP configuration (always multi-level TP)
func (v *DCAValidator) validateTPConfig(cfg *DCAConfig) error {
    if cfg.TPPercent <= 0 || cfg.TPPercent > MaxThreshold {
        return fmt.Errorf("tp_percent must be within (0, %.2f], got %.4f", MaxThreshold, cfg.TPPercent)
    }
    return nil
}

// validateClassicIndicators validates classic combo technical indicator parameters
func (v *DCAValidator) validateClassicIndicators(cfg *DCAConfig) error {
	if cfg.RSIPeriod < MinRSIPeriod {
		return fmt.Errorf("RSI period must be at least %d, got: %d", MinRSIPeriod, cfg.RSIPeriod)
	}
	
	if cfg.RSIOversold <= 0 || cfg.RSIOversold >= MaxRSIValue {
		return fmt.Errorf("RSI oversold must be between 0 and %d, got: %.1f", MaxRSIValue, cfg.RSIOversold)
	}
	
	if cfg.RSIOverbought <= 0 || cfg.RSIOverbought >= MaxRSIValue {
		return fmt.Errorf("RSI overbought must be between 0 and %d, got: %.1f", MaxRSIValue, cfg.RSIOverbought)
	}
	
	if cfg.RSIOversold >= cfg.RSIOverbought {
		return fmt.Errorf("RSI oversold (%.1f) must be less than overbought (%.1f)", cfg.RSIOversold, cfg.RSIOverbought)
	}
	
	if cfg.MACDFast < MinMACDPeriod || cfg.MACDSlow < MinMACDPeriod || cfg.MACDSignal < MinMACDPeriod {
		return fmt.Errorf("MACD periods must be at least %d, got: fast=%d, slow=%d, signal=%d", MinMACDPeriod, cfg.MACDFast, cfg.MACDSlow, cfg.MACDSignal)
	}
	
	if cfg.MACDFast >= cfg.MACDSlow {
		return fmt.Errorf("MACD fast period (%d) must be less than slow period (%d)", cfg.MACDFast, cfg.MACDSlow)
	}
	
	if cfg.BBPeriod < MinBBPeriod {
		return fmt.Errorf("bollinger Bands period must be at least %d, got: %d", MinBBPeriod, cfg.BBPeriod)
	}
	
	if cfg.BBStdDev <= 0 {
		return fmt.Errorf("bollinger Bands standard deviation must be positive, got: %.2f", cfg.BBStdDev)
	}
	
	if cfg.EMAPeriod < MinEMAPeriod {
		return fmt.Errorf("EMA period must be at least %d, got: %d", MinEMAPeriod, cfg.EMAPeriod)
	}
	
	return nil
}

// validateAdvancedIndicators validates advanced combo indicator parameters
func (v *DCAValidator) validateAdvancedIndicators(cfg *DCAConfig) error {
	if cfg.HullMAPeriod < MinHullMAPeriod {
		return fmt.Errorf("Hull MA period must be at least %d, got: %d", MinHullMAPeriod, cfg.HullMAPeriod)
	}
	
	if cfg.MFIPeriod < MinMFIPeriod {
		return fmt.Errorf("MFI period must be at least %d, got: %d", MinMFIPeriod, cfg.MFIPeriod)
	}
	
	if cfg.MFIOversold <= 0 || cfg.MFIOversold >= MaxRSIValue {
		return fmt.Errorf("MFI oversold must be between 0 and %d, got: %.1f", MaxRSIValue, cfg.MFIOversold)
	}
	
	if cfg.MFIOverbought <= 0 || cfg.MFIOverbought >= MaxRSIValue {
		return fmt.Errorf("MFI overbought must be between 0 and %d, got: %.1f", MaxRSIValue, cfg.MFIOverbought)
	}
	
	if cfg.MFIOversold >= cfg.MFIOverbought {
		return fmt.Errorf("MFI oversold (%.1f) must be less than overbought (%.1f)", cfg.MFIOversold, cfg.MFIOverbought)
	}
	
	if cfg.KeltnerPeriod < MinKeltnerPeriod {
		return fmt.Errorf("Keltner period must be at least %d, got: %d", MinKeltnerPeriod, cfg.KeltnerPeriod)
	}
	
	if cfg.KeltnerMultiplier <= 0 {
		return fmt.Errorf("Keltner multiplier must be positive, got: %.2f", cfg.KeltnerMultiplier)
	}
	
	if cfg.WaveTrendN1 < MinWaveTrendPeriod {
		return fmt.Errorf("WaveTrend N1 must be at least %d, got: %d", MinWaveTrendPeriod, cfg.WaveTrendN1)
	}
	
	if cfg.WaveTrendN2 < MinWaveTrendPeriod {
		return fmt.Errorf("WaveTrend N2 must be at least %d, got: %d", MinWaveTrendPeriod, cfg.WaveTrendN2)
	}
	
	if cfg.WaveTrendN1 >= cfg.WaveTrendN2 {
		return fmt.Errorf("WaveTrend N1 (%d) must be less than N2 (%d)", cfg.WaveTrendN1, cfg.WaveTrendN2)
	}
	
	if cfg.WaveTrendOversold >= cfg.WaveTrendOverbought {
		return fmt.Errorf("WaveTrend oversold (%.1f) must be less than overbought (%.1f)", cfg.WaveTrendOversold, cfg.WaveTrendOverbought)
	}
	
	return nil
}

// Implement the DCAConfig Validate method
func (cfg *DCAConfig) Validate() error {
	validator := NewDCAValidator()
	return validator.Validate(cfg)
}
