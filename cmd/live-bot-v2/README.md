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

## migrating From V1

The V2 bot can automatically convert a V1 configuration file to the new format using the `-legacy` flag:

```bash
# Convert a legacy config file for use with Bybit
./live-bot-v2 -config configs/btc_5.json -legacy -exchange bybit -demo
```

## 📋 Command-Line Flags

| Flag        | Description                                     | Default |
| ----------- | ----------------------------------------------- | ------- |
| `-config`   | Path to the configuration file.                 | -       |
| `-exchange` | The exchange to use (e.g., `bybit`, `binance`). | -       |
| `-demo`     | Set to `false` to enable live trading.          | `true`  |
| `-legacy`   | Set to `true` to convert a V1 config file.      | `false` |
| `-env`      | Path to the environment file.                   | `.env`  |

## ⚙️ Configuration

The V2 bot uses a nested configuration structure that separates the strategy, exchange, and risk parameters. You can find examples in the `configs/bybit/` and `configs/binance/` directories.

---

**⚠️ Important**: Always test in demo mode before live trading!
