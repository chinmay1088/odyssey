# Odyssey

A secure command-line cryptocurrency wallet supporting Ethereum, Bitcoin, and Solana networks.

## Overview

Odyssey is a terminal-based deterministic & minimalistic wallet system for managing multiple cryptocurrency networks. The system allows you to generate and securely store private keys derived from a single entropy source. Transactions are constructed locally, signed deterministically, and submitted to public blockchain nodes.

All sensitive data remains encrypted at rest and is processed entirely offline during execution. The design assumes a hostile external environment and minimizes trust in network infrastructure.

## Features

- **Multi-chain support**: Manage Ethereum, Bitcoin, and Solana from a single wallet
- **BIP-39 mnemonic generation**: Industry-standard seed phrase creation and management
- **BIP-44 hierarchical deterministic wallets**: Proper derivation paths for all supported chains
- **AES-256-GCM encrypted vault storage**: Military-grade encryption for your keys
- **Real-time balance checking**: Query current balances across all chains
- **Transaction signing and broadcasting**: Full transaction lifecycle management
- **Mainnet and Testnet support**: Switch between networks with a simple command
- **Fiat conversion support**: Work with USD values alongside crypto amounts
- **Recovery phrase management**: Backup and restore functionality

## Installation

### Windows

```powershell
# Download and run the installer
iwr -useb https://raw.githubusercontent.com/chinmay1088/odyssey/main/install.ps1 | iex
```

### Linux

```bash
# Download and run the installer
curl -O https://raw.githubusercontent.com/chinmay1088/odyssey/main/install.sh
chmod +x install.sh
./install.sh
```

### macOS

```bash
# Install via Homebrew
brew tap chinmay1088/odyssey
brew install odyssey
```

### From Source

```bash
git clone https://github.com/chinmay1088/odyssey.git
cd odyssey
go build -o odyssey
sudo mv odyssey /usr/local/bin/
```

## Usage

### Quick Start

```bash
# Create a new wallet
odyssey init

# Unlock your wallet
odyssey unlock

# View your addresses
odyssey address

# Check balances
odyssey balance
odyssey balance --usd  # Show in USD

# Send cryptocurrency
odyssey pay eth 0.1 0x742d35Cc6634C0532925a3b8D4C9db96C4b4d8b6

# View transaction history
odyssey transactions
odyssey transactions eth --page 2  # Paginated Ethereum transactions
```

### Available Commands

| Command | Description | Example |
|---------|-------------|---------|
| `init` | Create new wallet | `odyssey init` |
| `unlock` | Unlock existing wallet | `odyssey unlock` |
| `address` | Show wallet addresses | `odyssey address` |
| `balance` | Check balances | `odyssey balance --usd` |
| `pay` | Send cryptocurrency | `odyssey pay eth 0.1 0x123...` |
| `transactions` | View transaction history | `odyssey transactions --page 2` |
| `network` | Switch networks | `odyssey network testnet` |
| `recovery` | Export recovery phrase | `odyssey recovery` |
| `buy` | Buy cryptocurrency via MoonPay | `odyssey buy` |
| `update` | Update to latest version | `odyssey update` |

## Architecture

### Key Derivation

Wallet state is defined by a single entropy source — a 128- to 256-bit random seed generated at first use. A BIP-39 mnemonic phrase is derived from this entropy and optionally encrypted with a user-chosen passphrase.

Key derivation follows hierarchical deterministic wallet standards (BIP-32 and BIP-44) with chain-specific derivation paths:

- Ethereum: `m/44'/60'/0'/0/0` (mainnet) / `m/44'/1'/0'/0/0` (testnet)
- Bitcoin: `m/44'/0'/0'/0/0` (mainnet only)
- Solana: `m/44'/501'/0'/0'` (mainnet) / `m/44'/501'/0'/1'` (testnet)

### Security Model

The system assumes the following:

- Host device is secure at runtime (trusted input and memory)
- Vault file may be stored on an insecure filesystem
- Network infrastructure may be compromised or malicious
- Users are capable of maintaining strong passwords

Under this model, the system protects against:

- Key extraction from disk
- Unauthorized transaction signing
- Network spoofing during broadcast
- Passive leakage of metadata

The system does not protect against:

- Memory scraping by active malware
- Insecure copy-paste behavior (e.g., clipboard hijacks)

### Encrypted Storage

The user's mnemonic and key material are stored in a local encrypted container using:

- AES-256 in Galois/Counter Mode (GCM)
- scrypt key derivation (N=2¹⁵, r=8, p=1)
- 16-byte salt and 12-byte nonce

## Network Communication

The wallet communicates with public blockchain nodes via HTTPS using authenticated APIs:

- Ethereum: JSON-RPC (via public nodes)
- Bitcoin: REST API (e.g., Blockstream)
- Solana: JSON-RPC (e.g., `api.mainnet-beta.solana.com`)

Queries are read-only unless a transaction is explicitly submitted. The wallet does not expose or transmit private keys.

## Contributing

Contributions are welcome. Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.