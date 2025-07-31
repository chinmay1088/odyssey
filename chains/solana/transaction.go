package solana

import (
	"fmt"
	"strings"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/mr-tron/base58"
)

// Transaction represents a Solana transaction
type Transaction struct {
	Instructions    []solana.Instruction
	Signers         []solana.PrivateKey
	FeePayer        solana.PublicKey
	RecentBlockhash string
}

func NewTransaction(feePayer solana.PublicKey) *Transaction {
	return &Transaction{
		Instructions: make([]solana.Instruction, 0),
		Signers:      make([]solana.PrivateKey, 0),
		FeePayer:     feePayer,
	}
}

func (tx *Transaction) AddTransferInstruction(from solana.PublicKey, to solana.PublicKey, amount uint64) {
	instruction := system.NewTransferInstruction(
		amount,
		from,
		to,
	).Build()
	tx.Instructions = append(tx.Instructions, instruction)
}

func (tx *Transaction) AddSigner(signer solana.PrivateKey) {
	tx.Signers = append(tx.Signers, signer)
}

func (tx *Transaction) SetRecentBlockhash(blockhash string) {
	tx.RecentBlockhash = blockhash
}

func (tx *Transaction) BuildAndSign() (string, error) {
	// Validate blockhash is present
	if tx.RecentBlockhash == "" {
		return "", fmt.Errorf("blockhash is empty")
	}

	// Validate blockhash is valid base58 format
	base58Chars := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	for i, c := range tx.RecentBlockhash {
		if !strings.ContainsRune(base58Chars, c) {
			return "", fmt.Errorf("blockhash contains invalid base58 character '%c' at position %d", c, i)
		}
	}

	// Validate blockhash length (Solana blockhashes are 32 bytes, base58 encoded)
	if len(tx.RecentBlockhash) < 32 {
		return "", fmt.Errorf("blockhash is too short: got %d chars, expected at least 32", len(tx.RecentBlockhash))
	}

	// Try to parse the blockhash
	blockhash, err := solana.HashFromBase58(tx.RecentBlockhash)
	if err != nil {
		return "", fmt.Errorf("invalid blockhash format: %w", err)
	}

	// Create transaction with validated blockhash
	stx, err := solana.NewTransaction(
		tx.Instructions,
		blockhash,
		solana.TransactionPayer(tx.FeePayer),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create transaction: %w", err)
	}

	// Check that we have signers
	if len(tx.Signers) == 0 {
		return "", fmt.Errorf("no signers provided for transaction")
	}

	// Sign the transaction
	_, err = stx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		for _, signer := range tx.Signers {
			if key.Equals(signer.PublicKey()) {
				return &signer
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Serialize the transaction
	serialized, err := stx.MarshalBinary()
	if err != nil {
		return "", fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Use base58 encoding for Solana transactions
	return base58.Encode(serialized), nil
}

// ValidateBase58 validates that a string is valid Base58
func ValidateBase58(s string) bool {
	if s == "" {
		return false
	}

	// Use the same base58 character set as in BuildAndSign
	base58Chars := "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

	for _, c := range s {
		if !strings.ContainsRune(base58Chars, c) {
			return false
		}
	}

	// Additional validation - try to decode with the Solana library
	_, err := solana.HashFromBase58(s)
	return err == nil
}

func ParseAddress(address string) (solana.PublicKey, error) {
	// First check for common errors - ensure string doesn't contain invalid characters
	for i, c := range address {
		// Base58 doesn't use 0, O, I, or l
		if c == '0' || c == 'O' || c == 'I' || c == 'l' {
			return solana.PublicKey{}, fmt.Errorf("invalid character '%c' at position %d in Solana address. Solana addresses use base58 encoding which doesn't include 0, O, I, or l characters", c, i)
		}
	}

	// Now try to parse using the library
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return solana.PublicKey{}, fmt.Errorf("invalid Solana address (%s): %w", address, err)
	}
	return pubKey, nil
}

func LamportsToSOL(lamports uint64) float64 {
	return float64(lamports) / 1000000000.0
}

func SOLToLamports(sol float64) uint64 {
	return uint64(sol * 1000000000.0)
}

func FormatBalance(lamports uint64) string {
	sol := LamportsToSOL(lamports)
	return fmt.Sprintf("%.9f SOL", sol)
}

func ValidateAddress(address string) error {
	_, err := ParseAddress(address)
	return err
}

func CreateTransferTransaction(from solana.PrivateKey, to solana.PublicKey, amount uint64, recentBlockhash string) (*Transaction, error) {
	tx := NewTransaction(from.PublicKey())
	tx.AddTransferInstruction(from.PublicKey(), to, amount)
	tx.AddSigner(from)
	tx.SetRecentBlockhash(recentBlockhash)
	return tx, nil
}
