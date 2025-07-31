package wallet

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/chinmay1088/odyssey/crypto"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/tyler-smith/go-bip39"
)

const (
	// Network type constants
	NetworkMainnet = "mainnet"
	NetworkTestnet = "testnet"

	// Derivation paths for different chains (mainnet)
	EthDerivationPath = "m/44'/60'/0'/0/0"
	BtcDerivationPath = "m/44'/0'/0'/0/0"
	SolDerivationPath = "m/44'/501'/0'/0'"

	// Derivation paths for testnet
	EthTestnetDerivationPath = "m/44'/1'/0'/0/0"  // Use coin type 1 for testnet
	SolTestnetDerivationPath = "m/44'/501'/0'/1'" // Use different account index for testnet

	// Session duration in minutes
	SessionDuration = 30
)

// SessionData holds the wallet session information
type SessionData struct {
	Token      string    `json:"token"`
	Mnemonic   string    `json:"mnemonic"`
	Expiration time.Time `json:"expiration"`
	Network    string    `json:"network"` // Store network with session
}

// Manager handles wallet operations and key derivation
type Manager struct {
	vaultPath   string
	sessionPath string
	vault       *crypto.Vault
	mnemonic    string
	password    string
	mu          sync.RWMutex
	unlocked    bool
	network     string // Current network (mainnet or testnet)
}

// NewManager creates a new wallet manager
func NewManager() *Manager {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Sprintf("failed to get home directory: %v", err))
	}

	// Determine the current network
	network := NetworkMainnet // Default to mainnet
	networkPath := filepath.Join(homeDir, ".odyssey", "network.txt")

	// Read network file if it exists
	if _, err := os.Stat(networkPath); err == nil {
		data, err := os.ReadFile(networkPath)
		if err == nil {
			network = strings.TrimSpace(string(data))
			// Validate network
			if network != NetworkMainnet && network != NetworkTestnet {
				network = NetworkMainnet // Default to mainnet if invalid
			}
		}
	}

	return &Manager{
		vaultPath:   filepath.Join(homeDir, ".odyssey", "wallet.vault"),
		sessionPath: filepath.Join(homeDir, ".odyssey", "session.json"),
		network:     network,
	}
}

// generateSessionToken creates a random session token
func generateSessionToken() (string, error) {
	tokenBytes := make([]byte, 32)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

// createSession creates and saves a new session
func (m *Manager) createSession() error {
	token, err := generateSessionToken()
	if err != nil {
		return fmt.Errorf("failed to generate session token: %w", err)
	}

	session := SessionData{
		Token:      token,
		Mnemonic:   m.mnemonic,
		Expiration: time.Now().Add(SessionDuration * time.Minute),
		Network:    m.network, // Save current network with session
	}

	data, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(m.sessionPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// loadSession loads the session if it exists and is valid
func (m *Manager) loadSession() bool {
	data, err := os.ReadFile(m.sessionPath)
	if err != nil {
		return false
	}

	var session SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		// Session file is corrupted, delete it
		os.Remove(m.sessionPath)
		return false
	}

	// Check if session has expired
	if time.Now().After(session.Expiration) {
		// Session expired, delete it
		os.Remove(m.sessionPath)
		return false
	}

	// Check if network matches current network
	if session.Network != m.network {
		// Network mismatch, session not valid for current network
		return false
	}

	// Session is valid, load the mnemonic
	m.mnemonic = session.Mnemonic
	m.unlocked = true

	return true
}

// clearSession removes the current session
func (m *Manager) clearSession() {
	os.Remove(m.sessionPath)
}

// Initialize creates a new wallet with a fresh mnemonic
func (m *Manager) Initialize(password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Generate new mnemonic
	entropy, err := bip39.NewEntropy(256) // 24 words
	if err != nil {
		return fmt.Errorf("failed to generate entropy: %w", err)
	}

	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return fmt.Errorf("failed to generate mnemonic: %w", err)
	}

	// Create vault
	vault, err := crypto.NewVault(mnemonic, password)
	if err != nil {
		return fmt.Errorf("failed to create vault: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.vaultPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save vault
	if err := m.saveVault(vault); err != nil {
		return fmt.Errorf("failed to save vault: %w", err)
	}

	m.vault = vault
	m.mnemonic = mnemonic
	m.password = password
	m.unlocked = true

	// Create session
	if err := m.createSession(); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// ImportFromMnemonic imports a wallet from an existing mnemonic
func (m *Manager) ImportFromMnemonic(mnemonic, password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate mnemonic
	if !bip39.IsMnemonicValid(mnemonic) {
		return fmt.Errorf("invalid mnemonic")
	}

	// Create vault
	vault, err := crypto.NewVault(mnemonic, password)
	if err != nil {
		return fmt.Errorf("failed to create vault: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(m.vaultPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Save vault
	if err := m.saveVault(vault); err != nil {
		return fmt.Errorf("failed to save vault: %w", err)
	}

	m.vault = vault
	m.mnemonic = mnemonic
	m.password = password
	m.unlocked = true

	// Create session
	if err := m.createSession(); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// Unlock unlocks the wallet with the provided password
func (m *Manager) Unlock(password string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// First try to load existing session
	if m.loadSession() {
		return nil
	}

	// Load vault if not loaded
	if m.vault == nil {
		vault, err := m.loadVault()
		if err != nil {
			return fmt.Errorf("failed to load vault: %w", err)
		}
		m.vault = vault
	}

	// Validate password
	if !m.vault.ValidatePassword(password) {
		return fmt.Errorf("invalid password")
	}

	// Decrypt mnemonic
	mnemonic, err := m.vault.Decrypt(password)
	if err != nil {
		return fmt.Errorf("failed to decrypt vault: %w", err)
	}

	m.mnemonic = mnemonic
	m.password = password
	m.unlocked = true

	// Create session
	if err := m.createSession(); err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

// Lock locks the wallet and clears sensitive data from memory
func (m *Manager) Lock() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.unlocked = false
	m.mnemonic = ""
	m.password = ""

	// Clear session
	m.clearSession()
}

// IsUnlocked returns whether the wallet is currently unlocked
func (m *Manager) IsUnlocked() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If already unlocked in memory, return true
	if m.unlocked && m.mnemonic != "" {
		return true
	}

	// Try to load session
	return m.loadSession()
}

// GetMnemonic returns the current mnemonic (only if unlocked)
func (m *Manager) GetMnemonic() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if already unlocked
	if m.unlocked && m.mnemonic != "" {
		return m.mnemonic, nil
	}

	// Try to load session
	if !m.loadSession() {
		return "", fmt.Errorf("wallet is locked")
	}

	return m.mnemonic, nil
}

// GetEthereumKey returns the Ethereum private key
func (m *Manager) GetEthereumKey() (*ecdsa.PrivateKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if already unlocked
	if !m.unlocked {
		// Try to load session
		if !m.loadSession() {
			return nil, fmt.Errorf("wallet is locked")
		}
	}

	// Derive seed from mnemonic
	seed := bip39.NewSeed(m.mnemonic, "")

	// Choose derivation path based on network
	derivationPath := EthDerivationPath
	if m.network == NetworkTestnet {
		derivationPath = EthTestnetDerivationPath
	}

	// Derive Ethereum key
	path, err := accounts.ParseDerivationPath(derivationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse derivation path: %w", err)
	}

	key, err := deriveEthereumKey(seed, path)
	if err != nil {
		return nil, fmt.Errorf("failed to derive Ethereum key: %w", err)
	}

	return key, nil
}

// GetEthereumAddress returns the Ethereum address
func (m *Manager) GetEthereumAddress() (common.Address, error) {
	key, err := m.GetEthereumKey()
	if err != nil {
		return common.Address{}, err
	}

	publicKey := key.Public().(*ecdsa.PublicKey)
	address := ethcrypto.PubkeyToAddress(*publicKey)

	return address, nil
}

// GetBitcoinKey returns the Bitcoin private key
func (m *Manager) GetBitcoinKey() (*btcec.PrivateKey, error) {
	// Bitcoin is only supported in mainnet
	if m.network == NetworkTestnet {
		return nil, fmt.Errorf("bitcoin is not supported in testnet mode")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if already unlocked
	if !m.unlocked {
		// Try to load session
		if !m.loadSession() {
			return nil, fmt.Errorf("wallet is locked")
		}
	}

	// Derive seed from mnemonic
	seed := bip39.NewSeed(m.mnemonic, "")

	// Derive Bitcoin key
	key, err := deriveBitcoinKey(seed, BtcDerivationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to derive Bitcoin key: %w", err)
	}

	return key, nil
}

// GetBitcoinAddress returns the Bitcoin address
func (m *Manager) GetBitcoinAddress() (btcutil.Address, error) {
	// Bitcoin is only supported in mainnet
	if m.network == NetworkTestnet {
		return nil, fmt.Errorf("bitcoin is not supported in testnet mode")
	}

	key, err := m.GetBitcoinKey()
	if err != nil {
		return nil, err
	}

	publicKey := key.PubKey()

	// Use native SegWit (bech32) address format for better compatibility with modern APIs
	witnessProg := btcutil.Hash160(publicKey.SerializeCompressed())
	address, err := btcutil.NewAddressWitnessPubKeyHash(witnessProg, &chaincfg.MainNetParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create Bitcoin address: %w", err)
	}

	return address, nil
}

// GetSolanaKey returns the Solana private key
func (m *Manager) GetSolanaKey() (solana.PrivateKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check if already unlocked
	if !m.unlocked {
		// Try to load session
		if !m.loadSession() {
			return nil, fmt.Errorf("wallet is locked")
		}
	}

	// Derive seed from mnemonic
	seed := bip39.NewSeed(m.mnemonic, "")

	// Choose derivation path based on network
	derivationPath := SolDerivationPath
	if m.network == NetworkTestnet {
		derivationPath = SolTestnetDerivationPath
	}

	// Derive Solana key
	key, err := deriveSolanaKey(seed, derivationPath)
	if err != nil {
		return nil, fmt.Errorf("failed to derive Solana key: %w", err)
	}

	return key, nil
}

// GetSolanaAddress returns the Solana address
func (m *Manager) GetSolanaAddress() (solana.PublicKey, error) {
	key, err := m.GetSolanaKey()
	if err != nil {
		return solana.PublicKey{}, err
	}

	return key.PublicKey(), nil
}

// saveVault saves the vault to disk
func (m *Manager) saveVault(vault *crypto.Vault) error {
	data, err := json.Marshal(vault)
	if err != nil {
		return fmt.Errorf("failed to marshal vault: %w", err)
	}

	if err := os.WriteFile(m.vaultPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write vault file: %w", err)
	}

	return nil
}

// loadVault loads the vault from disk
func (m *Manager) loadVault() (*crypto.Vault, error) {
	data, err := os.ReadFile(m.vaultPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read vault file: %w", err)
	}

	var vault crypto.Vault
	if err := json.Unmarshal(data, &vault); err != nil {
		return nil, fmt.Errorf("failed to unmarshal vault: %w", err)
	}

	return &vault, nil
}

// VaultExists checks if a vault file exists
func (m *Manager) VaultExists() bool {
	_, err := os.Stat(m.vaultPath)
	return err == nil
}

// IsTestnet returns true if the wallet is in testnet mode
func (m *Manager) IsTestnet() bool {
	return m.network == NetworkTestnet
}

// GetCurrentNetwork returns the current network (mainnet or testnet)
func (m *Manager) GetCurrentNetwork() string {
	return m.network
}
