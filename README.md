# Glasshouse

Run Python code in Firecracker microVMs with execution receipts.

## Quickstart (GCP VM)

1. **Create a GCP VM** with nested virtualization:
```bash
gcloud compute instances create glasshouse-vm \
  --zone=us-central1-a \
  --machine-type=n2-standard-4 \
  --min-cpu-platform="Intel Haswell" \
  --enable-nested-virtualization \
  --image-family=ubuntu-2204-lts \
  --image-project=ubuntu-os-cloud
```

2. **SSH and run quickstart**:
```bash
gcloud compute ssh glasshouse-vm --zone=us-central1-a

# Clone and setup
git clone https://github.com/USER/glasshouse.git
cd glasshouse
./scripts/quickstart.sh

# Start server
sudo ./glasshouse-server
```

3. **Execute Python code**:
```bash
curl -X POST localhost:8080/run \
  -H "Content-Type: application/json" \
  -d '{"code": "print(2+2)"}'
```

Response:
```json
{"stdout": "4\n", "exit_code": 0, "receipt_id": "exec-1234-1"}
```

## API

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/health` | GET | Health check |
| `/run` | POST | Execute Python code |
| `/receipts/{id}` | GET | Fetch execution receipt |

### POST /run

**Request:**
```json
{
  "code": "print('hello')",
  "timeout": 60
}
```

**Response:**
```json
{
  "stdout": "hello\n",
  "stderr": "",
  "exit_code": 0,
  "duration_ms": 142,
  "receipt_id": "exec-1234-1"
}
```

## Receipts

Every execution produces a receipt saved to `/var/lib/glasshouse/receipts/`:

```json
{
  "id": "exec-1234-1",
  "timestamp": "2026-01-16T18:00:00Z",
  "code_hash": "abc123",
  "exit_code": 0,
  "duration_ms": 142,
  "stdout": "4\n",
  "stderr": ""
}
```

## Architecture

```
┌────────────────────────────────────────┐
│              GCP VM                    │
│  ┌──────────────────────────────────┐  │
│  │     glasshouse-server :8080      │  │
│  └──────────────────────────────────┘  │
│           │                            │
│           ▼                            │
│  ┌──────────────────────────────────┐  │
│  │      Firecracker microVM         │  │
│  │  ┌────────────────────────────┐  │  │
│  │  │    Python execution        │  │  │
│  │  └────────────────────────────┘  │  │
│  └──────────────────────────────────┘  │
│           │                            │
│           ▼                            │
│  ┌──────────────────────────────────┐  │
│  │        Receipt saved             │  │
│  └──────────────────────────────────┘  │
└────────────────────────────────────────┘
```

Each `/run` request:
1. Boots a fresh Firecracker microVM
2. Executes the Python code
3. Captures stdout/stderr
4. Powers off the VM
5. Saves a receipt
6. Returns the result

## Local Development

```bash
# Build CLI (runs without Firecracker)
go build -o glasshouse ./cmd/glasshouse
./glasshouse run --profile disabled -- echo hello

# Build server (requires Firecracker + kernel + rootfs)
go build -o glasshouse-server ./cmd/glasshouse-server
```

## Requirements

- Linux x86_64 with KVM (`/dev/kvm`)
- Go 1.21+
- Firecracker (installed by quickstart)

## License

Apache-2.0
