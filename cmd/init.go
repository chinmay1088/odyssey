package cmd

import (
	"fmt"
	"syscall"

	"github.com/chinmay1088/odyssey/wallet"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new wallet",
	Long: `Initialize a new Odyssey wallet with a secure recovery phrase.
	
This command will:
  - Generate a new 24-word recovery phrase
  - Create an encrypted vault
  - Set up your wallet for Ethereum, Bitcoin, and Solana`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	manager := wallet.NewManager()

	// Check if wallet already exists
	if manager.VaultExists() {
		return fmt.Errorf("wallet already exists. Remove ~/.odyssey/wallet.vault to create a new wallet")
	}

	fmt.Println("🚀 Initializing Odyssey Wallet")
	fmt.Println()

	// Get password from user
	fmt.Print("Enter a password for your wallet: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmPassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}
	fmt.Println()

	if string(password) != string(confirmPassword) {
		return fmt.Errorf("passwords do not match")
	}

	// Initialize wallet
	fmt.Println("Generating wallet...")
	err = manager.Initialize(string(password))
	if err != nil {
		return fmt.Errorf("failed to initialize wallet: %w", err)
	}

	// Get and display recovery phrase
	mnemonic, err := manager.GetMnemonic()
	if err != nil {
		return fmt.Errorf("failed to get recovery phrase: %w", err)
	}

	fmt.Println("✅ Wallet initialized successfully!")
	fmt.Println()
	fmt.Println("🔐 Recovery Phrase (24 words):")
	fmt.Println()
	fmt.Printf("   %s\n", mnemonic)
	fmt.Println()
	fmt.Println("⚠️  IMPORTANT:")
	fmt.Println("   - Write down this recovery phrase and store it securely")
	fmt.Println("   - Anyone with this phrase can access your funds")
	fmt.Println("   - Keep it offline and never share it with anyone")
	fmt.Println("   - This is the only way to recover your wallet")
	fmt.Println()
	fmt.Println("🔑 Next steps:")
	fmt.Println("   - Run 'odyssey unlock' to unlock your wallet")
	fmt.Println("   - Run 'odyssey address' to see your addresses")
	fmt.Println("   - Run 'odyssey balance' to check your balances")

	return nil
}