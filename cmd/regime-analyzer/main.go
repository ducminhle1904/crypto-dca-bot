package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/internal/regime"
	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
)

// RegimeAnalysisResult holds the analysis results for a single data point
type RegimeAnalysisResult struct {
	Timestamp      time.Time            `json:"timestamp"`
	Price          float64              `json:"price"`
	DetectedRegime regime.RegimeType    `json:"detected_regime"`
	Confidence     float64              `json:"confidence"`
	TrendStrength  float64              `json:"trend_strength"`
	Volatility     float64              `json:"volatility"`
	NoiseLevel     float64              `json:"noise_level"`
	TransitionFlag bool                 `json:"transition_flag"`
}

// RegimeAnalysisSummary holds summary statistics for the analysis
type RegimeAnalysisSummary struct {
	TotalDataPoints    int                            `json:"total_data_points"`
	RegimeDistribution map[regime.RegimeType]int      `json:"regime_distribution"`
	RegimePercentages  map[regime.RegimeType]float64  `json:"regime_percentages"`
	TransitionCount    int                            `json:"transition_count"`
	AverageConfidence  float64                        `json:"average_confidence"`
	AnalysisDuration   time.Duration                  `json:"analysis_duration"`
	
	// Validation metrics
	FalseSignalRate    float64                        `json:"false_signal_rate"`    // Estimated
	RegimeStability    float64                        `json:"regime_stability"`     // % time in stable regime
}

func main() {
	var (
		csvFile   = flag.String("csv", "", "Path to CSV file with OHLCV data")
		outputDir = flag.String("output", "regime_analysis", "Output directory for results")
		symbol    = flag.String("symbol", "BTCUSDT", "Trading symbol for analysis")
		verbose   = flag.Bool("verbose", false, "Enable verbose output")
	)
	flag.Parse()

	if *csvFile == "" {
		log.Fatal("CSV file path is required. Use -csv flag.")
	}

	fmt.Printf("🔍 Enhanced DCA Bot - Regime Analysis Tool\n")
	fmt.Printf("📁 Analyzing: %s\n", *csvFile)
	fmt.Printf("📊 Symbol: %s\n", *symbol)
	fmt.Printf("📈 Output: %s/\n\n", *outputDir)

	// Load historical data
	fmt.Printf("📖 Loading historical data...\n")
	data, err := loadCSVData(*csvFile)
	if err != nil {
		log.Fatalf("Failed to load CSV data: %v", err)
	}

	fmt.Printf("✅ Loaded %d data points\n", len(data))
	if len(data) < 200 {
		log.Fatal("⚠️  Need at least 200 data points for meaningful regime analysis")
	}

	// Create regime detector
	fmt.Printf("🔧 Initializing regime detector...\n")
	detector := regime.NewRegimeDetector()

	// Run regime analysis
	fmt.Printf("🚀 Running regime analysis...\n")
	startTime := time.Now()
	results, summary, err := analyzeRegimes(detector, data, *verbose)
	if err != nil {
		log.Fatalf("Failed to analyze regimes: %v", err)
	}
	
	analysisTime := time.Since(startTime)
	summary.AnalysisDuration = analysisTime

	fmt.Printf("✅ Analysis complete in %v\n\n", analysisTime)

	// Print summary
	printSummary(summary)

	// Create output directory
	err = os.MkdirAll(*outputDir, 0755)
	if err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Save detailed results to JSON
	fmt.Printf("💾 Saving detailed results...\n")
	err = saveResultsJSON(results, fmt.Sprintf("%s/regime_analysis_detailed.json", *outputDir))
	if err != nil {
		log.Printf("Warning: Failed to save detailed results: %v", err)
	}

	// Save summary to JSON
	err = saveSummaryJSON(summary, fmt.Sprintf("%s/regime_analysis_summary.json", *outputDir))
	if err != nil {
		log.Printf("Warning: Failed to save summary: %v", err)
	}

	// Save CSV for visualization
	fmt.Printf("📊 Saving CSV for visualization...\n")
	err = saveResultsCSV(results, fmt.Sprintf("%s/regime_analysis.csv", *outputDir))
	if err != nil {
		log.Printf("Warning: Failed to save CSV results: %v", err)
	}

	fmt.Printf("\n🎉 Regime analysis complete!\n")
	fmt.Printf("📁 Results saved to: %s/\n", *outputDir)
}

// loadCSVData loads OHLCV data from a CSV file
func loadCSVData(filename string) ([]types.OHLCV, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("CSV file must have at least 2 rows (header + data)")
	}

	var data []types.OHLCV
	
	// Skip header row
	for i := 1; i < len(records); i++ {
		record := records[i]
		if len(record) < 6 {
			continue // Skip invalid rows
		}

		// Parse timestamp - handle both Unix milliseconds and datetime format
		var timestamp time.Time
		var err error
		
		// Try Unix timestamp first
		if ts, parseErr := strconv.ParseInt(record[0], 10, 64); parseErr == nil {
			timestamp = time.Unix(ts/1000, 0) // Assuming milliseconds
		} else {
			// Try datetime format: "2023-01-01 07:00:00"
			timestamp, err = time.Parse("2006-01-02 15:04:05", record[0])
			if err != nil {
				// Try alternative format without seconds
				timestamp, err = time.Parse("2006-01-02 15:04", record[0])
				if err != nil {
					continue // Skip invalid timestamps
				}
			}
		}

		open, err := strconv.ParseFloat(record[1], 64)
		if err != nil {
			continue
		}
		high, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			continue
		}
		low, err := strconv.ParseFloat(record[3], 64)
		if err != nil {
			continue
		}
		close, err := strconv.ParseFloat(record[4], 64)
		if err != nil {
			continue
		}
		volume, err := strconv.ParseFloat(record[5], 64)
		if err != nil {
			continue
		}

		ohlcv := types.OHLCV{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		}

		data = append(data, ohlcv)
	}

	return data, nil
}

// analyzeRegimes runs the regime detection on historical data
func analyzeRegimes(detector *regime.RegimeDetector, data []types.OHLCV, verbose bool) ([]RegimeAnalysisResult, *RegimeAnalysisSummary, error) {
	var results []RegimeAnalysisResult
	
	summary := &RegimeAnalysisSummary{
		RegimeDistribution: make(map[regime.RegimeType]int),
		RegimePercentages:  make(map[regime.RegimeType]float64),
	}

	// Start analyzing from a reasonable point (need enough data for indicators)
	minDataPoints := 250
	if len(data) < minDataPoints {
		return nil, nil, fmt.Errorf("need at least %d data points for analysis", minDataPoints)
	}

	totalConfidence := 0.0
	transitionCount := 0
	lastRegime := regime.RegimeUncertain

	// Analyze each data point
	for i := minDataPoints; i < len(data); i++ {
		// Get data slice up to current point
		currentData := data[:i+1]
		
		// Detect regime
		signal, err := detector.DetectRegime(currentData)
		if err != nil {
			if verbose {
				fmt.Printf("Warning at index %d: %v\n", i, err)
			}
			continue
		}

		// Count transitions
		if signal.Type != lastRegime && lastRegime != regime.RegimeUncertain {
			transitionCount++
		}
		lastRegime = signal.Type

		// Record result
		result := RegimeAnalysisResult{
			Timestamp:      data[i].Timestamp,
			Price:          data[i].Close,
			DetectedRegime: signal.Type,
			Confidence:     signal.Confidence,
			TrendStrength:  signal.TrendStrength,
			Volatility:     signal.Volatility,
			NoiseLevel:     signal.NoiseLevel,
			TransitionFlag: signal.TransitionFlag,
		}
		
		results = append(results, result)
		
		// Update summary statistics
		summary.RegimeDistribution[signal.Type]++
		totalConfidence += signal.Confidence

		if verbose && i%1000 == 0 {
			fmt.Printf("Processed %d/%d data points...\n", i-minDataPoints+1, len(data)-minDataPoints)
		}
	}

	summary.TotalDataPoints = len(results)
	summary.TransitionCount = transitionCount
	
	if len(results) > 0 {
		summary.AverageConfidence = totalConfidence / float64(len(results))
	}

	// Calculate percentages
	for regimeType, count := range summary.RegimeDistribution {
		summary.RegimePercentages[regimeType] = float64(count) / float64(summary.TotalDataPoints) * 100.0
	}

	// Estimate false signal rate (transitions per hour, assuming 5m data)
	if len(results) > 0 {
		dataHours := float64(len(results)) / 12.0 // 12 x 5min = 1 hour
		summary.FalseSignalRate = float64(transitionCount) / dataHours
	}

	// Calculate regime stability (how often we're not in transition)
	stableCount := 0
	for _, result := range results {
		if !result.TransitionFlag {
			stableCount++
		}
	}
	if len(results) > 0 {
		summary.RegimeStability = float64(stableCount) / float64(len(results)) * 100.0
	}

	return results, summary, nil
}

// printSummary prints a formatted summary of the analysis
func printSummary(summary *RegimeAnalysisSummary) {
	fmt.Printf("📈 REGIME ANALYSIS SUMMARY\n")
	fmt.Printf("═══════════════════════════\n\n")
	
	fmt.Printf("📊 Data Statistics:\n")
	fmt.Printf("   • Total Data Points: %d\n", summary.TotalDataPoints)
	fmt.Printf("   • Analysis Duration: %v\n", summary.AnalysisDuration)
	fmt.Printf("   • Average Confidence: %.2f%%\n\n", summary.AverageConfidence*100)
	
	fmt.Printf("🎯 Regime Distribution:\n")
	for regimeType, percentage := range summary.RegimePercentages {
		count := summary.RegimeDistribution[regimeType]
		fmt.Printf("   • %s: %d points (%.1f%%)\n", regimeType.String(), count, percentage)
	}
	
	fmt.Printf("\n🔄 Transition Statistics:\n")
	fmt.Printf("   • Total Transitions: %d\n", summary.TransitionCount)
	fmt.Printf("   • False Signal Rate: %.2f per hour\n", summary.FalseSignalRate)
	fmt.Printf("   • Regime Stability: %.1f%%\n\n", summary.RegimeStability)
	
	// Performance assessment based on Phase 1 acceptance criteria
	fmt.Printf("✅ PHASE 1 VALIDATION:\n")
	if summary.FalseSignalRate < 3.0 { // Less than 3 per hour = good
		fmt.Printf("   • False Signal Rate: ✅ PASS (%.1f < 3.0/hr)\n", summary.FalseSignalRate)
	} else {
		fmt.Printf("   • False Signal Rate: ❌ FAIL (%.1f >= 3.0/hr)\n", summary.FalseSignalRate)
	}
	
	if summary.RegimeStability > 85.0 {
		fmt.Printf("   • Regime Stability: ✅ PASS (%.1f%% > 85%%)\n", summary.RegimeStability)
	} else {
		fmt.Printf("   • Regime Stability: ❌ FAIL (%.1f%% <= 85%%)\n", summary.RegimeStability)
	}
	
	if summary.AverageConfidence > 0.6 {
		fmt.Printf("   • Average Confidence: ✅ PASS (%.1f%% > 60%%)\n", summary.AverageConfidence*100)
	} else {
		fmt.Printf("   • Average Confidence: ❌ FAIL (%.1f%% <= 60%%)\n", summary.AverageConfidence*100)
	}
	
	fmt.Printf("\n")
}

// saveResultsJSON saves detailed results to JSON file
func saveResultsJSON(results []RegimeAnalysisResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}

// saveSummaryJSON saves summary to JSON file
func saveSummaryJSON(summary *RegimeAnalysisSummary, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(summary)
}

// saveResultsCSV saves results to CSV file for visualization
func saveResultsCSV(results []RegimeAnalysisResult, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	header := []string{
		"Timestamp", "Price", "Regime", "Confidence", 
		"TrendStrength", "Volatility", "NoiseLevel", "TransitionFlag",
	}
	writer.Write(header)

	// Write data
	for _, result := range results {
		record := []string{
			result.Timestamp.Format("2006-01-02 15:04:05"),
			fmt.Sprintf("%.8f", result.Price),
			result.DetectedRegime.String(),
			fmt.Sprintf("%.4f", result.Confidence),
			fmt.Sprintf("%.4f", result.TrendStrength),
			fmt.Sprintf("%.4f", result.Volatility),
			fmt.Sprintf("%.4f", result.NoiseLevel),
			fmt.Sprintf("%t", result.TransitionFlag),
		}
		writer.Write(record)
	}

	return nil
}
