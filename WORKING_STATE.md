# Working State

Updated: 2026-01-09T08:00:11Z

## Repository Map
- `core/` contains execution semantics (`execution.Engine`), profiling contracts, receipt grammar, policy evaluation, training hooks, and versioning.
- `backend/` holds adapters (process, firecracker stub, kata stub, fake test backend) that implement `ExecutionBackend`.
- `cmd/` includes the CLI plus placeholders for `node-agent` and `guest-probe`.
- `ebpf/` now splits shared headers (`ebpf/common`), host programs (`ebpf/host`), guest placeholder (`ebpf/guest`), and build artifacts (`ebpf/objects`).
- `audit/` retains the legacy collector/aggregator; new receipts live in `core/receipt`.
- `deploy/`, `guest/`, `node/`, and `internal/` are scaffolded for future control-plane and guest-side components.

## System Overview
- Sandbox-only mode works everywhere; profiling is optional and fail-open.
- When profiling is enabled, the engine attaches a profiler, aggregates events, and emits a versioned receipt with provenance.
- Execution backends report stable identities for profiling attachment but do not evaluate policy or enforce decisions.
- Policy evaluation is deterministic and receipt-based; enforcement hooks live outside the core.

## Backend Abstraction (`core/execution`)
- `ExecutionBackend` defines `Prepare`, `Start`, `Wait`, `Kill`, `Cleanup`, and `ProfilingInfo`.
- Optional interfaces surface stdout/stderr buffers, extra errors, process state, and backend metadata.
- Process backend remains the default; firecracker/kata stubs declare isolation metadata.
- Profiling capabilities are declared via `BackendProfilingInfo`; profiling mode is per-execution.

## CLI Behavior (`cmd/glasshouse/main.go`)
- Command syntax: `glasshouse run [--guest] [--profile disabled|host|guest|combined] -- <command> [args...]`.
- Sandbox-only mode (default) runs without profiling and does not emit a receipt.
- Receipt emission occurs only when profiling is enabled; profiling defaults to a no-op controller for portability.
- Exit code mirrors the child exit code; errors before execution return 1.

## Execution Engine Flow (`core/execution/engine.go`)
- Validate spec -> `Prepare` backend -> `Start` -> optional profiling attach -> `Wait` -> aggregate -> `Cleanup`.
- Profiling failures do not block execution; errors are surfaced in receipt metadata when profiling is requested.
- Receipt metadata includes version, execution ID, provenance, artifacts, backend metadata, and optional redactions.
