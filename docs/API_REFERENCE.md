# DCA Backtesting System - API Reference

## Overview

The DCA Backtesting Suggestion System provides a Go-based API for determining optimal backtesting periods for cryptocurrency Dollar Cost Averaging strategies. This document covers the technical implementation details and API usage.

## Core Types

### TimeframeCategory

Defines the three main trading timeframes supported by the system.

```go
type TimeframeCategory string

const (
    ShortTerm TimeframeCategory = "short_term"  // Minutes to days
    MidTerm   TimeframeCategory = "mid_term"    // Days to weeks
    LongTerm  TimeframeCategory = "long_term"   // Months to years
)
```

### TimeframeBoundaries

Contains the definition and characteristics of each timeframe.

```go
type TimeframeBoundaries struct {
    Category        TimeframeCategory
    PositionDuration string    // Human-readable duration range
    TypicalHoldTime  string    // Specific hold time range
    BacktestRange    string    // Recommended backtest period range
    Description      string    // Strategy description
    KeyFocus        []string  // Key focus areas for this timeframe
}
```

### DCABacktestConfig

Pre-configured backtesting recommendations for common DCA strategies.

```go
type DCABacktestConfig struct {
    Timeframe        TimeframeCategory
    TradingStyle     string   // Human-readable strategy name
    RecommendedPeriod int     // Months
    MinimumPeriod    int     // Months
    OptimalPeriod    int     // Months
    Reasoning        string  // Explanation of the recommendation
    MarketConditions []string // Suitable market conditions
    RiskProfile      string  // Risk level description
}
```

### TimeframeSpecificBacktestSelector

Main configuration struct for generating custom backtesting recommendations.

```go
type TimeframeSpecificBacktestSelector struct {
    Timeframe         TimeframeCategory // Required
    TradingFrequency  string           // "minutes", "hours", "daily", "weekly", "monthly"
    PositionDuration  int              // Average days to hold
    MarketCondition   string           // "bull", "bear", "sideways", "mixed", "recovery"
    RiskTolerance     string           // "very_low", "low", "medium", "high", "very_high"
    VolatilityTarget  string           // "very_low", "low", "medium", "high", "extreme"
    WealthGoal        string           // "income", "growth", "preservation", "speculation"
}
```

### TimeframeBacktestRecommendation

Output struct containing calculated backtesting recommendations.

```go
type TimeframeBacktestRecommendation struct {
    Timeframe              TimeframeCategory
    RecommendedMonths      int      // Primary recommendation
    MinimumMonths         int      // Lower bound
    MaximumMonths         int      // Upper bound
    Reasoning             string   // Explanation of calculation
    SpecialConsiderations []string // Important notes
    TimeframeSpecificAdvice []string // Timeframe-specific guidance
    DataRequirements      []string // Data quality requirements
    RiskWarnings         []string // Risk-related warnings
}
```

### CryptoMarketCycleInfo

Information about crypto market cycles and their backtesting implications.

```go
type CryptoMarketCycleInfo struct {
    CycleType            string
    TypicalDuration      string
    ShortTermBacktest    int // Months for short-term strategies
    MidTermBacktest      int // Months for mid-term strategies
    LongTermBacktest     int // Months for long-term strategies
    Description          string
    TimeframeImpact      map[TimeframeCategory]string
    KeyCharacteristics   []string
}
```

### MarketRegimeAnalysis

Market analysis for contextual backtesting recommendations.

```go
type MarketRegimeAnalysis struct {
    CurrentDate      time.Time
    LastMajorCrash   time.Time
    LastBullPeak     time.Time
    CurrentPhase     string  // Market phase description
    VolatilityLevel  string  // Current volatility assessment
}
```

## Core Functions

### GetTimeframeBoundaries()

Returns the definitions for all supported timeframes.

```go
func GetTimeframeBoundaries() map[TimeframeCategory]TimeframeBoundaries
```

**Returns**: Map of timeframe categories to their boundary definitions.

**Usage**:
```go
boundaries := GetTimeframeBoundaries()
shortTerm := boundaries[ShortTerm]
fmt.Println(shortTerm.Description)
```

### GetRecommendedBacktestPeriods()

Returns pre-configured recommendations for common DCA strategies.

```go
func GetRecommendedBacktestPeriods() map[string]DCABacktestConfig
```

**Returns**: Map of strategy names to their configurations.

**Available Strategies**:
- `"scalping_dca"`
- `"short_swing_dca"`
- `"aggressive_daily_dca"`
- `"weekly_swing_dca"`
- `"trend_following_dca"`
- `"monthly_dca"`
- `"quarterly_dca"`
- `"annual_dca"`
- `"hodl_dca"`

**Usage**:
```go
configs := GetRecommendedBacktestPeriods()
scalpingConfig := configs["scalping_dca"]
fmt.Printf("Recommended period: %d months\n", scalpingConfig.RecommendedPeriod)
```

### GetCryptoMarketCycles()

Returns information about crypto market cycles and their backtesting implications.

```go
func GetCryptoMarketCycles() []CryptoMarketCycleInfo
```

**Returns**: Slice of market cycle information.

**Available Cycles**:
- Bear Market Bottom
- Bull Market Rise  
- Bull Market Peak
- Bear Market Crash
- Recovery Phase

**Usage**:
```go
cycles := GetCryptoMarketCycles()
for _, cycle := range cycles {
    fmt.Printf("%s: %d months for short-term\n", 
        cycle.CycleType, cycle.ShortTermBacktest)
}
```

### CalculateOptimalPeriod()

Core calculation method for generating custom backtesting recommendations.

```go
func (s *TimeframeSpecificBacktestSelector) CalculateOptimalPeriod() TimeframeBacktestRecommendation
```

**Returns**: Detailed backtesting recommendation based on input parameters.

**Usage**:
```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        ShortTerm,
    TradingFrequency: "daily",
    PositionDuration: 3,
    MarketCondition:  "mixed",
    RiskTolerance:    "medium",
    VolatilityTarget: "medium",
    WealthGoal:       "growth",
}

recommendation := selector.CalculateOptimalPeriod()
fmt.Printf("Recommended: %d months\n", recommendation.RecommendedMonths)
fmt.Printf("Range: %d-%d months\n", recommendation.MinimumMonths, recommendation.MaximumMonths)
```

### AnalyzeCurrentMarketForBacktest()

Analyzes current market conditions for contextual recommendations.

```go
func AnalyzeCurrentMarketForBacktest() MarketRegimeAnalysis
```

**Returns**: Current market analysis with contextual backtesting suggestions.

**Usage**:
```go
analysis := AnalyzeCurrentMarketForBacktest()
recommendation := analysis.GetContextualRecommendation()
fmt.Printf("Current market phase: %s\n", analysis.CurrentPhase)
fmt.Printf("Suggested period: %d months\n", recommendation.RecommendedMonths)
```

## Calculation Algorithm

The system uses a multi-factor approach to calculate optimal backtesting periods:

### Base Period Calculation
```go
func (s *TimeframeSpecificBacktestSelector) getBasePeriodFromTimeframeAndFrequency() int
```

Determines base period based on timeframe and trading frequency:
- **Short-term**: 6 months base, adjusted by frequency
- **Mid-term**: 18 months base, adjusted by frequency  
- **Long-term**: 60 months base, adjusted by frequency

### Multiplier Adjustments

The system applies several multipliers to the base period:

1. **Duration Multiplier** (0.7x - 2.0x)
   - Based on typical position holding duration
   - Shorter holds = less data needed

2. **Market Condition Multiplier** (0.9x - 1.6x)  
   - Mixed markets require more data
   - Single-direction trends need less data

3. **Volatility Multiplier** (0.8x - 1.5x)
   - High volatility requires more testing scenarios
   - Low volatility needs less data

4. **Wealth Goal Multiplier** (0.8x - 1.4x)
   - Preservation strategies need extensive testing
   - Speculation can use shorter periods

### Final Calculation
```
finalPeriod = basePeriod × durationMultiplier × marketMultiplier × volatilityMultiplier × wealthGoalMultiplier
```

### Bounds Enforcement

The system enforces timeframe-appropriate bounds:
- **Short-term**: 1-24 months
- **Mid-term**: 6-60 months
- **Long-term**: 24-180 months

## Display Functions

### GetComprehensiveRecommendations()

Displays comprehensive recommendations for all timeframes with example scenarios.

```go
func GetComprehensiveRecommendations()
```

### ShowMarketCycleAnalysis()

Displays detailed market cycle analysis and backtesting implications.

```go
func ShowMarketCycleAnalysis()
```

### ShowQuickReferenceTable()

Displays a quick reference table for common strategies.

```go
func ShowQuickReferenceTable()
```

### ShowTimeframeSummary()

Shows summary information for each timeframe with key insights.

```go
func ShowTimeframeSummary()
```

## Usage Patterns

### Basic Recommendation Lookup

```go
// Get pre-configured recommendation
configs := GetRecommendedBacktestPeriods()
config := configs["weekly_swing_dca"]
fmt.Printf("Strategy: %s\n", config.TradingStyle)
fmt.Printf("Recommended: %d months\n", config.RecommendedPeriod)
```

### Custom Strategy Analysis

```go
// Create custom selector
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        MidTerm,
    TradingFrequency: "weekly",
    PositionDuration: 14,
    MarketCondition:  "bull",
    RiskTolerance:    "medium",
    VolatilityTarget: "high",
    WealthGoal:       "growth",
}

// Get recommendation
rec := selector.CalculateOptimalPeriod()

// Use the recommendation
fmt.Printf("Test your strategy with %d months of data\n", rec.RecommendedMonths)
fmt.Printf("Minimum viable: %d months\n", rec.MinimumMonths)
fmt.Printf("Maximum useful: %d months\n", rec.MaximumMonths)
```

### Market Context Integration

```go
// Analyze current market
analysis := AnalyzeCurrentMarketForBacktest()
contextRec := analysis.GetContextualRecommendation()

// Combine with strategy-specific analysis
selector := TimeframeSpecificBacktestSelector{
    Timeframe: ShortTerm,
    // ... other parameters
}
strategyRec := selector.CalculateOptimalPeriod()

// Use both recommendations to make informed decision
finalPeriod := max(contextRec.RecommendedMonths, strategyRec.RecommendedMonths)
```

## Integration Examples

### Integration with Backtesting Engine

```go
// Get recommendation
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        ShortTerm,
    TradingFrequency: "daily",
    // ... other config
}
rec := selector.CalculateOptimalPeriod()

// Use recommendation in backtesting engine
startDate := time.Now().AddDate(0, -rec.RecommendedMonths, 0)
endDate := time.Now()

// Run backtest with recommended period
results := runBacktest(strategy, startDate, endDate)
```

### Configuration File Generation

```go
// Generate configuration for backtesting system
rec := selector.CalculateOptimalPeriod()

config := BacktestConfig{
    StartDate: time.Now().AddDate(0, -rec.RecommendedMonths, 0),
    EndDate:   time.Now(),
    Strategy:  "daily_dca",
    Symbol:    "BTCUSDT",
    // Use recommendations for other parameters
}

saveConfig(config, "backtest_config.json")
```

## Error Handling

The system is designed to be robust with sensible defaults:

- Invalid timeframes default to `MidTerm`
- Unknown trading frequencies use base periods
- Out-of-range values are clamped to acceptable bounds
- Missing parameters use conservative defaults

## Performance Considerations

- All calculations are performed in-memory with O(1) complexity
- Pre-computed lookup tables for common scenarios
- Minimal external dependencies
- Suitable for high-frequency recommendation generation

## Thread Safety

The system is read-only after initialization and is thread-safe for concurrent access to recommendation functions.

---

*For practical usage examples, see the main documentation and README files.*
