# Enhanced DCA Bot

This bot implements the **Enhanced DCA (Dollar Cost Averaging)** strategy with multiple technical indicators for live trading. It uses the same logic as the backtest system and loads **optimized configuration files** for optimal performance.

## 🚀 Features

- **Enhanced DCA Strategy**: Implements the same logic as the backtest system
- **Optimized Parameters**: Loads best-performing configurations from backtest results
- **Price Threshold**: Prevents DCA entries until price drops by a minimum percentage
- **Take-Profit Cycles**: Automatically completes cycles when profit targets are reached
- **Multiple Technical Indicators**: RSI, MACD, Bollinger Bands, and EMA with optimized settings
- **Risk Management**: Position sizing and validation
- **Telegram Notifications**: Real-time alerts for trades and cycle completions
- **Health Monitoring**: Prometheus metrics and health checks
- **Exchange Integration**: Configurable for testnet/production

## 📊 Strategy Logic

The Enhanced DCA strategy works as follows:

1. **Indicator Consensus**: Buys when multiple technical indicators show buy signals
2. **Price Threshold**: Only enters new positions when price has dropped by the threshold amount
3. **Position Sizing**: Adjusts trade size based on signal strength
4. **Cycle Management**: Tracks DCA cycles and completes them at profit targets
5. **Risk Control**: Validates all trades through risk management

### Technical Indicators (Optimized)

The bot automatically loads optimized parameters for each indicator:

- **RSI**: Custom periods and oversold/overbought levels
- **MACD**: Optimized fast/slow/signal periods
- **Bollinger Bands**: Custom periods and standard deviation
- **EMA**: Optimized periods for trend detection

## 🛠️ Configuration

### Optimized Configuration Files

The bot automatically loads optimized configurations from the `configs/` directory:

```
configs/
├── btc_15m.json    # Bitcoin 15m optimized config
├── eth_30m.json    # Ethereum 30m optimized config
├── ada_5m.json     # Cardano 5m optimized config
├── sol_1h.json     # Solana 1h optimized config
├── doge_15m.json   # Dogecoin 15m optimized config
├── xrp_5m.json     # Ripple 5m optimized config
├── sui_5m.json     # Sui 5m optimized config
└── arb_5m.json     # Arbitrum 5m optimized config
```

**Important**: The bot automatically extracts the **data interval** from the config filename:

- `btc_15m.json` → Uses 15m data for analysis
- `eth_30m.json` → Uses 30m data for analysis
- `ada_5m.json` → Uses 5m data for analysis
- `sol_1h.json` → Uses 1h data for analysis

This ensures the bot analyzes market data at the same timeframe that was used during backtesting optimization.

### Environment Variables

```bash
# Exchange Configuration
EXCHANGE_API_KEY=your_binance_api_key
EXCHANGE_SECRET=your_binance_secret
EXCHANGE_TESTNET=true  # Set to false for production

# Telegram Notifications (Optional)
TELEGRAM_TOKEN=your_telegram_bot_token
TELEGRAM_CHAT_ID=your_chat_id

# Monitoring
PROMETHEUS_PORT=8080
HEALTH_PORT=8081
```

**Note**: Trading parameters (symbol, base amount, indicators, etc.) are automatically loaded from the optimized config file.

## 🚀 Running the Bot

### 1. Build the Bot

```bash
cd cmd/enhanced-dca-bot
make build
```

### 2. Set Environment Variables

```bash
export EXCHANGE_API_KEY="your_api_key"
export EXCHANGE_SECRET="your_secret"
export TELEGRAM_TOKEN="your_token"
export TELEGRAM_CHAT_ID="your_chat_id"
```

### 3. Run with Optimized Config

#### Quick Commands (Recommended)

```bash
# Bitcoin 15m optimized config
make run-btc

# Ethereum 30m optimized config
make run-eth

# Cardano 5m optimized config
make run-ada

# Solana 1h optimized config
make run-sol

# Dogecoin 15m optimized config
make run-doge

# Ripple 5m optimized config
make run-xrp

# Sui 5m optimized config
make run-sui

# Arbitrum 5m optimized config
make run-arb
```

#### Custom Config

```bash
# Run with specific config file
./enhanced-dca-bot -config=btc_15m
./enhanced-dca-bot -config=eth_30m
./enhanced-dca-bot -config=your_custom_config

# List available configs
make list-configs
```

### 4. Monitor

- Health: http://localhost:8081/health
- Metrics: http://localhost:8080/metrics

## 📈 How It Works

### Trading Cycle

1. **Config Loading**: Automatically loads optimized parameters from config file
2. **Interval Detection**: Extracts data interval from config filename (e.g., 15m, 30m, 1h)
3. **Market Analysis**: Fetches current market data at the detected interval
4. **Signal Generation**: Combines multiple indicator signals with optimized settings
5. **Decision Making**: Determines whether to buy, sell, or hold
6. **Risk Validation**: Checks if the trade meets risk management criteria
7. **Order Execution**: Places market orders when conditions are met
8. **Cycle Tracking**: Monitors for take-profit opportunities

### DCA Cycle Management

- **Cycle Start**: Begins when first buy order is placed
- **Entry Tracking**: Records entry price for the current cycle
- **Take Profit**: Automatically completes cycle at optimized profit target
- **Cycle Reset**: Starts new cycle with fresh strategy state
- **Performance Logging**: Tracks cycle duration and profitability

### Price Threshold Logic

The bot implements an optimized price threshold to prevent premature DCA entries:

- **First Entry**: No threshold restriction
- **Subsequent Entries**: Only allowed when price drops by optimized threshold
- **Threshold Value**: Automatically loaded from optimized config
- **Purpose**: Ensures meaningful price drops before additional investments

## 🔍 Monitoring

### Health Endpoint

```
GET /health
```

Returns bot health status including:

- Connection status
- Last trade time
- Current price
- Error count

### Prometheus Metrics

```
GET /metrics
```

Provides metrics for:

- Trade execution
- Strategy decisions
- Performance indicators
- Error rates

## 📱 Telegram Notifications

The bot sends real-time notifications for:

- **Bot Startup**: Configuration, optimized parameters, and cycle start
- **Trade Execution**: Buy order details and cycle information
- **Cycle Completion**: Performance summary and new cycle start
- **Bot Shutdown**: Final status and cycle summary

## 🧪 Testing

### Testnet Mode

By default, the bot runs in testnet mode:

- Uses Binance testnet API
- Simulated trading environment
- Safe for testing strategies

### Production Mode

To run in production:

```bash
export EXCHANGE_TESTNET=false
export EXCHANGE_API_KEY="production_key"
export EXCHANGE_SECRET="production_secret"
```

## 🔒 Risk Management

The bot includes several risk management features:

- **Position Size Limits**: Maximum trade size relative to balance
- **Balance Validation**: Ensures sufficient funds before trading
- **Multiplier Caps**: Limits position size multipliers (from optimized config)
- **Cycle Management**: Prevents over-investment in single cycles

## 📊 Performance Tracking

The bot tracks:

- **Cycle Performance**: Entry/exit prices and duration
- **Trade Statistics**: Success rate and average returns
- **Risk Metrics**: Position sizes and exposure levels
- **Strategy Effectiveness**: Indicator consensus and signal strength

## 🚨 Important Notes

1. **Demo Mode**: Currently uses simulated orders (ready for real API integration)
2. **Risk Warning**: Cryptocurrency trading involves significant risk
3. **Testing**: Always test thoroughly in testnet before live trading
4. **Monitoring**: Monitor bot performance and adjust parameters as needed
5. **Backup**: Keep backup of configuration and API keys
6. **Optimized Configs**: Uses the same parameters that performed best in backtesting

## 🔧 Customization

### Using Your Own Optimized Config

1. **Create Custom Config**: Add your optimized config to the `configs/` directory
2. **Run with Custom Config**: `./enhanced-dca-bot -config=your_config_name`
3. **Format**: Follow the same JSON structure as existing configs

### Strategy Parameters

Modify the `createEnhancedDCAStrategy` function to adjust:

- Indicator parameters (periods, thresholds)
- Price threshold values
- Take-profit percentages
- Risk management rules

### Adding Indicators

To add new indicators:

```go
// In createEnhancedDCAStrategy function
newIndicator := indicators.NewYourIndicator(params...)
dca.AddIndicator(newIndicator)
```

### Risk Management

Customize risk rules in the `risk` package:

- Position size calculations
- Balance requirements
- Multiplier limits
- Stop-loss logic

## 📚 Related Documentation

- [Backtesting Guide](../docs/BACKTESTING_GUIDE.md)
- [DCA Backtesting Guide](../docs/DCA_BACKTESTING_GUIDE.md)
- [Main Bot Documentation](../bot/README.md)

## 🤝 Contributing

To contribute to the Enhanced DCA Bot:

1. Test changes thoroughly in backtest mode
2. Follow existing code patterns and structure
3. Add appropriate logging and error handling
4. Update documentation for new features
5. Ensure backward compatibility

## 📄 License

This project is licensed under the same terms as the main crypto-dca-bot project.
