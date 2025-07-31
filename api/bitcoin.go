package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// GetBitcoinRPC returns the Bitcoin RPC URL (mainnet only)
func (c *Client) GetBitcoinRPC() string {
	return MainnetBitcoinRPC
}

// GetBitcoinBalance fetches Bitcoin balance
func (c *Client) GetBitcoinBalance(address string) (float64, error) {
	// Bitcoin only supported in mainnet
	if c.IsTestnet() {
		return 0, fmt.Errorf("bitcoin is not supported in testnet mode")
	}

	// Use blockchain.info API
	url := fmt.Sprintf("%s/balance?active=%s", c.GetBitcoinRPC(), address)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch balance: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	// Blockchain.info returns address as key in JSON object
	var result map[string]struct {
		FinalBalance int64 `json:"final_balance"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract address data
	addrData, exists := result[address]
	if !exists {
		return 0, fmt.Errorf("address data not found in response")
	}

	// Convert balance from satoshis to BTC
	return float64(addrData.FinalBalance) / 100000000.0, nil
}

// GetBitcoinUTXOs fetches Bitcoin UTXOs
func (c *Client) GetBitcoinUTXOs(address string) ([]BitcoinUTXO, error) {
	// Bitcoin only supported in mainnet
	if c.IsTestnet() {
		return nil, fmt.Errorf("bitcoin is not supported in testnet mode")
	}

	// Use Blockchair API
	url := fmt.Sprintf("https://api.blockchair.com/bitcoin/outputs?q=recipient(%s),is_spent(false)", address)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch UTXOs: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Data struct {
			Items []struct {
				TransactionHash string `json:"transaction_hash"`
				Index           uint32 `json:"index"`
				Value           string `json:"value"`
				ScriptHex       string `json:"script_hex"`
			} `json:"outputs"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to our UTXO format
	var utxos []BitcoinUTXO
	for _, item := range result.Data.Items {
		value, err := strconv.ParseFloat(item.Value, 64)
		if err != nil {
			continue // Skip this UTXO if value can't be parsed
		}

		utxo := BitcoinUTXO{
			TxID:   item.TransactionHash,
			Vout:   item.Index,
			Value:  value / 100000000.0, // Convert from satoshis to BTC
			Script: item.ScriptHex,
		}
		utxos = append(utxos, utxo)
	}

	return utxos, nil
}

// SendBitcoinTransaction sends a Bitcoin transaction
func (c *Client) SendBitcoinTransaction(signedTx string) (string, error) {
	// Bitcoin only supported in mainnet
	if c.IsTestnet() {
		return "", fmt.Errorf("bitcoin is not supported in testnet mode")
	}

	// Use mempool.space API
	url := "https://mempool.space/api/tx"

	resp, err := c.httpClient.Post(url, "text/plain", strings.NewReader(signedTx))
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("transaction failed: %s", string(body))
	}

	return string(body), nil
}

// GetBitcoinTransactions fetches transaction history for a Bitcoin address
func (c *Client) GetBitcoinTransactions(address string) ([]Transaction, error) {
	// Bitcoin only supported in mainnet
	if c.IsTestnet() {
		return nil, fmt.Errorf("bitcoin is not supported in testnet mode")
	}

	// Use Blockchain.info API
	url := fmt.Sprintf("https://blockchain.info/rawaddr/%s?limit=50", address)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Address      string `json:"address"`
		TxCount      int    `json:"n_tx"`
		Transactions []struct {
			Hash        string `json:"hash"`
			BlockHeight int64  `json:"block_height"`
			Time        int64  `json:"time"`
			Inputs      []struct {
				PrevOut struct {
					Addr  string `json:"addr"`
					Value int64  `json:"value"`
				} `json:"prev_out"`
			} `json:"inputs"`
			Out []struct {
				Addr  string `json:"addr"`
				Value int64  `json:"value"`
			} `json:"out"`
			Fee int64 `json:"fee"`
		} `json:"txs"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to generic transaction format
	transactions := make([]Transaction, 0, len(result.Transactions))
	for _, tx := range result.Transactions {
		// Determine if transaction is incoming or outgoing
		var from, to string
		var amount int64
		isIncoming := false

		// For simplicity, we'll just use the first input and output
		if len(tx.Inputs) > 0 && len(tx.Out) > 0 {
			from = tx.Inputs[0].PrevOut.Addr

			// Find our address in outputs
			for _, out := range tx.Out {
				if out.Addr == address {
					isIncoming = true
					amount = out.Value
					to = out.Addr
					break
				}
			}

			// If not found in outputs, check if we're the sender
			if !isIncoming {
				for _, inp := range tx.Inputs {
					if inp.PrevOut.Addr == address {
						// Find the recipient (first non-self output)
						for _, out := range tx.Out {
							if out.Addr != address {
								to = out.Addr
								amount = out.Value
								break
							}
						}
						break
					}
				}
			}
		}

		// Convert satoshis to BTC
		btcAmount := float64(amount) / 100000000.0
		btcFee := float64(tx.Fee) / 100000000.0

		transactions = append(transactions, Transaction{
			Hash:        tx.Hash,
			From:        from,
			To:          to,
			Amount:      fmt.Sprintf("%.8f BTC", btcAmount),
			Fee:         fmt.Sprintf("%.8f BTC", btcFee),
			BlockNumber: tx.BlockHeight,
			Timestamp:   time.Unix(tx.Time, 0),
			IsIncoming:  isIncoming,
		})
	}

	return transactions, nil
}

// GetBitcoinFeeEstimate returns the estimated fee rate for Bitcoin in satoshis/byte
func (c *Client) GetBitcoinFeeEstimate() (int64, error) {
	if c.IsTestnet() {
		return 0, fmt.Errorf("bitcoin is not supported in testnet mode")
	}

	// Try mempool.space API first
	url := "https://mempool.space/api/v1/fees/recommended"
	resp, err := c.httpClient.Get(url)
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err == nil {
			var feeResponse struct {
				FastestFee  int64 `json:"fastestFee"`
				HalfHourFee int64 `json:"halfHourFee"`
				HourFee     int64 `json:"hourFee"`
				EconomyFee  int64 `json:"economyFee"`
				MinimumFee  int64 `json:"minimumFee"`
			}

			if err := json.Unmarshal(body, &feeResponse); err == nil && feeResponse.HalfHourFee > 0 {
				// Use the half hour fee rate (average priority)
				return feeResponse.HalfHourFee, nil
			}
		}
	}

	// Fallback to blockchain.info
	url = "https://api.blockchain.info/mempool/fees"
	resp, err = c.httpClient.Get(url)
	if err != nil {
		return 10, nil // Default to 10 sat/byte if API fails
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 10, nil // Default to 10 sat/byte if reading fails
	}

	var feeResponse struct {
		Regular  int64 `json:"regular"`
		Priority int64 `json:"priority"`
	}

	if err := json.Unmarshal(body, &feeResponse); err != nil {
		return 10, nil // Default to 10 sat/byte if parsing fails
	}

	if feeResponse.Regular > 0 {
		return feeResponse.Regular, nil
	}

	// Default if both APIs fail or return 0
	return 10, nil
}
