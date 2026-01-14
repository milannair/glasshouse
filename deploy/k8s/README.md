# Kubernetes Deployment

Guidance for running Glasshouse in clusters.

- Run `cmd/node-agent` as a DaemonSet to cover every node; bundle `cmd/guest-probe` for guest-side hooks.
- Mount BPF filesystem (`/sys/fs/bpf`) read-only and provide `ebpf/vmlinux.h` + compiled objects via ConfigMap or CSI volume when profiling is enabled.
- Default to sandbox-only mode (`profiling=disabled`); allow per-job opt-in to profiling via annotations that the node-agent interprets.
- Backends are registered per node; process backend is portable, VM backends require node-specific privileges.
- Receipts should be shipped off-node via your log/receipt pipeline; avoid writing to container-local disks only.
