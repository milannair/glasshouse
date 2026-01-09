# Working State

Updated: 2026-01-09T05:54:32Z

## Repository Overview
- glasshouse is an auditing-first execution runner that emits `receipt.json` with OS-level activity.
- Core execution path is in `runner/` with audit types in `audit/` and the CLI in `cmd/glasshouse/`.
- eBPF-related pieces live in `ebpf/`, with scripts and demos in `scripts/` and `demo/`.

## Current Behavior
- Running a missing binary yields `exit_code: 1` and an error like `fork/exec /bin/does-not-exist: no such file or directory`.
- The receipt includes environment/runtime details and empty activity sections when nothing runs.

## Recent Changes (feature/guest)
- Use `golang.org/x/sys/unix` for guest memlock rlimit handling.
- Adjust signal handling to avoid canceling the run context and to handle PID 1 cases.
- Keep the command/collector tied to the base context while using a separate signal context when needed.

## Repo Status
- Branch: `feature/guest`.
- Working tree: clean.

## Verification
- Manual: `/bin/does-not-exist` run returns `exit_code: 1` with missing binary error (user run).

## Tests
- Not run.
