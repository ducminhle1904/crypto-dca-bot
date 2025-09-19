# Market Regime-Based Signal Consensus Guide

## Overview

The market regime system enhances the DCA strategy by dynamically adjusting the required number of indicator confirmations based on current market conditions. This reduces false signals in difficult market conditions while allowing more aggressive entry in favorable conditions.

## How It Works

The system analyzes two key market dimensions:

### 1. **Trend Strength**

- Measures price progression over recent candles
- Requires at least 5% price increase over the analysis period
- Default period: 20 candles

### 2. **Volatility Regime**

- Compares current ATR (Average True Range) to historical percentiles
- Low volatility: ATR below 30th percentile
- High volatility: ATR above 80th percentile
- Default lookback: 50 candles

## Market Regimes

### 🟢 **Favorable Regime** (2 indicators required)

**Conditions:**

- Strong bullish trend (75%+ candles trending up)
- Low volatility environment (ATR < 30th percentile)

**Why less confirmation needed:**

- Market momentum is strong and reliable
- False signals are less common
- Trend continuation is more likely

### 🟡 **Normal Regime** (3 indicators required)

**Conditions:**

- Mixed or moderate trend conditions
- Medium volatility environment
- Default fallback for unclear conditions

**Why moderate confirmation needed:**

- Balanced approach for unclear conditions
- Standard risk/reward ratio

### 🔴 **Hostile Regime** (4 indicators required)

**Conditions:**

- Bearish market (significant price decline)
- High volatility environment (ATR > 80th percentile)
- Choppy, unpredictable price action

**Why more confirmation needed:**

- High false signal probability
- Increased risk of whipsaws
- Need stronger conviction for entries

## Configuration

### Basic Configuration

```json
{
  "market_regime": {
    "enabled": true,
    "favorable_indicators_required": 2,
    "normal_indicators_required": 3,
    "hostile_indicators_required": 4
  }
}
```

### Advanced Configuration

```json
{
  "market_regime": {
    "enabled": true,
    "trend_strength_period": 20,
    "trend_strength_threshold": 0.75,
    "volatility_lookback": 50,
    "low_volatility_percentile": 0.3,
    "high_volatility_percentile": 0.8,
    "favorable_indicators_required": 2,
    "normal_indicators_required": 3,
    "hostile_indicators_required": 4
  }
}
```

## Configuration Parameters

| Parameter                       | Default | Description                                                    |
| ------------------------------- | ------- | -------------------------------------------------------------- |
| `enabled`                       | `false` | Enable/disable market regime detection                         |
| `trend_strength_period`         | `20`    | Candles to analyze for trend strength                          |
| `trend_strength_threshold`      | `0.75`  | Percentage of candles that must be trending for "strong" trend |
| `volatility_lookback`           | `50`    | Historical candles for volatility percentile calculation       |
| `low_volatility_percentile`     | `0.3`   | ATR percentile threshold for low volatility                    |
| `high_volatility_percentile`    | `0.8`   | ATR percentile threshold for high volatility                   |
| `favorable_indicators_required` | `2`     | Indicators needed in favorable conditions                      |
| `normal_indicators_required`    | `3`     | Indicators needed in normal conditions                         |
| `hostile_indicators_required`   | `4`     | Indicators needed in hostile conditions                        |

## Example Scenarios

### Scenario 1: Bull Market Rally

- **Market**: Bitcoin in strong uptrend, low volatility
- **Regime**: Favorable (2/4 indicators required)
- **Result**: More frequent entries during trend

### Scenario 2: Bear Market Decline

- **Market**: Significant price drops, high volatility
- **Regime**: Hostile (4/4 indicators required)
- **Result**: Very selective entries, higher conviction required

### Scenario 3: Sideways Consolidation

- **Market**: Range-bound price action, medium volatility
- **Regime**: Normal (3/4 indicators required)
- **Result**: Balanced approach, standard confirmation

## Integration with Existing Features

The market regime system works seamlessly with:

- **DCA Spacing**: Price thresholds still apply after regime check
- **Dynamic TP**: Take profit calculations remain independent
- **All Indicators**: RSI, MACD, Bollinger Bands, EMA, etc.

## Backward Compatibility

- Market regime is **disabled by default**
- Existing configurations continue working unchanged
- When disabled, uses normal indicator requirements (typically 3)

## Performance Impact

- **Minimal computational overhead**: Simple calculations
- **Stateless operation**: No complex state management
- **Real-time processing**: Results available immediately

## Best Practices

1. **Start Conservative**: Begin with default settings
2. **Monitor Performance**: Track regime distribution in your markets
3. **Adjust Gradually**: Modify thresholds based on market behavior
4. **Combine with Risk Management**: Use alongside position sizing and stop losses

## Troubleshooting

### Issue: Too many hostile regimes detected

**Solution**: Lower `high_volatility_percentile` or `trend_strength_threshold`

### Issue: Too many favorable regimes detected

**Solution**: Increase `trend_strength_threshold` or lower `low_volatility_percentile`

### Issue: No regime changes observed

**Solution**: Check `volatility_lookback` period and ensure sufficient historical data

## Example Output

```
Buy consensus: 2/2 required (Favorable regime)
Insufficient buy consensus: 1/4 required (Hostile regime)
Buy consensus: 3/3 required (Normal regime)
```

## Migration Guide

To enable market regime on existing configurations:

1. Add market regime section to your config file
2. Start with `enabled: true` and default values
3. Monitor performance for several days
4. Adjust parameters based on observed behavior
5. Fine-tune requirements based on your risk tolerance
