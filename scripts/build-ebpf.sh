#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
OBJ_DIR="$ROOT_DIR/ebpf/objects"
VMLINUX="$ROOT_DIR/ebpf/vmlinux.h"
COMMON_DIR="$ROOT_DIR/ebpf/common"
HOST_DIR="$ROOT_DIR/ebpf/host"

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

COMMON_FLAGS=(
  -g -O2 -target bpf
  -D__TARGET_ARCH_${TARGET_ARCH}
  -I"$COMMON_DIR"
  -I"$HOST_DIR"
  -I"$ROOT_DIR/ebpf"
  -I/usr/include/${ARCH}-linux-gnu
)

clang "${COMMON_FLAGS[@]}" -c "$HOST_DIR/exec.c" -o "$OBJ_DIR/exec.o"
clang "${COMMON_FLAGS[@]}" -c "$HOST_DIR/exec_argv.c" -o "$OBJ_DIR/exec-argv.o"
clang "${COMMON_FLAGS[@]}" -c "$HOST_DIR/fs.c" -o "$OBJ_DIR/fs.o"
clang "${COMMON_FLAGS[@]}" -c "$HOST_DIR/net.c" -o "$OBJ_DIR/net.o"

echo "Built eBPF objects in $OBJ_DIR"
