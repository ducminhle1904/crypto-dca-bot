# Enhanced DCA Bot - Backtesting Setup Script for Windows
# This script sets up the backtesting environment

Write-Host "🚀 Enhanced DCA Bot - Backtesting Setup" -ForegroundColor Cyan
Write-Host "=======================================" -ForegroundColor Cyan

# Create necessary directories
Write-Host "`n📁 Creating directories..." -ForegroundColor Yellow
$directories = @("data/historical", "results", "docs", "configs")
foreach ($dir in $directories) {
    if (!(Test-Path $dir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
        Write-Host "  ✓ Created $dir" -ForegroundColor Green
    } else {
        Write-Host "  - $dir already exists" -ForegroundColor Gray
    }
}

# Check if Go is installed
Write-Host "`n🔍 Checking Go installation..." -ForegroundColor Yellow
try {
    $goVersion = go version
    Write-Host "  ✓ Go is installed: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "  ❌ Go is not installed!" -ForegroundColor Red
    Write-Host "  Please install Go from https://golang.org/dl/" -ForegroundColor Yellow
    exit 1
}

# Download dependencies
Write-Host "`n📦 Downloading Go dependencies..." -ForegroundColor Yellow
go mod download
if ($LASTEXITCODE -eq 0) {
    Write-Host "  ✓ Dependencies downloaded" -ForegroundColor Green
} else {
    Write-Host "  ❌ Failed to download dependencies" -ForegroundColor Red
    exit 1
}

# Download sample historical data
Write-Host "`n📊 Downloading sample historical data..." -ForegroundColor Yellow
Write-Host "  This will download 1 month of Bitcoin hourly data" -ForegroundColor Gray

$startDate = (Get-Date).AddMonths(-1).ToString("yyyy-MM-dd")
$endDate = (Get-Date).ToString("yyyy-MM-dd")

go run scripts/download_historical_data.go -symbol BTCUSDT -interval 1h -start $startDate -end $endDate

if ($LASTEXITCODE -eq 0) {
    Write-Host "  ✓ Historical data downloaded" -ForegroundColor Green
} else {
    Write-Host "  ⚠️  Failed to download data (will use generated data)" -ForegroundColor Yellow
}

# Run a test backtest
Write-Host "`n🧪 Running test backtest..." -ForegroundColor Yellow
go run cmd/backtest/main.go -balance 1000 -base-amount 50

if ($LASTEXITCODE -eq 0) {
    Write-Host "`n✅ Backtesting setup complete!" -ForegroundColor Green
} else {
    Write-Host "`n⚠️  Test backtest failed, but setup is complete" -ForegroundColor Yellow
}

# Display next steps
Write-Host "`n📋 Next Steps:" -ForegroundColor Cyan
Write-Host "  1. Download more historical data:" -ForegroundColor White
Write-Host "     go run scripts/download_historical_data.go -symbol BTCUSDT -interval 1h" -ForegroundColor Gray
Write-Host ""
Write-Host "  2. Run a basic backtest:" -ForegroundColor White
Write-Host "     go run cmd/backtest/main.go" -ForegroundColor Gray
Write-Host ""
Write-Host "  3. Run parameter optimization:" -ForegroundColor White
Write-Host "     go run cmd/backtest/main.go -optimize" -ForegroundColor Gray
Write-Host ""
Write-Host "  4. Read the backtesting guide:" -ForegroundColor White
Write-Host "     docs/BACKTESTING_GUIDE.md" -ForegroundColor Gray
Write-Host ""

Write-Host "Happy backtesting! 🚀" -ForegroundColor Green 