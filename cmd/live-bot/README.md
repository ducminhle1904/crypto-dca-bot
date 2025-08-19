# Enhanced DCA Live Bot

A futures DCA trading bot with 10x leverage for the Bybit exchange.

## 🚀 Quick Start

### 1. Set Up API Credentials

Create a `.env` file and add your Bybit API key and secret:

```bash
cp .env.example .env
# Edit the .env file with your credentials
```

### 2. Run in Demo Mode

The bot starts in demo mode by default, which is safe for paper trading. To run the bot with a pre-defined configuration:

```bash
# Run with the BTC/USDT 5-minute configuration
go run cmd/live-bot/main.go -config btc_5.json

# Run with the SUI/USDT 5-minute configuration
go run cmd/live-bot/main.go -config sui_5.json
```

### 3. Run in Live Mode

To run the bot with real funds, use the `-demo=false` flag:

```bash
# WARNING: This will trade with real money
go run cmd/live-bot/main.go -config btc_5.json -demo=false
```

## 📋 Command-Line Flags

| Flag      | Description                               | Default |
| --------- | ----------------------------------------- | ------- |
| `-config` | The configuration file to use (required). | -       |
| `-demo`   | Set to `false` to enable live trading.    | `true`  |
| `-env`    | The environment file to use.              | `.env`  |

## ⚙️ Configuration Files

Configuration files are located in the `configs/` directory. You can create your own configurations based on the `btc_5.json` and `sui_5.json` examples.

## ⚠️ Safety

- **Demo Mode**: The bot runs in a safe paper trading mode by default.
- **Graceful Shutdown**: Use `Ctrl+C` to stop the bot. It will automatically close any open positions.
