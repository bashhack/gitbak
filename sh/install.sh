#!/bin/bash
#
# gitbak shell script installer
# This script installs the gitbak shell script to the specified directory
# or ~/.local/bin by default.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/bashhack/gitbak/main/sh/install.sh | bash
#   
#   # Or specify an installation directory:
#   curl -fsSL https://raw.githubusercontent.com/bashhack/gitbak/main/sh/install.sh | INSTALL_DIR=/usr/local/bin bash
#
# Options:
#   INSTALL_DIR - Directory to install gitbak (default: ~/.local/bin)
#   GITBAK_VERSION - Version to install (default: latest)

set -e

# Default installation directory
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"
GITBAK_VERSION="${GITBAK_VERSION:-latest}"
GITHUB_REPO="bashhack/gitbak"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}gitbak${NC} shell script installer"
echo "==============================================="

# Check if curl or wget is available
if command -v curl >/dev/null 2>&1; then
    DOWNLOAD_CMD="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOAD_CMD="wget -qO-"
else
    echo -e "${RED}Error:${NC} Neither curl nor wget found. Please install one of them and try again."
    exit 1
fi

# Check if git is installed
if ! command -v git >/dev/null 2>&1; then
    echo -e "${YELLOW}Warning:${NC} Git is not installed. gitbak requires git to function."
    echo "Please install git before using gitbak."
fi

# Create installation directory if it doesn't exist
if [ ! -d "$INSTALL_DIR" ]; then
    echo -e "Creating installation directory: ${BLUE}$INSTALL_DIR${NC}"
    mkdir -p "$INSTALL_DIR"
fi

# Download the script
echo -e "Downloading gitbak shell script (${BLUE}$GITBAK_VERSION${NC})..."

if [ "$GITBAK_VERSION" = "latest" ]; then
    # Download directly from the main branch
    DOWNLOAD_URL="https://raw.githubusercontent.com/$GITHUB_REPO/main/sh/gitbak.sh"
    $DOWNLOAD_CMD "$DOWNLOAD_URL" > "$INSTALL_DIR/gitbak"
else
    # Download from a specific release
    DOWNLOAD_URL="https://github.com/$GITHUB_REPO/releases/download/$GITBAK_VERSION/gitbak.sh"
    $DOWNLOAD_CMD "$DOWNLOAD_URL" > "$INSTALL_DIR/gitbak"
fi

# Make the script executable
chmod +x "$INSTALL_DIR/gitbak"

echo -e "${GREEN}✓${NC} gitbak shell script installed successfully to $INSTALL_DIR/gitbak"

# Check if the installation directory is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo -e "${YELLOW}Warning:${NC} $INSTALL_DIR is not in your PATH."
    echo "To make gitbak available from anywhere, add the following line to your shell profile:"
    echo -e "  ${BLUE}export PATH=\"$INSTALL_DIR:\$PATH\"${NC}"
    
    # Suggest the appropriate profile file based on the shell
    if [ -n "$BASH_VERSION" ]; then
        echo "For bash, add it to ~/.bashrc or ~/.bash_profile"
    elif [ -n "$ZSH_VERSION" ]; then
        echo "For zsh, add it to ~/.zshrc"
    else
        echo "Add it to your shell's profile file"
    fi
fi

echo ""
echo "To use gitbak, navigate to your git repository and run:"
echo -e "  ${BLUE}gitbak${NC}"
echo ""
echo "For more information, visit: https://github.com/$GITHUB_REPO"