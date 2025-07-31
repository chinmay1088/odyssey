package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var buyCmd = &cobra.Command{
	Use:   "buy [chain] [amount] --usd",
	Short: "Buy cryptocurrency with fiat",
	Long: `Buy cryptocurrency with fiat currency using MoonPay.
	
Examples:
  odyssey buy eth 100 --usd    # Buy $100 worth of ETH
  odyssey buy btc 50 --usd     # Buy $50 worth of BTC
  odyssey buy sol 200 --usd    # Buy $200 worth of SOL
  
Note: This command is disabled in testnet mode.`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		// Check if in testnet mode
		network, _ := getCurrentNetwork()
		if network == NetworkTestnet {
			fmt.Println("⚠️  Buy command is disabled in testnet mode.")
			fmt.Println("Please switch to mainnet mode with 'odyssey network mainnet' to buy cryptocurrency.")
			return
		}

		chain := args[0]
		amountStr := args[1]

		usdFlag, _ := cmd.Flags().GetBool("usd")

		// This check is handled by MarkFlagRequired now, but keeping as a double-check
		if !usdFlag {
			fmt.Println("Error: The --usd flag is required to confirm you're specifying an amount in USD")
			fmt.Println("Example: odyssey buy eth 100 --usd")
			return
		}

		amount, err := strconv.ParseFloat(amountStr, 64)
		if err != nil {
			fmt.Printf("Error: Invalid amount: %v\n", err)
			return
		}

		// Generate MoonPay link
		baseURL := "https://buy.moonpay.com"
		var currency string

		switch chain {
		case "eth", "ethereum":
			currency = "eth"
		case "btc", "bitcoin":
			currency = "btc"
		case "sol", "solana":
			currency = "sol"
		default:
			fmt.Printf("Error: Unsupported chain: %s\n", chain)
			return
		}

		url := fmt.Sprintf("%s?currencyCode=%s&baseCurrencyAmount=%f", baseURL, currency, amount)
		fmt.Printf("MoonPay Purchase Link:\n%s\n", url)
		fmt.Println("\nNote: This will open MoonPay's platform for fiat-to-crypto purchase.")
		fmt.Println("Please complete the purchase through MoonPay's secure interface.")
	},
}

func init() {
	buyCmd.Flags().Bool("usd", false, "Required flag to confirm USD amount")
	buyCmd.MarkFlagRequired("usd")
}
