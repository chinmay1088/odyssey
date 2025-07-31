package ethereum

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	// Ethereum chain IDs
	MainnetChainID = 1        // Ethereum mainnet
	SepoliaChainID = 11155111 // Sepolia testnet

	// Network types
	NetworkMainnet = "mainnet"
	NetworkTestnet = "testnet"
)

// Transaction represents an Ethereum transaction
type Transaction struct {
	Nonce    uint64          `json:"nonce"`
	GasPrice *big.Int        `json:"gasPrice"`
	GasLimit uint64          `json:"gasLimit"`
	To       *common.Address `json:"to"`
	Value    *big.Int        `json:"value"`
	Data     []byte          `json:"data"`
	ChainID  *big.Int        `json:"chainId"`
}

// getCurrentNetwork returns the current network (mainnet or testnet)
func getCurrentNetwork() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return NetworkMainnet // Default to mainnet on error
	}

	networkPath := filepath.Join(homeDir, ".odyssey", "network.txt")

	// Check if network file exists
	if _, err := os.Stat(networkPath); os.IsNotExist(err) {
		// File doesn't exist, default to mainnet
		return NetworkMainnet
	}

	// Read network from file
	data, err := os.ReadFile(networkPath)
	if err != nil {
		// Error reading file, default to mainnet
		return NetworkMainnet
	}

	network := strings.TrimSpace(string(data))

	// Validate network
	if network != NetworkMainnet && network != NetworkTestnet {
		// Invalid network, default to mainnet
		return NetworkMainnet
	}

	return network
}

// GetChainID returns the correct chain ID based on the current network
func GetChainID() *big.Int {
	if getCurrentNetwork() == NetworkTestnet {
		return big.NewInt(SepoliaChainID)
	}
	return big.NewInt(MainnetChainID)
}

// NewTransaction creates a new Ethereum transaction
func NewTransaction(nonce uint64, to common.Address, value *big.Int, gasLimit uint64, gasPrice *big.Int, data []byte) *Transaction {
	return &Transaction{
		Nonce:    nonce,
		GasPrice: gasPrice,
		GasLimit: gasLimit,
		To:       &to,
		Value:    value,
		Data:     data,
		ChainID:  GetChainID(), // Dynamically get chain ID based on network
	}
}

// SignTransaction signs an Ethereum transaction with the provided private key
func SignTransaction(tx *Transaction, privateKey *ecdsa.PrivateKey) (string, error) {
	// Create the transaction
	ethereumTx := types.NewTransaction(
		tx.Nonce,
		*tx.To,
		tx.Value,
		tx.GasLimit,
		tx.GasPrice,
		tx.Data,
	)

	// Sign the transaction
	signedTx, err := types.SignTx(ethereumTx, types.NewEIP155Signer(tx.ChainID), privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Serialize to hex
	serialized, err := rlp.EncodeToBytes(signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to serialize transaction: %w", err)
	}

	return hexutil.Encode(serialized), nil
}

// ParseAddress parses an Ethereum address
func ParseAddress(address string) (common.Address, error) {
	if !common.IsHexAddress(address) {
		return common.Address{}, fmt.Errorf("invalid Ethereum address: %s", address)
	}
	return common.HexToAddress(address), nil
}

// WeiToEther converts wei to ether
func WeiToEther(wei *big.Int) float64 {
	ether := new(big.Float).SetInt(wei)
	ether.Quo(ether, big.NewFloat(1e18))
	etherFloat, _ := ether.Float64()
	return etherFloat
}

// EtherToWei converts ether to wei
func EtherToWei(ether *big.Float) *big.Int {
	wei := new(big.Float).Mul(ether, big.NewFloat(1e18))
	weiInt := new(big.Int)
	wei.Int(weiInt)
	return weiInt
}

// FormatBalance formats balance in a human-readable format
func FormatBalance(balance *big.Int) string {
	etherValue := WeiToEther(balance)
	return fmt.Sprintf("%.18f ETH", etherValue)
}

// EstimateGasLimit estimates gas limit for a transaction
func EstimateGasLimit(data []byte) uint64 {
	// Base gas limit for simple transfers
	baseGas := uint64(21000)

	// Additional gas for data
	if len(data) > 0 {
		baseGas += uint64(len(data)) * 16 // 16 gas per byte
	}

	return baseGas
}

// ValidateTransaction validates transaction parameters
func ValidateTransaction(tx *Transaction) error {
	if tx.To == nil {
		return fmt.Errorf("transaction must have a recipient address")
	}
	if tx.Value == nil || tx.Value.Sign() < 0 {
		return fmt.Errorf("transaction value must be non-negative")
	}
	if tx.GasPrice == nil || tx.GasPrice.Sign() <= 0 {
		return fmt.Errorf("gas price must be positive")
	}
	if tx.GasLimit == 0 {
		return fmt.Errorf("gas limit must be greater than 0")
	}
	return nil
}
