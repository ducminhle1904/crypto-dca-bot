# DCA Bot Backtesting Guide

This guide consolidates quickstart steps, optimization, configuration, and detailed outputs for the Enhanced DCA Bot.

## Overview

- Cycles (DCA -> TP -> reset) are ON by default
- Without `-tp` and not optimizing, TP defaults to 2% so cycles can trigger
- With `-optimize`, we sweep TP candidates (1%..5%), base/max, indicator params, and indicator combos
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

## 3) Optimize (parameters + indicator combos + TP)

```bash
# Single interval optimize
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Compare intervals for one symbol
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize -period 30d

# Restrict indicator universe
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize -indicators rsi,macd
```

On success:

- Best config printed and saved to `results/<SYMBOL>_<INTERVAL>/best.json`
- Trades saved to `results/<SYMBOL>_<INTERVAL>/trades.xlsx` (Trades + Cycles tabs)

## 4) Apply the best configuration

```bash
# Use exact optimized settings
go run cmd/backtest/main.go -config results/<SYMBOL>_<INTERVAL>/best.json

# Export trades to Excel explicitly
go run cmd/backtest/main.go -config results/<SYMBOL>_<INTERVAL>/best.json \
  -trades-csv-out results/<SYMBOL>_<INTERVAL>/trades.xlsx

# Override just TP for this run
go run cmd/backtest/main.go -config results/<SYMBOL>_<INTERVAL>/best.json -tp 0.025

# Force hold mode (no cycles)
go run cmd/backtest/main.go -config results/<SYMBOL>_<INTERVAL>/best.json -cycle=false
```

## Flags reference

- Data selection
  - `-symbol` (e.g., BTCUSDT)
  - `-interval` (e.g., 5m, 15m, 1h, 4h, 1d)
  - `-data` explicit file path overrides `-interval`/`-data-root`
  - `-data-root` (default: `data/historical`)
  - `-period` trailing window: `7d`, `30d`, `180d`, `365d` or any Go duration (e.g., `168h`)
- Strategy / indicators
  - `-indicators` subset of `rsi,macd,bb,sma`
- Cycles / TP
  - `-cycle` (default: true) enable TP cycles; set `false` to hold
  - `-tp` decimal TP (e.g., `0.02` = 2%); applied only when `-cycle=true`
- Optimization
  - `-optimize` optimize base/max, indicator params, combos, and TP (1%..5%)
  - `-all-intervals` optimize each interval and compare
  - `-best-config-out` save winning config; auto-saves to `results/<SYMBOL>_<INTERVAL>/best.json` if omitted
  - `-verbose` print best-so-far improvements
- Output
  - `-output console|json|csv`
  - `-trades-csv-out <path>` write trades export (use `.xlsx` to get Trades + Cycles tabs)

## Output structure

- Console summary: initial/final balance, total return, max drawdown, Sharpe (placeholder), trades, cycles completed
- Excel (`trades.xlsx`):
  - Trades sheet: Cycle, Entry/Exit price & time, Quantity (USDT), Summary
  - Cycles sheet: Cycle number, Start/End time, Entries, Target price, Average price, PnL (USDT), Capital (USDT)

## Strategy logic (concise)

- Indicators elected by `-indicators` or optimizer; each candle:
  - confidence = buySignals / totalIndicators
  - If confidence ≥ minConfidence (default 0.5) -> BUY
  - amount = BaseAmount × (1 + confidence × strength), capped by MaxMultiplier
  - Commission deducted; quantity = netUSDT / price
- Cycle TP (when `-cycle=true`):
  - TP triggers at `price ≥ avgEntry × (1 + TP)`
  - Realize PnL for all open entries (allocate sell fee proportionally), close cycle, then restart on next BUY
- Hold mode (`-cycle=false`):
  - No TP; unrealized PnL applied at the end

## Tips

- More trades: use 3+ indicators so ≥2 confirm buys (minConfidence = 0.5)
- Faster optimize: restrict `-indicators`, shorten `-period`, use a higher timeframe
- Capital usage grows after profitable cycles (balance increases after TP); per-BUY sizing still respects BaseAmount and MaxMultiplier
