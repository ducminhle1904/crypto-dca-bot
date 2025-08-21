# Hedge Backtest System

A specialized backtest engine designed specifically for dual-position hedging strategies that open both LONG and SHORT positions simultaneously.

## Overview

The Hedge Backtest System provides dedicated tools for testing hedging strategies that:

- ✅ Open both LONG and SHORT positions at the same entry points
- ✅ Track each position independently with separate P&L
- ✅ Provide hedge-specific metrics and analysis
- ✅ Advanced risk management for both sides

## Key Features

### 🎯 **Hedge-Specific Metrics**

- **Hedge Efficiency**: How well short positions offset long losses
- **Net Exposure**: Average net position exposure (long - short)
- **Volatility Capture**: Total profit from both directions
- **Side-by-Side Performance**: Separate win rates for longs/shorts

### 🛡️ **Advanced Risk Management**

- Individual stop loss and take profit for each position
- Trailing stops with dynamic adjustment
- Maximum drawdown limits per position
- Position size limits and volatility thresholds

### 📊 **Detailed Position Tracking**

- Real-time P&L for each LONG/SHORT position
- Maximum profit/loss tracking per position
- Position duration and exit reason logging
- Commission tracking per side

## Usage

### Basic Command

```bash
./bin/hedge-backtest.exe -hedge-ratio=0.6 -base-amount=200 -balance=2000
```

### Advanced Configuration

```bash
./bin/hedge-backtest.exe \
  -hedge-ratio=0.7 \
  -base-amount=150 \
  -balance=2000 \
  -stop-loss=0.04 \
  -take-profit=0.025 \
  -trailing-stop=0.015 \
  -volatility-threshold=0.012 \
  -time-between-entries=20
```

## Configuration Parameters

### Core Strategy Parameters

- `hedge-ratio`: Ratio of short to long positions (0.0=only long, 0.5=equal, 1.0=only short)
- `base-amount`: Base hedge amount per position
- `balance`: Initial balance for backtest
- `commission`: Trading commission percentage

### Exchange Integration

The system automatically fetches real trading constraints:

- **Minimum Order Quantity**: Retrieved from Bybit API for realistic position sizing
- **API Credentials**: Set `BYBIT_API_KEY` and `BYBIT_API_SECRET` environment variables
- **Fallback**: Uses default 0.01 if API credentials unavailable

```bash
# Set your Bybit API credentials (optional but recommended)
export BYBIT_API_KEY="your_api_key"
export BYBIT_API_SECRET="your_secret"

# Or create a .env file
echo "BYBIT_API_KEY=your_api_key" > .env
echo "BYBIT_API_SECRET=your_secret" >> .env
```

### Risk Management

- `stop-loss`: Stop loss percentage (default: 5%)
- `take-profit`: Take profit percentage (default: 3%)
- `trailing-stop`: Trailing stop percentage (default: 2%)
- `max-drawdown`: Maximum drawdown per position (default: 10%)
- `volatility-threshold`: Minimum volatility to open positions (default: 1.5%)
- `time-between-entries`: Minimum minutes between entries (default: 15)

### Advanced Indicators (Fixed Set)

The hedge backtest uses a curated set of advanced indicators:

- **Hull Moving Average**: Trend detection
- **Money Flow Index (MFI)**: Volume-weighted momentum
- **Keltner Channels**: Volatility-based signals
- **WaveTrend**: Advanced oscillator

#### Indicator Parameters

- `hull-ma-period`: Hull MA period (default: 20)
- `mfi-period`: MFI period (default: 14)
- `mfi-oversold/overbought`: MFI signal levels (default: 20/80)
- `keltner-period`: Keltner Channels period (default: 20)
- `keltner-multiplier`: Keltner multiplier (default: 2.0)
- `wavetrend-n1/n2`: WaveTrend parameters (default: 10/21)
- `wavetrend-oversold/overbought`: WaveTrend levels (default: -60/60)

## Configuration File

Create `configs/hedge_btc_advanced.json`:

```json
{
  "symbol": "BTCUSDT",
  "initial_balance": 2000.0,
  "commission": 0.0005,
  "base_amount": 200.0,
  "hedge_ratio": 0.6,
  "stop_loss_pct": 0.04,
  "take_profit_pct": 0.025,
  "trailing_stop_pct": 0.015,
  "max_drawdown_pct": 0.08,
  "volatility_threshold": 0.012,
  "time_between_entries": 20
}
```

## Output Metrics

### Performance Summary

```
💰 Initial Balance:    $2000.00
💰 Final Balance:      $2247.35
📈 Total Return:       12.37%
📉 Max Drawdown:       8.42%
```

### Position Statistics

```
🎯 Total Positions:    24
📈 Long Positions:     12 (Win Rate: 66.7%)
📉 Short Positions:    12 (Win Rate: 58.3%)
```

### Hedge-Specific Metrics

```
🔄 Hedge Efficiency:   78.45% (how well shorts offset long losses)
⚖️  Avg Net Exposure:   $156.78
🎯 Volatility Capture: $247.35
```

### Risk Analysis

```
📈 Largest Long Win:   $34.52
📈 Largest Long Loss:  -$18.34
📉 Largest Short Win:  $28.91
📉 Largest Short Loss: -$22.17
```

## Strategy Logic

### Market Regime Adaptation

1. **Trending Markets**: Reduces hedge ratio for more directional exposure
2. **Ranging Markets**: Increases hedge ratio for neutral positioning
3. **Volatile Markets**: Equal positions with reduced size and tight stops

### Entry Conditions

- All 4 indicators must provide signal consensus
- Minimum volatility threshold must be met
- Time between entries must be respected
- Sufficient balance for both positions required

### Exit Conditions

- Stop loss or take profit hit
- Maximum drawdown exceeded
- Trailing stop triggered
- End of backtest period

## Differences from Standard DCA Backtest

| Feature            | Standard DCA     | Hedge Backtest        |
| ------------------ | ---------------- | --------------------- |
| Position Types     | LONG only        | LONG + SHORT          |
| Entry Logic        | Single direction | Dual direction        |
| P&L Tracking       | Combined         | Separate per side     |
| Risk Management    | Single position  | Per-position          |
| Metrics            | Return-focused   | Hedge-focused         |
| Take-Profit Cycles | Yes              | No (individual exits) |

## Building and Running

### Build

```bash
go build -o bin/hedge-backtest.exe cmd/hedge-backtest/main.go
```

### Run with Sample Data

```bash
./bin/hedge-backtest.exe -hedge-ratio=0.6 -base-amount=200
```

The system will automatically generate sample data if no data file is specified.

## Optimization

The hedge backtest supports genetic algorithm optimization to find optimal parameters for your hedging strategy.

### Quick Optimization

```bash
./bin/hedge-backtest.exe -optimize=true -objective=balanced -generations=15 -population=25
```

### Full Optimization

```bash
./bin/hedge-backtest.exe -optimize=true \
  -objective=hedge-efficiency \
  -generations=25 \
  -population=40 \
  -balance=2000
```

### Optimization Objectives

Choose what to optimize for using the `-objective` parameter:

- **`balanced`** (default): Multi-objective optimization considering return, hedge efficiency, and drawdown
- **`hedge-efficiency`**: Maximize how well short positions offset long losses
- **`return`**: Maximize total return with drawdown penalty
- **`sharpe`**: Maximize risk-adjusted return (return/drawdown ratio)
- **`volatility-capture`**: Maximize absolute profit from market volatility

### Optimization Parameters

- **`-optimize`**: Enable optimization mode
- **`-objective`**: Optimization objective (see above)
- **`-generations`**: Number of GA generations (default: 25)
- **`-population`**: Population size (default: 40)

### What Gets Optimized

The genetic algorithm optimizes these parameters:

#### Core Strategy Parameters

- Base amount per position (dynamic based on minimum order quantity)
- Hedge ratio (0.2-0.9)
- Stop loss percentage (2-8%)
- Take profit percentage (1.5-5%)
- Trailing stop percentage (1-4%)
- Max drawdown threshold (5-15%)
- Volatility threshold (0.5-2.5%)
- Time between entries (10-60 minutes)

#### Smart Minimum Order Quantity Integration

The system automatically fetches the actual minimum order quantity from the exchange and adjusts optimization bounds accordingly:

```
🔧 Updated BaseAmountMin to $50.00 based on MinOrderQty=0.010000 BTCUSDT (est. price $50000.00)
```

**How it works:**

1. **Fetches real constraints** from Bybit API using your credentials
2. **Estimates price** based on symbol (BTC~$50k, ETH~$3k, others~$100)
3. **Calculates minimum order value** (minOrderQty × estimatedPrice)
4. **Sets BaseAmountMin** to 10× minimum order value for meaningful positions
5. **Ensures realistic bounds** (minimum $10, maximum 10% of BaseAmountMax)

**Examples:**

- BTCUSDT (0.001 min qty): BaseAmountMin = $500 (0.001 × $50,000 × 10)
- ETHUSDT (0.01 min qty): BaseAmountMin = $300 (0.01 × $3,000 × 10)
- ADAUSDT (1.0 min qty): BaseAmountMin = $10 (1.0 × $1 × 10, capped at $10 minimum)

#### Indicator Parameters

- Hull MA period (10-40)
- MFI period and levels (8-25, 15-35/65-85)
- Keltner Channels period and multiplier (10-35, 1.5-3.0)
- WaveTrend parameters and levels (6-15/15-30, -80/-40/40/80)

### Optimization Output

```
🧬 HEDGE OPTIMIZATION RESULTS
============================================================
🎯 Objective: Hedge Efficiency
⏱️ Duration: 2m34s
🏆 Best Fitness: 0.8234

OPTIMAL PARAMETERS
----------------------------------------
💰 Base Amount: $187.50
🔄 Hedge Ratio: 0.675
🛑 Stop Loss: 3.8%
🎯 Take Profit: 2.4%
📈 Trailing Stop: 1.7%
⚠️ Max Drawdown: 7.2%
📊 Volatility Threshold: 1.3%
⏰ Time Between Entries: 23 min

INDICATOR PARAMETERS
------------------------------
Hull MA Period: 18
MFI Period: 12 (22.5/76.8)
Keltner: 16 (2.15)
WaveTrend: 8/19 (-62.3/58.7)

PERFORMANCE RESULTS
----------------------------------------
💰 Return: 14.7%
📉 Max Drawdown: 6.8%
🔄 Hedge Efficiency: 82.3%
🎯 Volatility Capture: $294.50
📊 Total Positions: 28 (14 Long, 14 Short)
✅ Win Rates: Long 71.4%, Short 64.3%
```

## Integration

The hedge backtest system:

- ✅ Uses the same `DualPositionStrategy` from the main system
- ✅ Independent from standard DCA backtests
- ✅ Can run alongside existing backtest tools
- ✅ Shares indicator implementations

## Best Practices

### Recommended Settings

- **Conservative**: hedge-ratio=0.8, stop-loss=0.03, take-profit=0.02
- **Balanced**: hedge-ratio=0.6, stop-loss=0.04, take-profit=0.025
- **Aggressive**: hedge-ratio=0.4, stop-loss=0.05, take-profit=0.03

### Market Conditions

- **Bull Markets**: Lower hedge ratio (0.3-0.5) for long bias
- **Bear Markets**: Higher hedge ratio (0.7-0.9) for short bias
- **Sideways Markets**: Balanced ratio (0.5-0.6) for volatility capture

### Risk Management

- Start with smaller base amounts to test strategy
- Monitor hedge efficiency - aim for >60%
- Keep net exposure reasonable relative to balance
- Adjust parameters based on market volatility

## Troubleshooting

### Common Issues

1. **No positions opened**: Check volatility threshold and indicator parameters
2. **High drawdown**: Reduce position sizes or tighten stop losses
3. **Low hedge efficiency**: Adjust hedge ratio or entry timing
4. **Compilation errors**: Ensure all dependencies are installed

### Performance Tips

- Use appropriate window size (100+ for stable signals)
- Ensure sufficient historical data (1000+ bars)
- Test different time between entries (15-60 minutes)
- Monitor commission impact on small positions
- **Set API credentials** for realistic minimum order quantity bounds
- **Test with actual symbols** you plan to trade for accurate constraints

### API Credentials Setup

For optimal optimization bounds, set up Bybit API credentials:

1. **Create API Key** on [Bybit](https://www.bybit.com/app/user/api-management)
2. **Set Environment Variables**:

   ```bash
   # Windows PowerShell
   $env:BYBIT_API_KEY="your_api_key"
   $env:BYBIT_API_SECRET="your_secret"

   # Linux/Mac
   export BYBIT_API_KEY="your_api_key"
   export BYBIT_API_SECRET="your_secret"
   ```

3. **Or Create .env File**:
   ```bash
   echo "BYBIT_API_KEY=your_api_key" > .env
   echo "BYBIT_API_SECRET=your_secret" >> .env
   ```

**Note**: The system uses demo mode for fetching instrument info, so it's safe even with live API keys.

---

_This hedge backtest system provides professional-grade tools for testing dual-position strategies with comprehensive risk management and hedge-specific analytics._
