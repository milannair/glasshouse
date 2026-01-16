# Policy

- Policies are declarative and evaluated deterministically against kernel events and receipts.
- Evaluation is separate from enforcement; backends never enforce policy at runtime.
- Verdicts include reasons to support auditability and testing.
- Policy inputs are kernel events or receipts only; the core does not read backend internals or LSM state.
- Enforcement hooks live in `core/agent` and `node/enforcement`; they act via observe+kill and are best-effort.
- Tests must cover policy correctness, determinism, and schema/version compatibility.
- Receipts include policy violations and enforcement decisions for explainability.

Phases:

- Pre-execution: static constraints evaluated when an execution is registered (labels, config hints).
- Runtime: event-driven checks on kernel observations; violations may trigger kill of a process or cgroup.
- Post-execution: receipt evaluation that marks the receipt trusted or untrusted.
