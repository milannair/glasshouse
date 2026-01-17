# Learn Glasshouse

This walkthrough explains the repo structure, what each component does, and how they interact.

## Quickstart (Firecracker Server)

The fastest way to run Python code in isolated microVMs:

```bash
# On a GCP VM with nested virt
./scripts/quickstart.sh
sudo ./glasshouse-server

# Execute Python
curl -X POST localhost:8080/run -d '{"code": "print(2+2)"}'
```

## Top-Level Binaries (cmd/)

- `cmd/glasshouse`: CLI for running a single command with optional eBPF profiling. Writes `receipt.json` when profiling is enabled.
- `cmd/glasshouse-server`: HTTP server that runs Python code in Firecracker microVMs. Exposes `/run`, `/health`, and `/receipts/{id}` endpoints.
- `cmd/glasshouse-agent`: Daemon mode for long-running eBPF observation.

## Core Semantics (core/)

- `core/execution`: `ExecutionBackend` contract and `Engine` orchestration. Engine sequence: validate spec → backend Prepare/Start → optional profiling attach → Wait → aggregate → Cleanup.
- `core/profiling`: Mode/Target/Capabilities types. Controllers start profiling sessions that emit portable events.
- `core/receipt`: Receipt grammar, aggregator for profiling events, metadata enrichment.
- `core/policy`: Declarative, deterministic policy evaluator over receipts.
- `core/training`: Derives training signals from receipts.
- `core/version`: Version constants.

## Execution Backends (backend/)

- `backend/process`: Default backend; runs host processes. Captures stdout/stderr, reports exit code.
- `backend/firecracker`: Runs code in Firecracker microVMs. Creates workspace image, boots VM, reads result.
- `backend/fake`: Configurable test backend for contract tests.

## Guest (guest/)

- `guest/init`: Guest init binary (Go) that runs as PID 1 in Firecracker. Mounts workspace, executes Python, writes result, powers off.
- `guest/rootfs`: Dockerfile to build minimal Python rootfs.
- `guest/transport`: Transport interface for guest → host event delivery.

## Profiling Layer (ebpf/ + audit/)

- `ebpf/`: CO-RE BPF sources and compiled objects.
- `audit/`: Legacy eBPF collector used by the profiling controller.
- Profiling is opt-in; if disabled or unavailable, execution continues without receipts.

## Deployment (deploy/)

- `deploy/local`: Local dev flows.
- `deploy/docker`: Docker container runs.
- `deploy/k8s`: Kubernetes DaemonSet guidance.

## Scripts (scripts/)

- `scripts/quickstart.sh`: One-command setup for GCP VM.
- `scripts/build-rootfs.sh`: Build guest rootfs with Docker.
- `scripts/build-minimal-rootfs.sh`: Build rootfs without Docker.
- `scripts/build-ebpf.sh`: Compile eBPF objects.

## Repository Layout

```
glasshouse/
├── cmd/
│   ├── glasshouse/           # CLI
│   ├── glasshouse-server/    # HTTP API server
│   └── glasshouse-agent/     # eBPF daemon
├── backend/
│   ├── process/              # Host process backend
│   ├── firecracker/          # Firecracker VM backend
│   └── fake/                 # Test backend
├── core/
│   ├── execution/            # Engine + interfaces
│   ├── receipt/              # Receipt types
│   ├── profiling/            # Profiling controllers
│   └── policy/               # Policy evaluation
├── guest/
│   ├── init/                 # Guest init binary
│   ├── rootfs/               # Dockerfile
│   └── transport/            # Event transport
├── audit/                    # eBPF collector
├── ebpf/                     # BPF programs
├── scripts/                  # Setup scripts
└── deploy/                   # Deployment guides
```
