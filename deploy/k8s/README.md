# Kubernetes Deployment

Guidance for running Glasshouse in clusters.

- Topology: run `cmd/node-agent` as a DaemonSet; use a ServiceAccount with only the RBAC needed for watch/list ConfigMaps or CRDs describing executions.
- Assets: mount `/sys/fs/bpf` read-only; mount `ebpf/vmlinux.h` and `ebpf/objects` via ConfigMap/CSI when profiling is enabled. Sandbox-only mode requires none of these mounts.
- Execution requests: annotate Pods/Jobs with profiling mode and backend preference (e.g., `glasshouse.io/profile=host`, `glasshouse.io/backend=process`); node-agent should read these and choose ExecutionSpec accordingly.
- Backends: process backend is default and portable; VM backends (Firecracker/Kata) require node features (KVM, nested virt) and are registered per node in `node/registry`.
- Receipts: stream to a log sink (e.g., stdout + sidecar, or direct upload) instead of writing to container FS. Include provenance and version in logs for downstream policy/training consumers.
- Hardening: when profiling is disabled, drop CAPs; when enabled, scope capabilities to BPF/TRACE only. Prefer hostPID=false except when profiling needs host PID visibility.
