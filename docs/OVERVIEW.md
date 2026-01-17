# Glasshouse Overview

Glasshouse runs code in isolated Firecracker microVMs and produces execution receipts.

## Primary Use Case

Run untrusted Python code via HTTP API:

```bash
curl -X POST localhost:8080/run -d '{"code": "print(2+2)"}'
# {"stdout": "4\n", "exit_code": 0, "receipt_id": "..."}
```

Each execution:
1. Boots a fresh Firecracker microVM
2. Runs the Python code
3. Captures stdout/stderr
4. Saves a receipt
5. Returns the result

## Key Principles

- **Isolation**: Each execution runs in a fresh VM
- **Receipts**: Every execution produces a verifiable record
- **Simple API**: One endpoint to run code

## Components

| Component | Purpose |
|-----------|---------|
| `glasshouse-server` | HTTP API for Firecracker execution |
| `glasshouse` CLI | Run commands with optional profiling |
| `backend/firecracker` | VM lifecycle management |
| `guest/init` | Guest init that runs Python |

## Quickstart

```bash
# On GCP VM with nested virt
./scripts/quickstart.sh
sudo ./glasshouse-server
```
