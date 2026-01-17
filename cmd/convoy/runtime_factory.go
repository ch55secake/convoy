package main

import (
	"fmt"

	"convoy/internal/app"
	"convoy/internal/orchestrator"
)

// dockerRuntimeFactory builds the Docker runtime using application config.
func dockerRuntimeFactory(cfg *app.Config) (orchestrator.Runtime, error) {
	dRuntime, err := orchestrator.NewDockerRuntime(cfg)
	if err != nil {
		return nil, fmt.Errorf("init docker runtime: %w", err)
	}

	return dRuntime, nil
}
