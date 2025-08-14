# DCA Backtesting System - Practical Examples

This document provides real-world examples of how to use the DCA Backtesting Suggestion System for different trading scenarios.

## Table of Contents

- [Quick Start Examples](#quick-start-examples)
- [Short-Term Trading Scenarios](#short-term-trading-scenarios)
- [Mid-Term Trading Scenarios](#mid-term-trading-scenarios)
- [Long-Term Investment Scenarios](#long-term-investment-scenarios)
- [Market Condition Adaptations](#market-condition-adaptations)
- [Risk-Based Recommendations](#risk-based-recommendations)
- [Integration Examples](#integration-examples)

## Quick Start Examples

### Running the System

```bash
# Clone and navigate to the project
git clone <repository-url>
cd enhanced-dca-bot

# Run the complete suggestion system
go run cmd/suggestion-system/main.go
```

### Getting Pre-configured Recommendations

```go
package main

import (
    "fmt"
)

func main() {
    // Get all pre-configured strategies
    configs := GetRecommendedBacktestPeriods()

    // Check a specific strategy
    scalping := configs["scalping_dca"]
    fmt.Printf("Strategy: %s\n", scalping.TradingStyle)
    fmt.Printf("Recommended Period: %d months\n", scalping.RecommendedPeriod)
    fmt.Printf("Risk Profile: %s\n", scalping.RiskProfile)
}
```

## Short-Term Trading Scenarios

### Scenario 1: Crypto Day Trader

**Background**: Active day trader focusing on major cryptocurrencies, making 2-5 trades per day.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        ShortTerm,
    TradingFrequency: "daily",
    PositionDuration: 1,           // Hold for ~1 day
    MarketCondition:  "mixed",     // Various market conditions
    RiskTolerance:    "high",      // Comfortable with risk
    VolatilityTarget: "high",      // Targets volatile moves
    WealthGoal:       "income",    // Daily income generation
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~4-6 months backtesting period
```

**Output Analysis**:

- **Recommended**: 4-6 months
- **Why**: Daily trading needs recent market patterns, but enough data to see various volatility regimes
- **Special Considerations**: Include weekend gaps, different trading sessions

### Scenario 2: Scalping Bot Developer

**Background**: Developing a high-frequency scalping bot for Bitcoin, executing trades every few minutes.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        ShortTerm,
    TradingFrequency: "minutes",
    PositionDuration: 0,           // Intraday only
    MarketCondition:  "mixed",
    RiskTolerance:    "very_high",
    VolatilityTarget: "extreme",
    WealthGoal:       "income",
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~3 months backtesting period
```

**Key Insights**:

- **Recommended**: 3 months
- **Why**: High-frequency strategies need very recent data, older patterns may not be relevant
- **Critical**: Must include transaction costs, slippage, and liquidity considerations

### Scenario 3: Swing Trading with DCA

**Background**: Swing trader using DCA principles, holding positions for 3-7 days during trending moves.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        ShortTerm,
    TradingFrequency: "daily",
    PositionDuration: 5,           // Average 5-day hold
    MarketCondition:  "bull",      // Trending markets
    RiskTolerance:    "medium",
    VolatilityTarget: "medium",
    WealthGoal:       "growth",
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~6-8 months backtesting period
```

## Mid-Term Trading Scenarios

### Scenario 4: Weekly DCA Strategy

**Background**: Professional trader implementing weekly DCA purchases with technical analysis for timing.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        MidTerm,
    TradingFrequency: "weekly",
    PositionDuration: 14,          // 2-week average hold
    MarketCondition:  "mixed",
    RiskTolerance:    "medium",
    VolatilityTarget: "medium",
    WealthGoal:       "growth",
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~18-24 months backtesting period
```

**Analysis**:

- **Recommended**: 18-24 months
- **Why**: Weekly strategies need multiple seasonal cycles and market phases
- **Include**: Holiday effects, quarter-end rebalancing, earnings seasons

### Scenario 5: Monthly Accumulation Strategy

**Background**: Institutional investor using monthly DCA with position sizing based on market conditions.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        MidTerm,
    TradingFrequency: "monthly",
    PositionDuration: 45,          // ~1.5 month hold
    MarketCondition:  "sideways",  // Range-bound markets
    RiskTolerance:    "low",
    VolatilityTarget: "low",
    WealthGoal:       "preservation",
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~30-36 months backtesting period
```

### Scenario 6: Trend Following DCA

**Background**: Quantitative fund using trend-following signals to adjust DCA frequency and amounts.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        MidTerm,
    TradingFrequency: "weekly",
    PositionDuration: 30,          // Monthly turnover
    MarketCondition:  "bull",      // Trend-following
    RiskTolerance:    "high",
    VolatilityTarget: "high",
    WealthGoal:       "growth",
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~24-36 months backtesting period
```

## Long-Term Investment Scenarios

### Scenario 7: Retirement Planning DCA

**Background**: Individual investor planning for retirement, making quarterly investments for 20+ years.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        LongTerm,
    TradingFrequency: "quarterly",
    PositionDuration: 365,         // Annual rebalancing
    MarketCondition:  "mixed",     // Long-term cycles
    RiskTolerance:    "low",
    VolatilityTarget: "low",
    WealthGoal:       "preservation",
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~84-120 months (7-10 years) backtesting period
```

**Long-term Considerations**:

- **Include**: Complete crypto adoption cycles
- **Test**: Technology regime changes
- **Consider**: Regulatory evolution impacts

### Scenario 8: Generational Wealth Building

**Background**: Family office implementing multi-generational crypto allocation strategy.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        LongTerm,
    TradingFrequency: "quarterly",
    PositionDuration: 730,         // Multi-year holds
    MarketCondition:  "mixed",
    RiskTolerance:    "very_low",
    VolatilityTarget: "low",
    WealthGoal:       "preservation",
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~120-180 months (10-15 years) backtesting period
```

### Scenario 9: Corporate Treasury DCA

**Background**: Public company implementing systematic crypto treasury accumulation.

```go
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        LongTerm,
    TradingFrequency: "monthly",
    PositionDuration: 365,         // Annual review
    MarketCondition:  "mixed",
    RiskTolerance:    "medium",
    VolatilityTarget: "medium",
    WealthGoal:       "growth",
}

recommendation := selector.CalculateOptimalPeriod()
// Expected: ~60-84 months (5-7 years) backtesting period
```

## Market Condition Adaptations

### Bull Market Strategy

```go
// During strong bull markets
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        MidTerm,
    TradingFrequency: "weekly",
    PositionDuration: 14,
    MarketCondition:  "bull",      // Key difference
    RiskTolerance:    "high",      // Can take more risk
    VolatilityTarget: "high",      // Capture momentum
    WealthGoal:       "growth",
}
```

**Bull Market Insights**:

- Shorter backtesting periods acceptable
- Focus on momentum strategies
- Test take-profit mechanisms thoroughly

### Bear Market Strategy

```go
// During bear markets or crashes
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        MidTerm,
    TradingFrequency: "weekly",
    PositionDuration: 21,
    MarketCondition:  "bear",      // Key difference
    RiskTolerance:    "low",       // Lower risk tolerance
    VolatilityTarget: "low",       // Avoid high volatility
    WealthGoal:       "preservation",
}
```

**Bear Market Insights**:

- Longer backtesting periods needed
- Must include stress-testing scenarios
- Focus on capital preservation

### Recovery Phase Strategy

```go
// During uncertain recovery periods
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        MidTerm,
    TradingFrequency: "weekly",
    PositionDuration: 14,
    MarketCondition:  "recovery",  // Key difference
    RiskTolerance:    "medium",
    VolatilityTarget: "high",      // Recovery can be volatile
    WealthGoal:       "growth",
}
```

## Risk-Based Recommendations

### Conservative Approach

```go
// Risk-averse investor
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        LongTerm,
    TradingFrequency: "monthly",
    PositionDuration: 90,
    MarketCondition:  "mixed",
    RiskTolerance:    "very_low",  // Very conservative
    VolatilityTarget: "very_low",  // Minimize volatility
    WealthGoal:       "preservation",
}

// Result: Longer backtesting period for thorough validation
```

### Aggressive Approach

```go
// High-risk, high-reward trader
selector := TimeframeSpecificBacktestSelector{
    Timeframe:        ShortTerm,
    TradingFrequency: "daily",
    PositionDuration: 3,
    MarketCondition:  "mixed",
    RiskTolerance:    "very_high", // Maximum risk tolerance
    VolatilityTarget: "extreme",   // Seek maximum volatility
    WealthGoal:       "speculation",
}

// Result: Shorter period acceptable due to high risk tolerance
```

## Integration Examples

### Backtesting Engine Integration

```go
package main

import (
    "fmt"
    "time"
)

func runBacktestWithRecommendation() {
    // Get recommendation
    selector := TimeframeSpecificBacktestSelector{
        Timeframe:        MidTerm,
        TradingFrequency: "weekly",
        PositionDuration: 14,
        MarketCondition:  "mixed",
        RiskTolerance:    "medium",
        VolatilityTarget: "medium",
        WealthGoal:       "growth",
    }

    rec := selector.CalculateOptimalPeriod()

    // Calculate date range
    endDate := time.Now()
    startDate := endDate.AddDate(0, -rec.RecommendedMonths, 0)

    fmt.Printf("Running backtest from %s to %s\n",
        startDate.Format("2006-01-02"),
        endDate.Format("2006-01-02"))

    // Configure backtesting engine
    config := BacktestConfig{
        StartDate:    startDate,
        EndDate:      endDate,
        Symbol:       "BTCUSDT",
        Strategy:     "weekly_dca",
        InitialCapital: 10000,
    }

    // Run backtest
    results := runBacktest(config)

    // Validate results with recommendation
    if results.TotalTrades < rec.MinimumMonths*4 {
        fmt.Println("Warning: Insufficient trades for reliable results")
    }
}
```

### Configuration File Generation

```go
func generateBacktestConfig(symbol string, strategy string) {
    // Determine appropriate selector based on strategy
    var selector TimeframeSpecificBacktestSelector

    switch strategy {
    case "scalping":
        selector = TimeframeSpecificBacktestSelector{
            Timeframe:        ShortTerm,
            TradingFrequency: "minutes",
            PositionDuration: 0,
            MarketCondition:  "mixed",
            RiskTolerance:    "very_high",
            VolatilityTarget: "extreme",
            WealthGoal:       "income",
        }
    case "swing":
        selector = TimeframeSpecificBacktestSelector{
            Timeframe:        MidTerm,
            TradingFrequency: "weekly",
            PositionDuration: 14,
            MarketCondition:  "mixed",
            RiskTolerance:    "medium",
            VolatilityTarget: "medium",
            WealthGoal:       "growth",
        }
    case "hodl":
        selector = TimeframeSpecificBacktestSelector{
            Timeframe:        LongTerm,
            TradingFrequency: "monthly",
            PositionDuration: 365,
            MarketCondition:  "mixed",
            RiskTolerance:    "low",
            VolatilityTarget: "low",
            WealthGoal:       "preservation",
        }
    }

    rec := selector.CalculateOptimalPeriod()

    // Generate configuration
    config := map[string]interface{}{
        "symbol":          symbol,
        "strategy":        strategy,
        "backtest_months": rec.RecommendedMonths,
        "min_months":      rec.MinimumMonths,
        "max_months":      rec.MaximumMonths,
        "reasoning":       rec.Reasoning,
        "special_notes":   rec.SpecialConsiderations,
        "generated_at":    time.Now().Format(time.RFC3339),
    }

    // Save to file
    saveJSONConfig(config, fmt.Sprintf("%s_%s_config.json", symbol, strategy))
}
```

### Market Regime Adaptation

```go
func adaptStrategyToMarket() {
    // Analyze current market
    analysis := AnalyzeCurrentMarketForBacktest()

    var marketCondition string
    switch analysis.CurrentPhase {
    case "Recovery/Early Bull":
        marketCondition = "recovery"
    case "Bull Run":
        marketCondition = "bull"
    case "Bear Market":
        marketCondition = "bear"
    default:
        marketCondition = "mixed"
    }

    // Create adaptive selector
    selector := TimeframeSpecificBacktestSelector{
        Timeframe:        MidTerm,
        TradingFrequency: "weekly",
        PositionDuration: 14,
        MarketCondition:  marketCondition, // Adaptive
        RiskTolerance:    "medium",
        VolatilityTarget: "medium",
        WealthGoal:       "growth",
    }

    rec := selector.CalculateOptimalPeriod()
    contextRec := analysis.GetContextualRecommendation()

    // Use the more conservative recommendation
    finalMonths := max(rec.RecommendedMonths, contextRec.RecommendedMonths)

    fmt.Printf("Adapted recommendation: %d months based on %s market\n",
        finalMonths, analysis.CurrentPhase)
}
```

## Validation and Testing

### Cross-Validation Example

```go
func validateStrategy() {
    selector := TimeframeSpecificBacktestSelector{
        Timeframe:        MidTerm,
        TradingFrequency: "weekly",
        PositionDuration: 14,
        MarketCondition:  "mixed",
        RiskTolerance:    "medium",
        VolatilityTarget: "medium",
        WealthGoal:       "growth",
    }

    rec := selector.CalculateOptimalPeriod()

    // Split recommended period for cross-validation
    trainPeriod := rec.RecommendedMonths * 2 / 3
    testPeriod := rec.RecommendedMonths / 3

    // Train on earlier data
    trainEndDate := time.Now().AddDate(0, -testPeriod, 0)
    trainStartDate := trainEndDate.AddDate(0, -trainPeriod, 0)

    // Test on more recent data
    testEndDate := time.Now()
    testStartDate := testEndDate.AddDate(0, -testPeriod, 0)

    fmt.Printf("Training period: %s to %s (%d months)\n",
        trainStartDate.Format("2006-01"), trainEndDate.Format("2006-01"), trainPeriod)
    fmt.Printf("Testing period: %s to %s (%d months)\n",
        testStartDate.Format("2006-01"), testEndDate.Format("2006-01"), testPeriod)
}
```

## Common Patterns and Best Practices

### Pattern 1: Progressive Testing

```go
// Start with minimum period, gradually increase
func progressiveTest(strategy string) {
    selector := getStrategySelector(strategy)
    rec := selector.CalculateOptimalPeriod()

    periods := []int{
        rec.MinimumMonths,
        rec.RecommendedMonths,
        rec.MaximumMonths,
    }

    for _, period := range periods {
        fmt.Printf("Testing with %d months of data...\n", period)
        results := runBacktestWithPeriod(strategy, period)

        if results.IsStatisticallySignificant() {
            fmt.Printf("Strategy validated with %d months\n", period)
            break
        }
    }
}
```

### Pattern 2: Multi-Scenario Testing

```go
// Test strategy across different market conditions
func multiScenarioTest(strategy string) {
    baseSelector := getStrategySelector(strategy)

    scenarios := []string{"bull", "bear", "sideways", "mixed", "recovery"}

    for _, scenario := range scenarios {
        testSelector := baseSelector
        testSelector.MarketCondition = scenario

        rec := testSelector.CalculateOptimalPeriod()
        fmt.Printf("Scenario %s: %d months recommended\n",
            scenario, rec.RecommendedMonths)
    }
}
```

These examples demonstrate the flexibility and power of the DCA Backtesting Suggestion System for various trading scenarios and use cases. The system adapts to different market conditions, risk profiles, and investment goals to provide tailored backtesting recommendations.

---

_For additional technical details, see the API Reference documentation._
