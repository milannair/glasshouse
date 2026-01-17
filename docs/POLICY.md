# Policy

- Policies are declarative and evaluated deterministically against receipts.
- Evaluation is separate from enforcement; backends never enforce policy at runtime.
- Verdicts include reasons to support auditability and testing.
- Policy inputs are receipts only; the core does not read backend internals or LSM state.
- Tests must cover policy correctness, determinism, and schema/version compatibility.
- Receipts include policy violations for explainability.

Phases:

- Pre-execution: static constraints evaluated when an execution is registered.
- Post-execution: receipt evaluation that marks the receipt trusted or untrusted.

The `core/policy` package provides the policy evaluation logic.
