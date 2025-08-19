# DCA Bot Backtesting Guide

This guide provides a quickstart for running backtests and optimizing strategy parameters for the Enhanced DCA Bot.

## 1. Download Historical Data

First, download the historical data required for backtesting.

```bash
# For a single symbol and interval from Bybit
go run scripts/download_bybit_historical_data.go -symbol BTCUSDT -interval 60 -category spot

# For multiple symbols and intervals
go run scripts/download_bybit_historical_data.go -symbols BTCUSDT,ETHUSDT -intervals 60,240 -category spot
```

Data will be saved to the `data/<EXCHANGE>/<CATEGORY>/<SYMBOL>/<INTERVAL>/` directory.

## 2. Run a Backtest

Once you have the data, you can run a backtest with different configurations.

```bash
# Run a backtest using the best available data for a symbol
go run cmd/backtest/main.go -symbol BTCUSDT -interval 60 -exchange bybit

# Set a specific Take-Profit (TP) percentage (e.g., 2%)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -tp 0.02

# Run in "hold" mode (no cycles, TP is ignored)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -cycle=false

# Limit the backtest to the last 30 days of data
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -period 30d
```

## 3. Optimize Strategy Parameters

The backtesting tool can run a genetic algorithm to find the optimal strategy parameters.

```bash
# Run a full optimization for a specific symbol and interval
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Compare performance across all available intervals for a symbol
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize -period 30d
```

Optimized results, including a `best.json` configuration file and a `trades.xlsx` spreadsheet, will be saved to the `results/<SYMBOL>_<INTERVAL>/` directory.

## 4. Apply Optimized Configuration

You can use the `best.json` file from an optimization to run a backtest with the exact same settings.

```bash
# Run a backtest with an optimized configuration file
go run cmd/backtest/main.go -config results/BTCUSDT_1h/best.json

# Override a specific parameter (e.g., Take-Profit) for the run
go run cmd/backtest/main.go -config results/BTCUSDT_1h/best.json -tp 0.025
```

## Command-Line Flags Reference

- **Data Selection**
  - `-symbol`: Trading symbol (e.g., `BTCUSDT`).
  - `-interval`: Data interval (e.g., `5m`, `1h`, `4d`).
  - `-data`: Explicit path to a data file.
  - `-exchange`: Exchange to use (default: `bybit`).
  - `-period`: Trailing window for the backtest (e.g., `7d`, `30d`).
- **Strategy Configuration**
  - `-cycle`: Enable or disable Take-Profit cycles (default: `true`).
  - `-tp`: Set the Take-Profit percentage (e.g., `0.02` for 2%).
- **Optimization**
  - `-optimize`: Enable parameter optimization.
  - `-all-intervals`: Run optimization for all available intervals and compare the results.
