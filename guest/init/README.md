# Guest Init

Go binary that runs as PID 1 inside Firecracker microVMs.

## What It Does

1. Mounts `/proc`, `/sys`, `/dev`
2. Mounts workspace from `/dev/vdb` to `/workspace`
3. Reads Python code from `/workspace/.pending/code.py`
4. Executes `python3 -c <code>`
5. Writes result to `/workspace/.pending/result.json`
6. Powers off the VM

## Building

The init binary is built as part of `scripts/build-rootfs.sh`:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o guest-init ./guest/init/
```

## Result Format

```json
{
  "stdout": "...",
  "stderr": "...",
  "exit_code": 0,
  "duration_ms": 142,
  "error": ""
}
```
