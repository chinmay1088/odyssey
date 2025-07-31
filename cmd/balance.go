package cmd

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/chinmay1088/odyssey/api"
	"github.com/chinmay1088/odyssey/wallet"
	"github.com/spf13/cobra"
)

var balanceCmd = &cobra.Command{
	Use:   "balance [chain]",
	Short: "Check cryptocurrency balances",
	Long: `Check your cryptocurrency balances for supported chains.
	
Supported chains: eth, btc, sol
	
Examples:
  odyssey balance        # Check all balances
  odyssey balance eth    # Check Ethereum balance
  odyssey balance btc    # Check Bitcoin balance
  odyssey balance sol    # Check Solana balance`,
	Args: cobra.MaximumNArgs(1),
	RunE: runBalance,
}

func runBalance(cmd *cobra.Command, args []string) error {
	manager := wallet.NewManager()
	client := api.NewClient()

	// Check if wallet is unlocked
	if !manager.IsUnlocked() {
		return fmt.Errorf("wallet is locked. Run 'odyssey unlock' first")
	}

	usdFlag, _ := cmd.Flags().GetBool("usd")

	// Determine which chains to check
	var chains []string
	if len(args) == 0 {
		if manager.IsTestnet() {
			// Bitcoin not supported in testnet mode
			chains = []string{"eth", "sol"}
		} else {
			chains = []string{"eth", "btc", "sol"}
		}
	} else {
		chain := strings.ToLower(args[0])
		switch chain {
		case "eth", "ethereum":
			chains = []string{"eth"}
		case "btc", "bitcoin":
			if manager.IsTestnet() {
				return fmt.Errorf("bitcoin is not supported in testnet mode")
			}
			chains = []string{"btc"}
		case "sol", "solana":
			chains = []string{"sol"}
		default:
			return fmt.Errorf("unsupported chain: %s. Supported chains: eth, btc, sol", chain)
		}
	}

	fmt.Println("💰 Wallet Balances")

	// Display network information
	networkType := "Mainnet"
	if manager.IsTestnet() {
		networkType = "Testnet"
	}
	fmt.Printf("🌐 Network: %s\n", networkType)
	fmt.Println()

	for _, chain := range chains {
		switch chain {
		case "eth":
			if err := displayEthereumBalance(manager, client, usdFlag); err != nil {
				fmt.Printf("❌ Ethereum: Error - %v\n", err)
			}
		case "btc":
			if err := displayBitcoinBalance(manager, client, usdFlag); err != nil {
				fmt.Printf("❌ Bitcoin: Error - %v\n", err)
			}
		case "sol":
			if err := displaySolanaBalance(manager, client, usdFlag); err != nil {
				fmt.Printf("❌ Solana: Error - %v\n", err)
			}
		}
	}

	return nil
}

func displayEthereumBalance(manager *wallet.Manager, client *api.Client, usdFlag bool) error {
	address, err := manager.GetEthereumAddress()
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	balance, err := client.GetEthereumBalance(address.Hex())
	if err != nil {
		return fmt.Errorf("failed to fetch balance: %w", err)
	}

	if manager.IsTestnet() {
		fmt.Printf("🔷 Ethereum (Sepolia): %s\n", formatEthereumBalance(balance))
	} else {
		fmt.Printf("🔷 Ethereum: %s\n", formatEthereumBalance(balance))
	}

	if usdFlag && !manager.IsTestnet() {
		price, err := client.GetPrice("ethereum")
		if err != nil {
			fmt.Printf("   💵 USD: Error fetching price - %v\n", err)
		} else {
			ethValue := float64(balance.Uint64()) / 1e18
			usdValue := ethValue * price.USD.InexactFloat64()
			fmt.Printf("   💵 USD: $%.2f\n", usdValue)
		}
	}

	fmt.Printf("   📍 Address: %s\n", address.Hex())
	fmt.Println()
	return nil
}

func displayBitcoinBalance(manager *wallet.Manager, client *api.Client, usdFlag bool) error {
	// Bitcoin is only supported in mainnet
	if manager.IsTestnet() {
		return fmt.Errorf("bitcoin is not supported in testnet mode")
	}

	address, err := manager.GetBitcoinAddress()
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	balance, err := client.GetBitcoinBalance(address.String())
	if err != nil {
		return fmt.Errorf("failed to fetch balance: %w", err)
	}

	fmt.Printf("🟠 Bitcoin: %.8f BTC\n", balance)

	if usdFlag {
		price, err := client.GetPrice("bitcoin")
		if err != nil {
			fmt.Printf("   💵 USD: Error fetching price - %v\n", err)
		} else {
			usdValue := balance * price.USD.InexactFloat64()
			fmt.Printf("   💵 USD: $%.2f\n", usdValue)
		}
	}

	fmt.Printf("   📍 Address: %s\n", address.String())
	fmt.Println()
	return nil
}

func displaySolanaBalance(manager *wallet.Manager, client *api.Client, usdFlag bool) error {
	address, err := manager.GetSolanaAddress()
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	balance, err := client.GetSolanaBalance(address.String())
	if err != nil {
		return fmt.Errorf("failed to fetch balance: %w", err)
	}

	solBalance := float64(balance) / 1e9
	if manager.IsTestnet() {
		fmt.Printf("🟣 Solana (Devnet): %.9f SOL\n", solBalance)
	} else {
		fmt.Printf("🟣 Solana: %.9f SOL\n", solBalance)
	}

	// If balance is 0, this account likely doesn't exist on-chain yet
	if balance == 0 {
		fmt.Printf("   ℹ️ Note: This account doesn't exist on-chain yet. Send SOL to this address to activate it.\n")
	}

	if usdFlag && !manager.IsTestnet() {
		price, err := client.GetPrice("solana")
		if err != nil {
			fmt.Printf("   💵 USD: Error fetching price - %v\n", err)
		} else {
			usdValue := solBalance * price.USD.InexactFloat64()
			fmt.Printf("   💵 USD: $%.2f\n", usdValue)
		}
	}

	fmt.Printf("   📍 Address: %s\n", address.String())
	fmt.Println()
	return nil
}

func formatEthereumBalance(balance interface{}) string {
	// Convert different balance types to appropriate string representation
	switch b := balance.(type) {
	case *big.Int:
		// Convert Wei to Ether (1 ETH = 10^18 Wei)
		ether := new(big.Float).SetInt(b)
		ether.Quo(ether, big.NewFloat(1e18))

		// Format to 6 decimal places
		ethValue, _ := ether.Float64()
		return fmt.Sprintf("%.6f ETH", ethValue)
	case uint64:
		return fmt.Sprintf("%.6f ETH", float64(b)/1e18)
	default:
		return fmt.Sprintf("%v ETH", balance)
	}
}

func init() {
	balanceCmd.Flags().Bool("usd", false, "Show balances in USD")
}
