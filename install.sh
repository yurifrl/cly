#!/usr/bin/env bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Configuration
REPO="yurifrl/cly"
BINARY_NAME="cly"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

# Detect OS and Architecture
detect_platform() {
    OS="$(uname -s)"
    ARCH="$(uname -m)"

    case "$OS" in
        Linux*)     OS_NAME="linux" ;;
        Darwin*)    OS_NAME="darwin" ;;
        *)          echo -e "${RED}Unsupported OS: $OS${NC}" && exit 1 ;;
    esac

    case "$ARCH" in
        x86_64|amd64)   ARCH_NAME="amd64" ;;
        arm64|aarch64)  ARCH_NAME="arm64" ;;
        *)              echo -e "${RED}Unsupported architecture: $ARCH${NC}" && exit 1 ;;
    esac

    echo -e "${GREEN}Detected: ${OS_NAME}-${ARCH_NAME}${NC}"
}

# Get latest release
get_latest_version() {
    echo -e "${YELLOW}Fetching latest release...${NC}"
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')

    if [ -z "$LATEST_VERSION" ]; then
        echo -e "${RED}Failed to fetch latest version${NC}"
        exit 1
    fi

    echo -e "${GREEN}Latest version: $LATEST_VERSION${NC}"
}

# Download and install
install_binary() {
    DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}-${OS_NAME}-${ARCH_NAME}"

    echo -e "${YELLOW}Downloading from $DOWNLOAD_URL${NC}"

    # Create install directory if it doesn't exist
    mkdir -p "$INSTALL_DIR"

    # Download binary
    if command -v curl >/dev/null 2>&1; then
        curl -L -o "$INSTALL_DIR/$BINARY_NAME" "$DOWNLOAD_URL"
    elif command -v wget >/dev/null 2>&1; then
        wget -O "$INSTALL_DIR/$BINARY_NAME" "$DOWNLOAD_URL"
    else
        echo -e "${RED}Neither curl nor wget found. Please install one.${NC}"
        exit 1
    fi

    # Make executable
    chmod +x "$INSTALL_DIR/$BINARY_NAME"

    echo -e "${GREEN}✓ Installed to $INSTALL_DIR/$BINARY_NAME${NC}"
}

# Check if install dir is in PATH
check_path() {
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo -e "${YELLOW}⚠ $INSTALL_DIR is not in your PATH${NC}"
        echo -e "${YELLOW}Add this to your shell config:${NC}"
        echo -e "${YELLOW}  export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
    fi
}

# Main
main() {
    echo -e "${GREEN}CLY Installer${NC}"
    echo ""

    detect_platform
    get_latest_version
    install_binary
    check_path

    echo ""
    echo -e "${GREEN}✓ Installation complete!${NC}"
    echo -e "Run: ${GREEN}$BINARY_NAME --help${NC}"
}

main
