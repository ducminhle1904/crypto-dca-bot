package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	bybit_api "github.com/bybit-exchange/bybit.go.api"
)

// KlineInterval represents the time interval for kline data
type KlineInterval string

const (
	Interval1m   KlineInterval = "1"
	Interval3m   KlineInterval = "3"
	Interval5m   KlineInterval = "5"
	Interval15m  KlineInterval = "15"
	Interval30m  KlineInterval = "30"
	Interval1h   KlineInterval = "60"
	Interval2h   KlineInterval = "120"
	Interval4h   KlineInterval = "240"
	Interval6h   KlineInterval = "360"
	Interval12h  KlineInterval = "720"
	Interval1d   KlineInterval = "D"
	Interval1w   KlineInterval = "W"
	Interval1M   KlineInterval = "M"
)

// Kline represents a single kline/candlestick data point
type Kline struct {
	StartTime    time.Time
	OpenPrice    float64
	HighPrice    float64
	LowPrice     float64
	ClosePrice   float64
	Volume       float64
	Turnover     float64
}

// KlineParams holds parameters for fetching kline data
type KlineParams struct {
	Category string        // "spot", "linear", "inverse"
	Symbol   string        // Trading pair symbol (e.g., "BTCUSDT")
	Interval KlineInterval // Time interval
	Start    *time.Time    // Start time (optional)
	End      *time.Time    // End time (optional)
	Limit    int           // Number of records to return (max 1000, default 200)
}

// GetKlines fetches kline/candlestick data from Bybit
func (c *Client) GetKlines(ctx context.Context, params KlineParams) ([]Kline, error) {
	if params.Category == "" {
		params.Category = "spot"
	}
	if params.Limit == 0 {
		params.Limit = 200
	}
	if params.Limit > 1000 {
		params.Limit = 1000
	}

	// Build request parameters
	reqParams := map[string]interface{}{
		"category": params.Category,
		"symbol":   params.Symbol,
		"interval": string(params.Interval),
		"limit":    params.Limit,
	}

	// Add optional time filters
	if params.Start != nil {
		reqParams["start"] = params.Start.UnixMilli()
	}
	if params.End != nil {
		reqParams["end"] = params.End.UnixMilli()
	}

	// Make API call
	result, err := c.httpClient.NewUtaBybitServiceWithParams(reqParams).GetMarketKline(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	// Parse response
	klines, err := c.parseKlineResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kline response: %w", err)
	}

	return klines, nil
}

// GetLatestPrice gets the latest price for a symbol
func (c *Client) GetLatestPrice(ctx context.Context, category, symbol string) (float64, error) {
	if category == "" {
		category = "spot"
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   symbol,
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetMarketTickers(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get latest price: %w", err)
	}

	// Extract price from response
	price, err := c.parseLatestPriceResponse(result)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price response: %w", err)
	}

	return price, nil
}

// parseKlineResponse parses the API response into Kline structs
func (c *Client) parseKlineResponse(response interface{}) ([]Kline, error) {
	// Convert response to ServerResponse first
	serverResp, ok := response.(*bybit_api.ServerResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type")
	}

	if serverResp.RetCode != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", serverResp.RetMsg, serverResp.RetCode)
	}

	// Parse the result as KlineResponse
	resultBytes, err := json.Marshal(serverResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var klineResult struct {
		Symbol   string     `json:"symbol"`
		Category string     `json:"category"`
		List     [][]string `json:"list"`
	}

	if err := json.Unmarshal(resultBytes, &klineResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kline result: %w", err)
	}

	var klines []Kline
	for _, item := range klineResult.List {
		if len(item) < 7 {
			continue // Skip incomplete data
		}

		// Bybit kline format: [startTime, openPrice, highPrice, lowPrice, closePrice, volume, turnover]
		kline := Kline{
			StartTime:  time.UnixMilli(parseInt64(item[0])),
			OpenPrice:  parseFloat64(item[1]),
			HighPrice:  parseFloat64(item[2]),
			LowPrice:   parseFloat64(item[3]),
			ClosePrice: parseFloat64(item[4]),
			Volume:     parseFloat64(item[5]),
			Turnover:   parseFloat64(item[6]),
		}
		klines = append(klines, kline)
	}

	return klines, nil
}

// parseLatestPriceResponse parses the ticker response to extract the latest price
func (c *Client) parseLatestPriceResponse(response interface{}) (float64, error) {
	// Convert response to ServerResponse first
	serverResp, ok := response.(*bybit_api.ServerResponse)
	if !ok {
		return 0, fmt.Errorf("invalid response type")
	}

	if serverResp.RetCode != 0 {
		return 0, fmt.Errorf("API error: %s (code: %d)", serverResp.RetMsg, serverResp.RetCode)
	}

	// Parse the result as TickerResponse
	resultBytes, err := json.Marshal(serverResp.Result)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal result: %w", err)
	}

	var tickerResult struct {
		Category string `json:"category"`
		List     []struct {
			Symbol    string `json:"symbol"`
			LastPrice string `json:"lastPrice"`
		} `json:"list"`
	}

	if err := json.Unmarshal(resultBytes, &tickerResult); err != nil {
		return 0, fmt.Errorf("failed to unmarshal ticker result: %w", err)
	}

	if len(tickerResult.List) == 0 {
		return 0, fmt.Errorf("no ticker data found")
	}

	// Return the last price of the first (and should be only) ticker
	return parseFloat64(tickerResult.List[0].LastPrice), nil
}

// GetOrderBook gets the order book for a symbol
func (c *Client) GetOrderBook(ctx context.Context, category, symbol string, limit int) (interface{}, error) {
	if category == "" {
		category = "spot"
	}
	if limit == 0 {
		limit = 25
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   symbol,
		"limit":    limit,
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetOrderBookInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get order book: %w", err)
	}

	return result, nil
}

// FuturesMarketData contains futures-specific market information
type FuturesMarketData struct {
	Symbol           string    `json:"symbol"`
	MarkPrice        float64   `json:"markPrice"`
	IndexPrice       float64   `json:"indexPrice"`
	LastPrice        float64   `json:"lastPrice"`
	FundingRate      float64   `json:"fundingRate"`
	NextFundingTime  time.Time `json:"nextFundingTime"`
	OpenInterest     float64   `json:"openInterest"`
	Volume24h        float64   `json:"volume24h"`
	Turnover24h      float64   `json:"turnover24h"`
	Price24hPcnt     float64   `json:"price24hPcnt"`
	HighPrice24h     float64   `json:"highPrice24h"`
	LowPrice24h      float64   `json:"lowPrice24h"`
}

// GetFuturesMarketData gets comprehensive futures market data
func (c *Client) GetFuturesMarketData(ctx context.Context, category, symbol string) (*FuturesMarketData, error) {
	if category == "" {
		category = "linear" // Default to linear futures
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   symbol,
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetMarketTickers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get futures market data: %w", err)
	}

	// Parse the response
	marketData, err := c.parseFuturesMarketDataResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse futures market data response: %w", err)
	}

	return marketData, nil
}

// GetFundingRate gets the current funding rate for a futures symbol
func (c *Client) GetFundingRate(ctx context.Context, category, symbol string) (float64, time.Time, error) {
	if category == "" {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   symbol,
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetMarketTickers(ctx)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to get funding rate: %w", err)
	}

	// Parse funding rate from response
	fundingRate, nextFundingTime, err := c.parseFundingRateResponse(result)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("failed to parse funding rate response: %w", err)
	}

	return fundingRate, nextFundingTime, nil
}

// GetOpenInterest gets the open interest for a futures symbol
func (c *Client) GetOpenInterest(ctx context.Context, category, symbol string, intervalTime string) (interface{}, error) {
	if category == "" {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
		"symbol":   symbol,
	}

	if intervalTime != "" {
		params["intervalTime"] = intervalTime
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetOpenInterests(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get open interest: %w", err)
	}

	return result, nil
}

// GetInstrumentInfo gets detailed instrument information for futures
func (c *Client) GetInstrumentInfo(ctx context.Context, category, symbol string) (interface{}, error) {
	if category == "" {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
	}

	if symbol != "" {
		params["symbol"] = symbol
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetInstrumentInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get instrument info: %w", err)
	}

	return result, nil
}



// parseFuturesMarketDataResponse parses the futures market data response
func (c *Client) parseFuturesMarketDataResponse(response interface{}) (*FuturesMarketData, error) {
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

	var tickerResult struct {
		Category string `json:"category"`
		List     []struct {
			Symbol                 string `json:"symbol"`
			LastPrice              string `json:"lastPrice"`
			MarkPrice              string `json:"markPrice"`
			IndexPrice             string `json:"indexPrice"`
			FundingRate            string `json:"fundingRate"`
			NextFundingTime        string `json:"nextFundingTime"`
			OpenInterest           string `json:"openInterest"`
			Volume24h              string `json:"volume24h"`
			Turnover24h            string `json:"turnover24h"`
			Price24hPcnt           string `json:"price24hPcnt"`
			HighPrice24h           string `json:"highPrice24h"`
			LowPrice24h            string `json:"lowPrice24h"`
		} `json:"list"`
	}

	if err := json.Unmarshal(resultBytes, &tickerResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ticker result: %w", err)
	}

	if len(tickerResult.List) == 0 {
		return nil, fmt.Errorf("no futures market data found")
	}

	ticker := tickerResult.List[0]
	marketData := &FuturesMarketData{
		Symbol:           ticker.Symbol,
		LastPrice:        parseFloat64(ticker.LastPrice),
		MarkPrice:        parseFloat64(ticker.MarkPrice),
		IndexPrice:       parseFloat64(ticker.IndexPrice),
		FundingRate:      parseFloat64(ticker.FundingRate),
		NextFundingTime:  parseTimestamp(ticker.NextFundingTime),
		OpenInterest:     parseFloat64(ticker.OpenInterest),
		Volume24h:        parseFloat64(ticker.Volume24h),
		Turnover24h:      parseFloat64(ticker.Turnover24h),
		Price24hPcnt:     parseFloat64(ticker.Price24hPcnt),
		HighPrice24h:     parseFloat64(ticker.HighPrice24h),
		LowPrice24h:      parseFloat64(ticker.LowPrice24h),
	}

	return marketData, nil
}

// parseFundingRateResponse parses funding rate from ticker response
func (c *Client) parseFundingRateResponse(response interface{}) (float64, time.Time, error) {
	marketData, err := c.parseFuturesMarketDataResponse(response)
	if err != nil {
		return 0, time.Time{}, err
	}

	return marketData.FundingRate, marketData.NextFundingTime, nil
}


