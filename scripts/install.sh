#!/bin/bash

# notes-cli Installer (Mac/Linux)
# This script builds and installs the notes-cli tool globally.

set -e

if ! command -v go &> /dev/null
then
    echo "❌ Error: 'go' command not found. Please install Go (https://go.dev/dl/) first."
    exit 1
fi

echo "🚀 Building notes-cli..."
go build -o notes-cli

echo "📦 Installing binary to /usr/local/bin..."
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS
  sudo cp notes-cli /usr/local/bin/notes-cli
else
  # Linux
  sudo cp notes-cli /usr/local/bin/notes-cli
fi

echo "⚙️  Initializing configuration..."
HOME_DIR=$(eval echo ~$USER)
if [ ! -f "$HOME_DIR/.notes-cli.json" ]; then
  echo '{"courses": {}}' > "$HOME_DIR/.notes-cli.json"
fi

echo "✅ Installation complete!"
echo "You can now run 'notes-cli' from anywhere."
echo "Launch the interactive wizard by running: notes-cli"
