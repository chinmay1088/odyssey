package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	version = "1.0.5"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "odyssey",
	Aliases: []string{"ody"},
	Short:   "A secure command-line cryptocurrency wallet",
	Long: `Odyssey is a secure, deterministic cryptocurrency wallet that supports
Ethereum, Bitcoin, and Solana. It provides local key generation, encrypted
storage, and offline transaction signing.

Features:
  • Multi-chain support (ETH, BTC, SOL)
  • BIP-39 mnemonic generation
  • BIP-44 hierarchical deterministic wallets
  • AES-256-GCM encrypted vault storage
  • Real-time balance checking
  • Transaction signing and broadcasting
  • Fiat conversion support
  • Recovery phrase management
  • Mainnet and Testnet support

Security:
  • All keys generated locally
  • Encrypted vault storage
  • Secure memory handling
  • No keys transmitted over network

Examples:
  odyssey init                    # Create new wallet
  odyssey unlock                  # Unlock wallet
  odyssey address                 # Show all addresses
  odyssey balance --usd           # Check balances with USD values
  odyssey pay eth 0.1 0x1234...  # Send 0.1 ETH
  odyssey network testnet        # Switch to testnet mode
  odyssey update                  # Update to latest version`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "suppress output")

	// Add subcommands
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(unlockCmd)
	rootCmd.AddCommand(addressCmd)
	rootCmd.AddCommand(balanceCmd)
	rootCmd.AddCommand(payCmd)
	rootCmd.AddCommand(transactionsCmd)
	rootCmd.AddCommand(recoveryPhraseCmd)
	rootCmd.AddCommand(buyCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(networkCmd) // Add network command
	rootCmd.AddCommand(exportCmd)  // Add export command
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Odyssey Wallet v%s\n", version)
	},
}
