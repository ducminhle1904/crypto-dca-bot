# Enhanced DCA Live Bot

A live trading bot that executes Dollar Cost Averaging (DCA) strategies using Bybit exchange with built-in paper trading support and technical indicators.

## 🚀 Features

- **Bybit Integration**: Optimized for Bybit exchange with demo mode support
- **Paper Trading**: Demo mode enabled by default for safe testing
- **Multi-indicator Analysis**: RSI, MACD, Bollinger Bands, EMA, SMA
- **Configuration-driven**: Uses config files for strategy parameters
- **Risk Management**: Position sizing, take-profit levels, DCA multipliers
- **Safety Features**: Demo mode (paper trading), timestamp sync, balance checks
- **Live Monitoring**: Real-time status updates and trade logging

## 📋 Prerequisites

1. **Bybit API Credentials**:

   - Sign up for Bybit (demo mode enabled by default for safe testing)
   - Generate API key and secret
   - Set appropriate permissions (spot trading, futures trading)

2. **API Credentials** (Choose one method):

   **Method 1: .env file (Recommended)**:

   ```bash
   # Copy the example file
   cp env.example .env

   # Edit .env file with your credentials
   BYBIT_API_KEY=your_api_key_here
   BYBIT_API_SECRET=your_api_secret_here
   ```

   **Method 2: Environment Variables**:

   ```bash
   export BYBIT_API_KEY="your_api_key_here"
   export BYBIT_API_SECRET="your_api_secret_here"
   ```

3. **Configuration Files**: Available in `configs/` directory

## 🔧 Usage

### Basic Usage

```bash
# Run with a specific config (demo mode - paper trading)
go run cmd/live-bot/main.go -config btc_5.json

# Same as above (defaults to demo mode - paper trading)
go run cmd/live-bot/main.go -config btc_5
```

### Command Line Options

| Flag      | Description                      | Default |
| --------- | -------------------------------- | ------- |
| `-config` | Configuration file (required)    | -       |
| `-demo`   | Use demo trading (paper trading) | `true`  |
| `-env`    | Environment file path            | `.env`  |

### Advanced Usage

```bash
# Run live trading with real money (⚠️ Use real API keys)
go run cmd/live-bot/main.go -config btc_5.json -demo=false

# Use custom .env file
go run cmd/live-bot/main.go -config btc_5.json -env=production.env
```

### Production Trading

⚠️ **WARNING: Only use with small amounts and proper risk management**

```bash
# Step 1: Demo trading environment (paper trading - SAFE)
go run cmd/live-bot/main.go -config btc_5.json -demo=true

# Step 2: Live trading with real money (use with extreme caution)
go run cmd/live-bot/main.go -config btc_5.json -demo=false
```

## 📁 Configuration Files

The bot reads strategy parameters from JSON config files in the `configs/` directory:

### Available Configs

| File           | Symbol  | Interval | Category | Description                 |
| -------------- | ------- | -------- | -------- | --------------------------- |
| `btc_15m.json` | BTCUSDT | 15m      | Linear   | Bitcoin 15-minute strategy  |
| `eth_30m.json` | ETHUSDT | 30m      | Spot     | Ethereum 30-minute strategy |
| `ada_5m.json`  | ADAUSDT | 5m       | Spot     | Cardano 5-minute strategy   |
| `sol_1h.json`  | SOLUSDT | 1h       | Linear   | Solana 1-hour strategy      |

### Config File Structure

```json
{
  "symbol": "BTCUSDT",
  "initial_balance": 500,
  "commission": 0.0005,
  "window_size": 100,
  "base_amount": 100,
  "max_multiplier": 1.5,
  "price_threshold": 0.01,
  "rsi_period": 10,
  "rsi_oversold": 25,
  "rsi_overbought": 70,
  "macd_fast": 6,
  "macd_slow": 26,
  "macd_signal": 7,
  "bb_period": 16,
  "bb_std_dev": 2.5,
  "ema_period": 20,
  "indicators": ["rsi", "macd", "bb", "ema"],
  "tp_percent": 0.04,
  "cycle": true
}
```

### Configuration Parameters

#### Trading Parameters

- **`symbol`**: Trading pair (e.g., "BTCUSDT")
- **`initial_balance`**: Starting balance for position tracking
- **`commission`**: Trading fee percentage (0.0005 = 0.05%)
- **`base_amount`**: Base DCA purchase amount ($)
- **`max_multiplier`**: Maximum DCA multiplier (1.5 = 150% of base)
- **`price_threshold`**: Minimum price drop % for next DCA entry
- **`tp_percent`**: Take profit percentage (0.04 = 4%)

#### Technical Indicators

- **`rsi_period`**: RSI calculation period
- **`rsi_oversold`**: RSI oversold threshold
- **`rsi_overbought`**: RSI overbought threshold
- **`macd_fast`**: MACD fast EMA period
- **`macd_slow`**: MACD slow EMA period
- **`macd_signal`**: MACD signal line period
- **`bb_period`**: Bollinger Bands period
- **`bb_std_dev`**: Bollinger Bands standard deviation
- **`ema_period`**: EMA period

#### Strategy Settings

- **`window_size`**: Number of historical candles to analyze
- **`indicators`**: List of indicators to use
- **`cycle`**: Enable take-profit cycles

## 🏗️ Trading Environments

### Demo Trading (Recommended for Testing)

The **demo environment** is perfect for testing your strategies with paper trading:

```bash
# Use demo trading environment with your demo account credentials
go run cmd/live-bot/main.go -config btc_5.json -demo=true -dry-run=false
```

**Benefits:**

- ✅ **Paper Trading**: No real money involved
- ✅ **Real Market Data**: Uses live price feeds
- ✅ **Full API Testing**: Tests all API integrations
- ✅ **Safe Learning**: Perfect for learning and testing strategies

### Environment Options

| Environment | Flag             | Description                   | Use Case                   |
| ----------- | ---------------- | ----------------------------- | -------------------------- |
| **Demo**    | `-demo=true`     | Paper trading with demo API   | Testing strategies safely  |
| **Testnet** | `-testnet=true`  | Bybit testnet with test funds | Testing with virtual funds |
| **Mainnet** | `-testnet=false` | Live trading with real money  | Production trading         |

**Your demo account credentials work with `-demo=true` flag!** 🎯

## 🎯 How It Works

### 1. Initialization

- Loads configuration from JSON file
- Extracts symbol and timeframe from filename
- Creates Bybit API client
- **Syncs with real account balance** (when API credentials provided)
- Initializes technical indicators
- Sets up DCA strategy

### 2. Market Analysis

- Fetches current price and recent klines
- Runs technical indicator analysis
- Determines market conditions (BUY/SELL/HOLD)
- Applies DCA logic and risk management

### 3. Position Management

- **DCA Entries**: Incremental purchases based on price drops
- **Position Sizing**: Multiplier increases with each DCA level
- **Take Profit**: Sells entire position when target reached
- **Risk Limits**: Respects balance and multiplier constraints

### 4. Live Monitoring

```
[2025-01-18 15:30:15] 📊 Market Status
💰 Price: $94,250.50 | Action: BUY
💼 Balance: $400.00 | Position: 0.001234 BTCUSDT
📈 Avg Price: $93,800.25 | Current Value: $116.32
📊 Unrealized P&L: $5.55 (0.48%) | DCA Level: 2
```

## 🔍 Example Output

### Dry Run Mode

```bash
🚀 Enhanced DCA Live Bot Starting...
📊 Symbol: BTCUSDT
⏰ Interval: 15
🏪 Category: linear
🔧 Testnet: true
🧪 Dry Run: true
==================================================

💰 Account Balance Sync:
   Config Balance: $500.00
   Real Balance:   $1,250.75 (USDT)
   Account Type:   UNIFIED
   Category:       linear
✅ Real balance is higher than config balance

🔄 Starting live bot for BTCUSDT/15
💰 Initial Balance: $1,250.75
📈 Base DCA Amount: $100.00
🔄 Max Multiplier: 1.50
📊 Price Threshold: 1.00%
🎯 Take Profit: 4.00%
🧪 DRY RUN MODE - No actual trades will be executed
============================================================

[2025-01-18 15:30:15] 📊 Market Status
💰 Price: $94,250.50 | Action: BUY
💼 Balance: $1,250.75 | Position: 0.000000 BTCUSDT
🧪 DRY RUN: Would execute BUY at $94250.50
```

### Live Trading

```bash
✅ BUY ORDER EXECUTED
   Order ID: 1234567890
   Quantity: 0.001061 BTCUSDT
   Price: $94250.50
   Amount: $100.00
   DCA Level: 1

✅ SELL ORDER EXECUTED
   Order ID: 1234567891
   Quantity: 0.003184 BTCUSDT
   Price: $98020.52
   Sale Value: $312.00
   Profit: $12.00 (4.00%)
```

## 💰 Account Balance Integration

### Real Balance Sync

When you provide valid API credentials, the bot automatically syncs with your real Bybit account balance:

#### **Balance Comparison**

- **Config Balance**: From the `initial_balance` in your JSON config
- **Real Balance**: Fetched from your actual Bybit account (USDT)
- **Used Balance**: The bot uses the **real balance** for trading decisions

#### **Account Types Supported**

- **UNIFIED**: Primary account type for spot and futures trading
- **Auto-Detection**: Bot automatically detects the correct account type based on trading category

#### **Currency Mapping**

- **Spot Trading**: Uses USDT balance
- **Linear Futures**: Uses USDT balance
- **Inverse Futures**: Uses USDT balance (for consistency)

#### **Real-time Updates**

- **Before Each Trade**: Balance is refreshed to ensure sufficient funds
- **After Each Sale**: Balance is updated to reflect the sale proceeds
- **Error Handling**: Falls back to config balance if API calls fail

### Example Balance Sync Output

```bash
💰 Account Balance Sync:
   Config Balance: $500.00
   Real Balance:   $1,250.75 (USDT)
   Account Type:   UNIFIED
   Category:       linear
✅ Real balance is higher than config balance
```

### Balance States

| Scenario      | Display         | Description                                 |
| ------------- | --------------- | ------------------------------------------- |
| Real > Config | ✅ Higher       | You have more funds available than expected |
| Real < Config | ⚠️ Lower        | You have less funds than expected           |
| Real = Config | ✅ Matches      | Balances are identical                      |
| API Error     | 💡 Using config | Falls back to config balance                |

## ⚠️ Safety & Risk Management

### Built-in Safety Features

1. **Demo Mode Default**: Always starts in paper trading mode (safe)
2. **Paper Trading**: Uses demo environment by default
3. **Balance Checks**: Prevents trades exceeding available balance
4. **Position Limits**: Respects maximum multiplier settings
5. **Error Handling**: Graceful error recovery and logging

### Risk Considerations

- **Start Small**: Use small amounts for testing
- **Monitor Closely**: Watch the bot during initial runs
- **Set Limits**: Configure appropriate multipliers and thresholds
- **Have Exit Plan**: Know how to stop the bot and close positions
- **Market Volatility**: DCA works best in ranging markets

### Emergency Stop

- **Ctrl+C**: Gracefully stops the bot
- **Manual Trading**: Close positions manually on exchange if needed

## 🎛️ Advanced Configuration

### Creating Custom Configs

1. Copy an existing config file:

   ```bash
   cp configs/btc_15m.json configs/my_strategy.json
   ```

2. Modify parameters for your strategy

3. Test with dry-run:
   ```bash
   go run cmd/live-bot/main.go -config my_strategy.json
   ```

### Timeframe Mapping

- **5m** → Bybit interval "5"
- **15m** → Bybit interval "15"
- **30m** → Bybit interval "30"
- **1h** → Bybit interval "60"
- **4h** → Bybit interval "240"
- **1d** → Bybit interval "D"

### Market Categories

- **spot**: Spot trading
- **linear**: USDT-settled futures
- **inverse**: Coin-settled futures

## 📊 Performance Monitoring

The bot provides detailed logging of:

- Market conditions and price movements
- Trading decisions and reasoning
- Position updates and P&L calculations
- Error conditions and recovery attempts
- Real-time status every interval

## 🔧 Troubleshooting

### Common Issues

1. **API Credentials**:

   ```bash
   Please set BYBIT_API_KEY and BYBIT_API_SECRET in .env file or environment variables
   ```

   **Solution**: Create a .env file with your Bybit API credentials or set environment variables

2. **Config File Not Found**:

   ```bash
   Failed to read config file configs/invalid.json
   ```

   **Solution**: Ensure config file exists and is valid JSON

3. **Insufficient Balance**:

   ```bash
   ⚠️ Insufficient balance: $50.00 < $100.00
   ```

   **Solution**: Reduce base_amount or add more funds

4. **Network Errors**:
   ```bash
   ❌ Failed to get current price: connection timeout
   ```
   **Solution**: Check internet connection and Bybit API status

### Debug Mode

Enable verbose logging by modifying log levels in the code or using environment variables.

## 🔧 Setup Guide

### 1. Get Bybit API Credentials

1. **Sign up for Bybit**:

   - Testnet: https://testnet.bybit.com/
   - Mainnet: https://www.bybit.com/

2. **Create API Key**:

   - Go to Account → API Management
   - Create new API key
   - Set permissions: Spot Trading, Derivatives Trading
   - **Save the API key and secret securely**

3. **Setup .env file**:

   ```bash
   # Copy the example
   cp env.example .env

   # Edit with your credentials
   nano .env  # or use any text editor
   ```

### 2. Test Configuration

```bash
# 1. Test with existing config
go run cmd/live-bot/main.go -config btc_5.json

# 2. Should show: credentials loaded, demo mode active
# 3. If successful, you'll see market data being fetched
```

### 3. Create Custom Strategy

```bash
# 1. Copy existing config
cp configs/btc_5.json configs/my_strategy.json

# 2. Edit parameters
nano configs/my_strategy.json

# 3. Test new strategy
go run cmd/live-bot/main.go -config my_strategy.json
```

## 🎯 Best Practices

1. **Start with Demo Mode**: Always test strategies in demo mode first (paper trading)
2. **Validate Strategy**: Test your strategy thoroughly before live trading
3. **Monitor Performance**: Track P&L and adjust parameters
4. **Risk Management**: Never risk more than you can afford to lose
5. **Regular Updates**: Keep strategy parameters updated based on market conditions
6. **Secure Credentials**: Never commit .env files to version control

The Enhanced DCA Live Bot brings together all the components we've built: Bybit integration, technical indicators, backtesting insights, and configuration management into a production-ready trading system! 🚀
