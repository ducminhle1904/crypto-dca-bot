package bybit

import (
	bybit_api "github.com/bybit-exchange/bybit.go.api"
)

// Client wraps the Bybit API client with additional functionality
type Client struct {
	httpClient *bybit_api.Client
	apiKey     string
	apiSecret  string
	testnet    bool
	demo       bool
}

// Config holds the configuration for the Bybit client
type Config struct {
	APIKey    string
	APISecret string
	Testnet   bool
	Demo      bool // Demo trading environment
}

// NewClient creates a new Bybit client
func NewClient(config Config) *Client {
	var baseURL string
	if config.Demo {
		// Demo trading environment (paper trading)
		baseURL = "https://api-demo.bybit.com"
	} else if config.Testnet {
		baseURL = bybit_api.TESTNET
	} else {
		baseURL = bybit_api.MAINNET
	}

	// Create client with extended recv_window to handle timestamp sync issues
	httpClient := bybit_api.NewBybitHttpClient(
		config.APIKey,
		config.APISecret,
		bybit_api.WithBaseURL(baseURL),
	)

	return &Client{
		httpClient: httpClient,
		apiKey:     config.APIKey,
		apiSecret:  config.APISecret,
		testnet:    config.Testnet,
		demo:       config.Demo,
	}
}

// IsTestnet returns whether the client is configured for testnet
func (c *Client) IsTestnet() bool {
	return c.testnet
}

// IsDemo returns whether the client is configured for demo trading
func (c *Client) IsDemo() bool {
	return c.demo
}

// GetEnvironment returns a string describing the current environment
func (c *Client) GetEnvironment() string {
	if c.demo {
		return "demo"
	} else if c.testnet {
		return "testnet"
	} else {
		return "mainnet"
	}
}


