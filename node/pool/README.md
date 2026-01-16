# Node Pool

Lightweight concurrency limiter for node-agent executions.

- `Pool.Go` runs a function while respecting a fixed concurrency cap.
- Use `Wait` during shutdown or before node drain to ensure in-flight work finishes.
- Keep pools small on nodes where profiling or resource contention is sensitive.
