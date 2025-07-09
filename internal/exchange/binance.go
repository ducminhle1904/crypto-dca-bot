package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Zmey56/enhanced-dca-bot/pkg/types"
)

// BinanceExchange implements the Exchange interface for Binance
type BinanceExchange struct {
	apiKey  string
	secret  string
	testnet bool
	client  *http.Client
	baseURL string
	wsConn  *WebSocketConnection
}

// NewBinanceExchange creates a new Binance exchange instance
func NewBinanceExchange(apiKey, secret string, testnet bool) *BinanceExchange {
	baseURL := "https://api.binance.com"
	if testnet {
		baseURL = "https://testnet.binance.vision"
	}

	return &BinanceExchange{
		apiKey:  apiKey,
		secret:  secret,
		testnet: testnet,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: baseURL,
	}
}

func (b *BinanceExchange) GetName() string {
	return "Binance"
}

func (b *BinanceExchange) Connect(ctx context.Context) error {
	// Test connection by getting server time
	_, err := b.getServerTime()
	if err != nil {
		return fmt.Errorf("failed to connect to Binance: %w", err)
	}
	return nil
}

func (b *BinanceExchange) Disconnect() error {
	if b.wsConn != nil {
		return b.wsConn.Close()
	}
	return nil
}

func (b *BinanceExchange) GetTicker(symbol string) (*types.Ticker, error) {
	url := fmt.Sprintf("%s/api/v3/ticker/24hr?symbol=%s", b.baseURL, symbol)

	resp, err := b.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get ticker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var tickerData struct {
		Symbol    string `json:"symbol"`
		Price     string `json:"lastPrice"`
		Volume    string `json:"volume"`
		Timestamp int64  `json:"closeTime"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tickerData); err != nil {
		return nil, fmt.Errorf("failed to decode ticker response: %w", err)
	}

	price, err := strconv.ParseFloat(tickerData.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse price: %w", err)
	}

	volume, err := strconv.ParseFloat(tickerData.Volume, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse volume: %w", err)
	}

	return &types.Ticker{
		Symbol:    tickerData.Symbol,
		Price:     price,
		Volume:    volume,
		Timestamp: time.Unix(tickerData.Timestamp/1000, 0),
	}, nil
}

func (b *BinanceExchange) GetKlines(symbol, interval string, limit int) ([]types.OHLCV, error) {
	url := fmt.Sprintf("%s/api/v3/klines?symbol=%s&interval=%s&limit=%d",
		b.baseURL, symbol, interval, limit)

	resp, err := b.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var klinesData [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&klinesData); err != nil {
		return nil, fmt.Errorf("failed to decode klines response: %w", err)
	}

	klines := make([]types.OHLCV, len(klinesData))
	for i, kline := range klinesData {
		if len(kline) < 6 {
			continue
		}

		open, _ := strconv.ParseFloat(kline[1].(string), 64)
		high, _ := strconv.ParseFloat(kline[2].(string), 64)
		low, _ := strconv.ParseFloat(kline[3].(string), 64)
		close, _ := strconv.ParseFloat(kline[4].(string), 64)
		volume, _ := strconv.ParseFloat(kline[5].(string), 64)
		timestamp := time.Unix(int64(kline[0].(float64))/1000, 0)

		klines[i] = types.OHLCV{
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			Timestamp: timestamp,
		}
	}

	return klines, nil
}

func (b *BinanceExchange) PlaceMarketOrder(symbol string, side OrderSide, quantity float64) (*types.Order, error) {
	// For now, return a mock order
	// In a real implementation, you would need to sign the request with API key
	return &types.Order{
		ID:        "mock_order_id",
		Symbol:    symbol,
		Side:      int(side),
		Quantity:  quantity,
		Price:     0, // Market order
		Status:    "FILLED",
		Timestamp: time.Now(),
	}, nil
}

func (b *BinanceExchange) GetBalance(asset string) (*types.Balance, error) {
	// For now, return a mock balance
	// In a real implementation, you would need to sign the request with API key
	return &types.Balance{
		Asset:  asset,
		Free:   10000.0, // Mock balance
		Locked: 0.0,
	}, nil
}

func (b *BinanceExchange) SubscribeToTicker(symbol string, callback func(*types.Ticker)) error {
	// This would be implemented with WebSocket
	return fmt.Errorf("not implemented")
}

func (b *BinanceExchange) StartWebSocket(ctx context.Context) error {
	wsURL := "wss://stream.binance.com:9443/ws"
	if b.testnet {
		wsURL = "wss://testnet.binance.vision/ws"
	}

	conn, err := NewWebSocketConnection(wsURL)
	if err != nil {
		return fmt.Errorf("failed to connect to WebSocket: %w", err)
	}

	b.wsConn = conn
	return nil
}

func (b *BinanceExchange) SubscribeToKlines(symbol string, interval string) error {
	if b.wsConn == nil {
		return fmt.Errorf("WebSocket not connected")
	}

	// Subscribe to kline stream
	stream := fmt.Sprintf("%s@kline_%s", symbol, interval)
	return b.wsConn.Subscribe(stream)
}

func (b *BinanceExchange) getServerTime() (int64, error) {
	url := fmt.Sprintf("%s/api/v3/time", b.baseURL)

	resp, err := b.client.Get(url)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var timeData struct {
		ServerTime int64 `json:"serverTime"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&timeData); err != nil {
		return 0, err
	}

	return timeData.ServerTime, nil
}
