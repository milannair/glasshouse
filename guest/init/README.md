# Guest Init

Boot-time hooks for guest images used by VM-style backends.

- Provision the guest user, SSH keys (if any), and minimal tooling required by `cmd/guest-probe`.
- Ensure BPF filesystem and proc/sys mounts are available when guest-side profiling is enabled.
- Keep this minimal and deterministic; policy enforcement lives on the host/control plane.
