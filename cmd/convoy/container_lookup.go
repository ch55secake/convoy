package main

import "convoy/internal/orchestrator"

// resolveContainer finds a container by name or ID.
// It first checks by name, then by ID.
func resolveContainer(ref string, containers []*orchestrator.Container) *orchestrator.Container {
	byID := make(map[string]*orchestrator.Container)
	byName := make(map[string]*orchestrator.Container)
	for _, c := range containers {
		if c == nil || c.ID == "" {
			continue
		}
		byID[c.ID] = c
		if c.Name != "" {
			byName[c.Name] = c
		}
	}

	if c := byName[ref]; c != nil {
		return c
	}
	return byID[ref]
}
