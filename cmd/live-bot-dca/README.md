# Enhanced DCA Live Bot V2

This is a modular, multi-exchange version of the Enhanced DCA Live Bot, featuring a completely redesigned architecture.

## 🚀 Quick Start

### 1. Installation

```bash
# Clone the repository and build the V2 bot
git clone https://github.com/ducminhle1904/crypto-dca-bot.git
cd crypto-dca-bot
go mod download
go build -o live-bot-v2 ./cmd/live-bot-v2
```

### 2. Set Up API Credentials

Create a `.env` file and add your API keys for the exchanges you want to use:

```bash
cp .env.example .env
# Edit the .env file with your credentials
```

### 3. Run the Bot

The V2 bot uses a new, modular configuration format. You can run it with a Bybit or Binance configuration in demo mode:

```bash
# Run with a Bybit configuration in demo mode
./live-bot-v2 -config configs/bybit/btc_5m_bybit.json -demo

# Run with a Binance configuration in demo mode
./live-bot-v2 -config configs/binance/btc_5m_binance.json -demo
```

To run in live mode with real funds, set the `-demo` flag to `false`:

```bash
# WARNING: This will trade with real money
./live-bot-v2 -config configs/bybit/btc_5m_bybit.json -demo=false
```

## 📋 Command-Line Flags

| Flag        | Description                                     | Default |
| ----------- | ----------------------------------------------- | ------- |
| `-config`   | Path to the configuration file.                 | -       |
| `-exchange` | The exchange to use (e.g., `bybit`, `binance`). | -       |
| `-demo`     | Set to `false` to enable live trading.          | `true`  |
| `-env`      | Path to the environment file.                   | `.env`  |

## ⚙️ Configuration

The V2 bot uses a nested configuration structure that separates the strategy, exchange, and risk parameters. You can find examples in the `configs/bybit/` and `configs/binance/` directories.

### Supported Indicators

The live bot supports all 11 technical indicators:

- **Classic**: RSI, MACD, Enhanced Bollinger Bands with %B, EMA
- **Advanced**: Hull MA, MFI, Keltner Channels, WaveTrend
- **Momentum**: Stochastic RSI, SuperTrend
- **Volume**: OBV (On-Balance Volume)

Configure indicators in your JSON config file with custom thresholds and parameters. Use the DCA backtest tool first to find optimal settings:

```bash
# Find optimal parameters first
dca-backtest -symbol BTCUSDT -indicators "bb,stochrsi,obv" -optimize

# Then use the optimized config in live trading
./live-bot-v2 -config optimized_config.json -demo
```

---

**⚠️ Important**: Always test in demo mode before live trading!
