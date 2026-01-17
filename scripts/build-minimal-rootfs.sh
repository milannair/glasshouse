#!/bin/bash
set -euo pipefail

# build-minimal-rootfs.sh - Create a minimal rootfs without Docker
# Downloads Alpine mini rootfs and adds Python

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
OUTPUT_DIR="$ROOT_DIR/assets"

mkdir -p "$OUTPUT_DIR"

echo "Creating Alpine rootfs with Python..."

ROOTFS_PATH="$OUTPUT_DIR/rootfs.ext4"
ROOTFS_SIZE_MB=512

# Create ext4 image
dd if=/dev/zero of="$ROOTFS_PATH" bs=1M count=$ROOTFS_SIZE_MB status=progress
mkfs.ext4 -F "$ROOTFS_PATH"

# Mount
MNT=$(mktemp -d)
sudo mount "$ROOTFS_PATH" "$MNT"

# Download Alpine mini rootfs
ALPINE_VERSION="3.19"
ALPINE_URL="https://dl-cdn.alpinelinux.org/alpine/v${ALPINE_VERSION}/releases/x86_64/alpine-minirootfs-${ALPINE_VERSION}.0-x86_64.tar.gz"
echo "Downloading Alpine ${ALPINE_VERSION} mini rootfs..."
curl -fsSL "$ALPINE_URL" | sudo tar -xz -C "$MNT"

# Configure Alpine for chroot
sudo cp /etc/resolv.conf "$MNT/etc/resolv.conf"

# Install Python in chroot
echo "Installing Python..."
sudo chroot "$MNT" /bin/sh -c "apk add --no-cache python3"

# Create workspace directory
sudo mkdir -p "$MNT/workspace"

# Remove Alpine's init system - we use our own
sudo rm -f "$MNT/sbin/init" 2>/dev/null || true
sudo rm -rf "$MNT/etc/init.d" 2>/dev/null || true
sudo rm -rf "$MNT/etc/inittab" 2>/dev/null || true

# Build guest init
echo "Building guest init..."
cd "$ROOT_DIR"
INIT_TMP=$(mktemp)
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o "$INIT_TMP" ./guest/init/
sudo cp "$INIT_TMP" "$MNT/sbin/init"
sudo chmod +x "$MNT/sbin/init"
rm "$INIT_TMP"

# Verify init is our binary
echo "Verifying init binary..."
file "$MNT/sbin/init" || sudo file "$MNT/sbin/init"

# Cleanup
sudo rm -f "$MNT/etc/resolv.conf"
sudo umount "$MNT"
rmdir "$MNT"

echo "Rootfs created at: $ROOTFS_PATH"
echo "Size: $(du -h "$ROOTFS_PATH" | cut -f1)"
