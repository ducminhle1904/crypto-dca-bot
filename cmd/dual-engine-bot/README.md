# 🚀 Dual Engine Bot

The **Dual Engine Bot** is an advanced trading system that automatically switches between different trading strategies based on real-time market regime detection.

## 🎯 Overview

The bot integrates:

- **Phase 1**: Regime Detection Foundation ✅
- **Phase 2B**: Grid Engine (mean-reversion hedge grid) ✅
- **Phase 2C**: Trend Engine (multi-timeframe trend following) ✅
- **Phase 2D**: Orchestrator (automatic engine selection) ✅

## 🏗️ Architecture

```
DualEngineBot Orchestrator
├── 🎯 Regime Detector (Phase 1)
├── 🔄 Grid Engine (ranging/volatile markets)
├── 📈 Trend Engine (trending markets)
└── 🔄 Engine Transition Manager
```

### 🤖 Engine Selection Logic

| **Market Regime** | **Primary Engine** | **Compatibility** | **Strategy**                         |
| ----------------- | ------------------ | ----------------- | ------------------------------------ |
| **TRENDING**      | Trend Engine       | 100%              | Multi-timeframe trend following      |
| **RANGING**       | Grid Engine        | 100%              | VWAP/EMA anchored hedge grid         |
| **VOLATILE**      | Grid Engine        | 80%               | Symmetric grid with hedge management |
| **UNCERTAIN**     | Grid Engine        | 60%               | Market-neutral grid approach         |

## 🚀 Usage

### Basic Usage

```bash
# Run with live trading
go run cmd/dual-engine-bot/main.go -config configs/bybit/btc_5m_bybit.json

# Run in demo mode (paper trading)
go run cmd/dual-engine-bot/main.go -config configs/bybit/btc_5m_bybit.json -demo

# Run with verbose status logging
go run cmd/dual-engine-bot/main.go -config configs/bybit/btc_5m_bybit.json -verbose
```

### Command Line Options

| Option           | Description                        | Example                                   |
| ---------------- | ---------------------------------- | ----------------------------------------- |
| `-config <file>` | Configuration file path (REQUIRED) | `-config configs/bybit/btc_5m_bybit.json` |
| `-demo`          | Run in demo mode (paper trading)   | `-demo`                                   |
| `-verbose`       | Enable verbose status logging      | `-verbose`                                |
| `-help`          | Show help information              | `-help`                                   |

## 📊 Live Monitoring

The bot provides real-time status updates:

```
🎯 Current Regime: TRENDING
🏗️ Active Engine: Multi-Timeframe Trend Engine

🔧 ENGINE STATUS:
  ▶️ TREND: 🟢 ACTIVE (Positions: 1)
  ⏸️ DCA: 🔴 INACTIVE (Positions: 0)

📈 SESSION METRICS:
  • Regime Changes: 3
  • Total Signals: 15
  • Session Duration: 2h 15m
  • Total P&L: $45.20
```

## 🔄 Engine Switching

The bot automatically switches engines when:

1. **Regime Change Detected**: Market regime changes (e.g., RANGING → TRENDING)
2. **Confidence Threshold Met**: Regime confidence > 70%
3. **Cooldown Period Elapsed**: At least 15 minutes since last switch
4. **Compatibility Score**: New engine has better compatibility (>50% improvement)

### Example Engine Switch Log:

```
🔄 REGIME CHANGE: RANGING → TRENDING (Confidence: 85%)
🔄 ENGINE SWITCH: Enhanced DCA Engine → Multi-Timeframe Trend Engine (TRENDING regime)
```

## ⚙️ Configuration

Uses the same configuration format as the original LiveBot. The dual engine system adds automatic engine selection on top of the existing configuration.

### Key Configuration Sections:

- **Trading**: Symbol, interval, exchange settings
- **Strategy**: Multi-engine parameters (grid spacing, trend thresholds)
- **Risk**: Position limits, daily loss limits
- **Exchange**: API credentials, demo mode

## 🛡️ Risk Management

### Engine-Level Risk Controls:

- **Grid Engine**: Max 80% total exposure, 10% max net exposure, symmetric hedging
- **Trend Engine**: ATR-based stops, 30% max position size, pullback entries

### Global Risk Controls:

- **Portfolio Level**: 80% max total position size
- **Daily Limits**: 5% max daily loss
- **Emergency Stop**: 15% portfolio drawdown limit

### Transition Safety:

- **Cooldown Period**: 15 minutes between engine switches
- **Position Compatibility**: Evaluates existing positions before switching
- **Gradual Transitions**: Minimizes switching costs

## 📈 Performance Tracking

### Per-Engine Metrics:

- Win rate and profit factor
- Average trade duration
- Total PnL contribution
- Activation frequency

### Orchestrator Metrics:

- Engine switching efficiency
- Regime detection accuracy
- Total system performance
- Transition costs

## 🧪 Testing Modes

### Demo Mode (`-demo`)

- **Paper Trading**: No real money at risk
- **Full Functionality**: Complete regime detection and engine switching
- **Live Data**: Uses real market data
- **Safe Testing**: Perfect for validating strategies

### Verbose Mode (`-verbose`)

- **Detailed Logging**: Enhanced status information
- **Real-time Updates**: 30-second status updates
- **Regime Changes**: Immediate notification of regime switches
- **Engine Transitions**: Detailed engine switch logging

## 🔧 Advanced Features

### Grid Engine Features:

- **VWAP Anchoring**: Uses Volume-Weighted Average Price for grid center
- **EMA 100 Anchor**: Combines with VWAP for weighted anchor price (60%/40%)
- **ATR-Based Spacing**: Grid levels spaced at 0.25x ATR multiplier
- **Symmetric Hedging**: Balanced long/short exposure with 10% max net exposure
- **Safety Mechanisms**: BB width monitoring, ADX trend popup detection

### Multi-timeframe Analysis:

- **30m Bias**: Higher timeframe trend direction
- **5m Execution**: Entry timing and signal generation
- **1m Precision**: Fine-tuned entry points (future enhancement)

### Regime-Aware Position Sizing:

- **High Compatibility**: Larger position sizes
- **Low Compatibility**: Reduced position sizes
- **Regime Uncertainty**: Conservative sizing

### Intelligent Engine Selection:

- **Compatibility Scoring**: Each engine rates regime fitness (0-100%)
- **Stability Bonus**: 10% bonus for currently active engine
- **Confidence Filtering**: Only switch on high-confidence regime changes

## 🚨 Emergency Controls

### Graceful Shutdown:

- **Ctrl+C**: Initiates graceful shutdown
- **Position Safety**: Maintains existing positions
- **Order Management**: Cancels pending orders if configured

### Circuit Breakers:

- **Daily Loss Limit**: Stops trading if daily loss exceeded
- **Drawdown Protection**: Emergency flattening at 15% drawdown
- **System Errors**: Automatic fallback to safe mode

## 📋 Requirements

- **Go 1.19+**: Required for compilation
- **Exchange API Access**: Valid API credentials
- **Configuration File**: Properly configured JSON file
- **Network Connectivity**: Stable internet connection

## 🎉 Success Metrics

Based on our **Phase 1 validation** with 279,975 BTCUSDT data points:

- ✅ **Regime Detection**: 99.2% stability, 0.09 false signals/hour
- ✅ **Grid Engine**: VWAP/EMA anchored hedge system for ranging markets
- ✅ **Trend Engine**: Multi-timeframe trend following for directional markets
- ✅ **Risk Management**: Multiple layers of protection with engine-specific controls
- ✅ **System Reliability**: Enterprise-grade concurrency and error handling

## 🎯 **Original Vision Realized**

The Dual Engine Bot represents the **true implementation of the original plan**:

- **🔄 Grid Engine**: Mean-reversion hedge trading for ranging/volatile markets
- **📈 Trend Engine**: Multi-timeframe trend following for trending markets
- **🎯 Regime Detection**: Intelligent market condition classification
- **🤖 Orchestration**: Automatic engine selection based on regime compatibility

**Note**: The DCA system continues to run independently as the proven LiveBot for users who prefer the traditional DCA approach. The Dual Engine Bot offers a sophisticated alternative with two complementary strategies working in harmony.
