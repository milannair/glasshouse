#!/bin/bash
set -euo pipefail

# quickstart.sh - One-command setup for Glasshouse Firecracker sandbox
#
# Usage: curl -fsSL https://raw.githubusercontent.com/USER/glasshouse/main/scripts/quickstart.sh | bash

echo "=== Glasshouse Quickstart ==="
echo ""

# Check architecture
ARCH="$(uname -m)"
if [[ "$ARCH" != "x86_64" ]]; then
    echo "Error: x86_64 required (got $ARCH)"
    exit 1
fi

# Check for KVM
if [[ ! -e /dev/kvm ]]; then
    echo "Error: /dev/kvm not found"
    echo "Enable nested virtualization on your VM or run on bare metal"
    exit 1
fi

# Check KVM access
if [[ ! -r /dev/kvm ]] || [[ ! -w /dev/kvm ]]; then
    echo "Adding user to kvm group..."
    sudo usermod -aG kvm "$USER"
    echo "Please log out and back in, then re-run this script"
    exit 1
fi

echo "[1/5] Checking dependencies..."

# Install Go if needed
if ! command -v go &>/dev/null; then
    echo "Installing Go..."
    wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
    rm go1.21.5.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
fi

# Install Firecracker if needed
if ! command -v firecracker &>/dev/null; then
    echo "Installing Firecracker v1.7.0..."
    FC_VERSION="v1.7.0"
    curl -fsSL -o /tmp/firecracker.tgz \
        "https://github.com/firecracker-microvm/firecracker/releases/download/${FC_VERSION}/firecracker-${FC_VERSION}-x86_64.tgz"
    sudo tar -xzf /tmp/firecracker.tgz -C /usr/local/bin --strip-components=1 \
        "release-${FC_VERSION}-x86_64/firecracker-${FC_VERSION}-x86_64"
    sudo mv "/usr/local/bin/firecracker-${FC_VERSION}-x86_64" /usr/local/bin/firecracker
    rm /tmp/firecracker.tgz
fi

echo "[2/5] Downloading kernel..."
mkdir -p assets
if [[ ! -f assets/vmlinux.bin ]]; then
    curl -fsSL -o assets/vmlinux.bin \
        https://s3.amazonaws.com/spec.ccfc.min/img/hello/kernel/hello-vmlinux.bin
fi

echo "[3/5] Building rootfs..."
if [[ ! -f assets/rootfs.ext4 ]]; then
    if command -v docker &>/dev/null; then
        ./scripts/build-rootfs.sh
    else
        echo "Docker not found - downloading pre-built rootfs..."
        # Fallback: create minimal busybox rootfs
        ./scripts/build-minimal-rootfs.sh
    fi
fi

echo "[4/5] Building glasshouse-server..."
go build -o glasshouse-server ./cmd/glasshouse-server

echo "[5/5] Creating receipt directory..."
sudo mkdir -p /var/lib/glasshouse/receipts
sudo chown "$USER:$USER" /var/lib/glasshouse/receipts

echo ""
echo "=== Setup Complete ==="
echo ""
echo "Start the server:"
echo "  sudo ./glasshouse-server"
echo ""
echo "Test execution:"
echo "  curl -X POST localhost:8080/run -d '{\"code\": \"print(2+2)\"}'"
echo ""
echo "View receipts:"
echo "  ls /var/lib/glasshouse/receipts/"
echo ""
