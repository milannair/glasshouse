# Node Manager

Coordinates backend selection, execution, optional profiling, and enforcement.

- Accepts a `Request` (args, env, backend, profiling) and returns a `Response` wrapping `ExecutionResult`.
- Uses `registry.Registry` to resolve backends and `core/execution.Engine` to run them.
- Enforcement is optional; plug in an `Enforcer` to gate execution or audit receipts.
- Concurrency is managed via `node/pool`; default is conservative (1).
