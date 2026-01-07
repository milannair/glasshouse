#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
OBJ_DIR="$ROOT_DIR/ebpf/objects"
VMLINUX="$ROOT_DIR/ebpf/vmlinux.h"

if [[ ! -f "$VMLINUX" ]]; then
  echo "Missing $VMLINUX. Generate it with:" >&2
  echo "  sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c > $VMLINUX" >&2
  exit 1
fi

mkdir -p "$OBJ_DIR"

ARCH=$(uname -m)
TARGET_ARCH="x86"
if [[ "$ARCH" == "aarch64" ]]; then
  TARGET_ARCH="arm64"
fi

# Find kernel headers directory
KERNEL_HEADERS="/usr/src/linux-headers-$(uname -r)"
if [[ ! -d "$KERNEL_HEADERS" ]]; then
  # Fallback: try to find any kernel headers
  KERNEL_HEADERS=$(find /usr/src -maxdepth 1 -type d -name "linux-headers-*" | head -1)
fi

COMMON_FLAGS=(
  -g -O2 -target bpf
  -D__TARGET_ARCH_${TARGET_ARCH}
  -I"$ROOT_DIR/ebpf"
)

# Add kernel headers if found
if [[ -d "$KERNEL_HEADERS" ]]; then
  # Use -isystem for system headers to handle asm includes properly
  COMMON_FLAGS+=(-isystem "$KERNEL_HEADERS/include")
  COMMON_FLAGS+=(-isystem "$KERNEL_HEADERS/include/uapi")
  COMMON_FLAGS+=(-isystem "$KERNEL_HEADERS/include/generated/uapi")
  COMMON_FLAGS+=(-isystem "$KERNEL_HEADERS/arch/${TARGET_ARCH}/include")
  COMMON_FLAGS+=(-isystem "$KERNEL_HEADERS/arch/${TARGET_ARCH}/include/uapi")
  COMMON_FLAGS+=(-isystem "$KERNEL_HEADERS/arch/${TARGET_ARCH}/include/generated/uapi")
  COMMON_FLAGS+=(-isystem "$KERNEL_HEADERS/include/asm-generic")
fi

clang "${COMMON_FLAGS[@]}" -c "$ROOT_DIR/ebpf/exec.c" -o "$OBJ_DIR/exec.o"
clang "${COMMON_FLAGS[@]}" -c "$ROOT_DIR/ebpf/fs.c" -o "$OBJ_DIR/fs.o"
clang "${COMMON_FLAGS[@]}" -c "$ROOT_DIR/ebpf/net.c" -o "$OBJ_DIR/net.o"

echo "Built eBPF objects in $OBJ_DIR"
