package wallet

import (
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"fmt"
	"math/big"
	"strings"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
)

// HDKey represents a hierarchical deterministic key
type HDKey struct {
	PrivateKey  []byte
	PublicKey   []byte
	ChainCode   []byte
	Depth       uint8
	ChildNum    uint32
	Fingerprint uint32
}

// deriveEthereumKey derives an Ethereum private key from seed and path
func deriveEthereumKey(seed []byte, path accounts.DerivationPath) (*ecdsa.PrivateKey, error) {
	// Create master key
	masterKey, err := newMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// Derive child key
	childKey, err := deriveChildKey(masterKey, path)
	if err != nil {
		return nil, fmt.Errorf("failed to derive child key: %w", err)
	}

	// Convert to ECDSA private key
	privateKey, err := crypto.ToECDSA(childKey.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to ECDSA key: %w", err)
	}

	return privateKey, nil
}

// deriveBitcoinKey derives a Bitcoin private key from seed and path
func deriveBitcoinKey(seed []byte, path string) (*btcec.PrivateKey, error) {
	// Create master key
	masterKey, err := newMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to create master key: %w", err)
	}

	// Parse derivation path
	pathParts := strings.Split(path, "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid derivation path")
	}

	// Derive child key
	childKey := masterKey
	for i := 1; i < len(pathParts); i++ {
		childNum, err := parseChildNum(pathParts[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse child number: %w", err)
		}

		childKey, err = deriveChild(childKey, childNum)
		if err != nil {
			return nil, fmt.Errorf("failed to derive child: %w", err)
		}
	}

	// Convert to Bitcoin private key
	privateKey, _ := btcec.PrivKeyFromBytes(childKey.PrivateKey)
	return privateKey, nil
}

// deriveSolanaKey derives a Solana private key from seed and path
func deriveSolanaKey(seed []byte, path string) (solana.PrivateKey, error) {
	// For Solana, which uses Ed25519, we need to take a different approach
	// Ed25519 doesn't use the same BIP32 derivation process as secp256k1

	// Create a unique seed for each path by adding the path to the seed data
	combinedSeed := append(seed, []byte(path)...)

	// Generate a seed specifically for Solana with the path as context
	h := hmac.New(sha512.New, []byte("ed25519 seed"))
	h.Write(combinedSeed)
	seedHash := h.Sum(nil)

	// Generate an Ed25519 key from this seed
	privateKey := ed25519.NewKeyFromSeed(seedHash[:32])

	// Convert to Solana private key format
	return solana.PrivateKey(privateKey), nil
}

// newMasterKey creates a master key from seed
func newMasterKey(seed []byte) (*HDKey, error) {
	// HMAC-SHA512 with "Bitcoin seed" as key
	key := []byte("Bitcoin seed")
	data := seed

	hash := hmacSHA512(key, data)
	if len(hash) != 64 {
		return nil, fmt.Errorf("invalid hash length")
	}

	privateKey := hash[:32]
	chainCode := hash[32:]

	// Validate private key
	if !isValidPrivateKey(privateKey) {
		return nil, fmt.Errorf("invalid private key")
	}

	// Derive public key
	publicKey := derivePublicKey(privateKey)

	return &HDKey{
		PrivateKey:  privateKey,
		PublicKey:   publicKey,
		ChainCode:   chainCode,
		Depth:       0,
		ChildNum:    0,
		Fingerprint: 0,
	}, nil
}

// deriveChildKey derives a child key from parent key and derivation path
func deriveChildKey(parent *HDKey, derivationPath accounts.DerivationPath) (*HDKey, error) {
	childKey := parent
	for _, childNum := range derivationPath {
		var err error
		childKey, err = deriveChild(childKey, childNum)
		if err != nil {
			return nil, fmt.Errorf("failed to derive child: %w", err)
		}
	}
	return childKey, nil
}

// deriveChild derives a child key from parent
func deriveChild(parent *HDKey, childNum uint32) (*HDKey, error) {
	var data []byte

	if isHardened(childNum) {
		// Hardened derivation
		data = append([]byte{0x00}, parent.PrivateKey...)
	} else {
		// Normal derivation
		data = parent.PublicKey
	}

	// Append child number
	childNumBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(childNumBytes, childNum)
	data = append(data, childNumBytes...)

	// HMAC-SHA512
	hash := hmacSHA512(parent.ChainCode, data)
	if len(hash) != 64 {
		return nil, fmt.Errorf("invalid hash length")
	}

	// Split hash
	il := hash[:32]
	ir := hash[32:]

	// Calculate new private key
	parentPrivateKey := new(big.Int).SetBytes(parent.PrivateKey)
	ilInt := new(big.Int).SetBytes(il)

	// Use secp256k1 curve order
	n := new(big.Int)
	n.SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)

	newPrivateKeyInt := new(big.Int).Add(parentPrivateKey, ilInt)
	newPrivateKeyInt.Mod(newPrivateKeyInt, n)

	if newPrivateKeyInt.Sign() == 0 {
		return nil, fmt.Errorf("invalid private key")
	}

	newPrivateKey := newPrivateKeyInt.Bytes()
	if len(newPrivateKey) < 32 {
		// Pad with zeros
		padded := make([]byte, 32)
		copy(padded[32-len(newPrivateKey):], newPrivateKey)
		newPrivateKey = padded
	}

	// Derive new public key
	newPublicKey := derivePublicKey(newPrivateKey)

	// Calculate fingerprint
	fingerprint := parentFingerprint(parent.PublicKey)

	return &HDKey{
		PrivateKey:  newPrivateKey,
		PublicKey:   newPublicKey,
		ChainCode:   ir,
		Depth:       parent.Depth + 1,
		ChildNum:    childNum,
		Fingerprint: fingerprint,
	}, nil
}

// derivePublicKey derives public key from private key
func derivePublicKey(privateKey []byte) []byte {
	curve := crypto.S256()
	x, y := curve.ScalarBaseMult(privateKey)
	if y.Bit(0) == 0 {
		return append([]byte{0x02}, x.Bytes()...)
	} else {
		return append([]byte{0x03}, x.Bytes()...)
	}
}

// hmacSHA512 computes HMAC-SHA512
func hmacSHA512(key, data []byte) []byte {
	h := hmac.New(sha512.New, key)
	h.Write(data)
	return h.Sum(nil)
}

// isValidPrivateKey checks if private key is valid
func isValidPrivateKey(privateKey []byte) bool {
	if len(privateKey) != 32 {
		return false
	}

	keyInt := new(big.Int).SetBytes(privateKey)
	if keyInt.Sign() == 0 {
		return false
	}

	// Use secp256k1 curve order
	n := new(big.Int)
	n.SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
	if keyInt.Cmp(n) >= 0 {
		return false
	}

	return true
}

// isHardened checks if child number is hardened
func isHardened(childNum uint32) bool {
	return childNum >= 0x80000000
}

// parentFingerprint computes fingerprint from public key
func parentFingerprint(publicKey []byte) uint32 {
	hash := crypto.Keccak256(publicKey)
	return binary.BigEndian.Uint32(hash[:4])
}

func parseChildNum(childStr string) (uint32, error) {
	var childNum uint32
	var err error

	if strings.HasSuffix(childStr, "'") {
		// Hardened
		childNum, err = parseUint32(childStr[:len(childStr)-1])
		if err != nil {
			return 0, err
		}
		childNum += 0x80000000
	} else {
		// Normal
		childNum, err = parseUint32(childStr)
		if err != nil {
			return 0, err
		}
	}

	return childNum, nil
}

func parseUint32(s string) (uint32, error) {
	var result uint32
	_, err := fmt.Sscanf(s, "%d", &result)
	return result, err
}
