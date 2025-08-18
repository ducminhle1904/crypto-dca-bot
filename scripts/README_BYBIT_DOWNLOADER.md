# Bybit Historical Data Downloader

A comprehensive tool for downloading historical kline (candlestick) data from Bybit exchange, supporting spot, linear futures, and inverse futures markets.

## 🚀 Features

- **Multiple Market Categories**: Spot, Linear Futures, Inverse Futures
- **Batch Downloads**: Multiple symbols, intervals, and categories in one run
- **Flexible Time Ranges**: Custom start and end dates
- **All Time Intervals**: From 1 minute to 1 month
- **Rate Limiting**: Built-in rate limiting to respect API limits
- **CSV Output**: Clean, organized CSV files with timestamps
- **Progress Tracking**: Real-time download progress
- **Data Validation**: Automatic data validation and error handling
- **Resume Support**: Intelligent handling of partial downloads

## 📊 Supported Markets

### Market Categories

- **`spot`**: Spot trading pairs (BTCUSDT, ETHUSDT, etc.)
- **`linear`**: USDT-settled perpetual futures
- **`inverse`**: Coin-settled perpetual futures

### Time Intervals

- **Minutes**: `1`, `3`, `5`, `15`, `30`
- **Hours**: `60` (1h), `120` (2h), `240` (4h), `360` (6h), `720` (12h)
- **Days**: `D` (1 day)
- **Weeks**: `W` (1 week)
- **Months**: `M` (1 month)

## 🔧 Usage

### Basic Usage

```bash
# Download 1 year of hourly BTCUSDT spot data
go run ./scripts/download_bybit_historical_data.go -symbol BTCUSDT -interval 60 -category spot

# Download specific date range
go run ./scripts/download_bybit_historical_data.go \
    -symbol BTCUSDT \
    -interval 240 \
    -category linear \
    -start 2024-01-01 \
    -end 2024-12-31

# Custom output directory
go run ./scripts/download_bybit_historical_data.go \
    -symbol ETHUSDT \
    -interval 60 \
    -category spot \
    -outdir /path/to/data \
    -start 2024-06-01 \
    -end 2024-06-30
```

### Batch Downloads

```bash
# Multiple symbols
go run ./scripts/download_bybit_historical_data.go \
    -symbols "BTCUSDT,ETHUSDT,SOLUSDT" \
    -interval 60 \
    -category spot

# Multiple intervals
go run ./scripts/download_bybit_historical_data.go \
    -symbol BTCUSDT \
    -intervals "60,240,D" \
    -category linear

# Multiple categories
go run ./scripts/download_bybit_historical_data.go \
    -symbol BTCUSDT \
    -interval 60 \
    -categories "spot,linear"

# Full batch mode - everything combined
go run ./scripts/download_bybit_historical_data.go \
    -symbols "BTCUSDT,ETHUSDT" \
    -intervals "60,240" \
    -categories "spot,linear" \
    -start 2024-01-01 \
    -end 2024-01-31
```

### Advanced Usage

```bash
# Single file output (only for single symbol/interval/category)
go run ./scripts/download_bybit_historical_data.go \
    -symbol BTCUSDT \
    -interval 60 \
    -category spot \
    -output btc_hourly.csv

# Large datasets with custom limits
go run ./scripts/download_bybit_historical_data.go \
    -symbol BTCUSDT \
    -interval 1 \
    -category spot \
    -start 2024-01-01 \
    -end 2024-01-02 \
    -limit 1000
```

## 📋 Command Line Options

| Flag          | Description                           | Default      | Example                      |
| ------------- | ------------------------------------- | ------------ | ---------------------------- |
| `-symbol`     | Single trading symbol                 | `BTCUSDT`    | `-symbol ETHUSDT`            |
| `-symbols`    | Multiple symbols (comma-separated)    | -            | `-symbols "BTCUSDT,ETHUSDT"` |
| `-interval`   | Single time interval                  | `60`         | `-interval 240`              |
| `-intervals`  | Multiple intervals (comma-separated)  | -            | `-intervals "60,240,D"`      |
| `-category`   | Single market category                | `spot`       | `-category linear`           |
| `-categories` | Multiple categories (comma-separated) | -            | `-categories "spot,linear"`  |
| `-start`      | Start date (YYYY-MM-DD)               | 1 year ago   | `-start 2024-01-01`          |
| `-end`        | End date (YYYY-MM-DD)                 | Today        | `-end 2024-12-31`            |
| `-outdir`     | Output directory                      | `data/bybit` | `-outdir /path/to/data`      |
| `-output`     | Single file output path               | -            | `-output btc.csv`            |
| `-limit`      | Klines per API request                | `1000`       | `-limit 500`                 |

## 📁 Output Structure

### Default Directory Structure

```
data/bybit/
├── spot/
│   ├── BTCUSDT/
│   │   ├── 60/
│   │   │   └── candles.csv
│   │   ├── 240/
│   │   │   └── candles.csv
│   │   └── D/
│   │       └── candles.csv
│   └── ETHUSDT/
│       └── 60/
│           └── candles.csv
├── linear/
│   └── BTCUSDT/
│       └── 60/
│           └── candles.csv
└── inverse/
    └── BTCUSD/
        └── 60/
            └── candles.csv
```

### CSV Format

```csv
timestamp,open,high,low,close,volume,turnover
2024-01-01 00:00:00,42500.50,42850.75,42400.25,42750.30,125.45,5367890.25
2024-01-01 01:00:00,42750.30,42900.00,42650.80,42825.60,98.32,4201234.50
```

## 📈 Example Output

```
🚀 Bybit Historical Data Downloader
====================================
📊 Categories: spot, linear
🎯 Symbols: BTCUSDT, ETHUSDT
⏱️  Intervals: 60, 240
📅 Date Range: 2024-01-01 to 2024-01-31

📊 Downloading spot 60 data for BTCUSDT
📅 Period: 2024-01-01 to 2024-01-31
📁 Output: data/bybit/spot/BTCUSDT/60/candles.csv
🔄 Fetching data...
  Progress: 744 klines downloaded...
✅ Downloaded 744 klines
💾 Data saved to data/bybit/spot/BTCUSDT/60/candles.csv

📊 DATA SUMMARY:
  First: 2024-01-01 00:00:00
  Last:  2024-01-31 23:00:00
  Total: 744 1h candles
  High:  $48500.25
  Low:   $38750.80
  Avg Volume: 245.67
  Avg Turnover: $10567890.45

🎉 All downloads completed!
```

## ⚡ Performance & Rate Limits

- **Rate Limiting**: 500ms delay between requests (respects Bybit's 120 req/min limit)
- **Max Limit**: 1000 klines per request (Bybit maximum)
- **Concurrent Downloads**: Processes symbols sequentially to avoid rate limits
- **Error Handling**: Automatic retry on temporary failures
- **Progress Tracking**: Real-time progress updates

## 🎯 Use Cases

### DCA Bot Data Preparation

```bash
# Download data for backtesting DCA strategies
go run ./scripts/download_bybit_historical_data.go \
    -symbols "BTCUSDT,ETHUSDT,BNBUSDT,ADAUSDT,SOLUSDT" \
    -intervals "60,240,D" \
    -category spot \
    -start 2023-01-01 \
    -end 2024-01-01
```

### Futures Trading Analysis

```bash
# Get futures data with funding rates context
go run ./scripts/download_bybit_historical_data.go \
    -symbols "BTCUSDT,ETHUSDT" \
    -intervals "60,240" \
    -categories "spot,linear" \
    -start 2024-01-01 \
    -end 2024-12-31
```

### High-Frequency Analysis

```bash
# Download minute data for short-term analysis
go run ./scripts/download_bybit_historical_data.go \
    -symbol BTCUSDT \
    -interval 1 \
    -category spot \
    -start 2024-08-01 \
    -end 2024-08-02
```

### Research & Backtesting

```bash
# Comprehensive dataset for research
go run ./scripts/download_bybit_historical_data.go \
    -symbols "BTCUSDT,ETHUSDT,BNBUSDT,ADAUSDT,DOTUSDT,SOLUSDT,AVAXUSDT,MATICUSDT" \
    -intervals "60,240,D" \
    -categories "spot,linear" \
    -start 2022-01-01 \
    -end 2024-01-01 \
    -outdir research_data
```

## 🔍 Data Validation

The downloader includes several validation features:

- **Time Range Validation**: Ensures data is within requested range
- **Duplicate Detection**: Prevents duplicate data entries
- **Completeness Check**: Reports any gaps in the data
- **Format Validation**: Validates price and volume data format
- **API Error Handling**: Proper handling of Bybit API errors

## 🚨 Error Handling

Common issues and solutions:

### Symbol Not Found

```
❌ Failed to download data for spot INVALID 60: Bybit API error 10001: symbol not found
```

**Solution**: Check symbol format (must be uppercase, e.g., BTCUSDT)

### Rate Limit Exceeded

```
❌ Failed to download data: API error 10006: rate limit exceeded
```

**Solution**: The downloader automatically handles rate limits with delays

### Invalid Date Range

```
❌ Invalid start date format: parsing time "2024-1-1": cannot parse "1-1" as "01-02"
```

**Solution**: Use YYYY-MM-DD format (e.g., 2024-01-01)

## 📊 Integration with DCA Bot

The downloaded data can be directly used with your DCA bot:

```go
// Load historical data for backtesting
data, err := loadCSV("data/bybit/spot/BTCUSDT/60/candles.csv")
if err != nil {
    log.Fatal(err)
}

// Use with your existing backtesting framework
strategy := NewDCAStrategy(data)
results := strategy.Backtest()
```

## 🎉 Benefits Over Original Binance Downloader

1. **Multiple Markets**: Supports spot + futures (Binance script was futures-only)
2. **Enhanced Data**: Includes turnover data (additional to volume)
3. **Better Organization**: Category-based directory structure
4. **Futures Support**: Full futures market coverage with leverage data
5. **Real-time Markets**: Uses same API as live trading
6. **Funding Rate Context**: Futures data aligns with funding rate analysis

## 🔧 Build & Install

```bash
# Build standalone executable
go build -o bybit-downloader ./scripts/download_bybit_historical_data.go

# Run directly
./bybit-downloader -symbol BTCUSDT -interval 60 -category spot

# Or use go run
go run ./scripts/download_bybit_historical_data.go [options]
```

This downloader provides everything you need for comprehensive Bybit historical data analysis and is fully compatible with your existing DCA bot infrastructure!
