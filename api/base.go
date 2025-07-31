package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// Client handles API calls to external services
type Client struct {
	httpClient *http.Client
	network    string
}

// NewClient creates a new API client
func NewClient() *Client {
	// Determine the current network
	network := NetworkMainnet // Default to mainnet

	homeDir, err := os.UserHomeDir()
	if err == nil {
		networkPath := filepath.Join(homeDir, ".odyssey", "network.txt")

		// Read network file if it exists
		if _, err := os.Stat(networkPath); err == nil {
			data, err := os.ReadFile(networkPath)
			if err == nil {
				network = strings.TrimSpace(string(data))
				// Validate network
				if network != NetworkMainnet && network != NetworkTestnet {
					network = NetworkMainnet // Default to mainnet if invalid
				}
			}
		}
	}

	return &Client{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		network: network,
	}
}

// IsTestnet returns true if the client is using testnet
func (c *Client) IsTestnet() bool {
	return c.network == NetworkTestnet
}

// GetPrice fetches current price for a cryptocurrency
func (c *Client) GetPrice(symbol string) (*PriceData, error) {
	// Use CoinGecko API
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd", symbol)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch price: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]map[string]float64
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if priceData, exists := result[symbol]; exists {
		if usdPrice, exists := priceData["usd"]; exists {
			return &PriceData{
				Symbol: symbol,
				Price:  decimal.NewFromFloat(usdPrice),
				USD:    decimal.NewFromFloat(usdPrice),
			}, nil
		}
	}

	return nil, fmt.Errorf("price not found for symbol: %s", symbol)
}



// Helper to convert Wei to Ether
func weiToEth(wei *big.Int) float64 {
	if wei == nil {
		return 0
	}

	// Convert wei to ether (1 ETH = 10^18 Wei)
	ether := new(big.Float).SetInt(wei)
	ether.Quo(ether, big.NewFloat(1e18))

	// Convert to float64 for display
	result, _ := ether.Float64()
	return result
}

// postJSON sends a POST request with JSON payload
func (c *Client) postJSON(url string, payload interface{}) ([]byte, error) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	resp, err := c.httpClient.Post(url, "application/json", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	return body, nil
}
