#!/bin/bash
set -e

cd /workspace

# Always rebuild eBPF objects (they may be missing due to volume mount)
echo "Building eBPF programs..."
# Generate vmlinux.h
bpftool btf dump file /sys/kernel/btf/vmlinux format c > ebpf/vmlinux.h
# Build eBPF programs
./scripts/build-ebpf.sh
# Verify objects were created
echo "Checking eBPF objects..."
ls -la ebpf/objects/ || echo "Objects directory not found"
ls -la ebpf/objects/*.o 2>/dev/null || echo "No .o files found"

# Always rebuild binary (volume mount may have overwritten it with old version)
echo "Building glasshouse..."
go mod tidy
go build -o glasshouse ./cmd/glasshouse

exec ./glasshouse "$@"

