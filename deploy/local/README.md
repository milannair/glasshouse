# Local Deployment

Use this folder for developer setups on a laptop or single VM.

- Build the CLI: `go build -o glasshouse ./cmd/glasshouse`.
- Optional profiling: `./scripts/build-ebpf.sh` (requires `ebpf/vmlinux.h` from `bpftool btf dump ...`) and `sudo` to run.
- Sandbox-only run (portable): `./glasshouse run --profile disabled -- echo hello`.
- Profiling run (Linux): `sudo GLASSHOUSE_BPF_DIR=./ebpf/objects ./glasshouse run --profile host -- echo hello`.
- Node-agent local: `go run ./cmd/node-agent --backend process --profile disabled -- ls`; set `GLASSHOUSE_BPF_DIR` when enabling profiling.
- Receipts: written to the CWD; use `--profile disabled` to skip receipt emission for quick smoke tests.
