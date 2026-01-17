package orchestrator

import (
	"errors"
	"fmt"
	"sync"
)

// Registry stores metadata about managed containers.
type Registry struct {
	mu         sync.RWMutex
	containers map[string]*Container
}

// NewRegistry creates an empty container registry.
func NewRegistry() *Registry {
	return &Registry{
		containers: make(map[string]*Container),
	}
}

// Register adds or updates a container entry.
func (r *Registry) Register(container *Container) error {
	if container == nil {
		return errors.New("container is required")
	}

	if container.ID == "" {
		return errors.New("container id is required")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.containers[container.ID] = container
	return nil
}

// Remove deletes a container from the registry.
func (r *Registry) Remove(id string) {
	if id == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.containers, id)
}

// Get returns a container by ID.
func (r *Registry) Get(id string) (*Container, bool) {
	if id == "" {
		return nil, false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	container, ok := r.containers[id]
	return container, ok
}

// List returns all registered containers.
func (r *Registry) List() []*Container {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Container, 0, len(r.containers))
	for _, container := range r.containers {
		result = append(result, container)
	}

	return result
}

// Require ensures the container exists otherwise returns an error.
func (r *Registry) Require(id string) (*Container, error) {
	container, ok := r.Get(id)
	if !ok {
		return nil, fmt.Errorf("container %s not found", id)
	}

	return container, nil
}
