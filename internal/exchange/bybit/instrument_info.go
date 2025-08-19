package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	bybit_api "github.com/bybit-exchange/bybit.go.api"
)

// InstrumentInfo represents detailed information about a trading instrument
type InstrumentInfo struct {
	Symbol           string `json:"symbol"`
	Status           string `json:"status"`
	BaseCoin         string `json:"baseCoin"`
	QuoteCoin        string `json:"quoteCoin"`
	Category         string `json:"category"`
	ContractType     string `json:"contractType"`
	LaunchTime       string `json:"launchTime"`
	DeliveryTime     string `json:"deliveryTime"`
	DeliveryFeeRate  string `json:"deliveryFeeRate"`
	PriceScale       string `json:"priceScale"`
	LeverageFilter   struct {
		MinLeverage string `json:"minLeverage"`
		MaxLeverage string `json:"maxLeverage"`
		LeverageStep string `json:"leverageStep"`
	} `json:"leverageFilter"`
	PriceFilter struct {
		MinPrice string `json:"minPrice"`
		MaxPrice string `json:"maxPrice"`
		TickSize string `json:"tickSize"`
	} `json:"priceFilter"`
	LotSizeFilter struct {
		MinNotionalValue    string `json:"minNotionalValue"`
		MaxOrderQty         string `json:"maxOrderQty"`
		MaxMktOrderQty      string `json:"maxMktOrderQty"`
		MinOrderQty         string `json:"minOrderQty"`
		QtyStep             string `json:"qtyStep"`
		PostOnlyMaxOrderQty string `json:"postOnlyMaxOrderQty"`
	} `json:"lotSizeFilter"`
	UnifiedMarginTrade bool   `json:"unifiedMarginTrade"`
	FundingInterval    int    `json:"fundingInterval"`
	SettleCoin         string `json:"settleCoin"`
	CopyTrading        string `json:"copyTrading"`
	UpperFundingRate   string `json:"upperFundingRate"`
	LowerFundingRate   string `json:"lowerFundingRate"`
	DisplayName        string `json:"displayName"`
	IsPreListing       bool   `json:"isPreListing"`
}

// InstrumentManager manages instrument information and provides quantity validation
type InstrumentManager struct {
	client        *Client
	instruments   map[string]*InstrumentInfo
	mutex         sync.RWMutex
	lastUpdate    time.Time
	updateInterval time.Duration
}

// NewInstrumentManager creates a new instrument manager
func NewInstrumentManager(client *Client) *InstrumentManager {
	return &InstrumentManager{
		client:         client,
		instruments:    make(map[string]*InstrumentInfo),
		updateInterval: 1 * time.Hour, // Update every hour
	}
}

// GetInstrumentInfo retrieves and caches instrument information
func (im *InstrumentManager) GetInstrumentInfo(ctx context.Context, category, symbol string) (*InstrumentInfo, error) {
	// Check cache first
	im.mutex.RLock()
	if instrument, exists := im.instruments[symbol]; exists && time.Since(im.lastUpdate) < im.updateInterval {
		im.mutex.RUnlock()
		return instrument, nil
	}
	im.mutex.RUnlock()

	// Fetch from API
	instrument, err := im.fetchInstrumentInfo(ctx, category, symbol)
	if err != nil {
		return nil, err
	}

	// Cache the result
	im.mutex.Lock()
	im.instruments[symbol] = instrument
	im.lastUpdate = time.Now()
	im.mutex.Unlock()

	return instrument, nil
}

// fetchInstrumentInfo fetches instrument information from Bybit API
func (im *InstrumentManager) fetchInstrumentInfo(ctx context.Context, category, symbol string) (*InstrumentInfo, error) {
	params := map[string]interface{}{
		"category": category,
	}

	if symbol != "" {
		params["symbol"] = symbol
	}

	result, err := im.client.httpClient.NewUtaBybitServiceWithParams(params).GetInstrumentInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch instrument info: %w", err)
	}

	// Parse the response
	instrument, err := im.parseInstrumentInfoResponse(result, symbol)
	if err != nil {
		return nil, fmt.Errorf("failed to parse instrument info: %w", err)
	}

	return instrument, nil
}

// parseInstrumentInfoResponse parses the instrument info API response
func (im *InstrumentManager) parseInstrumentInfoResponse(response interface{}, targetSymbol string) (*InstrumentInfo, error) {
	// Convert response to ServerResponse first
	serverResp, ok := response.(*bybit_api.ServerResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type")
	}

	if serverResp.RetCode != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", serverResp.RetMsg, serverResp.RetCode)
	}

	// Parse the result
	resultBytes, err := json.Marshal(serverResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var instrumentResult struct {
		Category string `json:"category"`
		List     []struct {
			Symbol           string `json:"symbol"`
			Status           string `json:"status"`
			BaseCoin         string `json:"baseCoin"`
			QuoteCoin        string `json:"quoteCoin"`
			Category         string `json:"category"`
			ContractType     string `json:"contractType"`
			LaunchTime       string `json:"launchTime"`
			DeliveryTime     string `json:"deliveryTime"`
			DeliveryFeeRate  string `json:"deliveryFeeRate"`
			PriceScale       string `json:"priceScale"`
			LeverageFilter   struct {
				MinLeverage string `json:"minLeverage"`
				MaxLeverage string `json:"maxLeverage"`
				LeverageStep string `json:"leverageStep"`
			} `json:"leverageFilter"`
			PriceFilter struct {
				MinPrice string `json:"minPrice"`
				MaxPrice string `json:"maxPrice"`
				TickSize string `json:"tickSize"`
			} `json:"priceFilter"`
			LotSizeFilter struct {
				MinNotionalValue    string `json:"minNotionalValue"`
				MaxOrderQty         string `json:"maxOrderQty"`
				MaxMktOrderQty      string `json:"maxMktOrderQty"`
				MinOrderQty         string `json:"minOrderQty"`
				QtyStep             string `json:"qtyStep"`
				PostOnlyMaxOrderQty string `json:"postOnlyMaxOrderQty"`
			} `json:"lotSizeFilter"`
			UnifiedMarginTrade bool   `json:"unifiedMarginTrade"`
			FundingInterval    int    `json:"fundingInterval"`
			SettleCoin         string `json:"settleCoin"`
			CopyTrading        string `json:"copyTrading"`
			UpperFundingRate   string `json:"upperFundingRate"`
			LowerFundingRate   string `json:"lowerFundingRate"`
			DisplayName        string `json:"displayName"`
			IsPreListing       bool   `json:"isPreListing"`
		} `json:"list"`
	}

	if err := json.Unmarshal(resultBytes, &instrumentResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instrument result: %w", err)
	}

	// Find the target symbol
	var targetInstrument *InstrumentInfo
	for _, item := range instrumentResult.List {
		if item.Symbol == targetSymbol {
			targetInstrument = &InstrumentInfo{
				Symbol:           item.Symbol,
				BaseCoin:         item.BaseCoin,
				QuoteCoin:        item.QuoteCoin,
				Category:         item.Category,
				ContractType:     item.ContractType,
				LaunchTime:       item.LaunchTime,
				DeliveryTime:     item.DeliveryTime,
				DeliveryFeeRate:  item.DeliveryFeeRate,
				PriceScale:       item.PriceScale,
				LeverageFilter:   item.LeverageFilter,
				PriceFilter:      item.PriceFilter,
				LotSizeFilter:    item.LotSizeFilter,
				UnifiedMarginTrade: item.UnifiedMarginTrade,
				FundingInterval:    item.FundingInterval,
				CopyTrading:        item.CopyTrading,
				SettleCoin:         item.SettleCoin,
				UpperFundingRate:   item.UpperFundingRate,
				LowerFundingRate:   item.LowerFundingRate,
				DisplayName:        item.DisplayName,
				IsPreListing:       item.IsPreListing,
			}
			break
		}
	}

	if targetInstrument == nil {
		return nil, fmt.Errorf("instrument %s not found", targetSymbol)
	}

	return targetInstrument, nil
}

// ValidateAndAdjustQuantity validates and adjusts a quantity based on real instrument constraints
func (im *InstrumentManager) ValidateAndAdjustQuantity(ctx context.Context, category, symbol, qty string) (string, error) {
	// Get instrument info
	instrument, err := im.GetInstrumentInfo(ctx, category, symbol)
	if err != nil {
		return "", fmt.Errorf("failed to get instrument info: %w", err)
	}

	// Parse the original quantity
	originalQty, err := strconv.ParseFloat(qty, 64)
	if err != nil {
		return "", fmt.Errorf("invalid quantity format: %w", err)
	}

	// Get constraints from instrument info
	minQty := parseFloat64(instrument.LotSizeFilter.MinOrderQty)
	maxQty := parseFloat64(instrument.LotSizeFilter.MaxOrderQty)
	qtyStep := parseFloat64(instrument.LotSizeFilter.QtyStep)

	// Apply constraints
	adjustedQty := im.applyQuantityConstraints(originalQty, minQty, maxQty, qtyStep)
	
	return strconv.FormatFloat(adjustedQty, 'f', -1, 64), nil
}

// applyQuantityConstraints applies instrument-specific quantity constraints
func (im *InstrumentManager) applyQuantityConstraints(qty, minQty, maxQty, qtyStep float64) float64 {
	// Apply minimum quantity constraint
	if qty < minQty {
		qty = minQty
	}
	
	// Apply maximum quantity constraint
	if qty > maxQty {
		qty = maxQty
	}
	
	// Apply quantity step constraint
	if qtyStep > 0 {
		steps := math.Round(qty / qtyStep)
		qty = steps * qtyStep
	}
	
	// Round to appropriate precision based on qtyStep
	if qtyStep > 0 {
		precision := int(math.Abs(math.Log10(qtyStep)))
		multiplier := math.Pow(10, float64(precision))
		qty = math.Round(qty*multiplier) / multiplier
	}
	
	return qty
}

// GetQuantityConstraints returns the quantity constraints for a symbol
func (im *InstrumentManager) GetQuantityConstraints(ctx context.Context, category, symbol string) (minQty, maxQty, qtyStep float64, err error) {
	instrument, err := im.GetInstrumentInfo(ctx, category, symbol)
	if err != nil {
		return 0, 0, 0, err
	}

	minQty = parseFloat64(instrument.LotSizeFilter.MinOrderQty)
	maxQty = parseFloat64(instrument.LotSizeFilter.MaxOrderQty)
	qtyStep = parseFloat64(instrument.LotSizeFilter.QtyStep)

	return minQty, maxQty, qtyStep, nil
}

// ValidateQuantityFormat checks if a quantity meets the instrument's requirements
func (im *InstrumentManager) ValidateQuantityFormat(ctx context.Context, category, symbol, qty string) error {
	// Get constraints
	minQty, maxQty, qtyStep, err := im.GetQuantityConstraints(ctx, category, symbol)
	if err != nil {
		return fmt.Errorf("failed to get quantity constraints: %w", err)
	}

	// Parse quantity
	qtyFloat, err := strconv.ParseFloat(qty, 64)
	if err != nil {
		return fmt.Errorf("quantity must be a valid number: %w", err)
	}

	// Check minimum quantity
	if qtyFloat < minQty {
		return fmt.Errorf("quantity %s is below minimum %f", qty, minQty)
	}

	// Check maximum quantity
	if qtyFloat > maxQty {
		return fmt.Errorf("quantity %s is above maximum %f", qty, maxQty)
	}

	// Check quantity step alignment
	if qtyStep > 0 {
		remainder := math.Mod(qtyFloat, qtyStep)
		if remainder > 0.000001 { // Small tolerance for floating point precision
			return fmt.Errorf("quantity %s is not aligned with step size %f", qty, qtyStep)
		}
	}

	return nil
}

// RefreshInstruments refreshes the instrument cache
func (im *InstrumentManager) RefreshInstruments(ctx context.Context) error {
	im.mutex.Lock()
	defer im.mutex.Unlock()

	// Clear cache
	im.instruments = make(map[string]*InstrumentInfo)
	im.lastUpdate = time.Time{}

	return nil
}

// GetCachedInstruments returns all cached instruments
func (im *InstrumentManager) GetCachedInstruments() map[string]*InstrumentInfo {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	// Return a copy to avoid race conditions
	instruments := make(map[string]*InstrumentInfo)
	for k, v := range im.instruments {
		instruments[k] = v
	}

	return instruments
}

// IsInstrumentCached checks if an instrument is cached
func (im *InstrumentManager) IsInstrumentCached(symbol string) bool {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	_, exists := im.instruments[symbol]
	return exists
}

// GetCacheInfo returns cache information
func (im *InstrumentManager) GetCacheInfo() (int, time.Time) {
	im.mutex.RLock()
	defer im.mutex.RUnlock()

	return len(im.instruments), im.lastUpdate
}

// Helper methods to convert string fields to boolean
func (ii *InstrumentInfo) IsUnifiedMarginTrade() bool {
	return ii.UnifiedMarginTrade
}

func (ii *InstrumentInfo) IsCopyTrading() bool {
	return ii.CopyTrading == "true" || ii.CopyTrading == "1"
}

func (ii *InstrumentInfo) GetIsPreListing() bool {
	return ii.IsPreListing
}

// Debug method to log instrument info
func (ii *InstrumentInfo) LogDebug() {
	fmt.Printf("Instrument Debug Info for %s:\n", ii.Symbol)
	fmt.Printf("  Status: %s\n", ii.Status)
	fmt.Printf("  UnifiedMarginTrade: %t\n", ii.UnifiedMarginTrade)
	fmt.Printf("  CopyTrading: %s (bool: %t)\n", ii.CopyTrading, ii.IsCopyTrading())
	fmt.Printf("  IsPreListing: %t\n", ii.GetIsPreListing())
	fmt.Printf("  QtyStep: %s\n", ii.LotSizeFilter.QtyStep)
	fmt.Printf("  MinOrderQty: %s\n", ii.LotSizeFilter.MinOrderQty)
	fmt.Printf("  MaxOrderQty: %s\n", ii.LotSizeFilter.MaxOrderQty)
	fmt.Printf("  MinNotionalValue: %s\n", ii.LotSizeFilter.MinNotionalValue)
	fmt.Printf("  MaxMktOrderQty: %s\n", ii.LotSizeFilter.MaxMktOrderQty)
}
