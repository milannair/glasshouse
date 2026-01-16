# Receipt Schema

- Versioned via core/version.ReceiptVersion.
- Includes provenance (host/guest/host+guest), execution metadata (execution_id, start_time, end_time), observation_mode, completeness, process tree, filesystem/network/syscall summaries, artifacts, and resources.
- Policy metadata captures violations and enforcement decisions for explainability.
- Supports masking via path prefixes to redact sensitive entries while recording redactions.
- Receipts are only produced when profiling is enabled and attached.
- Deterministic serialization: stable field ordering and hashes for stdout/stderr artifacts.
- Redactions are explicit in `redactions` to aid audits and training pipelines.
- Legacy fields remain for backward compatibility, but `version` + `provenance` are the primary anchors.
