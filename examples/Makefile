# Makefile for Enhanced DCA Bot Examples
# ======================================

.PHONY: build run clean test help examples

# Build the examples
build:
	@echo "🔨 Building Enhanced DCA Bot Examples..."
	go build -o examples main.go data_loader.go
	@echo "✅ Build complete!"

# Run the examples
run: build
	@echo "🚀 Running Enhanced DCA Bot Examples..."
	./examples

# Run examples directly without building
run-dev:
	@echo "🚀 Running Enhanced DCA Bot Examples (dev mode)..."
	go run main.go data_loader.go

# Clean build artifacts
clean:
	@echo "🧹 Cleaning build artifacts..."
	rm -f examples
	@echo "✅ Clean complete!"

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Show help
help:
	@echo "Enhanced DCA Bot Examples - Available Commands:"
	@echo ""
	@echo "  build     - Build the examples executable"
	@echo "  run       - Build and run the examples"
	@echo "  run-dev   - Run examples directly with go run"
	@echo "  examples  - Run interactive examples from project root"
	@echo "  clean     - Clean build artifacts"
	@echo "  test      - Run tests"
	@echo "  help      - Show this help message"
	@echo ""
	@echo "Examples include:"
	@echo "  📊 Data Analysis"
	@echo "  🎯 Strategy Testing"
	@echo "  📈 Backtesting"
	@echo "  🎮 Interactive Trading"
	@echo "  📋 Performance Comparison"
	@echo "  🔧 Configuration Examples"

# Default target
.DEFAULT_GOAL := help 

examples:
	cd examples && go run main.go data_loader.go 