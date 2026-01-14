# Learn Glasshouse

This walkthrough explains the repo structure, what each component does, and how they interact. The branch is `feature/modularity`; profiling is optional and fail-open.

## Top-Level Binaries (cmd/)
- `cmd/glasshouse`: CLI for running a single command. Parses flags (`--guest`, `--profile disabled|host|guest|combined`), builds an `ExecutionSpec`, selects a profiler (noop by default, eBPF when enabled), and runs the `execution.Engine`. Writes `receipt.json` only when profiling is on.
- `cmd/node-agent`: Node-side orchestrator. Flags for backend, profiling, and concurrency. Uses `node/manager` + registry to run requests, applies profiling selection, and writes receipts when profiling is enabled.
- `cmd/guest-probe`: Guest-side probe scaffold. Emits structured JSON events (heartbeat/info) via the guest transport interface; ready for vsock/shared-memory swaps later.

## Core Semantics (core/)
- `core/execution`: `ExecutionBackend` contract and `Engine` orchestration. Engine sequence: validate spec → backend Prepare/Start → optional profiling attach → Wait → aggregate → Cleanup; profiling errors surface but do not block execution. Optional interfaces expose stdout/stderr, extra errors, process state, metadata overrides.
- `core/profiling`: Mode/Target/Capabilities types. Controllers start profiling sessions that emit portable events. `core/profiling/ebpf` wraps the legacy audit collector on Linux; stub elsewhere. `core/profiling/noop` is the default when profiling is disabled.
- `core/receipt`: Receipt grammar (v0.3.0), aggregator for profiling events, metadata enrichment (execution ID, provenance, artifacts, environment), and redaction support (`MaskPaths`).
- `core/policy`: Declarative, deterministic policy evaluator over receipts. Enforcement is intentionally separate.
- `core/training`: Derives coarse training signals (feature map) from receipts.
- `core/version`: Version constants for receipts and core semantics.

## Execution Backends (backend/)
- `backend/process`: Default backend; runs host process with optional guest prep (mounts/proc/sysfs/bpf). Reports profiling identity (host PID), captures stdout/stderr, records extra shutdown signal info, and exposes process state for resource reporting.
- `backend/firecracker`: Stub that validates config, reports isolation `vm`, and returns not-implemented from Start; ready to be fleshed out with VM lifecycle.
- `backend/kata`: Stub backend with VM isolation metadata; Start/Wait/Kill/Cleanup are placeholders.
- `backend/fake`: Configurable test backend for engine/manager contract tests.
- `backend/backend.go`: Type aliases to the core interfaces for compatibility.

## Profiling Layer (ebpf/ + audit/)
- `ebpf/common`, `ebpf/host`: CO-RE BPF sources and headers. `ebpf/objects` stores compiled objects; `scripts/build-ebpf.sh` builds them given `ebpf/vmlinux.h`.
- `audit/collector_linux.go`: Legacy eBPF collector that loads objects and streams events; used by the eBPF profiling controller. Stub on non-Linux.
- Profiling is opt-in per execution; if BTF/objects are missing or the controller fails, execution continues without receipts.

## Receipts & Policy
- Receipts exist only when profiling is enabled. Provenance values: host/guest/host+guest (guest programs pending). Redactions are explicit. Hashes cover stdout/stderr to prevent tampering.
- Policy evaluation (`core/policy`) is deterministic over receipts; enforcement hooks live outside core (see node/enforcement).

## Node Control Plane (node/)
- `node/config`: Validates node-agent defaults (backend, profiling mode, concurrency).
- `node/registry`: Thread-safe backend registry (name → backend).
- `node/pool`: Concurrency limiter for node-agent workers.
- `node/manager`: Orchestrates backend selection, engine execution, optional enforcement; returns `ExecutionResult`.
- `node/enforcement`: Interfaces for runtime enforcement; default is noop. Intended to act on receipts/policies without changing core semantics.

## Guest (guest/)
- `guest/transport`: Transport interface for guest → host event delivery; loopback implementation emits JSON lines (tests included). Designed for future vsock/shared-memory transports.
- `guest/init`, `guest/rootfs`: Notes/placeholders for guest image preparation; keep minimal and CO-RE friendly.

## Deployment (deploy/)
- `deploy/docker`: Steps for building/running sandbox-only or profiling-enabled containers; notes on mounts/capabilities and receipt persistence.
- `deploy/k8s`: Guidance for DaemonSet topology, BPF mounts, per-job profiling annotations, backend registration per node, and receipt streaming.
- `deploy/local`: Local dev flows for sandbox-only and profiling runs, plus node-agent usage examples.

## Legacy vs New
- Legacy receipt/collector live in `audit/`; new receipt grammar lives in `core/receipt`. The eBPF controller bridges legacy events into the new aggregator.
- Runner package was removed; `execution.Engine` replaces it for orchestration.

## Tests & Gaps
- Passing: `go test ./...` covering engine profiling on/off, process backend contract, policy determinism, node registry/pool/manager, guest transport, CLI basics.
- Gaps to fill: guest/combined profiling, real Firecracker/Kata lifecycle + contract tests, node-agent long-running/integration tests, CO-RE validation across distros, backend parity contract suite, deployment artifacts (Dockerfile/DaemonSet), policy enforcement wiring in node-agent.

## Quickstart Pointers
- Sandbox-only CLI: `./glasshouse run --profile disabled -- echo hello`
- Profiling CLI (Linux): `sudo GLASSHOUSE_BPF_DIR=./ebpf/objects ./glasshouse run --profile host -- echo hello`
- Node agent: `go run ./cmd/node-agent --backend process --profile disabled -- ls`
- Build eBPF: `./scripts/build-ebpf.sh` (requires `ebpf/vmlinux.h` from `bpftool btf dump ...`)
