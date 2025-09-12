# Enhanced DCA Bot

A sophisticated cryptocurrency trading bot that implements an enhanced Dollar Cost Averaging (DCA) strategy with multi-exchange support, advanced backtesting, and real-time monitoring.

## Features

### 🎯 **Enhanced DCA Strategy**

- **Multi-indicator approach** with 11 technical indicators:
  - **Classic Combo**: RSI, MACD, Enhanced Bollinger Bands with %B, EMA
  - **Advanced Combo**: Hull MA, MFI, Keltner Channels, WaveTrend
  - **Momentum**: Stochastic RSI, SuperTrend
  - **Volume**: OBV (On-Balance Volume)
- **Dynamic position sizing** based on signal strength and confidence
- **Precision %B signals** from enhanced Bollinger Bands
- **Configurable thresholds** for all indicators with optimization support
- **Genetic algorithm optimization** for all indicator parameters

### 📊 **Advanced Backtesting & Analytics**

- **Single/Multi-Level Take Profit**: Default single TP at 4.5%, optional 5-level TP system
- **Comprehensive Excel Reports**: 4 detailed sheets with professional analytics
- **Cycle Analysis**: Sequential DCA cycle tracking with balance progression
- **Performance Metrics**: Sharpe Ratio, Profit Factor, Max Drawdown, Win Rate
- **Strategic Insights**: AI-powered recommendations and optimization tips
- **Visual Analytics**: Color-coded performance indicators and trend analysis
- **Historical Testing**: Support for multiple timeframes and market conditions

### arquitectura multi-intercambio

- Modular design supporting multiple exchanges (Bybit, Binance)
- Unified interface for seamless switching between exchanges
- Standardized error handling and data models

### 🛡️ **Risk Management**

- Configurable initial balance and commission rates
- Minimum order quantity enforcement
- Demo and testnet modes for safe testing

### 📈 **Monitoring & Analytics**

- Prometheus metrics integration for real-time performance tracking
- Health check endpoints for system monitoring
- Grafana dashboards for data visualization

### 🔔 **Notifications**

- Telegram integration for real-time trade alerts
- Configurable notification settings

## Architecture

The V2 bot introduces a modular, exchange-agnostic architecture:

```
LiveBot V2 → Exchange Interface → Exchange Adapter → Exchange API
```

- **Exchange Interface**: A universal API for all trading operations.
- **Exchange Adapters**: Exchange-specific implementations for Bybit, Binance, etc.
- **Exchange Factory**: Dynamically loads the correct exchange adapter based on the configuration.

## Quick Start

### Prerequisites

- Go 1.24 or later
- Docker and Docker Compose
- Bybit or Binance API credentials

### Installation

1.  **Clone the Repository**

    ```bash
    git clone https://github.com/ducminhle1904/crypto-dca-bot.git
    cd crypto-dca-bot
    ```

2.  **Install Dependencies**

    ```bash
    go mod download
    ```

3.  **Configure Environment**
    Create a `.env` file and add your API keys:

    ```bash
    cp .env.example .env
    # Edit the .env file with your API credentials
    ```

4.  **Run with Docker Compose**
    ```bash
    docker-compose up -d
    ```

## Configuration

The bot uses a nested JSON configuration format and environment variables.

### Configuration File (`configs/bybit/btc_5m_bybit.json`)

```json
{
  "strategy": {
    "symbol": "BTCUSDT",
    "base_amount": 40,
    ...
  },
  "exchange": {
    "name": "bybit",
    "bybit": { ... }
  },
  "risk": {
    "initial_balance": 1000.0,
    ...
  }
}
```

### Environment Variables (`.env`)

```bash
# Bybit API credentials
BYBIT_API_KEY="your_bybit_api_key"
BYBIT_API_SECRET="your_bybit_api_secret"

# Binance API credentials
BINANCE_API_KEY="your_binance_api_key"
BINANCE_API_SECRET="your_binance_api_secret"

# Telegram notifications
TELEGRAM_TOKEN="your_telegram_bot_token"
TELEGRAM_CHAT_ID="your_telegram_chat_id"
```

## Usage

### Running the Live Bot (V2)

```bash
# Run the V2 bot with a Bybit configuration in demo mode
go run cmd/live-bot-v2/main.go -config configs/bybit/btc_5m_bybit.json -demo

# Run with a Binance configuration in live mode
go run cmd/live-bot-v2/main.go -config configs/binance/btc_5m_binance.json -demo=false
```

### Running DCA Backtests

```bash
# Basic backtest with enhanced Bollinger Bands
go run cmd/dca-backtest/main.go -symbol BTCUSDT -interval 5m -indicators "bb"

# Advanced multi-indicator backtest
go run cmd/dca-backtest/main.go -symbol ETHUSDT -interval 1h -indicators "rsi,macd,bb,stochrsi,obv"

# Individual indicator flags
go run cmd/dca-backtest/main.go -symbol HYPEUSDT -rsi -bb -stochrsi -obv

# Run with optimization (finds best thresholds)
go run cmd/dca-backtest/main.go -symbol SUIUSDT -indicators "bb,stochrsi" -optimize

# Use configuration file with custom indicator settings
go run cmd/dca-backtest/main.go -config configs/bybit/dca/hype_5m_bybit.json

# Advanced combo with all momentum indicators
go run cmd/dca-backtest/main.go -symbol BTCUSDT -indicators "supertrend,stochrsi,obv,keltner,wavetrend"
```

#### **Backtest Outputs**

- **Console**: Real-time performance metrics and statistics
- **Excel Report**: Professional 4-sheet analysis (`trades.xlsx`)
  - **Trades**: Cycle-organized trade details
  - **Cycles**: Balance tracking and capital usage analysis
  - **Detailed Analysis**: Comprehensive performance insights
  - **Timeline**: Chronological trading activity view
- **Configuration**: Optimized parameters (`best.json`)

For detailed DCA backtesting documentation, see [`cmd/dca-backtest/README.md`](cmd/dca-backtest/README.md).

### Monitoring

- **Health Check**: `http://localhost:8081/health`
- **Prometheus Metrics**: `http://localhost:8080/metrics`
- **Grafana Dashboard**: `http://localhost:3000` (admin/admin)

## Deployment

### Docker

```bash
# Build the Docker image
docker build -t crypto-dca-bot .

# Run the container with your environment variables
docker run -d \
  --name dca-bot \
  -p 8080:8080 \
  -p 8081:8081 \
  --env-file .env \
  crypto-dca-bot
```

### Kubernetes

A sample Kubernetes deployment configuration is available in the repository.

## Disclaimer

This software is for educational purposes only. Cryptocurrency trading involves substantial risk.

---

**Note**: Always test thoroughly in a testnet or demo environment before using real funds.
