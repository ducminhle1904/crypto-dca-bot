package bybit

import (
	"strconv"
	"time"
)

// ServerResponse represents the standard Bybit API response wrapper
type ServerResponse struct {
	RetCode    int         `json:"retCode"`
	RetMsg     string      `json:"retMsg"`
	Result     interface{} `json:"result"`
	RetExtInfo interface{} `json:"retExtInfo"`
	Time       int64       `json:"time"`
}

// KlineResponse represents the response for kline data
type KlineResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Symbol   string      `json:"symbol"`
		Category string      `json:"category"`
		List     [][]string  `json:"list"` // Array of arrays containing kline data
	} `json:"result"`
	RetExtInfo interface{} `json:"retExtInfo"`
	Time       int64       `json:"time"`
}

// TickerResponse represents the response for ticker data
type TickerResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		Category string `json:"category"`
		List     []struct {
			Symbol                 string `json:"symbol"`
			Bid1Price              string `json:"bid1Price"`
			Bid1Size               string `json:"bid1Size"`
			Ask1Price              string `json:"ask1Price"`
			Ask1Size               string `json:"ask1Size"`
			LastPrice              string `json:"lastPrice"`
			PrevPrice24h           string `json:"prevPrice24h"`
			Price24hPcnt           string `json:"price24hPcnt"`
			HighPrice24h           string `json:"highPrice24h"`
			LowPrice24h            string `json:"lowPrice24h"`
			Turnover24h            string `json:"turnover24h"`
			Volume24h              string `json:"volume24h"`
			UsdIndexPrice          string `json:"usdIndexPrice"`
			MarkPrice              string `json:"markPrice"`
			OpenInterest           string `json:"openInterest"`
			OpenInterestValue      string `json:"openInterestValue"`
			NextFundingTime        string `json:"nextFundingTime"`
			PredictedDeliveryPrice string `json:"predictedDeliveryPrice"`
			BasisRate              string `json:"basisRate"`
			DeliveryFeeRate        string `json:"deliveryFeeRate"`
			DeliveryTime           string `json:"deliveryTime"`
			FundingRate            string `json:"fundingRate"`
			PrevFundingRate        string `json:"prevFundingRate"`
		} `json:"list"`
	} `json:"result"`
	RetExtInfo interface{} `json:"retExtInfo"`
	Time       int64       `json:"time"`
}

// WalletBalanceResponse represents the response for wallet balance
type WalletBalanceResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			TotalEquity            string `json:"totalEquity"`
			AccountIMRate          string `json:"accountIMRate"`
			TotalMarginBalance     string `json:"totalMarginBalance"`
			TotalInitialMargin     string `json:"totalInitialMargin"`
			AccountType            string `json:"accountType"`
			TotalAvailableBalance  string `json:"totalAvailableBalance"`
			TotalPerpUPL           string `json:"totalPerpUPL"`
			TotalWalletBalance     string `json:"totalWalletBalance"`
			AccountMMRate          string `json:"accountMMRate"`
			TotalMaintenanceMargin string `json:"totalMaintenanceMargin"`
			Coin                   []struct {
				Coin                string `json:"coin"`
				Equity              string `json:"equity"`
				UsdValue            string `json:"usdValue"`
				WalletBalance       string `json:"walletBalance"`
				AvailableToTrade    string `json:"availableToTrade"`
				AvailableToWithdraw string `json:"availableToWithdraw"`
				BorrowAmount        string `json:"borrowAmount"`
				AccruedInterest     string `json:"accruedInterest"`
				TotalOrderIM        string `json:"totalOrderIM"`
				TotalPositionIM     string `json:"totalPositionIM"`
				TotalPositionMM     string `json:"totalPositionMM"`
				UnrealisedPnl       string `json:"unrealisedPnl"`
				CumRealisedPnl      string `json:"cumRealisedPnl"`
				Bonus               string `json:"bonus"`
				MarginCollateral    bool   `json:"marginCollateral"`
				CollateralSwitch    bool   `json:"collateralSwitch"`
			} `json:"coin"`
		} `json:"list"`
	} `json:"result"`
	RetExtInfo interface{} `json:"retExtInfo"`
	Time       int64       `json:"time"`
}

// OrderResponse represents the response for order operations
type OrderResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
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
		TpTriggerBy   string `json:"tpTriggerBy"`
		SlTriggerBy   string `json:"slTriggerBy"`
		TpLimitPrice  string `json:"tpLimitPrice"`
		SlLimitPrice  string `json:"slLimitPrice"`
		TriggerPrice  string `json:"triggerPrice"`
		TriggerBy     string `json:"triggerBy"`
		TriggerDirection int `json:"triggerDirection"`
		PlaceType     string `json:"placeType"`
		LeavesQty     string `json:"leavesQty"`
		LeavesValue   string `json:"leavesValue"`
		CumExecFee    string `json:"cumExecFee"`
		FeeCurrency   string `json:"feeCurrency"`
		ReduceOnly    bool   `json:"reduceOnly"`
		PostOnly      bool   `json:"postOnly"`
		CloseOnTrigger bool  `json:"closeOnTrigger"`
		SmpType       string `json:"smpType"`
		SmpGroup      int    `json:"smpGroup"`
		SmpOrderId    string `json:"smpOrderId"`
	} `json:"result"`
	RetExtInfo interface{} `json:"retExtInfo"`
	Time       int64       `json:"time"`
}

// OrderListResponse represents the response for order lists
type OrderListResponse struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			OrderID       string `json:"orderId"`
			OrderLinkID   string `json:"orderLinkId"`
			BlockTradeID  string `json:"blockTradeId"`
			Symbol        string `json:"symbol"`
			Price         string `json:"price"`
			Qty           string `json:"qty"`
			Side          string `json:"side"`
			IsLeverage    string `json:"isLeverage"`
			PositionIdx   int    `json:"positionIdx"`
			OrderStatus   string `json:"orderStatus"`
			CancelType    string `json:"cancelType"`
			RejectReason  string `json:"rejectReason"`
			AvgPrice      string `json:"avgPrice"`
			LeavesQty     string `json:"leavesQty"`
			LeavesValue   string `json:"leavesValue"`
			CumExecQty    string `json:"cumExecQty"`
			CumExecValue  string `json:"cumExecValue"`
			CumExecFee    string `json:"cumExecFee"`
			FeeCurrency   string `json:"feeCurrency"`
			TimeInForce   string `json:"timeInForce"`
			OrderType     string `json:"orderType"`
			StopOrderType string `json:"stopOrderType"`
			OrderIv       string `json:"orderIv"`
			TriggerPrice  string `json:"triggerPrice"`
			TakeProfit    string `json:"takeProfit"`
			StopLoss      string `json:"stopLoss"`
			TpTriggerBy   string `json:"tpTriggerBy"`
			SlTriggerBy   string `json:"slTriggerBy"`
			TriggerDirection int    `json:"triggerDirection"`
			TriggerBy     string `json:"triggerBy"`
			LastPriceOnCreated string `json:"lastPriceOnCreated"`
			ReduceOnly    bool   `json:"reduceOnly"`
			CloseOnTrigger bool  `json:"closeOnTrigger"`
			SmpType       string `json:"smpType"`
			SmpGroup      int    `json:"smpGroup"`
			SmpOrderId    string `json:"smpOrderId"`
			TpslMode      string `json:"tpslMode"`
			TpLimitPrice  string `json:"tpLimitPrice"`
			SlLimitPrice  string `json:"slLimitPrice"`
			PlaceType     string `json:"placeType"`
			CreatedTime   string `json:"createdTime"`
			UpdatedTime   string `json:"updatedTime"`
		} `json:"list"`
		NextPageCursor string `json:"nextPageCursor"`
		Category       string `json:"category"`
	} `json:"result"`
	RetExtInfo interface{} `json:"retExtInfo"`
	Time       int64       `json:"time"`
}

// Helper functions for parsing string numbers
func parseFloat64(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

func parseInt64(s string) int64 {
	if s == "" {
		return 0
	}
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

// parseTimestamp converts milliseconds timestamp to time.Time
func parseTimestamp(ts string) time.Time {
	if ts == "" {
		return time.Time{}
	}
	msec, _ := strconv.ParseInt(ts, 10, 64)
	return time.UnixMilli(msec)
}
