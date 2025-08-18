package bybit

import (
	"context"
	"encoding/json"
	"fmt"

	bybit_api "github.com/bybit-exchange/bybit.go.api"
)

// AccountType represents different account types in Bybit
type AccountType string

const (
	AccountTypeUnified AccountType = "UNIFIED"
	AccountTypeSpot    AccountType = "SPOT"
	AccountTypeContract AccountType = "CONTRACT"
	AccountTypeInverse AccountType = "INVERSE"
	AccountTypeFund    AccountType = "FUND"
	AccountTypeOption  AccountType = "OPTION"
)

// Balance represents a coin balance in the account
type Balance struct {
	Coin            string  `json:"coin"`
	WalletBalance   float64 `json:"walletBalance"`
	AvailableToTrade float64 `json:"availableToTrade"`
	AvailableToWithdraw float64 `json:"availableToWithdraw"`
	Locked          float64 `json:"locked"`
	Bonus           float64 `json:"bonus"`
}

// AccountInfo represents account information
type AccountInfo struct {
	TotalEquity      string    `json:"totalEquity"`
	AccountIMRate    string    `json:"accountIMRate"`
	TotalMarginBalance string  `json:"totalMarginBalance"`
	TotalInitialMargin string  `json:"totalInitialMargin"`
	AccountType      string    `json:"accountType"`
	TotalAvailableBalance string `json:"totalAvailableBalance"`
	TotalPerpUPL     string    `json:"totalPerpUPL"`
	TotalWalletBalance string  `json:"totalWalletBalance"`
	AccountMMRate    string    `json:"accountMMRate"`
	TotalMaintenanceMargin string `json:"totalMaintenanceMargin"`
	Coin             []Balance `json:"coin"`
}

// GetAccountBalance retrieves account balance information
func (c *Client) GetAccountBalance(ctx context.Context, accountType AccountType, coins ...string) (*AccountInfo, error) {
	params := map[string]interface{}{
		"accountType": string(accountType),
	}

	// Add specific coins if provided
	if len(coins) > 0 {
		coinStr := ""
		for i, coin := range coins {
			if i > 0 {
				coinStr += ","
			}
			coinStr += coin
		}
		params["coin"] = coinStr
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetAccountWallet(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	// Parse the response
	accountInfo, err := c.parseAccountBalanceResponse(result)
	if err != nil {
		return nil, fmt.Errorf("failed to parse account balance response: %w", err)
	}

	return accountInfo, nil
}

// GetCoinBalance retrieves balance for a specific coin
func (c *Client) GetCoinBalance(ctx context.Context, accountType AccountType, coin string) (*Balance, error) {
	accountInfo, err := c.GetAccountBalance(ctx, accountType, coin)
	if err != nil {
		return nil, err
	}

	// Find the specific coin balance
	for _, balance := range accountInfo.Coin {
		if balance.Coin == coin {
			return &balance, nil
		}
	}

	return nil, fmt.Errorf("coin %s not found in account", coin)
}

// GetTradableBalance returns the available balance for trading a specific coin
func (c *Client) GetTradableBalance(ctx context.Context, accountType AccountType, coin string) (float64, error) {
	balance, err := c.GetCoinBalance(ctx, accountType, coin)
	if err != nil {
		return 0, err
	}

	return balance.AvailableToTrade, nil
}

// GetAccountInfo retrieves general account information
func (c *Client) GetAccountInfo(ctx context.Context) (interface{}, error) {
	result, err := c.httpClient.NewUtaBybitServiceWithParams(map[string]interface{}{}).GetAccountInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get account info: %w", err)
	}

	return result, nil
}

// GetFeeRate retrieves trading fee rates for symbols
func (c *Client) GetFeeRate(ctx context.Context, category, symbol string) (interface{}, error) {
	params := map[string]interface{}{
		"category": category,
	}

	if symbol != "" {
		params["symbol"] = symbol
	}

	result, err := c.httpClient.NewUtaBybitServiceWithParams(params).GetFeeRates(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get fee rate: %w", err)
	}

	return result, nil
}

// parseAccountBalanceResponse parses the account balance API response
func (c *Client) parseAccountBalanceResponse(response interface{}) (*AccountInfo, error) {
	// Convert response to ServerResponse first
	serverResp, ok := response.(*bybit_api.ServerResponse)
	if !ok {
		return nil, fmt.Errorf("invalid response type")
	}

	if serverResp.RetCode != 0 {
		return nil, fmt.Errorf("API error: %s (code: %d)", serverResp.RetMsg, serverResp.RetCode)
	}

	// Parse the result as WalletBalanceResponse
	resultBytes, err := json.Marshal(serverResp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	var walletResult struct {
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
	}

	if err := json.Unmarshal(resultBytes, &walletResult); err != nil {
		return nil, fmt.Errorf("failed to unmarshal wallet result: %w", err)
	}

	if len(walletResult.List) == 0 {
		return nil, fmt.Errorf("no account data found")
	}

	account := walletResult.List[0]
	accountInfo := &AccountInfo{
		TotalEquity:            account.TotalEquity,
		AccountIMRate:          account.AccountIMRate,
		TotalMarginBalance:     account.TotalMarginBalance,
		TotalInitialMargin:     account.TotalInitialMargin,
		AccountType:            account.AccountType,
		TotalAvailableBalance:  account.TotalAvailableBalance,
		TotalPerpUPL:           account.TotalPerpUPL,
		TotalWalletBalance:     account.TotalWalletBalance,
		AccountMMRate:          account.AccountMMRate,
		TotalMaintenanceMargin: account.TotalMaintenanceMargin,
		Coin:                   make([]Balance, len(account.Coin)),
	}

	// Convert coin balances
	for i, coin := range account.Coin {
		accountInfo.Coin[i] = Balance{
			Coin:                coin.Coin,
			WalletBalance:       parseFloat64(coin.WalletBalance),
			AvailableToTrade:    parseFloat64(coin.AvailableToTrade),
			AvailableToWithdraw: parseFloat64(coin.AvailableToWithdraw),
			Locked:              parseFloat64(coin.TotalOrderIM) + parseFloat64(coin.TotalPositionIM),
			Bonus:               parseFloat64(coin.Bonus),
		}
	}

	return accountInfo, nil
}

// HasSufficientBalance checks if the account has sufficient balance for a trade
func (c *Client) HasSufficientBalance(ctx context.Context, accountType AccountType, coin string, requiredAmount float64) (bool, error) {
	balance, err := c.GetTradableBalance(ctx, accountType, coin)
	if err != nil {
		return false, err
	}

	return balance >= requiredAmount, nil
}

