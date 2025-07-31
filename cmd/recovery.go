package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/chinmay1088/odyssey/wallet"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var recoveryPhraseCmd = &cobra.Command{
	Use:   "recovery-phrase [show|import]",
	Short: "Manage recovery phrase",
	Long: `Manage your wallet's recovery phrase (mnemonic).
	
Commands:
  show    - Display the recovery phrase (requires password)
  import  - Import wallet from existing recovery phrase`,
	Args: cobra.ExactArgs(1),
	RunE: runRecoveryPhrase,
}

func runRecoveryPhrase(cmd *cobra.Command, args []string) error {
	manager := wallet.NewManager()
	action := strings.ToLower(args[0])

	switch action {
	case "show":
		return showRecoveryPhrase(manager)
	case "import":
		return importRecoveryPhrase(manager)
	default:
		return fmt.Errorf("invalid action: %s. Use 'show' or 'import'", action)
	}
}

func showRecoveryPhrase(manager *wallet.Manager) error {
	// Check if wallet exists
	if !manager.VaultExists() {
		return fmt.Errorf("no wallet found. Run 'odyssey init' first")
	}

	// Check if wallet is unlocked
	if !manager.IsUnlocked() {
		return fmt.Errorf("wallet is locked. Run 'odyssey unlock' first")
	}

	// Get mnemonic
	mnemonic, err := manager.GetMnemonic()
	if err != nil {
		return fmt.Errorf("failed to get mnemonic: %w", err)
	}

	fmt.Println("🔐 Recovery Phrase:")
	fmt.Println()
	fmt.Printf("   %s\n", mnemonic)
	fmt.Println()
	fmt.Println("⚠️  Security Warning:")
	fmt.Println("   - Keep this phrase secure and private")
	fmt.Println("   - Anyone with this phrase can access your funds")
	fmt.Println("   - Write it down and store it safely")
	fmt.Println("   - Never share it with anyone")

	return nil
}

func importRecoveryPhrase(manager *wallet.Manager) error {
	// Check if wallet already exists
	if manager.VaultExists() {
		return fmt.Errorf("wallet already exists. Remove existing wallet first")
	}

	fmt.Println("📝 Import Wallet from Recovery Phrase")
	fmt.Println()

	// Get mnemonic from user
	fmt.Print("Enter recovery phrase (24 words): ")
	reader := bufio.NewReader(os.Stdin)
	mnemonic, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read mnemonic: %w", err)
	}

	mnemonic = strings.TrimSpace(mnemonic)

	// Validate mnemonic
	if !isValidMnemonic(mnemonic) {
		return fmt.Errorf("invalid mnemonic. Must be 24 words")
	}

	// Get password
	fmt.Print("Enter password for new wallet: ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read password: %w", err)
	}
	fmt.Println()

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmPassword, err := term.ReadPassword(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to read password confirmation: %w", err)
	}
	fmt.Println()

	if string(password) != string(confirmPassword) {
		return fmt.Errorf("passwords do not match")
	}

	// Import wallet
	err = manager.ImportFromMnemonic(mnemonic, string(password))
	if err != nil {
		return fmt.Errorf("failed to import wallet: %w", err)
	}

	fmt.Println("✅ Wallet imported successfully!")
	fmt.Println()
	fmt.Println("🔑 Next steps:")
	fmt.Println("   - Run 'odyssey unlock' to unlock the wallet")
	fmt.Println("   - Run 'odyssey address' to see your addresses")

	return nil
}

func isValidMnemonic(mnemonic string) bool {
	words := strings.Fields(mnemonic)
	return len(words) == 24
}