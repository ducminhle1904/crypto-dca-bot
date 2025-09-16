package bot

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"math"

	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// parseStringToFloat safely parses a string to float64 with comprehensive validation
func parseStringToFloat(s, fieldName string) (float64, error) {
	if s == "" {
		return 0, fmt.Errorf("%s is empty string", fieldName)
	}
	// Trim whitespace that might cause parsing errors
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("%s contains only whitespace", fieldName)
	}
	// Check for common invalid values
	if s == "null" || s == "undefined" || s == "NaN" {
		return 0, fmt.Errorf("%s has invalid numeric value: %s", fieldName, s)
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("%s parse error: %w", fieldName, err)
	}
	return val, nil
}

// syncPositionData syncs bot internal state with real exchange position data
func (bot *LiveBot) syncPositionData() error {
	ctx := context.Background()
	
	// For better sync reliability, retry up to 3 times with delays
	// This is especially important right after placing orders
	var positions []exchange.Position
	var err error
	
	for attempt := 1; attempt <= 3; attempt++ {
		positions, err = bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
		if err != nil {
			log.Printf("⚠️ Position sync attempt %d failed: %v", attempt, err)
			if attempt < 3 {
				time.Sleep(time.Duration(attempt) * 500 * time.Millisecond) // 500ms, 1s, then fail
				continue
			}
			return fmt.Errorf("failed to get current positions after %d attempts: %w", attempt, err)
		}
		
		// Log to file only for debugging (not terminal)
		// Only log to terminal if positions found
		if len(positions) > 0 {
			// Check if any position has meaningful data before logging
			hasAnyValidData := false
			for _, pos := range positions {
				if pos.Symbol == bot.symbol {
					if pos.Size != "0" && pos.Size != "" || pos.PositionValue != "" && pos.PositionValue != "0" {
						hasAnyValidData = true
						break
					}
				}
			}
			if hasAnyValidData {
				log.Printf("🔍 Found %d positions for %s", len(positions), bot.symbol)
			}
		}
		
		break
	}
	
	// Look for our symbol position - check for both "Buy" and "Long" sides
	var foundPosition bool
	var positionValue, avgPrice float64
	var needsStrategySync bool
	
	for _, pos := range positions {
		// Check for exact symbol match first
		if pos.Symbol == bot.symbol {
			// Parse numeric values with better error handling
			posValue, posValueErr := parseStringToFloat(pos.PositionValue, "PositionValue")
			avgPriceVal, avgPriceErr := parseStringToFloat(pos.AvgPrice, "AvgPrice")
			posSize, posSizeErr := parseStringToFloat(pos.Size, "Size")
			
			// Skip positions with all parsing errors
			if posValueErr != nil && avgPriceErr != nil && posSizeErr != nil {
				continue
			}
			
			// Check if position has meaningful data (non-zero size OR non-zero value)
			hasValidSize := posSizeErr == nil && posSize > 0.001
			hasValidValue := posValueErr == nil && posValue > 0.01
			hasValidPrice := avgPriceErr == nil && avgPriceVal > 0
			
			// Accept position if it has size and price, regardless of side field
			if (hasValidSize || hasValidValue) && hasValidPrice {
				positionValue = posValue
				avgPrice = avgPriceVal
				foundPosition = true
				log.Printf("✅ Found valid position: Value=$%.2f, AvgPrice=$%.4f, Size=%.6f", 
					positionValue, avgPrice, posSize)
				break
			}
		}
	}
	
	// Update bot state with proper mutex protection
	bot.positionMutex.Lock()
	defer bot.positionMutex.Unlock()
	
	if foundPosition {
		// Check if average price has changed (indicating new DCA entry)
		if bot.averagePrice > 0 && math.Abs(avgPrice-bot.averagePrice) > 0.0001 {
			log.Printf("💰 Average price changed: $%.4f → $%.4f", bot.averagePrice, avgPrice)
		}
		
		// Use exchange data directly - no self-calculation
		bot.currentPosition = positionValue
		bot.averagePrice = avgPrice
		bot.totalInvested = positionValue
		
		// Only estimate DCA level during bot startup when dcaLevel is 0
		// NEVER override DCA level during active trading to maintain proper progression
		if bot.config.Strategy.BaseAmount > 0 && bot.dcaLevel == 0 {
			// Add defensive check to prevent division by zero
			if bot.config.Strategy.BaseAmount > 0 {
				estimatedLevel := max(1, int(positionValue/bot.config.Strategy.BaseAmount))
				bot.dcaLevel = estimatedLevel
				log.Printf("🔄 Startup: Estimated DCA level: %d (position: $%.2f, base: $%.2f)", 
					bot.dcaLevel, positionValue, bot.config.Strategy.BaseAmount)
			}
		}
		// During active trading, preserve the tracked DCA level progression
		
		// Position synced successfully
		return nil
	}
	
	// No position found - reset state if bot thinks it has one
	if bot.currentPosition > 0 {
		bot.currentPosition = 0
		bot.averagePrice = 0
		bot.totalInvested = 0
		bot.dcaLevel = 0
		needsStrategySync = true
	}
	
	// Sync strategy state after releasing mutex to avoid deadlock
	if needsStrategySync {
		// Release mutex before calling syncStrategyState
		bot.positionMutex.Unlock()
		bot.syncStrategyState()
		// Re-acquire mutex for proper cleanup
		bot.positionMutex.Lock()
	}
	
	return nil
}

// closeOpenPositions closes all open positions using real position data
func (bot *LiveBot) closeOpenPositions() error {
	ctx := context.Background()
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get current positions: %w", err)
	}
	
	// Look for positions to close
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			positionSize, _ := strconv.ParseFloat(strings.TrimSpace(pos.Size), 64)
			if positionSize <= 0 {
				continue
			}
			
			bot.logger.Info("🚨 Closing position: %s %s (value: $%s)", pos.Size, pos.Symbol, pos.PositionValue)
			
			// Get current price for P&L calculation
			currentPrice, err := bot.exchange.GetLatestPrice(ctx, bot.symbol)
			if err != nil {
				currentPrice, _ = strconv.ParseFloat(strings.TrimSpace(pos.MarkPrice), 64) // Use mark price as fallback
			}
			
			// Place market sell order to close entire position
			orderParams := exchange.OrderParams{
				Category:  bot.category,
				Symbol:    bot.symbol,
				Side:      exchange.OrderSideSell,
				Quantity:  pos.Size,
				OrderType: exchange.OrderTypeMarket,
			}
			
			order, err := bot.exchange.PlaceMarketOrder(ctx, orderParams)
			if err != nil {
				return fmt.Errorf("failed to place sell order: %w", err)
			}
			
			// Calculate P&L based on real position data
			avgPrice, _ := strconv.ParseFloat(strings.TrimSpace(pos.AvgPrice), 64)
			positionValue, _ := strconv.ParseFloat(strings.TrimSpace(pos.PositionValue), 64)
			saleValue := positionSize * currentPrice
			profit := saleValue - positionValue
			
		// Add defensive check to prevent division by zero
		var profitPercent float64
		if positionValue > 0 {
			profitPercent = (profit / positionValue) * 100
		} else {
			log.Printf("⚠️ Warning: Position value is zero, cannot calculate profit percentage")
		}
			
			bot.logger.Info("🔄 POSITION CLOSED (SHUTDOWN) - Order: %s, Qty: %s %s, Entry: $%.2f, Exit: $%.2f, P&L: $%.2f (%.2f%%)", 
				order.OrderID, pos.Size, bot.symbol, avgPrice, currentPrice, profit, profitPercent)
			
			// Reset bot state with mutex protection
			bot.positionMutex.Lock()
			bot.currentPosition = 0
			bot.averagePrice = 0
			bot.totalInvested = 0
			bot.dcaLevel = 0
			bot.positionMutex.Unlock()
			
			// Sync strategy state after position closure
			bot.syncStrategyState()
			
			// Refresh balance after closing position
			if err := bot.syncAccountBalance(); err != nil {
				bot.logger.LogWarning("Balance refresh", "Could not refresh balance after closing position: %v", err)
			}
			
			return nil // Only close one position at a time
		}
	}
	
	bot.logger.Info("✅ No open positions to close")
	return nil
}



// getIntervalDuration converts interval string to time.Duration
func (bot *LiveBot) getIntervalDuration() time.Duration {
	normalizedInterval := normalizeInterval(bot.interval)
	
	switch normalizedInterval {
	case "1":
		return 1 * time.Minute
	case "3":
		return 3 * time.Minute
	case "5":
		return 5 * time.Minute
	case "15":
		return 15 * time.Minute
	case "30":
		return 30 * time.Minute
	case "60":
		return 1 * time.Hour
	case "240":
		return 4 * time.Hour
	case "D":
		return 24 * time.Hour
	default:
		// Try to parse as minutes for any numeric values
		if minutes, err := strconv.Atoi(normalizedInterval); err == nil {
			return time.Duration(minutes) * time.Minute
		}
		log.Printf("⚠️ Unknown interval format '%s' (normalized: '%s'), defaulting to 5 minutes", bot.interval, normalizedInterval)
		return 5 * time.Minute // Safer default than 1 hour
	}
}

// normalizeInterval converts various interval formats to a standard numeric format
func normalizeInterval(interval string) string {
	interval = strings.ToLower(interval)
	
	// Handle standard formats with units - convert to numeric
	switch interval {
	case "1m":
		return "1"
	case "3m":
		return "3"
	case "5m":
		return "5"
	case "15m":
		return "15"
	case "30m":
		return "30"
	case "1h":
		return "60"
	case "4h":
		return "240"
	case "1d":
		return "D"
	}
	
	// Try to parse formats like "123m"
	if strings.HasSuffix(interval, "m") {
		if minutes, err := strconv.Atoi(strings.TrimSuffix(interval, "m")); err == nil {
			return strconv.Itoa(minutes)
		}
	}
	
	// Try to parse formats like "2h"
	if strings.HasSuffix(interval, "h") {
		if hours, err := strconv.Atoi(strings.TrimSuffix(interval, "h")); err == nil {
			return strconv.Itoa(hours * 60) // Convert to minutes
		}
	}
	
	// Return as-is for numeric formats
	return interval
}

// getTimeUntilNextCandle calculates how long to wait until the next candle closes
func (bot *LiveBot) getTimeUntilNextCandle() time.Duration {
	now := time.Now().UTC()
	normalizedInterval := normalizeInterval(bot.interval)
	
	switch normalizedInterval {
	case "1":
		next := now.Truncate(time.Minute).Add(time.Minute)
		return next.Sub(now)
	case "3":
		minutes := now.Minute()
		nextMinute := ((minutes / 3) + 1) * 3
		if nextMinute >= 60 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
	case "5":
		minutes := now.Minute()
		nextMinute := ((minutes / 5) + 1) * 5
		if nextMinute >= 60 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
	case "15":
		minutes := now.Minute()
		nextMinute := ((minutes / 15) + 1) * 15
		if nextMinute >= 60 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
	case "30":
		minutes := now.Minute()
		var nextMinute int
		if minutes < 30 {
			nextMinute = 30
		} else {
			nextMinute = 0 // Next hour
		}
		
		if nextMinute == 0 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
	case "60":
		next := now.Truncate(time.Hour).Add(time.Hour)
		return next.Sub(now)
	case "240":
		hours := now.Hour()
		nextHour := ((hours / 4) + 1) * 4
		if nextHour >= 24 {
			next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), nextHour, 0, 0, 0, time.UTC)
		return next.Sub(now)
	case "D":
		next := now.Truncate(24 * time.Hour).Add(24 * time.Hour)
		return next.Sub(now)
	default:
		// Try to parse normalized interval as minutes for custom intervals
		if minutes, err := strconv.Atoi(normalizedInterval); err == nil {
			intervalMinutes := minutes
			currentMinutes := now.Minute()
			nextMinute := ((currentMinutes / intervalMinutes) + 1) * intervalMinutes
			
			if nextMinute >= 60 {
				next := now.Truncate(time.Hour).Add(time.Hour)
				return next.Sub(now)
			}
			next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
			return next.Sub(now)
		}
		
		// Fallback to 5-minute alignment
		log.Printf("⚠️ Unknown interval format '%s' (normalized: '%s'), using 5-minute alignment", bot.interval, normalizedInterval)
		minutes := now.Minute()
		nextMinute := ((minutes / 5) + 1) * 5
		if nextMinute >= 60 {
			next := now.Truncate(time.Hour).Add(time.Hour)
			return next.Sub(now)
		}
		next := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), nextMinute, 0, 0, time.UTC)
		return next.Sub(now)
	}
}

// printStartupInfo prints initial startup information
func (bot *LiveBot) printStartupInfo() {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("BOT INITIALIZATION")
	t.SetStyle(table.StyleRounded)
	
	t.AppendRows([]table.Row{
		{"📊 Symbol", bot.symbol},
		{"⏰ Interval", bot.interval},
		{"🏪 Category", bot.category},
		{"🏪 Exchange", bot.exchange.GetName()},
		{"🔧 Environment", bot.getEnvironmentString()},
	})
	
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMin: 15, WidthMax: 15, Align: text.AlignLeft},
		{Number: 2, WidthMin: 30, WidthMax: 35, Align: text.AlignLeft},
	})
	
	t.Render()
	fmt.Println()
}

// printBotConfiguration prints the bot configuration
func (bot *LiveBot) printBotConfiguration() {
	ctx := context.Background()
	
	// Get trading constraints
	var minQtyInfo, minNotionalInfo string
	constraints, err := bot.exchange.GetTradingConstraints(ctx, bot.category, bot.symbol)
	if err != nil {
		minQtyInfo = "⚠️ Could not fetch"
		minNotionalInfo = "⚠️ Could not fetch"
	} else {
		minQtyInfo = fmt.Sprintf("%.6f %s", constraints.MinOrderQty, bot.symbol)
		minNotionalInfo = fmt.Sprintf("~$%.2f", constraints.MinOrderValue)
	}

	// Get active indicators
	indicators := bot.config.Strategy.Indicators
	if len(indicators) == 0 {
		indicators = []string{"none"}
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.SetTitle("BOT CONFIGURATION")
	t.SetStyle(table.StyleRounded)
	
	// Trading Parameters Section
	t.AppendRows([]table.Row{
		{"💰 Initial Balance", fmt.Sprintf("$%.2f", bot.balance)},
		{"📈 Base DCA Amount", fmt.Sprintf("$%.2f", bot.config.Strategy.BaseAmount)},
		{"🔄 Max Multiplier", fmt.Sprintf("%.2f", bot.config.Strategy.MaxMultiplier)},
		{"📊 DCA Spacing", bot.getDCASpacingDisplay()},
		{"🎯 Take Profit", fmt.Sprintf("%.2f%%", bot.config.Strategy.TPPercent*100)},
	})
	
	t.AppendSeparator()
	
	// Order Constraints Section
	t.AppendRows([]table.Row{
		{"📏 Min Order Qty", minQtyInfo},
		{"💵 Min Notional", minNotionalInfo},
	})
	
	t.AppendSeparator()
	
	// Technical & Mode Section
	indicatorStr := strings.Join(indicators, ", ")
	t.AppendRows([]table.Row{
		{"📊 Indicators", indicatorStr},
		{"🚨 Trading Mode", bot.getTradingModeString()},
	})
	
	t.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, WidthMin: 18, WidthMax: 18, Align: text.AlignLeft},
		{Number: 2, WidthMin: 25, WidthMax: 40, Align: text.AlignLeft},
	})
	
	t.Render()
	fmt.Println()
}

// getEnvironmentString returns a formatted environment string
func (bot *LiveBot) getEnvironmentString() string {
	env := bot.exchange.GetEnvironment()
	if bot.exchange.IsDemo() {
		return fmt.Sprintf("%s (paper trading)", env)
	}
	return fmt.Sprintf("%s (live trading)", env)
}

// getTradingModeString returns a formatted trading mode string
func (bot *LiveBot) getTradingModeString() string {
	if bot.exchange.IsDemo() {
		return "🧪 DEMO MODE (Paper Trading)"
	}
	return "💰 LIVE TRADING MODE (Real Money!)"
}

// getDCASpacingDisplay returns a formatted string for DCA spacing strategy display
func (bot *LiveBot) getDCASpacingDisplay() string {
	if bot.config.Strategy.DCASpacing == nil {
		return "❌ Not configured"
	}
	
	strategy := bot.config.Strategy.DCASpacing.Strategy
	params := bot.config.Strategy.DCASpacing.Parameters
	
	switch strategy {
	case "fixed":
		if baseThresh, ok := params["base_threshold"].(float64); ok {
			if multiplier, ok := params["threshold_multiplier"].(float64); ok && multiplier > 1.0 {
				return fmt.Sprintf("Fixed Progressive (%.1f%% × %.2fx)", baseThresh*100, multiplier)
			}
			return fmt.Sprintf("Fixed (%.1f%%)", baseThresh*100)
		}
		return "Fixed Progressive"
		
	case "volatility_adaptive":
		if baseThresh, ok := params["base_threshold"].(float64); ok {
			if sensitivity, ok := params["volatility_sensitivity"].(float64); ok {
				return fmt.Sprintf("ATR-Adaptive (%.1f%%, %.1fx sens)", baseThresh*100, sensitivity)
			}
			return fmt.Sprintf("ATR-Adaptive (%.1f%%)", baseThresh*100)
		}
		return "ATR-Adaptive"
		
	default:
		return fmt.Sprintf("%s Strategy", strategy)
	}
}

// Helper functions


