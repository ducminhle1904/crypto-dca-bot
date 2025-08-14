# DCA Bot Backtesting Guide

This guide consolidates quickstart steps, optimization, configuration, and detailed outputs for the Enhanced DCA Bot.

## Overview

- **Always uses all 4 indicators:** RSI, MACD, Bollinger Bands, and SMA
- Cycles (DCA -> TP -> reset) are ON by default
- Without `-tp` and not optimizing, TP defaults to 2% so cycles can trigger
- With `-optimize`, we sweep TP candidates (1%..6%), base/max, and indicator parameters
- **Two optimization modes:** Fixed indicators (fast) or full parameter optimization (thorough)
- Exports can be CSV or Excel; by default, auto-saves Excel (trades.xlsx) in optimize flows

## Prerequisites

- Go installed
- Data directory: `data/historical/<SYMBOL>/<INTERVAL>/candles.csv`

## 1) Download historical data

```bash
# Single symbol/interval
go run scripts/download_historical_data.go -symbol BTCUSDT -interval 1h

# Multiple symbols/intervals
go run scripts/download_historical_data.go -symbols BTCUSDT,ETHUSDT -intervals 1h,4h
```

## 2) Run a backtest fast

```bash
# Auto-picks data/historical/BTCUSDT/1h/candles.csv
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h

# Explicit TP (2%)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -tp 0.02

# Hold mode (no cycles, TP ignored)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -cycle=false

# Use only the last 30 days
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -period 30d
```

## 3) Optimize parameters

```bash
# Fast optimization - uses recommended indicator parameters for the timeframe
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize -fixed-indicators

# Full optimization - optimizes all indicator parameters too (slower but thorough)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Compare intervals for one symbol
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize -period 30d

# Use fixed indicators with interval comparison (much faster)
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize -fixed-indicators -period 30d
```

**What gets optimized:**

- **Fixed indicators mode**: Base amount, Max multiplier, Price threshold, TP percent (3-4 parameters)
- **Full optimization mode**: Above + RSI period/oversold, MACD fast/slow/signal, BB period/stddev, SMA period (11+ parameters)

On success:

- **Fixed indicators mode**: Results saved to `results/<SYMBOL>_<INTERVAL>_fixed/`
- **Full optimization mode**: Results saved to `results/<SYMBOL>_<INTERVAL>_full/`
- Each directory contains: `best.json` and `trades.xlsx` (Trades + Cycles tabs)

## 4) Apply the best configuration

```bash
# Use exact optimized settings from fixed indicators mode
go run cmd/backtest/main.go -config results/<SYMBOL>_<INTERVAL>_fixed/best.json

# Use exact optimized settings from full optimization mode
go run cmd/backtest/main.go -config results/<SYMBOL>_<INTERVAL>_full/best.json

# Override just TP for this run
go run cmd/backtest/main.go -config results/BTCUSDT_1h_full/best.json -tp 0.025

# Force hold mode (no cycles)
go run cmd/backtest/main.go -config results/BTCUSDT_1h_fixed/best.json -cycle=false
```

## Flags reference

- **Data selection**
  - `-symbol` (e.g., BTCUSDT)
  - `-interval` (e.g., 5m, 15m, 1h, 4h, 1d)
  - `-data` explicit file path overrides `-interval`/`-data-root`
  - `-data-root` (default: `data/historical`)
  - `-period` trailing window: `7d`, `30d`, `180d`, `365d` or any Go duration (e.g., `168h`)
- **Strategy / indicators**
  - Always uses all 4 indicators: RSI, MACD, Bollinger Bands, SMA
- **Cycles / TP**
  - `-cycle` (default: true) enable TP cycles; set `false` to hold
  - `-tp` decimal TP (e.g., `0.02` = 2%); applied only when `-cycle=true`
- **Optimization**
  - `-optimize` enable parameter optimization
  - `-fixed-indicators` use recommended indicator parameters for timeframe (faster), only optimize strategy parameters
  - `-all-intervals` optimize each interval and compare

### Optimization Modes

- **Fixed indicators mode** (`-fixed-indicators`):
  - Uses timeframe-optimized indicator parameters (RSI 14/30, MACD 12/26/9, etc.)
  - Only optimizes: Base amount, Max multiplier, Price threshold, TP percent
  - ~75% faster optimization (25 population × 15 generations)
- **Full optimization mode** (default):
  - Optimizes all parameters including indicator settings
  - Optimizes: Strategy parameters + RSI period/oversold + MACD fast/slow/signal + BB period/stddev + SMA period
  - More thorough but slower (60 population × 35 generations)

### Fixed Indicator Parameters by Timeframe

When using `-fixed-indicators`, these community-recommended values are applied:

| Timeframe | RSI (Period/Oversold/Overbought) | MACD (Fast/Slow/Signal) | BB (Period/Std Dev) | SMA (Period) |
| --------- | -------------------------------- | ----------------------- | ------------------- | ------------ |
| **5m**    | 9 / 30 / 70                      | 5/13/1                  | 10 / 2              | 9            |
| **15m**   | 14 / 30 / 70                     | 8/17/9                  | 20 / 2              | 21           |
| **30m**   | 14 / 30 / 70                     | 12/26/9                 | 20 / 2              | 50           |
| **1h**    | 14 / 30 / 70                     | 12/26/9                 | 20 / 2              | 50           |
| **4h**    | 14 / 30 / 70                     | 12/26/9                 | 20 / 2              | 100          |

_These values are based on trading community recommendations and work well for crypto volatility._

## Output structure

- Console summary: initial/final balance, total return, max drawdown, Sharpe (placeholder), trades, cycles completed
- Excel (`trades.xlsx`):
  - Trades sheet: Cycle, Entry/Exit price & time, Quantity (USDT), Summary
  - Cycles sheet: Cycle number, Start/End time, Entries, Target price, Average price, PnL (USDT), Capital (USDT)

## Strategy logic (concise)

- **All 4 indicators active:** RSI, MACD, Bollinger Bands, SMA; each candle:
  - confidence = buySignals / 4 (total indicators)
  - If confidence ≥ minConfidence (default 0.5) -> BUY (≥2 indicators must agree)
  - amount = BaseAmount × (1 + confidence × strength), capped by MaxMultiplier
  - Commission deducted; quantity = netUSDT / price
- **Cycle TP** (when `-cycle=true`):
  - TP triggers at `price ≥ avgEntry × (1 + TP)`
  - Realize PnL for all open entries (allocate sell fee proportionally), close cycle, then restart on next BUY
- **Hold mode** (`-cycle=false`):
  - No TP; unrealized PnL applied at the end

## Tips

- **All 4 indicators active** ensures good signal quality (≥2 must agree for buys with minConfidence = 0.5)
- **Fast optimization:** Use `-fixed-indicators` for ~75% speed boost when you trust standard parameters
- **Thorough optimization:** Skip `-fixed-indicators` to optimize all indicator parameters (slower but more comprehensive)
- **Faster testing:** Shorten `-period` (e.g., `-period 30d`), use higher timeframe (4h vs 15m)
- **Capital growth:** Balance increases after profitable cycles; per-BUY sizing still respects BaseAmount and MaxMultiplier

### Recommended Workflow

1. **Quick test:** `-fixed-indicators` to get fast results with proven indicator settings
2. **Full optimization:** Remove `-fixed-indicators` if you want to squeeze out maximum performance
3. **Interval comparison:** Use `-all-intervals -fixed-indicators` to quickly find best timeframe
4. **Final tune:** Run full optimization on your chosen best interval

### Comparing Fixed vs Full Optimization

Results are saved to separate directories so you can easily compare both modes:

```bash
# Run both optimizations on the same timeframe
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize -fixed-indicators
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Results are saved separately:
# results/BTCUSDT_1h_fixed/   <- Fixed indicators results
# results/BTCUSDT_1h_full/    <- Full optimization results

# Compare the Excel files to see which performed better
```

This allows you to determine if the extra optimization time for indicator parameters is worth it for your specific use case.
