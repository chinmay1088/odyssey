package cmd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chinmay1088/odyssey/api"
	"github.com/chinmay1088/odyssey/wallet"
	"github.com/spf13/cobra"
)

var (
	pageFlag  int
	limitFlag int
)

type ChainResult struct {
	Chain        string
	Transactions []api.Transaction
	Address      string
	Error        error
}

var transactionsCmd = &cobra.Command{
	Use:   "transactions [chain]",
	Short: "Show transaction history with pagination",
	Long: `Show transaction history for the specified blockchain with pagination support.
Supported chains: eth, btc, sol

Examples:
  odyssey transactions               # Show all transactions (page 1)
  odyssey transactions --page 2     # Show page 2 of all transactions
  odyssey transactions eth --page 1 # Show page 1 of Ethereum transactions
  odyssey transactions sol --limit 5 # Show 5 Solana transactions per page

Pagination: Max 3 pages, 10 transactions per page by default`,
	Args: cobra.MaximumNArgs(1),
	RunE: runTransactions,
}

func init() {
	transactionsCmd.Flags().IntVarP(&pageFlag, "page", "p", 1, "Page number (1-3)")
	transactionsCmd.Flags().IntVarP(&limitFlag, "limit", "l", 10, "Transactions per page (1-20)")
}

func runTransactions(cmd *cobra.Command, args []string) error {
	// Validate pagination parameters
	if pageFlag < 1 || pageFlag > 3 {
		return fmt.Errorf("page must be between 1 and 3")
	}
	if limitFlag < 1 || limitFlag > 20 {
		return fmt.Errorf("limit must be between 1 and 20")
	}

	manager := wallet.NewManager()
	client := api.NewClient()

	// Check if wallet is unlocked
	if !manager.IsUnlocked() {
		return fmt.Errorf("wallet is locked. Run 'odyssey unlock' first")
	}

	// Show loading indicator
	fmt.Println("🔄 Loading transactions...")
	startTime := time.Now()

	// If no chain specified, show all transactions
	if len(args) == 0 {
		err := showAllTransactionsPaginated(manager, client)
		elapsed := time.Since(startTime)
		fmt.Printf("\n⏱️ Loaded in %v\n", elapsed.Round(time.Millisecond*10))
		return err
	}

	// Show specific chain transactions
	chain := strings.ToLower(args[0])
	err := showChainTransactionsPaginated(manager, client, chain)
	elapsed := time.Since(startTime)
	fmt.Printf("\n⏱️ Loaded in %v\n", elapsed.Round(time.Millisecond*10))
	return err
}

func showAllTransactionsPaginated(manager *wallet.Manager, client *api.Client) error {
	// Display network information
	networkType := "Mainnet"
	if manager.IsTestnet() {
		networkType = "Testnet"
	}

	fmt.Printf("📜 Transaction history (Page %d/%d):\n", pageFlag, 3)
	fmt.Printf("🌐 Network: %s\n", networkType)
	fmt.Println()

	// Calculate offset for pagination
	offset := (pageFlag - 1) * limitFlag

	// Prepare channels for parallel fetching
	resultChan := make(chan ChainResult, 3)
	var wg sync.WaitGroup

	// Fetch Ethereum transactions in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		address, err := manager.GetEthereumAddress()
		if err != nil {
			resultChan <- ChainResult{Chain: "ethereum", Error: err}
			return
		}

		// Create context with timeout to avoid long waits
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Channel for API result
		txChan := make(chan []api.Transaction, 1)
		errChan := make(chan error, 1)

		// Fetch with timeout
		go func() {
			txs, err := client.GetEthereumTransactions(address.Hex())
			if err != nil {
				errChan <- err
			} else {
				txChan <- txs
			}
		}()

		// Wait for result or timeout
		var allTxs []api.Transaction
		var fetchErr error

		select {
		case allTxs = <-txChan:
			// Success
		case fetchErr = <-errChan:
			// Error
		case <-ctx.Done():
			fetchErr = fmt.Errorf("timeout fetching transactions (>60s)")
		}

		txs := applyPagination(allTxs, offset, limitFlag)
		resultChan <- ChainResult{
			Chain:        "ethereum",
			Transactions: txs,
			Address:      address.Hex(),
			Error:        fetchErr,
		}
	}()

	// Fetch Bitcoin transactions in parallel (only on mainnet)
	if !manager.IsTestnet() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			address, err := manager.GetBitcoinAddress()
			if err != nil {
				resultChan <- ChainResult{Chain: "bitcoin", Error: err}
				return
			}

			// Create context with timeout to avoid long waits
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			// Channel for API result
			txChan := make(chan []api.Transaction, 1)
			errChan := make(chan error, 1)

			// Fetch with timeout
			go func() {
				txs, err := client.GetBitcoinTransactions(address.String())
				if err != nil {
					errChan <- err
				} else {
					txChan <- txs
				}
			}()

			// Wait for result or timeout
			var allTxs []api.Transaction
			var fetchErr error

			select {
			case allTxs = <-txChan:
				// Success
			case fetchErr = <-errChan:
				// Error
			case <-ctx.Done():
				fetchErr = fmt.Errorf("timeout fetching transactions (>60s)")
			}

			txs := applyPagination(allTxs, offset, limitFlag)
			resultChan <- ChainResult{
				Chain:        "bitcoin",
				Transactions: txs,
				Address:      address.String(),
				Error:        fetchErr,
			}
		}()
	}

	// Fetch Solana transactions in parallel
	wg.Add(1)
	go func() {
		defer wg.Done()
		address, err := manager.GetSolanaAddress()
		if err != nil {
			resultChan <- ChainResult{Chain: "solana", Error: err}
			return
		}

		// Create context with timeout to avoid long waits
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		// Channel for API result
		txChan := make(chan []api.Transaction, 1)
		errChan := make(chan error, 1)

		// Fetch with timeout
		go func() {
			txs, err := client.GetSolanaTransactions(address.String())
			if err != nil {
				errChan <- err
			} else {
				txChan <- txs
			}
		}()

		// Wait for result or timeout
		var allTxs []api.Transaction
		var fetchErr error

		select {
		case allTxs = <-txChan:
			// Success
		case fetchErr = <-errChan:
			// Error
		case <-ctx.Done():
			fetchErr = fmt.Errorf("timeout fetching transactions (>60s)")
		}

		txs := applyPagination(allTxs, offset, limitFlag)
		resultChan <- ChainResult{
			Chain:        "solana",
			Transactions: txs,
			Address:      address.String(),
			Error:        fetchErr,
		}
	}()

	// Wait for all goroutines to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := make(map[string]ChainResult)
	for result := range resultChan {
		results[result.Chain] = result
	}

	// Display results in order
	displayChainResult(results["ethereum"], "🔷", "Ethereum", manager.IsTestnet(), client)

	if !manager.IsTestnet() {
		displayChainResult(results["bitcoin"], "🟠", "Bitcoin", false, client)
	}

	displayChainResult(results["solana"], "🟣", "Solana", manager.IsTestnet(), client)

	// Show pagination info
	showPaginationInfo()
	return nil
}

func showChainTransactionsPaginated(manager *wallet.Manager, client *api.Client, chain string) error {
	// Display network information
	networkType := "Mainnet"
	if manager.IsTestnet() {
		networkType = "Testnet"
	}
	fmt.Printf("📜 Transaction history:\n")
	fmt.Printf("🌐 Network: %s\n", networkType)
	fmt.Println()

	// Calculate offset for pagination
	offset := (pageFlag - 1) * limitFlag

	switch chain {
	case "eth", "ethereum":
		address, err := manager.GetEthereumAddress()
		if err != nil {
			return fmt.Errorf("failed to get Ethereum address: %w", err)
		}

		chainName := "Ethereum (ETH)"
		explorerBase := "https://etherscan.io"
		if manager.IsTestnet() {
			chainName = "Ethereum (Sepolia)"
			explorerBase = "https://sepolia.etherscan.io"
		}

		fmt.Printf("🔷 %s transactions for: %s\n", chainName, address.Hex())
		fmt.Printf("📄 Page %d/%d (%d per page)\n\n", pageFlag, 3, limitFlag)

		// Fetch transactions with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		txChan := make(chan []api.Transaction, 1)
		errChan := make(chan error, 1)

		go func() {
			txs, err := client.GetEthereumTransactions(address.Hex())
			if err != nil {
				errChan <- err
			} else {
				txChan <- txs
			}
		}()

		var allTxs []api.Transaction
		var fetchErr error

		select {
		case allTxs = <-txChan:
			// Success
		case fetchErr = <-errChan:
			// Error
		case <-ctx.Done():
			fetchErr = fmt.Errorf("timeout fetching transactions (>60s)")
		}

		txs := applyPagination(allTxs, offset, limitFlag)
		if fetchErr != nil {
			fmt.Printf("❌ Error fetching transactions: %v\n", fetchErr)
			fmt.Printf("💡 View on Etherscan: %s/address/%s\n", explorerBase, address.Hex())
		} else if len(txs) == 0 {
			if pageFlag == 1 {
				fmt.Println("No transactions found")
			} else {
				fmt.Println("No more transactions on this page")
			}
		} else {
			printTransactionsPaginated(txs, client, "ethereum", manager.IsTestnet())
		}

	case "btc", "bitcoin":
		if manager.IsTestnet() {
			return fmt.Errorf("bitcoin is not supported in testnet mode")
		}

		address, err := manager.GetBitcoinAddress()
		if err != nil {
			return fmt.Errorf("failed to get Bitcoin address: %w", err)
		}

		fmt.Printf("🟠 Bitcoin (BTC) transactions for: %s\n", address.String())
		fmt.Printf("📄 Page %d/%d (%d per page)\n\n", pageFlag, 3, limitFlag)

		// Fetch transactions with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		txChan := make(chan []api.Transaction, 1)
		errChan := make(chan error, 1)

		go func() {
			txs, err := client.GetBitcoinTransactions(address.String())
			if err != nil {
				errChan <- err
			} else {
				txChan <- txs
			}
		}()

		var allTxs []api.Transaction
		var fetchErr error

		select {
		case allTxs = <-txChan:
			// Success
		case fetchErr = <-errChan:
			// Error
		case <-ctx.Done():
			fetchErr = fmt.Errorf("timeout fetching transactions (>60s)")
		}

		txs := applyPagination(allTxs, offset, limitFlag)
		if fetchErr != nil {
			fmt.Printf("❌ Error fetching transactions: %v\n", fetchErr)
			fmt.Printf("💡 View on Blockstream: https://blockstream.info/address/%s\n", address.String())
		} else if len(txs) == 0 {
			if pageFlag == 1 {
				fmt.Println("No transactions found")
			} else {
				fmt.Println("No more transactions on this page")
			}
		} else {
			printTransactionsPaginated(txs, client, "bitcoin", manager.IsTestnet())
		}

	case "sol", "solana":
		address, err := manager.GetSolanaAddress()
		if err != nil {
			return fmt.Errorf("failed to get Solana address: %w", err)
		}

		chainName := "Solana"
		explorerBase := "https://solscan.io/account"
		clusterParam := ""
		if manager.IsTestnet() {
			chainName = "Solana (Devnet)"
			clusterParam = "?cluster=devnet"
		}

		fmt.Printf("🟣 %s transactions for: %s\n", chainName, address.String())
		fmt.Printf("📄 Page %d/%d (%d per page)\n", pageFlag, 3, limitFlag)
		fmt.Printf("💡 View on Solscan: %s/%s%s\n\n", explorerBase, address.String(), clusterParam)

		// Fetch transactions with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		txChan := make(chan []api.Transaction, 1)
		errChan := make(chan error, 1)

		go func() {
			txs, err := client.GetSolanaTransactions(address.String())
			if err != nil {
				errChan <- err
			} else {
				txChan <- txs
			}
		}()

		var allTxs []api.Transaction
		var fetchErr error

		select {
		case allTxs = <-txChan:
			// Success
		case fetchErr = <-errChan:
			// Error
		case <-ctx.Done():
			fetchErr = fmt.Errorf("timeout fetching transactions (>60s)")
		}

		txs := applyPagination(allTxs, offset, limitFlag)
		if fetchErr != nil {
			fmt.Printf("❌ Error fetching transactions: %v\n", fetchErr)
		} else if len(txs) == 0 {
			if pageFlag == 1 {
				fmt.Println("No transactions found")
				fmt.Println("💡 Tip: Solana accounts don't exist until they receive SOL")
			} else {
				fmt.Println("No more transactions on this page")
			}
		} else {
			printTransactionsPaginated(txs, client, "solana", manager.IsTestnet())
		}

	default:
		return fmt.Errorf("unsupported chain: %s. Supported chains: eth, btc, sol", chain)
	}

	// Show pagination info
	showPaginationInfo()
	return nil
}

func displayChainResult(result ChainResult, emoji, name string, isTestnet bool, client *api.Client) {
	// Handle case where result might be empty
	if result.Chain == "" {
		return
	}
	displayName := name
	explorerBase := ""

	switch name {
	case "Ethereum":
		if isTestnet {
			displayName = "Ethereum (Sepolia)"
			explorerBase = "https://sepolia.etherscan.io"
		} else {
			explorerBase = "https://etherscan.io"
		}
	case "Bitcoin":
		explorerBase = "https://blockstream.info"
	case "Solana":
		if isTestnet {
			displayName = "Solana (Devnet)"
			explorerBase = "https://solscan.io"
		} else {
			explorerBase = "https://solscan.io"
		}
	}

	fmt.Printf("%s %s:\n", emoji, displayName)
	if result.Error != nil {
		fmt.Printf("   ❌ Error fetching transactions: %v\n", result.Error)
		if result.Address != "" {
			if name == "Solana" && isTestnet {
				fmt.Printf("   💡 View on explorer: %s/account/%s?cluster=devnet\n", explorerBase, result.Address)
			} else if name == "Solana" {
				fmt.Printf("   💡 View on explorer: %s/account/%s\n", explorerBase, result.Address)
			} else {
				fmt.Printf("   💡 View on explorer: %s/address/%s\n", explorerBase, result.Address)
			}
		}
	} else if len(result.Transactions) == 0 {
		if pageFlag == 1 {
			fmt.Println("   No transactions found")
			if name == "Solana" {
				fmt.Println("   💡 Tip: Solana accounts don't exist until they receive SOL")
			}
		} else {
			fmt.Println("   No more transactions on this page")
		}
	} else {
		fmt.Printf("   Address: %s\n", result.Address)
		fmt.Println("   Recent transactions:")

		// Determine crypto symbol from chain name
		var cryptoSymbol string
		switch name {
		case "Ethereum":
			cryptoSymbol = "ethereum"
		case "Bitcoin":
			cryptoSymbol = "bitcoin"
		case "Solana":
			cryptoSymbol = "solana"
		default:
			cryptoSymbol = "unknown"
		}

		printTransactionsIndented(result.Transactions, client, cryptoSymbol, isTestnet)
	}
	fmt.Println()
}

func printTransactionsPaginated(txs []api.Transaction, client *api.Client, cryptoSymbol string, isTestnet bool) {
	for i, tx := range txs {
		// Direction indicator
		direction := "⬅️ IN"
		if !tx.IsIncoming {
			direction = "➡️ OUT"
		}

		// Format timestamp
		timeStr := tx.Timestamp.Format("2006-01-02 15:04:05")

		// Truncate addresses for display
		fromShort := truncateAddress(tx.From)
		toShort := truncateAddress(tx.To)

		// Get USD values
		amountUSD := getUSDValue(client, cryptoSymbol, tx.Amount, isTestnet)
		feeUSD := getUSDValue(client, cryptoSymbol, tx.Fee, isTestnet)

		fmt.Printf("%d. %s | %s\n", i+1, direction, timeStr)
		fmt.Printf("   Hash: %s\n", tx.Hash)
		fmt.Printf("   From: %s\n", fromShort)
		fmt.Printf("   To:   %s\n", toShort)

		if amountUSD != "" {
			fmt.Printf("   Amount: %s (%s)\n", tx.Amount, amountUSD)
		} else {
			fmt.Printf("   Amount: %s\n", tx.Amount)
		}

		if feeUSD != "" {
			fmt.Printf("   Fee: %s (%s)\n", tx.Fee, feeUSD)
		} else {
			fmt.Printf("   Fee: %s\n", tx.Fee)
		}

		if i < len(txs)-1 {
			fmt.Println()
		}
	}
}

func printTransactionsIndented(txs []api.Transaction, client *api.Client, cryptoSymbol string, isTestnet bool) {
	for i, tx := range txs {
		// Direction indicator
		direction := "⬅️ IN"
		if !tx.IsIncoming {
			direction = "➡️ OUT"
		}

		// Format timestamp
		timeStr := tx.Timestamp.Format("2006-01-02 15:04:05")

		// Truncate addresses for display
		fromShort := truncateAddress(tx.From)
		toShort := truncateAddress(tx.To)

		// Get USD values
		amountUSD := getUSDValue(client, cryptoSymbol, tx.Amount, isTestnet)
		feeUSD := getUSDValue(client, cryptoSymbol, tx.Fee, isTestnet)

		fmt.Printf("   %d. %s | %s\n", i+1, direction, timeStr)
		fmt.Printf("      Hash: %s\n", tx.Hash)
		fmt.Printf("      From: %s\n", fromShort)
		fmt.Printf("      To:   %s\n", toShort)

		if amountUSD != "" {
			fmt.Printf("      Amount: %s (%s)\n", tx.Amount, amountUSD)
		} else {
			fmt.Printf("      Amount: %s\n", tx.Amount)
		}

		if feeUSD != "" {
			fmt.Printf("      Fee: %s (%s)\n", tx.Fee, feeUSD)
		} else {
			fmt.Printf("      Fee: %s\n", tx.Fee)
		}

		if i < len(txs)-1 {
			fmt.Println()
		}
	}
}

func showPaginationInfo() {
	fmt.Println()
	fmt.Println("📄 Pagination:")
	if pageFlag > 1 {
		fmt.Printf("   ⬅️  Previous: --page %d\n", pageFlag-1)
	}
	if pageFlag < 3 {
		fmt.Printf("   ➡️  Next: --page %d\n", pageFlag+1)
	}
	fmt.Printf("   📊 Showing page %d of 3 (%d transactions per page)\n", pageFlag, limitFlag)
	fmt.Println("   💡 Use --limit to change transactions per page (max 20)")
}

func applyPagination(txs []api.Transaction, offset, limit int) []api.Transaction {
	// Instead of fetching all and slicing, we should limit the fetch itself
	// For now, return early to avoid slow sequential calls
	if len(txs) == 0 {
		return []api.Transaction{}
	}

	// Limit to first 30 transactions max to avoid slow API calls
	maxFetch := 30
	if len(txs) > maxFetch {
		txs = txs[:maxFetch]
	}

	if offset >= len(txs) {
		return []api.Transaction{}
	}

	end := offset + limit
	if end > len(txs) {
		end = len(txs)
	}

	return txs[offset:end]
}

// truncateAddress shortens long blockchain addresses for display
func truncateAddress(address string) string {
	if len(address) <= 12 {
		return address
	}
	return address[:6] + "..." + address[len(address)-6:]
}

// getUSDValue fetches price and converts crypto amount to USD
func getUSDValue(client *api.Client, cryptoSymbol, amountStr string, isTestnet bool) string {
	// Don't show USD for testnet
	if isTestnet {
		return ""
	}

	// Get price
	price, err := client.GetPrice(cryptoSymbol)
	if err != nil {
		return ""
	}

	// Parse amount based on crypto type
	var cryptoAmount float64
	var success bool

	switch cryptoSymbol {
	case "ethereum":
		// Parse ETH amount (format: "0.123456 ETH")
		cryptoAmount, success = parseEthAmount(amountStr)
	case "bitcoin":
		// Parse BTC amount (format: "0.12345678 BTC")
		cryptoAmount, success = parseBtcAmount(amountStr)
	case "solana":
		// Parse SOL amount (format: "1.234567890 SOL")
		cryptoAmount, success = parseSolAmount(amountStr)
	default:
		return ""
	}

	if !success {
		return ""
	}

	usdValue := cryptoAmount * price.USD.InexactFloat64()
	return fmt.Sprintf("~$%.2f", usdValue)
}

// parseEthAmount extracts numeric value from ETH amount string
func parseEthAmount(amountStr string) (float64, bool) {
	// Remove "ETH" suffix and parse
	if strings.HasSuffix(amountStr, " ETH") {
		numStr := strings.TrimSuffix(amountStr, " ETH")
		if amount, err := parseFloat(numStr); err == nil {
			return amount, true
		}
	}
	return 0, false
}

// parseBtcAmount extracts numeric value from BTC amount string
func parseBtcAmount(amountStr string) (float64, bool) {
	// Remove "BTC" suffix and parse
	if strings.HasSuffix(amountStr, " BTC") {
		numStr := strings.TrimSuffix(amountStr, " BTC")
		if amount, err := parseFloat(numStr); err == nil {
			return amount, true
		}
	}
	return 0, false
}

// parseSolAmount extracts numeric value from SOL amount string
func parseSolAmount(amountStr string) (float64, bool) {
	// Remove "SOL" suffix and parse
	if strings.HasSuffix(amountStr, " SOL") {
		numStr := strings.TrimSuffix(amountStr, " SOL")
		if amount, err := parseFloat(numStr); err == nil {
			return amount, true
		}
	}
	return 0, false
}
