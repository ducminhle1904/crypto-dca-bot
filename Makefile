# Enhanced DCA Bot Makefile

.PHONY: help build test clean run docker-build docker-run lint fmt examples

# Default target
help:
	@echo "Enhanced DCA Bot - Available commands:"
	@echo "  build        - Build the application"
	@echo "  test         - Run tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  run          - Run the application"
	@echo "  examples     - Run interactive examples"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run with Docker Compose"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  download-bybit-data - Build Bybit data downloader"
	@echo "  download-bybit-quick - Download popular pairs (30 days)"
	@echo "  download-bybit-backtest - Download full dataset (1 year)"

# Build the application
build:
	@echo "Building Enhanced DCA Bot..."
	go build -o bin/dca-bot ./cmd/bot

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run the application
run:
	@echo "Running Enhanced DCA Bot..."
	go run ./cmd/bot

# Build Docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t crypto-dca-bot .

# Run with Docker Compose
docker-run:
	@echo "Starting services with Docker Compose..."
	docker-compose up -d

# Stop Docker Compose
docker-stop:
	@echo "Stopping services..."
	docker-compose down

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Download Bybit historical data
download-bybit-data:
	@echo "Building Bybit data downloader..."
	go build -o bin/bybit-downloader ./scripts/download_bybit_historical_data.go
	@echo "Bybit downloader built successfully!"
	@echo ""
	@echo "Usage examples:"
	@echo "  # Download 1 year of hourly BTCUSDT spot data:"
	@echo "  ./bin/bybit-downloader -symbol BTCUSDT -interval 60 -category spot"
	@echo ""
	@echo "  # Download multiple symbols and intervals:"
	@echo "  ./bin/bybit-downloader -symbols \"BTCUSDT,ETHUSDT\" -intervals \"60,240\" -categories \"spot,linear\""
	@echo ""
	@echo "  # Custom date range:"
	@echo "  ./bin/bybit-downloader -symbol BTCUSDT -interval 60 -category spot -start 2024-01-01 -end 2024-12-31"
	@echo ""
	@echo "See scripts/README_BYBIT_DOWNLOADER.md for full documentation"

# Quick Bybit data download for common pairs
download-bybit-quick:
	@echo "Downloading popular trading pairs (last 30 days)..."
	go run ./scripts/download_bybit_historical_data.go \
		-symbols "BTCUSDT,ETHUSDT,BNBUSDT,SOLUSDT" \
		-intervals "60,240" \
		-categories "spot,linear" \
		-start $(shell date -d '30 days ago' +%Y-%m-%d 2>/dev/null || date -v-30d +%Y-%m-%d 2>/dev/null || echo "2024-07-19") \
		-outdir data/bybit

# Download Bybit data for backtesting
download-bybit-backtest:
	@echo "Downloading comprehensive dataset for backtesting (last 1 year)..."
	go run ./scripts/download_bybit_historical_data.go \
		-symbols "BTCUSDT,ETHUSDT,BNBUSDT,ADAUSDT,SOLUSDT,AVAXUSDT,DOTUSDT,MATICUSDT" \
		-intervals "60,240,D" \
		-categories "spot,linear" \
		-start $(shell date -d '1 year ago' +%Y-%m-%d 2>/dev/null || date -v-1y +%Y-%m-%d 2>/dev/null || echo "2024-01-01") \
		-outdir data/bybit/backtest

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Generate mocks (if using mockery)
mocks:
	@echo "Generating mocks..."
	mockery --all

# Run backtest
backtest:
	@echo "Running backtest..."
	go run ./cmd/backtest

# Download historical data
download-data:
	@echo "Downloading historical data..."
	go run scripts/download_historical_data.go -symbol BTCUSDT -interval 1h

# Run backtest with optimization
backtest-optimize:
	@echo "Running backtest optimization..."
	go run ./cmd/backtest -optimize

# Run backtest with custom config
backtest-config:
	@echo "Running backtest with custom config..."
	go run ./cmd/backtest -config configs/backtest-config.json

# Development setup
dev-setup: deps fmt lint test
	@echo "Development setup complete!"

# Production build
prod-build:
	@echo "Building for production..."
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/dca-bot ./cmd/bot

# Check for security vulnerabilities
security:
	@echo "Checking for security vulnerabilities..."
	gosec ./...

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Run interactive examples
examples:
	@echo "Running interactive examples..."
	cd examples && go run main.go data_loader.go 