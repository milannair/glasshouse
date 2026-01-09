package backend

import "os"

type ProcessStateProvider interface {
	ProcessState() *os.ProcessState
}
