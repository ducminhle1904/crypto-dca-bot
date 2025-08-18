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
# Full optimization - optimizes all indicator parameters
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Compare intervals for one symbol
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize -period 30d


```

**What gets optimized:**

- **All parameters**: Base amount, Max multiplier, Price threshold, TP percent, RSI period/oversold, MACD fast/slow/signal, BB period/stddev, SMA period (11+ parameters)

On success:

- **Results saved to**: `results/<SYMBOL>_<INTERVAL>/`
- Each directory contains: `best.json` and `trades.xlsx` (Trades + Cycles tabs)

## 4) Apply the best configuration

```bash
# Use exact optimized settings
go run cmd/backtest/main.go -config results/<SYMBOL>_<INTERVAL>/best.json

# Override just TP for this run
go run cmd/backtest/main.go -config results/BTCUSDT_1h/best.json -tp 0.025

# Force hold mode (no cycles)
go run cmd/backtest/main.go -config results/BTCUSDT_1h/best.json -cycle=false
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

  - `-all-intervals` optimize each interval and compare

### Optimization Modes

- **Full optimization mode**:
  - Optimizes all parameters including indicator settings
  - Optimizes: Strategy parameters + RSI period/oversold + MACD fast/slow/signal + BB period/stddev + SMA period
  - Population: 60, Generations: 35

### Recommended Indicator Parameters by Timeframe

| Timeframe | RSI (Period/Oversold/Overbought) | MACD (Fast/Slow/Signal) | BB (Period/Std Dev) | SMA (Period) |
| --------- | -------------------------------- | ----------------------- | ------------------- | ------------ |
| **5m**    | 9 / 30 / 70                      | 5/13/1                  | 10 / 2              | 9            |
| **15m**   | 14 / 30 / 70                     | 8/17/9                  | 20 / 2              | 21           |
| **30m**   | 14 / 30 / 70                     | 12/26/9                 | 20 / 2              | 50           |
| **1h**    | 14 / 30 / 70                     | 12/26/9                 | 20 / 2              | 50           |
| **4h**    | 14 / 30 / 70                     | 12/26/9                 | 20 / 2              | 100          |

_These values are based on trading community recommendations and work well for crypto volatility. They can be used as starting points for manual configuration._

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
- **Thorough optimization:** Optimize all indicator parameters for maximum performance
- **Faster testing:** Shorten `-period` (e.g., `-period 30d`), use higher timeframe (4h vs 15m)
- **Capital growth:** Balance increases after profitable cycles; per-BUY sizing still respects BaseAmount and MaxMultiplier

### Recommended Workflow

1. **Quick test:** Use shorter periods (e.g., `-period 30d`) to get fast results
2. **Full optimization:** Optimize all parameters for maximum performance
3. **Interval comparison:** Use `-all-intervals` to find the best timeframe
4. **Final tune:** Run full optimization on your chosen best interval

### Comparing Fixed vs Full Optimization

Results are saved to separate directories so you can easily compare both modes:

```bash
# Run both optimizations on the same timeframe
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Results are saved separately:
# results/BTCUSDT_1h_fixed/   <- Fixed indicators results
# results/BTCUSDT_1h_full/    <- Full optimization results

# Compare the Excel files to see which performed better
```

This allows you to determine if the extra optimization time for indicator parameters is worth it for your specific use case.
