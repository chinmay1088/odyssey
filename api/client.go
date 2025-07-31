package api

// API Client-
//
// Files:
//   config.go    - RPC endpoints and network constants
//   types.go     - Struct definitions (transaction, priceData, etc.)
//   base.go      - Core client functionality (client struct, newClient, helpers)
//   ethereum.go  - Ethereum-specific functions (balance, transactions, gas, etc.)
//   bitcoin.go   - Bitcoin-specific functions (balance, utxos, transactions, etc.)
//   solana.go    - Solana-specific functions (balance, transactions, blockhash, etc.)
//
// Usage:
//   client := api.NewClient()  // from base.go
//   balance, err := client.GetEthereumBalance(address)  // from ethereum.go
//   utxos, err := client.GetBitcoinUTXOs(address)      // from bitcoin.go
//   txHash, err := client.SendSolanaTransaction(tx)    // from solana.go
