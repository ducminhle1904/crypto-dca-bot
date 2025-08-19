package bybit

import (
	"context"
	"fmt"

	bybit_api "github.com/bybit-exchange/bybit.go.api"
)

// Client wraps the Bybit API client with additional functionality
type Client struct {
	httpClient        *bybit_api.Client
	apiKey            string
	apiSecret         string
	testnet           bool
	demo              bool
	instrumentManager *InstrumentManager
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

	client := &Client{
		httpClient: httpClient,
		apiKey:     config.APIKey,
		apiSecret:  config.APISecret,
		testnet:    config.Testnet,
		demo:       config.Demo,
	}
	
	// Initialize instrument manager
	client.instrumentManager = NewInstrumentManager(client)
	
	return client
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

// GetInstrumentManager returns the instrument manager
func (c *Client) GetInstrumentManager() *InstrumentManager {
	return c.instrumentManager
}

// PreloadInstruments preloads instrument information for commonly traded symbols
func (c *Client) PreloadInstruments(ctx context.Context, symbols []string) error {
	if c.instrumentManager == nil {
		return fmt.Errorf("instrument manager not initialized")
	}
	
	for _, symbol := range symbols {
		// Try to get instrument info for each symbol
		_, err := c.instrumentManager.GetInstrumentInfo(ctx, "linear", symbol)
		if err != nil {
			// Log error but continue with other symbols
			fmt.Printf("Warning: Failed to preload instrument info for %s: %v\n", symbol, err)
		}
	}
	
	return nil
}


