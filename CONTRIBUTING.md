# Contributing

Thanks for taking the time to contribute to glasshouse.

## Getting started

Requirements:
- Go 1.21+
- Linux (profiling is Linux-only)
- Optional for profiling: clang/llvm, bpftool, and root or CAP_BPF/CAP_SYS_ADMIN

Quick dev loop:
- Build: `go build ./...`
- Test: `go test ./...`
- Format Go code: `gofmt -w <files>`

Profiling dev setup (optional):
- Generate `ebpf/vmlinux.h`: `sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c > ebpf/vmlinux.h`
- Build eBPF objects: `./scripts/build-ebpf.sh`

## Pull requests

- Keep changes focused and well-scoped.
- Update docs when behavior or flags change.
- Add tests for new behavior when practical.
- Note any Linux or WSL assumptions in the PR description.

## Reporting issues

Please include:
- OS and kernel version
- Go version
- Whether profiling is enabled and which mode
- Any relevant logs (redact sensitive data)
