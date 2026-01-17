#!/bin/bash
set -euo pipefail

# quickstart.sh - One-command setup for Glasshouse Firecracker sandbox
#
# Usage: git clone ... && cd glasshouse && ./scripts/quickstart.sh

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

have_command() {
    command -v "$1" >/dev/null 2>&1
}

version_ge() {
    [ "$(printf '%s\n' "$2" "$1" | sort -V | head -n1)" = "$2" ]
}

ensure_path_line() {
    local line="export PATH=/usr/local/go/bin:\$PATH"
    local files=()
    if [ -f "$HOME/.bashrc" ]; then
        files+=("$HOME/.bashrc")
    fi
    if [ -f "$HOME/.zshrc" ]; then
        files+=("$HOME/.zshrc")
    fi
    if [ ${#files[@]} -eq 0 ]; then
        files+=("$HOME/.profile")
    fi
    for f in "${files[@]}"; do
        if ! grep -qxF "$line" "$f" 2>/dev/null; then
            echo "$line" >> "$f"
        fi
    done
}

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
sudo apt-get install -y -qq ca-certificates curl wget git make e2fsprogs util-linux tar gzip xz-utils file

# Check KVM access - add to group and use sg to continue in same session
if [[ ! -r /dev/kvm ]] || [[ ! -w /dev/kvm ]]; then
    echo "Adding user to kvm group..."
    if ! getent group kvm >/dev/null; then
        sudo groupadd kvm
    fi
    sudo usermod -aG kvm "$USER"
    if have_command sg; then
        echo "Re-running script with kvm group..."
        sg kvm -c "$0"
        exit 0
    fi
    echo "Log out/in to pick up group membership, then re-run this script."
    exit 1
fi

echo "[2/6] Installing Go..."
GO_VERSION="1.21.5"
GO_MIN_VERSION="1.21.0"
NEED_GO="true"
if have_command go; then
    INSTALLED_GO="$(go version | awk '{print $3}' | sed 's/^go//')"
    if version_ge "$INSTALLED_GO" "$GO_MIN_VERSION"; then
        NEED_GO="false"
    fi
fi
if [[ "$NEED_GO" == "true" ]]; then
    wget -q "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
    rm "go${GO_VERSION}.linux-amd64.tar.gz"
fi
export PATH=/usr/local/go/bin:$PATH
ensure_path_line

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
    if have_command docker && docker info >/dev/null 2>&1; then
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
