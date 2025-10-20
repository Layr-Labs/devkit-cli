#!/bin/bash

set -e

# DevKit version
DEVKIT_VERSION=$(curl -fsSL https://raw.githubusercontent.com/Layr-Labs/devkit-cli/main/VERSION)
DEVKIT_BASE_URL="https://s3.amazonaws.com/eigenlayer-devkit-releases"

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $OS in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *) echo "Error: Unsupported OS: $OS"; exit 1 ;;
esac

case $ARCH in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Error: Unsupported architecture: $ARCH"; exit 1 ;;
esac

PLATFORM="${OS}-${ARCH}"

# Prompt for installation directory
if [[ -t 0 ]]; then
    # Interactive terminal available
    echo "Where would you like to install DevKit?"
    echo "1) $HOME/bin (recommended)"
    echo "2) /usr/local/bin (system-wide, requires sudo)"
    echo "3) Custom path"
    read -p "Enter choice (1-3) [1]: " choice
else
    # Non-interactive (piped), use default
    echo "Installing to $HOME/bin (default for non-interactive install)"
    choice=1
fi

case ${choice:-1} in
    1) INSTALL_DIR="$HOME/bin" ;;
    2) INSTALL_DIR="/usr/local/bin" ;;
    3) 
        read -p "Enter custom path: " INSTALL_DIR
        if [[ -z "$INSTALL_DIR" ]]; then
            echo "Error: No path provided"
            exit 1
        fi
        ;;
    *) echo "Invalid choice"; exit 1 ;;
esac

# Create directory if it doesn't exist
if [[ "$INSTALL_DIR" == "/usr/local/bin" ]]; then
    sudo mkdir -p "$INSTALL_DIR"
else
    mkdir -p "$INSTALL_DIR"
fi

# Download and install
DEVKIT_URL="${DEVKIT_BASE_URL}/${DEVKIT_VERSION}/devkit-${PLATFORM}-${DEVKIT_VERSION}.tar.gz"
echo "Downloading DevKit ${DEVKIT_VERSION} for ${PLATFORM}..."

if [[ "$INSTALL_DIR" == "/usr/local/bin" ]]; then
    curl -sL "$DEVKIT_URL" | sudo tar -x -C "$INSTALL_DIR" -f -
else
    curl -sL "$DEVKIT_URL" | tar -x -C "$INSTALL_DIR" -f -
fi

echo "✅ DevKit installed to $INSTALL_DIR/devkit"

# Add to PATH if needed
if [[ "$INSTALL_DIR" == "$HOME/bin" ]] && [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
    echo "💡 Add $HOME/bin to your PATH:"
    echo "   echo 'export PATH=\"\$HOME/bin:\$PATH\"' >> ~/.$(basename $SHELL)rc"
fi

echo "🚀 Verify installation: $INSTALL_DIR/devkit --help"