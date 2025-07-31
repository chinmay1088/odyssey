package api

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

// GetEthereumRPC returns the appropriate Ethereum RPC URL
func (c *Client) GetEthereumRPC() string {
	if c.IsTestnet() {
		return TestnetEthereumRPC
	}
	return MainnetEthereumRPC
}

// GetEthereumBalance fetches Ethereum balance
func (c *Client) GetEthereumBalance(address string) (*big.Int, error) {
	// Use network-specific Ethereum RPC
	url := c.GetEthereumRPC()

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getBalance",
		"params":  []string{address, "latest"},
		"id":      1,
	}

	response, err := c.postJSON(url, payload)
	if err != nil {
		return nil, err
	}

	var rpcResp EthereumRPCResponse
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}

	balanceStr, ok := rpcResp.Result.(string)
	if !ok {
		return nil, fmt.Errorf("invalid balance format")
	}

	balance := new(big.Int)
	balance.SetString(strings.TrimPrefix(balanceStr, "0x"), 16)
	return balance, nil
}

// GetEthereumNonce fetches Ethereum nonce
func (c *Client) GetEthereumNonce(address string) (uint64, error) {
	url := c.GetEthereumRPC()

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_getTransactionCount",
		"params":  []string{address, "latest"},
		"id":      1,
	}

	response, err := c.postJSON(url, payload)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch nonce: %w", err)
	}

	var rpcResp EthereumRPCResponse
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return 0, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return 0, fmt.Errorf("no result in response")
	}

	nonceStr, ok := rpcResp.Result.(string)
	if !ok {
		return 0, fmt.Errorf("invalid nonce format")
	}

	nonce := new(big.Int)
	nonce.SetString(strings.TrimPrefix(nonceStr, "0x"), 16)
	return nonce.Uint64(), nil
}

// GetEthereumGasPrice fetches current gas price
func (c *Client) GetEthereumGasPrice() (*big.Int, error) {
	url := c.GetEthereumRPC()

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_gasPrice",
		"params":  []interface{}{},
		"id":      1,
	}

	response, err := c.postJSON(url, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch gas price: %w", err)
	}

	var rpcResp EthereumRPCResponse
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, fmt.Errorf("RPC error: %s", rpcResp.Error.Message)
	}

	if rpcResp.Result == nil {
		return nil, fmt.Errorf("no result in response")
	}

	gasPriceStr, ok := rpcResp.Result.(string)
	if !ok {
		return nil, fmt.Errorf("invalid gas price format")
	}

	gasPrice := new(big.Int)
	gasPrice.SetString(strings.TrimPrefix(gasPriceStr, "0x"), 16)
	return gasPrice, nil
}

// SendEthereumTransaction sends an Ethereum transaction
func (c *Client) SendEthereumTransaction(signedTx string) (string, error) {
	url := c.GetEthereumRPC()

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_sendRawTransaction",
		"params":  []string{signedTx},
		"id":      1,
	}

	response, err := c.postJSON(url, payload)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	var rpcResp EthereumRPCResponse
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if rpcResp.Error != nil {
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

// GetEthereumTransactions fetches transaction history for an Ethereum address
func (c *Client) GetEthereumTransactions(address string) ([]Transaction, error) {
	url := c.GetEthereumRPC()

	// For testnets, we'll use a more direct approach instead of logs filtering
	// since many test networks don't have great log support
	if c.IsTestnet() {
		return c.getEthereumTransactionsDirect(address)
	}

	// Get block number for filtering (last 10000 blocks)
	blockPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
	}

	blockResp, err := c.postJSON(url, blockPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block number: %w", err)
	}

	var blockResult struct {
		Result string `json:"result"`
	}

	if err := json.Unmarshal(blockResp, &blockResult); err != nil {
		return nil, fmt.Errorf("failed to parse block number: %w", err)
	}

	// Parse current block number
	currentBlockHex := blockResult.Result
	currentBlock, err := parseHexInt(currentBlockHex)
	if err != nil {
		return nil, fmt.Errorf("invalid block number: %w", err)
	}

	// Create filter for transactions
	fromBlock := fmt.Sprintf("0x%x", currentBlock-10000) // Last ~10000 blocks
	filterPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_getLogs",
		"params": []interface{}{map[string]interface{}{
			"fromBlock": fromBlock,
			"toBlock":   "latest",
			"address":   []string{address},
		}},
	}

	filterResp, err := c.postJSON(url, filterPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch logs: %w", err)
	}

	var logsResult struct {
		Result []struct {
			TransactionHash string `json:"transactionHash"`
			BlockNumber     string `json:"blockNumber"`
		} `json:"result"`
	}

	if err := json.Unmarshal(filterResp, &logsResult); err != nil {
		return nil, fmt.Errorf("failed to parse logs: %w", err)
	}

	// Get unique transaction hashes
	txHashes := make(map[string]bool)
	for _, log := range logsResult.Result {
		txHashes[log.TransactionHash] = true
	}

	// Get transaction details for each hash
	var transactions []Transaction
	for txHash := range txHashes {
		txPayload := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "eth_getTransactionByHash",
			"params":  []interface{}{txHash},
		}

		txResp, err := c.postJSON(url, txPayload)
		if err != nil {
			continue // Skip this transaction
		}

		var txResult struct {
			Result struct {
				Hash        string `json:"hash"`
				From        string `json:"from"`
				To          string `json:"to"`
				Value       string `json:"value"`
				GasPrice    string `json:"gasPrice"`
				Gas         string `json:"gas"`
				BlockNumber string `json:"blockNumber"`
			} `json:"result"`
		}

		if err := json.Unmarshal(txResp, &txResult); err != nil {
			continue // Skip this transaction
		}

		if txResult.Result.Hash == "" {
			continue // Skip empty results
		}

		// Get block info for timestamp
		blockPayload := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "eth_getBlockByNumber",
			"params":  []interface{}{txResult.Result.BlockNumber, false},
		}

		blockResp, err := c.postJSON(url, blockPayload)
		if err != nil {
			continue // Skip this transaction
		}

		var blockInfo struct {
			Result struct {
				Timestamp string `json:"timestamp"`
			} `json:"result"`
		}

		if err := json.Unmarshal(blockResp, &blockInfo); err != nil {
			continue // Skip this transaction
		}

		// Parse values
		value, _ := parseHexBigInt(txResult.Result.Value)
		gasPrice, _ := parseHexBigInt(txResult.Result.GasPrice)
		gas, _ := parseHexInt(txResult.Result.Gas)
		blockNumber, _ := parseHexInt(txResult.Result.BlockNumber)
		timestamp, _ := parseHexInt(blockInfo.Result.Timestamp)

		// Calculate fee (gas * gasPrice)
		gasBigInt := big.NewInt(int64(gas))
		fee := new(big.Int).Mul(gasBigInt, gasPrice)

		// Convert values
		valueEth := weiToEth(value)
		feeEth := weiToEth(fee)

		// Determine if incoming or outgoing
		isIncoming := strings.EqualFold(txResult.Result.To, address)

		tx := Transaction{
			Hash:        txResult.Result.Hash,
			From:        txResult.Result.From,
			To:          txResult.Result.To,
			Amount:      fmt.Sprintf("%.6f ETH", valueEth),
			Fee:         fmt.Sprintf("%.6f ETH", feeEth),
			BlockNumber: int64(blockNumber),
			Timestamp:   time.Unix(int64(timestamp), 0),
			IsIncoming:  isIncoming,
		}

		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// getEthereumTransactionsDirect gets transactions using a simpler approach for testnets
// that works better with Sepolia and other test networks
func (c *Client) getEthereumTransactionsDirect(address string) ([]Transaction, error) {
	url := c.GetEthereumRPC()

	// Get the latest block number
	blockPayload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_blockNumber",
		"params":  []interface{}{},
	}

	blockResp, err := c.postJSON(url, blockPayload)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch block number: %w", err)
	}

	var blockResult struct {
		Result string `json:"result"`
	}

	if err := json.Unmarshal(blockResp, &blockResult); err != nil {
		return nil, fmt.Errorf("failed to parse block number: %w", err)
	}

	// Parse current block number
	currentBlockHex := blockResult.Result
	currentBlock, err := parseHexInt(currentBlockHex)
	if err != nil {
		return nil, fmt.Errorf("invalid block number: %w", err)
	}

	var transactions []Transaction

	// Check for transactions in last 50 blocks (limit our search for performance)
	blocksToCheck := uint64(50)
	if currentBlock < blocksToCheck {
		blocksToCheck = currentBlock
	}

	// Map to keep track of processed transactions to avoid duplicates
	processedTxs := make(map[string]bool)

	// We'll check each block for transactions to/from the address
	for i := uint64(0); i < blocksToCheck; i++ {
		blockNumber := currentBlock - i
		blockNumberHex := fmt.Sprintf("0x%x", blockNumber)

		// Get block with transactions
		blockWithTxsPayload := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "eth_getBlockByNumber",
			"params":  []interface{}{blockNumberHex, true},
		}

		blockWithTxsResp, err := c.postJSON(url, blockWithTxsPayload)
		if err != nil {
			continue // Skip this block if we can't fetch it
		}

		var blockWithTxs struct {
			Result struct {
				Transactions []struct {
					Hash     string `json:"hash"`
					From     string `json:"from"`
					To       string `json:"to"`
					Value    string `json:"value"`
					Gas      string `json:"gas"`
					GasPrice string `json:"gasPrice"`
				} `json:"transactions"`
				Timestamp string `json:"timestamp"`
			} `json:"result"`
		}

		if err := json.Unmarshal(blockWithTxsResp, &blockWithTxs); err != nil {
			continue // Skip this block if we can't parse it
		}

		// Skip if block has no transactions
		if blockWithTxs.Result.Transactions == nil {
			continue
		}

		// Get timestamp
		timestamp, _ := parseHexInt(blockWithTxs.Result.Timestamp)

		// Loop through transactions in the block
		for _, tx := range blockWithTxs.Result.Transactions {
			// Check if transaction involves our address
			addrLower := strings.ToLower(address)
			fromLower := strings.ToLower(tx.From)
			toLower := strings.ToLower(tx.To)

			// Skip if transaction doesn't involve our address
			if fromLower != addrLower && toLower != addrLower {
				continue
			}

			// Skip if we've already processed this transaction
			if processedTxs[tx.Hash] {
				continue
			}

			processedTxs[tx.Hash] = true

			// Parse values
			value, _ := parseHexBigInt(tx.Value)
			gasPrice, _ := parseHexBigInt(tx.GasPrice)
			gas, _ := parseHexInt(tx.Gas)

			// Calculate fee (gas * gasPrice)
			gasBigInt := big.NewInt(int64(gas))
			fee := new(big.Int).Mul(gasBigInt, gasPrice)

			// Convert values
			valueEth := weiToEth(value)
			feeEth := weiToEth(fee)

			// Determine if incoming or outgoing
			isIncoming := strings.EqualFold(tx.To, address)

			transaction := Transaction{
				Hash:        tx.Hash,
				From:        tx.From,
				To:          tx.To,
				Amount:      fmt.Sprintf("%.6f ETH", valueEth),
				Fee:         fmt.Sprintf("%.6f ETH", feeEth),
				BlockNumber: int64(blockNumber),
				Timestamp:   time.Unix(int64(timestamp), 0),
				IsIncoming:  isIncoming,
			}

			transactions = append(transactions, transaction)
		}
	}

	return transactions, nil
}

// GetEthereumGasEstimate estimates the gas needed for an ETH transaction
func (c *Client) GetEthereumGasEstimate(from string, to string, value *big.Int, data []byte) (uint64, error) {
	url := c.GetEthereumRPC()

	// Prepare transaction object for gas estimation
	txObject := map[string]interface{}{
		"from": from,
		"to":   to,
	}

	// Add value if present
	if value != nil && value.Cmp(big.NewInt(0)) > 0 {
		txObject["value"] = "0x" + value.Text(16)
	}

	// Add data if present
	if len(data) > 0 {
		txObject["data"] = "0x" + fmt.Sprintf("%x", data)
	}

	payload := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "eth_estimateGas",
		"params":  []interface{}{txObject},
	}

	response, err := c.postJSON(url, payload)
	if err != nil {
		// If estimation fails, use a conservative default
		return 50000, nil
	}

	var rpcResp EthereumRPCResponse
	if err := json.Unmarshal(response, &rpcResp); err != nil {
		// If parsing fails, use a conservative default
		return 50000, nil
	}

	if rpcResp.Error != nil {
		// If RPC returns error, use a conservative default
		return 50000, nil
	}

	resultStr, ok := rpcResp.Result.(string)
	if !ok {
		return 50000, fmt.Errorf("unexpected gas estimate result format")
	}

	gas, err := parseHexInt(resultStr)
	if err != nil {
		return 50000, nil // Default if parsing fails
	}

	// Add 20% buffer to account for potential variations
	gas = gas + (gas / 5)

	return gas, nil
}

// Helper to convert hex string to int
func parseHexInt(hexStr string) (uint64, error) {
	// Remove '0x' prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")
	return strconv.ParseUint(hexStr, 16, 64)
}

// Helper to convert hex string to big.Int
func parseHexBigInt(hexStr string) (*big.Int, error) {
	// Remove '0x' prefix if present
	hexStr = strings.TrimPrefix(hexStr, "0x")

	value := new(big.Int)
	_, success := value.SetString(hexStr, 16)
	if !success {
		return nil, fmt.Errorf("invalid hex value: %s", hexStr)
	}

	return value, nil
}
