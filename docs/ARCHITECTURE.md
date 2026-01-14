# Architecture

- Core (`core/`) defines execution, receipt, policy, and version grammars without assuming a substrate.
- Backends (`backend/`) implement the `ExecutionBackend` contract and expose profiling capabilities; process backend today, VM stubs for Firecracker/Kata.
- Profiling (`core/profiling`) is optional, fail-open, and CO-RE compatible; absence of profiling must not block execution.
- Receipts (`core/receipt`) are deterministic, versioned, and tied to profiling provenance (host/guest/combined).
- Control-plane components (`cmd/node-agent`, `node/`) coordinate long-running execution and enforcement; `cmd/guest-probe` is the guest-side hook for VM backends.

Execution lifecycle:

1. CLI or node-agent builds an `ExecutionSpec`.
2. `ExecutionBackend` prepares and starts the workload, returning an identity suitable for profiling attachment.
3. If profiling is enabled, a profiler attaches (host/guest/combined) and streams events.
4. Aggregators build receipts; metadata is enriched with backend information and redactions.
5. Policy evaluation runs on receipts; enforcement, if present, is substrate-specific and optional.

Separation of concerns:

- Execution vs Observation: execution works with profiling off; observation is additive.
- Policy vs Enforcement: policy evaluation is deterministic and receipt-driven; enforcement hooks may apply runtime controls but are not required for execution.
- Core vs Backends: core never references substrate-specific APIs; backends are replaceable without core refactors.
