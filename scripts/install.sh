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

# 1. Create installation directory
mkdir -p "$INSTALL_DIR"

# 2. Download the latest source code without git clone
echo "Downloading source code..."
curl -sL "https://github.com/$REPO/archive/refs/heads/main.tar.gz" | tar xz -C "$INSTALL_DIR" --strip-components=1

# 3. Enter directory and run the existing setup script
cd "$INSTALL_DIR"
chmod +x scripts/setup.sh
./scripts/setup.sh

# 4. Success message and PATH instruction
echo -e "\n${GREEN}l2c is successfully installed in $INSTALL_DIR${NC}"
echo -e "------------------------------------------------"
echo -e "To use l2c from anywhere, add it to your PATH:"
echo -e ""
echo -e "  ${BLUE}echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.zshrc${NC} (for Zsh users)"
echo -e "  OR"
echo -e "  ${BLUE}echo 'export PATH=\"\$PATH:$INSTALL_DIR\"' >> ~/.bashrc${NC} (for Bash users)"
echo -e ""
echo -e "Then restart your terminal or run ${BLUE}source ~/.zshrc${NC}"
echo -e "After that, you can start your tunnel by just running: ${GREEN}make run${NC} inside any project."
echo -e "------------------------------------------------"
