# Local Deployment

## Quickstart (Firecracker Server)

On a Linux VM with nested virtualization:

```bash
./scripts/quickstart.sh
sudo ./glasshouse-server
```

Test:
```bash
curl -X POST localhost:8080/run -d '{"code": "print(2+2)"}'
```

## CLI Mode

Build and run without Firecracker:

```bash
go build -o glasshouse ./cmd/glasshouse
./glasshouse run --profile disabled -- echo hello
```

## Profiling Mode (Linux)

With eBPF profiling enabled:

```bash
sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c > ebpf/vmlinux.h
./scripts/build-ebpf.sh
sudo GLASSHOUSE_BPF_DIR=./ebpf/objects ./glasshouse run --profile host -- echo hello
```

Writes `receipt.json` with syscall/file/network activity.

## Receipts

- Firecracker mode: saved to `/var/lib/glasshouse/receipts/`
- CLI mode: written to current directory as `receipt.json`
