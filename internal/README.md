# Internal

Place small shared helpers here that are not part of the public core API. Keep the surface minimal:

- Logging, metrics, or feature-flag helpers used by multiple binaries.
- Utilities must remain substrate-agnostic and testable.
- Avoid coupling to specific backends or profiling providers to keep replacements easy.
