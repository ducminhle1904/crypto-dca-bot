package main

import (
	"fmt"
	"strings"
	"time"
)

// TimeframeCategory defines the types of trading timeframes
type TimeframeCategory string

const (
	ShortTerm TimeframeCategory = "short_term"
	MidTerm   TimeframeCategory = "mid_term"
	LongTerm  TimeframeCategory = "long_term"
)

// TimeframeBoundaries defines the boundaries for each timeframe
type TimeframeBoundaries struct {
	Category        TimeframeCategory
	PositionDuration string
	TypicalHoldTime  string
	BacktestRange    string
	Description      string
	KeyFocus        []string
}

// GetTimeframeBoundaries returns clear definitions for each timeframe
func GetTimeframeBoundaries() map[TimeframeCategory]TimeframeBoundaries {
	return map[TimeframeCategory]TimeframeBoundaries{
		ShortTerm: {
			Category:        ShortTerm,
			PositionDuration: "Minutes to Days",
			TypicalHoldTime:  "1 minute - 7 days",
			BacktestRange:    "3-12 months",
			Description:     "Focus on quick entries/exits, market noise, high-frequency patterns",
			KeyFocus: []string{
				"Market microstructure",
				"Intraday volatility patterns",
				"News reaction speed",
				"Liquidity dynamics",
				"High-frequency noise filtering",
			},
		},
		MidTerm: {
			Category:        MidTerm,
			PositionDuration: "Days to Weeks",
			TypicalHoldTime:  "1 week - 3 months",
			BacktestRange:    "12-36 months",
			Description:     "Balance between trend-following and mean reversion, multiple market cycles",
			KeyFocus: []string{
				"Trend identification",
				"Support/resistance levels",
				"Seasonal patterns",
				"Economic cycles",
				"Multiple timeframe confluence",
			},
		},
		LongTerm: {
			Category:        LongTerm,
			PositionDuration: "Months to Years",
			TypicalHoldTime:  "3 months - 2+ years",
			BacktestRange:    "2-10 years",
			Description:     "Focus on fundamental trends, major market cycles, wealth building",
			KeyFocus: []string{
				"Market cycle analysis",
				"Fundamental trends",
				"Macro economic factors",
				"Technology adoption cycles",
				"Long-term value accumulation",
			},
		},
	}
}

// DCABacktestConfig configuration for DCA backtesting by timeframe
type DCABacktestConfig struct {
	Timeframe        TimeframeCategory
	TradingStyle     string
	RecommendedPeriod int    // Months
	MinimumPeriod    int    // Months
	OptimalPeriod    int    // Months
	Reasoning        string
	MarketConditions []string
	RiskProfile      string
}

// GetRecommendedBacktestPeriods returns recommendations for different trading styles
func GetRecommendedBacktestPeriods() map[string]DCABacktestConfig {
	return map[string]DCABacktestConfig{
		"ultra_short_scalping": {
			TradingStyle:     "Ultra Short-term (1-3 days)",
			RecommendedPeriod: 2,
			MinimumPeriod:    1,
			OptimalPeriod:    3,
			Reasoning: "Focus on volatility patterns, need most recent data to capture market microstructure",
			MarketConditions: []string{"High volatility periods", "Recent market regime"},
		},
		"short_swing": {
			TradingStyle:     "Short Swing (3-14 days)",
			RecommendedPeriod: 6,
			MinimumPeriod:    3,
			OptimalPeriod:    9,
			Reasoning: "Need enough cycles to test entry/exit timing, including both trending and sideways periods",
			MarketConditions: []string{"Mix of trends", "Multiple volatility regimes", "Different market hours"},
		},
		"medium_swing": {
			TradingStyle:     "Medium Swing (2-8 weeks)",
			RecommendedPeriod: 12,
			MinimumPeriod:    6,
			OptimalPeriod:    18,
			Reasoning: "Cover multiple market cycles, seasonal effects, and major news events",
			MarketConditions: []string{"Full market cycle", "Major events", "Seasonal patterns"},
		},
		"aggressive_dca": {
			TradingStyle:     "Aggressive DCA (Daily entries)",
			RecommendedPeriod: 4,
			MinimumPeriod:    2,
			OptimalPeriod:    6,
			Reasoning: "Daily DCA needs to test reaction to market noise and short-term volatility",
			MarketConditions: []string{"Daily volatility patterns", "Intraday trends"},
		},
	}
}

// CryptoMarketCycleInfo information about crypto market cycles for different timeframes
type CryptoMarketCycleInfo struct {
	CycleType            string
	TypicalDuration      string
	ShortTermBacktest    int // Months for short-term
	MidTermBacktest      int // Months for mid-term  
	LongTermBacktest     int // Months for long-term
	Description          string
	TimeframeImpact      map[TimeframeCategory]string
	KeyCharacteristics   []string
}

func GetCryptoMarketCycles() []CryptoMarketCycleInfo {
	return []CryptoMarketCycleInfo{
		{
			CycleType:         "Bear Market Bottom",
			TypicalDuration:   "6-18 months",
			ShortTermBacktest: 6,
			MidTermBacktest:   12,
			LongTermBacktest:  24,
			Description:       "Accumulation phase, low volatility, DCA most effective",
			TimeframeImpact: map[TimeframeCategory]string{
				ShortTerm: "Perfect for scalping DCA, low noise, predictable patterns",
				MidTerm:   "Excellent for building positions, clear support levels",
				LongTerm:  "Optimal accumulation phase, generational buying opportunity",
			},
			KeyCharacteristics: []string{
				"Low volatility",
				"Sideways price action", 
				"Good for aggressive DCA",
				"Long accumulation periods",
				"Strong support levels",
			},
		},
		{
			CycleType:         "Bull Market Rise",
			TypicalDuration:   "12-24 months",
			ShortTermBacktest: 8,
			MidTermBacktest:   18,
			LongTermBacktest:  36,
			Description:       "Trending upward, many pullbacks, need to test take-profit strategy",
			TimeframeImpact: map[TimeframeCategory]string{
				ShortTerm: "High volatility creates scalping opportunities but increases risk",
				MidTerm:   "Excellent for trend-following, clear directional bias",
				LongTerm:  "Strong wealth building phase, regular take-profits important",
			},
			KeyCharacteristics: []string{
				"Strong uptrends",
				"Frequent pullbacks",
				"High volatility",
				"Take-profit optimization critical",
				"FOMO-driven moves",
			},
		},
		{
			CycleType:         "Bull Market Peak",
			TypicalDuration:   "2-6 months",
			ShortTermBacktest: 4,
			MidTermBacktest:   8,
			LongTermBacktest:  12,
			Description:       "Extremely volatile, need to test risk management",
			TimeframeImpact: map[TimeframeCategory]string{
				ShortTerm: "Extreme volatility - high profit potential but very risky",
				MidTerm:   "Difficult trending, frequent whipsaws, tight risk management",
				LongTerm:  "Distribution phase, consider reducing positions",
			},
			KeyCharacteristics: []string{
				"Extreme volatility",
				"Quick reversals",
				"FOMO periods",
				"Risk management crucial",
				"Media attention peaks",
			},
		},
		{
			CycleType:         "Bear Market Crash",
			TypicalDuration:   "3-12 months",
			ShortTermBacktest: 6,
			MidTermBacktest:   12,
			LongTermBacktest:  24,
			Description:       "Strong downtrend, test strategy's survival ability",
			TimeframeImpact: map[TimeframeCategory]string{
				ShortTerm: "Very dangerous for scalping, high drawdowns possible",
				MidTerm:   "Trend-following works but requires strong risk management",
				LongTerm:  "Ultimate stress test, validates long-term strategy robustness",
			},
			KeyCharacteristics: []string{
				"Strong downtrends",
				"Panic selling",
				"DCA averaging down",
				"Strategy stress test",
				"Liquidation cascades",
			},
		},
		{
			CycleType:         "Recovery Phase",
			TypicalDuration:   "4-10 months",
			ShortTermBacktest: 6,
			MidTermBacktest:   15,
			LongTermBacktest:  30,
			Description:       "Transition from bear to bull, high uncertainty but with opportunities",
			TimeframeImpact: map[TimeframeCategory]string{
				ShortTerm: "Mixed signals, good for range trading strategies",
				MidTerm:   "Early trend identification crucial, breakout opportunities",
				LongTerm:  "Re-accumulation phase, gradual position building",
			},
			KeyCharacteristics: []string{
				"Mixed signals",
				"False breakouts",
				"Uncertainty periods",
				"Early trend formation",
				"Volume analysis important",
			},
		},
	}
}

// TimeframeSpecificBacktestSelector allows selecting period based on timeframe and factors
type TimeframeSpecificBacktestSelector struct {
	Timeframe         TimeframeCategory // short_term, mid_term, long_term
	TradingFrequency  string           // "minutes", "hours", "daily", "weekly", "monthly"
	PositionDuration  int              // Average days to hold
	MarketCondition   string           // "bull", "bear", "sideways", "mixed", "recovery"
	RiskTolerance     string           // "very_low", "low", "medium", "high", "very_high"
	VolatilityTarget  string           // "very_low", "low", "medium", "high", "extreme"
	WealthGoal        string           // "income", "growth", "preservation", "speculation"
}

func (s *TimeframeSpecificBacktestSelector) CalculateOptimalPeriod() TimeframeBacktestRecommendation {
	recommendation := TimeframeBacktestRecommendation{}
	
	// Base period based on timeframe and trading frequency
	basePeriod := s.getBasePeriodFromTimeframeAndFrequency()
	
	// Adjust according to position duration
	durationMultiplier := s.getDurationMultiplier()
	
	// Adjust according to market condition
	marketMultiplier := s.getMarketConditionMultiplier()
	
	// Adjust according to volatility
	volatilityMultiplier := s.getVolatilityMultiplier()
	
	// Adjust according to wealth goal
	wealthGoalMultiplier := s.getWealthGoalMultiplier()
	
	// Calculate final period
	finalPeriod := float64(basePeriod) * durationMultiplier * marketMultiplier * volatilityMultiplier * wealthGoalMultiplier
	
	recommendation.Timeframe = s.Timeframe
	recommendation.RecommendedMonths = int(finalPeriod)
	recommendation.MinimumMonths = int(finalPeriod * 0.6)
	recommendation.MaximumMonths = int(finalPeriod * 1.8)
	
	// Ensure timeframe-appropriate bounds
	recommendation = s.ensureTimeframeBounds(recommendation)
	
	recommendation.Reasoning = s.generateReasoning()
	recommendation.SpecialConsiderations = s.getSpecialConsiderations()
	recommendation.TimeframeSpecificAdvice = s.getTimeframeSpecificAdvice()
	
	return recommendation
}

type TimeframeBacktestRecommendation struct {
	Timeframe              TimeframeCategory
	RecommendedMonths      int
	MinimumMonths         int
	MaximumMonths         int
	Reasoning             string
	SpecialConsiderations []string
	TimeframeSpecificAdvice []string
	DataRequirements      []string
	RiskWarnings         []string
}

// Legacy type for backward compatibility  
type BacktestPeriodRecommendation = TimeframeBacktestRecommendation

func (s *TimeframeSpecificBacktestSelector) getBasePeriodFromTimeframeAndFrequency() int {
	// Base periods by timeframe
	basePeriods := map[TimeframeCategory]int{
		ShortTerm: 6,   // 6 months base for short-term
		MidTerm:   18,  // 18 months base for mid-term
		LongTerm:  60,  // 60 months (5 years) base for long-term
	}
	
	base := basePeriods[s.Timeframe]
	
	// Adjust according to frequency within timeframe
	switch s.Timeframe {
	case ShortTerm:
		switch s.TradingFrequency {
		case "minutes":
			return base - 3  // 3 months for scalping
		case "hours":
			return base - 2  // 4 months for hourly
		case "daily":
			return base      // 6 months for daily
		default:
			return base
		}
	case MidTerm:
		switch s.TradingFrequency {
		case "daily":
			return base - 6   // 12 months
		case "weekly":
			return base       // 18 months
		case "monthly":
			return base + 12  // 30 months
		default:
			return base
		}
	case LongTerm:
		switch s.TradingFrequency {
		case "weekly":
			return base - 24  // 36 months (3 years)
		case "monthly":
			return base       // 60 months (5 years)
		case "quarterly":
			return base + 24  // 84 months (7 years)
		default:
			return base
		}
	}
	
	return base
}

func (s *TimeframeSpecificBacktestSelector) getDurationMultiplier() float64 {
	// Adjust according to timeframe and position duration
	switch s.Timeframe {
	case ShortTerm:
		if s.PositionDuration <= 1 {
			return 0.7  // Intraday scalping
		} else if s.PositionDuration <= 7 {
			return 1.0  // Short swing
		} else {
			return 1.3  // Longer short-term holds
		}
	case MidTerm:
		if s.PositionDuration <= 7 {
			return 0.8  // Weekly trades
		} else if s.PositionDuration <= 30 {
			return 1.0  // Monthly trades
		} else {
			return 1.4  // Quarterly trades
		}
	case LongTerm:
		if s.PositionDuration <= 90 {
			return 0.7  // 3-month holds
		} else if s.PositionDuration <= 365 {
			return 1.0  // Annual holds
		} else {
			return 1.5  // Multi-year holds
		}
	}
	return 1.0
}

func (s *TimeframeSpecificBacktestSelector) getMarketConditionMultiplier() float64 {
	// Market condition impact varies by timeframe
	switch s.Timeframe {
	case ShortTerm:
		switch s.MarketCondition {
		case "bull", "bear":
			return 1.1  // Trending markets for short-term
		case "sideways":
			return 0.9  // Range-bound good for scalping
		case "mixed":
			return 1.4  // Need more data for mixed
		case "recovery":
			return 1.3  // Uncertain periods need more testing
		default:
			return 1.0
		}
	case MidTerm:
		switch s.MarketCondition {
		case "bull":
			return 1.2  // Bull markets for trend following
		case "bear":
			return 1.4  // Bear markets have more phases
		case "sideways":
			return 1.0  // Normal for range trading
		case "mixed":
			return 1.6  // Need full cycle testing
		case "recovery":
			return 1.5  // Transition periods complex
		default:
			return 1.0
		}
	case LongTerm:
		switch s.MarketCondition {
		case "bull", "bear":
			return 1.1  // Single trends are simpler
		case "sideways":
			return 1.2  // Long sideways rare, need more data
		case "mixed":
			return 1.0  // Expected for long-term
		case "recovery":
			return 1.3  // Include transition analysis
		default:
			return 1.0
		}
	}
	return 1.0
}

func (s *TimeframeSpecificBacktestSelector) getVolatilityMultiplier() float64 {
	// Volatility impact by timeframe
	switch s.Timeframe {
	case ShortTerm:
		switch s.VolatilityTarget {
		case "very_low", "low":
			return 0.8  // Low vol easier to predict short-term
		case "medium":
			return 1.0
		case "high":
			return 1.3  // High vol creates more scenarios
		case "extreme":
			return 1.5  // Extreme vol needs extensive testing
		default:
			return 1.0
		}
	case MidTerm:
		switch s.VolatilityTarget {
		case "very_low":
			return 0.9
		case "low":
			return 0.95
		case "medium":
			return 1.0
		case "high":
			return 1.2
		case "extreme":
			return 1.4
		default:
			return 1.0
		}
	case LongTerm:
		switch s.VolatilityTarget {
		case "very_low", "low":
			return 0.85  // Low vol long-term very stable
		case "medium":
			return 1.0
		case "high", "extreme":
			return 1.15  // Long-term smooths volatility
		default:
			return 1.0
		}
	}
	return 1.0
}

func (s *TimeframeSpecificBacktestSelector) getWealthGoalMultiplier() float64 {
	switch s.WealthGoal {
	case "income":
		return 1.2  // Income strategies need consistent testing
	case "growth":
		return 1.0  // Standard growth approach
	case "preservation":
		return 1.4  // Capital preservation needs stress testing
	case "speculation":
		return 0.8  // Speculation can use shorter periods
	default:
		return 1.0
	}
}

func (s *TimeframeSpecificBacktestSelector) ensureTimeframeBounds(rec TimeframeBacktestRecommendation) TimeframeBacktestRecommendation {
	switch s.Timeframe {
	case ShortTerm:
		// Short-term bounds: 1-24 months
		if rec.MinimumMonths < 1 {
			rec.MinimumMonths = 1
		}
		if rec.RecommendedMonths < 3 {
			rec.RecommendedMonths = 3
		}
		if rec.MaximumMonths > 24 {
			rec.MaximumMonths = 24
		}
	case MidTerm:
		// Mid-term bounds: 6-60 months
		if rec.MinimumMonths < 6 {
			rec.MinimumMonths = 6
		}
		if rec.RecommendedMonths < 12 {
			rec.RecommendedMonths = 12
		}
		if rec.MaximumMonths > 60 {
			rec.MaximumMonths = 60
		}
	case LongTerm:
		// Long-term bounds: 24-180 months (15 years)
		if rec.MinimumMonths < 24 {
			rec.MinimumMonths = 24
		}
		if rec.RecommendedMonths < 36 {
			rec.RecommendedMonths = 36
		}
		if rec.MaximumMonths > 180 {
			rec.MaximumMonths = 180
		}
	}
	return rec
}

func (s *TimeframeSpecificBacktestSelector) generateReasoning() string {
	return fmt.Sprintf("Based on %s timeframe with %s frequency, %d-day position duration, %s market conditions, %s volatility, and %s wealth goal",
		s.Timeframe, s.TradingFrequency, s.PositionDuration, s.MarketCondition, s.VolatilityTarget, s.WealthGoal)
}

func (s *TimeframeSpecificBacktestSelector) getSpecialConsiderations() []string {
	var considerations []string
	
	// Timeframe-specific considerations
	switch s.Timeframe {
	case ShortTerm:
		considerations = append(considerations, "Include market microstructure effects")
		considerations = append(considerations, "Test across different market sessions")
		if s.TradingFrequency == "minutes" || s.TradingFrequency == "hours" {
			considerations = append(considerations, "Consider spread and slippage impact")
			considerations = append(considerations, "Test during low liquidity periods")
		}
	case MidTerm:
		considerations = append(considerations, "Include seasonal patterns")
		considerations = append(considerations, "Test economic announcement impacts")
		considerations = append(considerations, "Validate across different trend regimes")
	case LongTerm:
		considerations = append(considerations, "Include major market cycles")
		considerations = append(considerations, "Test technology adoption phases")
		considerations = append(considerations, "Consider regulatory environment changes")
	}
	
	// Market condition specific
	if s.MarketCondition == "mixed" {
		considerations = append(considerations, "Ensure data includes both bull and bear periods")
		considerations = append(considerations, "Test major market events and crashes")
	}
	
	if s.MarketCondition == "recovery" {
		considerations = append(considerations, "Include transition period analysis")
		considerations = append(considerations, "Test false breakout scenarios")
	}
	
	// Volatility specific
	if s.VolatilityTarget == "high" || s.VolatilityTarget == "extreme" {
		considerations = append(considerations, "Include flash crash events")
		considerations = append(considerations, "Test extreme volatility periods")
	}
	
	// Risk tolerance specific
	if s.RiskTolerance == "very_low" || s.RiskTolerance == "low" {
		considerations = append(considerations, "Focus on drawdown protection")
		considerations = append(considerations, "Test worst-case scenarios extensively")
	}
	
	return considerations
}

func (s *TimeframeSpecificBacktestSelector) getTimeframeSpecificAdvice() []string {
	var advice []string
	
	switch s.Timeframe {
	case ShortTerm:
		advice = append(advice, "Focus on recent market regime - older data may be less relevant")
		advice = append(advice, "Pay attention to transaction costs and slippage")
		advice = append(advice, "Consider paper trading before live implementation")
		advice = append(advice, "Monitor performance daily and adjust quickly")
	case MidTerm:
		advice = append(advice, "Balance between recent data and historical patterns")
		advice = append(advice, "Include both trending and ranging market periods")
		advice = append(advice, "Test performance across different market cap coins")
		advice = append(advice, "Consider gradual position sizing")
	case LongTerm:
		advice = append(advice, "Include complete crypto market cycles in testing")
		advice = append(advice, "Focus on fundamental trend validation")
		advice = append(advice, "Test across major economic cycles")
		advice = append(advice, "Consider generational wealth building aspects")
	}
	
	return advice
}

// GetComprehensiveRecommendations provides recommendations for all timeframes
func GetComprehensiveRecommendations() {
	fmt.Println("📊 COMPREHENSIVE DCA BACKTESTING RECOMMENDATIONS")
	fmt.Println("=" + strings.Repeat("=", 55))
	
	// Show timeframe boundaries first
	fmt.Println("\n🎯 TIMEFRAME DEFINITIONS")
	fmt.Println(strings.Repeat("-", 25))
	boundaries := GetTimeframeBoundaries()
	for category, boundary := range boundaries {
		fmt.Printf("\n%s:\n", strings.ToUpper(string(category)))
		fmt.Printf("  Position Duration: %s\n", boundary.PositionDuration)
		fmt.Printf("  Typical Hold: %s\n", boundary.TypicalHoldTime)
		fmt.Printf("  Backtest Range: %s\n", boundary.BacktestRange)
		fmt.Printf("  Description: %s\n", boundary.Description)
	}
	
	// Short-term scenarios
	fmt.Println("\n\n🔥 SHORT-TERM STRATEGIES")
	fmt.Println("=" + strings.Repeat("=", 30))
	shortTermScenarios := []struct {
		scenario string
		config   TimeframeSpecificBacktestSelector
	}{
		{
			"⚡ Scalping DCA (Minutes-based)",
			TimeframeSpecificBacktestSelector{
				Timeframe:        ShortTerm,
				TradingFrequency: "minutes",
				PositionDuration: 1,
				MarketCondition:  "mixed",
				RiskTolerance:    "very_high",
				VolatilityTarget: "high",
				WealthGoal:       "income",
			},
		},
		{
			"🎯 Daily Swing DCA",
			TimeframeSpecificBacktestSelector{
				Timeframe:        ShortTerm,
				TradingFrequency: "daily",
				PositionDuration: 3,
				MarketCondition:  "bull",
				RiskTolerance:    "high",
				VolatilityTarget: "medium",
				WealthGoal:       "growth",
			},
		},
		{
			"📈 Short Trend Following",
			TimeframeSpecificBacktestSelector{
				Timeframe:        ShortTerm,
				TradingFrequency: "daily",
				PositionDuration: 7,
				MarketCondition:  "mixed",
				RiskTolerance:    "medium",
				VolatilityTarget: "medium",
				WealthGoal:       "growth",
			},
		},
	}
	
	for _, s := range shortTermScenarios {
		fmt.Printf("\n%s\n", s.scenario)
		fmt.Println(strings.Repeat("-", len(s.scenario)))
		
		rec := s.config.CalculateOptimalPeriod()
		printRecommendation(rec)
	}
	
	// Mid-term scenarios
	fmt.Println("\n\n⚡ MID-TERM STRATEGIES")
	fmt.Println("=" + strings.Repeat("=", 25))
	midTermScenarios := []struct {
		scenario string
		config   TimeframeSpecificBacktestSelector
	}{
		{
			"📊 Weekly Swing Trading",
			TimeframeSpecificBacktestSelector{
				Timeframe:        MidTerm,
				TradingFrequency: "weekly",
				PositionDuration: 14,
				MarketCondition:  "mixed",
				RiskTolerance:    "medium",
				VolatilityTarget: "medium",
				WealthGoal:       "growth",
			},
		},
		{
			"🎯 Monthly Position Building",
			TimeframeSpecificBacktestSelector{
				Timeframe:        MidTerm,
				TradingFrequency: "monthly",
				PositionDuration: 45,
				MarketCondition:  "bull",
				RiskTolerance:    "medium",
				VolatilityTarget: "low",
				WealthGoal:       "growth",
			},
		},
		{
			"📈 Trend Following DCA",
			TimeframeSpecificBacktestSelector{
				Timeframe:        MidTerm,
				TradingFrequency: "weekly",
				PositionDuration: 30,
				MarketCondition:  "recovery",
				RiskTolerance:    "medium",
				VolatilityTarget: "high",
				WealthGoal:       "growth",
			},
		},
	}
	
	for _, s := range midTermScenarios {
		fmt.Printf("\n%s\n", s.scenario)
		fmt.Println(strings.Repeat("-", len(s.scenario)))
		
		rec := s.config.CalculateOptimalPeriod()
		printRecommendation(rec)
	}
	
	// Long-term scenarios  
	fmt.Println("\n\n🏦 LONG-TERM STRATEGIES")
	fmt.Println("=" + strings.Repeat("=", 25))
	longTermScenarios := []struct {
		scenario string
		config   TimeframeSpecificBacktestSelector
	}{
		{
			"💎 Quarterly DCA (Conservative)",
			TimeframeSpecificBacktestSelector{
				Timeframe:        LongTerm,
				TradingFrequency: "quarterly",
				PositionDuration: 180,
				MarketCondition:  "mixed",
				RiskTolerance:    "low",
				VolatilityTarget: "low",
				WealthGoal:       "preservation",
			},
		},
		{
			"🚀 Annual Growth DCA",
			TimeframeSpecificBacktestSelector{
				Timeframe:        LongTerm,
				TradingFrequency: "monthly",
				PositionDuration: 365,
				MarketCondition:  "mixed",
				RiskTolerance:    "medium",
				VolatilityTarget: "medium",
				WealthGoal:       "growth",
			},
		},
		{
			"🏛️ Generational HODL DCA",
			TimeframeSpecificBacktestSelector{
				Timeframe:        LongTerm,
				TradingFrequency: "quarterly",
				PositionDuration: 730,
				MarketCondition:  "mixed",
				RiskTolerance:    "very_low",
				VolatilityTarget: "low",
				WealthGoal:       "preservation",
			},
		},
	}
	
	for _, s := range longTermScenarios {
		fmt.Printf("\n%s\n", s.scenario)
		fmt.Println(strings.Repeat("-", len(s.scenario)))
		
		rec := s.config.CalculateOptimalPeriod()
		printRecommendation(rec)
	}
}

func printRecommendation(rec TimeframeBacktestRecommendation) {
	fmt.Printf("🎯 Timeframe: %s\n", strings.ToUpper(string(rec.Timeframe)))
	fmt.Printf("📅 Recommended: %d months (%s)\n", rec.RecommendedMonths, formatDuration(rec.RecommendedMonths))
	fmt.Printf("📅 Range: %d-%d months\n", rec.MinimumMonths, rec.MaximumMonths)
	fmt.Printf("💡 Reasoning: %s\n", rec.Reasoning)
	
	if len(rec.SpecialConsiderations) > 0 {
		fmt.Println("⚠️  Special Considerations:")
		for _, consideration := range rec.SpecialConsiderations {
			fmt.Printf("   • %s\n", consideration)
		}
	}
	
	if len(rec.TimeframeSpecificAdvice) > 0 {
		fmt.Println("💫 Timeframe-Specific Advice:")
		for _, advice := range rec.TimeframeSpecificAdvice {
			fmt.Printf("   • %s\n", advice)
		}
	}
}

// MarketRegimeAnalysis analyzes to choose appropriate periods
type MarketRegimeAnalysis struct {
	CurrentDate      time.Time
	LastMajorCrash   time.Time
	LastBullPeak     time.Time
	CurrentPhase     string
	VolatilityLevel  string
}

func AnalyzeCurrentMarketForBacktest() MarketRegimeAnalysis {
	// Example analysis - in practice you would get real data
	return MarketRegimeAnalysis{
		CurrentDate:     time.Now(),
		LastMajorCrash:  time.Date(2022, 11, 1, 0, 0, 0, 0, time.UTC), // FTX crash
		LastBullPeak:    time.Date(2021, 11, 1, 0, 0, 0, 0, time.UTC), // ATH 2021
		CurrentPhase:    "Recovery/Early Bull",
		VolatilityLevel: "Medium-High",
	}
}

func (ma *MarketRegimeAnalysis) GetContextualRecommendation() BacktestPeriodRecommendation {
	monthsSinceLastCrash := int(time.Since(ma.LastMajorCrash).Hours() / 24 / 30)
	_ = int(time.Since(ma.LastBullPeak).Hours() / 24 / 30) // monthsSinceLastPeak - used for reference
	
	var recommendation BacktestPeriodRecommendation
	
	if monthsSinceLastCrash < 12 {
		// Recent crash nearby - need to include crash data
		recommendation.RecommendedMonths = 18
		recommendation.Reasoning = "Include recent crash data to test downside protection"
		recommendation.SpecialConsiderations = []string{
			"Must include crash period for stress testing",
			"Test DCA performance during extreme drawdowns",
		}
	} else if ma.CurrentPhase == "Recovery/Early Bull" {
		// Recovery phase - need to test both bear and early bull
		recommendation.RecommendedMonths = 12
		recommendation.Reasoning = "Cover transition from bear to bull market"
		recommendation.SpecialConsiderations = []string{
			"Include bear market bottom",
			"Test early bull market breakouts",
		}
	} else {
		// Stable period
		recommendation.RecommendedMonths = 8
		recommendation.Reasoning = "Standard period for current market conditions"
	}
	
	// Minimum and maximum bounds
	recommendation.MinimumMonths = int(float64(recommendation.RecommendedMonths) * 0.6)
	recommendation.MaximumMonths = int(float64(recommendation.RecommendedMonths) * 1.8)
	
	return recommendation
}

// formatDuration converts months to a readable duration string
func formatDuration(months int) string {
	if months < 12 {
		return fmt.Sprintf("%d months", months)
	} else if months < 24 {
		years := float64(months) / 12.0
		return fmt.Sprintf("%.1f years", years)
	} else {
		years := months / 12
		return fmt.Sprintf("%d years", years)
	}
}

// ShowMarketCycleAnalysis displays market cycle analysis
func ShowMarketCycleAnalysis() {
	cycles := GetCryptoMarketCycles()
	
	for _, cycle := range cycles {
		fmt.Printf("\n📈 %s (%s)\n", cycle.CycleType, cycle.TypicalDuration)
		fmt.Println(strings.Repeat("-", len(cycle.CycleType)+15))
		fmt.Printf("📝 %s\n", cycle.Description)
		
		fmt.Printf("\n📊 Recommended Backtesting Periods:\n")
		fmt.Printf("   • Short-term: %d months\n", cycle.ShortTermBacktest)
		fmt.Printf("   • Mid-term: %d months\n", cycle.MidTermBacktest)
		fmt.Printf("   • Long-term: %d months\n", cycle.LongTermBacktest)
		
		fmt.Printf("\n🎯 Timeframe-Specific Impact:\n")
		for timeframe, impact := range cycle.TimeframeImpact {
			fmt.Printf("   • %s: %s\n", strings.Title(string(timeframe)), impact)
		}
		
		fmt.Printf("\n🔑 Key Characteristics:\n")
		for _, characteristic := range cycle.KeyCharacteristics {
			fmt.Printf("   • %s\n", characteristic)
		}
	}
}

// MultiPeriodBacktestStrategy comprehensive multi-timeframe approach
func MultiPeriodBacktestStrategy() {
	fmt.Println("🔄 COMPREHENSIVE MULTI-PERIOD BACKTESTING")
	fmt.Println("=" + strings.Repeat("=", 45))
	
	timeframePeriods := map[TimeframeCategory][]struct {
		name   string
		months int
		focus  string
	}{
		ShortTerm: {
			{"Quick Validation", 3, "Recent market patterns"},
			{"Pattern Recognition", 6, "Short-term trends"},
			{"Noise Testing", 9, "Market microstructure"},
			{"Regime Validation", 12, "Current market regime"},
		},
		MidTerm: {
			{"Trend Validation", 12, "Medium-term trends"},
			{"Cycle Testing", 18, "Multiple market phases"},
			{"Robustness Check", 24, "Different market conditions"},
			{"Full Cycle", 36, "Complete market cycles"},
		},
		LongTerm: {
			{"Foundation Test", 36, "Core strategy validation"},
			{"Cycle Robustness", 60, "Multiple complete cycles"},
			{"Regime Changes", 84, "Technology/regulatory shifts"},
			{"Generational Test", 120, "Long-term viability"},
		},
	}
	
	for timeframe, periods := range timeframePeriods {
		fmt.Printf("\n🎯 %s TIMEFRAME\n", strings.ToUpper(string(timeframe)))
		fmt.Println(strings.Repeat("-", 20))
		
		for _, period := range periods {
			fmt.Printf("📊 %s (%d months - %s)\n", period.name, period.months, formatDuration(period.months))
			fmt.Printf("   Focus: %s\n", period.focus)
			fmt.Printf("   Use case: %s\n", getUseCaseForPeriod(period.months))
			fmt.Printf("   Expected insights: %s\n", getExpectedInsights(period.months))
			fmt.Println()
		}
	}
	
	fmt.Println("💡 COMPREHENSIVE STRATEGY RECOMMENDATIONS:")
	fmt.Println("📋 SHORT-TERM (Fast iteration):")
	fmt.Println("   1. Start with 3-6 month backtest for rapid prototyping")
	fmt.Println("   2. Focus on recent market regime and noise resistance")
	fmt.Println("   3. Quick validation cycles (1-2 weeks each)")
	
	fmt.Println("\n📋 MID-TERM (Balanced approach):")
	fmt.Println("   1. Use 12-24 month backtest for strategy development")
	fmt.Println("   2. Include multiple market phases and seasonal effects")
	fmt.Println("   3. Test across different market cap assets")
	
	fmt.Println("\n📋 LONG-TERM (Wealth building):")
	fmt.Println("   1. Minimum 3-5 year backtest for generational strategies")
	fmt.Println("   2. Include full crypto cycles and major events")
	fmt.Println("   3. Focus on compound growth and capital preservation")
}

// ShowQuickReferenceTable creates quick reference table for all timeframes
func ShowQuickReferenceTable() {
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Strategy Type", "Timeframe", "Recommended", "Primary Focus")
	fmt.Println(strings.Repeat("-", 80))
	
	// Short-term strategies
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Scalping DCA", "SHORT", "3-6 months", "Market microstructure")
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Day Trading DCA", "SHORT", "4-8 months", "Daily patterns")
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Short Swing DCA", "SHORT", "6-12 months", "Quick trends")
	
	fmt.Println()
	// Mid-term strategies
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Weekly Swing DCA", "MID", "12-24 months", "Weekly cycles")
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Monthly DCA", "MID", "18-36 months", "Monthly trends")
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Trend Following DCA", "MID", "24-48 months", "Market cycles")
	
	fmt.Println()
	// Long-term strategies
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Quarterly DCA", "LONG", "36-84 months", "Market cycles")
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "Annual DCA", "LONG", "60-120 months", "Technology trends")
	fmt.Printf("%-25s %-15s %-15s %-25s\n", "HODL DCA", "LONG", "84-180 months", "Generational wealth")
}

// ShowTimeframeSummary summarizes by each timeframe
func ShowTimeframeSummary() {
	boundaries := GetTimeframeBoundaries()
	
	for _, category := range []TimeframeCategory{ShortTerm, MidTerm, LongTerm} {
		boundary := boundaries[category]
		fmt.Printf("\n🎯 %s SUMMARY:\n", strings.ToUpper(string(category)))
		fmt.Println(strings.Repeat("-", 25))
		
		switch category {
		case ShortTerm:
			fmt.Println("   • MINIMUM: 3 months (pattern recognition)")
			fmt.Println("   • OPTIMAL: 6-9 months (noise vs signal balance)")
			fmt.Println("   • MAXIMUM: 12 months (avoid old regime overfitting)")
			fmt.Println("   • FOCUS: Market microstructure, recent patterns")
			fmt.Println("   • RISK: High frequency requires recent data")
		case MidTerm:
			fmt.Println("   • MINIMUM: 12 months (multiple cycles)")
			fmt.Println("   • OPTIMAL: 18-36 months (robust validation)")
			fmt.Println("   • MAXIMUM: 60 months (comprehensive testing)")
			fmt.Println("   • FOCUS: Trend following, seasonal patterns")
			fmt.Println("   • RISK: Balance historical vs current relevance")
		case LongTerm:
			fmt.Println("   • MINIMUM: 36 months (basic cycle coverage)")
			fmt.Println("   • OPTIMAL: 60-84 months (full cycle robustness)")
			fmt.Println("   • MAXIMUM: 180 months (generational testing)")
			fmt.Println("   • FOCUS: Market cycles, fundamental trends")
			fmt.Println("   • RISK: Technology/regulatory regime changes")
		}
		
		fmt.Printf("   • KEY INSIGHT: %s\n", boundary.Description)
	}
	
	fmt.Println("\n🚀 UNIVERSAL PRINCIPLES:")
	fmt.Println("   ✅ Always include recent major market events")
	fmt.Println("   ✅ Test across different volatility regimes")
	fmt.Println("   ✅ Validate with out-of-sample data")
	fmt.Println("   ✅ Consider transaction costs in your timeframe")
	fmt.Println("   ✅ Paper trade before live implementation")
	fmt.Println("   ✅ Monitor performance and adapt accordingly")
}

func getUseCaseForPeriod(months int) string {
	switch {
	case months <= 3:
		return "Quick validation, parameter fine-tuning"
	case months <= 6:
		return "Strategy development, indicator optimization"
	case months <= 12:
		return "Robustness testing, risk assessment"
	case months <= 24:
		return "Full cycle testing, drawdown analysis"
	default:
		return "Long-term viability assessment"
	}
}

func getExpectedInsights(months int) string {
	switch {
	case months <= 3:
		return "Recent performance, current market fit"
	case months <= 6:
		return "Trend following ability, volatility handling"
	case months <= 12:
		return "Multiple market conditions, seasonal effects"
	case months <= 24:
		return "Bull/bear cycle performance, major event survival"
	default:
		return "Long-term viability, regime change adaptation"
	}
}

// MAIN FUNCTION to use comprehensive timeframe system
func main() {
	fmt.Println("🚀 COMPREHENSIVE DCA BACKTESTING GUIDE")
	fmt.Println("📈 Short-term • Mid-term • Long-term Strategies")
	fmt.Println("=" + strings.Repeat("=", 50))
	
	// 1. Show comprehensive recommendations for all timeframes
	GetComprehensiveRecommendations()
	
	// 2. Show market cycle analysis
	fmt.Println("\n\n🔄 CRYPTO MARKET CYCLE ANALYSIS")
	fmt.Println("=" + strings.Repeat("=", 35))
	ShowMarketCycleAnalysis()
	
	// 3. Analyze current market context
	fmt.Println("\n🔍 CURRENT MARKET CONTEXT")
	fmt.Println("=" + strings.Repeat("=", 30))
	
	analysis := AnalyzeCurrentMarketForBacktest()
	contextRec := analysis.GetContextualRecommendation()
	
	fmt.Printf("📊 Current Phase: %s\n", analysis.CurrentPhase)
	fmt.Printf("📈 Volatility Level: %s\n", analysis.VolatilityLevel)
	fmt.Printf("⏰ Months since last crash: %.0f\n", time.Since(analysis.LastMajorCrash).Hours()/24/30)
	fmt.Printf("⏰ Months since last peak: %.0f\n", time.Since(analysis.LastBullPeak).Hours()/24/30)
	fmt.Printf("\n📅 Contextual Recommendation: %d months (%s)\n", contextRec.RecommendedMonths, formatDuration(contextRec.RecommendedMonths))
	fmt.Printf("💭 Reasoning: %s\n", contextRec.Reasoning)
	
	if len(contextRec.SpecialConsiderations) > 0 {
		fmt.Println("⚠️  Context-specific considerations:")
		for _, consideration := range contextRec.SpecialConsiderations {
			fmt.Printf("   • %s\n", consideration)
		}
	}
	
	// 4. Show multi-period approach
	fmt.Println("\n" + strings.Repeat("=", 50))
	MultiPeriodBacktestStrategy()
	
	// 5. Comprehensive quick reference table
	fmt.Println("\n📋 COMPREHENSIVE QUICK REFERENCE")
	fmt.Println("=" + strings.Repeat("=", 35))
	ShowQuickReferenceTable()
	
	// 6. Final summary
	fmt.Println("\n🎯 COMPREHENSIVE SUMMARY BY TIMEFRAME:")
	fmt.Println("=" + strings.Repeat("=", 45))
	ShowTimeframeSummary()
}