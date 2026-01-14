# Node Config

Defines the configuration used by `cmd/node-agent` and higher-level control planes.

- Defaults favor sandbox-only execution (`profiling=disabled`) and the `process` backend.
- `Concurrency` limits simultaneous executions on a node; keep conservative on resource-constrained hosts.
- Validate configs before use; missing backend names or zero concurrency is treated as an error.
