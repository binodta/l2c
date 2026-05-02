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

# 3. Enter directory and run the existing setup script
cd "$INSTALL_DIR"
chmod +x scripts/setup.sh
./scripts/setup.sh

# 4. Success message and PATH instruction
echo -e "\n${GREEN}l2c is successfully installed in $INSTALL_DIR${NC}"
echo -e "------------------------------------------------"

# 5. Add to PATH automatically
SHELL_PROFILE=""
if [ -f "$HOME/.zshrc" ]; then
    SHELL_PROFILE="$HOME/.zshrc"
elif [ -f "$HOME/.bashrc" ]; then
    SHELL_PROFILE="$HOME/.bashrc"
fi

if [ -n "$SHELL_PROFILE" ]; then
    if ! grep -q "$INSTALL_DIR" "$SHELL_PROFILE"; then
        echo "Adding $INSTALL_DIR to PATH in $SHELL_PROFILE..."
        echo "" >> "$SHELL_PROFILE"
        echo "# l2c tunnel path" >> "$SHELL_PROFILE"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_PROFILE"
        echo -e "${GREEN}PATH updated!${NC} Please run: ${BLUE}source $SHELL_PROFILE${NC}"
    else
        echo -e "${BLUE}PATH already configured in $SHELL_PROFILE${NC}"
    fi
else
    echo -e "${RED}Could not find .zshrc or .bashrc.${NC}"
    echo -e "Please manually add this to your PATH:"
    echo -e "export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo -e "------------------------------------------------"
echo -e "You can now start your tunnel by just running: ${GREEN}l2c setup${NC} or ${GREEN}l2c run${NC}"
echo -e "------------------------------------------------"
