# Glasshouse Overview

Glasshouse is a modular sandbox and auditing substrate with optional profiling. Execution, observation, policy evaluation, and enforcement are separate concerns. Receipts are only emitted when profiling is enabled.

Key principles:

- The OS is the source of truth; observation is orthogonal to execution.
- Profiling is opt-in, fail-open, and CO-RE compatible; sandbox-only mode must work everywhere.
- Policy evaluation is deterministic and receipt-driven; enforcement is optional and substrate-specific.
- Backends are swappable without touching the core semantics.

Execution modes (per the master prompt):

- Sandbox-only (profiling off) for maximum portability.
- Sandbox + profiling (host/guest/combined) emitting structured receipts.
- Tool/agent execution with long-running state and subprocess spawning.
- Mixed-trust chains (agent -> tool -> untrusted code) that preserve boundaries.
- Future robotics via new backends and policy surfaces without refactoring core.

Artifacts:

- `core/execution.Engine` orchestrates backends and optional profiling.
- `core/receipt` defines the receipt grammar and metadata enrichment.
- `core/policy` evaluates receipts deterministically; enforcement is decoupled.
- `node/` packages scaffold the control plane and node-agent orchestration.
