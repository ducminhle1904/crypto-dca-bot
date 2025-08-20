# DCA Bot Backtesting Guide

This comprehensive guide covers running backtests, optimizing strategy parameters, and analyzing results for the Enhanced DCA Bot with both Classic and Advanced combo indicators.

## 🚀 Quick Start

### 1. Download Historical Data

First, download the historical data required for backtesting.

```bash
# For a single symbol and interval from Bybit
go run scripts/download_bybit_historical_data.go -symbol BTCUSDT -interval 60 -category spot

# For multiple symbols and intervals
go run scripts/download_bybit_historical_data.go -symbols BTCUSDT,ETHUSDT -intervals 60,240 -category spot
```

Data will be saved to the `data/<EXCHANGE>/<CATEGORY>/<SYMBOL>/<INTERVAL>/` directory structure.

### 2. Run a Basic Backtest

```bash
# Run a backtest using the best available data for a symbol
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -exchange bybit

# Use classic combo indicators (RSI + MACD + Bollinger Bands + EMA)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h

# Use advanced combo indicators (Hull MA + MFI + Keltner + WaveTrend)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -advanced-combo

# Limit the backtest to the last 30 days of data
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -period 30d
```

### 3. Optimize Strategy Parameters

The backtesting tool uses a genetic algorithm to find optimal parameters for both indicator combinations.

```bash
# Run optimization with classic combo indicators
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Run optimization with advanced combo indicators
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -advanced-combo -optimize

# Compare performance across all available intervals for a symbol
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize -period 30d

# Console-only mode (no file output)
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize -console-only
```

## 🎯 Strategy Combinations

### Classic Combo (Default)

- **RSI**: Relative Strength Index with configurable oversold/overbought levels
- **MACD**: Moving Average Convergence Divergence with fast/slow/signal periods
- **Bollinger Bands**: Price volatility bands with standard deviation
- **EMA**: Exponential Moving Average for trend direction

### Advanced Combo

- **Hull MA**: Hull Moving Average for smoother trend signals
- **MFI**: Money Flow Index for volume-based momentum
- **Keltner Channels**: Volatility-based price channels
- **WaveTrend**: Advanced oscillator for trend reversals

## ⚙️ Configuration Options

### Command-Line Flags

#### Data Selection

- `-symbol`: Trading symbol (e.g., `BTCUSDT`, `ETHUSDT`)
- `-interval`: Data interval (e.g., `5m`, `15m`, `1h`, `4h`, `1d`)
- `-data`: Explicit path to a data file
- `-exchange`: Exchange to use (default: `bybit`)
- `-data-root`: Root folder for data (default: `data`)
- `-period`: Trailing window (e.g., `7d`, `30d`, `180d`, `365d`)

#### Strategy Configuration

- `-base-amount`: Base DCA amount in USD (default: $40)
- `-max-multiplier`: Maximum position multiplier (default: 3.0)
- `-price-threshold`: Minimum price drop % for next DCA entry (default: 2%)
- `-window-size`: Data window size for analysis (default: 100)
- `-advanced-combo`: Use advanced combo indicators instead of classic

#### Risk Management

- `-balance`: Initial balance in USD (default: $500)
- `-commission`: Trading commission (default: 0.05%)
- `-tp-percent`: Take-profit percentage (default: 2% when optimizing)

#### Output Control

- `-console-only`: Only display results in console, skip file output
- `-env`: Environment file path for API credentials (default: `.env`)

### Configuration Files

You can use JSON configuration files for complex setups:

```bash
# Load configuration from file
go run cmd/backtest/main.go -config configs/bybit/btc_5m_bybit.json

# The tool automatically prepends "configs/" if no path is specified
go run cmd/backtest/main.go -config btc_5m_bybit
```

#### Configuration File Format

The tool supports both flat and nested JSON formats:

**Flat Format (Legacy):**

```json
{
  "symbol": "BTCUSDT",
  "interval": "5m",
  "base_amount": 40,
  "max_multiplier": 3.0,
  "price_threshold": 0.02,
  "use_advanced_combo": false,
  "rsi_period": 14,
  "rsi_oversold": 30
}
```

**Nested Format (Recommended):**

```json
{
  "strategy": {
    "symbol": "BTCUSDT",
    "interval": "5m",
    "base_amount": 40,
    "max_multiplier": 3.0,
    "price_threshold": 0.02,
    "use_advanced_combo": false,
    "rsi": {
      "period": 14,
      "oversold": 30,
      "overbought": 70
    }
  },
  "risk": {
    "initial_balance": 500,
    "commission": 0.0005,
    "min_order_qty": 0.01
  }
}
```

## 🔬 Genetic Algorithm Optimization

### Optimization Parameters

The genetic algorithm optimizes:

- **Strategy Parameters**: Base amount, max multiplier, price threshold, TP percentage
- **Indicator Parameters**: All periods, thresholds, and multipliers for the selected combo
- **Risk Parameters**: Commission rates and minimum order quantities

### GA Settings

- **Population Size**: 60 individuals
- **Generations**: 35 generations
- **Mutation Rate**: 10%
- **Crossover Rate**: 80%
- **Elite Size**: 6 individuals
- **Tournament Size**: 3 individuals

### Optimization Ranges

#### Classic Combo

- **RSI**: Periods 10-25, Oversold 20-40
- **MACD**: Fast 6-18, Slow 20-35, Signal 7-14
- **Bollinger Bands**: Periods 10-30, StdDev 1.5-3.0
- **EMA**: Periods 15-120

#### Advanced Combo

- **Hull MA**: Periods 10-50
- **MFI**: Periods 10-22, Oversold 15-30, Overbought 70-85
- **Keltner**: Periods 15-50, Multipliers 1.5-3.0
- **WaveTrend**: N1 8-20, N2 18-35, Overbought 50-80, Oversold -80 to -50

## 📊 Output and Results

### Console Output

The tool provides comprehensive console output including:

- Strategy configuration summary
- Combo type and indicator parameters
- Performance metrics (Total Return, Sharpe Ratio, Max Drawdown)
- Trade statistics (Win Rate, Profit Factor, Total Trades)

### File Output

#### Standard Outputs

- **`best.json`**: Optimized configuration in nested JSON format
- **`trades.xlsx`**: Professional Excel workbook with multiple sheets

#### Excel Workbook Structure

1. **Trades Sheet**: Detailed trade-by-trade analysis with professional formatting
2. **Cycles Sheet**: DCA cycle performance and timing analysis
3. **Dashboard Sheet**: Key metrics and performance visualization

#### CSV Alternative

- **`trades.csv`**: Simplified trade data for external analysis

### Results Directory Structure

```
results/
├── BTCUSDT_5m/
│   ├── best.json
│   └── trades.xlsx
├── BTCUSDT_1h/
│   ├── best.json
│   └── trades.xlsx
└── ETHUSDT_1h/
    ├── best.json
    └── trades.xlsx
```

## 📈 Performance Metrics

### Key Metrics Calculated

- **Total Return**: Overall strategy performance
- **Max Drawdown**: Largest peak-to-trough decline
- **Sharpe Ratio**: Risk-adjusted returns
- **Profit Factor**: Ratio of gross profit to gross loss
- **Win Rate**: Percentage of profitable trades
- **Total Trades**: Number of completed DCA cycles

### Advanced Analysis

- **Cycle Analysis**: Entry/exit timing and duration
- **Price Drop Analysis**: Entry price relative to cycle start
- **Capital Efficiency**: Return on invested capital
- **Risk-Adjusted Metrics**: Performance relative to volatility

## 🔧 Advanced Features

### Multi-Interval Analysis

```bash
# Compare performance across all available intervals
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -optimize

# This will:
# 1. Scan all available intervals for the symbol
# 2. Run optimization on each interval
# 3. Compare results and identify the best interval
# 4. Provide comprehensive comparison table
```

### Data Filtering

```bash
# Limit analysis to recent data
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -period 30d

# Use specific time periods
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -period 7d
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -period 180d
```

### Realistic Trading Simulation

- **Minimum Order Quantities**: Automatically fetched from Bybit API
- **Commission Handling**: Realistic trading costs
- **Price Validation**: Data quality checks and filtering
- **Cycle Mode**: Automatic take-profit and re-entry logic

## 🚨 Troubleshooting

### Common Issues

1. **No Data Found**

   - Ensure data files exist in the correct directory structure
   - Check symbol and interval parameters
   - Verify exchange and category settings

2. **Optimization Not Converging**

   - Increase population size or generations
   - Adjust mutation and crossover rates
   - Check data quality and quantity

3. **API Errors**
   - Verify Bybit API credentials in `.env` file
   - Check network connectivity
   - Ensure API permissions include instrument info

### Performance Tips

1. **Data Caching**: The tool automatically caches loaded data for faster subsequent runs
2. **Parallel Processing**: Optimization uses multiple workers for faster evaluation
3. **Period Filtering**: Use `-period` flag to focus on relevant timeframes
4. **Console-Only Mode**: Use `-console-only` for quick testing without file I/O

## 📚 Examples

### Complete Optimization Workflow

```bash
# 1. Download data
go run scripts/download_bybit_historical_data.go -symbol BTCUSDT -intervals 5m,15m,1h,4h -category linear

# 2. Run optimization across all intervals
go run cmd/backtest/main.go -symbol BTCUSDT -all-intervals -advanced-combo -optimize -period 90d

# 3. Use best configuration for live trading
go run cmd/backtest/main.go -config results/BTCUSDT_1h/best.json
```

### Strategy Comparison

```bash
# Compare classic vs advanced combo
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -advanced-combo -optimize

# Compare results in results/BTCUSDT_1h/ directory
```

### Parameter Tuning

```bash
# Start with default parameters
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h

# Optimize specific parameters
go run cmd/backtest/main.go -symbol BTCUSDT -interval 1h -optimize

# Fine-tune with custom ranges
# (Modify OptimizationRanges in main.go for custom parameter ranges)
```

## 🔗 Related Documentation

- **Live Bot**: `cmd/live-bot-v2/README.md`
- **Suggestion System**: `cmd/suggestion-system/DCA_BACKTESTING_GUIDE.md`
- **Data Downloader**: `scripts/README_BYBIT_DOWNLOADER.md`
- **Strategy Implementation**: `internal/strategy/`
- **Technical Indicators**: `internal/indicators/`

---

_This guide covers the latest features as of the current codebase. For additional help or feature requests, please refer to the project documentation or create an issue._
