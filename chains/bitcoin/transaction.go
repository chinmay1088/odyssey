package bitcoin

import (
	"crypto/ecdsa"
	"fmt"

	"bytes"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

// UTXO represents an unspent transaction output
type UTXO struct {
	TxID   string
	Vout   uint32
	Value  int64
	Script []byte
}

// Transaction represents a Bitcoin transaction
type Transaction struct {
	Version  int32
	Inputs   []*wire.TxIn
	Outputs  []*wire.TxOut
	LockTime uint32
}

// NewTransaction creates a new Bitcoin transaction
func NewTransaction() *Transaction {
	return &Transaction{
		Version:  2, // SegWit version
		Inputs:   make([]*wire.TxIn, 0),
		Outputs:  make([]*wire.TxOut, 0),
		LockTime: 0,
	}
}

// AddInput adds an input to the transaction
func (tx *Transaction) AddInput(utxo *UTXO, _ *btcec.PrivateKey, _ btcutil.Address) error {
	prevHash, err := chainhash.NewHashFromStr(utxo.TxID)
	if err != nil {
		return fmt.Errorf("invalid previous transaction hash: %w", err)
	}
	input := wire.NewTxIn(
		wire.NewOutPoint(prevHash, utxo.Vout),
		nil, // Signature script will be set later
		nil, // Witness will be set later
	)
	tx.Inputs = append(tx.Inputs, input)
	return nil
}

// AddOutput adds an output to the transaction
func (tx *Transaction) AddOutput(value int64, address btcutil.Address) error {
	script, err := txscript.PayToAddrScript(address)
	if err != nil {
		return fmt.Errorf("failed to create output script: %w", err)
	}
	output := wire.NewTxOut(value, script)
	tx.Outputs = append(tx.Outputs, output)
	return nil
}

// SignTransaction signs all inputs in the transaction
func (tx *Transaction) SignTransaction(utxos []*UTXO, privateKey *btcec.PrivateKey, address btcutil.Address) error {
	wireTx := tx.toWireTx()
	fetcher := txscript.NewMultiPrevOutFetcher(nil)
	hashes := txscript.NewTxSigHashes(wireTx, fetcher)
	for i, input := range tx.Inputs {
		if i >= len(utxos) {
			return fmt.Errorf("insufficient UTXOs for signing")
		}
		utxo := utxos[i]
		// For real SegWit, you need the correct scriptPubKey and value
		script, err := txscript.PayToAddrScript(address)
		if err != nil {
			return fmt.Errorf("failed to create script: %w", err)
		}
		sighash, err := txscript.CalcWitnessSigHash(script, hashes, txscript.SigHashAll, wireTx, i, utxo.Value)
		if err != nil {
			return fmt.Errorf("failed to calculate sighash: %w", err)
		}

		// Convert to ECDSA private key for signing
		ecdsaPrivKey := privateKey.ToECDSA()
		sig, err := ecdsa.SignASN1(nil, ecdsaPrivKey, sighash)
		if err != nil {
			return fmt.Errorf("failed to sign input %d: %w", i, err)
		}

		pubKey := privateKey.PubKey()
		witness := wire.TxWitness{
			append(sig, byte(txscript.SigHashAll)),
			pubKey.SerializeCompressed(),
		}
		input.Witness = witness
	}
	return nil
}

// Serialize serializes the transaction to hex
func (tx *Transaction) Serialize() (string, error) {
	wireTx := tx.toWireTx()
	var buf bytes.Buffer
	err := wireTx.Serialize(&buf)
	if err != nil {
		return "", fmt.Errorf("failed to serialize transaction: %w", err)
	}
	return fmt.Sprintf("%x", buf.Bytes()), nil
}

// toWireTx converts to wire.MsgTx
func (tx *Transaction) toWireTx() *wire.MsgTx {
	wireTx := wire.NewMsgTx(tx.Version)
	for _, input := range tx.Inputs {
		wireTx.AddTxIn(input)
	}
	for _, output := range tx.Outputs {
		wireTx.AddTxOut(output)
	}
	wireTx.LockTime = tx.LockTime
	return wireTx
}

// CalculateFee calculates the transaction fee
func (tx *Transaction) CalculateFee(inputValue int64, outputValue int64) int64 {
	return inputValue - outputValue
}

// EstimateFee estimates the transaction fee based on size
func (tx *Transaction) EstimateFee(inputCount int, outputCount int, feeRate int64) int64 {
	// Estimate transaction size
	// Base size: 4 bytes version + 4 bytes locktime
	baseSize := 8

	// Input size: 32 bytes prev hash + 4 bytes index + 1 byte script length + 0 bytes script + 4 bytes sequence
	inputSize := 41

	// Output size: 8 bytes value + 1 byte script length + 25 bytes P2WPKH script
	outputSize := 34

	// Witness size: 1 byte witness count + 1 byte signature length + 73 bytes signature + 1 byte pubkey length + 33 bytes pubkey
	witnessSize := 109

	totalSize := baseSize + (inputCount * inputSize) + (outputCount * outputSize) + (inputCount * witnessSize)

	// Convert to virtual bytes (weight units / 4)
	virtualBytes := totalSize

	return int64(virtualBytes) * feeRate
}

// ParseAddress parses a Bitcoin address
func ParseAddress(address string) (btcutil.Address, error) {
	return btcutil.DecodeAddress(address, &chaincfg.MainNetParams)
}

// SatoshisToBTC converts satoshis to BTC
func SatoshisToBTC(satoshis int64) float64 {
	return float64(satoshis) / 100000000.0
}

// BTCToSatoshis converts BTC to satoshis
func BTCToSatoshis(btc float64) int64 {
	return int64(btc * 100000000.0)
}

// FormatBalance formats balance in a human-readable format
func FormatBalance(satoshis int64) string {
	btc := SatoshisToBTC(satoshis)
	return fmt.Sprintf("%.8f BTC", btc)
}

// ValidateAddress validates a Bitcoin address
func ValidateAddress(address string) error {
	_, err := ParseAddress(address)
	return err
}

// CreateP2WPKHAddress creates a P2WPKH address from public key
func CreateP2WPKHAddress(publicKey *btcec.PublicKey) (btcutil.Address, error) {
	pubKeyHash := btcutil.Hash160(publicKey.SerializeCompressed())
	return btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, &chaincfg.MainNetParams)
}

// UpdateChangeOutput updates the value of the last output in the transaction (change output)
func (tx *Transaction) UpdateChangeOutput(value int64) error {
	if len(tx.Outputs) < 2 {
		return fmt.Errorf("no change output to update")
	}
	// Last output is assumed to be the change output
	tx.Outputs[len(tx.Outputs)-1].Value = value
	return nil
}
