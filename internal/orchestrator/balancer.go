package orchestrator

import (
	"errors"

	"convoy/pkg/loadbalancer"
)

// Balancer wraps a loadbalancer.Balancer to select containers for work.
type Balancer struct {
	lb loadbalancer.Balancer
}

// NewBalancer creates a new Balancer.
func NewBalancer(lb loadbalancer.Balancer) (*Balancer, error) {
	if lb == nil {
		return nil, errors.New("load balancer is required")
	}

	return &Balancer{lb: lb}, nil
}

// Next returns the next container endpoint to use.
func (b *Balancer) Next() string {
	return b.lb.Next()
}

// Add registers a container endpoint with the balancer.
func (b *Balancer) Add(endpoint string) {
	if endpoint == "" {
		return
	}

	b.lb.AddServer(endpoint)
}

// Remove deregisters a container endpoint from the balancer.
func (b *Balancer) Remove(endpoint string) {
	if endpoint == "" {
		return
	}

	b.lb.RemoveServer(endpoint)
}
