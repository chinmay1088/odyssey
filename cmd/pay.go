package cmd

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/chinmay1088/odyssey/api"
	"github.com/chinmay1088/odyssey/chains/bitcoin"
	"github.com/chinmay1088/odyssey/chains/ethereum"
	"github.com/chinmay1088/odyssey/chains/solana"
	"github.com/chinmay1088/odyssey/wallet"
	"github.com/spf13/cobra"
)

var payCmd = &cobra.Command{
	Use:   "pay [chain] [amount] [address]",
	Short: "Send cryptocurrency",
	Long: `Send cryptocurrency to another address.
	
Supported chains: eth, btc, sol
	
Examples:
  odyssey pay eth 0.1 0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6
  odyssey pay btc 0.001 bc1qxy2kgdygjrsqtzq2n0yrf2493p83kkfjhx0wlh
  odyssey pay sol 1.5 7xKXtg2CW87d97TXJSDpbD5jBkheTqA83TZRuJosgAsU`,
	Args: cobra.ExactArgs(3),
	RunE: runPay,
}

func runPay(cmd *cobra.Command, args []string) error {
	manager := wallet.NewManager()
	client := api.NewClient()

	// Check if wallet is unlocked
	if !manager.IsUnlocked() {
		return fmt.Errorf("wallet is locked. Run 'odyssey unlock' first")
	}

	// Get confirmation before proceeding with any transaction
	if !getTransactionConfirmation(manager) {
		fmt.Println("❌ Transaction cancelled by user")
		return nil
	}

	chain := strings.ToLower(args[0])
	amountStr := args[1]
	recipientAddress := args[2]

	usdFlag, _ := cmd.Flags().GetBool("usd")

	switch chain {
	case "eth", "ethereum":
		return sendEthereum(manager, client, amountStr, recipientAddress, usdFlag)
	case "btc", "bitcoin":
		return sendBitcoin(manager, client, amountStr, recipientAddress, usdFlag)
	case "sol", "solana":
		return sendSolana(manager, client, amountStr, recipientAddress, usdFlag)
	default:
		return fmt.Errorf("unsupported chain: %s. Supported chains: eth, btc, sol", chain)
	}
}

func sendEthereum(manager *wallet.Manager, client *api.Client, amountStr, recipientAddress string, usdFlag bool) error {
	fmt.Println("🔷 Sending Ethereum Transaction")
	fmt.Println()

	// Parse recipient address
	recipient, err := ethereum.ParseAddress(recipientAddress)
	if err != nil {
		return fmt.Errorf("invalid Ethereum address: %w", err)
	}

	// Get sender address
	senderAddress, err := manager.GetEthereumAddress()
	if err != nil {
		return fmt.Errorf("failed to get sender address: %w", err)
	}

	// Parse amount
	var amount float64
	if usdFlag {
		// Convert USD to ETH
		price, err := client.GetPrice("ethereum")
		if err != nil {
			return fmt.Errorf("failed to get ETH price: %w", err)
		}
		usdAmount, err := parseFloat(amountStr)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
		amount = usdAmount / price.USD.InexactFloat64()
	} else {
		amount, err = parseFloat(amountStr)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
	}

	// Convert to Wei
	value := ethereum.EtherToWei(big.NewFloat(amount))

	// Check balance
	balance, err := client.GetEthereumBalance(senderAddress.Hex())
	if err != nil {
		return fmt.Errorf("failed to check balance: %w", err)
	}

	// Check if balance is sufficient
	if balance.Cmp(value) < 0 {
		ethAmount := ethereum.WeiToEther(value)
		currentBalance := ethereum.WeiToEther(balance)
		return fmt.Errorf("insufficient funds in your Ethereum wallet. You're trying to send %.6f ETH but your balance is only %.6f ETH. Please deposit more ETH to your address (%s) before making this payment", ethAmount, currentBalance, senderAddress.Hex())
	}

	// Get nonce
	nonce, err := client.GetEthereumNonce(senderAddress.Hex())
	if err != nil {
		return fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := client.GetEthereumGasPrice()
	if err != nil {
		return fmt.Errorf("failed to get gas price: %w", err)
	}

	// Add 20% to gas price to ensure faster inclusion
	gasPrice.Mul(gasPrice, big.NewInt(120))
	gasPrice.Div(gasPrice, big.NewInt(100))

	// Dynamically estimate gas limit based on the transaction
	estimatedGas, err := client.GetEthereumGasEstimate(senderAddress.Hex(), recipient.Hex(), value, nil)
	if err != nil {
		// Fall back to the basic estimator
		estimatedGas = ethereum.EstimateGasLimit(nil)
	}

	// Use estimated gas with a 20% buffer for safety
	gasLimit := estimatedGas

	// Create transaction
	tx := ethereum.NewTransaction(nonce, recipient, value, gasLimit, gasPrice, nil)

	// Validate transaction
	if err := ethereum.ValidateTransaction(tx); err != nil {
		return fmt.Errorf("invalid transaction: %w", err)
	}

	// Calculate max transaction fee
	maxFee := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))
	totalCost := new(big.Int).Add(value, maxFee)

	// Ensure user has enough for value + gas
	if balance.Cmp(totalCost) < 0 {
		ethAmount := ethereum.WeiToEther(value)
		gasEth := ethereum.WeiToEther(maxFee)
		totalEth := ethereum.WeiToEther(totalCost)
		currentBalance := ethereum.WeiToEther(balance)

		return fmt.Errorf("insufficient funds for transaction with gas. You're trying to send %.6f ETH with approximately %.6f ETH in gas fees (total %.6f ETH) but your balance is only %.6f ETH",
			ethAmount, gasEth, totalEth, currentBalance)
	}

	// Display transaction details for confirmation
	fmt.Printf("📊 Transaction Details:\n")
	fmt.Printf("   From:    %s\n", senderAddress.Hex())
	fmt.Printf("   To:      %s\n", recipient.Hex())

	ethAmount := ethereum.WeiToEther(value)
	feeAmount := ethereum.WeiToEther(maxFee)

	// Show USD values for mainnet
	if !manager.IsTestnet() {
		price, err := client.GetPrice("ethereum")
		if err != nil {
			fmt.Printf("   Amount:  %.6f ETH\n", ethAmount)
			fmt.Printf("   Max Fee: ~%.6f ETH\n", feeAmount)
		} else {
			amountUSD := ethAmount * price.USD.InexactFloat64()
			feeUSD := feeAmount * price.USD.InexactFloat64()
			fmt.Printf("   Amount:  %.6f ETH (~$%.2f)\n", ethAmount, amountUSD)
			fmt.Printf("   Max Fee: ~%.6f ETH (~$%.2f)\n", feeAmount, feeUSD)
		}
	} else {
		fmt.Printf("   Amount:  %.6f ETH\n", ethAmount)
		fmt.Printf("   Max Fee: ~%.6f ETH\n", feeAmount)
	}

	fmt.Printf("   Gas:     %d units\n", gasLimit)
	fmt.Printf("   Gas Price: %.2f Gwei\n", float64(gasPrice.Uint64())/1e9)
	fmt.Printf("   Network: %s\n", manager.GetCurrentNetwork())
	fmt.Println()

	// Get private key
	privateKey, err := manager.GetEthereumKey()
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}

	// Sign transaction
	signedTx, err := ethereum.SignTransaction(tx, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	txHash, err := client.SendEthereumTransaction(signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	fmt.Printf("✅ Transaction sent successfully!\n")
	fmt.Printf("📝 Transaction Hash: %s\n", txHash)

	// Use appropriate explorer URL based on network
	if manager.IsTestnet() {
		fmt.Printf("🔗 Explorer: https://sepolia.etherscan.io/tx/%s\n", txHash)
	} else {
		fmt.Printf("🔗 Explorer: https://etherscan.io/tx/%s\n", txHash)
	}

	return nil
}

func sendBitcoin(manager *wallet.Manager, client *api.Client, amountStr, recipientAddress string, usdFlag bool) error {
	fmt.Println("🟠 Sending Bitcoin Transaction")
	fmt.Println()

	// Parse recipient address
	recipient, err := bitcoin.ParseAddress(recipientAddress)
	if err != nil {
		return fmt.Errorf("invalid Bitcoin address: %w", err)
	}

	// Get sender address
	senderAddress, err := manager.GetBitcoinAddress()
	if err != nil {
		return fmt.Errorf("failed to get sender address: %w", err)
	}

	// Parse amount
	var amount float64
	if usdFlag {
		// Convert USD to BTC
		price, err := client.GetPrice("bitcoin")
		if err != nil {
			return fmt.Errorf("failed to get BTC price: %w", err)
		}
		usdAmount, err := parseFloat(amountStr)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
		amount = usdAmount / price.USD.InexactFloat64()
	} else {
		amount, err = parseFloat(amountStr)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
	}

	// Convert to satoshis
	value := bitcoin.BTCToSatoshis(amount)

	// Get UTXOs
	apiUtxos, err := client.GetBitcoinUTXOs(senderAddress.String())
	if err != nil {
		return fmt.Errorf("failed to get UTXOs: %w", err)
	}

	if len(apiUtxos) == 0 {
		return fmt.Errorf("your Bitcoin wallet has no funds. You need to receive Bitcoin to your address (%s) before you can send any payments. Use 'odyssey balance btc' to check your current balance", senderAddress.String())
	}

	// Convert API UTXOs to bitcoin UTXOs
	var utxos []*bitcoin.UTXO
	totalInput := int64(0)
	for _, apiUtxo := range apiUtxos {
		utxoValue := bitcoin.BTCToSatoshis(apiUtxo.Value)
		totalInput += utxoValue

		utxo := &bitcoin.UTXO{
			TxID:   apiUtxo.TxID,
			Vout:   apiUtxo.Vout,
			Value:  utxoValue,
			Script: []byte(apiUtxo.Script),
		}
		utxos = append(utxos, utxo)
	}

	// Get dynamic fee rate
	feeRate, err := client.GetBitcoinFeeEstimate()
	if err != nil {
		// Default to 10 sat/byte if estimation fails
		feeRate = 10
	}

	// Create transaction
	tx := bitcoin.NewTransaction()

	// Add inputs
	for _, utxo := range utxos {
		err := tx.AddInput(utxo, nil, senderAddress)
		if err != nil {
			return fmt.Errorf("failed to add input: %w", err)
		}
	}

	// Add output
	err = tx.AddOutput(value, recipient)
	if err != nil {
		return fmt.Errorf("failed to add output: %w", err)
	}

	// Estimate transaction size (simplified)
	// P2WPKH: ~110 bytes per input + ~34 bytes per output + ~10 bytes overhead
	txSize := 10 + (len(utxos) * 110) + (1 * 34) // 1 output initially

	// Calculate fee based on estimated size and fee rate
	estimatedFee := int64(txSize) * feeRate

	// Calculate change
	change := totalInput - value - estimatedFee

	// If change is very small (dust), add it to the fee instead
	dustThreshold := int64(546) // Standard dust threshold in satoshis
	if change > 0 && change < dustThreshold {
		estimatedFee += change
		change = 0
	}

	// If we have change to return, add a change output
	if change > 0 {
		err = tx.AddOutput(change, senderAddress)
		if err != nil {
			return fmt.Errorf("failed to add change output: %w", err)
		}
		// Adjust size calculation for the additional output
		txSize += 34
		// Recalculate fee with the new size
		newFee := int64(txSize) * feeRate
		// If fee increased significantly, adjust change
		if newFee > estimatedFee {
			feeIncrease := newFee - estimatedFee
			if change > feeIncrease {
				change -= feeIncrease
				// Update output with new change amount
				tx.UpdateChangeOutput(change)
			}
		}
	}

	// Check if we have enough funds
	if totalInput < value+estimatedFee {
		btcAmount := float64(value) / 100000000.0
		feeAmount := float64(estimatedFee) / 100000000.0
		totalAmount := float64(value+estimatedFee) / 100000000.0
		availableAmount := float64(totalInput) / 100000000.0

		return fmt.Errorf("insufficient funds for transaction with fees. You're trying to send %.8f BTC with approximately %.8f BTC in fees (total %.8f BTC) but your available balance is only %.8f BTC",
			btcAmount, feeAmount, totalAmount, availableAmount)
	}

	// Display transaction details
	fmt.Printf("📊 Transaction Details:\n")
	fmt.Printf("   From:    %s\n", senderAddress.String())
	fmt.Printf("   To:      %s\n", recipient.String())

	btcAmount := float64(value) / 100000000.0
	feeAmount := float64(estimatedFee) / 100000000.0

	// Always show USD for Bitcoin (Bitcoin is mainnet only)
	price, err := client.GetPrice("bitcoin")
	if err != nil {
		fmt.Printf("   Amount:  %.8f BTC\n", btcAmount)
		fmt.Printf("   Fee:     %.8f BTC (%.0f sat/byte)\n", feeAmount, float64(feeRate))
	} else {
		amountUSD := btcAmount * price.USD.InexactFloat64()
		feeUSD := feeAmount * price.USD.InexactFloat64()
		fmt.Printf("   Amount:  %.8f BTC (~$%.2f)\n", btcAmount, amountUSD)
		fmt.Printf("   Fee:     %.8f BTC (~$%.2f) (%.0f sat/byte)\n", feeAmount, feeUSD, float64(feeRate))
	}

	if change > 0 {
		changeAmount := float64(change) / 100000000.0
		if err == nil {
			changeUSD := changeAmount * price.USD.InexactFloat64()
			fmt.Printf("   Change:  %.8f BTC (~$%.2f)\n", changeAmount, changeUSD)
		} else {
			fmt.Printf("   Change:  %.8f BTC\n", changeAmount)
		}
	}
	fmt.Println()

	// Get private key
	privateKey, err := manager.GetBitcoinKey()
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}

	// Sign transaction
	err = tx.SignTransaction(utxos, privateKey, senderAddress)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Serialize transaction
	signedTx, err := tx.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Send transaction
	txHash, err := client.SendBitcoinTransaction(signedTx)
	if err != nil {
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	fmt.Printf("✅ Transaction sent successfully!\n")
	fmt.Printf("📝 Transaction Hash: %s\n", txHash)
	fmt.Printf("🔗 Explorer: https://blockstream.info/tx/%s\n", txHash)

	return nil
}

func sendSolana(manager *wallet.Manager, client *api.Client, amountStr, recipientAddress string, usdFlag bool) error {
	fmt.Println("🟣 Sending Solana Transaction")
	fmt.Println()

	// Parse recipient address
	recipient, err := solana.ParseAddress(recipientAddress)
	if err != nil {
		return fmt.Errorf("invalid Solana address: %w", err)
	}

	// Parse amount
	var amount float64
	if usdFlag {
		// Convert USD to SOL
		price, err := client.GetPrice("solana")
		if err != nil {
			return fmt.Errorf("failed to get SOL price: %w", err)
		}
		usdAmount, err := parseFloat(amountStr)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
		amount = usdAmount / price.USD.InexactFloat64()
	} else {
		amount, err = parseFloat(amountStr)
		if err != nil {
			return fmt.Errorf("invalid amount: %w", err)
		}
	}

	// Convert to lamports
	value := solana.SOLToLamports(amount)

	// Check balance
	senderAddress, err := manager.GetSolanaAddress()
	if err != nil {
		return fmt.Errorf("failed to get sender address: %w", err)
	}

	balance, err := client.GetSolanaBalance(senderAddress.String())
	if err != nil {
		return fmt.Errorf("failed to check balance: %w", err)
	}

	// Solana transaction fees are currently fixed at 5000 lamports (0.000005 SOL)
	const solanaFee = uint64(5000)

	// Add some extra lamports for transaction fee
	requiredBalance := value + solanaFee

	// Check if balance is sufficient
	if balance < requiredBalance {
		solAmount := float64(value) / 1000000000.0
		feeAmount := float64(solanaFee) / 1000000000.0
		totalAmount := float64(requiredBalance) / 1000000000.0
		currentBalance := float64(balance) / 1000000000.0

		return fmt.Errorf("insufficient funds in your Solana wallet. You're trying to send %.9f SOL plus %.9f SOL in fees (total %.9f SOL) but your balance is only %.9f SOL. Please deposit more SOL to your address (%s) before making this payment",
			solAmount, feeAmount, totalAmount, currentBalance, senderAddress.String())
	}

	// Display transaction details
	fmt.Printf("📊 Transaction Details:\n")
	fmt.Printf("   From:    %s\n", senderAddress.String())
	fmt.Printf("   To:      %s\n", recipient.String())

	solAmount := float64(value) / 1000000000.0
	feeAmount := float64(solanaFee) / 1000000000.0

	// Show USD values for mainnet
	if !manager.IsTestnet() {
		price, err := client.GetPrice("solana")
		if err != nil {
			fmt.Printf("   Amount:  %.9f SOL\n", solAmount)
			fmt.Printf("   Fee:     %.9f SOL\n", feeAmount)
		} else {
			amountUSD := solAmount * price.USD.InexactFloat64()
			feeUSD := feeAmount * price.USD.InexactFloat64()
			fmt.Printf("   Amount:  %.9f SOL (~$%.2f)\n", solAmount, amountUSD)
			fmt.Printf("   Fee:     %.9f SOL (~$%.2f)\n", feeAmount, feeUSD)
		}
	} else {
		fmt.Printf("   Amount:  %.9f SOL\n", solAmount)
		fmt.Printf("   Fee:     %.9f SOL\n", feeAmount)
	}

	fmt.Printf("   Network: %s\n", manager.GetCurrentNetwork())
	fmt.Println()

	// Get private key
	privateKey, err := manager.GetSolanaKey()
	if err != nil {
		return fmt.Errorf("failed to get private key: %w", err)
	}

	// Create transaction structure first (without blockhash)
	fmt.Println("⏳ Preparing transaction...")
	tx, err := solana.CreateTransferTransaction(privateKey, recipient, value, "")
	if err != nil {
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	// Get blockhash IMMEDIATELY before sending
	fmt.Println("⏳ Getting fresh blockhash and sending immediately...")
	recentBlockhash, err := client.GetSolanaRecentBlockhash()
	if err != nil {
		return fmt.Errorf("failed to get blockhash: %w", err)
	}

	// Set the fresh blockhash and sign immediately
	tx.SetRecentBlockhash(recentBlockhash)
	signedTx, err := tx.BuildAndSign()
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send immediately - no delay between blockhash fetch and send
	txHash, err := client.SendSolanaTransaction(signedTx)
	if err != nil {
		// Check for common error patterns and provide user-friendly messages
		if strings.Contains(err.Error(), "insufficient funds") || strings.Contains(err.Error(), "0x1") {
			return fmt.Errorf("transaction failed: insufficient funds. Ensure your account has enough SOL for the payment plus network fees")
		}
		if strings.Contains(err.Error(), "blockhash expired") || strings.Contains(err.Error(), "0x1b") || strings.Contains(err.Error(), "BlockhashNotFound") {
			return fmt.Errorf("transaction failed: blockhash expired. The network is busy, please try again")
		}
		if strings.Contains(err.Error(), "invalid base58") {
			return fmt.Errorf("transaction failed due to encoding issues with the RPC. Please try again in a moment")
		}
		return fmt.Errorf("failed to send transaction: %w", err)
	}

	fmt.Printf("✅ Transaction sent successfully!\n")
	fmt.Printf("📝 Transaction Hash: %s\n", txHash)

	// Use appropriate explorer URL based on network
	if manager.IsTestnet() {
		fmt.Printf("🔗 Explorer: https://solscan.io/tx/%s?cluster=devnet\n", txHash)
	} else {
		fmt.Printf("🔗 Explorer: https://solscan.io/tx/%s\n", txHash)
	}

	return nil
}

func getTransactionConfirmation(manager *wallet.Manager) bool {
	fmt.Println()
	if manager.IsTestnet() {
		fmt.Printf("⚠️ You are on testnet (Ethereum Sepolia Testnet / Solana Devnet). By confirming this transaction no real funds will be sent to this address.\n")
	} else {
		fmt.Printf("🚨 You are on main network. By confirming this transaction real funds will be sent to this address.\n")
	}

	fmt.Printf("Press y to confirm or n to stop (y/n): ")

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func parseFloat(s string) (float64, error) {
	// Simple float parsing - in production you'd want more robust parsing
	var result float64
	_, err := fmt.Sscanf(s, "%f", &result)
	return result, err
}

func init() {
	payCmd.Flags().Bool("usd", false, "Specify amount in USD")
}
