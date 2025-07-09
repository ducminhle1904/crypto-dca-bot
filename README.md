# Enhanced DCA Bot

A sophisticated cryptocurrency trading bot that implements an enhanced Dollar Cost Averaging (DCA) strategy with multiple technical indicators, risk management, and real-time monitoring.

## Features

### 🎯 **Enhanced DCA Strategy**
- Multi-indicator approach combining RSI, MACD, Bollinger Bands, and SMA
- Dynamic position sizing based on signal strength and confidence
- Configurable base amounts and maximum multipliers
- Minimum interval enforcement between trades

### 📊 **Technical Indicators**
- **RSI (Relative Strength Index)**: Identifies overbought/oversold conditions
- **MACD (Moving Average Convergence Divergence)**: Trend following and momentum
- **Bollinger Bands**: Volatility and price channel analysis
- **SMA (Simple Moving Average)**: Trend direction identification

### 🛡️ **Risk Management**
- Maximum position size limits
- Balance percentage restrictions
- Minimum balance thresholds
- Configurable risk parameters

### 📈 **Monitoring & Analytics**
- Prometheus metrics integration
- Health check endpoints
- Real-time performance tracking
- Grafana dashboards

### 🔔 **Notifications**
- Telegram integration for trade alerts
- Configurable notification levels
- Real-time status updates

### 🔄 **Backtesting**
- Historical data analysis
- Performance metrics calculation
- Strategy optimization tools

## Architecture

```
enhanced-dca-bot/
├── cmd/bot/                 # Main application entry point
├── internal/
│   ├── backtest/           # Backtesting engine
│   ├── config/             # Configuration management
│   ├── exchange/           # Exchange integrations
│   ├── indicators/         # Technical indicators
│   ├── monitoring/         # Health checks and metrics
│   ├── notifications/      # Alert systems
│   ├── risk/              # Risk management
│   └── strategy/          # Trading strategies
├── pkg/types/             # Shared data types
├── configs/               # Configuration files
├── monitoring/            # Prometheus and Grafana configs
└── scripts/              # Deployment and utility scripts
```

## Quick Start

### Prerequisites
- Go 1.24 or later
- Docker and Docker Compose
- Binance API credentials (for live trading)

### Installation

1. **Clone the repository**
   ```bash
   git clone https://github.com/Zmey56/enhanced-dca-bot.git
   cd enhanced-dca-bot
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Configure environment variables**
   ```bash
   export EXCHANGE_API_KEY="your_binance_api_key"
   export EXCHANGE_SECRET="your_binance_secret"
   export TELEGRAM_TOKEN="your_telegram_bot_token"
   export TELEGRAM_CHAT_ID="your_chat_id"
   ```

4. **Run with Docker Compose**
   ```bash
   docker-compose up -d
   ```

### Configuration

The bot can be configured through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `ENV` | Environment (development/production) | development |
| `LOG_LEVEL` | Logging level | debug |
| `EXCHANGE_NAME` | Exchange name | binance |
| `EXCHANGE_API_KEY` | API key | - |
| `EXCHANGE_SECRET` | API secret | - |
| `EXCHANGE_TESTNET` | Use testnet | true |
| `TRADING_SYMBOL` | Trading pair | BTCUSDT |
| `BASE_AMOUNT` | Base DCA amount | 100.0 |
| `MAX_MULTIPLIER` | Maximum position multiplier | 3.0 |
| `TRADING_INTERVAL` | Trading interval | 1h |
| `PROMETHEUS_PORT` | Prometheus metrics port | 8080 |
| `HEALTH_PORT` | Health check port | 8081 |

## Usage

### Running the Bot

```bash
# Development mode
go run cmd/bot/main.go

# Production mode
docker-compose up -d
```

### Monitoring

- **Health Check**: http://localhost:8081/health
- **Prometheus Metrics**: http://localhost:8080/metrics
- **Grafana Dashboard**: http://localhost:3000 (admin/admin123)

### Backtesting

```bash
# Run backtest with configuration
go run cmd/backtest/main.go -config configs/backtest.yaml
```

## Strategy Details

### Enhanced DCA Logic

The bot implements an enhanced DCA strategy that:

1. **Analyzes multiple indicators** to determine market conditions
2. **Calculates confidence scores** based on indicator consensus
3. **Adjusts position sizes** dynamically based on signal strength
4. **Enforces risk limits** to protect capital
5. **Maintains minimum intervals** between trades

### Signal Generation

- **Buy Signal**: When multiple indicators show oversold conditions
- **Hold Signal**: When indicators are neutral or conflicting
- **Position Sizing**: Based on confidence level and risk parameters

### Risk Management

- Maximum 50% of balance per trade
- Configurable maximum position size
- Minimum balance thresholds
- Stop trading on low balance

## Development

### Project Structure

The project follows clean architecture principles:

- **Domain Layer**: Core business logic (strategies, indicators)
- **Application Layer**: Use cases and orchestration
- **Infrastructure Layer**: External integrations (exchanges, notifications)
- **Interface Layer**: HTTP handlers and API endpoints

### Adding New Indicators

1. Implement the `TechnicalIndicator` interface
2. Add calculation logic
3. Define buy/sell conditions
4. Register in the strategy

### Adding New Exchanges

1. Implement the `Exchange` interface
2. Add API integration
3. Handle authentication
4. Update configuration

## Testing

```bash
# Run all tests
go test ./...

# Run specific test
go test ./internal/indicators -v

# Run with coverage
go test ./... -cover
```

## Deployment

### Docker Deployment

```bash
# Build image
docker build -t enhanced-dca-bot .

# Run container
docker run -d \
  --name dca-bot \
  -p 8080:8080 \
  -p 8081:8081 \
  -e EXCHANGE_API_KEY=your_key \
  -e EXCHANGE_SECRET=your_secret \
  enhanced-dca-bot
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: dca-bot
spec:
  replicas: 1
  selector:
    matchLabels:
      app: dca-bot
  template:
    metadata:
      labels:
        app: dca-bot
    spec:
      containers:
      - name: dca-bot
        image: enhanced-dca-bot:latest
        ports:
        - containerPort: 8080
        - containerPort: 8081
        env:
        - name: EXCHANGE_API_KEY
          valueFrom:
            secretKeyRef:
              name: dca-bot-secrets
              key: api-key
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Disclaimer

This software is for educational and research purposes only. Cryptocurrency trading involves substantial risk of loss and is not suitable for all investors. The authors are not responsible for any financial losses incurred through the use of this software.

## Support

For questions and support:
- Create an issue on GitHub
- Check the documentation
- Review the configuration examples

---

**Note**: Always test thoroughly in a testnet environment before using real funds.