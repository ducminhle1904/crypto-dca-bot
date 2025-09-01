# 🚀 Enhanced DCA Backtesting Engine

A sophisticated backtesting system for the Enhanced DCA (Dollar-Cost Averaging) strategy with optional multi-level Take Profit (TP), robust walk-forward validation, comprehensive analytics, and professional Excel reporting.

## ✨ Features

### 🎯 **Advanced DCA Strategy Testing**

- **Multi-Level Take Profit**: 5-level TP system (always enabled by default)
- **Dynamic TP Levels**: Automatically derived from base TP percentage (20%, 40%, 60%, 80%, 100%)
- **DCA Entry Logic**: Automatic averaging down when price drops below thresholds
- **Cycle Management**: Sequential cycle processing with proper balance tracking
- **Dynamic TP Filling**: If a candle reaches multiple TP levels, they are combined into one exit (no artificial splits)

### 📊 **Comprehensive Analytics**

- **Performance Metrics**: ROI, Sharpe Ratio, Profit Factor, Max Drawdown
- **Cycle Analysis**: Completion rates, duration tracking, capital efficiency
- **TP Level Statistics**: Success rates and profitability per TP level
- **Risk Assessment**: Drawdown analysis and capital deployment insights

### 📈 **Professional Excel Reporting**

- **3 Detailed Sheets**: Trades, Cycles, Detailed Analysis (Timeline removed)
- **Visual Insights**: Color-coded price changes (entries in red, exits in green)
- **Balance Tracking**: Running balance and balance change per event; per-cycle start/end balance
- **TP Info**: Combined TP execution details shown inline in Trades

### 🔧 **Flexible Configuration**

- **Multiple Timeframes**: 1m, 5m, 15m, 1h, 4h, 1d support
- **Customizable Parameters**: DCA thresholds, TP levels, position sizing
- **Commission Modeling**: Realistic trading costs
- **Historical Data**: Comprehensive market data analysis

### 🧪 **Walk-Forward Validation (WFV)**

- **Holdout mode**: Split by ratio (e.g., 70% train, 30% test)
- **Rolling mode**: Sliding train/test windows across time (e.g., 180d train, 60d test, 30d roll)
- **Train/Test Reporting**: Optimize on train, report on unseen test

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or later
- Historical market data (automatically downloaded if needed)

### Basic Usage

```bash
# Navigate to the backtest directory
cd cmd/backtest

# Run a backtest (auto-resolves data from data/<exchange>/<category>/<symbol>/<interval>/candles.csv)
go run main.go -symbol BTCUSDT -interval 5m

# Limit to a trailing window (e.g., last 90 days)
go run main.go -symbol SUIUSDT -interval 5m -period 90d

# Use a specific data file (forces single-interval run)
go run main.go -data "../../data/bybit/linear/BTCUSDT/60/candles.csv"

# Use a specific configuration file
go run main.go -config ../../configs/bybit/sui_5m_bybit.json
```

### Advanced Options

```bash
# Enable optimization mode (GA)
go run main.go -symbol BTCUSDT -interval 1h -optimize

# Multi-level TP system is always enabled (5 levels, 20% each)
go run main.go -symbol SUIUSDT -interval 5m

# Custom balance and commission
go run main.go -symbol ETHUSDT -interval 15m -balance 5000 -commission 0.0005

# Run across all available intervals for a symbol
go run main.go -symbol BTCUSDT -all-intervals -optimize

# Walk-Forward Validation (holdout)
go run main.go -symbol BTCUSDT -interval 1h -optimize -wf-enable -wf-split-ratio 0.7

# Walk-Forward Validation (rolling)
go run main.go -symbol BTCUSDT -interval 1h -optimize -wf-enable -wf-rolling -wf-train-days 180 -wf-test-days 60 -wf-roll-days 30

# Console-only (skip file output)
go run main.go -symbol ADAUSDT -interval 1h -console-only
```

## 📋 Command-Line Parameters

| Parameter          | Description                                                                    | Default   | Example                                                  |
| ------------------ | ------------------------------------------------------------------------------ | --------- | -------------------------------------------------------- |
| `-config`          | Path to configuration file                                                     |           | `-config ../../configs/bybit/sui_5m_bybit.json`          |
| `-data`            | Path to historical data file (overrides `-interval`)                           |           | `-data "../../data/bybit/linear/BTCUSDT/60/candles.csv"` |
| `-data-root`       | Root folder containing `<EXCHANGE>/<CATEGORY>/<SYMBOL>/<INTERVAL>/candles.csv` | `data`    | `-data-root ./data`                                      |
| `-symbol`          | Trading symbol                                                                 | `BTCUSDT` | `-symbol SUIUSDT`                                        |
| `-interval`        | Data interval (e.g., `5m`, `1h`, `4h`, `1d`)                                   | `1h`      | `-interval 5m`                                           |
| `-exchange`        | Exchange for data discovery                                                    | `bybit`   | `-exchange binance`                                      |
| `-balance`         | Initial balance                                                                | `500`     | `-balance 10000`                                         |
| `-commission`      | Trading fee (quote currency)                                                   | `0.0005`  | `-commission 0.001`                                      |
| `-window`          | Indicator/window size for analysis                                             | `100`     | `-window 200`                                            |
| `-base-amount`     | Base DCA order size                                                            | `40`      | `-base-amount 20`                                        |
| `-max-multiplier`  | Maximum position multiplier                                                    | `3`       | `-max-multiplier 4`                                      |
| `-price-threshold` | Min price drop % to add next DCA                                               | `0.02`    | `-price-threshold 0.015`                                 |
| `-advanced-combo`  | Use Hull/MFI/Keltner/WaveTrend instead of RSI/MACD/BB/EMA                      | `false`   | `-advanced-combo`                                        |

| `-optimize` | Enable GA optimization | `false` | `-optimize` |
| `-all-intervals` | Discover and run all available intervals for the symbol | `false` | `-all-intervals` |
| `-period` | Trailing window (e.g., `30d`, `180d`, `365d`, or `Nd`) | | `-period 90d` |
| `-console-only` | Print results to console only (no files) | `false` | `-console-only` |
| `-wf-enable` | Enable walk-forward validation | `false` | `-wf-enable` |
| `-wf-split-ratio` | Train/test split ratio (holdout) | `0.7` | `-wf-split-ratio 0.8` |
| `-wf-rolling` | Rolling WFV instead of holdout | `false` | `-wf-rolling` |
| `-wf-train-days` | Training window size (rolling) | `180` | `-wf-train-days 120` |
| `-wf-test-days` | Test window size (rolling) | `60` | `-wf-test-days 30` |
| `-wf-roll-days` | Roll step size (rolling) | `30` | `-wf-roll-days 15` |
| `-env` | Environment file for Bybit credentials | `.env` | `-env .env.local` |

## 📊 Output Files

### 1. **Console Output**

```
🎯 BACKTEST RESULTS for SUIUSDT (5m)
═══════════════════════════════════════════════════════════════

📈 PERFORMANCE METRICS
Initial Balance: $500.00
Final Balance: $2,505.88
Total Return: 401.18%
Total P&L: $2,005.88

🎯 TRADING STATISTICS
Total Trades: 735
Winning Trades: 735 (100.0%)
Losing Trades: 0 (0.0%)
Win Rate: 100.0%

📊 RISK METRICS
Max Drawdown: 2.15%
Sharpe Ratio: 15.23
Profit Factor: +Inf

🔄 CYCLE ANALYSIS
Total Cycles: 147
Completed Cycles: 147 (100.0%)
Avg Cycle Duration: 49.2 hours
```

### 2. **Excel Report (`trades.xlsx`)**

#### **📈 Trades Sheet**

- Chronological events per cycle with a single unified view
- Columns: Cycle, Type, Sequence, Timestamp, Price, Quantity, Commission, PnL, Price Change %, Running Balance, Balance Change, TP Info
- Price Change %: red for entries (down-moves), green for exits (profits)
- Running Balance updates on every event; Balance Change shows delta per row
- Cycle header and cycle summary are seamless, summary includes start/end balance and cycle profit
- TP Info shows combined TP fills when multiple levels are hit in the same candle

#### **🔄 Cycles Sheet**

- **Balance Tracking**: Before/after balance per cycle
- **Capital Usage**: Amount and percentage of balance used
- **Performance**: Duration, ROI
- Status column removed; improved per-cycle summaries

#### **🚀 Detailed Analysis Sheet**

- **Executive Summary**: Key performance indicators
- **Performance Metrics**: Detailed risk-adjusted returns
- **Cycle Analysis**: Comprehensive cycle statistics
- **TP Level Analysis**: Success rates per TP level
- **Strategic Recommendations**: AI-powered optimization tips

> Note: The previous Timeline sheet has been removed. Its information is now consolidated in the Trades sheet (sequence, running balance, and cycle summaries).

### 3. **Configuration Files**

- `best.json`: Optimized parameters (if using `-optimize`)
- `backtest_config.json`: Used configuration snapshot

## 🎯 Strategy Configuration

### Sample Configuration (`config.json`)

```json
{
  "strategy": {
    "symbol": "SUIUSDT",
    "base_amount": 40,
    "max_multiplier": 4,
    "price_threshold": 0.01,
    "interval": "5m",
    "tp_percent": 0.045,
    "cycle": true,
    "indicators": ["hull_ma", "mfi", "keltner", "wavetrend"],
    "use_advanced_combo": true
  },
  "exchange": {
    "name": "bybit",
    "bybit": {
      "api_key": "${BYBIT_API_KEY}",
      "api_secret": "${BYBIT_API_SECRET}",
      "testnet": false,
      "demo": true
    }
  },
  "risk": {
    "initial_balance": 500,
    "commission": 0.0005,
    "min_order_qty": 10
  }
}
```

### Key Parameters Explained

#### **DCA Settings**

- `base_amount`: Base investment per DCA entry
- `max_multiplier`: Maximum position size multiplier
- `price_threshold`: Price drop % to trigger next DCA

#### **Take Profit Settings**

- `tp_percent`: Base TP percentage for multi-level system (always enabled)
- **5-Level TP System**: Automatically generates 5 levels from base TP (20%, 40%, 60%, 80%, 100%)
- **Dynamic TP Filling**: If a candle reaches multiple TP thresholds, levels are combined into a single exit

#### **Risk Management**

- `initial_balance`: Starting capital
- `commission`: Trading fee percentage
- `min_order_quantity`: Minimum order size

## 📈 Understanding the Results

### **Performance Metrics**

#### **Return Metrics**

- **Total Return**: Overall percentage gain/loss
- **ROI**: Return on invested capital
- **Profit Factor**: Gross profits ÷ Gross losses

#### **Risk Metrics**

- **Max Drawdown**: Largest peak-to-trough decline
- **Sharpe Ratio**: Risk-adjusted return measurement
- **Win Rate**: Percentage of profitable trades

### **Cycle Analysis**

#### **Cycle Lifecycle**

1. **Start**: Price drops, DCA entry triggered
2. **Averaging**: Additional DCA entries on further drops
3. **Recovery**: Price rises, TP levels start hitting
4. **Completion**: All TP levels hit, cycle closes
5. **Next Cycle**: New cycle starts with updated balance

#### **Capital Efficiency**

- **Capital %**: Percentage of available balance used per cycle
- **Balance Growth**: How balance increases with successful cycles
- **Compound Effect**: Larger positions possible as balance grows

### **TP Level Performance**

- **Hit Rate**: How often each TP level is reached
- **Profitability**: Average profit per TP level
- **Success Patterns**: Which levels are most reliable

## 🔧 Optimization Tips

### **Strategy Optimization**

1. **Analyze TP Success Rates**: Adjust levels based on hit rates
2. **Monitor Capital Usage**: Optimize position sizing
3. **Review Cycle Duration**: Balance frequency vs profitability
4. **Check Market Conditions**: Test across different market phases

### **Parameter Tuning**

- **Conservative**: Lower DCA amounts, wider TP levels
- **Aggressive**: Higher DCA amounts, tighter TP levels
- **Balanced**: Medium settings with good risk/reward ratio

### **Risk Management**

- **Drawdown Control**: Limit position sizes if drawdown > 15%
- **Capital Allocation**: Don't use more than 70% of balance per cycle
- **Market Adaptation**: Adjust parameters for different volatility periods

## 🚨 Important Notes

### **Backtesting Limitations**

- **Historical Performance**: Past results don't guarantee future performance
- **Market Conditions**: Strategy may perform differently in various market phases
- **Slippage**: Real trading may have additional costs not modeled
- **Liquidity**: Large orders may face execution challenges

### **Best Practices**

1. **Test Multiple Timeframes**: Validate across different intervals
2. **Use Recent Data**: Focus on recent market conditions
3. **Paper Trade First**: Test with demo accounts before live trading
4. **Monitor Performance**: Regularly review and adjust parameters
5. **Risk Management**: Never risk more than you can afford to lose

## 📚 Advanced Usage

### **Batch Testing**

```bash
# Test multiple symbols
for symbol in BTCUSDT ETHUSDT ADAUSDT SUIUSDT; do
  go run main.go -symbol $symbol -interval 5m -output ./results/$symbol
done
```

### **Optimization Workflow**

```bash
# Step 1: Run optimization
go run main.go -symbol SUIUSDT -interval 5m -optimize

# Step 2: Use optimized config (best.json)
go run main.go -config ./results/SUIUSDT_5/best.json

# Step 3: Test on different timeframes
go run main.go -config ./results/SUIUSDT_5/best.json -interval 15m
```

### **Walk-Forward Validation**

```bash
# Holdout WFV (70/30 split)
go run main.go -symbol BTCUSDT -interval 1h -optimize -wf-enable -wf-split-ratio 0.7

# Rolling WFV (180d train, 60d test, 30d roll)
go run main.go -symbol BTCUSDT -interval 1h -optimize -wf-enable -wf-rolling -wf-train-days 180 -wf-test-days 60 -wf-roll-days 30

# Combine with all-intervals scanning
go run main.go -symbol BTCUSDT -all-intervals -optimize -wf-enable
```

### **Performance Analysis**

1. **Review Excel Reports**: Focus on Detailed Analysis sheet
2. **Check Cycle Patterns**: Look for consistent performance
3. **Analyze TP Distribution**: Optimize level percentages
4. **Monitor Risk Metrics**: Ensure acceptable drawdown levels

## 🤝 Contributing

Contributions are welcome! Please feel free to submit issues, feature requests, or pull requests.

## ⚠️ Disclaimer

This backtesting tool is for educational and research purposes only. Cryptocurrency trading involves substantial risk of loss. Always:

- Test thoroughly before live trading
- Use proper risk management
- Never invest more than you can afford to lose
- Consider market conditions and volatility
- Seek professional financial advice if needed

---

**Happy Backtesting! 🚀**

_For more information, visit the main repository or check the live trading documentation._
