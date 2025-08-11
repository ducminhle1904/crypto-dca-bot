# DCA Bot Backtesting Guide

## Overview

This guide helps you run backtests, optimize parameters, and compare timeframes for the Enhanced DCA Bot. It now supports auto interval selection, trailing time windows, indicator-combo optimization, and saving the best configuration.

## Quick Start

### 1. Download Historical Data (per symbol/interval)

- Output structure: `data/historical/<SYMBOL>/<INTERVAL>/candles.csv`

```bash
# Single symbol/timeframe (default 1 year)
go run scripts/download_historical_data.go -symbol BTCUSDT -interval 1h

# Multiple symbols/intervals
go run scripts/download_historical_data.go -symbols BTCUSDT,ETHUSDT -intervals 1h,4h
```

### 2. Run a Backtest (no -data needed)

```bash
# Uses data/historical/BTCUSDT/1h/candles.csv automatically
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h

# Limit to trailing period
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -period 30d
```

### 3. Optimize (parameters + indicator combos)

```bash
# Single interval: optimizes parameters and indicator combinations automatically
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Save best config (auto-named if omitted)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize -best-config-out results/best_BTCUSDT_1h.json

# Multi-interval: optimizes per interval and compares
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize

# Restrict indicator universe
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize -indicators rsi,macd
```

## Common Flags

- Data selection

  - `-symbol` (e.g., BTCUSDT)
  - `-interval` (e.g., 5m, 15m, 1h, 4h, 1d)
  - `-data` (explicit path overrides -interval)
  - `-data-root` (default: `data/historical`)
  - `-period` trailing window: `7d`, `30d`, `180d`, `365d` or any Go duration (e.g., `168h`)

- Strategy/indicators

  - `-indicators`: limit the indicator set (subset of `rsi,macd,bb,sma`)
  - `-min-interval`: override minimum time between trades (e.g., `30m`, `1h`). Default: matches data interval if detectable

- Optimization

  - `-optimize`: optimizes base/max and indicator parameters; also evaluates all non-empty indicator combos by default
  - `-all-intervals`: run per-interval backtests/optimizations for the given `-symbol`
  - `-best-config-out`: save the winning configuration to a JSON file; if omitted, auto-saves to `results/best_<SYMBOL>_<INTERVAL>.json`

- Output
  - `-output console|json|csv`
  - `-verbose`: print best-so-far improvements during optimization

## How Decisions Are Made

- Indicators are chosen via `-indicators` or by the optimizer. The strategy evaluates each indicator’s `ShouldBuy/ShouldSell` and aggregates:

  - confidence = buySignals / totalIndicators
  - If confidence ≥ minConfidence (default 0.6) and minInterval has passed, it buys
  - Position size = base × (1 + confidence × strength), capped at maxMultiplier

- Tip to increase trades: use 3 indicators (e.g., `rsi,macd,sma`) so any two buys yield confidence ≈ 0.67 ≥ 0.6.

## Examples

### Faster trading on a short timeframe

```bash
go run cmd/backtest/main.go -symbol BTCUSDT -interval 5m -indicators rsi,bb,sma -period 7d -optimize
```

### Compare timeframes

```bash
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize -period 30d
```

### Use custom minInterval

```bash
go run cmd/backtest/main.go -symbol BTCUSDT -interval 15m -min-interval 15m
```

## Output and Best Config

- The best configuration (for a single interval optimize or the best among `-all-intervals`) is printed in JSON and saved to file.
- Auto filename: `results/best_<SYMBOL>_<INTERVAL>.json` if `-best-config-out` not provided.

## Troubleshooting

- “No data file found”: Run the downloader and/or verify `-symbol` and `-interval`
- MinInterval not matching timeframe: pass `-min-interval 5m` to override
- Optimization is slow: restrict `-indicators`, use shorter `-period`, or try a faster interval
