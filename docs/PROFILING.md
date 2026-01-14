# Profiling

- Profiling modes: disabled, host, guest, combined.
- Profilers attach via substrate-agnostic targets (pid, cgroup, namespaces).
- eBPF CO-RE is the reference implementation for host/guest observation; when unavailable profiling is skipped and execution proceeds.
