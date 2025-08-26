# 🚀 Enhanced DCA Backtesting Engine

A sophisticated backtesting system for the Enhanced DCA (Dollar-Cost Averaging) strategy with multi-level Take Profit (TP) support, comprehensive analytics, and professional Excel reporting.

## ✨ Features

### 🎯 **Advanced DCA Strategy Testing**

- **Single Take Profit**: Default 4.5% TP system (sells entire position at target)
- **Optional Multi-Level TP**: 5-level TP system available with `-tp-levels` flag
- **DCA Entry Logic**: Automatic averaging down when price drops below thresholds
- **Cycle Management**: Sequential cycle processing with proper balance tracking

### 📊 **Comprehensive Analytics**

- **Performance Metrics**: ROI, Sharpe Ratio, Profit Factor, Max Drawdown
- **Cycle Analysis**: Completion rates, duration tracking, capital efficiency
- **TP Level Statistics**: Success rates and profitability per TP level
- **Risk Assessment**: Drawdown analysis and capital deployment insights

### 📈 **Professional Excel Reporting**

- **4 Detailed Sheets**: Trades, Cycles, Detailed Analysis, Timeline
- **Visual Insights**: Color-coded performance indicators and smart recommendations
- **Balance Tracking**: Accurate capital usage and balance progression per cycle
- **Strategic Recommendations**: AI-powered optimization suggestions

### 🔧 **Flexible Configuration**

- **Multiple Timeframes**: 1m, 5m, 15m, 1h, 4h, 1d support
- **Customizable Parameters**: DCA thresholds, TP levels, position sizing
- **Commission Modeling**: Realistic trading costs and slippage simulation
- **Historical Data**: Comprehensive market data analysis

## 🚀 Quick Start

### Prerequisites

- Go 1.21 or later
- Historical market data (automatically downloaded if needed)

### Basic Usage

```bash
# Navigate to the backtest directory
cd cmd/backtest

# Run a basic backtest
go run main.go -symbol BTCUSDT -interval 5m

# Run with custom date range
go run main.go -symbol SUIUSDT -interval 5m -start "2024-01-01" -end "2024-12-31"

# Use a specific configuration file
go run main.go -config ../../configs/bybit/sui_5m_bybit.json
```

### Advanced Options

```bash
# Enable optimization mode
go run main.go -symbol BTCUSDT -interval 1h -optimize

# Use multi-level TP system (5 levels)
go run main.go -symbol SUIUSDT -interval 5m -tp-levels

# Custom balance and commission
go run main.go -symbol ETHUSDT -interval 15m -balance 5000 -commission 0.001

# Specify output directory
go run main.go -symbol ADAUSDT -interval 1h -output ./my_results
```

## 📋 Command-Line Parameters

| Parameter     | Description             | Default     | Example                |
| ------------- | ----------------------- | ----------- | ---------------------- |
| `-symbol`     | Trading pair symbol     | `BTCUSDT`   | `-symbol SUIUSDT`      |
| `-interval`   | Timeframe for analysis  | `5m`        | `-interval 1h`         |
| `-start`      | Start date (YYYY-MM-DD) | 30 days ago | `-start "2024-01-01"`  |
| `-end`        | End date (YYYY-MM-DD)   | Today       | `-end "2024-12-31"`    |
| `-balance`    | Initial balance         | `500`       | `-balance 10000`       |
| `-commission` | Trading commission rate | `0.0006`    | `-commission 0.001`    |
| `-tp-levels`  | Use 5-level TP system   | `false`     | `-tp-levels`           |
| `-config`     | Configuration file path | -           | `-config config.json`  |
| `-optimize`   | Enable optimization     | `false`     | `-optimize`            |
| `-output`     | Output directory        | `./results` | `-output ./my_results` |

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

- Cycle-organized trade listing
- DCA entries and TP exits clearly separated
- Price change tracking and cumulative costs
- Visual indicators for trade types

#### **🔄 Cycles Sheet**

- **Balance Tracking**: Before/after balance per cycle
- **Capital Usage**: Amount and percentage of balance used
- **Performance**: Duration, ROI, completion status
- **Sequential Analysis**: Proper cycle progression

#### **🚀 Detailed Analysis Sheet**

- **Executive Summary**: Key performance indicators
- **Performance Metrics**: Detailed risk-adjusted returns
- **Cycle Analysis**: Comprehensive cycle statistics
- **TP Level Analysis**: Success rates per TP level
- **Strategic Recommendations**: AI-powered optimization tips

#### **📈 Timeline Sheet**

- Chronological view of all trading activity
- Running balance progression
- Trade-by-trade analysis with timestamps

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

- `tp_percent`: Single TP percentage (default mode)
- **Multi-level TP**: Use `-tp-levels` flag to enable 5-level system
- **TP Levels**: Auto-generated from `tp_percent` (20%, 40%, 60%, 80%, 100%)

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

# Step 2: Use optimized config
go run main.go -config ./results/SUIUSDT_5m/best.json

# Step 3: Test on different timeframes
go run main.go -config ./results/SUIUSDT_5m/best.json -interval 15m
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
