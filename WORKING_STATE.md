# Working State

Updated: 2026-01-09T05:46:50Z

## Summary
- Use `golang.org/x/sys/unix` for guest memlock rlimit handling.
- Adjust signal handling to avoid canceling the run context and to handle PID 1 cases.
- Keep the command/collector tied to the base context while using a separate signal context when needed.

## Verification
- Manual: `/bin/does-not-exist` run returns `exit_code: 1` with missing binary error (user run).

## Tests
- Not run.
