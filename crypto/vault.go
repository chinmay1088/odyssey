package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	ScryptN = 32768 // 2^15
	ScryptR = 8
	ScryptP = 1
	KeyLen  = 32 // AES-256 key length
)

type Vault struct {
	Salt   []byte `json:"salt"`
	Nonce  []byte `json:"nonce"`
	Data   []byte `json:"data"`
	MAC    []byte `json:"mac"`
}

type VaultData struct {
	Mnemonic string `json:"mnemonic"`
	Version  int    `json:"version"`
}

func NewVault(mnemonic, password string) (*Vault, error) {
	// Generate random salt
	salt := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	// Derive key from password
	key, err := deriveKey(password, salt)
	if err != nil {
		return nil, fmt.Errorf("failed to derive key: %w", err)
	}
	defer clearBytes(key)

	// Create vault data
	vaultData := VaultData{
		Mnemonic: mnemonic,
		Version:  1,
	}

	// Serialize vault data
	data, err := json.Marshal(vaultData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize vault data: %w", err)
	}

	// Generate random nonce
	nonce := make([]byte, 12)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	encryptedData, mac, err := encrypt(key, nonce, data)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	return &Vault{
		Salt:  salt,
		Nonce: nonce,
		Data:  encryptedData,
		MAC:   mac,
	}, nil
}

func (v *Vault) Decrypt(password string) (string, error) {
	// Derive key from password
	key, err := deriveKey(password, v.Salt)
	if err != nil {
		return "", fmt.Errorf("failed to derive key: %w", err)
	}
	defer clearBytes(key)

	// Decrypt data
	decryptedData, err := decrypt(key, v.Nonce, v.Data, v.MAC)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %w", err)
	}

	// Deserialize vault data
	var vaultData VaultData
	if err := json.Unmarshal(decryptedData, &vaultData); err != nil {
		return "", fmt.Errorf("failed to deserialize vault data: %w", err)
	}

	return vaultData.Mnemonic, nil
}

func deriveKey(password string, salt []byte) ([]byte, error) {
	key, err := scrypt.Key([]byte(password), salt, ScryptN, ScryptR, ScryptP, KeyLen)
	if err != nil {
		return nil, fmt.Errorf("scrypt key derivation failed: %w", err)
	}
	return key, nil
}

func encrypt(key, nonce, data []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	ciphertext := aesGCM.Seal(nil, nonce, data, nil)
	return ciphertext, nil, nil
}

func decrypt(key, nonce, data, mac []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := aesGCM.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func clearBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
}

func (v *Vault) ValidatePassword(password string) bool {
	_, err := v.Decrypt(password)
	return err == nil
}