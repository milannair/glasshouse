package registry

import (
	"fmt"
	"sync"

	"glasshouse/core/execution"
)

// Registry maps backend names to ExecutionBackend implementations.
type Registry struct {
	mu       sync.RWMutex
	backends map[string]execution.ExecutionBackend
}

func New() *Registry {
	return &Registry{backends: map[string]execution.ExecutionBackend{}}
}

// Register adds or replaces a backend under a given name.
func (r *Registry) Register(name string, backend execution.ExecutionBackend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends[name] = backend
}

// Get returns a backend by name or an error if missing.
func (r *Registry) Get(name string) (execution.ExecutionBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if backend, ok := r.backends[name]; ok {
		return backend, nil
	}
	return nil, fmt.Errorf("backend %q not registered", name)
}

// Names returns the registered backend names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.backends))
	for name := range r.backends {
		out = append(out, name)
	}
	return out
}
