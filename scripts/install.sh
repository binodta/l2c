#!/usr/bin/env bash
# l2c - One-line Installer for Mac and Linux
set -e

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

REPO="binodta/l2c"
INSTALL_DIR="$HOME/.l2c"
BINARY_NAME="l2c"

echo -e "${BLUE}Installing l2c (Local to Cloud)...${NC}"

# 1. Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
IS_WSL=false

# Detect WSL (Windows Subsystem for Linux)
if [ -f /proc/version ] && grep -qi "microsoft\|wsl" /proc/version 2>/dev/null; then
    IS_WSL=true
fi

case "$OS" in
    linux)
        BINARY_FILE="l2c-linux-amd64"
        ;;
    darwin)
        if [ "$ARCH" = "arm64" ]; then
            BINARY_FILE="l2c-darwin-arm64"
        else
            BINARY_FILE="l2c-darwin-amd64"
        fi
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS ($OS).${NC}"
        echo -e "Windows users: please run this inside WSL (Windows Subsystem for Linux)."
        echo -e "Install WSL: ${BLUE}https://aka.ms/wsl${NC}"
        exit 1
        ;;
esac

if [ "$IS_WSL" = true ]; then
    echo -e "Detected: ${BLUE}Windows Subsystem for Linux (WSL)${NC}"
fi
echo -e "Detected platform: ${BLUE}$OS/$ARCH${NC} → downloading ${BINARY_FILE}"

# 2. Create install directory (clean if exists)
if [ -f "$INSTALL_DIR" ]; then
    rm -f "$INSTALL_DIR"
fi
mkdir -p "$INSTALL_DIR"

# 3. Download the binary directly from GitHub
DOWNLOAD_URL="https://raw.githubusercontent.com/$REPO/main/bin/$BINARY_FILE"
echo "Downloading from $DOWNLOAD_URL..."

if ! curl -fsSL "$DOWNLOAD_URL" -o "$INSTALL_DIR/$BINARY_NAME"; then
    echo -e "${RED}Error: Failed to download binary. Check your internet connection or try again later.${NC}"
    exit 1
fi

chmod +x "$INSTALL_DIR/$BINARY_NAME"
echo -e "${GREEN}Binary downloaded successfully.${NC}"

# 4. Add to PATH in shell profiles
PROFILES=("$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile")
UPDATED=false

for PROFILE in "${PROFILES[@]}"; do
    if [ -f "$PROFILE" ]; then
        if ! grep -q "l2c tunnel path" "$PROFILE"; then
            echo "" >> "$PROFILE"
            echo "# l2c tunnel path" >> "$PROFILE"
            echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$PROFILE"
            UPDATED=true
        fi
    fi
done

if [ "$UPDATED" = true ]; then
    echo -e "${GREEN}PATH updated in your shell profile.${NC}"
fi

# 5. Run setup
echo -e "\n${BLUE}Starting l2c setup...${NC}\n"
"$INSTALL_DIR/$BINARY_NAME" setup

# 6. Final instructions
echo -e "\n${GREEN}Installation complete!${NC}"
echo -e "------------------------------------------------"
echo -e "To use ${GREEN}l2c${NC} in this terminal, run:"
echo -e "  ${BLUE}export PATH=\"\$PATH:$INSTALL_DIR\"${NC}"
echo -e ""
echo -e "New terminals will have it automatically."
echo -e "------------------------------------------------"
