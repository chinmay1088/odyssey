package api

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// GetSolanaRPC returns the appropriate Solana RPC URL
func (c *Client) GetSolanaRPC() string {
	if c.IsTestnet() {
		return TestnetSolanaRPC
	}
	return MainnetSolanaRPC
}

// GetSolanaBalance fetches Solana balance
func (c *Client) GetSolanaBalance(address string) (uint64, error) {
	url := c.GetSolanaRPC()

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "getBalance",
		"params":  []interface{}{address},
		"id":      1,
	}

	response, err := c.postJSON(url, payload)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch Solana balance: %w", err)
	}

	var rpcResp SolanaRPCResponse
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		// For Solana, it's normal for accounts to not exist until they receive SOL
		if strings.Contains(rpcResp.Error.Message, "could not find account") {
			return 0, nil // Return 0 balance instead of error
		}
		return 0, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return 0, fmt.Errorf("no result in response")
	}

	// Handle Solana's response structure
	resultMap, ok := rpcResp.Result.(map[string]interface{})
	if !ok {
		// Try direct value format
		if value, ok := rpcResp.Result.(float64); ok {
			return uint64(value), nil
		}
		return 0, fmt.Errorf("invalid balance format")
	}

	// Check if the response has a "value" field (newer API format)
	if valueFloat, ok := resultMap["value"].(float64); ok {
		return uint64(valueFloat), nil
	}

	return 0, fmt.Errorf("could not find balance value in response")
}

// GetSolanaRecentBlockhash gets a recent blockhash for Solana transactions
func (c *Client) GetSolanaRecentBlockhash() (string, error) {
	url := c.GetSolanaRPC()

	fmt.Printf("🔍 Debug: Getting blockhash from: %s\n", url)

	// Use "finalized" commitment for the freshest blockhash that's already confirmed
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getLatestBlockhash",
		"params":  []interface{}{map[string]interface{}{"commitment": "finalized"}},
	}

	response, err := c.postJSON(url, payload)
	if err != nil {
		return "", fmt.Errorf("failed to get recent blockhash: %w", err)
	}

	fmt.Printf("🔍 Debug: Blockhash response: %s\n", string(response))

	var rpcResp SolanaRPCResponse
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return "", fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return "", fmt.Errorf("no result in response")
	}

	// Parse result as map
	resultMap, ok := rpcResp.Result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected result format")
	}

	// For the standard response format from getLatestBlockhash
	valueMap, ok := resultMap["value"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("missing 'value' in result")
	}

	blockhash, ok := valueMap["blockhash"].(string)
	if !ok {
		return "", fmt.Errorf("missing 'blockhash' in result")
	}

	fmt.Printf("🔍 Debug: Got blockhash: %s\n", blockhash)
	return blockhash, nil
}

// SendSolanaTransaction sends a Solana transaction
func (c *Client) SendSolanaTransaction(signedTx string) (string, error) {
	url := c.GetSolanaRPC()

	// Debug logging
	fmt.Printf("🔍 Debug: Sending to RPC: %s\n", url)
	fmt.Printf("🔍 Debug: Transaction length: %d chars\n", len(signedTx))

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "sendTransaction",
		"params":  []string{signedTx},
		"id":      1,
	}

	response, err := c.postJSON(url, payload)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	// Debug logging
	fmt.Printf("🔍 Debug: Raw response: %s\n", string(response))

	var rpcResp SolanaRPCResponse
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		fmt.Printf("🔍 Debug: RPC Error Code: %v\n", rpcResp.Error.Code)
		fmt.Printf("🔍 Debug: RPC Error Message: %s\n", rpcResp.Error.Message)
		return "", fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return "", fmt.Errorf("no result in response")
	}

	txHash, ok := rpcResp.Result.(string)
	if !ok {
		return "", fmt.Errorf("invalid transaction hash format")
	}

	return txHash, nil
}

// GetSolanaTransactions fetches transaction history for a Solana address
func (c *Client) GetSolanaTransactions(address string) ([]Transaction, error) {
	url := c.GetSolanaRPC()

	// First check if account exists
	balancePayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getBalance",
		"params":  []interface{}{address},
	}

	balanceResp, err := c.postJSON(url, balancePayload)
	if err == nil {
		var balanceResult SolanaRPCResponse
		if err := json.Unmarshal(balanceResp, &balanceResult); err == nil {
			// If we get an error like "could not find account", the account doesn't exist
			// This is normal for Solana - accounts don't exist until they receive SOL
			if balanceResult.Error != nil &&
				strings.Contains(balanceResult.Error.Message, "could not find account") {
				// Return empty list - no transactions for non-existent account
				return []Transaction{}, nil
			}
		}
	}

	// Get signature history first
	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getSignaturesForAddress",
		"params":  []interface{}{address, map[string]interface{}{"limit": 20}},
	}

	signaturesResp, err := c.postJSON(url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch signatures: %w", err)
	}

	var signaturesResult struct {
		Result []struct {
			Signature string `json:"signature"`
			Slot      int64  `json:"slot"`
			BlockTime int64  `json:"blockTime"`
		} `json:"result"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(signaturesResp, &signaturesResult); err != nil {
		return nil, fmt.Errorf("failed to parse signatures: %w", err)
	}

	// Check for specific error related to non-existent accounts
	if signaturesResult.Error != nil {
		// This error is normal for accounts that don't exist yet
		if strings.Contains(signaturesResult.Error.Message, "could not find account") {
			return []Transaction{}, nil
		}
		return nil, fmt.Errorf("RPC error: %s", signaturesResult.Error.Message)
	}

	if len(signaturesResult.Result) == 0 {
		// No transactions found
		return []Transaction{}, nil
	}

	// Now get transaction details for each signature
	transactions := make([]Transaction, 0, len(signaturesResult.Result))

	for _, sig := range signaturesResult.Result {
		// Get transaction details with parsed data
		txPayload := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "getTransaction",
			"params":  []interface{}{sig.Signature, map[string]interface{}{"encoding": "jsonParsed", "maxSupportedTransactionVersion": 0}},
		}

		txResp, err := c.postJSON(url, txPayload)
		if err != nil {
			continue // Skip this transaction if we can't fetch it
		}

		var txResult struct {
			Result struct {
				Meta struct {
					Fee          int64   `json:"fee"`
					PreBalances  []int64 `json:"preBalances"`
					PostBalances []int64 `json:"postBalances"`
				} `json:"meta"`
				Transaction struct {
					Message struct {
						AccountKeys []struct {
							Pubkey string `json:"pubkey"`
						} `json:"accountKeys"`
					} `json:"message"`
				} `json:"transaction"`
				BlockTime int64 `json:"blockTime"`
				Slot      int64 `json:"slot"`
			} `json:"result"`
		}

		if err := json.Unmarshal(txResp, &txResult); err != nil || txResult.Result.BlockTime == 0 {
			continue // Skip if we can't parse the transaction
		}

		if len(txResult.Result.Transaction.Message.AccountKeys) < 2 {
			continue // Skip transactions with insufficient accounts
		}

		// Try to detect balance change for the user's address
		addressIndex := -1
		for i, acc := range txResult.Result.Transaction.Message.AccountKeys {
			if acc.Pubkey == address {
				addressIndex = i
				break
			}
		}

		if addressIndex == -1 || addressIndex >= len(txResult.Result.Meta.PreBalances) {
			continue // Can't find address in accounts or insufficient balance data
		}

		// Calculate balance change for the address
		preBal := txResult.Result.Meta.PreBalances[addressIndex]
		postBal := txResult.Result.Meta.PostBalances[addressIndex]
		balChange := postBal - preBal

		// Determine direction and amount
		isIncoming := balChange > 0
		amount := math.Abs(float64(balChange)) / 1000000000.0 // Convert lamports to SOL

		// Fee is always paid by the first account
		fee := float64(txResult.Result.Meta.Fee) / 1000000000.0

		// Get from/to addresses (simplification - first two accounts)
		from := txResult.Result.Transaction.Message.AccountKeys[0].Pubkey
		to := ""
		if len(txResult.Result.Transaction.Message.AccountKeys) > 1 {
			to = txResult.Result.Transaction.Message.AccountKeys[1].Pubkey
		}

		transactions = append(transactions, Transaction{
			Hash:        sig.Signature,
			From:        from,
			To:          to,
			Amount:      fmt.Sprintf("%.9f SOL", amount),
			Fee:         fmt.Sprintf("%.9f SOL", fee),
			BlockNumber: sig.Slot,
			Timestamp:   time.Unix(sig.BlockTime, 0),
			IsIncoming:  isIncoming,
		})
	}

	return transactions, nil
}
