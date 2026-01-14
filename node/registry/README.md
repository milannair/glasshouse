# Node Registry

Maps backend names to `ExecutionBackend` instances.

- Register backends during node-agent startup (process by default; Firecracker/Kata when available).
- Lookups return an error for unknown names; callers decide whether to fall back or fail open.
- Designed to be thread-safe for concurrent access by worker pools.
