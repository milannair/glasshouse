# Guest Transport

Host/guest communication contracts.

- The `transport` package provides a `Transport` interface; `Loopback` emits JSON lines to stdout for local testing.
- Backends running in VMs can swap in shared-memory or vsock transports without changing the guest probe.
- Messages are structured events, not logs, so control planes can parse them deterministically.
