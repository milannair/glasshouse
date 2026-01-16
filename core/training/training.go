package training

import "glasshouse/core/receipt"

// Signal is a structured artifact derived from receipts for downstream learners.
type Signal struct {
	ReceiptID string
	Features  map[string]interface{}
}

// FromReceipt emits a coarse training signal; downstream pipelines can enrich it.
func FromReceipt(r receipt.Receipt) Signal {
	syscallCount := 0
	if r.Syscalls != nil {
		syscallCount = len(r.Syscalls.Counts)
	}
	readCount := 0
	writeCount := 0
	netCount := 0
	if r.Filesystem != nil {
		readCount = len(r.Filesystem.Reads)
		writeCount = len(r.Filesystem.Writes)
	}
	if r.Network != nil {
		netCount = len(r.Network.Attempts)
	}
	features := map[string]interface{}{
		"exit_code": r.ExitCode,
		"syscalls":  syscallCount,
		"reads":     readCount,
		"writes":    writeCount,
		"net":       netCount,
	}
	return Signal{
		ReceiptID: r.ExecutionID,
		Features:  features,
	}
}
