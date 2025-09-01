package data

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
)

// CSVProvider implements DataProvider for CSV files
type CSVProvider struct {
	format CSVColumnMapping
}

// NewCSVProvider creates a new CSV data provider with default format
func NewCSVProvider() *CSVProvider {
	return &CSVProvider{
		format: DefaultCSVFormat,
	}
}

// NewCSVProviderWithFormat creates a new CSV data provider with custom format
func NewCSVProviderWithFormat(format CSVColumnMapping) *CSVProvider {
	return &CSVProvider{
		format: format,
	}
}

// GetName returns the name of the data provider
func (p *CSVProvider) GetName() string {
	return "CSV Provider"
}

// LoadData loads historical data from a CSV file
func (p *CSVProvider) LoadData(source string) ([]types.OHLCV, error) {
	return p.loadHistoricalDataWithFormat(source, p.format)
}

// loadHistoricalDataWithFormat loads data with a specific CSV format
func (p *CSVProvider) loadHistoricalDataWithFormat(filename string, format CSVColumnMapping) ([]types.OHLCV, error) {
	file, err := os.Open(filename)
	if err != nil {
		// If file doesn't exist, generate sample data
		if os.IsNotExist(err) {
			log.Println("⚠️  Historical data file not found, generating sample data...")
			return p.generateSampleData(), nil
		}
		return nil, err
	}
	defer file.Close()
	
	reader := csv.NewReader(file)
	
	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}
	
	var data []types.OHLCV
	
	lineNum := 1 // Start from 1 since we already read header
	for {
		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("error reading CSV at line %d: %v", lineNum, err)
		}
		lineNum++
		
		// Check minimum columns based on format
		if len(record) < format.MinColumns {
			log.Printf("⚠️ Insufficient columns at line %d (expected %d, got %d), skipping", lineNum, format.MinColumns, len(record))
			continue
		}
		
		// Parse timestamp with configurable format
		timestamp, err := time.Parse(format.DateFormat, record[format.TimestampCol])
		if err != nil {
			log.Printf("⚠️ Invalid timestamp '%s' at line %d, skipping: %v", record[format.TimestampCol], lineNum, err)
			continue
		}
		
		// Parse price data using configurable columns
		open, err := strconv.ParseFloat(record[format.OpenCol], 64)
		if err != nil {
			log.Printf("⚠️ Invalid open price '%s' at line %d, skipping: %v", record[format.OpenCol], lineNum, err)
			continue
		}
		
		high, err := strconv.ParseFloat(record[format.HighCol], 64)
		if err != nil {
			log.Printf("⚠️ Invalid high price '%s' at line %d, skipping: %v", record[format.HighCol], lineNum, err)
			continue
		}
		
		low, err := strconv.ParseFloat(record[format.LowCol], 64)
		if err != nil {
			log.Printf("⚠️ Invalid low price '%s' at line %d, skipping: %v", record[format.LowCol], lineNum, err)
			continue
		}
		
		close, err := strconv.ParseFloat(record[format.CloseCol], 64)
		if err != nil {
			log.Printf("⚠️ Invalid close price '%s' at line %d, skipping: %v", record[format.CloseCol], lineNum, err)
			continue
		}
		
		volume, err := strconv.ParseFloat(record[format.VolumeCol], 64)
		if err != nil {
			log.Printf("⚠️ Invalid volume '%s' at line %d, skipping: %v", record[format.VolumeCol], lineNum, err)
			continue
		}
		
		// Basic data validation
		if open <= 0 || high <= 0 || low <= 0 || close <= 0 {
			log.Printf("⚠️ Invalid price data (negative or zero) at line %d, skipping", lineNum)
			continue
		}
		
		if high < open || high < close || high < low {
			log.Printf("⚠️ High price is lower than other prices at line %d, skipping", lineNum)
			continue
		}
		
		if low > open || low > close || low > high {
			log.Printf("⚠️ Low price is higher than other prices at line %d, skipping", lineNum)
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
	
	return data, nil
}

// generateSampleData creates sample data for testing when no real data is available
func (p *CSVProvider) generateSampleData() []types.OHLCV {
	// Generate 365 days of sample data
	data := make([]types.OHLCV, 365*24) // Hourly data
	startTime := time.Now().AddDate(-1, 0, 0)
	basePrice := 30000.0
	
	for i := range data {
		// Simulate price movements
		volatility := 0.02
		trend := float64(i) * 0.1 // Slight upward trend
		randomWalk := (rand.Float64() - 0.5) * basePrice * volatility
		
		price := basePrice + trend + randomWalk
		
		// Ensure price stays positive
		if price < basePrice*0.5 {
			price = basePrice * 0.5
		}
		
		data[i] = types.OHLCV{
			Timestamp: startTime.Add(time.Duration(i) * time.Hour),
			Open:      price * (1 + (rand.Float64()-0.5)*0.01),
			High:      price * (1 + rand.Float64()*0.02),
			Low:       price * (1 - rand.Float64()*0.02),
			Close:     price,
			Volume:    rand.Float64() * 1000000,
		}
		
		basePrice = price
	}
	
	return data
}

// ValidateData validates the integrity of loaded data
func (p *CSVProvider) ValidateData(data []types.OHLCV) error {
	if len(data) == 0 {
		return fmt.Errorf("no data provided")
	}
	
	for i, candle := range data {
		// Validate price data
		if candle.Open <= 0 || candle.High <= 0 || candle.Low <= 0 || candle.Close <= 0 {
			return fmt.Errorf("invalid price data at index %d: prices must be positive", i)
		}
		
		if candle.High < candle.Low {
			return fmt.Errorf("invalid price data at index %d: high (%.4f) cannot be less than low (%.4f)", 
				i, candle.High, candle.Low)
		}
		
		if candle.High < candle.Open || candle.High < candle.Close {
			return fmt.Errorf("invalid price data at index %d: high (%.4f) must be >= open (%.4f) and close (%.4f)", 
				i, candle.High, candle.Open, candle.Close)
		}
		
		if candle.Low > candle.Open || candle.Low > candle.Close {
			return fmt.Errorf("invalid price data at index %d: low (%.4f) must be <= open (%.4f) and close (%.4f)", 
				i, candle.Low, candle.Open, candle.Close)
		}
		
		// Validate timestamp sequence (should be in chronological order)
		if i > 0 && candle.Timestamp.Before(data[i-1].Timestamp) {
			return fmt.Errorf("invalid timestamp sequence at index %d: timestamps must be in chronological order", i)
		}
	}
	
	return nil
}
