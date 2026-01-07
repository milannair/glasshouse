FROM ubuntu:22.04

# Install dependencies
RUN apt-get update && apt-get install -y \
    clang \
    llvm \
    build-essential \
    python3 \
    git \
    make \
    gcc \
    libelf-dev \
    zlib1g-dev \
    libssl-dev \
    linux-headers-generic \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install Go 1.21+
RUN curl -L https://go.dev/dl/go1.21.5.linux-arm64.tar.gz | tar -C /usr/local -xz
ENV PATH="/usr/local/go/bin:${PATH}"

# Build and install bpftool from source
RUN git clone --recursive --depth 1 https://github.com/libbpf/bpftool.git /tmp/bpftool && \
    cd /tmp/bpftool/src && \
    make && \
    cp bpftool /usr/local/bin/ && \
    rm -rf /tmp/bpftool

# Install libbpf headers (needed for eBPF compilation)
RUN apt-get update && apt-get install -y libbpf-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace

# Copy project files
COPY . .

# Generate vmlinux.h and build eBPF programs (will be rebuilt at runtime due to volume mount)
RUN bpftool btf dump file /sys/kernel/btf/vmlinux format c > ebpf/vmlinux.h && \
    ./scripts/build-ebpf.sh && \
    go mod tidy && \
    go build -o glasshouse ./cmd/glasshouse

# Copy and set up entrypoint script
COPY docker-entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]

