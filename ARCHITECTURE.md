# Glasshouse Architecture

## Overview

Glasshouse runs code in isolated environments and produces execution receipts. Two primary modes:

1. **Firecracker Server**: HTTP API that runs Python in microVMs
2. **CLI with Profiling**: Run any command with optional eBPF-based observation

## Execution Flow (Firecracker Server)

```
HTTP POST /run {"code": "..."}
        │
        ▼
┌─────────────────────────────────┐
│     glasshouse-server           │
│  1. Create workspace image      │
│  2. Boot Firecracker VM         │
│  3. Wait for completion         │
│  4. Read result from workspace  │
│  5. Save receipt                │
└─────────────────────────────────┘
        │
        ▼
┌─────────────────────────────────┐
│      Firecracker microVM        │
│  guest-init (PID 1):            │
│  - Mount /workspace             │
│  - Read code.py                 │
│  - python3 -c <code>            │
│  - Write result.json            │
│  - poweroff                     │
└─────────────────────────────────┘
```

## Backend Interface

All backends implement `ExecutionBackend`:

```go
type ExecutionBackend interface {
    Name() string
    Prepare(ctx context.Context) error
    Start(spec ExecutionSpec) (ExecutionHandle, error)
    Wait(h ExecutionHandle) (ExecutionResult, error)
    Kill(h ExecutionHandle) error
    Cleanup(h ExecutionHandle) error
}
```

### Available Backends

| Backend | Isolation | Use Case |
|---------|-----------|----------|
| `process` | None | CLI, direct execution |
| `firecracker` | VM | API server, untrusted code |

## Receipt Pipeline

When profiling is enabled (CLI mode), the execution engine:

1. Attaches eBPF programs to syscall tracepoints
2. Collects events into a ring buffer
3. Aggregates events into a receipt
4. Writes `receipt.json` on exit

Firecracker mode produces simpler receipts with stdout/stderr, exit code, and timing.

## Design Principles

- **Backends are swappable**: Same interface, different isolation
- **Profiling is optional**: Fail-open, not required for basic execution
- **Receipts are immutable**: Hash-protected, timestamped records
- **Guest is minimal**: Only what's needed to run code and report results
