package profiling

import "context"

// Mode expresses how profiling should be attached.
// Profiling is optional and defaults to disabled.
type Mode string

const (
	ProfilingDisabled Mode = "disabled"
	ProfilingHost     Mode = "host"
	ProfilingGuest    Mode = "guest"
	ProfilingCombined Mode = "combined"
)

// Capabilities declares which profiling attachment points a provider supports.
type Capabilities struct {
	Host     bool
	Guest    bool
	Combined bool
}

// Target describes the process identity a profiler should attach to.
// The fields are intentionally substrate-agnostic so backends can be swapped.
type Target struct {
	// RootPID is the host-visible PID to attach to (or 0 if unknown).
	RootPID int
	// CgroupPath identifies the execution cgroup if available.
	CgroupPath string
	// Namespaces carries namespace identifiers to help select attachment scope.
	Namespaces map[string]string
	Mode       Mode
}

// EventType is the classification of an observation emitted by profiling.
type EventType uint32

const (
	EventExec    EventType = 1
	EventOpen    EventType = 2
	EventConnect EventType = 3
)

// Event is a substrate-agnostic observation captured during execution.
// The fields mirror the current eBPF emission format but do not assume eBPF.
type Event struct {
	Type       EventType
	PID        uint32
	PPID       uint32
	CgroupID   uint64
	Flags      uint32
	Comm       string
	Path       string
	AddrFamily uint8
	Proto      uint8
	Addr       [16]byte
	Port       uint16
}

// Session represents a running profiling attachment.
type Session interface {
	Events() <-chan Event
	Errors() <-chan error
	Close() error
}

// Controller creates profiling sessions and advertises support.
type Controller interface {
	Start(ctx context.Context, target Target) (Session, error)
	Capabilities() Capabilities
}
