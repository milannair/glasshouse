package receipt

import (
	"strings"
	"time"
)

func observationModeFromProvenance(provenance string) string {
	switch strings.ToLower(strings.TrimSpace(provenance)) {
	case "guest":
		return "guest"
	case "host+guest", "guest+host", "combined":
		return "host+guest"
	case "host":
		fallthrough
	default:
		return "host"
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339Nano)
}
