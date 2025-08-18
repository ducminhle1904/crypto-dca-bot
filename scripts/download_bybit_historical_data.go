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

// BybitKlineData represents candlestick data from Bybit
type BybitKlineData struct {
	StartTime int64
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
	Turnover  string
}

// BybitResponse represents the API response structure
type BybitResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Symbol   string     `json:"symbol"`
		Category string     `json:"category"`
		List     [][]string `json:"list"`
	} `json:"result"`
	Time int64 `json:"time"`
}

func main() {
	var (
		// Single-symbol backward compatible flags
		symbol    = flag.String("symbol", "BTCUSDT", "Trading symbol (e.g. BTCUSDT)")
		interval  = flag.String("interval", "60", "Kline interval (1, 3, 5, 15, 30, 60, 120, 240, 360, 720, D, W, M)")
		category  = flag.String("category", "spot", "Market category (spot, linear, inverse)")

		// New multi options
		symbols    = flag.String("symbols", "", "Comma-separated list of symbols (overrides -symbol if provided)")
		intervals  = flag.String("intervals", "", "Comma-separated list of intervals (overrides -interval if provided)")
		categories = flag.String("categories", "", "Comma-separated list of categories (overrides -category if provided)")
		outdir     = flag.String("outdir", "data/bybit", "Directory to write CSV files")

		startDate = flag.String("start", "", "Start date (YYYY-MM-DD)")
		endDate   = flag.String("end", "", "End date (YYYY-MM-DD)")
		output    = flag.String("output", "", "Explicit output file path (only for single symbol/interval/category)")
		limit     = flag.Int("limit", 1000, "Number of klines per request (max 1000)")
	)

	flag.Parse()

	if *limit > 1000 {
		*limit = 1000 // Bybit max limit
	}

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

	// Build category list
	catList := []string{}
	if strings.TrimSpace(*categories) != "" {
		for _, cat := range strings.Split(*categories, ",") {
			catVal := strings.ToLower(strings.TrimSpace(cat))
			if catVal != "" {
				catList = append(catList, catVal)
			}
		}
	} else {
		catList = []string{strings.ToLower(strings.TrimSpace(*category))}
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

	// Print Bybit API info
	fmt.Println("🚀 Bybit Historical Data Downloader")
	fmt.Println("====================================")
	fmt.Printf("📊 Categories: %s\n", strings.Join(catList, ", "))
	fmt.Printf("🎯 Symbols: %s\n", strings.Join(symList, ", "))
	fmt.Printf("⏱️  Intervals: %s\n", strings.Join(intList, ", "))
	fmt.Printf("📅 Date Range: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Println()

	// Decide if we're in single or batch mode
	singleMode := len(symList) == 1 && len(intList) == 1 && len(catList) == 1

	if singleMode && strings.TrimSpace(*output) != "" {
		// Use explicit output path
		downloadOne(catList[0], symList[0], intList[0], start, end, *limit, *output)
		return
	}

	// Batch mode (or single with default directory structure)
	for _, cat := range catList {
		for _, sym := range symList {
			for _, ival := range intList {
				comboDir := filepath.Join(*outdir, cat, sym, ival)
				outPath := filepath.Join(comboDir, "candles.csv")
				downloadOne(cat, sym, ival, start, end, *limit, outPath)
			}
		}
	}

	fmt.Println("\n🎉 All downloads completed!")
}

func downloadOne(category, symbol, interval string, start, end time.Time, limit int, outputPath string) {
	fmt.Printf("\n📊 Downloading %s %s data for %s\n", category, interval, symbol)
	fmt.Printf("📅 Period: %s to %s\n", start.Format("2006-01-02"), end.Format("2006-01-02"))
	fmt.Printf("📁 Output: %s\n", outputPath)
	fmt.Println("🔄 Fetching data...")

	klines, err := downloadBybitKlines(category, symbol, interval, start, end, limit)
	if err != nil {
		log.Printf("❌ Failed to download data for %s %s %s: %v", category, symbol, interval, err)
		return
	}

	fmt.Printf("✅ Downloaded %d klines\n", len(klines))

	// Ensure parent directory exists for this file
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		log.Printf("❌ Failed to prepare output directory %s: %v", filepath.Dir(outputPath), err)
		return
	}

	if err := saveToCSV(klines, outputPath); err != nil {
		log.Printf("❌ Failed to save %s %s %s: %v", category, symbol, interval, err)
		return
	}

	fmt.Printf("💾 Data saved to %s\n", outputPath)
	printSummary(klines, interval)
}

func printSummary(klines []BybitKlineData, interval string) {
	if len(klines) == 0 {
		return
	}
	firstKline := klines[0]
	lastKline := klines[len(klines)-1]

	fmt.Println("\n📊 DATA SUMMARY:")
	fmt.Printf("  First: %s\n", time.Unix(firstKline.StartTime/1000, 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("  Last:  %s\n", time.Unix(lastKline.StartTime/1000, 0).Format("2006-01-02 15:04:05"))
	fmt.Printf("  Total: %d %s candles\n", len(klines), formatInterval(interval))

	// Calculate some basic stats
	var totalVolume, totalTurnover float64
	highPrice := 0.0
	lowPrice := 1e9

	for _, kline := range klines {
		volume, _ := strconv.ParseFloat(kline.Volume, 64)
		turnover, _ := strconv.ParseFloat(kline.Turnover, 64)
		high, _ := strconv.ParseFloat(kline.High, 64)
		low, _ := strconv.ParseFloat(kline.Low, 64)

		totalVolume += volume
		totalTurnover += turnover
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
	fmt.Printf("  Avg Turnover: $%.2f\n", totalTurnover/float64(len(klines)))
}

func formatInterval(interval string) string {
	switch interval {
	case "1":
		return "1m"
	case "3":
		return "3m"
	case "5":
		return "5m"
	case "15":
		return "15m"
	case "30":
		return "30m"
	case "60":
		return "1h"
	case "120":
		return "2h"
	case "240":
		return "4h"
	case "360":
		return "6h"
	case "720":
		return "12h"
	case "D":
		return "1d"
	case "W":
		return "1w"
	case "M":
		return "1M"
	default:
		return interval
	}
}

func downloadBybitKlines(category, symbol, interval string, start, end time.Time, limit int) ([]BybitKlineData, error) {
	var allKlines []BybitKlineData

	// Convert times to milliseconds
	startMs := start.Unix() * 1000
	endMs := end.Unix() * 1000
	currentEndMs := endMs

	for currentEndMs > startMs {
		// Build Bybit API URL
		// Use 'end' parameter since Bybit returns data in descending order (newest first)
		url := fmt.Sprintf("https://api.bybit.com/v5/market/kline?category=%s&symbol=%s&interval=%s&end=%d&limit=%d",
			category, symbol, interval, currentEndMs, limit)

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
		var bybitResp BybitResponse
		if err := json.NewDecoder(resp.Body).Decode(&bybitResp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("JSON decode error: %w", err)
		}
		resp.Body.Close()

		// Check for API errors
		if bybitResp.RetCode != 0 {
			return nil, fmt.Errorf("Bybit API error %d: %s", bybitResp.RetCode, bybitResp.RetMsg)
		}

		if len(bybitResp.Result.List) == 0 {
			break
		}

		// Convert to BybitKlineData
		oldestTimestamp := int64(0)
		for _, raw := range bybitResp.Result.List {
			if len(raw) < 7 {
				continue
			}

			// Bybit format: [startTime, openPrice, highPrice, lowPrice, closePrice, volume, turnover]
			startTime, err := strconv.ParseInt(raw[0], 10, 64)
			if err != nil {
				continue
			}

			kline := BybitKlineData{
				StartTime: startTime,
				Open:      raw[1],
				High:      raw[2],
				Low:       raw[3],
				Close:     raw[4],
				Volume:    raw[5],
				Turnover:  raw[6],
			}

			// Only add if within our time range
			if kline.StartTime >= startMs && kline.StartTime <= endMs {
				allKlines = append(allKlines, kline)
			}

			// Track the oldest timestamp in this batch
			if oldestTimestamp == 0 || startTime < oldestTimestamp {
				oldestTimestamp = startTime
			}
		}

		// If we haven't reached our start time yet, continue with the oldest timestamp
		if oldestTimestamp <= startMs {
			break
		}

		// Update end time for next request (go back in time)
		// Since Bybit returns data in descending order, we want the timestamp just before the oldest one we got
		currentEndMs = oldestTimestamp - 1

		// Progress indicator
		fmt.Printf("\r  Progress: %d klines downloaded...", len(allKlines))

		// Rate limiting (Bybit allows up to 120 requests per minute for public endpoints)
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println() // New line after progress

	// Sort klines by timestamp (ascending order)
	// Bybit returns data in descending order, so we need to reverse it
	for i, j := 0, len(allKlines)-1; i < j; i, j = i+1, j-1 {
		allKlines[i], allKlines[j] = allKlines[j], allKlines[i]
	}

	return allKlines, nil
}

func saveToCSV(klines []BybitKlineData, filepath string) error {
	file, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"timestamp", "open", "high", "low", "close", "volume", "turnover"}); err != nil {
		return err
	}

	// Write data
	for _, kline := range klines {
		timestamp := time.Unix(kline.StartTime/1000, 0).Format("2006-01-02 15:04:05")

		record := []string{
			timestamp,
			kline.Open,
			kline.High,
			kline.Low,
			kline.Close,
			kline.Volume,
			kline.Turnover,
		}

		if err := writer.Write(record); err != nil {
			return err
		}
	}

	return nil
}
