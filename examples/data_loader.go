package main

/*
Data Loader for BTC Market Data
===============================

This package provides functionality to load and process Bitcoin market data
from various sources including Excel files, CSV files, and API endpoints.

Features:
- Excel file parsing (.xlsx format)
- CSV file parsing
- Data validation and cleaning
- OHLCV data structure conversion
- Historical data analysis
*/

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/ducminhle1904/crypto-dca-bot/pkg/types"
	"github.com/xuri/excelize/v2"
)

// DataLoader handles loading market data from various sources
type DataLoader struct {
	dataPath string
}

// NewDataLoader creates a new data loader instance
func NewDataLoader(dataPath string) *DataLoader {
	return &DataLoader{
		dataPath: dataPath,
	}
}

// LoadBTCData loads Bitcoin data from the configured data path
func (dl *DataLoader) LoadBTCData() ([]types.OHLCV, error) {
	log.Printf("🔍 Attempting to load data from: %s", dl.dataPath)

	// Try to load from Excel file first
	log.Println("📊 Trying to load from Excel file...")
	if data, err := dl.loadFromExcel(); err == nil {
		log.Printf("✅ Loaded %d records from Excel file", len(data))
		return data, nil
	} else {
		log.Printf("❌ Failed to load from Excel: %v", err)
	}

	// Try to load from CSV file
	log.Println("📊 Trying to load from CSV file...")
	if data, err := dl.loadFromCSV(); err == nil {
		log.Printf("✅ Loaded %d records from CSV file", len(data))
		return data, nil
	} else {
		log.Printf("❌ Failed to load from CSV: %v", err)
	}

	// Fallback to sample data
	log.Println("⚠️  Could not load data from files, using sample data")
	return dl.generateSampleBTCData(), nil
}

// loadFromExcel attempts to load data from Excel file
func (dl *DataLoader) loadFromExcel() ([]types.OHLCV, error) {
	excelPath := dl.dataPath + "/bitcoin.xlsx"
	if _, err := os.Stat(excelPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("Excel file not found: %s", excelPath)
	}

	log.Printf("📊 Loading from Excel: %s", excelPath)

	// Open Excel file
	f, err := excelize.OpenFile(excelPath)
	if err != nil {
		log.Printf("❌ Failed to open Excel file: %v", err)
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer f.Close()

	// Get the first sheet
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("no sheets found in Excel file")
	}

	sheetName := sheets[0]
	log.Printf("📋 Reading sheet: %s", sheetName)

	// Get all rows from the sheet
	rows, err := f.GetRows(sheetName)
	if err != nil {
		log.Printf("❌ Failed to get rows: %v", err)
		return nil, fmt.Errorf("failed to get rows: %w", err)
	}

	log.Printf("📋 Found %d rows in sheet", len(rows))

	if len(rows) < 2 {
		log.Printf("❌ Insufficient data: only %d rows found", len(rows))
		return nil, fmt.Errorf("insufficient data in Excel file")
	}

	var data []types.OHLCV

	// Skip header row and process data rows
	log.Printf("📋 Processing %d data rows (skipping header)", len(rows)-1)

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 6 {
			log.Printf("⚠️  Skipping row %d: insufficient columns (%d < 6)", i, len(row))
			continue // Skip incomplete rows
		}

		// Parse timestamp (assuming it's in the first column)
		// Try different timestamp formats
		var timestamp time.Time
		var err error

		// Try parsing as Excel date number
		if excelDate, err := strconv.ParseFloat(row[0], 64); err == nil {
			timestamp, err = excelize.ExcelDateToTime(excelDate, false)
			if err != nil {
				log.Printf("⚠️  Row %d: invalid Excel date format: %s", i, row[0])
				continue
			}
		} else {
			// Try parsing as string date
			timestamp, err = time.Parse("2006-01-02", row[0])
			if err != nil {
				timestamp, err = time.Parse("2006-01-02 15:04:05", row[0])
				if err != nil {
					log.Printf("⚠️  Row %d: invalid date format: %s", i, row[0])
					continue // Skip rows with invalid dates
				}
			}
		}

		// Parse OHLCV values
		open, err := strconv.ParseFloat(row[1], 64)
		if err != nil {
			log.Printf("⚠️  Row %d: invalid open price: %s", i, row[1])
			continue
		}

		high, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			log.Printf("⚠️  Row %d: invalid high price: %s", i, row[2])
			continue
		}

		low, err := strconv.ParseFloat(row[3], 64)
		if err != nil {
			log.Printf("⚠️  Row %d: invalid low price: %s", i, row[3])
			continue
		}

		close, err := strconv.ParseFloat(row[4], 64)
		if err != nil {
			log.Printf("⚠️  Row %d: invalid close price: %s", i, row[4])
			continue
		}

		volume, err := strconv.ParseFloat(row[5], 64)
		if err != nil {
			log.Printf("⚠️  Row %d: invalid volume: %s", i, row[5])
			continue
		}

		data = append(data, types.OHLCV{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("no valid data found in Excel file")
	}

	log.Printf("📈 Successfully parsed %d records from Excel", len(data))
	return data, nil
}

// loadFromCSV attempts to load data from CSV file
func (dl *DataLoader) loadFromCSV() ([]types.OHLCV, error) {
	csvPath := dl.dataPath + "/BTCUSDT_1h.csv"
	if _, err := os.Stat(csvPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("CSV file not found: %s", csvPath)
	}

	file, err := os.Open(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open CSV file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header if present
	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV header: %w", err)
	}

	var data []types.OHLCV

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("Warning: skipping invalid CSV row: %v", err)
			continue
		}

		// Parse CSV record
		// Expected format: timestamp,open,high,low,close,volume
		if len(record) < 6 {
			continue
		}

		// Parse timestamp (assuming Unix timestamp)
		timestamp, err := strconv.ParseInt(record[0], 10, 64)
		if err != nil {
			continue
		}

		// Parse OHLCV values
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

		data = append(data, types.OHLCV{
			Timestamp: time.Unix(timestamp, 0),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}

	return data, nil
}

// generateSampleBTCData creates realistic sample Bitcoin data
func (dl *DataLoader) generateSampleBTCData() []types.OHLCV {
	data := make([]types.OHLCV, 0)
	basePrice := 45000.0
	startTime := time.Now().AddDate(0, -3, 0) // 3 months ago

	// Seed for reproducible results
	rand.Seed(42)

	for i := 0; i < 1000; i++ {
		// Simulate realistic price movements with trend and volatility
		trend := 0.0001    // Slight upward trend
		volatility := 0.02 // 2% daily volatility

		// Random walk with trend
		change := (rand.Float64()-0.5)*volatility + trend
		basePrice *= (1 + change)

		// Ensure price stays reasonable
		if basePrice < 10000 {
			basePrice = 10000
		}
		if basePrice > 100000 {
			basePrice = 100000
		}

		// Generate OHLC from base price
		open := basePrice
		high := basePrice * (1 + rand.Float64()*0.01)
		low := basePrice * (1 - rand.Float64()*0.01)
		close := basePrice * (1 + (rand.Float64()-0.5)*0.005)

		// Ensure OHLC relationships are valid
		if high < open {
			high = open
		}
		if high < close {
			high = close
		}
		if low > open {
			low = open
		}
		if low > close {
			low = close
		}

		volume := rand.Float64()*1000 + 100

		data = append(data, types.OHLCV{
			Timestamp: startTime.AddDate(0, 0, i),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})
	}

	return data
}

// ValidateData checks if the loaded data is valid
func (dl *DataLoader) ValidateData(data []types.OHLCV) error {
	if len(data) == 0 {
		return fmt.Errorf("no data loaded")
	}

	for i, candle := range data {
		// Check OHLC relationships
		if candle.High < candle.Open || candle.High < candle.Close {
			return fmt.Errorf("invalid high price at index %d", i)
		}
		if candle.Low > candle.Open || candle.Low > candle.Close {
			return fmt.Errorf("invalid low price at index %d", i)
		}
		if candle.Volume < 0 {
			return fmt.Errorf("negative volume at index %d", i)
		}
	}

	return nil
}

// GetDataSummary returns a summary of the loaded data
func (dl *DataLoader) GetDataSummary(data []types.OHLCV) DataSummary {
	if len(data) == 0 {
		return DataSummary{}
	}

	summary := DataSummary{
		TotalRecords: len(data),
		StartDate:    data[0].Timestamp,
		EndDate:      data[len(data)-1].Timestamp,
		MinPrice:     data[0].Low,
		MaxPrice:     data[0].High,
		TotalVolume:  0,
	}

	for _, candle := range data {
		if candle.Low < summary.MinPrice {
			summary.MinPrice = candle.Low
		}
		if candle.High > summary.MaxPrice {
			summary.MaxPrice = candle.High
		}
		summary.TotalVolume += candle.Volume
	}

	summary.AveragePrice = (summary.MinPrice + summary.MaxPrice) / 2
	summary.AverageVolume = summary.TotalVolume / float64(len(data))

	return summary
}

// DataSummary contains summary statistics for market data
type DataSummary struct {
	TotalRecords  int
	StartDate     time.Time
	EndDate       time.Time
	MinPrice      float64
	MaxPrice      float64
	AveragePrice  float64
	TotalVolume   float64
	AverageVolume float64
}

func (ds DataSummary) String() string {
	return fmt.Sprintf(
		"Records: %d | Date Range: %s to %s | Price Range: $%.2f - $%.2f | Avg Volume: %.2f",
		ds.TotalRecords,
		ds.StartDate.Format("2006-01-02"),
		ds.EndDate.Format("2006-01-02"),
		ds.MinPrice,
		ds.MaxPrice,
		ds.AverageVolume,
	)
}
