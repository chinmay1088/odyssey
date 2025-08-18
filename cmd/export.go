package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chinmay1088/odyssey/api"
	"github.com/chinmay1088/odyssey/wallet"
	"github.com/schollz/progressbar/v3"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Export wallet data",
	Long: `Export your wallet data including balances and transaction history.
	
File formats:
  --csv        Export to CSV format (default)
  --json       Export to JSON format
  --txt        Export to txt format
  
Data exported:
  • All supported currencies (ETH, BTC, SOL)
  • Current balances with USD values
  • Transaction history (capped at 50 per chain)
  • Data from your current network (mainnet or testnet)
  
Examples:
  odyssey export                    # Export to CSV (default)
  odyssey export --json            # Export to JSON
  odyssey export --csv --json      # Export to both formats`,
	RunE: runExport,
}

var (
	csvFlag  bool
	jsonFlag bool
	txtFlag  bool
)

func init() {
	exportCmd.Flags().BoolVar(&csvFlag, "csv", false, "Export to CSV format")
	exportCmd.Flags().BoolVar(&jsonFlag, "json", false, "Export to JSON format")
	exportCmd.Flags().BoolVar(&txtFlag, "txt", false, "Export to txt format")
}

func runExport(cmd *cobra.Command, args []string) error {
	manager := wallet.NewManager()
	client := api.NewClient()
	if !manager.IsUnlocked() {
		return fmt.Errorf("wallet is locked. Run 'odyssey unlock' first")
	}
	if !csvFlag && !jsonFlag && !txtFlag {
		csvFlag = true
	}
	currentNetwork := manager.GetCurrentNetwork()

	fmt.Printf("🌐 Current Network: %s\n", strings.ToUpper(currentNetwork))
	fmt.Printf("📊 Exporting %s data...\n", strings.ToUpper(currentNetwork))
	fmt.Println()
	exportData := &ExportData{
		ExportDate:     time.Now().Format("2006-01-02 15:04:05"),
		CurrentNetwork: currentNetwork,
		Data:           &NetworkData{},
	}
	fmt.Println("📊 Preparing export data...")
	bar := progressbar.NewOptions(100,
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionShowBytes(false),
		progressbar.OptionSetWidth(50),
		progressbar.OptionSetDescription("[cyan][1/3][reset] Collecting data..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:     "[green]=[reset]",
			SaucerHead: "[green]>[reset]",
			BarStart:   "[",
			BarEnd:     "]",
		}),
	)

	// collect data for the current network
	bar.Set(0)
	isTestnet := currentNetwork == "testnet"
	if err := collectNetworkData(manager, client, exportData.Data, isTestnet, bar); err != nil {
		return fmt.Errorf("failed to collect data: %w", err)
	}

	bar.Set(70)
	bar.Describe("[cyan][2/3][reset] Preparing export files...")
	exportDir, err := prepareExportDirectory()
	if err != nil {
		return fmt.Errorf("failed to prepare export directory: %w", err)
	}

	bar.Set(85)
	bar.Describe("[cyan][3/3][reset] Writing export files...")
	if err := writeExportFiles(exportData, exportDir, bar); err != nil {
		return fmt.Errorf("failed to write export files: %w", err)
	}

	bar.Set(100)
	bar.Describe("[green][✓][reset] Export completed!")
	fmt.Println()

	fmt.Println("📁 Export completed successfully!")
	fmt.Printf("📍 Files saved to: %s\n", exportDir)
	fmt.Println()
	fmt.Println("📊 Export Summary:")
	fmt.Printf("   Network: %s\n", strings.ToUpper(currentNetwork))
	fmt.Printf("   Currencies: %d\n", len(exportData.Data.Currencies))
	fmt.Printf("   Transactions: %d\n", exportData.Data.TotalTransactions)
	fmt.Println()
	fmt.Println("💡 You can now import these files into spreadsheet applications or use them for record keeping.")

	return nil
}

// export structure
type ExportData struct {
	ExportDate     string       `json:"export_date"`
	CurrentNetwork string       `json:"current_network"`
	Data           *NetworkData `json:"data"`
}

// network data
type NetworkData struct {
	Currencies        []CurrencyData    `json:"currencies"`
	TotalTransactions int               `json:"total_transactions"`
	Transactions      []TransactionData `json:"transactions"`
}

// currency data
type CurrencyData struct {
	Symbol   string `json:"symbol"`
	Name     string `json:"name"`
	Balance  string `json:"balance"`
	USDValue string `json:"usd_value"`
	Address  string `json:"address"`
}

// transaction data
type TransactionData struct {
	Chain       string `json:"chain"`
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Amount      string `json:"amount"`
	Fee         string `json:"fee"`
	USDValue    string `json:"usd_value"`
	Direction   string `json:"direction"`
	Timestamp   string `json:"timestamp"`
	BlockNumber int64  `json:"block_number"`
}

func collectNetworkData(manager *wallet.Manager, client *api.Client, networkData *NetworkData, isTestnet bool, bar *progressbar.ProgressBar) error {
	// collect ethereum data
	if err := collectEthereumData(manager, client, networkData, isTestnet); err != nil {
		// log error but continue with other currencies
		fmt.Printf("⚠️  Warning: Failed to collect Ethereum data: %v\n", err)
	}
	bar.Add(20)

	// collect bitcoin data (mainnet only)
	if !isTestnet {
		if err := collectBitcoinData(manager, client, networkData); err != nil {
			fmt.Printf("⚠️  Warning: Failed to collect Bitcoin data: %v\n", err)
		}
		bar.Add(20) 
	} else {
		// for testnet, bitcoin is not supported
		bar.Add(20)
	}

	// collect solana data
	if err := collectSolanaData(manager, client, networkData, isTestnet); err != nil {
		fmt.Printf("⚠️  Warning: Failed to collect Solana data: %v\n", err)
	}
	bar.Add(20)
	networkData.TotalTransactions = len(networkData.Transactions)

	return nil
}

func collectEthereumData(manager *wallet.Manager, client *api.Client, networkData *NetworkData, isTestnet bool) error {
	address, err := manager.GetEthereumAddress()
	if err != nil {
		return err
	}

	// get eth balance
	balance, err := client.GetEthereumBalance(address.Hex())
	if err != nil {
		return err
	}

	// get usd value
	var usdValue string
	if !isTestnet {
		price, err := client.GetPrice("ethereum")
		if err == nil {
			ethValue := float64(balance.Uint64()) / 1e18
			usdValue = fmt.Sprintf("$%.2f", ethValue*price.USD.InexactFloat64())
		} else {
			usdValue = "N/A"
		}
	} else {
		usdValue = "N/A (Testnet)"
	}

	// add currency data
	networkData.Currencies = append(networkData.Currencies, CurrencyData{
		Symbol:   "ETH",
		Name:     "Ethereum",
		Balance:  fmt.Sprintf("%.6f ETH", float64(balance.Uint64())/1e18),
		USDValue: usdValue,
		Address:  address.Hex(),
	})

	// get transactions (capped at 50)
	transactions, err := client.GetEthereumTransactions(address.Hex())
	if err != nil {
		// continue without transactions
		return nil
	}
	if len(transactions) > 50 {
		transactions = transactions[:50]
	}
	for _, tx := range transactions {
		var txUSDValue string
		if !isTestnet {
			price, err := client.GetPrice("ethereum")
			if err == nil {
				if strings.Contains(tx.Amount, "ETH") {
					ethStr := strings.TrimSpace(strings.Replace(tx.Amount, "ETH", "", -1))
					if ethAmount, err := parseFloat(ethStr); err == nil {
						usdVal := ethAmount * price.USD.InexactFloat64()
						txUSDValue = fmt.Sprintf("$%.2f", usdVal)
					}
				}
			}
		}
		if txUSDValue == "" {
			txUSDValue = "N/A"
		}

		direction := "IN"
		if !tx.IsIncoming {
			direction = "OUT"
		}

		networkData.Transactions = append(networkData.Transactions, TransactionData{
			Chain:       "Ethereum",
			Hash:        tx.Hash,
			From:        tx.From,
			To:          tx.To,
			Amount:      tx.Amount,
			Fee:         tx.Fee,
			USDValue:    txUSDValue,
			Direction:   direction,
			Timestamp:   tx.Timestamp.Format("2006-01-02 15:04:05"),
			BlockNumber: tx.BlockNumber,
		})
	}

	return nil
}

func collectBitcoinData(manager *wallet.Manager, client *api.Client, networkData *NetworkData) error {
	address, err := manager.GetBitcoinAddress()
	if err != nil {
		return err
	}
	balance, err := client.GetBitcoinBalance(address.String())
	if err != nil {
		return err
	}
	var usdValue string
	price, err := client.GetPrice("bitcoin")
	if err == nil {
		usdVal := balance * price.USD.InexactFloat64()
		usdValue = fmt.Sprintf("$%.2f", usdVal)
	} else {
		usdValue = "N/A"
	}
	networkData.Currencies = append(networkData.Currencies, CurrencyData{
		Symbol:   "BTC",
		Name:     "Bitcoin",
		Balance:  fmt.Sprintf("%.8f BTC", balance),
		USDValue: usdValue,
		Address:  address.String(),
	})

	transactions, err := client.GetBitcoinTransactions(address.String())
	if err != nil {
		return nil
	}
	if len(transactions) > 50 {
		transactions = transactions[:50]
	}	
	for _, tx := range transactions {
		var txUSDValue string
		price, err := client.GetPrice("bitcoin")
		if err == nil {
			if strings.Contains(tx.Amount, "BTC") {
				btcStr := strings.TrimSpace(strings.Replace(tx.Amount, "BTC", "", -1))
				if btcAmount, err := parseFloat(btcStr); err == nil {
					usdVal := btcAmount * price.USD.InexactFloat64()
					txUSDValue = fmt.Sprintf("$%.2f", usdVal)
				}
			}
		}
		if txUSDValue == "" {
			txUSDValue = "N/A"
		}

		direction := "IN"
		if !tx.IsIncoming {
			direction = "OUT"
		}

		networkData.Transactions = append(networkData.Transactions, TransactionData{
			Chain:       "Bitcoin",
			Hash:        tx.Hash,
			From:        tx.From,
			To:          tx.To,
			Amount:      tx.Amount,
			Fee:         tx.Fee,
			USDValue:    txUSDValue,
			Direction:   direction,
			Timestamp:   tx.Timestamp.Format("2006-01-02 15:04:05"),
			BlockNumber: tx.BlockNumber,
		})
	}

	return nil
}

func collectSolanaData(manager *wallet.Manager, client *api.Client, networkData *NetworkData, isTestnet bool) error {
	// get solana address
	address, err := manager.GetSolanaAddress()
	if err != nil {
		return err
	}
	balance, err := client.GetSolanaBalance(address.String())
	if err != nil {
		return err
	}
	var usdValue string
	if !isTestnet {
		price, err := client.GetPrice("solana")
		if err == nil {
			solValue := float64(balance) / 1e9
			usdVal := solValue * price.USD.InexactFloat64()
			usdValue = fmt.Sprintf("$%.2f", usdVal)
		} else {
			usdValue = "N/A"
		}
	} else {
		usdValue = "N/A (Testnet)"
	}
	networkData.Currencies = append(networkData.Currencies, CurrencyData{
		Symbol:   "SOL",
		Name:     "Solana",
		Balance:  fmt.Sprintf("%.9f SOL", float64(balance)/1e9),
		USDValue: usdValue,
		Address:  address.String(),
	})

	transactions, err := client.GetSolanaTransactions(address.String())
	if err != nil {
		return nil
	}

	if len(transactions) > 50 {
		transactions = transactions[:50]
	}

	for _, tx := range transactions {
		var txUSDValue string
		if !isTestnet {
			price, err := client.GetPrice("solana")
			if err == nil {
				if strings.Contains(tx.Amount, "SOL") {
					solStr := strings.TrimSpace(strings.Replace(tx.Amount, "SOL", "", -1))
					if solAmount, err := parseFloat(solStr); err == nil {
						usdVal := solAmount * price.USD.InexactFloat64()
						txUSDValue = fmt.Sprintf("$%.2f", usdVal)
					}
				}
			}
		}
		if txUSDValue == "" {
			txUSDValue = "N/A"
		}

		direction := "IN"
		if !tx.IsIncoming {
			direction = "OUT"
		}

		networkData.Transactions = append(networkData.Transactions, TransactionData{
			Chain:       "Solana",
			Hash:        tx.Hash,
			From:        tx.From,
			To:          tx.To,
			Amount:      tx.Amount,
			Fee:         tx.Fee,
			USDValue:    txUSDValue,
			Direction:   direction,
			Timestamp:   tx.Timestamp.Format("2006-01-02 15:04:05"),
			BlockNumber: tx.BlockNumber,
		})
	}

	return nil
}

func prepareExportDirectory() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	exportDir := filepath.Join(homeDir, ".odyssey", "exports")
	if err := os.MkdirAll(exportDir, 0700); err != nil {
		return "", err
	}

	return exportDir, nil
}

func writeExportFiles(exportData *ExportData, exportDir string, bar *progressbar.ProgressBar) error {
	timestamp := time.Now().Format("20060102_150405")
	networkSuffix := "mainnet"
	if exportData.CurrentNetwork == "testnet" {
		networkSuffix = "testnet"
	}

	// write csv files
	if csvFlag {
		if err := writeCSVExport(exportData, exportDir, timestamp, networkSuffix); err != nil {
			return fmt.Errorf("failed to write CSV export: %w", err)
		}
		bar.Add(5)
	}

	// write json files
	if jsonFlag {
		if err := writeJSONExport(exportData, exportDir, timestamp, networkSuffix); err != nil {
			return fmt.Errorf("failed to write JSON export: %w", err)
		}
		bar.Add(5)
	}

	// write txt files
	if txtFlag {
		if err := writeTXTExport(exportData, exportDir, timestamp, networkSuffix); err != nil {
			return fmt.Errorf("failed to write txt export: %w", err)
		}
		bar.Add(5)
	}

	return nil
}

func writeCSVExport(exportData *ExportData, exportDir, timestamp, networkSuffix string) error {
	// write data for the specified network
	if len(exportData.Data.Currencies) > 0 || len(exportData.Data.Transactions) > 0 {
		filename := filepath.Join(exportDir, fmt.Sprintf("odyssey_%s_%s.csv", networkSuffix, timestamp))
		if err := writeCSVFile(filename, exportData.Data, exportData.CurrentNetwork); err != nil {
			return err
		}
	}

	return nil
}

func writeCSVFile(filename string, networkData *NetworkData, networkType string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// write header
	if err := writer.Write([]string{"Network", "Data Type", "Details"}); err != nil {
		return err
	}

	// write currency data
	for _, currency := range networkData.Currencies {
		if err := writer.Write([]string{
			networkType,
			"Currency",
			fmt.Sprintf("%s (%s): %s = %s | Address: %s",
				currency.Name, currency.Symbol, currency.Balance, currency.USDValue, currency.Address),
		}); err != nil {
			return err
		}
	}

	// write transaction data
	for _, tx := range networkData.Transactions {
		if err := writer.Write([]string{
			networkType,
			"Transaction",
			fmt.Sprintf("%s | %s | %s -> %s | Amount: %s (%s) | Fee: %s | Hash: %s | Time: %s",
				tx.Chain, tx.Direction, tx.From, tx.To, tx.Amount, tx.USDValue, tx.Fee, tx.Hash, tx.Timestamp),
		}); err != nil {
			return err
		}
	}

	return nil
}

func writeJSONExport(exportData *ExportData, exportDir, timestamp, networkSuffix string) error {
	// write complete export data
	jsonFile := filepath.Join(exportDir, fmt.Sprintf("odyssey_%s_%s.json", networkSuffix, timestamp))

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(jsonFile, data, 0600); err != nil {
		return err
	}

	return nil
}

func writeTXTExport(exportData *ExportData, exportDir, timestamp, networkSuffix string) error {
	txtFile := filepath.Join(exportDir, fmt.Sprintf("odyssey_%s_%s.txt", networkSuffix, timestamp))

	var content strings.Builder
	content.WriteString("ODYSSEY WALLET EXPORT\n")
	content.WriteString("======================\n\n")
	content.WriteString(fmt.Sprintf("Export Date: %s\n", exportData.ExportDate))
	content.WriteString(fmt.Sprintf("Current Network: %s\n", strings.ToUpper(exportData.CurrentNetwork)))
	content.WriteString(fmt.Sprintf("Export Network: %s\n\n", strings.ToUpper(exportData.CurrentNetwork)))

	// network data
	content.WriteString(fmt.Sprintf("%s DATA\n", strings.ToUpper(exportData.CurrentNetwork)))
	content.WriteString(strings.Repeat("=", len(exportData.CurrentNetwork)+6))
	content.WriteString("\n")

	if len(exportData.Data.Currencies) > 0 {
		content.WriteString("\nCurrencies:\n")
		for _, currency := range exportData.Data.Currencies {
			content.WriteString(fmt.Sprintf("  %s (%s): %s = %s\n",
				currency.Name, currency.Symbol, currency.Balance, currency.USDValue))
			content.WriteString(fmt.Sprintf("    Address: %s\n", currency.Address))
		}
	}

	if len(exportData.Data.Transactions) > 0 {
		content.WriteString(fmt.Sprintf("\nTransactions (%d):\n", len(exportData.Data.Transactions)))
		for i, tx := range exportData.Data.Transactions {
			content.WriteString(fmt.Sprintf("  %d. %s | %s | %s -> %s\n",
				i+1, tx.Chain, tx.Direction, tx.From, tx.To))
			content.WriteString(fmt.Sprintf("     Amount: %s (%s) | Fee: %s\n",
				tx.Amount, tx.USDValue, tx.Fee))
			content.WriteString(fmt.Sprintf("     Hash: %s | Time: %s\n", tx.Hash, tx.Timestamp))
		}
	}

	if err := os.WriteFile(txtFile, []byte(content.String()), 0600); err != nil {
		return err
	}

	return nil
}
