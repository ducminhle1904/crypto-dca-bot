# DCA Backtest Command

The DCA Backtest command provides a clean, focused interface specifically designed for Dollar Cost Averaging (DCA) strategy backtesting and optimization.

## Overview

This command represents the completion of **Phase 5** of the backtest system refactor, providing:

- **Clean Interface**: DCA-specific command line interface (~200 lines vs 4,147 original)
- **Focused Functionality**: Streamlined for DCA strategies only
- **Orchestrator Integration**: Uses the refactored orchestrator system
- **Modern CLI**: Enhanced help, validation, and user experience

## Features

### Core Functionality

- ✅ **Single Backtests**: Test DCA strategy on historical data
- ✅ **Parameter Optimization**: Genetic algorithm optimization
- ✅ **Multi-Interval Analysis**: Test across all available timeframes
- ✅ **Walk-Forward Validation**: Robust strategy validation
- ✅ **Advanced/Classic Indicators**: Two complete indicator combos

### Output Formats

- ✅ **Console Output**: Formatted results with context
- ✅ **Excel Reports**: Detailed trade analysis with charts
- ✅ **JSON Config**: Optimized parameters for reuse
- ✅ **CSV Export**: Raw trade data

## Quick Start

### Basic Usage

```bash
# Simple backtest
dca-backtest -symbol BTCUSDT -interval 1h

# Load from configuration file
dca-backtest -config configs/bybit/btc_1h.json

# Optimize parameters
dca-backtest -symbol ETHUSDT -optimize

# Test all intervals
dca-backtest -symbol BTCUSDT -all-intervals
```

### Advanced Usage

```bash
# Walk-forward validation with optimization
dca-backtest -symbol BTCUSDT -optimize -wf-enable -wf-rolling

# Advanced indicators with custom parameters
dca-backtest -symbol ADAUSDT -advanced-combo -base-amount 50 -max-multiplier 2.5

# Limited time period analysis
dca-backtest -symbol BTCUSDT -period 30d -optimize
```

## Configuration

### DCA Strategy Parameters

| Parameter         | Default | Description                             |
| ----------------- | ------- | --------------------------------------- |
| `base-amount`     | 40      | Base DCA investment amount              |
| `max-multiplier`  | 3.0     | Maximum position multiplier             |
| `price-threshold` | 0.02    | Minimum price drop % for next DCA entry |
| `advanced-combo`  | false   | Use advanced indicators vs classic      |

### Indicator Combos

**Classic Combo (default)**:

- RSI (Relative Strength Index)
- MACD (Moving Average Convergence Divergence)
- Bollinger Bands
- EMA (Exponential Moving Average)

**Advanced Combo**:

- Hull Moving Average
- Money Flow Index (MFI)
- Keltner Channels
- WaveTrend Oscillator

### Account Settings

| Parameter    | Default | Description                |
| ------------ | ------- | -------------------------- |
| `balance`    | 500     | Initial account balance    |
| `commission` | 0.0005  | Trading commission (0.05%) |

## Optimization

The genetic algorithm optimization automatically finds the best parameters:

- **Population Size**: 60 individuals
- **Generations**: 35 iterations
- **Parallel Processing**: Up to 4 concurrent evaluations
- **Elite Preservation**: Top 6 solutions carried forward
- **Mutation Rate**: 10% for parameter exploration

### Walk-Forward Validation

Robust validation to prevent overfitting:

```bash
# Simple holdout (70% train, 30% test)
dca-backtest -symbol BTCUSDT -optimize -wf-enable

# Rolling validation (custom windows)
dca-backtest -symbol BTCUSDT -optimize -wf-enable -wf-rolling \
  -wf-train-days 90 -wf-test-days 30 -wf-roll-days 15
```

## Output

### Console Output

- **Strategy Summary**: Configuration overview
- **Performance Metrics**: Return, drawdown, Sharpe ratio
- **Trade Statistics**: Win rate, average trade, etc.
- **Risk Analysis**: Maximum drawdown, volatility

### File Output

Results are saved to `results/<SYMBOL>_<INTERVAL>/`:

- `trades.xlsx` - Detailed Excel report with analysis
- `best_config.json` - Optimized configuration
- `trades.csv` - Raw trade data (if requested)

## Examples

### Configuration File Example

```json
{
  "strategy": {
    "symbol": "BTCUSDT",
    "base_amount": 50,
    "max_multiplier": 3.5,
    "price_threshold": 0.025,
    "tp_percent": 0.02,
    "use_advanced_combo": true,
    "indicators": ["hull_ma", "mfi", "keltner", "wavetrend"]
  },
  "risk": {
    "initial_balance": 1000,
    "commission": 0.0005
  }
}
```

### Command Line Options

```bash
# Full help
dca-backtest --help

# Version information
dca-backtest --version

# Console only (no file output)
dca-backtest -symbol BTCUSDT -console-only
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
- **Cleaner Code**: ~200 lines vs 612 lines
- **Enhanced Help**: DCA-specific examples and documentation
- **Validation**: Built-in parameter validation
- **Future Ready**: Easy to extend for new features

## Performance

- **Startup Time**: < 1 second
- **Memory Usage**: Minimal increase from modular design
- **Optimization Speed**: No degradation vs original monolithic code
- **Caching**: Intelligent data caching for repeated analysis

## Development

Built using the enhanced architecture from the backtest refactor:

- **Modular Design**: Clean separation of concerns
- **Interface-Based**: Easy to test and extend
- **Type Safe**: Strong typing throughout
- **Error Handling**: Comprehensive error management
- **Documentation**: Self-documenting code with clear interfaces

## Future Extensions

The modular architecture makes it easy to add:

- **New Indicators**: Simply implement indicator interface
- **Different Strategies**: Grid, momentum, mean reversion
- **Advanced Optimization**: Bayesian optimization, PSO
- **More Validation**: K-fold, Monte Carlo simulation
- **Additional Outputs**: HTML dashboards, PDF reports
