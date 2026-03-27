#!/bin/bash
set -e

# Ensure running as root
if [ "$EUID" -ne 0 ]; then 
  echo "Please run as root"
  exit 1
fi

echo "[1/4] Downloading nptx Core (Compiled via GitHub Actions)..."
REPO="bolandi-org/ntpx"
LATEST_URL=$(curl -s https://api.github.com/repos/$REPO/releases/latest | grep "browser_download_url.*nptx_core_linux_amd64" | cut -d '"' -f 4)

if [ -z "$LATEST_URL" ]; then
    echo "⚠️  Could not fetch the latest release from GitHub (Is it a public repo or are there no releases yet?)."
    echo "Falling back to local build..."
    
    if ! command -v go &> /dev/null; then
      echo "  -> Installing Go 1.22..."
      wget -qO- https://go.dev/dl/go1.22.2.linux-amd64.tar.gz | tar -C /usr/local -xzf -
      export PATH=$PATH:/usr/local/go/bin
    fi
    make build-linux
    cp build/nptx_core_linux_amd64 /usr/local/bin/nptx_core
else
    echo "  -> Found latest release: $LATEST_URL"
    wget -q --show-progress "$LATEST_URL" -O /usr/local/bin/nptx_core
fi

echo "[2/4] Setting execution permissions..."
chmod +x /usr/local/bin/nptx_core

echo "[3/4] Setting up config directory..."
mkdir -p /etc/nptx
if [ ! -f /etc/nptx/config.json ]; then
    cp config.example.json /etc/nptx/config.json
    echo "  -> Created default config at /etc/nptx/config.json. PLEASE EDIT IT BEFORE STARTING."
else
    echo "  -> Config already exists at /etc/nptx/config.json, skipping overwrite."
fi

echo "[4/4] Installing Systemd Service..."
cp nptx.service /etc/systemd/system/nptx.service
systemctl daemon-reload
systemctl enable nptx.service
# systemctl restart nptx.service # Commented out to let user edit config first

echo "=========================================="
echo "Installation complete!"
echo "1. Edit your configuration file: nano /etc/nptx/config.json"
echo "2. Start the service: systemctl start nptx.service"
echo "3. Check status: systemctl status nptx.service"
echo "=========================================="
