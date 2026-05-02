#!/bin/bash

# l2c - One-line Installer for Mac and Linux
set -e

# Colors
BLUE='\033[0;34m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

INSTALL_DIR="$HOME/.l2c"
REPO="binodta/l2c"

echo -e "${BLUE}Installing l2c (Local to Cloud)...${NC}"

# 1. Create installation directory (replacing if exists)
if [ -d "$INSTALL_DIR" ] || [ -f "$INSTALL_DIR" ]; then
    echo "Replacing existing installation in $INSTALL_DIR..."
    rm -rf "$INSTALL_DIR"
fi
mkdir -p "$INSTALL_DIR"

# 2. Download the latest source code
echo "Downloading source code..."
curl -sL "https://github.com/$REPO/archive/refs/heads/main.tar.gz" | tar xz -C "$INSTALL_DIR" --strip-components=1

# 3. Platform Detection and Binary Setup
cd "$INSTALL_DIR"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$OS" in
    linux)
        BINARY="l2c-linux-amd64"
        ;;
    darwin)
        if [ "$ARCH" == "arm64" ]; then
            BINARY="l2c-darwin-arm64"
        else
            BINARY="l2c-darwin-amd64"
        fi
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS ($OS).${NC}"
        exit 1
        ;;
esac

if [ -f "bin/$BINARY" ]; then
    echo -e "Installing binary for ${BLUE}$OS-$ARCH${NC}..."
    cp "bin/$BINARY" ./l2c
    chmod +x l2c
    ./l2c setup
else
    echo -e "${RED}Error: Binary $BINARY not found in the package.${NC}"
    exit 1
fi

# 4. Success message and PATH instruction
echo -e "\n${GREEN}l2c is successfully installed in $INSTALL_DIR${NC}"
echo -e "------------------------------------------------"

# 5. Add to PATH automatically
PROFILES=("$HOME/.zshrc" "$HOME/.bashrc" "$HOME/.profile")
UPDATED=false

for PROFILE in "${PROFILES[@]}"; do
    if [ -f "$PROFILE" ]; then
        if ! grep -q "$INSTALL_DIR" "$PROFILE"; then
            echo "Adding $INSTALL_DIR to PATH in $PROFILE..."
            echo "" >> "$PROFILE"
            echo "# l2c tunnel path" >> "$PROFILE"
            echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$PROFILE"
            UPDATED=true
        fi
    fi
done

if [ "$UPDATED" = true ]; then
    echo -e "\n${GREEN}Success! PATH updated.${NC}"
    echo -e "${YELLOW}Please restart your terminal or run:${NC}"
    echo -e "${BLUE}source ~/.bashrc && source ~/.profile${NC}"
else
    echo -e "\n${BLUE}PATH already configured in your profile files.${NC}"
fi

echo -e "\n${GREEN}Installation Complete!${NC}"
echo -e "------------------------------------------------"
echo -e "1. Run: ${BLUE}source $SHELL_PROFILE${NC}"
echo -e "2. Start: ${GREEN}l2c run${NC}"
echo -e "------------------------------------------------"
