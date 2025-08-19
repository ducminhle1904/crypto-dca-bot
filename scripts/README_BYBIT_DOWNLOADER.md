# Bybit Historical Data Downloader

A tool for downloading historical kline (candlestick) data from the Bybit exchange, supporting spot, linear futures, and inverse futures markets.

## 🚀 Quick Start

### Download Data for a Single Symbol

```bash
# Download one year of hourly BTC/USDT spot data
go run ./scripts/download_bybit_historical_data.go -symbol BTCUSDT -interval 60 -category spot

# Download data for a specific date range
go run ./scripts/download_bybit_historical_data.go \
    -symbol BTCUSDT \
    -interval 240 \
    -category linear \
    -start 2023-01-01 \
    -end 2023-12-31
```

### Download Data for Multiple Symbols & Intervals

```bash
# Download data for multiple symbols at once
go run ./scripts/download_bybit_historical_data.go \
    -symbols "BTCUSDT,ETHUSDT,SOLUSDT" \
    -interval 60 \
    -category spot

# Download data for multiple intervals at once
go run ./scripts/download_bybit_historical_data.go \
    -symbol BTCUSDT \
    -intervals "60,240,D" \
    -category linear
```

## 📋 Command-Line Flags

| Flag          | Description                                             | Default      |
| ------------- | ------------------------------------------------------- | ------------ |
| `-symbol`     | A single trading symbol to download.                    | `BTCUSDT`    |
| `-symbols`    | A comma-separated list of symbols to download.          | -            |
| `-interval`   | A single time interval.                                 | `60`         |
| `-intervals`  | A comma-separated list of intervals.                    | -            |
| `-category`   | A single market category (`spot`, `linear`, `inverse`). | `spot`       |
| `-categories` | A comma-separated list of categories.                   | -            |
| `-start`      | The start date for the data (YYYY-MM-DD).               | 1 year ago   |
| `-end`        | The end date for the data (YYYY-MM-DD).                 | Today        |
| `-outdir`     | The directory to save the data in.                      | `data/bybit` |

## 📁 Output Structure

The downloaded data is organized in a directory structure based on the market category, symbol, and interval:

```
data/bybit/
├── spot/
│   └── BTCUSDT/
│       └── 60/
│           └── candles.csv
└── linear/
    └── BTCUSDT/
        └── 240/
            └── candles.csv
```
