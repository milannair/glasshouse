# Architecture

- Core defines execution, receipt, policy, and version grammars without assuming a substrate.
- Backends implement the ExecutionBackend contract and expose profiling capabilities.
- Profiling is optional, fail-open, and CO-RE compatible; absence of profiling must not block execution.
- Receipts are deterministic, versioned, and tied to profiling provenance (host/guest/combined).
- Control-plane components (node agent, guest probe) can be added without refactoring the core.
