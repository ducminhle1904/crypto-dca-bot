# Enhanced DCA Bot Makefile
# This Makefile provides common commands for building and running the Enhanced DCA bot

.PHONY: help build run clean test docker-build docker-run

# Default target
help:
	@echo "Enhanced DCA Bot - Available Commands:"
	@echo ""
	@echo "  build        - Build the bot binary"
	@echo "  run          - Run the bot (requires environment variables)"
	@echo "  run-btc      - Run bot with BTC 15m optimized config"
	@echo "  run-eth      - Run bot with ETH 30m optimized config"
	@echo "  run-ada      - Run bot with ADA 5m optimized config"
	@echo "  run-sol      - Run bot with SOL 1h optimized config"
	@echo "  run-doge     - Run bot with DOGE 15m optimized config"
	@echo "  run-xrp      - Run bot with XRP 5m optimized config"
	@echo "  run-sui      - Run bot with SUI 5m optimized config"
	@echo "  run-arb      - Run bot with ARB 5m optimized config"
	@echo "  clean        - Clean build artifacts"
	@echo "  test         - Run tests"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run bot in Docker container"
	@echo "  install      - Install dependencies"
	@echo "  lint         - Run code linting"
	@echo ""

# Build the bot
build:
	@echo "🔨 Building Enhanced DCA Bot..."
	go build -o enhanced-dca-bot .
	@echo "✅ Build complete: enhanced-dca-bot"

# Run the bot (requires environment variables)
run: build
	@echo "🚀 Starting Enhanced DCA Bot..."
	@echo "⚠️  Make sure you have set the required environment variables:"
	@echo "   - EXCHANGE_API_KEY"
	@echo "   - EXCHANGE_SECRET"
	@echo "   - TRADING_SYMBOL (default: BTCUSDT)"
	@echo "   - BASE_AMOUNT (default: 100.0)"
	@echo "   - TELEGRAM_TOKEN (optional)"
	@echo "   - TELEGRAM_CHAT_ID (optional)"
	@echo ""
	@echo "📁 Available optimized configs:"
	@echo "   - btc_15m (Bitcoin 15m)"
	@echo "   - eth_30m (Ethereum 30m)"
	@echo "   - ada_5m (Cardano 5m)"
	@echo "   - sol_1h (Solana 1h)"
	@echo "   - doge_15m (Dogecoin 15m)"
	@echo "   - xrp_5m (Ripple 5m)"
	@echo "   - sui_5m (Sui 5m)"
	@echo "   - arb_5m (Arbitrum 5m)"
	@echo ""
	@echo "Usage: ./enhanced-dca-bot -config=btc_15m"
	@echo ""
	./enhanced-dca-bot

# Run with specific optimized configs
run-btc: build
	@echo "🚀 Starting Enhanced DCA Bot with BTC 15m optimized config..."
	./enhanced-dca-bot -config=btc_15m

run-eth: build
	@echo "🚀 Starting Enhanced DCA Bot with ETH 30m optimized config..."
	./enhanced-dca-bot -config=eth_30m

run-ada: build
	@echo "🚀 Starting Enhanced DCA Bot with ADA 5m optimized config..."
	./enhanced-dca-bot -config=ada_5m

run-sol: build
	@echo "🚀 Starting Enhanced DCA Bot with SOL 1h optimized config..."
	./enhanced-dca-bot -config=sol_1h

run-doge: build
	@echo "🚀 Starting Enhanced DCA Bot with DOGE 15m optimized config..."
	./enhanced-dca-bot -config=doge_15m

run-xrp: build
	@echo "🚀 Starting Enhanced DCA Bot with XRP 5m optimized config..."
	./enhanced-dca-bot -config=xrp_5m

run-sui: build
	@echo "🚀 Starting Enhanced DCA Bot with SUI 5m optimized config..."
	./enhanced-dca-bot -config=sui_5m

run-arb: build
	@echo "🚀 Starting Enhanced DCA Bot with ARB 5m optimized config..."
	./enhanced-dca-bot -config=arb_5m

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f enhanced-dca-bot
	@echo "✅ Clean complete"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...
	@echo "✅ Tests complete"

# Install dependencies
install:
	@echo "📦 Installing dependencies..."
	go mod download
	go mod tidy
	@echo "✅ Dependencies installed"

# Run code linting
lint:
	@echo "🔍 Running code linting..."
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Build Docker image
docker-build:
	@echo "🐳 Building Docker image..."
	docker build -t enhanced-dca-bot .
	@echo "✅ Docker image built: enhanced-dca-bot"

# Run bot in Docker container
docker-run:
	@echo "🐳 Running Enhanced DCA Bot in Docker..."
	@echo "⚠️  Make sure you have set the required environment variables:"
	@echo "   - EXCHANGE_API_KEY"
	@echo "   - EXCHANGE_SECRET"
	@echo "   - TRADING_SYMBOL"
	@echo "   - BASE_AMOUNT"
	@echo "   - TELEGRAM_TOKEN (optional)"
	@echo "   - TELEGRAM_CHAT_ID (optional)"
	@echo ""
	docker run --rm \
		-e EXCHANGE_API_KEY \
		-e EXCHANGE_SECRET \
		-e EXCHANGE_TESTNET \
		-e TRADING_SYMBOL \
		-e BASE_AMOUNT \
		-e MAX_MULTIPLIER \
		-e TRADING_INTERVAL \
		-e TELEGRAM_TOKEN \
		-e TELEGRAM_CHAT_ID \
		-e PROMETHEUS_PORT \
		-e HEALTH_PORT \
		-p 8080:8080 \
		-p 8081:8081 \
		enhanced-dca-bot

# Development mode with hot reload (requires air)
dev:
	@echo "🔥 Starting development mode with hot reload..."
	@if command -v air > /dev/null; then \
		air; \
	else \
		echo "⚠️  air not found. Install with: go install github.com/cosmtrek/air@latest"; \
		echo "   Or run: make run"; \
	fi

# Check if all required tools are installed
check-tools:
	@echo "🔍 Checking required tools..."
	@command -v go > /dev/null || (echo "❌ Go not found" && exit 1)
	@command -v git > /dev/null || (echo "❌ Git not found" && exit 1)
	@echo "✅ All required tools are installed"

# Setup development environment
setup: check-tools install
	@echo "🚀 Development environment setup complete!"
	@echo ""
	@echo "Next steps:"
	@echo "1. Set your environment variables"
	@echo "2. Run 'make run-btc' to start with BTC config"
	@echo "3. Or run 'make run-eth' for ETH config"
	@echo "4. Or run 'make dev' for development mode"

# Show current configuration
config:
	@echo "⚙️  Current Configuration:"
	@echo "   EXCHANGE_API_KEY: $(if $(EXCHANGE_API_KEY),$(EXCHANGE_API_KEY),not set)"
	@echo "   EXCHANGE_SECRET: $(if $(EXCHANGE_SECRET),$(EXCHANGE_SECRET),not set)"
	@echo "   TRADING_SYMBOL: $(if $(TRADING_SYMBOL),$(TRADING_SYMBOL),BTCUSDT)"
	@echo "   BASE_AMOUNT: $(if $(BASE_AMOUNT),$(BASE_AMOUNT),100.0)"
	@echo "   TELEGRAM_TOKEN: $(if $(TELEGRAM_TOKEN),$(TELEGRAM_TOKEN),not set)"
	@echo "   TELEGRAM_CHAT_ID: $(if $(TELEGRAM_CHAT_ID),$(TELEGRAM_CHAT_ID),not set)"

# Quick start with sample configuration
quickstart:
	@echo "🚀 Quick Start - Enhanced DCA Bot with Optimized Configs"
	@echo ""
	@echo "1. Set your Binance API credentials:"
	@echo "   export EXCHANGE_API_KEY='your_api_key'"
	@echo "   export EXCHANGE_SECRET='your_secret'"
	@echo ""
	@echo "2. Set trading parameters:"
	@echo "   export TRADING_SYMBOL='BTCUSDT'"
	@echo "   export BASE_AMOUNT='100.0'"
	@echo ""
	@echo "3. (Optional) Set Telegram notifications:"
	@echo "   export TELEGRAM_TOKEN='your_bot_token'"
	@echo "   export TELEGRAM_CHAT_ID='your_chat_id'"
	@echo ""
	@echo "4. Build and run with optimized config:"
	@echo "   make build"
	@echo "   make run-btc    # For Bitcoin 15m config"
	@echo "   make run-eth    # For Ethereum 30m config"
	@echo "   make run-ada    # For Cardano 5m config"
	@echo "   make run-sol    # For Solana 1h config"
	@echo "   make run-doge   # For Dogecoin 15m config"
	@echo "   make run-xrp    # For Ripple 5m config"
	@echo "   make run-sui    # For Sui 5m config"
	@echo "   make run-arb    # For Arbitrum 5m config"
	@echo ""
	@echo "5. Monitor the bot:"
	@echo "   Health: http://localhost:8081/health"
	@echo "   Metrics: http://localhost:8080/metrics"
	@echo ""
	@echo "6. Or run with custom config:"
	@echo "   ./enhanced-dca-bot -config=your_custom_config"

# List available configs
list-configs:
	@echo "📁 Available Optimized Configurations:"
	@echo ""
	@echo "  btc_15m  - Bitcoin 15m (Base: $100, TP: 5.5%%, Data: 15m)"
	@echo "  eth_30m  - Ethereum 30m (Base: $100, TP: 5.5%%, Data: 30m)"
	@echo "  ada_5m   - Cardano 5m (Base: $100, TP: 5.5%%, Data: 5m)"
	@echo "  sol_1h   - Solana 1h (Base: $100, TP: 5.5%%, Data: 1h)"
	@echo "  doge_15m - Dogecoin 15m (Base: $100, TP: 5.5%%, Data: 15m)"
	@echo "  xrp_5m   - Ripple 5m (Base: $100, TP: 5.5%%, Data: 5m)"
	@echo "  sui_5m   - Sui 5m (Base: $100, TP: 5.5%%, Data: 5m)"
	@echo "  arb_5m   - Arbitrum 5m (Base: $100, TP: 5.5%%, Data: 5m)"
	@echo ""
	@echo "Usage: make run-<symbol> or ./enhanced-dca-bot -config=<config_name>"
	@echo ""
	@echo "Note: The bot automatically uses the data interval from the config filename"
	@echo "      (e.g., btc_15m.json will use 15m data for analysis)"
