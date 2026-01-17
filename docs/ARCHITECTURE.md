# Architecture (Detailed)

See also: [ARCHITECTURE.md](../ARCHITECTURE.md) in the root.

## Core Components

### Execution Engine (`core/execution`)

The engine orchestrates backend lifecycle:
1. Validate spec
2. Backend Prepare → Start → Wait → Cleanup
3. Optional profiling attach during Wait
4. Receipt aggregation

### Backends (`backend/`)

| Backend | Isolation | Description |
|---------|-----------|-------------|
| `process` | None | Direct host execution |
| `firecracker` | VM | Firecracker microVM |
| `fake` | None | Test mock |

### Profiling (`core/profiling`)

- Optional and fail-open
- eBPF-based on Linux
- Captures syscalls, file I/O, network

### Receipts (`core/receipt`)

Structured execution records including:
- Exit code, timing
- stdout/stderr hashes
- Process tree (when profiling enabled)

## Server Mode

`glasshouse-server` handles HTTP requests:

```
POST /run → Create workspace → Boot VM → Execute → Return result
```

Each request is isolated in its own Firecracker VM.

## CLI Mode

`glasshouse run` executes directly via process backend with optional eBPF profiling.
