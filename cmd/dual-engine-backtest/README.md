# Dual-Engine Backtesting System

The **Dual-Engine Backtesting System** is a comprehensive testing framework for validating the performance, profitability, and robustness of the enhanced DCA bot's dual-engine regime detection system.

## 🎯 **Overview**

This backtesting system tests the integration of:

- **🎯 Regime Detection** - Market condition classification (trending/ranging/volatile/uncertain)
- **🏗️ Grid Engine** - VWAP/EMA anchored hedge grid for ranging markets
- **📈 Trend Engine** - Multi-timeframe trend following for trending markets
- **🔄 Engine Switching** - Automatic engine transitions based on regime changes
- **🛡️ Risk Management** - Transition costs, position evaluation, and safety controls

## 🚀 **Quick Start**

### **Basic Backtest**

```bash
# Run backtest with dual-engine configuration
./dual-engine-backtest -config configs/dual_engine/btc_development.json

# Specific symbol and timeframe
./dual-engine-backtest -config configs/dual_engine/btc_development.json -symbol ETHUSDT -interval 15m

# With date range
./dual-engine-backtest -config configs/dual_engine/btc_development.json -start 2024-01-01 -end 2024-12-31
```

### **Advanced Testing**

```bash
# Validate regime detection accuracy
./dual-engine-backtest -config configs/dual_engine/btc_development.json -validate

# Custom data file
./dual-engine-backtest -config configs/dual_engine/btc_development.json -data custom_data.csv

# Generate specific report formats
./dual-engine-backtest -config configs/dual_engine/btc_development.json -format excel
```

## 📊 **Key Metrics Analyzed**

### **Financial Performance**

- **Total Return** - Overall profitability percentage
- **Annualized Return** - Risk-adjusted annual performance
- **Maximum Drawdown** - Worst peak-to-trough decline
- **Sharpe Ratio** - Risk-adjusted return measure
- **Sortino Ratio** - Downside risk-adjusted returns
- **Value at Risk (VaR)** - Potential losses at 95% confidence

### **Trading Effectiveness**

- **Win Rate** - Percentage of profitable trades
- **Profit Factor** - Gross profit vs gross loss ratio
- **Average Trade Duration** - Time spent in positions
- **Total Trades** - Overall trading activity

### **Regime Detection Accuracy**

- **Regime Classification Success** - How often regimes are correctly identified
- **Regime Change Frequency** - Number of regime switches
- **Average Regime Duration** - Stability of regime detection
- **False Signal Rate** - Percentage of incorrect regime changes

### **Engine Performance**

- **Individual Engine P&L** - Grid vs Trend engine profitability
- **Engine Utilization** - Time spent active per engine
- **Regime Compatibility** - Performance by market condition
- **Engine-Specific Metrics** - Win rates, avg duration per engine

### **Transition Efficiency**

- **Transition Costs** - Total cost of engine switches
- **Transition Success Rate** - Successful vs failed transitions
- **Position Migration Quality** - How well positions transfer between engines
- **Switching Frequency** - Rate of engine changes

## 📁 **Output Reports**

The backtester generates comprehensive reports in multiple formats:

### **📄 JSON Report** (`backtest_results.json`)

```json
{
  "symbol": "BTCUSDT",
  "timeframe": "5m",
  "total_return": 15.75,
  "annualized_return": 45.23,
  "max_drawdown": 8.12,
  "sharpe_ratio": 1.85,
  "win_rate": 68.5,
  "total_trades": 245,
  "regime_changes": 18,
  "total_transitions": 12
}
```

### **📊 CSV Reports**

- `results.csv` - Main performance metrics
- `regime_history.csv` - Detailed regime change log
- `transition_history.csv` - Engine transition records

### **📈 Excel Analysis** (`.txt` format for now)

- Summary performance report
- Risk analysis
- Trade breakdown

## 🔧 **Configuration Options**

| Flag        | Description                               | Example                                    |
| ----------- | ----------------------------------------- | ------------------------------------------ |
| `-config`   | Dual-engine configuration file (required) | `configs/dual_engine/btc_development.json` |
| `-symbol`   | Trading symbol                            | `ETHUSDT`                                  |
| `-interval` | Time interval                             | `5m`, `15m`, `30m`, `1h`                   |
| `-exchange` | Exchange name                             | `bybit`, `binance`                         |
| `-start`    | Start date                                | `2024-01-01`                               |
| `-end`      | End date                                  | `2024-12-31`                               |
| `-output`   | Output directory                          | `backtest_results`                         |
| `-data`     | Custom CSV data file                      | `custom_data.csv`                          |
| `-format`   | Report format                             | `json`, `excel`, `csv`, `all`              |
| `-validate` | Validate regime detection                 |                                            |
| `-verbose`  | Enable verbose logging                    |                                            |

## 📈 **Data Requirements**

### **Standard Data Structure**

The backtester expects historical OHLCV data in the standard format:

```
data/
  bybit/
    linear/
      BTCUSDT/
        5/candles.csv
        15/candles.csv
        30/candles.csv
```

### **CSV Format**

```csv
timestamp,open,high,low,close,volume
2024-01-01 00:00:00,43250.5,43280.0,43200.0,43260.5,1250.75
```

## 🧪 **Testing Scenarios**

### **1. Performance Validation**

```bash
# Test profitability across different market conditions
./dual-engine-backtest -config configs/dual_engine/btc_development.json -start 2024-01-01 -end 2024-12-31
```

### **2. Regime Detection Accuracy**

```bash
# Validate regime classification effectiveness
./dual-engine-backtest -config configs/dual_engine/btc_development.json -validate -verbose
```

### **3. Multi-Symbol Testing**

```bash
# Test across different instruments
./dual-engine-backtest -config configs/dual_engine/btc_development.json -symbol ETHUSDT
./dual-engine-backtest -config configs/dual_engine/btc_development.json -symbol SOLUSDT
```

### **4. Timeframe Analysis**

```bash
# Compare performance across timeframes
./dual-engine-backtest -config configs/dual_engine/btc_development.json -interval 5m
./dual-engine-backtest -config configs/dual_engine/btc_development.json -interval 15m
./dual-engine-backtest -config configs/dual_engine/btc_development.json -interval 30m
```

## 📊 **Interpreting Results**

### **🟢 Good Performance Indicators**

- **Total Return > 15%** annually
- **Sharpe Ratio > 1.5**
- **Max Drawdown < 10%**
- **Win Rate > 60%**
- **Regime Accuracy > 75%**
- **Transition Costs < 1%** of portfolio

### **🟡 Areas for Optimization**

- **High transition frequency** (>1 per day)
- **Low engine utilization** (<30% for either engine)
- **Regime flip-flopping** (rapid regime changes)
- **Poor engine-specific performance**

### **🔴 Warning Signs**

- **Negative returns** or poor risk metrics
- **Excessive drawdown** (>20%)
- **Very low win rate** (<40%)
- **High transition costs** (>2% portfolio)
- **Poor regime detection** (<60% accuracy)

## 🛠️ **Troubleshooting**

### **Common Issues**

1. **No data available**

   ```bash
   # Download historical data first
   go run scripts/download_bybit_historical_data.go -symbol BTCUSDT -interval 5m
   ```

2. **Configuration errors**

   ```bash
   # Validate configuration syntax
   ./dual-engine-backtest -config configs/dual_engine/btc_development.json -help
   ```

3. **Memory issues with large datasets**
   ```bash
   # Use date ranges to limit data size
   ./dual-engine-backtest -config configs/dual_engine/btc_development.json -start 2024-06-01 -end 2024-12-31
   ```

## 🎯 **Success Criteria**

A **profitable and robust** dual-engine system should demonstrate:

✅ **Financial Excellence**

- Positive risk-adjusted returns (Sharpe > 1.0)
- Controlled drawdowns (< 15%)
- Consistent profitability across time periods

✅ **Regime Detection Quality**

- High classification accuracy (> 75%)
- Stable regime durations (> 2 hours average)
- Low false signal rate (< 20%)

✅ **Engine Coordination**

- Effective engine utilization (both engines active 30%+ of time)
- Successful transitions (> 80% success rate)
- Low switching costs (< 0.5% per transition)

✅ **Risk Management**

- Proper position sizing
- Effective stop losses
- Emergency controls functioning

## 🚀 **Next Steps**

After successful backtesting:

1. **📈 Paper Trading** - Test with live market data
2. **🔧 Parameter Optimization** - Fine-tune based on results
3. **📊 Multi-Asset Testing** - Expand to other instruments
4. **🏭 Production Deployment** - Deploy with real capital

---

## 💡 **Pro Tips**

- **Start with shorter time periods** to validate logic before full backtests
- **Compare against benchmark** (buy & hold) to assess value-add
- **Test in different market conditions** (bull, bear, sideways)
- **Validate with out-of-sample data** to avoid overfitting
- **Monitor transition costs** - they can erode profits quickly
- **Use regime validation** to ensure detection accuracy

**Happy backtesting! 🧪📈**
