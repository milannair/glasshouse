#!/bin/bash
set -euo pipefail

# build-minimal-rootfs.sh - Create a minimal rootfs without Docker
# Fallback for environments without Docker

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="$ROOT_DIR/assets"

mkdir -p "$OUTPUT_DIR"

echo "Creating minimal rootfs with busybox and Python..."

ROOTFS_PATH="$OUTPUT_DIR/rootfs.ext4"
ROOTFS_SIZE_MB=256

# Create ext4 image
dd if=/dev/zero of="$ROOTFS_PATH" bs=1M count=$ROOTFS_SIZE_MB status=progress
mkfs.ext4 -F "$ROOTFS_PATH"

# Mount and populate
MNT=$(mktemp -d)
sudo mount "$ROOTFS_PATH" "$MNT"

# Create basic structure
sudo mkdir -p "$MNT"/{bin,sbin,proc,sys,dev,tmp,workspace,usr/bin,usr/lib}

# Copy busybox
sudo cp /bin/busybox "$MNT/bin/busybox"
sudo chmod +x "$MNT/bin/busybox"

# Create symlinks for common utilities
for cmd in sh cat echo ls mkdir mount poweroff; do
    sudo ln -sf /bin/busybox "$MNT/bin/$cmd"
done
sudo ln -sf /bin/busybox "$MNT/sbin/poweroff"

# Check for Python and copy it
if command -v python3 &>/dev/null; then
    PYTHON_BIN=$(which python3)
    sudo cp "$PYTHON_BIN" "$MNT/usr/bin/python3"
    sudo chmod +x "$MNT/usr/bin/python3"
    
    # Copy Python libraries (basic)
    PYTHON_LIB=$(python3 -c "import sys; print(sys.prefix)")/lib
    if [[ -d "$PYTHON_LIB" ]]; then
        sudo cp -r "$PYTHON_LIB"/python* "$MNT/usr/lib/" 2>/dev/null || true
    fi
fi

# Build guest init
echo "Building guest init..."
cd "$ROOT_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$MNT/sbin/init" ./guest/init/

sudo umount "$MNT"
rmdir "$MNT"

echo "Minimal rootfs created at: $ROOTFS_PATH"
