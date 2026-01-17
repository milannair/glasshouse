#!/bin/bash
set -euo pipefail

# build-rootfs.sh - Build the guest rootfs.ext4 from Dockerfile

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
ROOTFS_DIR="$ROOT_DIR/guest/rootfs"
OUTPUT_DIR="$ROOT_DIR/assets"

mkdir -p "$OUTPUT_DIR"

echo "=== Building guest rootfs ==="

# Build Docker image
echo "Building Docker image..."
docker build -t glasshouse-rootfs -f "$ROOTFS_DIR/Dockerfile" "$ROOT_DIR"

# Create container and export filesystem
echo "Exporting filesystem..."
CONTAINER_ID=$(docker create glasshouse-rootfs)
docker export "$CONTAINER_ID" -o "$OUTPUT_DIR/rootfs.tar"
docker rm "$CONTAINER_ID" > /dev/null

# Create ext4 filesystem
echo "Creating ext4 image..."
ROOTFS_SIZE_MB=256
ROOTFS_PATH="$OUTPUT_DIR/rootfs.ext4"

dd if=/dev/zero of="$ROOTFS_PATH" bs=1M count=$ROOTFS_SIZE_MB status=progress
mkfs.ext4 -F "$ROOTFS_PATH"

# Mount and extract
echo "Extracting to ext4..."
MNT=$(mktemp -d)
sudo mount "$ROOTFS_PATH" "$MNT"
sudo tar -xf "$OUTPUT_DIR/rootfs.tar" -C "$MNT"
sudo umount "$MNT"
rmdir "$MNT"

# Cleanup
rm "$OUTPUT_DIR/rootfs.tar"

echo "=== Done ==="
echo "Rootfs created at: $ROOTFS_PATH"
echo "Size: $(du -h "$ROOTFS_PATH" | cut -f1)"
