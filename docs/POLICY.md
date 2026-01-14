# Policy

- Policies are declarative and evaluated against receipts deterministically.
- Evaluation is separate from enforcement; backends never enforce policy at runtime.
- Verdicts include reasons to support auditability and testing.
- Policy inputs are receipts only; the core does not read backend internals or kernel-specific state.
- Enforcement hooks live in `node/enforcement`; they may act on verdicts but execution must succeed even if enforcement is absent.
- Tests must cover policy correctness, determinism, and schema/version compatibility.
