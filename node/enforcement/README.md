# Node Enforcement

Runtime enforcement hooks for node-level decisions.

- `Enforcer` operates on receipts and can block/record executions; default is `NoopEnforcer`.
- Enforcement is intentionally separate from policy evaluation; decisions should be derived from receipts/policies upstream.
- Keep this layer substrate-specific; core logic must remain unaffected by enforcement presence.
