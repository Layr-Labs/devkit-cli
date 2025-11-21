#!/bin/bash

# devkit_install.sh - Optimized installation script for EigenLayer DevKit

# Exit immediately if a command exits with a non-zero status.
set -e
# Treat unset variables as an error.
set -u

# --- 1. CONFIGURATION ---

# Base URL for DevKit releases.
DEVKIT_BASE_URL="https://s3.amazonaws.com/eigenlayer-devkit-releases"
# File extension for the binary archive.
ARCHIVE_EXT="tar.gz"
# Final executable name after extraction.
EXECUTABLE_NAME="devkit"


# --- 2. VERSION AND PLATFORM DETECTION ---

# Fetch the latest DevKit version securely.
# Use 'https' explicitly and rely on standard cURL behavior for verification.
DEVKIT_VERSION=$(curl -fsSL "https://raw.githubusercontent.com/Layr-Labs/devkit-cli/main/VERSION")
if [ -z "${DEVKIT_VERSION}" ]; then
    echo "Error: Failed to retrieve DevKit version." >&2
    exit 1
fi

# Detect OS and ARCH and map them to standard release names.
OS_KERNEL=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH_MACHINE=$(uname -m)

case "${OS_KERNEL}" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *) echo "Error: Unsupported OS: ${OS_KERNEL}"; exit 1 ;;
esac

case "${ARCH_MACHINE}" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Error: Unsupported architecture: ${ARCH_MACHINE}"; exit 1 ;;
esac

PLATFORM="${OS}-${ARCH}"
DEVKIT_FILENAME="${EXECUTABLE_NAME}-${PLATFORM}-${DEVKIT_VERSION}.${ARCHIVE_EXT}"
DEVKIT_URL="${DEVKIT_BASE_URL}/${DEVKIT_VERSION}/${DEVKIT_FILENAME}"
TEMP_ARCHIVE=$(mktemp -t devkit_XXXXXXXXXX).${ARCHIVE_EXT}


# --- 3. INSTALLATION PATH PROMPT ---

# Default installation directory for interactive use.
DEFAULT_INSTALL_DIR="$HOME/bin"
INSTALL_DIR=""

if [[ -t 0 ]]; then
    # Interactive terminal available (tty)
    echo "Current DevKit version: ${DEVKIT_VERSION} (${PLATFORM})"
    echo "Where would you like to install DevKit?"
    echo "1) ${DEFAULT_INSTALL_DIR} (recommended, user path)"
    echo "2) /usr/local/bin (system-wide, requires sudo)"
    echo "3) Custom path"
    
    read -r -p "Enter choice (1-3) [1]: " choice
    
    # Safely convert input to an integer (using parameter expansion default value)
    CHOICE=${choice:-1}
    
    case ${CHOICE} in
        1) INSTALL_DIR="${DEFAULT_INSTALL_DIR}" ;;
        2) INSTALL_DIR="/usr/local/bin" ;;
        3)  
            read -r -p "Enter custom path: " CUSTOM_PATH
            if [[ -z "${CUSTOM_PATH}" ]]; then
                echo "Error: No path provided." >&2
                exit 1
            fi
            INSTALL_DIR="${CUSTOM_PATH}"
            ;;
        *) echo "Error: Invalid choice (${CHOICE}). Exiting." >&2; exit 1 ;;
    esac
else
    # Non-interactive (piped), use the default path.
    echo "Installing to ${DEFAULT_INSTALL_DIR} (non-interactive default)"
    INSTALL_DIR="${DEFAULT_INSTALL_DIR}"
fi


# --- 4. DOWNLOAD, VERIFICATION, AND EXTRACTION ---

echo "Attempting to download DevKit ${DEVKIT_VERSION} for ${PLATFORM} from ${DEVKIT_URL}..."

# Download the archive to a temporary file (Security Best Practice)
if ! curl -fsSL "${DEVKIT_URL}" -o "${TEMP_ARCHIVE}"; then
    echo "Error: Download failed. Check URL and connectivity." >&2
    rm -f "${TEMP_ARCHIVE}"
    exit 1
fi

# NOTE: For maximum security, a separate call to download and verify the SHA256 checksum 
# of the archive should be added here, e.g., checking against a known CHECKSUMS file.
# We skip full checksum validation here to preserve the original script's scope.
echo "Download successful. Installing..."

# Handle directory creation and extraction, using sudo only when necessary.
if [ "${INSTALL_DIR}" == "/usr/local/bin" ]; then
    echo "Sudo privileges required for installation to ${INSTALL_DIR}."
    sudo mkdir -p "${INSTALL_DIR}"
    # Use the temporary file as input to tar
    sudo tar -x -C "${INSTALL_DIR}" -f "${TEMP_ARCHIVE}"
else
    mkdir -p "${INSTALL_DIR}"
    tar -x -C "${INSTALL_DIR}" -f "${TEMP_ARCHIVE}"
fi

# Cleanup temporary archive
rm -f "${TEMP_ARCHIVE}"

echo "✅ DevKit installed successfully to ${INSTALL_DIR}/${EXECUTABLE_NAME}"


# --- 5. POST-INSTALLATION & PATH CHECK ---

# Check if the install directory is the default one and if it is in PATH
if [ "${INSTALL_DIR}" == "${DEFAULT_INSTALL_DIR}" ] && [[ ":${PATH}:" != *":${DEFAULT_INSTALL_DIR}:"* ]]; then
    echo "💡 Warning: ${DEFAULT_INSTALL_
