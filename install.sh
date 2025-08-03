#!/bin/bash

# Odyssey Wallet Installer for Linux
# This script installs the Odyssey cryptocurrency wallet on Linux systems

set -e

echo "🚀 Odyssey Installer"
echo "============================"

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is required but not installed."
    echo "Please install Go 1.18 or higher: https://golang.org/doc/install"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
GO_MAJOR=$(echo $GO_VERSION | cut -d. -f1)
GO_MINOR=$(echo $GO_VERSION | cut -d. -f2)

if [ "$GO_MAJOR" -lt 1 ] || ([ "$GO_MAJOR" -eq 1 ] && [ "$GO_MINOR" -lt 18 ]); then
    echo "❌ Error: Go 1.18+ is required (found $GO_VERSION)"
    exit 1
fi

echo "✅ Go version $GO_VERSION detected"

# Define installation paths
INSTALL_DIR="$HOME/.odyssey"
BIN_DIR="/usr/local/bin"
REPO_URL="https://github.com/chinmay1088/odyssey.git"
TEMP_DIR=$(mktemp -d)

echo "📦 Downloading Odyssey source code..."

# Clone or use current dir
if command -v git &> /dev/null; then
    echo "📥 Cloning repository..."
    if git clone --depth=1 $REPO_URL $TEMP_DIR; then
        cd $TEMP_DIR
    else
        echo "❌ Failed to clone repository"
        exit 1
    fi
else
    echo "⚠️ Git not found, using current directory..."
    TEMP_DIR=$(pwd)
    cd $TEMP_DIR

    if [ ! -f "go.mod" ]; then
        echo "❌ Odyssey source not found in current directory."
        echo "Please either install Git or place the Odyssey source code (with go.mod) here."
        exit 1
    fi
fi

echo "🔨 Building Odyssey..."
go build -o odyssey

# Create installation directory
mkdir -p $INSTALL_DIR
echo "📂 Created $INSTALL_DIR"

# Move the binary and create alias
if [ -w "$BIN_DIR" ]; then
    sudo mv odyssey "$BIN_DIR/"
    sudo ln -sf "$BIN_DIR/odyssey" "$BIN_DIR/ody"
    echo "✅ Installed odyssey to $BIN_DIR"
    echo "🔗 Created alias 'ody' -> odyssey"
else
    echo "⚠️ Cannot write to $BIN_DIR, installing to $HOME/bin instead"
    mkdir -p "$HOME/bin"
    mv odyssey "$HOME/bin/"
    ln -sf "$HOME/bin/odyssey" "$HOME/bin/ody"
    echo "✅ Installed odyssey to $HOME/bin"
    echo "🔗 Created alias 'ody' -> odyssey in $HOME/bin"
    
    # Add to PATH if needed
    if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
        echo 'export PATH="$HOME/bin:$PATH"' >> $HOME/.bashrc
        echo 'export PATH="$HOME/bin:$PATH"' >> $HOME/.profile
        echo "✅ Added $HOME/bin to PATH"
    fi
fi

# Create default config dir
mkdir -p "$INSTALL_DIR"

echo ""
echo "🎉 Odyssey installation complete!"
echo ""
echo "To get started:"
echo "  1. Initialize a new wallet: odyssey init"
echo "     or: ody init"
echo "  2. Unlock your wallet: odyssey unlock"
echo "     or: ody unlock"
echo "  3. View your addresses: odyssey address"
echo "     or: ody address"
echo ""
echo "For more information, run: odyssey --help or ody --help"
echo ""
echo "🔁 Please restart your terminal or run:"
echo "    source ~/.bashrc"
echo "    # or if you use .profile"
echo "    source ~/.profile"
echo ""
echo "Then run: odyssey or ody"
