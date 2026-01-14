# Working State

Updated: 2026-01-09T10:30:00Z (branch `feature/modularity`)

## Repository Map (current)
- `core/`: `execution.Engine`, profiling contracts (modes/targets/capabilities), receipt grammar + metadata (`core/receipt`), policy evaluator, training helpers, version constants.
- `backend/`: process backend fully implemented; firecracker/kata are stubs that return not-implemented start/zero exit; fake backend for tests. `backend/backend.go` re-exports core interfaces.
- `cmd/`: `glasshouse` CLI runs commands; `node-agent` executes via manager/registry with profiling flags; `guest-probe` emits structured guest events.
- `ebpf/`: `common/` headers, `host/` programs, `objects/` output. Guest programs are placeholder. Scripts build CO-RE objects given `ebpf/vmlinux.h`.
- `audit/`: legacy collector and aggregator used by the eBPF profiling controller; legacy receipt kept for backward-compat tests.
- `node/`: config, registry, pool, manager, enforcement packages with tests; manager wraps `execution.Engine` and optional enforcer.
- `guest/`: transport interface with loopback + tests; init/rootfs docs only.
- `deploy/`: docker/k8s/local docs with concrete steps; `internal/` empty placeholder.

## Execution & Profiling
- Modes: profiling disabled (default) → sandbox-only, no receipt; profiling host via eBPF on Linux; guest/combined not yet implemented.
- CLI/node-agent select profiler: noop when disabled, eBPF controller when enabled (fails open if BPF missing). `GLASSHOUSE_BPF_DIR` overrides object dir.
- Engine flow: validate spec → backend Prepare/Start → optional profiling attach → Wait → aggregate → Cleanup. Profiling errors surfaced but never block execution.
- Receipts: emitted only when profiling enabled; version v0.3.0; include provenance, execution ID, artifacts hashes, environment, redactions via `MaskPaths`.

## Backends
- Process: real exec, signal handling, optional guest env prep, profiling info (host PID). Implements stdout/stderr capture, extra errors, process state.
- Firecracker/Kata: metadata + profiling capability declarations only; Start returns not-implemented (firecracker) or no-op (kata). Isolation marked `vm`.
- Fake: test backend for engine/manager contract tests.

## Node Agent / Control Plane
- `cmd/node-agent`: flags `--backend`, `--profile`, `--concurrency`; uses registry (process registered; firecracker stub also registered) and manager.
- Manager resolves backend, builds `ExecutionSpec`, runs engine, optionally enforces via enforcer (default noop). Concurrency limited by `node/pool`.
- Config validates backend name and concurrency > 0; defaults: backend=process, profiling=disabled, concurrency=1.

## Guest Probe / Transport
- `cmd/guest-probe`: emits heartbeat/info event JSON via loopback transport (stdout). Transport interface ready for vsock/shared-memory replacements.

## Profiling Layer
- `core/profiling/ebpf`: Linux controller wraps `audit.Collector`; stub on non-Linux. Capabilities: Host=true, Guest/Combined pending guest programs.
- Profiling must be optional; controller fails open when BTF/objects missing.

## Docs / Deploy
- README updated with profiling modes, node-agent usage, receipt v0.3.0 schema, repo layout.
- Deploy docs: docker/k8s/local include build/run steps, mounts, capability guidance, and receipt handling notes.

## Tests / Status
- `go test ./...` passes (requires Go build cache writable). Coverage includes engine profiling on/off, process backend contract, policy determinism, node registry/pool/manager, guest transport, CLI basics.
- Gaps: no guest/combined profiling tests; firecracker/kata lifecycle unimplemented; no backend parity contract tests across all adapters; no integration tests for node-agent long-running mode.

## Next Actions (targeted)
1) Implement guest/combined profiling paths + CO-RE guest programs; add capability detection and cross-distro validation tests.
2) Flesh out Firecracker/Kata backends (VM lifecycle, stable identity, profiling capabilities); add backend contract suite run across all backends.
3) Wire policy evaluation + enforcement into node-agent using `core/policy` and receipts; add tests.
4) Add deploy artifacts (Dockerfile, k8s DaemonSet/ConfigMap) matching deploy docs; include profiling/non-profiling variants.
5) Expand training/policy tests for receipt determinism across profiling on/off and redaction paths.
