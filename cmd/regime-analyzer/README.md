# Regime Analysis Tool

A validation tool for the Enhanced DCA Bot's regime detection system. This tool analyzes historical market data to validate the accuracy and stability of regime classifications.

## Purpose

This tool helps validate **Phase 1** of the Dual Engine Regime Bot implementation by:

- Testing regime detection on historical data
- Measuring false signal rates (target: <15% per plan acceptance criteria)
- Analyzing regime distribution and transitions
- Generating validation reports for performance assessment

## Usage

### Basic Usage

```bash
go run cmd/regime-analyzer/main.go -csv path/to/data.csv -symbol BTCUSDT
```

### Advanced Usage

```bash
go run cmd/regime-analyzer/main.go \
  -csv data/bybit/linear/BTCUSDT/5/candles.csv \
  -symbol BTCUSDT \
  -output regime_analysis_btc \
  -verbose
```

### Flags

- `-csv`: Path to CSV file with OHLCV data (required)
- `-symbol`: Trading symbol for analysis (default: BTCUSDT)
- `-output`: Output directory for results (default: regime_analysis)
- `-verbose`: Enable verbose output during analysis

## Input Data Format

The CSV file must have the following format:

```csv
timestamp,open,high,low,close,volume
1640995200000,47686.01,47733.56,47682.37,47697.32,1.234
1640995500000,47697.32,47745.89,47689.14,47712.45,2.456
...
```

Where:

- `timestamp`: Unix timestamp in milliseconds
- `open,high,low,close`: Price values
- `volume`: Trading volume

## Output Files

The tool generates three output files:

### 1. `regime_analysis_summary.json`

Summary statistics including:

- Regime distribution percentages
- False signal rate analysis
- Average confidence scores
- Phase 1 validation results

### 2. `regime_analysis_detailed.json`

Detailed results for each data point:

- Detected regime type
- Confidence score
- Technical indicators (trend strength, volatility, noise level)
- Transition flags

### 3. `regime_analysis.csv`

CSV format for visualization tools:

- Compatible with Excel, Python pandas, R
- Suitable for creating charts and graphs
- Time series format for regime visualization

## Phase 1 Validation Criteria

The tool automatically validates against Phase 1 acceptance criteria:

✅ **PASS Criteria:**

- False signal rate < 3.0 per hour
- Regime stability > 85%
- Average confidence > 60%

❌ **FAIL Criteria:**

- High false signal rate (frequent regime switching)
- Low regime stability (too many transitions)
- Low average confidence scores

## Example Output

```
🔍 Enhanced DCA Bot - Regime Analysis Tool
📁 Analyzing: data/btc_5m.csv
📊 Symbol: BTCUSDT
📈 Output: regime_analysis/

✅ Loaded 5000 data points
🔧 Initializing regime detector...
🚀 Running regime analysis...
✅ Analysis complete in 1.234s

📈 REGIME ANALYSIS SUMMARY
═══════════════════════════

📊 Data Statistics:
   • Total Data Points: 4750
   • Analysis Duration: 1.234s
   • Average Confidence: 72.45%

🎯 Regime Distribution:
   • TRENDING: 1425 points (30.0%)
   • RANGING: 2375 points (50.0%)
   • VOLATILE: 712 points (15.0%)
   • UNCERTAIN: 238 points (5.0%)

🔄 Transition Statistics:
   • Total Transitions: 234
   • False Signal Rate: 2.34 per hour
   • Regime Stability: 88.2%

✅ PHASE 1 VALIDATION:
   • False Signal Rate: ✅ PASS (2.3 < 3.0/hr)
   • Regime Stability: ✅ PASS (88.2% > 85%)
   • Average Confidence: ✅ PASS (72.5% > 60%)

🎉 Regime analysis complete!
📁 Results saved to: regime_analysis/
```

## Data Sources

Compatible with data from:

- Enhanced DCA Bot's data downloader
- `scripts/download_bybit_historical_data.go`
- Any CSV with standard OHLCV format

## Troubleshooting

**Error: "Need at least 250 data points"**

- Ensure your CSV has sufficient historical data
- Minimum ~21 hours of 5-minute data required

**Error: "CSV file must have at least 2 rows"**

- Check CSV format and header row
- Verify timestamp format (Unix milliseconds)

**High False Signal Rate**

- May indicate need for parameter tuning
- Consider adjusting confirmation bars or thresholds
- Review market conditions during test period

## Integration with Development Workflow

This tool supports the Phase 1 development workflow:

1. Implement regime detection logic
2. Run validation on historical data
3. Tune parameters based on results
4. Verify acceptance criteria are met
5. Proceed to Phase 2 implementation
