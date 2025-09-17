# 🚀 Multi-Bot Portfolio Launcher

The **Portfolio Launcher** enables you to run and manage multiple DCA trading bots in a coordinated portfolio, with centralized monitoring, risk management, and profit sharing.

## ✨ Features

- 🤖 **Multi-Bot Coordination**: Launch and manage multiple bots simultaneously
- 💼 **Portfolio Management**: Centralized balance allocation and profit sharing
- 📊 **Real-Time Monitoring**: Health checks, status updates, and alerts
- 🛡️ **Risk Management**: Portfolio-level exposure limits and emergency stops
- 🔄 **Graceful Shutdown**: Coordinated shutdown of all bots
- 📈 **Performance Tracking**: Individual bot and portfolio-level metrics

## 🏗️ Architecture

```
Portfolio Launcher
├── Portfolio Config Parser     # Reads multi-bot configurations
├── Bot Manager                 # Manages bot lifecycles
├── Portfolio Monitor          # Health checks and alerting
├── CLI Interface              # Command-line interface
└── Graceful Shutdown          # Coordinated cleanup
```

## 📋 Quick Start

### 1. **Build the Launcher**

```bash
cd cmd/portfolio-launcher
go build -o portfolio-launcher
```

### 2. **View Portfolio Configuration**

```bash
./portfolio-launcher -portfolio aave_hype_equal_weight.json -status
```

### 3. **Start Portfolio (Demo Mode)**

```bash
./portfolio-launcher -portfolio aave_hype_equal_weight.json -demo
```

### 4. **Start Portfolio (Live Trading)**

```bash
./portfolio-launcher -portfolio aave_hype_equal_weight.json -demo=false
```

## 🎛️ Command Line Options

| Flag         | Default    | Description                    |
| ------------ | ---------- | ------------------------------ |
| `-portfolio` | _required_ | Portfolio configuration file   |
| `-demo`      | `true`     | Use demo/testnet mode          |
| `-exchange`  | ""         | Override exchange for all bots |
| `-env`       | `.env`     | Environment file path          |
| `-status`    | `false`    | Show portfolio status only     |
| `-detailed`  | `false`    | Show detailed monitoring       |
| `-monitor`   | `30s`      | Status update interval         |

## 📁 Portfolio Configuration

Portfolio configurations define multiple bots and their coordination:

```json
{
  "description": "AAVE + HYPE Equal Weight Portfolio",
  "bots": [
    {
      "bot_id": "aave_bot",
      "config_file": "configs/portfolio/individual/aave_10x.json",
      "enabled": true
    },
    {
      "bot_id": "hype_bot",
      "config_file": "configs/portfolio/individual/hype_10x.json",
      "enabled": true
    }
  ],
  "portfolio": {
    "total_balance": 1000.0,
    "allocation_strategy": "equal_weight",
    "shared_state_file": "portfolio_state.json",
    "max_total_exposure": 3.0,
    "max_drawdown_percent": 25.0,
    "profit_sharing": {
      "enabled": true,
      "method": "equal_weight"
    }
  },
  "monitoring": {
    "heartbeat_interval": "30s",
    "health_check_interval": "60s"
  }
}
```

## 🔧 Portfolio Settings

### **Allocation Strategies**

- `equal_weight`: Equal balance distribution across all bots
- `performance_based`: Allocate more to better performing bots
- `risk_weighted`: Allocate based on individual bot risk profiles
- `fixed_percentage`: Fixed allocation percentages per bot
- `custom`: User-defined allocation logic

### **Risk Management**

- **Max Total Exposure**: Maximum leverage across entire portfolio
- **Max Drawdown**: Emergency stop threshold for portfolio losses
- **Per-Bot Risk Limits**: Individual bot exposure limitations
- **Correlation Monitoring**: Detect when bots become too correlated

### **Profit Sharing**

- **Equal Weight**: Redistribute profits equally across all bots
- **Performance Based**: Reward better performing strategies
- **Rebalance Threshold**: Trigger rebalancing when allocation drifts

## 📊 Monitoring & Alerts

### **Health Checks**

- ✅ Bot status monitoring (running, stopped, error)
- 🔍 Error rate tracking and thresholds
- 📡 Heartbeat monitoring for responsiveness
- 💰 Balance and position synchronization

### **Portfolio Alerts**

- 🚨 **Critical**: No bots running, system failures
- ⚠️ **Warning**: High error rates, allocation drift
- ℹ️ **Info**: Bot restarts, configuration changes

### **Status Display**

```
╔══════════════════════════════════════════════════════════════╗
║                    PORTFOLIO STATUS REPORT                  ║
╠══════════════════════════════════════════════════════════════╣
║ Portfolio: AAVE + HYPE Equal Weight Portfolio               ║
║ Uptime: 2h 15m 30s                                         ║
║ Bots: 2/2 running                                          ║
╠══════════════════════════════════════════════════════════════╣
║ 🟢 aave_bot        │ Status: running    │ Uptime: 2h 15m   ║
║ 🟢 hype_bot        │ Status: running    │ Uptime: 2h 15m   ║
╚══════════════════════════════════════════════════════════════╝
```

## 🛠️ Usage Examples

### **Basic Portfolio Launch**

```bash
# Show configuration
./portfolio-launcher -portfolio aave_hype_equal_weight.json -status

# Start in demo mode
./portfolio-launcher -portfolio aave_hype_equal_weight.json

# Start with detailed monitoring
./portfolio-launcher -portfolio aave_hype_equal_weight.json -detailed
```

### **Advanced Options**

```bash
# Override exchange for all bots
./portfolio-launcher -portfolio my_portfolio.json -exchange bybit

# Custom monitoring interval
./portfolio-launcher -portfolio my_portfolio.json -monitor 60s

# Live trading (disable demo)
./portfolio-launcher -portfolio my_portfolio.json -demo=false
```

## 🔐 Security & API Keys

Ensure your API credentials are set in environment variables or `.env` file:

```bash
# .env file
BYBIT_API_KEY=your_bybit_api_key
BYBIT_API_SECRET=your_bybit_api_secret
BINANCE_API_KEY=your_binance_api_key
BINANCE_API_SECRET=your_binance_api_secret
```

## 🎯 Best Practices

### **Portfolio Design**

1. **Diversification**: Use different symbols, timeframes, or strategies
2. **Risk Management**: Set appropriate exposure limits and drawdown thresholds
3. **Monitoring**: Regular health checks and alert monitoring
4. **Testing**: Always test in demo mode before live trading

### **Configuration Management**

1. **Version Control**: Keep portfolio configs in version control
2. **Environment Separation**: Separate configs for demo/live environments
3. **Backup**: Regular backups of portfolio state files
4. **Documentation**: Document portfolio strategies and rationale

### **Operational Procedures**

1. **Gradual Rollout**: Start with small allocations and scale up
2. **Monitoring**: Active monitoring during initial deployment
3. **Emergency Procedures**: Know how to quickly stop all bots
4. **Performance Review**: Regular analysis of portfolio performance

## 🚨 Emergency Procedures

### **Emergency Stop**

```bash
# Send interrupt signal (Ctrl+C or SIGTERM)
kill -TERM <launcher_pid>

# Or use process management
pkill -f portfolio-launcher
```

### **Individual Bot Control**

Currently, individual bot control requires stopping the entire portfolio. Future versions will support:

- Individual bot start/stop
- Runtime configuration updates
- Hot-swapping of strategies

## 🧪 Testing

### **Demo Mode Testing**

```bash
# Test with your actual portfolio config
./portfolio-launcher -portfolio aave_hype_equal_weight.json -demo

# Verify all bots start successfully
./portfolio-launcher -portfolio aave_hype_equal_weight.json -status -detailed
```

### **Configuration Validation**

The launcher validates:

- ✅ Portfolio configuration syntax
- ✅ Individual bot config file existence
- ✅ API credential availability
- ✅ Risk parameter ranges
- ✅ Allocation strategy validity

## 📈 Future Enhancements

- 🌐 **Web Dashboard**: Real-time portfolio monitoring interface
- 🔄 **Hot Reloading**: Update configurations without restart
- 📊 **Advanced Analytics**: Performance attribution and risk metrics
- 🎛️ **Dynamic Allocation**: ML-based allocation adjustments
- 🔗 **External Integrations**: Slack/Discord notifications, webhooks
- 💾 **Database Storage**: Enhanced state persistence and history

## 🆘 Troubleshooting

### **Common Issues**

**Bots fail to start:**

- Check API credentials in environment
- Verify individual bot configurations
- Ensure sufficient account balance
- Check exchange connectivity

**Portfolio coordination issues:**

- Verify shared state file permissions
- Check for file locking conflicts
- Monitor memory and CPU usage
- Review bot error logs

**Performance issues:**

- Reduce monitoring intervals
- Optimize bot configurations
- Check network connectivity
- Monitor system resources

### **Getting Help**

1. **Logs**: Check individual bot logs in `logs/` directory
2. **Status**: Use `-detailed` flag for comprehensive status
3. **Configuration**: Validate with `-status` flag before starting
4. **Demo Mode**: Test thoroughly before live trading

---

**⚠️ Important**: Always test thoroughly in demo mode before deploying to live trading. Multi-bot portfolios can amplify both profits and losses.
