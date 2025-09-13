# Enhanced DCA Live Bot

This is a modular, multi-exchange version of the Enhanced DCA Live Bot, featuring a completely redesigned architecture.

## 🚀 Quick Start

### 1. Installation

```bash
# Clone the repository and build the live bot
git clone https://github.com/ducminhle1904/crypto-dca-bot.git
cd crypto-dca-bot
go mod download
go build -o live-bot-dca ./cmd/live-bot-dca
```

### 2. Set Up API Credentials

Create a `.env` file and add your API keys for the exchanges you want to use:

```bash
cp .env.example .env
# Edit the .env file with your credentials
```

### 3. Run the Bot

The live bot uses a modular configuration format. You can run it with a Bybit or Binance configuration in demo mode:

```bash
# Run with a Bybit configuration in demo mode
./live-bot-dca -config configs/bybit/btc_5m_bybit.json -demo

# Run with a Binance configuration in demo mode
./live-bot-dca -config configs/binance/btc_5m_binance.json -demo
```

To run in live mode with real funds, set the `-demo` flag to `false`:

```bash
# WARNING: This will trade with real money
./live-bot-dca -config configs/bybit/btc_5m_bybit.json -demo=false
```

## 📋 Command-Line Flags

| Flag        | Description                                     | Default |
| ----------- | ----------------------------------------------- | ------- |
| `-config`   | Path to the configuration file.                 | -       |
| `-exchange` | The exchange to use (e.g., `bybit`, `binance`). | -       |
| `-demo`     | Set to `false` to enable live trading.          | `true`  |
| `-env`      | Path to the environment file.                   | `.env`  |

## ⚙️ Configuration

The live bot uses a nested configuration structure that separates the strategy, exchange, and risk parameters. You can find examples in the `configs/bybit/` and `configs/binance/` directories.

### Supported Indicators (12 Total)

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

## 🏆 Recommended Indicator Combinations

Based on extensive testing and analysis, here are the **Tier 1** recommended combinations for optimal DCA performance:

### 1. "The Golden Trio" - 3 Indicators ⭐⭐⭐⭐⭐

```json
"indicators": ["hull_ma", "stochastic_rsi", "keltner"]
```

**Why this works:**

- **Hull MA**: Trend confirmation (price above = uptrend)
- **Stochastic RSI**: Momentum timing (oversold recovery)
- **Keltner Channels**: Volatility-based entry (lower band = oversold)

**Perfect for**: Most market conditions, balanced risk/reward

### 2. "The Momentum Master" - 4 Indicators ⭐⭐⭐⭐⭐

```json
"indicators": ["hull_ma", "stochastic_rsi", "macd", "obv"]
```

**Why this works:**

- **Hull MA**: Trend direction
- **Stochastic RSI**: Short-term momentum
- **MACD**: Medium-term momentum confirmation
- **OBV**: Volume confirmation

**Perfect for**: Trending markets, high-volume assets

### 3. "The Complete System" - 5 Indicators ⭐⭐⭐⭐⭐

```json
"indicators": ["hull_ma", "stochastic_rsi", "keltner", "macd", "obv"]
```

**Why this works:**

- All major signal types covered
- Multiple confirmation layers
- Reduces false signals significantly

**Perfect for**: Conservative trading, maximum reliability

### 4. "The Classic Power" - 4 Indicators ⭐⭐⭐⭐

```json
"indicators": ["rsi", "macd", "bb", "ema"]
```

**Why this works:**

- **RSI**: Classic overbought/oversold signals
- **MACD**: Trend momentum confirmation
- **Bollinger Bands**: %B-based precision entries
- **EMA**: Trend direction filter

**Perfect for**: Traditional technical analysis approach

### 5. "The Advanced Momentum" - 4 Indicators ⭐⭐⭐⭐

```json
"indicators": ["hull_ma", "mfi", "wavetrend", "keltner"]
```

**Why this works:**

- **Hull MA**: Smooth trend following
- **MFI**: Volume-weighted momentum
- **WaveTrend**: Advanced oscillator signals
- **Keltner Channels**: Volatility-based entries

**Perfect for**: Advanced traders, complex market conditions

### 6. "The Trend Master" - 4 Indicators ⭐⭐⭐⭐⭐

```json
"indicators": ["supertrend", "stochastic_rsi", "keltner", "obv"]
```

**Why this works:**

- **SuperTrend**: ATR-based trend following with dynamic support/resistance
- **Stochastic RSI**: Momentum timing for entries
- **Keltner Channels**: Volatility-based confirmation
- **OBV**: Volume trend confirmation

**Perfect for**: Strong trending markets, high volatility assets

### Quick Setup Commands

```bash
# Find optimal parameters for Golden Trio (3 indicators)
dca-backtest -symbol BTCUSDT -indicators "hull_ma,stochastic_rsi,keltner" -optimize

# Find optimal parameters for Momentum Master (4 indicators)
dca-backtest -symbol ETHUSDT -indicators "hull_ma,stochastic_rsi,macd,obv" -optimize

# Find optimal parameters for Complete System (5 indicators)
dca-backtest -symbol SUIUSDT -indicators "hull_ma,stochastic_rsi,keltner,macd,obv" -optimize

# Find optimal parameters for Classic Power (4 indicators)
dca-backtest -symbol ADAUSDT -indicators "rsi,macd,bb,ema" -optimize

# Find optimal parameters for Advanced Momentum (4 indicators)
dca-backtest -symbol SOLUSDT -indicators "hull_ma,mfi,wavetrend,keltner" -optimize

# Find optimal parameters for Trend Master (4 indicators)
dca-backtest -symbol BNBUSDT -indicators "supertrend,stochastic_rsi,keltner,obv" -optimize

# Find optimal parameters for all 12 indicators
dca-backtest -symbol HYPEUSDT -indicators "rsi,macd,bb,ema,hull_ma,supertrend,mfi,keltner,wavetrend,obv,stochastic_rsi" -optimize

# Then use the optimized config in live trading
./live-bot-dca -config results/BTCUSDT_5m/best_config.json -demo
```

---

**⚠️ Important**: Always test in demo mode before live trading!
