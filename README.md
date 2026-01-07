# glasshouse

Auditing-first execution runner for AI agents. This MVP observes system activity via eBPF and emits a JSON execution receipt.

## What it does

- Executes an arbitrary command as a child process
- Captures:
  - process exec events
  - process parent relationships (ppid)
  - command line snapshots (argv, truncated)
  - file open events (classified as read vs write by flags)
  - outbound connect attempts
- Aggregates events into `receipt.json`

No enforcement, no allow/deny lists, no sandboxing.

## Requirements (Linux only)

- Linux kernel with BTF enabled (`/sys/kernel/btf/vmlinux`) and ringbuf support (5.8+)
- Go 1.21+
- clang/llvm
- bpftool
- root or CAP_BPF/CAP_SYS_ADMIN to load eBPF programs

**Note:** This project requires Linux. If you're on macOS or Windows, use Docker (see below).

## Quick Start with Docker (macOS/Windows)

If you're on macOS or Windows, you can run this project using Docker:

```bash
# Build and run
docker-compose build
docker-compose run --rm glasshouse
```

Or run a custom command:

```bash
docker-compose run --rm glasshouse run -- python3 demo/sneaky.py
```

The `receipt.json` will be generated in the current directory.

## Build

1) Generate `ebpf/vmlinux.h`:

```bash
sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c > ebpf/vmlinux.h
```

2) Compile eBPF programs:

```bash
./scripts/build-ebpf.sh
```

3) Build the CLI:

```bash
go build -o glasshouse ./cmd/glasshouse
```

## Run

```bash
sudo ./glasshouse run -- python3 demo/sneaky.py
```

Outputs:

- The child process stdout/stderr (streamed)
- `receipt.json` (execution receipt)

## Receipt schema (v0)

```json
{
  "exit_code": 0,
  "duration_ms": 312,
  "processes": [
    { "pid": 123, "ppid": 1, "cmd": "python3 demo/sneaky.py" }
  ],
  "filesystem": {
    "read": ["/work/input.txt"],
    "written": ["/work/output.txt"]
  },
  "network": {
    "connections": [
      { "dst": "1.2.3.4:443", "protocol": "tcp", "attempted": true }
    ]
  },
  "resources": {
    "cpu_time_ms": 3,
    "max_rss_kb": 8124
  }
}
```

## Notes and limitations

- File paths are captured from `open/openat` arguments. Relative paths are recorded as-is.
- Writes are inferred from open flags; actual write syscalls are not traced yet.
- Network protocol is inferred from `socket` arguments; inherited sockets may show protocol as unknown.
- Only `connect` attempts are tracked for outbound networking.
- Process tree is inferred from exec events; very short-lived processes might be missed.
- Command lines are captured from argv at exec time (first 8 args) and truncated to 256 bytes.
- If eBPF objects are missing, the runner still executes but omits audit data.

## Configuration

- `GLASSHOUSE_BPF_DIR`: override the directory containing `exec.o`, `fs.o`, and `net.o`.
