# Receipt Schema

- Versioned via core/version.ReceiptVersion.
- Includes provenance (host/guest/host+guest), execution metadata, process tree, filesystem/network/syscall summaries, artifacts, and resources.
- Supports masking via path prefixes to redact sensitive entries while recording redactions.
- Receipts are only produced when profiling is enabled and attached.
