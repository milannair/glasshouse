# Local Deployment

Use this folder for developer setups on a laptop or single VM.

- Build the CLI: `go build -o glasshouse ./cmd/glasshouse`.
- Build eBPF objects (optional): `./scripts/build-ebpf.sh` with `ebpf/vmlinux.h` present.
- Run sandbox-only mode anywhere; profiling requires Linux with BTF and `sudo`.
- Store receipts in the working directory or configure the node-agent to stream them to your preferred sink.
