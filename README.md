# Enhanced DCA Bot

A sophisticated cryptocurrency trading bot that implements an enhanced Dollar Cost Averaging (DCA) strategy with multi-exchange support, advanced backtesting, and real-time monitoring.

## Why This Bot?

### 🎯 **Intelligent DCA Strategy**

Unlike simple DCA bots, this system uses 12 technical indicators with genetic algorithm optimization to make intelligent entry decisions, maximizing returns while minimizing risk.

### 🔬 **Scientific Approach**

- **Backtesting First**: Test strategies thoroughly before risking real money
- **Optimization**: Genetic algorithms find optimal parameters for your specific market conditions
- **Validation**: Walk-forward testing ensures strategies work in different market conditions

### 🏗️ **Production Ready**

- **Multi-Exchange**: Support for Bybit, Binance, and easily extensible to more exchanges
- **Risk Management**: Built-in safety controls and position sizing
- **Monitoring**: Comprehensive observability with Prometheus and Grafana
- **Scalable**: Modular architecture designed for production deployment

## Features

### 🎯 **Enhanced DCA Strategy**

- **Multi-indicator approach** with 12 technical indicators:
  - **Trend Indicators**: SMA, EMA, Hull MA, SuperTrend
  - **Oscillators**: RSI, MACD, Stochastic RSI, MFI, WaveTrend
  - **Bands**: Bollinger Bands, Keltner Channels
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

## Project Structure

```
crypto-dca-bot/
├── cmd/                          # Command-line applications
│   ├── dca-backtest/            # DCA strategy backtesting engine
│   ├── live-bot-dca/            # Live trading bot
│   └── grid-backtest/           # Grid trading backtesting
├── internal/                     # Core business logic
│   ├── indicators/              # 12 technical indicators
│   ├── strategy/                # Trading strategies
│   ├── exchange/                # Exchange adapters
│   └── monitoring/              # Health checks and metrics
├── pkg/                         # Reusable packages
│   ├── optimization/            # Genetic algorithm optimization
│   ├── reporting/               # Excel and JSON reporting
│   └── config/                  # Configuration management
└── configs/                     # Example configurations
```

## Architecture

The system uses a modular, exchange-agnostic architecture:

```
LiveBot → Exchange Interface → Exchange Adapter → Exchange API
```

- **Exchange Interface**: A universal API for all trading operations
- **Exchange Adapters**: Exchange-specific implementations for Bybit, Binance, etc.
- **Exchange Factory**: Dynamically loads the correct exchange adapter based on configuration
- **Strategy Engine**: Pluggable strategy system supporting DCA, Grid, and custom strategies
- **Optimization Engine**: Genetic algorithm optimization for parameter tuning

## Quick Start

### Prerequisites

- Go 1.24 or later
- Docker and Docker Compose (optional)
- Exchange API credentials (Bybit/Binance)

### Installation

```bash
# Clone the repository
git clone https://github.com/ducminhle1904/crypto-dca-bot.git
cd crypto-dca-bot

# Install dependencies
go mod download

# Configure environment
cp .env.example .env
# Edit .env with your API credentials
```

### Getting Started

1. **Backtesting**: Start with the DCA backtesting engine to develop and optimize your strategies
2. **Live Trading**: Use the live bot to execute your optimized strategies in real-time
3. **Monitoring**: Set up monitoring and alerts for production trading

For detailed usage instructions, see the component-specific documentation:

- [DCA Backtesting Guide](cmd/dca-backtest/README.md)
- [Live Trading Bot Guide](cmd/live-bot-dca/README.md)
- [Grid Trading Guide](cmd/grid-backtest/README.md)

## Configuration

The system uses a modular configuration approach with JSON files and environment variables.

### Configuration Structure

- **Strategy Configuration**: DCA parameters, indicators, and thresholds
- **Exchange Configuration**: Exchange-specific settings and credentials
- **Risk Management**: Balance limits, commission rates, and safety controls
- **Environment Variables**: API keys and sensitive configuration data

### Example Configuration

```json
{
  "strategy": {
    "symbol": "BTCUSDT",
    "base_amount": 40,
    "indicators": ["hull_ma", "stochastic_rsi", "keltner"],
    "optimization": true
  },
  "exchange": {
    "name": "bybit",
    "demo_mode": true
  },
  "risk": {
    "initial_balance": 1000.0,
    "commission": 0.001
  }
}
```

### Environment Setup

```bash
# Exchange API credentials
BYBIT_API_KEY="your_bybit_api_key"
BYBIT_API_SECRET="your_bybit_api_secret"
BINANCE_API_KEY="your_binance_api_key"
BINANCE_API_SECRET="your_binance_api_secret"

# Notifications
TELEGRAM_TOKEN="your_telegram_bot_token"
TELEGRAM_CHAT_ID="your_telegram_chat_id"
```

## Components

### 🤖 **Live Trading Bot**

A production-ready live trading bot that executes DCA strategies in real-time across multiple exchanges.

**Key Features:**

- Multi-exchange support (Bybit, Binance)
- Real-time indicator analysis
- Automated position management
- Risk management and safety controls
- Demo and live trading modes

**Documentation:** [`cmd/live-bot-dca/README.md`](cmd/live-bot-dca/README.md)

### 📊 **DCA Backtesting Engine**

A comprehensive backtesting system for strategy development and optimization.

**Key Features:**

- 12 technical indicators with genetic algorithm optimization
- Walk-forward validation for robust testing
- Multi-interval analysis across all timeframes
- Professional Excel reporting with 4 detailed sheets
- Configuration export for live trading

**Documentation:** [`cmd/dca-backtest/README.md`](cmd/dca-backtest/README.md)

### 📈 **Grid Trading Backtest**

Advanced grid trading strategy backtesting with optimization capabilities.

**Key Features:**

- Grid strategy implementation
- Parameter optimization
- Performance analysis
- Multi-timeframe support

**Documentation:** [`cmd/grid-backtest/README.md`](cmd/grid-backtest/README.md)

## Monitoring & Observability

### Real-time Monitoring

- **Health Checks**: System status and component health monitoring
- **Prometheus Metrics**: Performance metrics and trading statistics
- **Grafana Dashboards**: Visual analytics and performance tracking
- **Telegram Alerts**: Real-time notifications for trades and system events

### Key Metrics

- Trading performance and P&L
- System health and uptime
- API rate limits and errors
- Risk management alerts

## Deployment Options

### Docker Deployment

Containerized deployment with Docker Compose for easy setup and management.

### Kubernetes Deployment

Production-ready Kubernetes manifests for scalable deployment in cloud environments.

### Local Development

Direct Go execution for development and testing with hot reloading.

## Disclaimer

This software is for educational purposes only. Cryptocurrency trading involves substantial risk.

---

**Note**: Always test thoroughly in a testnet or demo environment before using real funds.
