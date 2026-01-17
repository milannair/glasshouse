#!/bin/bash
set -euo pipefail

# quickstart.sh - One-command setup for Glasshouse Firecracker sandbox
#
# Usage: git clone ... && cd glasshouse && ./scripts/quickstart.sh

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

echo "[1/6] Installing system dependencies..."
sudo apt-get update -qq
sudo apt-get install -y -qq curl wget git make

# Check KVM access - add to group and use sg to continue in same session
if [[ ! -r /dev/kvm ]] || [[ ! -w /dev/kvm ]]; then
    echo "Adding user to kvm group..."
    sudo usermod -aG kvm "$USER"
    echo "Re-running script with kvm group..."
    exec sg kvm -c "$0"
fi

echo "[2/6] Installing Go..."
if ! command -v go &>/dev/null; then
    wget -q https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
    rm go1.21.5.linux-amd64.tar.gz
fi
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

echo "[3/6] Installing Firecracker..."
if ! command -v firecracker &>/dev/null; then
    FC_VERSION="v1.7.0"
    curl -fsSL -o /tmp/firecracker.tgz \
        "https://github.com/firecracker-microvm/firecracker/releases/download/${FC_VERSION}/firecracker-${FC_VERSION}-x86_64.tgz"
    sudo tar -xzf /tmp/firecracker.tgz -C /usr/local/bin --strip-components=1 \
        "release-${FC_VERSION}-x86_64/firecracker-${FC_VERSION}-x86_64"
    sudo mv "/usr/local/bin/firecracker-${FC_VERSION}-x86_64" /usr/local/bin/firecracker
    rm /tmp/firecracker.tgz
fi

echo "[4/6] Downloading kernel..."
mkdir -p assets
if [[ ! -f assets/vmlinux.bin ]]; then
    curl -fsSL -o assets/vmlinux.bin \
        https://s3.amazonaws.com/spec.ccfc.min/img/hello/kernel/hello-vmlinux.bin
fi

echo "[5/6] Building rootfs..."
if [[ ! -f assets/rootfs.ext4 ]]; then
    if command -v docker &>/dev/null; then
        ./scripts/build-rootfs.sh
    else
        ./scripts/build-minimal-rootfs.sh
    fi
fi

echo "[6/6] Building glasshouse-server..."
go build -o glasshouse-server ./cmd/glasshouse-server

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
