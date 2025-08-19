package bot

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/exchange"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// syncPositionData syncs bot internal state with real exchange position data
func (bot *LiveBot) syncPositionData() error {
	ctx := context.Background()
	
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get current positions: %w", err)
	}
	
	// Look for our symbol position
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			positionValue := parseFloat(pos.PositionValue)
			avgPrice := parseFloat(pos.AvgPrice)
			
			if positionValue > 0 && avgPrice > 0 {
				bot.currentPosition = positionValue
				bot.averagePrice = avgPrice
				bot.totalInvested = positionValue
				
				// Estimate DCA level based on position size vs base amount
				if bot.config.Strategy.BaseAmount > 0 {
					estimatedLevel := int(positionValue / bot.config.Strategy.BaseAmount)
					if estimatedLevel > bot.dcaLevel {
						bot.dcaLevel = estimatedLevel
					}
				}
				return nil
			}
		}
	}
	
	// No position found - reset state if bot thinks it has one
	if bot.currentPosition > 0 {
		bot.currentPosition = 0
		bot.averagePrice = 0
		bot.totalInvested = 0
		bot.dcaLevel = 0
	}
	
	return nil
}

// closeOpenPositions closes all open positions using real position data
func (bot *LiveBot) closeOpenPositions() error {
	fmt.Printf("🔍 Checking for open positions to close...\n")
	
	ctx := context.Background()
	positions, err := bot.exchange.GetPositions(ctx, bot.category, bot.symbol)
	if err != nil {
		return fmt.Errorf("failed to get current positions: %w", err)
	}
	
	// Look for positions to close
	for _, pos := range positions {
		if pos.Symbol == bot.symbol && pos.Side == "Buy" {
			positionSize := parseFloat(pos.Size)
			if positionSize <= 0 {
				continue
			}
			
			fmt.Printf("🚨 Closing position: %s %s (value: $%s)\n", pos.Size, pos.Symbol, pos.PositionValue)
			
			// Get current price for P&L calculation
			currentPrice, err := bot.exchange.GetLatestPrice(ctx, bot.symbol)
			if err != nil {
				currentPrice = parseFloat(pos.MarkPrice) // Use mark price as fallback
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
			avgPrice := parseFloat(pos.AvgPrice)
			positionValue := parseFloat(pos.PositionValue)
			saleValue := positionSize * currentPrice
			profit := saleValue - positionValue
			profitPercent := (profit / positionValue) * 100
			
			fmt.Printf("🔄 POSITION CLOSED (SHUTDOWN)\n")
			fmt.Printf("   Order ID: %s\n", order.OrderID)
			fmt.Printf("   Quantity: %s %s\n", pos.Size, bot.symbol)
			fmt.Printf("   Entry Price: $%.2f\n", avgPrice)
			fmt.Printf("   Exit Price: $%.2f\n", currentPrice)
			fmt.Printf("   P&L: $%.2f (%.2f%%)\n", profit, profitPercent)
			
			// Reset bot state
			bot.currentPosition = 0
			bot.averagePrice = 0
			bot.totalInvested = 0
			bot.dcaLevel = 0
			
			// Refresh balance after closing position
			if err := bot.syncAccountBalance(); err != nil {
				fmt.Printf("⚠️ Could not refresh balance after closing position: %v\n", err)
			}
			
			return nil // Only close one position at a time
		}
	}
	
	fmt.Printf("✅ No open positions to close\n")
	return nil
}

// logStatus logs the current trading status
func (bot *LiveBot) logStatus(currentPrice float64, action string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	
	fmt.Printf("\n[%s] 📊 Market Status\n", timestamp)
	fmt.Printf("💰 Price: $%.2f | Action: %s\n", currentPrice, action)
	fmt.Printf("💼 Balance: $%.2f | Notional Position: $%.2f\n", bot.balance, bot.currentPosition)
	
	if bot.currentPosition > 0 && bot.averagePrice > 0 {
		// Calculate current value based on price change from average
		priceRatio := currentPrice / bot.averagePrice
		currentValue := bot.totalInvested * priceRatio
		unrealizedPnL := currentValue - bot.totalInvested
		unrealizedPercent := (currentPrice - bot.averagePrice) / bot.averagePrice * 100
		
		fmt.Printf("📈 Entry Price: $%.2f | Current Value: $%.2f\n", bot.averagePrice, currentValue)
		fmt.Printf("📊 Unrealized P&L: $%.2f (%.2f%%) | DCA Level: %d\n", unrealizedPnL, unrealizedPercent, bot.dcaLevel)
		fmt.Printf("💰 Margin Used: $%.2f (10x leverage)\n", bot.totalInvested/10)
	} else {
		fmt.Printf("📊 No active position\n")
	}
	fmt.Println(strings.Repeat("-", 50))
}

// getIntervalDuration converts interval string to time.Duration
func (bot *LiveBot) getIntervalDuration() time.Duration {
	switch bot.interval {
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
		// Try to parse as minutes
		if minutes, err := strconv.Atoi(bot.interval); err == nil {
			return time.Duration(minutes) * time.Minute
		}
		return 1 * time.Hour // Default
	}
}

// getTimeUntilNextCandle calculates how long to wait until the next candle closes
func (bot *LiveBot) getTimeUntilNextCandle() time.Duration {
	now := time.Now().UTC()
	
	switch bot.interval {
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
		// Try to parse as minutes for custom intervals
		if minutes, err := strconv.Atoi(bot.interval); err == nil {
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
		{"📊 Price Threshold", fmt.Sprintf("%.2f%%", bot.config.Strategy.PriceThreshold*100)},
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

// Helper functions

// parseFloat safely parses a string to float64
func parseFloat(s string) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return 0
}

// determineTradingCategory determines the trading category based on exchange and symbol
func determineTradingCategory(exchangeName, symbol string) string {
	// This is a simple heuristic - in practice you might want more sophisticated logic
	switch strings.ToLower(exchangeName) {
	case "bybit":
		// For Bybit, assume linear futures for crypto pairs
		if strings.Contains(symbol, "USDT") || strings.Contains(symbol, "USD") {
			return "linear"
		}
		return "spot"
	case "binance":
		// For Binance, determine based on symbol format
		if strings.Contains(symbol, "USDT") || strings.Contains(symbol, "BUSD") {
			return "spot" // Binance spot pairs
		}
		return "futures" // Binance futures
	default:
		return "spot" // Default fallback
	}
}
