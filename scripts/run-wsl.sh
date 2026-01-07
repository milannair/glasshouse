#!/bin/bash
#
# Run glasshouse on WSL2
# Usage: ./scripts/run-wsl.sh [command...]
# Example: ./scripts/run-wsl.sh python3 demo/sneaky.py
#
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log() { echo -e "${GREEN}[+]${NC} $1"; }
warn() { echo -e "${YELLOW}[!]${NC} $1"; }
error() { echo -e "${RED}[x]${NC} $1"; exit 1; }

# Check if running on WSL
if ! grep -qi microsoft /proc/version 2>/dev/null; then
    warn "This script is designed for WSL2. Proceeding anyway..."
fi

# Check if running as root for eBPF
if [ "$EUID" -ne 0 ]; then
    error "Please run with sudo: sudo $0 $*"
fi

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$SCRIPT_DIR")"
cd "$ROOT_DIR"

log "Working directory: $ROOT_DIR"

# Install dependencies if missing
install_deps() {
    log "Checking dependencies..."
    
    local packages_needed=""
    
    command -v clang >/dev/null 2>&1 || packages_needed="$packages_needed clang"
    command -v llvm-strip >/dev/null 2>&1 || packages_needed="$packages_needed llvm"
    command -v make >/dev/null 2>&1 || packages_needed="$packages_needed build-essential"
    [ -f /usr/include/libelf.h ] || packages_needed="$packages_needed libelf-dev"
    [ -f /usr/include/zlib.h ] || packages_needed="$packages_needed zlib1g-dev"
    [ -d /usr/include/bpf ] || packages_needed="$packages_needed libbpf-dev"
    command -v bpftool >/dev/null 2>&1 || packages_needed="$packages_needed linux-tools-generic"
    command -v python3 >/dev/null 2>&1 || packages_needed="$packages_needed python3"
    
    if [ -n "$packages_needed" ]; then
        log "Installing missing packages:$packages_needed"
        apt-get update
        apt-get install -y $packages_needed
    else
        log "All dependencies already installed"
    fi
}

# Install Go if missing or outdated
install_go() {
    local GO_VERSION="1.21.5"
    local GO_INSTALLED=""
    
    if command -v go >/dev/null 2>&1; then
        GO_INSTALLED=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' || echo "0.0")
        if [ "$(printf '%s\n' "1.21" "$GO_INSTALLED" | sort -V | head -n1)" = "1.21" ]; then
            log "Go $GO_INSTALLED already installed"
            return
        fi
    fi
    
    log "Installing Go $GO_VERSION..."
    
    # Detect architecture
    local ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="amd64" ;;
        aarch64) ARCH="arm64" ;;
        *) error "Unsupported architecture: $ARCH" ;;
    esac
    
    curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" | tar -C /usr/local -xz
    
    # Add to PATH for this session
    export PATH=$PATH:/usr/local/go/bin
    
    # Add to profile for future sessions
    if ! grep -q '/usr/local/go/bin' /etc/profile.d/go.sh 2>/dev/null; then
        echo 'export PATH=$PATH:/usr/local/go/bin' > /etc/profile.d/go.sh
    fi
    
    log "Go $(go version) installed"
}

# Mount tracefs if needed
mount_tracefs() {
    if [ -d /sys/kernel/tracing ] && [ -f /sys/kernel/tracing/available_events ]; then
        log "tracefs already mounted"
        return
    fi
    
    log "Mounting tracefs..."
    mount -t tracefs tracefs /sys/kernel/tracing 2>/dev/null || \
    mount -t debugfs debugfs /sys/kernel/debug 2>/dev/null || \
    warn "Could not mount tracefs/debugfs - eBPF tracepoints may not work"
}

# Generate vmlinux.h
generate_vmlinux() {
    if [ -f ebpf/vmlinux.h ]; then
        log "vmlinux.h already exists"
        return
    fi
    
    log "Generating vmlinux.h..."
    
    # Try different bpftool locations
    local BPFTOOL=""
    if command -v bpftool >/dev/null 2>&1; then
        BPFTOOL="bpftool"
    elif [ -f /usr/lib/linux-tools/*/bpftool ]; then
        BPFTOOL=$(ls /usr/lib/linux-tools/*/bpftool | head -1)
    else
        error "bpftool not found. Install with: apt install linux-tools-generic"
    fi
    
    $BPFTOOL btf dump file /sys/kernel/btf/vmlinux format c > ebpf/vmlinux.h
    log "Generated ebpf/vmlinux.h"
}

# Build eBPF programs
build_ebpf() {
    if [ -f ebpf/objects/exec.o ] && [ -f ebpf/objects/fs.o ] && [ -f ebpf/objects/net.o ]; then
        log "eBPF objects already built"
        return
    fi
    
    log "Building eBPF programs..."
    ./scripts/build-ebpf.sh
}

# Build the CLI
build_cli() {
    if [ -f glasshouse ]; then
        log "glasshouse binary already exists"
        return
    fi
    
    log "Building glasshouse CLI..."
    export PATH=$PATH:/usr/local/go/bin
    go mod tidy
    go build -o glasshouse ./cmd/glasshouse
    log "Built glasshouse"
}

# Main
main() {
    log "Setting up glasshouse on WSL2..."
    
    install_deps
    install_go
    mount_tracefs
    generate_vmlinux
    build_ebpf
    build_cli
    
    log "Setup complete!"
    echo ""
    
    # Run the command if provided, otherwise run the demo
    if [ $# -gt 0 ]; then
        log "Running: $*"
        ./glasshouse run -- "$@"
    else
        log "Running demo: python3 demo/sneaky.py"
        ./glasshouse run -- python3 demo/sneaky.py
    fi
    
    echo ""
    log "Receipt saved to receipt.json"
    cat receipt.json
}

main "$@"

