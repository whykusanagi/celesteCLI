#!/bin/bash

# CelesteCLI Installation Script
# This script installs celestecli to your PATH

set -e

echo "🚀 Installing CelesteCLI..."

# Check if celestecli binary exists
if [ ! -f "./celestecli" ]; then
    echo "❌ Error: celestecli binary not found in current directory"
    echo "Please run this script from the celesteCLI directory"
    exit 1
fi

# Check if personality.yml exists
if [ ! -f "./personality.yml" ]; then
    echo "⚠️  Warning: personality.yml not found in current directory"
    echo "The CLI will work but personality features may be limited"
else
    # Create ~/.celeste config directory if it doesn't exist
    mkdir -p ~/.celeste
    
    # Copy personality.yml to config directory
    cp personality.yml ~/.celeste/
    echo "✅ Copied personality.yml to ~/.celeste/"
fi

# Create ~/.local/bin if it doesn't exist
mkdir -p ~/.local/bin

# Copy celestecli to ~/.local/bin
cp celestecli ~/.local/bin/

# Make it executable
chmod +x ~/.local/bin/celestecli

# Check if ~/.local/bin is in PATH
if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    echo "⚠️  Warning: ~/.local/bin is not in your PATH"
    echo "Add this line to your ~/.bashrc, ~/.zshrc, or ~/.profile:"
    echo "export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
    echo "Then run: source ~/.bashrc (or ~/.zshrc)"
fi

# Test installation
if command -v celestecli &> /dev/null; then
    echo "✅ CelesteCLI installed successfully!"
    echo "📍 Location: $(which celestecli)"
    echo ""
    echo "🎯 Test it with: celestecli --help"
else
    echo "❌ Installation failed or PATH not updated"
    echo "Try running: source ~/.bashrc (or ~/.zshrc)"
fi
