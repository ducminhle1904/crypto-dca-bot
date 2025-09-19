# DCA Backtest Command

The DCA Backtest command provides a clean, focused interface specifically designed for Dollar Cost Averaging (DCA) strategy backtesting and optimization with advanced features including dynamic take profit and adaptive DCA spacing.

## Overview

This command provides a clean, focused interface for DCA strategy backtesting and optimization:

- **Clean Interface**: DCA-specific command line interface
- **Focused Functionality**: Streamlined for DCA strategies only
- **Orchestrator Integration**: Uses the modular orchestrator system
- **Modern CLI**: Enhanced help, validation, and user experience
- **Advanced Features**: Dynamic take profit and adaptive DCA spacing strategies

## Features

### Core Functionality

- ✅ **Single Backtests**: Test DCA strategy on historical data
- ✅ **Parameter Optimization**: Genetic algorithm optimization
- ✅ **Multi-Interval Analysis**: Test across all available timeframes
- ✅ **Walk-Forward Validation**: Robust strategy validation
- ✅ **12 Technical Indicators**: Complete indicator library
- ✅ **Dynamic Take Profit**: Adaptive TP based on market conditions
- ✅ **Adaptive DCA Spacing**: Volatility-based entry timing

### Advanced Features

- ✅ **Dynamic Take Profit Strategies**:

  - Volatility-adaptive TP (ATR-based)
  - Indicator-based TP (signal strength)
  - Fixed multi-level TP (5 levels)

- ✅ **DCA Spacing Strategies**:
  - Fixed progressive spacing
  - Volatility-adaptive spacing (ATR-based)

### Output Formats

- ✅ **Console Output**: Formatted results with context
- ✅ **Excel Reports**: Detailed trade analysis with charts
- ✅ **JSON Config**: Optimized parameters for reuse
- ✅ **CSV Export**: Raw trade data

## Quick Start

### Basic Usage

```bash
# Simple backtest with enhanced Bollinger Bands
dca-backtest -symbol BTCUSDT -interval 1h -indicators "bb"

# Multi-indicator backtest
dca-backtest -symbol ETHUSDT -indicators "rsi,macd,bb,stochrsi"

# Load from configuration file
dca-backtest -config configs/bybit/dca/hype_5m_bybit.json

# Optimize indicator parameters
dca-backtest -symbol SUIUSDT -indicators "bb,stochrsi,obv" -optimize

# Test all intervals
dca-backtest -symbol BTCUSDT -indicators "rsi,bb" -all-intervals
```

### Advanced Usage with New Features

```bash
# Dynamic take profit with volatility adaptation
dca-backtest -symbol BTCUSDT -indicators "hull_ma,stochrsi,keltner" -dynamic-tp volatility_adaptive -tp-volatility-mult 0.5

# Indicator-based dynamic TP with custom weights
dca-backtest -symbol ETHUSDT -indicators "rsi,macd,bb,obv" -dynamic-tp indicator_based -tp-indicator-weights "rsi:0.4,macd:0.3,bb:0.3"

# Adaptive DCA spacing with volatility sensitivity
dca-backtest -symbol ADAUSDT -indicators "supertrend,mfi,keltner" -dca-spacing volatility_adaptive -spacing-sensitivity 2.0

# Combined advanced features
dca-backtest -symbol BTCUSDT -indicators "hull_ma,stochrsi,keltner" -dynamic-tp volatility_adaptive -dca-spacing volatility_adaptive -optimize

# Walk-forward validation with advanced features
dca-backtest -symbol BTCUSDT -indicators "supertrend,keltner,obv" -dynamic-tp indicator_based -dca-spacing volatility_adaptive -optimize -wf-enable -wf-rolling

# Custom TP bounds with dynamic strategy
dca-backtest -symbol ETHUSDT -dynamic-tp volatility_adaptive -tp-min-percent 0.005 -tp-max-percent 0.08 -tp-volatility-mult 0.8
```

## Configuration

### DCA Strategy Parameters

| Parameter        | Default | Description                 |
| ---------------- | ------- | --------------------------- |
| `base-amount`    | 40      | Base DCA investment amount  |
| `max-multiplier` | 3.0     | Maximum position multiplier |

### DCA Spacing Strategy

| Parameter             | Default | Description                                       |
| --------------------- | ------- | ------------------------------------------------- |
| `dca-spacing`         | fixed   | DCA spacing strategy (fixed, volatility_adaptive) |
| `spacing-threshold`   | 0.01    | Base threshold for DCA spacing (1%)               |
| `spacing-multiplier`  | 1.15    | Multiplier for fixed progressive spacing          |
| `spacing-sensitivity` | 1.8     | Volatility sensitivity for adaptive spacing       |
| `spacing-atr-period`  | 14      | ATR period for adaptive spacing                   |

### Take Profit Configuration

| Parameter       | Default | Description                             |
| --------------- | ------- | --------------------------------------- |
| `tp-percent`    | 0.02    | Base take profit percentage (2%)        |
| `use-tp-levels` | true    | Enable multi-level TP system (5 levels) |

### Dynamic Take Profit Strategy

| Parameter              | Default | Description                                                       |
| ---------------------- | ------- | ----------------------------------------------------------------- |
| `dynamic-tp`           | fixed   | Dynamic TP strategy (fixed, volatility_adaptive, indicator_based) |
| `tp-volatility-mult`   | 0.5     | Volatility multiplier for dynamic TP                              |
| `tp-min-percent`       | 0.01    | Minimum TP percentage (1%)                                        |
| `tp-max-percent`       | 0.05    | Maximum TP percentage (5%)                                        |
| `tp-strength-mult`     | 0.3     | Signal strength multiplier for indicator-based TP                 |
| `tp-indicator-weights` | ""      | Comma-separated indicator:weight pairs                            |

### Available Indicators (12 Total)

**Trend Indicators (4)**:

- **SMA** (Simple Moving Average) - `sma` - Basic trend following
- **EMA** (Exponential Moving Average) - `ema` - Responsive trend direction
- **Hull MA** (Hull Moving Average) - `hull_ma`, `hullma` - Smooth, low-lag trend
- **SuperTrend** - `supertrend`, `st` - ATR-based trend following with dynamic support/resistance

**Oscillators (5)**:

- **RSI** (Relative Strength Index) - `rsi` - Overbought/oversold momentum
- **MACD** (Moving Average Convergence Divergence) - `macd` - Trend momentum
- **Stochastic RSI** - `stochastic_rsi`, `stochrsi`, `stoch_rsi` - Enhanced RSI oscillator
- **MFI** (Money Flow Index) - `mfi` - Volume-weighted RSI
- **WaveTrend** - `wavetrend` - Advanced momentum oscillator

**Bands (2)**:

- **Bollinger Bands** - `bb`, `bollinger` - %B-based precision signals
- **Keltner Channels** - `keltner` - Volatility-based bands

**Volume (1)**:

- **OBV** (On-Balance Volume) - `obv` - Volume-price trend analysis

## 🚀 Dynamic Take Profit Strategies

### 1. Volatility-Adaptive TP ⭐⭐⭐⭐⭐

Adapts take profit targets based on market volatility using ATR (Average True Range).

```bash
# Basic volatility-adaptive TP
dca-backtest -symbol BTCUSDT -indicators "hull_ma,stochrsi" -dynamic-tp volatility_adaptive

# Custom volatility sensitivity
dca-backtest -symbol ETHUSDT -indicators "bb,keltner" -dynamic-tp volatility_adaptive -tp-volatility-mult 0.8

# Custom TP bounds
dca-backtest -symbol ADAUSDT -indicators "rsi,macd" -dynamic-tp volatility_adaptive -tp-min-percent 0.005 -tp-max-percent 0.08
```

**How it works:**

- Higher volatility = Higher TP targets (capture larger moves)
- Lower volatility = Lower TP targets (faster exits)
- Formula: `TP = BaseTP × (1 + normalizedVolatility × multiplier)`

### 2. Indicator-Based TP ⭐⭐⭐⭐⭐

Adjusts take profit based on technical indicator signal strength.

```bash
# Basic indicator-based TP with default weights
dca-backtest -symbol BTCUSDT -indicators "rsi,macd,bb,ema" -dynamic-tp indicator_based

# Custom indicator weights
dca-backtest -symbol ETHUSDT -indicators "rsi,macd,bb,hull_ma" -dynamic-tp indicator_based -tp-indicator-weights "rsi:0.4,macd:0.3,bb:0.2,hull_ma:0.1"

# Custom strength multiplier
dca-backtest -symbol SUIUSDT -indicators "stochrsi,keltner,obv" -dynamic-tp indicator_based -tp-strength-mult 0.5
```

**How it works:**

- Stronger bullish signals = Higher TP targets
- Weaker signals = Lower TP targets
- Formula: `TP = BaseTP × (0.7 + avgSignalStrength × strengthMultiplier)`

### 3. Fixed Multi-Level TP ⭐⭐⭐⭐

Traditional 5-level take profit system (default mode).

```bash
# Multi-level TP (default)
dca-backtest -symbol BTCUSDT -indicators "rsi,bb" -use-tp-levels

# Single-level fixed TP
dca-backtest -symbol ETHUSDT -indicators "macd,ema" -use-tp-levels=false
```

## 📊 DCA Spacing Strategies

### 1. Fixed Progressive Spacing ⭐⭐⭐⭐

Traditional fixed percentage drops with progressive multipliers.

```bash
# Basic fixed spacing (2% base, 1.2x multiplier)
dca-backtest -symbol BTCUSDT -indicators "rsi,bb" -dca-spacing fixed -spacing-threshold 0.02 -spacing-multiplier 1.2

# Aggressive progression
dca-backtest -symbol ETHUSDT -indicators "macd,ema" -dca-spacing fixed -spacing-threshold 0.015 -spacing-multiplier 1.3
```

**Progression Example:**

- Level 1: 2.0% drop
- Level 2: 2.4% drop (2.0% × 1.2)
- Level 3: 2.88% drop (2.4% × 1.2)
- Level 4: 3.46% drop (2.88% × 1.2)
- Level 5: 4.15% drop (3.46% × 1.2)

### 2. Volatility-Adaptive Spacing ⭐⭐⭐⭐⭐

Adjusts DCA entry thresholds based on market volatility.

```bash
# Basic volatility-adaptive spacing
dca-backtest -symbol BTCUSDT -indicators "hull_ma,stochrsi" -dca-spacing volatility_adaptive

# High sensitivity (more responsive to volatility)
dca-backtest -symbol ETHUSDT -indicators "bb,keltner" -dca-spacing volatility_adaptive -spacing-sensitivity 2.5

# Custom ATR period
dca-backtest -symbol ADAUSDT -indicators "rsi,macd" -dca-spacing volatility_adaptive -spacing-atr-period 21
```

**How it works:**

- High volatility = Wider entry thresholds (wait for bigger drops)
- Low volatility = Tighter entry thresholds (enter sooner)
- Formula: `Threshold = BaseThreshold × (1 + ATR/Price × sensitivity)`

## 🏆 Recommended Advanced Combinations

### 1. "The Adaptive Master" - Volatility-Adaptive Everything ⭐⭐⭐⭐⭐

```bash
dca-backtest -symbol BTCUSDT -indicators "hull_ma,stochrsi,keltner" -dynamic-tp volatility_adaptive -dca-spacing volatility_adaptive -optimize
```

**Why this works:**

- **Volatility-adaptive DCA spacing**: Enters on appropriate drops for current volatility
- **Volatility-adaptive TP**: Exits at appropriate targets for current volatility
- **Perfect for**: All market conditions, automatically adapts

### 2. "The Signal Master" - Indicator-Based Everything ⭐⭐⭐⭐⭐

```bash
dca-backtest -symbol ETHUSDT -indicators "rsi,macd,bb,obv" -dynamic-tp indicator_based -dca-spacing fixed -optimize
```

**Why this works:**

- **Indicator-based TP**: Adjusts targets based on signal strength
- **Fixed DCA spacing**: Consistent entry timing
- **Perfect for**: Signal-driven trading, technical analysis focus

### 3. "The Hybrid Power" - Mixed Strategies ⭐⭐⭐⭐⭐

```bash
dca-backtest -symbol ADAUSDT -indicators "hull_ma,stochrsi,keltner,obv" -dynamic-tp indicator_based -dca-spacing volatility_adaptive -optimize
```

**Why this works:**

- **Indicator-based TP**: Signal-driven profit targets
- **Volatility-adaptive spacing**: Market-condition-aware entries
- **Perfect for**: Advanced traders, maximum adaptability

### 4. "The Classic Enhanced" - Traditional with Modern Features ⭐⭐⭐⭐

```bash
dca-backtest -symbol SUIUSDT -indicators "rsi,macd,bb,ema" -dynamic-tp volatility_adaptive -dca-spacing fixed -optimize
```

**Why this works:**

- **Classic indicators**: Proven technical analysis
- **Volatility-adaptive TP**: Modern profit targeting
- **Fixed spacing**: Predictable entry timing
- **Perfect for**: Conservative traders, gradual adoption

## Optimization

The genetic algorithm optimization automatically finds the best parameters for **ALL indicators** and **ALL strategies**:

### Optimized Parameters

**Trend Indicators:**

- **SMA**: Period
- **EMA**: Period
- **Hull MA**: Period
- **SuperTrend**: ATR period, multiplier

**Oscillators:**

- **RSI**: Period, overbought/oversold thresholds
- **MACD**: Fast/slow/signal periods
- **Stochastic RSI**: Period, overbought/oversold thresholds
- **MFI**: Period, overbought/oversold thresholds
- **WaveTrend**: N1/N2 periods, overbought/oversold levels

**Bands:**

- **Bollinger Bands**: Period, standard deviation, %B overbought/oversold thresholds
- **Keltner Channels**: Period, multiplier

**Volume:**

- **OBV**: Trend change threshold

**Dynamic TP Parameters:**

- **Volatility Multiplier**: 0.1 to 2.0
- **Strength Multiplier**: 0.1 to 1.0
- **Min/Max TP Percent**: 0.005 to 0.08

**DCA Spacing Parameters:**

- **Base Threshold**: 0.005 to 0.05
- **Volatility Sensitivity**: 0.5 to 5.0
- **Threshold Multiplier**: 1.0 to 2.0

### Algorithm Configuration

- **Population Size**: 60 individuals
- **Generations**: 35 iterations
- **Parallel Processing**: Up to 4 concurrent evaluations
- **Elite Preservation**: Top 6 solutions carried forward
- **Mutation Rate**: 10% for parameter exploration

### Optimization Examples

```bash
# Optimize with dynamic TP
dca-backtest -symbol BTCUSDT -indicators "hull_ma,stochrsi,keltner" -dynamic-tp volatility_adaptive -optimize

# Optimize with adaptive DCA spacing
dca-backtest -symbol ETHUSDT -indicators "rsi,macd,bb" -dca-spacing volatility_adaptive -optimize

# Optimize everything (indicators + dynamic TP + adaptive spacing)
dca-backtest -symbol ADAUSDT -indicators "hull_ma,stochrsi,keltner,obv" -dynamic-tp indicator_based -dca-spacing volatility_adaptive -optimize

# Walk-forward validation with advanced features
dca-backtest -symbol SUIUSDT -indicators "supertrend,mfi,keltner" -dynamic-tp volatility_adaptive -dca-spacing volatility_adaptive -optimize -wf-enable -wf-rolling
```

### Walk-Forward Validation

Robust validation to prevent overfitting:

```bash
# Simple holdout (70% train, 30% test)
dca-backtest -symbol BTCUSDT -dynamic-tp volatility_adaptive -optimize -wf-enable

# Rolling validation (custom windows)
dca-backtest -symbol BTCUSDT -dynamic-tp indicator_based -dca-spacing volatility_adaptive -optimize -wf-enable -wf-rolling \
  -wf-train-days 90 -wf-test-days 30 -wf-roll-days 15
```

## Output

### Console Output

- **Strategy Summary**: Configuration overview including dynamic TP and DCA spacing
- **Performance Metrics**: Return, drawdown, Sharpe ratio
- **Trade Statistics**: Win rate, average trade, etc.
- **Risk Analysis**: Maximum drawdown, volatility
- **Dynamic TP Analysis**: TP adaptation effectiveness, range utilization
- **DCA Spacing Analysis**: Entry timing effectiveness

### File Output

Results are saved to `results/<SYMBOL>_<INTERVAL>/`:

- `optimized_trades.xlsx` - Detailed Excel report with analysis
- `best_config.json` - Optimized configuration including dynamic TP and DCA spacing settings

## Examples

### Configuration File Example

```json
{
  "strategy": {
    "symbol": "BTCUSDT",
    "base_amount": 50,
    "max_multiplier": 3.5,
    "tp_percent": 0.02,
    "use_tp_levels": true,
    "indicators": ["hull_ma", "stochrsi", "keltner", "obv"],
    "dynamic_tp": {
      "strategy": "volatility_adaptive",
      "base_tp_percent": 0.02,
      "volatility_config": {
        "multiplier": 0.6,
        "min_tp_percent": 0.01,
        "max_tp_percent": 0.05,
        "atr_period": 20
      }
    },
    "dca_spacing": {
      "strategy": "volatility_adaptive",
      "parameters": {
        "base_threshold": 0.015,
        "volatility_sensitivity": 2.0,
        "atr_period": 20
      }
    }
  },
  "risk": {
    "initial_balance": 1000,
    "commission": 0.0005
  }
}
```

### Advanced Configuration Example

```json
{
  "strategy": {
    "symbol": "ETHUSDT",
    "base_amount": 75,
    "max_multiplier": 4.0,
    "tp_percent": 0.025,
    "use_tp_levels": false,
    "indicators": ["rsi", "macd", "bb", "hull_ma", "stochrsi"],
    "dynamic_tp": {
      "strategy": "indicator_based",
      "base_tp_percent": 0.025,
      "indicator_config": {
        "weights": {
          "rsi": 0.3,
          "macd": 0.25,
          "bb": 0.2,
          "hull_ma": 0.15,
          "stochrsi": 0.1
        },
        "strength_multiplier": 0.4,
        "min_tp_percent": 0.015,
        "max_tp_percent": 0.06
      }
    },
    "dca_spacing": {
      "strategy": "fixed",
      "parameters": {
        "base_threshold": 0.02,
        "threshold_multiplier": 1.2
      }
    }
  }
}
```

### Command Line Examples

```bash
# Full help
dca-backtest --help

# Version information
dca-backtest --version

# Console only (no file output)
dca-backtest -symbol BTCUSDT -dynamic-tp volatility_adaptive -console-only

# Test all intervals with advanced features
dca-backtest -symbol BTCUSDT -indicators "hull_ma,stochrsi,keltner" -dynamic-tp indicator_based -dca-spacing volatility_adaptive -all-intervals
```

## Architecture

This command integrates with the refactored architecture:

```
cmd/dca-backtest/main.go
├── pkg/orchestrator/ (Phase 4)
│   ├── Coordinates all components
│   └── Handles complex workflows
├── pkg/optimization/ (Phase 2)
│   └── Genetic algorithm optimization
├── pkg/validation/ (Phase 2)
│   └── Walk-forward validation
├── pkg/reporting/ (Phase 3)
│   └── Multi-format output
├── pkg/config/ (Phase 1)
│   └── Configuration management
└── pkg/data/ (Phase 1)
    └── Data loading & caching
```

## Migration from Generic Backtest

The DCA command provides the same functionality as the generic backtest but with:

- **Better UX**: Focused on DCA strategies
- **Cleaner Code**: Streamlined and modular design
- **Enhanced Help**: DCA-specific examples and documentation
- **Validation**: Built-in parameter validation
- **Advanced Features**: Dynamic TP and adaptive DCA spacing
- **Future Ready**: Easy to extend for new features

## Performance

- **Startup Time**: < 1 second
- **Memory Usage**: Minimal increase from modular design
- **Optimization Speed**: No degradation vs original monolithic code
- **Caching**: Intelligent data caching for repeated analysis
- **Dynamic Features**: < 5% performance overhead for advanced features

## Development

Built using the enhanced architecture from the backtest refactor:

- **Modular Design**: Clean separation of concerns
- **Interface-Based**: Easy to test and extend
- **Type Safe**: Strong typing throughout
- **Error Handling**: Comprehensive error management
- **Documentation**: Self-documenting code with clear interfaces
- **Advanced Features**: Dynamic TP and adaptive spacing strategies

## Future Extensions

The modular architecture makes it easy to add:

- **New Indicators**: Simply implement indicator interface
- **Different Strategies**: Grid, momentum, mean reversion
- **Advanced Optimization**: Bayesian optimization, PSO
- **More Validation**: K-fold, Monte Carlo simulation
- **Additional Outputs**: HTML dashboards, PDF reports
- **New TP Strategies**: Time-based, volume-based, momentum-based
- **New Spacing Strategies**: Support/resistance-based, volume-based
