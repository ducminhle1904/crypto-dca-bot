package main

import (
	"fmt"
	"log"
)

func main() {
	fmt.Println("🧪 Testing Excel Data Loading...")

	// Test Excel data loading
	loader := NewDataLoader("examples/data")

	data, err := loader.LoadBTCData()
	if err != nil {
		log.Fatalf("Failed to load data: %v", err)
	}

	fmt.Printf("✅ Successfully loaded %d data points\n", len(data))

	// Validate data
	if err := loader.ValidateData(data); err != nil {
		log.Fatalf("Data validation failed: %v", err)
	}

	// Get summary
	summary := loader.GetDataSummary(data)
	fmt.Printf("📊 Data Summary: %s\n", summary.String())

	// Show first few records
	fmt.Println("\n📈 First 5 records:")
	for i := 0; i < 5 && i < len(data); i++ {
		record := data[i]
		fmt.Printf("  %s: O=%.2f H=%.2f L=%.2f C=%.2f V=%.2f\n",
			record.Timestamp.Format("2006-01-02 15:04"),
			record.Open, record.High, record.Low, record.Close, record.Volume)
	}

	fmt.Println("\n🎉 Excel data loading test completed successfully!")
}
