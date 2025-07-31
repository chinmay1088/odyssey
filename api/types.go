package api

import (
	"time"

	"github.com/shopspring/decimal"
)

// Transaction represents a generic cryptocurrency transaction
type Transaction struct {
	Hash        string    `json:"hash"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	Amount      string    `json:"amount"`
	Fee         string    `json:"fee"`
	BlockNumber int64     `json:"block_number"`
	Timestamp   time.Time `json:"timestamp"`
	IsIncoming  bool      `json:"is_incoming"` // true for receiving, false for sending
}

// PriceData represents cryptocurrency price information
type PriceData struct {
	Symbol string          `json:"symbol"`
	Price  decimal.Decimal `json:"current_price"`
	USD    decimal.Decimal `json:"usd"`
}

// EthereumRPCResponse represents Ethereum RPC response
type EthereumRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// BitcoinUTXO represents a Bitcoin UTXO
type BitcoinUTXO struct {
	TxID   string  `json:"txid"`
	Vout   uint32  `json:"vout"`
	Value  float64 `json:"value"`
	Script string  `json:"scriptPubKey"`
}

// SolanaRPCResponse represents Solana RPC response
type SolanaRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}
