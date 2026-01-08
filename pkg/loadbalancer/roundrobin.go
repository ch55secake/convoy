package loadbalancer

import "sync"

// RoundRobin implements the Balancer interface
type RoundRobin struct {
	servers []string
	index   int
	mu      sync.Mutex
}

// NewRoundRobin creates a new RoundRobin balancer
func NewRoundRobin() *RoundRobin {
	return &RoundRobin{}
}

// Next returns the next server
func (rr *RoundRobin) Next() string {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if len(rr.servers) == 0 {
		return ""
	}
	server := rr.servers[rr.index]
	rr.index = (rr.index + 1) % len(rr.servers)
	return server
}

// AddServer adds a server to the list
func (rr *RoundRobin) AddServer(server string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.servers = append(rr.servers, server)
}

// RemoveServer removes a server from the list
func (rr *RoundRobin) RemoveServer(server string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	for i, s := range rr.servers {
		if s == server {
			rr.servers = append(rr.servers[:i], rr.servers[i+1:]...)
			break
		}
	}
}
