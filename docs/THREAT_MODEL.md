# Threat Model

- Host OS is the source of truth for observation; receipts capture host-observed behavior.
- Profiling unavailability must not break execution but must be surfaced in metadata.
- Backends may run untrusted code; sandbox-only mode must function without profiling or policy.
- Kernel version and distro differences are expected; CO-RE avoids hard-coded offsets.
- Enforcement is optional; absence of enforcement must not prevent truthful observation.
- Mixed-trust chains (agent → tool → untrusted code) must preserve execution boundaries and identities.
- Receipts are treated as audit artifacts; training and policy evaluation rely on their determinism.
