package cmds

import (
	"fmt"
	"strings"
	"time"

	"convoy/internal/orchestrator"
)

// ContainerIndex provides a fast lookup of containers by ID or name.
type ContainerIndex struct {
	byID   map[string]*orchestrator.Container
	byName map[string]*orchestrator.Container
	list   []*orchestrator.Container
}

// NewContainerIndex builds an index from a list of containers.
func NewContainerIndex(containers []*orchestrator.Container) *ContainerIndex {
	idx := &ContainerIndex{
		byID:   make(map[string]*orchestrator.Container),
		byName: make(map[string]*orchestrator.Container),
		list:   containers,
	}
	for _, c := range containers {
		if c == nil || c.ID == "" {
			continue
		}
		idx.byID[c.ID] = c
		if c.Name != "" {
			idx.byName[c.Name] = c
		}
	}
	return idx
}

// Resolve finds a container by name or ID. Returns nil if not found.
func (idx *ContainerIndex) Resolve(ref string) *orchestrator.Container {
	if c := idx.byName[ref]; c != nil {
		return c
	}
	return idx.byID[ref]
}

// ResolveWithEndpoint finds a container and validates it has an endpoint.
func (idx *ContainerIndex) ResolveWithEndpoint(ref string) (*orchestrator.Container, error) {
	c := idx.Resolve(ref)
	if c == nil {
		return nil, fmt.Errorf("container not found: %s", ref)
	}
	if c.Endpoint == "" {
		return nil, fmt.Errorf("container %s has no gRPC endpoint", ref)
	}
	return c, nil
}

// List returns all containers in the index.
func (idx *ContainerIndex) List() []*orchestrator.Container {
	return idx.list
}

// LoadContainers fetches the container list from the manager and returns an index.
func LoadContainers() (*ContainerIndex, error) {
	app, err := getApp()
	if err != nil {
		return nil, err
	}

	mgr, err := app.Manager()
	if err != nil {
		return nil, err
	}

	containers, err := mgr.List()
	if err != nil {
		return nil, err
	}

	return NewContainerIndex(containers), nil
}

// RPCClient wraps an RPC instance with a cleanup function.
type RPCClient struct {
	*orchestrator.RPC
}

// NewRPCClient creates an RPC client with the given timeout configuration.
// The caller should defer Close() after use.
func NewRPCClient(dialTimeout, callTimeout time.Duration) *RPCClient {
	rpc := orchestrator.NewRPC(orchestrator.RPCConfig{
		DialTimeout: dialTimeout,
		CallTimeout: callTimeout,
	})
	return &RPCClient{RPC: rpc}
}

// NewRPCClientWithTimeout creates an RPC client using the same timeout for dial and call.
func NewRPCClientWithTimeout(timeout time.Duration) *RPCClient {
	return NewRPCClient(timeout, timeout)
}

// ParseEnvVars converts ["KEY=value", ...] to map[string]string.
func ParseEnvVars(envVars []string) map[string]string {
	env := make(map[string]string)
	for _, e := range envVars {
		if idx := strings.Index(e, "="); idx > 0 {
			env[e[:idx]] = e[idx+1:]
		}
	}
	return env
}

// ContainerLabel returns the best display label for a container (Name if available, otherwise ID).
func ContainerLabel(c *orchestrator.Container) string {
	if c == nil {
		return ""
	}
	if c.Name != "" {
		return c.Name
	}
	return c.ID
}

// ResolveContainerIDs resolves a list of refs to container IDs.
// Returns resolved IDs and any refs that couldn't be found.
func (idx *ContainerIndex) ResolveContainerIDs(refs []string) (resolved []string, missing []string) {
	for _, ref := range refs {
		ref = strings.TrimSpace(ref)
		if ref == "" {
			continue
		}
		c := idx.Resolve(ref)
		if c == nil {
			missing = append(missing, ref)
			continue
		}
		resolved = append(resolved, c.ID)
	}
	return resolved, missing
}

// AllContainerIDs returns IDs of all containers in the index.
func (idx *ContainerIndex) AllContainerIDs() []string {
	ids := make([]string, 0, len(idx.list))
	for _, c := range idx.list {
		if c != nil && c.ID != "" {
			ids = append(ids, c.ID)
		}
	}
	return ids
}
