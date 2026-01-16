# Docker Deployment

This directory holds Docker tooling to run Glasshouse on a single host.

- Build steps (host):
  1) `go build -o glasshouse ./cmd/glasshouse`
  2) (optional) `./scripts/build-ebpf.sh` to populate `ebpf/objects` and `ebpf/vmlinux.h`
  3) Package binaries and `ebpf/*` into an image; keep the image minimal (scratch/distroless acceptable for sandbox-only).
- Run (sandbox-only):
  - `docker run --rm -v $(pwd):/work gh/glasshouse ./glasshouse run --profile disabled -- ls`
- Run with profiling (Linux host with BPF privileges):
  - `docker run --rm --pid=host --privileged -v /sys/fs/bpf:/sys/fs/bpf -v $(pwd)/ebpf:/app/ebpf gh/glasshouse ./glasshouse run --profile host -- cmd`
- Receipts: written to the container CWD (`/work` in examples); mount a host path with `-v` to persist receipts.
- Backends: use `process` inside containers. VM backends need nested virtualization and are out of scope for this profile.
- Hardening tips: drop ambient capabilities when profiling is disabled; use read-only rootfs and limited volumes for sandbox-only runs.
