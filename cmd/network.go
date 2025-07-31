package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Network type constants
const (
	// Network types
	NetworkMainnet = "mainnet"
	NetworkTestnet = "testnet"

	// Mainnet RPC endpoints
	MainnetEthereumRPC = "https://ethereum-rpc.publicnode.com"
	MainnetSolanaRPC   = "https://api.mainnet-beta.solana.com"
	MainnetBitcoinRPC  = "https://blockchain.info"

	// Testnet RPC endpoints
	TestnetEthereumRPC = "https://ethereum-sepolia.publicnode.com"
	TestnetSolanaRPC   = "https://api.devnet.solana.com"
	TestnetBitcoinRPC  = "NOT_SUPPORTED" // Bitcoin testnet is not supported
)

var networkCmd = &cobra.Command{
	Use:   "network [mainnet|testnet]",
	Short: "Show or change network",
	Long: `Show the current network or switch between mainnet and testnet.
	
Only Ethereum (Sepolia) and Solana devnet are supported.
Bitcoin is only supported on mainnet.
	
Examples:
  odyssey network            # Show current network
  odyssey network mainnet    # Switch to mainnet
  odyssey network testnet    # Switch to testnet`,
	Args: cobra.MaximumNArgs(1),
	RunE: runNetwork,
}

func runNetwork(cmd *cobra.Command, args []string) error {
	// If no arguments provided, show current network
	if len(args) == 0 {
		return showCurrentNetwork()
	}

	network := strings.ToLower(args[0])

	// Validate network argument
	if network != NetworkMainnet && network != NetworkTestnet {
		return fmt.Errorf("invalid network: %s. Use 'mainnet' or 'testnet'", network)
	}

	// Set the network
	return setNetwork(network)
}

func showCurrentNetwork() error {
	network, err := getCurrentNetwork()
	if err != nil {
		return err
	}

	if network == NetworkMainnet {
		fmt.Printf("🌐 Current network: %s\n", color.GreenString("Mainnet"))
		fmt.Println()
		fmt.Println("Network details:")
		fmt.Println("   - Ethereum: Mainnet")
		fmt.Println("   - Bitcoin: Mainnet")
		fmt.Println("   - Solana: Mainnet")
		fmt.Println("💡 Odyssey uses different wallets per network for your safety")
		fmt.Println("🔐 Your mainnet, devnet, and testnet addresses are all separate")
	} else {
		fmt.Printf("🌐 Current network: %s\n", color.YellowString("Testnet"))
		fmt.Println()
		fmt.Println("Network details:")
		fmt.Println("   - Ethereum: Sepolia")
		fmt.Printf("   - Bitcoin: %s\n", color.RedString("Not supported"))
		fmt.Println("   - Solana: Devnet")
		fmt.Println()
		fmt.Println("⚠️  Warning: Bitcoin is not supported in testnet mode")
		fmt.Println("⚠️  Warning: Buy command is disabled in testnet mode")
		fmt.Println("💡 Odyssey uses different wallets per network for your safety")
		fmt.Println("🔐 Your mainnet, devnet, and testnet addresses are all separate")
	}

	return nil
}

func setNetwork(network string) error {
	// Get home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create .odyssey directory if it doesn't exist
	configDir := filepath.Join(homeDir, ".odyssey")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write network to network.txt file
	networkPath := filepath.Join(configDir, "network.txt")
	if err := os.WriteFile(networkPath, []byte(network), 0600); err != nil {
		return fmt.Errorf("failed to write network file: %w", err)
	}

	fmt.Printf("🌐 Switched to %s network\n", strings.ToUpper(network))

	if network == NetworkTestnet {
		fmt.Println()
		fmt.Println("⚠️  You are now on TESTNET mode")
		fmt.Println("   - Ethereum: Sepolia Testnet")
		fmt.Println("   - Solana: Devnet")
		fmt.Println("   - Bitcoin: Not supported in testnet mode")
		fmt.Println()
		fmt.Println("   Buy command is disabled in testnet mode")
		fmt.Println("💡 Odyssey uses different wallets per network for your safety")
		fmt.Println("🔐 Your mainnet, devnet, and testnet addresses are all separate")
	} else {
		fmt.Println()
		fmt.Println("✅ You are now on MAINNET mode")
		fmt.Println("   All features are available in mainnet mode")
		fmt.Println("💡 Odyssey uses different wallets per network for your safety")
		fmt.Println("🔐 Your mainnet, devnet, and testnet addresses are all separate")
	}

	return nil
}

// GetCurrentNetwork returns the current network (mainnet or testnet)
func getCurrentNetwork() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return NetworkMainnet, nil // Default to mainnet on error
	}

	networkPath := filepath.Join(homeDir, ".odyssey", "network.txt")

	// Check if network file exists
	if _, err := os.Stat(networkPath); os.IsNotExist(err) {
		// File doesn't exist, default to mainnet
		return NetworkMainnet, nil
	}

	// Read network from file
	data, err := os.ReadFile(networkPath)
	if err != nil {
		// Error reading file, default to mainnet
		return NetworkMainnet, nil
	}

	network := strings.TrimSpace(string(data))

	// Validate network
	if network != NetworkMainnet && network != NetworkTestnet {
		// Invalid network, default to mainnet
		return NetworkMainnet, nil
	}

	return network, nil
}

// IsTestnetActive returns true if the current network is testnet
func IsTestnetActive() bool {
	network, _ := getCurrentNetwork()
	return network == NetworkTestnet
}

// GetEthereumRPC returns the Ethereum RPC URL for the current network
func GetEthereumRPC() string {
	if IsTestnetActive() {
		return TestnetEthereumRPC
	}
	return MainnetEthereumRPC
}

// GetSolanaRPC returns the Solana RPC URL for the current network
func GetSolanaRPC() string {
	if IsTestnetActive() {
		return TestnetSolanaRPC
	}
	return MainnetSolanaRPC
}

// GetBitcoinRPC returns the Bitcoin RPC URL
// Note: Only mainnet is supported for Bitcoin
func GetBitcoinRPC() string {
	return MainnetBitcoinRPC
}

func init() {
	rootCmd.AddCommand(networkCmd)
}
