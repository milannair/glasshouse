# glasshouse

Auditing-first execution runner for AI agents. glasshouse runs a command, observes OS-level activity via eBPF, and emits a JSON execution receipt.

## Why it exists

Most agents can explain what they intended to do, but not what actually happened on the machine. glasshouse captures what the OS observed: process execs, file opens, and outbound connects. It is a recorder, not an enforcer.

## Features

- Run any command as a child process
- Capture process execs and parent relationships
- Track file opens (read vs write inferred from flags)
- Track outbound connect attempts
- Emit a single `receipt.json` artifact

## How it works (short)

1) eBPF programs attach to syscall tracepoints.
2) Events are written to a ring buffer.
3) A Go collector decodes events and aggregates them.
4) The CLI writes a JSON receipt and exits with the child’s exit code.

For a beginner-friendly walkthrough, see `info.md`.

## Requirements (Linux)

- Linux only. WSL is partially supported and may not work depending on kernel/verifier behavior.
- Linux kernel with BTF enabled (`/sys/kernel/btf/vmlinux`) and ringbuf support (5.8+)
- Go 1.21+
- clang/llvm
- bpftool
- root or CAP_BPF/CAP_SYS_ADMIN to load eBPF programs

### WSL notes

WSL’s eBPF verifier is stricter. The argv-capture program is skipped by default on WSL to avoid verifier failures. You can force it, but it may fail.

## Quick start (Linux)

```bash
sudo ./scripts/run-wsl.sh
```

Run a custom command:

```bash
sudo ./scripts/run-wsl.sh -- python3 demo/sneaky.py
```

## Build (manual)

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

- child process stdout/stderr
- `receipt.json`

## Configuration

- `GLASSHOUSE_BPF_DIR`: override the directory containing `exec.o`, `fs.o`, and `net.o`.
- `GLASSHOUSE_CAPTURE_ARGV=1`: request argv capture (skipped on WSL).
- `GLASSHOUSE_CAPTURE_ARGV=force`: force argv capture on WSL (may fail verification).

WSL helpers:

- `scripts/run-wsl.sh --capture-argv`
- `scripts/run-wsl.sh --force-capture-argv`

## Receipt schema (v0.2)

```json
{
  "receipt_version": "v0.2",
  "execution_id": "<stable unique id>",
  "timestamp": "<RFC3339 timestamp>",
  "outcome": {
    "exit_code": 0,
    "signal": null,
    "error": null
  },
  "timing": {
    "duration_ms": 312,
    "cpu_time_ms": 3
  },
  "process_tree": [
    {
      "pid": 123,
      "ppid": 1,
      "exe": "/usr/bin/python3",
      "argv": ["python3", "demo/sneaky.py"],
      "working_dir": "/work"
    }
  ],
  "filesystem": {
    "reads": ["/work/input.txt"],
    "writes": ["/work/output.txt"],
    "deletes": [],
    "policy_violations": []
  },
  "network": {
    "attempts": [
      { "dst": "1.2.3.4:443", "protocol": "tcp", "result": "attempted" }
    ],
    "bytes_sent": 0,
    "bytes_received": 0
  },
  "syscalls": {
    "counts": {},
    "denied": []
  },
  "resources": {
    "max_rss_kb": 8124
  },
  "environment": {
    "runtime": "python3.x",
    "os": "linux",
    "arch": "amd64",
    "sandbox": { "network": "enabled" }
  },
  "artifacts": {
    "stdout_hash": "<sha256>",
    "stderr_hash": "<sha256>"
  },
  "exit_code": 0,
  "duration_ms": 312,
  "processes": [
    { "pid": 123, "ppid": 1, "cmd": "python3 demo/sneaky.py" }
  ]
}
```

Legacy fields (`exit_code`, `duration_ms`, `processes`, `filesystem.read`, `filesystem.written`, `network.connections`) are preserved for backward compatibility.

## Troubleshooting

- `vmlinux.h` missing:
  - `sudo bpftool btf dump file /sys/kernel/btf/vmlinux format c > ebpf/vmlinux.h`
- eBPF verifier errors on WSL:
  - use default exec capture (no argv) or `GLASSHOUSE_CAPTURE_ARGV=force`
- no events:
  - check tracefs is mounted and `ebpf/objects/*.o` exist

## Learn the project

Start with the hands-on tutorial in `info.md`.
