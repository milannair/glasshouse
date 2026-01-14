# Profiling

- Profiling modes: disabled, host, guest, combined.
- Profilers attach via substrate-agnostic targets (pid, cgroup, namespaces).
- eBPF CO-RE is the reference implementation for host/guest observation; when unavailable profiling is skipped and execution proceeds.
- Profiling is opt-in per execution; default is `disabled` to maximize portability.
- Events are structured and feed `core/receipt.Aggregator`; receipts are only emitted when profiling is enabled and attached.
- CO-RE expectations: use `vmlinux.h`, `BPF_CORE_READ*`, and avoid kernel-version-specific offsets.
- Distros: Ubuntu LTS, Debian, Amazon Linux, Fedora/RHEL-like are in scope; if BTF is missing, profiling disables itself automatically.
