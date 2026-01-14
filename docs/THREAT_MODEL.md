# Threat Model

- Host OS is the source of truth for observation; receipts capture host-observed behavior.
- Profiling unavailability must not break execution but must be surfaced in metadata.
- Backends may run untrusted code; sandbox-only mode must function without profiling or policy.
