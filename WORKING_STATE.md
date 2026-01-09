# Working State

Updated: 2026-01-09T05:57:09Z

## Repository Map
- `cmd/glasshouse/` is the CLI entrypoint that parses arguments and writes `receipt.json`.
- `runner/` owns process execution, signal handling, receipt metadata, and resources.
- `audit/` defines the collector interface, event types, receipt schema, and the in-memory aggregator.
- `ebpf/` contains the BPF programs (exec, fs, net), shared headers, and build artifacts.
- `scripts/` contains build and WSL setup helpers; `demo/` is the activity generator.

## System Overview
- glasshouse runs a command, observes OS-level activity via eBPF tracepoints, and emits a JSON receipt.
- Events flow from BPF programs into a ring buffer, then into a Go collector and an aggregator.
- The aggregator scopes events to the root PID and its descendants and builds derived summaries.
- The CLI exits with the child exit code and persists a single `receipt.json` artifact.

## CLI Behavior (`cmd/glasshouse/main.go`)
- Command syntax is `glasshouse run [--guest] -- <command> [args...]`; unknown flags error out.
- `--guest` enables guest setup (mounts + memlock rlimit) before launching the command.
- After execution, the CLI always attempts to write `receipt.json` to the CWD.
- Exit code mirrors the child exit code; if an error occurs before exit code is set, it returns 1.

## Runner Execution Flow (`runner/run.go`)
- `Run` starts the child process with `exec.CommandContext`, capturing stdout/stderr to buffers.
- An `audit.Collector` is created if available; errors are recorded but do not prevent execution.
- The aggregator is rooted at the child PID and consumes events on collector channels.
- Signal handling is enabled in guest mode or when PID 1, with a dedicated signal context.
- Exit code uses `WaitStatus` and handles the guest reaper path to avoid ECHILD issues.

## Guest Mode and Signals (`runner/guest.go`)
- Guest setup mounts `proc`, `sysfs`, and `bpf` filesystems if missing and raises memlock.
- Memlock uses `golang.org/x/sys/unix` to set RLIMIT_MEMLOCK to infinity.
- A signal handler watches SIGCHLD to reap children and SIGTERM/SIGINT to forward shutdown.
- Shutdown signals are recorded into receipt errors for visibility.

## Receipt Assembly (`audit/receipt.go`, `runner/run.go`)
- `audit.Aggregator` tracks process tree, file reads/writes, network attempts, and syscall counts.
- Receipt v0.2 includes `outcome`, `timing`, `process_tree`, `environment`, and `artifacts`.
- Legacy fields (`exit_code`, `duration_ms`, `processes`, `filesystem`, `network`) are preserved.
- `execution_id` is a hash of timestamp, PID, and argv; stdout/stderr hashes are SHA256.
- Runtime name is inferred from argv (e.g., `python3.x` for python3).

## Collector Behavior (`audit/collector_linux.go`)
- Collector loads eBPF objects from `ebpf/objects` or `GLASSHOUSE_BPF_DIR`.
- It prefers `exec-argv.o` when `GLASSHOUSE_CAPTURE_ARGV` is enabled and allowed.
- Each object attaches to syscall tracepoints and yields events via ringbuf readers.
- Debug envs: `GLASSHOUSE_DEBUG_EVENTS` prints a limited event stream to stderr.
- On non-Linux builds, the collector is a stub that returns an error.

## eBPF Programs (`ebpf/*.c`)
- `exec.c` traces `execve`/`execveat` and captures the executable path.
- `exec_argv.c` captures up to 8 argv entries, space-joined; falls back to filename if empty.
- `fs.c` traces `open`/`openat` and records path plus flags for read/write inference.
- `net.c` traces `socket` and `connect`, infers protocol, and captures IPv4/IPv6 addresses.
- `common.h` defines the event struct and ringbuf map shared across programs.

## Scripts and Tooling
- `scripts/build-ebpf.sh` builds BPF objects with clang and a generated `vmlinux.h`.
- `scripts/run-wsl.sh` installs deps, builds BPF + CLI, and runs the demo or given command.
- The WSL script can install Go, bpftool, and tracefs/debugfs when missing.

## Demo Workload (`demo/sneaky.py`)
- Exercises filesystem reads/writes, directory churn, and JSON read/write.
- Makes TCP and UDP connection attempts, including IPv6.
- Spawns subprocesses to produce exec events and file writes.
- Cleans up temporary files and directories at the end.

## Configuration and Environment
- `GLASSHOUSE_BPF_DIR` selects the eBPF object directory.
- `GLASSHOUSE_CAPTURE_ARGV` enables argv capture; `force` bypasses WSL gating.
- `GLASSHOUSE_CAPTURE_ARGV_FORCE` is a legacy override for WSL.
- `GLASSHOUSE_DEBUG_TRACKING` and `GLASSHOUSE_DEBUG_COUNTS` add runtime diagnostics.

## Expected Runtime Behavior
- Missing binaries yield `exit_code: 1` and an error like `fork/exec ... no such file`.
- When no activity occurs, filesystem/network/syscall sections are empty but present.
- Network is reported as attempts, not byte counts (bytes are always 0 today).
- Filesystem read/write inference is derived from open flags.

## Platform Constraints
- Linux only; WSL is partially supported and may miss argv, syscalls, or IO events.
- Requires BTF (`/sys/kernel/btf/vmlinux`) and ringbuf support (kernel 5.8+).
- Root or CAP_BPF/CAP_SYS_ADMIN is needed to load BPF programs.

## Recent Changes (feature/guest)
- `runner/guest.go` uses `golang.org/x/sys/unix` for RLIMIT_MEMLOCK.
- Signal handling now avoids canceling the run context while still forwarding shutdown.
- Signal handling is enabled when running as guest or PID 1.

## Repository Status
- Branch: `feature/guest`.
- Receipt schema: v0.2 (legacy fields preserved for backward compatibility).

## Verification
- Manual: `/bin/does-not-exist` run returns `exit_code: 1` with missing binary error (user run).

## Tests
- Not run.
