# Grid Backtest System

A standardized, professional backtesting system for grid trading strategies with comprehensive reporting.

## Features

✅ **Standardized Config Files** - JSON-based configuration similar to DCA backtest  
✅ **Command-Line Interface** - Professional CLI with flags and options  
✅ **Auto Data Detection** - Automatically finds data files based on symbol/interval  
✅ **Exchange Constraints** - Real exchange minimum quantities and step sizes  
✅ **Advanced Reporting** - Excel workbooks and CSV files with visualizations  
✅ **Multiple Trading Modes** - Long, Short, and Both directions supported

## Quick Start

### Basic Usage

```bash
# Run with BTC long strategy
go run cmd/grid-backtest/main.go -config configs/bybit/grid/btc_long_5m_bybit.json

# Run with ETH both directions strategy
go run cmd/grid-backtest/main.go -config configs/bybit/grid/eth_both_5m_bybit.json

# Run with SOL short strategy
go run cmd/grid-backtest/main.go -config configs/bybit/grid/sol_short_5m_bybit.json
```

### Advanced Usage

```bash
# Custom output directory and specific data file
go run cmd/grid-backtest/main.go \
  -config configs/bybit/grid/btc_long_5m_bybit.json \
  -output my_results \
  -data custom_data/BTCUSDT_5m.csv

# Override symbol and interval from config
go run cmd/grid-backtest/main.go \
  -config configs/bybit/grid/btc_long_5m_bybit.json \
  -symbol ETHUSDT \
  -interval 15m

# Disable report generation (console output only)
go run cmd/grid-backtest/main.go \
  -config configs/bybit/grid/eth_both_5m_bybit.json \
  -report=false

# Limit data for testing (use only last 5000 candles)
go run cmd/grid-backtest/main.go \
  -config configs/bybit/grid/btc_long_5m_bybit.json \
  -max-candles 5000

# Verbose output with detailed grid levels (uses ALL data by default)
go run cmd/grid-backtest/main.go \
  -config configs/bybit/grid/btc_long_5m_bybit.json \
  -verbose
```

## Configuration Files

### File Structure

```
configs/bybit/grid/
├── btc_long_5m_bybit.json     # Bitcoin long-only grid
├── eth_both_5m_bybit.json     # Ethereum both directions
└── sol_short_5m_bybit.json    # Solana short-only grid
```

### Config File Format

```json
{
  "strategy": {
    "symbol": "BTCUSDT",
    "category": "linear",
    "trading_mode": "long",
    "interval": "5m",
    "lower_bound": 95000.0,
    "upper_bound": 110000.0,
    "grid_count": 20,
    "grid_spacing_percent": 1.5,
    "profit_percent": 0.015,
    "position_size": 100.0,
    "leverage": 10.0,
    "use_exchange_constraints": true
  },
  "exchange": {
    "name": "bybit"
  },
  "risk": {
    "initial_balance": 5000.0,
    "commission": 0.0006,
    "min_order_qty": 0.001,
    "qty_step": 0.001,
    "tick_size": 0.1,
    "min_notional": 5.0,
    "max_leverage": 100.0
  }
}
```

## Command Line Options

| Flag           | Description                                           | Default      |
| -------------- | ----------------------------------------------------- | ------------ |
| `-config`      | **Required** - Path to JSON config file               | -            |
| `-data`        | Path to CSV data file (auto-detected if not provided) | Auto         |
| `-output`      | Base output directory for reports                     | `results`    |
| `-symbol`      | Override symbol from config                           | Config value |
| `-interval`    | Override interval from config                         | Config value |
| `-max-candles` | Maximum number of candles to use (0 = all data)       | `0` (all)    |
| `-report`      | Generate comprehensive Excel report                   | `true`       |
| `-verbose`     | Show detailed grid levels and verbose output          | `false`      |
| `-version`     | Show version information                              | `false`      |

## Reports Generated

### Directory Structure

Reports are automatically organized in: `results/grid/{symbol}_{mode}_{timeframe}/`

Example: `results/grid/BTCUSDT_long_5m/`

### Comprehensive Excel Report (5 sheets)

Single file: `grid_backtest_report.xlsx`

- **Grid Summary** - Overview, performance, exchange constraints
- **Grid Levels** - Individual level performance with color coding
- **Grid Positions** - Detailed position tracking with entry/exit data
- **Performance Heat Map** - Visual performance analysis
- **Trading Timeline** - Chronological view of all trading activities

### Text Summary

Quick reference file: `backtest_summary.txt`

- Configuration details
- Financial performance metrics
- Trading statistics
- Grid utilization analysis

## Trading Modes

| Mode    | Description     | Grid Behavior                                   |
| ------- | --------------- | ----------------------------------------------- |
| `long`  | Long-only       | Buy at grid levels, sell at profit targets      |
| `short` | Short-only      | Sell at grid levels, buy back at profit targets |
| `both`  | Both directions | Long and short grids in same price range        |

## Data Requirements

### Auto Data Detection

The system automatically looks for data files in the standard format:

```
data/{exchange}/linear/{symbol}/{interval}/candles.csv
```

Example: `data/bybit/linear/BTCUSDT/5/candles.csv`

### Custom Data Files

Use the `-data` flag to specify custom CSV files with OHLCV format:

```csv
timestamp,open,high,low,close,volume
2023-01-01T00:00:00Z,50000.0,50100.0,49900.0,50050.0,1000000
```

## Examples

### Long Strategy Example

```bash
go run cmd/grid-backtest/main.go -config configs/bybit/grid/btc_long_5m_bybit.json -verbose

Output:
🚀 Grid Backtest v1.0.0
📋 Loading configuration: configs/bybit/grid/btc_long_5m_bybit.json

📊 Grid Strategy Configuration:
   Symbol: BTCUSDT (linear)
   Trading Mode: long
   Price Range: $95000.00 - $110000.00
   Grid Setup: 20 levels, 1.5% spacing, 1.5% profit
   Position: $100.00 size, 10.0x leverage
   Risk: $5000.00 balance, 0.06% commission
   Exchange Constraints: bybit (min: 0.001000, step: 0.001000)

🎯 Grid Levels (20 total):
   Level 1: $95000.00
   Level 2: $96425.00
   ...

📊 Generating comprehensive Excel report: results/grid/BTCUSDT_long_5m/grid_backtest_report.xlsx
✅ Comprehensive report generated successfully!
📁 Report directory: results/grid/BTCUSDT_long_5m
📊 Excel report: grid_backtest_report.xlsx
📄 Text summary: backtest_summary.txt
```

### Performance Report

```
💰 FINANCIAL PERFORMANCE:
   Initial Balance: $5000.00
   Final Balance:   $5150.25
   Total Return:    3.01%
   Realized P&L:    $145.30
   Unrealized P&L:  $4.95

📊 TRADING STATISTICS:
   Total Trades:      12
   Successful Trades: 8
   Win Rate:          66.7%
   Max Concurrent:    3 positions

🎯 GRID METRICS:
   Grid Levels:       20
   Active Grids:      2
```

## Tips

1. **Start Simple** - Begin with long-only strategies before trying both directions
2. **Use Realistic Ranges** - Set grid bounds within reasonable price movements
3. **Enable Exchange Constraints** - For realistic position sizing and order requirements
4. **Monitor Grid Utilization** - Higher utilization often means better performance
5. **Analyze Heat Maps** - Visual reports help identify best-performing grid levels
6. **Use Full Datasets** - By default, all available data is used (~280K candles for comprehensive analysis)
7. **Limit Data for Testing** - Use `-max-candles` flag to test strategies with smaller datasets first

## Troubleshooting

| Issue                               | Solution                                                              |
| ----------------------------------- | --------------------------------------------------------------------- |
| "Configuration file does not exist" | Check file path, use absolute path if needed                          |
| "No data found"                     | Ensure data files exist or let system generate sample data            |
| "Configuration validation failed"   | Check grid bounds, ensure upper > lower bound                         |
| Excel report generation failed      | Install required dependencies, check disk space                       |
| Sample data generated               | Real data file not found, system using synthetic data                 |
| Can't find report files             | Check `results/grid/{symbol}_{mode}_{timeframe}/` directory structure |
