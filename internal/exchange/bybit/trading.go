package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	bybit_api "github.com/bybit-exchange/bybit.go.api"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "Buy"
	OrderSideSell OrderSide = "Sell"
)

// OrderType represents the type of an order
type OrderType string

const (
	OrderTypeMarket OrderType = "Market"
	OrderTypeLimit  OrderType = "Limit"
)

// TimeInForce represents how long an order remains active
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancelled
	TimeInForceIOC TimeInForce = "IOC" // Immediate Or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill Or Kill
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "New"
	OrderStatusPartiallyFilled OrderStatus = "PartiallyFilled"
	OrderStatusFilled          OrderStatus = "Filled"
	OrderStatusCancelled       OrderStatus = "Cancelled"
	OrderStatusRejected        OrderStatus = "Rejected"
)

// Order represents a trading order
type Order struct {
	OrderID       string      `json:"orderId"`
	OrderLinkID   string      `json:"orderLinkId"`
	Symbol        string      `json:"symbol"`
	Side          OrderSide   `json:"side"`
	OrderType     OrderType   `json:"orderType"`
	Qty           string      `json:"qty"`
	Price         string      `json:"price"`
	TimeInForce   TimeInForce `json:"timeInForce"`
	OrderStatus   OrderStatus `json:"orderStatus"`
	CreatedTime   time.Time   `json:"createdTime"`
	UpdatedTime   time.Time   `json:"updatedTime"`
	CumExecQty    string      `json:"cumExecQty"`
	CumExecValue  string      `json:"cumExecValue"`
	AvgPrice      string      `json:"avgPrice"`
	StopOrderType string      `json:"stopOrderType"`
	TakeProfit    string      `json:"takeProfit"`
	StopLoss      string      `json:"stopLoss"`
}

// PlaceOrderParams holds parameters for placing an order
type PlaceOrderParams struct {
	Category      string      `json:"category"`      // "spot", "linear", "inverse", "option"
	Symbol        string      `json:"symbol"`        // Trading pair symbol
	Side          OrderSide   `json:"side"`          // Buy or Sell
	OrderType     OrderType   `json:"orderType"`     // Market or Limit
	Qty           string      `json:"qty"`           // Order quantity
	Price         string      `json:"price,omitempty"` // Price for limit orders
	TimeInForce   TimeInForce `json:"timeInForce,omitempty"` // GTC, IOC, FOK
	OrderLinkID   string      `json:"orderLinkId,omitempty"` // Unique order ID set by user
	TakeProfit    string      `json:"takeProfit,omitempty"`  // Take profit price
	StopLoss      string      `json:"stopLoss,omitempty"`    // Stop loss price
	ReduceOnly    bool        `json:"reduceOnly,omitempty"`  // Reduce only flag
	PostOnly      bool        `json:"postOnly,omitempty"`    // Post only flag
	MarketUnit    string      `json:"marketUnit,omitempty"`  // baseCoin, quoteCoin (for spot market orders)
}

// PlaceOrder places a new order
func (c *Client) PlaceOrder(ctx context.Context, params PlaceOrderParams) (*Order, error) {
	// Validate required parameters
	if params.Category == "" {
		params.Category = "spot"
	}
	if params.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if params.Side == "" {
		return nil, fmt.Errorf("side is required")
	}
	if params.OrderType == "" {
		return nil, fmt.Errorf("orderType is required")
	}
	if params.Qty == "" {
		return nil, fmt.Errorf("qty is required")
	}

	// For limit orders, price is required
	if params.OrderType == OrderTypeLimit && params.Price == "" {
		return nil, fmt.Errorf("price is required for limit orders")
	}

	// Set default time in force for limit orders
	if params.OrderType == OrderTypeLimit && params.TimeInForce == "" {
		params.TimeInForce = TimeInForceGTC
	}

	// Validate and adjust quantity using instrument info
	if c.instrumentManager != nil {
		adjustedQty, err := c.instrumentManager.ValidateAndAdjustQuantity(ctx, params.Category, params.Symbol, params.Qty)
		if err != nil {
			return nil, fmt.Errorf("quantity validation failed: %w", err)
		}
		
		// Update the quantity if it was adjusted
		if adjustedQty != params.Qty {
			params.Qty = adjustedQty
		}
	}

	// Convert params to map for API call
	apiParams := map[string]interface{}{
		"category":  params.Category,
		"symbol":    params.Symbol,
		"side":      string(params.Side),
		"orderType": string(params.OrderType),
		"qty":       params.Qty,
	}

	// Add optional parameters
	if params.Price != "" {
		apiParams["price"] = params.Price
	}
	if params.TimeInForce != "" {
		apiParams["timeInForce"] = string(params.TimeInForce)
	}
	if params.OrderLinkID != "" {
		apiParams["orderLinkId"] = params.OrderLinkID
	}
	if params.TakeProfit != "" {
		apiParams["takeProfit"] = params.TakeProfit
	}
	if params.StopLoss != "" {
		apiParams["stopLoss"] = params.StopLoss
	}
	if params.ReduceOnly {
		apiParams["reduceOnly"] = params.ReduceOnly
	}
	if params.PostOnly {
		apiParams["postOnly"] = params.PostOnly
	}
	if params.MarketUnit != "" {
		apiParams["marketUnit"] = params.MarketUnit
	}

	// Make API call
	result, err := c.httpClient.NewUtaBybitServiceWithParams(apiParams).PlaceOrder(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	// Parse response
	order, err := c.parseOrderResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse order response: %w", err)
	}

	return order, nil
}

// PlaceMarketOrder places a market order (simplified method)
func (c *Client) PlaceMarketOrder(ctx context.Context, category, symbol string, side OrderSide, qty string) (*Order, error) {
	params := PlaceOrderParams{
		Category:  category,
		Symbol:    symbol,
		Side:      side,
		OrderType: OrderTypeMarket,
		Qty:       qty,
	}

	return c.PlaceOrder(ctx, params)
}

// PlaceLimitOrder places a limit order (simplified method)
func (c *Client) PlaceLimitOrder(ctx context.Context, category, symbol string, side OrderSide, qty, price string) (*Order, error) {
	params := PlaceOrderParams{
		Category:    category,
		Symbol:      symbol,
		Side:        side,
		OrderType:   OrderTypeLimit,
		Qty:         qty,
		Price:       price,
		TimeInForce: TimeInForceGTC,
	}

	return c.PlaceOrder(ctx, params)
}

// CancelOrder cancels an existing order
func (c *Client) CancelOrder(ctx context.Context, category, symbol, orderID string) error {
	params := map[string]interface{}{
		"category": category,
		"symbol":   symbol,
		"orderId":  orderID,
	}

	_, err := c.httpClient.NewUtaBybitServiceWithParams(params).CancelOrder(ctx)
	if err != nil {
		return fmt.Errorf("failed to cancel order: %w", err)
	}

	return nil
}

// CancelAllOrders cancels all open orders for a symbol
func (c *Client) CancelAllOrders(ctx context.Context, category, symbol string) error {
	params := map[string]interface{}{
		"category": category,
		"symbol":   symbol,
	}

	_, err := c.httpClient.NewUtaBybitServiceWithParams(params).CancelAllOrders(ctx)
	if err != nil {
		return fmt.Errorf("failed to cancel all orders: %w", err)
	}

	return nil
}

// GetOpenOrders retrieves open orders
func (c *Client) GetOpenOrders(ctx context.Context, category, symbol string) ([]Order, error) {
	params := map[string]interface{}{
		"category": category,
	}

	if symbol != "" {
		params["symbol"] = symbol
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetOpenOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get open orders: %w", err)
	}

	// Parse response
	orders, err := c.parseOrdersResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse orders response: %w", err)
	}

	return orders, nil
}

// GetOrderHistory retrieves order history
func (c *Client) GetOrderHistory(ctx context.Context, category, symbol string, limit int) ([]Order, error) {
	params := map[string]interface{}{
		"category": category,
	}

	if symbol != "" {
		params["symbol"] = symbol
	}
	if limit > 0 {
		params["limit"] = limit
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetOrderHistory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get order history: %w", err)
	}

	// Parse response
	orders, err := c.parseOrdersResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse order history response: %w", err)
	}

	return orders, nil
}

// GetOrderStatus retrieves the status of a specific order
func (c *Client) GetOrderStatus(ctx context.Context, category, symbol, orderID string) (*Order, error) {
	params := map[string]interface{}{
		"category": category,
		"symbol":   symbol,
		"orderId":  orderID,
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetOpenOrders(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}

	// Parse response and find the specific order
	orders, err := c.parseOrdersResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse order status response: %w", err)
	}

	for _, order := range orders {
		if order.OrderID == orderID {
			return &order, nil
		}
	}

	return nil, fmt.Errorf("order with ID %s not found", orderID)
}

// parseOrderResponse parses the order placement API response
func (c *Client) parseOrderResponse(response interface{}) (*Order, error) {
	// Convert response to ServerResponse first
	serverResp, ok := response.(*bybit_api.ServerResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type")
	}

	if serverResp.RetCode != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", serverResp.RetMsg, serverResp.RetCode)
	}

	// Parse the result as OrderResponse
	resultBytes, err := json.Marshal(serverResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var orderResult struct {
		OrderID       string `json:"orderId"`
		OrderLinkID   string `json:"orderLinkId"`
		Symbol        string `json:"symbol"`
		CreateType    string `json:"createType"`
		OrderFilter   string `json:"orderFilter"`
		CreatedTime   string `json:"createdTime"`
		UpdatedTime   string `json:"updatedTime"`
		Side          string `json:"side"`
		OrderType     string `json:"orderType"`
		Qty           string `json:"qty"`
		Price         string `json:"price"`
		TimeInForce   string `json:"timeInForce"`
		OrderStatus   string `json:"orderStatus"`
		CumExecQty    string `json:"cumExecQty"`
		CumExecValue  string `json:"cumExecValue"`
		AvgPrice      string `json:"avgPrice"`
		StopOrderType string `json:"stopOrderType"`
		TakeProfit    string `json:"takeProfit"`
		StopLoss      string `json:"stopLoss"`
	}

	if err := json.Unmarshal(resultBytes, &orderResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order result: %w", err)
	}

	order := &Order{
		OrderID:       orderResult.OrderID,
		OrderLinkID:   orderResult.OrderLinkID,
		Symbol:        orderResult.Symbol,
		Side:          OrderSide(orderResult.Side),
		OrderType:     OrderType(orderResult.OrderType),
		Qty:           orderResult.Qty,
		Price:         orderResult.Price,
		TimeInForce:   TimeInForce(orderResult.TimeInForce),
		OrderStatus:   OrderStatus(orderResult.OrderStatus),
		CreatedTime:   parseTimestamp(orderResult.CreatedTime),
		UpdatedTime:   parseTimestamp(orderResult.UpdatedTime),
		CumExecQty:    orderResult.CumExecQty,
		CumExecValue:  orderResult.CumExecValue,
		AvgPrice:      orderResult.AvgPrice,
		StopOrderType: orderResult.StopOrderType,
		TakeProfit:    orderResult.TakeProfit,
		StopLoss:      orderResult.StopLoss,
	}

	return order, nil
}

// parseOrdersResponse parses the orders list API response
func (c *Client) parseOrdersResponse(response interface{}) ([]Order, error) {
	// Convert response to ServerResponse first
	serverResp, ok := response.(*bybit_api.ServerResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type")
	}

	if serverResp.RetCode != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", serverResp.RetMsg, serverResp.RetCode)
	}

	// Parse the result as OrderListResponse
	resultBytes, err := json.Marshal(serverResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var orderListResult struct {
		List []struct {
			OrderID       string `json:"orderId"`
			OrderLinkID   string `json:"orderLinkId"`
			Symbol        string `json:"symbol"`
			Price         string `json:"price"`
			Qty           string `json:"qty"`
			Side          string `json:"side"`
			OrderStatus   string `json:"orderStatus"`
			AvgPrice      string `json:"avgPrice"`
			CumExecQty    string `json:"cumExecQty"`
			CumExecValue  string `json:"cumExecValue"`
			TimeInForce   string `json:"timeInForce"`
			OrderType     string `json:"orderType"`
			StopOrderType string `json:"stopOrderType"`
			TriggerPrice  string `json:"triggerPrice"`
			TakeProfit    string `json:"takeProfit"`
			StopLoss      string `json:"stopLoss"`
			CreatedTime   string `json:"createdTime"`
			UpdatedTime   string `json:"updatedTime"`
		} `json:"list"`
		NextPageCursor string `json:"nextPageCursor"`
		Category       string `json:"category"`
	}

	if err := json.Unmarshal(resultBytes, &orderListResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal order list result: %w", err)
	}

	var orders []Order
	for _, orderData := range orderListResult.List {
		order := Order{
			OrderID:       orderData.OrderID,
			OrderLinkID:   orderData.OrderLinkID,
			Symbol:        orderData.Symbol,
			Side:          OrderSide(orderData.Side),
			OrderType:     OrderType(orderData.OrderType),
			Qty:           orderData.Qty,
			Price:         orderData.Price,
			TimeInForce:   TimeInForce(orderData.TimeInForce),
			OrderStatus:   OrderStatus(orderData.OrderStatus),
			CreatedTime:   parseTimestamp(orderData.CreatedTime),
			UpdatedTime:   parseTimestamp(orderData.UpdatedTime),
			CumExecQty:    orderData.CumExecQty,
			CumExecValue:  orderData.CumExecValue,
			AvgPrice:      orderData.AvgPrice,
			StopOrderType: orderData.StopOrderType,
			TakeProfit:    orderData.TakeProfit,
			StopLoss:      orderData.StopLoss,
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// FuturesOrderParams holds parameters for placing futures orders
type FuturesOrderParams struct {
	Category      string      `json:"category"`      // "linear", "inverse"
	Symbol        string      `json:"symbol"`        // Trading pair symbol
	Side          OrderSide   `json:"side"`          // Buy or Sell
	OrderType     OrderType   `json:"orderType"`     // Market or Limit
	Qty           string      `json:"qty"`           // Order quantity
	Price         string      `json:"price,omitempty"` // Price for limit orders
	TimeInForce   TimeInForce `json:"timeInForce,omitempty"` // GTC, IOC, FOK
	OrderLinkID   string      `json:"orderLinkId,omitempty"` // Unique order ID
	TakeProfit    string      `json:"takeProfit,omitempty"`  // Take profit price
	StopLoss      string      `json:"stopLoss,omitempty"`    // Stop loss price
	ReduceOnly    bool        `json:"reduceOnly,omitempty"`  // Reduce only flag
	CloseOnTrigger bool       `json:"closeOnTrigger,omitempty"` // Close on trigger
	PositionIdx   int         `json:"positionIdx,omitempty"` // Position index (0: one-way, 1: hedge buy, 2: hedge sell)
	TriggerPrice  string      `json:"triggerPrice,omitempty"` // Trigger price for conditional orders
	TriggerBy     string      `json:"triggerBy,omitempty"`    // Trigger price type
	TpTriggerBy   string      `json:"tpTriggerBy,omitempty"`  // TP trigger price type
	SlTriggerBy   string      `json:"slTriggerBy,omitempty"`  // SL trigger price type
}

// PositionInfo represents a futures position
type PositionInfo struct {
	Symbol         string    `json:"symbol"`
	Side           string    `json:"side"`
	Size           string    `json:"size"`
	PositionValue  string    `json:"positionValue"`
	EntryPrice     string    `json:"entryPrice"`
	MarkPrice      string    `json:"markPrice"`
	LiqPrice       string    `json:"liqPrice"`
	UnrealisedPnl  string    `json:"unrealisedPnl"`
	CumRealisedPnl string    `json:"cumRealisedPnl"`
	PositionMM     string    `json:"positionMM"`
	PositionIM     string    `json:"positionIM"`
	TakeProfit     string    `json:"takeProfit"`
	StopLoss       string    `json:"stopLoss"`
	TrailingStop   string    `json:"trailingStop"`
	PositionIdx    int       `json:"positionIdx"`
	RiskId         int       `json:"riskId"`
	RiskLimitValue string    `json:"riskLimitValue"`
	CreatedTime    time.Time `json:"createdTime"`
	UpdatedTime    time.Time `json:"updatedTime"`
}

// PlaceFuturesOrder places a futures order with leverage support
func (c *Client) PlaceFuturesOrder(ctx context.Context, params FuturesOrderParams) (*Order, error) {
	// Validate required parameters
	if params.Category == "" {
		params.Category = "linear" // Default to linear futures
	}
	if params.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if params.Side == "" {
		return nil, fmt.Errorf("side is required")
	}
	if params.OrderType == "" {
		return nil, fmt.Errorf("orderType is required")
	}
	if params.Qty == "" {
		return nil, fmt.Errorf("qty is required")
	}

	// For limit orders, price is required
	if params.OrderType == OrderTypeLimit && params.Price == "" {
		return nil, fmt.Errorf("price is required for limit orders")
	}

	// Set default time in force for limit orders
	if params.OrderType == OrderTypeLimit && params.TimeInForce == "" {
		params.TimeInForce = TimeInForceGTC
	}

	// Convert params to map for API call
	apiParams := map[string]interface{}{
		"category":  params.Category,
		"symbol":    params.Symbol,
		"side":      string(params.Side),
		"orderType": string(params.OrderType),
		"qty":       params.Qty,
	}

	// Add optional parameters
	if params.Price != "" {
		apiParams["price"] = params.Price
	}
	if params.TimeInForce != "" {
		apiParams["timeInForce"] = string(params.TimeInForce)
	}
	if params.OrderLinkID != "" {
		apiParams["orderLinkId"] = params.OrderLinkID
	}
	if params.TakeProfit != "" {
		apiParams["takeProfit"] = params.TakeProfit
	}
	if params.StopLoss != "" {
		apiParams["stopLoss"] = params.StopLoss
	}
	if params.ReduceOnly {
		apiParams["reduceOnly"] = params.ReduceOnly
	}
	if params.CloseOnTrigger {
		apiParams["closeOnTrigger"] = params.CloseOnTrigger
	}
	if params.PositionIdx != 0 {
		apiParams["positionIdx"] = params.PositionIdx
	}
	if params.TriggerPrice != "" {
		apiParams["triggerPrice"] = params.TriggerPrice
	}
	if params.TriggerBy != "" {
		apiParams["triggerBy"] = params.TriggerBy
	}
	if params.TpTriggerBy != "" {
		apiParams["tpTriggerBy"] = params.TpTriggerBy
	}
	if params.SlTriggerBy != "" {
		apiParams["slTriggerBy"] = params.SlTriggerBy
	}

	// Make API call
	result, err := c.httpClient.NewUtaBybitServiceWithParams(apiParams).PlaceOrder(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to place futures order: %w", err)
	}

	// Parse response
	order, err := c.parseOrderResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse futures order response: %w", err)
	}

	return order, nil
}

// PlaceFuturesMarketOrder places a market order for futures (simplified method)
func (c *Client) PlaceFuturesMarketOrder(ctx context.Context, category, symbol string, side OrderSide, qty string, positionIdx int) (*Order, error) {
	params := FuturesOrderParams{
		Category:    category,
		Symbol:      symbol,
		Side:        side,
		OrderType:   OrderTypeMarket,
		Qty:         qty,
		PositionIdx: positionIdx,
	}

	return c.PlaceFuturesOrder(ctx, params)
}

// PlaceFuturesLimitOrder places a limit order for futures (simplified method)
func (c *Client) PlaceFuturesLimitOrder(ctx context.Context, category, symbol string, side OrderSide, qty, price string, positionIdx int) (*Order, error) {
	params := FuturesOrderParams{
		Category:    category,
		Symbol:      symbol,
		Side:        side,
		OrderType:   OrderTypeLimit,
		Qty:         qty,
		Price:       price,
		TimeInForce: TimeInForceGTC,
		PositionIdx: positionIdx,
	}

	return c.PlaceFuturesOrder(ctx, params)
}

// GetPositions retrieves futures positions
func (c *Client) GetPositions(ctx context.Context, category, symbol string) ([]PositionInfo, error) {
	if category == "" {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
	}

	if symbol != "" {
		params["symbol"] = symbol
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetPositionList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	// Parse response
	positions, err := c.parsePositionsResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse positions response: %w", err)
	}

	return positions, nil
}

// SetLeverage sets the leverage for a symbol
func (c *Client) SetLeverage(ctx context.Context, category, symbol string, buyLeverage, sellLeverage string) error {
	if category == "" {
		category = "linear"
	}

	params := map[string]interface{}{
		"category":     category,
		"symbol":       symbol,
		"buyLeverage":  buyLeverage,
		"sellLeverage": sellLeverage,
	}

	_, err := c.httpClient.NewUtaBybitServiceWithParams(params).SetPositionLeverage(ctx)
	if err != nil {
		return fmt.Errorf("failed to set leverage: %w", err)
	}

	return nil
}

// SetTradingStop sets take profit and stop loss for a position
func (c *Client) SetTradingStop(ctx context.Context, category, symbol string, positionIdx int, takeProfit, stopLoss string) error {
	if category == "" {
		category = "linear"
	}

	params := map[string]interface{}{
		"category":    category,
		"symbol":      symbol,
		"positionIdx": positionIdx,
	}

	if takeProfit != "" {
		params["takeProfit"] = takeProfit
	}
	if stopLoss != "" {
		params["stopLoss"] = stopLoss
	}

	_, err := c.httpClient.NewUtaBybitServiceWithParams(params).SetPositionTradingStop(ctx)
	if err != nil {
		return fmt.Errorf("failed to set trading stop: %w", err)
	}

	return nil
}

// SwitchPositionMode switches between one-way and hedge position modes
func (c *Client) SwitchPositionMode(ctx context.Context, category, symbol, mode string) error {
	if category == "" {
		category = "linear"
	}

	params := map[string]interface{}{
		"category": category,
		"mode":     mode, // 0: one-way mode, 3: hedge mode
	}

	if symbol != "" {
		params["symbol"] = symbol
	}

	_, err := c.httpClient.NewUtaBybitServiceWithParams(params).SwitchPositionMode(ctx)
	if err != nil {
		return fmt.Errorf("failed to switch position mode: %w", err)
	}

	return nil
}

// parsePositionsResponse parses the positions API response
func (c *Client) parsePositionsResponse(response interface{}) ([]PositionInfo, error) {
	// Convert response to ServerResponse first
	serverResp, ok := response.(*bybit_api.ServerResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type")
	}

	if serverResp.RetCode != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", serverResp.RetMsg, serverResp.RetCode)
	}

	// Parse the result as position list
	resultBytes, err := json.Marshal(serverResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var positionResult struct {
		List []struct {
			Symbol         string `json:"symbol"`
			Side           string `json:"side"`
			Size           string `json:"size"`
			PositionValue  string `json:"positionValue"`
			EntryPrice     string `json:"entryPrice"`
			MarkPrice      string `json:"markPrice"`
			LiqPrice       string `json:"liqPrice"`
			UnrealisedPnl  string `json:"unrealisedPnl"`
			CumRealisedPnl string `json:"cumRealisedPnl"`
			PositionMM     string `json:"positionMM"`
			PositionIM     string `json:"positionIM"`
			TakeProfit     string `json:"takeProfit"`
			StopLoss       string `json:"stopLoss"`
			TrailingStop   string `json:"trailingStop"`
			PositionIdx    int    `json:"positionIdx"`
			RiskId         int    `json:"riskId"`
			RiskLimitValue string `json:"riskLimitValue"`
			CreatedTime    string `json:"createdTime"`
			UpdatedTime    string `json:"updatedTime"`
		} `json:"list"`
		NextPageCursor string `json:"nextPageCursor"`
		Category       string `json:"category"`
	}

	if err := json.Unmarshal(resultBytes, &positionResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal position result: %w", err)
	}

	var positions []PositionInfo
	for _, posData := range positionResult.List {
		position := PositionInfo{
			Symbol:         posData.Symbol,
			Side:           posData.Side,
			Size:           posData.Size,
			PositionValue:  posData.PositionValue,
			EntryPrice:     posData.EntryPrice,
			MarkPrice:      posData.MarkPrice,
			LiqPrice:       posData.LiqPrice,
			UnrealisedPnl:  posData.UnrealisedPnl,
			CumRealisedPnl: posData.CumRealisedPnl,
			PositionMM:     posData.PositionMM,
			PositionIM:     posData.PositionIM,
			TakeProfit:     posData.TakeProfit,
			StopLoss:       posData.StopLoss,
			TrailingStop:   posData.TrailingStop,
			PositionIdx:    posData.PositionIdx,
			RiskId:         posData.RiskId,
			RiskLimitValue: posData.RiskLimitValue,
			CreatedTime:    parseTimestamp(posData.CreatedTime),
			UpdatedTime:    parseTimestamp(posData.UpdatedTime),
		}
		positions = append(positions, position)
	}

	return positions, nil
}
