package api

// network type constants
const (
	NetworkMainnet = "mainnet"
	NetworkTestnet = "testnet"
)

// RPC endpoints
const (
	// mainnet rpc's
	MainnetEthereumRPC = "https://ethereum-rpc.publicnode.com"
	MainnetSolanaRPC   = "https://api.mainnet-beta.solana.com"
	MainnetBitcoinRPC  = "https://blockchain.info"

	// testnet rpc's
	TestnetEthereumRPC = "https://ethereum-sepolia.publicnode.com"
	TestnetSolanaRPC   = "https://api.devnet.solana.com"
	// bitcoin is not supported for testnet
)
