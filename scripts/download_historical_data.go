package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// KlineData represents candlestick data from Binance
type KlineData struct {
	OpenTime  int64
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
	CloseTime int64
}

func main() {
	var (
		// Single-symbol backward compatible flags
		symbol    = flag.String("symbol", "BTCUSDT", "Trading symbol (e.g. BTCUSDT)")
		interval  = flag.String("interval", "1h", "Kline interval (1m, 5m, 15m, 30m, 1h, 4h, 1d)")

		// New multi options
		symbols    = flag.String("symbols", "", "Comma-separated list of symbols (overrides -symbol if provided)")
		intervals  = flag.String("intervals", "", "Comma-separated list of intervals (overrides -interval if provided)")
		outdir     = flag.String("outdir", "data/historical", "Directory to write CSV files")

		startDate = flag.String("start", "", "Start date (YYYY-MM-DD)")
		endDate   = flag.String("end", "", "End date (YYYY-MM-DD)")
		output    = flag.String("output", "", "Explicit output file path (only for single symbol/interval)")
		limit     = flag.Int("limit", 1000, "Number of klines per request")
	)

	flag.Parse()

	// Build symbol list
	symList := []string{}
	if strings.TrimSpace(*symbols) != "" {
		for _, s := range strings.Split(*symbols, ",") {
			ss := strings.ToUpper(strings.TrimSpace(s))
			if ss != "" {
				symList = append(symList, ss)
			}
		}
	} else {
		symList = []string{strings.ToUpper(strings.TrimSpace(*symbol))}
	}

	// Build interval list
	intList := []string{}
	if strings.TrimSpace(*intervals) != "" {
		for _, it := range strings.Split(*intervals, ",") {
			ival := strings.TrimSpace(it)
			if ival != "" {
				intList = append(intList, ival)
			}
		}
	} else {
		intList = []string{strings.TrimSpace(*interval)}
	}

	// Set default dates if not provided
	end := time.Now()
	start := end.AddDate(-1, 0, 0) // Default to 1 year of data

	if *startDate != "" {
		parsedStart, err := time.Parse("2006-01-02", *startDate)
		if err != nil {
			log.Fatalf("Invalid start date format: %v", err)
		}
		start = parsedStart
	}

	if *endDate != "" {
		parsedEnd, err := time.Parse("2006-01-02", *endDate)
		if err != nil {
			log.Fatalf("Invalid end date format: %v", err)
		}
		end = parsedEnd
	}

	// Ensure base output directory exists
	if err := os.MkdirAll(*outdir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Decide if we're in single or batch mode
	singleMode := len(symList) == 1 && len(intList) == 1

	if singleMode && strings.TrimSpace(*output) != "" {
		// Use explicit output path
		downloadOne(symList[0], intList[0], start, end, *limit, *output)
		return
	}

	// Batch mode (or single with default directory structure)
	for _, sym := range symList {
		for _, ival := range intList {
			comboDir := filepath.Join(*outdir, sym, ival)
			outPath := filepath.Join(comboDir, "candles.csv")
			downloadOne(sym, ival, start, end, *limit, outPath)
		}
	}
}

func downloadOne(symbol, interval string, start, end time.Time, limit int, outputPath string) {
	fmt.Printf("\n📊 Downloading %s data for %s\n", interval, symbol)
	fmt.Printf("📅 Period: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Printf("📁 Output: %s\n", outputPath)
	fmt.Println("🔄 Fetching data...")

	klines, err := downloadKlines(symbol, interval, start, end, limit)
	if err != nil {
		log.Fatalf("Failed to download data for %s %s: %v", symbol, interval, err)
	}

	fmt.Printf("✅ Downloaded %d klines\n", len(klines))

	// Ensure parent directory exists for this file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		log.Fatalf("Failed to prepare output directory %s: %v", filepath.Dir(outputPath), err)
	}

	if err := saveToCSV(klines, outputPath); err != nil {
		log.Fatalf("Failed to save %s %s: %v", symbol, interval, err)
	}

	fmt.Printf("💾 Data saved to %s\n", outputPath)
	printSummary(klines, interval)
}

func printSummary(klines []KlineData, interval string) {
	if len(klines) == 0 {
		return
	}
	firstKline := klines[0]
	lastKline := klines[len(klines)-1]

	fmt.Println("\n📊 DATA SUMMARY:")
	fmt.Printf("  First: %s\n", time.Unix(firstKline.OpenTime/1000, 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("  Last:  %s\n", time.Unix(lastKline.OpenTime/1000, 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("  Total: %d %s candles\n", len(klines), interval)

	// Calculate some basic stats
	var totalVolume float64
	highPrice := 0.0
	lowPrice := 1e9

	for _, kline := range klines {
		volume, _ := strconv.ParseFloat(kline.Volume, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)

		totalVolume += volume
		if high > highPrice {
			highPrice = high
		}
		if low < lowPrice {
			lowPrice = low
		}
	}

	fmt.Printf("  High:  $%.2f\n", highPrice)
	fmt.Printf("  Low:   $%.2f\n", lowPrice)
	fmt.Printf("  Avg Volume: %.2f\n", totalVolume/float64(len(klines)))
}

func downloadKlines(symbol, interval string, start, end time.Time, limit int) ([]KlineData, error) {
	var allKlines []KlineData

	// Convert times to milliseconds
	startMs := start.Unix() * 1000
	endMs := end.Unix() * 1000

	for startMs < endMs {
		// Build URL
		url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&startTime=%d&limit=%d",
			symbol, interval, startMs, limit)

		// Make request
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("HTTP request failed: %w", err)
		}

		if resp.StatusCode != 200 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
		}

		// Parse response
		var rawKlines [][]interface{}
		if err := json.NewDecoder(resp.Body).Decode(&rawKlines); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("JSON decode error: %w", err)
		}
		resp.Body.Close()

		if len(rawKlines) == 0 {
			break
		}

		// Convert to KlineData
		for _, raw := range rawKlines {
			if len(raw) < 6 {
				continue
			}

			kline := KlineData{
				OpenTime: int64(raw[0].(float64)),
				Open:     raw[1].(string),
				High:     raw[2].(string),
				Low:      raw[3].(string),
				Close:    raw[4].(string),
				Volume:   raw[5].(string),
				CloseTime: int64(raw[6].(float64)),
			}

			// Only add if within our time range
			if kline.OpenTime >= startMs && kline.OpenTime < endMs {
				allKlines = append(allKlines, kline)
			}
		}

		// Update start time for next request
		lastKline := rawKlines[len(rawKlines)-1]
		startMs = int64(lastKline[0].(float64)) + 1

		// Progress indicator
		fmt.Printf("\r  Progress: %d klines downloaded...", len(allKlines))

		// Rate limiting
		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println() // New line after progress

	return allKlines, nil
}

func saveToCSV(klines []KlineData, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"timestamp", "open", "high", "low", "close", "volume"}); err != nil {
		return err
	}

	// Write data
	for _, kline := range klines {
		timestamp := time.Unix(kline.OpenTime/1000, 0).Format("2006-01-02 15:04:05")

		record := []string{
			timestamp,
			kline.Open,
			kline.High,
			kline.Low,
			kline.Close,
			kline.Volume,
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
} 