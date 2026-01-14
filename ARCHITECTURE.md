# Glasshouse Architecture

## Overview
Glasshouse runs a command, observes its host-side activity through the kernel, and emits a single execution receipt (`receipt.json`). The core runtime flow is:

1) CLI parses arguments and selects a backend.
2) Runner orchestrates the backend lifecycle and audit collection.
3) eBPF programs emit events into a ring buffer.
4) The collector decodes events and the aggregator builds a receipt.

Glasshouse is an observer, not an enforcer. It reports what the OS observed rather than what the workload claims to have done.

## Runner Role
The runner is the orchestrator. It:

- Calls the backend lifecycle in order: Prepare -> Start -> Wait -> Cleanup.
- Starts the audit collector and feeds events into the aggregator.
- Builds the receipt, including process tree, filesystem/network activity, syscalls, artifacts, and execution metadata.
- Exits with the child exit code and always writes `receipt.json` via the CLI.

The runner does not contain backend-specific logic. It only interacts with backends through the `backend.Backend` interface and optional data providers for receipt enrichment.

## Backend Role
Backends encapsulate how a workload is executed. Each backend is responsible for:

- Preparing the execution environment (if required).
- Starting the workload and returning the root PID for aggregation.
- Waiting for completion and returning the exit code.
- Cleaning up any backend resources.
- Reporting execution metadata (`backend` and `isolation`).

The default process backend executes a host process and preserves current behavior. Future VM backends (Firecracker, Kata) will implement the same interface without changing the runner.

## Host-Side Observability Model
Glasshouse observes host kernel activity using eBPF tracepoints. This design keeps the workload unmodified:

- No guest-side instrumentation is required.
- The host kernel is the single source of truth for process, file, and network activity.
- The collector and aggregator are decoupled from execution mode.

This approach is stable across execution backends because the observation layer is anchored in the host OS rather than the workload runtime.

## Why Kernel Observation (Not Workload Instrumentation)
Instrumenting workloads adds complexity, requires cooperation from the target process, and can be bypassed or disabled. Kernel observation provides:

- Uniform coverage across languages and runtimes.
- Minimal operational burden on the workload.
- A consistent, host-verifiable record of activity.

## Future VM Backends (Firecracker/Kata)
VM backends will fit into the same architecture by implementing the backend interface:

- The backend will manage VM lifecycle and report a root PID that represents the host-side process tree to observe (e.g., the VM monitor process).
- The runner and audit pipeline remain unchanged.
- The receipt will include `execution.backend = "firecracker"` and `execution.isolation = "vm"` to make isolation mode explicit.

This preserves a stable core while enabling new execution environments without changing collector or aggregation logic.
