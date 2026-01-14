# Docker Deployment

This directory holds Docker tooling to run Glasshouse on a single host.

- Images should include the CLI (`cmd/glasshouse`) and optional `cmd/node-agent`.
- Containers must mount `ebpf/vmlinux.h` and `ebpf/objects` if profiling is desired; sandbox-only mode works without them.
- Use the process backend by default; VM backends require host privileges and are not wired here yet.
- Receipts are written inside the container; mount a host path if you need to persist them.
