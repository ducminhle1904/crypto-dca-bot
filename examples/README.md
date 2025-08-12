# Examples

```
cd crypto-dca-bot
make examples
```

## 📋 Available Examples

### 1. 📊 Data Analysis

Analyzes BTC market data including:

- Price and volume statistics
- Market volatility
- Price ranges
- Recent price movements

### 2. 🎯 Strategy Testing

Tests different trading strategies:

- Multi-Indicator Strategy
- Enhanced DCA Strategy
- Shows decisions and confidence levels

### 3. 📈 Backtesting

Runs historical strategy testing:

- Configurable parameters (balance, commissions)
- Trading results
- Return and risk analysis

### 4. 🎮 Interactive Trading

Simulates live trading:

- Step-by-step trade execution
- Position and balance tracking
- Interactive control

### 5. 📋 Performance Comparison

Compares strategy effectiveness:

- Total returns
- Maximum drawdown
- Number of trades

### 6. 🔧 Configuration Examples

Shows various settings:

- Conservative DCA
- Aggressive DCA
- Balanced DCA

## 📁 File Structure

```
examples/
├── main.go           # Main file with interactive examples
├── data_loader.go    # Data loader from Excel/CSV
├── Makefile          # Build and run commands
├── README.md         # This file
└── data/             # Data directory
    ├── bitcoin.xlsx  # Bitcoin price data
    └── historical/   # Historical data files
```

## 📊 Data Sources

Examples can use data from the following sources:

1. **Excel files** (`examples/data/bitcoin.xlsx`)
2. **CSV files** (`examples/data/historical/BTCUSDT_1h.csv`)
3. **Generated data** (fallback)

### Data Format

Expected CSV format:

```csv
timestamp,open,high,low,close,volume
1640995200,46200.50,46500.00,46000.00,46350.25,1250.5
```

## 🛠️ Makefile Commands

```bash
make build     # Build executable
make run       # Build and run
make run-dev   # Run directly with go run
make clean     # Clean build artifacts
make test      # Run tests
make help      # Show help
make examples  # Run interactive examples from project root
```

## 🔧 Configuration

### Adding BTC Data

1. Place `bitcoin.xlsx` file in `examples/data/` directory
2. Or create CSV file `examples/data/historical/BTCUSDT_1h.csv`
3. Restart examples

### Parameter Settings

In `main.go` file you can change:

- Initial balance
- Position size
- Commissions
- Analysis periods

### Data Path Configuration

You can specify the data directory in three ways (priority order):

1. **Command-line flag:**
   ```bash
   go run main.go --data examples/data
   ```
2. **Environment variable:**
   ```bash
   export DATA_PATH=examples/data
   make examples
   ```
3. **Default:**
   If not set, uses `examples/data`.

## 📈 Usage Examples

### Market Data Analysis

```bash
# Select option 1 in interactive menu
# Get BTC statistics
```

### Strategy Testing

```bash
# Select option 2
# Compare different strategy decisions
```

### Backtesting

```bash
# Select option 3
# Enter parameters:
# - Initial balance: $10000
# - Commission: 0.001 (0.1%)
# - Analysis window: 100
```

## 🎯 Features

- **Interactivity**: Step-by-step execution with user input
- **Realistic Data**: Generation of plausible market data
- **Validation**: Verification of loaded data correctness
- **Flexibility**: Parameter configuration for various scenarios
- **Visualization**: Emojis and formatted output for better perception

## 🔍 Debugging

For debugging use:

```bash
# Run with detailed logging
go run main.go data_loader.go 2>&1 | tee debug.log

# Check data
echo "1" | go run main.go data_loader.go  # Data analysis
```

## 📝 Notes

- All trading operations are simulated
- Data is generated deterministically for reproducibility
- Examples are for educational purposes
- For production use real APIs and data

## 🤝 Contributing

To add new examples:

1. Create a new function in `main.go`
2. Add option to main menu
3. Update documentation
4. Add tests if necessary

## 📞 Support

If you encounter problems:

1. Check for data files presence
2. Ensure data format correctness
3. Check logs for errors
4. Create issue in project repository
