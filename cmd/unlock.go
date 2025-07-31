package cmd

import (
	"fmt"
	"syscall"

	"github.com/chinmay1088/odyssey/wallet"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var unlockCmd = &cobra.Command{
	Use:   "unlock",
	Short: "Unlock wallet for session",
	Long: `Unlock your Odyssey wallet for the current session.
This command will decrypt your vault and load your keys into memory.
The wallet will remain unlocked until you close the terminal or run 'odyssey lock'.

Example:
  odyssey unlock`,
	RunE: runUnlock,
}

func runUnlock(cmd *cobra.Command, args []string) error {
	manager := wallet.NewManager()

	// Check if wallet exists
	if !manager.VaultExists() {
		return fmt.Errorf("no wallet found. Run 'odyssey init' to create a new wallet")
	}

	// Check if already unlocked
	if manager.IsUnlocked() {
		fmt.Println("✅ Wallet is already unlocked")
		return nil
	}

	// Get password from user
	fmt.Print("Enter your wallet password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println() // New line after password input

	// Unlock wallet
	fmt.Println("Unlocking wallet...")
	err = manager.Unlock(string(password))
	if err != nil {
		return fmt.Errorf("failed to unlock wallet: %w", err)
	}

	fmt.Println("✅ Wallet unlocked successfully!")
	fmt.Println("💡 Use 'odyssey address [chain]' to see your addresses")
	fmt.Println("💡 Use 'odyssey balance [chain]' to check your balances")

	return nil
}