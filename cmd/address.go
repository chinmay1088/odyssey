package cmd

import (
	"fmt"
	"strings"

	"github.com/chinmay1088/odyssey/wallet"
	"github.com/spf13/cobra"
)

var addressCmd = &cobra.Command{
	Use:   "address [chain]",
	Short: "Show wallet address",
	Long: `Show your wallet address for the specified blockchain.
Supported chains: eth, btc, sol

Examples:
  odyssey address eth     # Show Ethereum address
  odyssey address btc     # Show Bitcoin address
  odyssey address sol     # Show Solana address
  odyssey address         # Show all addresses`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAddress,
}

func runAddress(cmd *cobra.Command, args []string) error {
	manager := wallet.NewManager()

	// Check if wallet is unlocked
	if !manager.IsUnlocked() {
		return fmt.Errorf("wallet is locked. Run 'odyssey unlock' first")
	}

	// If no chain specified, show all addresses
	if len(args) == 0 {
		return showAllAddresses(manager)
	}

	// Show specific chain address
	chain := strings.ToLower(args[0])
	return showChainAddress(manager, chain)
}

func showAllAddresses(manager *wallet.Manager) error {
	fmt.Println("🔑 Your wallet addresses:")

	// Display network information
	networkType := "Mainnet"
	if manager.IsTestnet() {
		networkType = "Testnet"
	}
	fmt.Printf("🌐 Network: %s\n", networkType)
	fmt.Println()

	// Ethereum address
	ethAddress, err := manager.GetEthereumAddress()
	if err != nil {
		return fmt.Errorf("failed to get Ethereum address: %w", err)
	}
	if manager.IsTestnet() {
		fmt.Printf("Ethereum (ETH - Sepolia): %s\n", ethAddress.Hex())
	} else {
		fmt.Printf("Ethereum (ETH): %s\n", ethAddress.Hex())
	}

	// Bitcoin address - only on mainnet
	if !manager.IsTestnet() {
		btcAddress, err := manager.GetBitcoinAddress()
		if err != nil {
			return fmt.Errorf("failed to get Bitcoin address: %w", err)
		}
		fmt.Printf("Bitcoin (BTC):  %s\n", btcAddress.String())
	} else {
		fmt.Println("Bitcoin (BTC):  Not supported in testnet mode")
	}

	// Solana address
	solAddress, err := manager.GetSolanaAddress()
	if err != nil {
		return fmt.Errorf("failed to get Solana address: %w", err)
	}
	if manager.IsTestnet() {
		fmt.Printf("Solana (SOL - Devnet): %s\n", solAddress.String())
		fmt.Println("   📝 Note: Solana addresses need to be initialized by receiving SOL first.")
		fmt.Println("   📝 The address is valid but shows as 'Account does not exist' until then.")
	} else {
		fmt.Printf("Solana (SOL): %s\n", solAddress.String())
	}

	return nil
}

func showChainAddress(manager *wallet.Manager, chain string) error {
	// Display network information
	networkType := "Mainnet"
	if manager.IsTestnet() {
		networkType = "Testnet"
	}
	fmt.Printf("🌐 Network: %s\n\n", networkType)

	switch chain {
	case "eth", "ethereum":
		address, err := manager.GetEthereumAddress()
		if err != nil {
			return fmt.Errorf("failed to get Ethereum address: %w", err)
		}
		if manager.IsTestnet() {
			fmt.Printf("Ethereum (ETH - Sepolia): %s\n", address.Hex())
		} else {
			fmt.Printf("Ethereum (ETH): %s\n", address.Hex())
		}

	case "btc", "bitcoin":
		if manager.IsTestnet() {
			fmt.Println("Bitcoin (BTC): Not supported in testnet mode")
		} else {
			address, err := manager.GetBitcoinAddress()
			if err != nil {
				return fmt.Errorf("failed to get Bitcoin address: %w", err)
			}
			fmt.Printf("Bitcoin (BTC): %s\n", address.String())
		}

	case "sol", "solana":
		address, err := manager.GetSolanaAddress()
		if err != nil {
			return fmt.Errorf("failed to get Solana address: %w", err)
		}
		if manager.IsTestnet() {
			fmt.Printf("Solana (SOL - Devnet): %s\n", address.String())
			fmt.Println("   📝 Note: Solana addresses need to be initialized by receiving SOL first.")
			fmt.Println("   📝 The address is valid but shows as 'Account does not exist' until then.")
		} else {
			fmt.Printf("Solana (SOL): %s\n", address.String())
		}

	default:
		return fmt.Errorf("unsupported chain: %s. Supported chains: eth, btc, sol", chain)
	}

	return nil
}
